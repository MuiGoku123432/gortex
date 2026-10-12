//go:build darwin || linux

package languages

import (
	"slices"

	"github.com/zzet/gortex/internal/graph"
)

// cobolDataBind emits the REDEFINES and RENAMES references of idx's items.
// They bind once every item of the owner is known; Extract's copybook branch
// calls it too, for items the copybook file owns.
func cobolDataBind(w *cobolWalk, idx *cobolDataIndex) {
	obs := w.h.Facts.Observations
	for _, it := range idx.items {
		for _, c := range w.children[it.entry] {
			var role string
			switch obs[c].Kind {
			case "redefines_clause":
				role = "redefines"
			case "renames_clause":
				role = "renames_from"
			default:
				continue
			}
			for _, k := range w.children[c] {
				if obs[k].Kind != "qualified_word" {
					continue
				}
				if name, qualifiers := w.qualifiedName(k); name != "" {
					cobolDataReference(w, idx, w.filePath+"::"+it.symbol, c, it.owner, name, qualifiers, role)
				}
				if role == "renames_from" {
					role = "renames_thru" // RENAMES A THRU B: the second operand
				}
			}
		}
	}
}

// qualifiedName returns the data name and IN/OF qualifiers (innermost
// first, as written) of qualified_word observation i: its first WORD child
// and the remaining WORD children. IN and OF are anonymous, never observed.
func (w *cobolWalk) qualifiedName(i int) (name string, qualifiers []string) {
	var words []string
	for _, c := range w.children[i] {
		if w.h.Facts.Observations[c].Kind == "WORD" {
			words = append(words, w.text(c))
		}
	}
	if len(words) == 0 {
		return "", nil
	}
	return words[0], words[1:]
}

// lookup returns the sorted symbols of owner's non-filler items named name
// under cobol-name-norm/1 whose ancestors include every qualifier, in the
// written order from inner to outer.
func (idx *cobolDataIndex) lookup(owner, name string, qualifiers []string) []string {
	key := cobolNameNorm(name)
	var matches []string
items:
	for _, it := range idx.items {
		if it.owner != owner || it.key == "" || it.key != key {
			continue
		}
		rest := it.ancestors
		for _, q := range qualifiers {
			j := slices.Index(rest, cobolNameNorm(q))
			if j < 0 {
				continue items
			}
			rest = rest[j+1:]
		}
		matches = append(matches, it.symbol)
	}
	slices.Sort(matches)
	return matches
}

// cobolDataReference appends and returns the references edge (cobol_ref_role
// role, built for observation occurrence) from node ID from to owner's item
// that name and its IN/OF qualifiers match. Zero or several matches go to
// unresolved::cobol::data::<owner>::<key>[/<qualifier key>...] with reason
// not_declared_in_program or ambiguous plus cobol_candidate_symbols (D-12).
// In a copybook (w.root set) zero matches say not_declared_in_copybook
// instead: the member, not a program, owns the name, and an including
// program may declare it, so it is never bound across files (D-06).
// It is the one data-name path the reference, mainframe, and SQL walkers use.
func cobolDataReference(w *cobolWalk, idx *cobolDataIndex, from string, occurrence int, owner, name string, qualifiers []string, role string) *graph.Edge {
	matches := idx.lookup(owner, name, qualifiers)
	target := "unresolved::cobol::data::" + owner + "::" + cobolIDSegment(cobolNameNorm(name))
	for _, q := range qualifiers {
		target += "/" + cobolIDSegment(cobolNameNorm(q))
	}
	if len(matches) == 1 {
		target = w.filePath + "::" + matches[0]
	}
	edge := w.relationEdge(from, target, graph.EdgeReferences, occurrence)
	edge.Meta["cobol_ref_role"] = role
	edge.Meta["cobol_name_norm"] = cobolNameNormRule
	switch {
	case len(matches) == 0 && w.root != "":
		edge.Meta["cobol_unresolved_reason"] = "not_declared_in_copybook"
	case len(matches) == 0:
		edge.Meta["cobol_unresolved_reason"] = "not_declared_in_program"
	case len(matches) > 1:
		edge.Meta["cobol_unresolved_reason"] = "ambiguous"
		edge.Meta["cobol_candidate_symbols"] = matches
	}
	return edge
}
