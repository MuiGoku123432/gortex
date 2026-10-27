package store_sqlite

import (
	"database/sql"

	"github.com/zzet/gortex/internal/graph"
)

// findingHistoryTableBody is the append-only finding lifecycle log (D-06).
// seq orders one repository's rows in a generation; no wall-clock time is
// stored, so a row carries no volatile value. class, kind, container, and
// start_byte let the next ledger diff pair a gone finding without its node.
const findingHistoryTableBody = ` (
    view_gen          INTEGER NOT NULL DEFAULT 0,
    repo_prefix       TEXT NOT NULL,
    seq               INTEGER NOT NULL,
    finding_id        TEXT NOT NULL,
    prior_status      TEXT NOT NULL DEFAULT '',
    new_status        TEXT NOT NULL,
    successor_id      TEXT NOT NULL DEFAULT '',
    class             TEXT NOT NULL DEFAULT '',
    kind              TEXT NOT NULL DEFAULT '',
    container         TEXT NOT NULL DEFAULT '',
    start_byte        INTEGER NOT NULL DEFAULT -1,
    evidence_digest   TEXT NOT NULL DEFAULT '',
    parser_tool_id    TEXT NOT NULL DEFAULT '',
    parser_grammar_id TEXT NOT NULL DEFAULT '',
    extractor_version TEXT NOT NULL DEFAULT '',
    workspace_id      TEXT NOT NULL DEFAULT '',
    project_id        TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (view_gen, repo_prefix, seq)
) WITHOUT ROWID`

// findingHistoryByFindingIndexSQL serves the latest-row-per-finding read, so
// the ledger never scans a repository's whole log.
const findingHistoryByFindingIndexSQL = `CREATE INDEX IF NOT EXISTS finding_history_by_finding
    ON finding_history(view_gen, repo_prefix, finding_id, seq)`

// findingHistoryChunk bounds rows per multi-row INSERT: 17 parameters per
// row, 50 rows = 850 host parameters, under SQLite's conservative 999.
const findingHistoryChunk = 50

// findingHistoryColumns lists every non-key column of a row aliased h.
const findingHistoryColumns = `h.finding_id, h.prior_status, h.new_status, h.successor_id,
h.class, h.kind, h.container, h.start_byte, h.evidence_digest, h.parser_tool_id,
h.parser_grammar_id, h.extractor_version, h.workspace_id, h.project_id`

// createFindingHistoryTable is the v23 migration and is idempotent: it adds
// the table and its finding index to an older store, which starts with an
// empty history (D-07).
func createFindingHistoryTable(tx *sql.Tx) error {
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS finding_history" + findingHistoryTableBody); err != nil {
		return err
	}
	_, err := tx.Exec(findingHistoryByFindingIndexSQL)
	return err
}

// FindingHistory returns one repository's finding history in append order.
func (s *Store) FindingHistory(repoPrefix string) ([]graph.FindingTransition, error) {
	return s.queryFindingHistory(repoPrefix, `SELECT `+findingHistoryColumns+` FROM finding_history h
WHERE h.view_gen = ? AND h.repo_prefix = ? ORDER BY h.seq`, s.viewGen, repoPrefix)
}

// LatestFindingTransitions returns each finding's latest history row, ordered
// by finding ID, in one query: the grouped subquery is served by
// finding_history_by_finding and the join back by the primary key. CROSS JOIN
// pins that loop order, so a store without planner statistics for the table
// cannot fall back to rescanning the subquery once per history row.
func (s *Store) LatestFindingTransitions(repoPrefix string) ([]graph.FindingTransition, error) {
	return s.queryFindingHistory(repoPrefix, `SELECT `+findingHistoryColumns+`
FROM (SELECT finding_id, MAX(seq) AS seq FROM finding_history
      WHERE view_gen = ? AND repo_prefix = ? GROUP BY finding_id) latest
CROSS JOIN finding_history h
  ON h.view_gen = ? AND h.repo_prefix = ? AND h.seq = latest.seq
ORDER BY h.finding_id`, s.viewGen, repoPrefix, s.viewGen, repoPrefix)
}

func (s *Store) queryFindingHistory(repoPrefix, query string, args ...any) ([]graph.FindingTransition, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []graph.FindingTransition{}
	for rows.Next() {
		row := graph.FindingTransition{RepoPrefix: repoPrefix}
		if err := rows.Scan(&row.FindingID, &row.PriorStatus, &row.NewStatus, &row.SuccessorID,
			&row.Class, &row.Kind, &row.Container, &row.StartByte, &row.EvidenceDigest, &row.ParserToolID,
			&row.ParserGrammarID, &row.ExtractorVersion, &row.WorkspaceID, &row.ProjectID); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// AppendFindingTransitions appends rows to one repository's history in one
// write transaction, numbering them after the repository's highest seq, so a
// failed batch leaves no gap and no partial history.
func (s *Store) AppendFindingTransitions(repoPrefix string, rows []graph.FindingTransition) error {
	if len(rows) == 0 {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.beginWrite()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // rollback after Commit is a no-op
	var seq int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(seq), 0) FROM finding_history WHERE view_gen = ? AND repo_prefix = ?`,
		s.viewGen, repoPrefix).Scan(&seq); err != nil {
		return err
	}
	// 17 parameters per row; findingHistoryChunk keeps a statement bounded.
	for start := 0; start < len(rows); start += findingHistoryChunk {
		batch := rows[start:min(start+findingHistoryChunk, len(rows))]
		stmt := []byte(`INSERT INTO finding_history (view_gen, repo_prefix, seq, finding_id, prior_status,
new_status, successor_id, class, kind, container, start_byte, evidence_digest, parser_tool_id,
parser_grammar_id, extractor_version, workspace_id, project_id) VALUES `)
		args := make([]any, 0, len(batch)*17)
		for i, row := range batch {
			if i > 0 {
				stmt = append(stmt, ',')
			}
			stmt = append(stmt, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"...)
			seq++
			args = append(args, s.viewGen, repoPrefix, seq, row.FindingID, row.PriorStatus,
				row.NewStatus, row.SuccessorID, row.Class, row.Kind, row.Container, row.StartByte,
				row.EvidenceDigest, row.ParserToolID, row.ParserGrammarID, row.ExtractorVersion,
				row.WorkspaceID, row.ProjectID)
		}
		if _, err := tx.Exec(string(stmt), args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

var (
	_ graph.FindingHistoryReader   = (*Store)(nil)
	_ graph.FindingHistoryAppender = (*Store)(nil)
)
