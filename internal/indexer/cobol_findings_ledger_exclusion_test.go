//go:build darwin || linux

package indexer

import (
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/reach"
)

// TestCobolFindings_LedgerExcludesTornMutation proves a settle point's
// finding ledger can no longer read a repository lane between its file
// commit, which evicts the file's external findings, and its reconcile,
// which re-adds them (D-23, WR-01). Each subtest is one settle point whose
// ledger covers fdepb through fdepb's resolve_from on fdepa.
func TestCobolFindings_LedgerExcludesTornMutation(t *testing.T) {
	t.Run("track_tail", func(t *testing.T) {
		// The cold TrackRepoCtx tail's call outside a topology batch.
		requireCobolLedgerExcludesTornMutation(t, func(env *cobolLifecycleEnv) {
			env.mi.reconcileCobolRepos(map[string]struct{}{"fdepa": {}})
		})
	})
}

// requireCobolLedgerExcludesTornMutation interleaves settle, a settle point
// over fdepa whose reconcile widens to fdepb, with an incremental lane on
// fdepb that commits a comment-only edit of cobol/b.cbl and then reconciles
// it, using the cobolSettleHook seam and channels only: no sleep and no time
// window.
//
// settle runs first and parks at its ledger stage. The one
// TopologyWriterActive read happens then, while settle is the only running
// goroutine, so it says exactly whether settle holds the writer. If it does,
// the lane cannot commit before settle's ledger finishes, so settle is
// released first. If it does not (the mutation run), the test waits for the
// lane's commit before releasing settle, which is the torn read: settle then
// records fdepb's unchanged external finding active -> closed and the lane's
// own ledger closed -> active. The finding must keep its ID and gain no
// history row.
func requireCobolLedgerExcludesTornMutation(t *testing.T, settle func(env *cobolLifecycleEnv)) {
	t.Helper()
	env := newCobolLifecycleEnv(t, cobolDepRepos(false))
	env.track(t, "fdepa", "fdepb")
	phi := cobolLedgerFindingOf(t, env, "fdepb", graph.FindingClassExternal, "external-reference")
	n0 := len(cobolHistory(t, env, "fdepb"))
	commented := strings.Replace(env.repos["fdepb"]["cobol/b.cbl"],
		"PROCEDURE DIVISION.\n", "PROCEDURE DIVISION.\n      * UNCHANGED EVIDENCE.\n", 1)
	require.NotEqual(t, env.repos["fdepb"]["cobol/b.cbl"], commented)

	aParked := make(chan struct{}, 1)
	releaseA := make(chan struct{})
	bCommitted := make(chan struct{}, 1)
	releaseB := make(chan struct{})
	var ledgerSeen atomic.Bool
	env.mi.cobolSettleHook = func(stage string, repos []string) {
		switch {
		case stage == "ledger" && ledgerSeen.CompareAndSwap(false, true):
			aParked <- struct{}{}
			<-releaseA
		case stage == "committed" && slices.Equal(repos, []string{"fdepb"}):
			select {
			case bCommitted <- struct{}{}:
			default:
			}
			<-releaseB
		}
	}
	t.Cleanup(func() { env.mi.cobolSettleHook = nil })

	aDone := make(chan struct{})
	go func() {
		defer close(aDone)
		settle(env)
	}()
	<-aParked
	held := reach.TopologyWriterActive()

	env.write(t, "fdepb", "cobol/b.cbl", commented)
	bErr := make(chan error, 1)
	go func() {
		_, err := env.mi.IncrementalReindexRepo("fdepb", nil)
		bErr <- err
	}()
	if !held {
		<-bCommitted
	}
	close(releaseA)
	<-aDone
	close(releaseB)
	require.NoError(t, <-bErr)

	require.Equal(t, phi, cobolLedgerFindingOf(t, env, "fdepb", graph.FindingClassExternal, "external-reference"),
		"the unchanged finding keeps its ID")
	var torn []graph.FindingTransition
	for _, row := range cobolHistory(t, env, "fdepb")[n0:] {
		if row.FindingID == phi {
			torn = append(torn, row)
		}
	}
	require.Empty(t, torn, "no history row for unchanged evidence (writer held at the settle point: %v)", held)
}
