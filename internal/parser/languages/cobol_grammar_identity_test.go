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

const cobolPlaceholderFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. PHPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CUSTREC.
-INC CUSTREC
           EXEC SQL INCLUDE CUSTREC END-EXEC.
       COPY IDMS RECORD EMP-REC.
       01 WS-PGM PIC X(8).
       PROCEDURE DIVISION.
       MAIN-PARA.
           CALL 'SUBPGM'.
           EXEC CICS LINK PROGRAM('SUBPGM') END-EXEC.
           EXEC CICS XCTL PROGRAM('SUBPGM') END-EXEC.
           CALL WS-PGM.
           GOBACK.
`

const cobolLibraryFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. LIBPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CUSTREC OF prodlib.
       COPY PLAINREC.
       PROCEDURE DIVISION.
       MAIN-PARA.
           GOBACK.
`

// cobolEdgesTo returns every edge of kind whose target is to.
func cobolEdgesTo(result *parser.ExtractionResult, to string, kind graph.EdgeKind) []*graph.Edge {
	var edges []*graph.Edge
	for _, e := range result.Edges {
		if e.To == to && e.Kind == kind {
			edges = append(edges, e)
		}
	}
	return edges
}

func TestCobolGrammar_Placeholder(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/ph.cbl"
	result := extractCobolGrammar(t, e, path, cobolPlaceholderFixture)

	t.Run("call", func(t *testing.T) {
		const program = "unresolved::cobol::program::SUBPGM"
		calls := cobolEdgesTo(result, program, graph.EdgeCalls)
		require.Len(t, calls, 3, "CALL, CICS LINK, and CICS XCTL")
		for _, edge := range calls {
			assert.Equal(t, program, edge.Meta["cobol_placeholder"], "line %d", edge.Line)
			assert.Equal(t, edge.To, edge.Meta["cobol_placeholder"], "line %d", edge.Line)
		}
	})

	t.Run("copy", func(t *testing.T) {
		const book = "unresolved::cobol::copybook::CUSTREC"
		forms := map[string]bool{}
		for _, edge := range cobolEdgesTo(result, book, graph.EdgeImports) {
			forms[edge.Meta["cobol_copy_form"].(string)] = true
			assert.Equal(t, book, edge.Meta["cobol_placeholder"], "form %v", edge.Meta["cobol_copy_form"])
		}
		assert.Equal(t, map[string]bool{"copy": true, "inc": true, "sql_include": true}, forms)
	})

	t.Run("never_bound", func(t *testing.T) {
		idms := cobolEdgesTo(result, "unresolved::cobol::copybook::idms/RECORD/EMP-REC", graph.EdgeImports)
		require.Len(t, idms, 1)
		assert.NotContains(t, idms[0].Meta, "cobol_placeholder")

		dynamic := cobolEdgesTo(result, "unresolved::cobol::dynamic::program::"+path+"::PHPGM::WS-PGM", graph.EdgeCalls)
		require.Len(t, dynamic, 1)
		assert.NotContains(t, dynamic[0].Meta, "cobol_placeholder")
	})

	t.Run("library_key", func(t *testing.T) {
		lib := extractCobolGrammar(t, e, "src/lib.cbl", cobolLibraryFixture)
		with := cobolEdgesTo(lib, "unresolved::cobol::copybook::CUSTREC", graph.EdgeImports)
		require.Len(t, with, 1)
		assert.Equal(t, "prodlib", with[0].Meta["cobol_copy_library"])
		assert.Equal(t, "PRODLIB", with[0].Meta["cobol_copy_library_key"])

		without := cobolEdgesTo(lib, "unresolved::cobol::copybook::PLAINREC", graph.EdgeImports)
		require.Len(t, without, 1)
		assert.NotContains(t, without[0].Meta, "cobol_copy_library")
		assert.NotContains(t, without[0].Meta, "cobol_copy_library_key")
	})
}

func TestCobolGrammar_Occurrences(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/occ.cbl"
	const target = "unresolved::cobol::program::SUBPGM"

	t.Run("distinct_lines", func(t *testing.T) {
		src := cobolNestedPair("OCCPGM", "INNERPGM", "           CALL 'SUBPGM'.\n           CALL 'SUBPGM'.\n")
		calls := cobolEdgesTo(extractCobolGrammar(t, e, path, src), target, graph.EdgeCalls)
		require.Len(t, calls, 2)
		assert.NotEqual(t, calls[0].Line, calls[1].Line)
		assert.NotEqual(t, calls[0].Meta["cobol_statement_symbol"], calls[1].Meta["cobol_statement_symbol"])
		assert.NotContains(t, calls[0].Meta, "cobol_occurrence_count")
	})

	t.Run("same_line", func(t *testing.T) {
		src := cobolNestedPair("OCCPGM", "INNERPGM", "           CALL 'SUBPGM'. CALL 'SUBPGM'.\n")
		result := extractCobolGrammar(t, e, path, src)
		calls := cobolEdgesTo(result, target, graph.EdgeCalls)
		require.Len(t, calls, 1)
		assert.Equal(t, 2, calls[0].Meta["cobol_occurrence_count"])
		symbols, ok := calls[0].Meta["cobol_statement_symbols"].([]string)
		require.True(t, ok, "cobol_statement_symbols = %#v", calls[0].Meta["cobol_statement_symbols"])
		require.Len(t, symbols, 2)
		assert.NotEqual(t, symbols[0], symbols[1])

		columns := map[int]bool{}
		for _, sym := range symbols {
			for _, n := range result.Nodes {
				if n.ID == path+"::"+sym {
					assert.Equal(t, graph.KindLocal, n.Kind, sym)
					columns[n.StartColumn] = true
				}
			}
		}
		assert.Len(t, columns, 2, "each statement node has its own start column")
	})
}
