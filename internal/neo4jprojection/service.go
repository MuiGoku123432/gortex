package neo4jprojection

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/progress"
)

// Request identifies one exact projection owner and immutable source scope.
type Request struct {
	Profile            string                `json:"profile"`
	Namespace          string                `json:"namespace"`
	Workspace          string                `json:"workspace"`
	Project            string                `json:"project"`
	Repositories       []string              `json:"repository"`
	Owner              string                `json:"owner"`
	OperationID        string                `json:"operation_id"`
	Scope              graph.ProjectionScope `json:"scope"`
	BatchSize          int                   `json:"batch_size"`
	Timeout            time.Duration         `json:"-"`
	OperationTimeout   string                `json:"operation_timeout"`
	TransactionTimeout string                `json:"transaction_timeout"`
	RetryTimeout       string                `json:"retry_timeout"`
	DryRun             bool                  `json:"dry_run"`
}

// Result describes the source and activated target generation without secrets.
type Result struct {
	Profile           string `json:"profile"`
	Namespace         string `json:"namespace"`
	Owner             string `json:"owner"`
	OperationID       string `json:"operation_id"`
	PendingGeneration string `json:"pending_generation,omitempty"`
	ActiveGeneration  string `json:"active_generation,omitempty"`
	SourceGeneration  int64  `json:"source_generation"`
	NodeCount         int    `json:"node_count"`
	EdgeCount         int    `json:"edge_count"`
	StaleNodeCount    int    `json:"stale_node_count"`
	StaleEdgeCount    int    `json:"stale_edge_count"`
	Phase             string `json:"phase"`
	Complete          bool   `json:"complete"`
	CleanupComplete   bool   `json:"cleanup_complete"`
	CleanupStatus     string `json:"cleanup_status,omitempty"`
	CleanupAction     string `json:"cleanup_action,omitempty"`
	DryRun            bool   `json:"dry_run"`
	Cancelled         bool   `json:"cancelled"`
	ErrorCode         string `json:"error_code,omitempty"`
	ErrorMessage      string `json:"error_message,omitempty"`
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
	Abort(context.Context, string, string, string) error
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

// NormalizeRequest validates the public adapter contract and derives the
// stable exact-owner identity used for retries.
func NormalizeRequest(request Request) (Request, error) {
	request.Profile = strings.TrimSpace(request.Profile)
	request.Namespace = strings.TrimSpace(request.Namespace)
	request.Workspace = strings.TrimSpace(request.Workspace)
	request.Project = strings.TrimSpace(request.Project)
	for i := range request.Repositories {
		request.Repositories[i] = strings.TrimSpace(request.Repositories[i])
	}
	request.Repositories = slices.DeleteFunc(request.Repositories, func(repo string) bool { return repo == "" })
	sort.Strings(request.Repositories)
	request.Repositories = slices.Compact(request.Repositories)
	if request.Profile == "" || request.Namespace == "" || request.Workspace == "" || request.Project == "" || len(request.Repositories) == 0 {
		return Request{}, fmt.Errorf("neo4j profile, namespace, workspace, project, and at least one repository are required")
	}
	if request.BatchSize == 0 {
		request.BatchSize = defaultBatchSize
	}
	if request.BatchSize < 1 || request.BatchSize > defaultBatchSize {
		return Request{}, fmt.Errorf("neo4j projection batch size must be between 1 and %d", defaultBatchSize)
	}
	if request.OperationTimeout == "" {
		request.OperationTimeout = defaultTimeout.String()
	}
	if request.TransactionTimeout == "" {
		request.TransactionTimeout = (30 * time.Second).String()
	}
	if request.RetryTimeout == "" {
		request.RetryTimeout = (30 * time.Second).String()
	}
	operationTimeout, err := time.ParseDuration(request.OperationTimeout)
	if err != nil || operationTimeout <= 0 {
		return Request{}, fmt.Errorf("operation timeout must be a positive duration")
	}
	for label, value := range map[string]string{"transaction": request.TransactionTimeout, "retry": request.RetryTimeout} {
		d, parseErr := time.ParseDuration(value)
		if parseErr != nil || d <= 0 {
			return Request{}, fmt.Errorf("%s timeout must be a positive duration", label)
		}
	}
	request.Timeout = operationTimeout
	request.Scope = graph.ProjectionScope{
		Workspace: request.Workspace, Project: request.Project,
		Repositories: append([]string(nil), request.Repositories...),
	}
	request.Owner = projectionOwnerKey(request.Namespace, request.Workspace, request.Project, request.Repositories)
	if request.OperationID == "" {
		request.OperationID = request.Owner
	}
	return request, nil
}

const (
	defaultBatchSize = 500
	defaultTimeout   = 30 * time.Minute
)

func (s *Service) Push(ctx context.Context, request Request) (result Result, retErr error) {
	defer func() {
		if retErr == nil {
			return
		}
		if result.CleanupAction == "" {
			result.CleanupAction = "rerun the same projection command"
		}
		result.Cancelled = result.Cancelled || errors.Is(retErr, context.Canceled) || errors.Is(retErr, context.DeadlineExceeded)
		if result.Cancelled {
			result.ErrorCode, result.ErrorMessage = "cancelled", "neo4j projection cancelled"
		} else if errors.Is(retErr, ErrCleanupIncomplete) {
			result.ErrorCode, result.ErrorMessage = "cleanup_incomplete", "projection activated; exact-owner cleanup is incomplete"
		} else {
			result.ErrorCode, result.ErrorMessage = "projection_failed", "neo4j projection failed"
		}
	}()
	request.Owner = strings.TrimSpace(request.Owner)
	request.OperationID = strings.TrimSpace(request.OperationID)
	if request.Profile != "" || request.Namespace != "" || request.Workspace != "" || request.Project != "" || len(request.Repositories) > 0 {
		var err error
		request, err = NormalizeRequest(request)
		if err != nil {
			return result, err
		}
	}
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

	result = Result{Profile: request.Profile, Namespace: request.Namespace, Owner: request.Owner, OperationID: request.OperationID, Phase: "opening_snapshot", DryRun: request.DryRun}
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
	activated := false
	defer func() {
		if retErr != nil && !activated {
			retErr = errors.Join(retErr, transport.Abort(context.WithoutCancel(ctx), request.Owner, request.OperationID, result.PendingGeneration))
		}
	}()
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
	activated = true
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
