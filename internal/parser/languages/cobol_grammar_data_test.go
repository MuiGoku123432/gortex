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

const cobolDataItemsFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. DATAPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-REC.
          05 WS-DATE.
             10 WS-YY PIC 9(2).
             10 WS-MM PIC 9(2).
          05 FILLER PIC X(3).
          05 WS-FLAG PIC X VALUE 'Y'.
             88 FLAG-ON VALUE 'Y'.
          05 FILLER PIC X.
          05 WS-TAB OCCURS 1 TO 10 TIMES DEPENDING ON WS-CNT PIC X.
          05 PIC X(2).
       77 WS-CNT PIC 9(4) COMP-3.
       78 WS-ODD VALUE 5.
       LINKAGE SECTION.
       01 LK-REC PIC X(10).
       PROCEDURE DIVISION.
       P1.
           EXIT.
`

const cobolNestedDataFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. OUTERPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 INNERPGM PIC X.
       PROCEDURE DIVISION.
       OUTER-PARA.
           STOP RUN.
       IDENTIFICATION DIVISION.
       PROGRAM-ID. INNERPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-I PIC X.
       PROCEDURE DIVISION.
       INNER-PARA.
           GOBACK.
       END PROGRAM INNERPGM.
       END PROGRAM OUTERPGM.
`

const cobolDataRefsFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. REFPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-REC.
          05 WS-A PIC X.
          05 WS-B PIC X.
       66 WS-RN RENAMES WS-A THRU WS-B.
       01 WS-ALT REDEFINES WS-REC PIC X(2).
       01 WS-LOW REDEFINES ws-rec PIC X(2).
       01 WS-BAD REDEFINES WS-NONE PIC X(2).
       01 G1.
          05 X PIC X.
       01 G2.
          05 X PIC X.
          05 Y REDEFINES X PIC X.
          05 Z REDEFINES X OF G2 PIC X.
          05 Q REDEFINES X OF G9 PIC X.
       PROCEDURE DIVISION.
       P1.
           EXIT.
`

const cobolDataCopyFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. CPYPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CPYW.
       01 WS-REC.
          COPY CPYA.
          05 WS-B PIC X.
       01 WS-T.
          05 WS-TA PIC X.
       COPY CPYT.
       LINKAGE SECTION.
       01 LK-REC PIC X.
       COPY CPYE.
       PROCEDURE DIVISION.
       P1.
           EXIT.
`

// cobolDataItemIDs returns the IDs of every data-item node in emission order.
func cobolDataItemIDs(result *parser.ExtractionResult) []string {
	var ids []string
	for _, n := range result.Nodes {
		if n.Meta["cobol_kind"] == "data_item" {
			ids = append(ids, n.ID)
		}
	}
	return ids
}

// assertCobolDataItem checks one data-item node: kind, Name, cobol_kind,
// level, the document keys, an exact range, one defines edge from the file,
// and exactly one member_of edge to parent.
func assertCobolDataItem(t *testing.T, result *parser.ExtractionResult, id string, kind graph.NodeKind, name string, level int, parent string) *graph.Node {
	t.Helper()
	n := cobolNodeByID(t, result, id)
	assert.Equal(t, kind, n.Kind, id)
	assert.Equal(t, name, n.Name, id)
	assert.Equal(t, "data_item", n.Meta["cobol_kind"], id)
	assert.Equal(t, level, n.Meta["cobol_level"], id)
	assert.Equal(t, "data_description", n.Meta["prov_observation_kind"], id)
	for _, key := range cobolDocumentKeys {
		assert.Contains(t, n.Meta, key, id)
	}
	assertCobolRange(t, n)
	assert.Len(t, cobolEdges(result, n.FilePath, n.ID, graph.EdgeDefines), 1, id)
	members := cobolMemberOf(result, n.ID)
	require.Len(t, members, 1, id)
	assert.Equal(t, parent, members[0].To, id)
	return n
}

func TestCobolGrammar_DataItems(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/data.cbl"
	result := extractCobolGrammar(t, e, path, cobolDataItemsFixture)
	const p = path + "::DATAPGM"
	const rec = p + "/WS-REC"

	assert.Equal(t, []string{
		rec, rec + "/WS-DATE", rec + "/WS-DATE/WS-YY", rec + "/WS-DATE/WS-MM",
		rec + "/FILLER#1", rec + "/WS-FLAG", rec + "/WS-FLAG/FLAG-ON", rec + "/FILLER#2",
		rec + "/WS-TAB", rec + "/FILLER#3", p + "/WS-CNT", p + "/LK-REC",
	}, cobolDataItemIDs(result))

	ws, lk := p+"/@working_storage_section", p+"/@linkage_section"
	recNode := assertCobolDataItem(t, result, rec, graph.KindVariable, "WS-REC", 1, ws)
	assert.Equal(t, "WS-REC", recNode.Meta["cobol_name_raw"])
	assertCobolDataItem(t, result, rec+"/WS-DATE", graph.KindField, "WS-DATE", 5, rec)
	yy := assertCobolDataItem(t, result, rec+"/WS-DATE/WS-YY", graph.KindField, "WS-YY", 10, rec+"/WS-DATE")
	assert.Equal(t, "9(2)", yy.Meta["cobol_pic"])
	assertCobolDataItem(t, result, rec+"/WS-DATE/WS-MM", graph.KindField, "WS-MM", 10, rec+"/WS-DATE")
	assertCobolDataItem(t, result, rec+"/FILLER#1", graph.KindField, "FILLER", 5, rec)
	flag := assertCobolDataItem(t, result, rec+"/WS-FLAG", graph.KindField, "WS-FLAG", 5, rec)
	assert.Equal(t, []string{"'Y'"}, flag.Meta["cobol_value"])
	assertCobolDataItem(t, result, rec+"/WS-FLAG/FLAG-ON", graph.KindField, "FLAG-ON", 88, rec+"/WS-FLAG")
	assertCobolDataItem(t, result, rec+"/FILLER#2", graph.KindField, "FILLER", 5, rec)
	tab := assertCobolDataItem(t, result, rec+"/WS-TAB", graph.KindField, "WS-TAB", 5, rec)
	assert.Equal(t, 1, tab.Meta["cobol_occurs_min"])
	assert.Equal(t, 10, tab.Meta["cobol_occurs_max"])
	assert.Equal(t, "WS-CNT", tab.Meta["cobol_occurs_depending"])
	assert.Equal(t, "X", tab.Meta["cobol_pic"])
	// An implicit filler (no entry name) is named FILLER and counted with
	// the written fillers.
	assertCobolDataItem(t, result, rec+"/FILLER#3", graph.KindField, "FILLER", 5, rec)
	cnt := assertCobolDataItem(t, result, p+"/WS-CNT", graph.KindVariable, "WS-CNT", 77, ws)
	assert.Equal(t, "COMP-3", cnt.Meta["cobol_usage"])
	assert.Equal(t, "9(4)", cnt.Meta["cobol_pic"])
	assertCobolDataItem(t, result, p+"/LK-REC", graph.KindVariable, "LK-REC", 1, lk)

	// A level outside 01-49, 66, 77, and 88 is never placed by guess: it is
	// counted on its section instead.
	assert.Equal(t, 1, cobolNodeByID(t, result, ws).Meta["cobol_unleveled_entry_count"])
	assert.NotContains(t, cobolNodeByID(t, result, lk).Meta, "cobol_unleveled_entry_count")

	// Clause Meta is present only when the clause is.
	assert.NotContains(t, recNode.Meta, "cobol_pic")
	assert.NotContains(t, yy.Meta, "cobol_occurs_min")

	// A 5 level nests exactly like a 05 level.
	short := extractCobolGrammar(t, e, path, strings.ReplaceAll(cobolDataItemsFixture, "          05 ", "          5 "))
	assert.Equal(t, cobolDataItemIDs(result), cobolDataItemIDs(short))
	for _, id := range cobolDataItemIDs(result) {
		assert.Equal(t, cobolMemberOf(result, id)[0].To, cobolMemberOf(short, id)[0].To, id)
	}

	// Extracting the same source twice yields identical ID lists.
	again := extractCobolGrammar(t, e, path, cobolDataItemsFixture)
	assert.Equal(t, cobolDataItemIDs(result), cobolDataItemIDs(again))

	// An 01 record named like a nested program gets #2; the nested
	// program's items sit under its own program symbol.
	const np = "src/nest.cbl"
	nested := extractCobolGrammar(t, e, np, cobolNestedDataFixture)
	assert.Equal(t, []string{np + "::OUTERPGM/INNERPGM#2", np + "::OUTERPGM/INNERPGM/WS-I"}, cobolDataItemIDs(nested))
	assertCobolDataItem(t, nested, np+"::OUTERPGM/INNERPGM#2", graph.KindVariable, "INNERPGM", 1,
		np+"::OUTERPGM/@working_storage_section")
	assertCobolDataItem(t, nested, np+"::OUTERPGM/INNERPGM/WS-I", graph.KindVariable, "WS-I", 1,
		np+"::OUTERPGM/INNERPGM/@working_storage_section")
}

func TestCobolGrammar_DataReferences(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/ref.cbl"
	result := extractCobolGrammar(t, e, path, cobolDataRefsFixture)
	const p = path + "::REFPGM"
	const unresolved = "unresolved::cobol::data::" + p + "::"

	redefines := cobolEdges(result, p+"/WS-ALT", p+"/WS-REC", graph.EdgeReferences)
	require.Len(t, redefines, 1)
	edge := redefines[0]
	for _, key := range append(append([]string{}, cobolDocumentKeys...),
		"prov_observation_kind", "prov_affected", "prov_start_row", "prov_start_column",
		"prov_end_row", "prov_end_column", "prov_start_byte", "prov_end_byte", "prov_range_exact") {
		assert.Contains(t, edge.Meta, key)
	}
	assert.Equal(t, "redefines_clause", edge.Meta["prov_observation_kind"])
	assert.Equal(t, "redefines", edge.Meta["cobol_ref_role"])
	assert.Equal(t, cobolNameNormRule, edge.Meta["cobol_name_norm"])
	assert.Equal(t, edge.Meta["prov_start_row"].(int)+1, edge.Line)
	assert.Equal(t, graph.OriginASTResolved, edge.Origin)
	assert.Equal(t, "EXTRACTED", edge.ConfidenceLabel)
	assert.NotContains(t, edge.Meta, "cobol_unresolved_reason")

	from := cobolEdges(result, p+"/WS-REC/WS-RN", p+"/WS-REC/WS-A", graph.EdgeReferences)
	require.Len(t, from, 1)
	assert.Equal(t, "renames_from", from[0].Meta["cobol_ref_role"])
	thru := cobolEdges(result, p+"/WS-REC/WS-RN", p+"/WS-REC/WS-B", graph.EdgeReferences)
	require.Len(t, thru, 1)
	assert.Equal(t, "renames_thru", thru[0].Meta["cobol_ref_role"])

	// Matching is case-insensitive under cobol-name-norm/1.
	require.Len(t, cobolEdges(result, p+"/WS-LOW", p+"/WS-REC", graph.EdgeReferences), 1)

	missing := cobolEdges(result, p+"/WS-BAD", unresolved+"WS-NONE", graph.EdgeReferences)
	require.Len(t, missing, 1)
	assert.Equal(t, "not_declared_in_program", missing[0].Meta["cobol_unresolved_reason"])
	assert.Equal(t, "redefines", missing[0].Meta["cobol_ref_role"])

	ambiguous := cobolEdges(result, p+"/G2/Y", unresolved+"X", graph.EdgeReferences)
	require.Len(t, ambiguous, 1)
	assert.Equal(t, "ambiguous", ambiguous[0].Meta["cobol_unresolved_reason"])
	assert.Equal(t, []string{"REFPGM/G1/X", "REFPGM/G2/X"}, ambiguous[0].Meta["cobol_candidate_symbols"])

	// An IN/OF qualifier narrows the match to the item under that group.
	require.Len(t, cobolEdges(result, p+"/G2/Z", p+"/G2/X", graph.EdgeReferences), 1)
	qualified := cobolEdges(result, p+"/G2/Q", unresolved+"X/G9", graph.EdgeReferences)
	require.Len(t, qualified, 1)
	assert.Equal(t, "not_declared_in_program", qualified[0].Meta["cobol_unresolved_reason"])
}

func TestCobolGrammar_DataContainsCopy(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/cpy.cbl"
	result := extractCobolGrammar(t, e, path, cobolDataCopyFixture)
	const p = path + "::CPYPGM"

	// The copybook's items are absent, so the open item says so, and later
	// entries still nest by level.
	assert.Equal(t, true, cobolNodeByID(t, result, p+"/WS-REC").Meta["cobol_contains_copy"])
	assertCobolDataItem(t, result, p+"/WS-REC/WS-B", graph.KindField, "WS-B", 5, p+"/WS-REC")
	assert.NotContains(t, cobolNodeByID(t, result, p+"/WS-REC/WS-B").Meta, "cobol_contains_copy")

	// A COPY before any record marks the section node.
	assert.Equal(t, true, cobolNodeByID(t, result, p+"/@working_storage_section").Meta["cobol_contains_copy"])

	// A COPY after a section's last entry lies outside the section's span
	// but still marks the open item.
	assert.Equal(t, true, cobolNodeByID(t, result, p+"/WS-T/WS-TA").Meta["cobol_contains_copy"])
	assert.NotContains(t, cobolNodeByID(t, result, p+"/WS-T").Meta, "cobol_contains_copy")
	assert.Equal(t, true, cobolNodeByID(t, result, p+"/LK-REC").Meta["cobol_contains_copy"])
	assert.NotContains(t, cobolNodeByID(t, result, p+"/@linkage_section").Meta, "cobol_contains_copy")

	assert.Equal(t, []string{p + "/WS-REC", p + "/WS-REC/WS-B", p + "/WS-T", p + "/WS-T/WS-TA", p + "/LK-REC"},
		cobolDataItemIDs(result))
}
