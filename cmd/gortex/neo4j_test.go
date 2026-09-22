package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zzet/gortex/internal/neo4jprojection"
)

func TestNeo4jPushFlags(t *testing.T) {
	cmd := newNeo4jPushCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "profile")

	cmd = newNeo4jPushCommand()
	cmd.SetArgs([]string{"--profile", "prod", "--namespace", "view", "--workspace", "ws", "--project", "p", "--repo", "a", "--repo", "b", "--batch-size", "250", "--operation-timeout", "2m", "--transaction-timeout", "10s", "--retry-timeout", "12s", "--dry-run"})
	var got map[string]any
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	withNeo4jPushTool(t, func(_ context.Context, _ string, _ string, args map[string]any) ([]byte, error) {
		got = args
		return json.Marshal(neo4jprojection.Result{Complete: true, DryRun: true, Phase: "complete"})
	})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "prod", got["profile"])
	require.Equal(t, "view", got["namespace"])
	require.Equal(t, "ws", got["workspace"])
	require.Equal(t, "p", got["project"])
	require.Equal(t, []string{"a", "b"}, got["repository"])
	require.Equal(t, 250, got["batch_size"])
	require.Equal(t, "2m0s", got["operation_timeout"])
	require.Equal(t, "10s", got["transaction_timeout"])
	require.Equal(t, "12s", got["retry_timeout"])
	require.Equal(t, true, got["dry_run"])

	cmd = newNeo4jPushCommand()
	cmd.SetArgs([]string{"unexpected"})
	require.Error(t, cmd.Execute())

	for _, flag := range []string{"--operation-timeout", "--transaction-timeout", "--retry-timeout"} {
		cmd = newNeo4jPushCommand()
		cmd.SetArgs([]string{"--profile", "prod", "--namespace", "view", "--workspace", "ws", "--project", "p", "--repo", "a", flag, "31m"})
		require.Error(t, cmd.Execute(), flag)
	}
}

func TestNeo4jPushJSON(t *testing.T) {
	cmd := newNeo4jPushCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--profile", "prod", "--namespace", "view", "--workspace", "ws", "--project", "p", "--repo", "a", "--json"})
	withNeo4jPushTool(t, func(_ context.Context, _ string, _ string, _ map[string]any) ([]byte, error) {
		return []byte(`{"profile":"prod","namespace":"view","phase":"complete","complete":true,"cleanup_complete":true,"node_count":2,"edge_count":1}`), nil
	})
	require.NoError(t, cmd.Execute())
	require.JSONEq(t, `{"profile":"prod","namespace":"view","phase":"complete","complete":true,"cleanup_complete":true,"node_count":2,"edge_count":1}`, stdout.String())
	require.Empty(t, stderr.String())
	require.NotContains(t, stdout.String(), "secret-canary")
}

func TestNeo4jPushProgress(t *testing.T) {
	cmd := newNeo4jPushCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--profile", "prod", "--namespace", "view", "--workspace", "ws", "--project", "p", "--repo", "a"})
	withNeo4jPushTool(t, func(_ context.Context, _ string, _ string, _ map[string]any) ([]byte, error) {
		return json.Marshal(neo4jprojection.Result{Profile: "prod", Namespace: "view", Phase: "complete", Complete: true, CleanupComplete: true, NodeCount: 1500, EdgeCount: 12})
	})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stderr.String(), "neo4j push: starting")
	require.Contains(t, stderr.String(), "phase=complete")
	require.Contains(t, stderr.String(), "records=1512")
	require.Contains(t, stdout.String(), "projected 1500 nodes and 12 edges")
}

func TestNeo4jPushIncomplete(t *testing.T) {
	cmd := newNeo4jPushCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--profile", "prod", "--namespace", "view", "--workspace", "ws", "--project", "p", "--repo", "a"})
	withNeo4jPushTool(t, func(ctx context.Context, _ string, _ string, _ map[string]any) ([]byte, error) {
		require.NotNil(t, ctx)
		return json.Marshal(neo4jprojection.Result{Profile: "prod", Namespace: "view", Phase: "activating", Complete: false, ErrorCode: "activation_failed", ErrorMessage: "activation failed", CleanupAction: "rerun the same projection command"})
	})
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, stderr.String(), "phase=activating")
	require.Contains(t, stderr.String(), "activation_failed")
	require.Contains(t, stderr.String(), "rerun the same projection command")
	require.NotContains(t, stderr.String(), "secret-canary")

	cmd = newNeo4jPushCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--profile", "prod", "--namespace", "view", "--workspace", "ws", "--project", "p", "--repo", "a"})
	withNeo4jPushTool(t, func(ctx context.Context, _ string, _ string, _ map[string]any) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	cmd.SetContext(ctx)
	require.ErrorIs(t, cmd.Execute(), context.DeadlineExceeded)
}

func withNeo4jPushTool(t *testing.T, fn func(context.Context, string, string, map[string]any) ([]byte, error)) {
	t.Helper()
	old := callNeo4jPushTool
	callNeo4jPushTool = fn
	t.Cleanup(func() { callNeo4jPushTool = old })
}
