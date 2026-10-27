package cobolprobe

import (
	"context"
	"strings"
	"testing"

	cobolforest "github.com/alexaandru/go-sitter-forest/cobol"
	sitter "github.com/zzet/gortex/internal/parser/tsitter"
)

func parseStats(t *testing.T, src string) (errs, kinds int, top string) {
	p := sitter.NewParser()
	defer p.Close()
	p.SetLanguage(sitter.NewLanguage(cobolforest.GetLanguage()))
	tree, err := p.ParseCtx(context.Background(), nil, []byte(src))
	if err != nil || tree == nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Close()
	counts := map[string]int{}
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.IsError() {
			errs++
		}
		counts[n.Type()]++
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	best, bn := "", 0
	for k, v := range counts {
		if v > bn {
			best, bn = k, v
		}
	}
	return errs, counts["data_description"] + counts["data_description_entry"], best
}

func TestHypothesisIdentificationParagraphs(t *testing.T) {
	body := `       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01  WS-FLAG              PIC X(01).
       01  WS-COUNT             PIC 9(04).
       PROCEDURE DIVISION.
       0001-MAIN.
           MOVE 'Y' TO WS-FLAG.
           PERFORM 0002-WORK.
       0002-WORK.
           STOP RUN.
`
	cases := []struct {
		name string
		head string
	}{
		{"minimal", "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. T.\n"},
		{"+INSTALLATION", "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. T.\n       INSTALLATION.  ALLTEL CORP - DCRIS PROJECT.\n"},
		{"+AUTHOR", "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. T.\n       AUTHOR. SOMEBODY.\n"},
		{"+DATE-WRITTEN", "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. T.\n       DATE-WRITTEN. 01/01/99.\n"},
		{"+banner comment", "       IDENTIFICATION DIVISION.\n      *----------------\n       PROGRAM-ID. T.\n"},
		{"+cols1-6 marker", "I1035  IDENTIFICATION DIVISION.\nI1035  PROGRAM-ID. T.\n"},
	}
	t.Logf("%-18s %6s %11s   %s", "head", "errs", "dataItems", "dominant node")
	for _, c := range cases {
		errs, di, top := parseStats(t, c.head+body)
		t.Logf("%-18s %6d %11d   %s", c.name, errs, di, top)
	}

	t.Log("")
	t.Log("--- copybook fragment (no program structure) ---")
	frag := "       01  CUST-REC.\n           05  CUST-ID     PIC X(08).\n           05  CUST-NAME   PIC X(30).\n"
	errs, di, top := parseStats(t, frag)
	t.Logf("%-18s %6d %11d   %s", "bare fragment", errs, di, top)
	errs, di, top = parseStats(t, "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. W.\n       DATA DIVISION.\n       WORKING-STORAGE SECTION.\n"+frag+"       PROCEDURE DIVISION.\n       0001-M.\n           STOP RUN.\n")
	t.Logf("%-18s %6d %11d   %s", "wrapped", errs, di, top)
	_ = strings.TrimSpace
}

func TestHypothesisIDMS(t *testing.T) {
	head := "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. T.\n       DATA DIVISION.\n       WORKING-STORAGE SECTION.\n       01  WS-FLAG   PIC X(01).\n"
	tail := "       PROCEDURE DIVISION.\n       0001-MAIN.\n           MOVE 'Y' TO WS-FLAG.\n           STOP RUN.\n"
	cases := []struct{ name, inject, where string }{
		{"baseline", "", "data"},
		{"COPY IDMS SUBSCHEMA-CTRL", "       COPY IDMS SUBSCHEMA-CTRL.\n", "data"},
		{"COPY IDMS RECORD X", "       COPY IDMS RECORD CUST-REC.\n", "data"},
		{"plain COPY", "       COPY MYBOOK.\n", "data"},
		{"SCHEMA SECTION", "       SCHEMA SECTION.\n       DB DCCSS000 WITHIN CAMSC000.\n", "data"},
		{"OBTAIN/FIND DML", "           OBTAIN FIRST CUST-REC WITHIN CUST-AREA.\n           FIND CALC CUST-REC.\n", "proc"},
		{"READY/FINISH", "           READY.\n           FINISH.\n", "proc"},
		{"EXEC CICS", "           EXEC CICS SEND MAP('M') END-EXEC.\n", "proc"},
		{"EXEC SQL", "           EXEC SQL SELECT 1 INTO :WS-FLAG FROM T END-EXEC.\n", "proc"},
	}
	t.Logf("%-26s %6s %11s   %s", "construct", "errs", "dataItems", "dominant")
	for _, c := range cases {
		src := head + c.inject + tail
		if c.where != "data" {
			src = head + "       PROCEDURE DIVISION.\n       0001-MAIN.\n" + c.inject + "           STOP RUN.\n"
		}
		errs, di, top := parseStats(t, src)
		t.Logf("%-26s %6d %11d   %s", c.name, errs, di, top)
	}
}
