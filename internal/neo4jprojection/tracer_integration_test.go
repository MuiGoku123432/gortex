package neo4jprojection

import (
	"context"
	"crypto/sha256"
	"errors"
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
		t.Skip("production tracer runs only in the disposable Neo4j gate")
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
		operation          = "production-tracer-operation"
		failureOwner       = "workspace:failure/repo"
		failurePrior       = "generation-prior"
		failureOperation   = "production-tracer-failure"
		unresolvedTargetID = "fixture/repo/missing.go::Missing"
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
		{ID: graphfixture.ScopedNodeID, RepoPrefix: "fixture/repo", WorkspaceID: "fixture", ProjectID: "fixture", FilePath: "fixture/repo/main.go", Kind: graph.KindFunction, Name: "Run", Meta: map[string]any{"secret_canary": "fixture-secret"}},
		{ID: graphfixture.ScopedTargetID, RepoPrefix: "fixture/repo", WorkspaceID: "fixture", ProjectID: "fixture", FilePath: "fixture/repo/main.go", Kind: graph.KindType, Name: "Store"},
		{ID: "neighbor/repo/main.go::Run", RepoPrefix: "neighbor/repo", FilePath: "neighbor/repo/main.go", Kind: graph.KindFunction, Name: "Neighbor"},
	}, []*graph.Edge{
		{From: graphfixture.ScopedNodeID, To: graphfixture.ScopedTargetID, Kind: graph.EdgeReferences, FilePath: "fixture/repo/main.go", Line: 12},
		{From: graphfixture.ScopedNodeID, To: unresolvedTargetID, Kind: graph.EdgeCalls, FilePath: "fixture/repo/main.go", Line: 13},
	})
	if err := store.CheckpointWAL(); err != nil {
		t.Fatal(err)
	}
	scope := graph.ProjectionScope{Workspace: "fixture", Project: "fixture", Repositories: []string{"fixture/repo"}}
	warm, err := store.OpenScopedProjectionSnapshot(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if err := warm.ReadNodePages(ctx, func([]*graph.Node) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := warm.ReadEdgePages(ctx, func([]graph.ScopedEdgeRow) error { return nil }); err != nil {
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
	beforeDryRun := neo4jOwnerCensus(t, ctx, assertionDriver, profile.Database, graphfixture.OwnerKey)
	dryRun, err := service.Push(ctx, Request{Owner: graphfixture.OwnerKey, OperationID: "dry-run", Scope: scope, DryRun: true})
	if err != nil || !dryRun.Complete || !dryRun.DryRun || dryRun.NodeCount != 2 || dryRun.EdgeCount != 2 {
		t.Fatalf("dry run failed: result=%#v err=%v", dryRun, err)
	}
	if afterDryRun := neo4jOwnerCensus(t, ctx, assertionDriver, profile.Database, graphfixture.OwnerKey); afterDryRun != beforeDryRun {
		t.Fatalf("dry run mutated target: before=%v after=%v", beforeDryRun, afterDryRun)
	}

	first, err := service.Push(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Push(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Complete || !first.CleanupComplete || first.ActiveGeneration == "" || first.ActiveGeneration != second.ActiveGeneration || first.NodeCount != 3 || first.EdgeCount != 2 {
		t.Fatalf("non-idempotent tracer results: first=%#v second=%#v", first, second)
	}

	cancelService := NewService(store, func(ctx context.Context) (Transport, error) {
		transport, err := NewNeo4jTransport(profile)
		if err != nil {
			return nil, err
		}
		return &cancelBeforeActivationTransport{Transport: transport}, nil
	})
	cancelled, cancelErr := cancelService.Push(ctx, Request{Owner: "workspace:cancel/repo", OperationID: "cancel-operation", Scope: scope})
	if !errors.Is(cancelErr, context.Canceled) || cancelled.Complete || cancelled.ActiveGeneration != "" {
		t.Fatalf("cancellation did not preserve inactive state: result=%#v err=%v", cancelled, cancelErr)
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
	changedStore := store.AtGeneration(1)
	crashOwner := "workspace:crash/repo"
	crashGeneration := generationKey(crashOwner, operation, 0)
	liveOwner := "workspace:live/repo"
	if _, err := neo4j.ExecuteQuery(ctx, assertionDriver, `
MERGE (m:GortexProjectionManifest {gortex_owner: $owner})
SET m.active_generation = $prior, m.pending_generation = $generation,
    m.operation_id = $operation, m.attempt_id = 'dead-process',
    m.operation_state = 'active', m.pending_complete = false,
    m.lease_until = datetime() - duration('PT1S')`, map[string]any{
		"owner": crashOwner, "prior": failurePrior, "generation": crashGeneration, "operation": operation,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(profile.Database)); err != nil {
		t.Fatal(err)
	}
	crashRecovered, err := NewService(changedStore, factory).Push(ctx, Request{Owner: crashOwner, OperationID: operation, Scope: scope})
	if err != nil || !crashRecovered.Complete || crashRecovered.SourceGeneration != 1 || crashRecovered.ActiveGeneration == failurePrior {
		t.Fatalf("ordinary rerun did not recover a crash-stale lease: result=%#v err=%v", crashRecovered, err)
	}
	if _, err := neo4j.ExecuteQuery(ctx, assertionDriver, `
MERGE (m:GortexProjectionManifest {gortex_owner: $owner})
SET m.active_generation = $prior, m.pending_generation = 'live-generation',
    m.operation_id = 'live-operation', m.attempt_id = 'live-attempt',
    m.operation_state = 'active', m.pending_complete = false,
    m.lease_until = datetime() + duration('PT5M')`, map[string]any{
		"owner": liveOwner, "prior": failurePrior,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(profile.Database)); err != nil {
		t.Fatal(err)
	}
	if liveResult, liveErr := NewService(changedStore, factory).Push(ctx, Request{Owner: liveOwner, OperationID: operation, Scope: scope}); liveErr == nil || liveResult.Complete {
		t.Fatalf("live concurrent lease was taken over: result=%#v err=%v", liveResult, liveErr)
	}
	changedSource, err := NewService(changedStore, factory).Push(ctx, Request{Owner: failureOwner, OperationID: failureOperation, Scope: scope})
	if err != nil || !changedSource.Complete || changedSource.SourceGeneration != 1 || changedSource.ActiveGeneration == failurePrior || changedSource.ActiveGeneration == failed.PendingGeneration {
		t.Fatalf("ordinary rerun did not recover after changed source generation: failed=%#v changed=%#v err=%v", failed, changedSource, err)
	}

	cleanupService := NewService(store, func(ctx context.Context) (Transport, error) {
		transport, err := NewNeo4jTransport(profile)
		if err != nil {
			return nil, err
		}
		return &failCleanupTransport{Transport: transport}, nil
	})
	cleanupResult, cleanupErr := cleanupService.Push(ctx, Request{Owner: graphfixture.OwnerKey, OperationID: "changed-operation", Scope: scope, BatchSize: 1})
	if !errors.Is(cleanupErr, ErrCleanupIncomplete) || !cleanupResult.Complete || cleanupResult.CleanupComplete || cleanupResult.StaleNodeCount == 0 {
		t.Fatalf("cleanup interruption was not reported truthfully: result=%#v err=%v", cleanupResult, cleanupErr)
	}
	resumed, err := service.Push(ctx, Request{Owner: graphfixture.OwnerKey, OperationID: "changed-operation", Scope: scope, BatchSize: 1})
	if err != nil || !resumed.Complete || !resumed.CleanupComplete || resumed.ActiveGeneration != cleanupResult.ActiveGeneration {
		t.Fatalf("ordinary rerun did not resume cleanup: result=%#v err=%v", resumed, err)
	}

	target, err := neo4j.ExecuteQuery(ctx, assertionDriver, `
MATCH (manifest:GortexProjectionManifest {gortex_owner: $owner})
OPTIONAL MATCH (node:GortexNode {gortex_owner: $owner, gortex_generation: manifest.active_generation})
WITH manifest, count(node) AS node_count
OPTIONAL MATCH ()-[relationship {gortex_owner: $owner, gortex_generation: manifest.active_generation}]->()
RETURN manifest.active_generation AS active_generation, manifest.complete AS complete,
       manifest.node_count AS manifest_node_count, manifest.edge_count AS manifest_edge_count,
       node_count, count(relationship) AS relationship_count,
       count { MATCH (:GortexNode:GortexUnresolved {gortex_owner: $owner, gortex_generation: manifest.active_generation, id: $unresolved_target}) } AS unresolved_count,
       count { MATCH (:GortexNode {gortex_owner: $owner, gortex_generation: manifest.active_generation})-[r {gortex_owner: $owner, gortex_generation: manifest.active_generation}]->(:GortexNode:GortexUnresolved {gortex_owner: $owner, gortex_generation: manifest.active_generation, id: $unresolved_target}) RETURN r } AS unresolved_edge_count,
       count { MATCH (:GortexNeighborCanary {canary: 'survive'}) } = 1 AS canary_survives,
       count { MATCH (:GortexNode {gortex_physical_key: 'neighbor-physical', gortex_owner: $neighbor}) } = 1 AS neighbor_survives,
       count { MATCH (:GortexProjectionManifest {gortex_owner: $failure_owner, active_generation: $failure_recovered}) } = 1 AS failure_recovered,
       count { MATCH (leaked) WHERE any(key IN keys(leaked) WHERE toString(leaked[key]) CONTAINS $password OR toString(leaked[key]) CONTAINS $source_secret) RETURN leaked } AS leaked_nodes`, map[string]any{
		"owner": graphfixture.OwnerKey, "neighbor": graphfixture.NeighborOwner, "unresolved_target": unresolvedTargetID,
		"failure_owner": failureOwner, "failure_recovered": changedSource.ActiveGeneration, "password": profile.Password, "source_secret": "fixture-secret",
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(profile.Database))
	if err != nil {
		t.Fatal(err)
	}
	if len(target.Records) != 1 {
		t.Fatalf("target assertion returned %d records", len(target.Records))
	}
	record := target.Records[0]
	assertNeo4jValue(t, record, "active_generation", resumed.ActiveGeneration)
	assertNeo4jValue(t, record, "complete", true)
	assertNeo4jValue(t, record, "node_count", int64(3))
	assertNeo4jValue(t, record, "relationship_count", int64(2))
	assertNeo4jValue(t, record, "manifest_node_count", int64(3))
	assertNeo4jValue(t, record, "manifest_edge_count", int64(2))
	assertNeo4jValue(t, record, "unresolved_count", int64(1))
	assertNeo4jValue(t, record, "unresolved_edge_count", int64(1))
	assertNeo4jValue(t, record, "canary_survives", true)
	assertNeo4jValue(t, record, "neighbor_survives", true)
	assertNeo4jValue(t, record, "failure_recovered", true)
	assertNeo4jValue(t, record, "leaked_nodes", int64(0))
	if strings.Contains(fmt.Sprint(dryRun, first, second, cancelled, cancelErr, failed, failureErr, cleanupResult, cleanupErr, resumed), profile.Password) {
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

type cancelBeforeActivationTransport struct{ Transport }

func (t *cancelBeforeActivationTransport) Activate(context.Context, string, string, string, string, string, Result) error {
	return context.Canceled
}

func (t *cancelBeforeActivationTransport) Reconcile(context.Context, string, string, int) (CleanupCounts, error) {
	return CleanupCounts{}, nil
}

type failCleanupTransport struct{ Transport }

func (t *failCleanupTransport) Reconcile(context.Context, string, string, int) (CleanupCounts, error) {
	return CleanupCounts{Nodes: 2, Relationships: 1}, ErrCleanupIncomplete
}

type failBeforeActivationTransport struct{ Transport }

func (t *failBeforeActivationTransport) Activate(context.Context, string, string, string, string, string, Result) error {
	return fmt.Errorf("injected failure before activation")
}

func (t *failBeforeActivationTransport) Reconcile(context.Context, string, string, int) (CleanupCounts, error) {
	return CleanupCounts{}, nil
}

func requiredIntegrationEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("integration environment missing %s", name)
	}
	return value
}

func neo4jOwnerCensus(t *testing.T, ctx context.Context, driver neo4j.Driver, database, owner string) [3]int64 {
	t.Helper()
	result, err := neo4j.ExecuteQuery(ctx, driver, `
OPTIONAL MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WITH count(m) AS manifests
OPTIONAL MATCH (n:GortexNode {gortex_owner: $owner})
WITH manifests, count(n) AS nodes
OPTIONAL MATCH ()-[r {gortex_owner: $owner}]->()
RETURN manifests, nodes, count(r) AS relationships`, map[string]any{"owner": owner}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database))
	if err != nil || len(result.Records) != 1 {
		t.Fatalf("owner census: result=%#v err=%v", result, err)
	}
	return [3]int64{recordInt64(result.Records[0], "manifests"), recordInt64(result.Records[0], "nodes"), recordInt64(result.Records[0], "relationships")}
}

func recordInt64(record *neo4j.Record, key string) int64 {
	value, _ := record.Get(key)
	count, _ := value.(int64)
	return count
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
	var nodes []*graph.Node
	if err := snapshot.ReadNodePages(ctx, func(page []*graph.Node) error { nodes = append(nodes, page...); return nil }); err != nil {
		t.Fatal(err)
	}
	var edges []graph.ScopedEdgeRow
	if err := snapshot.ReadEdgePages(ctx, func(page []graph.ScopedEdgeRow) error { edges = append(edges, page...); return nil }); err != nil {
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
