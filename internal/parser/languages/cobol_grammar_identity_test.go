//go:build darwin || linux

package languages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// Every fixture in this file is invented 7-column fixed-format COBOL (P2
// D-14). None may be replaced with estate-derived source: this repository is
// public.

// cobolNestedPair is top-level program outer containing nested program
// inner; outer's paragraph runs body, inner's only GOBACKs.
func cobolNestedPair(outer, inner, body string) string {
	return "       IDENTIFICATION DIVISION.\n" +
		"       PROGRAM-ID. " + outer + ".\n" +
		"       PROCEDURE DIVISION.\n" +
		"       MAIN-PARA.\n" +
		body +
		"           GOBACK.\n" +
		cobolProgram(inner, inner) +
		"       END PROGRAM " + outer + ".\n"
}

// cobolProgramByID returns the program node with ID id.
func cobolProgramByID(t *testing.T, result *parser.ExtractionResult, id string) *graph.Node {
	t.Helper()
	for _, n := range cobolProgramNodes(result) {
		if n.ID == id {
			return n
		}
	}
	require.Failf(t, "no program node", "id %s in %v", id, cobolProgramIDs(result))
	return nil
}

// cobolHasNode reports whether result holds a node of kind with ID id.
func cobolHasNode(result *parser.ExtractionResult, id string, kind graph.NodeKind) bool {
	for _, n := range result.Nodes {
		if n.ID == id && n.Kind == kind {
			return true
		}
	}
	return false
}

// The pinned grammar keeps a literal PROGRAM-ID's quotes in program_name, so
// the literal 'A/B' is the ID component 'A%2FB' (quotes verbatim, D-20).
func TestCobolGrammar_ProgramIDEncoding(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/ident.cbl"

	t.Run("slash_literal", func(t *testing.T) {
		src := cobolProgram("'A/B'", "'A/B'") + cobolNestedPair("A", "B", "")
		result := extractCobolGrammar(t, e, path, src)
		assert.Equal(t, []string{path + "::'A%2FB'", path + "::A", path + "::A/B"}, cobolProgramIDs(result))
		literal := cobolProgramByID(t, result, path+"::'A%2FB'")
		assert.Equal(t, "'A/B'", literal.Name, "Node.Name keeps the PROGRAM-ID as written")
		assert.Equal(t, false, literal.Meta["cobol_nested"])
		assert.Equal(t, true, cobolProgramByID(t, result, path+"::A/B").Meta["cobol_nested"])
	})

	t.Run("hash_literal", func(t *testing.T) {
		src := cobolProgram("X", "X") + cobolProgram("X", "X") + cobolProgram("'X#2'", "'X#2'")
		result := extractCobolGrammar(t, e, path, src)
		assert.Equal(t, []string{path + "::X", path + "::X#2", path + "::'X%232'"}, cobolProgramIDs(result))
	})

	t.Run("paragraph_owner", func(t *testing.T) {
		src := cobolProgram("P1", "P1") + cobolProgram("P2", "P2") + cobolProgram("'A/B'", "'A/B'")
		result := extractCobolGrammar(t, e, path, src)
		for _, id := range []string{
			path + "::P1/@proc/MAIN-PARA",
			path + "::P2/@proc/MAIN-PARA",
			path + "::'A%2FB'/@proc/MAIN-PARA",
		} {
			assert.True(t, cobolHasNode(result, id, graph.KindFunction), "missing paragraph %s", id)
		}
	})

	t.Run("container_encoded", func(t *testing.T) {
		src := cobolNestedPair("'A/B'", "INNERPGM", "           CALL 'INNERPGM'.\n")
		result := extractCobolGrammar(t, e, path, src)
		const outer = path + "::'A%2FB'"
		inner := cobolProgramByID(t, result, outer+"/INNERPGM")
		assert.Equal(t, outer, inner.Meta["cobol_container"])
		call := cobolOneEdge(t, result, outer+"/@proc/MAIN-PARA", "unresolved::cobol::program::INNERPGM", graph.EdgeCalls)
		assert.Equal(t, outer, call.Meta["cobol_caller_program"])
	})
}
