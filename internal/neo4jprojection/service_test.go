package neo4jprojection

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/progress"
)

type tracerSnapshot struct{ closed bool }

func (s *tracerSnapshot) Descriptor() graph.ProjectionSnapshotDescriptor {
	return graph.ProjectionSnapshotDescriptor{SourceGeneration: 7, Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}}
}
func (s *tracerSnapshot) ReadNodePages(_ context.Context, consume func([]*graph.Node) error) error {
	return consume([]*graph.Node{{ID: "fixture/repo/main.go::Run", Kind: graph.KindFunction, Name: "Run", RepoPrefix: "fixture/repo"}, {ID: "fixture/repo/main.go::Store", Kind: graph.KindType, Name: "Store", RepoPrefix: "fixture/repo"}})
}
func (s *tracerSnapshot) ReadEdgePages(_ context.Context, consume func([]graph.ScopedEdgeRow) error) error {
	source, target := &graph.Node{ID: "fixture/repo/main.go::Run"}, &graph.Node{ID: "fixture/repo/main.go::Store"}
	return consume([]graph.ScopedEdgeRow{{Edge: &graph.Edge{From: source.ID, To: target.ID, Kind: graph.EdgeReferences}, Source: source, Target: target}})
}
func (s *tracerSnapshot) Close() error { s.closed = true; return nil }

type tracerOpener struct{ snapshot *tracerSnapshot }

func (o tracerOpener) OpenScopedProjectionSnapshot(context.Context, graph.ProjectionScope) (graph.ScopedProjectionSnapshot, error) {
	return o.snapshot, nil
}

type tracerTransport struct {
	operations []string
	active     string
	failAt     string
	closed     bool
	closeBlock bool
	closeErr   error
}

func (t *tracerTransport) Inspect(context.Context, bool) error { return t.record("inspect") }
func (t *tracerTransport) Plan(context.Context, string, IntendedPlan) (TargetPlan, error) {
	return TargetPlan{ActiveGeneration: t.active}, nil
}
func (t *tracerTransport) LeaseDuration() time.Duration { return time.Minute }
func (t *tracerTransport) Renew(context.Context, string, string, string, string, time.Time) error {
	return t.record("renew")
}
func (t *tracerTransport) Acquire(context.Context, string, string, string, string, time.Time) (string, error) {
	if err := t.record("lock"); err != nil {
		return "", err
	}
	return t.active, nil
}
func (t *tracerTransport) Abort(context.Context, string, string, string, string, int) error {
	return t.record("abort")
}
func (t *tracerTransport) Stage(context.Context, ProjectionBatch) error { return t.record("stage") }
func (t *tracerTransport) MarkComplete(context.Context, string, string, string, string, MaterializedCounts) (MaterializedCounts, error) {
	return MaterializedCounts{Nodes: 2, Relationships: 1}, t.record("mark_complete")
}
func (t *tracerTransport) Activate(_ context.Context, owner, operation, generation, attempt, prior string, result Result) error {
	if err := t.record("activate"); err != nil {
		return err
	}
	t.active = generation
	return nil
}
func (t *tracerTransport) Reconcile(context.Context, string, string, string, string, int) (CleanupCounts, error) {
	return CleanupCounts{}, t.record("cleanup")
}
func (t *tracerTransport) Close(ctx context.Context) error {
	t.closed = true
	if t.closeBlock {
		<-ctx.Done()
		return errors.Join(ctx.Err(), t.closeErr)
	}
	if err := t.record("close"); err != nil {
		return err
	}
	return t.closeErr
}
func (t *tracerTransport) record(operation string) error {
	t.operations = append(t.operations, operation)
	if t.failAt == operation {
		return errors.New("transport failure")
	}
	return nil
}

type batchSnapshot struct {
	nodes       []*graph.Node
	edges       []graph.ScopedEdgeRow
	pageSize    int
	outstanding int
	maxRead     int
}

func (s *batchSnapshot) Descriptor() graph.ProjectionSnapshotDescriptor {
	return graph.ProjectionSnapshotDescriptor{SourceGeneration: 9, Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}}
}
func (s *batchSnapshot) ReadNodePages(ctx context.Context, consume func([]*graph.Node) error) error {
	for start := 0; start < len(s.nodes); start += s.pageSize {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := min(start+s.pageSize, len(s.nodes))
		s.outstanding = end - start
		s.maxRead = max(s.maxRead, s.outstanding)
		if err := consume(s.nodes[start:end]); err != nil {
			return err
		}
		s.outstanding = 0
	}
	return nil
}
func (s *batchSnapshot) ReadEdgePages(ctx context.Context, consume func([]graph.ScopedEdgeRow) error) error {
	for start := 0; start < len(s.edges); start += s.pageSize {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := min(start+s.pageSize, len(s.edges))
		s.outstanding = end - start
		s.maxRead = max(s.maxRead, s.outstanding)
		if err := consume(s.edges[start:end]); err != nil {
			return err
		}
		s.outstanding = 0
	}
	return nil
}
func (s *batchSnapshot) Close() error { return nil }

type batchOpener struct{ snapshot *batchSnapshot }

func (o batchOpener) OpenScopedProjectionSnapshot(context.Context, graph.ProjectionScope) (graph.ScopedProjectionSnapshot, error) {
	return o.snapshot, nil
}

type batchTransport struct {
	batches    []ProjectionBatch
	cancel     context.CancelFunc
	markCounts *MaterializedCounts
	activated  bool
}

func (t *batchTransport) Inspect(context.Context, bool) error { return nil }
func (t *batchTransport) Plan(context.Context, string, IntendedPlan) (TargetPlan, error) {
	return TargetPlan{}, nil
}
func (t *batchTransport) LeaseDuration() time.Duration { return time.Minute }
func (t *batchTransport) Renew(context.Context, string, string, string, string, time.Time) error {
	return nil
}
func (t *batchTransport) Acquire(context.Context, string, string, string, string, time.Time) (string, error) {
	return "prior", nil
}
func (t *batchTransport) Abort(context.Context, string, string, string, string, int) error {
	return nil
}
func (t *batchTransport) Stage(ctx context.Context, batch ProjectionBatch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.batches = append(t.batches, batch)
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	return nil
}
func (t *batchTransport) MarkComplete(_ context.Context, _, _, _, _ string, intended MaterializedCounts) (MaterializedCounts, error) {
	if t.markCounts != nil {
		if *t.markCounts != intended {
			return *t.markCounts, errors.New("intended census mismatch")
		}
		return *t.markCounts, nil
	}
	counts := MaterializedCounts{}
	seen := map[string]bool{}
	edges := map[string]bool{}
	for _, batch := range t.batches {
		for _, row := range batch.Edges {
			edges[projectionEdgeLogicalKey("owner", row.Edge)] = true
		}
		for _, node := range batch.Nodes {
			seen[node.ID] = true
		}
		for _, row := range batch.Edges {
			seen[row.Edge.From] = true
			seen[row.Edge.To] = true
		}
	}
	counts.Nodes = len(seen)
	counts.Relationships = len(edges)
	if counts != intended {
		return counts, errors.New("intended census mismatch")
	}
	return counts, nil
}
func (t *batchTransport) Activate(context.Context, string, string, string, string, string, Result) error {
	t.activated = true
	return nil
}
func (t *batchTransport) Reconcile(context.Context, string, string, string, string, int) (CleanupCounts, error) {
	return CleanupCounts{}, nil
}
func (t *batchTransport) Close(context.Context) error { return nil }

type recordingReporter struct {
	ticks []struct {
		stage          string
		current, total int
	}
}

func (r *recordingReporter) Report(stage string, current, total int) {
	r.ticks = append(r.ticks, struct {
		stage          string
		current, total int
	}{stage, current, total})
}

func projectionRecords(count int) ([]*graph.Node, []graph.ScopedEdgeRow) {
	nodes := make([]*graph.Node, count)
	edges := make([]graph.ScopedEdgeRow, count)
	for i := range count {
		nodes[i] = &graph.Node{ID: string(rune(i + 1)), Kind: graph.KindFunction, RepoPrefix: "fixture/repo"}
		edges[i] = graph.ScopedEdgeRow{Edge: &graph.Edge{From: nodes[i].ID, To: nodes[i].ID, Kind: graph.EdgeReferences}, Source: nodes[i], Target: nodes[i]}
	}
	return nodes, edges
}

func TestProjectionDryRunReportsMapperOmissions(t *testing.T) {
	source := &graph.Node{ID: "fixture/repo/main.go::Run", Kind: graph.KindFunction, RepoPrefix: "fixture/repo", Meta: map[string]any{"api_token": "secret", "unsupported": make(chan int)}}
	snapshot := &batchSnapshot{nodes: []*graph.Node{source}, pageSize: 1}
	transport := &batchTransport{}
	result, err := NewService(batchOpener{snapshot}, func(context.Context) (Transport, error) { return transport, nil }).Push(context.Background(), Request{Owner: "owner", OperationID: "dry", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.SecretOmissions != 1 || result.UnsupportedOmissions != 1 {
		t.Fatalf("omissions not aggregated: %#v", result)
	}
}

func TestProjectionMissingPendingRecordBlocksActivation(t *testing.T) {
	for _, missing := range []string{"node", "relationship"} {
		t.Run(missing, func(t *testing.T) {
			transport := &batchTransport{}
			transportCounts := MaterializedCounts{Nodes: 2, Relationships: 1}
			if missing == "node" {
				transportCounts.Nodes--
			} else {
				transportCounts.Relationships--
			}
			transport.markCounts = &transportCounts
			result, err := NewService(tracerOpener{&tracerSnapshot{}}, func(context.Context) (Transport, error) {
				return transport, nil
			}).Push(context.Background(), Request{Owner: "owner", OperationID: "operation", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}})
			if err == nil || result.Complete || result.ActiveGeneration != "prior" || transport.activated {
				t.Fatalf("missing pending %s activated: result=%#v err=%v", missing, result, err)
			}
		})
	}
}

func TestProjectionIntendedCountsAreDeduplicated(t *testing.T) {
	node := &graph.Node{ID: "n", Kind: graph.KindFunction}
	edge := graph.ScopedEdgeRow{Edge: &graph.Edge{From: "n", To: "n", Kind: graph.EdgeCalls}, Source: node, Target: node}
	snapshot := &batchSnapshot{nodes: []*graph.Node{node, node}, edges: []graph.ScopedEdgeRow{edge, edge}, pageSize: 2}
	transport := &batchTransport{}
	result, err := NewService(batchOpener{snapshot}, func(context.Context) (Transport, error) { return transport, nil }).Push(context.Background(), Request{Owner: "owner", OperationID: "apply", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.NodeCount != 1 || result.EdgeCount != 1 {
		t.Fatalf("intended counts were not deduplicated: %#v", result)
	}
}

func TestProjectionDryRunCountsMaterializedPlaceholders(t *testing.T) {
	source := &graph.Node{ID: "fixture/repo/main.go::Run", Kind: graph.KindFunction}
	edge := &graph.Edge{From: source.ID, To: "fixture/repo/missing.go::Missing", Kind: graph.EdgeCalls}
	snapshot := &batchSnapshot{
		nodes:    []*graph.Node{source},
		edges:    []graph.ScopedEdgeRow{{Edge: edge, Source: source, Target: nil}},
		pageSize: 1,
	}
	transport := &batchTransport{}
	result, err := NewService(batchOpener{snapshot}, func(context.Context) (Transport, error) { return transport, nil }).Push(context.Background(), Request{Owner: "owner", OperationID: "dry", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Complete || result.NodeCount != 2 || result.EdgeCount != 1 || len(transport.batches) != 0 {
		t.Fatalf("dry-run materialized counts are false: result=%#v batches=%d", result, len(transport.batches))
	}
}

func TestNormalizeRequestDurationBounds(t *testing.T) {
	base := Request{Profile: "prod", Namespace: "view", Workspace: "ws", Project: "p", Repositories: []string{"repo"}}
	for name, mutate := range map[string]func(*Request){
		"operation":   func(r *Request) { r.OperationTimeout = "30m1s" },
		"transaction": func(r *Request) { r.TransactionTimeout = "31s" },
		"retry":       func(r *Request) { r.RetryTimeout = "31s" },
		"overflow":    func(r *Request) { r.OperationTimeout = "999999999999999999999h" },
	} {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			if _, err := NormalizeRequest(request); err == nil {
				t.Fatal("oversized or invalid duration was accepted")
			}
		})
	}
}

func TestProjectionBatchAggregation(t *testing.T) {
	nodes, edges := projectionRecords(1201)
	t.Run("dry run counts without target mutation", func(t *testing.T) {
		transport := &batchTransport{}
		result, err := NewService(batchOpener{&batchSnapshot{nodes: nodes, edges: edges, pageSize: 127}}, func(context.Context) (Transport, error) { return transport, nil }).Push(context.Background(), Request{Owner: "owner", OperationID: "dry", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}, DryRun: true})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Complete || !result.DryRun || result.NodeCount != len(nodes) || result.EdgeCount != len(edges) || len(transport.batches) != 0 {
			t.Fatalf("unexpected dry-run result: %#v batches=%d", result, len(transport.batches))
		}
	})
	for _, batchSize := range []int{0, 73} {
		snapshot := &batchSnapshot{nodes: nodes, edges: edges, pageSize: 127}
		transport := &batchTransport{}
		result, err := NewService(batchOpener{snapshot}, func(context.Context) (Transport, error) { return transport, nil }).Push(context.Background(), Request{Owner: "owner", OperationID: "op", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}, BatchSize: batchSize})
		if err != nil {
			t.Fatal(err)
		}
		wantMax := 500
		if batchSize > 0 {
			wantMax = batchSize
		}
		for _, batch := range transport.batches {
			if len(batch.Nodes)+len(batch.Edges) > wantMax {
				t.Fatalf("batch records = %d, max %d", len(batch.Nodes)+len(batch.Edges), wantMax)
			}
		}
		if result.NodeCount != len(nodes) || result.EdgeCount != len(edges) || !result.Complete {
			t.Fatalf("unexpected result: %#v", result)
		}
	}
}

func TestProjectionProgressCadence(t *testing.T) {
	nodes, _ := projectionRecords(2100)
	reporter := &recordingReporter{}
	ctx := progress.WithReporter(context.Background(), reporter)
	_, err := NewService(batchOpener{&batchSnapshot{nodes: nodes, pageSize: 211}}, func(context.Context) (Transport, error) { return &batchTransport{}, nil }).Push(ctx, Request{Owner: "owner", OperationID: "op", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}})
	if err != nil {
		t.Fatal(err)
	}
	last := -1
	seen1000, seen2000 := false, false
	for _, tick := range reporter.ticks {
		if tick.current < last {
			t.Fatalf("non-monotonic progress: %v", reporter.ticks)
		}
		last = tick.current
		seen1000 = seen1000 || tick.current == 1000
		seen2000 = seen2000 || tick.current == 2000
	}
	if !seen1000 || !seen2000 {
		t.Fatalf("missing exact record cadence: %v", reporter.ticks)
	}
}

func TestProjectionBoundedReads(t *testing.T) {
	nodes, edges := projectionRecords(701)
	ctx, cancel := context.WithCancel(context.Background())
	snapshot := &batchSnapshot{nodes: nodes, edges: edges, pageSize: 89}
	transport := &batchTransport{cancel: cancel}
	result, err := NewService(batchOpener{snapshot}, func(context.Context) (Transport, error) { return transport, nil }).Push(ctx, Request{Owner: "owner", OperationID: "op", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}, Timeout: time.Minute})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
	if result.Complete || snapshot.maxRead > 89 || snapshot.outstanding != 0 {
		t.Fatalf("unbounded/incomplete state: result=%#v snapshot=%#v", result, snapshot)
	}
}

func TestProjectionCloseIsBoundedAndPreservesOperationError(t *testing.T) {
	operationErr := errors.New("operation failed")
	closeErr := errors.New("close failed")
	transport := &tracerTransport{active: "generation-old", failAt: "inspect", closeBlock: true, closeErr: closeErr}
	start := time.Now()
	_, err := NewService(tracerOpener{&tracerSnapshot{}}, func(context.Context) (Transport, error) {
		return transport, nil
	}).Push(context.Background(), Request{Owner: "owner", OperationID: "operation", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}})
	elapsed := time.Since(start)
	if elapsed < closeTimeout || elapsed > closeTimeout+2*time.Second {
		t.Fatalf("close duration = %s, want bounded near %s", elapsed, closeTimeout)
	}
	if err == nil || !strings.Contains(err.Error(), operationErr.Error()) && !strings.Contains(err.Error(), "transport failure") || !errors.Is(err, closeErr) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("joined operation/close error = %v", err)
	}
}

func TestNeo4jTracerContract(t *testing.T) {
	t.Run("orders inspection locking staging activation and close", func(t *testing.T) {
		snapshot := &tracerSnapshot{}
		transport := &tracerTransport{active: "generation-old"}
		service := NewService(tracerOpener{snapshot}, func(context.Context) (Transport, error) { return transport, nil })
		result, err := service.Push(context.Background(), Request{Owner: "workspace:fixture/repo", OperationID: "operation-1", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Complete || result.ActiveGeneration == "" || result.ActiveGeneration == "generation-old" || result.NodeCount != 2 || result.EdgeCount != 1 {
			t.Fatalf("unexpected result: %#v", result)
		}
		if want := []string{"inspect", "lock", "renew", "stage", "renew", "stage", "renew", "mark_complete", "renew", "activate", "cleanup", "close"}; !reflect.DeepEqual(transport.operations, want) {
			t.Fatalf("operations = %v, want %v", transport.operations, want)
		}
		if !snapshot.closed || !transport.closed {
			t.Fatalf("lifecycle incomplete: snapshot=%v transport=%v", snapshot.closed, transport.closed)
		}
	})

	t.Run("preserves prior active generation before activation", func(t *testing.T) {
		transport := &tracerTransport{active: "generation-old", failAt: "stage"}
		service := NewService(tracerOpener{&tracerSnapshot{}}, func(context.Context) (Transport, error) { return transport, nil })
		result, err := service.Push(context.Background(), Request{Owner: "workspace:fixture/repo", OperationID: "operation-2", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}})
		if err == nil {
			t.Fatal("expected staging failure")
		}
		if result.Complete || result.ActiveGeneration != "generation-old" || transport.active != "generation-old" {
			t.Fatalf("old generation changed: result=%#v active=%q", result, transport.active)
		}
	})

	t.Run("does not construct transport until explicit push", func(t *testing.T) {
		constructed := 0
		service := NewService(tracerOpener{&tracerSnapshot{}}, func(context.Context) (Transport, error) { constructed++; return &tracerTransport{}, nil })
		if service == nil || constructed != 0 {
			t.Fatalf("transport constructed at startup: %d", constructed)
		}
	})
}
