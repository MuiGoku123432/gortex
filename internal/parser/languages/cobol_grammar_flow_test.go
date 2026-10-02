//go:build darwin || linux

package languages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// Every fixture in this file is invented 7-column fixed-format COBOL (P2
// D-14). None may be replaced with estate-derived source: this repository is
// public.

const cobolOwnSectionFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. OWNPGM.
       PROCEDURE DIVISION.
       S1 SECTION.
       A.
           PERFORM P-X.
           PERFORM P-X OF S2.
           PERFORM P-X IN S9.
           PERFORM S2.
           PERFORM 0100.
       P-X.
           EXIT.
       S2 SECTION.
       P-X.
           EXIT.
       0100.
           EXIT.
`

const cobolThruFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. THRUPGM.
       PROCEDURE DIVISION.
       S1 SECTION.
       A.
           PERFORM A THRU C.
           PERFORM S1 THROUGH S2.
           PERFORM C THRU A.
           PERFORM S1 THRU C.
           PERFORM A THRU NOPE.
       B.
           EXIT.
       S2 SECTION.
       C.
           EXIT.
`

const cobolGoToFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. GOTOPGM.
       PROCEDURE DIVISION.
       P1.
           GO TO P2.
       P2.
           GO TO P1 P2 P1 DEPENDING ON WS-IX.
       P3.
           GO TO NOPE.
       P4.
           GO TO.
       P5.
           EXIT.
`

// cobolStatementCalls returns the calls edges from from that point back to
// the statement symbol stmt, in emission order.
func cobolStatementCalls(result *parser.ExtractionResult, from, stmt string) []*graph.Edge {
	var edges []*graph.Edge
	for _, e := range result.Edges {
		if e.From == from && e.Kind == graph.EdgeCalls && e.Meta["cobol_statement_symbol"] == stmt {
			edges = append(edges, e)
		}
	}
	return edges
}

func TestCobolGrammar_ControlFlow(t *testing.T) {
	e := NewCobolGrammarExtractor()
	own := extractCobolGrammar(t, e, "src/own.cbl", cobolOwnSectionFixture)
	const ownProc = "src/own.cbl::OWNPGM/@proc/"
	const ownA = ownProc + "S1/A"

	t.Run("own_section_first", func(t *testing.T) {
		edges := cobolStatementCalls(own, ownA, "OWNPGM/@proc/S1/A/@perform_statement_call_proc")
		require.Len(t, edges, 1)
		assert.Equal(t, ownProc+"S1/P-X", edges[0].To)
		assert.Equal(t, "perform", edges[0].Meta["cobol_flow"])
		assert.Equal(t, cobolNameNormRule, edges[0].Meta["cobol_name_norm"])
		assert.NotContains(t, edges[0].Meta, "cobol_unresolved_reason")
	})

	t.Run("qualified", func(t *testing.T) {
		of := cobolStatementCalls(own, ownA, "OWNPGM/@proc/S1/A/@perform_statement_call_proc#2")
		require.Len(t, of, 1)
		assert.Equal(t, ownProc+"S2/P-X", of[0].To)
		assert.Equal(t, "perform", of[0].Meta["cobol_flow"])

		in := cobolStatementCalls(own, ownA, "OWNPGM/@proc/S1/A/@perform_statement_call_proc#3")
		require.Len(t, in, 1)
		assert.Equal(t, "unresolved::cobol::label::src/own.cbl::OWNPGM::S9/P-X", in[0].To)
		assert.Equal(t, "label_not_found", in[0].Meta["cobol_unresolved_reason"])
	})

	t.Run("section_target", func(t *testing.T) {
		edges := cobolStatementCalls(own, ownA, "OWNPGM/@proc/S1/A/@perform_statement_call_proc#4")
		require.Len(t, edges, 1)
		assert.Equal(t, ownProc+"S2", edges[0].To)
		assert.Equal(t, "perform", edges[0].Meta["cobol_flow"])
	})

	t.Run("numeric_labels", func(t *testing.T) {
		edges := cobolStatementCalls(own, ownA, "OWNPGM/@proc/S1/A/@perform_statement_call_proc#5")
		require.Len(t, edges, 1)
		assert.Equal(t, ownProc+"S2/0100", edges[0].To)
	})

	thru := extractCobolGrammar(t, e, "src/thru.cbl", cobolThruFixture)
	const thruProc = "src/thru.cbl::THRUPGM/@proc/"
	const thruA = thruProc + "S1/A"
	const thruStmt = "THRUPGM/@proc/S1/A/@perform_statement_call_proc"
	const thruLabel = "unresolved::cobol::label::src/thru.cbl::THRUPGM::"

	t.Run("thru_paragraphs", func(t *testing.T) {
		edges := cobolStatementCalls(thru, thruA, thruStmt)
		require.Len(t, edges, 3)
		for i, to := range []string{thruProc + "S1/A", thruProc + "S1/B", thruProc + "S2/C"} {
			assert.Equal(t, to, edges[i].To)
			assert.Equal(t, "perform_thru", edges[i].Meta["cobol_flow"])
			assert.Equal(t, "A", edges[i].Meta["cobol_thru_start"])
			assert.Equal(t, "C", edges[i].Meta["cobol_thru_end"])
			assert.Equal(t, edges[0].Line, edges[i].Line)
			assert.Equal(t, "perform_statement_call_proc", edges[i].Meta["prov_observation_kind"])
			assert.NotContains(t, edges[i].Meta, "cobol_unresolved_reason")
		}
	})

	t.Run("thru_sections", func(t *testing.T) {
		edges := cobolStatementCalls(thru, thruA, thruStmt+"#2")
		require.Len(t, edges, 2)
		assert.Equal(t, thruProc+"S1", edges[0].To)
		assert.Equal(t, thruProc+"S2", edges[1].To)
		assert.Equal(t, "S1", edges[0].Meta["cobol_thru_start"])
		assert.Equal(t, "S2", edges[1].Meta["cobol_thru_end"])
	})

	t.Run("thru_invalid", func(t *testing.T) {
		for stmt, want := range map[string]struct{ to, reason, start, end string }{
			thruStmt + "#3": {thruLabel + "C::thru::A", "thru_range_invalid", "C", "A"},
			thruStmt + "#4": {thruLabel + "S1::thru::C", "thru_range_invalid", "S1", "C"},
			thruStmt + "#5": {thruLabel + "A::thru::NOPE", "label_not_found", "A", "NOPE"},
		} {
			edges := cobolStatementCalls(thru, thruA, stmt)
			require.Len(t, edges, 1, stmt)
			assert.Equal(t, want.to, edges[0].To, stmt)
			assert.Equal(t, "perform_thru", edges[0].Meta["cobol_flow"], stmt)
			assert.Equal(t, want.reason, edges[0].Meta["cobol_unresolved_reason"], stmt)
			assert.Equal(t, want.start, edges[0].Meta["cobol_thru_start"], stmt)
			assert.Equal(t, want.end, edges[0].Meta["cobol_thru_end"], stmt)
		}
	})

	t.Run("nested_isolation", func(t *testing.T) {
		nested := extractCobolGrammar(t, e, "src/nest.cbl", cobolNestedPerformFixture)
		edges := cobolStatementCalls(nested, "src/nest.cbl::OUTERPGM/INNERPGM/@proc/INNER-PARA",
			"OUTERPGM/INNERPGM/@proc/INNER-PARA/@perform_statement_call_proc")
		require.Len(t, edges, 1)
		assert.Equal(t, "unresolved::cobol::label::src/nest.cbl::OUTERPGM/INNERPGM::OUTER-PARA", edges[0].To)
		assert.Equal(t, "label_not_found", edges[0].Meta["cobol_unresolved_reason"])
	})

	gt := extractCobolGrammar(t, e, "src/goto.cbl", cobolGoToFixture)
	const gotoProc = "src/goto.cbl::GOTOPGM/@proc/"

	t.Run("goto", func(t *testing.T) {
		edges := cobolEdges(gt, gotoProc+"P1", gotoProc+"P2", graph.EdgeCalls)
		require.Len(t, edges, 1)
		assert.Equal(t, "goto", edges[0].Meta["cobol_flow"])
		assert.Equal(t, "goto_statement", edges[0].Meta["prov_observation_kind"])
		assert.Equal(t, "GOTOPGM/@proc/P1/@goto_statement", edges[0].Meta["cobol_statement_symbol"])
		assert.NotContains(t, edges[0].Meta, "cobol_goto_depending")
		assert.NotContains(t, edges[0].Meta, "cobol_unresolved_reason")
	})

	t.Run("goto_depending", func(t *testing.T) {
		edges := cobolStatementCalls(gt, gotoProc+"P2", "GOTOPGM/@proc/P2/@goto_statement")
		require.Len(t, edges, 2)
		for i, want := range []struct {
			to        string
			positions []string
		}{{gotoProc + "P1", []string{"1", "3"}}, {gotoProc + "P2", []string{"2"}}} {
			assert.Equal(t, want.to, edges[i].To)
			assert.Equal(t, "goto", edges[i].Meta["cobol_flow"])
			assert.Equal(t, true, edges[i].Meta["cobol_goto_depending"])
			assert.Equal(t, want.positions, edges[i].Meta["cobol_depending_positions"])
			assert.Equal(t, "WS-IX", edges[i].Meta["cobol_goto_depending_on"])
			assert.NotContains(t, edges[i].Meta, "cobol_occurrence_count")
		}
	})

	t.Run("goto_unresolved", func(t *testing.T) {
		edges := cobolEdges(gt, gotoProc+"P3", "unresolved::cobol::label::src/goto.cbl::GOTOPGM::NOPE", graph.EdgeCalls)
		require.Len(t, edges, 1)
		assert.Equal(t, "goto", edges[0].Meta["cobol_flow"])
		assert.Equal(t, "label_not_found", edges[0].Meta["cobol_unresolved_reason"])
	})

	t.Run("goto_alterable", func(t *testing.T) {
		assert.Empty(t, cobolStatementCalls(gt, gotoProc+"P4", "GOTOPGM/@proc/P4/@goto_statement"))
		stmt := cobolNodeByID(t, gt, gotoProc+"P4/@goto_statement")
		assert.Equal(t, true, stmt.Meta["cobol_goto_alterable"])
		assert.NotContains(t, cobolNodeByID(t, gt, gotoProc+"P1/@goto_statement").Meta, "cobol_goto_alterable")
	})

	t.Run("no_fallthrough", func(t *testing.T) {
		for _, edge := range gt.Edges {
			assert.False(t, edge.Kind == graph.EdgeCalls && edge.To == gotoProc+"P5", "fall-through edge %s", edge.From)
		}
		assert.Equal(t, 4, cobolNodeByID(t, gt, gotoProc+"P4").Meta["cobol_ordinal"])
		assert.Equal(t, 5, cobolNodeByID(t, gt, gotoProc+"P5").Meta["cobol_ordinal"])
	})
}
