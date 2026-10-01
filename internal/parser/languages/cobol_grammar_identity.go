//go:build !windows

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
	symbol string // containment path joined by "/", plus "#k" for k >= 2
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
// (D-20). END PROGRAM markers match a program the way COBOL compares
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
			base := name
			if len(open) > 0 {
				base = open[len(open)-1].symbol + "/" + name
				open[len(open)-1].hasChildren = true
			}
			occurrences[base]++
			symbol := base
			if k := occurrences[base]; k >= 2 {
				symbol += "#" + strconv.Itoa(k)
			}
			program := cobolProgramSymbol{index: i, name: name, symbol: symbol}
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
