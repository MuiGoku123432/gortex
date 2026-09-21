package neo4jprojection

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	neo4j "github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
	"github.com/zzet/gortex/internal/testutil/graphfixture"
)

func TestNeo4jProductionTracer(t *testing.T) {
	if os.Getenv("GORTEX_NEO4J_INTEGRATION") != "1" {
		t.Fatal("production tracer requires the disposable Neo4j gate")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	profile := config.ResolvedNeo4jProfile{
		Name: "disposable-tracer", URI: requiredIntegrationEnv(t, "GORTEX_NEO4J_URI"), Database: "neo4j",
		Username: requiredIntegrationEnv(t, "GORTEX_NEO4J_USERNAME"), Password: requiredIntegrationEnv(t, "GORTEX_NEO4J_PASSWORD"),
	}
	assertionDriver, err := neo4j.NewDriver(profile.URI, neo4j.BasicAuth(profile.Username, profile.Password, ""))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := assertionDriver.Close(context.Background()); err != nil {
			t.Error(err)
		}
	}()

	const (
		operation        = "production-tracer-operation"
		failureOwner     = "workspace:failure/repo"
		failurePrior     = "generation-prior"
		failureOperation = "production-tracer-failure"
	)
	if _, err := neo4j.ExecuteQuery(ctx, assertionDriver, `
CREATE (canary:GortexNeighborCanary {canary: 'survive'})
CREATE (neighbor:GortexNode {gortex_physical_key: 'neighbor-physical', gortex_owner: $neighbor, gortex_generation: 'neighbor-active', gortex_logical_key: 'neighbor-logical'})
CREATE (failure:GortexProjectionManifest {gortex_owner: $failure_owner, active_generation: $failure_prior, pending_generation: '', operation_id: '', complete: true, cleanup_complete: false})`, map[string]any{
		"neighbor": graphfixture.NeighborOwner, "failure_owner": failureOwner, "failure_prior": failurePrior,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(profile.Database)); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(t.TempDir(), "production-tracer.sqlite")
	store, err := store_sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.AddBatch([]*graph.Node{
		{ID: graphfixture.ScopedNodeID, RepoPrefix: "fixture/repo", FilePath: "fixture/repo/main.go", Kind: graph.KindFunction, Name: "Run", Meta: map[string]any{"secret_canary": "fixture-secret"}},
		{ID: graphfixture.ScopedTargetID, RepoPrefix: "fixture/repo", FilePath: "fixture/repo/main.go", Kind: graph.KindType, Name: "Store"},
		{ID: "neighbor/repo/main.go::Run", RepoPrefix: "neighbor/repo", FilePath: "neighbor/repo/main.go", Kind: graph.KindFunction, Name: "Neighbor"},
	}, []*graph.Edge{{From: graphfixture.ScopedNodeID, To: graphfixture.ScopedTargetID, Kind: graph.EdgeReferences, FilePath: "fixture/repo/main.go", Line: 12}})
	if err := store.CheckpointWAL(); err != nil {
		t.Fatal(err)
	}
	scope := graph.ProjectionScope{Repositories: []string{"fixture/repo"}}
	warm, err := store.OpenScopedProjectionSnapshot(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := warm.ReadNodes(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := warm.ReadEdges(ctx); err != nil {
		t.Fatal(err)
	}
	if err := warm.Close(); err != nil {
		t.Fatal(err)
	}
	beforeFiles := tracerSQLiteFiles(t, dbPath)
	beforeCanonical := tracerSQLiteCanonical(t, ctx, store, scope)

	factory := func(context.Context) (Transport, error) { return NewNeo4jTransport(profile) }
	service := NewService(store, factory)
	request := Request{Owner: graphfixture.OwnerKey, OperationID: operation, Scope: scope}
	first, err := service.Push(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Push(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Complete || first.ActiveGeneration == "" || first.ActiveGeneration != second.ActiveGeneration || first.NodeCount != 2 || first.EdgeCount != 1 {
		t.Fatalf("non-idempotent tracer results: first=%#v second=%#v", first, second)
	}

	failureService := NewService(store, func(ctx context.Context) (Transport, error) {
		transport, err := NewNeo4jTransport(profile)
		if err != nil {
			return nil, err
		}
		return &failBeforeActivationTransport{Transport: transport}, nil
	})
	failed, failureErr := failureService.Push(ctx, Request{Owner: failureOwner, OperationID: failureOperation, Scope: scope})
	if failureErr == nil || failed.Complete || failed.ActiveGeneration != failurePrior {
		t.Fatalf("pre-activation failure did not preserve prior visibility: result=%#v err=%v", failed, failureErr)
	}

	target, err := neo4j.ExecuteQuery(ctx, assertionDriver, `
MATCH (manifest:GortexProjectionManifest {gortex_owner: $owner})
OPTIONAL MATCH (node:GortexNode {gortex_owner: $owner, gortex_generation: manifest.active_generation})
WITH manifest, count(node) AS node_count
OPTIONAL MATCH ()-[relationship:GORTEX_RELATIONSHIP {gortex_owner: $owner, gortex_generation: manifest.active_generation}]->()
RETURN manifest.active_generation AS active_generation, manifest.complete AS complete,
       node_count, count(relationship) AS relationship_count,
       count { MATCH (:GortexNeighborCanary {canary: 'survive'}) } = 1 AS canary_survives,
       count { MATCH (:GortexNode {gortex_physical_key: 'neighbor-physical', gortex_owner: $neighbor}) } = 1 AS neighbor_survives,
       count { MATCH (:GortexProjectionManifest {gortex_owner: $failure_owner, active_generation: $failure_prior}) } = 1 AS failure_prior_survives,
       count { MATCH (leaked) WHERE any(key IN keys(leaked) WHERE toString(leaked[key]) CONTAINS $password OR toString(leaked[key]) CONTAINS $source_secret) RETURN leaked } AS leaked_nodes`, map[string]any{
		"owner": graphfixture.OwnerKey, "neighbor": graphfixture.NeighborOwner,
		"failure_owner": failureOwner, "failure_prior": failurePrior, "password": profile.Password, "source_secret": "fixture-secret",
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(profile.Database))
	if err != nil {
		t.Fatal(err)
	}
	if len(target.Records) != 1 {
		t.Fatalf("target assertion returned %d records", len(target.Records))
	}
	record := target.Records[0]
	assertNeo4jValue(t, record, "active_generation", first.ActiveGeneration)
	assertNeo4jValue(t, record, "complete", true)
	assertNeo4jValue(t, record, "node_count", int64(2))
	assertNeo4jValue(t, record, "relationship_count", int64(1))
	assertNeo4jValue(t, record, "canary_survives", true)
	assertNeo4jValue(t, record, "neighbor_survives", true)
	assertNeo4jValue(t, record, "failure_prior_survives", true)
	assertNeo4jValue(t, record, "leaked_nodes", int64(0))
	if strings.Contains(fmt.Sprint(first, second, failed, failureErr), profile.Password) {
		t.Fatal("Neo4j password leaked through result or error")
	}

	afterCanonical := tracerSQLiteCanonical(t, ctx, store, scope)
	afterFiles := tracerSQLiteFiles(t, dbPath)
	if beforeCanonical != afterCanonical || !reflect.DeepEqual(beforeFiles, afterFiles) {
		t.Fatalf("SQLite source changed: canonical_equal=%v files_equal=%v", beforeCanonical == afterCanonical, reflect.DeepEqual(beforeFiles, afterFiles))
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

type failBeforeActivationTransport struct{ Transport }

func (t *failBeforeActivationTransport) Activate(context.Context, string, string, string, string, Result) error {
	return fmt.Errorf("injected failure before activation")
}

func requiredIntegrationEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("integration environment missing %s", name)
	}
	return value
}

func assertNeo4jValue(t *testing.T, record *neo4j.Record, key string, want any) {
	t.Helper()
	got, ok := record.Get(key)
	if !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}

func tracerSQLiteFiles(t *testing.T, path string) map[string][32]byte {
	t.Helper()
	result := make(map[string][32]byte)
	for _, suffix := range []string{"", "-wal", "-shm"} {
		data, err := os.ReadFile(path + suffix)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		result[suffix] = sha256.Sum256(data)
	}
	return result
}

func tracerSQLiteCanonical(t *testing.T, ctx context.Context, store *store_sqlite.Store, scope graph.ProjectionScope) string {
	t.Helper()
	snapshot, err := store.OpenScopedProjectionSnapshot(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := snapshot.Close(); err != nil {
			t.Error(err)
		}
	}()
	nodes, err := snapshot.ReadNodes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	edges, err := snapshot.ReadEdges(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]string, 0, len(nodes)+len(edges))
	for _, node := range nodes {
		rows = append(rows, fmt.Sprintf("node\x1f%s\x1f%s\x1f%s\x1f%s", node.ID, node.Kind, node.Name, node.FilePath))
	}
	for _, row := range edges {
		rows = append(rows, fmt.Sprintf("edge\x1f%s\x1f%s\x1f%s\x1f%s\x1f%d", row.Edge.From, row.Edge.To, row.Edge.Kind, row.Edge.FilePath, row.Edge.Line))
	}
	sort.Strings(rows)
	return strings.Join(rows, "\n")
}
