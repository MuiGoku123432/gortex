//go:build darwin || linux

package indexer

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	t.Run("master_drain", func(t *testing.T) {
		// A writer-free master ResolveAll (daemon warmup, deferred catch-up):
		// its settle ledger takes the writer itself. With scoped global
		// passes (the default) the master reconciles fdepa widened to fdepb
		// through resolve_from, so its drained record names fdepb (D-24).
		requireCobolLedgerExcludesTornMutation(t, func(env *cobolLifecycleEnv) {
			env.mi.runMasterResolve(map[string]struct{}{"fdepa": {}}, false)
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

// TestCobolFindings_MasterReconcileExcludesLane proves a writer-free master
// resolve reconciles COBOL and records the result in one reach topology
// writer section, so an edit that lands after the master's reconcile
// cannot record the master's close as a supersede (D-24, review WR-01).
// Each subtest is one writer-free entry: the daemon warmup resolve and the
// deferred catch-up's full fallback.
func TestCobolFindings_MasterReconcileExcludesLane(t *testing.T) {
	scope := map[string]struct{}{"finda": {}}
	t.Run("pre_enrich_resolve", func(t *testing.T) {
		requireCobolMasterCloseBeforeLane(t, func(env *cobolLifecycleEnv) error {
			return env.mi.RunPreEnrichResolve(context.Background(), scope, nil)
		})
	})
	t.Run("deferred_fallback", func(t *testing.T) {
		requireCobolMasterCloseBeforeLane(t, func(env *cobolLifecycleEnv) error {
			if mode, ok := env.mi.resolveDeferredMutations(nil, true, scope, false); !ok || mode != deferredResolveFallback {
				return fmt.Errorf("deferred catch-up: mode %s, complete %v", mode, ok)
			}
			return nil
		})
	})
}

// requireCobolMasterCloseBeforeLane interleaves master, a writer-free master
// resolve whose reconcile closes finda's NOPGM finding (NOPGM was retrieved
// through an interrupted reindex), with an incremental lane that adds CALL
// 'NEWPGM' to the same paragraph, using the cobolSettleHook seam and
// channels only.
//
// master runs first and parks right after its COBOL reconcile. The one
// TopologyWriterActive read happens then, while master is the only running
// goroutine, so it says exactly whether master holds the writer between its
// reconcile and its ledger. If it does, the lane cannot commit before
// master's ledger finishes, so master is released first. If it does not
// (the mutation run), the lane commits, reconciles, and ledgers first, and
// its ledger sees NOPGM's close and NEWPGM's open in one container: a
// supersede. NOPGM's row must be active -> closed, then NEWPGM's "" ->
// active. Every wait is bounded, so a regression fails instead of hanging.
func requireCobolMasterCloseBeforeLane(t *testing.T, master func(env *cobolLifecycleEnv) error) {
	t.Helper()
	env := newCobolLedgerEnv(t)
	nopgm := cobolExternalFindingFor(t, env, "finda", "NOPGM")
	idx := env.mi.GetIndexer("finda")
	require.NotNil(t, idx)
	// note: the same retrieval as TestCobolFindings_MasterDrainClosesBeforeLaterEdit.
	idx.incrementalResolveFilesHook = func([]string) {}
	env.write(t, "finda", "cobol/nopgm.cbl", cobolDepNopgm)
	env.reindex(t, "finda")
	idx.incrementalResolveFilesHook = nil
	require.NotNil(t, env.store.GetNode(nopgm), "precondition: the interruption kept the finding")
	before := len(cobolHistory(t, env, "finda"))
	withNew := strings.Replace(cobolLedgerFixture, "           CALL 'NOPGM'.\n",
		"           CALL 'NOPGM'.\n           CALL 'NEWPGM'.\n", 1)
	require.NotEqual(t, cobolLedgerFixture, withNew)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	finished := func(errs <-chan error, what string) {
		t.Helper()
		select {
		case err := <-errs:
			require.NoError(t, err, what)
		case <-ctx.Done():
			t.Fatalf("%s did not finish (writer held between the master's reconcile and its ledger: %v)", what,
				reach.TopologyWriterActive())
		}
	}

	aParked := make(chan struct{}, 1)
	releaseA := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseA) }) }
	t.Cleanup(release) // a failed wait must not leave master parked
	var parked atomic.Bool
	env.mi.cobolSettleHook = func(stage string, _ []string) {
		if stage == "master_reconciled" && parked.CompareAndSwap(false, true) {
			aParked <- struct{}{}
			<-releaseA
		}
	}
	t.Cleanup(func() { env.mi.cobolSettleHook = nil })

	aErr := make(chan error, 1)
	go func() { aErr <- master(env) }()
	select {
	case <-aParked:
	case <-ctx.Done():
		t.Fatal("the master never reached its post-reconcile stage")
	}
	held := reach.TopologyWriterActive()

	env.write(t, "finda", "cobol/main.cbl", withNew)
	bErr := make(chan error, 1)
	go func() {
		_, err := env.mi.IncrementalReindexRepo("finda", nil)
		bErr <- err
	}()
	if !held {
		finished(bErr, "the lane")
	}
	release()
	finished(aErr, "the master")
	if held {
		finished(bErr, "the lane")
	}

	require.Nil(t, env.store.GetNode(nopgm), "the master's reconcile closed NOPGM")
	newpgm := cobolExternalFindingFor(t, env, "finda", "NEWPGM")
	var got [][3]string
	for _, row := range cobolHistory(t, env, "finda")[before:] {
		got = append(got, [3]string{row.FindingID, row.PriorStatus, row.NewStatus})
	}
	require.Equal(t, [][3]string{
		{nopgm, graph.FindingStatusActive, graph.FindingStatusClosed},
		{newpgm, "", graph.FindingStatusActive},
	}, got, "the master records its close at its own step; the lane only opens NEWPGM (writer held: %v)", held)
}
