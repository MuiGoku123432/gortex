package graphdump

import (
	"context"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
)

// TestGraphdump_Dump locks the Dump contract (D-01, D-02) on a seeded
// SQLite store: sorted rows, a stable digest, justified and versioned
// volatile keys only, and a FirstDiff that names the first differing index.
func TestGraphdump_Dump(t *testing.T) {
	store, err := store_sqlite.Open(filepath.Join(t.TempDir(), "graph.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	node := func(id, name string) *graph.Node {
		return &graph.Node{
			ID: id, Kind: graph.KindFunction, Name: name, FilePath: "fx/repo/a.go",
			RepoPrefix: "fx/repo", WorkspaceID: "ws", ProjectID: "project",
			Meta: map[string]any{"keep": "k", "stamp": "2026-10-12T00:00:00Z"},
		}
	}
	store.AddBatch([]*graph.Node{node("fx/repo/a.go::Zeta", "Zeta"), node("fx/repo/a.go::Alpha", "Alpha")},
		[]*graph.Edge{{From: "fx/repo/a.go::Zeta", To: "fx/repo/a.go::Alpha", Kind: graph.EdgeCalls, FilePath: "fx/repo/a.go", Line: 3,
			Meta: map[string]any{"stamp": "2026-10-12T00:00:00Z"}}})
	scope := graph.ProjectionScope{Workspace: "ws", Project: "project", Repositories: []string{"fx/repo"}}
	ctx := context.Background()

	full, err := Dump(ctx, store, scope, DeterministicVolatile)
	require.NoError(t, err)
	require.Len(t, full.Rows, 3)
	require.True(t, sort.StringsAreSorted(full.Rows))
	require.Equal(t, "graphdump-volatile/3", full.VolatileVersion)
	require.Contains(t, strings.Join(full.Rows, "\n"), `"stamp"`, "a key outside the allowlist stays in the rows")
	again, err := Dump(ctx, store, scope, DeterministicVolatile)
	require.NoError(t, err)
	require.Equal(t, full.Digest, again.Digest)
	require.Equal(t, full.Rows, again.Rows)

	v := Volatile{Version: "test-volatile/1", Keys: map[string]string{"stamp": "wall-clock write time"}}
	masked, err := Dump(ctx, store, scope, v)
	require.NoError(t, err)
	require.Equal(t, "test-volatile/1", masked.VolatileVersion)
	require.Len(t, masked.Rows, 3)
	for _, row := range masked.Rows {
		require.NotContains(t, row, "stamp")
	}
	require.NotEqual(t, full.Digest, masked.Digest)

	_, err = Dump(ctx, store, scope, Volatile{Keys: map[string]string{"stamp": "wall-clock write time"}})
	require.ErrorContains(t, err, "no version")
	_, err = Dump(ctx, store, scope, Volatile{Version: "test-volatile/1", Keys: map[string]string{"stamp": " "}})
	require.ErrorContains(t, err, "no justification")

	require.Empty(t, FirstDiff(full.Rows, again.Rows, 5))
	got := slices.Clone(full.Rows)
	got[1] += "x"
	diff := FirstDiff(full.Rows, got, 1)
	require.Equal(t, "- [1] "+strings.ReplaceAll(full.Rows[1], "\x1f", " | ")+"\n", diff)
}

// TestGraphdump_EmptiedMetaIsNull locks the v3 rule: a Meta left empty by
// stripping volatile keys renders exactly like a node with no Meta, so an
// incrementally patched file node equals its clean build.
func TestGraphdump_EmptiedMetaIsNull(t *testing.T) {
	store, err := store_sqlite.Open(filepath.Join(t.TempDir(), "graph.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	node := func(id string, meta map[string]any) *graph.Node {
		return &graph.Node{
			ID: id, Kind: graph.KindFile, Name: id, FilePath: id,
			RepoPrefix: "fx/repo", WorkspaceID: "ws", ProjectID: "project", Meta: meta,
		}
	}
	store.AddBatch([]*graph.Node{
		node("fx/repo/clean.yaml", nil),
		node("fx/repo/patched.yaml", map[string]any{"source_core_fingerprint": "f"}),
	}, nil)
	scope := graph.ProjectionScope{Workspace: "ws", Project: "project", Repositories: []string{"fx/repo"}}
	res, err := Dump(context.Background(), store, scope, DeterministicVolatile)
	require.NoError(t, err)
	require.Len(t, res.Rows, 2)
	for _, row := range res.Rows {
		require.True(t, strings.HasSuffix(row, "\x1fnull"), "row %q", row)
	}
}
