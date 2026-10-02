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
// document, Name and table as written, schema only when written, dialect
// db2, the name rule, and the shared-resource provenance.
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
	assert.Equal(t, false, n.Meta["prov_range_exact"])
	assert.Equal(t, "shared_resource", n.Meta["prov_range_absence"])
	for _, key := range cobolDocumentKeys {
		assert.Contains(t, n.Meta, key)
	}
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
}
