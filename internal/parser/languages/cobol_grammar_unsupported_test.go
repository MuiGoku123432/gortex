//go:build !darwin && !linux

package languages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Platforms other than darwin and linux cannot link the enhanced COBOL
// parser, so the stub must fail every Extract instead of falling back to the
// regex or stock grammar (D-05, D-17), while still claiming COBOL files so
// they fail visibly.
func TestCobolGrammarUnsupportedPlatformStub(t *testing.T) {
	e := NewCobolGrammarExtractor()
	// Invented fixture (D-14).
	src := []byte("       IDENTIFICATION DIVISION.\n       PROGRAM-ID. DEMOPGM.\n")
	result, err := e.Extract("src/demo.cbl", src)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, errCobolParserUnsupportedPlatform)
	// The message is the only place a user on such a platform learns what
	// to do.
	assert.Contains(t, err.Error(), "index COBOL from macOS or Linux")

	// Same language and extension set as the darwin/linux extractor.
	assert.Equal(t, "cobol", e.Language())
	assert.ElementsMatch(t, []string{".cob", ".cbl", ".cpy", ".COB", ".CBL", ".CPY"}, e.Extensions())
}
