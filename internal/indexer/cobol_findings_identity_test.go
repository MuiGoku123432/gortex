//go:build darwin || linux

package indexer

import (
	"cmp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
)

// The finding-identity regressions on the real path (D-21, D-25, D-26): the
// registered COBOL extractor through a MultiIndexer into SQLite and the
// finding ledger. Every fixture is invented fixed-format COBOL (see
// cobol_lifecycle_test.go); this repository is public.

// cobolIdentityFixture is repository fid's cobol/shift.cbl, equal to the
// extractor's shift fixture: two SQL tails, ZQXCOL and ZQXOTH, in MAIN-PARA
// and one ERROR, so an edit has unrelated findings to leave alone.
const cobolIdentityFixture = `       IDENTIFICATION DIVISION.
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

// cobolIdentityPath is the repository-relative path of cobolIdentityFixture.
const cobolIdentityPath = "cobol/shift.cbl"

// newCobolIdentityEnv tracks a fresh repository fid holding only
// cobolIdentityFixture.
func newCobolIdentityEnv(t *testing.T) *cobolLifecycleEnv {
	t.Helper()
	env := newCobolLifecycleEnv(t, map[string]map[string]string{"fid": {cobolIdentityPath: cobolIdentityFixture}})
	env.track(t, "fid")
	return env
}

// TestCobolFindings_SameKindInsertKeepsIDs is VERIFICATION regression 1 on
// the real path: an unrelated EXEC SQL inserted above MAIN-PARA's gaps keeps
// every stored finding ID and only opens the inserted statement's findings.
func TestCobolFindings_SameKindInsertKeepsIDs(t *testing.T) {
	env := newCobolIdentityEnv(t)
	before := cobolLedgerFindings(env, "fid")
	require.GreaterOrEqual(t, len(before), 3, "the fixture's tails and ERROR")
	recorded := len(cobolHistory(t, env, "fid"))

	inserted := strings.Replace(cobolIdentityFixture, "       MAIN-PARA.\n",
		"       MAIN-PARA.\n           EXEC SQL SELECT AAA FROM APP.BBB END-EXEC.\n", 1)
	require.NotEqual(t, cobolIdentityFixture, inserted)
	env.write(t, "fid", cobolIdentityPath, inserted)
	env.reindex(t, "fid")

	after := cobolLedgerFindings(env, "fid")
	for id := range before {
		assert.Contains(t, after, id, "pre-existing finding keeps its ID (D-21)")
	}
	rows := cobolHistory(t, env, "fid")[recorded:]
	require.NotEmpty(t, rows, "the inserted statement opens its own findings")
	for _, row := range rows {
		assert.Equal(t, "", row.PriorStatus, "%+v", row)
		assert.Equal(t, graph.FindingStatusActive, row.NewStatus, "%+v", row)
		assert.NotContains(t, before, row.FindingID, "no row for a pre-existing finding")
	}
	env.requireCleanEquivalent(t)
}

// cobolTransitions returns history rows as FindingID -> "prior -> new",
// with " => successor" on a supersede, so a test compares them at once.
func cobolTransitions(rows []graph.FindingTransition) map[string]string {
	out := map[string]string{}
	for _, row := range rows {
		s := row.PriorStatus + " -> " + row.NewStatus
		if row.SuccessorID != "" {
			s += " => " + row.SuccessorID
		}
		out[row.FindingID] = s
	}
	return out
}

// TestCobolFindings_SameKindDeleteClosesOnlyItsOwn is VERIFICATION
// regression 2 on the real path: deleting the ZQXCOL EXEC SQL closes only
// its tail; the ZQXOTH tail and the ERROR keep their IDs and get no row.
func TestCobolFindings_SameKindDeleteClosesOnlyItsOwn(t *testing.T) {
	env := newCobolIdentityEnv(t)
	var tails []*graph.Node
	var errID string
	for id, n := range cobolLedgerFindings(env, "fid") {
		switch n.Meta[graph.MetaFindingKind] {
		case "unparsed-tail":
			tails = append(tails, n)
		case "error":
			errID = id
		}
	}
	require.Len(t, tails, 2, "the ZQXCOL and ZQXOTH tails")
	require.NotEmpty(t, errID, "the ERROR")
	// ZQXCOL's statement comes first in the source.
	slices.SortFunc(tails, func(a, b *graph.Node) int {
		return cmp.Compare(metaToInt(a.Meta["prov_start_byte"]), metaToInt(b.Meta["prov_start_byte"]))
	})
	col, oth := tails[0].ID, tails[1].ID
	recorded := len(cobolHistory(t, env, "fid"))

	deleted := strings.Replace(cobolIdentityFixture,
		"           EXEC SQL SELECT ZQXCOL FROM APP.ZQXTAB END-EXEC.\n", "", 1)
	require.NotEqual(t, cobolIdentityFixture, deleted)
	env.write(t, "fid", cobolIdentityPath, deleted)
	env.reindex(t, "fid")

	after := cobolLedgerFindings(env, "fid")
	assert.NotContains(t, after, col, "the deleted statement's tail is gone")
	assert.Contains(t, after, oth, "the ZQXOTH tail keeps its ID (D-21)")
	assert.Contains(t, after, errID, "the ERROR keeps its ID (D-21)")
	rows := cobolHistory(t, env, "fid")[recorded:]
	require.Len(t, rows, 1, "only the deleted statement's finding gets a row: %+v", rows)
	assert.Equal(t, map[string]string{col: "active -> closed"}, cobolTransitions(rows))
	requireNoSuperseded(t, env, "fid")
	env.requireCleanEquivalent(t)
}

// cobolIdentitySecondProgram is an invented clean program appended to a
// document to change its program count.
const cobolIdentitySecondProgram = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. ZQXSECND.
       PROCEDURE DIVISION.
           STOP RUN.
`

// TestCobolFindings_SecondProgramKeepsDocumentFinding is VERIFICATION
// regression 3 on the real path (WR-07): adding and removing a second
// program keeps a cascaded document's uncontained-cascade finding, anchored
// to the file, and appends no row for it.
func TestCobolFindings_SecondProgramKeepsDocumentFinding(t *testing.T) {
	src := cobolCascadeInject("", "           OBTAIN FIRST CUSTOMER-REC WITHIN CUST-ORDER-SET.\n", "")
	env := newCobolLifecycleEnv(t, map[string]map[string]string{"fsec": {cobolCascadeProgram: src}})
	env.track(t, "fsec")
	// cascade checks the file's program count and returns its one
	// uncontained-cascade finding, which must exist (positive control).
	cascade := func(programs int) string {
		t.Helper()
		var count int
		for _, n := range env.store.GetFileNodes("fsec/" + cobolCascadeProgram) {
			if n.Kind == graph.KindFunction && n.Meta["cobol_kind"] == "program" {
				count++
			}
		}
		require.Equal(t, programs, count, "programs in the file")
		_, _, cascades := cobolCascadeCounts(t, env, "fsec/"+cobolCascadeProgram)
		n := requireCobolCascadeFinding(t, cascades)
		assert.Equal(t, cobolCascadeProgram, n.Meta[graph.MetaFindingContainer],
			"document scope anchors to the file (D-21)")
		return n.ID
	}
	id := cascade(1)
	recorded := len(cobolHistory(t, env, "fsec"))

	env.write(t, "fsec", cobolCascadeProgram, src+cobolIdentitySecondProgram)
	env.reindex(t, "fsec")
	assert.Equal(t, id, cascade(2), "a second program keeps the ID")

	env.write(t, "fsec", cobolCascadeProgram, src)
	env.reindex(t, "fsec")
	assert.Equal(t, id, cascade(1), "removing it again keeps the ID")
	for _, row := range cobolHistory(t, env, "fsec")[recorded:] {
		assert.NotEqual(t, id, row.FindingID, "no row for the cascade finding: %+v", row)
	}
	env.requireCleanEquivalent(t)
}

// cobolIdentityRefFixture is repository fref's cobol/ref.cbl: a CALL and an
// EXEC CICS XCTL of the missing program NOPGM in one paragraph.
const cobolIdentityRefFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. REFPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           CALL 'NOPGM'.
           EXEC CICS XCTL PROGRAM('NOPGM') END-EXEC.
           STOP RUN.
`

// TestCobolFindings_ReferenceFormEditSupersedes proves D-25 on the real
// path: a CALL and an XCTL of one missing program are two findings, and
// editing the XCTL into a LINK supersedes only its finding.
func TestCobolFindings_ReferenceFormEditSupersedes(t *testing.T) {
	const path = "cobol/ref.cbl"
	env := newCobolLifecycleEnv(t, map[string]map[string]string{"fref": {path: cobolIdentityRefFixture}})
	env.track(t, "fref")
	// byLine returns fref's EXTERNAL_UNRESOLVED finding IDs by start line.
	byLine := func() map[int]string {
		out := map[int]string{}
		for id, n := range cobolLedgerFindings(env, "fref") {
			if n.Meta[graph.MetaEvidenceClass] == graph.FindingClassExternal {
				out[n.StartLine] = id
			}
		}
		return out
	}
	found := byLine()
	require.Len(t, found, 2, "the CALL and the XCTL are two findings (D-25): %v", found)
	call, xctl := found[5], found[6]
	require.NotEmpty(t, call, "the CALL on line 5")
	require.NotEmpty(t, xctl, "the XCTL on line 6")
	assert.NotContains(t, call, "#")
	assert.NotContains(t, xctl, "#")
	recorded := len(cobolHistory(t, env, "fref"))

	env.write(t, "fref", path, strings.Replace(cobolIdentityRefFixture, "XCTL", "LINK", 1))
	env.reindex(t, "fref")

	found = byLine()
	require.Len(t, found, 2, "%v", found)
	assert.Equal(t, call, found[5], "the CALL keeps its ID")
	link := found[6]
	require.NotEmpty(t, link, "the LINK on line 6")
	require.NotEqual(t, xctl, link, "editing the form changes the ID")
	rows := cobolHistory(t, env, "fref")[recorded:]
	require.Len(t, rows, 2, "%+v", rows)
	assert.Equal(t, map[string]string{
		xctl: "active -> superseded => " + link,
		link: " -> active",
	}, cobolTransitions(rows))
}

// cobolIdentitySameLineFixture is repository fdup's cobol/dup.cbl: two
// identical CALLs of the missing program NOPGM on one line.
const cobolIdentitySameLineFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. DUPPGM.
       PROCEDURE DIVISION.
       MAIN-PARA.
           CALL 'NOPGM'. CALL 'NOPGM'.
           STOP RUN.
`

// TestCobolFindings_SameLineCallsPerOccurrence proves D-26 on the real
// path: two same-line CALLs collapse into one stored edge but keep one
// finding each, and removing the second closes only base#2.
func TestCobolFindings_SameLineCallsPerOccurrence(t *testing.T) {
	const path = "cobol/dup.cbl"
	env := newCobolLifecycleEnv(t, map[string]map[string]string{"fdup": {path: cobolIdentitySameLineFixture}})
	env.track(t, "fdup")
	var calls []*graph.Edge
	for _, n := range env.store.GetFileNodes("fdup/" + path) {
		if n.Meta["cobol_kind"] != "paragraph" {
			continue
		}
		for _, e := range env.store.GetOutEdges(n.ID) {
			if e.Kind == graph.EdgeCalls {
				calls = append(calls, e)
			}
		}
	}
	require.Len(t, calls, 1, "the extractor collapses same-line duplicates into one edge")
	require.Equal(t, 2, metaToInt(calls[0].Meta["cobol_occurrence_count"]))

	external := func() []string {
		var ids []string
		for id, n := range cobolLedgerFindings(env, "fdup") {
			if n.Meta[graph.MetaEvidenceClass] == graph.FindingClassExternal {
				ids = append(ids, id)
			}
		}
		slices.Sort(ids)
		return ids
	}
	ids := external()
	require.Len(t, ids, 2, "one finding per occurrence (D-26): %v", ids)
	base := ids[0]
	require.Equal(t, base+"#2", ids[1])
	env.requireCleanEquivalent(t)
	recorded := len(cobolHistory(t, env, "fdup"))

	single := strings.Replace(cobolIdentitySameLineFixture, "CALL 'NOPGM'. CALL 'NOPGM'.", "CALL 'NOPGM'.", 1)
	require.NotEqual(t, cobolIdentitySameLineFixture, single)
	env.write(t, "fdup", path, single)
	env.reindex(t, "fdup")

	assert.Equal(t, []string{base}, external(), "the first occurrence keeps its ID")
	rows := cobolHistory(t, env, "fdup")[recorded:]
	require.Len(t, rows, 1, "%+v", rows)
	assert.Equal(t, map[string]string{base + "#2": "active -> closed"}, cobolTransitions(rows))
	env.requireCleanEquivalent(t)
}
