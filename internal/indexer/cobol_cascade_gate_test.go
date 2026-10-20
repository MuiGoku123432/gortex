//go:build darwin || linux

package indexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
)

// The COBOL cascade gate (GAP-08, D-14, D-15): the cobolprobe
// TestErrorCascade cases ported onto the production path, the registered
// COBOL extractor through a MultiIndexer into SQLite. An injected shape that
// cascades must persist one uncontained-cascade finding and still index; a
// contained shape must keep every paragraph and data item of the control.
// Every fixture is invented fixed-format COBOL (see cobol_lifecycle_test.go).

// cobolCascadeControl returns the parts of the invented 20-data-item,
// 20-paragraph control of cobolprobe TestErrorCascade: head ends at the
// WORKING-STORAGE header, data holds the 20 items, mid is the PROCEDURE
// DIVISION header, and proc holds the 20 paragraphs.
func cobolCascadeControl() (head, data, mid, proc string) {
	var d, p strings.Builder
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&d, "       01  WS-FIELD-%02d          PIC X(10).\n", i)
	}
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&p, "       %04d-PARA.\n           MOVE 'X' TO WS-FIELD-01.\n", i)
	}
	head = "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. T.\n       DATA DIVISION.\n       WORKING-STORAGE SECTION.\n"
	mid = "       PROCEDURE DIVISION.\n"
	return head, d.String(), mid, p.String()
}

// cobolCascadeInject returns the control with beforeData ahead of the first
// data item, afterData after it, and afterPara after the first paragraph
// (the probe's injection points). Three empty strings give the clean control.
func cobolCascadeInject(beforeData, afterData, afterPara string) string {
	head, data, mid, proc := cobolCascadeControl()
	dataLines := strings.SplitN(data, "\n", 2)
	lines := strings.SplitN(proc, "\n", 3)
	return head + beforeData + dataLines[0] + "\n" + afterData + dataLines[1] +
		mid + lines[0] + "\n" + lines[1] + "\n" + afterPara + lines[2] + "           STOP RUN.\n"
}

// cobolCascadeCounts returns, for the nodes SQLite holds for path, the
// number of data items and paragraphs (by Meta cobol_kind) and the
// uncontained-cascade findings.
func cobolCascadeCounts(t *testing.T, env *cobolLifecycleEnv, path string) (data, paras int, cascades []*graph.Node) {
	t.Helper()
	nodes := env.store.GetFileNodes(path)
	require.NotEmpty(t, nodes, "no nodes persisted for %s", path)
	for _, n := range nodes {
		switch {
		case n.Kind == graph.KindFinding && n.Meta[graph.MetaFindingKind] == "uncontained-cascade":
			cascades = append(cascades, n)
		case n.Meta["cobol_kind"] == "data_item":
			data++
		case n.Meta["cobol_kind"] == "paragraph":
			paras++
		}
	}
	return data, paras, cascades
}

// cobolCascadeTrack tracks src as cobol/t.cbl, the only file of repository
// gate in a fresh store, and returns its counts. Tracking must succeed and
// persist the file node whatever the producer grade (D-14: report, never
// block).
func cobolCascadeTrack(t *testing.T, src string) (data, paras int, cascades []*graph.Node) {
	t.Helper()
	env := newCobolLifecycleEnv(t, map[string]map[string]string{"gate": {"cobol/t.cbl": src}})
	env.track(t, "gate")
	require.NotNil(t, env.store.GetNode("gate/cobol/t.cbl"), "file node persisted")
	return cobolCascadeCounts(t, env, "gate/cobol/t.cbl")
}

// requireCobolCascadeFinding checks that cascades holds exactly one
// high-severity PARSE_UNRESOLVED uncontained-cascade finding that carries
// the producer reasons.
func requireCobolCascadeFinding(t *testing.T, cascades []*graph.Node) *graph.Node {
	t.Helper()
	require.Len(t, cascades, 1, "one uncontained-cascade finding per cascaded document")
	n := cascades[0]
	assert.Equal(t, graph.FindingClassParse, n.Meta[graph.MetaEvidenceClass], n.ID)
	assert.Equal(t, "high", n.Meta[graph.MetaFindingSeverity], n.ID)
	assert.NotEmpty(t, n.Meta[graph.MetaFindingProducerReasons], n.ID)
	return n
}

func TestCobolCascadeGate_CleanControl(t *testing.T) {
	d, pa, cascades := cobolCascadeTrack(t, cobolCascadeInject("", "", ""))
	if d != 20 || pa != 20 {
		t.Fatalf("clean control moved: got %d data items and %d paragraphs, want 20 and 20", d, pa)
	}
	assert.Empty(t, cascades, "the clean control has no uncontained-cascade finding")
}

func TestCobolCascadeGate_Positive(t *testing.T) {
	clean := cobolCascadeInject("", "", "")
	for _, tc := range []struct{ name, src string }{
		{"obtain_in_working_storage", cobolCascadeInject("",
			"           OBTAIN FIRST CUSTOMER-REC WITHIN CUST-ORDER-SET.\n", "")},
		{"declare_cursor_in_working_storage", cobolCascadeInject("",
			"           EXEC SQL DECLARE C1 CURSOR FOR SELECT A FROM T1 END-EXEC.\n", "")},
		{"declare_with_hold_in_procedure", cobolCascadeInject("", "",
			"           EXEC SQL DECLARE C2 CURSOR WITH HOLD FOR\n"+
				"               SELECT A FROM T1 END-EXEC.\n")},
		{"unknown_verb_in_procedure", cobolCascadeInject("", "",
			"           FROBNICATE THE WIDGET NOW.\n")},
		{"broken_data_clause", cobolCascadeInject("",
			"       01  BROKEN PIC X(10) GARBAGE CLAUSE HERE.\n", "")},
		{"data_division_without_period",
			strings.Replace(clean, "DATA DIVISION.\n", "DATA DIVISION\n", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NotEqual(t, clean, tc.src, "the shape must change the control")
			_, _, cascades := cobolCascadeTrack(t, tc.src)
			requireCobolCascadeFinding(t, cascades)
		})
	}
}

func TestCobolCascadeGate_Negative(t *testing.T) {
	// The clean control anchors every comparison below, so a regression
	// that lowered the baseline cannot make a broken shape look contained.
	d, pa, _ := cobolCascadeTrack(t, cobolCascadeInject("", "", ""))
	if d != 20 || pa != 20 {
		t.Fatalf("clean control moved: got %d data items and %d paragraphs, want 20 and 20", d, pa)
	}
	for _, tc := range []struct{ name, src string }{
		{"exec_cics_paren_option_after_para_1", cobolCascadeInject("", "",
			"           EXEC CICS SEND MAP('M') END-EXEC.\n")},
		{"exec_cics_no_option_after_para_1", cobolCascadeInject("", "",
			"           EXEC CICS RETURN END-EXEC.\n")},
		{"exec_cics_bare_option_after_para_1", cobolCascadeInject("", "",
			"           EXEC CICS SEND ERASE END-EXEC.\n")},
		{"exec_cics_multi_line_after_para_1", cobolCascadeInject("", "",
			"           EXEC CICS\n               SEND MAP('MENU01')\n               MAPSET('MENUSET')\n           END-EXEC.\n")},
		{"idms_obtain_after_para_1", cobolCascadeInject("", "",
			"           OBTAIN FIRST CUSTOMER-REC WITHIN CUST-ORDER-SET.\n")},
		{"exec_sql_select_after_para_1", cobolCascadeInject("", "",
			"           EXEC SQL SELECT ID FROM APP.CUSTOMER END-EXEC.\n")},
		{"schema_section_before_data", cobolCascadeInject(
			"       SCHEMA SECTION.\n       DB DCCSS000 WITHIN CAMSC000.\n", "", "")},
		{"exec_sql_include_sqlca_in_working_storage", cobolCascadeInject("",
			"       EXEC SQL INCLUDE SQLCA END-EXEC.\n", "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d2, pa2, cascades := cobolCascadeTrack(t, tc.src)
			if d2 != d || pa2 != pa {
				t.Fatalf("%s cascaded: got %d data items and %d paragraphs, want %d and %d (clean control)",
					tc.name, d2, pa2, d, pa)
			}
			assert.Empty(t, cascades, "a contained shape has no uncontained-cascade finding")
		})
	}
}
