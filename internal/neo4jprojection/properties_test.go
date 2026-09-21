package neo4jprojection

import (
	"testing"

	"github.com/zzet/gortex/internal/graph"
)

func TestProjectionOwnerIdentity(t *testing.T) {
	left := projectionOwnerKey("team", "workspace", "project", []string{"repo-b", "repo-a"})
	right := projectionOwnerKey("team", "workspace", "project", []string{"repo-a", "repo-b"})
	if left != right {
		t.Fatalf("repository ordering changed owner key: %q != %q", left, right)
	}
	for name, got := range map[string]string{
		"namespace":  projectionOwnerKey("other", "workspace", "project", []string{"repo-a", "repo-b"}),
		"workspace":  projectionOwnerKey("team", "other", "project", []string{"repo-a", "repo-b"}),
		"project":    projectionOwnerKey("team", "workspace", "other", []string{"repo-a", "repo-b"}),
		"repository": projectionOwnerKey("team", "workspace", "project", []string{"repo-a"}),
	} {
		if got == left {
			t.Fatalf("%s component did not affect owner key", name)
		}
	}
}

func TestProjectionIdentity(t *testing.T) {
	owner := projectionOwnerKey("team", "workspace", "project", []string{"repo"})
	nodeLogical := projectionNodeLogicalKey(owner, "repo/file.go::Thing")
	if nodeLogical == projectionNodeLogicalKey(owner+"x", "repo/file.go::Thing") || nodeLogical == projectionNodeLogicalKey(owner, "repo/file.go::Other") {
		t.Fatal("node logical key does not bind owner and authoritative node ID")
	}
	if projectionPhysicalKey(nodeLogical, "generation-a") == projectionPhysicalKey(nodeLogical, "generation-b") {
		t.Fatal("physical key does not bind generation")
	}

	base := &graph.Edge{From: "a", To: "b", Kind: graph.EdgeCalls, FilePath: "a.go", Line: 7, Origin: "ast", Meta: map[string]any{"occurrence": "first"}}
	second := *base
	second.Meta = map[string]any{"occurrence": "second"}
	if projectionEdgeLogicalKey(owner, base) == projectionEdgeLogicalKey(owner, &second) {
		t.Fatal("edge occurrence discriminator collapsed distinct evidence")
	}
	changed := *base
	changed.Origin = "lsp"
	if projectionEdgeLogicalKey(owner, base) == projectionEdgeLogicalKey(owner, &changed) {
		t.Fatal("edge provenance did not affect identity")
	}
}

func TestProjectionTokenSanitization(t *testing.T) {
	labels := []string{
		sanitizeLabel("function"),
		sanitizeLabel("a-b"),
		sanitizeLabel("a b"),
		sanitizeLabel("9 hostile`) MATCH (n) DETACH DELETE n //"),
	}
	seen := map[string]bool{}
	for _, token := range labels {
		if !closedNeo4jToken.MatchString(token) {
			t.Fatalf("label is outside closed grammar: %q", token)
		}
		if seen[token] {
			t.Fatalf("distinct labels collided: %q", token)
		}
		seen[token] = true
	}
	for _, kind := range []string{"calls", "a-b", "a b", "9 hostile"} {
		if token := sanitizeRelationshipType(graph.EdgeKind(kind)); !closedNeo4jToken.MatchString(token) {
			t.Fatalf("relationship type is outside closed grammar: %q", token)
		}
	}
	if sanitizeRelationshipType("a-b") == sanitizeRelationshipType("a b") {
		t.Fatal("distinct relationship kinds collided")
	}
}
