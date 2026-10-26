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

// cobolFindingsFixture is an invented 7-column fixed-format program. Its
// EXEC SQL statement leaves one producer unparsed-tail gap on the token
// ZQXCOL; the CALL and COPY name artifacts that do not exist, for the
// external findings later plans add to this fixture. It must never be
// replaced with estate-derived source: this repository is public.
const cobolFindingsFixture = "       IDENTIFICATION DIVISION.\n" +
	"       PROGRAM-ID. FINDPGM.\n" +
	"       PROCEDURE DIVISION.\n" +
	"       MAIN-PARA.\n" +
	"           EXEC SQL SELECT ZQXCOL FROM APP.ZQXTAB END-EXEC.\n" +
	"           CALL 'NOPGM'.\n" +
	"           COPY NOCPY.\n" +
	"           STOP RUN.\n"

// cobolFindingsTailToken is the source text of the tail gap. No finding
// Name, ID, or Meta value may carry it (GAP-01 privacy).
const cobolFindingsTailToken = "ZQXCOL"

// cobolFindingsRepo is the tracked, indexed findings fixture repository.
type cobolFindingsRepo struct {
	dir, root string
	store     *store_sqlite.Store
	idx       *indexer.Indexer
	mi        *indexer.MultiIndexer
	cm        *config.ConfigManager
	prefix    string
	fileID    string
	tailID    string
}

// indexCobolFindingsRepo commits the findings fixture to a fresh git
// repository and tracks it through the real MultiIndexer into a SQLite
// store, shadow off (AddBatch, not BulkLoad) and AI providers off, then
// locates the persisted unparsed-tail finding of cobol/findpgm.cbl. It
// asserts only on that finding, never on a total finding count.
func indexCobolFindingsRepo(t *testing.T) cobolFindingsRepo {
	t.Helper()
	t.Setenv("GORTEX_SHADOW_MAX_FILES", "0")
	t.Setenv("GORTEX_LLM_PROVIDER", "")

	dir := testenv.ShortTempDir(t)
	root := filepath.Join(dir, "repo")
	gitInitRepo(t, root)
	rel := "cobol/findpgm.cbl"
	require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, rel), []byte(cobolFindingsFixture), 0o644))
	for _, args := range [][]string{{"add", rel}, {"commit", "-m", "add cobol findings fixture"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}

	store, err := store_sqlite.Open(filepath.Join(dir, "store.sqlite"))
	require.NoError(t, err)
	recorder := &cobolAddBatchRecorder{Store: store, ids: map[string]bool{}}
	reg := parser.NewRegistry()
	languages.RegisterAll(reg)
	idx := indexer.New(recorder, reg, config.Default().Index, zap.NewNop())
	cm, err := config.NewConfigManager(filepath.Join(dir, "config.yaml"))
	require.NoError(t, err)
	mi := indexer.NewMultiIndexer(recorder, reg, idx.Search(), cm, zap.NewNop())
	t.Cleanup(func() {
		_ = mi.Close(context.Background())
		_ = store.Close()
	})
	res, err := mi.TrackRepoCtx(context.Background(), config.RepoEntry{Path: root})
	require.NoError(t, err)
	require.NotEmpty(t, res.RepoPrefix)

	repo := cobolFindingsRepo{
		dir: dir, root: root, store: store, idx: idx, mi: mi, cm: cm,
		prefix: res.RepoPrefix, fileID: res.RepoPrefix + "/" + rel,
	}
	var tails []string
	for n := range store.NodesByKind(graph.KindFinding) {
		if n.FilePath == repo.fileID && n.Meta[graph.MetaFindingKind] == "unparsed-tail" {
			tails = append(tails, n.ID)
		}
	}
	require.Len(t, tails, 1, "want one unparsed-tail finding in %s", repo.fileID)
	repo.tailID = tails[0]
	// Asserted before any retrieval: get_symbol's ensureFresh may reindex.
	require.True(t, recorder.saw(repo.tailID), "AddBatch never received %s", repo.tailID)
	return repo
}

// cobolFindingSymbol is the get_symbol detail=full payload.
type cobolFindingSymbol struct {
	Node     map[string]any   `json:"node"`
	OutEdges []map[string]any `json:"out_edges"`
}

// assertCobolTailFinding checks the decoded get_symbol payload of the tail
// finding against the GAP-04 contract. Numbers compare as float64 because
// JSON decoding yields float64.
func assertCobolTailFinding(t *testing.T, repo cobolFindingsRepo, payload cobolFindingSymbol) {
	t.Helper()
	node := payload.Node
	require.Equal(t, repo.tailID, node["id"])
	idPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(repo.fileID) + `::@finding/parse/[0-9a-f]{32}$`)
	assert.Regexp(t, idPattern, node["id"])
	assert.Equal(t, string(graph.KindFinding), node["kind"])
	assert.Equal(t, "unparsed-tail", node["name"])
	assert.Equal(t, repo.fileID, node["file_path"])

	meta, ok := node["meta"].(map[string]any)
	require.True(t, ok, "node has no meta: %v", node)
	assert.Equal(t, graph.FindingClassParse, meta[graph.MetaEvidenceClass])
	assert.Equal(t, "unparsed-tail", meta[graph.MetaFindingKind])
	assert.Equal(t, "low", meta[graph.MetaFindingSeverity])
	assert.Equal(t, graph.FindingSeverityRule, meta[graph.MetaFindingSeverityRule])
	assert.Equal(t, graph.FindingStatusActive, meta[graph.MetaFindingStatus])
	assert.Equal(t, graph.FindingIDRule, meta[graph.MetaFindingIDRule])
	assert.Equal(t, graph.FindingEvidenceRule, meta[graph.MetaFindingEvidenceRule])
	assert.Equal(t, graph.FindingEvidenceDigest(cobolFindingsTailToken), meta[graph.MetaFindingEvidenceDigest])
	assert.NotEmpty(t, meta[graph.MetaFindingContainer])
	for _, key := range []string{"prov_parser_tool_id", "prov_parser_grammar_id", "prov_extractor_version"} {
		assert.NotEmpty(t, meta[key], "meta lacks %s", key)
	}
	assert.Equal(t, true, meta["prov_range_exact"])
	start, startOK := meta["prov_start_byte"].(float64)
	end, endOK := meta["prov_end_byte"].(float64)
	require.True(t, startOK && endOK, "meta lacks an exact byte range: %v", meta)
	assert.Equal(t, cobolFindingsTailToken, cobolFindingsFixture[int(start):int(end)])
	for _, key := range []string{"prov_start_row", "prov_start_column", "prov_end_row", "prov_end_column"} {
		assert.Contains(t, meta, key)
	}

	var container string
	for _, e := range payload.OutEdges {
		if e["kind"] == string(graph.EdgeMemberOf) {
			container, _ = e["to"].(string)
		}
	}
	require.NotEmpty(t, container, "out_edges lacks member_of: %v", payload.OutEdges)
	assert.NotNil(t, repo.store.GetNode(container), "member_of target %s is not a node", container)
}

// TestCobolFindings_MCPGetSymbol retrieves the persisted tail finding by its
// exact prefixed ID through the real daemon relay and MCP get_symbol.
func TestCobolFindings_MCPGetSymbol(t *testing.T) {
	repo := indexCobolFindingsRepo(t)
	socket, _ := serveCobolTestDaemon(t, repo.dir, repo.store, repo.idx, repo.mi, repo.cm)

	// detail=full: the brief detail omits Meta and edges.
	text := callCobolTracerTool(t, socket, repo.root, "get_symbol",
		map[string]any{"id": repo.tailID, "detail": "full", "format": "json"})
	var payload cobolFindingSymbol
	require.NoError(t, json.Unmarshal([]byte(text), &payload), "get_symbol output: %s", text)
	assertCobolTailFinding(t, repo, payload)
}

// TestCobolFindings_CLIGetSymbol retrieves the same finding through the
// matching CLI verb, `gortex call get_symbol`, over the real daemon relay.
func TestCobolFindings_CLIGetSymbol(t *testing.T) {
	repo := indexCobolFindingsRepo(t)
	serveCobolTestDaemon(t, repo.dir, repo.store, repo.idx, repo.mi, repo.cm)

	cmd, buf := newCallTestCmd(t)
	callIndex = repo.root
	callArgs = []string{"id=" + repo.tailID, "detail=full"}
	require.NoError(t, runCall(cmd, []string{"get_symbol"}))

	var payload cobolFindingSymbol
	require.NoError(t, json.Unmarshal(buf.Bytes(), &payload), "CLI output: %s", buf.String())
	assertCobolTailFinding(t, repo, payload)
}

// TestCobolFindings_SearchByNameAndKind finds the tail finding through MCP
// search_symbols filtered to kind finding by its name term, and proves the
// tail's source text is in neither its Name nor any Meta value.
func TestCobolFindings_SearchByNameAndKind(t *testing.T) {
	repo := indexCobolFindingsRepo(t)
	socket, _ := serveCobolTestDaemon(t, repo.dir, repo.store, repo.idx, repo.mi, repo.cm)

	text := callCobolTracerTool(t, socket, repo.root, "search_symbols",
		map[string]any{"query": "kind:finding unparsed", "format": "json"})
	var found struct {
		Results []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &found), "search_symbols output: %s", text)
	var ids []string
	for _, r := range found.Results {
		ids = append(ids, r.ID)
		assert.Equal(t, string(graph.KindFinding), r.Kind, "kind:finding returned %s", r.ID)
	}
	assert.Contains(t, ids, repo.tailID)

	node := repo.store.GetNode(repo.tailID)
	require.NotNil(t, node)
	assert.NotContains(t, node.Name, cobolFindingsTailToken)
	assert.NotContains(t, node.QualName, cobolFindingsTailToken)
	for key, value := range node.Meta {
		switch v := value.(type) {
		case string:
			assert.NotContains(t, v, cobolFindingsTailToken, "meta %s carries source text", key)
		case []string:
			for _, s := range v {
				assert.NotContains(t, s, cobolFindingsTailToken, "meta %s carries source text", key)
			}
		case []any:
			for _, s := range v {
				str, _ := s.(string)
				assert.NotContains(t, str, cobolFindingsTailToken, "meta %s carries source text", key)
			}
		}
	}
	assert.False(t, strings.Contains(repo.tailID, cobolFindingsTailToken))
}

// TestCobolFindings_ExternalPersisted indexes the findings fixture through
// the real path and proves its unobtained CALL 'NOPGM' and COPY NOCPY each
// persist in SQLite as one EXTERNAL_UNRESOLVED finding carrying the full
// RESEARCH Pattern 3 Meta, owned by its repository, with a member_of edge
// to an existing node and a references edge to its placeholder. Findings
// are located by finding_target_key, never by recomputing the hash.
func TestCobolFindings_ExternalPersisted(t *testing.T) {
	repo := indexCobolFindingsRepo(t)

	byKey := map[string][]*graph.Node{}
	for n := range repo.store.NodesByKind(graph.KindFinding) {
		if n.Meta[graph.MetaEvidenceClass] == graph.FindingClassExternal {
			key, _ := n.Meta[graph.MetaFindingTargetKey].(string)
			byKey[key] = append(byKey[key], n)
		}
	}
	asInt := func(v any) (int, bool) {
		switch n := v.(type) {
		case int:
			return n, true
		case int64:
			return int(n), true
		case float64:
			return int(n), true
		}
		return 0, false
	}
	idPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(repo.fileID) + `::@finding/external/[0-9a-f]{32}$`)
	for key, want := range map[string][3]string{
		"NOPGM": {"program", repo.prefix + "::cobol-missing::program/NOPGM", "'NOPGM'"},
		"NOCPY": {"copybook", repo.prefix + "::cobol-missing::copybook/NOCPY", ""},
	} {
		require.Len(t, byKey[key], 1, "one EXTERNAL_UNRESOLVED finding for %s", key)
		n := byKey[key][0]
		assert.Regexp(t, idPattern, n.ID)
		assert.Equal(t, repo.fileID, n.FilePath)
		assert.Equal(t, repo.prefix, n.RepoPrefix)
		assert.Equal(t, key, n.Name)

		meta := n.Meta
		for _, k := range []string{"prov_start_row", "prov_start_column", "prov_end_row", "prov_end_column"} {
			_, ok := asInt(meta[k])
			assert.True(t, ok, "%s: meta lacks %s: %v", key, k, meta)
		}
		start, startOK := asInt(meta["prov_start_byte"])
		end, endOK := asInt(meta["prov_end_byte"])
		require.True(t, startOK && endOK, "%s: meta lacks an exact byte range: %v", key, meta)
		assert.Contains(t, cobolFindingsFixture[start:end], key, "the range covers the referencing statement")
		assert.Equal(t, true, meta["prov_range_exact"])
		assert.NotEmpty(t, meta["prov_parser_tool_id"])
		assert.NotEmpty(t, meta["prov_parser_grammar_id"])
		assert.Equal(t, "gortex-cobol-grammar/6", meta["prov_extractor_version"])
		assert.Equal(t, "external-reference", meta[graph.MetaFindingKind])
		assert.Equal(t, "missing", meta[graph.MetaFindingReason])
		assert.Equal(t, "medium", meta[graph.MetaFindingSeverity])
		assert.Equal(t, graph.FindingSeverityRule, meta[graph.MetaFindingSeverityRule])
		assert.Equal(t, "unavailable", meta[graph.MetaFindingScope])
		assert.Equal(t, want[0], meta[graph.MetaFindingExpectedFamily])
		assert.Equal(t, graph.FindingStatusActive, meta[graph.MetaFindingStatus])
		assert.NotEmpty(t, meta[graph.MetaFindingEvidenceDigest])
		assert.Equal(t, graph.FindingEvidenceRule, meta[graph.MetaFindingEvidenceRule])
		assert.Equal(t, graph.FindingIDRule, meta[graph.MetaFindingIDRule])
		assert.Equal(t, want[0], meta[graph.MetaFindingArtifactType])
		assert.Equal(t, key, meta[graph.MetaFindingTargetKey])
		for k, v := range meta {
			assert.False(t, strings.HasPrefix(k, "cobol_"), "%s: no cobol_* key, found %s", key, k)
			if want[2] != "" {
				assert.NotEqual(t, want[2], v, "%s: the as-written target is never stored (%s)", key, k)
			}
		}

		targets := map[graph.EdgeKind]string{}
		for _, e := range repo.store.GetOutEdges(n.ID) {
			targets[e.Kind] = e.To
		}
		require.Len(t, targets, 2, "%s: a member_of and a references edge: %v", key, targets)
		assert.Equal(t, want[1], targets[graph.EdgeReferences])
		require.NotNil(t, repo.store.GetNode(want[1]), "%s: the placeholder exists", key)
		container := targets[graph.EdgeMemberOf]
		require.NotNil(t, repo.store.GetNode(container), "%s: member_of target %s is a node", key, container)
		assert.Equal(t, strings.TrimPrefix(container, repo.prefix+"/"), meta[graph.MetaFindingContainer])
	}

	diag, ok := graph.ReadPrefixDiagnostics(repo.store, 10)
	require.True(t, ok)
	assert.Zero(t, diag.UnownedCodeNodes, "unowned: %v", diag.UnownedSamples)
	assert.Zero(t, diag.MisprefixedNodes, "misprefixed: %v", diag.MisprefixedSamples)

	assert.Regexp(t, `^`+regexp.QuoteMeta(repo.fileID)+`::@finding/parse/[0-9a-f]{32}$`, repo.tailID)
	assert.NotNil(t, repo.store.GetNode(repo.tailID), "the unparsed-tail finding keeps its ID")
}

// cobolFindingsNOPGM locates, through the store and never by recomputing the
// hash, the EXTERNAL_UNRESOLVED finding of CALL 'NOPGM', its placeholder, and
// the From of the code edge that targets the placeholder (the caller node).
func cobolFindingsNOPGM(t *testing.T, repo cobolFindingsRepo) (findingID, placeholderID, callerID string) {
	t.Helper()
	var ids []string
	for n := range repo.store.NodesByKind(graph.KindFinding) {
		if n.Meta[graph.MetaEvidenceClass] == graph.FindingClassExternal && n.Meta[graph.MetaFindingTargetKey] == "NOPGM" {
			ids = append(ids, n.ID)
		}
	}
	require.Len(t, ids, 1, "one EXTERNAL_UNRESOLVED finding for NOPGM")
	placeholderID = repo.prefix + "::cobol-missing::program/NOPGM"
	var callers []string
	for _, e := range repo.store.GetInEdges(placeholderID) {
		if e.Kind == graph.EdgeCalls {
			callers = append(callers, e.From)
		}
	}
	require.Len(t, callers, 1, "one calls edge into %s", placeholderID)
	return ids[0], placeholderID, callers[0]
}

// assertCobolExternalFinding checks the decoded get_symbol detail=full payload
// of the NOPGM external finding against the GAP-04 contract (RESEARCH
// Pattern 3) and its two out-edges (D-08).
func assertCobolExternalFinding(t *testing.T, repo cobolFindingsRepo, payload cobolFindingSymbol) {
	t.Helper()
	findingID, placeholderID, callerID := cobolFindingsNOPGM(t, repo)
	node := payload.Node
	require.Equal(t, findingID, node["id"])
	assert.Equal(t, string(graph.KindFinding), node["kind"])
	assert.Equal(t, repo.fileID, node["file_path"])

	meta, ok := node["meta"].(map[string]any)
	require.True(t, ok, "node has no meta: %v", node)
	for key, want := range map[string]any{
		graph.MetaEvidenceClass:         graph.FindingClassExternal,
		graph.MetaFindingKind:           "external-reference",
		graph.MetaFindingReason:         "missing",
		graph.MetaFindingSeverity:       "medium",
		graph.MetaFindingSeverityRule:   graph.FindingSeverityRule,
		graph.MetaFindingScope:          "unavailable",
		graph.MetaFindingStatus:         graph.FindingStatusActive,
		graph.MetaFindingEvidenceRule:   graph.FindingEvidenceRule,
		graph.MetaFindingIDRule:         graph.FindingIDRule,
		graph.MetaFindingArtifactType:   "program",
		graph.MetaFindingTargetKey:      "NOPGM",
		graph.MetaFindingContainer:      strings.TrimPrefix(callerID, repo.prefix+"/"),
		"prov_range_exact":              true,
		"prov_extractor_version":        "gortex-cobol-grammar/6",
		graph.MetaFindingExpectedFamily: "program",
	} {
		assert.Equal(t, want, meta[key], "meta %s", key)
	}
	assert.NotEmpty(t, meta[graph.MetaFindingEvidenceDigest])
	assert.NotEmpty(t, meta["prov_parser_tool_id"])
	for _, key := range []string{"prov_start_row", "prov_start_column", "prov_end_row", "prov_end_column"} {
		_, isNum := meta[key].(float64)
		assert.True(t, isNum, "meta lacks %s: %v", key, meta)
	}
	start, startOK := meta["prov_start_byte"].(float64)
	end, endOK := meta["prov_end_byte"].(float64)
	require.True(t, startOK && endOK, "meta lacks an exact byte range: %v", meta)
	assert.Contains(t, cobolFindingsFixture[int(start):int(end)], "NOPGM", "the range covers the CALL")

	targets := map[string]string{}
	for _, e := range payload.OutEdges {
		kind, _ := e["kind"].(string)
		targets[kind], _ = e["to"].(string)
	}
	assert.Equal(t, callerID, targets[string(graph.EdgeMemberOf)], "member_of the caller: %v", payload.OutEdges)
	assert.Equal(t, placeholderID, targets[string(graph.EdgeReferences)], "references the placeholder: %v", payload.OutEdges)
}

// TestCobolFindings_ExternalMCPGetSymbol retrieves the EXTERNAL_UNRESOLVED
// finding of CALL 'NOPGM' by its exact prefixed ID through the real daemon
// relay and MCP get_symbol (D-08, P2 D-11).
func TestCobolFindings_ExternalMCPGetSymbol(t *testing.T) {
	repo := indexCobolFindingsRepo(t)
	findingID, _, _ := cobolFindingsNOPGM(t, repo)
	socket, _ := serveCobolTestDaemon(t, repo.dir, repo.store, repo.idx, repo.mi, repo.cm)

	text := callCobolTracerTool(t, socket, repo.root, "get_symbol",
		map[string]any{"id": findingID, "detail": "full", "format": "json"})
	var payload cobolFindingSymbol
	require.NoError(t, json.Unmarshal([]byte(text), &payload), "get_symbol output: %s", text)
	assertCobolExternalFinding(t, repo, payload)
}

// TestCobolFindings_ExternalCLIGetSymbol retrieves the same external finding
// through `gortex call get_symbol` over the real daemon relay.
func TestCobolFindings_ExternalCLIGetSymbol(t *testing.T) {
	repo := indexCobolFindingsRepo(t)
	findingID, _, _ := cobolFindingsNOPGM(t, repo)
	serveCobolTestDaemon(t, repo.dir, repo.store, repo.idx, repo.mi, repo.cm)

	cmd, buf := newCallTestCmd(t)
	callIndex = repo.root
	callArgs = []string{"id=" + findingID, "detail=full"}
	require.NoError(t, runCall(cmd, []string{"get_symbol"}))

	var payload cobolFindingSymbol
	require.NoError(t, json.Unmarshal(buf.Bytes(), &payload), "CLI output: %s", buf.String())
	assertCobolExternalFinding(t, repo, payload)
}

// TestCobolFindings_PlaceholderMCPGetSymbol retrieves the not-obtained
// placeholder of NOPGM through MCP get_symbol and proves it carries no
// fabricated content (GAP-05, D-11): no line, no range, no parser stamp,
// with the code calls edge and the finding's references edge as in-edges.
func TestCobolFindings_PlaceholderMCPGetSymbol(t *testing.T) {
	repo := indexCobolFindingsRepo(t)
	findingID, placeholderID, callerID := cobolFindingsNOPGM(t, repo)
	socket, _ := serveCobolTestDaemon(t, repo.dir, repo.store, repo.idx, repo.mi, repo.cm)

	text := callCobolTracerTool(t, socket, repo.root, "get_symbol",
		map[string]any{"id": placeholderID, "detail": "full", "format": "json"})
	var payload cobolTracerSymbol
	require.NoError(t, json.Unmarshal([]byte(text), &payload), "get_symbol output: %s", text)
	node := payload.Node
	require.Equal(t, placeholderID, node["id"])
	assert.Equal(t, string(graph.KindResource), node["kind"])
	assert.Equal(t, "NOPGM", node["name"])
	assert.Equal(t, repo.prefix, node["repo_prefix"])
	if line, ok := node["start_line"]; ok {
		assert.Zero(t, line, "a placeholder has no start_line")
	}
	assert.NotContains(t, node, "end_line")

	meta, ok := node["meta"].(map[string]any)
	require.True(t, ok, "node has no meta: %v", node)
	assert.Equal(t, "not_obtained", meta["artifact_status"])
	assert.Equal(t, true, meta["synthetic"])
	assert.Equal(t, "program", meta["artifact_type"])
	for _, key := range []string{"prov_start_row", "prov_end_row", "prov_start_byte", "prov_end_byte"} {
		assert.NotContains(t, meta, key, "a placeholder has no range")
	}
	for key := range meta {
		assert.False(t, strings.HasPrefix(key, "prov_parser_"), "a placeholder has no parser stamp, found %s", key)
	}

	var calledBy, referencedBy []string
	for _, e := range payload.InEdges {
		from, _ := e["from"].(string)
		switch e["kind"] {
		case string(graph.EdgeCalls):
			calledBy = append(calledBy, from)
		case string(graph.EdgeReferences):
			referencedBy = append(referencedBy, from)
		}
	}
	assert.Contains(t, calledBy, callerID, "in_edges lacks the code calls edge: %v", payload.InEdges)
	assert.Contains(t, referencedBy, findingID, "in_edges lacks the finding's references edge: %v", payload.InEdges)
}

// TestCobolFindings_PlaceholderSearch finds the placeholder by its key through
// MCP search_symbols, and proves a kind:function search for the same key
// never presents it as an implementation (GRAPH-SCHEMA Required
// Verification, T-05-36). When a kind clause filters every candidate out,
// search_symbols retries on the free text alone and says so with
// filters_relaxed; the placeholder may come back only through that
// declared fallback, and then as a resource, never as a function.
func TestCobolFindings_PlaceholderSearch(t *testing.T) {
	repo := indexCobolFindingsRepo(t)
	_, placeholderID, _ := cobolFindingsNOPGM(t, repo)
	socket, _ := serveCobolTestDaemon(t, repo.dir, repo.store, repo.idx, repo.mi, repo.cm)

	type hit struct {
		ID   string `json:"id"`
		Kind string `json:"kind"`
	}
	search := func(query string) ([]hit, bool) {
		text := callCobolTracerTool(t, socket, repo.root, "search_symbols",
			map[string]any{"query": query, "format": "json"})
		var found struct {
			Results        []hit `json:"results"`
			FiltersRelaxed bool  `json:"filters_relaxed"`
		}
		require.NoError(t, json.Unmarshal([]byte(text), &found), "search_symbols %q output: %s", query, text)
		return found.Results, found.FiltersRelaxed
	}

	hits, _ := search("NOPGM")
	var ids []string
	for _, h := range hits {
		ids = append(ids, h.ID)
	}
	assert.Contains(t, ids, placeholderID)

	hits, relaxed := search("kind:function NOPGM")
	for _, h := range hits {
		if h.ID != placeholderID {
			continue
		}
		assert.True(t, relaxed, "kind:function returned the placeholder without relaxing the kind clause")
		assert.Equal(t, string(graph.KindResource), h.Kind, "the placeholder is never a function")
	}
}
