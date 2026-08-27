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

	t.Log("")
	t.Logf("recovery after EXEC CICS : %d/%d paragraphs, %d/%d data items", pa2, pa, d2, d)
	t.Logf("recovery after SCHEMA SEC: %d/%d paragraphs, %d/%d data items", pa3, pa, d3, d)
}
