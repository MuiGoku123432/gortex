package neo4jprojection

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/zzet/gortex/internal/graph"
)

type tracerSnapshot struct{ closed bool }

func (s *tracerSnapshot) Descriptor() graph.ProjectionSnapshotDescriptor {
	return graph.ProjectionSnapshotDescriptor{SourceGeneration: 7, Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}}
}
func (s *tracerSnapshot) ReadNodes(context.Context) ([]*graph.Node, error) {
	return []*graph.Node{{ID: "fixture/repo/main.go::Run", Kind: graph.KindFunction, Name: "Run", RepoPrefix: "fixture/repo"}, {ID: "fixture/repo/main.go::Store", Kind: graph.KindType, Name: "Store", RepoPrefix: "fixture/repo"}}, nil
}
func (s *tracerSnapshot) ReadEdges(context.Context) ([]graph.ScopedEdgeRow, error) {
	source, target := &graph.Node{ID: "fixture/repo/main.go::Run"}, &graph.Node{ID: "fixture/repo/main.go::Store"}
	return []graph.ScopedEdgeRow{{Edge: &graph.Edge{From: source.ID, To: target.ID, Kind: graph.EdgeReferences}, Source: source, Target: target}}, nil
}
func (s *tracerSnapshot) Close() error { s.closed = true; return nil }

type tracerOpener struct{ snapshot *tracerSnapshot }
func (o tracerOpener) OpenScopedProjectionSnapshot(context.Context, graph.ProjectionScope) (graph.ScopedProjectionSnapshot, error) { return o.snapshot, nil }

type tracerTransport struct {
	operations []string
	active string
	failAt string
	closed bool
}
func (t *tracerTransport) Inspect(context.Context) error { return t.record("inspect") }
func (t *tracerTransport) Acquire(context.Context, string, string) (string, error) { if err := t.record("lock"); err != nil { return "", err }; return t.active, nil }
func (t *tracerTransport) Stage(context.Context, ProjectionBatch) error { return t.record("stage") }
func (t *tracerTransport) Activate(_ context.Context, owner, operation, generation, prior string, result Result) error { if err := t.record("activate"); err != nil { return err }; t.active = generation; return nil }
func (t *tracerTransport) Close(context.Context) error { t.closed = true; return t.record("close") }
func (t *tracerTransport) record(operation string) error { t.operations = append(t.operations, operation); if t.failAt == operation { return errors.New("transport failure") }; return nil }

func TestNeo4jTracerContract(t *testing.T) {
	t.Run("orders inspection locking staging activation and close", func(t *testing.T) {
		snapshot := &tracerSnapshot{}
		transport := &tracerTransport{active: "generation-old"}
		service := NewService(tracerOpener{snapshot}, func(context.Context) (Transport, error) { return transport, nil })
		result, err := service.Push(context.Background(), Request{Owner: "workspace:fixture/repo", OperationID: "operation-1", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}})
		if err != nil { t.Fatal(err) }
		if !result.Complete || result.ActiveGeneration == "" || result.ActiveGeneration == "generation-old" || result.NodeCount != 2 || result.EdgeCount != 1 { t.Fatalf("unexpected result: %#v", result) }
		if want := []string{"inspect", "lock", "stage", "activate", "close"}; !reflect.DeepEqual(transport.operations, want) { t.Fatalf("operations = %v, want %v", transport.operations, want) }
		if !snapshot.closed || !transport.closed { t.Fatalf("lifecycle incomplete: snapshot=%v transport=%v", snapshot.closed, transport.closed) }
	})

	t.Run("preserves prior active generation before activation", func(t *testing.T) {
		transport := &tracerTransport{active: "generation-old", failAt: "stage"}
		service := NewService(tracerOpener{&tracerSnapshot{}}, func(context.Context) (Transport, error) { return transport, nil })
		result, err := service.Push(context.Background(), Request{Owner: "workspace:fixture/repo", OperationID: "operation-2", Scope: graph.ProjectionScope{Repositories: []string{"fixture/repo"}}})
		if err == nil { t.Fatal("expected staging failure") }
		if result.Complete || result.ActiveGeneration != "generation-old" || transport.active != "generation-old" { t.Fatalf("old generation changed: result=%#v active=%q", result, transport.active) }
	})

	t.Run("does not construct transport until explicit push", func(t *testing.T) {
		constructed := 0
		service := NewService(tracerOpener{&tracerSnapshot{}}, func(context.Context) (Transport, error) { constructed++; return &tracerTransport{}, nil })
		if service == nil || constructed != 0 { t.Fatalf("transport constructed at startup: %d", constructed) }
	})
}
