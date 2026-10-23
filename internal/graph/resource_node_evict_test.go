package graph

import "testing"

func TestEvictResourceNodesByIDs_InMemoryKindGuarded(t *testing.T) {
	const (
		repo     = "r1"
		file     = "r1/src/app.cbl"
		resource = "r1::cobol-missing::program/NOPGM"
		finding  = "r1/src/app.cbl::@finding/external/0123456789abcdef0123456789abcdef"
		function = "r1/src/app.cbl::APP"
	)
	g := New()
	g.AddNode(&Node{ID: resource, Kind: KindResource, Name: "NOPGM", RepoPrefix: repo})
	g.AddNode(&Node{ID: finding, Kind: KindFinding, Name: "external-unresolved", FilePath: file, RepoPrefix: repo})
	g.AddNode(&Node{ID: function, Kind: KindFunction, Name: "APP", FilePath: file, RepoPrefix: repo})
	g.AddNode(&Node{ID: file, Kind: KindFile, Name: "app.cbl", FilePath: file, RepoPrefix: repo})
	g.AddEdge(&Edge{From: finding, To: resource, Kind: EdgeReferences, FilePath: file})
	g.AddEdge(&Edge{From: function, To: resource, Kind: EdgeCalls, FilePath: file})
	g.AddEdge(&Edge{From: finding, To: function, Kind: EdgeMemberOf, FilePath: file})
	g.AddEdge(&Edge{From: file, To: function, Kind: EdgeDefines, FilePath: file})

	nodes, edges, supported := EvictResourceNodesByIDs(g, []string{resource, resource, finding, function, file, ""})
	if nodes != 1 || edges != 2 || !supported {
		t.Fatalf("EvictResourceNodesByIDs = (%d, %d, %v), want (1, 2, true)", nodes, edges, supported)
	}
	if g.GetNode(resource) != nil {
		t.Fatalf("resource %q survived eviction", resource)
	}
	for _, id := range []string{finding, function, file} {
		if g.GetNode(id) == nil {
			t.Fatalf("kind guard failed: %q was evicted", id)
		}
	}
	for n := range g.NodesByKind(KindResource) {
		t.Fatalf("NodesByKind(KindResource) still yields %q", n.ID)
	}
	for _, n := range g.GetRepoNodes(repo) {
		if n.ID == resource {
			t.Fatalf("repo bucket still holds %q", resource)
		}
	}
	if got := len(g.GetInEdges(resource)) + len(g.GetOutEdges(resource)); got != 0 {
		t.Fatalf("incident edges survived: %d", got)
	}
	if got := len(g.GetInEdges(function)); got != 2 {
		t.Fatalf("edges not touching the resource: in-edges of %q = %d, want 2", function, got)
	}
}

// resourceEvictlessStore embeds a Store without promoting the resource
// eviction capability.
type resourceEvictlessStore struct{ Store }

func TestEvictResourceNodesByIDs_Unsupported(t *testing.T) {
	store := resourceEvictlessStore{Store: New()}
	if nodes, edges, supported := EvictResourceNodesByIDs(store, []string{"r1::cobol-missing::program/X"}); nodes != 0 || edges != 0 || supported {
		t.Fatalf("store without capability = (%d, %d, %v), want (0, 0, false)", nodes, edges, supported)
	}
	if nodes, edges, supported := EvictResourceNodesByIDs(nil, []string{"r1::cobol-missing::program/X"}); nodes != 0 || edges != 0 || !supported {
		t.Fatalf("nil store = (%d, %d, %v), want (0, 0, true)", nodes, edges, supported)
	}
	if nodes, edges, supported := EvictResourceNodesByIDs(store, nil); nodes != 0 || edges != 0 || !supported {
		t.Fatalf("empty ids = (%d, %d, %v), want (0, 0, true)", nodes, edges, supported)
	}
}
