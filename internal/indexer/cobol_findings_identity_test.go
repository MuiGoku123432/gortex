//go:build darwin || linux

package indexer

import (
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
