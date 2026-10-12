//go:build darwin || linux

package languages

import (
	"errors"
	"testing"

	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// Every fixture in this file is invented 7-column fixed-format COBOL (D-14).
// None may be replaced with estate-derived source: this repository is public.

// cobolCopybookProcedureFixture is an invented procedure-fragment member.
const cobolCopybookProcedureFixture = "       VALIDATE-PARA.\n" +
	"           MOVE 1 TO WS-X.\n"

// cobolCopybookPseudoTextFixture is an invented member written with the
// :TAG: pseudo-text a COPY ... REPLACING would substitute.
const cobolCopybookPseudoTextFixture = "       01 :TAG:-REC PIC X.\n" +
	"           05 :TAG:-A PIC 9.\n"

// cobolCopybookIncludeFormsFixture is an invented data member that writes
// every COPY form: Endevor -INC, COPY IDMS, EXEC SQL INCLUDE, and COPY OF.
const cobolCopybookIncludeFormsFixture = "       01 INC-REC.\n" +
	"           05 INC-A PIC X.\n" +
	"-INC INCMEM\n" +
	"       COPY IDMS RECORD EMP-REC.\n" +
	"           EXEC SQL INCLUDE SQLCA END-EXEC.\n" +
	"       COPY CPYB OF MYLIB.\n" +
	"       01 INC-TAIL PIC X.\n"

// cobolCopybookLeadingCopyFixture is an invented member whose first code
// line is a COPY, which the producer wraps with its procedure header.
const cobolCopybookLeadingCopyFixture = "       COPY CUSTREC.\n" +
	"       01 CUST-X PIC X.\n"

// cobolCopybookImports returns result's imports edges by target, asserting
// each comes from the copybook file node path and no target repeats.
func cobolCopybookImports(t *testing.T, result *parser.ExtractionResult, path string) map[string]*graph.Edge {
	t.Helper()
	imports := map[string]*graph.Edge{}
	for _, edge := range result.Edges {
		if edge.Kind != graph.EdgeImports {
			continue
		}
		assert.Equal(t, path, edge.From, "imports edge to %s", edge.To)
		require.NotContains(t, imports, edge.To, "two imports edges to %s", edge.To)
		imports[edge.To] = edge
	}
	return imports
}

// assertCobolCopybookFile checks the copybook file node of result: the full
// document provenance under the copybook grade policy, the COPY binding
// keys, the member's own content ID, and no analysis-absence marker (D-16).
func assertCobolCopybookFile(t *testing.T, result *parser.ExtractionResult, path, src string) *graph.Node {
	t.Helper()
	file := cobolFileNode(t, result, path)
	for _, key := range cobolDocumentKeys {
		assert.Contains(t, file.Meta, key)
	}
	assert.Equal(t, "copybook", file.Meta["cobol_kind"])
	assert.Equal(t, cobolNameNormRule, file.Meta["cobol_name_norm"])
	assert.Equal(t, "cobol-grade-policy-copybook-v1", file.Meta["prov_grade_policy"])
	assert.Equal(t, string(preprocessor.NewContentID([]byte(src))), file.Meta["prov_revision_content_id"])
	assert.NotContains(t, file.Meta, "prov_analysis_absence")
	return file
}

func TestCobolGrammar_CopybookDataItems(t *testing.T) {
	e := NewCobolGrammarExtractor()
	for _, path := range []string{"copy/custrec.cpy", "COPY/CUSTREC.CPY"} {
		t.Run(path, func(t *testing.T) {
			result := extractCobolGrammar(t, e, path, cobolCopybookFixture)
			file := assertCobolCopybookFile(t, result, path, cobolCopybookFixture)
			assert.Equal(t, "CUSTREC", file.Meta["cobol_name_key"])
			assert.Equal(t, "green", file.Meta["prov_document_grade"])

			// The wrapper's program, divisions, and sections never surface.
			nodes := map[string]*graph.Node{}
			for _, n := range result.Nodes {
				if n.Kind != graph.KindFile {
					assert.Equal(t, "data_item", n.Meta["cobol_kind"], n.ID)
					nodes[n.ID] = n
				}
			}
			rec := path + "::CUST-REC"
			for row, tc := range []struct {
				id       string
				kind     graph.NodeKind
				memberOf string
			}{
				{rec, graph.KindVariable, ""},
				{rec + "/CUST-ID", graph.KindField, rec},
				{rec + "/CUST-NAME", graph.KindField, rec},
			} {
				n := nodes[tc.id]
				require.NotNil(t, n, "missing %s", tc.id)
				assert.Equal(t, tc.kind, n.Kind, tc.id)
				assert.Equal(t, row, n.Meta["prov_start_row"], tc.id)
				assert.Equal(t, true, n.Meta["prov_range_exact"], tc.id)
				assert.Equal(t, "cobol-grade-policy-copybook-v1", n.Meta["prov_grade_policy"], tc.id)
				assert.NotContains(t, n.Meta, "cobol_name_key", "only the file node binds a COPY")
				var defined bool
				var memberOf []string
				for _, edge := range result.Edges {
					switch {
					case edge.To == tc.id && edge.Kind == graph.EdgeDefines:
						defined = edge.From == path
					case edge.From == tc.id && edge.Kind == graph.EdgeMemberOf:
						memberOf = append(memberOf, edge.To)
					}
				}
				assert.True(t, defined, "%s has no defines edge from the copybook", tc.id)
				if tc.memberOf == "" {
					assert.Empty(t, memberOf, tc.id)
				} else {
					assert.Equal(t, []string{tc.memberOf}, memberOf, tc.id)
				}
			}
			assert.Len(t, nodes, 3)
		})
	}
}

// cobolCopybookNestedCopyFixture is an invented member that copies two more
// members: one inside an open group, one after a standalone 77 item.
const cobolCopybookNestedCopyFixture = "       01 CUST-REC.\n" +
	"           05 CUST-ADDR.\n" +
	"           COPY ADDRREC.\n" +
	"       77 CUST-FLAG PIC X.\n" +
	"       COPY TAILREC.\n" +
	"       01 CUST-TAIL PIC X.\n"

// A COPY inside a copybook marks the innermost open item, else the copybook
// file node, and the mark never leaks into another item's Meta. Each COPY is
// also one imports edge from the file node (D-11) that keeps the member as
// written and the statement's original range, and never carries the
// file-node-only keys.
func TestCobolGrammar_CopybookNestedCopy(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "copy/custall.cpy"
	result := extractCobolGrammar(t, e, path, cobolCopybookNestedCopyFixture)
	file := assertCobolCopybookFile(t, result, path, cobolCopybookNestedCopyFixture)
	assert.Equal(t, true, file.Meta["cobol_contains_copy"])
	marked := map[string]bool{}
	for _, n := range result.Nodes {
		if n.Kind != graph.KindFile && n.Meta["cobol_contains_copy"] == true {
			marked[n.ID] = true
		}
	}
	assert.Equal(t, map[string]bool{path + "::CUST-REC/CUST-ADDR": true}, marked)

	imports := cobolCopybookImports(t, result, path)
	require.Len(t, imports, 2)
	for member, row := range map[string]int{"ADDRREC": 2, "TAILREC": 4} {
		edge := imports["unresolved::cobol::copybook::"+member]
		require.NotNil(t, edge, member)
		assertCobolRelationProvenance(t, edge, "copy_statement")
		assert.Equal(t, "copy", edge.Meta["cobol_copy_form"], member)
		assert.Equal(t, member, edge.Meta["cobol_copy_member_raw"], member)
		assert.Equal(t, cobolNameNormRule, edge.Meta["cobol_name_norm"], member)
		assert.Equal(t, true, edge.Meta["prov_range_exact"], member)
		assert.Equal(t, row, edge.Meta["prov_start_row"], member)
		for _, key := range []string{"prov_start_column", "prov_end_row", "prov_end_column", "prov_start_byte", "prov_end_byte"} {
			assert.Contains(t, edge.Meta, key, member)
		}
		for _, key := range []string{"cobol_kind", "cobol_name_key", "cobol_contains_copy"} {
			assert.NotContains(t, edge.Meta, key, member)
		}
	}
}

// Every COPY form written in a copybook is an imports edge from the file
// node with the 03-07 COPY vocabulary (D-11, D-19), including a COPY on the
// member's first code line (review WR-04).
func TestCobolGrammar_CopybookIncludeForms(t *testing.T) {
	e := NewCobolGrammarExtractor()

	t.Run("data_member", func(t *testing.T) {
		const path = "copy/increc.cpy"
		result := extractCobolGrammar(t, e, path, cobolCopybookIncludeFormsFixture)
		assertCobolCopybookFile(t, result, path, cobolCopybookIncludeFormsFixture)
		imports := cobolCopybookImports(t, result, path)
		require.Len(t, imports, 4)

		inc := imports["unresolved::cobol::copybook::INCMEM"]
		require.NotNil(t, inc, "-INC INCMEM")
		assert.Equal(t, "inc", inc.Meta["cobol_copy_form"])
		assert.Equal(t, "INCMEM", inc.Meta["cobol_copy_member_raw"])
		assert.Equal(t, false, inc.Meta["prov_range_exact"], "the Endevor rewrite is an inexact projection")
		assert.Equal(t, 2, inc.Meta["prov_start_row"])
		assert.Equal(t, 3, inc.Line)

		idms := imports["unresolved::cobol::copybook::idms/RECORD/EMP-REC"]
		require.NotNil(t, idms, "COPY IDMS RECORD EMP-REC")
		assertCobolRelationProvenance(t, idms, "copy_statement")
		assert.Equal(t, "idms", idms.Meta["cobol_copy_form"])
		assert.Equal(t, "RECORD", idms.Meta["cobol_copy_idms_kind"])
		assert.Equal(t, "EMP-REC", idms.Meta["cobol_copy_member_raw"])
		assert.Equal(t, "idms_dictionary_member", idms.Meta["cobol_unresolved_reason"])

		sql := imports["unresolved::cobol::copybook::SQLCA"]
		require.NotNil(t, sql, "EXEC SQL INCLUDE SQLCA")
		assertCobolRelationProvenance(t, sql, "exec_sql_include")
		assert.Equal(t, "sql_include", sql.Meta["cobol_copy_form"])
		assert.Equal(t, true, sql.Meta["sql_include_builtin"])

		lib := imports["unresolved::cobol::copybook::CPYB"]
		require.NotNil(t, lib, "COPY CPYB OF MYLIB")
		assertCobolRelationProvenance(t, lib, "copy_statement")
		assert.Equal(t, "copy", lib.Meta["cobol_copy_form"])
		assert.Equal(t, "CPYB", lib.Meta["cobol_copy_member_raw"])
		assert.Equal(t, "MYLIB", lib.Meta["cobol_copy_library"])
		assert.NotContains(t, lib.Meta, "cobol_unresolved_reason")
	})

	t.Run("leading_copy_member", func(t *testing.T) {
		const path = "copy/leadcpy.cpy"
		result := extractCobolGrammar(t, e, path, cobolCopybookLeadingCopyFixture)
		file := assertCobolCopybookFile(t, result, path, cobolCopybookLeadingCopyFixture)
		assert.NotEqual(t, "green", file.Meta["prov_document_grade"])
		imports := cobolCopybookImports(t, result, path)
		require.Len(t, imports, 1)
		edge := imports["unresolved::cobol::copybook::CUSTREC"]
		require.NotNil(t, edge, "COPY CUSTREC")
		assert.Equal(t, "copy", edge.Meta["cobol_copy_form"])
		assert.Equal(t, "CUSTREC", edge.Meta["cobol_copy_member_raw"])
	})
}

// A copybook's handoff passes the same approved-grammar check as a program's
// (T-03-41): a wrong pin fails it and returns no result.
func TestCobolGrammar_CopybookGrammarPinned(t *testing.T) {
	e := &CobolGrammarExtractor{approvedGrammarID: cobolWrongGrammarID, approvedPreprocessorModule: cobolApprovedPreprocessorModule}
	result, err := e.Extract("copy/custrec.cpy", []byte(cobolCopybookFixture))
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, errCobolGrammarNotApproved), "err = %v", err)
}

// A procedure-fragment member still yields its copybook file node with the
// COPY binding keys (D-16). Plan 03-16 asserts its procedure structure.
func TestCobolGrammar_CopybookProcedureFragment(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "copy/valpara.cpy", cobolCopybookProcedureFixture)
	file := assertCobolCopybookFile(t, result, "copy/valpara.cpy", cobolCopybookProcedureFixture)
	assert.Equal(t, "VALPARA", file.Meta["cobol_name_key"])
}

func TestCobolGrammar_CopybookReplacingDegrades(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "copy/tagrec.cpy", cobolCopybookPseudoTextFixture)
	file := assertCobolCopybookFile(t, result, "copy/tagrec.cpy", cobolCopybookPseudoTextFixture)
	assert.NotEqual(t, "green", file.Meta["prov_document_grade"])
	assert.NotEmpty(t, file.Meta["prov_document_grade_reasons"])
}
