package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCobolConfig_Validate pins the fail-closed rules for index.cobol: no
// wildcard or blank allowlist entry, repository-relative library paths, a
// library repository that the allowlist names, and library names and default
// qualifiers already in cobol-name-norm/1 form.
func TestCobolConfig_Validate(t *testing.T) {
	lib := func(name, repo, p string) []CobolCopybookLibrary {
		return []CobolCopybookLibrary{{Name: name, Repo: repo, Path: p}}
	}
	cases := []struct {
		name  string
		cfg   CobolConfig
		valid bool
	}{
		{"zero", CobolConfig{}, true},
		{"resolve_from", CobolConfig{ResolveFrom: []string{"repob"}}, true},
		{"resolve_from_star", CobolConfig{ResolveFrom: []string{"*"}}, false},
		{"resolve_from_question", CobolConfig{ResolveFrom: []string{"rep?"}}, false},
		{"resolve_from_class", CobolConfig{ResolveFrom: []string{"[ab]"}}, false},
		{"resolve_from_blank", CobolConfig{ResolveFrom: []string{" "}}, false},
		{"library_path", CobolConfig{CopybookLibraries: lib("", "", "copy/prod")}, true},
		{"library_path_empty", CobolConfig{CopybookLibraries: lib("", "", "")}, false},
		{"library_path_absolute", CobolConfig{CopybookLibraries: lib("", "", "/abs")}, false},
		{"library_path_up", CobolConfig{CopybookLibraries: lib("", "", "../up")}, false},
		{"library_path_inner_up", CobolConfig{CopybookLibraries: lib("", "", "copy/../x")}, false},
		{"library_path_wildcard", CobolConfig{CopybookLibraries: lib("", "", "copy/*")}, false},
		{"library_repo_not_allowed", CobolConfig{
			ResolveFrom: []string{"repob"}, CopybookLibraries: lib("", "repoc", "copy"),
		}, false},
		{"library_repo_allowed", CobolConfig{
			ResolveFrom: []string{"repob"}, CopybookLibraries: lib("", "repob", "copy"),
		}, true},
		{"library_name", CobolConfig{CopybookLibraries: lib("PRODLIB", "", "copy")}, true},
		{"library_name_lower", CobolConfig{CopybookLibraries: lib("prodlib", "", "copy")}, false},
		{"library_name_quoted", CobolConfig{CopybookLibraries: lib("'LIB'", "", "copy")}, false},
		{"library_name_slash", CobolConfig{CopybookLibraries: lib("A/B", "", "copy")}, false},
		{"db2_default", CobolConfig{DB2DefaultSchema: "MYSCH"}, true},
		{"db2_default_mixed_case", CobolConfig{DB2DefaultSchema: "MySch"}, false},
		{"idms_default_space", CobolConfig{IDMSDefaultSubschema: "EMP SS"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}

	t.Run("errors_joined", func(t *testing.T) {
		err := CobolConfig{ResolveFrom: []string{"*"}, DB2DefaultSchema: "MySch"}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolve_from")
		assert.Contains(t, err.Error(), "db2_default_schema")
		assert.Contains(t, err.Error(), "; ")
	})

	t.Run("global_schema", func(t *testing.T) {
		cfg := Default()
		cfg.Index.Cobol.ResolveFrom = []string{"*"}
		err := cfg.validateWorkspaceSchema()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "index.cobol")
	})
}

// TestCobolConfig_WorkspaceLoad proves index.cobol loads from a repository's
// .gortex.yaml and that an invalid block fails closed: a reload keeps the
// last good parse, and a first load leaves the defaults (no policy).
func TestCobolConfig_WorkspaceLoad(t *testing.T) {
	const valid = `index:
  cobol:
    resolve_from: [repob]
    copybook_libraries:
      - name: PRODLIB
        path: copy/prod
    db2_default_schema: MYSCH
`
	const invalid = `index:
  cobol:
    resolve_from: ["*"]
`
	newManager := func(t *testing.T) *ConfigManager {
		t.Helper()
		globalPath := filepath.Join(t.TempDir(), "config.yaml")
		require.NoError(t, os.WriteFile(globalPath, []byte("repos: []\n"), 0o600))
		cm, err := NewConfigManager(globalPath)
		require.NoError(t, err)
		return cm
	}
	write := func(t *testing.T, dir, body string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".gortex.yaml"), []byte(body), 0o600))
	}
	want := CobolConfig{
		ResolveFrom:       []string{"repob"},
		CopybookLibraries: []CobolCopybookLibrary{{Name: "PRODLIB", Path: "copy/prod"}},
		DB2DefaultSchema:  "MYSCH",
	}

	t.Run("valid", func(t *testing.T) {
		cm, dir := newManager(t), t.TempDir()
		write(t, dir, valid)
		cm.LoadWorkspaceConfig("repoa", dir)
		assert.Equal(t, want, cm.GetRepoConfig("repoa").Index.Cobol)
	})

	t.Run("invalid_keeps_last_good", func(t *testing.T) {
		cm, dir := newManager(t), t.TempDir()
		write(t, dir, valid)
		cm.LoadWorkspaceConfig("repoa", dir)
		write(t, dir, invalid)
		cm.LoadWorkspaceConfig("repoa", dir)
		assert.Equal(t, want, cm.GetRepoConfig("repoa").Index.Cobol)
	})

	t.Run("invalid_first_load", func(t *testing.T) {
		cm, dir := newManager(t), t.TempDir()
		write(t, dir, invalid)
		cm.LoadWorkspaceConfig("repoa", dir)
		got := cm.GetRepoConfig("repoa").Index.Cobol
		assert.Empty(t, got.ResolveFrom)
		assert.Empty(t, strings.TrimSpace(got.DB2DefaultSchema))
		assert.Equal(t, CobolConfig{}, got)
	})
}
