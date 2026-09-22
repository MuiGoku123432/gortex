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
		"## Accepted risks",
		"high or critical findings remain release\nblockers",
	} {
		if !strings.Contains(policy, required) {
			t.Fatalf("SECURITY.md missing %q", required)
		}
	}
}
