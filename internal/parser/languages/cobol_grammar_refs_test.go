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

const cobolCallsFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. CALLPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-PGM PIC X(8).
       PROCEDURE DIVISION.
       MAIN-PARA.
           CALL 'SUBPGM'.
           CALL "lowpgm".
           CALL 'A/B#1'.
           CALL WS-PGM.
           CALL WS-NONE.
           CALL FUNCTION TRIM(WS-PGM).
           GOBACK.
       END PROGRAM CALLPGM.
       IDENTIFICATION DIVISION.
       PROGRAM-ID. OTHERPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-REC.
          05 WS-PGM PIC X(8).
       PROCEDURE DIVISION.
       OTHER-PARA.
           CALL WS-PGM.
           CALL WS-PGM OF WS-REC.
           GOBACK.
       END PROGRAM OTHERPGM.
`

const cobolCopyFixture = `       COPY CPYF.
       IDENTIFICATION DIVISION.
       PROGRAM-ID. COPYPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CPYA.
       COPY CPYB OF MYLIB.
       COPY 'cpyc.cpy'.
       COPY 'cpé''x'.
-INC CPYD
       COPY IDMS RECORD EMP-REC.
       COPY IDMS SUBSCHEMA-NAMES.
           EXEC SQL INCLUDE SQLCA END-EXEC.
       01 WS-A PIC X.
       PROCEDURE DIVISION.
       P1.
           COPY CPYP.
           GOBACK.
`

const cobolNestedCopyFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. OUTERPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CPYO.
       PROCEDURE DIVISION.
       OUTER-PARA.
           GOBACK.
       IDENTIFICATION DIVISION.
       PROGRAM-ID. INNERPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CPYI.
       PROCEDURE DIVISION.
       INNER-PARA.
           GOBACK.
       END PROGRAM INNERPGM.
       END PROGRAM OUTERPGM.
`

// cobolOneEdge returns the single edge of kind from from to to.
func cobolOneEdge(t *testing.T, result *parser.ExtractionResult, from, to string, kind graph.EdgeKind) *graph.Edge {
	t.Helper()
	edges := cobolEdges(result, from, to, kind)
	require.Len(t, edges, 1, "%s edges %s -> %s", kind, from, to)
	return edges[0]
}

// assertCobolRelationProvenance checks the document keys, the observation
// kind, and the exact original range a relationship edge must carry.
func assertCobolRelationProvenance(t *testing.T, e *graph.Edge, observationKind string) {
	t.Helper()
	for _, key := range cobolDocumentKeys {
		assert.Contains(t, e.Meta, key)
	}
	assert.Equal(t, observationKind, e.Meta["prov_observation_kind"])
	assert.Equal(t, graph.OriginASTResolved, e.Origin)
	startRow, ok := e.Meta["prov_start_row"].(int)
	require.True(t, ok, "prov_start_row = %#v", e.Meta["prov_start_row"])
	assert.Equal(t, startRow+1, e.Line)
}

func TestCobolGrammar_Calls(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/calls.cbl"
	result := extractCobolGrammar(t, e, path, cobolCallsFixture)
	const main = path + "::CALLPGM/@proc/MAIN-PARA"

	t.Run("literal", func(t *testing.T) {
		edge := cobolOneEdge(t, result, main, "unresolved::cobol::program::SUBPGM", graph.EdgeCalls)
		assertCobolRelationProvenance(t, edge, "call_statement")
		assert.Equal(t, true, edge.Meta["prov_range_exact"])
		assert.Equal(t, "literal", edge.Meta["cobol_call"])
		assert.Equal(t, "'SUBPGM'", edge.Meta["cobol_target_raw"])
		assert.Equal(t, cobolNameNormRule, edge.Meta["cobol_name_norm"])
		assert.Equal(t, "CALLPGM/@proc/MAIN-PARA/@call_statement", edge.Meta["cobol_statement_symbol"])
		assert.Equal(t, path+"::CALLPGM", edge.Meta["cobol_caller_program"], "the resolver scopes nested candidates by caller")
		assert.NotContains(t, edge.Meta, "cobol_unresolved_reason")

		low := cobolOneEdge(t, result, main, "unresolved::cobol::program::LOWPGM", graph.EdgeCalls)
		assert.Equal(t, `"lowpgm"`, low.Meta["cobol_target_raw"])
		assert.Equal(t, "CALLPGM/@proc/MAIN-PARA/@call_statement#2", low.Meta["cobol_statement_symbol"])

		// ID metacharacters are percent-encoded (T-03-22).
		enc := cobolOneEdge(t, result, main, "unresolved::cobol::program::A%2FB%231", graph.EdgeCalls)
		assert.Equal(t, "'A/B#1'", enc.Meta["cobol_target_raw"])
	})

	t.Run("dynamic", func(t *testing.T) {
		const target = "unresolved::cobol::dynamic::program::" + path + "::CALLPGM::WS-PGM"
		edge := cobolOneEdge(t, result, main, target, graph.EdgeCalls)
		assertCobolRelationProvenance(t, edge, "call_statement")
		assert.Equal(t, "dynamic", edge.Meta["cobol_call"])
		assert.Equal(t, "dynamic", edge.Meta["cobol_unresolved_reason"])
		assert.Equal(t, "WS-PGM", edge.Meta["cobol_target_raw"])
		assert.Equal(t, cobolNameNormRule, edge.Meta["cobol_name_norm"])

		ref := cobolOneEdge(t, result, main, path+"::CALLPGM/WS-PGM", graph.EdgeReferences)
		assert.Equal(t, "call_target", ref.Meta["cobol_ref_role"])
		assert.NotContains(t, ref.Meta, "cobol_unresolved_reason")
		assert.Equal(t, "CALLPGM/@proc/MAIN-PARA/@call_statement#4", ref.Meta["cobol_statement_symbol"])

		none := cobolOneEdge(t, result, main, "unresolved::cobol::data::"+path+"::CALLPGM::WS-NONE", graph.EdgeReferences)
		assert.Equal(t, "call_target", none.Meta["cobol_ref_role"])
		assert.Equal(t, "not_declared_in_program", none.Meta["cobol_unresolved_reason"])
		cobolOneEdge(t, result, main, "unresolved::cobol::dynamic::program::"+path+"::CALLPGM::WS-NONE", graph.EdgeCalls)
	})

	t.Run("function", func(t *testing.T) {
		var fn []*graph.Edge
		for _, edge := range result.Edges {
			if edge.From == main && edge.Meta["cobol_statement_symbol"] == "CALLPGM/@proc/MAIN-PARA/@call_statement#6" {
				fn = append(fn, edge)
			}
		}
		require.Len(t, fn, 1, "a FUNCTION target gets no data reference")
		assert.Equal(t, graph.EdgeCalls, fn[0].Kind)
		assert.Equal(t, "dynamic", fn[0].Meta["cobol_call"])
		assert.Equal(t, "dynamic", fn[0].Meta["cobol_unresolved_reason"])
		assert.Contains(t, fn[0].To, "unresolved::cobol::dynamic::program::"+path+"::CALLPGM::")
	})

	t.Run("qualified_and_program_scoped", func(t *testing.T) {
		const other = path + "::OTHERPGM/@proc/OTHER-PARA"
		plain := cobolOneEdge(t, result, other, "unresolved::cobol::dynamic::program::"+path+"::OTHERPGM::WS-PGM", graph.EdgeCalls)
		assert.Equal(t, "dynamic", plain.Meta["cobol_call"])
		qualified := cobolOneEdge(t, result, other, "unresolved::cobol::dynamic::program::"+path+"::OTHERPGM::WS-PGM/WS-REC", graph.EdgeCalls)
		assert.Equal(t, "WS-PGM OF WS-REC", qualified.Meta["cobol_target_raw"])

		refs := cobolEdges(result, other, path+"::OTHERPGM/WS-REC/WS-PGM", graph.EdgeReferences)
		require.Len(t, refs, 2, "both CALLs reference the record's WS-PGM")
		for _, ref := range refs {
			assert.Equal(t, "call_target", ref.Meta["cobol_ref_role"])
		}
	})

	t.Run("never_regex_namespaces", func(t *testing.T) {
		for _, edge := range result.Edges {
			if edge.Kind != graph.EdgeCalls || edge.Meta["cobol_call"] == nil {
				continue
			}
			assert.Regexp(t, `^unresolved::cobol::(program|dynamic::program)::`, edge.To)
		}
	})
}

func TestCobolGrammar_Copy(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/copy.cbl"
	result := extractCobolGrammar(t, e, path, cobolCopyFixture)
	const program = path + "::COPYPGM"
	const book = "unresolved::cobol::copybook::"

	t.Run("copy", func(t *testing.T) {
		edge := cobolOneEdge(t, result, program, book+"CPYA", graph.EdgeImports)
		assertCobolRelationProvenance(t, edge, "copy_statement")
		assert.Equal(t, "copy", edge.Meta["cobol_copy_form"])
		assert.Equal(t, "CPYA", edge.Meta["cobol_copy_member_raw"])
		assert.Equal(t, cobolNameNormRule, edge.Meta["cobol_name_norm"])
		assert.NotContains(t, edge.Meta, "cobol_copy_library")
		assert.NotContains(t, edge.Meta, "cobol_unresolved_reason")
		assert.NotContains(t, edge.Meta, "cobol_statement_symbol")

		lib := cobolOneEdge(t, result, program, book+"CPYB", graph.EdgeImports)
		assert.Equal(t, "MYLIB", lib.Meta["cobol_copy_library"])

		quoted := cobolOneEdge(t, result, program, book+"CPYC", graph.EdgeImports)
		assert.Equal(t, "'cpyc.cpy'", quoted.Meta["cobol_copy_member_raw"])

		// One quote level removed, the doubled quote un-doubled, ASCII
		// upper-folded, non-ASCII bytes unchanged; the raw spelling is kept.
		encoded := cobolOneEdge(t, result, program, book+"CPé'X", graph.EdgeImports)
		assert.Equal(t, "'cpé''x'", encoded.Meta["cobol_copy_member_raw"])
		assert.Equal(t, cobolNameNormRule, encoded.Meta["cobol_name_norm"])
	})

	t.Run("inc", func(t *testing.T) {
		edge := cobolOneEdge(t, result, program, book+"CPYD", graph.EdgeImports)
		assert.Equal(t, "inc", edge.Meta["cobol_copy_form"])
		assert.Equal(t, "CPYD", edge.Meta["cobol_copy_member_raw"])
	})

	t.Run("idms", func(t *testing.T) {
		edge := cobolOneEdge(t, result, program, book+"idms/RECORD/EMP-REC", graph.EdgeImports)
		assert.Equal(t, "idms", edge.Meta["cobol_copy_form"])
		assert.Equal(t, "RECORD", edge.Meta["cobol_copy_idms_kind"])
		assert.Equal(t, "EMP-REC", edge.Meta["cobol_copy_member_raw"])
		assert.Equal(t, "idms_dictionary_member", edge.Meta["cobol_unresolved_reason"])

		bare := cobolOneEdge(t, result, program, book+"idms/_/SUBSCHEMA-NAMES", graph.EdgeImports)
		assert.NotContains(t, bare.Meta, "cobol_copy_idms_kind")
		assert.Equal(t, "idms_dictionary_member", bare.Meta["cobol_unresolved_reason"])
	})

	t.Run("sql_include", func(t *testing.T) {
		edge := cobolOneEdge(t, result, program, book+"SQLCA", graph.EdgeImports)
		assertCobolRelationProvenance(t, edge, "exec_sql_include")
		assert.Equal(t, "sql_include", edge.Meta["cobol_copy_form"])
		assert.Equal(t, true, edge.Meta["sql_include_builtin"])
		assert.Equal(t, "SQLCA", edge.Meta["cobol_copy_member_raw"])
	})

	t.Run("placement", func(t *testing.T) {
		para := cobolOneEdge(t, result, program+"/@proc/P1", book+"CPYP", graph.EdgeImports)
		assert.Equal(t, "COPYPGM/@proc/P1/@copy_statement", para.Meta["cobol_statement_symbol"])

		file := cobolOneEdge(t, result, path, book+"CPYF", graph.EdgeImports)
		assert.Equal(t, "copy", file.Meta["cobol_copy_form"])
	})

	t.Run("nested_owner", func(t *testing.T) {
		const nest = "src/nest.cbl"
		nested := extractCobolGrammar(t, e, nest, cobolNestedCopyFixture)
		cobolOneEdge(t, nested, nest+"::OUTERPGM", book+"CPYO", graph.EdgeImports)
		cobolOneEdge(t, nested, nest+"::OUTERPGM/INNERPGM", book+"CPYI", graph.EdgeImports)
	})

	t.Run("never_regex_namespaces", func(t *testing.T) {
		for _, edge := range result.Edges {
			if edge.Kind == graph.EdgeImports {
				assert.Regexp(t, `^unresolved::cobol::copybook::`, edge.To)
			}
		}
	})
}

func TestCobolGrammar_CopybookKey(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/demo.cbl", cobolDemoFixture)
	programs := cobolProgramNodes(result)
	require.Len(t, programs, 1)
	assert.Equal(t, "DEMOPGM", programs[0].Meta["cobol_name_key"])
	assert.Equal(t, cobolNameNormRule, programs[0].Meta["cobol_name_norm"])

	for _, path := range []string{"copy/custrec.cpy", "COPY/CUSTREC.CPY"} {
		t.Run(path, func(t *testing.T) {
			file := cobolFileNode(t, extractCobolGrammar(t, e, path, cobolCopybookFixture), path)
			assert.Equal(t, "copybook", file.Meta["cobol_kind"])
			assert.Equal(t, "CUSTREC", file.Meta["cobol_name_key"])
			assert.Equal(t, cobolNameNormRule, file.Meta["cobol_name_norm"])
		})
	}
}
