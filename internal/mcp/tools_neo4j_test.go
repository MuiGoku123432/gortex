package mcp

import (
	"context"
	"encoding/json"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"github.com/zzet/gortex/internal/daemon"
	"github.com/zzet/gortex/internal/neo4jprojection"
)

func neo4jArgs() map[string]any {
	return map[string]any{
		"profile": "prod", "namespace": "view", "workspace": "ws", "project": "p",
		"repository": []any{"b", "a"}, "batch_size": 250.0, "operation_timeout": "2m0s",
		"transaction_timeout": "10s", "retry_timeout": "12s", "dry_run": true,
	}
}

func TestNeo4jParity(t *testing.T) {
	mcpRequest, err := normalizeNeo4jPushRequest(neo4jArgs())
	require.NoError(t, err)
	cliRequest, err := neo4jprojection.NormalizeRequest(neo4jprojection.Request{
		Profile: "prod", Namespace: "view", Workspace: "ws", Project: "p", Repositories: []string{"b", "a"},
		BatchSize: 250, OperationTimeout: "2m0s", TransactionTimeout: "10s", RetryTimeout: "12s", DryRun: true,
	})
	require.NoError(t, err)
	require.Equal(t, cliRequest, mcpRequest)

	result := neo4jprojection.Result{Profile: "prod", Namespace: "view", Complete: true, CleanupComplete: true, Phase: "complete", NodeCount: 2, EdgeCount: 1}
	cliJSON, err := json.Marshal(result)
	require.NoError(t, err)
	var got neo4jprojection.Request
	srv := &Server{neo4jPush: func(_ context.Context, request neo4jprojection.Request) (neo4jprojection.Result, error) {
		got = request
		return result, nil
	}}
	req := mcplib.CallToolRequest{}
	req.Params.Name = "neo4j_push"
	req.Params.Arguments = neo4jArgs()
	response, err := srv.handleNeo4jPush(context.Background(), req)
	require.NoError(t, err)
	require.False(t, response.IsError)
	require.Equal(t, cliRequest, got)
	text := response.Content[0].(mcplib.TextContent).Text
	require.JSONEq(t, string(cliJSON), text)
}

func TestNeo4jScope(t *testing.T) {
	_, err := normalizeNeo4jPushRequest(map[string]any{"profile": "prod", "namespace": "view", "workspace": "ws", "project": "p"})
	require.Error(t, err)
	_, err = normalizeNeo4jPushRequest(map[string]any{"profile": "prod", "namespace": "view", "workspace": "ws", "project": "p", "repository": []any{"", " "}})
	require.Error(t, err)
	_, err = stringListArg(map[string]any{"repository": []any{"a", 2}}, "repository")
	require.Error(t, err)
	for _, field := range []string{"operation_timeout", "transaction_timeout", "retry_timeout"} {
		args := neo4jArgs()
		args[field] = "999999999999999999999h"
		_, err = normalizeNeo4jPushRequest(args)
		require.Error(t, err, field)
	}
}

func TestNeo4jMutation(t *testing.T) {
	require.True(t, daemon.HasEffect("neo4j_push", daemon.EffectExternalWrite))
	request, err := normalizeNeo4jPushRequest(neo4jArgs())
	require.NoError(t, err)
	require.True(t, request.DryRun)
	args := neo4jArgs()
	delete(args, "dry_run")
	request, err = normalizeNeo4jPushRequest(args)
	require.NoError(t, err)
	require.False(t, request.DryRun)
	require.NotContains(t, args, "confirm")
}

func TestNeo4jCleanupIncompletePreservesActivatedSnapshot(t *testing.T) {
	result := neo4jprojection.Result{
		Profile:          "prod",
		Namespace:        "view",
		Phase:            "cleanup",
		Complete:         true,
		CleanupComplete:  false,
		CleanupStatus:    "incomplete",
		CleanupAction:    "rerun the same projection command",
		ActiveGeneration: "generation-new",
	}
	srv := &Server{neo4jPush: func(_ context.Context, _ neo4jprojection.Request) (neo4jprojection.Result, error) {
		return result, neo4jprojection.ErrCleanupIncomplete
	}}
	req := mcplib.CallToolRequest{}
	req.Params.Name = "neo4j_push"
	req.Params.Arguments = neo4jArgs()

	response, err := srv.handleNeo4jPush(context.Background(), req)
	require.NoError(t, err)

	var got neo4jprojection.Result
	text := response.Content[0].(mcplib.TextContent).Text
	require.NoError(t, json.Unmarshal([]byte(text), &got))
	require.True(t, got.Complete, "successful activation must remain complete when only physical cleanup is incomplete")
	require.False(t, got.CleanupComplete)
	require.Equal(t, result.ActiveGeneration, got.ActiveGeneration)
	require.Equal(t, result.CleanupAction, got.CleanupAction)
}

func TestNeo4jNoServer(t *testing.T) {
	srv, _ := setupTestServer(t)
	require.NotNil(t, srv)
	require.Nil(t, srv.neo4jPush, "server startup must not construct a Neo4j driver")

	req := mcplib.CallToolRequest{}
	req.Params.Name = "neo4j_push"
	req.Params.Arguments = map[string]any{"profile": "prod"}
	res, err := srv.handleNeo4jPush(context.Background(), req)
	require.NoError(t, err)
	require.True(t, res.IsError)
}
