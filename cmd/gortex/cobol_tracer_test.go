//go:build !windows

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

// cobolTracerFixture is an invented 7-column fixed-format program. It must
// never be replaced with estate-derived source: this repository is public.
const cobolTracerFixture = "       IDENTIFICATION DIVISION.\n" +
	"       PROGRAM-ID. DEMOPGM.\n" +
	"       PROCEDURE DIVISION.\n" +
	"       MAIN-PARA.\n" +
	"           STOP RUN.\n"

// cobolTracerApprovedGrammarID mirrors languages.cobolApprovedGrammarID,
// which is unexported; the tracer asserts the persisted value equals it.
const cobolTracerApprovedGrammarID = "f97452e11a2b80b92acb7edf776131470bc2e1ffd7c077f4ade4ce3c3a47503d"

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
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cobol"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cobol", "demopgm.cbl"), []byte(cobolTracerFixture), 0o644))
	for _, args := range [][]string{{"add", "cobol/demopgm.cbl"}, {"commit", "-m", "add cobol fixture"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}

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
	}
	// Asserted before any retrieval: get_symbol's ensureFresh may reindex.
	require.True(t, recorder.saw(repo.programID), "AddBatch never received %s", repo.programID)
	require.True(t, recorder.saw(repo.fileID), "AddBatch never received %s", repo.fileID)
	return repo
}

var cobolTracerHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// assertCobolTracerProgram checks one program node, as decoded JSON, against
// the Phase 2 provenance contract. Numbers compare numerically because JSON
// decoding yields float64.
func assertCobolTracerProgram(t *testing.T, node map[string]any, prefix string) {
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
	assert.Equal(t, "gortex-cobol-grammar/1", meta["prov_extractor_version"])
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
	assertCobolTracerProgram(t, cobolTracerNodeJSON(t, live), repo.prefix)

	require.NoError(t, repo.mi.Close(context.Background()))
	require.NoError(t, repo.store.Close())

	reopened, err := store_sqlite.Open(repo.dbPath)
	require.NoError(t, err)
	defer func() { require.NoError(t, reopened.Close()) }()

	program := reopened.GetNode(repo.programID)
	require.NotNil(t, program, "program missing after reopen")
	assertCobolTracerProgram(t, cobolTracerNodeJSON(t, program), repo.prefix)
	assert.Equal(t, repo.prefix, program.RepoPrefix)
	assert.Empty(t, program.Origin)

	file := reopened.GetNode(repo.fileID)
	require.NotNil(t, file, "file node missing after reopen")
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

// TestCobolTracer_MCPGetSymbol retrieves the program by its exact prefixed ID
// through the real daemon relay and MCP get_symbol with detail=full.
func TestCobolTracer_MCPGetSymbol(t *testing.T) {
	repo := indexCobolTracerRepo(t)

	// note: second occurrence of spinUpDaemonWithConfig's wiring; that helper
	// hard-codes an in-memory graph and a Go fixture, so it cannot host this
	// SQLite-backed COBOL repository.
	socket := filepath.Join(repo.dir, "s")
	t.Setenv("GORTEX_DAEMON_SOCKET", socket)
	t.Setenv("GORTEX_DAEMON_PIDFILE", filepath.Join(repo.dir, "p"))
	srv := gortexmcp.NewServer(query.NewEngine(repo.store), repo.store, repo.idx, nil, zap.NewNop(), nil,
		gortexmcp.MultiRepoOptions{MultiIndexer: repo.mi, ConfigManager: repo.cm})
	lifecycle, err := indexer.NewCheckoutLifecycle(indexer.CheckoutLifecycleConfig{
		MultiIndexer:  repo.mi,
		ConfigManager: repo.cm,
		Graph:         repo.store,
		Logger:        zap.NewNop(),
	})
	require.NoError(t, err)
	d := daemon.New(socket, "test", zap.NewNop())
	d.Controller = &realController{
		graph:         repo.store,
		multiIndexer:  repo.mi,
		configManager: repo.cm,
		lifecycle:     lifecycle,
		logger:        zap.NewNop(),
	}
	d.MCPDispatcher = newMCPDispatcher(srv, repo.mi, zap.NewNop())
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

	client, err := daemon.DialTo(socket, daemon.Handshake{
		Mode:       daemon.ModeMCP,
		CWD:        repo.root,
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
		"params": map[string]any{
			"name":      "get_symbol",
			"arguments": map[string]any{"id": repo.programID, "detail": "full", "format": "json"},
		},
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
	require.Nil(t, resp.Error, "get_symbol must not error: %s", reply)
	require.False(t, resp.Result.IsError, "get_symbol returned a tool error: %s", reply)
	require.NotEmpty(t, resp.Result.Content, "no content: %s", reply)

	var payload struct {
		Node    map[string]any   `json:"node"`
		InEdges []map[string]any `json:"in_edges"`
	}
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &payload))
	require.Equal(t, repo.programID, payload.Node["id"])
	assertCobolTracerProgram(t, payload.Node, repo.prefix)

	var definedByFile bool
	for _, e := range payload.InEdges {
		if e["kind"] == string(graph.EdgeDefines) && e["from"] == repo.fileID {
			definedByFile = true
		}
	}
	assert.True(t, definedByFile, "in_edges lacks defines from %s: %v", repo.fileID, payload.InEdges)
}
