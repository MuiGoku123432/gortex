//go:build darwin || linux

package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/config"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
	"github.com/zzet/gortex/internal/parser/languages"
)

// cobolBOMFixture is an invented fixed-format program. It must never be
// replaced with estate-derived source: this repository is public.
const cobolBOMFixture = "       IDENTIFICATION DIVISION.\n" +
	"       PROGRAM-ID. DEMOPGM.\n" +
	"       PROCEDURE DIVISION.\n" +
	"       MAIN-PARA.\n" +
	"           STOP RUN.\n"

// TestCobolBOMByteOffsets covers the CR-01 residual (PROV-03) end to end:
// the always-on BOM strip runs before the real grammar extractor, so only
// the indexer can turn the program node's byte offsets into file offsets.
// The same program with and without a UTF-8 BOM must differ by exactly
// the three BOM bytes, while rows and lines stay BOM-excluded.
func TestCobolBOMByteOffsets(t *testing.T) {
	root := t.TempDir()
	writeRepoFile(t, root, "bom/demo.cbl", "\xEF\xBB\xBF"+cobolBOMFixture)
	writeRepoFile(t, root, "plain/demo.cbl", cobolBOMFixture)

	reg := parser.NewRegistry()
	reg.Register(languages.NewCobolGrammarExtractor())
	cfg := config.Default().Index
	cfg.Workers = 2
	g := graph.New()
	_, err := New(g, reg, cfg, zap.NewNop()).Index(root)
	require.NoError(t, err)

	bom := g.GetNode("bom/demo.cbl::DEMOPGM")
	plain := g.GetNode("plain/demo.cbl::DEMOPGM")
	require.NotNil(t, bom, "BOM-prefixed program node")
	require.NotNil(t, plain, "plain program node")

	for _, key := range []string{"prov_start_byte", "prov_end_byte"} {
		b, ok := bom.Meta[key].(int)
		require.True(t, ok, "bom %s is an int", key)
		p, ok := plain.Meta[key].(int)
		require.True(t, ok, "plain %s is an int", key)
		assert.Equal(t, p+3, b, "%s moves by exactly the BOM length", key)
	}
	assert.Equal(t, plain.Meta["prov_start_row"], bom.Meta["prov_start_row"])
	assert.Equal(t, plain.Meta["prov_end_row"], bom.Meta["prov_end_row"])
	assert.Equal(t, plain.StartLine, bom.StartLine)
	assert.Equal(t, plain.EndLine, bom.EndLine)
	assert.Equal(t, true, bom.Meta["prov_range_exact"])
	assert.Equal(t, true, plain.Meta["prov_range_exact"])
}
