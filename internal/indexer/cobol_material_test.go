//go:build darwin || linux

package indexer

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
	"github.com/zzet/gortex/internal/parser/languages"
)

// TestCobolMaterialFingerprint_Stable: the same identity and config always
// give the same fingerprint.
func TestCobolMaterialFingerprint_Stable(t *testing.T) {
	cfg := config.CobolConfig{ResolveFrom: []string{"other"}, DB2DefaultSchema: "MYSCH"}
	assert.Equal(t, cobolMaterialFingerprint("id", cfg), cobolMaterialFingerprint("id", cfg))
	assert.NotEqual(t, cobolMaterialFingerprint("id", cfg), cobolMaterialFingerprint("id2", cfg))
}

// TestCobolMaterialFingerprint_ConfigOnly: every index.cobol field reaches
// the fingerprint. The function takes only CobolConfig, so no other
// IndexConfig field can.
func TestCobolMaterialFingerprint_ConfigOnly(t *testing.T) {
	base := cobolMaterialFingerprint("id", config.CobolConfig{})
	for name, cfg := range map[string]config.CobolConfig{
		"resolve_from":           {ResolveFrom: []string{"other"}},
		"copybook_library":       {CopybookLibraries: []config.CobolCopybookLibrary{{Path: "copy"}}},
		"db2_default_schema":     {DB2DefaultSchema: "MYSCH"},
		"idms_default_subschema": {IDMSDefaultSubschema: "SUBS"},
	} {
		assert.NotEqual(t, base, cobolMaterialFingerprint("id", cfg), name)
	}
}

// TestCobolMaterialFingerprint_MerkleSalt: in Merkle mode a COBOL leaf salt
// carries the material fingerprint (D-06), so a config or identity change
// restages exactly the COBOL files; other languages keep their salt.
func TestCobolMaterialFingerprint_MerkleSalt(t *testing.T) {
	reg := parser.NewRegistry()
	languages.RegisterAll(reg)
	idx := New(graph.New(), reg, config.IndexConfig{}, zap.NewNop())

	fp, ok := idx.cobolMaterial()
	require.True(t, ok, "COBOL material fingerprint available")
	salt := idx.merkleSaltFor("cobol/a.cbl")
	assert.True(t, strings.HasSuffix(salt, "|cobol-material@"+strconv.FormatInt(fp, 10)), salt)
	assert.True(t, strings.HasPrefix(salt, merkleSaltFor("cobol/a.cbl")), salt)

	idx.SetCobolConfig(config.CobolConfig{DB2DefaultSchema: "MYSCH"})
	assert.NotEqual(t, salt, idx.merkleSaltFor("cobol/a.cbl"))
	assert.Equal(t, merkleSaltFor("a.go"), idx.merkleSaltFor("a.go"))
}

// TestCobolMaterialFingerprint_StaleLangs: a missing or different stored
// fingerprint marks cobol stale, an equal one does not, and the reserved key
// is never reported as a language.
func TestCobolMaterialFingerprint_StaleLangs(t *testing.T) {
	current := map[string]int{"go": 1, cobolMaterialKey: 42}
	assert.Equal(t, []string{"cobol"}, staleLangsBetween(map[string]int{"go": 1}, current))
	assert.Empty(t, staleLangsBetween(map[string]int{"go": 1, cobolMaterialKey: 42}, current))
	assert.Equal(t, []string{"cobol"}, staleLangsBetween(map[string]int{"go": 1, cobolMaterialKey: 7}, current))
	for _, stored := range []map[string]int{{"go": 1}, {"go": 1, cobolMaterialKey: 7}} {
		assert.NotContains(t, staleLangsBetween(stored, current), cobolMaterialKey)
	}
}
