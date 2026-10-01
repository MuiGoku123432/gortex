//go:build darwin || linux

package languages

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/handoff"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/sourcemap"
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

// cobolProgram is one invented program named name, closed by END PROGRAM
// end when end is not empty.
func cobolProgram(name, end string) string {
	src := "       IDENTIFICATION DIVISION.\n" +
		"       PROGRAM-ID. " + name + ".\n" +
		"       PROCEDURE DIVISION.\n" +
		"       MAIN-PARA.\n" +
		"           GOBACK.\n"
	if end != "" {
		src += "       END PROGRAM " + end + ".\n"
	}
	return src
}

// WR-01: a program whose END PROGRAM marker is matched by a different
// program cannot be placed; the IDs are emitted, but never as resolved.
func TestCobolGrammar_StolenEndMarkerUnresolved(t *testing.T) {
	e := NewCobolGrammarExtractor()
	for _, tc := range []struct {
		name, src string
		ids       []string
	}{
		{
			// APGM has no marker of its own; the later END PROGRAM APGM
			// belongs to the second APGM, so the first stays open with
			// children at end of file.
			"marker_belongs_to_later_program",
			cobolProgram("APGM", "") + cobolProgram("BPGM", "BPGM") + cobolProgram("APGM", "APGM"),
			[]string{"src/stolen.cbl::APGM", "src/stolen.cbl::APGM/BPGM", "src/stolen.cbl::APGM/APGM"},
		},
		{
			// END PROGRAM APGM closes the nested BPGM frame implicitly.
			"outer_marker_skips_open_frame",
			cobolProgram("APGM", "") + cobolProgram("BPGM", "APGM") + cobolProgram("BPGM", "BPGM"),
			[]string{"src/stolen.cbl::APGM", "src/stolen.cbl::APGM/BPGM", "src/stolen.cbl::BPGM"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := extractCobolGrammar(t, e, "src/stolen.cbl", tc.src)
			assert.Equal(t, tc.ids, cobolProgramIDs(result))
			assert.Equal(t, true, cobolFileNode(t, result, "src/stolen.cbl").Meta["prov_containment_unresolved"])
		})
	}
}

// WR-02: END PROGRAM matches a program name case-insensitively, as COBOL
// compares user-defined words, while IDs keep the name as written (D-20).
func TestCobolGrammar_EndProgramMatchesCaseInsensitively(t *testing.T) {
	e := NewCobolGrammarExtractor()
	for _, tc := range []struct {
		name, outer, end string
		ids              []string
	}{
		{"mixed_case", "OuterPgm", "OUTERPGM", []string{"src/case.cbl::OuterPgm", "src/case.cbl::OuterPgm/INNERPGM"}},
		{"literal_name", `"LitOuter"`, "LITOUTER", []string{`src/case.cbl::"LitOuter"`, `src/case.cbl::"LitOuter"/INNERPGM`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := cobolProgram(tc.outer, "") + cobolProgram("INNERPGM", "innerpgm") + "       END PROGRAM " + tc.end + ".\n"
			result := extractCobolGrammar(t, e, "src/case.cbl", src)
			assert.Equal(t, tc.ids, cobolProgramIDs(result))
			assert.NotContains(t, cobolFileNode(t, result, "src/case.cbl").Meta, "prov_containment_unresolved")
		})
	}
}

// WR-03: only a projection that images original bytes yields coordinates;
// no-image and wholly synthetic projections carry anchors, not locations.
func TestCobolGrammar_RangeNeverInvented(t *testing.T) {
	anchor := sourcemap.Point{Byte: 40, Row: 2, Column: 7}
	for _, tc := range []struct {
		name    string
		r       *sourcemap.Projection
		absence string
	}{
		{"missing", nil, "no_original_projection"},
		{"no_image", &sourcemap.Projection{Start: anchor, End: anchor, NoImage: true}, "no_original_projection"},
		{"synthetic", &sourcemap.Projection{Start: anchor, End: anchor, Synthetic: true}, "synthetic_projection"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &graph.Node{ID: "src/demo.cbl::DEMOPGM", Meta: map[string]any{}}
			cobolStampRange(node, tc.r)
			assert.Equal(t, false, node.Meta["prov_range_exact"])
			assert.Equal(t, tc.absence, node.Meta["prov_range_absence"])
			assert.Zero(t, node.StartLine)
			assert.Zero(t, node.EndLine)
			for key := range node.Meta {
				assert.False(t, strings.HasPrefix(key, "prov_start_") || strings.HasPrefix(key, "prov_end_"),
					"%s carries invented coordinate %s", tc.name, key)
			}
		})
	}

	t.Run("imaged", func(t *testing.T) {
		node := &graph.Node{ID: "src/demo.cbl::DEMOPGM", Meta: map[string]any{}}
		cobolStampRange(node, &sourcemap.Projection{
			Start: sourcemap.Point{Byte: 33, Row: 1, Column: 7},
			End:   sourcemap.Point{Byte: 120, Row: 4, Column: 20},
			Exact: true,
		})
		assert.Equal(t, true, node.Meta["prov_range_exact"])
		assert.NotContains(t, node.Meta, "prov_range_absence")
		assert.Equal(t, 33, node.Meta["prov_start_byte"])
		assertCobolRange(t, node)
	})
}

func TestCobolGrammar_UnnamedProgramCounted(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/unnamed.cbl", cobolUnnamedFixture)
	assert.Empty(t, cobolProgramNodes(result))
	file := cobolFileNode(t, result, "src/unnamed.cbl")
	assert.Equal(t, 1, file.Meta["prov_unnamed_program_count"])
	assert.Equal(t, "red", file.Meta["prov_document_grade"])
}

const cobolAmberFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. AMBERPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A PIC X(
       PROCEDURE DIVISION.
       MAIN-PARA.
           GOBACK.
`

const cobolRedFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. BADPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           MOVE ))) TO (((.
           GOBACK.
`

const cobolCopybookFixture = `       01 CUST-REC.
           05 CUST-ID      PIC 9(6).
           05 CUST-NAME    PIC X(30).
`

var cobolGrammarHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// cobolWrongGrammarID is an approved-grammar pin no real grammar produces.
var cobolWrongGrammarID = strings.Repeat("0", 64)

func TestCobolGrammar_DegradedAmber(t *testing.T) {
	e := NewCobolGrammarExtractor()
	for _, tc := range []struct {
		name, path, src, id, grade string
	}{
		{"amber", "src/amber.cbl", cobolAmberFixture, "src/amber.cbl::AMBERPGM", "amber"},
		{"red", "src/bad.cbl", cobolRedFixture, "src/bad.cbl::BADPGM", "red"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			programs := cobolProgramNodes(extractCobolGrammar(t, e, tc.path, tc.src))
			require.Len(t, programs, 1, "a degraded document still emits its program (D-07)")
			program := programs[0]
			assert.Equal(t, tc.id, program.ID)
			assert.Equal(t, tc.grade, program.Meta["prov_document_grade"])
			reasons, ok := program.Meta["prov_document_grade_reasons"].([]string)
			require.True(t, ok)
			assert.NotEmpty(t, reasons)
			if tc.grade == "amber" {
				assert.Contains(t, reasons, "missing-node")
			}
			// Confidence is not evidence (handoff section 6): degradation
			// shows in the grade keys, never in a lowered confidence.
			assert.InDelta(t, 1.0, program.Meta["prov_confidence"], 0)

			h := cobolTestHandoff(t, e, tc.path, tc.src)
			affected, found := false, false
			for _, o := range h.Facts.Observations {
				if o.Kind == "program_definition" {
					affected, found = o.Affected, true
					break
				}
			}
			require.True(t, found)
			assert.Equal(t, affected, program.Meta["prov_affected"])
		})
	}
}

func TestCobolGrammar_CopybookSkipsAnalysis(t *testing.T) {
	for _, path := range []string{"copy/custrec.cpy", "COPY/CUSTREC.CPY"} {
		t.Run(path, func(t *testing.T) {
			for _, e := range []*CobolGrammarExtractor{
				NewCobolGrammarExtractor(),
				// A wrong pin still succeeds: Analyze and the grammar check
				// are never reached for a copybook.
				{approvedGrammarID: cobolWrongGrammarID},
			} {
				result := extractCobolGrammar(t, e, path, cobolCopybookFixture)
				require.Len(t, result.Nodes, 1)
				assert.Empty(t, result.Edges)
				file := cobolFileNode(t, result, path)
				assert.Equal(t, "copybook_standalone_analysis_unsupported", file.Meta["prov_analysis_absence"])
				assert.Equal(t, path, file.Meta["prov_source_path"])
				assert.Equal(t, cobolGrammarExtractorVersion, file.Meta["prov_extractor_version"])
				assert.Regexp(t, cobolGrammarHex64, file.Meta["prov_source_id"])
				assert.Equal(t, string(preprocessor.NewContentID([]byte(cobolCopybookFixture))),
					file.Meta["prov_revision_content_id"])
				assert.NotContains(t, file.Meta, "prov_document_grade")
				for key := range file.Meta {
					assert.False(t, strings.HasPrefix(key, "prov_parser_"), "copybook carries %s", key)
				}
			}
		})
	}
}

func TestCobolGrammar_EmptySource(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result, err := e.Extract("src/empty.cbl", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Nodes, 1, "no program is fabricated from an empty source")
	assert.Empty(t, result.Edges)
	file := cobolFileNode(t, result, "src/empty.cbl")
	for _, key := range cobolDocumentKeys {
		assert.Contains(t, file.Meta, key)
	}
	assert.Equal(t, "red", file.Meta["prov_document_grade"])
}

func TestCobolGrammar_ConcurrentExtract(t *testing.T) {
	// A fresh extractor, so the sync.Once initialization races as well.
	e := NewCobolGrammarExtractor()
	const workers = 8
	ids := make([][]string, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := e.Extract("src/nest.cbl", []byte(cobolNestedFixture))
			errs[i] = err
			if err == nil {
				ids[i] = cobolProgramIDs(result)
			}
		}()
	}
	wg.Wait()
	want := []string{"src/nest.cbl::OUTERPGM", "src/nest.cbl::OUTERPGM/INNERPGM"}
	for i := range workers {
		require.NoError(t, errs[i])
		assert.Equal(t, want, ids[i], "worker %d", i)
	}
}

func TestCobolGrammar_ApprovedGrammarPinned(t *testing.T) {
	e := NewCobolGrammarExtractor()
	first := extractCobolGrammar(t, e, "src/demo.cbl", cobolDemoFixture)
	second := extractCobolGrammar(t, e, "src/nest.cbl", cobolNestedFixture)

	firstFile := cobolFileNode(t, first, "src/demo.cbl")
	secondFile := cobolFileNode(t, second, "src/nest.cbl")
	programs := cobolProgramNodes(first)
	require.Len(t, programs, 1)
	for _, n := range []*graph.Node{firstFile, programs[0], secondFile} {
		assert.Equal(t, cobolApprovedGrammarID, n.Meta["prov_parser_grammar_id"], n.ID)
		for _, key := range []string{
			"prov_parser_tool_id", "prov_source_id", "prov_revision_content_id",
			"prov_parse_config_id", "prov_transform_config_id",
		} {
			assert.Regexp(t, cobolGrammarHex64, n.Meta[key], "%s %s", n.ID, key)
		}
	}
	// PROV-05: one process, one parser and extractor identity for every file.
	for _, key := range []string{"prov_parser_tool_id", "prov_parser_module", "prov_extractor_version"} {
		assert.Equal(t, firstFile.Meta[key], secondFile.Meta[key], key)
	}
}

func TestCobolGrammar_GrammarMismatchFails(t *testing.T) {
	e := &CobolGrammarExtractor{approvedGrammarID: cobolWrongGrammarID}
	result, err := e.Extract("src/demo.cbl", []byte(cobolDemoFixture))
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, errCobolGrammarNotApproved), "err = %v", err)
	assert.Contains(t, err.Error(), cobolApprovedGrammarID, "names the observed grammar")
	assert.Contains(t, err.Error(), cobolWrongGrammarID, "names the expected grammar")
}
