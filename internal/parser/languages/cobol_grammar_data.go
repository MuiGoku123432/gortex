//go:build darwin || linux

package languages

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// cobolDataItem is one data item registered in a cobolDataIndex.
type cobolDataItem struct {
	owner     string   // owner ID that unresolved data targets name
	key       string   // cobolNameNorm of the name; "" for a filler
	symbol    string   // QualName (owner-relative node symbol)
	ancestors []string // name keys of the non-filler ancestor items, innermost first
	entry     int      // data_description observation index
}

// cobolDataIndex holds every data item of one document, so every walker
// resolves data names against the same items.
type cobolDataIndex struct {
	items []cobolDataItem
	seen  map[string]int  // full symbol path without ordinal -> items seen
	taken map[string]bool // symbols already used by a program or an item
}

// cobolDataItems emits the data items of every named program's data-division
// sections and returns the document's data index (D-06, D-07).
//
// PURPOSE: dynamic CALL identifiers, CICS operands, SQL host variables, and
// REDEFINES/RENAMES point at these items, so they are emitted and indexed
// once per document.
// RATIONALE: data_description observations are flat under their section,
// so nesting comes from level numbers (cobolDataSection). Program symbols
// are reserved first, so an 01 record named like a nested program takes the
// next free #k instead of the program's ID.
// KEYWORDS: cobol, data item, level number, FILLER, working-storage, linkage.
func cobolDataItems(w *cobolWalk) *cobolDataIndex {
	obs := w.h.Facts.Observations
	idx := &cobolDataIndex{seen: make(map[string]int), taken: make(map[string]bool)}
	for _, prog := range w.programs {
		idx.taken[prog] = true
	}
	for s := range obs {
		sect := w.sections[s]
		p, ok := w.parent(s)
		if sect == nil || !ok || obs[p].Kind != "data_division" {
			continue
		}
		prog := w.programSymbol(s)
		cobolDataSection(w, idx, w.filePath+"::"+prog, prog+"/", sect.ID, s)
	}
	// REDEFINES and RENAMES bind once every item of the owner is known.
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
	return idx
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
	case len(matches) == 0:
		edge.Meta["cobol_unresolved_reason"] = "not_declared_in_program"
	case len(matches) > 1:
		edge.Meta["cobol_unresolved_reason"] = "ambiguous"
		edge.Meta["cobol_candidate_symbols"] = matches
	}
	return edge
}

// cobolDataFrame is an emitted item a later entry can attach to.
type cobolDataFrame struct {
	level int
	node  *graph.Node
	keys  []string // ancestor keys for this item's children, innermost first
}

// cobolDataSection emits the items of one data section in source order.
// owner is the owner ID unresolved data targets name, idPrefix the symbol
// prefix of 01 and 77 items (such as "PROG/"), and parentID their member_of
// target ("" for none).
//
// Levels compare numerically: 01 and 77 reset the stack (77 is standalone),
// 02-49 pop while the top level is >= the new level and attach to the new
// top, 66 attaches to the current 01 record, and 88 to the preceding non-88
// entry without popping (A8); an item with no parent attaches to parentID.
// An entry with no level in 01-49, 66, 77, or 88 (a CONSTANT entry has
// none) gets no node and is counted in cobol_unleveled_entry_count on the
// section node. A COPY marks the innermost open item, else the section
// node, with cobol_contains_copy, since its items are absent.
func cobolDataSection(w *cobolWalk, idx *cobolDataIndex, owner, idPrefix, parentID string, section int) {
	obs := w.h.Facts.Observations
	sect := w.sections[section]
	// COPY is a grammar extra (Pitfall 3): one after the last entry falls
	// outside the section span, so the span runs to the next non-COPY.
	start, end, open := obs[section].Generated.Start, obs[section].Generated.End, true
	for j := w.last[section] + 1; j < len(obs) && open; j = w.last[j] + 1 {
		if obs[j].Kind != "copy_statement" {
			end, open = obs[j].Generated.Start, false
		} else if obs[j].Generated.End > end {
			end = obs[j].Generated.End
		}
	}
	var entries []int
	for i, o := range obs {
		inside := i > section && i <= w.last[section]
		copied := o.Kind == "copy_statement" && o.Generated.Start >= start && o.Generated.Start < end
		if copied || (inside && o.Kind == "data_description") {
			entries = append(entries, i)
		}
	}
	slices.SortStableFunc(entries, func(a, b int) int {
		return cmp.Compare(obs[a].Generated.Start, obs[b].Generated.Start)
	})

	var stack []cobolDataFrame
	var prev cobolDataFrame // the last non-88 item
	unleveled := 0
	for _, i := range entries {
		if obs[i].Kind == "copy_statement" {
			target := sect
			if len(stack) > 0 {
				target = stack[len(stack)-1].node
			}
			if target != nil {
				target.Meta["cobol_contains_copy"] = true
			}
			continue
		}
		level, raw, named := -1, "", false
		for _, c := range w.children[i] {
			switch obs[c].Kind {
			case "level_number":
				if n, err := strconv.Atoi(w.text(c)); err == nil {
					level = n
				}
			case "entry_name":
				raw, named = w.text(c), true
			}
		}
		if (level < 1 || level > 49) && level != 66 && level != 77 && level != 88 {
			unleveled++
			continue
		}
		var parent cobolDataFrame
		switch {
		case level == 1 || level == 77:
			stack = stack[:0]
		case level == 66:
			if len(stack) > 0 {
				parent = stack[0]
			}
		case level == 88:
			parent = prev
		default:
			for len(stack) > 0 && stack[len(stack)-1].level >= level {
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 {
				parent = stack[len(stack)-1]
			}
		}

		name, seg, filler := raw, cobolIDSegment(raw), !named || strings.EqualFold(raw, "FILLER")
		if !named {
			name = "FILLER"
		}
		if filler {
			seg = "FILLER"
		}
		base, memberOf, keys := idPrefix+seg, parentID, []string(nil)
		if parent.node != nil {
			base, memberOf, keys = parent.node.QualName+"/"+seg, parent.node.ID, parent.keys
		}
		// FILLER is always FILLER#k; any other repeated path gets #k, k >= 2.
		// The loop skips a #k already taken by a program symbol.
		var symbol string
		for symbol == "" || idx.taken[symbol] {
			idx.seen[base]++
			symbol = base
			if k := idx.seen[base]; filler || k >= 2 {
				symbol += "#" + strconv.Itoa(k)
			}
		}
		idx.taken[symbol] = true

		kind := graph.KindField
		if level == 1 || level == 77 {
			kind = graph.KindVariable
		}
		node := w.symbolNode(symbol, name, kind, "data_item", i)
		node.Meta["cobol_level"] = level
		node.Meta["cobol_name_raw"] = raw
		cobolDataClauses(w, node, i)
		cobolStampRange(node, obs[i].Original)
		w.defines(node.ID, node.StartLine)
		if memberOf != "" {
			w.memberOf(node.ID, memberOf, node.StartLine)
		}

		key, frame := "", cobolDataFrame{level: level, node: node, keys: keys}
		if !filler {
			key = cobolNameNorm(name)
			frame.keys = append([]string{key}, keys...)
		}
		idx.items = append(idx.items, cobolDataItem{owner: owner, key: key, symbol: symbol, ancestors: keys, entry: i})
		if level != 88 {
			prev = frame
		}
		if level <= 49 {
			stack = append(stack, frame)
		}
	}
	if unleveled > 0 && sect != nil {
		sect.Meta["cobol_unleveled_entry_count"] = unleveled
	}
}
