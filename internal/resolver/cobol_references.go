package resolver

import (
	"iter"
	"slices"
	"sort"
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// COBOL placeholder shapes this pass owns, after the unresolved:: marker.
// COPY IDMS members (`copybook::idms/…`) are dictionary members, never files
// (D-19), so they are excluded from the copybook prefix explicitly.
const (
	cobolProgramTargetPrefix  = "cobol::program::"
	cobolCopybookTargetPrefix = "cobol::copybook::"
	cobolIDMSCopyTargetPrefix = "cobol::copybook::idms/"

	// Edge Meta written by this pass (D-12 reason vocabulary).
	metaCobolUnresolvedReason = "cobol_unresolved_reason"
	metaCobolCandidateIDs     = "cobol_candidate_ids"
	// Edge Meta read by this pass: the original unresolved target the
	// extractor stamps on every literal CALL and non-IDMS COPY edge.
	metaCobolPlaceholder = "cobol_placeholder"

	// Program node Meta read by this pass: true on a program contained in
	// another program, stamped by the extractor from END PROGRAM nesting,
	// and the node ID of that directly containing program.
	metaCobolNested    = "cobol_nested"
	metaCobolContainer = "cobol_container"
	// Literal CALL edge Meta read by this pass: the calling program's node ID.
	metaCobolCallerProgram = "cobol_caller_program"
)

// ReconcileCobol re-decides every literal COBOL CALL and COPY edge whose
// caller repository is in changed (every edge when changed is empty) under
// the resolve mutex. It is the entry for the incremental mutation tails, so
// they decide each binding with the same code as ResolveAll.
func (r *Resolver) ReconcileCobol(changed map[string]struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reconcileCobol(changed)
}

// reconcileCobol decides every literal COBOL CALL and non-IDMS COPY edge
// whose caller repository is in changed, binding it to exactly one candidate
// or returning it to its placeholder.
//
// PURPOSE — the extractor emits `unresolved::cobol::program::<key>` for a
// literal CALL / LINK / XCTL and `unresolved::cobol::copybook::<key>` for a
// COPY / -INC / SQL INCLUDE, with <key> already normalized under
// cobol-name-norm/1, and keeps that target in cobol_placeholder. This pass
// visits both the edges still on a placeholder (EdgesWithUnresolvedTarget)
// and the edges already bound to a program or copybook candidate (their
// in-edges that carry cobol_placeholder), and decides each one again from
// (placeholder, candidate set, caller policy) alone. A candidate is the
// KindFunction program (Meta cobol_kind=program) or KindFile copybook
// (cobol_kind=copybook) whose cobol_name_key equals <key> and that the
// caller repository's graph.Scope admits; in this pass the scope is the
// caller repository alone. A contained program (cobol_nested=true) is a
// candidate only for a CALL whose cobol_caller_program is its
// cobol_container, and such a candidate wins first. Otherwise top-level
// programs in the edge's file win, then top-level programs in other files.
// A sibling or unrelated program never reaches a contained one: COMMON is
// not observed, so fail closed. A program node without cobol_nested
// predates the flag and is never a candidate until the file is re-extracted.
// One candidate binds. Zero sets To back to the placeholder with
// cobol_unresolved_reason=missing; two or more set it back with ambiguous
// plus the sorted candidate IDs in cobol_candidate_ids. A caller node with
// no RepoPrefix never binds: its edge keeps the placeholder with
// caller_unowned.
//
// RATIONALE — a placeholder-only pass can never re-check a bound edge, so a
// second same-named program or a deleted target left a stale binding (WR-01).
// Re-deciding every edge in the repository set from its placeholder makes the
// full and incremental paths one function, so incremental equals clean by
// construction. The generic passes bind the first same-file match and lift
// bare names across repositories, which would be false deterministic
// resolution (D-09, D-11); the cobol:: namespace is therefore excluded from
// every generic path and owned here. Candidates pass through graph.Scope, the
// one fail-closed repository boundary (D-16), which never admits an empty
// repository. Keys compare by exact string equality, so the resolver carries
// no second normalization rule (D-18). The pass reads the extractor's
// cobol_nested flag and never infers nesting from the shape of a node ID or
// QualName (a literal PROGRAM-ID may itself contain '/'). Dynamic, label,
// data, idms_scope, and COPY IDMS targets are never touched (D-19, DET-04).
//
// Runs serially in ResolveAll's relative-import settle window, after
// resolveLuaRequires, with changed = the resolve scope. It writes only the
// edges whose decision differs from their current state, so a second run
// issues no write and leaves every edge byte-identical.
//
// KEYWORDS — cobol, call, copybook, reconcile, rebind, exactly-one,
// same-repo, ambiguous, missing, caller_unowned.
func (r *Resolver) reconcileCobol(changed map[string]struct{}) {
	if !r.graphHasLanguage("cobol") {
		return
	}
	programs := cobolCandidateIndex(r.nodesByKindLang(graph.KindFunction, "cobol"), "program")
	copybooks := cobolCandidateIndex(r.nodesByKindLang(graph.KindFile, "cobol"), "copybook")

	type cobolRef struct {
		e           *graph.Edge
		placeholder string
		key         string
		isProgram   bool
	}
	var refs []cobolRef
	add := func(e *graph.Edge, placeholder string) {
		if e == nil {
			return
		}
		if len(changed) > 0 {
			if _, ok := changed[graph.RepoPrefixOfID(e.From)]; !ok {
				return
			}
		}
		target, ok := strings.CutPrefix(placeholder, graph.UnresolvedMarker)
		if !ok {
			return
		}
		ref := cobolRef{e: e, placeholder: placeholder}
		switch {
		case strings.HasPrefix(target, cobolIDMSCopyTargetPrefix):
			return
		case strings.HasPrefix(target, cobolProgramTargetPrefix):
			ref.key, ref.isProgram = strings.TrimPrefix(target, cobolProgramTargetPrefix), true
		case strings.HasPrefix(target, cobolCopybookTargetPrefix):
			ref.key = strings.TrimPrefix(target, cobolCopybookTargetPrefix)
		default:
			return
		}
		if ref.key != "" {
			refs = append(refs, ref)
		}
	}

	// Edges still on a placeholder. A legacy edge without cobol_placeholder
	// uses its own unresolved target.
	for e := range r.graph.EdgesWithUnresolvedTarget() {
		if e == nil {
			continue
		}
		placeholder, _ := e.Meta[metaCobolPlaceholder].(string)
		if placeholder == "" {
			placeholder = e.To
		}
		add(e, placeholder)
	}
	// Edges already bound to a candidate. Each edge has one To, so the
	// in-edge lists are disjoint from each other and from the placeholders.
	var candidateIDs []string
	for _, index := range []map[string][]*graph.Node{programs, copybooks} {
		for _, nodes := range index {
			for _, n := range nodes {
				candidateIDs = append(candidateIDs, n.ID)
			}
		}
	}
	if len(candidateIDs) > 0 {
		for _, edges := range r.graph.GetInEdgesByNodeIDs(candidateIDs) {
			for _, e := range edges {
				if e == nil {
					continue
				}
				if placeholder, _ := e.Meta[metaCobolPlaceholder].(string); placeholder != "" {
					add(e, placeholder)
				}
			}
		}
	}

	scopes := make(map[string]graph.Scope)
	var rebound []graph.EdgeReindex
	var stamped []*graph.Edge
	for _, ref := range refs {
		e := ref.e
		index := copybooks
		if ref.isProgram {
			index = programs
		}
		d := r.decideCobol(e, ref.key, ref.isProgram, index, scopes)

		to, origin, crossRepo := ref.placeholder, e.Origin, false
		if d.bind != nil {
			to, origin, crossRepo = d.bind.ID, graph.OriginASTResolved, d.crossRepo
		}
		reason, _ := e.Meta[metaCobolUnresolvedReason].(string)
		if e.To == to && e.Origin == origin && e.CrossRepo == crossRepo &&
			reason == d.reason && cobolIDsEqual(e.Meta[metaCobolCandidateIDs], d.ids) {
			continue
		}

		oldTo, oldCrossRepo := e.To, e.CrossRepo
		e.To, e.Origin, e.CrossRepo = to, origin, crossRepo
		if d.reason == "" {
			delete(e.Meta, metaCobolUnresolvedReason)
			delete(e.Meta, metaCobolCandidateIDs)
		} else {
			if e.Meta == nil {
				e.Meta = make(map[string]any, 2)
			}
			e.Meta[metaCobolUnresolvedReason] = d.reason
			if len(d.ids) > 0 {
				e.Meta[metaCobolCandidateIDs] = d.ids
			} else {
				delete(e.Meta, metaCobolCandidateIDs)
			}
		}
		if oldTo != to || oldCrossRepo != crossRepo {
			rebound = append(rebound, graph.EdgeReindex{Edge: e, OldTo: oldTo})
		} else {
			stamped = append(stamped, e)
		}
	}

	// ReindexEdges persists the moved edges' full payload, Meta included.
	if len(rebound) > 0 {
		r.noteImportEdgeReindexes(rebound)
		r.graph.ReindexEdges(rebound)
	}
	// Meta- and Origin-only changes. The in-memory store hands out live
	// edges, so the in-place mutation above is already durable there.
	if len(stamped) > 0 {
		if p, ok := r.graph.(graph.EdgeMetaBatchPersister); ok {
			p.PersistEdgeAttributesBatch(stamped)
		}
	}
}

// cobolDecision is one edge's reconcile outcome: the single bound candidate,
// or the reason the edge stays on its placeholder and, when ambiguous, the
// sorted candidate IDs.
type cobolDecision struct {
	bind      *graph.Node
	crossRepo bool
	reason    string
	ids       []string
}

// decideCobol applies the reconcile decision rule to one edge. scopes caches
// each caller repository's allowed scope for the pass.
func (r *Resolver) decideCobol(e *graph.Edge, key string, isProgram bool, index map[string][]*graph.Node, scopes map[string]graph.Scope) cobolDecision {
	callerRepo := r.callerRepoPrefix(e)
	if callerRepo == "" {
		return cobolDecision{reason: "caller_unowned"}
	}
	scope, ok := scopes[callerRepo]
	if !ok {
		// The allowed set is the caller repository alone; 04-14 adds the
		// caller policy's ResolveFrom repositories (D-09).
		s, err := graph.NewScope("", "", []string{callerRepo})
		if err != nil {
			return cobolDecision{reason: "caller_unowned"}
		}
		scopes[callerRepo], scope = s, s
	}
	var candidates []*graph.Node
	for _, c := range index[callerRepo+"\x00"+key] {
		if scope.Allows(c) {
			candidates = append(candidates, c)
		}
	}

	if isProgram {
		caller, _ := e.Meta[metaCobolCallerProgram].(string)
		var contained, sameFile, otherFile []*graph.Node
		for _, c := range candidates {
			nested, known := c.Meta[metaCobolNested].(bool)
			switch {
			case !known:
				// Indexed before the flag existed: fail closed until reindex.
			case nested:
				if container, _ := c.Meta[metaCobolContainer].(string); caller != "" && container == caller {
					contained = append(contained, c)
				}
			case e.FilePath != "" && c.FilePath == e.FilePath:
				sameFile = append(sameFile, c)
			default:
				otherFile = append(otherFile, c)
			}
		}
		switch {
		case len(contained) > 0:
			candidates = contained
		case len(sameFile) > 0:
			candidates = sameFile
		default:
			candidates = otherFile
		}
	}

	switch len(candidates) {
	case 1:
		return cobolDecision{bind: candidates[0], crossRepo: candidates[0].RepoPrefix != callerRepo}
	case 0:
		return cobolDecision{reason: "missing"}
	}
	ids := make([]string, 0, len(candidates))
	for _, c := range candidates {
		ids = append(ids, c.ID)
	}
	sort.Strings(ids)
	return cobolDecision{reason: "ambiguous", ids: ids}
}

// cobolIDsEqual reports whether a stored cobol_candidate_ids value equals
// want. A disk store decodes the list as []any, the in-memory store keeps
// []string, and an absent value equals an empty list.
func cobolIDsEqual(v any, want []string) bool {
	switch got := v.(type) {
	case nil:
		return len(want) == 0
	case []string:
		return slices.Equal(got, want)
	case []any:
		if len(got) != len(want) {
			return false
		}
		for i, g := range got {
			if s, _ := g.(string); s != want[i] {
				return false
			}
		}
		return true
	}
	return false
}

// cobolCandidateIndex keys COBOL nodes of one cobol_kind by RepoPrefix and
// cobol_name_key, so a lookup never sees another repository's candidates.
func cobolCandidateIndex(nodes iter.Seq[*graph.Node], cobolKind string) map[string][]*graph.Node {
	index := make(map[string][]*graph.Node)
	for n := range nodes {
		if n == nil {
			continue
		}
		if kind, _ := n.Meta["cobol_kind"].(string); kind != cobolKind {
			continue
		}
		key, _ := n.Meta["cobol_name_key"].(string)
		if key == "" {
			continue
		}
		k := n.RepoPrefix + "\x00" + key
		index[k] = append(index[k], n)
	}
	return index
}
