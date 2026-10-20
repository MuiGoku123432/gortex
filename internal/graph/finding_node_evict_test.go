package graph

import "testing"

func TestEvictFindingNodesByIDs_InMemoryKindGuarded(t *testing.T) {
	const (
		repo    = "estate"
		file    = "estate/pkg/app.cbl"
		finding = "estate/pkg/app.cbl::@finding/parse/0123456789abcdef0123456789abcdef"
		res     = "estate/pkg/app.cbl::resource"
		program = "estate/pkg/app.cbl::APP"
	)
	g := New()
	g.AddNode(&Node{ID: finding, Kind: KindFinding, Name: "unparsed-tail", FilePath: file, RepoPrefix: repo})
	g.AddNode(&Node{ID: res, Kind: KindResource, Name: "resource", FilePath: file, RepoPrefix: repo})
	g.AddNode(&Node{ID: program, Kind: KindFunction, Name: "APP", FilePath: file, RepoPrefix: repo})
	g.AddEdge(&Edge{From: finding, To: program, Kind: EdgeMemberOf, FilePath: file})
	g.AddEdge(&Edge{From: finding, To: res, Kind: EdgeReferences, FilePath: file})

	nodes, edges, supported := EvictFindingNodesByIDs(g, []string{finding, finding, res, program, ""})
	if nodes != 1 || edges != 2 || !supported {
		t.Fatalf("EvictFindingNodesByIDs = (%d, %d, %v), want (1, 2, true)", nodes, edges, supported)
	}
	if g.GetNode(finding) != nil {
		t.Fatalf("finding %q survived eviction", finding)
	}
	if g.GetNode(res) == nil || g.GetNode(program) == nil {
		t.Fatalf("kind guard failed: resource=%v program=%v", g.GetNode(res), g.GetNode(program))
	}
	for n := range g.NodesByKind(KindFinding) {
		t.Fatalf("NodesByKind(KindFinding) still yields %q", n.ID)
	}
	for _, n := range g.GetFileNodes(file) {
		if n.ID == finding {
			t.Fatalf("file bucket still holds %q", finding)
		}
	}
	for _, n := range g.GetRepoNodes(repo) {
		if n.ID == finding {
			t.Fatalf("repo bucket still holds %q", finding)
		}
	}
	if got := len(g.GetOutEdges(finding)) + len(g.GetInEdges(program)) + len(g.GetInEdges(res)); got != 0 {
		t.Fatalf("incident edges survived: %d", got)
	}
}

// findingEvictlessStore embeds a Store without promoting the finding
// eviction capability.
type findingEvictlessStore struct{ Store }

func TestEvictFindingNodesByIDs_Unsupported(t *testing.T) {
	store := findingEvictlessStore{Store: New()}
	if nodes, edges, supported := EvictFindingNodesByIDs(store, []string{"a::@finding/parse/x"}); nodes != 0 || edges != 0 || supported {
		t.Fatalf("store without capability = (%d, %d, %v), want (0, 0, false)", nodes, edges, supported)
	}
	if nodes, edges, supported := EvictFindingNodesByIDs(nil, []string{"a::@finding/parse/x"}); nodes != 0 || edges != 0 || !supported {
		t.Fatalf("nil store = (%d, %d, %v), want (0, 0, true)", nodes, edges, supported)
	}
	if nodes, edges, supported := EvictFindingNodesByIDs(store, nil); nodes != 0 || edges != 0 || !supported {
		t.Fatalf("empty ids = (%d, %d, %v), want (0, 0, true)", nodes, edges, supported)
	}
}
