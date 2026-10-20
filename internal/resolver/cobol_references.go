package resolver

import (
	"iter"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/zzet/gortex/internal/graph"
)

// COBOL placeholder shapes this pass owns, after the unresolved:: marker.
// COPY IDMS members (`copybook::idms/…`) are dictionary members, never files
// (D-19), so they are matched before the copybook prefix and never bound.
const (
	cobolProgramTargetPrefix  = "cobol::program::"
	cobolCopybookTargetPrefix = "cobol::copybook::"
	cobolIDMSCopyTargetPrefix = "cobol::copybook::idms/"

	// Edge Meta written by this pass (D-12 reason vocabulary: missing,
	// ambiguous, caller_unowned; a COPY IDMS edge keeps the extractor's
	// idms_dictionary_member). A missing or IDMS edge's To is its
	// repository's missing-artifact placeholder node (D-11).
	metaCobolUnresolvedReason = "cobol_unresolved_reason"
	metaCobolCandidateIDs     = "cobol_candidate_ids"
	// Edge Meta written on a COPY bound by library rank (D-10): the
	// configured index of the library that holds it, and the rule name.
	metaCobolCopySearchRank = "cobol_copy_search_rank"
	metaCobolCopySearchRule = "cobol_copy_search_rule"
	// Edge Meta read by this pass: the original unresolved target the
	// extractor stamps on every literal CALL and non-IDMS COPY edge, and
	// the normalized library of a COPY ... OF/IN clause.
	metaCobolPlaceholder    = "cobol_placeholder"
	metaCobolCopyLibraryKey = "cobol_copy_library_key"

	// Program node Meta read by this pass: true on a program contained in
	// another program, stamped by the extractor from END PROGRAM nesting,
	// and the node ID of that directly containing program.
	metaCobolNested    = "cobol_nested"
	metaCobolContainer = "cobol_container"
	// Literal CALL edge Meta read by this pass: the calling program's node ID.
	metaCobolCallerProgram = "cobol_caller_program"
)

// SetCobolReconcileLock installs a mutex every COBOL reconcile pass of this
// resolver holds, so the passes of resolvers that share it (a MultiIndexer's
// master and per-repository resolvers) are serialised. nil, the default,
// takes no lock.
func (r *Resolver) SetCobolReconcileLock(mu *sync.Mutex) {
	r.cobolReconcileMu = mu
}

// ReconcileCobol re-decides every literal COBOL CALL and COPY edge whose
// caller repository is in changed, or resolves from a repository in changed
// (every edge when changed is empty), under the resolve mutex. It is the
// entry for the incremental mutation tails, so they decide each binding with
// the same code as ResolveAll.
func (r *Resolver) ReconcileCobol(changed map[string]struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reconcileCobol(changed)
}

// reconcileCobol decides every literal COBOL CALL and COPY edge whose caller
// repository is in the reconcile set, binding it to exactly one candidate,
// moving it onto its repository's missing-artifact placeholder node, or
// returning it to its unresolved placeholder. The set is changed plus every
// repository whose policy's ResolveFrom names a changed repository; an empty
// changed means every edge.
//
// PURPOSE — the extractor emits `unresolved::cobol::program::<key>` for a
// literal CALL / LINK / XCTL and `unresolved::cobol::copybook::<key>` for a
// COPY / -INC / SQL INCLUDE, with <key> already normalized under
// cobol-name-norm/1, and keeps that target in cobol_placeholder. This pass
// visits the edges still on a placeholder (EdgesWithUnresolvedTarget), the
// edges already bound to a program or copybook candidate (their in-edges
// that carry cobol_placeholder), and the edges on a missing-artifact
// placeholder node (its in-edges; a COPY IDMS edge's unresolved target is
// recovered from the node ID), and decides each one again from
// (placeholder, candidate set, caller policy) alone. A candidate is the
// KindFunction program (Meta cobol_kind=program) or KindFile copybook
// (cobol_kind=copybook) whose cobol_name_key equals <key> and that the
// caller repository's graph.Scope admits. The scope is the caller repository
// plus its policy's directed ResolveFrom list (D-09), and candidates from all
// of them form one union (D-19), so a same-repository candidate never
// shadows an allowed one. Inside the caller repository a contained program
// (cobol_nested=true) is a candidate only for a CALL whose
// cobol_caller_program is its cobol_container, and such a candidate wins
// outright. Otherwise top-level programs in the edge's file, else top-level
// programs in other files, join every top-level program of the other allowed
// repositories. A sibling or unrelated program never reaches a contained one:
// COMMON is not observed, so fail closed. A program node without
// cobol_nested predates the flag and is never a candidate until the file is
// re-extracted. When the caller policy lists copybook libraries, a COPY
// candidate ranks by the first library (configured order, narrowed to the
// OF/IN library when the edge names one) whose repository and directory hold
// it; unranked candidates drop out and the lowest rank wins (D-10), recorded
// in cobol_copy_search_rank and cobol_copy_search_rule. One candidate binds.
// Zero stamps cobol_unresolved_reason=missing and sets To to the caller
// repository's placeholder node `<repo>/cobol-missing::<type>/<key>` (D-11),
// except an EXEC SQL INCLUDE of SQLCA or SQLDA (sql_include_builtin), which
// keeps its unresolved target (D-19). Two or more set To back to the
// unresolved placeholder with ambiguous plus the sorted candidate IDs in
// cobol_candidate_ids. A COPY IDMS member is never bound (P3 D-19): it keeps
// idms_dictionary_member and targets its idms_dictionary_member placeholder
// node. A caller node with no RepoPrefix never binds and gets no placeholder
// node: its edge keeps the unresolved placeholder with caller_unowned.
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
// data, and idms_scope targets are never touched (D-19, DET-04).
//
// Runs serially in ResolveAll's relative-import settle window, after
// resolveLuaRequires, with changed = the resolve scope. It writes only the
// edges whose decision differs from their current state, so a second run
// issues no write and leaves every edge byte-identical. Every edge that
// targets a missing-artifact placeholder, unchanged ones included, is one
// desired entry (cobolExternalRef) handed to reconcileCobolFindings, which
// owns the placeholder nodes; the pass ends with reconcileCobolResources over
// the same repository set.
//
// KEYWORDS — cobol, call, copybook, reconcile, rebind, exactly-one,
// allowlist, resolve_from, union, library, search-order, ambiguous, missing,
// caller_unowned, placeholder, missing artifact, idms.
func (r *Resolver) reconcileCobol(changed map[string]struct{}) {
	if !r.graphHasLanguage("cobol") {
		return
	}
	if r.cobolReconcileMu != nil {
		r.cobolReconcileMu.Lock()
		defer r.cobolReconcileMu.Unlock()
	}
	programs := cobolCandidateIndex(r.nodesByKindLang(graph.KindFunction, "cobol"), "program")
	copybooks := cobolCandidateIndex(r.nodesByKindLang(graph.KindFile, "cobol"), "copybook")

	// Each COBOL repository's policy, read once for the pass. A non-empty
	// changed set grows by every repository that resolves from it.
	policies := make(map[string]CobolPolicy)
	var repos map[string]struct{}
	if len(changed) > 0 {
		for _, index := range []map[string][]*graph.Node{programs, copybooks} {
			for _, nodes := range index {
				for _, n := range nodes {
					if n.RepoPrefix != "" {
						r.cobolPolicyFor(policies, n.RepoPrefix)
					}
				}
			}
		}
		repos = cobolReconcileRepos(changed, policies)
	}

	var refs []cobolRef
	add := func(e *graph.Edge, placeholder string) {
		if e == nil {
			return
		}
		if repos != nil {
			if _, ok := repos[graph.RepoPrefixOfID(e.From)]; !ok {
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
			// A dictionary member: never bound, but its edge targets a
			// placeholder (P3 D-19, D-09).
			ref.key, ref.idms = strings.TrimPrefix(target, cobolCopybookTargetPrefix), true
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
	// Edges on a missing-artifact placeholder node: off EdgesWithUnresolvedTarget,
	// so they are reached through the placeholders' in-edges (Pitfall 10). A
	// COPY IDMS edge carries no cobol_placeholder; its unresolved target is
	// recovered from the placeholder ID. Edges from findings carry neither and
	// are skipped.
	var placeholderIDs []string
	for n := range r.nodesByKindLang(graph.KindResource, "cobol") {
		if n != nil && n.Meta["cobol_kind"] == cobolMissingArtifactKind {
			placeholderIDs = append(placeholderIDs, n.ID)
		}
	}
	if len(placeholderIDs) > 0 {
		for _, edges := range r.graph.GetInEdgesByNodeIDs(placeholderIDs) {
			for _, e := range edges {
				if e == nil {
					continue
				}
				placeholder, _ := e.Meta[metaCobolPlaceholder].(string)
				if form, _ := e.Meta[metaCobolCopyForm].(string); placeholder == "" && form == "idms" {
					placeholder = cobolUnresolvedFromPlaceholder(e.To)
				}
				if placeholder != "" {
					add(e, placeholder)
				}
			}
		}
	}

	scopes := make(map[string]graph.Scope)
	var rebound []graph.EdgeReindex
	var stamped []*graph.Edge
	var desired []cobolExternalRef
	for _, ref := range refs {
		e := ref.e
		var d cobolDecision
		if ref.idms {
			d = cobolDecision{reason: "idms_dictionary_member"}
		} else {
			index := copybooks
			if ref.isProgram {
				index = programs
			}
			d = r.decideCobol(e, ref, index, policies, scopes)
		}

		to, origin, crossRepo := ref.placeholder, e.Origin, false
		builtin, _ := e.Meta[metaSQLIncludeBuiltin].(bool)
		switch {
		case d.bind != nil:
			to, origin, crossRepo = d.bind.ID, graph.OriginASTResolved, d.crossRepo
		case ref.idms || (d.reason == "missing" && !builtin):
			// A target never obtained moves onto its repository's
			// placeholder node (D-11); a SQLCA/SQLDA include keeps its
			// unresolved target (D-19). Unchanged edges are collected too.
			if callerRepo := r.callerRepoPrefix(e); callerRepo != "" {
				id, artifactType, key := cobolPlaceholderTarget(callerRepo, ref)
				to = id
				desired = append(desired, cobolExternalRef{
					e: e, callerRepo: callerRepo, placeholderID: id,
					artifactType: artifactType, key: key, reason: d.reason,
				})
			}
		}
		reason, _ := e.Meta[metaCobolUnresolvedReason].(string)
		if e.To == to && e.Origin == origin && e.CrossRepo == crossRepo &&
			reason == d.reason && cobolIDsEqual(e.Meta[metaCobolCandidateIDs], d.ids) &&
			cobolRankEqual(e.Meta, d) {
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
		if d.ranked {
			if e.Meta == nil {
				e.Meta = make(map[string]any, 2)
			}
			e.Meta[metaCobolCopySearchRank] = d.rank
			e.Meta[metaCobolCopySearchRule] = cobolCopySearchRule
		} else {
			delete(e.Meta, metaCobolCopySearchRank)
			delete(e.Meta, metaCobolCopySearchRule)
		}
		if oldTo != to || oldCrossRepo != crossRepo {
			rebound = append(rebound, graph.EdgeReindex{Edge: e, OldTo: oldTo})
		} else {
			stamped = append(stamped, e)
		}
	}

	// ReindexEdges persists the moved edges' full payload, Meta included.
	// A CALL bound across repositories gets its parallel cross_repo_calls
	// edge here, the settle step CrossRepoResolver runs on its own batch, so
	// the clean track path and the incremental derived pass agree.
	if len(rebound) > 0 {
		r.noteImportEdgeReindexes(rebound)
		r.graph.ReindexEdges(rebound)
		DetectCrossRepoEdgesForReindexes(r.graph, rebound)
	}
	// Meta- and Origin-only changes. The in-memory store hands out live
	// edges, so the in-place mutation above is already durable there.
	if len(stamped) > 0 {
		if p, ok := r.graph.(graph.EdgeMetaBatchPersister); ok {
			p.PersistEdgeAttributesBatch(stamped)
		}
	}
	r.reconcileCobolFindings(repos, desired)
	r.reconcileCobolResources(repos, policies)
}

// cobolRef is one literal COBOL CALL or COPY edge the pass decides: its
// placeholder target and the normalized key after the program or copybook
// prefix (for a COPY IDMS member, idms/<kind>/<key>, and idms is set).
type cobolRef struct {
	e           *graph.Edge
	placeholder string
	key         string
	isProgram   bool
	idms        bool
}

// cobolDecision is one edge's reconcile outcome: the single bound candidate
// (with its library rank when a copybook library list decided it), or the
// reason the edge stays on its placeholder and, when ambiguous, the sorted
// candidate IDs.
type cobolDecision struct {
	bind      *graph.Node
	crossRepo bool
	ranked    bool
	rank      int
	reason    string
	ids       []string
}

// decideCobol applies the reconcile decision rule to one edge. policies
// caches each repository's policy and scopes each caller repository's
// allowed scope for the pass.
func (r *Resolver) decideCobol(e *graph.Edge, ref cobolRef, index map[string][]*graph.Node, policies map[string]CobolPolicy, scopes map[string]graph.Scope) cobolDecision {
	callerRepo := r.callerRepoPrefix(e)
	if callerRepo == "" {
		return cobolDecision{reason: "caller_unowned"}
	}
	policy := r.cobolPolicyFor(policies, callerRepo)
	scope, ok := scopes[callerRepo]
	if !ok {
		// The caller repository plus its directed ResolveFrom list (D-09).
		allowed := []string{callerRepo}
		for _, from := range policy.ResolveFrom {
			if from = strings.TrimSpace(from); from != "" {
				allowed = append(allowed, from)
			}
		}
		s, err := graph.NewScope("", "", allowed)
		if err != nil {
			return cobolDecision{reason: "caller_unowned"}
		}
		scopes[callerRepo], scope = s, s
	}
	var candidates []*graph.Node
	for _, repo := range scope.Repositories {
		for _, c := range index[repo+"\x00"+ref.key] {
			if scope.Allows(c) {
				candidates = append(candidates, c)
			}
		}
	}

	if ref.isProgram {
		caller, _ := e.Meta[metaCobolCallerProgram].(string)
		var contained, sameFile, otherFile, allowedRepo []*graph.Node
		for _, c := range candidates {
			nested, known := c.Meta[metaCobolNested].(bool)
			switch {
			case !known:
				// Indexed before the flag existed: fail closed until reindex.
			case c.RepoPrefix != callerRepo:
				// Another allowed repository: top-level programs only (D-19).
				if !nested {
					allowedRepo = append(allowedRepo, c)
				}
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
			candidates = append(sameFile, allowedRepo...)
		default:
			candidates = append(otherFile, allowedRepo...)
		}
	}

	rank := -1
	if !ref.isProgram && len(policy.CopybookLibraries) > 0 {
		candidates, rank = cobolLibraryRank(e, candidates, callerRepo, policy.CopybookLibraries)
	}

	switch len(candidates) {
	case 1:
		return cobolDecision{
			bind: candidates[0], crossRepo: candidates[0].RepoPrefix != callerRepo,
			ranked: rank >= 0, rank: rank,
		}
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

// cobolLibraryRank keeps the copybook candidates of the lowest library rank
// (D-10) and returns that rank, or -1 when no candidate is in any library. A
// candidate's rank is the configured index of the first library whose
// repository (the caller's own when Repo is empty) is the candidate's and
// whose cleaned Path equals the directory of the candidate's
// repository-relative path ("." for the root). Membership is exact directory
// equality, like a flat PDS. When the edge carries a COPY ... OF/IN library
// key, only libraries of that Name are searched.
func cobolLibraryRank(e *graph.Edge, candidates []*graph.Node, callerRepo string, libs []CobolLibrary) ([]*graph.Node, int) {
	libKey, _ := e.Meta[metaCobolCopyLibraryKey].(string)
	best := -1
	var kept []*graph.Node
	for _, c := range candidates {
		dir := path.Dir(strings.TrimPrefix(c.FilePath, c.RepoPrefix+"/"))
		for i, lib := range libs {
			if libKey != "" && lib.Name != libKey {
				continue
			}
			repo := lib.Repo
			if repo == "" {
				repo = callerRepo
			}
			if repo != c.RepoPrefix || path.Clean(lib.Path) != dir {
				continue
			}
			switch {
			case best < 0 || i < best:
				best, kept = i, []*graph.Node{c}
			case i == best:
				kept = append(kept, c)
			}
			break
		}
	}
	return kept, best
}

// cobolRankEqual reports whether an edge's stored library-rank Meta matches
// d: both keys present with d's rank and the rule when d is a library bind,
// both absent otherwise. A disk store may decode the rank as int64 or
// float64.
func cobolRankEqual(meta map[string]any, d cobolDecision) bool {
	rank, hasRank := meta[metaCobolCopySearchRank]
	rule, hasRule := meta[metaCobolCopySearchRule]
	if !d.ranked {
		return !hasRank && !hasRule
	}
	if rule != cobolCopySearchRule {
		return false
	}
	switch n := rank.(type) {
	case int:
		return n == d.rank
	case int64:
		return n == int64(d.rank)
	case float64:
		return n == float64(d.rank)
	}
	return false
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
