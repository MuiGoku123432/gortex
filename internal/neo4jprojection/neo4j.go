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
	"github.com/zzet/gortex/internal/graph"
)

const (
	minimumNeo4jMajor       = 5
	minimumNeo4jMinor       = 26
	neo4jTransactionTimeout = 30 * time.Second
	neo4jRetryCeiling       = 30 * time.Second
)

type neo4jTransport struct {
	driver                neo4j.Driver
	database              string
	transactionTimeout    time.Duration
	leaseDuration         time.Duration
	relationshipTypes     map[string]struct{}
	beforeStageMutation   func(string)
	beforeCleanupMutation func(string)
	beforeCleanupRelease  func()
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
	leaseDuration := 2*(transactionTimeout+retryTimeout) + 5*time.Second
	return &neo4jTransport{driver: driver, database: profile.Database, transactionTimeout: transactionTimeout, leaseDuration: leaseDuration, relationshipTypes: make(map[string]struct{})}, nil
}

func (t *neo4jTransport) LeaseDuration() time.Duration { return t.leaseDuration }

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

func (t *neo4jTransport) Plan(ctx context.Context, owner string, intended IntendedPlan) (TargetPlan, error) {
	result, err := t.query(ctx, `
OPTIONAL MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WITH coalesce(m.active_generation, '') AS active_generation
OPTIONAL MATCH (n:GortexNode {gortex_owner: $owner})
WITH active_generation,
     count(n) AS physical_nodes,
     count(CASE WHEN NOT n.gortex_logical_key IN $node_keys THEN 1 END) AS logical_removed_nodes
OPTIONAL MATCH ()-[r {gortex_owner: $owner}]->()
RETURN active_generation, physical_nodes,
       count(r) AS physical_relationships,
       logical_removed_nodes,
       count(CASE WHEN NOT r.gortex_logical_key IN $edge_keys THEN 1 END) AS logical_removed_relationships`, map[string]any{"owner": owner, "node_keys": intended.NodeLogicalKeys, "edge_keys": intended.EdgeLogicalKeys})
	if err != nil {
		return TargetPlan{}, classifyNeo4jError(err)
	}
	if len(result.Records) != 1 {
		return TargetPlan{}, fmt.Errorf("inspect exact-owner target plan")
	}
	active, _ := result.Records[0].Get("active_generation")
	activeGeneration, _ := active.(string)
	constraints, err := t.query(ctx, `SHOW CONSTRAINTS YIELD name RETURN collect(name) AS names`, nil)
	if err != nil || len(constraints.Records) != 1 {
		return TargetPlan{}, classifyNeo4jError(err)
	}
	namesValue, _ := constraints.Records[0].Get("names")
	existing := make(map[string]struct{})
	if names, ok := namesValue.([]any); ok {
		for _, name := range names {
			if text, ok := name.(string); ok {
				existing[text] = struct{}{}
			}
		}
	}
	required := []string{"gortex_node_physical", "gortex_manifest_owner"}
	for _, relationshipType := range intended.RelationshipTypes {
		required = append(required, "gortex_rel_"+strings.ToLower(relationshipType)+"_physical")
	}
	missing := make([]string, 0)
	for _, name := range required {
		if _, ok := existing[name]; !ok {
			missing = append(missing, name)
		}
	}
	return TargetPlan{
		ActiveGeneration:    activeGeneration,
		StaleNodes:          int(recordCount(result, "physical_nodes")),
		StaleRelationships:  int(recordCount(result, "physical_relationships")),
		LogicalRemovedNodes: int(recordCount(result, "logical_removed_nodes")),
		LogicalRemovedEdges: int(recordCount(result, "logical_removed_relationships")),
		MissingConstraints:  missing,
	}, nil
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

func (t *neo4jTransport) Acquire(ctx context.Context, owner, operation, generation, attempt string, leaseUntil time.Time) (string, error) {
	result, err := t.query(ctx, `
MERGE (m:GortexProjectionManifest {gortex_owner: $owner})
ON CREATE SET m.active_generation = '', m.pending_generation = '', m.operation_id = '', m.operation_state = 'idle'
WITH m
WHERE m.operation_id = '' OR m.operation_state IN ['failed', 'abandoned']
   OR m.lease_until < datetime()
   OR (m.operation_id = $operation AND m.pending_generation = $generation AND m.attempt_id = $attempt)
SET m.pending_generation = $generation, m.operation_id = $operation, m.attempt_id = $attempt,
    m.operation_state = 'active', m.lease_until = datetime($lease_until),
    m.pending_complete = false, m.cleanup_complete = false
RETURN m.active_generation AS active_generation`, map[string]any{
		"owner": owner, "operation": operation, "generation": generation,
		"attempt": attempt, "lease_until": leaseUntil.UTC().Format(time.RFC3339Nano),
	})
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

func (t *neo4jTransport) Renew(ctx context.Context, owner, operation, generation, attempt string, leaseUntil time.Time) error {
	result, err := t.query(ctx, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.pending_generation = $generation
  AND m.attempt_id = $attempt AND m.operation_state = 'active'
  AND m.lease_until >= datetime()
SET m.lease_until = datetime($lease_until)
RETURN m.attempt_id AS attempt_id`, map[string]any{
		"owner": owner, "operation": operation, "generation": generation,
		"attempt": attempt, "lease_until": leaseUntil.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return classifyNeo4jError(err)
	}
	if len(result.Records) != 1 {
		return fmt.Errorf("projection owner %q lost its attempt lease", owner)
	}
	return nil
}

func (t *neo4jTransport) Abort(ctx context.Context, owner, operation, generation, attempt string, batchSize int) error {
	if err := t.execute(ctx, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.pending_generation = $generation AND m.attempt_id = $attempt
SET m.operation_state = 'failed', m.pending_complete = false
RETURN m.active_generation AS active_generation`, map[string]any{"owner": owner, "operation": operation, "generation": generation, "attempt": attempt}); err != nil {
		return classifyNeo4jError(err)
	}
	for {
		result, err := t.query(ctx, `MATCH (m:GortexProjectionManifest {gortex_owner: $owner}) WHERE coalesce(m.active_generation, '') <> $generation MATCH ()-[r {gortex_owner: $owner, gortex_generation: $generation, gortex_attempt: $attempt}]->() WITH r LIMIT $batch_size DELETE r RETURN count(r) AS deleted`, map[string]any{"owner": owner, "generation": generation, "attempt": attempt, "batch_size": batchSize})
		if err != nil {
			return classifyNeo4jError(err)
		}
		if recordCount(result, "deleted") == 0 {
			break
		}
	}
	for {
		result, err := t.query(ctx, `MATCH (m:GortexProjectionManifest {gortex_owner: $owner}) WHERE coalesce(m.active_generation, '') <> $generation MATCH (n:GortexNode {gortex_owner: $owner, gortex_generation: $generation, gortex_attempt: $attempt}) WHERE NOT (n)--() WITH n LIMIT $batch_size DELETE n RETURN count(n) AS deleted`, map[string]any{"owner": owner, "generation": generation, "attempt": attempt, "batch_size": batchSize})
		if err != nil {
			return classifyNeo4jError(err)
		}
		if recordCount(result, "deleted") == 0 {
			break
		}
	}
	return t.execute(ctx, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.pending_generation = $generation AND m.attempt_id = $attempt AND m.operation_state = 'failed'
SET m.operation_id = '', m.pending_generation = '', m.attempt_id = '', m.operation_state = 'idle'`, map[string]any{"owner": owner, "operation": operation, "generation": generation, "attempt": attempt})
}

func (t *neo4jTransport) Stage(ctx context.Context, batch ProjectionBatch) error {
	nodesByLabel := make(map[string][]map[string]any)
	knownNodes := make(map[string]struct{}, len(batch.Nodes))
	for _, node := range batch.Nodes {
		projected, _, err := projectNode(batch.Owner, batch.PendingGeneration, node)
		if err != nil {
			return err
		}
		label := projected.Labels[1]
		projected.Properties["gortex_attempt"] = batch.AttemptID
		if len(projected.Labels) == 3 {
			label += ":GortexUnresolved"
		}
		nodesByLabel[label] = append(nodesByLabel[label], map[string]any{"physical": projected.Properties["gortex_physical_key"], "properties": projected.Properties})
		knownNodes[node.ID] = struct{}{}
	}
	for _, row := range batch.Edges {
		for _, endpoint := range []*graph.Node{row.Source, row.Target} {
			if endpoint == nil {
				continue
			}
			if _, exists := knownNodes[endpoint.ID]; exists {
				continue
			}
			projected, _, err := projectNode(batch.Owner, batch.PendingGeneration, endpoint)
			if err != nil {
				return err
			}
			label := projected.Labels[1]
			projected.Properties["gortex_attempt"] = batch.AttemptID
			if len(projected.Labels) == 3 {
				label += ":GortexUnresolved"
			}
			nodesByLabel[label] = append(nodesByLabel[label], map[string]any{"physical": projected.Properties["gortex_physical_key"], "properties": projected.Properties})
			knownNodes[endpoint.ID] = struct{}{}
		}
		if row.Target == nil {
			unresolved := &graph.Node{ID: row.Edge.To, Kind: graph.NodeKind("unresolved"), Name: row.Edge.To, Meta: map[string]any{"synthetic": true}}
			projected, _, err := projectNode(batch.Owner, batch.PendingGeneration, unresolved)
			if err != nil {
				return err
			}
			projected.Properties["gortex_attempt"] = batch.AttemptID
			nodesByLabel[projected.Labels[1]+":GortexUnresolved"] = append(nodesByLabel[projected.Labels[1]+":GortexUnresolved"], map[string]any{"physical": projected.Properties["gortex_physical_key"], "properties": projected.Properties})
			knownNodes[unresolved.ID] = struct{}{}
		}
	}
	labels := make([]string, 0, len(nodesByLabel))
	for label := range nodesByLabel {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		if t.beforeStageMutation != nil {
			t.beforeStageMutation("node")
		}
		params := t.stageParams(batch)
		params["rows"] = nodesByLabel[label]
		result, err := t.query(ctx, nodeMergeQuery(label), params)
		if err != nil {
			return classifyNeo4jError(err)
		}
		if len(result.Records) != 1 {
			return fmt.Errorf("projection owner %q lost its attempt lease while staging nodes", batch.Owner)
		}
	}

	edgesByType := make(map[string][]map[string]any)
	for _, row := range batch.Edges {
		projected, _, err := projectEdge(batch.Owner, batch.PendingGeneration, row.Edge)
		if err != nil {
			return err
		}
		projected.Properties["gortex_attempt"] = batch.AttemptID
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
			if err := t.Renew(ctx, batch.Owner, batch.OperationID, batch.PendingGeneration, batch.AttemptID, time.Now().Add(t.leaseDuration)); err != nil {
				return err
			}
			for _, query := range constraintQueries([]string{relationshipType})[2:] {
				if err := t.execute(ctx, query, nil); err != nil {
					return classifyNeo4jError(err)
				}
			}
			t.relationshipTypes[relationshipType] = struct{}{}
		}
		if t.beforeStageMutation != nil {
			t.beforeStageMutation("relationship")
		}
		params := t.stageParams(batch)
		params["rows"] = edgesByType[relationshipType]
		result, err := t.query(ctx, relationshipMergeQuery(relationshipType), params)
		if err != nil {
			return classifyNeo4jError(err)
		}
		if len(result.Records) != 1 {
			return fmt.Errorf("projection owner %q lost its attempt lease while staging relationships", batch.Owner)
		}
	}
	return nil
}

func (t *neo4jTransport) stageParams(batch ProjectionBatch) map[string]any {
	return map[string]any{
		"owner": batch.Owner, "operation": batch.OperationID,
		"generation": batch.PendingGeneration, "attempt": batch.AttemptID,
		"lease_until": time.Now().Add(t.leaseDuration).UTC().Format(time.RFC3339Nano),
	}
}

func nodeMergeQuery(label string) string {
	return fmt.Sprintf("MATCH (m:GortexProjectionManifest {gortex_owner: $owner})\nWHERE m.operation_id = $operation AND m.pending_generation = $generation AND m.attempt_id = $attempt AND m.operation_state = 'active' AND m.lease_until >= datetime()\nSET m.lease_until = datetime($lease_until)\nWITH m\nUNWIND $rows AS row\nMERGE (n:GortexNode:%s {gortex_physical_key: row.physical})\nSET n += row.properties\nRETURN m.attempt_id AS attempt_id", label)
}

func relationshipMergeQuery(relationshipType string) string {
	return fmt.Sprintf("MATCH (m:GortexProjectionManifest {gortex_owner: $owner})\nWHERE m.operation_id = $operation AND m.pending_generation = $generation AND m.attempt_id = $attempt AND m.operation_state = 'active' AND m.lease_until >= datetime()\nSET m.lease_until = datetime($lease_until)\nWITH m\nUNWIND $rows AS row\nMATCH (source:GortexNode {gortex_physical_key: row.source})\nMATCH (target:GortexNode {gortex_physical_key: row.target})\nMERGE (source)-[r:%s {gortex_physical_key: row.physical}]->(target)\nSET r += row.properties\nRETURN m.attempt_id AS attempt_id", relationshipType)
}

func (t *neo4jTransport) MarkComplete(ctx context.Context, owner, operation, generation, attempt string, intended MaterializedCounts) (MaterializedCounts, error) {
	queryResult, err := t.query(ctx, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.pending_generation = $generation AND m.attempt_id = $attempt
  AND m.operation_state = 'active' AND m.lease_until >= datetime()
OPTIONAL MATCH (n:GortexNode {gortex_owner: $owner, gortex_generation: $generation})
WITH m, count(n) AS nodes
OPTIONAL MATCH ()-[r {gortex_owner: $owner, gortex_generation: $generation}]->()
WITH m, nodes, count(r) AS relationships
SET m.intended_node_count = $intended_nodes, m.intended_edge_count = $intended_relationships,
    m.observed_node_count = nodes, m.observed_edge_count = relationships,
    m.pending_complete = nodes = $intended_nodes AND relationships = $intended_relationships
WITH m, nodes, relationships
WHERE m.pending_complete
RETURN nodes, relationships`, map[string]any{"owner": owner, "operation": operation, "generation": generation, "attempt": attempt, "intended_nodes": intended.Nodes, "intended_relationships": intended.Relationships})
	if err != nil {
		return MaterializedCounts{}, classifyNeo4jError(err)
	}
	if len(queryResult.Records) != 1 {
		return MaterializedCounts{}, fmt.Errorf("projection owner %q lost its pending generation before completion", owner)
	}
	return MaterializedCounts{Nodes: int(recordCount(queryResult, "nodes")), Relationships: int(recordCount(queryResult, "relationships"))}, nil
}

func (t *neo4jTransport) Activate(ctx context.Context, owner, operation, generation, attempt, prior string, result Result) error {
	queryResult, err := t.query(ctx, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.pending_generation = $generation AND m.attempt_id = $attempt
  AND m.operation_state = 'active' AND m.lease_until >= datetime()
  AND m.pending_complete = true
  AND m.intended_node_count = $node_count AND m.intended_edge_count = $edge_count
  AND m.observed_node_count = $node_count AND m.observed_edge_count = $edge_count
  AND coalesce(m.active_generation, '') = $prior
SET m.active_generation = $generation, m.operation_state = 'cleanup',
    m.source_generation = $source_generation, m.node_count = $node_count,
    m.edge_count = $edge_count, m.complete = true, m.cleanup_complete = false
RETURN m.active_generation AS active_generation`, map[string]any{
		"owner": owner, "operation": operation, "generation": generation, "attempt": attempt, "prior": prior,
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

func (t *neo4jTransport) Reconcile(ctx context.Context, owner, operation, active, attempt string, batchSize int) (CleanupCounts, error) {
	params := func() map[string]any {
		return map[string]any{
			"owner": owner, "operation": operation, "active": active, "attempt": attempt,
			"lease_until": time.Now().Add(t.leaseDuration).UTC().Format(time.RFC3339Nano), "batch_size": batchSize,
		}
	}
	for {
		if t.beforeCleanupMutation != nil {
			t.beforeCleanupMutation("relationship")
		}
		result, err := t.query(ctx, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.active_generation = $active
  AND m.attempt_id = $attempt AND m.operation_state = 'cleanup'
  AND m.lease_until >= datetime()
SET m.lease_until = datetime($lease_until)
WITH m
CALL (m) {
  OPTIONAL MATCH ()-[r {gortex_owner: $owner}]->()
  WHERE r.gortex_generation <> $active
  WITH r LIMIT $batch_size
  DELETE r
  RETURN count(r) AS deleted
}
RETURN true AS fenced, deleted`, params())
		if err != nil {
			return t.cleanupCensus(ctx, owner, active, classifyNeo4jError(err))
		}
		deleted, fenceErr := cleanupDeleteResult(result, owner)
		if fenceErr != nil {
			return t.cleanupCensus(ctx, owner, active, fenceErr)
		}
		if deleted == 0 {
			break
		}
	}
	for {
		if t.beforeCleanupMutation != nil {
			t.beforeCleanupMutation("node")
		}
		result, err := t.query(ctx, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.active_generation = $active
  AND m.attempt_id = $attempt AND m.operation_state = 'cleanup'
  AND m.lease_until >= datetime()
SET m.lease_until = datetime($lease_until)
WITH m
CALL (m) {
  OPTIONAL MATCH (n:GortexNode {gortex_owner: $owner})
  WHERE n.gortex_generation <> $active AND NOT (n)--()
  WITH n LIMIT $batch_size
  DELETE n
  RETURN count(n) AS deleted
}
RETURN true AS fenced, deleted`, params())
		if err != nil {
			return t.cleanupCensus(ctx, owner, active, classifyNeo4jError(err))
		}
		deleted, fenceErr := cleanupDeleteResult(result, owner)
		if fenceErr != nil {
			return t.cleanupCensus(ctx, owner, active, fenceErr)
		}
		if deleted == 0 {
			break
		}
	}
	if t.beforeCleanupRelease != nil {
		t.beforeCleanupRelease()
	}
	result, err := t.query(ctx, `
MATCH (m:GortexProjectionManifest {gortex_owner: $owner})
WHERE m.operation_id = $operation AND m.active_generation = $active
  AND m.attempt_id = $attempt AND m.operation_state = 'cleanup'
  AND m.lease_until >= datetime()
OPTIONAL MATCH (n:GortexNode {gortex_owner: $owner})
WHERE n.gortex_generation <> $active
WITH m, count(n) AS nodes
OPTIONAL MATCH ()-[r {gortex_owner: $owner}]->()
WHERE r.gortex_generation <> $active
WITH m, nodes, count(r) AS relationships
FOREACH (_ IN CASE WHEN nodes = 0 AND relationships = 0 THEN [1] ELSE [] END |
  SET m.cleanup_complete = true, m.pending_generation = '', m.operation_id = '',
      m.attempt_id = '', m.operation_state = 'idle'
)
RETURN nodes, relationships, nodes = 0 AND relationships = 0 AS released`, map[string]any{
		"owner": owner, "operation": operation, "active": active, "attempt": attempt,
	})
	if err != nil {
		return t.cleanupCensus(ctx, owner, active, classifyNeo4jError(err))
	}
	if len(result.Records) != 1 {
		return t.cleanupCensus(ctx, owner, active, fmt.Errorf("projection owner %q lost its cleanup lease", owner))
	}
	releasedValue, _ := result.Records[0].Get("released")
	if released, _ := releasedValue.(bool); !released {
		return CleanupCounts{
			Nodes: int(recordCount(result, "nodes")), Relationships: int(recordCount(result, "relationships")),
		}, ErrCleanupIncomplete
	}
	return CleanupCounts{}, nil
}

func cleanupDeleteResult(result *neo4j.EagerResult, owner string) (int64, error) {
	if result == nil || len(result.Records) != 1 {
		return 0, fmt.Errorf("projection owner %q lost its cleanup lease", owner)
	}
	fencedValue, _ := result.Records[0].Get("fenced")
	if fenced, _ := fencedValue.(bool); !fenced {
		return 0, fmt.Errorf("projection owner %q lost its cleanup lease", owner)
	}
	return recordCount(result, "deleted"), nil
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
