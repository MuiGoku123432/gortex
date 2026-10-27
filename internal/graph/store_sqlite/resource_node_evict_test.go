package store_sqlite

import (
	"path/filepath"
	"testing"

	"github.com/zzet/gortex/internal/graph"
)

func TestSQLiteEvictResourceNodesByIDsIsKindGuardedAndSetOriented(t *testing.T) {
	const (
		repo     = "r1"
		file     = "r1/src/app.cbl"
		resource = "r1::cobol-missing::program/NOPGM"
		finding  = "r1/src/app.cbl::@finding/external/0123456789abcdef0123456789abcdef"
		function = "r1/src/app.cbl::APP"
	)
	store, err := Open(filepath.Join(t.TempDir(), "graph.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	store.AddBatch([]*graph.Node{
		{ID: resource, Kind: graph.KindResource, Name: "NOPGM", RepoPrefix: repo},
		{ID: finding, Kind: graph.KindFinding, Name: "external-unresolved", FilePath: file, RepoPrefix: repo},
		{ID: function, Kind: graph.KindFunction, Name: "APP", FilePath: file, RepoPrefix: repo},
		{ID: file, Kind: graph.KindFile, Name: "app.cbl", FilePath: file, RepoPrefix: repo},
	}, []*graph.Edge{
		{From: finding, To: resource, Kind: graph.EdgeReferences, FilePath: file},
		{From: function, To: resource, Kind: graph.EdgeCalls, FilePath: file},
		{From: finding, To: function, Kind: graph.EdgeMemberOf, FilePath: file},
		{From: file, To: function, Kind: graph.EdgeDefines, FilePath: file},
	})

	nodes, edges := store.EvictResourceNodesByIDs([]string{resource, resource, finding, function, file, ""})
	if nodes != 1 || edges != 2 {
		t.Fatalf("EvictResourceNodesByIDs = (%d,%d), want (1,2)", nodes, edges)
	}
	if store.GetNode(resource) != nil {
		t.Fatalf("resource %q survived eviction", resource)
	}
	for _, id := range []string{finding, function, file} {
		if store.GetNode(id) == nil {
			t.Fatalf("kind guard failed: %q was evicted", id)
		}
	}
	if got := store.EdgeCount(); got != 2 {
		t.Fatalf("EdgeCount = %d, want 2 (edges not touching the resource survive)", got)
	}
	if got := len(store.GetInEdges(function)); got != 2 {
		t.Fatalf("in-edges of %q = %d, want 2", function, got)
	}
}
