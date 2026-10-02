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

const cobolIDMSFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. IDMSPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-DBKEY PIC S9(8) COMP.
       PROCEDURE DIVISION.
       P1.
           OBTAIN CALC EMP-REC.
       P2.
           FIND NEXT EMP-REC WITHIN DEPT-EMP.
       P3.
           CONNECT EMP-REC TO DEPT-EMP.
           DISCONNECT EMP-REC FROM DEPT-EMP.
       P4.
           STORE EMP-REC.
           MODIFY EMP-REC.
           ERASE EMP-REC PERMANENT MEMBERS.
           ERASE EMP-REC.
           GET EMP-REC.
       P5.
           BIND RUN-UNIT.
           FINISH.
           COMMIT.
           ROLLBACK.
           READY DEPT-AREA.
       P6.
           ACCEPT WS-DBKEY FROM EMP-REC CURRENCY.
       P7.
           CONNECT EMP-DEPT TO EMP-DEPT.
           FIND OWNER WITHIN DEPT-EMP. OBTAIN OWNER WITHIN DEPT-EMP.
           OBTAIN OWNER WITHIN DEPT-EMP.
       P8.
           obtain calc Emp-Rec.
           GOBACK.
`

// cobolStatementEdges returns the relationship edges built for the
// statement with QualName stmt, in emission order.
func cobolStatementEdges(result *parser.ExtractionResult, stmt string) []*graph.Edge {
	var edges []*graph.Edge
	for _, e := range result.Edges {
		if e.Meta["cobol_statement_symbol"] == stmt {
			edges = append(edges, e)
		}
	}
	return edges
}

// cobolResourceIDs returns the IDs of the KindResource nodes in emission
// order.
func cobolResourceIDs(result *parser.ExtractionResult) []string {
	var ids []string
	for _, n := range result.Nodes {
		if n.Kind == graph.KindResource {
			ids = append(ids, n.ID)
		}
	}
	return ids
}

// assertCobolResource checks the shared-resource node contract: one node
// per document, the document provenance, no range, and no defines edge.
func assertCobolResource(t *testing.T, result *parser.ExtractionResult, id, name, resourceType string) {
	t.Helper()
	var found []*graph.Node
	for _, n := range result.Nodes {
		if n.ID == id {
			found = append(found, n)
		}
	}
	require.Len(t, found, 1, "resource node %s", id)
	n := found[0]
	assert.Equal(t, graph.KindResource, n.Kind)
	assert.Equal(t, name, n.Name)
	assert.Equal(t, "cobol", n.Language)
	assert.Equal(t, 0, n.StartLine)
	assert.Equal(t, resourceType, n.Meta["resource_type"])
	assert.Equal(t, cobolNameNormRule, n.Meta["cobol_name_norm"])
	assert.Equal(t, false, n.Meta["prov_range_exact"])
	assert.Equal(t, "shared_resource", n.Meta["prov_range_absence"])
	for _, key := range cobolDocumentKeys {
		assert.Contains(t, n.Meta, key)
	}
	for _, e := range result.Edges {
		assert.False(t, e.To == id && (e.Kind == graph.EdgeDefines || e.Kind == graph.EdgeMemberOf),
			"structural %s edge to shared resource %s", e.Kind, id)
	}
}

func TestCobolGrammar_IDMS(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/idms.cbl"
	result := extractCobolGrammar(t, e, path, cobolIDMSFixture)
	const proc = "IDMSPGM/@proc/"
	const record = "idms::_unqualified/record/EMP-REC"
	const set = "idms::_unqualified/set/DEPT-EMP"
	para := func(p string) string { return path + "::" + proc + p }

	t.Run("obtain", func(t *testing.T) {
		edge := cobolOneEdge(t, result, para("P1"), record, graph.EdgeReads)
		assertCobolRelationProvenance(t, edge, "idms_navigation_statement")
		assert.Equal(t, "OBTAIN", edge.Meta["idms_verb"])
		assert.Equal(t, proc+"P1/@idms_navigation_statement", edge.Meta["cobol_statement_symbol"])
		assert.Equal(t, cobolNameNormRule, edge.Meta["cobol_name_norm"])
		stmt := cobolNodeByID(t, result, path+"::"+proc+"P1/@idms_navigation_statement")
		assert.Equal(t, "OBTAIN", stmt.Meta["idms_verb"])
		assertCobolResource(t, result, record, "EMP-REC", "idms_record")
	})

	t.Run("find_within", func(t *testing.T) {
		edges := cobolStatementEdges(result, proc+"P2/@idms_navigation_statement")
		require.Len(t, edges, 2)
		assert.Equal(t, record, edges[0].To)
		assert.Equal(t, graph.EdgeReads, edges[0].Kind)
		scope := edges[1]
		assert.Equal(t, "unresolved::cobol::idms_scope::_unqualified/DEPT-EMP", scope.To)
		assert.Equal(t, graph.EdgeReferences, scope.Kind)
		assert.Equal(t, "scope_role_unknown", scope.Meta["cobol_unresolved_reason"])
		assert.Equal(t, "FIND", scope.Meta["idms_verb"])
		assertCobolRelationProvenance(t, scope, "idms_navigation_statement")
	})

	t.Run("set_operand", func(t *testing.T) {
		for _, s := range []struct{ stmt, verb string }{
			{proc + "P3/@idms_update_statement", "CONNECT"},
			{proc + "P3/@idms_update_statement#2", "DISCONNECT"},
		} {
			edges := cobolStatementEdges(result, s.stmt)
			require.Len(t, edges, 2, s.stmt)
			assert.Equal(t, record, edges[0].To)
			assert.Equal(t, set, edges[1].To)
			for _, edge := range edges {
				assert.Equal(t, graph.EdgeWrites, edge.Kind)
				assert.Equal(t, s.verb, edge.Meta["idms_verb"])
			}
		}
		assertCobolResource(t, result, set, "DEPT-EMP", "idms_set")
	})

	t.Run("update", func(t *testing.T) {
		for _, s := range []struct {
			stmt, verb string
			kind       graph.EdgeKind
		}{
			{"P4/@idms_update_statement", "STORE", graph.EdgeWrites},
			{"P4/@idms_update_statement#2", "MODIFY", graph.EdgeWrites},
			{"P4/@idms_update_statement#3", "ERASE", graph.EdgeWrites},
			{"P4/@idms_update_statement#4", "ERASE", graph.EdgeWrites},
			{"P4/@idms_update_statement#5", "GET", graph.EdgeReads},
		} {
			edges := cobolStatementEdges(result, proc+s.stmt)
			require.Len(t, edges, 1, s.stmt)
			assert.Equal(t, record, edges[0].To, s.stmt)
			assert.Equal(t, s.kind, edges[0].Kind, s.stmt)
			assert.Equal(t, s.verb, edges[0].Meta["idms_verb"], s.stmt)
		}
		erase := cobolStatementEdges(result, proc+"P4/@idms_update_statement#3")[0]
		assert.Equal(t, "PERMANENT MEMBERS", erase.Meta["idms_erase_scope"])
		stmt := cobolNodeByID(t, result, path+"::"+proc+"P4/@idms_update_statement#3")
		assert.Equal(t, "PERMANENT MEMBERS", stmt.Meta["idms_erase_scope"])
		plain := cobolNodeByID(t, result, path+"::"+proc+"P4/@idms_update_statement#4")
		assert.NotContains(t, plain.Meta, "idms_erase_scope")
		assert.NotContains(t, cobolStatementEdges(result, proc+"P4/@idms_update_statement#4")[0].Meta, "idms_erase_scope")
	})

	t.Run("session", func(t *testing.T) {
		for _, s := range []struct{ stmt, verb string }{
			{"P5/@idms_session_statement", "BIND"},
			{"P5/@idms_session_statement#2", "FINISH"},
			{"P5/@idms_session_statement#3", "COMMIT"},
			{"P5/@idms_session_statement#4", "ROLLBACK"},
			{"P5/@idms_session_statement#5", "READY"},
		} {
			stmt := cobolNodeByID(t, result, path+"::"+proc+s.stmt)
			assert.Equal(t, s.verb, stmt.Meta["idms_verb"], s.stmt)
			assert.Empty(t, cobolStatementEdges(result, proc+s.stmt), s.stmt)
		}
		ready := cobolNodeByID(t, result, path+"::"+proc+"P5/@idms_session_statement#5")
		assert.Equal(t, "DEPT-AREA", ready.Meta["idms_area"])
	})

	t.Run("accept", func(t *testing.T) {
		edge := cobolOneEdge(t, result, para("P6"), record, graph.EdgeReferences)
		assert.Equal(t, "ACCEPT", edge.Meta["idms_verb"])
		assertCobolRelationProvenance(t, edge, "idms_accept_statement")
	})

	t.Run("adjacency", func(t *testing.T) {
		connect := cobolStatementEdges(result, proc+"P7/@idms_update_statement")
		require.Len(t, connect, 2)
		assert.Equal(t, "idms::_unqualified/record/EMP-DEPT", connect[0].To)
		assert.Equal(t, "idms::_unqualified/set/EMP-DEPT", connect[1].To)
		assertCobolResource(t, result, "idms::_unqualified/record/EMP-DEPT", "EMP-DEPT", "idms_record")
		assertCobolResource(t, result, "idms::_unqualified/set/EMP-DEPT", "EMP-DEPT", "idms_set")

		// Two same-line reads of one set collapse; a third line stays apart.
		reads := cobolEdges(result, para("P7"), set, graph.EdgeReads)
		require.Len(t, reads, 2)
		assert.Equal(t, 2, reads[0].Meta["cobol_occurrence_count"])
		assert.Equal(t, []string{proc + "P7/@idms_navigation_statement", proc + "P7/@idms_navigation_statement#2"},
			reads[0].Meta["cobol_statement_symbols"])
		assert.NotContains(t, reads[1].Meta, "cobol_occurrence_count")
		assert.Less(t, reads[0].Line, reads[1].Line)

		// Resource nodes appear once each, in first-sighting order.
		assert.Equal(t, []string{record, set, "idms::_unqualified/record/EMP-DEPT", "idms::_unqualified/set/EMP-DEPT"},
			cobolResourceIDs(result))
	})

	t.Run("name_norm", func(t *testing.T) {
		// A lower-case spelling folds to the same node; the verb stays as written.
		edge := cobolOneEdge(t, result, para("P8"), record, graph.EdgeReads)
		assert.Equal(t, "obtain", edge.Meta["idms_verb"])
		assertCobolResource(t, result, record, "EMP-REC", "idms_record")
	})

	t.Run("deterministic", func(t *testing.T) {
		again := extractCobolGrammar(t, e, path, cobolIDMSFixture)
		ids := func(r *parser.ExtractionResult) (nodes, edges []string) {
			for _, n := range r.Nodes {
				nodes = append(nodes, n.ID)
			}
			for _, e := range r.Edges {
				edges = append(edges, e.From+"|"+e.To+"|"+string(e.Kind))
			}
			return nodes, edges
		}
		n1, e1 := ids(result)
		n2, e2 := ids(again)
		assert.Equal(t, n1, n2)
		assert.Equal(t, e1, e2)
	})
}
