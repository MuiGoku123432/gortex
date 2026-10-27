//go:build darwin || linux

package languages

import (
	"strconv"

	"github.com/zzet/gortex/internal/graph"
)

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
