//go:build darwin || linux

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
	"github.com/zzet/gortex/internal/indexer"
	"github.com/zzet/gortex/internal/parser"
	"github.com/zzet/gortex/internal/parser/languages"
	"github.com/zzet/gortex/internal/testenv"
)

// Every fixture in this file is invented 7-column fixed-format COBOL. None
// may be replaced with estate-derived source: this repository is public.

// cobolBreadthOrdFixture exercises every Phase 3 construct in one program:
// data items with a REDEFINES, COPY and COPY IDMS, PERFORM THRU, GO TO
// DEPENDING ON, literal and dynamic CALLs, IDMS, CICS, and embedded SQL.
const cobolBreadthOrdFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. ORDPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CUSTREC.
       COPY IDMS RECORD EMP-REC.
       01 WS-REC.
          05 WS-ID PIC X(8).
          05 WS-AMT PIC 9(5).
       01 WS-ALT REDEFINES WS-REC PIC X(13).
       01 WS-PGM PIC X(8).
       01 WS-IX PIC 9.
       01 WS-X PIC X(10).
       01 WS-SQLTXT PIC X(200).
       PROCEDURE DIVISION.
       MAIN-PARA.
           PERFORM STEP-A THRU STEP-C.
           GO TO STEP-D STEP-E DEPENDING ON WS-IX.
           CALL 'SUBPGM'.
           CALL 'TWINPGM'.
           CALL 'ONLYB'.
           CALL WS-PGM.
           STOP RUN.
       STEP-A.
           OBTAIN CALC EMP-REC.
       STEP-B.
           EXEC CICS SEND MAP('MAP1') MAPSET('MSET1') END-EXEC.
       STEP-C.
           EXEC SQL SELECT A INTO :WS-X FROM MYSCH.ORDERS AS O
           END-EXEC.
           EXEC SQL INSERT INTO AUDIT_LOG VALUES (1) END-EXEC.
       STEP-D.
           EXEC SQL DECLARE C1 CURSOR FOR SELECT X FROM ORDHIST
           END-EXEC.
           EXEC SQL FETCH C1 INTO :WS-X END-EXEC.
       STEP-E.
           EXEC SQL PREPARE S1 FROM :WS-SQLTXT END-EXEC.
           GOBACK.
`

const cobolBreadthSubFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SUBPGM.
       PROCEDURE DIVISION.
       SUB-PARA.
           GOBACK.
`

// cobolBreadthTwinFixture is written to two files, so the repository holds
// two TWINPGM programs and a CALL 'TWINPGM' stays ambiguous.
const cobolBreadthTwinFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. TWINPGM.
       PROCEDURE DIVISION.
       TWIN-PARA.
           GOBACK.
`

// cobolBreadthBOMFixture is stored behind a UTF-8 BOM.
const cobolBreadthBOMFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. BOMPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           PERFORM WORK-PARA.
           GOBACK.
       WORK-PARA.
           EXIT.
`

const cobolBreadthCopybookFixture = `       01 CUST-REC.
          05 CUST-ID PIC X(8).
`

// cobolBreadthCustAllFixture is a copybook that copies another copybook.
const cobolBreadthCustAllFixture = `       01 CUST-ALL.
           COPY CUSTREC.
`

// cobolBreadthValParaFixture is a procedure copybook: its PERFORM MAIN-PARA
// names a paragraph only ORDPGM defines, and its EXEC SQL names a table
// ORDPGM already uses.
const cobolBreadthValParaFixture = `       VALIDATE-PARA.
           PERFORM CHECK-PARA.
           PERFORM MAIN-PARA.
           CALL 'SUBPGM'.
           EXEC SQL SELECT A INTO :WS-X FROM MYSCH.ORDERS END-EXEC.
       CHECK-PARA.
           EXIT.
`

// cobolBreadthOnlyBFixture lives only in the second repository.
const cobolBreadthOnlyBFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. ONLYB.
       PROCEDURE DIVISION.
       ONLYB-PARA.
           GOBACK.
`

// cobolBreadthNestFixture is OUTER with a contained INNER; OUTER calls INNER
// within the file.
const cobolBreadthNestFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. OUTER.
       PROCEDURE DIVISION.
       OUTER-PARA.
           CALL 'INNER'.
           GOBACK.
       IDENTIFICATION DIVISION.
       PROGRAM-ID. INNER.
       PROCEDURE DIVISION.
       INNER-PARA.
           GOBACK.
       END PROGRAM INNER.
       END PROGRAM OUTER.
`

// cobolBreadthNestCallFixture is a separately compiled caller of INNER,
// which COBOL scoping hides inside OUTER.
const cobolBreadthNestCallFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. NESTCALL.
       PROCEDURE DIVISION.
       NESTCALL-PARA.
           CALL 'INNER'.
           GOBACK.
`

// cobolBreadthRepos is two tracked repositories sharing one SQLite store.
type cobolBreadthRepos struct {
	dir, rootA, rootB string
	store             *store_sqlite.Store
	idx               *indexer.Indexer
	mi                *indexer.MultiIndexer
	cm                *config.ConfigManager
	prefixA, prefixB  string
}

// writeCobolBreadthRepo commits files to a fresh git repository at root.
func writeCobolBreadthRepo(t *testing.T, root string, files map[string]string) {
	t.Helper()
	gitInitRepo(t, root)
	add := []string{"add"}
	for rel, src := range files {
		require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, rel), []byte(src), 0o644))
		add = append(add, rel)
	}
	for _, args := range [][]string{add, {"commit", "-m", "add cobol fixture"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
}

// indexCobolBreadthRepos tracks repository B, then repository A, through one
// MultiIndexer into one SQLite store, with the in-memory shadow disabled and
// AI providers off. B goes first so A's resolve runs while ONLYB exists.
func indexCobolBreadthRepos(t *testing.T) cobolBreadthRepos {
	t.Helper()
	t.Setenv("GORTEX_SHADOW_MAX_FILES", "0")
	t.Setenv("GORTEX_LLM_PROVIDER", "")

	dir := testenv.ShortTempDir(t)
	repos := cobolBreadthRepos{dir: dir, rootA: filepath.Join(dir, "a"), rootB: filepath.Join(dir, "b")}
	writeCobolBreadthRepo(t, repos.rootA, map[string]string{
		"cobol/ordpgm.cbl":   cobolBreadthOrdFixture,
		"cobol/subpgm.cbl":   cobolBreadthSubFixture,
		"cobol/twin1.cbl":    cobolBreadthTwinFixture,
		"cobol/twin2.cbl":    cobolBreadthTwinFixture,
		"cobol/bompgm.cbl":   "\xEF\xBB\xBF" + cobolBreadthBOMFixture,
		"copy/custrec.cpy":   cobolBreadthCopybookFixture,
		"copy/custall.cpy":   cobolBreadthCustAllFixture,
		"copy/valpara.cpy":   cobolBreadthValParaFixture,
		"cobol/nest.cbl":     cobolBreadthNestFixture,
		"cobol/nestcall.cbl": cobolBreadthNestCallFixture,
	})
	writeCobolBreadthRepo(t, repos.rootB, map[string]string{
		"cobol/onlyb.cbl": cobolBreadthOnlyBFixture,
	})

	// note: second occurrence of indexCobolTracerRepo's store and indexer
	// wiring; that helper tracks one repository and records AddBatch, so it
	// cannot host two repositories sharing one store.
	store, err := store_sqlite.Open(filepath.Join(dir, "store.sqlite"))
	require.NoError(t, err)
	reg := parser.NewRegistry()
	languages.RegisterAll(reg)
	repos.store = store
	repos.idx = indexer.New(store, reg, config.Default().Index, zap.NewNop())
	repos.cm, err = config.NewConfigManager(filepath.Join(dir, "config.yaml"))
	require.NoError(t, err)
	repos.mi = indexer.NewMultiIndexer(store, reg, repos.idx.Search(), repos.cm, zap.NewNop())
	t.Cleanup(func() {
		_ = repos.mi.Close(context.Background())
		_ = store.Close()
	})

	resB, err := repos.mi.TrackRepoCtx(context.Background(), config.RepoEntry{Path: repos.rootB})
	require.NoError(t, err)
	resA, err := repos.mi.TrackRepoCtx(context.Background(), config.RepoEntry{Path: repos.rootA})
	require.NoError(t, err)
	repos.prefixA, repos.prefixB = resA.RepoPrefix, resB.RepoPrefix
	require.NotEmpty(t, repos.prefixA)
	require.NotEmpty(t, repos.prefixB)
	require.NotEqual(t, repos.prefixA, repos.prefixB)
	return repos
}

// ordID returns the full ID of a symbol in repository A's ORDPGM.
func (r cobolBreadthRepos) ordID(symbol string) string {
	return r.prefixA + "/cobol/ordpgm.cbl::ORDPGM/" + symbol
}

// cobolBreadthOutEdge returns the one persisted edge of kind from from to
// to, failing unless exactly one exists.
func cobolBreadthOutEdge(t *testing.T, store *store_sqlite.Store, from, to string, kind graph.EdgeKind) *graph.Edge {
	t.Helper()
	var found []*graph.Edge
	for _, e := range store.GetOutEdges(from) {
		if e.Kind == kind && e.To == to {
			found = append(found, e)
		}
	}
	require.Len(t, found, 1, "%s edges %s -> %s", kind, from, to)
	return found[0]
}

// assertCobolBreadthMissing asserts the store holds the missing-artifact
// placeholder node id (D-11) with artifact_status not_obtained.
func assertCobolBreadthMissing(t *testing.T, store *store_sqlite.Store, id string) {
	t.Helper()
	n := store.GetNode(id)
	require.NotNil(t, n, "no placeholder node %s", id)
	assert.Equal(t, graph.KindResource, n.Kind, id)
	assert.Equal(t, "not_obtained", n.Meta["artifact_status"], id)
}

// cobolBreadthStrings reads a []string Meta value, which a SQLite round
// trip may decode as []any.
func cobolBreadthStrings(t *testing.T, value any) []string {
	t.Helper()
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, v := range typed {
			text, ok := v.(string)
			require.True(t, ok, "non-string element %#v", v)
			out = append(out, text)
		}
		return out
	}
	require.Failf(t, "not a string list", "%#v", value)
	return nil
}

// assertCobolBreadthLine checks that a persisted edge's Line is its
// 0-based prov_start_row plus one.
func assertCobolBreadthLine(t *testing.T, e *graph.Edge) {
	t.Helper()
	startRow, ok := e.Meta["prov_start_row"].(int)
	require.True(t, ok, "%s -> %s prov_start_row = %#v", e.From, e.To, e.Meta["prov_start_row"])
	assert.Equal(t, startRow+1, e.Line, "%s -> %s", e.From, e.To)
}

// cobolBreadthNodeIDs decodes the node IDs of a get_callers or
// get_call_chain JSON payload.
func cobolBreadthNodeIDs(t *testing.T, text string) []string {
	t.Helper()
	var payload struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &payload), "payload: %s", text)
	ids := make([]string, 0, len(payload.Nodes))
	for _, n := range payload.Nodes {
		ids = append(ids, n.ID)
	}
	return ids
}

// TestCobolBreadth_StructureAndFlow proves structure and control flow come
// back out of the unchanged MCP tools over the daemon relay and the CLI
// relay (D-02, D-03), with no LLM service and no Neo4j configured.
func TestCobolBreadth_StructureAndFlow(t *testing.T) {
	repos := indexCobolBreadthRepos(t)
	socket, srv := serveCobolTestDaemon(t, repos.dir, repos.store, repos.idx, repos.mi, repos.cm)
	assert.Nil(t, srv.LLMService())
	mainID := repos.ordID("@proc/MAIN-PARA")
	stepB := repos.ordID("@proc/STEP-B")

	for _, tc := range []struct{ id, kind, cobolKind string }{
		{repos.ordID("WS-REC/WS-ID"), "field", "data_item"},
		{repos.ordID("WS-REC"), "variable", "data_item"},
		{repos.ordID("@proc/MAIN-PARA/@call_statement"), "local", "statement"},
		{repos.ordID("@procedure_division"), "package", "division"},
	} {
		var payload cobolTracerSymbol
		text := callCobolTracerTool(t, socket, repos.rootA, "get_symbol",
			map[string]any{"id": tc.id, "detail": "full", "format": "json"})
		require.NoError(t, json.Unmarshal([]byte(text), &payload), "get_symbol output: %s", text)
		require.Equal(t, tc.id, payload.Node["id"])
		assert.Equal(t, tc.kind, payload.Node["kind"], tc.id)
		meta, ok := payload.Node["meta"].(map[string]any)
		require.True(t, ok, "%s has no meta", tc.id)
		assert.Equal(t, tc.cobolKind, meta["cobol_kind"], tc.id)
		assert.Equal(t, "gortex-cobol-grammar/6", meta["prov_extractor_version"], tc.id)
	}

	chain := cobolBreadthNodeIDs(t, callCobolTracerTool(t, socket, repos.rootA, "get_call_chain",
		map[string]any{"id": mainID, "format": "json"}))
	for _, step := range []string{"STEP-A", "STEP-B", "STEP-C", "STEP-D", "STEP-E"} {
		assert.Contains(t, chain, repos.ordID("@proc/"+step), "get_call_chain from MAIN-PARA")
	}

	callers := cobolBreadthNodeIDs(t, callCobolTracerTool(t, socket, repos.rootA, "get_callers",
		map[string]any{"id": stepB, "format": "json"}))
	assert.Contains(t, callers, mainID, "MCP get_callers on a THRU-range paragraph")

	cmd, buf := newCallTestCmd(t)
	callIndex = repos.rootA
	callArgs = []string{"id=" + stepB}
	require.NoError(t, runCall(cmd, []string{"get_callers"}))
	assert.Contains(t, cobolBreadthNodeIDs(t, buf.String()), mainID, "CLI get_callers")
}

// TestCobolBreadth_CallsAndCopy proves binding on the persisted graph is
// limited to a single same-repository candidate (D-09, D-10, D-12, D-19):
// a program present only in repository B is never bound from repository A.
func TestCobolBreadth_CallsAndCopy(t *testing.T) {
	repos := indexCobolBreadthRepos(t)
	mainID := repos.ordID("@proc/MAIN-PARA")

	calls := map[string]*graph.Edge{}
	for _, e := range repos.store.GetOutEdges(mainID) {
		if e.Kind == graph.EdgeCalls && e.Meta["cobol_call"] != nil {
			require.NotContains(t, calls, e.To, "two CALL edges to %s", e.To)
			calls[e.To] = e
		}
	}
	require.Len(t, calls, 4, "MAIN-PARA CALL edges: %v", calls)

	sub := calls[repos.prefixA+"/cobol/subpgm.cbl::SUBPGM"]
	require.NotNil(t, sub, "CALL 'SUBPGM' is not bound")
	assert.Equal(t, "literal", sub.Meta["cobol_call"])
	assert.NotContains(t, sub.Meta, "cobol_unresolved_reason")

	twin := calls["unresolved::cobol::program::TWINPGM"]
	require.NotNil(t, twin, "CALL 'TWINPGM' lost its unresolved target")
	assert.Equal(t, "ambiguous", twin.Meta["cobol_unresolved_reason"])
	assert.Equal(t, []string{
		repos.prefixA + "/cobol/twin1.cbl::TWINPGM",
		repos.prefixA + "/cobol/twin2.cbl::TWINPGM",
	}, cobolBreadthStrings(t, twin.Meta["cobol_candidate_ids"]))

	onlyB := calls[repos.prefixA+"/cobol-missing::program/ONLYB"]
	require.NotNil(t, onlyB, "CALL 'ONLYB' was bound across repositories")
	assert.Equal(t, "missing", onlyB.Meta["cobol_unresolved_reason"])
	assertCobolBreadthMissing(t, repos.store, onlyB.To)
	require.NotNil(t, repos.store.GetNode(repos.prefixB+"/cobol/onlyb.cbl::ONLYB"),
		"ONLYB must exist in repository B for the missing outcome to mean anything")

	dynamic := calls["unresolved::cobol::dynamic::program::cobol/ordpgm.cbl::ORDPGM::WS-PGM"]
	require.NotNil(t, dynamic, "CALL WS-PGM lost its dynamic target")
	assert.Equal(t, "dynamic", dynamic.Meta["cobol_call"])
	assert.Equal(t, "dynamic", dynamic.Meta["cobol_unresolved_reason"])

	programID := repos.prefixA + "/cobol/ordpgm.cbl::ORDPGM"
	copybook := cobolBreadthOutEdge(t, repos.store, programID, repos.prefixA+"/copy/custrec.cpy", graph.EdgeImports)
	assert.Equal(t, "copy", copybook.Meta["cobol_copy_form"])
	assert.NotContains(t, copybook.Meta, "cobol_unresolved_reason")
	idms := cobolBreadthOutEdge(t, repos.store, programID,
		repos.prefixA+"/cobol-missing::idms_dictionary_member/RECORD/EMP-REC", graph.EdgeImports)
	assert.Equal(t, "idms", idms.Meta["cobol_copy_form"])
	assert.Equal(t, "idms_dictionary_member", idms.Meta["cobol_unresolved_reason"])
	assertCobolBreadthMissing(t, repos.store, idms.To)

	for _, e := range []*graph.Edge{sub, twin, onlyB, dynamic, copybook, idms} {
		assertCobolBreadthLine(t, e)
	}
}

// TestCobolBreadth_CopybookNestedCopy proves a COPY inside a copybook
// persists as one imports edge from the copybook file node, bound to the
// copied copybook under the unchanged exactly-one same-repository rule
// (D-11, DET-06), while ORDPGM's COPY of the same member still binds.
func TestCobolBreadth_CopybookNestedCopy(t *testing.T) {
	repos := indexCobolBreadthRepos(t)
	custrec := repos.prefixA + "/copy/custrec.cpy"
	nested := cobolBreadthOutEdge(t, repos.store, repos.prefixA+"/copy/custall.cpy", custrec, graph.EdgeImports)
	assert.Equal(t, "copy", nested.Meta["cobol_copy_form"])
	assert.Equal(t, "CUSTREC", nested.Meta["cobol_copy_member_raw"])
	assert.Equal(t, 1, nested.Meta["prov_start_row"])
	assert.Equal(t, true, nested.Meta["prov_range_exact"])
	assert.Equal(t, "copy_statement", nested.Meta["prov_observation_kind"])
	assert.NotContains(t, nested.Meta, "cobol_unresolved_reason")
	assertCobolBreadthLine(t, nested)

	program := cobolBreadthOutEdge(t, repos.store, repos.prefixA+"/cobol/ordpgm.cbl::ORDPGM", custrec, graph.EdgeImports)
	assert.NotContains(t, program.Meta, "cobol_unresolved_reason")
}

// TestCobolBreadth_CopybookProcedure proves a procedure copybook's
// paragraphs, PERFORMs, and CALLs persist owned by the copybook at the
// member's own rows (D-16, D-01, D-05), resolve labels member-locally only
// (D-02, D-12), bind its literal CALL (D-09), and reach MCP get_callers over
// the daemon relay, with no wrapper program or division node.
func TestCobolBreadth_CopybookProcedure(t *testing.T) {
	repos := indexCobolBreadthRepos(t)
	path := repos.prefixA + "/copy/valpara.cpy"
	validate := path + "::@proc/VALIDATE-PARA"
	check := path + "::@proc/CHECK-PARA"
	subpgm := repos.prefixA + "/cobol/subpgm.cbl::SUBPGM"

	for _, tc := range []struct {
		id       string
		row, ord int
	}{{validate, 0, 1}, {check, 5, 2}} {
		n := repos.store.GetNode(tc.id)
		require.NotNil(t, n, "missing %s", tc.id)
		assert.Equal(t, graph.KindFunction, n.Kind, tc.id)
		assert.Equal(t, "paragraph", n.Meta["cobol_kind"], tc.id)
		assert.Equal(t, tc.ord, n.Meta["cobol_ordinal"], tc.id)
		assert.Equal(t, tc.row, n.Meta["prov_start_row"], tc.id)
		assert.Equal(t, true, n.Meta["prov_range_exact"], tc.id)
		assert.Equal(t, "cobol-grade-policy-copybook-v1", n.Meta["prov_grade_policy"], tc.id)
		assert.Equal(t, "gortex-cobol-grammar/6", n.Meta["prov_extractor_version"], tc.id)
	}
	stmt := repos.store.GetNode(validate + "/@call_statement")
	require.NotNil(t, stmt, "missing the CALL statement node")
	assert.Equal(t, graph.KindLocal, stmt.Kind)

	perform := cobolBreadthOutEdge(t, repos.store, validate, check, graph.EdgeCalls)
	assert.Equal(t, "perform", perform.Meta["cobol_flow"])
	assert.Equal(t, "@proc/VALIDATE-PARA/@perform_statement_call_proc", perform.Meta["cobol_statement_symbol"])
	assertCobolBreadthLine(t, perform)

	// ORDPGM defines a MAIN-PARA in the same repository; it is not the target.
	require.NotNil(t, repos.store.GetNode(repos.ordID("@proc/MAIN-PARA")))
	foreign := cobolBreadthOutEdge(t, repos.store, validate,
		"unresolved::cobol::label::copy/valpara.cpy::MAIN-PARA", graph.EdgeCalls)
	assert.Equal(t, "label_not_in_copybook", foreign.Meta["cobol_unresolved_reason"])

	call := cobolBreadthOutEdge(t, repos.store, validate, subpgm, graph.EdgeCalls)
	assert.Equal(t, "literal", call.Meta["cobol_call"])
	assert.NotContains(t, call.Meta, "cobol_unresolved_reason")

	var copybookNodes int
	for _, n := range repos.store.GetNodesByLanguage("cobol") {
		if !strings.HasPrefix(n.ID, path) {
			continue
		}
		copybookNodes++
		assert.NotContains(t, []any{"program", "division"}, n.Meta["cobol_kind"], n.ID)
		assert.NotContains(t, n.ID, "CPYWRAP")
	}
	assert.Greater(t, copybookNodes, 3, "copybook nodes under %s", path)

	socket, _ := serveCobolTestDaemon(t, repos.dir, repos.store, repos.idx, repos.mi, repos.cm)
	for _, target := range []string{check, subpgm} {
		callers := cobolBreadthNodeIDs(t, callCobolTracerTool(t, socket, repos.rootA, "get_callers",
			map[string]any{"id": target, "format": "json"}))
		assert.Contains(t, callers, validate, "MCP get_callers on %s", target)
	}
}

// TestCobolBreadth_CopybookProcedureReferences proves a procedure
// copybook's EXEC SQL persists a statement-positioned DB2 edge From its
// paragraph to the shared table ORDPGM also uses (DET-11, D-13), without
// adding a table to repository A.
func TestCobolBreadth_CopybookProcedureReferences(t *testing.T) {
	repos := indexCobolBreadthRepos(t)
	validate := repos.prefixA + "/copy/valpara.cpy::@proc/VALIDATE-PARA"

	orders := cobolBreadthOutEdge(t, repos.store, validate, repos.prefixA+"/db2::MYSCH/ORDERS", graph.EdgeReads)
	assert.Equal(t, "SELECT", orders.Meta["sql_verb"])
	assert.Equal(t, "@proc/VALIDATE-PARA/@exec_sql_statement", orders.Meta["cobol_statement_symbol"])
	assert.Equal(t, graph.OriginASTResolved, orders.Origin)
	assert.Equal(t, 4, orders.Meta["prov_start_row"])
	assertCobolBreadthLine(t, orders)
}

// TestCobolBreadth_NestedProgramScope proves on the persisted graph that a
// contained program is a CALL candidate only in its own file (D-09): OUTER's
// CALL 'INNER' binds to OUTER/INNER, while NESTCALL's CALL 'INNER' from
// another file stays missing.
func TestCobolBreadth_NestedProgramScope(t *testing.T) {
	repos := indexCobolBreadthRepos(t)
	outerID := repos.prefixA + "/cobol/nest.cbl::OUTER"
	innerID := outerID + "/INNER"
	for id, nested := range map[string]bool{outerID: false, innerID: true} {
		n := repos.store.GetNode(id)
		require.NotNil(t, n, id)
		assert.Equal(t, nested, n.Meta["cobol_nested"], id)
	}

	same := cobolBreadthOutEdge(t, repos.store, outerID+"/@proc/OUTER-PARA", innerID, graph.EdgeCalls)
	assert.NotContains(t, same.Meta, "cobol_unresolved_reason")

	cross := cobolBreadthOutEdge(t, repos.store, repos.prefixA+"/cobol/nestcall.cbl::NESTCALL/@proc/NESTCALL-PARA",
		repos.prefixA+"/cobol-missing::program/INNER", graph.EdgeCalls)
	assert.Equal(t, "missing", cross.Meta["cobol_unresolved_reason"])
	assert.NotContains(t, cross.Meta, "cobol_candidate_ids")
	assertCobolBreadthMissing(t, repos.store, cross.To)

	for _, e := range []*graph.Edge{same, cross} {
		assertCobolBreadthLine(t, e)
	}
}

// TestCobolBreadth_MainframeAndSQL proves IDMS, CICS, and DB2 references
// persist with their verb or command Meta and statement ranges, and that
// the persisted tables are exactly the physical tables the SQL names
// (D-13, DET-12).
func TestCobolBreadth_MainframeAndSQL(t *testing.T) {
	repos := indexCobolBreadthRepos(t)
	step := func(name string) string { return repos.ordID("@proc/" + name) }

	record := cobolBreadthOutEdge(t, repos.store, step("STEP-A"),
		repos.prefixA+"/idms::_unqualified/record/EMP-REC", graph.EdgeReads)
	assert.Equal(t, "OBTAIN", record.Meta["idms_verb"])

	cicsMap := cobolBreadthOutEdge(t, repos.store, step("STEP-B"),
		repos.prefixA+"/cics::map/MSET1/MAP1", graph.EdgeWrites)
	assert.Equal(t, "SEND", cicsMap.Meta["cics_command"])

	orders := cobolBreadthOutEdge(t, repos.store, step("STEP-C"), repos.prefixA+"/db2::MYSCH/ORDERS", graph.EdgeReads)
	assert.Equal(t, "SELECT", orders.Meta["sql_verb"])
	assert.Equal(t, []string{"O"}, cobolBreadthStrings(t, orders.Meta["sql_aliases"]))

	audit := cobolBreadthOutEdge(t, repos.store, step("STEP-C"),
		repos.prefixA+"/db2::_unqualified/AUDIT_LOG", graph.EdgeWrites)
	assert.Equal(t, "INSERT", audit.Meta["sql_verb"])

	var fetch *graph.Edge
	for _, e := range repos.store.GetOutEdges(step("STEP-D")) {
		if e.Kind == graph.EdgeReads && e.Meta["sql_via_cursor"] != nil {
			require.Nil(t, fetch, "more than one cursor-use edge")
			fetch = e
		}
	}
	require.NotNil(t, fetch, "FETCH has no cursor-use edge")
	assert.Equal(t, repos.prefixA+"/db2::_unqualified/ORDHIST", fetch.To)
	assert.Equal(t, "C1", fetch.Meta["sql_via_cursor"])

	for _, e := range []*graph.Edge{record, cicsMap, orders, audit, fetch} {
		assert.Equal(t, graph.OriginASTResolved, e.Origin)
		assert.NotEmpty(t, e.Meta["cobol_statement_symbol"])
		assertCobolBreadthLine(t, e)
	}

	tables := map[string][]string{}
	for _, n := range repos.store.NodesByKinds([]graph.NodeKind{graph.KindTable}) {
		tables[n.RepoPrefix] = append(tables[n.RepoPrefix], n.ID)
	}
	sort.Strings(tables[repos.prefixA])
	assert.Equal(t, []string{
		repos.prefixA + "/db2::MYSCH/ORDERS",
		repos.prefixA + "/db2::_unqualified/AUDIT_LOG",
		repos.prefixA + "/db2::_unqualified/ORDHIST",
	}, tables[repos.prefixA])
	assert.Empty(t, tables[repos.prefixB])
}

// TestCobolBreadth_BOMEdgeOffsets proves the CR-01 fix reaches relationship
// edges: in a BOM-prefixed file the PERFORM edge's prov_start_byte is the
// statement's offset in the raw file, BOM included, and its row is the
// statement's 0-based line.
func TestCobolBreadth_BOMEdgeOffsets(t *testing.T) {
	repos := indexCobolBreadthRepos(t)
	raw, err := os.ReadFile(filepath.Join(repos.rootA, "cobol/bompgm.cbl"))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(raw), "\xEF\xBB\xBF"), "fixture lost its BOM")
	at := strings.Index(string(raw), "PERFORM")
	require.Positive(t, at)

	program := repos.prefixA + "/cobol/bompgm.cbl::BOMPGM/@proc/"
	edge := cobolBreadthOutEdge(t, repos.store, program+"MAIN-PARA", program+"WORK-PARA", graph.EdgeCalls)
	assert.Equal(t, "perform", edge.Meta["cobol_flow"])
	assert.Equal(t, at, edge.Meta["prov_start_byte"])
	assert.Equal(t, strings.Count(string(raw[:at]), "\n"), edge.Meta["prov_start_row"])
	assert.Equal(t, true, edge.Meta["prov_range_exact"])
	assertCobolBreadthLine(t, edge)
}
