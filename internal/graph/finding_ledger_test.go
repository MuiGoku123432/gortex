package graph

import (
	"reflect"
	"testing"
)

// ledgerFinding returns an active finding record in container C, class
// PARSE_UNRESOLVED, kind error, with the given ID and start byte.
func ledgerFinding(id string, start int) FindingTransition {
	return FindingTransition{
		FindingID:        id,
		RepoPrefix:       "r1",
		WorkspaceID:      "ws",
		ProjectID:        "p",
		Class:            FindingClassParse,
		Kind:             "error",
		Container:        "a.cbl::PGM",
		StartByte:        start,
		EvidenceDigest:   "digest-" + id,
		ParserToolID:     "tool/1",
		ParserGrammarID:  "grammar/1",
		ExtractorVersion: "gortex-cobol-grammar/6",
	}
}

// ledgerRow returns rec as a history row moving prior -> next.
func ledgerRow(rec FindingTransition, prior, next, successor string) FindingTransition {
	rec.PriorStatus = prior
	rec.NewStatus = next
	rec.SuccessorID = successor
	return rec
}

func TestFindingLedger_OpensNew(t *testing.T) {
	b := ledgerFinding("b", 40)
	a := ledgerFinding("a", 10)
	got := FindingLedgerTransitions(nil, []FindingTransition{b, a})
	want := []FindingTransition{
		ledgerRow(a, "", FindingStatusActive, ""),
		ledgerRow(b, "", FindingStatusActive, ""),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("transitions = %#v, want %#v", got, want)
	}
}

func TestFindingLedger_ClosesGone(t *testing.T) {
	a := ledgerFinding("a", 10)
	history := []FindingTransition{ledgerRow(a, "", FindingStatusActive, "")}
	got := FindingLedgerTransitions(history, nil)
	want := []FindingTransition{ledgerRow(a, FindingStatusActive, FindingStatusClosed, "")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("transitions = %#v, want %#v", got, want)
	}
}

func TestFindingLedger_SupersedePairsInSourceOrder(t *testing.T) {
	a1, a2 := ledgerFinding("a1", 10), ledgerFinding("a2", 50)
	history := []FindingTransition{
		ledgerRow(a2, "", FindingStatusActive, ""),
		ledgerRow(a1, "", FindingStatusActive, ""),
	}
	// IDs sort against source order, so pairing must follow StartByte.
	b1, b2 := ledgerFinding("z1", 12), ledgerFinding("b2", 60)
	got := FindingLedgerTransitions(history, []FindingTransition{b2, b1})
	want := []FindingTransition{
		ledgerRow(a1, FindingStatusActive, FindingStatusSuperseded, "z1"),
		ledgerRow(a2, FindingStatusActive, FindingStatusSuperseded, "b2"),
		ledgerRow(b2, "", FindingStatusActive, ""),
		ledgerRow(b1, "", FindingStatusActive, ""),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("transitions = %#v, want %#v", got, want)
	}
}

func TestFindingLedger_NeverPairsAcrossContainerOrClass(t *testing.T) {
	gone := ledgerFinding("a", 10)
	cases := map[string]func(*FindingTransition){
		"container": func(f *FindingTransition) { f.Container = "a.cbl::OTHER" },
		"class":     func(f *FindingTransition) { f.Class = FindingClassExternal },
		"kind":      func(f *FindingTransition) { f.Kind = "missing" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			fresh := ledgerFinding("b", 12)
			change(&fresh)
			history := []FindingTransition{ledgerRow(gone, "", FindingStatusActive, "")}
			got := FindingLedgerTransitions(history, []FindingTransition{fresh})
			want := []FindingTransition{
				ledgerRow(gone, FindingStatusActive, FindingStatusClosed, ""),
				ledgerRow(fresh, "", FindingStatusActive, ""),
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("transitions = %#v, want %#v", got, want)
			}
		})
	}
}

func TestFindingLedger_ReopenAfterClose(t *testing.T) {
	a := ledgerFinding("a", 10)
	history := []FindingTransition{
		ledgerRow(a, "", FindingStatusActive, ""),
		ledgerRow(a, FindingStatusActive, FindingStatusClosed, ""),
	}
	got := FindingLedgerTransitions(history, []FindingTransition{a})
	want := []FindingTransition{ledgerRow(a, FindingStatusClosed, FindingStatusActive, "")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("transitions = %#v, want %#v", got, want)
	}
}

func TestFindingLedger_IdenticalInputNoTransitions(t *testing.T) {
	a, b, c := ledgerFinding("a", 10), ledgerFinding("b", 20), ledgerFinding("c", 30)
	history := []FindingTransition{
		ledgerRow(a, "", FindingStatusActive, ""),
		ledgerRow(c, "", FindingStatusActive, ""),
		ledgerRow(c, FindingStatusActive, FindingStatusClosed, ""),
		ledgerRow(b, "", FindingStatusActive, ""),
	}
	if got := FindingLedgerTransitions(history, []FindingTransition{b, a}); got != nil {
		t.Fatalf("transitions = %#v, want nil", got)
	}
	if got := FindingLedgerTransitions(nil, nil); got != nil {
		t.Fatalf("empty transitions = %#v, want nil", got)
	}
}

func TestFindingLedger_RestampSameIDNoTransition(t *testing.T) {
	a := ledgerFinding("a", 10)
	history := []FindingTransition{ledgerRow(a, "", FindingStatusActive, "")}
	restamped := a
	restamped.ParserToolID = "tool/2"
	restamped.ParserGrammarID = "grammar/2"
	restamped.ExtractorVersion = "gortex-cobol-grammar/7"
	restamped.StartByte = 90
	if got := FindingLedgerTransitions(history, []FindingTransition{restamped}); got != nil {
		t.Fatalf("transitions = %#v, want nil (D-02: stamps are not compared)", got)
	}
}

func TestFindingLedger_LatestRowsEqualFullHistory(t *testing.T) {
	a, b, c := ledgerFinding("a", 10), ledgerFinding("b", 20), ledgerFinding("c", 30)
	d := ledgerFinding("d", 25)
	full := []FindingTransition{
		ledgerRow(a, "", FindingStatusActive, ""),
		ledgerRow(b, "", FindingStatusActive, ""),
		ledgerRow(c, "", FindingStatusActive, ""),
		ledgerRow(b, FindingStatusActive, FindingStatusSuperseded, "d"),
		ledgerRow(d, "", FindingStatusActive, ""),
		ledgerRow(a, FindingStatusActive, FindingStatusClosed, ""),
		ledgerRow(a, FindingStatusClosed, FindingStatusActive, ""),
		ledgerRow(c, FindingStatusActive, FindingStatusClosed, ""),
	}
	// The latest row per FindingID, ordered by FindingID, as
	// LatestFindingTransitions returns it.
	latest := []FindingTransition{full[6], full[3], full[7], full[4]}

	e := ledgerFinding("e", 26)
	for _, active := range [][]FindingTransition{
		nil,
		{a, d},
		{b, c},
		{a, c, e},
	} {
		fromFull := FindingLedgerTransitions(full, active)
		fromLatest := FindingLedgerTransitions(latest, active)
		if !reflect.DeepEqual(fromFull, fromLatest) {
			t.Fatalf("active %v: full-log transitions = %#v, latest-row transitions = %#v", ledgerIDs(active), fromFull, fromLatest)
		}
	}
	// Guard against both sides being trivially empty.
	if got := FindingLedgerTransitions(latest, []FindingTransition{a, c, e}); len(got) != 3 {
		t.Fatalf("transitions = %#v, want 3 rows (c reopened, d superseded by e, e opened)", got)
	}
}

func ledgerIDs(records []FindingTransition) []string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.FindingID)
	}
	return ids
}
