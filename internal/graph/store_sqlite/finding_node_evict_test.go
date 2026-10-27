package store_sqlite

import (
	"path/filepath"
	"testing"

	"github.com/zzet/gortex/internal/graph"
)

func TestSQLiteEvictFindingNodesByIDsIsKindGuardedAndSetOriented(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "graph.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	store.AddBatch([]*graph.Node{
		{ID: "reader", Kind: graph.KindFunction},
		{ID: "f1", Kind: graph.KindFinding},
		{ID: "keep", Kind: graph.KindResource},
	}, []*graph.Edge{
		{From: "f1", To: "reader", Kind: graph.EdgeMemberOf},
		{From: "keep", To: "reader", Kind: graph.EdgeCalls},
	})

	nodes, edges := store.EvictFindingNodesByIDs([]string{"f1", "f1", "keep"})
	if nodes != 1 || edges != 1 {
		t.Fatalf("EvictFindingNodesByIDs = (%d,%d), want (1,1)", nodes, edges)
	}
	if store.GetNode("f1") != nil || store.GetNode("keep") == nil || store.GetNode("reader") == nil {
		t.Fatalf("kind guard failed: f1=%#v keep=%#v reader=%#v",
			store.GetNode("f1"), store.GetNode("keep"), store.GetNode("reader"))
	}
}
