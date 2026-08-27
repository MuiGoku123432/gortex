package cobolprobe

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	cobolforest "github.com/alexaandru/go-sitter-forest/cobol"
	sitter "github.com/zzet/gortex/internal/parser/tsitter"
)

var neutCorpus = flag.String("neut-corpus", "", "COBOL directory to probe with neutralization")

var (
	// The four construct families measured to derail this grammar. Each is
	// replaced by a valid COBOL statement of the same shape so the parse
	// survives; the real content is extracted separately by the island
	// regexes, which already handle these correctly.
	reExec   = regexp.MustCompile(`(?is)EXEC\s+(CICS|SQL|DLI)\b.*?END-EXEC`)
	reDML    = regexp.MustCompile(`(?im)^(\s*)(OBTAIN|FIND|GET|STORE|MODIFY|ERASE|CONNECT|DISCONNECT|READY|FINISH|BIND|ACCEPT|COMMIT|ROLLBACK|KEEP|IF\s+.*\s+MEMBER)\b[^.]*\.`)
	reSchema = regexp.MustCompile(`(?im)^\s*(SCHEMA\s+SECTION\.|DB\s+\S+\s+WITHIN\s+\S+\s*\.)`)
)

// neutralize rewrites the constructs the grammar cannot read into valid COBOL
// of equivalent shape, preserving line count so line numbers stay usable.
func neutralize(src string) string {
	pad := func(orig, repl string) string {
		if n := strings.Count(orig, "\n"); n > 0 {
			return repl + strings.Repeat("\n", n)
		}
		return repl
	}
	src = reExec.ReplaceAllStringFunc(src, func(m string) string { return pad(m, "CONTINUE") })
	src = reSchema.ReplaceAllStringFunc(src, func(m string) string { return pad(m, "") })
	src = reDML.ReplaceAllStringFunc(src, func(m string) string {
		indent := m[:len(m)-len(strings.TrimLeft(m, " \t"))]
		return pad(m, indent+"CONTINUE.")
	})
	return src
}

func TestNeutralizedParseRate(t *testing.T) {
	if *neutCorpus == "" {
		t.Skip("set -neut-corpus")
	}
	lang := sitter.NewLanguage(cobolforest.GetLanguage())
	parse := func(src string) (errs, data, paras int) {
		p := sitter.NewParser()
		defer p.Close()
		p.SetLanguage(lang)
		tree, err := p.ParseCtx(context.Background(), nil, []byte(src))
		if err != nil || tree == nil {
			return -1, 0, 0
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
				data++
			case "paragraph_header":
				paras++
			}
			for i := 0; i < int(n.ChildCount()); i++ {
				walk(n.Child(i))
			}
		}
		walk(tree.RootNode())
		return
	}

	shell := func(body string) string {
		return "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. WRAPPED.\n" +
			"       DATA DIVISION.\n       WORKING-STORAGE SECTION.\n" + body +
			"       PROCEDURE DIVISION.\n       0001-M.\n           STOP RUN.\n"
	}

	var files []string
	_ = filepath.Walk(*neutCorpus, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		if e := strings.ToLower(filepath.Ext(p)); e == ".cbl" || e == ".cpy" {
			files = append(files, p)
		}
		return nil
	})

	type agg struct{ files, cleanBefore, cleanAfter, dataBefore, dataAfter, paraBefore, paraAfter int }
	tot := map[string]*agg{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f))
		if tot[ext] == nil {
			tot[ext] = &agg{}
		}
		a := tot[ext]
		a.files++

		raw := string(b)
		e1, d1, p1 := parse(raw)
		if e1 == 0 {
			a.cleanBefore++
		}
		a.dataBefore += d1
		a.paraBefore += p1

		fixed := neutralize(raw)
		if ext == ".cpy" {
			fixed = shell(fixed)
		}
		e2, d2, p2 := parse(fixed)
		if e2 == 0 {
			a.cleanAfter++
		}
		a.dataAfter += d2
		a.paraAfter += p2
	}

	t.Logf("%-6s %6s | %-18s | %-18s | %s", "ext", "files", "clean before/after", "dataItems b/a", "paragraphs b/a")
	for _, ext := range []string{".cbl", ".cpy"} {
		a := tot[ext]
		if a == nil {
			continue
		}
		t.Logf("%-6s %6d | %7d -> %-8d | %7d -> %-8d | %6d -> %d",
			ext, a.files,
			a.cleanBefore, a.cleanAfter,
			a.dataBefore, a.dataAfter,
			a.paraBefore, a.paraAfter)
		t.Logf("%-6s %6s | %6d%% -> %d%%", "", "", 100*a.cleanBefore/a.files, 100*a.cleanAfter/a.files)
	}
}
