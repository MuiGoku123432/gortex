//go:build windows

package languages

import (
	"errors"

	"github.com/zzet/gortex/internal/parser"
)

// errCobolParserUnavailableOnWindows explains why a Windows build cannot
// extract COBOL, and what to do about it.
var errCobolParserUnavailableOnWindows = errors.New(
	"the enhanced COBOL parser's secure filesystem layer supports only darwin and linux; " +
		"index COBOL from macOS or Linux")

// CobolGrammarExtractor is the Windows stand-in for the enhanced-grammar
// COBOL extractor.
//
// The enhanced grammar arrives through the tree-sitter-cobol-upgrade
// preprocessor, whose internal secure filesystem package builds only on
// darwin and linux, so a Windows binary cannot link it. Rather than fall back
// to the regex extractor or the stock grammar, which would persist facts from
// an unapproved parser, every Extract fails (D-05). The indexer then records
// each COBOL file as a parse failure instead of guessing its contents.
type CobolGrammarExtractor struct{}

// NewCobolGrammarExtractor returns the Windows stub extractor.
func NewCobolGrammarExtractor() *CobolGrammarExtractor {
	return &CobolGrammarExtractor{}
}

// Language returns "cobol".
func (e *CobolGrammarExtractor) Language() string { return "cobol" }

// Extensions returns the same COBOL extension set as the non-Windows
// extractor, so COBOL files still route here and fail visibly.
func (e *CobolGrammarExtractor) Extensions() []string {
	return []string{".cob", ".cbl", ".cpy", ".COB", ".CBL", ".CPY"}
}

// Extract always fails on Windows with errCobolParserUnavailableOnWindows.
func (e *CobolGrammarExtractor) Extract(_ string, _ []byte) (*parser.ExtractionResult, error) {
	return nil, errCobolParserUnavailableOnWindows
}

var _ parser.Extractor = (*CobolGrammarExtractor)(nil)
