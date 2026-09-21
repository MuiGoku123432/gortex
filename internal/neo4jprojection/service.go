package neo4jprojection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// Request identifies one exact projection owner and immutable source scope.
type Request struct {
	Owner       string
	OperationID string
	Scope       graph.ProjectionScope
}

// Result describes the source and activated target generation without secrets.
type Result struct {
	Owner             string
	OperationID       string
	PendingGeneration string
	ActiveGeneration  string
	SourceGeneration  int64
	NodeCount         int
	EdgeCount         int
	Phase             string
	Complete          bool
	CleanupComplete   bool
}

// ProjectionBatch is the complete tracer slice staged under one generation.
type ProjectionBatch struct {
	Owner             string
	OperationID       string
	PendingGeneration string
	SourceGeneration  int64
	Nodes             []*graph.Node
	Edges             []graph.ScopedEdgeRow
}

// Transport is an invocation-local projection target. Activate is the only
// operation allowed to change the exact owner's visible generation pointer.
type Transport interface {
	Inspect(context.Context) error
	Acquire(context.Context, string, string) (string, error)
	Stage(context.Context, ProjectionBatch) error
	Activate(context.Context, string, string, string, string, Result) error
	Close(context.Context) error
}

type TransportFactory func(context.Context) (Transport, error)

type Service struct {
	snapshots graph.ScopedProjectionSnapshotOpener
	transport TransportFactory
}

func NewService(snapshots graph.ScopedProjectionSnapshotOpener, transport TransportFactory) *Service {
	return &Service{snapshots: snapshots, transport: transport}
}

func (s *Service) Push(ctx context.Context, request Request) (result Result, retErr error) {
	request.Owner = strings.TrimSpace(request.Owner)
	request.OperationID = strings.TrimSpace(request.OperationID)
	if request.Owner == "" || request.OperationID == "" {
		return result, fmt.Errorf("neo4j projection owner and operation ID are required")
	}
	if s == nil || s.snapshots == nil || s.transport == nil {
		return result, fmt.Errorf("neo4j projection service is not configured")
	}

	result = Result{Owner: request.Owner, OperationID: request.OperationID, Phase: "opening_snapshot"}
	snapshot, err := s.snapshots.OpenScopedProjectionSnapshot(ctx, request.Scope)
	if err != nil {
		return result, fmt.Errorf("open projection snapshot: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, snapshot.Close()) }()

	descriptor := snapshot.Descriptor()
	result.SourceGeneration = descriptor.SourceGeneration
	result.PendingGeneration = generationKey(request.Owner, request.OperationID, descriptor.SourceGeneration)

	result.Phase = "reading_snapshot"
	nodes, err := snapshot.ReadNodes(ctx)
	if err != nil {
		return result, fmt.Errorf("read projection nodes: %w", err)
	}
	edges, err := snapshot.ReadEdges(ctx)
	if err != nil {
		return result, fmt.Errorf("read projection edges: %w", err)
	}
	result.NodeCount, result.EdgeCount = len(nodes), len(edges)

	transport, err := s.transport(ctx)
	if err != nil {
		return result, fmt.Errorf("create neo4j transport: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, transport.Close(context.WithoutCancel(ctx))) }()

	result.Phase = "inspecting"
	if err := transport.Inspect(ctx); err != nil {
		return result, fmt.Errorf("inspect neo4j: %w", err)
	}
	result.Phase = "locking"
	prior, err := transport.Acquire(ctx, request.Owner, request.OperationID)
	if err != nil {
		return result, fmt.Errorf("acquire projection owner: %w", err)
	}
	result.ActiveGeneration = prior
	result.Phase = "staging"
	if err := transport.Stage(ctx, ProjectionBatch{Owner: request.Owner, OperationID: request.OperationID, PendingGeneration: result.PendingGeneration, SourceGeneration: descriptor.SourceGeneration, Nodes: nodes, Edges: edges}); err != nil {
		return result, fmt.Errorf("stage projection: %w", err)
	}
	result.Phase = "activating"
	if err := transport.Activate(ctx, request.Owner, request.OperationID, result.PendingGeneration, prior, result); err != nil {
		return result, fmt.Errorf("activate projection: %w", err)
	}
	result.ActiveGeneration = result.PendingGeneration
	result.Phase = "complete"
	result.Complete = true
	return result, nil
}

func generationKey(owner, operation string, sourceGeneration int64) string {
	return projectionGenerationKey(owner, operation, sourceGeneration)
}
