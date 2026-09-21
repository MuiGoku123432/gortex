package neo4jprojection

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	neo4j "github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j/config"
	gortexconfig "github.com/zzet/gortex/internal/config"
)

const (
	minimumNeo4jMajor       = 5
	minimumNeo4jMinor       = 26
	neo4jTransactionTimeout = 30 * time.Second
	neo4jRetryCeiling       = 30 * time.Second
)

type neo4jTransport struct {
	driver             neo4j.Driver
	database           string
	transactionTimeout time.Duration
	relationshipTypes  map[string]struct{}
}

func NewNeo4jTransport(profile gortexconfig.ResolvedNeo4jProfile) (Transport, error) {
	return NewNeo4jTransportWithTimeouts(profile, neo4jTransactionTimeout, neo4jRetryCeiling)
}

func NewNeo4jTransportWithTimeouts(profile gortexconfig.ResolvedNeo4jProfile, transactionTimeout, retryTimeout time.Duration) (Transport, error) {
	driver, err := neo4j.NewDriver(profile.URI, neo4j.BasicAuth(profile.Username, profile.Password, ""), func(cfg *config.Config) {
		cfg.MaxTransactionRetryTime = retryTimeout
	})
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver for profile %q", profile.Name)
	}
	return &neo4jTransport{driver: driver, database: profile.Database, transactionTimeout: transactionTimeout, relationshipTypes: make(map[string]struct{})}, nil
}

func (t *neo4jTransport) Inspect(ctx context.Context, dryRun bool) error {
	if err := t.driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("verify neo4j connectivity")
	}
	info, err := t.driver.GetServerInfo(ctx)
	if err != nil {
		return fmt.Errorf("inspect neo4j server")
	}
	version := strings.TrimPrefix(info.Agent(), "Neo4j/")
	parts := strings.SplitN(version, ".", 3)
	major, majorErr := strconv.Atoi(parts[0])
	minor := 0
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
	}
	if majorErr != nil || err != nil || major < minimumNeo4jMajor || (major == minimumNeo4jMajor && minor < minimumNeo4jMinor) {
		return fmt.Errorf("neo4j 5.26 or newer is required")
	}
	if dryRun {
		return t.execute(ctx, `SHOW CONSTRAINTS YIELD name RETURN collect(name) AS names`, nil)
	}
	for _, query := range constraintQueries(nil) {
		if err := t.execute(ctx, query, nil); err != nil {
			return classifyNeo4jError(err)
		}
	}
	return nil
}

func constraintQueries(relationshipTypes []string) []string {
	queries := []string{
		`CREATE CONSTRAINT gortex_node_physical IF NOT EXISTS FOR (n:GortexNode) REQUIRE n.gortex_physical_key IS UNIQUE`,
		`CREATE CONSTRAINT gortex_manifest_owner IF NOT EXISTS FOR (m:GortexProjectionManifest) REQUIRE m.gortex_owner IS UNIQUE`,
	}
	types := append([]string(nil), relationshipTypes...)
	sort.Strings(types)
	for _, relationshipType := range types {
		queries = append(queries, fmt.Sprintf(`CREATE CONSTRAINT gortex_rel_%s_physical IF NOT EXISTS FOR ()-[r:%s]-() REQUIRE r.gortex_physical_key IS UNIQUE`, strings.ToLower(relationshipType), relationshipType))
	}
	return queries
}

func (t *neo4jTransport) Acquire(ctx context.Context, owner, operation, generation string) (string, error) {
	result, err := t.query(ctx, `
MERGE (m:GortexProjectionManifest {gortex_owner: $owner})
ON CREATE SET m.active_generation = '', m.pending_generation = '', m.operation_id = ''
WITH m
WHERE m.operation_id = '' OR (m.operation_id = $operation AND m.pending_generation = $generation)
SET m.pending_generation = $generation, m.operation_id = $operation,
    m.pending_complete = false, m.cleanup_complete = false
RETURN m.active_generation AS active_generation`, map[string]any{"owner": owner, "operation": operation, "generation": generation})
	if err != nil {
		return "", classifyNeo4jError(err)
	}
	if len(result.Records) != 1 {
		return "", fmt.Errorf("projection owner %q already has an active operation", owner)
	}
	active, _ := result.Records[0].Get("active_generation")
	value, _ := active.(string)
	return value, nil
}

func (t *neo4jTransport) Stage(ctx context.Context, batch ProjectionBatch) error {
	nodesByLabel := make(map[string][]map[string]any)
	for _, node := range batch.Nodes {
		projected, _, err := projectNode(batch.Owner, batch.PendingGeneration, node)
		if err != nil {
			return err
		}
		label := projected.Labels[1]
		if len(projected.Labels) == 3 {
			label += ":GortexUnresolved"
		}
		nodesByLabel[label] = append(nodesByLabel[label], map[string]any{"physical": projected.Properties["gortex_physical_key"], "properties": projected.Properties})
	}
	labels := make([]string, 0, len(nodesByLabel))
	for label := range nodesByLabel {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		if err := t.execute(ctx, nodeMergeQuery(label), map[string]any{"rows": nodesByLabel[label]}); err != nil {
			return classifyNeo4jError(err)
		}
	}

	edgesByType := make(map[string][]map[string]any)
	for _, row := range batch.Edges {
		projected, _, err := projectEdge(batch.Owner, batch.PendingGeneration, row.Edge)
		if err != nil {
			return err
		}
		edgesByType[projected.Type] = append(edgesByType[projected.Type], map[string]any{
			"source":   projectionPhysicalKey(projectionNodeLogicalKey(batch.Owner, row.Edge.From), batch.PendingGeneration),
			"target":   projectionPhysicalKey(projectionNodeLogicalKey(batch.Owner, row.Edge.To), batch.PendingGeneration),
			"physical": projected.Properties["gortex_physical_key"], "properties": projected.Properties,
		})
	}
	types := make([]string, 0, len(edgesByType))
	for relationshipType := range edgesByType {
		types = append(types, relationshipType)
	}
	sort.Strings(types)
	for _, relationshipType := range types {
		if _, known := t.relationshipTypes[relationshipType]; !known {
			for _, query := range constraintQueries([]string{relationshipType})[2:] {
				if err := t.execute(ctx, query, nil); err != nil {
					return classifyNeo4jError(err)
				}
			}
			t.relationshipTypes[relationshipType] = struct{}{}
		}
		if err := t.execute(ctx, relationshipMergeQuery(relationshipType), map[string]any{"rows": edgesByType[relationshipType]}); err != nil {
			return classifyNeo4jError(err)
		}
	}
	return nil
}

func nodeMergeQuery(label string) string {
	return fmt.Sprintf("UNWIND $rows AS row\nMERGE (n:GortexNode:%s {gortex_physical_key: row.physical})\nSET n += row.properties", label)
}

func relationshipMergeQuery(relationshipType string) string {
	return fmt.Sprintf("UNWIND $rows AS row\nMATCH (source:GortexNode {gortex_physical_key: row.source})\nMATCH (target:GortexNode {gortex_physical_key: row.target})\nMERGE (source)-[r:%s {gortex_physical_key: row.physical}]->(target)\nSET r += row.properties", relationshipType)
}

func (t *neo4jTransport) Activate(ctx context.Context, owner, operation, generation, prior string, result Result) error {
	queryResult, err := t.query(ctx, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.pending_generation = $generation
  AND coalesce(m.active_generation, '') = $prior
SET m.active_generation = $generation, m.pending_generation = '', m.operation_id = '',
    m.source_generation = $source_generation, m.node_count = $node_count,
    m.edge_count = $edge_count, m.complete = true, m.cleanup_complete = false
RETURN m.active_generation AS active_generation`, map[string]any{
		"owner": owner, "operation": operation, "generation": generation, "prior": prior,
		"source_generation": result.SourceGeneration, "node_count": result.NodeCount, "edge_count": result.EdgeCount,
	})
	if err != nil {
		return classifyNeo4jError(err)
	}
	if len(queryResult.Records) != 1 {
		return fmt.Errorf("projection owner %q activation lost its lock or prior generation changed", owner)
	}
	return nil
}

func (t *neo4jTransport) Reconcile(ctx context.Context, owner, active string, batchSize int) (CleanupCounts, error) {
	for {
		result, err := t.query(ctx, `MATCH ()-[r {gortex_owner: $owner}]->() WHERE r.gortex_generation <> $active WITH r LIMIT $batch_size DELETE r RETURN count(r) AS deleted`, map[string]any{"owner": owner, "active": active, "batch_size": batchSize})
		if err != nil {
			return t.cleanupCensus(ctx, owner, active, classifyNeo4jError(err))
		}
		if recordCount(result, "deleted") == 0 {
			break
		}
	}
	for {
		result, err := t.query(ctx, `MATCH (n:GortexNode {gortex_owner: $owner}) WHERE n.gortex_generation <> $active AND NOT (n)--() WITH n LIMIT $batch_size DELETE n RETURN count(n) AS deleted`, map[string]any{"owner": owner, "active": active, "batch_size": batchSize})
		if err != nil {
			return t.cleanupCensus(ctx, owner, active, classifyNeo4jError(err))
		}
		if recordCount(result, "deleted") == 0 {
			break
		}
	}
	if err := t.execute(ctx, `MATCH (m:GortexProjectionManifest {gortex_owner: $owner, active_generation: $active}) SET m.cleanup_complete = true`, map[string]any{"owner": owner, "active": active}); err != nil {
		return t.cleanupCensus(ctx, owner, active, classifyNeo4jError(err))
	}
	return CleanupCounts{}, nil
}

func (t *neo4jTransport) cleanupCensus(ctx context.Context, owner, active string, cause error) (CleanupCounts, error) {
	result, err := t.query(context.WithoutCancel(ctx), `OPTIONAL MATCH (n:GortexNode {gortex_owner: $owner}) WHERE n.gortex_generation <> $active WITH count(n) AS nodes OPTIONAL MATCH ()-[r {gortex_owner: $owner}]->() WHERE r.gortex_generation <> $active RETURN nodes, count(r) AS relationships`, map[string]any{"owner": owner, "active": active})
	if err != nil || len(result.Records) != 1 {
		return CleanupCounts{}, errors.Join(ErrCleanupIncomplete, cause)
	}
	return CleanupCounts{Nodes: int(recordCount(result, "nodes")), Relationships: int(recordCount(result, "relationships"))}, errors.Join(ErrCleanupIncomplete, cause)
}

func recordCount(result *neo4j.EagerResult, key string) int64 {
	if result == nil || len(result.Records) == 0 {
		return 0
	}
	value, _ := result.Records[0].Get(key)
	count, _ := value.(int64)
	return count
}

func (t *neo4jTransport) Close(ctx context.Context) error { return t.driver.Close(ctx) }

func (t *neo4jTransport) query(ctx context.Context, query string, params map[string]any) (*neo4j.EagerResult, error) {
	return neo4j.ExecuteQuery(ctx, t.driver, query, params, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(t.database),
		neo4j.ExecuteQueryWithTransactionConfig(neo4j.WithTxTimeout(t.transactionTimeout)))
}

func (t *neo4jTransport) execute(ctx context.Context, query string, params map[string]any) error {
	_, err := t.query(ctx, query, params)
	return err
}

func classifyNeo4jError(err error) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return errors.New("neo4j projection operation failed")
}

func physicalKey(owner, generation, recordType, logical string) string {
	return generationKey(owner, recordType+"\x00"+logical, int64(len(generation))) + ":" + generation
}
