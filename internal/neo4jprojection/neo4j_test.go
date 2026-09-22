package neo4jprojection

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zzet/gortex/internal/graph"
)

var expectedRelationshipType = sanitizeRelationshipType(graph.EdgeReferences)

type protocolTransport struct {
	active              string
	lockOperation       string
	lockGeneration      string
	lockAttempt         string
	leaseUntil          time.Time
	failed              bool
	staged              []ProjectionBatch
	activated           bool
	cleanupRemaining    CleanupCounts
	cleanupErr          error
	cancelDuringCleanup bool
}

func (t *protocolTransport) Inspect(context.Context, bool) error { return nil }
func (t *protocolTransport) Plan(context.Context, string, IntendedPlan) (TargetPlan, error) {
	return TargetPlan{ActiveGeneration: t.active}, nil
}
func (t *protocolTransport) LeaseDuration() time.Duration { return 20 * time.Millisecond }
func (t *protocolTransport) Renew(_ context.Context, _ string, operation, generation, attempt string, leaseUntil time.Time) error {
	if operation != t.lockOperation || generation != t.lockGeneration || attempt != t.lockAttempt || t.failed || time.Now().After(t.leaseUntil) {
		return errors.New("attempt lease lost")
	}
	t.leaseUntil = leaseUntil
	return nil
}
func (t *protocolTransport) Acquire(_ context.Context, _ string, operation, generation, attempt string, leaseUntil time.Time) (string, error) {
	if t.lockOperation != "" && !t.failed && time.Now().Before(t.leaseUntil) && t.lockAttempt != attempt {
		return "", errors.New("owner locked")
	}
	t.lockOperation = operation
	t.lockGeneration = generation
	t.lockAttempt = attempt
	t.leaseUntil = leaseUntil
	t.failed = false
	return t.active, nil
}
func (t *protocolTransport) Abort(_ context.Context, _ string, operation, generation, attempt string, _ int) error {
	if t.lockOperation == operation && t.lockGeneration == generation && t.lockAttempt == attempt {
		t.lockOperation, t.lockGeneration, t.lockAttempt = "", "", ""
	}
	return nil
}
func (t *protocolTransport) Stage(_ context.Context, batch ProjectionBatch) error {
	if batch.OperationID != t.lockOperation || batch.PendingGeneration != t.lockGeneration || batch.AttemptID != t.lockAttempt || time.Now().After(t.leaseUntil) {
		return errors.New("stage does not own lock")
	}
	t.staged = append(t.staged, batch)
	return nil
}

func TestNeo4jSlowStageRenewsLeaseAndSupersededWriterFailsClosed(t *testing.T) {
	transport := &protocolTransport{active: "generation-old"}
	_, err := transport.Acquire(context.Background(), "owner", "operation", "generation", "attempt-one", time.Now().Add(20*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		time.Sleep(10 * time.Millisecond)
		if err := transport.Renew(context.Background(), "owner", "operation", "generation", "attempt-one", time.Now().Add(20*time.Millisecond)); err != nil {
			t.Fatal(err)
		}
		if _, err := transport.Acquire(context.Background(), "owner", "replacement", "generation", "attempt-two", time.Now().Add(time.Minute)); err == nil {
			t.Fatal("takeover overlapped a renewed writer")
		}
	}
	time.Sleep(25 * time.Millisecond)
	if _, err := transport.Acquire(context.Background(), "owner", "replacement", "generation", "attempt-two", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := transport.Stage(context.Background(), ProjectionBatch{Owner: "owner", OperationID: "operation", PendingGeneration: "generation", AttemptID: "attempt-one"}); err == nil {
		t.Fatal("superseded writer continued staging")
	}
	transport.Abort(context.Background(), "owner", "operation", "generation", "attempt-one", 1)
	if transport.lockAttempt != "attempt-two" {
		t.Fatal("old abort cleared replacement ownership")
	}
}
func (t *protocolTransport) MarkComplete(context.Context, string, string, string, string, MaterializedCounts) (MaterializedCounts, error) {
	counts := MaterializedCounts{}
	seen := map[string]bool{}
	for _, batch := range t.staged {
		counts.Relationships += len(batch.Edges)
		for _, node := range batch.Nodes {
			seen[node.ID] = true
		}
		for _, row := range batch.Edges {
			seen[row.Edge.From] = true
			seen[row.Edge.To] = true
		}
	}
	counts.Nodes = len(seen)
	return counts, nil
}
func (t *protocolTransport) Activate(_ context.Context, _ string, operation, generation, _ string, prior string, _ Result) error {
	if operation != t.lockOperation || generation != t.lockGeneration || prior != t.active {
		return errors.New("activation compare-and-switch failed")
	}
	t.active = generation
	t.activated = true
	t.lockOperation = ""
	return nil
}
func (t *protocolTransport) Reconcile(ctx context.Context, _ string, _ string, active string, _ string, batchSize int) (CleanupCounts, error) {
	if !t.activated || active != t.active {
		return CleanupCounts{}, errors.New("cleanup before activation")
	}
	if t.cancelDuringCleanup {
		<-ctx.Done()
		return t.cleanupRemaining, ctx.Err()
	}
	if t.cleanupRemaining.Nodes > batchSize || t.cleanupRemaining.Relationships > batchSize {
		return CleanupCounts{}, errors.New("cleanup batch exceeded")
	}
	return t.cleanupRemaining, t.cleanupErr
}
func (t *protocolTransport) Close(context.Context) error { return nil }

func protocolService(transport Transport) *Service {
	return NewService(tracerOpener{&tracerSnapshot{}}, func(context.Context) (Transport, error) { return transport, nil })
}

func TestNeo4jConstraints(t *testing.T) {
	queries := constraintQueries([]string{"GORTEX_CALLS", expectedRelationshipType})
	if len(queries) != 4 {
		t.Fatalf("constraint query count = %d, want 4", len(queries))
	}
	joined := strings.Join(queries, "\n")
	for _, required := range []string{"GortexNode", "GortexProjectionManifest", "GORTEX_CALLS", expectedRelationshipType, "IS UNIQUE"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("constraints omit %q: %s", required, joined)
		}
	}
	for _, query := range queries {
		if strings.Contains(query, "$type") || strings.Contains(query, "+") {
			t.Fatalf("constraint contains dynamic token construction: %s", query)
		}
	}
}

func TestNeo4jGenerationActivation(t *testing.T) {
	transport := &protocolTransport{active: "generation-old"}
	result, err := protocolService(transport).Push(context.Background(), Request{
		Owner: "owner", OperationID: "operation", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}, BatchSize: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !transport.activated || transport.active != result.PendingGeneration || result.ActiveGeneration != result.PendingGeneration {
		t.Fatalf("activation was not atomic: transport=%#v result=%#v", transport, result)
	}
	if !result.Complete || !result.CleanupComplete || result.CleanupStatus != "complete" {
		t.Fatalf("unexpected completion result: %#v", result)
	}

	transport.lockOperation = "other-operation"
	if _, err := transport.Acquire(context.Background(), "owner", "new-operation", "new-generation", "attempt", time.Now().Add(time.Minute)); err == nil {
		t.Fatal("same-owner overlap was accepted")
	}
}

func TestNeo4jAbandonedAttemptRecovery(t *testing.T) {
	transport := &protocolTransport{
		active: "generation-old", lockOperation: "operation", lockGeneration: "generation-one",
		lockAttempt: "dead-attempt", leaseUntil: time.Now().Add(-time.Second),
	}
	result, err := protocolService(transport).Push(context.Background(), Request{
		Owner: "owner", OperationID: "operation", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}, BatchSize: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Complete || result.ActiveGeneration == "generation-old" {
		t.Fatalf("abandoned attempt was not recovered: %#v", result)
	}

	transport.lockOperation = "live-operation"
	transport.lockAttempt = "live-attempt"
	transport.leaseUntil = time.Now().Add(time.Minute)
	if _, err := transport.Acquire(context.Background(), "owner", "other", "other-generation", "other-attempt", time.Now().Add(time.Minute)); err == nil {
		t.Fatal("live concurrent attempt was taken over")
	}
}

func TestNeo4jStageQueriesFenceEveryMutationTransaction(t *testing.T) {
	for _, query := range []string{nodeMergeQuery("Function"), relationshipMergeQuery(expectedRelationshipType)} {
		for _, required := range []string{"m.operation_id = $operation", "m.pending_generation = $generation", "m.attempt_id = $attempt", "m.lease_until >= datetime()", "SET m.lease_until = datetime($lease_until)"} {
			if !strings.Contains(query, required) {
				t.Fatalf("stage query missing transactional fence %q: %s", required, query)
			}
		}
	}
}

func TestNeo4jCompletionRequiresIndependentIntendedCensus(t *testing.T) {
	source, err := os.ReadFile("neo4j.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{"m.intended_node_count", "m.observed_node_count", "nodes = $intended_nodes", "relationships = $intended_relationships"} {
		if !strings.Contains(text, required) {
			t.Fatalf("completion protocol missing %q", required)
		}
	}
}

func TestNeo4jRetryBounds(t *testing.T) {
	if neo4jTransactionTimeout != 30*time.Second || neo4jRetryCeiling != 30*time.Second {
		t.Fatalf("timeouts = transaction %s retry %s", neo4jTransactionTimeout, neo4jRetryCeiling)
	}
	for _, query := range []string{nodeMergeQuery("Function"), relationshipMergeQuery(expectedRelationshipType)} {
		if !strings.Contains(query, "UNWIND $rows") || !strings.Contains(query, "MERGE") {
			t.Fatalf("write is not bounded idempotent UNWIND/MERGE: %s", query)
		}
	}
}

func TestNeo4jStaleReconciliation(t *testing.T) {
	transport := &protocolTransport{active: "generation-old", cleanupRemaining: CleanupCounts{Nodes: 2, Relationships: 1}, cleanupErr: ErrCleanupIncomplete}
	result, err := protocolService(transport).Push(context.Background(), Request{
		Owner: "owner", OperationID: "operation", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}, BatchSize: 2,
	})
	if !errors.Is(err, ErrCleanupIncomplete) {
		t.Fatalf("error = %v, want cleanup incomplete", err)
	}
	if !result.Complete || result.ActiveGeneration != result.PendingGeneration || result.CleanupComplete || result.StaleNodeCount != 2 || result.StaleEdgeCount != 1 || result.CleanupStatus != "incomplete" || result.CleanupAction == "" {
		t.Fatalf("cleanup result is not truthful: %#v", result)
	}
}

func TestNeo4jAbortIsBoundedAndCancellable(t *testing.T) {
	abortSource, err := os.ReadFile("neo4j.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(abortSource)
	for _, required := range []string{"LIMIT $batch_size DELETE r", "LIMIT $batch_size DELETE n", "m.operation_state = 'failed'"} {
		if !strings.Contains(source, required) {
			t.Fatalf("abort path missing %q", required)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	transport := &protocolTransport{lockOperation: "operation", lockGeneration: "generation", lockAttempt: "attempt"}
	if err := transport.Abort(ctx, "owner", "operation", "generation", "attempt", 1); err != nil {
		t.Fatal(err)
	}
}

func TestNeo4jCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := &protocolTransport{active: "generation-old", cleanupRemaining: CleanupCounts{Nodes: 1}, cancelDuringCleanup: true}
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	result, err := protocolService(transport).Push(ctx, Request{
		Owner: "owner", OperationID: "operation", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}, BatchSize: 2, Timeout: time.Minute,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
	if !result.Complete || !result.Cancelled || result.CleanupComplete || result.ActiveGeneration != result.PendingGeneration {
		t.Fatalf("activation/cleanup cancellation result is not truthful: %#v", result)
	}
}
