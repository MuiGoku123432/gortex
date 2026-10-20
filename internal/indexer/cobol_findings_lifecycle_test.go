//go:build darwin || linux

package indexer

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/graph/store_sqlite"
)

// The finding-ledger half of the lifecycle harness (GAP-06, GAP-07, D-03):
// every fixture below is invented fixed-format COBOL. It must never be
// replaced with estate-derived source: this repository is public.

// cobolLedgerFixture is repository finda's cobol/main.cbl. It carries one
// finding of each shape, each in its own container so an edit to one never
// moves another: a missing COPY NOCPY (the program), an SQL tail on ZQXCOL
// (TAIL-PARA's EXEC SQL statement), a localized ERROR (ERR-PARA, contained
// by the procedure division), and a missing CALL 'NOPGM' (CALL-PARA).
const cobolLedgerFixture = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. FINDPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY NOCPY.
       01 WS-B PIC 9(4).
       PROCEDURE DIVISION.
       TAIL-PARA.
           EXEC SQL SELECT ZQXCOL FROM APP.ZQXTAB END-EXEC.
       ERR-PARA.
           COMPUTE WS-B = (1 + 2.
       CALL-PARA.
           CALL 'NOPGM'.
           STOP RUN.
`

// cobolLedgerFixed is cobolLedgerFixture with the ERROR fixed, and
// cobolLedgerRetoken with the tail token ZQXCOL edited to ZQXNEW.
var (
	cobolLedgerFixed   = strings.Replace(cobolLedgerFixture, "(1 + 2.", "(1 + 2).", 1)
	cobolLedgerRetoken = strings.Replace(cobolLedgerFixture, "ZQXCOL", "ZQXNEW", 1)
)

// cobolLedgerRepos returns a fresh copy of the one-repository corpus finda.
func cobolLedgerRepos() map[string]map[string]string {
	return map[string]map[string]string{"finda": {"cobol/main.cbl": cobolLedgerFixture}}
}

// newCobolLedgerEnv tracks a fresh copy of repository finda.
func newCobolLedgerEnv(t *testing.T) *cobolLifecycleEnv {
	t.Helper()
	env := newCobolLifecycleEnv(t, cobolLedgerRepos())
	env.track(t, "finda")
	return env
}

// cobolLedgerFindings returns repo's KindFinding nodes in the store by ID.
func cobolLedgerFindings(env *cobolLifecycleEnv, repo string) map[string]*graph.Node {
	out := map[string]*graph.Node{}
	for n := range graph.NodesInScopeSeq(env.store, []string{repo}, nil, graph.KindFinding) {
		if n != nil {
			out[n.ID] = n
		}
	}
	return out
}

// cobolLedgerFindingOf returns the ID of repo's one finding of class and
// finding_kind kind.
func cobolLedgerFindingOf(t *testing.T, env *cobolLifecycleEnv, repo, class, kind string) string {
	t.Helper()
	var ids []string
	for id, n := range cobolLedgerFindings(env, repo) {
		if n.Meta[graph.MetaEvidenceClass] == class && n.Meta[graph.MetaFindingKind] == kind {
			ids = append(ids, id)
		}
	}
	require.Len(t, ids, 1, "one %s %s finding in %s", class, kind, repo)
	return ids[0]
}

// cobolHistory returns repo's full finding history in append order.
func cobolHistory(t *testing.T, env *cobolLifecycleEnv, repo string) []graph.FindingTransition {
	t.Helper()
	rows, err := env.store.FindingHistory(repo)
	require.NoError(t, err)
	return rows
}

// requireCobolOpenRows asserts repo's history is exactly one "" -> active
// row per finding node of the repository, with both classes present.
func requireCobolOpenRows(t *testing.T, env *cobolLifecycleEnv, repo string) {
	t.Helper()
	findings := cobolLedgerFindings(env, repo)
	classes := map[any]bool{}
	for _, n := range findings {
		classes[n.Meta[graph.MetaEvidenceClass]] = true
	}
	require.True(t, classes[graph.FindingClassParse] && classes[graph.FindingClassExternal],
		"both finding classes in %s: %v", repo, classes)
	history := cobolHistory(t, env, repo)
	require.Len(t, history, len(findings), "one row per finding of %s: %+v", repo, history)
	seen := map[string]bool{}
	for _, row := range history {
		require.Contains(t, findings, row.FindingID)
		require.False(t, seen[row.FindingID], "one row for %s", row.FindingID)
		seen[row.FindingID] = true
		assert.Equal(t, "", row.PriorStatus, row.FindingID)
		assert.Equal(t, graph.FindingStatusActive, row.NewStatus, row.FindingID)
	}
}

// cobolShadowAdmissions returns the shadow admissions granted so far by the
// budget env's indexers use, exactly as the indexer picks it: a delta across
// a track proves the track took the in-memory shadow.
func cobolShadowAdmissions(env *cobolLifecycleEnv) uint64 {
	admission := env.mi.shadowAdmission
	if admission == nil {
		admission = processShadowAdmission
	}
	return admission.snapshot().admissions
}

func TestCobolFindings_HistoryOpensOnTrack(t *testing.T) {
	env := newCobolLedgerEnv(t)
	requireCobolOpenRows(t, env, "finda")
	require.Len(t, cobolLedgerFindings(env, "finda"), 4, "the fixture's four findings")
}

func TestCobolFindings_HistoryIdenticalRerun(t *testing.T) {
	env := newCobolLedgerEnv(t)
	idx := env.mi.GetIndexer("finda")
	require.NotNil(t, idx)
	var mu sync.Mutex
	var cobolRuns [][]string
	idx.incrementalCatchupHook = func(kind string, files []string) {
		if kind != "cobol" {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		cobolRuns = append(cobolRuns, files)
	}
	// reindex re-extracts cobol/main.cbl and requires the COBOL reconcile
	// and ledger tail to have run. Only the first identical rewrite after the
	// cold track lists the path; a later one is classified inert (no listed
	// file), and cobolMutationTouched runs the tail for the stale file anyway.
	reindex := func(i int) {
		res, err := env.mi.IncrementalReindexRepo("finda", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.StaleFileCount, "reindex %d re-extracted cobol/main.cbl", i)
		mu.Lock()
		defer mu.Unlock()
		require.Len(t, cobolRuns, 1, "reindex %d ran the COBOL reconcile and ledger tail once", i)
		for _, p := range cobolRuns[0] {
			assert.True(t, strings.HasSuffix(p, "cobol/main.cbl"), "reindex %d tail path %s", i, p)
		}
		cobolRuns = nil
	}
	ids := slices.Sorted(maps.Keys(cobolLedgerFindings(env, "finda")))
	before := len(cobolHistory(t, env, "finda"))
	require.Positive(t, before)

	for i := range 2 {
		// Identical bytes, newer mtime: re-extracted, evicted, and re-added.
		env.write(t, "finda", "cobol/main.cbl", cobolLedgerFixture)
		reindex(i)
		assert.Len(t, cobolHistory(t, env, "finda"), before, "reindex %d appends no row (GAP-06)", i)
		assert.Equal(t, ids, slices.Sorted(maps.Keys(cobolLedgerFindings(env, "finda"))), "reindex %d keeps the IDs", i)
	}
	for i := range 2 {
		env.mi.reconcileCobolRepos(map[string]struct{}{"finda": {}})
		assert.Len(t, cobolHistory(t, env, "finda"), before, "direct pass %d appends no row", i)
	}

	env.write(t, "finda", "cobol/main.cbl", cobolLedgerRetoken)
	reindex(2)
	assert.Greater(t, len(cobolHistory(t, env, "finda")), before, "a token edit through the same path appends rows")
}

func TestCobolFindings_HistoryFixCloses(t *testing.T) {
	env := newCobolLedgerEnv(t)
	errID := cobolLedgerFindingOf(t, env, "finda", graph.FindingClassParse, "error")
	before := len(cobolHistory(t, env, "finda"))

	env.write(t, "finda", "cobol/main.cbl", cobolLedgerFixed)
	env.reindex(t, "finda")

	rows := cobolHistory(t, env, "finda")[before:]
	require.Len(t, rows, 1, "only the fixed ERROR's row: %+v", rows)
	assert.Equal(t, errID, rows[0].FindingID)
	assert.Equal(t, graph.FindingStatusActive, rows[0].PriorStatus)
	assert.Equal(t, graph.FindingStatusClosed, rows[0].NewStatus)
	assert.Empty(t, rows[0].SuccessorID)
}

func TestCobolFindings_HistorySupersede(t *testing.T) {
	env := newCobolLedgerEnv(t)
	oldTail := cobolLedgerFindingOf(t, env, "finda", graph.FindingClassParse, "unparsed-tail")
	before := len(cobolHistory(t, env, "finda"))

	env.write(t, "finda", "cobol/main.cbl", cobolLedgerRetoken)
	env.reindex(t, "finda")

	newTail := cobolLedgerFindingOf(t, env, "finda", graph.FindingClassParse, "unparsed-tail")
	require.NotEqual(t, oldTail, newTail, "the tail's own token is its evidence")
	rows := cobolHistory(t, env, "finda")[before:]
	require.Len(t, rows, 2, "only the edited tail's rows: %+v", rows)
	byID := map[string]graph.FindingTransition{}
	for _, row := range rows {
		byID[row.FindingID] = row
	}
	require.Contains(t, byID, oldTail)
	require.Contains(t, byID, newTail)
	assert.Equal(t, graph.FindingStatusActive, byID[oldTail].PriorStatus)
	assert.Equal(t, graph.FindingStatusSuperseded, byID[oldTail].NewStatus)
	assert.Equal(t, newTail, byID[oldTail].SuccessorID)
	assert.Equal(t, "", byID[newTail].PriorStatus)
	assert.Equal(t, graph.FindingStatusActive, byID[newTail].NewStatus)
}

func TestCobolFindings_HistoryRestoreReopens(t *testing.T) {
	env := newCobolLedgerEnv(t)
	ids := slices.Sorted(maps.Keys(cobolLedgerFindings(env, "finda")))
	require.Len(t, ids, 4)
	statuses := func(rows []graph.FindingTransition) map[string]string {
		out := map[string]string{}
		for _, row := range rows {
			out[row.FindingID] = row.PriorStatus + " -> " + row.NewStatus
		}
		return out
	}
	all := func(status string) map[string]string {
		out := map[string]string{}
		for _, id := range ids {
			out[id] = status
		}
		return out
	}

	// Deleting the repository's last COBOL file closes every finding of it
	// (Pitfall 8: the ledger is not gated on COBOL presence).
	before := len(cobolHistory(t, env, "finda"))
	env.remove(t, "finda", "cobol/main.cbl")
	env.reindex(t, "finda")
	require.Empty(t, cobolLedgerFindings(env, "finda"))
	rows := cobolHistory(t, env, "finda")[before:]
	require.Len(t, rows, len(ids), "%+v", rows)
	assert.Equal(t, all("active -> closed"), statuses(rows))

	before = len(cobolHistory(t, env, "finda"))
	env.write(t, "finda", "cobol/main.cbl", cobolLedgerFixture)
	env.reindex(t, "finda")
	require.Equal(t, ids, slices.Sorted(maps.Keys(cobolLedgerFindings(env, "finda"))), "the same IDs return")
	rows = cobolHistory(t, env, "finda")[before:]
	require.Len(t, rows, len(ids), "%+v", rows)
	assert.Equal(t, all("closed -> active"), statuses(rows))
}

// The dependent-repository corpus: fdepb's B-PARA calls NOPGM and resolves
// from fdepa, which holds NOPGM only when withNopgm is set.
const (
	cobolDepNopgmCall = "unresolved::cobol::program::NOPGM"
	cobolDepNopgm     = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. NOPGM.
       PROCEDURE DIVISION.
       NO-PARA.
           GOBACK.
`
)

func cobolDepRepos(withNopgm bool) map[string]map[string]string {
	a := map[string]string{"cobol/a.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. APGM.
       PROCEDURE DIVISION.
       A-PARA.
           GOBACK.
`}
	if withNopgm {
		a["cobol/nopgm.cbl"] = cobolDepNopgm
	}
	return map[string]map[string]string{
		"fdepa": a,
		"fdepb": {
			".gortex.yaml": "index:\n  cobol:\n    resolve_from: [fdepa]\n",
			"cobol/b.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. BPGM.
       PROCEDURE DIVISION.
       B-PARA.
           CALL 'NOPGM'.
           GOBACK.
`,
		},
	}
}

// requireNoSuperseded asserts no history row of repos is a supersede.
func requireNoSuperseded(t *testing.T, env *cobolLifecycleEnv, repos ...string) {
	t.Helper()
	for _, repo := range repos {
		for _, row := range cobolHistory(t, env, repo) {
			assert.NotEqual(t, graph.FindingStatusSuperseded, row.NewStatus, "%s: %+v", repo, row)
		}
	}
}

func TestCobolFindings_HistoryDependentRepo(t *testing.T) {
	const placeholder = "fdepb/cobol-missing::program/NOPGM"

	t.Run("incremental", func(t *testing.T) {
		env := newCobolLifecycleEnv(t, cobolDepRepos(false))
		env.track(t, "fdepa", "fdepb")
		id := cobolLedgerFindingOf(t, env, "fdepb", graph.FindingClassExternal, "external-reference")
		history := cobolHistory(t, env, "fdepb")
		require.Len(t, history, 1)
		require.Equal(t, id, history[0].FindingID)
		require.Equal(t, "", history[0].PriorStatus)
		require.Equal(t, graph.FindingStatusActive, history[0].NewStatus)

		// A change in fdepa binds fdepb's CALL: fdepb's row lands at this step.
		env.write(t, "fdepa", "cobol/nopgm.cbl", cobolDepNopgm)
		env.reindex(t, "fdepa")
		rows := cobolHistory(t, env, "fdepb")[1:]
		require.Len(t, rows, 1, "%+v", rows)
		assert.Equal(t, id, rows[0].FindingID)
		assert.Equal(t, graph.FindingStatusActive, rows[0].PriorStatus)
		assert.Equal(t, graph.FindingStatusClosed, rows[0].NewStatus)
		assert.Nil(t, env.store.GetNode(placeholder))
		requireNoSuperseded(t, env, "fdepa", "fdepb")

		env.remove(t, "fdepa", "cobol/nopgm.cbl")
		env.reindex(t, "fdepa")
		rows = cobolHistory(t, env, "fdepb")[2:]
		require.Len(t, rows, 1, "%+v", rows)
		assert.Equal(t, id, rows[0].FindingID, "the same finding reopens")
		assert.Equal(t, graph.FindingStatusClosed, rows[0].PriorStatus)
		assert.Equal(t, graph.FindingStatusActive, rows[0].NewStatus)
		requireNoSuperseded(t, env, "fdepa", "fdepb")
	})

	for _, shadow := range []bool{false, true} {
		name := "shadow=off"
		if shadow {
			name = "shadow=on"
		}
		t.Run(name, func(t *testing.T) {
			// newEnv builds the env with NOPGM already in fdepa and returns
			// a track that, with the shadow on, proves each track took it.
			newEnv := func(t *testing.T) (*cobolLifecycleEnv, func(name string)) {
				env := newCobolLifecycleEnv(t, cobolDepRepos(true))
				if shadow {
					// newCobolLifecycleEnv sets the cap to 0 on every call.
					t.Setenv("GORTEX_SHADOW_MAX_FILES", "1000000")
				}
				return env, func(name string) {
					before := cobolShadowAdmissions(env)
					env.track(t, name)
					if shadow {
						require.Greater(t, cobolShadowAdmissions(env), before, "track of %s took the shadow", name)
					}
				}
			}

			t.Run("fdepb-then-fdepa", func(t *testing.T) {
				env, track := newEnv(t)
				track("fdepb")
				id := cobolLedgerFindingOf(t, env, "fdepb", graph.FindingClassExternal, "external-reference")
				history := cobolHistory(t, env, "fdepb")
				require.Len(t, history, 1, "%+v", history)
				require.Equal(t, id, history[0].FindingID)
				require.Equal(t, graph.FindingStatusActive, history[0].NewStatus)

				// fdepa supplies NOPGM: fdepb's finding closes at this track.
				track("fdepa")
				rows := cobolHistory(t, env, "fdepb")[1:]
				require.Len(t, rows, 1, "%+v", rows)
				assert.Equal(t, id, rows[0].FindingID)
				assert.Equal(t, graph.FindingStatusClosed, rows[0].NewStatus)
				assert.Nil(t, env.store.GetNode(id))
				assert.Nil(t, env.store.GetNode(placeholder))
			})

			t.Run("fdepa-then-fdepb", func(t *testing.T) {
				env, track := newEnv(t)
				track("fdepa")
				track("fdepb")
				for id, n := range cobolLedgerFindings(env, "fdepb") {
					assert.NotEqual(t, graph.FindingClassExternal, n.Meta[graph.MetaEvidenceClass], "no external finding %s", id)
				}
				assert.Nil(t, env.store.GetNode(placeholder), "no not-obtained placeholder")
				var nopgm []string
				for n := range graph.NodesInScopeSeq(env.store, []string{"fdepa"}, nil, graph.KindFunction) {
					if n != nil && n.Meta["cobol_kind"] == "program" && n.Name == "NOPGM" {
						nopgm = append(nopgm, n.ID)
					}
				}
				require.Len(t, nopgm, 1, "fdepa's program NOPGM")
				call := cobolPlaceholderEdge(t, cobolDumpEdges(t, env.dump(t)), "fdepb", cobolDepNopgmCall)
				assert.Equal(t, nopgm[0], call.to, "fdepb's CALL binds into fdepa")
				for _, row := range cobolHistory(t, env, "fdepb") {
					assert.NotEqual(t, graph.FindingClassExternal, row.Class, "no external history row: %+v", row)
				}
			})
		})
	}
}

func TestCobolFindings_LedgerSerialized(t *testing.T) {
	store, err := store_sqlite.Open(filepath.Join(t.TempDir(), "ledger.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	const n = 64
	nodes := make([]*graph.Node, 0, n)
	for i := range n {
		nodes = append(nodes, &graph.Node{
			ID:   fmt.Sprintf("r1/a.cbl::@finding/parse/%032x", i),
			Kind: graph.KindFinding, Name: "error", FilePath: "r1/a.cbl", Language: "cobol", RepoPrefix: "r1",
			Meta: map[string]any{
				graph.MetaEvidenceClass:    graph.FindingClassParse,
				graph.MetaFindingKind:      "error",
				graph.MetaFindingContainer: "a.cbl::APGM",
				"prov_start_byte":          i,
			},
		})
	}
	store.AddBatch(nodes, nil)

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() { recordFindingLedger(store, []string{"r1"}, zap.NewNop()) })
	}
	wg.Wait()

	rows, err := store.FindingHistory("r1")
	require.NoError(t, err)
	assert.Len(t, rows, n, "each open is appended exactly once")
}

// TestCobolFindings_HistoryBatchDeferredTrack pins the cold-track tail's
// batch gating: only BeginParallelBatch (the daemon warmup's entry) defers
// resolution, so only it defers the tail to EndBatch.
func TestCobolFindings_HistoryBatchDeferredTrack(t *testing.T) {
	t.Run("parallel", func(t *testing.T) {
		env := newCobolLifecycleEnv(t, cobolLedgerRepos())
		env.mi.BeginParallelBatch()
		env.track(t, "finda")
		assert.Empty(t, cobolHistory(t, env, "finda"), "the deferred track's reconcile and ledger wait for EndBatch")
		env.mi.EndBatch()
		requireCobolOpenRows(t, env, "finda")
	})

	t.Run("plain", func(t *testing.T) {
		env := newCobolLifecycleEnv(t, cobolLedgerRepos())
		env.mi.BeginBatch() // defers global passes only, never resolution
		env.track(t, "finda")
		requireCobolOpenRows(t, env, "finda")
		rows := len(cobolHistory(t, env, "finda"))
		env.mi.EndBatch()
		assert.Len(t, cobolHistory(t, env, "finda"), rows, "EndBatch appends no row")
	})
}

// TestCobolFindings_HistoryShadowPath tracks finda through the in-memory
// shadow: the history lands in SQLite anyway (Pitfall 3).
func TestCobolFindings_HistoryShadowPath(t *testing.T) {
	env := newCobolLifecycleEnv(t, cobolLedgerRepos())
	t.Setenv("GORTEX_SHADOW_MAX_FILES", "1000000") // newCobolLifecycleEnv set 0
	before := cobolShadowAdmissions(env)
	env.track(t, "finda")
	require.Greater(t, cobolShadowAdmissions(env), before, "the track took the shadow")
	requireCobolOpenRows(t, env, "finda")
}

// TestCobolFindings_HistoryInterrupted interrupts the reindex that fixes the
// ERROR before its resolver tail: nothing is recorded until the warm-restart
// reconcile, which records exactly the transition to the final state.
func TestCobolFindings_HistoryInterrupted(t *testing.T) {
	env := newCobolLedgerEnv(t)
	errID := cobolLedgerFindingOf(t, env, "finda", graph.FindingClassParse, "error")
	idx := env.mi.GetIndexer("finda")
	require.NotNil(t, idx)
	idx.incrementalResolveFilesHook = func([]string) {}
	before := len(cobolHistory(t, env, "finda"))

	env.write(t, "finda", "cobol/main.cbl", cobolLedgerFixed)
	env.reindex(t, "finda")
	assert.Len(t, cobolHistory(t, env, "finda"), before, "an interrupted reconcile appends no row")

	// Daemon restart: the warm reconcile reconciles COBOL unconditionally.
	env.restart(t)
	env.warmReconcile(t, "finda")
	rows := cobolHistory(t, env, "finda")[before:]
	require.Len(t, rows, 1, "only the fixed ERROR's row: %+v", rows)
	assert.Equal(t, errID, rows[0].FindingID)
	assert.Equal(t, graph.FindingStatusActive, rows[0].PriorStatus)
	assert.Equal(t, graph.FindingStatusClosed, rows[0].NewStatus)
}

// TestCobolFindings_HistoryParserRestamp re-stamps a stored finding's parser
// and extractor versions without changing its evidence: same ID, no row
// (D-02).
func TestCobolFindings_HistoryParserRestamp(t *testing.T) {
	env := newCobolLedgerEnv(t)
	tailID := cobolLedgerFindingOf(t, env, "finda", graph.FindingClassParse, "unparsed-tail")
	n := env.store.GetNode(tailID)
	require.NotNil(t, n)
	n.Meta["prov_parser_tool_id"] = "restamped-parser/2"
	n.Meta["prov_extractor_version"] = "restamped-extractor/2"
	env.store.AddNode(n)
	require.Equal(t, "restamped-parser/2", env.store.GetNode(tailID).Meta["prov_parser_tool_id"])
	before := len(cobolHistory(t, env, "finda"))

	env.mi.reconcileCobolRepos(map[string]struct{}{"finda": {}})

	assert.Len(t, cobolHistory(t, env, "finda"), before, "a re-stamp appends no row")
	assert.Contains(t, cobolLedgerFindings(env, "finda"), tailID, "and keeps the ID")
}
