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

const cobolDivisionsFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. DIVPGM.
       ENVIRONMENT DIVISION.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A PIC X.
       PROCEDURE DIVISION.
       P1.
           DISPLAY WS-A
           MOVE 'Y' TO WS-A.
       S1 SECTION.
       P2.
           EXIT.
       P2.
           EXIT.
`

const cobolPerformFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. PERFPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           PERFORM work-para.
           PERFORM NO-SUCH-PARA.
           PERFORM DUP-PARA.
           STOP RUN.
       WORK-PARA.
           EXIT.
       S1 SECTION.
       DUP-PARA.
           EXIT.
       S2 SECTION.
       DUP-PARA.
           EXIT.
`

const cobolNestedPerformFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. OUTERPGM.
       PROCEDURE DIVISION.
       OUTER-PARA.
           STOP RUN.
       IDENTIFICATION DIVISION.
       PROGRAM-ID. INNERPGM.
       PROCEDURE DIVISION.
       INNER-PARA.
           PERFORM OUTER-PARA.
           GOBACK.
       END PROGRAM INNERPGM.
       END PROGRAM OUTERPGM.
`

const cobolDedupeFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. DUPPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           PERFORM WORK-PARA. PERFORM WORK-PARA.
           PERFORM WORK-PARA.
           STOP RUN.
       WORK-PARA.
           EXIT.
`

func cobolNodeByID(t *testing.T, result *parser.ExtractionResult, id string) *graph.Node {
	t.Helper()
	for _, n := range result.Nodes {
		if n.ID == id {
			return n
		}
	}
	require.Failf(t, "no node", "id %s", id)
	return nil
}

func cobolEdges(result *parser.ExtractionResult, from, to string, kind graph.EdgeKind) []*graph.Edge {
	var edges []*graph.Edge
	for _, e := range result.Edges {
		if e.From == from && e.To == to && e.Kind == kind {
			edges = append(edges, e)
		}
	}
	return edges
}

func TestCobolGrammar_Divisions(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/div.cbl", cobolDivisionsFixture)
	for _, division := range []string{"identification", "environment", "data", "procedure"} {
		id := "src/div.cbl::DIVPGM/@" + division + "_division"
		n := cobolNodeByID(t, result, id)
		assert.Equal(t, graph.KindPackage, n.Kind, id)
		assert.Equal(t, "division", n.Meta["cobol_kind"], id)
		assert.Equal(t, division, n.Meta["cobol_division"], id)
		assertCobolRange(t, n)
		defines := cobolEdges(result, "src/div.cbl", id, graph.EdgeDefines)
		require.Len(t, defines, 1, id)
		assert.Nil(t, defines[0].Meta, id)
		members := cobolEdges(result, id, "src/div.cbl::DIVPGM", graph.EdgeMemberOf)
		require.Len(t, members, 1, id)
		assert.Nil(t, members[0].Meta, id)
		assert.Equal(t, graph.OriginASTResolved, members[0].Origin, id)
	}
}

func TestCobolGrammar_Paragraphs(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/div.cbl", cobolDivisionsFixture)
	const proc = "src/div.cbl::DIVPGM/@proc/"
	for i, want := range []struct{ id, name, kind, parent string }{
		{proc + "P1", "P1", "paragraph", "src/div.cbl::DIVPGM/@procedure_division"},
		{proc + "S1", "S1", "section", "src/div.cbl::DIVPGM/@procedure_division"},
		{proc + "S1/P2", "P2", "paragraph", proc + "S1"},
		{proc + "S1/P2#2", "P2", "paragraph", proc + "S1"},
	} {
		n := cobolNodeByID(t, result, want.id)
		assert.Equal(t, graph.KindFunction, n.Kind, want.id)
		assert.Equal(t, want.name, n.Name, want.id)
		assert.Equal(t, want.kind, n.Meta["cobol_kind"], want.id)
		assert.Equal(t, i+1, n.Meta["cobol_ordinal"], want.id)
		assert.Equal(t, true, n.Meta["prov_range_exact"], want.id)
		assertCobolRange(t, n)
		require.Len(t, cobolEdges(result, "src/div.cbl", want.id, graph.EdgeDefines), 1, want.id)
		require.Len(t, cobolEdges(result, want.id, want.parent, graph.EdgeMemberOf), 1, want.id)
	}
	assert.Equal(t, "procedure", cobolNodeByID(t, result, proc+"S1").Meta["cobol_section_kind"])

	// Pitfall 1: P1 runs from its header to the end of its last statement.
	h := cobolTestHandoff(t, e, "src/div.cbl", cobolDivisionsFixture)
	moveEnd := -1
	for _, o := range h.Facts.Observations {
		if o.Kind == "move_statement" {
			moveEnd = int(o.Original.End.Row)
		}
	}
	require.GreaterOrEqual(t, moveEnd, 0)
	p1 := cobolNodeByID(t, result, proc+"P1")
	assert.Equal(t, moveEnd, p1.Meta["prov_end_row"])
	assert.Less(t, p1.Meta["prov_start_row"].(int), moveEnd)

	nested := extractCobolGrammar(t, e, "src/nest.cbl", cobolNestedFixture)
	cobolNodeByID(t, nested, "src/nest.cbl::OUTERPGM/@proc/OUTER-PARA")
	cobolNodeByID(t, nested, "src/nest.cbl::OUTERPGM/INNERPGM/@proc/INNER-PARA")

	lower := extractCobolGrammar(t, e, "src/demo.cbl", cobolLowercaseFixture)
	assert.Equal(t, "main-para", cobolNodeByID(t, lower, "src/demo.cbl::demopgm/@proc/main-para").Name)
}

func TestCobolGrammar_PerformEdge(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/perf.cbl", cobolPerformFixture)
	const main = "src/perf.cbl::PERFPGM/@proc/MAIN-PARA"

	// A mixed-case label matches the upper-case paragraph (cobol-name-norm/1).
	resolved := cobolEdges(result, main, "src/perf.cbl::PERFPGM/@proc/WORK-PARA", graph.EdgeCalls)
	require.Len(t, resolved, 1)
	edge := resolved[0]
	for _, key := range append(append([]string{}, cobolDocumentKeys...),
		"prov_observation_kind", "prov_affected", "prov_start_row", "prov_start_column",
		"prov_end_row", "prov_end_column", "prov_start_byte", "prov_end_byte", "prov_range_exact") {
		assert.Contains(t, edge.Meta, key)
	}
	assert.Equal(t, "perform_statement_call_proc", edge.Meta["prov_observation_kind"])
	assert.Equal(t, "perform", edge.Meta["cobol_flow"])
	assert.Equal(t, cobolNameNormRule, edge.Meta["cobol_name_norm"])
	assert.Equal(t, edge.Meta["prov_start_row"].(int)+1, edge.Line)
	assert.Equal(t, graph.OriginASTResolved, edge.Origin)
	assert.InDelta(t, 1.0, edge.Confidence, 0)
	assert.Equal(t, "EXTRACTED", edge.ConfidenceLabel)

	missing := cobolEdges(result, main, "unresolved::cobol::label::src/perf.cbl::PERFPGM::NO-SUCH-PARA", graph.EdgeCalls)
	require.Len(t, missing, 1)
	assert.Equal(t, "label_not_found", missing[0].Meta["cobol_unresolved_reason"])

	ambiguous := cobolEdges(result, main, "unresolved::cobol::label::src/perf.cbl::PERFPGM::DUP-PARA", graph.EdgeCalls)
	require.Len(t, ambiguous, 1)
	assert.Equal(t, "ambiguous", ambiguous[0].Meta["cobol_unresolved_reason"])
	assert.Equal(t, []string{"PERFPGM/@proc/S1/DUP-PARA", "PERFPGM/@proc/S2/DUP-PARA"},
		ambiguous[0].Meta["cobol_candidate_symbols"])

	// Labels never resolve across programs, nested programs included.
	nested := extractCobolGrammar(t, e, "src/nest.cbl", cobolNestedPerformFixture)
	inner := cobolEdges(nested, "src/nest.cbl::OUTERPGM/INNERPGM/@proc/INNER-PARA",
		"unresolved::cobol::label::src/nest.cbl::OUTERPGM/INNERPGM::OUTER-PARA", graph.EdgeCalls)
	require.Len(t, inner, 1)
	assert.Equal(t, "label_not_found", inner[0].Meta["cobol_unresolved_reason"])
	assert.Empty(t, cobolEdges(nested, "src/nest.cbl::OUTERPGM/INNERPGM/@proc/INNER-PARA",
		"src/nest.cbl::OUTERPGM/@proc/OUTER-PARA", graph.EdgeCalls))
}

func TestCobolGrammar_EdgeDedupe(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/dup.cbl", cobolDedupeFixture)
	edges := cobolEdges(result, "src/dup.cbl::DUPPGM/@proc/MAIN-PARA", "src/dup.cbl::DUPPGM/@proc/WORK-PARA", graph.EdgeCalls)
	require.Len(t, edges, 2, "same line collapses, a different line stays distinct")
	assert.Equal(t, 2, edges[0].Meta["cobol_occurrence_count"])
	assert.NotContains(t, edges[1].Meta, "cobol_occurrence_count")
	assert.Less(t, edges[0].Line, edges[1].Line)
}

func TestCobolGrammar_NameNorm(t *testing.T) {
	for raw, want := range map[string]string{
		"main-para":    "MAIN-PARA",
		`'sub''pgm'`:   "SUB'PGM",
		`"x"`:          "X",
		"caf\xc3\xa9":  "CAF\xc3\xa9",
		`'unbalanced"`: `'UNBALANCED"`,
	} {
		assert.Equal(t, want, cobolNameNorm(raw), raw)
	}
	assert.Equal(t, "a%2Fb%3Ac%23d%25e", cobolIDSegment("a/b:c#d%e"))
	assert.Equal(t, "MAIN-PARA", cobolIDSegment("MAIN-PARA"))
	assert.Equal(t, "a%09b%7F", cobolIDSegment("a\tb\x7f"))
}
