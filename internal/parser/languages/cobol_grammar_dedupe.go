//go:build darwin || linux

package languages

import (
	"strconv"

	"github.com/zzet/gortex/internal/graph"
)

// cobolDedupeEdges collapses relationship edges (those whose Meta carries
// prov_observation_kind) that share From, To, Kind, and Line, so SQLite
// (first insert wins) and the in-memory graph (last write wins) persist the
// same edge (Pitfall 4). The survivor keeps the first occurrence's Meta and
// gains int cobol_occurrence_count; when the duplicate carries
// cobol_statement_symbol, both symbols are listed in source order in
// cobol_statement_symbols. Structural edges pass through; order is kept.
func cobolDedupeEdges(edges []*graph.Edge) []*graph.Edge {
	first := make(map[string]*graph.Edge)
	out := edges[:0:0]
	for _, e := range edges {
		if _, relation := e.Meta["prov_observation_kind"]; !relation {
			out = append(out, e)
			continue
		}
		key := e.From + "|" + e.To + "|" + string(e.Kind) + "|" + strconv.Itoa(e.Line)
		survivor, dup := first[key]
		if !dup {
			first[key] = e
			out = append(out, e)
			continue
		}
		count, _ := survivor.Meta["cobol_occurrence_count"].(int)
		if count == 0 {
			count = 1
		}
		survivor.Meta["cobol_occurrence_count"] = count + 1
		if sym, ok := e.Meta["cobol_statement_symbol"].(string); ok {
			syms, _ := survivor.Meta["cobol_statement_symbols"].([]string)
			if len(syms) == 0 {
				if own, ok := survivor.Meta["cobol_statement_symbol"].(string); ok {
					syms = append(syms, own)
				}
			}
			survivor.Meta["cobol_statement_symbols"] = append(syms, sym)
		}
	}
	return out
}
