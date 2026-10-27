//go:build darwin || linux

package languages

import (
	"strconv"
	"strings"

	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/handoff"
)

// cobolProgramSymbol is one named program_definition observation and the
// symbol its node ID is built from.
type cobolProgramSymbol struct {
	index  int    // program_definition index in h.Facts.Observations
	name   string // verbatim program_name bytes from h.Generated
	symbol string // cobolIDSegment-encoded names joined by "/", plus "#k" for k >= 2
	// nested: the program was opened inside another program, so its symbol
	// has a container segment; COBOL scoping makes it callable only from its
	// directly containing program, whose symbol is container.
	nested    bool
	container string
}

// cobolProgramSymbols derives each named program's containment-path symbol
// (D-10, ID-02). The grammar's Parent links are flat, since every
// program_definition hangs off the start node, so nesting comes from an END
// PROGRAM name stack instead:
//
//   - Walking observations in order, a program closes every open program
//     that has no END PROGRAM marker left, because a program without a
//     marker cannot contain another. What stays open is its container path.
//   - An END PROGRAM N pops down to the topmost open N. If N is not open,
//     containmentUnresolved is set and the stack is left alone (no guess).
//   - A program that closes without its own END PROGRAM marker while it
//     contains another program (popped by an outer marker, by the walk, or
//     still open at the end) also sets containmentUnresolved: its marker
//     was matched by a different program, so the nesting is a guess.
//   - The k-th program with the same container and name, for k >= 2, gets
//     "#k" in observation order. The first occurrence has no ordinal.
//
// Names in symbols are the verbatim program_name bytes, never case-folded
// (D-20), each passed through cobolIDSegment so a literal PROGRAM-ID such
// as 'A/B' or 'X#2' cannot alias a containment path or a #k ordinal
// (ID-03); name keeps the bytes unencoded. END PROGRAM markers match a program the way COBOL compares
// user-defined words: case-insensitively and ignoring the quotes of a
// literal name, so `PROGRAM-ID. Outer.` is closed by `END PROGRAM OUTER.`.
// A program_definition with no program_name is counted in unnamed and gets
// no symbol. Line numbers and byte offsets never enter a symbol.
func cobolProgramSymbols(h handoff.Handoff) (programs []cobolProgramSymbol, containmentUnresolved bool, unnamed int) {
	obs := h.Facts.Observations
	parentOf := func(i int) (int, bool) {
		p := obs[i].Parent
		return p, p >= 0 && p < len(obs) && p != i
	}
	defNames := make(map[int]string) // program_definition index -> name
	endNames := make(map[int]string) // end_program index -> name
	for i, o := range obs {
		if o.Kind != "program_name" {
			continue
		}
		parent, ok := parentOf(i)
		if !ok {
			continue
		}
		name := string(h.Generated[o.Generated.Start:o.Generated.End])
		switch obs[parent].Kind {
		case "end_program":
			if _, seen := endNames[parent]; !seen {
				endNames[parent] = name
			}
		case "identification_division":
			def, ok := parentOf(parent)
			if !ok || obs[def].Kind != "program_definition" {
				continue
			}
			if _, seen := defNames[def]; !seen {
				defNames[def] = name
			}
		}
	}

	// key is the name END PROGRAM matching compares.
	key := func(name string) string { return strings.ToUpper(strings.Trim(name, `"'`)) }
	remaining := make(map[string]int) // END PROGRAM markers not yet reached, by key
	for _, name := range endNames {
		remaining[key(name)]++
	}
	occurrences := make(map[string]int) // base symbol -> programs seen
	type frame struct {
		cobolProgramSymbol
		hasChildren bool // another program was opened inside this one
	}
	var open []frame
	for i, o := range obs {
		switch o.Kind {
		case "program_definition":
			name, ok := defNames[i]
			if !ok {
				unnamed++
				continue
			}
			for len(open) > 0 && remaining[key(open[len(open)-1].name)] == 0 {
				if open[len(open)-1].hasChildren {
					containmentUnresolved = true
				}
				open = open[:len(open)-1]
			}
			base, container := cobolIDSegment(name), ""
			if len(open) > 0 {
				container = open[len(open)-1].symbol
				base = container + "/" + cobolIDSegment(name)
				open[len(open)-1].hasChildren = true
			}
			occurrences[base]++
			symbol := base
			if k := occurrences[base]; k >= 2 {
				symbol += "#" + strconv.Itoa(k)
			}
			program := cobolProgramSymbol{index: i, name: name, symbol: symbol, nested: container != "", container: container}
			programs = append(programs, program)
			open = append(open, frame{cobolProgramSymbol: program})
		case "end_program":
			name, ok := endNames[i]
			if !ok {
				continue
			}
			remaining[key(name)]--
			top := len(open) - 1
			for top >= 0 && key(open[top].name) != key(name) {
				top--
			}
			if top < 0 {
				containmentUnresolved = true
				continue
			}
			if top < len(open)-1 {
				// The frames above top close without their own marker.
				containmentUnresolved = true
			}
			open = open[:top]
		}
	}
	for _, f := range open {
		if f.hasChildren {
			containmentUnresolved = true
		}
	}
	return programs, containmentUnresolved, unnamed
}

// cobolNameNormRule identifies the versioned label and name matching rule
// cobolNameNorm implements (D-18). Record it wherever a match depends on it.
const cobolNameNormRule = "cobol-name-norm/1"

// cobolNameNorm is the cobol-name-norm/1 matching key (D-18): when raw is
// wrapped in one matching pair of quotes (' or "), drop that pair and
// un-double the embedded quote; then upper-fold ASCII letters only, leaving
// every other byte untouched. Node names and IDs keep the source spelling.
func cobolNameNorm(raw string) string {
	b := []byte(cobolUnquote(raw))
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
	}
	return string(b)
}

// cobolUnquote drops one matching pair of quotes (' or ") around raw and
// un-doubles the embedded quote, keeping every other byte and the case as
// written; any other raw is returned unchanged.
func cobolUnquote(raw string) string {
	if n := len(raw); n >= 2 && (raw[0] == '\'' || raw[0] == '"') && raw[n-1] == raw[0] {
		q := raw[:1]
		return strings.ReplaceAll(raw[1:n-1], q+q, q)
	}
	return raw
}

// cobolSQLName is the SQL clause of cobol-name-norm/1: a delimited "…"
// identifier is kept verbatim with its quotes, and an ordinary identifier
// is ASCII upper-folded.
func cobolSQLName(part string) string {
	if strings.HasPrefix(part, `"`) {
		return part
	}
	return cobolNameNorm(part)
}

// cobolSQLParts splits a table name as written on the dots outside double
// quotes, trimming the space around each part.
func cobolSQLParts(text string) []string {
	var parts []string
	start, quoted := 0, false
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '"':
			quoted = !quoted
		case '.':
			if !quoted {
				parts = append(parts, strings.TrimSpace(text[start:i]))
				start = i + 1
			}
		}
	}
	return append(parts, strings.TrimSpace(text[start:]))
}

// cobolIDSegment percent-encodes the ID metacharacters %, :, /, # and every
// control byte (below 0x20, or 0x7f) in a name-derived ID segment, so source
// text cannot forge an ::unresolved:: marker, an extra path segment, or a
// #k ordinal (Pitfall 9). Every other byte passes through.
func cobolIDSegment(raw string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c == '%' || c == ':' || c == '/' || c == '#' || c < 0x20 || c == 0x7f {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0x0f])
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
