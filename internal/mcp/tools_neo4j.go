package mcp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/neo4jprojection"
)

func (s *Server) registerNeo4jTools() {
	s.addTool(mcp.NewTool("neo4j_push",
		mcp.WithDescription("Project an explicitly scoped SQLite snapshot to a named Neo4j profile. Mutates Neo4j unless dry_run is true."),
		mcp.WithString("profile", mcp.Required(), mcp.Description("Named machine-level Neo4j profile")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Projection namespace")),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Exact workspace slug")),
		mcp.WithString("project", mcp.Required(), mcp.Description("Exact project name")),
		mcp.WithArray("repository", mcp.Required(), mcp.Description("One or more concrete repository prefixes"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithBoolean("dry_run", mcp.Description("Validate and plan without target writes")),
		mcp.WithNumber("batch_size", mcp.Description("Maximum records per batch (default 500)")),
		mcp.WithString("operation_timeout", mcp.Description("Whole-operation timeout (default 30m)")),
		mcp.WithString("transaction_timeout", mcp.Description("Neo4j transaction timeout (default 30s)")),
		mcp.WithString("retry_timeout", mcp.Description("Neo4j managed retry timeout (default 30s)")),
	), s.handleNeo4jPush)
}

func (s *Server) handleNeo4jPush(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	request, err := normalizeNeo4jPushRequest(req.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.multiIndexer != nil {
		resolved, scopeErr := s.resolveScopeForRequest(ctx, req, IntentAnalyze)
		if scopeErr != nil {
			return mcp.NewToolResultError(scopeErr.Error()), nil
		}
		if len(resolved.RepoAllow) == 0 {
			return mcp.NewToolResultError("neo4j push requires a concrete non-empty repository scope"), nil
		}
		resolvedRepos := make([]string, 0, len(resolved.RepoAllow))
		for repo, allowed := range resolved.RepoAllow {
			if allowed {
				resolvedRepos = append(resolvedRepos, repo)
			}
		}
		sort.Strings(resolvedRepos)
		if !sameNeo4jRepositories(request.Repositories, resolvedRepos) {
			return mcp.NewToolResultError("neo4j push repository scope does not resolve exactly to the requested repositories"), nil
		}
		if request.Workspace != resolved.WorkspaceID || request.Project != resolved.ProjectID {
			return mcp.NewToolResultError("neo4j push workspace/project scope does not match the active canonical scope"), nil
		}
		request.Repositories = resolvedRepos
		request.Scope = graph.ProjectionScope{Workspace: resolved.WorkspaceID, Project: resolved.ProjectID, Repositories: resolvedRepos}
	}

	push := s.neo4jPush
	if push == nil {
		if s.configManager == nil {
			return mcp.NewToolResultError("neo4j push: global configuration is not available"), nil
		}
		profile, profileErr := s.configManager.Global().ResolveNeo4jProfile(request.Profile)
		if profileErr != nil {
			return mcp.NewToolResultError(profileErr.Error()), nil
		}
		opener, ok := s.graph.(graph.ScopedProjectionSnapshotOpener)
		if !ok {
			return mcp.NewToolResultError("neo4j push: graph does not support immutable scoped snapshots"), nil
		}
		transactionTimeout, _ := time.ParseDuration(request.TransactionTimeout)
		retryTimeout, _ := time.ParseDuration(request.RetryTimeout)
		service := neo4jprojection.NewService(opener, func(context.Context) (neo4jprojection.Transport, error) {
			return neo4jprojection.NewNeo4jTransportWithTimeouts(profile,
				transactionTimeout, retryTimeout)
		})
		push = service.Push
	}

	result, pushErr := push(s.progressCtx(ctx, req), request)
	if pushErr != nil {
		if !errors.Is(pushErr, neo4jprojection.ErrCleanupIncomplete) {
			result.Complete = false
		}
		result.ErrorCode, result.ErrorMessage = classifyNeo4jPushError(pushErr)
		if result.CleanupAction == "" {
			result.CleanupAction = "rerun the same projection command"
		}
	}
	return s.respondJSONOrTOON(ctx, req, result)
}

func normalizeNeo4jPushRequest(args map[string]any) (neo4jprojection.Request, error) {
	repositories, err := stringListArg(args, "repository")
	if err != nil {
		return neo4jprojection.Request{}, err
	}
	return neo4jprojection.NormalizeRequest(neo4jprojection.Request{
		Profile: stringArg(args, "profile"), Namespace: stringArg(args, "namespace"),
		Workspace: stringArg(args, "workspace"), Project: stringArg(args, "project"), Repositories: repositories,
		BatchSize: intArgOrDefault(args, "batch_size", 500), OperationTimeout: stringArgOrDefault(args, "operation_timeout", "30m0s"),
		TransactionTimeout: stringArgOrDefault(args, "transaction_timeout", "30s"), RetryTimeout: stringArgOrDefault(args, "retry_timeout", "30s"),
		DryRun: boolArgValue(args, "dry_run"),
	})
}

func stringListArg(args map[string]any, key string) ([]string, error) {
	value, ok := args[key]
	if !ok {
		return nil, nil
	}
	switch values := value.(type) {
	case []string:
		return append([]string(nil), values...), nil
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("%s must contain only strings", key)
			}
			out = append(out, text)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}
}

func sameNeo4jRepositories(a, b []string) bool {
	return strings.Join(a, "\x00") == strings.Join(b, "\x00")
}

func classifyNeo4jPushError(err error) (string, string) {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "cancelled", "neo4j projection cancelled"
	case errors.Is(err, neo4jprojection.ErrCleanupIncomplete):
		return "cleanup_incomplete", "projection activated; exact-owner cleanup is incomplete"
	default:
		return "projection_failed", "neo4j projection failed"
	}
}
