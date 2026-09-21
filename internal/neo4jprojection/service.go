package neo4jprojection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/progress"
)

// Request identifies one exact projection owner and immutable source scope.
type Request struct {
	Owner       string
	OperationID string
	Scope       graph.ProjectionScope
	BatchSize   int
	Timeout     time.Duration
	DryRun      bool
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
	StaleNodeCount    int
	StaleEdgeCount    int
	Phase             string
	Complete          bool
	CleanupComplete   bool
	CleanupStatus     string
	CleanupAction     string
	DryRun            bool
	Cancelled         bool
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

type CleanupCounts struct {
	Nodes         int
	Relationships int
}

var ErrCleanupIncomplete = errors.New("neo4j projection cleanup incomplete")

// Transport is an invocation-local projection target. Activate is the only
// operation allowed to change the exact owner's visible generation pointer.
type Transport interface {
	Inspect(context.Context, bool) error
	Acquire(context.Context, string, string, string) (string, error)
	Stage(context.Context, ProjectionBatch) error
	Activate(context.Context, string, string, string, string, Result) error
	Reconcile(context.Context, string, string, int) (CleanupCounts, error)
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

const (
	defaultBatchSize = 500
	defaultTimeout   = 30 * time.Minute
)

func (s *Service) Push(ctx context.Context, request Request) (result Result, retErr error) {
	request.Owner = strings.TrimSpace(request.Owner)
	request.OperationID = strings.TrimSpace(request.OperationID)
	if request.Owner == "" || request.OperationID == "" {
		return result, fmt.Errorf("neo4j projection owner and operation ID are required")
	}
	if s == nil || s.snapshots == nil || s.transport == nil {
		return result, fmt.Errorf("neo4j projection service is not configured")
	}
	if request.BatchSize < 0 || request.BatchSize > defaultBatchSize {
		return result, fmt.Errorf("neo4j projection batch size must be between 1 and %d", defaultBatchSize)
	}
	if request.BatchSize == 0 {
		request.BatchSize = defaultBatchSize
	}
	if request.Timeout <= 0 {
		request.Timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	reporter := progress.FromContext(ctx)
	report := func(stage string, current int) {
		reporter.Report(stage, current, 0)
	}

	result = Result{Owner: request.Owner, OperationID: request.OperationID, Phase: "opening_snapshot", DryRun: request.DryRun}
	report(result.Phase, 0)
	snapshot, err := s.snapshots.OpenScopedProjectionSnapshot(ctx, request.Scope)
	if err != nil {
		return result, fmt.Errorf("open projection snapshot: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, snapshot.Close()) }()

	descriptor := snapshot.Descriptor()
	result.SourceGeneration = descriptor.SourceGeneration
	result.PendingGeneration = generationKey(request.Owner, request.OperationID, descriptor.SourceGeneration)

	transport, err := s.transport(ctx)
	if err != nil {
		return result, fmt.Errorf("create neo4j transport: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, transport.Close(context.WithoutCancel(ctx))) }()

	result.Phase = "inspecting"
	report(result.Phase, 0)
	if err := transport.Inspect(ctx, request.DryRun); err != nil {
		result.Cancelled = errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
		return result, fmt.Errorf("inspect neo4j: %w", err)
	}
	if request.DryRun {
		result.Phase = "reading_snapshot"
		report(result.Phase, 0)
		err = snapshot.ReadNodePages(ctx, func(nodes []*graph.Node) error { result.NodeCount += len(nodes); return nil })
		if err == nil {
			err = snapshot.ReadEdgePages(ctx, func(edges []graph.ScopedEdgeRow) error { result.EdgeCount += len(edges); return nil })
		}
		if err != nil {
			result.Cancelled = errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
			return result, fmt.Errorf("read projection snapshot: %w", err)
		}
		result.Phase, result.Complete = "complete", true
		report(result.Phase, result.NodeCount+result.EdgeCount)
		return result, nil
	}
	result.Phase = "locking"
	report(result.Phase, 0)
	prior, err := transport.Acquire(ctx, request.Owner, request.OperationID, result.PendingGeneration)
	if err != nil {
		return result, fmt.Errorf("acquire projection owner: %w", err)
	}
	result.ActiveGeneration = prior
	result.Phase = "staging"
	report(result.Phase, 0)
	processed, lastReport := 0, time.Now()
	stage := func(nodes []*graph.Node, edges []graph.ScopedEdgeRow) error {
		for len(nodes)+len(edges) > 0 {
			batch := ProjectionBatch{Owner: request.Owner, OperationID: request.OperationID, PendingGeneration: result.PendingGeneration, SourceGeneration: descriptor.SourceGeneration}
			takeNodes := min(len(nodes), request.BatchSize)
			batch.Nodes, nodes = nodes[:takeNodes], nodes[takeNodes:]
			remaining := request.BatchSize - takeNodes
			takeEdges := min(len(edges), remaining)
			batch.Edges, edges = edges[:takeEdges], edges[takeEdges:]
			if err := transport.Stage(ctx, batch); err != nil {
				result.Cancelled = errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
				return err
			}
			previous := processed
			processed += takeNodes + takeEdges
			for milestone := (previous/1000 + 1) * 1000; milestone <= processed; milestone += 1000 {
				report(result.Phase, milestone)
				lastReport = time.Now()
			}
			if time.Since(lastReport) >= time.Second {
				report(result.Phase, processed)
				lastReport = time.Now()
			}
		}
		return nil
	}
	var pendingNodes []*graph.Node
	err = snapshot.ReadNodePages(ctx, func(page []*graph.Node) error {
		pendingNodes = append(pendingNodes, page...)
		for len(pendingNodes) >= request.BatchSize {
			if err := stage(pendingNodes[:request.BatchSize], nil); err != nil {
				return err
			}
			result.NodeCount += request.BatchSize
			pendingNodes = pendingNodes[request.BatchSize:]
		}
		return nil
	})
	if err == nil && len(pendingNodes) > 0 {
		err = stage(pendingNodes, nil)
		result.NodeCount += len(pendingNodes)
	}
	if err == nil {
		err = snapshot.ReadEdgePages(ctx, func(page []graph.ScopedEdgeRow) error {
			for len(page) > 0 {
				take := min(len(page), request.BatchSize)
				if err := stage(nil, page[:take]); err != nil {
					return err
				}
				result.EdgeCount += take
				page = page[take:]
			}
			return nil
		})
	}
	if err != nil {
		return result, fmt.Errorf("stage projection: %w", err)
	}
	result.Phase = "activating"
	report(result.Phase, processed)
	if err := transport.Activate(ctx, request.Owner, request.OperationID, result.PendingGeneration, prior, result); err != nil {
		return result, fmt.Errorf("activate projection: %w", err)
	}
	result.ActiveGeneration = result.PendingGeneration
	result.Complete = true
	result.Phase = "cleanup"
	report(result.Phase, processed)
	remaining, cleanupErr := transport.Reconcile(ctx, request.Owner, result.ActiveGeneration, request.BatchSize)
	result.StaleNodeCount = remaining.Nodes
	result.StaleEdgeCount = remaining.Relationships
	result.CleanupComplete = cleanupErr == nil && remaining.Nodes == 0 && remaining.Relationships == 0
	if result.CleanupComplete {
		result.CleanupStatus = "complete"
		result.Phase = "complete"
		report(result.Phase, processed)
		return result, nil
	}
	result.CleanupStatus = "incomplete"
	result.CleanupAction = "rerun the same projection command to resume exact-owner cleanup"
	result.Cancelled = errors.Is(cleanupErr, context.Canceled) || errors.Is(cleanupErr, context.DeadlineExceeded)
	if cleanupErr == nil {
		cleanupErr = ErrCleanupIncomplete
	}
	return result, cleanupErr
}

func generationKey(owner, operation string, sourceGeneration int64) string {
	return projectionGenerationKey(owner, operation, sourceGeneration)
}
