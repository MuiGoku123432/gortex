package cobolprobe

import (
	"context"
	"fmt"
	"strings"
	"testing"

	cobolforest "github.com/alexaandru/go-sitter-forest/cobol"
	sitter "github.com/zzet/gortex/internal/parser/tsitter"
)

// countAll tallies the node kinds that matter for adoption.
func countAll(t *testing.T, src string) (errs, dataItems, paras, commentEntry int) {
	p := sitter.NewParser()
	defer p.Close()
	p.SetLanguage(sitter.NewLanguage(cobolforest.GetLanguage()))
	tree, err := p.ParseCtx(context.Background(), nil, []byte(src))
	if err != nil || tree == nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Close()
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.IsError() {
			errs++
		}
		switch n.Type() {
		case "data_description", "data_description_entry":
			dataItems++
		case "paragraph_header":
			paras++
		case "comment_entry":
			commentEntry++
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return
}

// TestErrorCascade asks the question that decides adoption: when the grammar
// hits a construct it does not know (EXEC CICS, IDMS DML), does it lose only
// that statement, or everything after it?
func TestErrorCascade(t *testing.T) {
	var data, proc strings.Builder
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&data, "       01  WS-FIELD-%02d          PIC X(10).\n", i)
	}
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&proc, "       %04d-PARA.\n           MOVE 'X' TO WS-FIELD-01.\n", i)
	}
	head := "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. T.\n       DATA DIVISION.\n       WORKING-STORAGE SECTION.\n"
	mid := "       PROCEDURE DIVISION.\n"

	clean := head + data.String() + mid + proc.String() + "           STOP RUN.\n"
	e, d, pa, ce := countAll(t, clean)
	t.Logf("%-34s errs=%-3d data=%-3d paras=%-3d commentEntry=%d", "clean (20 data, 20 paras)", e, d, pa, ce)

	// Inject an unknown construct after the FIRST paragraph. If recovery is
	// local, the remaining 19 paragraphs still parse.
	lines := strings.SplitN(proc.String(), "\n", 3)
	injectedProc := lines[0] + "\n" + lines[1] + "\n           EXEC CICS SEND MAP('M') END-EXEC.\n" + lines[2]
	hurt := head + data.String() + mid + injectedProc + "           STOP RUN.\n"
	e2, d2, pa2, ce2 := countAll(t, hurt)
	t.Logf("%-34s errs=%-3d data=%-3d paras=%-3d commentEntry=%d", "EXEC CICS after para 1", e2, d2, pa2, ce2)

	// And in the DATA DIVISION, where a SCHEMA SECTION would sit.
	hurt2 := head + "       SCHEMA SECTION.\n       DB DCCSS000 WITHIN CAMSC000.\n" + data.String() + mid + proc.String() + "           STOP RUN.\n"
	e3, d3, pa3, ce3 := countAll(t, hurt2)
	t.Logf("%-34s errs=%-3d data=%-3d paras=%-3d commentEntry=%d", "SCHEMA SECTION before data", e3, d3, pa3, ce3)

	// IDMS DML, injected in exactly the place the EXEC CICS case injects, so
	// the two are directly comparable. All record and set names below are
	// invented and neutral - nothing is derived from any real estate.
	const idmsStmt = "           OBTAIN FIRST CUSTOMER-REC WITHIN CUST-ORDER-SET.\n"
	injectedProcIDMS := lines[0] + "\n" + lines[1] + "\n" + idmsStmt + lines[2]
	hurt3 := head + data.String() + mid + injectedProcIDMS + "           STOP RUN.\n"
	e4, d4, pa4, ce4 := countAll(t, hurt3)
	t.Logf("%-34s errs=%-3d data=%-3d paras=%-3d commentEntry=%d", "IDMS DML after para 1", e4, d4, pa4, ce4)

	// The other adjacency direction from IDMS-04: a DML statement immediately
	// followed by a data item.
	dataLines := strings.SplitN(data.String(), "\n", 2)
	injectedData := dataLines[0] + "\n" + idmsStmt + dataLines[1]
	hurt4 := head + injectedData + mid + proc.String() + "           STOP RUN.\n"
	e5, d5, pa5, ce5 := countAll(t, hurt4)
	t.Logf("%-34s errs=%-3d data=%-3d paras=%-3d commentEntry=%d", "IDMS DML before data item", e5, d5, pa5, ce5)

	t.Log("")
	t.Logf("recovery after EXEC CICS : %d/%d paragraphs, %d/%d data items", pa2, pa, d2, d)
	t.Logf("recovery after SCHEMA SEC: %d/%d paragraphs, %d/%d data items", pa3, pa, d3, d)
	t.Logf("recovery after IDMS DML  : %d/%d paragraphs, %d/%d data items", pa4, pa, d4, d)
	t.Logf("recovery IDMS DML in data: %d/%d paragraphs, %d/%d data items", pa5, pa, d5, d)

	// --- IDMS-04 gate ---
	//
	// Assertions are deliberately scoped to the IDMS cases only. The EXEC CICS
	// case is Phase 3's gate and SCHEMA SECTION belongs to preprocessing per
	// locked decision D3; hard-asserting either here would fail the build on a
	// known-open problem this phase does not fix.

	// The clean control anchors every parity comparison below. Without this,
	// a regression that lowered the baseline would make a broken parity check
	// look green.
	if d != 20 || pa != 20 {
		t.Fatalf("clean control moved: got %d data items and %d paragraphs, want 20 and 20", d, pa)
	}

	// IDMS DML injected after paragraph 1 must not cascade: every following
	// paragraph and every data item survives at parity with the clean control.
	if pa4 != pa {
		t.Errorf("IDMS DML after para 1 cascaded: got %d paragraphs, want %d (clean control)", pa4, pa)
	}
	if d4 != d {
		t.Errorf("IDMS DML after para 1 cascaded: got %d data items, want %d (clean control)", d4, d)
	}

	// Negative control. OBTAIN is a procedural DML verb, so it is invalid
	// inside WORKING-STORAGE and must NOT parse there - the IDMS statements
	// attach to the procedure-division statement set only. Parity here would
	// mean the grammar had started accepting invalid COBOL, so the gate asserts
	// the error is still reported rather than asserting recovery.
	if e5 == 0 {
		t.Errorf("IDMS DML in WORKING-STORAGE parsed without error: procedural DML is not valid in the DATA DIVISION and must still be reported")
	}
}
