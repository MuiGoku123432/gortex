package neo4jprojection

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	neo4j "github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/zzet/gortex/internal/config"
)

const minimumNeo4jMajor = 5

type neo4jTransport struct {
	driver   neo4j.Driver
	database string
}

// NewNeo4jTransport creates an invocation-local official-driver transport.
// Construction does not connect; Inspect performs the explicit connectivity and
// version checks before any mutation.
func NewNeo4jTransport(profile config.ResolvedNeo4jProfile) (Transport, error) {
	driver, err := neo4j.NewDriver(profile.URI, neo4j.BasicAuth(profile.Username, profile.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver for profile %q: %w", profile.Name, err)
	}
	return &neo4jTransport{driver: driver, database: profile.Database}, nil
}

func (t *neo4jTransport) Inspect(ctx context.Context) error {
	if err := t.driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("verify connectivity: %w", err)
	}
	info, err := t.driver.GetServerInfo(ctx)
	if err != nil {
		return fmt.Errorf("get server info: %w", err)
	}
	version := strings.TrimPrefix(info.Agent(), "Neo4j/")
	major, err := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
	if err != nil || major < minimumNeo4jMajor {
		return fmt.Errorf("neo4j 5.26 or newer is required (server %q)", info.Agent())
	}
	for _, query := range []string{
		`CREATE CONSTRAINT gortex_node_physical IF NOT EXISTS FOR (n:GortexNode) REQUIRE n.gortex_physical_key IS UNIQUE`,
		`CREATE CONSTRAINT gortex_manifest_owner IF NOT EXISTS FOR (m:GortexProjectionManifest) REQUIRE m.gortex_owner IS UNIQUE`,
	} {
		if err := t.execute(ctx, query, nil); err != nil {
			return err
		}
	}
	return nil
}

func (t *neo4jTransport) Acquire(ctx context.Context, owner, operation string) (string, error) {
	result, err := neo4j.ExecuteQuery(ctx, t.driver, `
MERGE (m:GortexProjectionManifest {gortex_owner: $owner})
ON CREATE SET m.active_generation = '', m.pending_generation = $operation,
              m.operation_id = $operation, m.complete = false, m.cleanup_complete = false
WITH m
WHERE m.pending_generation = '' OR m.pending_generation = $operation
SET m.pending_generation = $operation, m.operation_id = $operation,
    m.complete = false, m.cleanup_complete = false
RETURN m.active_generation AS active_generation`, map[string]any{"owner": owner, "operation": operation}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(t.database))
	if err != nil {
		return "", err
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
		nodesByLabel[label] = append(nodesByLabel[label], map[string]any{
			"physical":   projected.Properties["gortex_physical_key"],
			"properties": projected.Properties,
		})
	}
	labels := make([]string, 0, len(nodesByLabel))
	for label := range nodesByLabel {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		query := fmt.Sprintf(`
UNWIND $rows AS row
MERGE (n:GortexNode:%s {gortex_physical_key: row.physical})
SET n += row.properties`, label)
		if err := t.execute(ctx, query, map[string]any{"rows": nodesByLabel[label]}); err != nil {
			return err
		}
	}

	edgesByType := make(map[string][]map[string]any)
	for _, row := range batch.Edges {
		projected, _, err := projectEdge(batch.Owner, batch.PendingGeneration, row.Edge)
		if err != nil {
			return err
		}
		edgesByType[projected.Type] = append(edgesByType[projected.Type], map[string]any{
			"source":     projectionPhysicalKey(projectionNodeLogicalKey(batch.Owner, row.Edge.From), batch.PendingGeneration),
			"target":     projectionPhysicalKey(projectionNodeLogicalKey(batch.Owner, row.Edge.To), batch.PendingGeneration),
			"physical":   projected.Properties["gortex_physical_key"],
			"properties": projected.Properties,
		})
	}
	types := make([]string, 0, len(edgesByType))
	for relationshipType := range edgesByType {
		types = append(types, relationshipType)
	}
	sort.Strings(types)
	for _, relationshipType := range types {
		query := fmt.Sprintf(`
UNWIND $rows AS row
MATCH (source:GortexNode {gortex_physical_key: row.source})
MATCH (target:GortexNode {gortex_physical_key: row.target})
MERGE (source)-[r:%s {gortex_physical_key: row.physical}]->(target)
SET r += row.properties`, relationshipType)
		if err := t.execute(ctx, query, map[string]any{"rows": edgesByType[relationshipType]}); err != nil {
			return err
		}
	}
	return nil
}

func (t *neo4jTransport) Activate(ctx context.Context, owner, operation, generation, prior string, result Result) error {
	queryResult, err := neo4j.ExecuteQuery(ctx, t.driver, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.pending_generation = $operation
  AND coalesce(m.active_generation, '') = $prior
SET m.active_generation = $generation, m.pending_generation = '',
    m.source_generation = $source_generation, m.node_count = $node_count,
    m.edge_count = $edge_count, m.complete = true, m.cleanup_complete = false
RETURN m.active_generation AS active_generation`, map[string]any{
		"owner": owner, "operation": operation, "generation": generation, "prior": prior,
		"source_generation": result.SourceGeneration, "node_count": result.NodeCount, "edge_count": result.EdgeCount,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(t.database))
	if err != nil {
		return err
	}
	if len(queryResult.Records) != 1 {
		return fmt.Errorf("projection owner %q activation lost its lock or prior generation changed", owner)
	}
	return nil
}

func (t *neo4jTransport) Close(ctx context.Context) error { return t.driver.Close(ctx) }

func (t *neo4jTransport) execute(ctx context.Context, query string, params map[string]any) error {
	_, err := neo4j.ExecuteQuery(ctx, t.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(t.database))
	return err
}

func physicalKey(owner, generation, recordType, logical string) string {
	return generationKey(owner, recordType+"\x00"+logical, int64(len(generation))) + ":" + generation
}
