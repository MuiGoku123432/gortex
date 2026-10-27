//go:build darwin || linux

package languages

import (
	"errors"
	"strings"
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
// keys, and the member's own content ID (D-16). Callers assert
// prov_analysis_absence themselves, since a degraded member that mapped
// nothing carries it (review WR-04).
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
			assert.NotContains(t, file.Meta, "prov_analysis_absence")

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
	assert.NotContains(t, file.Meta, "prov_analysis_absence")
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
		file := assertCobolCopybookFile(t, result, path, cobolCopybookIncludeFormsFixture)
		assert.NotContains(t, file.Meta, "prov_analysis_absence")
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

// assertCobolNoWrapper checks that no node of result is a program or
// division and that no node or edge names the producer's synthetic wrapper
// program CPYWRAP (D-16).
func assertCobolNoWrapper(t *testing.T, result *parser.ExtractionResult) {
	t.Helper()
	for _, n := range result.Nodes {
		assert.NotContains(t, []any{"program", "division"}, n.Meta["cobol_kind"], n.ID)
		assert.NotContains(t, n.ID, "CPYWRAP")
	}
	for _, e := range result.Edges {
		assert.NotContains(t, e.From+" "+e.To, "CPYWRAP")
	}
}

// A procedure member's paragraph and statement are first-class, owned by the
// copybook at the member's own rows (D-16, D-01, D-05, D-17): the file
// defines the paragraph, which is member_of nothing, and the statement is
// member_of the paragraph.
func TestCobolGrammar_CopybookProcedureFragment(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "copy/valpara.cpy"
	result := extractCobolGrammar(t, e, path, cobolCopybookProcedureFixture)
	file := assertCobolCopybookFile(t, result, path, cobolCopybookProcedureFixture)
	assert.Equal(t, "VALPARA", file.Meta["cobol_name_key"])
	assert.NotContains(t, file.Meta, "prov_analysis_absence")
	assertCobolNoWrapper(t, result)
	require.Len(t, result.Nodes, 3+len(cobolFindingNodes(result, "")), "three non-finding nodes")

	const para = path + "::@proc/VALIDATE-PARA"
	const stmt = para + "/@move_statement"
	p := cobolNodeByID(t, result, para)
	assert.Equal(t, graph.KindFunction, p.Kind)
	assert.Equal(t, "paragraph", p.Meta["cobol_kind"])
	assert.Equal(t, 1, p.Meta["cobol_ordinal"])
	assert.Equal(t, 0, p.Meta["prov_start_row"])
	assert.Equal(t, 1, p.Meta["prov_end_row"])
	assert.Equal(t, true, p.Meta["prov_range_exact"])
	s := cobolNodeByID(t, result, stmt)
	assert.Equal(t, graph.KindLocal, s.Kind)
	assert.Equal(t, "move_statement", s.Meta["cobol_statement_kind"])
	for _, n := range []*graph.Node{p, s} {
		for _, key := range cobolDocumentKeys {
			assert.Contains(t, n.Meta, key, n.ID)
		}
		assert.Equal(t, "cobol-grade-policy-copybook-v1", n.Meta["prov_grade_policy"], n.ID)
	}

	assert.Len(t, cobolEdges(result, path, para, graph.EdgeDefines), 1)
	for _, edge := range result.Edges {
		if edge.Kind == graph.EdgeMemberOf {
			assert.Equal(t, stmt+" -> "+para, edge.From+" -> "+edge.To, "the only member_of edge")
		}
	}
	assert.Len(t, cobolEdges(result, stmt, para, graph.EdgeMemberOf), 1)
}

// cobolCopybookRootFixture is an invented procedure member whose first
// statements come before any paragraph header.
const cobolCopybookRootFixture = "           MOVE 1 TO WS-X.\n" +
	"           PERFORM CHECK-PARA.\n" +
	"       CHECK-PARA.\n" +
	"           EXIT.\n"

// A statement before any header is <path>::@proc/@<kind>, member_of the
// copybook file node, and a transfer from there flows From the file node
// (D-16, D-17).
func TestCobolGrammar_CopybookProcedureRoot(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "copy/root.cpy"
	result := extractCobolGrammar(t, e, path, cobolCopybookRootFixture)
	assertCobolCopybookFile(t, result, path, cobolCopybookRootFixture)
	assertCobolNoWrapper(t, result)

	for _, kind := range []string{"move_statement", "perform_statement_call_proc"} {
		id := path + "::@proc/@" + kind
		assert.Equal(t, graph.KindLocal, cobolNodeByID(t, result, id).Kind, id)
		assert.Len(t, cobolEdges(result, id, path, graph.EdgeMemberOf), 1, id)
	}
	perform := cobolOneEdge(t, result, path, path+"::@proc/CHECK-PARA", graph.EdgeCalls)
	assertCobolRelationProvenance(t, perform, "perform_statement_call_proc")
	assert.Equal(t, "perform", perform.Meta["cobol_flow"])
	assert.Equal(t, "@proc/@perform_statement_call_proc", perform.Meta["cobol_statement_symbol"])
	assert.Equal(t, 1, perform.Meta["prov_start_row"])
}

// cobolCopybookFlowFixture is an invented procedure member with a section,
// a THRU range, a GO TO, and labels the member does not define.
const cobolCopybookFlowFixture = "       MAIN-SECT SECTION.\n" +
	"       P1.\n" +
	"           PERFORM P2 THRU P3.\n" +
	"           GO TO P3.\n" +
	"           PERFORM OTHER-PARA.\n" +
	"           PERFORM P2 THRU NOPE.\n" +
	"       P2.\n" +
	"           EXIT.\n" +
	"       P3.\n" +
	"           EXIT.\n"

// PERFORM, PERFORM THRU, and GO TO in a copybook resolve over the member's
// own headers only (D-02, D-03, D-04); a label the member does not define
// is an owned unresolved target with label_not_in_copybook (D-12).
func TestCobolGrammar_CopybookProcedureFlow(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "copy/flow.cpy"
	result := extractCobolGrammar(t, e, path, cobolCopybookFlowFixture)
	assertCobolCopybookFile(t, result, path, cobolCopybookFlowFixture)
	assertCobolNoWrapper(t, result)

	const sect = path + "::@proc/MAIN-SECT"
	s := cobolNodeByID(t, result, sect)
	assert.Equal(t, graph.KindFunction, s.Kind)
	assert.Equal(t, "section", s.Meta["cobol_kind"])
	assert.Equal(t, "procedure", s.Meta["cobol_section_kind"])
	for _, edge := range result.Edges {
		assert.False(t, edge.Kind == graph.EdgeMemberOf && edge.From == sect, "section member_of %s", edge.To)
	}
	for _, name := range []string{"P1", "P2", "P3"} {
		id := sect + "/" + name
		assert.Equal(t, "paragraph", cobolNodeByID(t, result, id).Meta["cobol_kind"], id)
		assert.Len(t, cobolEdges(result, id, sect, graph.EdgeMemberOf), 1, id)
	}

	p1 := sect + "/P1"
	flows := func(to, flow string) []*graph.Edge {
		var out []*graph.Edge
		for _, edge := range cobolEdges(result, p1, to, graph.EdgeCalls) {
			if edge.Meta["cobol_flow"] == flow {
				out = append(out, edge)
			}
		}
		return out
	}
	for _, to := range []string{sect + "/P2", sect + "/P3"} {
		require.Len(t, flows(to, "perform_thru"), 1, to)
		assert.NotContains(t, flows(to, "perform_thru")[0].Meta, "cobol_unresolved_reason", to)
	}
	require.Len(t, flows(sect+"/P3", "goto"), 1)

	const label = "unresolved::cobol::label::" + path + "::"
	other := flows(label+"OTHER-PARA", "perform")
	require.Len(t, other, 1)
	assert.Equal(t, "label_not_in_copybook", other[0].Meta["cobol_unresolved_reason"])
	thru := flows(label+"P2::thru::NOPE", "perform_thru")
	require.Len(t, thru, 1)
	assert.Equal(t, "label_not_in_copybook", thru[0].Meta["cobol_unresolved_reason"])
	assert.Equal(t, "P2", thru[0].Meta["cobol_thru_start"])
	assert.Equal(t, "NOPE", thru[0].Meta["cobol_thru_end"])

	for _, edge := range result.Edges {
		if edge.Kind == graph.EdgeCalls {
			inside := strings.HasPrefix(edge.To, path+"::") || strings.HasPrefix(edge.To, label)
			assert.True(t, inside, "calls edge leaves the member: %s", edge.To)
		}
	}
}

// cobolCopybookCallsFixture is an invented procedure member with a literal
// CALL, a dynamic CALL, and a COPY.
const cobolCopybookCallsFixture = "       CALL-PARA.\n" +
	"           CALL 'SUBPGM'.\n" +
	"           CALL WS-PGM.\n" +
	"           COPY INNERP.\n" +
	"           GOBACK.\n"

// CALL and COPY in a copybook paragraph come From the paragraph (D-09, D-10,
// D-11); dynamic targets are keyed under the copybook path.
func TestCobolGrammar_CopybookProcedureCalls(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "copy/calls.cpy"
	result := extractCobolGrammar(t, e, path, cobolCopybookCallsFixture)
	assertCobolCopybookFile(t, result, path, cobolCopybookCallsFixture)
	assertCobolNoWrapper(t, result)
	const para = path + "::@proc/CALL-PARA"

	literal := cobolOneEdge(t, result, para, "unresolved::cobol::program::SUBPGM", graph.EdgeCalls)
	assertCobolRelationProvenance(t, literal, "call_statement")
	assert.Equal(t, "literal", literal.Meta["cobol_call"])
	assert.NotContains(t, literal.Meta, "cobol_unresolved_reason")

	dynamic := cobolOneEdge(t, result, para, "unresolved::cobol::dynamic::program::"+path+"::WS-PGM", graph.EdgeCalls)
	assert.Equal(t, "dynamic", dynamic.Meta["cobol_call"])
	assert.Equal(t, "dynamic", dynamic.Meta["cobol_unresolved_reason"])
	data := cobolOneEdge(t, result, para, "unresolved::cobol::data::"+path+"::WS-PGM", graph.EdgeReferences)
	assert.Equal(t, "call_target", data.Meta["cobol_ref_role"])
	assert.Equal(t, "not_declared_in_copybook", data.Meta["cobol_unresolved_reason"])

	include := cobolOneEdge(t, result, para, "unresolved::cobol::copybook::INNERP", graph.EdgeImports)
	assertCobolRelationProvenance(t, include, "copy_statement")
	assert.Equal(t, "@proc/CALL-PARA/@copy_statement", include.Meta["cobol_statement_symbol"])
}

// cobolCopybookMainframeFixture is an invented procedure member with IDMS,
// CICS, and DB2 statements.
const cobolCopybookMainframeFixture = "       MAIN-PARA.\n" +
	"           OBTAIN CALC EMP-REC.\n" +
	"           EXEC CICS LINK PROGRAM('LNKPGM') END-EXEC.\n" +
	"           EXEC CICS RETURN TRANSID(WS-TRN) END-EXEC.\n" +
	"           EXEC SQL SELECT A INTO :WS-A FROM MYSCH.ORDERS AS O\n" +
	"           END-EXEC.\n" +
	"           EXEC SQL PREPARE S1 FROM :WS-SQLTXT END-EXEC.\n" +
	"           EXEC SQL FETCH C1 INTO :WS-A END-EXEC.\n"

// cobolCopybookRootSQLFixture is an invented procedure member whose IDMS
// and SQL statements come before any paragraph header.
const cobolCopybookRootSQLFixture = "           MOVE 1 TO WS-X.\n" +
	"           OBTAIN CALC EMP-REC.\n" +
	"           EXEC SQL DELETE FROM AUDIT_LOG END-EXEC.\n"

// IDMS, CICS, and EXEC SQL in a procedure copybook emit the same edges as
// in a program (D-13, D-14, D-15, DET-07..DET-12), From the paragraph or,
// before any header, the copybook file node; dynamic operands are keyed
// under the copybook path, and no alias, statement name, host variable, or
// include becomes a table.
func TestCobolGrammar_CopybookProcedureMainframe(t *testing.T) {
	e := NewCobolGrammarExtractor()

	t.Run("member", func(t *testing.T) {
		const path = "copy/mf.cpy"
		result := extractCobolGrammar(t, e, path, cobolCopybookMainframeFixture)
		assertCobolCopybookFile(t, result, path, cobolCopybookMainframeFixture)
		assertCobolNoWrapper(t, result)
		const para = path + "::@proc/MAIN-PARA"

		record := cobolOneEdge(t, result, para, "idms::_unqualified/record/EMP-REC", graph.EdgeReads)
		assert.Equal(t, "OBTAIN", record.Meta["idms_verb"])
		assertCobolResource(t, result, "idms::_unqualified/record/EMP-REC", "EMP-REC", "idms_record")

		link := cobolOneEdge(t, result, para, "unresolved::cobol::program::LNKPGM", graph.EdgeCalls)
		assert.Equal(t, "literal", link.Meta["cobol_call"])
		assert.Equal(t, "LINK", link.Meta["cics_command"])

		trn := cobolOneEdge(t, result, para, "unresolved::cobol::dynamic::transaction::"+path+"::WS-TRN", graph.EdgeReferences)
		assert.Equal(t, "dynamic", trn.Meta["cobol_unresolved_reason"])
		assert.Equal(t, "RETURN", trn.Meta["cics_command"])

		orders := cobolOneEdge(t, result, para, "db2::MYSCH/ORDERS", graph.EdgeReads)
		assert.Equal(t, "SELECT", orders.Meta["sql_verb"])
		assert.Equal(t, []string{"O"}, orders.Meta["sql_aliases"])
		assertCobolTable(t, result, "db2::MYSCH/ORDERS", "ORDERS", "MYSCH")

		prepare := cobolOneEdge(t, result, para, "unresolved::cobol::dynamic::sql::"+path+"::WS-SQLTXT", graph.EdgeReferences)
		assert.Equal(t, "dynamic", prepare.Meta["cobol_unresolved_reason"])

		// WR-09: the including program may declare the cursor.
		var fetch *graph.Node
		for _, n := range result.Nodes {
			if n.Meta["sql_cursor"] == "C1" {
				fetch = n
			}
		}
		require.NotNil(t, fetch, "FETCH C1 statement node")
		assert.Equal(t, "cursor_not_in_copybook", fetch.Meta["cobol_unresolved_reason"])
		assert.Equal(t, path, link.Meta["cobol_caller_program"], "a copybook CALL names the copybook, never a nested container")

		assert.Equal(t, []string{"db2::MYSCH/ORDERS"}, cobolTableIDs(result))
		for _, n := range result.Nodes {
			assert.NotContains(t, []string{"O", "S1", "WS-SQLTXT", "SQLCA"}, n.Name, n.ID)
			assertCobolMetaContract(t, n.ID, n.Meta)
		}
		for _, edge := range result.Edges {
			if edge.Meta != nil {
				assertCobolMetaContract(t, edge.From+" -> "+edge.To, edge.Meta)
			}
		}
	})

	t.Run("root_sql", func(t *testing.T) {
		const path = "copy/rootsql.cpy"
		result := extractCobolGrammar(t, e, path, cobolCopybookRootSQLFixture)
		assertCobolCopybookFile(t, result, path, cobolCopybookRootSQLFixture)
		assertCobolNoWrapper(t, result)

		audit := cobolOneEdge(t, result, path, "db2::_unqualified/AUDIT_LOG", graph.EdgeWrites)
		assert.Equal(t, "DELETE", audit.Meta["sql_verb"])
		assert.Equal(t, "@proc/@exec_sql_statement", audit.Meta["cobol_statement_symbol"])

		record := cobolOneEdge(t, result, path, "idms::_unqualified/record/EMP-REC", graph.EdgeReads)
		assert.Equal(t, "OBTAIN", record.Meta["idms_verb"])
	})
}

// cobolCopybookRedefinesFixture is an invented data member whose REDEFINES
// names one item it declares and one it does not.
const cobolCopybookRedefinesFixture = "       01 CPY-A PIC X.\n" +
	"       01 CPY-B REDEFINES NOPE PIC X.\n" +
	"       01 CPY-C REDEFINES CPY-A PIC X.\n"

// A data name a copybook references but does not declare stays unresolved
// under the copybook path with not_declared_in_copybook, since an including
// program may declare it (D-06, D-12); a name the member declares binds.
func TestCobolGrammar_CopybookDataNames(t *testing.T) {
	e := NewCobolGrammarExtractor()

	t.Run("procedure_member", func(t *testing.T) {
		const path = "copy/mf.cpy"
		result := extractCobolGrammar(t, e, path, cobolCopybookMainframeFixture)
		const para = path + "::@proc/MAIN-PARA"
		for name, role := range map[string]string{"WS-TRN": "cics_transaction", "WS-SQLTXT": "sql_dynamic_source"} {
			ref := cobolOneEdge(t, result, para, "unresolved::cobol::data::"+path+"::"+name, graph.EdgeReferences)
			assert.Equal(t, role, ref.Meta["cobol_ref_role"], name)
			assert.Equal(t, "not_declared_in_copybook", ref.Meta["cobol_unresolved_reason"], name)
		}
	})

	t.Run("data_member", func(t *testing.T) {
		const path = "copy/redef.cpy"
		result := extractCobolGrammar(t, e, path, cobolCopybookRedefinesFixture)
		assertCobolCopybookFile(t, result, path, cobolCopybookRedefinesFixture)

		missing := cobolOneEdge(t, result, path+"::CPY-B", "unresolved::cobol::data::"+path+"::NOPE", graph.EdgeReferences)
		assert.Equal(t, "redefines", missing.Meta["cobol_ref_role"])
		assert.Equal(t, "not_declared_in_copybook", missing.Meta["cobol_unresolved_reason"])

		bound := cobolOneEdge(t, result, path+"::CPY-C", path+"::CPY-A", graph.EdgeReferences)
		assert.Equal(t, "redefines", bound.Meta["cobol_ref_role"])
		assert.NotContains(t, bound.Meta, "cobol_unresolved_reason")
	})
}

func TestCobolGrammar_CopybookReplacingDegrades(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "copy/tagrec.cpy", cobolCopybookPseudoTextFixture)
	file := assertCobolCopybookFile(t, result, "copy/tagrec.cpy", cobolCopybookPseudoTextFixture)
	assert.NotEqual(t, "green", file.Meta["prov_document_grade"])
	assert.NotEmpty(t, file.Meta["prov_document_grade_reasons"])
	assert.Equal(t, "copybook_content_unmapped", file.Meta["prov_analysis_absence"])
}

// cobolCopybookExecCICSFixture is an invented procedure member that opens
// with EXEC CICS, which the producer wraps with its data header (WR-04).
const cobolCopybookExecCICSFixture = "           EXEC CICS RETURN TRANSID('T1') END-EXEC.\n"

// cobolCopybookFDFixture is an invented FILE SECTION member, which the
// producer wraps with its procedure header (WR-04).
const cobolCopybookFDFixture = "       FD CUST-FILE.\n" +
	"       01 CUST-LINE PIC X(80).\n"

// The producer picks a member's wrapper from its first code line and
// misclassifies some members (review WR-04). Gortex cannot re-wrap them, so
// the file node records the producer's choice in cobol_copybook_wrapper and,
// when a degraded analysis mapped nothing, prov_analysis_absence
// copybook_content_unmapped; a member that mapped something, or graded
// green, carries no absence.
func TestCobolGrammar_CopybookWrapperWindow(t *testing.T) {
	e := NewCobolGrammarExtractor()
	// grade "" means any non-green grade; imports is the imports edge count.
	for _, tc := range []struct {
		name, path, src, wrapper, grade string
		unmapped                        bool
		imports                         int
	}{
		{"exec_cics_member", "copy/cicsret.cpy", cobolCopybookExecCICSFixture, "data", "red", true, 0},
		{"fd_member", "copy/custfd.cpy", cobolCopybookFDFixture, "procedure", "red", true, 0},
		{"leading_copy_member", "copy/leadcpy.cpy", cobolCopybookLeadingCopyFixture, "procedure", "", false, 1},
		{"pseudo_text_member", "copy/tagrec.cpy", cobolCopybookPseudoTextFixture, "data", "", true, 0},
		{"data_member", "copy/custrec.cpy", cobolCopybookFixture, "data", "green", false, 0},
		{"procedure_member", "copy/valpara.cpy", cobolCopybookProcedureFixture, "procedure", "green", false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := extractCobolGrammar(t, e, tc.path, tc.src)
			file := assertCobolCopybookFile(t, result, tc.path, tc.src)
			assert.Equal(t, tc.wrapper, file.Meta["cobol_copybook_wrapper"])
			if tc.grade == "" {
				assert.NotEqual(t, "green", file.Meta["prov_document_grade"])
			} else {
				assert.Equal(t, tc.grade, file.Meta["prov_document_grade"])
			}
			assert.Len(t, cobolCopybookImports(t, result, tc.path), tc.imports)
			if !tc.unmapped {
				assert.NotContains(t, file.Meta, "prov_analysis_absence")
				return
			}
			assert.Equal(t, "copybook_content_unmapped", file.Meta["prov_analysis_absence"])
			// Only the file node and its parse findings; every edge is a
			// finding's member_of edge.
			findings := cobolFindingIDs(result)
			assert.Len(t, result.Nodes, 1+len(findings))
			for _, edge := range result.Edges {
				assert.True(t, findings[edge.From], "edge %s -> %s is not from a finding", edge.From, edge.To)
			}
		})
	}
}
