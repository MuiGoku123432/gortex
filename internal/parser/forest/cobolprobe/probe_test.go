// Command-style probe: how much of a real COBOL estate does the vendored
// yutaro-sakamoto/tree-sitter-cobol grammar actually parse?
//
// Run with:
//
//	go test ./internal/parser/forest/cobolprobe/ -run TestCobolParseRate -v \
//	    -corpus ~/repos/mine/cobolCode/cam-corpus-dcc/DCC
//
// This is a measurement harness, not a unit test: it skips when -corpus is
// unset so it never fails CI.
package cobolprobe

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	cobolforest "github.com/alexaandru/go-sitter-forest/cobol"
	sitter "github.com/zzet/gortex/internal/parser/tsitter"
)

var corpus = flag.String("corpus", "", "directory of COBOL source to probe")

type result struct {
	path       string
	ext        string
	lines      int
	errNodes   int
	dataItems  int
	paragraphs int
	isCBAP     bool
	hasIDMS    bool
}

// countKinds walks the whole tree once, tallying error nodes and the two
// node kinds that decide whether this grammar is worth adopting: data
// descriptions (the DATA DIVISION, which the regex extractor has none of)
// and paragraphs.
func countKinds(root *sitter.Node) (errs, dataItems, paras int) {
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.IsError() {
			errs++
		}
		switch k := n.Type(); {
		case strings.Contains(k, "data_description"):
			dataItems++
		case k == "paragraph" || strings.HasSuffix(k, "_paragraph"):
			paras++
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i))
		}
	}
	walk(root)
	return
}

func TestCobolParseRate(t *testing.T) {
	if *corpus == "" {
		t.Skip("set -corpus to a directory of COBOL source")
	}
	lang := sitter.NewLanguage(cobolforest.GetLanguage())

	var files []string
	err := filepath.Walk(*corpus, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".cbl", ".cpy", ".cbp":
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	t.Logf("probing %d files under %s", len(files), *corpus)

	results := make([]result, 0, len(files))
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		p := sitter.NewParser()
		p.SetLanguage(lang)
		tree, perr := p.ParseCtx(context.Background(), nil, src)
		r := result{
			path:    f,
			ext:     strings.ToLower(filepath.Ext(f)),
			lines:   strings.Count(string(src), "\n"),
			isCBAP:  strings.Contains(string(src), "$CBAP"),
			hasIDMS: strings.Contains(strings.ToUpper(string(src)), "COPY IDMS"),
		}
		if perr != nil || tree == nil {
			r.errNodes = -1 // parse failed outright
		} else {
			r.errNodes, r.dataItems, r.paragraphs = countKinds(tree.RootNode())
			tree.Close()
		}
		p.Close()
		results = append(results, r)
	}

	report(t, results)
}

func report(t *testing.T, rs []result) {
	byExt := map[string][]result{}
	for _, r := range rs {
		byExt[r.ext] = append(byExt[r.ext], r)
	}
	exts := make([]string, 0, len(byExt))
	for e := range byExt {
		exts = append(exts, e)
	}
	sort.Strings(exts)

	t.Log("")
	t.Logf("%-6s %6s %8s %8s %9s %9s", "ext", "files", "clean", "clean%", "dataItems", "paras")
	for _, e := range exts {
		g := byExt[e]
		clean, di, pa := 0, 0, 0
		for _, r := range g {
			if r.errNodes == 0 {
				clean++
			}
			di += r.dataItems
			pa += r.paragraphs
		}
		t.Logf("%-6s %6d %8d %7d%% %9d %9d", e, len(g), clean, 100*clean/len(g), di, pa)
	}

	// The two populations that a standard grammar cannot be expected to read.
	var cbap, idms, cbapClean, idmsClean, plain, plainClean int
	for _, r := range rs {
		switch {
		case r.isCBAP:
			cbap++
			if r.errNodes == 0 {
				cbapClean++
			}
		case r.hasIDMS:
			idms++
			if r.errNodes == 0 {
				idmsClean++
			}
		default:
			plain++
			if r.errNodes == 0 {
				plainClean++
			}
		}
	}
	t.Log("")
	t.Logf("%-22s %6s %8s %7s", "population", "files", "clean", "clean%")
	for _, row := range []struct {
		name         string
		total, clean int
	}{
		{"$CBAP macro source", cbap, cbapClean},
		{"COPY IDMS (no CBAP)", idms, idmsClean},
		{"plain COBOL", plain, plainClean},
	} {
		if row.total == 0 {
			t.Logf("%-22s %6d %8s %7s", row.name, 0, "-", "-")
			continue
		}
		t.Logf("%-22s %6d %8d %6d%%", row.name, row.total, row.clean, 100*row.clean/row.total)
	}

	// Worst offenders, to see whether failures cluster on a fixable pattern.
	sort.Slice(rs, func(i, j int) bool { return rs[i].errNodes > rs[j].errNodes })
	t.Log("")
	t.Log("most error nodes:")
	for i := 0; i < 8 && i < len(rs); i++ {
		r := rs[i]
		t.Logf("   %-34s errs=%-6d lines=%-6d cbap=%v idms=%v",
			filepath.Base(r.path), r.errNodes, r.lines, r.isCBAP, r.hasIDMS)
	}
}
