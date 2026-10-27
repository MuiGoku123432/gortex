//go:build darwin || linux

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/daemon"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
	"github.com/zzet/gortex/internal/indexer"
	gortexmcp "github.com/zzet/gortex/internal/mcp"
	"github.com/zzet/gortex/internal/parser"
	"github.com/zzet/gortex/internal/parser/languages"
	"github.com/zzet/gortex/internal/query"
	"github.com/zzet/gortex/internal/testenv"
	"github.com/zzet/gortex/internal/testutil/graphfixture"
)

// cobolTracerFixture, cobolTracerSubFixture, and cobolTracerCopybookFixture
// are an invented 7-column fixed-format program, the program it calls, and
// the copybook it copies. They must never be replaced with estate-derived
// source: this repository is public.
const cobolTracerFixture = "       IDENTIFICATION DIVISION.\n" +
	"       PROGRAM-ID. DEMOPGM.\n" +
	"       DATA DIVISION.\n" +
	"       WORKING-STORAGE SECTION.\n" +
	"       COPY CUSTREC.\n" +
	"       PROCEDURE DIVISION.\n" +
	"       MAIN-PARA.\n" +
	"           PERFORM WORK-PARA.\n" +
	"           CALL 'SUBPGM'.\n" +
	"           STOP RUN.\n" +
	"       WORK-PARA.\n" +
	"           EXIT.\n"

const cobolTracerSubFixture = "       IDENTIFICATION DIVISION.\n" +
	"       PROGRAM-ID. SUBPGM.\n" +
	"       PROCEDURE DIVISION.\n" +
	"       SUB-PARA.\n" +
	"           GOBACK.\n"

const cobolTracerCopybookFixture = "       01 CUST-REC.\n" +
	"          05 CUST-ID PIC X(8).\n" +
	"          05 CUST-NAME PIC X(30).\n"

// cobolTracerApprovedGrammarID mirrors languages.cobolApprovedGrammarID,
// which is unexported; the tracer asserts the persisted value equals it.
const cobolTracerApprovedGrammarID = "f97452e11a2b80b92acb7edf776131470bc2e1ffd7c077f4ade4ce3c3a47503d"

// cobolTracerApprovedPreprocessorModule mirrors
// languages.cobolApprovedPreprocessorModule (D-16 re-pin).
const cobolTracerApprovedPreprocessorModule = "github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor@v0.0.0-20261002100409-e64234cc493c"

// cobolAddBatchRecorder proves the direct SQLite write path (TRACE-04): it
// records every node ID handed to AddBatch or AddBatchChecked and then
// delegates. Embedding the concrete store keeps Catalog, BulkLoader, and
// SymbolSearcher promoted. Parse workers flush concurrently, hence the mutex.
type cobolAddBatchRecorder struct {
	*store_sqlite.Store
	mu  sync.Mutex
	ids map[string]bool
}

func (r *cobolAddBatchRecorder) record(nodes []*graph.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range nodes {
		r.ids[n.ID] = true
	}
}

func (r *cobolAddBatchRecorder) saw(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ids[id]
}

func (r *cobolAddBatchRecorder) AddBatch(nodes []*graph.Node, edges []*graph.Edge) {
	r.record(nodes)
	r.Store.AddBatch(nodes, edges)
}

func (r *cobolAddBatchRecorder) AddBatchChecked(nodes []*graph.Node, edges []*graph.Edge) error {
	r.record(nodes)
	return r.Store.AddBatchChecked(nodes, edges)
}

// cobolTracerRepo is one tracked, indexed fixture repository.
type cobolTracerRepo struct {
	dir, root, dbPath string
	store             *store_sqlite.Store
	idx               *indexer.Indexer
	mi                *indexer.MultiIndexer
	cm                *config.ConfigManager
	prefix            string
	fileID, programID string
	commit            string
}

// indexCobolTracerRepo commits the fixture to a fresh git repository and
// tracks it through the real MultiIndexer into a SQLite store, with the
// in-memory shadow disabled so the cold index goes through AddBatch rather
// than BulkLoad. AI providers stay off.
func indexCobolTracerRepo(t *testing.T) cobolTracerRepo {
	t.Helper()
	t.Setenv("GORTEX_SHADOW_MAX_FILES", "0")
	t.Setenv("GORTEX_LLM_PROVIDER", "")

	dir := testenv.ShortTempDir(t)
	root := filepath.Join(dir, "repo")
	gitInitRepo(t, root)
	files := map[string]string{
		"cobol/demopgm.cbl": cobolTracerFixture,
		"cobol/subpgm.cbl":  cobolTracerSubFixture,
		"copy/custrec.cpy":  cobolTracerCopybookFixture,
	}
	for rel, src := range files {
		require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, rel), []byte(src), 0o644))
	}
	add := []string{"add", "cobol/demopgm.cbl", "cobol/subpgm.cbl", "copy/custrec.cpy"}
	for _, args := range [][]string{add, {"commit", "-m", "add cobol fixture"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	commit := gitCommitHash(root)
	require.NotEmpty(t, commit, "fixture repository has no HEAD commit")

	dbPath := filepath.Join(dir, "store.sqlite")
	store, err := store_sqlite.Open(dbPath)
	require.NoError(t, err)
	recorder := &cobolAddBatchRecorder{Store: store, ids: map[string]bool{}}

	reg := parser.NewRegistry()
	languages.RegisterAll(reg)
	idx := indexer.New(recorder, reg, config.Default().Index, zap.NewNop())
	cm, err := config.NewConfigManager(filepath.Join(dir, "config.yaml"))
	require.NoError(t, err)
	mi := indexer.NewMultiIndexer(recorder, reg, idx.Search(), cm, zap.NewNop())
	// Both Close calls tolerate a repeat, so a test may close early to reopen.
	t.Cleanup(func() {
		_ = mi.Close(context.Background())
		_ = store.Close()
	})

	res, err := mi.TrackRepoCtx(context.Background(), config.RepoEntry{Path: root})
	require.NoError(t, err)
	require.NotEmpty(t, res.RepoPrefix)

	repo := cobolTracerRepo{
		dir: dir, root: root, dbPath: dbPath,
		store: store, idx: idx, mi: mi, cm: cm,
		prefix:    res.RepoPrefix,
		fileID:    res.RepoPrefix + "/cobol/demopgm.cbl",
		programID: res.RepoPrefix + "/cobol/demopgm.cbl::DEMOPGM",
		commit:    commit,
	}
	// Asserted before any retrieval: get_symbol's ensureFresh may reindex.
	require.True(t, recorder.saw(repo.programID), "AddBatch never received %s", repo.programID)
	require.True(t, recorder.saw(repo.fileID), "AddBatch never received %s", repo.fileID)
	return repo
}

var cobolTracerHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// assertCobolTracerProgram checks one program node, as decoded JSON, against
// the Phase 2 provenance contract, including the fixture's HEAD commit as its
// VCS revision (D-09). Numbers compare numerically because JSON decoding
// yields float64.
func assertCobolTracerProgram(t *testing.T, node map[string]any, prefix, commit string) {
	t.Helper()
	assert.Equal(t, "function", node["kind"])
	assert.Equal(t, "DEMOPGM", node["name"])
	assert.Equal(t, "DEMOPGM", node["qual_name"])
	assert.Equal(t, prefix+"/cobol/demopgm.cbl", node["file_path"])

	meta, ok := node["meta"].(map[string]any)
	require.True(t, ok, "node has no meta: %v", node)
	assert.Equal(t, "program", meta["cobol_kind"])
	assert.Equal(t, "cobol/demopgm.cbl", meta["prov_source_path"])
	assert.Equal(t, cobolTracerApprovedGrammarID, meta["prov_parser_grammar_id"])
	module, _ := meta["prov_parser_module"].(string)
	assert.True(t, strings.HasPrefix(module, "github.com/MuiGoku123432/tree-sitter-cobol-upgrade/forest-shim/cobol@v0.0.0-"),
		"prov_parser_module = %q", module)
	assert.Equal(t, cobolTracerApprovedPreprocessorModule, meta["prov_preprocessor_module"])
	assert.Equal(t, "gortex-cobol-grammar/6", meta["prov_extractor_version"])
	assert.Equal(t, "DETERMINISTIC", meta["prov_evidence_class"])
	assert.Equal(t, "ast_resolved", meta["prov_origin"])
	assert.InDelta(t, 1.0, meta["prov_confidence"], 0)
	assert.Equal(t, "green", meta["prov_document_grade"])
	reasons, present := meta["prov_document_grade_reasons"]
	assert.True(t, present, "prov_document_grade_reasons missing")
	assert.Empty(t, reasons)
	assert.NotEmpty(t, meta["prov_grade_policy"])
	assert.Equal(t, "cobol-handoff-v1", meta["prov_handoff_schema"])
	assert.Equal(t, "program_definition", meta["prov_observation_kind"])
	assert.Equal(t, false, meta["prov_affected"])
	assert.Equal(t, true, meta["prov_range_exact"])
	assert.Equal(t, commit, meta["prov_vcs_commit"])
	assert.NotContains(t, meta, "prov_vcs_commit_absence")

	startRow, ok := meta["prov_start_row"].(float64)
	require.True(t, ok, "prov_start_row = %v", meta["prov_start_row"])
	endRow, ok := meta["prov_end_row"].(float64)
	require.True(t, ok, "prov_end_row = %v", meta["prov_end_row"])
	assert.InDelta(t, startRow+1, node["start_line"], 0)
	assert.InDelta(t, endRow+1, node["end_line"], 0)

	for _, key := range []string{"prov_source_id", "prov_revision_content_id", "prov_parser_tool_id", "prov_parse_config_id", "prov_transform_config_id"} {
		value, _ := meta[key].(string)
		assert.Regexp(t, cobolTracerHex64, value, key)
	}
	for key, value := range meta {
		text, isString := value.(string)
		if !strings.HasPrefix(key, "prov_") || !isString {
			continue
		}
		assert.NotContains(t, text, "PROCEDURE DIVISION", key)
		assert.NotContains(t, text, "STOP RUN", key)
	}
}

// cobolTracerNodeJSON round-trips a stored node through its JSON form so the
// same contract helper checks stored and relayed nodes alike.
func cobolTracerNodeJSON(t *testing.T, node *graph.Node) map[string]any {
	t.Helper()
	raw, err := json.Marshal(node)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	return decoded
}

// TestCobolTracer_PersistsThroughAddBatch proves the program reaches SQLite
// through AddBatch with every provenance key, survives a reopen, carries the
// scope fields applyRepoPrefix stamps, and is defined by its file node.
func TestCobolTracer_PersistsThroughAddBatch(t *testing.T) {
	repo := indexCobolTracerRepo(t)

	live := repo.store.GetNode(repo.programID)
	require.NotNil(t, live)
	assertCobolTracerProgram(t, cobolTracerNodeJSON(t, live), repo.prefix, repo.commit)

	require.NoError(t, repo.mi.Close(context.Background()))
	require.NoError(t, repo.store.Close())

	reopened, err := store_sqlite.Open(repo.dbPath)
	require.NoError(t, err)
	defer func() { require.NoError(t, reopened.Close()) }()

	program := reopened.GetNode(repo.programID)
	require.NotNil(t, program, "program missing after reopen")
	assertCobolTracerProgram(t, cobolTracerNodeJSON(t, program), repo.prefix, repo.commit)
	assert.Equal(t, repo.prefix, program.RepoPrefix)
	assert.Empty(t, program.Origin)

	file := reopened.GetNode(repo.fileID)
	require.NotNil(t, file, "file node missing after reopen")
	assert.Equal(t, repo.commit, file.Meta["prov_vcs_commit"])
	assert.NotContains(t, file.Meta, "prov_vcs_commit_absence")
	assert.NotEmpty(t, program.WorkspaceID)
	assert.NotEmpty(t, program.ProjectID)
	assert.Equal(t, file.WorkspaceID, program.WorkspaceID)
	assert.Equal(t, file.ProjectID, program.ProjectID)

	var defines []*graph.Edge
	for _, e := range reopened.GetInEdges(repo.programID) {
		if e.Kind == graph.EdgeDefines {
			defines = append(defines, e)
		}
	}
	require.Len(t, defines, 1)
	assert.Equal(t, repo.fileID, defines[0].From)

	canonical, err := graphfixture.Canonical(repo.dbPath)
	require.NoError(t, err)
	// Canonical's SQLite printf keeps the separator as the literal text \x1f.
	assert.True(t, strings.Contains("\n"+canonical, "\nnode\\x1f"+repo.programID+"\\x1f"),
		"no SQLite node row for %s", repo.programID)
}

// startCobolTracerDaemon indexes the fixture repository and serves it from an
// in-process daemon over the real relay. The MCP server gets no LLM service,
// and no Neo4j profile or environment is configured.
func startCobolTracerDaemon(t *testing.T) (repo cobolTracerRepo, socket string, srv *gortexmcp.Server) {
	t.Helper()
	repo = indexCobolTracerRepo(t)
	socket, srv = serveCobolTestDaemon(t, repo.dir, repo.store, repo.idx, repo.mi, repo.cm)
	return repo, socket, srv
}

// serveCobolTestDaemon serves an indexed SQLite store from an in-process
// daemon over the real relay, with the socket and pidfile under dir. The
// COBOL tracer and breadth tests share it.
func serveCobolTestDaemon(t *testing.T, dir string, store *store_sqlite.Store, idx *indexer.Indexer,
	mi *indexer.MultiIndexer, cm *config.ConfigManager) (socket string, srv *gortexmcp.Server) {
	t.Helper()
	// note: second occurrence of spinUpDaemonWithConfig's wiring; that helper
	// hard-codes an in-memory graph and a Go fixture, so it cannot host this
	// SQLite-backed COBOL repository.
	socket = filepath.Join(dir, "s")
	t.Setenv("GORTEX_DAEMON_SOCKET", socket)
	t.Setenv("GORTEX_DAEMON_PIDFILE", filepath.Join(dir, "p"))
	srv = gortexmcp.NewServer(query.NewEngine(store), store, idx, nil, zap.NewNop(), nil,
		gortexmcp.MultiRepoOptions{MultiIndexer: mi, ConfigManager: cm})
	lifecycle, err := indexer.NewCheckoutLifecycle(indexer.CheckoutLifecycleConfig{
		MultiIndexer:  mi,
		ConfigManager: cm,
		Graph:         store,
		Logger:        zap.NewNop(),
	})
	require.NoError(t, err)
	d := daemon.New(socket, "test", zap.NewNop())
	d.Controller = &realController{
		graph:         store,
		multiIndexer:  mi,
		configManager: cm,
		lifecycle:     lifecycle,
		logger:        zap.NewNop(),
	}
	d.MCPDispatcher = newMCPDispatcher(srv, mi, zap.NewNop())
	require.NoError(t, d.Listen())
	serveDone := make(chan error, 1)
	go func() { serveDone <- d.Serve() }()
	t.Cleanup(func() {
		shutdownErr := d.Shutdown()
		select {
		case err := <-serveDone:
			require.NoError(t, shutdownErr)
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			require.NoError(t, shutdownErr)
			t.Error("daemon did not stop")
		}
	})
	require.Eventually(t, func() bool { return daemon.IsRunningAt(socket) },
		2*time.Second, 10*time.Millisecond)
	return socket, srv
}

// callCobolTracerTool runs one MCP tools/call over the daemon relay and
// returns the first content text of a successful result.
func callCobolTracerTool(t *testing.T, socket, cwd, tool string, args map[string]any) string {
	t.Helper()
	text, isError := callCobolToolResult(t, socket, cwd, tool, args)
	require.False(t, isError, "%s returned a tool error: %s", tool, text)
	return text
}

// callCobolToolResult runs one MCP tools/call over the daemon relay and
// returns the first content text and whether the tool reported an error. A
// JSON-RPC error or an empty result fails the test.
func callCobolToolResult(t *testing.T, socket, cwd, tool string, args map[string]any) (text string, isError bool) {
	t.Helper()
	client, err := daemon.DialTo(socket, daemon.Handshake{
		Mode:       daemon.ModeMCP,
		CWD:        cwd,
		ClientName: "cobol-tracer",
		Tools:      cliLegacyToolSurface,
		ToolsMode:  cliLegacyToolMode,
	})
	require.NoError(t, err)
	defer client.Close()

	require.NoError(t, client.WriteMCPFrame([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"cobol-tracer","version":"0"}}}`)))
	initReply, err := client.ReadMCPFrame()
	require.NoError(t, err)
	require.Contains(t, string(initReply), `"result"`, "initialize must succeed: %s", initReply)

	frame, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": args},
	})
	require.NoError(t, err)
	require.NoError(t, client.WriteMCPFrame(frame))
	reply, err := client.ReadMCPFrame()
	require.NoError(t, err)

	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error map[string]any `json:"error"`
	}
	require.NoError(t, json.Unmarshal(reply, &resp))
	require.Nil(t, resp.Error, "%s must not error: %s", tool, reply)
	require.NotEmpty(t, resp.Result.Content, "%s returned no content: %s", tool, reply)
	return resp.Result.Content[0].Text, resp.Result.IsError
}

// cobolTracerSymbol is the get_symbol detail=full payload.
type cobolTracerSymbol struct {
	Node    map[string]any   `json:"node"`
	InEdges []map[string]any `json:"in_edges"`
}

// TestCobolTracer_MCPGetSymbol retrieves the program by its exact prefixed ID
// through the real daemon relay and MCP get_symbol with detail=full.
func TestCobolTracer_MCPGetSymbol(t *testing.T) {
	repo, socket, _ := startCobolTracerDaemon(t)

	var payload cobolTracerSymbol
	// detail=full: the brief detail omits Meta, hiding every provenance key.
	text := callCobolTracerTool(t, socket, repo.root, "get_symbol",
		map[string]any{"id": repo.programID, "detail": "full", "format": "json"})
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	require.Equal(t, repo.programID, payload.Node["id"])
	assertCobolTracerProgram(t, payload.Node, repo.prefix, repo.commit)

	var definedByFile bool
	for _, e := range payload.InEdges {
		if e["kind"] == string(graph.EdgeDefines) && e["from"] == repo.fileID {
			definedByFile = true
		}
	}
	assert.True(t, definedByFile, "in_edges lacks defines from %s: %v", repo.fileID, payload.InEdges)
}

// TestCobolTracer_CLIGetSymbol retrieves the program through the matching CLI
// verb, `gortex call get_symbol`, over the real daemon relay (no stub).
func TestCobolTracer_CLIGetSymbol(t *testing.T) {
	repo, _, _ := startCobolTracerDaemon(t)

	cmd, buf := newCallTestCmd(t)
	callIndex = repo.root
	callArgs = []string{"id=" + repo.programID, "detail=full"}
	require.NoError(t, runCall(cmd, []string{"get_symbol"}))

	var payload cobolTracerSymbol
	require.NoError(t, json.Unmarshal(buf.Bytes(), &payload), "CLI output: %s", buf.String())
	require.Equal(t, repo.programID, payload.Node["id"])
	assertCobolTracerProgram(t, payload.Node, repo.prefix, repo.commit)
}

// TestCobolTracer_SearchSymbolsByName proves the program is discoverable by
// its name through MCP search_symbols.
func TestCobolTracer_SearchSymbolsByName(t *testing.T) {
	repo, socket, _ := startCobolTracerDaemon(t)

	text := callCobolTracerTool(t, socket, repo.root, "search_symbols",
		map[string]any{"query": "DEMOPGM", "format": "json"})
	var found struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &found), "search_symbols output: %s", text)
	var ids []string
	for _, r := range found.Results {
		ids = append(ids, r.ID)
	}
	assert.Contains(t, ids, repo.programID)
}

// TestCobolTracer_NoAINoNeo4j proves the slice runs with AI off: no LLM
// service, an empty provider setting, and no ask tool, while get_symbol still
// serves the program. No Neo4j profile or environment is configured here; the
// plan's verify step separately proves the Neo4j driver is absent from the
// indexer, parser, and SQLite store dependency closure.
func TestCobolTracer_NoAINoNeo4j(t *testing.T) {
	repo, socket, srv := startCobolTracerDaemon(t)

	assert.Nil(t, srv.LLMService())
	assert.Empty(t, os.Getenv("GORTEX_LLM_PROVIDER"))
	tools := srv.RegisteredScopedTools()
	assert.Contains(t, tools, "get_symbol")
	assert.NotContains(t, tools, "ask")

	var payload cobolTracerSymbol
	// note: same get_symbol arguments as TestCobolTracer_MCPGetSymbol.
	text := callCobolTracerTool(t, socket, repo.root, "get_symbol",
		map[string]any{"id": repo.programID, "detail": "full", "format": "json"})
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	assert.Equal(t, repo.programID, payload.Node["id"])
}

// TestCobolTracer_ParagraphFlow proves paragraph control flow travels the
// real path (D-01, D-02): MAIN-PARA's PERFORM WORK-PARA persists in SQLite as
// a calls edge with the provenance contract, and the unchanged MCP
// get_callers tool returns MAIN-PARA as a caller of WORK-PARA over the relay.
func TestCobolTracer_ParagraphFlow(t *testing.T) {
	repo, socket, _ := startCobolTracerDaemon(t)
	mainID := repo.prefix + "/cobol/demopgm.cbl::DEMOPGM/@proc/MAIN-PARA"
	workID := repo.prefix + "/cobol/demopgm.cbl::DEMOPGM/@proc/WORK-PARA"

	text := callCobolTracerTool(t, socket, repo.root, "get_callers",
		map[string]any{"id": workID, "format": "json"})
	var callers struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &callers), "get_callers output: %s", text)
	var callerIDs []string
	for _, n := range callers.Nodes {
		callerIDs = append(callerIDs, n.ID)
	}
	assert.Contains(t, callerIDs, mainID, "get_callers output: %s", text)

	var payload cobolTracerSymbol
	text = callCobolTracerTool(t, socket, repo.root, "get_symbol",
		map[string]any{"id": mainID, "detail": "full", "format": "json"})
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	require.Equal(t, mainID, payload.Node["id"])
	assert.Equal(t, "function", payload.Node["kind"])
	meta, ok := payload.Node["meta"].(map[string]any)
	require.True(t, ok, "paragraph has no meta: %v", payload.Node)
	assert.Equal(t, "paragraph", meta["cobol_kind"])
	assert.InDelta(t, 1, meta["cobol_ordinal"], 0)
	assert.Equal(t, "gortex-cobol-grammar/6", meta["prov_extractor_version"])
	assert.Equal(t, true, meta["prov_range_exact"])

	// Edge Meta is not part of the MCP JSON, so the persisted row is the proof.
	// Only control-flow edges count: MAIN-PARA's CALL is a calls edge too.
	var calls []*graph.Edge
	for _, e := range repo.store.GetOutEdges(mainID) {
		if e.Kind == graph.EdgeCalls && e.Meta["cobol_flow"] != nil {
			calls = append(calls, e)
		}
	}
	require.Len(t, calls, 1)
	edge := calls[0]
	assert.Equal(t, workID, edge.To)
	assert.Equal(t, graph.OriginASTResolved, edge.Origin)
	assert.Equal(t, "perform", edge.Meta["cobol_flow"])
	assert.Equal(t, "perform_statement_call_proc", edge.Meta["prov_observation_kind"])
	assert.Equal(t, "gortex-cobol-grammar/6", edge.Meta["prov_extractor_version"])
	startRow, ok := edge.Meta["prov_start_row"].(int)
	require.True(t, ok, "prov_start_row = %#v", edge.Meta["prov_start_row"])
	assert.Equal(t, startRow+1, edge.Line)
}

// TestCobolTracer_CallAndCopyBinding proves cross-file binding on the real
// path (D-09, D-11): tracking runs the production resolver, which binds
// MAIN-PARA's CALL 'SUBPGM' to the one SUBPGM program and COPY CUSTREC to the
// one custrec copybook in the repository, and the unchanged MCP get_callers
// tool returns MAIN-PARA as a caller of SUBPGM over the relay.
func TestCobolTracer_CallAndCopyBinding(t *testing.T) {
	repo, socket, _ := startCobolTracerDaemon(t)
	mainID := repo.prefix + "/cobol/demopgm.cbl::DEMOPGM/@proc/MAIN-PARA"
	subID := repo.prefix + "/cobol/subpgm.cbl::SUBPGM"
	copybookID := repo.prefix + "/copy/custrec.cpy"

	// Persisted rows first: get_callers' freshness check may reindex.
	var call *graph.Edge
	for _, e := range repo.store.GetOutEdges(mainID) {
		if e.Kind == graph.EdgeCalls && e.Meta["cobol_call"] != nil {
			require.Nil(t, call, "MAIN-PARA has more than one CALL edge")
			call = e
		}
	}
	require.NotNil(t, call, "MAIN-PARA has no CALL edge")
	assert.Equal(t, subID, call.To)
	assert.Equal(t, "literal", call.Meta["cobol_call"])
	assert.Equal(t, "'SUBPGM'", call.Meta["cobol_target_raw"])
	assert.NotContains(t, call.Meta, "cobol_unresolved_reason")

	var copies []*graph.Edge
	for _, e := range repo.store.GetOutEdges(repo.programID) {
		if e.Kind == graph.EdgeImports {
			copies = append(copies, e)
		}
	}
	require.Len(t, copies, 1)
	assert.Equal(t, copybookID, copies[0].To)
	assert.Equal(t, "copy", copies[0].Meta["cobol_copy_form"])
	assert.NotContains(t, copies[0].Meta, "cobol_unresolved_reason")

	copybook := repo.store.GetNode(copybookID)
	require.NotNil(t, copybook, "copybook file node missing")
	assert.Equal(t, "copybook", copybook.Meta["cobol_kind"])
	assert.Equal(t, "CUSTREC", copybook.Meta["cobol_name_key"])

	text := callCobolTracerTool(t, socket, repo.root, "get_callers",
		map[string]any{"id": subID, "format": "json"})
	var callers struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &callers), "get_callers output: %s", text)
	var callerIDs []string
	for _, n := range callers.Nodes {
		callerIDs = append(callerIDs, n.ID)
	}
	assert.Contains(t, callerIDs, mainID, "get_callers output: %s", text)
}

// TestCobolTracer_CopybookDataItems proves copybook-owned data items travel
// the real path (D-16, D-07): CUST-REC and its fields persist under the
// copybook, not the program that copies it, with the member's own
// coordinates; MCP get_symbol serves CUST-REC; and COPY CUSTREC still binds
// to the copybook file node.
func TestCobolTracer_CopybookDataItems(t *testing.T) {
	repo, socket, _ := startCobolTracerDaemon(t)
	copybookID := repo.prefix + "/copy/custrec.cpy"
	recID := copybookID + "::CUST-REC"

	// Persisted rows first: get_symbol's freshness check may reindex.
	rec := repo.store.GetNode(recID)
	require.NotNil(t, rec, "copybook data item %s missing", recID)
	assert.Equal(t, graph.KindVariable, rec.Kind)
	for _, field := range []string{"CUST-ID", "CUST-NAME"} {
		node := repo.store.GetNode(recID + "/" + field)
		require.NotNil(t, node, "copybook field %s missing", field)
		assert.Equal(t, graph.KindField, node.Kind)
	}
	copybook := repo.store.GetNode(copybookID)
	require.NotNil(t, copybook, "copybook file node missing")
	assert.Equal(t, "copybook", copybook.Meta["cobol_kind"])
	assert.NotContains(t, copybook.Meta, "prov_analysis_absence")

	var copies []string
	for _, e := range repo.store.GetOutEdges(repo.programID) {
		if e.Kind == graph.EdgeImports {
			copies = append(copies, e.To)
		}
	}
	assert.Equal(t, []string{copybookID}, copies)
	program := repo.store.GetNode(repo.programID)
	require.NotNil(t, program)
	assert.Equal(t, cobolTracerApprovedPreprocessorModule, program.Meta["prov_preprocessor_module"])

	var payload cobolTracerSymbol
	text := callCobolTracerTool(t, socket, repo.root, "get_symbol",
		map[string]any{"id": recID, "detail": "full", "format": "json"})
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	require.Equal(t, recID, payload.Node["id"])
	meta, ok := payload.Node["meta"].(map[string]any)
	require.True(t, ok, "data item has no meta: %v", payload.Node)
	assert.Equal(t, "data_item", meta["cobol_kind"])
	assert.Equal(t, "cobol-grade-policy-copybook-v1", meta["prov_grade_policy"])
	assert.InDelta(t, 0, meta["prov_start_row"], 0)
	assert.Equal(t, true, meta["prov_range_exact"])
}
