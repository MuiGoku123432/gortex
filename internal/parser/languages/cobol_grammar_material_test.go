//go:build darwin || linux

package languages

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCobolGrammar_MaterialIdentity proves D-07: the identity the indexer
// fingerprints is built from exactly the values Extract stamps on a program,
// so the stamp and the invalidation trigger cannot drift apart.
func TestCobolGrammar_MaterialIdentity(t *testing.T) {
	e := NewCobolGrammarExtractor()
	identity := e.MaterialIdentity()
	assert.Contains(t, identity, cobolGrammarExtractorVersion)
	assert.Contains(t, identity, cobolApprovedGrammarID)
	assert.Contains(t, identity, cobolNameNormRule)

	programs := cobolProgramNodes(extractCobolGrammar(t, e, "src/demo.cbl", cobolDemoFixture))
	require.Len(t, programs, 1)
	meta := programs[0].Meta
	stamped := strings.Join([]string{
		meta["prov_parser_grammar_id"].(string),
		meta["prov_parser_module"].(string),
		meta["prov_preprocessor_module"].(string),
		meta["prov_extractor_version"].(string),
		cobolNameNormRule,
	}, "|")
	assert.Equal(t, stamped, identity)
}
