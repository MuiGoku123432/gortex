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
}

// cobolDataIndex holds every data item of one document, so the data,
// reference, mainframe, and SQL walkers resolve data names against the same
// items.
type cobolDataIndex struct {
	items []cobolDataItem
	seen  map[string]int  // full symbol path without ordinal -> items seen
	taken map[string]bool // symbols already used by a program or an item
}

// cobolDataItems emits the data items of every named program's data-division
// sections and returns the document's data index (D-06, D-07).
//
// PURPOSE: dynamic CALL identifiers, CICS operands, SQL host variables, and
// REDEFINES/RENAMES point at these items, so they are emitted once and
// indexed once per document.
// RATIONALE: data_description observations are flat under their section,
// so the hierarchy comes from level numbers (cobolDataSection). Item IDs are
// the full group path from the 01 level under the program symbol; program
// symbols are reserved first so an 01 record named like a nested program
// takes the next free #k instead of the program's ID.
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
	return idx
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
// Nesting compares levels numerically: 01 and 77 reset the stack (77 is
// standalone), 02-49 pop while the top level is >= the new level and attach
// to the new top, 66 attaches to the current 01 record, and 88 to the
// preceding non-88 entry without popping (A8). An item with no parent
// attaches to parentID. An entry whose level is not 01-49, 66, 77, or 88
// (or that has none, such as a CONSTANT entry) gets no node and is counted
// in cobol_unleveled_entry_count on the section node: no level is guessed.
// A COPY in the section marks the innermost open item, else the section
// node, with cobol_contains_copy, since its items are absent.
func cobolDataSection(w *cobolWalk, idx *cobolDataIndex, owner, idPrefix, parentID string, section int) {
	obs := w.h.Facts.Observations
	sect := w.sections[section]
	// COPY is a grammar extra (Pitfall 3), and one after a section's last
	// entry falls outside its span, so the section reaches to the first
	// later observation that is not a COPY.
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

		name, filler := raw, !named || strings.EqualFold(raw, "FILLER")
		if !named {
			name = "FILLER"
		}
		base, memberOf, keys := idPrefix, parentID, []string(nil)
		if parent.node != nil {
			base, memberOf, keys = parent.node.QualName+"/", parent.node.ID, parent.keys
		}
		seg := cobolIDSegment(name)
		if filler {
			seg = "FILLER"
		}
		base += seg
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

		key := ""
		if !filler {
			key = cobolNameNorm(name)
		}
		idx.items = append(idx.items, cobolDataItem{owner: owner, key: key, symbol: symbol, ancestors: keys})
		frame := cobolDataFrame{level: level, node: node, keys: keys}
		if !filler {
			frame.keys = append([]string{key}, keys...)
		}
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

// cobolDataClauses records the raw clause text of data_description entry on
// node: the PIC string and USAGE value (not their keywords), the OCCURS
// bounds (one integer sets both) and DEPENDING ON name, each VALUE item, and
// the REDEFINES and RENAMES operands.
func cobolDataClauses(w *cobolWalk, node *graph.Node, entry int) {
	obs := w.h.Facts.Observations
	for _, c := range w.children[entry] {
		kids := w.children[c]
		if len(kids) == 0 {
			continue
		}
		switch obs[c].Kind {
		case "picture_clause":
			node.Meta["cobol_pic"] = w.text(kids[0])
		case "usage_clause":
			first, last := obs[kids[0]].Generated, obs[kids[len(kids)-1]].Generated
			node.Meta["cobol_usage"] = string(w.h.Generated[first.Start:last.End])
		case "occurs_clause":
			var bounds []int
			for _, k := range kids {
				switch obs[k].Kind {
				case "integer":
					if n, err := strconv.Atoi(w.text(k)); err == nil {
						bounds = append(bounds, n)
					}
				case "qualified_word":
					node.Meta["cobol_occurs_depending"] = w.text(k)
				}
			}
			if len(bounds) > 0 {
				node.Meta["cobol_occurs_min"] = bounds[0]
				node.Meta["cobol_occurs_max"] = bounds[min(1, len(bounds)-1)]
			}
		case "value_clause":
			values, _ := node.Meta["cobol_value"].([]string)
			for _, k := range kids {
				if obs[k].Kind == "value_item" {
					values = append(values, w.text(k))
				}
			}
			if len(values) > 0 {
				node.Meta["cobol_value"] = values
			}
		case "redefines_clause":
			node.Meta["cobol_redefines"] = w.text(kids[0])
		case "renames_clause":
			var names []string
			for _, k := range kids {
				if obs[k].Kind == "qualified_word" {
					names = append(names, w.text(k))
				}
			}
			if len(names) > 0 {
				node.Meta["cobol_renames"] = names
			}
		}
	}
}
