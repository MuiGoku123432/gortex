//go:build darwin || linux

package languages

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// Every fixture in this file is invented 7-column fixed-format COBOL (P2
// D-14). None may be replaced with estate-derived source: this repository is
// public.

const cobolSectionsFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SECTPGM.
       ENVIRONMENT DIVISION.
       CONFIGURATION SECTION.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A PIC X.
       LINKAGE SECTION.
       01 LK-A PIC X.
       PROCEDURE DIVISION.
       P1.
           EXIT.
`

const cobolNestedSectionsFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. OUTERPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A PIC X.
       PROCEDURE DIVISION.
       OUTER-PARA.
           STOP RUN.
       IDENTIFICATION DIVISION.
       PROGRAM-ID. INNERPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A PIC X.
       PROCEDURE DIVISION.
       INNER-PARA.
           GOBACK.
       END PROGRAM INNERPGM.
       END PROGRAM OUTERPGM.
`

const cobolStatementsFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. STMTPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A PIC X.
       PROCEDURE DIVISION.
           DISPLAY WS-A.
       P1.
           MOVE 'Y' TO WS-A
           CALL 'SUBPGM'
           MOVE 'N' TO WS-A.
       P2.
           PERFORM UNTIL WS-A = 'Y'
               MOVE 'Y' TO WS-A
           END-PERFORM.
       P3.
       S1 SECTION.
           DISPLAY WS-A.
       P4.
           EXIT.
`

const cobolFlatProcedureFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. FLATPGM.
       PROCEDURE DIVISION.
           DISPLAY 'X'.
           PERFORM NOWHERE.
           STOP RUN.
`

// cobolMemberOf returns every member_of edge whose From is id.
func cobolMemberOf(result *parser.ExtractionResult, id string) []*graph.Edge {
	var edges []*graph.Edge
	for _, e := range result.Edges {
		if e.From == id && e.Kind == graph.EdgeMemberOf {
			edges = append(edges, e)
		}
	}
	return edges
}

// assertCobolStatement checks one statement node against the D-05/D-17
// contract: KindLocal, its observation kind as Name, the document keys, its
// own range, exactly one member_of edge to container, no defines edge, and no
// statement text in Meta.
func assertCobolStatement(t *testing.T, result *parser.ExtractionResult, id, kind, container string, texts ...string) {
	t.Helper()
	n := cobolNodeByID(t, result, id)
	assert.Equal(t, graph.KindLocal, n.Kind, id)
	assert.Equal(t, kind, n.Name, id)
	assert.Equal(t, "statement", n.Meta["cobol_kind"], id)
	assert.Equal(t, kind, n.Meta["cobol_statement_kind"], id)
	for _, key := range cobolDocumentKeys {
		assert.Contains(t, n.Meta, key, id)
	}
	assertCobolRange(t, n)
	members := cobolMemberOf(result, n.ID)
	require.Len(t, members, 1, id)
	assert.Equal(t, container, members[0].To, id)
	assert.Nil(t, members[0].Meta, id)
	for _, e := range result.Edges {
		if e.Kind == graph.EdgeDefines {
			assert.NotEqual(t, n.ID, e.To, "statement %s has a defines edge", id)
		}
	}
	for key, v := range n.Meta {
		s, ok := v.(string)
		if !ok {
			continue
		}
		for _, text := range texts {
			assert.NotContains(t, s, text, "%s Meta %s carries statement text", id, key)
		}
	}
}

func TestCobolGrammar_Structure(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/sect.cbl", cobolSectionsFixture)
	const prog = "src/sect.cbl::SECTPGM"
	for _, want := range []struct{ kind, sectionKind, division string }{
		{"configuration_section", "configuration", "environment_division"},
		{"working_storage_section", "working_storage", "data_division"},
		{"linkage_section", "linkage", "data_division"},
	} {
		id := prog + "/@" + want.kind
		n := cobolNodeByID(t, result, id)
		assert.Equal(t, graph.KindPackage, n.Kind, id)
		assert.Equal(t, want.kind, n.Name, id)
		assert.Equal(t, "section", n.Meta["cobol_kind"], id)
		assert.Equal(t, want.sectionKind, n.Meta["cobol_section_kind"], id)
		assert.Equal(t, want.kind, n.Meta["prov_observation_kind"], id)
		for _, key := range cobolDocumentKeys {
			assert.Contains(t, n.Meta, key, id)
		}
		assertCobolRange(t, n)
		defines := cobolEdges(result, "src/sect.cbl", id, graph.EdgeDefines)
		require.Len(t, defines, 1, id)
		assert.Nil(t, defines[0].Meta, id)
		members := cobolMemberOf(result, id)
		require.Len(t, members, 1, id)
		assert.Equal(t, prog+"/@"+want.division, members[0].To, id)
		assert.Nil(t, members[0].Meta, id)
	}

	// Same-named sections in different programs stay distinct.
	nested := extractCobolGrammar(t, e, "src/nest.cbl", cobolNestedSectionsFixture)
	outer := cobolNodeByID(t, nested, "src/nest.cbl::OUTERPGM/@working_storage_section")
	inner := cobolNodeByID(t, nested, "src/nest.cbl::OUTERPGM/INNERPGM/@working_storage_section")
	require.Len(t, cobolMemberOf(nested, outer.ID), 1)
	assert.Equal(t, "src/nest.cbl::OUTERPGM/@data_division", cobolMemberOf(nested, outer.ID)[0].To)
	require.Len(t, cobolMemberOf(nested, inner.ID), 1)
	assert.Equal(t, "src/nest.cbl::OUTERPGM/INNERPGM/@data_division", cobolMemberOf(nested, inner.ID)[0].To)
}

func TestCobolGrammar_Statements(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/stmt.cbl", cobolStatementsFixture)
	const prog = "src/stmt.cbl::STMTPGM"
	const proc = prog + "/@proc/"
	texts := []string{"WS-A", "SUBPGM", "MOVE", "'Y'"}
	for _, want := range []struct{ id, kind, container string }{
		// Before any header: under the procedure division node.
		{prog + "/@procedure_division/@display_statement", "display_statement", prog + "/@procedure_division"},
		// Two same-kind statements in one container never merge.
		{proc + "P1/@move_statement", "move_statement", proc + "P1"},
		{proc + "P1/@call_statement", "call_statement", proc + "P1"},
		{proc + "P1/@move_statement#2", "move_statement", proc + "P1"},
		// A MOVE inside an inline PERFORM belongs to the paragraph.
		{proc + "P2/@perform_statement_loop", "perform_statement_loop", proc + "P2"},
		{proc + "P2/@move_statement", "move_statement", proc + "P2"},
		// Before a section's first paragraph: under the section.
		{proc + "S1/@display_statement", "display_statement", proc + "S1"},
		{proc + "S1/P4/@exit_statement", "exit_statement", proc + "S1/P4"},
	} {
		assertCobolStatement(t, result, want.id, want.kind, want.container, texts...)
	}

	// Source order: the second MOVE starts after the first.
	first := cobolNodeByID(t, result, proc+"P1/@move_statement")
	second := cobolNodeByID(t, result, proc+"P1/@move_statement#2")
	assert.Less(t, first.Meta["prov_start_row"].(int), second.Meta["prov_start_row"].(int))

	// A paragraph with no statements keeps its header-only range, and
	// adjacent extents never overlap.
	p3 := cobolNodeByID(t, result, proc+"P3")
	assert.Equal(t, p3.Meta["prov_start_row"], p3.Meta["prov_end_row"])
	for _, ed := range result.Edges {
		assert.False(t, ed.Kind == graph.EdgeMemberOf && ed.To == p3.ID, "%s is member_of the empty P3", ed.From)
	}
	p1 := cobolNodeByID(t, result, proc+"P1")
	p2 := cobolNodeByID(t, result, proc+"P2")
	s1 := cobolNodeByID(t, result, proc+"S1")
	assert.Less(t, p1.Meta["prov_end_row"].(int), p2.Meta["prov_start_row"].(int))
	assert.Less(t, p3.Meta["prov_end_row"].(int), s1.Meta["prov_start_row"].(int))

	// Statements of an inner program sit under that program's IDs.
	nested := extractCobolGrammar(t, e, "src/nest.cbl", cobolNestedFixture)
	assertCobolStatement(t, nested, "src/nest.cbl::OUTERPGM/@proc/OUTER-PARA/@stop_statement",
		"stop_statement", "src/nest.cbl::OUTERPGM/@proc/OUTER-PARA")
	assertCobolStatement(t, nested, "src/nest.cbl::OUTERPGM/INNERPGM/@proc/INNER-PARA/@goback_statement",
		"goback_statement", "src/nest.cbl::OUTERPGM/INNERPGM/@proc/INNER-PARA")

	// A header-free procedure division: statements under the division node,
	// relationship edges from the program.
	flat := extractCobolGrammar(t, e, "src/flat.cbl", cobolFlatProcedureFixture)
	const flatDiv = "src/flat.cbl::FLATPGM/@procedure_division"
	assertCobolStatement(t, flat, flatDiv+"/@display_statement", "display_statement", flatDiv)
	assertCobolStatement(t, flat, flatDiv+"/@perform_statement_call_proc", "perform_statement_call_proc", flatDiv)
	assertCobolStatement(t, flat, flatDiv+"/@stop_statement", "stop_statement", flatDiv)
	calls := cobolEdges(flat, "src/flat.cbl::FLATPGM",
		"unresolved::cobol::label::src/flat.cbl::FLATPGM::NOWHERE", graph.EdgeCalls)
	require.Len(t, calls, 1)

	// No statement node is ever created outside a procedure division.
	for _, n := range result.Nodes {
		if n.Kind == graph.KindLocal {
			assert.True(t, strings.Contains(n.ID, "/@proc/") || strings.Contains(n.ID, "/@procedure_division/"), n.ID)
		}
	}

	// An empty source still yields only the file node and no edges.
	empty, err := e.Extract("src/empty.cbl", nil)
	require.NoError(t, err)
	require.Len(t, empty.Nodes, 1)
	assert.Empty(t, empty.Edges)
}
