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

	// The remaining three EXEC CICS shapes from D-28, injected in exactly the
	// place the paren-option case above injects so all four are directly
	// comparable. Chosen for measured coverage, not variety:
	//   (b) no-option    - the corpus's most common shape, 1,251 blocks / 43%,
	//                      and the form most likely to break a repeat-based body
	//   (c) bare-option  - covering the bare options measured at 1,586/450/438
	//   (d) multi-line   - covering the 51 blocks with nothing after the opening
	//                      on the first line
	// Every command, map and mapset name below is invented and neutral -
	// nothing is derived from any real estate.
	const cicsNoOptStmt = "           EXEC CICS RETURN END-EXEC.\n"
	injectedProcCicsNoOpt := lines[0] + "\n" + lines[1] + "\n" + cicsNoOptStmt + lines[2]
	hurt5 := head + data.String() + mid + injectedProcCicsNoOpt + "           STOP RUN.\n"
	e6, d6, pa6, ce6 := countAll(t, hurt5)
	t.Logf("%-34s errs=%-3d data=%-3d paras=%-3d commentEntry=%d", "EXEC CICS no-option after para 1", e6, d6, pa6, ce6)

	const cicsBareOptStmt = "           EXEC CICS SEND ERASE END-EXEC.\n"
	injectedProcCicsBareOpt := lines[0] + "\n" + lines[1] + "\n" + cicsBareOptStmt + lines[2]
	hurt6 := head + data.String() + mid + injectedProcCicsBareOpt + "           STOP RUN.\n"
	e7, d7, pa7, ce7 := countAll(t, hurt6)
	t.Logf("%-34s errs=%-3d data=%-3d paras=%-3d commentEntry=%d", "EXEC CICS bare-option after para 1", e7, d7, pa7, ce7)

	const cicsMultiLineStmt = "           EXEC CICS\n               SEND MAP('MENU01')\n               MAPSET('MENUSET')\n           END-EXEC.\n"
	injectedProcCicsMultiLine := lines[0] + "\n" + lines[1] + "\n" + cicsMultiLineStmt + lines[2]
	hurt7 := head + data.String() + mid + injectedProcCicsMultiLine + "           STOP RUN.\n"
	e8, d8, pa8, ce8 := countAll(t, hurt7)
	t.Logf("%-34s errs=%-3d data=%-3d paras=%-3d commentEntry=%d", "EXEC CICS multi-line after para 1", e8, d8, pa8, ce8)

	t.Log("")
	t.Logf("recovery after EXEC CICS : %d/%d paragraphs, %d/%d data items", pa2, pa, d2, d)
	t.Logf("recovery CICS no-option  : %d/%d paragraphs, %d/%d data items", pa6, pa, d6, d)
	t.Logf("recovery CICS bare-option: %d/%d paragraphs, %d/%d data items", pa7, pa, d7, d)
	t.Logf("recovery CICS multi-line : %d/%d paragraphs, %d/%d data items", pa8, pa, d8, d)
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
