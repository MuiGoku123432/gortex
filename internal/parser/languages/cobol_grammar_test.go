//go:build !windows

package languages

import (
	"context"
	"strings"
	"testing"

	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/handoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// Every fixture in this file is invented 7-column fixed-format COBOL (D-14).
// None may be replaced with estate-derived source: this repository is public.

const cobolDemoFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. DEMOPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           STOP RUN.
`

const cobolNestedFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. OUTERPGM.
       PROCEDURE DIVISION.
       OUTER-PARA.
           STOP RUN.
       IDENTIFICATION DIVISION.
       PROGRAM-ID. INNERPGM.
       PROCEDURE DIVISION.
       INNER-PARA.
           GOBACK.
       END PROGRAM INNERPGM.
       END PROGRAM OUTERPGM.
`

// cobolSiblingProgram is one FIRSTPGM closed by its END PROGRAM marker; the
// duplicate fixtures repeat it as same-container siblings.
const cobolSiblingProgram = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. FIRSTPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           GOBACK.
       END PROGRAM FIRSTPGM.
`

const cobolLowercaseFixture = `       identification division.
       program-id. demopgm.
       procedure division.
       main-para.
           stop run.
`

const cobolUnresolvedFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. ONEPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           GOBACK.
       END PROGRAM OTHERPGM.
`

const cobolUnnamedFixture = `       IDENTIFICATION DIVISION.
       PROCEDURE DIVISION.
       MAIN-PARA.
           GOBACK.
`

// cobolShiftPrefix is three blank lines and two fixed-format comment lines
// (asterisk in column 7): five lines that must move ranges but never IDs.
const cobolShiftPrefix = "\n\n\n" +
	"      * INVENTED COMMENT LINE ONE\n" +
	"      * INVENTED COMMENT LINE TWO\n"

// cobolDocumentKeys are the document-level prov_* keys every analyzed file
// and program node carries (02-02-PLAN.md key contract).
var cobolDocumentKeys = []string{
	"prov_source_path", "prov_source_id", "prov_revision_content_id",
	"prov_parser_tool_id", "prov_parser_grammar_id", "prov_parser_module",
	"prov_parse_config_id", "prov_transform_config_id", "prov_handoff_schema",
	"prov_extractor_version", "prov_evidence_class", "prov_origin",
	"prov_confidence", "prov_document_grade", "prov_document_grade_reasons",
	"prov_grade_policy",
}

func extractCobolGrammar(t *testing.T, e *CobolGrammarExtractor, path, src string) *parser.ExtractionResult {
	t.Helper()
	result, err := e.Extract(path, []byte(src))
	require.NoError(t, err)
	require.NotNil(t, result)
	return result
}

func cobolProgramNodes(result *parser.ExtractionResult) []*graph.Node {
	var programs []*graph.Node
	for _, n := range result.Nodes {
		if n.Kind == graph.KindFunction && n.Meta["cobol_kind"] == "program" {
			programs = append(programs, n)
		}
	}
	return programs
}

func cobolProgramIDs(result *parser.ExtractionResult) []string {
	var ids []string
	for _, n := range cobolProgramNodes(result) {
		ids = append(ids, n.ID)
	}
	return ids
}

func cobolFileNode(t *testing.T, result *parser.ExtractionResult, path string) *graph.Node {
	t.Helper()
	for _, n := range result.Nodes {
		if n.Kind == graph.KindFile && n.ID == path {
			return n
		}
	}
	require.Failf(t, "no file node", "path %s", path)
	return nil
}

// cobolTestHandoff runs the extractor's own analyzer on src, so assertions
// compare persisted values with what the producer actually observed.
func cobolTestHandoff(t *testing.T, e *CobolGrammarExtractor, path, src string) handoff.Handoff {
	t.Helper()
	require.NotNil(t, e.analyzer, "extract once before reading the handoff")
	sourceID, err := preprocessor.NewSourceID(path)
	require.NoError(t, err)
	h, err := e.analyzer.Analyze(context.Background(), sourceID, []byte(src))
	require.NoError(t, err)
	return h
}

// assertCobolRange checks the PROV-03 ordering truth on one program node.
func assertCobolRange(t *testing.T, n *graph.Node) {
	t.Helper()
	startRow, ok := n.Meta["prov_start_row"].(int)
	require.True(t, ok, "%s has no int prov_start_row", n.ID)
	startCol := n.Meta["prov_start_column"].(int)
	endRow := n.Meta["prov_end_row"].(int)
	endCol := n.Meta["prov_end_column"].(int)
	assert.True(t, startRow < endRow || (startRow == endRow && startCol <= endCol),
		"%s range %d:%d-%d:%d is inverted", n.ID, startRow, startCol, endRow, endCol)
	assert.Equal(t, startRow+1, n.StartLine)
	assert.Equal(t, endRow+1, n.EndLine)
	assert.Equal(t, startCol, n.StartColumn)
	assert.Equal(t, endCol, n.EndColumn)
}

func TestCobolGrammar_ProgramNode(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/demo.cbl", cobolDemoFixture)

	programs := cobolProgramNodes(result)
	require.Len(t, programs, 1)
	program := programs[0]
	assert.Equal(t, "src/demo.cbl::DEMOPGM", program.ID)
	assert.Equal(t, graph.KindFunction, program.Kind)
	assert.Equal(t, "DEMOPGM", program.Name)
	assert.Equal(t, "DEMOPGM", program.QualName)
	assert.Equal(t, "program", program.Meta["cobol_kind"])
	assert.Empty(t, program.RepoPrefix)
	assert.Empty(t, program.WorkspaceID)
	assert.Empty(t, program.ProjectID)
	assertCobolRange(t, program)

	require.Len(t, result.Edges, 1)
	edge := findEdgeTo(result.Edges, program.ID)
	require.NotNil(t, edge)
	assert.Equal(t, "src/demo.cbl", edge.From)
	assert.Equal(t, graph.EdgeDefines, edge.Kind)
	assert.Equal(t, graph.OriginASTResolved, edge.Origin)
	assert.InDelta(t, 1.0, edge.Confidence, 0)
	assert.Equal(t, "EXTRACTED", edge.ConfidenceLabel)
	assert.Equal(t, program.StartLine, edge.Line)

	file := cobolFileNode(t, result, "src/demo.cbl")
	for _, key := range cobolDocumentKeys {
		assert.Contains(t, file.Meta, key)
		assert.Contains(t, program.Meta, key)
	}
	assert.Equal(t, "green", file.Meta["prov_document_grade"])
	assert.NotContains(t, file.Meta, "prov_containment_unresolved")
	assert.NotContains(t, file.Meta, "prov_unnamed_program_count")
	assert.Empty(t, file.RepoPrefix)
	assert.Empty(t, file.WorkspaceID)
	assert.Empty(t, file.ProjectID)
}

func TestCobolGrammar_NestedIDs(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/nest.cbl", cobolNestedFixture)

	programs := cobolProgramNodes(result)
	require.Len(t, programs, 2)
	outer, inner := programs[0], programs[1]
	assert.Equal(t, "src/nest.cbl::OUTERPGM", outer.ID)
	assert.Equal(t, "OUTERPGM", outer.QualName)
	assert.Equal(t, "src/nest.cbl::OUTERPGM/INNERPGM", inner.ID)
	assert.Equal(t, "OUTERPGM/INNERPGM", inner.QualName)
	assert.Equal(t, "INNERPGM", inner.Name)
	assertCobolRange(t, outer)
	assertCobolRange(t, inner)

	defines := 0
	for _, edge := range result.Edges {
		if edge.Kind == graph.EdgeDefines && edge.From == "src/nest.cbl" {
			defines++
		}
	}
	assert.Equal(t, 2, defines)

	// PROV-03, Pitfall 4: OUTERPGM keeps the parser-observed range, which
	// stops where INNERPGM begins; nothing is synthesized or merged.
	h := cobolTestHandoff(t, e, "src/nest.cbl", cobolNestedFixture)
	outerDef := -1
	for i, o := range h.Facts.Observations {
		if o.Kind == "program_definition" {
			outerDef = i
			break
		}
	}
	require.GreaterOrEqual(t, outerDef, 0)
	observedEnd := h.Facts.Observations[outerDef].Original.End.Row
	assert.Equal(t, int(observedEnd), outer.Meta["prov_end_row"])
	assert.LessOrEqual(t, outer.Meta["prov_end_row"].(int), inner.Meta["prov_start_row"].(int))
}

func TestCobolGrammar_DuplicateOrdinal(t *testing.T) {
	e := NewCobolGrammarExtractor()

	two := extractCobolGrammar(t, e, "src/dup.cbl", strings.Repeat(cobolSiblingProgram, 2))
	assert.Equal(t, []string{"src/dup.cbl::FIRSTPGM", "src/dup.cbl::FIRSTPGM#2"}, cobolProgramIDs(two))
	for _, n := range cobolProgramNodes(two) {
		assert.Equal(t, "FIRSTPGM", n.Name)
	}

	three := extractCobolGrammar(t, e, "src/dup.cbl", strings.Repeat(cobolSiblingProgram, 3))
	ids := cobolProgramIDs(three)
	assert.Equal(t, []string{"src/dup.cbl::FIRSTPGM", "src/dup.cbl::FIRSTPGM#2", "src/dup.cbl::FIRSTPGM#3"}, ids)
	for _, id := range ids {
		assert.False(t, strings.HasSuffix(id, "#1"), "%s carries a #1 ordinal", id)
	}

	again := extractCobolGrammar(t, e, "src/dup.cbl", strings.Repeat(cobolSiblingProgram, 3))
	assert.Equal(t, ids, cobolProgramIDs(again))
}

func TestCobolGrammar_IDIgnoresLines(t *testing.T) {
	e := NewCobolGrammarExtractor()
	for name, src := range map[string]string{
		"nested":    cobolNestedFixture,
		"duplicate": strings.Repeat(cobolSiblingProgram, 2),
	} {
		t.Run(name, func(t *testing.T) {
			base := cobolProgramNodes(extractCobolGrammar(t, e, "src/shift.cbl", src))
			shifted := cobolProgramNodes(extractCobolGrammar(t, e, "src/shift.cbl", cobolShiftPrefix+src))
			require.Len(t, shifted, len(base))
			require.NotEmpty(t, base)
			for i := range base {
				assert.Equal(t, base[i].ID, shifted[i].ID)
				assert.Equal(t, base[i].Meta["prov_start_row"].(int)+5, shifted[i].Meta["prov_start_row"])
			}
		})
	}
}

func TestCobolGrammar_NamesVerbatim(t *testing.T) {
	e := NewCobolGrammarExtractor()
	lower := cobolProgramNodes(extractCobolGrammar(t, e, "src/demo.cbl", cobolLowercaseFixture))
	upper := cobolProgramNodes(extractCobolGrammar(t, e, "src/demo.cbl", cobolDemoFixture))
	require.Len(t, lower, 1)
	require.Len(t, upper, 1)
	assert.Equal(t, "src/demo.cbl::demopgm", lower[0].ID)
	assert.Equal(t, "demopgm", lower[0].Name)
	assert.NotEqual(t, upper[0].ID, lower[0].ID)
}

func TestCobolGrammar_ContainmentUnresolved(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/unres.cbl", cobolUnresolvedFixture)
	assert.Equal(t, []string{"src/unres.cbl::ONEPGM"}, cobolProgramIDs(result))
	assert.Equal(t, true, cobolFileNode(t, result, "src/unres.cbl").Meta["prov_containment_unresolved"])
}

func TestCobolGrammar_UnnamedProgramCounted(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/unnamed.cbl", cobolUnnamedFixture)
	assert.Empty(t, cobolProgramNodes(result))
	file := cobolFileNode(t, result, "src/unnamed.cbl")
	assert.Equal(t, 1, file.Meta["prov_unnamed_program_count"])
	assert.Equal(t, "red", file.Meta["prov_document_grade"])
}
