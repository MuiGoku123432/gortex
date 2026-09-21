package config

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNeo4jTracerContractProfile(t *testing.T) {
	const usernameSecret = "neo4j-user-canary"
	const passwordSecret = "neo4j-password-canary"
	t.Setenv("GORTEX_TEST_NEO4J_USER", usernameSecret)
	t.Setenv("GORTEX_TEST_NEO4J_PASSWORD", passwordSecret)

	configPath := t.TempDir() + "/config.yaml"
	configYAML := `neo4j:
  production:
    uri: neo4j+s://graph.example.com
    database: estate
    username_env: GORTEX_TEST_NEO4J_USER
    password_env: GORTEX_TEST_NEO4J_PASSWORD
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	gc, err := LoadGlobal(configPath)
	if err != nil {
		t.Fatalf("load global config: %v", err)
	}
	profile, err := gc.ResolveNeo4jProfile("production")
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}
	if profile.URI != "neo4j+s://graph.example.com" || profile.Database != "estate" || profile.Username != usernameSecret || profile.Password != passwordSecret {
		t.Fatalf("unexpected resolved profile: %#v", profile)
	}
	encoded, err := yaml.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), usernameSecret) || strings.Contains(string(encoded), passwordSecret) {
		t.Fatalf("resolved credentials leaked through serialization: %s", encoded)
	}

	tests := []struct {
		name     string
		profiles map[string]Neo4jProfile
		profile  string
		want     string
	}{
		{name: "missing name", profiles: gc.Neo4j, want: "profile name is required"},
		{name: "unknown", profiles: gc.Neo4j, profile: "missing", want: `neo4j profile "missing" not found`},
		{name: "empty uri", profiles: map[string]Neo4jProfile{"bad": {Database: "neo4j", UsernameEnv: "U", PasswordEnv: "P"}}, profile: "bad", want: `neo4j profile "bad": uri is required`},
		{name: "uri userinfo", profiles: map[string]Neo4jProfile{"bad": {URI: "neo4j://secret:canary@example.com", Database: "neo4j", UsernameEnv: "U", PasswordEnv: "P"}}, profile: "bad", want: `neo4j profile "bad": uri must not contain userinfo`},
		{name: "empty database", profiles: map[string]Neo4jProfile{"bad": {URI: "neo4j://example.com", UsernameEnv: "U", PasswordEnv: "P"}}, profile: "bad", want: `neo4j profile "bad": database is required`},
		{name: "empty username ref", profiles: map[string]Neo4jProfile{"bad": {URI: "neo4j://example.com", Database: "neo4j", PasswordEnv: "P"}}, profile: "bad", want: `neo4j profile "bad": username_env is required`},
		{name: "empty password ref", profiles: map[string]Neo4jProfile{"bad": {URI: "neo4j://example.com", Database: "neo4j", UsernameEnv: "U"}}, profile: "bad", want: `neo4j profile "bad": password_env is required`},
		{name: "missing username value", profiles: map[string]Neo4jProfile{"bad": {URI: "neo4j://example.com", Database: "neo4j", UsernameEnv: "GORTEX_MISSING_USER", PasswordEnv: "GORTEX_TEST_NEO4J_PASSWORD"}}, profile: "bad", want: `neo4j profile "bad": environment variable GORTEX_MISSING_USER is empty`},
		{name: "missing password value", profiles: map[string]Neo4jProfile{"bad": {URI: "neo4j://example.com", Database: "neo4j", UsernameEnv: "GORTEX_TEST_NEO4J_USER", PasswordEnv: "GORTEX_MISSING_PASSWORD"}}, profile: "bad", want: `neo4j profile "bad": environment variable GORTEX_MISSING_PASSWORD is empty`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&GlobalConfig{Neo4j: tt.profiles}).ResolveNeo4jProfile(tt.profile)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("error = %q, want %q", err, tt.want)
			}
			if strings.Contains(err.Error(), usernameSecret) || strings.Contains(err.Error(), passwordSecret) || strings.Contains(err.Error(), "secret:canary") {
				t.Fatalf("error leaked credential material: %v", err)
			}
		})
	}
}
