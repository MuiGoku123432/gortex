package graph

import "sort"

// FindingTransition is one finding lifecycle row: a finding moving from
// PriorStatus ("" for a first open) to NewStatus, with the scope, the
// pairing key (Container, Class, Kind, StartByte), and the evidence and
// parser stamps recorded at that moment. SuccessorID is set only on a
// superseded row. StartByte is -1 when the finding has no range.
type FindingTransition struct {
	FindingID        string
	RepoPrefix       string
	WorkspaceID      string
	ProjectID        string
	PriorStatus      string
	NewStatus        string
	SuccessorID      string
	Class            string
	Kind             string
	Container        string
	StartByte        int
	EvidenceDigest   string
	ParserToolID     string
	ParserGrammarID  string
	ExtractorVersion string
}

// FindingHistoryReader is an optional, generation-scoped capability that
// reads one repository's finding history. Callers probe for it; stores
// without it keep no history.
//
// FindingHistory returns the full log in append order. LatestFindingTransitions
// returns one row per finding, its latest, ordered by FindingID; the finding
// ledger reads only this projection, so a pass never scans the whole log.
type FindingHistoryReader interface {
	FindingHistory(repoPrefix string) ([]FindingTransition, error)
	LatestFindingTransitions(repoPrefix string) ([]FindingTransition, error)
}

// FindingHistoryAppender is an optional capability that appends finding
// lifecycle rows to one repository's history, in slice order. Callers probe
// for it; an empty batch is a no-op.
type FindingHistoryAppender interface {
	AppendFindingTransitions(repoPrefix string, rows []FindingTransition) error
}

// FindingLedgerTransitions returns the lifecycle rows that move a
// repository's recorded finding history to the given active set, sorted by
// FindingID, one row per finding, or nil when nothing changed.
//
// PURPOSE — history is the log in append order (or only its latest row per
// finding; both give the same result). A finding is open when its latest row
// is active. Each active finding that is not open gets an open row whose
// PriorStatus is its latest status or "" (so a closed finding that returns is
// reopened under the same ID). Each open finding absent from active is gone:
// it is superseded by an unpaired newly opened finding with the same
// (Container, Class, Kind), pairing both sides in (StartByte, FindingID)
// order, and closed otherwise. Open rows copy the active record; close and
// supersede rows copy the gone finding's last recorded row.
//
// RATIONALE — D-03: the diff never pairs across containers, classes, or
// kinds and never guesses; leftovers close or open without a predecessor.
// D-02: only IDs are compared, never parser or extractor stamps, so a
// re-stamped finding with an unchanged ID writes no row (GAP-06). D-06/D-07:
// the rows feed a non-canonical, best-effort history log, so this function is
// pure and the caller decides whether a store can keep them.
//
// KEYWORDS — finding, ledger, lifecycle, history, open, close, supersede,
// reopen, diff, D-02, D-03, D-06, D-07, GAP-06, GAP-07.
func FindingLedgerTransitions(history, active []FindingTransition) []FindingTransition {
	latest := make(map[string]FindingTransition, len(history))
	for _, row := range history {
		latest[row.FindingID] = row
	}
	activeIDs := make(map[string]bool, len(active))

	type pairKey struct{ container, class, kind string }
	opened := map[pairKey][]FindingTransition{}
	gone := map[pairKey][]FindingTransition{}
	var out []FindingTransition

	for _, record := range active {
		if activeIDs[record.FindingID] {
			continue
		}
		activeIDs[record.FindingID] = true
		prior, seen := latest[record.FindingID]
		if seen && prior.NewStatus == FindingStatusActive {
			continue
		}
		record.PriorStatus = ""
		if seen {
			record.PriorStatus = prior.NewStatus
		}
		record.NewStatus = FindingStatusActive
		record.SuccessorID = ""
		out = append(out, record)
		key := pairKey{record.Container, record.Class, record.Kind}
		opened[key] = append(opened[key], record)
	}
	for id, row := range latest {
		if row.NewStatus != FindingStatusActive || activeIDs[id] {
			continue
		}
		key := pairKey{row.Container, row.Class, row.Kind}
		gone[key] = append(gone[key], row)
	}

	sourceOrder := func(rows []FindingTransition) {
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].StartByte != rows[j].StartByte {
				return rows[i].StartByte < rows[j].StartByte
			}
			return rows[i].FindingID < rows[j].FindingID
		})
	}
	for key, olds := range gone {
		news := opened[key]
		sourceOrder(olds)
		sourceOrder(news)
		for i, row := range olds {
			row.PriorStatus = FindingStatusActive
			row.NewStatus = FindingStatusClosed
			row.SuccessorID = ""
			if i < len(news) {
				row.NewStatus = FindingStatusSuperseded
				row.SuccessorID = news[i].FindingID
			}
			out = append(out, row)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].FindingID < out[j].FindingID })
	return out
}
