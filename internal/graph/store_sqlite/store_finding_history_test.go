package store_sqlite

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/zzet/gortex/internal/graph"
)

// historyRow returns a fully populated finding history row, so a read-back
// comparison proves every column round-trips.
func historyRow(repo, id, prior, next, successor string, start int) graph.FindingTransition {
	return graph.FindingTransition{
		FindingID:        repo + "/a.cbl::@finding/parse/" + id,
		RepoPrefix:       repo,
		WorkspaceID:      "ws-" + repo,
		ProjectID:        "project-" + repo,
		PriorStatus:      prior,
		NewStatus:        next,
		SuccessorID:      successor,
		Class:            graph.FindingClassParse,
		Kind:             "unparsed-tail",
		Container:        repo + "/a.cbl::PGM/MAIN",
		StartByte:        start,
		EvidenceDigest:   "digest-" + id,
		ParserToolID:     "tool-" + id,
		ParserGrammarID:  "grammar-" + id,
		ExtractorVersion: "gortex-cobol-grammar/6",
	}
}

func openFindingHistoryStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "graph.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func appendHistory(t *testing.T, s *Store, repo string, rows ...graph.FindingTransition) {
	t.Helper()
	if err := s.AppendFindingTransitions(repo, rows); err != nil {
		t.Fatalf("AppendFindingTransitions(%q): %v", repo, err)
	}
}

func readHistory(t *testing.T, s *Store, repo string) []graph.FindingTransition {
	t.Helper()
	got, err := s.FindingHistory(repo)
	if err != nil {
		t.Fatalf("FindingHistory(%q): %v", repo, err)
	}
	return got
}

func TestFindingHistory_AppendAndRead(t *testing.T) {
	s := openFindingHistoryStore(t)
	first := []graph.FindingTransition{
		historyRow("r1", "a", "", graph.FindingStatusActive, "", 10),
		historyRow("r1", "b", "", graph.FindingStatusActive, "", -1),
	}
	second := []graph.FindingTransition{
		historyRow("r1", "a", graph.FindingStatusActive, graph.FindingStatusSuperseded, "r1/a.cbl::@finding/parse/c", 10),
		historyRow("r1", "c", "", graph.FindingStatusActive, "", 12),
	}
	other := historyRow("r2", "x", "", graph.FindingStatusActive, "", 4)

	appendHistory(t, s, "r1", first...)
	appendHistory(t, s, "r2", other)
	appendHistory(t, s, "r1", second...)
	appendHistory(t, s, "r1")
	appendHistory(t, s, "r1", []graph.FindingTransition{}...)

	want := append(append([]graph.FindingTransition{}, first...), second...)
	if got := readHistory(t, s, "r1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("r1 history = %#v, want %#v", got, want)
	}
	if got := readHistory(t, s, "r2"); !reflect.DeepEqual(got, []graph.FindingTransition{other}) {
		t.Fatalf("r2 history = %#v, want %#v", got, other)
	}
	if got := readHistory(t, s, "r3"); got == nil || len(got) != 0 {
		t.Fatalf("unknown repository history = %#v, want an empty slice", got)
	}

	rows, err := s.db.Query(`SELECT seq FROM finding_history WHERE repo_prefix = 'r1' ORDER BY seq`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var seqs []int64
	for rows.Next() {
		var seq int64
		if err := rows.Scan(&seq); err != nil {
			t.Fatal(err)
		}
		seqs = append(seqs, seq)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seqs, []int64{1, 2, 3, 4}) {
		t.Fatalf("r1 seq = %v, want 1..4 in append order", seqs)
	}
}

func TestFindingHistory_LatestPerFinding(t *testing.T) {
	s := openFindingHistoryStore(t)
	reopened := historyRow("r1", "a", graph.FindingStatusClosed, graph.FindingStatusActive, "", 30)
	b := historyRow("r1", "b", "", graph.FindingStatusActive, "", 20)
	appendHistory(t, s, "r1",
		historyRow("r1", "a", "", graph.FindingStatusActive, "", 10),
		b,
	)
	appendHistory(t, s, "r1", historyRow("r1", "a", graph.FindingStatusActive, graph.FindingStatusClosed, "", 10))
	appendHistory(t, s, "r2", historyRow("r2", "a", "", graph.FindingStatusActive, "", 10))
	appendHistory(t, s, "r1", reopened)

	got, err := s.LatestFindingTransitions("r1")
	if err != nil {
		t.Fatal(err)
	}
	want := []graph.FindingTransition{reopened, b}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("latest r1 rows = %#v, want %#v", got, want)
	}
	if got, err := s.LatestFindingTransitions("r3"); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("unknown repository latest rows = %#v, %v; want an empty slice", got, err)
	}
}

func TestFindingHistory_SurvivesEvictRepo(t *testing.T) {
	s := openFindingHistoryStore(t)
	nodes := func() []*graph.Node {
		return []*graph.Node{{ID: "r1/a.cbl", Kind: graph.KindFile, Name: "a.cbl", FilePath: "r1/a.cbl", RepoPrefix: "r1"}}
	}
	s.AddBatch(nodes(), nil)
	want := []graph.FindingTransition{historyRow("r1", "a", "", graph.FindingStatusActive, "", 10)}
	appendHistory(t, s, "r1", want...)

	s.EvictRepo("r1")
	if s.GetNode("r1/a.cbl") != nil {
		t.Fatal("EvictRepo kept the repository's file node")
	}
	if got := readHistory(t, s, "r1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("history after EvictRepo = %#v, want %#v (D-07)", got, want)
	}
	s.AddBatch(nodes(), nil)
	if got := readHistory(t, s, "r1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("history after re-adding nodes = %#v, want %#v", got, want)
	}
}

func TestFindingHistory_PurgedWithRepo(t *testing.T) {
	s := openFindingHistoryStore(t)
	appendHistory(t, s, "r1",
		historyRow("r1", "a", "", graph.FindingStatusActive, "", 10),
		historyRow("r1", "a", graph.FindingStatusActive, graph.FindingStatusClosed, "", 10),
	)
	kept := []graph.FindingTransition{historyRow("r2", "x", "", graph.FindingStatusActive, "", 4)}
	appendHistory(t, s, "r2", kept...)

	if err := s.PurgeRepo("r1"); err != nil {
		t.Fatalf("PurgeRepo: %v", err)
	}
	if got := readHistory(t, s, "r1"); len(got) != 0 {
		t.Fatalf("history after PurgeRepo = %#v, want none", got)
	}
	if got := readHistory(t, s, "r2"); !reflect.DeepEqual(got, kept) {
		t.Fatalf("other repository history after PurgeRepo = %#v, want %#v", got, kept)
	}
}

func findingHistoryIndexCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	return scalarInt(t, db, `SELECT COUNT(*) FROM sqlite_master
WHERE type = 'index' AND name = 'finding_history_by_finding' AND tbl_name = 'finding_history'`)
}

func TestFindingHistory_MigrationV23(t *testing.T) {
	var step *schemaMigration
	for i := range schemaMigrations {
		if schemaMigrations[i].version == 23 {
			step = &schemaMigrations[i]
		}
	}
	if step == nil || step.name != "persist finding history" || step.inPlace == nil || step.rebuild {
		t.Fatalf("v23 migration = %+v, want the in-place persist finding history step", step)
	}

	path := filepath.Join(t.TempDir(), "graph.db")
	fresh, err := Open(path)
	if err != nil {
		t.Fatalf("open fresh store: %v", err)
	}
	if got := findingHistoryIndexCount(t, fresh.writerDB); got != 1 {
		t.Fatalf("fresh store finding_history_by_finding count = %d, want 1", got)
	}
	if err := fresh.Close(); err != nil {
		t.Fatal(err)
	}

	withRawDB(t, path, func(db *sql.DB) {
		execDDL(t, db, `DROP TABLE finding_history`)
		execDDL(t, db, `PRAGMA user_version = 22`)
	})
	migrated, err := Open(path)
	if err != nil {
		t.Fatalf("open v22 store: %v", err)
	}
	if migrated.NeedsRebuild() {
		t.Fatal("v23 is additive and must not signal a rebuild")
	}
	if v, err := readUserVersion(migrated.writerDB); err != nil || v != 23 {
		t.Fatalf("migrated user_version = %d (err %v), want 23", v, err)
	}
	if got := scalarInt(t, migrated.writerDB, `SELECT COUNT(*) FROM finding_history`); got != 0 {
		t.Fatalf("migrated finding_history rows = %d, want an empty table (D-07)", got)
	}
	if got := findingHistoryIndexCount(t, migrated.writerDB); got != 1 {
		t.Fatalf("migrated finding_history_by_finding count = %d, want 1", got)
	}
	want := []graph.FindingTransition{historyRow("r1", "a", "", graph.FindingStatusActive, "", 10)}
	appendHistory(t, migrated, "r1", want...)
	if err := migrated.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen v23 store: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	if v, err := readUserVersion(reopened.writerDB); err != nil || v != 23 {
		t.Fatalf("reopened user_version = %d (err %v), want 23", v, err)
	}
	if got := readHistory(t, reopened, "r1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("history after reopen = %#v, want %#v", got, want)
	}
	if got := findingHistoryIndexCount(t, reopened.writerDB); got != 1 {
		t.Fatalf("reopened finding_history_by_finding count = %d, want 1", got)
	}

	// The step itself is idempotent on a store that already has the table.
	for run := 0; run < 2; run++ {
		tx, err := reopened.writerDB.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if err := createFindingHistoryTable(tx); err != nil {
			_ = tx.Rollback()
			t.Fatalf("createFindingHistoryTable run %d: %v", run, err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	if got := readHistory(t, reopened, "r1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("history after re-running the step = %#v, want %#v", got, want)
	}
}
