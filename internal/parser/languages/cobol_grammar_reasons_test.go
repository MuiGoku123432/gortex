//go:build darwin || linux

package languages

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
)

// Every fixture in this file is invented 7-column fixed-format COBOL (P2
// D-14). None may be replaced with estate-derived source: this repository is
// public.

// cobolBrokenCopybookFixture is an invented data member; the tests make its
// analysis fail through the zero-analyzer seam, not through its content.
const cobolBrokenCopybookFixture = "       01 BRK-REC.\n" +
	"           05 BRK-A PIC X(4).\n"

// cobolAmbiguousCursorFixture declares C2 twice and C1 once, then fetches
// C2, C1, and the undeclared C9.
const cobolAmbiguousCursorFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. AMBPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-X PIC X(10).
       PROCEDURE DIVISION.
       P0.
           EXEC SQL DECLARE C2 CURSOR FOR SELECT X FROM T8 END-EXEC.
           EXEC SQL DECLARE C2 CURSOR FOR SELECT X FROM T9 END-EXEC.
           EXEC SQL DECLARE C1 CURSOR FOR SELECT X FROM T7 END-EXEC.
       P1.
           EXEC SQL FETCH C2 INTO :WS-X END-EXEC.
           EXEC SQL FETCH C1 INTO :WS-X END-EXEC.
           EXEC SQL FETCH C9 INTO :WS-X END-EXEC.
           GOBACK.
`

// cobolFailingAnalyzer returns an extractor whose init succeeded but whose
// analyzer is the zero value, so every Analyze and AnalyzeCopybook fails
// with "analyzer has no attested tool identity" (05-RESEARCH WR-06 seam).
func cobolFailingAnalyzer(t *testing.T) *CobolGrammarExtractor {
	t.Helper()
	e := NewCobolGrammarExtractor()
	e.once.Do(e.init)
	require.NoError(t, e.initErr)
	e.analyzer = preprocessor.Analyzer{}
	return e
}

// WR-06: a copybook whose analysis fails stays a present copybook, so a COPY
// still binds to it, and records why with one analysis-failed parse finding.
// The producer's error text never reaches the graph (T-05-32).
func TestCobolGrammar_CopybookAnalysisFailed(t *testing.T) {
	e := cobolFailingAnalyzer(t)
	const path = "copy/brokenc.cpy"
	result, err := e.Extract(path, []byte(cobolBrokenCopybookFixture))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Nodes, 2)

	file := cobolFileNode(t, result, path)
	assert.Equal(t, graph.KindFile, file.Kind)
	assert.Equal(t, path, file.Name)
	assert.Equal(t, path, file.FilePath)
	assert.Equal(t, "cobol", file.Language)
	assert.Equal(t, 1, file.StartLine)
	assert.Equal(t, bytes.Count([]byte(cobolBrokenCopybookFixture), []byte("\n"))+1, file.EndLine)
	sourceID, err := preprocessor.NewSourceID(path)
	require.NoError(t, err)
	for key, want := range map[string]any{
		"cobol_kind":               "copybook",
		"cobol_name_key":           "BROKENC",
		"cobol_name_norm":          cobolNameNormRule,
		"prov_source_path":         path,
		"prov_source_id":           string(sourceID),
		"prov_revision_content_id": string(preprocessor.NewContentID([]byte(cobolBrokenCopybookFixture))),
		"prov_parser_module":       e.parserModule,
		"prov_preprocessor_module": e.preprocessorModule,
		"prov_parser_grammar_id":   cobolApprovedGrammarID,
		"prov_extractor_version":   cobolGrammarExtractorVersion,
		"prov_evidence_class":      "DETERMINISTIC",
		"prov_origin":              graph.OriginASTResolved,
		"prov_confidence":          1.0,
		"prov_analysis_absence":    "copybook_analysis_failed",
	} {
		assert.Equal(t, want, file.Meta[key], key)
	}
	assert.NotEmpty(t, file.Meta["prov_parser_module"])
	// No handoff exists, so no handoff-derived key may be invented.
	for _, key := range []string{
		"prov_parser_tool_id", "prov_parse_config_id", "prov_transform_config_id",
		"prov_handoff_schema", "prov_document_grade", "prov_document_grade_reasons",
		"prov_grade_policy",
	} {
		assert.NotContains(t, file.Meta, key)
	}

	findings := cobolFindingNodes(result, "")
	require.Len(t, findings, 1)
	f := findings[0]
	assert.Equal(t, "analysis-failed", f.Name)
	assert.Equal(t, graph.FindingClassParse, f.Meta[graph.MetaEvidenceClass])
	assert.Equal(t, "analysis-failed", f.Meta[graph.MetaFindingKind])
	assert.Equal(t, "analysis-error", f.Meta[graph.MetaFindingReason])
	assert.Equal(t, "high", f.Meta[graph.MetaFindingSeverity])
	assert.Equal(t, "document", f.Meta[graph.MetaFindingScope])
	assert.Equal(t, path, f.Meta[graph.MetaFindingContainer])
	assert.Equal(t, false, f.Meta["prov_range_exact"])
	assert.Equal(t, "document_scope", f.Meta["prov_range_absence"])
	assert.True(t, strings.HasPrefix(f.ID, path+"::@finding/parse/"), f.ID)
	assert.NotContains(t, f.Meta, "cobol_kind")
	assert.NotContains(t, f.Meta, "prov_analysis_absence")

	require.Len(t, result.Edges, 1)
	assert.Equal(t, graph.EdgeMemberOf, result.Edges[0].Kind)
	assert.Equal(t, f.ID, result.Edges[0].From)
	assert.Equal(t, path, result.Edges[0].To)

	for _, n := range result.Nodes {
		assertCobolMetaContract(t, n.ID, n.Meta)
		assert.NotContains(t, n.Name, "attested")
		for key, v := range n.Meta {
			if s, ok := v.(string); ok {
				assert.NotContains(t, s, "attested", "%s %s", n.ID, key)
			}
		}
	}

	again, err := e.Extract(path, []byte(cobolBrokenCopybookFixture))
	require.NoError(t, err)
	assert.Equal(t, f.ID, cobolFindingNodes(again, "")[0].ID, "the finding ID is stable across reruns")
}

// D-20: a program whose analysis fails keeps the hard failure; no program
// identity is ever guessed from a filename.
func TestCobolGrammar_ProgramAnalysisFailedStillFails(t *testing.T) {
	e := cobolFailingAnalyzer(t)
	result, err := e.Extract("cobol/broken.cbl", []byte(cobolDemoFixture))
	require.Error(t, err)
	assert.Nil(t, result)
}

// WR-07: a cursor declared twice in one program is ambiguous; an undeclared
// one stays cursor_not_declared, and a once-declared one binds.
func TestCobolGrammar_CursorAmbiguous(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/amb.cbl"
	result := extractCobolGrammar(t, e, path, cobolAmbiguousCursorFixture)
	stmt := func(k int) string {
		s := "AMBPGM/@proc/P1/@exec_sql_statement"
		if k >= 2 {
			s += "#" + strconv.Itoa(k)
		}
		return s
	}

	c2 := cobolNodeByID(t, result, path+"::"+stmt(1))
	assert.Equal(t, "C2", c2.Meta["sql_cursor"])
	assert.Equal(t, "ambiguous", c2.Meta["cobol_unresolved_reason"])
	assert.Empty(t, cobolStatementEdges(result, stmt(1)))

	c1 := cobolNodeByID(t, result, path+"::"+stmt(2))
	assert.NotContains(t, c1.Meta, "cobol_unresolved_reason")
	assert.Equal(t, []string{"reads db2::_unqualified/T7"}, cobolEdgeTargets(cobolStatementEdges(result, stmt(2))))

	c9 := cobolNodeByID(t, result, path+"::"+stmt(3))
	assert.Equal(t, "C9", c9.Meta["sql_cursor"])
	assert.Equal(t, "cursor_not_declared", c9.Meta["cobol_unresolved_reason"])
}

// IN-08: a nested program whose END PROGRAM containment is unresolved is
// marked, so the resolver never reports a call to it as plainly missing.
func TestCobolGrammar_NestedContainmentUnresolved(t *testing.T) {
	e := NewCobolGrammarExtractor()
	for _, tc := range []struct {
		name, src string
		stamped   map[string]any // program QualName -> cobol_nested_unresolved
	}{
		{
			"stolen_marker",
			cobolProgram("APGM", "") + cobolProgram("BPGM", "BPGM") + cobolProgram("APGM", "APGM"),
			map[string]any{"APGM": nil, "APGM/BPGM": true, "APGM/APGM": true},
		},
		{"well_nested", cobolNestedFixture, map[string]any{"OUTERPGM": nil, "OUTERPGM/INNERPGM": nil}},
		{"top_level_unresolved", cobolUnresolvedFixture, map[string]any{"ONEPGM": nil}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := map[string]any{}
			for _, n := range cobolProgramNodes(extractCobolGrammar(t, e, "src/in08.cbl", tc.src)) {
				got[n.QualName] = n.Meta["cobol_nested_unresolved"]
				if n.Meta["cobol_nested"] == false {
					assert.NotContains(t, n.Meta, "cobol_nested_unresolved", n.ID)
				}
			}
			assert.Equal(t, tc.stamped, got)
		})
	}
}
