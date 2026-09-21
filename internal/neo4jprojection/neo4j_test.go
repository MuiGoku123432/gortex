package neo4jprojection

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zzet/gortex/internal/graph"
)

type protocolTransport struct {
	active             string
	lockOperation      string
	lockGeneration     string
	staged             []ProjectionBatch
	activated          bool
	cleanupRemaining   CleanupCounts
	cleanupErr         error
	cancelDuringCleanup bool
}

func (t *protocolTransport) Inspect(context.Context, bool) error { return nil }
func (t *protocolTransport) Acquire(_ context.Context, _ string, operation, generation string) (string, error) {
	if t.lockOperation != "" && t.lockOperation != operation {
		return "", errors.New("owner locked")
	}
	t.lockOperation = operation
	t.lockGeneration = generation
	return t.active, nil
}
func (t *protocolTransport) Stage(_ context.Context, batch ProjectionBatch) error {
	if batch.OperationID != t.lockOperation || batch.PendingGeneration != t.lockGeneration {
		return errors.New("stage does not own lock")
	}
	t.staged = append(t.staged, batch)
	return nil
}
func (t *protocolTransport) Activate(_ context.Context, _ string, operation, generation, prior string, _ Result) error {
	if operation != t.lockOperation || generation != t.lockGeneration || prior != t.active {
		return errors.New("activation compare-and-switch failed")
	}
	t.active = generation
	t.activated = true
	t.lockOperation = ""
	return nil
}
func (t *protocolTransport) Reconcile(ctx context.Context, _ string, active string, batchSize int) (CleanupCounts, error) {
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
	queries := constraintQueries([]string{"GORTEX_CALLS", "GORTEX_REFERENCES"})
	if len(queries) != 4 {
		t.Fatalf("constraint query count = %d, want 4", len(queries))
	}
	joined := strings.Join(queries, "\n")
	for _, required := range []string{"GortexNode", "GortexProjectionManifest", "GORTEX_CALLS", "GORTEX_REFERENCES", "IS UNIQUE"} {
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
	if _, err := transport.Acquire(context.Background(), "owner", "new-operation", "new-generation"); err == nil {
		t.Fatal("same-owner overlap was accepted")
	}
}

func TestNeo4jRetryBounds(t *testing.T) {
	if neo4jTransactionTimeout != 30*time.Second || neo4jRetryCeiling != 30*time.Second {
		t.Fatalf("timeouts = transaction %s retry %s", neo4jTransactionTimeout, neo4jRetryCeiling)
	}
	for _, query := range []string{nodeMergeQuery("Function"), relationshipMergeQuery("GORTEX_CALLS")} {
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
