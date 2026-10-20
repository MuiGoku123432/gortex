//go:build darwin || linux

package languages

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/grade"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/handoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// Every fixture in this file is invented 7-column fixed-format COBOL (P2
// D-14). None may be replaced with estate-derived source: this repository is
// public.

const cobolFindingTailFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. TAILPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           EXEC SQL INSERT INTO T1 SELECT A, A FROM T2 END-EXEC.
           STOP RUN.
`

const cobolFindingErrorFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. ERRPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-B PIC 9(4).
       PROCEDURE DIVISION.
       MAIN-PARA.
           COMPUTE WS-B = (1 + 2.
           STOP RUN.
`

const cobolFindingMissingFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. T
       PROCEDURE DIVISION.
       MAIN-PARA.
           STOP RUN.
`

const cobolFindingLostFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. LOSTPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A PIC X(4).
           OBTAIN FIRST CUSTOMER-REC WITHIN CUST-ORDER-SET.
       01 WS-B PIC X(4).
       01 WS-C PIC X(4).
       PROCEDURE DIVISION.
       MAIN-PARA.
           STOP RUN.
`

const cobolFindingCleanFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SYNPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           STOP RUN.
`

const cobolFindingIncFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. INCPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
-INC CUSTREC
       PROCEDURE DIVISION.
       MAIN-PARA.
           STOP RUN.
`

// cobolFindingShiftFixture holds two SQL tails, ZQXCOL and ZQXOTH, and an
// ERROR, so a one-token edit has other findings to leave alone.
const cobolFindingShiftFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SHIFTPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-B PIC 9(4).
       PROCEDURE DIVISION.
       MAIN-PARA.
           EXEC SQL SELECT ZQXCOL FROM APP.ZQXTAB END-EXEC.
           EXEC SQL SELECT ZQXOTH FROM APP.ZQXTAB END-EXEC.
           COMPUTE WS-B = (1 + 2.
           STOP RUN.
`

// cobolFindingNodes returns the KindFinding nodes of result, restricted to
// finding_kind kind unless kind is empty.
func cobolFindingNodes(result *parser.ExtractionResult, kind string) []*graph.Node {
	var findings []*graph.Node
	for _, n := range result.Nodes {
		if n.Kind == graph.KindFinding && (kind == "" || n.Meta[graph.MetaFindingKind] == kind) {
			findings = append(findings, n)
		}
	}
	return findings
}

// cobolFindingIDs returns the set of KindFinding node IDs in result, so
// tests written before findings existed can tell finding nodes and their
// member_of edges apart.
func cobolFindingIDs(result *parser.ExtractionResult) map[string]bool {
	ids := map[string]bool{}
	for _, n := range cobolFindingNodes(result, "") {
		ids[n.ID] = true
	}
	return ids
}

// cobolProducerGaps counts the producer gaps of h with class and kind; an
// empty class or kind matches any.
func cobolProducerGaps(h handoff.Handoff, class handoff.ResolutionClass, kind handoff.Kind) int {
	count := 0
	for _, g := range h.Gaps {
		if (class == "" || g.ResolutionClass == class) && (kind == "" || g.Kind == kind) {
			count++
		}
	}
	return count
}

// assertCobolFindingMember checks that finding has a member_of edge to the
// node finding_container names, and that the node is in result.
func assertCobolFindingMember(t *testing.T, result *parser.ExtractionResult, finding *graph.Node) {
	t.Helper()
	container, _ := finding.Meta[graph.MetaFindingContainer].(string)
	require.NotEmpty(t, container, "%s has no finding_container", finding.ID)
	var present bool
	for _, n := range result.Nodes {
		if n.ID == container {
			present = true
		}
	}
	assert.True(t, present, "%s: container %s is not a node", finding.ID, container)
	assert.Len(t, cobolEdges(result, finding.ID, container, graph.EdgeMemberOf), 1,
		"%s has no member_of edge to %s", finding.ID, container)
}

func TestCobolGrammar_Findings_UnparsedTailPerToken(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/tail.cbl"
	result := extractCobolGrammar(t, e, path, cobolFindingTailFixture)
	h := cobolTestHandoff(t, e, path, cobolFindingTailFixture)

	tails := cobolFindingNodes(result, string(handoff.KindUnparsedTail))
	want := cobolProducerGaps(h, handoff.ResolutionParseUnresolved, handoff.KindUnparsedTail)
	require.GreaterOrEqual(t, want, 2, "fixture must leave several per-token tails")
	assert.Len(t, tails, want, "one finding per producer tail gap (D-18)")

	byQual := map[string]*graph.Node{}
	for _, n := range tails {
		assert.Equal(t, "low", n.Meta[graph.MetaFindingSeverity], n.ID)
		byQual[n.QualName] = n
	}
	var second *graph.Node
	for _, n := range tails {
		if strings.HasSuffix(n.QualName, "#2") {
			second = n
		}
	}
	require.NotNil(t, second, "identical tail text in one container gets #2")
	first := byQual[strings.TrimSuffix(second.QualName, "#2")]
	require.NotNil(t, first, "%s has no unsuffixed twin", second.QualName)
	assert.Equal(t, first.Meta[graph.MetaFindingContainer], second.Meta[graph.MetaFindingContainer])
	assert.Equal(t, first.Meta[graph.MetaFindingEvidenceDigest], second.Meta[graph.MetaFindingEvidenceDigest])
	assert.Less(t, first.Meta["prov_start_byte"].(int), second.Meta["prov_start_byte"].(int),
		"the ordinal follows source order")
}

func TestCobolGrammar_Findings_ErrorContainer(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/err.cbl", cobolFindingErrorFixture)

	errs := cobolFindingNodes(result, string(handoff.KindError))
	require.Len(t, errs, 1)
	assert.Equal(t, "medium", errs[0].Meta[graph.MetaFindingSeverity])
	assert.Equal(t, "localized", errs[0].Meta[graph.MetaFindingScope])
	assertCobolFindingMember(t, result, errs[0])
}

func TestCobolGrammar_Findings_MissingZeroWidth(t *testing.T) {
	e := NewCobolGrammarExtractor()
	result := extractCobolGrammar(t, e, "src/miss.cbl", cobolFindingMissingFixture)

	missing := cobolFindingNodes(result, string(handoff.KindMissing))
	require.Len(t, missing, 1)
	assert.Equal(t, ".", missing[0].Meta[graph.MetaFindingNodeKind])
	assert.Equal(t, graph.FindingEvidenceDigest(""), missing[0].Meta[graph.MetaFindingEvidenceDigest])
	assertCobolFindingMember(t, result, missing[0])
}

func TestCobolGrammar_Findings_LostCapture(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/lost.cbl"
	result := extractCobolGrammar(t, e, path, cobolFindingLostFixture)
	h := cobolTestHandoff(t, e, path, cobolFindingLostFixture)

	lost := cobolFindingNodes(result, string(handoff.KindLostCapture))
	want := cobolProducerGaps(h, handoff.ResolutionParseUnresolved, handoff.KindLostCapture)
	require.Positive(t, want, "fixture must lose captures")
	assert.Len(t, lost, want)
	for _, n := range lost {
		assert.NotEmpty(t, n.Meta[graph.MetaFindingExpectedFamily], n.ID)
		assert.Equal(t, "high", n.Meta[graph.MetaFindingSeverity], n.ID)
		assertCobolFindingMember(t, result, n)
	}
}

func TestCobolGrammar_Findings_DocumentScopeSynthetic(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/syn.cbl"
	extracted := extractCobolGrammar(t, e, path, cobolFindingCleanFixture)
	programIDs := cobolProgramIDs(extracted)
	require.Len(t, programIDs, 1)
	h := cobolTestHandoff(t, e, path, cobolFindingCleanFixture)
	require.Zero(t, cobolProducerGaps(h, "", ""), "the clean fixture must have no producer gaps")

	type synthetic struct {
		kind     handoff.Kind
		severity handoff.Severity
		scope    handoff.Scope
		absence  string
	}
	cases := []synthetic{
		{handoff.KindNoTree, handoff.SeverityHigh, handoff.ScopeDocument, "no-tree"},
		{handoff.KindBudgetExceeded, handoff.SeverityHigh, handoff.ScopeDocument, "parse-timeout"},
		{handoff.KindInvocationFailure, handoff.SeverityHigh, handoff.ScopeDocument, "invocation-failure"},
		{handoff.KindUnmappedCapture, handoff.SeverityHigh, handoff.ScopeUnavailable, "original-projection-unavailable"},
	}
	for _, c := range cases {
		h.Gaps = append(h.Gaps, handoff.Gap{
			ResolutionClass: handoff.ResolutionParseUnresolved, Kind: c.kind, Reason: c.absence,
			Severity: c.severity, Scope: c.scope, Container: -1, AbsenceReason: c.absence,
		})
	}
	programs, _, _ := cobolProgramSymbols(h)
	result := &parser.ExtractionResult{}
	docMeta := maps.Clone(cobolFileNode(t, extracted, path).Meta)
	cobolFindings(newCobolWalk(h, []byte(cobolFindingCleanFixture), path, docMeta, programs, result))

	require.Len(t, cobolFindingNodes(result, ""), len(cases))
	ids := map[string]bool{}
	for _, c := range cases {
		found := cobolFindingNodes(result, string(c.kind))
		require.Len(t, found, 1, "kind %s", c.kind)
		n := found[0]
		ids[n.ID] = true
		assert.Equal(t, programIDs[0], n.Meta[graph.MetaFindingContainer], n.ID)
		assert.Len(t, cobolEdges(result, n.ID, programIDs[0], graph.EdgeMemberOf), 1, n.ID)
		assert.Equal(t, false, n.Meta["prov_range_exact"], n.ID)
		assert.Equal(t, c.absence, n.Meta["prov_range_absence"], n.ID)
		assert.NotContains(t, n.Meta, "prov_start_row", n.ID)
		assert.NotContains(t, n.Meta, "prov_end_row", n.ID)
		assert.Equal(t, string(c.severity), n.Meta[graph.MetaFindingSeverity], n.ID)
		assert.Equal(t, string(c.scope), n.Meta[graph.MetaFindingScope], n.ID)
	}
	assert.Len(t, ids, len(cases), "document-scope findings stay distinct by kind")
}

func TestCobolGrammar_Findings_NoProducerExternal(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/inc.cbl"
	result := extractCobolGrammar(t, e, path, cobolFindingIncFixture)
	h := cobolTestHandoff(t, e, path, cobolFindingIncFixture)

	require.Equal(t, "green", cobolFileNode(t, result, path).Meta["prov_document_grade"])
	require.Equal(t, 1, cobolProducerGaps(h, handoff.ResolutionExternalUnresolved, ""),
		"the producer reports the -INC member as not obtained")
	assert.Empty(t, cobolFindingNodes(result, ""), "producer external-reference gaps never become findings (D-17)")
}

func TestCobolGrammar_Findings_LineShiftKeepsIDs(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/shift.cbl"
	ids := func(src string) map[string]bool {
		set := map[string]bool{}
		for _, n := range cobolFindingNodes(extractCobolGrammar(t, e, path, src), "") {
			set[n.ID] = true
		}
		return set
	}

	base := ids(cobolFindingShiftFixture)
	require.GreaterOrEqual(t, len(base), 3, "fixture must hold several findings")

	shifted := strings.Replace(cobolFindingShiftFixture, "       PROCEDURE DIVISION.\n",
		"      *\n      *\n      *\n       PROCEDURE DIVISION.\n", 1)
	require.NotEqual(t, cobolFindingShiftFixture, shifted)
	assert.Equal(t, base, ids(shifted), "line shifts keep every finding ID (GAP-06)")

	edited := ids(strings.Replace(cobolFindingShiftFixture, "ZQXCOL", "ZQXNEW", 1))
	require.Len(t, edited, len(base))
	var removed, added int
	for id := range base {
		if !edited[id] {
			removed++
		}
	}
	for id := range edited {
		if !base[id] {
			added++
		}
	}
	assert.Equal(t, 1, removed, "editing one gap's text changes only its finding ID")
	assert.Equal(t, 1, added, "editing one gap's text changes only its finding ID")
}

func TestCobolGrammar_FindingMetaContract(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/shift.cbl"
	result := extractCobolGrammar(t, e, path, cobolFindingShiftFixture)

	findings := cobolFindingNodes(result, "")
	require.NotEmpty(t, findings)
	required := []string{
		graph.MetaEvidenceClass, graph.MetaFindingKind, graph.MetaFindingReason,
		graph.MetaFindingSeverity, graph.MetaFindingSeverityRule, graph.MetaFindingScope,
		graph.MetaFindingContainer, graph.MetaFindingStatus, graph.MetaFindingEvidenceDigest,
		graph.MetaFindingEvidenceRule, graph.MetaFindingIDRule,
		"prov_parser_tool_id", "prov_parser_grammar_id", "prov_extractor_version", "prov_range_exact",
	}
	for _, n := range findings {
		assertCobolMetaContract(t, n.ID, n.Meta)
		for _, key := range required {
			assert.Contains(t, n.Meta, key, n.ID)
		}
		assert.Equal(t, graph.FindingClassParse, n.Meta[graph.MetaEvidenceClass], n.ID)
		assert.Equal(t, graph.FindingStatusActive, n.Meta[graph.MetaFindingStatus], n.ID)
		assert.Equal(t, n.Meta[graph.MetaFindingKind], n.Name, n.ID)
		if n.Meta["prov_range_exact"] == true {
			for _, key := range []string{"prov_start_row", "prov_start_column", "prov_end_row",
				"prov_end_column", "prov_start_byte", "prov_end_byte"} {
				assert.Contains(t, n.Meta, key, n.ID)
			}
		} else {
			assert.NotEmpty(t, n.Meta["prov_range_absence"], n.ID)
		}
		assert.NotContains(t, n.Meta, "cobol_kind", n.ID)
		assert.NotContains(t, n.Meta, "cobol_name_key", n.ID)
		assertCobolFindingMember(t, result, n)

		assert.NotContains(t, n.Name, "ZQXCOL")
		assert.NotContains(t, n.ID, "ZQXCOL")
		for key, value := range n.Meta {
			switch v := value.(type) {
			case string:
				assert.NotContains(t, v, "ZQXCOL", "%s meta %s carries source text", n.ID, key)
			case []string:
				for _, s := range v {
					assert.NotContains(t, s, "ZQXCOL", "%s meta %s carries source text", n.ID, key)
				}
			}
		}
	}
}

// cobolTestCascadeReasons is the D-13 cascade family, listed here
// independently of the extractor so a wrong family list fails these tests.
var cobolTestCascadeReasons = []string{
	grade.ReasonDivisionScaleRecovery, grade.ReasonRetainedStructureCollapse,
	grade.ReasonMissingRequiredAnchor, grade.ReasonLostCapture,
}

// cobolTestCascaded reports whether reasons hold a D-13 cascade reason.
func cobolTestCascaded(reasons []string) bool {
	return slices.ContainsFunc(reasons, func(r string) bool { return slices.Contains(cobolTestCascadeReasons, r) })
}

// cobolCascadeSource returns the invented 20-data-item, 20-paragraph control
// of cobolprobe TestErrorCascade with data inserted after the first data
// item and proc after the first paragraph (the probe's injection points).
// Two empty strings give the clean control.
func cobolCascadeSource(data, proc string) string {
	var b strings.Builder
	b.WriteString("       IDENTIFICATION DIVISION.\n       PROGRAM-ID. T.\n" +
		"       DATA DIVISION.\n       WORKING-STORAGE SECTION.\n")
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&b, "       01  WS-FIELD-%02d          PIC X(10).\n", i)
		if i == 1 {
			b.WriteString(data)
		}
	}
	b.WriteString("       PROCEDURE DIVISION.\n")
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&b, "       %04d-PARA.\n           MOVE 'X' TO WS-FIELD-01.\n", i)
		if i == 1 {
			b.WriteString(proc)
		}
	}
	b.WriteString("           STOP RUN.\n")
	return b.String()
}

// cobolCascadeShapes are the probe-verified uncontained shapes (05-RESEARCH
// fixture table), each injected into the 20/20 control.
var cobolCascadeShapes = []struct{ name, src string }{
	{"obtain_in_working_storage", cobolCascadeSource(
		"           OBTAIN FIRST CUSTOMER-REC WITHIN CUST-ORDER-SET.\n", "")},
	{"declare_cursor_in_working_storage", cobolCascadeSource(
		"           EXEC SQL DECLARE C1 CURSOR FOR SELECT A FROM T1 END-EXEC.\n", "")},
	{"declare_with_hold_in_procedure", cobolCascadeSource("",
		"           EXEC SQL DECLARE C2 CURSOR WITH HOLD FOR\n"+
			"               SELECT A FROM T1 END-EXEC.\n")},
	{"unknown_verb_in_procedure", cobolCascadeSource("", "           FROBNICATE THE WIDGET NOW.\n")},
	{"broken_data_clause", cobolCascadeSource("       01  BROKEN PIC X(10) GARBAGE CLAUSE HERE.\n", "")},
	{"data_division_without_period",
		strings.Replace(cobolCascadeSource("", ""), "DATA DIVISION.\n", "DATA DIVISION\n", 1)},
}

// cobolFindingCopybookFixture is an invented data member whose second item
// carries an unknown clause, which cascades under the copybook policy.
const cobolFindingCopybookFixture = "       01 CUST-REC.\n" +
	"           05 CUST-ID PIC X(8) WHATEVER.\n" +
	"           05 CUST-NAME PIC X(30).\n"

// cobolTestCopybookHandoff runs the extractor's own copybook analysis on src.
func cobolTestCopybookHandoff(t *testing.T, e *CobolGrammarExtractor, path, src string) handoff.Handoff {
	t.Helper()
	require.NotNil(t, e.analyzer, "extract once before reading the handoff")
	sourceID, err := preprocessor.NewSourceID(path)
	require.NoError(t, err)
	h, err := e.analyzer.AnalyzeCopybook(context.Background(), sourceID, []byte(src))
	require.NoError(t, err)
	return h
}

// assertCobolFindingsPerGap checks that result holds exactly one finding per
// producer PARSE_UNRESOLVED gap of h, kind by kind, plus cascades
// uncontained-cascade findings.
func assertCobolFindingsPerGap(t *testing.T, result *parser.ExtractionResult, h handoff.Handoff, cascades int) {
	t.Helper()
	kinds := map[handoff.Kind]bool{}
	for _, g := range h.Gaps {
		if g.ResolutionClass == handoff.ResolutionParseUnresolved {
			kinds[g.Kind] = true
		}
	}
	for kind := range kinds {
		assert.Len(t, cobolFindingNodes(result, string(kind)),
			cobolProducerGaps(h, handoff.ResolutionParseUnresolved, kind), "kind %s", kind)
	}
	assert.Len(t, cobolFindingNodes(result, "uncontained-cascade"), cascades)
	assert.Len(t, cobolFindingNodes(result, ""),
		cobolProducerGaps(h, handoff.ResolutionParseUnresolved, "")+cascades)
}

// assertCobolCascadeFinding checks the D-13 contract of one
// uncontained-cascade finding of a document graded with reasons, attached
// to container.
func assertCobolCascadeFinding(t *testing.T, result *parser.ExtractionResult, n *graph.Node, reasons []string, container string) {
	t.Helper()
	assert.Equal(t, "uncontained-cascade", n.Name, n.ID)
	assert.Equal(t, graph.FindingClassParse, n.Meta[graph.MetaEvidenceClass], n.ID)
	assert.Equal(t, "uncontained-cascade", n.Meta[graph.MetaFindingKind], n.ID)
	assert.Equal(t, "uncontained-cascade", n.Meta[graph.MetaFindingReason], n.ID)
	assert.Equal(t, "high", n.Meta[graph.MetaFindingSeverity], n.ID)
	assert.Equal(t, graph.FindingSeverityRule, n.Meta[graph.MetaFindingSeverityRule], n.ID)
	assert.Equal(t, "document", n.Meta[graph.MetaFindingScope], n.ID)
	assert.Equal(t, graph.FindingStatusActive, n.Meta[graph.MetaFindingStatus], n.ID)
	assert.Equal(t, graph.FindingEvidenceDigest("uncontained-cascade"), n.Meta[graph.MetaFindingEvidenceDigest], n.ID)
	assert.Equal(t, container, n.Meta[graph.MetaFindingContainer], n.ID)
	assert.Equal(t, false, n.Meta["prov_range_exact"], n.ID)
	assert.Equal(t, "document_scope", n.Meta["prov_range_absence"], n.ID)
	assert.NotContains(t, n.Meta, "prov_start_row", n.ID)
	assert.NotContains(t, n.Meta, graph.MetaFindingNodeKind, n.ID)
	assert.NotContains(t, n.Meta, graph.MetaFindingExpectedFamily, n.ID)
	assert.NotContains(t, n.Meta, "cobol_kind", n.ID)
	assert.Equal(t, reasons, n.Meta[graph.MetaFindingProducerReasons], n.ID)
	assertCobolMetaContract(t, n.ID, n.Meta)
	assertCobolFindingMember(t, result, n)
}

func TestCobolGrammar_Findings_CascadePositive(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/cascade.cbl"
	for _, tc := range cobolCascadeShapes {
		t.Run(tc.name, func(t *testing.T) {
			result := extractCobolGrammar(t, e, path, tc.src)
			h := cobolTestHandoff(t, e, path, tc.src)
			require.Equal(t, grade.Red, h.Assessment.Grade, "reasons %v", h.Assessment.Reasons)
			require.True(t, cobolTestCascaded(h.Assessment.Reasons), "reasons %v", h.Assessment.Reasons)
			programIDs := cobolProgramIDs(result)
			require.Len(t, programIDs, 1)

			cascades := cobolFindingNodes(result, "uncontained-cascade")
			require.Len(t, cascades, 1, "one uncontained-cascade finding per cascaded document")
			assertCobolCascadeFinding(t, result, cascades[0], h.Assessment.Reasons, programIDs[0])
			assertCobolFindingsPerGap(t, result, h, 1)

			// finding_producer_reasons is a fresh copy, not the file node's slice.
			file := cobolFileNode(t, result, path)
			reasons := cascades[0].Meta[graph.MetaFindingProducerReasons].([]string)
			reasons[0] = "mutated"
			assert.Equal(t, h.Assessment.Reasons, file.Meta["prov_document_grade_reasons"])
		})
	}
	// Each family reason alone is enough (D-13), on a clean document whose
	// producer assessment is replaced.
	extracted := extractCobolGrammar(t, e, "src/syn.cbl", cobolFindingCleanFixture)
	programIDs := cobolProgramIDs(extracted)
	require.Len(t, programIDs, 1)
	for _, reason := range cobolTestCascadeReasons {
		t.Run("family_"+reason, func(t *testing.T) {
			h := cobolTestHandoff(t, e, "src/syn.cbl", cobolFindingCleanFixture)
			h.Assessment.Grade = grade.Red
			h.Assessment.Reasons = []string{reason}
			programs, _, _ := cobolProgramSymbols(h)
			result := &parser.ExtractionResult{Nodes: []*graph.Node{cobolFileNode(t, extracted, "src/syn.cbl")}}
			result.Nodes = append(result.Nodes, cobolProgramNodes(extracted)...)
			docMeta := maps.Clone(cobolFileNode(t, extracted, "src/syn.cbl").Meta)
			cobolFindings(newCobolWalk(h, []byte(cobolFindingCleanFixture), "src/syn.cbl", docMeta, programs, result))
			cascades := cobolFindingNodes(result, "uncontained-cascade")
			require.Len(t, cascades, 1)
			assertCobolCascadeFinding(t, result, cascades[0], []string{reason}, programIDs[0])
		})
	}
}

func TestCobolGrammar_Findings_CascadeNegative(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/contained.cbl"
	for _, tc := range []struct{ name, src string }{
		{"clean_control", cobolCascadeSource("", "")},
		{"exec_sql_select_after_paragraph", cobolCascadeSource("",
			"           EXEC SQL SELECT ID FROM APP.CUSTOMER END-EXEC.\n")},
		{"unbalanced_compute", cobolCascadeSource("", "           COMPUTE WS-B = (1 + 2.\n")},
		{"exec_cics_send_map", cobolCascadeSource("", "           EXEC CICS SEND MAP('M') END-EXEC.\n")},
		{"obtain_after_paragraph", cobolCascadeSource("",
			"           OBTAIN FIRST CUSTOMER-REC WITHIN CUST-ORDER-SET.\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := extractCobolGrammar(t, e, path, tc.src)
			h := cobolTestHandoff(t, e, path, tc.src)
			require.NotEqual(t, grade.Red, h.Assessment.Grade, "reasons %v", h.Assessment.Reasons)
			require.False(t, cobolTestCascaded(h.Assessment.Reasons), "reasons %v", h.Assessment.Reasons)
			assertCobolFindingsPerGap(t, result, h, 0)
		})
	}
	// A red document whose reasons hold no family reason gets no cascade
	// finding (D-13 is the family, not the grade).
	extracted := extractCobolGrammar(t, e, "src/syn.cbl", cobolFindingCleanFixture)
	for _, reason := range []string{grade.ReasonParseTimeout, grade.ReasonNoTree, grade.ReasonUnmappedCapture} {
		t.Run("red_"+reason, func(t *testing.T) {
			h := cobolTestHandoff(t, e, "src/syn.cbl", cobolFindingCleanFixture)
			h.Assessment.Grade = grade.Red
			h.Assessment.Reasons = []string{reason, grade.ReasonErrorNode, grade.ReasonUnparsedTail}
			programs, _, _ := cobolProgramSymbols(h)
			result := &parser.ExtractionResult{}
			docMeta := maps.Clone(cobolFileNode(t, extracted, "src/syn.cbl").Meta)
			cobolFindings(newCobolWalk(h, []byte(cobolFindingCleanFixture), "src/syn.cbl", docMeta, programs, result))
			assert.Empty(t, cobolFindingNodes(result, ""))
		})
	}
}

func TestCobolGrammar_Findings_CascadeIDStable(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "src/cascade.cbl"
	cascade := func(src string) (*graph.Node, []string) {
		result := extractCobolGrammar(t, e, path, src)
		h := cobolTestHandoff(t, e, path, src)
		cascades := cobolFindingNodes(result, "uncontained-cascade")
		require.Len(t, cascades, 1)
		return cascades[0], h.Assessment.Reasons
	}
	a, reasonsA := cascade(cobolCascadeShapes[0].src)
	b, reasonsB := cascade(cobolCascadeShapes[len(cobolCascadeShapes)-1].src)
	require.NotEqual(t, reasonsA, reasonsB, "the two shapes must grade with different reasons")
	require.Equal(t, a.Meta[graph.MetaFindingContainer], b.Meta[graph.MetaFindingContainer],
		"both shapes attach to the same container")
	assert.Equal(t, a.ID, b.ID, "producer reasons never enter the ID (D-01, D-02)")
	assert.Equal(t, reasonsA, a.Meta[graph.MetaFindingProducerReasons])
	assert.Equal(t, reasonsB, b.Meta[graph.MetaFindingProducerReasons])
}

func TestCobolGrammar_Findings_Copybook(t *testing.T) {
	e := NewCobolGrammarExtractor()
	const path = "copy/custbad.cpy"
	result := extractCobolGrammar(t, e, path, cobolFindingCopybookFixture)
	assertCobolCopybookFile(t, result, path, cobolFindingCopybookFixture)
	h := cobolTestCopybookHandoff(t, e, path, cobolFindingCopybookFixture)
	require.True(t, cobolTestCascaded(h.Assessment.Reasons), "reasons %v", h.Assessment.Reasons)
	require.Positive(t, cobolProducerGaps(h, handoff.ResolutionParseUnresolved, ""))

	assertCobolFindingsPerGap(t, result, h, 1)
	cascades := cobolFindingNodes(result, "uncontained-cascade")
	require.Len(t, cascades, 1)
	assertCobolCascadeFinding(t, result, cascades[0], h.Assessment.Reasons, path)
	for _, n := range cobolFindingNodes(result, "") {
		assert.True(t, strings.HasPrefix(n.ID, path+"::@finding/parse/"), n.ID)
		assert.Equal(t, path, n.Meta[graph.MetaFindingContainer], n.ID)
		assert.Len(t, cobolEdges(result, n.ID, path, graph.EdgeMemberOf), 1, n.ID)
		assert.NotContains(t, n.Meta, "cobol_kind", n.ID)
		assert.NotContains(t, n.Meta, "cobol_name_key", n.ID)
		assert.NotContains(t, n.Meta, "prov_analysis_absence", n.ID)
		assert.Equal(t, "cobol-grade-policy-copybook-v1", n.Meta["prov_grade_policy"], n.ID)
		assertCobolMetaContract(t, n.ID, n.Meta)
	}
}

func TestCobolGrammar_Findings_CopybookUnmappedFlag(t *testing.T) {
	e := NewCobolGrammarExtractor()
	t.Run("unmapped_member", func(t *testing.T) {
		const path = "copy/cicsret.cpy"
		result := extractCobolGrammar(t, e, path, cobolCopybookExecCICSFixture)
		file := assertCobolCopybookFile(t, result, path, cobolCopybookExecCICSFixture)
		h := cobolTestCopybookHandoff(t, e, path, cobolCopybookExecCICSFixture)
		require.NotEqual(t, grade.Green, h.Assessment.Grade)
		assert.Equal(t, "copybook_content_unmapped", file.Meta["prov_analysis_absence"])

		findings := cobolFindingIDs(result)
		require.NotEmpty(t, findings, "an unmapped degraded member still carries its findings")
		cascades := 0
		if cobolTestCascaded(h.Assessment.Reasons) {
			cascades = 1
		}
		assertCobolFindingsPerGap(t, result, h, cascades)
		assert.Len(t, result.Nodes, 1+len(findings), "the walk mapped only the file node")
		for _, edge := range result.Edges {
			assert.True(t, findings[edge.From], "edge %s -> %s is not from a finding", edge.From, edge.To)
		}
	})
	t.Run("mapped_member_with_gaps", func(t *testing.T) {
		const path = "copy/custbad.cpy"
		result := extractCobolGrammar(t, e, path, cobolFindingCopybookFixture)
		file := assertCobolCopybookFile(t, result, path, cobolFindingCopybookFixture)
		findings := cobolFindingIDs(result)
		require.NotEmpty(t, findings)
		require.Greater(t, len(result.Nodes), 1+len(findings), "the walk mapped a member node")
		assert.NotContains(t, file.Meta, "prov_analysis_absence")
	})
}
