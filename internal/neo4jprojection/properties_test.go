package neo4jprojection

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

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

func TestProjectionProperties(t *testing.T) {
	node := completeProjectionNode()
	projected, warnings, err := projectNode("owner", "generation", node)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "kind", "name", "qual_name", "file_path", "start_line", "end_line", "start_column", "end_column", "language", "repo_prefix", "workspace_id", "project_id", "origin", "stub", "fetched_at"} {
		if _, ok := projected.Properties[key]; !ok {
			t.Errorf("missing typed node property %q", key)
		}
	}
	if warnings.Unsupported != 2 || warnings.Secret != 1 {
		t.Fatalf("warnings = %+v, want unsupported=2 secret=1", warnings)
	}
	if projected.Properties["gortex_meta_key_map"] == "" || projected.Properties["gortex_meta_encoding"] == "" {
		t.Fatal("metadata key map or encoding markers were not recorded")
	}
}

func TestProjectionPropertiesGolden(t *testing.T) {
	node, nodeWarnings, err := projectNode("owner", "generation", completeProjectionNode())
	if err != nil {
		t.Fatal(err)
	}
	edge, edgeWarnings, err := projectEdge("owner", "generation", &graph.Edge{
		From: "repo/a.go::A", To: "unresolved::pkg::B", Kind: graph.EdgeCalls,
		FilePath: "repo/a.go", Line: 12, Confidence: .75, ConfidenceLabel: "high",
		Origin: "ast", Tier: "structural", CrossRepo: true, Context: "call",
		ReturnUsage: "assigned", Via: "framework", Alias: "renamed", NameOnly: true,
		Meta: map[string]any{"mixed": []any{"x", 2}, "safe": "value"},
	})
	if err != nil {
		t.Fatal(err)
	}
	actual, err := canonicalJSON(map[string]any{"edge": edge, "edge_warnings": edgeWarnings, "node": node, "node_warnings": nodeWarnings})
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/properties.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(append(actual, '\n')) != string(want) {
		t.Fatalf("golden mismatch\nactual: %s\nwant: %s", actual, want)
	}
}

func TestProjectionUnresolved(t *testing.T) {
	projected, _, err := projectNode("owner", "generation", &graph.Node{ID: "unresolved::pkg::Thing", Kind: "unresolved", Name: "Thing"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(projected.Labels, "GortexUnresolved") {
		t.Fatalf("unresolved labels = %v", projected.Labels)
	}
}

func TestProjectionMetadataBudgets(t *testing.T) {
	t.Run("oversized string", func(t *testing.T) {
		_, _, ok := projectMetadataValue(strings.Repeat("x", maxMetadataBytes+1))
		if ok {
			t.Fatal("oversized string was accepted")
		}
	})
	t.Run("oversized native list", func(t *testing.T) {
		_, _, ok := projectMetadataValue(make([]int64, maxMetadataElements+1))
		if ok {
			t.Fatal("oversized homogeneous list was accepted")
		}
	})
	t.Run("nested map and struct", func(t *testing.T) {
		type nested struct {
			Value any `json:"value"`
		}
		var value any = "leaf"
		for range maxMetadataDepth + 1 {
			value = map[string]any{"next": nested{Value: value}}
		}
		_, _, ok := projectMetadataValue(value)
		if ok {
			t.Fatal("over-depth map/struct graph was accepted")
		}
	})
	t.Run("cumulative across keys", func(t *testing.T) {
		properties := map[string]any{}
		warnings := appendMetadata(properties, map[string]any{
			"first":  strings.Repeat("a", maxMetadataBytes/2+1),
			"second": strings.Repeat("b", maxMetadataBytes/2+1),
		})
		if warnings.Unsupported != 1 {
			t.Fatalf("unsupported warnings = %d, want 1", warnings.Unsupported)
		}
		if _, first := properties["meta_first"]; !first {
			t.Fatal("first bounded value was not retained")
		}
		if _, second := properties["meta_second"]; second {
			t.Fatal("cumulative budget overflow was retained")
		}
	})
	t.Run("single hostile key exact envelope", func(t *testing.T) {
		properties := map[string]any{}
		hostile := strings.Repeat("?", 220_000)
		warnings := appendMetadata(properties, map[string]any{hostile: "x"})
		if warnings.Unsupported != 1 {
			t.Fatalf("unsupported warnings = %d, want 1", warnings.Unsupported)
		}
		encoded, err := canonicalJSON(properties)
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded) > maxMetadataBytes {
			t.Fatalf("hostile envelope = %d bytes, max %d", len(encoded), maxMetadataBytes)
		}
	})
	t.Run("many long keys and generated maps", func(t *testing.T) {
		metadata := make(map[string]any, maxMetadataElements)
		for i := range maxMetadataElements {
			metadata[fmt.Sprintf("%08d-%s", i, strings.Repeat("k", 128))] = "x"
		}
		properties := map[string]any{}
		warnings := appendMetadata(properties, metadata)
		if warnings.Unsupported == 0 {
			t.Fatal("metadata envelope exceeding key/generated-map budget was accepted")
		}
		encoded, err := canonicalJSON(properties)
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded) > maxMetadataBytes {
			t.Fatalf("projected metadata envelope = %d bytes, max %d", len(encoded), maxMetadataBytes)
		}
	})
	t.Run("escaped collision map budget", func(t *testing.T) {
		metadata := make(map[string]any)
		for i := range maxMetadataElements {
			metadata[fmt.Sprintf("collision/%08d/%s", i, strings.Repeat("?", 32))] = []string{"value"}
		}
		properties := map[string]any{}
		warnings := appendMetadata(properties, metadata)
		if warnings.Unsupported == 0 {
			t.Fatal("escaped key-map and encoding-marker expansion was accepted without rejection")
		}
		encoded, err := canonicalJSON(properties)
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded) > maxMetadataBytes {
			t.Fatalf("generated metadata properties = %d bytes, max %d", len(encoded), maxMetadataBytes)
		}
	})
}

func TestProjectionSecretFiltering(t *testing.T) {
	projected, warnings, err := projectNode("owner", "generation", &graph.Node{ID: "n", Kind: graph.KindFunction, Meta: map[string]any{"API-Key": "secret-canary", "credential_value": "secret-canary", "safe": "public"}})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := canonicalJSON(projected)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("secret-canary")) || warnings.Secret != 2 {
		t.Fatalf("secret filtering failed: warnings=%+v output=%s", warnings, encoded)
	}
}

func completeProjectionNode() *graph.Node {
	return &graph.Node{
		ID: "repo/a.go::A", Kind: graph.KindFunction, Name: "A", QualName: "pkg.A",
		FilePath: "repo/a.go", StartLine: 10, EndLine: 20, StartColumn: 2, EndColumn: 8,
		Language: "go", RepoPrefix: "repo", WorkspaceID: "workspace", ProjectID: "project",
		Origin: "remote:repo", Stub: true, FetchedAt: time.Date(2026, 9, 21, 12, 0, 0, 123, time.UTC),
		Meta: map[string]any{
			"safe": "value", "a-b": int64(1), "a b": int64(2),
			"nested": map[string]any{"b": 2, "a": 1}, "mixed": []any{"x", 2},
			"labels": []string{"one", "two"}, "api_token": "secret-canary",
			"nan": math.NaN(), "unsupported": make(chan int),
		},
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
