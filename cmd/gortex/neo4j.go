package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zzet/gortex/internal/neo4jprojection"
)

var callNeo4jPushTool = func(ctx context.Context, repoPath, tool string, args map[string]any) ([]byte, error) {
	exec, err := resolveExecutorWithToolSurface(repoPath, tool, "defer")
	if err != nil {
		if errors.Is(err, ErrNoExecutor) {
			return nil, daemonRequiredErr(repoPath)
		}
		return nil, err
	}
	defer exec.Close()
	return exec.CallTool(ctx, tool, args)
}

func newNeo4jPushCommand() *cobra.Command {
	var profile, namespace, workspace, project string
	var repositories []string
	var batchSize int
	var operationTimeout, transactionTimeout, retryTimeout time.Duration
	var dryRun, jsonOutput bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Project an explicit SQLite scope into Neo4j",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			request, err := neo4jprojection.NormalizeRequest(neo4jprojection.Request{
				Profile: profile, Namespace: namespace, Workspace: workspace, Project: project,
				Repositories: repositories, BatchSize: batchSize, OperationTimeout: operationTimeout.String(),
				TransactionTimeout: transactionTimeout.String(), RetryTimeout: retryTimeout.String(), DryRun: dryRun,
			})
			if err != nil {
				return err
			}
			args := map[string]any{
				"profile": request.Profile, "namespace": request.Namespace, "workspace": request.Workspace,
				"project": request.Project, "repository": request.Repositories, "batch_size": request.BatchSize,
				"operation_timeout": request.OperationTimeout, "transaction_timeout": request.TransactionTimeout,
				"retry_timeout": request.RetryTimeout, "dry_run": request.DryRun, "format": "json",
			}
			if !jsonOutput {
				fmt.Fprintln(cmd.ErrOrStderr(), "neo4j push: starting")
			}
			raw, err := callNeo4jPushTool(cmd.Context(), ".", "neo4j_push", args)
			if err != nil {
				return err
			}
			var result neo4jprojection.Result
			if err := json.Unmarshal(raw, &result); err != nil {
				return fmt.Errorf("decode neo4j_push result: %w", err)
			}
			if jsonOutput {
				if _, err := cmd.OutOrStdout().Write(append(raw, '\n')); err != nil {
					return fmt.Errorf("write neo4j_push result: %w", err)
				}
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "neo4j push: phase=%s records=%d\n", result.Phase, result.NodeCount+result.EdgeCount)
				if result.Complete {
					fmt.Fprintf(cmd.OutOrStdout(), "projected %d nodes and %d edges", result.NodeCount, result.EdgeCount)
					if !result.CleanupComplete && result.CleanupStatus != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "; cleanup %s (%d nodes, %d relationships remain)", result.CleanupStatus, result.StaleNodeCount, result.StaleEdgeCount)
					}
					fmt.Fprintln(cmd.OutOrStdout())
				}
			}
			if !result.Complete || result.CleanupStatus == "incomplete" {
				if result.ErrorCode != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "neo4j push: %s: %s\n", result.ErrorCode, result.ErrorMessage)
				}
				if result.CleanupAction != "" {
					fmt.Fprintln(cmd.ErrOrStderr(), result.CleanupAction)
				}
				return &exitCodeError{code: 1, msg: "neo4j projection incomplete"}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "named Neo4j profile")
	cmd.Flags().StringVar(&namespace, "namespace", "", "projection namespace")
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace slug")
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringSliceVar(&repositories, "repo", nil, "repository prefix (repeatable)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 500, "maximum records per batch")
	cmd.Flags().DurationVar(&operationTimeout, "operation-timeout", 30*time.Minute, "whole-operation timeout")
	cmd.Flags().DurationVar(&transactionTimeout, "transaction-timeout", 30*time.Second, "Neo4j transaction timeout")
	cmd.Flags().DurationVar(&retryTimeout, "retry-timeout", 30*time.Second, "Neo4j managed retry timeout")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and plan without target writes")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "emit only the structured result")
	return cmd
}

func init() {
	neo4jCmd := &cobra.Command{Use: "neo4j", Short: "Neo4j projections"}
	neo4jCmd.AddCommand(newNeo4jPushCommand())
	rootCmd.AddCommand(neo4jCmd)
}
