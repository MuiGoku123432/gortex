package neo4jprojection

import (
	"os"
	"strings"
	"testing"
)

func TestNeo4jLeastPrivilegeAndAcceptedRiskGuidance(t *testing.T) {
	data, err := os.ReadFile("../../SECURITY.md")
	if err != nil {
		t.Fatal(err)
	}
	policy := string(data)
	for _, required := range []string{
		"Neo4j projection least privilege",
		"dedicated Neo4j database",
		"does\n  not need DBMS administration",
		"GRANT ACCESS ON DATABASE gortex TO gortex_projection;",
		"GRANT MATCH {*} ON GRAPH gortex NODES *, RELATIONSHIPS * TO gortex_projection;",
		"GRANT CREATE NEW LABEL ON DATABASE gortex TO gortex_projection;",
		"GRANT CREATE NEW TYPE ON DATABASE gortex TO gortex_projection;",
		"GRANT CREATE NEW PROPERTY NAME ON DATABASE gortex TO gortex_projection;",
		"GRANT CONSTRAINT MANAGEMENT ON DATABASE gortex TO gortex_projection;",
		"`MERGE` is a query clause, not a grantable privilege",
		"CA-validated `neo4j+s` or `bolt+s`",
		"## Accepted risks",
		"high or critical findings remain release\nblockers",
		"Disposable Neo4j test principal has broad authority",
		"Pure metadata mapping performs no independent authorization",
		"CLI and MCP presentation adapters add no independent authorization",
	} {
		if !strings.Contains(policy, required) {
			t.Fatalf("SECURITY.md missing %q", required)
		}
	}
}
