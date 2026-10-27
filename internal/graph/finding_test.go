package graph

import (
	"regexp"
	"strings"
	"testing"
)

var findingSymbolPattern = regexp.MustCompile(`^@finding/(parse|external)/([0-9a-f]{32})$`)

func TestFindingID_Deterministic(t *testing.T) {
	args := [5]string{"src/a.cbl::PGM/@procedure_division", FindingClassParse, "unparsed-tail", "WORD", FindingEvidenceDigest("ZQXCOL")}
	symbol := func(a [5]string) string { return FindingSymbol(a[0], a[1], a[2], a[3], a[4]) }

	got := symbol(args)
	if again := symbol(args); again != got {
		t.Fatalf("FindingSymbol is not deterministic: got %q then %q", got, again)
	}
	m := findingSymbolPattern.FindStringSubmatch(got)
	if m == nil || m[1] != "parse" {
		t.Fatalf("FindingSymbol = %q, want @finding/parse/<32 lowercase hex>", got)
	}

	for i := range args {
		if i == 1 {
			continue // the class has its own test; any other value panics
		}
		alt := args
		alt[i] += "x"
		if other := symbol(alt); other == got {
			t.Fatalf("changing argument %d to %q kept symbol %q", i, alt[i], got)
		}
	}
}

func TestFindingID_ClassSeparation(t *testing.T) {
	digest := FindingEvidenceDigest("NOPGM")
	parse := FindingSymbol("src/a.cbl::PGM", FindingClassParse, "error", "WORD", digest)
	external := FindingSymbol("src/a.cbl::PGM", FindingClassExternal, "error", "WORD", digest)

	p := findingSymbolPattern.FindStringSubmatch(parse)
	e := findingSymbolPattern.FindStringSubmatch(external)
	if p == nil || e == nil {
		t.Fatalf("symbols %q and %q do not match the finding grammar", parse, external)
	}
	if p[1] != "parse" || e[1] != "external" {
		t.Fatalf("segments = %q, %q, want parse, external", p[1], e[1])
	}
	if p[2] == e[2] {
		t.Fatalf("parse and external share hex %s", p[2])
	}
}

func TestFindingID_LengthFramed(t *testing.T) {
	digest := FindingEvidenceDigest("")
	for _, tc := range []struct {
		name string
		a, b string
	}{
		{"container|kind",
			FindingSymbol("ab", FindingClassParse, "c", "f", digest),
			FindingSymbol("a", FindingClassParse, "bc", "f", digest)},
		{"kind|family",
			FindingSymbol("x", FindingClassParse, "ab", "c", digest),
			FindingSymbol("x", FindingClassParse, "a", "bc", digest)},
		{"family|digest",
			FindingSymbol("x", FindingClassParse, "k", "ab", "c"),
			FindingSymbol("x", FindingClassParse, "k", "a", "bc")},
	} {
		if tc.a == tc.b {
			t.Fatalf("%s: (\"ab\",\"c\") and (\"a\",\"bc\") alias to %q", tc.name, tc.a)
		}
	}
}

func TestFindingEvidenceDigest_Domain(t *testing.T) {
	empty := FindingEvidenceDigest("")
	space := FindingEvidenceDigest(" ")
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(empty) {
		t.Fatalf("FindingEvidenceDigest(\"\") = %q, want 32 lowercase hex", empty)
	}
	if empty == space {
		t.Fatalf("FindingEvidenceDigest(\"\") = FindingEvidenceDigest(\" \") = %q", empty)
	}
	if strings.Contains(FindingEvidenceDigest("ZQXCOL"), "ZQXCOL") {
		t.Fatal("FindingEvidenceDigest leaks its input")
	}
}
