//go:build darwin || linux

package languages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// Every fixture in this file is invented 7-column fixed-format COBOL (D-14).
// None may be replaced with estate-derived source: this repository is public.

// cobolResourceProgramFixture names one DB2 table, one IDMS record, and one
// CICS transaction in lower case.
const cobolResourceProgramFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. RESPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           obtain calc emp-rec.
           EXEC CICS RETURN TRANSID('trn1') END-EXEC.
           EXEC SQL SELECT A INTO :WS-A FROM mysch.orders
           END-EXEC.
           GOBACK.
`

// cobolResourceCopybookFixture is a procedure member naming the same three
// resources in mixed case.
const cobolResourceCopybookFixture = "       RES-PARA.\n" +
	"           OBTAIN CALC Emp-Rec.\n" +
	"           EXEC CICS RETURN TRANSID('Trn1') END-EXEC.\n" +
	"           EXEC SQL SELECT A INTO :WS-A FROM MySch.Orders\n" +
	"           END-EXEC.\n"

// cobolResourceQualifierFixture names one table unqualified, two-part, and
// three-part.
const cobolResourceQualifierFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. QUALPGM.
       PROCEDURE DIVISION.
       P1.
           EXEC SQL SELECT A INTO :WS-A FROM ORDERS
           END-EXEC.
           EXEC SQL SELECT A INTO :WS-A FROM MYSCH.ORDERS
           END-EXEC.
           EXEC SQL SELECT A INTO :WS-A FROM LOC.MYSCH.ORDERS
           END-EXEC.
           GOBACK.
`

// cobolResourceDelimitedFixture names one table delimited and ordinary.
const cobolResourceDelimitedFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. DELIMPGM.
       PROCEDURE DIVISION.
       P1.
           EXEC SQL SELECT A INTO :WS-A FROM "MySch"."Orders"
           END-EXEC.
           EXEC SQL SELECT A INTO :WS-A FROM MYSCH.ORDERS
           END-EXEC.
           GOBACK.
`

// cobolSharedNodeView is the part of a shared resource node that must not
// depend on its writer (D-18).
type cobolSharedNodeView struct {
	ID, Name, FilePath, Language string
	Kind                         graph.NodeKind
	Meta                         map[string]any
}

// A shared DB2, IDMS, or CICS resource is the same node whichever document
// writes it, and never merges names that differ in more than ASCII case or
// in their DB2 qualifier (D-11, D-18, ID-04).
func TestCobolGrammar_SharedResource(t *testing.T) {
	e := NewCobolGrammarExtractor()

	t.Run("program_and_copybook", func(t *testing.T) {
		program := extractCobolGrammar(t, e, "src/respgm.cbl", cobolResourceProgramFixture)
		copybook := extractCobolGrammar(t, e, "copy/res.cpy", cobolResourceCopybookFixture)
		for _, tc := range []struct {
			id, programRaw, copybookRaw string
			kind                        graph.EdgeKind
		}{
			{"db2::MYSCH/ORDERS", "mysch.orders", "MySch.Orders", graph.EdgeReads},
			{"idms::_unqualified/record/EMP-REC", "emp-rec", "Emp-Rec", graph.EdgeReads},
			{"cics::transaction/TRN1", "'trn1'", "'Trn1'", graph.EdgeReferences},
		} {
			view := func(result *parser.ExtractionResult) cobolSharedNodeView {
				n := cobolNodeByID(t, result, tc.id)
				return cobolSharedNodeView{ID: n.ID, Name: n.Name, FilePath: n.FilePath, Language: n.Language, Kind: n.Kind, Meta: n.Meta}
			}
			fromProgram, fromCopybook := view(program), view(copybook)
			assert.Equal(t, fromProgram, fromCopybook, tc.id)
			assert.Equal(t, tc.id, fromProgram.FilePath)
			for result, raw := range map[*parser.ExtractionResult]string{program: tc.programRaw, copybook: tc.copybookRaw} {
				edges := cobolEdgesTo(result, tc.id, tc.kind)
				require.Len(t, edges, 1, tc.id)
				assert.Equal(t, raw, edges[0].Meta["cobol_resource_raw"], tc.id)
			}
		}
	})

	t.Run("qualified_and_unqualified", func(t *testing.T) {
		result := extractCobolGrammar(t, e, "src/qualpgm.cbl", cobolResourceQualifierFixture)
		assert.Equal(t, []string{"db2::_unqualified/ORDERS", "db2::MYSCH/ORDERS"}, cobolTableIDs(result))
		assertCobolTable(t, result, "db2::_unqualified/ORDERS", "ORDERS", "")
		assertCobolTable(t, result, "db2::MYSCH/ORDERS", "ORDERS", "MYSCH")

		unqualified := cobolEdgesTo(result, "db2::_unqualified/ORDERS", graph.EdgeReads)
		require.Len(t, unqualified, 1)
		assert.NotContains(t, unqualified[0].Meta, "sql_location")
		locations := map[string]any{}
		for _, edge := range cobolEdgesTo(result, "db2::MYSCH/ORDERS", graph.EdgeReads) {
			locations[edge.Meta["cobol_resource_raw"].(string)] = edge.Meta["sql_location"]
		}
		assert.Equal(t, map[string]any{"MYSCH.ORDERS": nil, "LOC.MYSCH.ORDERS": "LOC"}, locations)
	})

	t.Run("delimited_name_exact", func(t *testing.T) {
		result := extractCobolGrammar(t, e, "src/delimpgm.cbl", cobolResourceDelimitedFixture)
		assert.Equal(t, []string{`db2::"MySch"/"Orders"`, "db2::MYSCH/ORDERS"}, cobolTableIDs(result))
		assertCobolTable(t, result, `db2::"MySch"/"Orders"`, `"Orders"`, `"MySch"`)
		assertCobolTable(t, result, "db2::MYSCH/ORDERS", "ORDERS", "MYSCH")

		// Only ASCII letters fold; every other byte, and every ID
		// metacharacter after encoding, keeps two names apart.
		assert.Equal(t, "ÄMP-REC", cobolNameNorm("Ämp-rec"))
		assert.Equal(t, "A%2FB", cobolIDSegment("A/B"))
		assert.NotEqual(t, cobolIDSegment("A/B"), cobolIDSegment("A%2FB"))
		assert.Equal(t, "A%3AB%23C%25D%0A", cobolIDSegment("A:B#C%D\n"))
	})
}
