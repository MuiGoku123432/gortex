//go:build darwin || linux

package languages

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/zzet/gortex/internal/graph"
)

// cobolStatementKinds holds every observation kind that becomes a statement
// node: each member of the pinned fork grammar's _statement choice
// (grammar.js, tree-sitter-cobol-upgrade f9eaf99) except the ERROR alias of
// _idms_invalid_update_statement, plus perform_statement_loop (an inline
// PERFORM, reached through _start_handler) and copy_statement (a grammar
// extra).
var cobolStatementKinds = map[string]bool{
	"accept_statement": true, "add_statement": true, "allocate_statement": true,
	"alter_statement": true, "call_statement": true, "cancel_statement": true,
	"close_statement": true, "continue_statement": true, "compute_statement": true,
	"delete_statement": true, "display_statement": true, "divide_statement": true,
	"exec_cics_statement": true, "exec_sql_statement": true, "exec_sql_include": true,
	"exit_statement": true, "goback_statement": true, "goto_statement": true,
	"idms_accept_statement": true, "idms_navigation_statement": true,
	"idms_session_statement": true, "idms_update_statement": true,
	"initialize_statement": true, "inspect_statement": true, "merge_statement": true,
	"move_statement": true, "multiply_statement": true, "open_statement": true,
	"perform_statement_call_proc": true, "read_statement": true, "release_statement": true,
	"return_statement": true, "rewrite_statement": true, "search_statement": true,
	"set_statement": true, "sort_statement": true, "start_statement": true,
	"stop_statement": true, "string_statement": true, "subtract_statement": true,
	"unstring_statement": true, "use_statement": true, "write_statement": true,
	"next_sentence_statement": true,
	"perform_statement_loop":  true,
	"copy_statement":          true,
}

// cobolStatements emits and registers one KindLocal node per named
// statement inside a named program's procedure division (D-05, D-17).
//
// PURPOSE: every statement occurrence is addressable, so relationship edges
// can point back to the statement they came from.
// RATIONALE: the node symbol is <container>/@<kind>, with #k for the k-th
// same-kind statement in that container in source order, so no source text
// enters an ID (T-03-11). The only structural edge is member_of to the
// header container that governs the statement (paragraph, else section,
// else procedure division, which is the file node in a copybook), never an
// enclosing statement; there is no
// defines edge, and KindLocal keeps statements out of BM25 search
// (T-03-10). Meta carries identities, kinds, and coordinates, never
// statement text (T-03-09).
// KEYWORDS: cobol, statement, KindLocal, member_of, cobol_statement_kind.
func cobolStatements(w *cobolWalk) {
	obs := w.h.Facts.Observations
	var divisions []int // named programs' procedure divisions
	for d, o := range obs {
		if o.Kind == "procedure_division" && w.container[d] != "" {
			divisions = append(divisions, d)
		}
	}
	// A COPY is a grammar extra whose Parent is arbitrary (Pitfall 3), so it
	// is placed by its generated start: in the procedure division whose span
	// contains it, under the container of that division's last other child
	// starting at or before it (else the division itself). Outside every
	// procedure division it gets no statement node.
	for i, o := range obs {
		if o.Kind != "copy_statement" {
			continue
		}
		start := o.Generated.Start
		container, flow := "", ""
		for _, d := range divisions {
			if start < obs[d].Generated.Start || start >= obs[d].Generated.End {
				continue
			}
			container, flow = w.container[d], w.flowFrom[d]
			for _, c := range w.children[d] {
				if c != i && obs[c].Generated.Start <= start {
					container, flow = w.container[c], w.flowFrom[c]
				}
			}
		}
		for j := i; j <= w.last[i]; j++ {
			w.container[j], w.flowFrom[j] = container, flow
		}
	}

	var stmts []int
	for i, o := range obs {
		if cobolStatementKinds[o.Kind] && w.container[i] != "" {
			stmts = append(stmts, i)
		}
	}
	slices.SortStableFunc(stmts, func(a, b int) int {
		return cmp.Compare(obs[a].Generated.Start, obs[b].Generated.Start)
	})
	seen := make(map[string]int) // statement symbol without ordinal -> count
	for _, i := range stmts {
		kind := obs[i].Kind
		symbol := w.container[i] + "/@" + kind
		seen[symbol]++
		if k := seen[symbol]; k >= 2 {
			symbol += "#" + strconv.Itoa(k)
		}
		node := w.symbolNode(symbol, kind, graph.KindLocal, "statement", i)
		node.Meta["cobol_statement_kind"] = kind
		cobolStampRange(node, obs[i].Original)
		w.memberOf(node.ID, w.id(w.container[i]), node.StartLine)
		w.statement[i] = node
	}
}
