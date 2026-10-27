//go:build darwin || linux

package languages

import (
	"strconv"
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

const cobolSQLFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SQLPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A PIC X(10).
       PROCEDURE DIVISION.
       P1.
           EXEC SQL SELECT A INTO :WS-A FROM MYSCH.AUDIT_LOG END-EXEC.
           EXEC SQL SELECT A FROM CUST END-EXEC.
           EXEC SQL SELECT A FROM "MixedCase" END-EXEC.
           EXEC SQL SELECT A FROM "A/B"."x""y" END-EXEC.
           EXEC SQL SELECT A FROM LOC1.SCH.TB_X END-EXEC.
           exec sql select a from cust end-exec.
       P3.
           EXEC SQL INSERT INTO T1 SELECT X FROM T2 END-EXEC.
           EXEC SQL UPDATE T6 SET A = 1 END-EXEC.
           EXEC SQL DELETE FROM T7 END-EXEC.
           EXEC SQL MERGE INTO T3 USING T4 ON A = B END-EXEC.
       P4.
           EXEC SQL SELECT A FROM CUST AS C1 JOIN CUST AS C2 END-EXEC.
           EXEC SQL SELECT A FROM (SELECT B FROM T5) AS D1 END-EXEC.
       P5.
           EXEC SQL WITH RECENT AS (SELECT * FROM ORDERS)
               SELECT * FROM RECENT END-EXEC.
           GOBACK.
`

// cobolSQLCursorFixture declares its cursors in the procedure division: the
// pinned grammar admits no EXEC SQL statement in WORKING-STORAGE (it parses
// as an ERROR and grades the document red).
const cobolSQLCursorFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. CURPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-X PIC X(10).
       01 WS-N PIC S9(4) COMP.
       01 WS-SQLTXT PIC X(200).
       PROCEDURE DIVISION.
       P0.
           EXEC SQL DECLARE C1 CURSOR FOR SELECT X FROM ORDERS
           END-EXEC.
           EXEC SQL DECLARE C_EMP CURSOR FOR SELECT X FROM EMP END-EXEC.
           EXEC SQL DECLARE C2 CURSOR FOR SELECT X FROM T8 END-EXEC.
           EXEC SQL DECLARE C2 CURSOR FOR SELECT X FROM T9 END-EXEC.
       P2.
           EXEC SQL OPEN C1 END-EXEC.
           EXEC SQL FETCH C1 INTO :WS-X END-EXEC.
           EXEC SQL FETCH NEXT FROM C1 INTO :WS-X END-EXEC.
           EXEC SQL FETCH ABSOLUTE :WS-N FROM C1 INTO :WS-X END-EXEC.
           EXEC SQL FETCH C_EMP INTO :WS-X END-EXEC.
           EXEC SQL CLOSE C1 END-EXEC.
       P3.
           EXEC SQL FETCH C9 INTO :WS-X END-EXEC.
           EXEC SQL FETCH C2 INTO :WS-X END-EXEC.
       P4.
           EXEC SQL PREPARE S1 FROM :WS-SQLTXT END-EXEC.
           EXEC SQL EXECUTE S1 END-EXEC.
           EXEC SQL EXECUTE IMMEDIATE :WS-SQLTXT END-EXEC.
       P5.
           EXEC SQL CREATE TABLE NEWT (A INT) END-EXEC.
           EXEC SQL ALTER TABLE S.T10 ADD B INT END-EXEC.
           EXEC SQL DROP TABLE T11 END-EXEC.
           GOBACK.
`

// cobolTableIDs returns the IDs of the KindTable nodes in emission order.
func cobolTableIDs(result *parser.ExtractionResult) []string {
	var ids []string
	for _, n := range result.Nodes {
		if n.Kind == graph.KindTable {
			ids = append(ids, n.ID)
		}
	}
	return ids
}

// assertCobolTable checks the DB2 table node contract: one node per
// document, Name and table the normalized key, schema normalized and only
// when written, dialect db2, the name rule, and the writer-independent
// shared-resource provenance.
func assertCobolTable(t *testing.T, result *parser.ExtractionResult, id, table, schema string) {
	t.Helper()
	var found []*graph.Node
	for _, n := range result.Nodes {
		if n.ID == id {
			found = append(found, n)
		}
	}
	require.Len(t, found, 1, "table node %s", id)
	n := found[0]
	assert.Equal(t, graph.KindTable, n.Kind)
	assert.Equal(t, table, n.Name)
	assert.Equal(t, table, n.Meta["table"])
	assert.Equal(t, "db2", n.Meta["dialect"])
	if schema == "" {
		assert.NotContains(t, n.Meta, "schema")
	} else {
		assert.Equal(t, schema, n.Meta["schema"])
	}
	assert.Equal(t, cobolNameNormRule, n.Meta["cobol_name_norm"])
	assertCobolSharedProvenance(t, n)
}

// cobolEdgeTargets returns "<kind> <to>" for each edge, in order.
func cobolEdgeTargets(edges []*graph.Edge) []string {
	var out []string
	for _, e := range edges {
		out = append(out, string(e.Kind)+" "+e.To)
	}
	return out
}

func TestCobolGrammar_SQL(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/sql.cbl"
	result := extractCobolGrammar(t, e, path, cobolSQLFixture)
	const proc = "SQLPGM/@proc/"
	para := func(p string) string { return path + "::" + proc + p }
	stmt := func(p string, k int) string {
		s := proc + p + "/@exec_sql_statement"
		if k >= 2 {
			s += "#" + strconv.Itoa(k)
		}
		return s
	}

	t.Run("select", func(t *testing.T) {
		edge := cobolOneEdge(t, result, para("P1"), "db2::MYSCH/AUDIT_LOG", graph.EdgeReads)
		assertCobolRelationProvenance(t, edge, "exec_sql_statement")
		assert.Equal(t, "SELECT", edge.Meta["sql_verb"])
		assert.Equal(t, stmt("P1", 1), edge.Meta["cobol_statement_symbol"])
		assert.Equal(t, cobolNameNormRule, edge.Meta["cobol_name_norm"])
		assert.NotContains(t, edge.Meta, "sql_aliases")
		assert.Equal(t, "SELECT", cobolNodeByID(t, result, path+"::"+stmt("P1", 1)).Meta["sql_verb"])
		assertCobolTable(t, result, "db2::MYSCH/AUDIT_LOG", "AUDIT_LOG", "MYSCH")

		assert.Equal(t, []string{"reads db2::_unqualified/CUST"}, cobolEdgeTargets(cobolStatementEdges(result, stmt("P1", 2))))
		assertCobolTable(t, result, "db2::_unqualified/CUST", "CUST", "")
		// Delimited identifiers keep their quotes and case; only ID
		// metacharacters are percent-encoded (T-03-28).
		assert.Equal(t, []string{`reads db2::_unqualified/"MixedCase"`}, cobolEdgeTargets(cobolStatementEdges(result, stmt("P1", 3))))
		assertCobolTable(t, result, `db2::_unqualified/"MixedCase"`, `"MixedCase"`, "")
		assert.Equal(t, []string{`reads db2::"A%2FB"/"x""y"`}, cobolEdgeTargets(cobolStatementEdges(result, stmt("P1", 4))))
		assertCobolTable(t, result, `db2::"A%2FB"/"x""y"`, `"x""y"`, `"A/B"`)
		// A three-part name keeps its location on the edge, not in the ID.
		located := cobolStatementEdges(result, stmt("P1", 5))
		require.Equal(t, []string{"reads db2::SCH/TB_X"}, cobolEdgeTargets(located))
		assert.Equal(t, "LOC1", located[0].Meta["sql_location"])
		assertCobolTable(t, result, "db2::SCH/TB_X", "TB_X", "SCH")
		// Ordinary identifiers fold to the same table; the verb stays as written.
		lower := cobolStatementEdges(result, stmt("P1", 6))
		require.Equal(t, []string{"reads db2::_unqualified/CUST"}, cobolEdgeTargets(lower))
		assert.Equal(t, "select", lower[0].Meta["sql_verb"])
	})

	t.Run("writes", func(t *testing.T) {
		insert := cobolStatementEdges(result, stmt("P3", 1))
		// The SELECT of INSERT … SELECT is an unparsed tail: T2 is never
		// observed, so it is not guessed.
		require.Equal(t, []string{"writes db2::_unqualified/T1"}, cobolEdgeTargets(insert))
		assert.Equal(t, "INSERT", insert[0].Meta["sql_verb"])
		assertCobolRelationProvenance(t, insert[0], "exec_sql_statement")
		assert.Equal(t, para("P3"), insert[0].From)
		assert.Equal(t, []string{"writes db2::_unqualified/T6"}, cobolEdgeTargets(cobolStatementEdges(result, stmt("P3", 2))))
		assert.Equal(t, []string{"writes db2::_unqualified/T7"}, cobolEdgeTargets(cobolStatementEdges(result, stmt("P3", 3))))
		merge := cobolStatementEdges(result, stmt("P3", 4))
		require.Equal(t, []string{"writes db2::_unqualified/T3", "reads db2::_unqualified/T4"}, cobolEdgeTargets(merge))
		assert.Equal(t, "MERGE", merge[1].Meta["sql_verb"])
		assert.Equal(t, "DELETE", cobolNodeByID(t, result, path+"::"+stmt("P3", 3)).Meta["sql_verb"])
	})

	t.Run("aliases", func(t *testing.T) {
		edges := cobolStatementEdges(result, stmt("P4", 1))
		require.Equal(t, []string{"reads db2::_unqualified/CUST"}, cobolEdgeTargets(edges))
		assert.Equal(t, []string{"C1", "C2"}, edges[0].Meta["sql_aliases"])
		for _, n := range result.Nodes {
			assert.False(t, strings.HasSuffix(n.ID, "/C1") || strings.HasSuffix(n.ID, "/C2") || strings.HasSuffix(n.ID, "/D1"),
				"alias node %s", n.ID)
		}
		derived := cobolStatementEdges(result, stmt("P4", 2))
		assert.Equal(t, []string{"reads db2::_unqualified/T5"}, cobolEdgeTargets(derived))
		assert.Equal(t, []string{"D1"}, cobolNodeByID(t, result, path+"::"+stmt("P4", 2)).Meta["sql_derived_aliases"])
	})

	t.Run("cte", func(t *testing.T) {
		assert.Equal(t, []string{"reads db2::_unqualified/ORDERS"}, cobolEdgeTargets(cobolStatementEdges(result, stmt("P5", 1))))
		n := cobolNodeByID(t, result, path+"::"+stmt("P5", 1))
		assert.Equal(t, []string{"RECENT"}, n.Meta["sql_ctes"])
		assert.Equal(t, []string{"RECENT"}, n.Meta["sql_cte_refs"])
		assert.Equal(t, "SELECT", n.Meta["sql_verb"])
		assert.NotContains(t, cobolTableIDs(result), "db2::_unqualified/RECENT")
	})

	t.Run("amber", func(t *testing.T) {
		assert.Equal(t, "amber", cobolNodeByID(t, result, path).Meta["prov_document_grade"])
		edge := cobolOneEdge(t, result, para("P3"), "db2::_unqualified/T1", graph.EdgeWrites)
		assert.Equal(t, "amber", edge.Meta["prov_document_grade"])
		assert.Contains(t, edge.Meta, "prov_affected")
	})

	t.Run("tables_exact", func(t *testing.T) {
		assert.Equal(t, []string{
			"db2::MYSCH/AUDIT_LOG", "db2::_unqualified/CUST", `db2::_unqualified/"MixedCase"`,
			`db2::"A%2FB"/"x""y"`, "db2::SCH/TB_X",
			"db2::_unqualified/T1", "db2::_unqualified/T6", "db2::_unqualified/T7",
			"db2::_unqualified/T3", "db2::_unqualified/T4", "db2::_unqualified/T5",
			"db2::_unqualified/ORDERS",
		}, cobolTableIDs(result))
	})
	const cpath = "src/cur.cbl"
	cursors := extractCobolGrammar(t, e, cpath, cobolSQLCursorFixture)
	const cproc = "CURPGM/@proc/"
	cstmt := func(p string, k int) string {
		s := cproc + p + "/@exec_sql_statement"
		if k >= 2 {
			s += "#" + strconv.Itoa(k)
		}
		return s
	}
	const orders = "db2::_unqualified/ORDERS"

	t.Run("cursor", func(t *testing.T) {
		declare := cobolStatementEdges(cursors, cstmt("P0", 1))
		require.Equal(t, []string{"reads " + orders}, cobolEdgeTargets(declare))
		assert.Equal(t, cpath+"::"+cproc+"P0", declare[0].From)
		assert.Equal(t, "C1", declare[0].Meta["sql_cursor"])
		assert.Equal(t, "DECLARE", declare[0].Meta["sql_verb"])
		assert.Equal(t, "C1", cobolNodeByID(t, cursors, cpath+"::"+cstmt("P0", 1)).Meta["sql_cursor"])
		// An underscore cursor name is a hidden token, so it is read from
		// the statement text.
		emp := cobolStatementEdges(cursors, cstmt("P0", 2))
		require.Equal(t, []string{"reads db2::_unqualified/EMP"}, cobolEdgeTargets(emp))
		assert.Equal(t, "C_EMP", emp[0].Meta["sql_cursor"])

		// OPEN, FETCH (plain, NEXT FROM, ABSOLUTE :hv FROM), and CLOSE of C1.
		for k, verb := range []string{"OPEN", "FETCH", "FETCH", "FETCH"} {
			edges := cobolStatementEdges(cursors, cstmt("P2", k+1))
			require.Equal(t, []string{"reads " + orders}, cobolEdgeTargets(edges), "P2 statement %d", k+1)
			assert.Equal(t, cpath+"::"+cproc+"P2", edges[0].From)
			assert.Equal(t, "C1", edges[0].Meta["sql_via_cursor"])
			assert.Equal(t, verb, edges[0].Meta["sql_verb"])
			assertCobolRelationProvenance(t, edges[0], "exec_sql_statement")
		}
		fetchEmp := cobolStatementEdges(cursors, cstmt("P2", 5))
		require.Equal(t, []string{"reads db2::_unqualified/EMP"}, cobolEdgeTargets(fetchEmp))
		assert.Equal(t, "C_EMP", fetchEmp[0].Meta["sql_via_cursor"])
		closeEdges := cobolStatementEdges(cursors, cstmt("P2", 6))
		require.Equal(t, []string{"reads " + orders}, cobolEdgeTargets(closeEdges))
		assert.Equal(t, "CLOSE", closeEdges[0].Meta["sql_verb"])
	})

	t.Run("cursor_unknown", func(t *testing.T) {
		// C9 is never declared; C2 is declared twice, so it is ambiguous
		// (WR-07).
		for k, tc := range []struct{ cursor, reason string }{
			{"C9", "cursor_not_declared"},
			{"C2", "ambiguous"},
		} {
			assert.Empty(t, cobolStatementEdges(cursors, cstmt("P3", k+1)))
			n := cobolNodeByID(t, cursors, cpath+"::"+cstmt("P3", k+1))
			assert.Equal(t, tc.reason, n.Meta["cobol_unresolved_reason"])
			assert.Equal(t, tc.cursor, n.Meta["sql_cursor"])
		}
		// Both DECLAREs of C2 keep their own reads edges.
		assert.Equal(t, []string{"reads db2::_unqualified/T8"}, cobolEdgeTargets(cobolStatementEdges(cursors, cstmt("P0", 3))))
		assert.Equal(t, []string{"reads db2::_unqualified/T9"}, cobolEdgeTargets(cobolStatementEdges(cursors, cstmt("P0", 4))))
	})

	t.Run("prepare", func(t *testing.T) {
		edges := cobolStatementEdges(cursors, cstmt("P4", 1))
		require.Equal(t, []string{
			"references unresolved::cobol::dynamic::sql::" + cpath + "::CURPGM::WS-SQLTXT",
			"references " + cpath + "::CURPGM/WS-SQLTXT",
		}, cobolEdgeTargets(edges))
		assert.Equal(t, "dynamic", edges[0].Meta["cobol_unresolved_reason"])
		assert.Equal(t, "PREPARE", edges[0].Meta["sql_verb"])
		assertCobolRelationProvenance(t, edges[0], "exec_sql_statement")
		assert.Equal(t, "sql_dynamic_source", edges[1].Meta["cobol_ref_role"])
		assert.NotContains(t, edges[1].Meta, "cobol_unresolved_reason")
	})

	t.Run("execute", func(t *testing.T) {
		for k := 2; k <= 3; k++ {
			assert.Empty(t, cobolStatementEdges(cursors, cstmt("P4", k)))
			n := cobolNodeByID(t, cursors, cpath+"::"+cstmt("P4", k))
			assert.Equal(t, true, n.Meta["sql_dynamic"])
			assert.Equal(t, "EXECUTE", n.Meta["sql_verb"])
		}
	})

	t.Run("ddl", func(t *testing.T) {
		assert.Equal(t, []string{"references db2::_unqualified/NEWT"}, cobolEdgeTargets(cobolStatementEdges(cursors, cstmt("P5", 1))))
		assert.Equal(t, []string{"references db2::S/T10"}, cobolEdgeTargets(cobolStatementEdges(cursors, cstmt("P5", 2))))
		assert.Equal(t, []string{"references db2::_unqualified/T11"}, cobolEdgeTargets(cobolStatementEdges(cursors, cstmt("P5", 3))))
		assertCobolTable(t, cursors, "db2::S/T10", "T10", "S")
	})

	t.Run("cursor_tables_exact", func(t *testing.T) {
		// No table for a cursor, a prepared statement, or a dynamic source.
		assert.Equal(t, []string{
			orders, "db2::_unqualified/EMP", "db2::_unqualified/T8", "db2::_unqualified/T9",
			"db2::_unqualified/NEWT", "db2::S/T10", "db2::_unqualified/T11",
		}, cobolTableIDs(cursors))
		assert.Equal(t, "amber", cobolNodeByID(t, cursors, cpath).Meta["prov_document_grade"])
	})
}
