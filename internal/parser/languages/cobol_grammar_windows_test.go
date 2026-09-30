//go:build windows

package languages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Windows cannot link the enhanced COBOL parser, so the stub must fail every
// Extract instead of falling back to the regex or stock grammar (D-05, D-17),
// while still claiming COBOL files so they fail visibly.
func TestCobolGrammarWindowsStub(t *testing.T) {
	e := NewCobolGrammarExtractor()
	// Invented fixture (D-14).
	src := []byte("       IDENTIFICATION DIVISION.\n       PROGRAM-ID. DEMOPGM.\n")
	result, err := e.Extract("src/demo.cbl", src)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, errCobolParserUnavailableOnWindows)
	// The message is the only place a Windows user learns what to do.
	assert.Contains(t, err.Error(), "index COBOL from macOS or Linux")

	// Same language and extension set as the non-Windows extractor.
	assert.Equal(t, "cobol", e.Language())
	assert.ElementsMatch(t, []string{".cob", ".cbl", ".cpy", ".COB", ".CBL", ".CPY"}, e.Extensions())
}
