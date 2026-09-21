package neo4jprojection

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNeo4jIntegrationGate(t *testing.T) {
	if os.Getenv("GORTEX_NEO4J_INTEGRATION") != "1" {
		t.Skip("disposable Neo4j gate is opt-in")
	}
	for _, scenario := range []string{
		"one_node_one_relationship_activation",
		"rerun",
		"stale_reconciliation",
		"owner_isolation",
		"pre_activation_failure",
		"cancellation",
		"dry_run",
		"shape",
		"redaction",
		"sqlite_fingerprints",
	} {
		t.Run(scenario, func(t *testing.T) {
			if scenario != "shape" {
				if os.Getenv("GORTEX_NEO4J_REQUIRE_SCENARIOS") == "1" {
					t.Fatal("mandatory Neo4j scenario is registered but not implemented")
				}
				t.Skip("registered for the production projection implementation")
			}
			for _, name := range []string{"GORTEX_NEO4J_URI", "GORTEX_NEO4J_USERNAME", "GORTEX_NEO4J_PASSWORD"} {
				if os.Getenv(name) == "" {
					t.Fatalf("integration environment missing %s", name)
				}
			}
		})
	}
}

func TestNeo4jScriptExactRunExecutesKnownTest(t *testing.T) {
	script := neo4jScriptPath(t)
	bin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "go.log")
	writeExecutable(t, filepath.Join(bin, "fake-go"), `#!/bin/sh
printf '%s\n' "$*" >> "$GORTEX_FAKE_GO_LOG"
case "$*" in
  *" -list ^TestNeo4jIntegrationGate$"*) printf '%s\n' TestNeo4jIntegrationGate ;;
esac
`)
	writeExecutable(t, filepath.Join(bin, "fake-docker"), `#!/bin/sh
case "$1" in
  run) printf '%s\n' fake-container-id ;;
  port) printf '%s\n' '127.0.0.1:17687' ;;
esac
exit 0
`)

	cmd := exec.Command("bash", script, "--timeout-seconds", "5", "--run", "TestNeo4jIntegrationGate")
	cmd.Env = append(os.Environ(),
		"GORTEX_NEO4J_TEST_GO="+filepath.Join(bin, "fake-go"),
		"GORTEX_NEO4J_TEST_DOCKER="+filepath.Join(bin, "fake-docker"),
		"GORTEX_FAKE_GO_LOG="+logPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("script failed: %v\n%s", err, output)
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	wantList := "test ./internal/neo4jprojection -list ^TestNeo4jIntegrationGate$"
	wantRun := "test ./internal/neo4jprojection -run ^TestNeo4jIntegrationGate$ -count=1"
	if !strings.Contains(string(log), wantList) || !strings.Contains(string(log), wantRun) || strings.Index(string(log), wantList) > strings.Index(string(log), wantRun) {
		t.Fatalf("expected anchored list then run, got:\n%s", log)
	}
}

func TestNeo4jScriptExactRunRejectsMissingTest(t *testing.T) {
	script := neo4jScriptPath(t)
	bin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "go.log")
	writeExecutable(t, filepath.Join(bin, "fake-go"), `#!/bin/sh
printf '%s\n' "$*" >> "$GORTEX_FAKE_GO_LOG"
exit 0
`)
	writeExecutable(t, filepath.Join(bin, "fake-docker"), `#!/bin/sh
printf '%s\n' invoked >> "$GORTEX_FAKE_DOCKER_LOG"
exit 0
`)
	dockerLog := filepath.Join(t.TempDir(), "docker.log")
	cmd := exec.Command("bash", script, "--timeout-seconds", "5", "--run", "TestDoesNotExist")
	cmd.Env = append(os.Environ(),
		"GORTEX_NEO4J_TEST_GO="+filepath.Join(bin, "fake-go"),
		"GORTEX_NEO4J_TEST_DOCKER="+filepath.Join(bin, "fake-docker"),
		"GORTEX_FAKE_GO_LOG="+logPath,
		"GORTEX_FAKE_DOCKER_LOG="+dockerLog,
	)
	if err := cmd.Run(); err == nil {
		t.Fatal("missing exact test was accepted")
	}
	if _, err := os.Stat(dockerLog); !os.IsNotExist(err) {
		t.Fatalf("docker was reached before exact-test validation: %v", err)
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(log), " -run ") {
		t.Fatalf("missing test reached execution: %s", log)
	}
}

func neo4jScriptPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate integration test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "scripts", "test-neo4j.sh"))
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
}
