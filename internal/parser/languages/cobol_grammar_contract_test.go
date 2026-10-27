//go:build darwin || linux

package languages

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
)

// Every fixture in this file is invented 7-column fixed-format COBOL (P2
// D-14). None may be replaced with estate-derived source: this repository is
// public.

// cobolContractFixture touches every Phase 3 walker: sections, data items
// with a REDEFINES, COPY and COPY IDMS, PERFORM THRU, GO TO DEPENDING ON,
// literal and dynamic CALLs, IDMS, CICS, and embedded SQL with a cursor.
const cobolContractFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. CONTPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY CPYA.
       COPY IDMS RECORD EMP-REC.
       01 WS-REC.
          05 WS-ID PIC X(8).
          05 WS-CNT PIC 9(4) VALUE 0.
       01 WS-ALT REDEFINES WS-REC PIC X(12).
       01 WS-PGM PIC X(8).
       01 WS-IX PIC 9.
       01 WS-X PIC X(10).
       PROCEDURE DIVISION.
       S1 SECTION.
       MAIN-PARA.
           PERFORM STEP-A THRU STEP-B.
           GO TO STEP-C STEP-D DEPENDING ON WS-IX.
           CALL 'SUBPGM'.
           CALL WS-PGM.
           GOBACK.
       STEP-A.
           OBTAIN CALC EMP-REC.
           EXEC CICS SEND MAP('MAP1') MAPSET('MSET1') END-EXEC.
       STEP-B.
           EXEC SQL SELECT A INTO :WS-X FROM MYSCH.ORDERS AS O
           END-EXEC.
       STEP-C.
           EXEC SQL DECLARE C1 CURSOR FOR SELECT X FROM ORDHIST
           END-EXEC.
           EXEC SQL FETCH C1 INTO :WS-X END-EXEC.
       STEP-D.
           EXIT.
`

// cobolContractAmberFixture and cobolContractRedFixture are degraded
// documents (a MISSING node; an ERROR node) that still hold a section,
// paragraphs, statements, and data items.
const cobolContractAmberFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. AMBPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-REC.
          05 WS-A PIC X(4).
       01 WS-BAD PIC X(
       PROCEDURE DIVISION.
       S1 SECTION.
       MAIN-PARA.
           DISPLAY WS-A.
           PERFORM WORK-PARA.
       WORK-PARA.
           GOBACK.
`

const cobolContractRedFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. REDPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-REC.
          05 WS-A PIC X(4).
       PROCEDURE DIVISION.
       S1 SECTION.
       MAIN-PARA.
           DISPLAY WS-A.
           MOVE ))) TO (((.
           PERFORM WORK-PARA.
       WORK-PARA.
           GOBACK.
`

// cobolSecretKeyWords lists the substrings the Neo4j projection treats as
// secret-bearing.
// note: second occurrence of neo4jprojection.sensitiveMetadataKey's list;
// that function is unexported in another package, so it stays duplicated.
var cobolSecretKeyWords = []string{"password", "passwd", "secret", "token", "credential", "privatekey", "apikey", "authorization"}

// assertCobolMetaContract checks one Meta map's value types and key names.
func assertCobolMetaContract(t *testing.T, owner string, meta map[string]any) {
	t.Helper()
	normalize := strings.NewReplacer("-", "", "_", "", ".", "", " ", "")
	for key, value := range meta {
		switch value.(type) {
		case string, bool, int, float64, []string:
		default:
			assert.Failf(t, "unsupported Meta value type", "%s %s = %#v", owner, key, value)
		}
		normalized := normalize.Replace(strings.ToLower(key))
		for _, word := range cobolSecretKeyWords {
			assert.NotContains(t, normalized, word, "%s key %s looks secret-bearing", owner, key)
		}
	}
}

func TestCobolGrammar_ProvKeys(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/contract.cbl"
	result := extractCobolGrammar(t, e, path, cobolContractFixture)

	kinds := map[string]int{}
	for _, n := range result.Nodes {
		assertCobolMetaContract(t, n.ID, n.Meta)
		// Findings have their own contract: TestCobolGrammar_FindingMetaContract.
		if n.ID == path || n.Kind == graph.KindFinding {
			continue
		}
		kinds[string(n.Kind)]++
		if n.Meta["prov_range_absence"] == "shared_resource" {
			assertCobolSharedProvenance(t, n)
		} else {
			for _, key := range cobolDocumentKeys {
				assert.Contains(t, n.Meta, key, n.ID)
			}
		}
		assert.Equal(t, "gortex-cobol-grammar/6", n.Meta["prov_extractor_version"], n.ID)
	}
	// The fixture must reach every node-producing walker, or the loop above
	// proves less than it claims.
	for _, kind := range []graph.NodeKind{graph.KindFunction, graph.KindPackage, graph.KindLocal,
		graph.KindVariable, graph.KindField, graph.KindResource, graph.KindTable} {
		assert.Positive(t, kinds[string(kind)], "fixture emits no %s node", kind)
	}

	relations := map[graph.EdgeKind]int{}
	findings := cobolFindingIDs(result)
	for _, edge := range result.Edges {
		// Finding member_of edges: TestCobolGrammar_FindingMetaContract.
		if findings[edge.From] {
			continue
		}
		owner := string(edge.Kind) + " " + edge.From + " -> " + edge.To
		if edge.Kind == graph.EdgeDefines || edge.Kind == graph.EdgeMemberOf {
			assert.Nil(t, edge.Meta, "structural edge carries Meta: %s", owner)
			continue
		}
		if edge.Meta == nil {
			continue
		}
		relations[edge.Kind]++
		assertCobolMetaContract(t, owner, edge.Meta)
		for _, key := range append(append([]string{}, cobolDocumentKeys...),
			"prov_observation_kind", "prov_affected", "prov_range_exact") {
			assert.Contains(t, edge.Meta, key, owner)
		}
		assert.Equal(t, "gortex-cobol-grammar/6", edge.Meta["prov_extractor_version"], owner)
	}
	for _, kind := range []graph.EdgeKind{graph.EdgeCalls, graph.EdgeImports, graph.EdgeReads,
		graph.EdgeWrites, graph.EdgeReferences} {
		assert.Positive(t, relations[kind], "fixture emits no %s relationship edge", kind)
	}
}

func TestCobolGrammar_DegradedBreadth(t *testing.T) {
	e := NewCobolGrammarExtractor()
	for _, tc := range []struct{ name, path, src, grade string }{
		{"amber", "src/amber.cbl", cobolContractAmberFixture, "amber"},
		{"red", "src/red.cbl", cobolContractRedFixture, "red"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := extractCobolGrammar(t, e, tc.path, tc.src)
			require.Equal(t, tc.grade, cobolFileNode(t, result, tc.path).Meta["prov_document_grade"])

			// An observation is identified by its kind and original start byte.
			type obsKey struct {
				kind  string
				start int
			}
			affected := map[obsKey]bool{}
			for _, o := range cobolTestHandoff(t, e, tc.path, tc.src).Facts.Observations {
				if o.Original != nil {
					affected[obsKey{o.Kind, int(o.Original.Start.Byte)}] = o.Affected
				}
			}

			emitted := map[string]int{}
			for _, n := range result.Nodes {
				// Findings have their own contract: TestCobolGrammar_FindingMetaContract.
				if n.Kind == graph.KindFinding {
					continue
				}
				cobolKind, _ := n.Meta["cobol_kind"].(string)
				switch cobolKind {
				case "section", "paragraph", "statement", "data_item":
				default:
					continue
				}
				emitted[cobolKind]++
				assert.Equal(t, tc.grade, n.Meta["prov_document_grade"], n.ID)
				reasons, ok := n.Meta["prov_document_grade_reasons"].([]string)
				require.True(t, ok, "%s has no grade reasons", n.ID)
				assert.NotEmpty(t, reasons, n.ID)
				kind, _ := n.Meta["prov_observation_kind"].(string)
				start, ok := n.Meta["prov_start_byte"].(int)
				require.True(t, ok, "%s has no prov_start_byte", n.ID)
				want, found := affected[obsKey{kind, start}]
				require.True(t, found, "%s matches no %s observation at byte %d", n.ID, kind, start)
				assert.Equal(t, want, n.Meta["prov_affected"], n.ID)
			}
			for _, kind := range []string{"section", "paragraph", "statement", "data_item"} {
				assert.Positive(t, emitted[kind], "degraded document emits no %s node", kind)
			}
		})
	}
}
