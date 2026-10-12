package resolver

import (
	"iter"
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

	// Program node Meta read by this pass: true on a program contained in
	// another program, stamped by the extractor from END PROGRAM nesting.
	metaCobolNested = "cobol_nested"
)

// resolveCobolReferences binds literal COBOL program calls and copybook
// references to exactly one candidate in the caller's repository.
//
// PURPOSE — the extractor emits `unresolved::cobol::program::<key>` for a
// literal CALL / LINK / XCTL and `unresolved::cobol::copybook::<key>` for a
// COPY / -INC / SQL INCLUDE, with <key> already normalized under
// cobol-name-norm/1. This pass lands each one on the single KindFunction
// program (Meta cobol_kind=program) or KindFile copybook (cobol_kind=copybook)
// whose cobol_name_key equals <key> and whose RepoPrefix is the caller's.
// Program candidates sharing the edge's file win first, so nested and
// contained programs resolve within the file. Otherwise only top-level
// programs (cobol_nested not true) in other files are candidates: a
// contained program is never bound from another file. Zero candidates stamps
// cobol_unresolved_reason=missing; two or more stamps ambiguous plus the
// sorted candidate IDs in cobol_candidate_ids, and nothing is picked.
//
// RATIONALE — the generic passes bind the first same-file match and lift bare
// names across repositories, which would be false deterministic resolution
// (D-09, D-11). The cobol:: namespace is therefore excluded from every generic
// path and owned here. Keys compare by exact string equality, so the resolver
// carries no second normalization rule (D-18). COBOL scoping makes a
// contained program callable only from inside its container, so a separately
// compiled caller cannot reach it; the pass reads the extractor's
// cobol_nested flag for this and never infers nesting from the shape of a
// node ID or QualName (a literal PROGRAM-ID may itself contain '/'). Dynamic,
// label, data, idms_scope, and COPY IDMS targets are never touched (D-19,
// DET-04).
//
// Runs serially in ResolveAll's relative-import settle window, after
// resolveLuaRequires. Re-running it is idempotent: a rebind clears any stale
// reason and candidate IDs, and a stamp rewrites the same values.
//
// KEYWORDS — cobol, call, copybook, exactly-one, same-repo, ambiguous, missing.
func (r *Resolver) resolveCobolReferences() {
	if !r.graphHasLanguage("cobol") {
		return
	}
	programs := cobolCandidateIndex(r.nodesByKindLang(graph.KindFunction, "cobol"), "program")
	copybooks := cobolCandidateIndex(r.nodesByKindLang(graph.KindFile, "cobol"), "copybook")

	var rebound []graph.EdgeReindex
	var stamped []*graph.Edge
	for e := range r.graph.EdgesWithUnresolvedTarget() {
		if e == nil {
			continue
		}
		target := graph.UnresolvedName(e.To)
		var index map[string][]*graph.Node
		var key string
		isProgram := false
		switch {
		case strings.HasPrefix(target, cobolIDMSCopyTargetPrefix):
			continue
		case strings.HasPrefix(target, cobolProgramTargetPrefix):
			index, key, isProgram = programs, strings.TrimPrefix(target, cobolProgramTargetPrefix), true
		case strings.HasPrefix(target, cobolCopybookTargetPrefix):
			index, key = copybooks, strings.TrimPrefix(target, cobolCopybookTargetPrefix)
		default:
			continue
		}
		if key == "" || !r.edgeFromInScope(e.From) {
			continue
		}

		candidates := index[r.callerRepoPrefix(e)+"\x00"+key]
		if isProgram {
			var sameFile, topLevel []*graph.Node
			for _, c := range candidates {
				if e.FilePath != "" && c.FilePath == e.FilePath {
					sameFile = append(sameFile, c)
				} else if nested, _ := c.Meta[metaCobolNested].(bool); !nested {
					topLevel = append(topLevel, c)
				}
			}
			candidates = sameFile
			if len(candidates) == 0 {
				candidates = topLevel
			}
		}

		if len(candidates) == 1 {
			oldTo := e.To
			e.To = candidates[0].ID
			e.Origin = graph.OriginASTResolved
			delete(e.Meta, metaCobolUnresolvedReason)
			delete(e.Meta, metaCobolCandidateIDs)
			rebound = append(rebound, graph.EdgeReindex{Edge: e, OldTo: oldTo})
			continue
		}
		if e.Meta == nil {
			e.Meta = make(map[string]any, 2)
		}
		if len(candidates) == 0 {
			e.Meta[metaCobolUnresolvedReason] = "missing"
			delete(e.Meta, metaCobolCandidateIDs)
		} else {
			ids := make([]string, 0, len(candidates))
			for _, c := range candidates {
				ids = append(ids, c.ID)
			}
			sort.Strings(ids)
			e.Meta[metaCobolUnresolvedReason] = "ambiguous"
			e.Meta[metaCobolCandidateIDs] = ids
		}
		stamped = append(stamped, e)
	}

	// ReindexEdges persists the rebound edges' full payload, Meta included.
	if len(rebound) > 0 {
		r.noteImportEdgeReindexes(rebound)
		r.graph.ReindexEdges(rebound)
	}
	// Meta-only stamps. The in-memory store hands out live edges, so the
	// in-place mutation above is already durable there.
	if len(stamped) > 0 {
		if p, ok := r.graph.(graph.EdgeMetaBatchPersister); ok {
			p.PersistEdgeAttributesBatch(stamped)
		}
	}
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
