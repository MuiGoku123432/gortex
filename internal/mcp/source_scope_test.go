package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
	"github.com/zzet/gortex/internal/indexer"
	"github.com/zzet/gortex/internal/query"
	"github.com/zzet/gortex/internal/search"
)

// The source gate (D-13, D-20) is exercised through srv.wrapToolHandler so
// the real middleware runs: the gate reads the request the middleware
// stores in ctx, and a handler called bare would carry none.

const alphaThingID = "alpha/main.go::AlphaThing"

// callSourceTool runs handler for tool through the production middleware.
func callSourceTool(t *testing.T, srv *Server, ctx context.Context, handler func(context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error), tool string, args map[string]any) *mcplib.CallToolResult {
	t.Helper()
	res, err := srv.wrapToolHandler(handler)(ctx, makeReq(tool, args))
	require.NoError(t, err)
	require.NotNil(t, res)
	return res
}

func sourceScopeText(res *mcplib.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(mcplib.TextContent); ok {
		return tc.Text
	}
	return ""
}

// TestSourceScope_UnboundMultiRepoRefused: with two tracked repositories and
// no narrowing, every source-returning tool refuses. newTwoRepoServer sets no
// server-default workspace or active project, and context.Background() carries
// no session, so no default narrowing applies to these requests.
func TestSourceScope_UnboundMultiRepoRefused(t *testing.T) {
	srv, _ := newTwoRepoServer(t)
	ctx := context.Background()

	cases := []struct {
		tool    string
		handler func(context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error)
		args    map[string]any
	}{
		{"read_file", srv.handleReadFile, map[string]any{"path": "alpha/main.go"}},
		{"get_symbol_source", srv.handleGetSymbolSource, map[string]any{"id": alphaThingID}},
		{"batch_symbols", srv.handleBatchSymbols, map[string]any{"ids": []any{alphaThingID}, "include_source": true}},
		{"get_editing_context", srv.handleGetEditingContext, map[string]any{"path": "alpha/main.go", "compress_bodies": true}},
		{"smart_context", srv.handleSmartContext, map[string]any{"task": "AlphaThing", "format": "json"}},
		{"export_context", srv.handleExportContext, map[string]any{"task": "AlphaThing"}},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			res := callSourceTool(t, srv, ctx, tc.handler, tc.tool, tc.args)
			text := sourceScopeText(res)
			require.Truef(t, res.IsError, "%s must refuse an unbound multi-repo request, got: %s", tc.tool, text)
			assert.Contains(t, text, "repo")
			assert.Contains(t, text, "project")
			assert.Contains(t, text, "workspace")
			assert.NotContains(t, text, "beta")
			assert.NotContains(t, text, "ZebraWidget")
			assert.NotContains(t, text, "func AlphaThing", "no source bytes may leak through a refusal")
		})
	}
}

// TestSourceScope_NarrowedAllowed: an explicit repo narrowing or a bound
// session reads in-scope source; an out-of-scope path gets the not-found
// answer whether or not the file exists.
func TestSourceScope_NarrowedAllowed(t *testing.T) {
	srv, alphaDir := newTwoRepoServer(t)
	ctx := context.Background()

	res := callSourceTool(t, srv, ctx, srv.handleReadFile, "read_file",
		map[string]any{"path": "alpha/main.go", "repo": "alpha"})
	require.Falsef(t, res.IsError, "repo-narrowed read_file must succeed: %s", sourceScopeText(res))
	assert.Contains(t, sourceScopeText(res), "AlphaThing")

	res = callSourceTool(t, srv, ctx, srv.handleGetSymbolSource, "get_symbol_source",
		map[string]any{"id": alphaThingID, "repo": "alpha"})
	require.Falsef(t, res.IsError, "repo-narrowed get_symbol_source must succeed: %s", sourceScopeText(res))
	assert.Contains(t, sourceScopeText(res), "AlphaThing")

	res = callSourceTool(t, srv, sessionCtx("s-src-alpha", alphaDir), srv.handleReadFile, "read_file",
		map[string]any{"path": "alpha/main.go"})
	require.Falsef(t, res.IsError, "a session bound to alpha must read alpha: %s", sourceScopeText(res))

	// Out of scope: beta's existing file and beta's missing file answer
	// identically, so the refusal does not reveal existence.
	existing := callSourceTool(t, srv, ctx, srv.handleReadFile, "read_file",
		map[string]any{"path": "beta/main.go", "repo": "alpha"})
	missing := callSourceTool(t, srv, ctx, srv.handleReadFile, "read_file",
		map[string]any{"path": "beta/nope.go", "repo": "alpha"})
	require.True(t, existing.IsError)
	require.True(t, missing.IsError)
	assert.Equal(t,
		sourceScopeText(missing),
		strings.ReplaceAll(sourceScopeText(existing), "beta/main.go", "beta/nope.go"),
		"an out-of-scope file must answer exactly like an out-of-scope missing file")
	assert.NotContains(t, sourceScopeText(existing), "ZebraWidget")

	// The same not-found answer a missing file in alpha gets.
	inScopeMissing := callSourceTool(t, srv, ctx, srv.handleReadFile, "read_file",
		map[string]any{"path": "alpha/nope.go", "repo": "alpha"})
	require.True(t, inScopeMissing.IsError)
	assert.Contains(t, sourceScopeText(inScopeMissing), "no such file or directory")
	assert.Contains(t, sourceScopeText(existing), "no such file or directory")

	// An out-of-scope symbol answers like a missing one.
	sym := callSourceTool(t, srv, ctx, srv.handleGetSymbolSource, "get_symbol_source",
		map[string]any{"id": "beta/main.go::ZebraWidget", "repo": "alpha"})
	require.True(t, sym.IsError)
	assert.Equal(t, "symbol not found: beta/main.go::ZebraWidget", sourceScopeText(sym))

	rename := callSourceTool(t, srv, ctx, srv.handleRenameSymbol, "rename_symbol",
		map[string]any{"id": "beta/main.go::ZebraWidget", "new_name": "Renamed", "repo": "alpha"})
	require.True(t, rename.IsError)
	assert.Equal(t, "symbol not found: beta/main.go::ZebraWidget", sourceScopeText(rename))
}

// TestSourceScope_WriteToolsGated: an unbound multi-repo write is refused and
// the bytes on disk stay put (D-20).
func TestSourceScope_WriteToolsGated(t *testing.T) {
	srv, alphaDir := newTwoRepoServer(t)
	ctx := context.Background()
	target := filepath.Join(alphaDir, "main.go")
	before, err := os.ReadFile(target)
	require.NoError(t, err)

	res := callSourceTool(t, srv, ctx, srv.handleEditFile, "edit_file", map[string]any{
		"path": "alpha/main.go", "old_string": "AlphaThing", "new_string": "AlphaRenamed",
	})
	require.Truef(t, res.IsError, "unbound edit_file must be refused: %s", sourceScopeText(res))

	res = callSourceTool(t, srv, ctx, srv.handleWriteFile, "write_file", map[string]any{
		"path": "alpha/main.go", "content": "package main\n\nfunc Overwritten() {}\n",
	})
	require.Truef(t, res.IsError, "unbound write_file must be refused: %s", sourceScopeText(res))

	// rename_symbol writes through resolveGraphPath, the third choke point.
	res = callSourceTool(t, srv, ctx, srv.handleRenameSymbol, "rename_symbol", map[string]any{
		"id": alphaThingID, "new_name": "AlphaRenamed",
	})
	require.Truef(t, res.IsError, "unbound rename_symbol must be refused: %s", sourceScopeText(res))

	after, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "a refused write must leave the file unchanged")
}

// TestSourceScope_SingleRepoNoop: with one tracked repository there is
// nothing to be ambiguous about, so an unbound read succeeds.
func TestSourceScope_SingleRepoNoop(t *testing.T) {
	alphaDir := filepath.Join(t.TempDir(), "alpha")
	require.NoError(t, os.MkdirAll(alphaDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(alphaDir, "main.go"),
		[]byte("package main\n\nfunc AlphaThing() {}\n"), 0o644))

	tmpCfg := filepath.Join(t.TempDir(), "config.yaml")
	gc := &config.GlobalConfig{Repos: []config.RepoEntry{{Path: alphaDir, Name: "alpha", Workspace: "shared"}}}
	gc.SetConfigPath(tmpCfg)
	require.NoError(t, gc.Save())
	cm, err := config.NewConfigManager(tmpCfg)
	require.NoError(t, err)

	store, err := store_sqlite.Open(filepath.Join(t.TempDir(), "fts.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	backend := search.NewSymbolSearcherBackend(store)
	mi := indexer.NewMultiIndexer(store, testRegistry(), backend, cm, zap.NewNop())
	_, err = mi.IndexScoped("", "")
	require.NoError(t, err)

	eng := query.NewEngine(store)
	eng.SetSearch(backend)
	srv := NewServer(eng, store, nil, nil, zap.NewNop(), nil, MultiRepoOptions{MultiIndexer: mi, ConfigManager: cm})

	res := callSourceTool(t, srv, context.Background(), srv.handleReadFile, "read_file",
		map[string]any{"path": "alpha/main.go"})
	require.Falsef(t, res.IsError, "single-repo unbound read must succeed: %s", sourceScopeText(res))
	assert.Contains(t, sourceScopeText(res), "AlphaThing")
}

// TestSourceScope_MetadataUnchanged: graph-metadata tools keep the
// all-repositories default for an unbound session (D-13).
func TestSourceScope_MetadataUnchanged(t *testing.T) {
	srv, _ := newTwoRepoServer(t)
	for _, name := range []string{"AlphaThing", "ZebraWidget"} {
		res := callSourceTool(t, srv, context.Background(), srv.handleSearchSymbols, "search_symbols",
			map[string]any{"query": name, "format": "json"})
		require.Falsef(t, res.IsError, "unbound search_symbols must succeed: %s", sourceScopeText(res))
		assert.Contains(t, sourceScopeText(res), name)
	}
}

// TestSourceScope_SymlinkIntoOtherRepoRefused: a symlink inside alpha that
// points into beta must not let an alpha-scoped request read or write beta's
// bytes. Root confinement accepts any tracked root, so the gate has to check
// the repository that physically owns the resolved location too (CR-01).
func TestSourceScope_SymlinkIntoOtherRepoRefused(t *testing.T) {
	srv, alphaDir := newTwoRepoServer(t)
	betaDir, ok := srv.multiIndexer.RepoRoot("beta")
	require.True(t, ok)
	require.NoError(t, os.Symlink(betaDir, filepath.Join(alphaDir, "vendor")))
	betaMain := filepath.Join(betaDir, "main.go")
	before, err := os.ReadFile(betaMain)
	require.NoError(t, err)
	ctx := context.Background()

	for _, path := range []string{"alpha/vendor/main.go", filepath.Join(alphaDir, "vendor", "main.go")} {
		res := callSourceTool(t, srv, ctx, srv.handleReadFile, "read_file",
			map[string]any{"path": path, "repo": "alpha"})
		require.Truef(t, res.IsError, "read_file %s through a symlink into beta must be refused: %s", path, sourceScopeText(res))
		assert.NotContains(t, sourceScopeText(res), "ZebraWidget")
	}

	res := callSourceTool(t, srv, ctx, srv.handleEditFile, "edit_file", map[string]any{
		"path": "alpha/vendor/main.go", "repo": "alpha", "old_string": "ZebraWidget", "new_string": "Hijacked",
	})
	require.Truef(t, res.IsError, "edit_file through a symlink into beta must be refused: %s", sourceScopeText(res))
	res = callSourceTool(t, srv, ctx, srv.handleWriteFile, "write_file", map[string]any{
		"path": "alpha/vendor/main.go", "repo": "alpha", "content": "package main\n\nfunc Hijacked() {}\n",
	})
	require.Truef(t, res.IsError, "write_file through a symlink into beta must be refused: %s", sourceScopeText(res))
	after, err := os.ReadFile(betaMain)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "beta's bytes must stay put")

	// The absolute arm of resolveGraphPath (rename, scaffold) applies the same rule.
	alphaCtx := withSourceScopeRequest(ctx, makeReq("rename_symbol", map[string]any{"repo": "alpha"}))
	_, err = srv.resolveGraphPath(alphaCtx, filepath.Join(alphaDir, "vendor", "main.go"))
	require.ErrorIs(t, err, errSourceOutOfScope)
	// The repository that owns the bytes still reads them.
	betaCtx := withSourceScopeRequest(ctx, makeReq("rename_symbol", map[string]any{"repo": "beta"}))
	_, err = srv.resolveGraphPath(betaCtx, betaMain)
	require.NoError(t, err)
}

// newNestedCheckoutServer builds alpha at <tmp>/alpha and beta at
// <tmp>/alpha/nested, both in workspace "shared", over a real SQLite store.
// With trackBetaLater, alpha is indexed alone first and beta is tracked
// afterwards, so the store keeps alpha-prefixed nodes for beta's files.
func newNestedCheckoutServer(t *testing.T, trackBetaLater bool) (*Server, string, string) {
	t.Helper()
	alphaDir := filepath.Join(t.TempDir(), "alpha")
	betaDir := filepath.Join(alphaDir, "nested")
	require.NoError(t, os.MkdirAll(betaDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(alphaDir, "main.go"),
		[]byte("package main\n\nfunc AlphaThing() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(betaDir, "main.go"),
		[]byte("package main\n\nfunc ZebraWidget() {}\n"), 0o644))

	alpha := config.RepoEntry{Path: alphaDir, Name: "alpha", Workspace: "shared"}
	beta := config.RepoEntry{Path: betaDir, Name: "beta", Workspace: "shared"}
	repos := []config.RepoEntry{alpha, beta}
	if trackBetaLater {
		// Beta stays out of the config: plan 04-18 seeds walk pruning from
		// the config, so a config entry would stop alpha from indexing the
		// nested files and remove the state this fixture must reproduce.
		repos = []config.RepoEntry{alpha}
	}
	tmpCfg := filepath.Join(t.TempDir(), "config.yaml")
	gc := &config.GlobalConfig{Repos: repos}
	gc.SetConfigPath(tmpCfg)
	require.NoError(t, gc.Save())
	cm, err := config.NewConfigManager(tmpCfg)
	require.NoError(t, err)
	store, err := store_sqlite.Open(filepath.Join(t.TempDir(), "fts.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	backend := search.NewSymbolSearcherBackend(store)
	mi := indexer.NewMultiIndexer(store, testRegistry(), backend, cm, zap.NewNop())
	_, err = mi.IndexScoped("", "")
	require.NoError(t, err)
	if trackBetaLater {
		_, err = mi.TrackRepoCtx(context.Background(), beta)
		require.NoError(t, err)
	}
	eng := query.NewEngine(store)
	eng.SetSearch(backend)
	srv := NewServer(eng, store, nil, nil, zap.NewNop(), nil, MultiRepoOptions{MultiIndexer: mi, ConfigManager: cm})
	return srv, alphaDir, betaDir
}

// TestSourceScope_NestedCheckoutRefused: a tracked repository whose root sits
// inside another's must not be readable through the outer repository's
// spelling by a request scoped to the outer repository (CR-01), whichever
// order the two were indexed in.
func TestSourceScope_NestedCheckoutRefused(t *testing.T) {
	const nestedID = "alpha/nested/main.go::ZebraWidget"
	const zebraSource = "func ZebraWidget() {}"

	run := func(t *testing.T, srv *Server, betaDir string) {
		ctx := context.Background()
		t.Run("read_file", func(t *testing.T) {
			res := callSourceTool(t, srv, ctx, srv.handleReadFile, "read_file",
				map[string]any{"path": "alpha/nested/main.go", "repo": "alpha"})
			require.Truef(t, res.IsError, "an alpha-scoped read of beta's nested checkout must be refused: %s", sourceScopeText(res))
			assert.NotContains(t, sourceScopeText(res), "ZebraWidget")

			res = callSourceTool(t, srv, ctx, srv.handleReadFile, "read_file",
				map[string]any{"path": "alpha/main.go", "repo": "alpha"})
			require.Falsef(t, res.IsError, "alpha's own file must still read: %s", sourceScopeText(res))
			assert.Contains(t, sourceScopeText(res), "AlphaThing")
		})
		t.Run("get_symbol_source", func(t *testing.T) {
			res := callSourceTool(t, srv, ctx, srv.handleGetSymbolSource, "get_symbol_source",
				map[string]any{"id": nestedID, "repo": "alpha"})
			require.Truef(t, res.IsError, "an alpha-scoped get_symbol_source of beta's file must be refused: %s", sourceScopeText(res))
			assert.Contains(t, sourceScopeText(res), "symbol not found")
			assert.NotContains(t, sourceScopeText(res), zebraSource)
		})
		t.Run("batch_symbols", func(t *testing.T) {
			res := callSourceTool(t, srv, ctx, srv.handleBatchSymbols, "batch_symbols",
				map[string]any{"ids": []any{nestedID}, "include_source": true, "repo": "alpha"})
			assert.NotContains(t, sourceScopeText(res), zebraSource)
		})
		t.Run("search_text", func(t *testing.T) {
			res := callSourceTool(t, srv, ctx, srv.handleSearchText, "search_text",
				map[string]any{"query": "Zebra", "repo": "alpha", "format": "json"})
			require.Falsef(t, res.IsError, "an alpha-scoped search_text must succeed: %s", sourceScopeText(res))
			assert.NotContains(t, sourceScopeText(res), "func ZebraWidget")

			res = callSourceTool(t, srv, ctx, srv.handleSearchText, "search_text",
				map[string]any{"query": "Zebra", "repo": "beta", "format": "json"})
			require.Falsef(t, res.IsError, "a beta-scoped search_text must succeed: %s", sourceScopeText(res))
			assert.Contains(t, sourceScopeText(res), "beta/main.go")
		})
		t.Run("find_declaration", func(t *testing.T) {
			alphaCtx := withSourceScopeRequest(ctx, makeReq("find_declaration", map[string]any{"repo": "alpha"}))
			for _, isRegex := range []bool{false, true} {
				matches, err := srv.findUseSiteMatches(alphaCtx, "Zebra", isRegex, "", 100)
				require.NoError(t, err)
				for _, m := range matches {
					assert.Falsef(t, strings.HasPrefix(m.Path, "alpha/nested/") || strings.Contains(m.Text, "ZebraWidget"),
						"an alpha-scoped use site from beta's nested checkout (regex=%v): %s %q", isRegex, m.Path, m.Text)
				}
			}
		})
		t.Run("rename_symbol", func(t *testing.T) {
			betaMain := filepath.Join(betaDir, "main.go")
			before, err := os.ReadFile(betaMain)
			require.NoError(t, err)
			res := callSourceTool(t, srv, ctx, srv.handleRenameSymbol, "rename_symbol",
				map[string]any{"id": nestedID, "new_name": "ZebraRenamed", "repo": "alpha"})
			require.Truef(t, res.IsError, "an alpha-scoped rename of beta's file must be refused: %s", sourceScopeText(res))
			assert.NotContains(t, sourceScopeText(res), zebraSource)
			after, err := os.ReadFile(betaMain)
			require.NoError(t, err)
			assert.Equal(t, string(before), string(after), "beta's main.go must be unchanged on disk")

			res = callSourceTool(t, srv, ctx, srv.handleRenameSymbol, "rename_symbol",
				map[string]any{"id": alphaThingID, "new_name": "AlphaRenamed", "repo": "alpha", "dry_run": true})
			require.Falsef(t, res.IsError, "an alpha-scoped dry-run rename of alpha's own symbol must succeed: %s", sourceScopeText(res))
		})
		t.Run("in_scope_reads", func(t *testing.T) {
			res := callSourceTool(t, srv, ctx, srv.handleGetSymbolSource, "get_symbol_source",
				map[string]any{"id": "beta/main.go::ZebraWidget", "repo": "beta"})
			require.Falsef(t, res.IsError, "a beta-scoped read of beta's own symbol must succeed: %s", sourceScopeText(res))
			assert.Contains(t, sourceScopeText(res), zebraSource)

			res = callSourceTool(t, srv, ctx, srv.handleGetSymbolSource, "get_symbol_source",
				map[string]any{"id": alphaThingID, "repo": "alpha"})
			require.Falsef(t, res.IsError, "alpha's own symbol must still read: %s", sourceScopeText(res))
			assert.Contains(t, sourceScopeText(res), "func AlphaThing() {}")
		})
	}

	t.Run("tracked_together", func(t *testing.T) {
		srv, _, betaDir := newNestedCheckoutServer(t, false)
		run(t, srv, betaDir)
	})
	t.Run("outer_indexed_first", func(t *testing.T) {
		srv, _, betaDir := newNestedCheckoutServer(t, true)
		require.NotNilf(t, srv.graph.GetNode(nestedID),
			"the fixture must reproduce a store indexed before the nested repository was tracked: %s is missing", nestedID)
		run(t, srv, betaDir)
	})
}

// TestSourceScope_TextSearchToolsGated: search_text and find_declaration
// return matching source lines, so an unbound multi-repo request is refused
// before any search runs, and a bound request never sees another
// repository's lines (CR-02).
func TestSourceScope_TextSearchToolsGated(t *testing.T) {
	srv, _ := newTwoRepoServer(t)
	ctx := context.Background()

	cases := []struct {
		tool    string
		handler func(context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error)
		args    map[string]any
	}{
		{"search_text", srv.handleSearchText, map[string]any{"query": "func ", "format": "json"}},
		{"search_text", srv.handleSearchText, map[string]any{"query": "NoSuchStringAnywhere", "format": "json"}},
		{"search_text", srv.handleSearchText, map[string]any{"query": ".", "regexp": true, "format": "json"}},
		{"find_declaration", srv.handleFindDeclaration, map[string]any{"use_site": "func ", "format": "json"}},
	}
	for _, tc := range cases {
		res := callSourceTool(t, srv, ctx, tc.handler, tc.tool, tc.args)
		text := sourceScopeText(res)
		require.Truef(t, res.IsError, "%s %v must refuse an unbound multi-repo request, got: %s", tc.tool, tc.args, text)
		assert.Contains(t, text, "repo")
		assert.Contains(t, text, "workspace")
		assert.NotContains(t, text, "ZebraWidget")
		assert.NotContains(t, text, "AlphaThing")
	}

	// Bound to alpha: only alpha's lines come back.
	res := callSourceTool(t, srv, ctx, srv.handleSearchText, "search_text",
		map[string]any{"query": "func ", "repo": "alpha", "format": "json"})
	require.Falsef(t, res.IsError, "repo-narrowed search_text must succeed: %s", sourceScopeText(res))
	assert.Contains(t, sourceScopeText(res), "AlphaThing")
	assert.NotContains(t, sourceScopeText(res), "ZebraWidget")

	alphaCtx := withSourceScopeRequest(ctx, makeReq("find_declaration", map[string]any{"repo": "alpha"}))
	for _, isRegex := range []bool{false, true} {
		matches, err := srv.findUseSiteMatches(alphaCtx, "func ", isRegex, "", 100)
		require.NoError(t, err)
		require.NotEmpty(t, matches)
		for _, m := range matches {
			assert.Truef(t, strings.HasPrefix(m.Path, "alpha/"), "find_declaration use site outside the scope: %s %q", m.Path, m.Text)
		}
	}
}
