package graph

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The D-16 finding writer inventory (T-05-35). A finding closes only through
// deterministic evidence, so only the extractor and the resolver may build a
// KindFinding node or write its finding_status, only the indexer's ledger may
// append finding history, and only the resolver (plus this package's
// definitions) may evict findings. No MCP, CLI, or AI package may write,
// re-status, or close a finding. Definitions and reads are never flagged.

// findingWriterRules maps each inventoried rule to the non-test files allowed
// to hold it. A path ending in "/" allows every file below it.
var findingWriterRules = map[string][]string{
	"KindFinding construction": {
		"internal/parser/languages/cobol_grammar_findings.go",
		"internal/resolver/cobol_findings.go",
		// Test-support code: internal/graph/storetest is imported only by
		// _test.go files, so a conformance case that seeds a finding there
		// is test data, not a production writer.
		"internal/graph/storetest/storetest.go",
	},
	"finding_status write": {
		"internal/parser/languages/cobol_grammar_findings.go",
		"internal/resolver/cobol_findings.go",
	},
	"AppendFindingTransitions call": {
		"internal/indexer/finding_ledger.go",
	},
	"EvictFindingNodesByIDs call": {
		"internal/resolver/cobol_findings.go",
		"internal/graph/",
	},
}

// findingWriterRequired lists, per rule, the files that must still hold it,
// so a scanner that silently stopped matching fails instead of passing.
var findingWriterRequired = map[string][]string{
	"KindFinding construction": {
		"internal/parser/languages/cobol_grammar_findings.go",
		"internal/resolver/cobol_findings.go",
	},
	"finding_status write": {
		"internal/parser/languages/cobol_grammar_findings.go",
		"internal/resolver/cobol_findings.go",
	},
	"AppendFindingTransitions call": {"internal/indexer/finding_ledger.go"},
	"EvictFindingNodesByIDs call":   {"internal/resolver/cobol_findings.go"},
}

func TestFindingWriters_Inventory(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("locating repository root: %v", err)
	}

	var offenders []string
	seen := map[string]map[string]bool{}
	for _, top := range []string{"internal", "cmd"} {
		walkErr := filepath.WalkDir(filepath.Join(root, top), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if _, skip := fenceSkippedDirs[d.Name()]; skip {
					return filepath.SkipDir
				}
				nested, gitErr := nestedGitCheckout(root, p)
				if gitErr != nil {
					return gitErr
				}
				if nested {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			rel, relErr := filepath.Rel(root, p)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)
			hits, parseErr := findingWriterHits(p)
			if parseErr != nil {
				t.Errorf("parsing %s: %v", rel, parseErr)
				return nil
			}
			for _, h := range hits {
				if findingWriterAllowed(h.rule, rel) {
					if seen[h.rule] == nil {
						seen[h.rule] = map[string]bool{}
					}
					seen[h.rule][rel] = true
					continue
				}
				offenders = append(offenders, fmt.Sprintf("%s:%d: %s", rel, h.line, h.rule))
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("scanning %s: %v", top, walkErr)
		}
	}

	// WalkDir visits files in lexical order and Inspect visits a file in
	// source order, so offenders are already deterministic.
	if len(offenders) > 0 {
		t.Fatalf("finding written outside the deterministic writers (D-16):\n  %s\n\n"+
			"Only the COBOL extractor and resolver build findings or set finding_status, only "+
			"internal/indexer/finding_ledger.go appends finding history, and only the resolver "+
			"evicts findings. A finding closes only through a source edit, a parser or extractor "+
			"change, or retrieval or binding of its target; no tool may close or re-status one.",
			strings.Join(offenders, "\n  "))
	}
	for rule, files := range findingWriterRequired {
		for _, f := range files {
			if !seen[rule][f] {
				t.Errorf("%s no longer holds a %s: the inventory scanner or its allow list is stale", f, rule)
			}
		}
	}
}

// findingWriterAllowed reports whether rel may hold rule.
func findingWriterAllowed(rule, rel string) bool {
	for _, allowed := range findingWriterRules[rule] {
		if rel == allowed || (strings.HasSuffix(allowed, "/") && strings.HasPrefix(rel, allowed)) {
			return true
		}
	}
	return false
}

// findingWriterHit is one inventoried write in a file.
type findingWriterHit struct {
	rule string
	line int
}

// findingWriterHits parses one Go file and returns its finding writes:
// composite literals with a Kind field of KindFinding, `x.Kind = KindFinding`
// assignments, index assignments or composite-literal keys of the
// finding_status key, and AppendFindingTransitions and EvictFindingNodesByIDs
// call expressions. Constant declarations, comparisons, and right-hand-side
// reads are not writes and never match.
func findingWriterHits(filePath string) ([]findingWriterHit, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var hits []findingWriterHit
	add := func(rule string, n ast.Node) {
		hits = append(hits, findingWriterHit{rule: rule, line: fset.Position(n.Pos()).Line})
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CompositeLit:
			for _, elt := range node.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Kind" && isFindingName(kv.Value, "KindFinding") {
					add("KindFinding construction", kv)
				}
				if isFindingStatusKey(kv.Key) {
					add("finding_status write", kv)
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range node.Lhs {
				switch target := lhs.(type) {
				case *ast.SelectorExpr:
					if target.Sel.Name == "Kind" && len(node.Rhs) == len(node.Lhs) && isFindingName(node.Rhs[i], "KindFinding") {
						add("KindFinding construction", node)
					}
				case *ast.IndexExpr:
					if isFindingStatusKey(target.Index) {
						add("finding_status write", node)
					}
				}
			}
		case *ast.CallExpr:
			for _, name := range []string{"AppendFindingTransitions", "EvictFindingNodesByIDs"} {
				if isFindingName(node.Fun, name) {
					add(name+" call", node)
				}
			}
		}
		return true
	})
	return hits, nil
}

// isFindingName reports whether e is the identifier name, bare or qualified
// (graph.name, x.name).
func isFindingName(e ast.Expr, name string) bool {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name == name
	case *ast.SelectorExpr:
		return v.Sel.Name == name
	}
	return false
}

// isFindingStatusKey reports whether e names the finding status Meta key:
// MetaFindingStatus, graph.MetaFindingStatus, or the literal "finding_status".
func isFindingStatusKey(e ast.Expr) bool {
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		s, err := strconv.Unquote(lit.Value)
		return err == nil && s == MetaFindingStatus
	}
	return isFindingName(e, "MetaFindingStatus")
}
