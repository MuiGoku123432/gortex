//go:build darwin || linux

package languages

import (
	"strconv"
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// cobolProcedure emits each named program's divisions, its data and
// environment sections, its procedure sections and paragraphs, its
// statements (cobolStatements), and then its control-flow calls edges
// (cobolFlow; D-01, D-02, D-05).
//
// In a copybook the walk root (the producer's wrapper program) has only its
// procedure division read, and no division node is emitted (D-16): the
// copybook file node is the container, so its top-level sections and
// paragraphs carry only the file defines edge and their IDs start with the
// root, as <path>::@proc/<NAME>.
//
// PURPOSE: paragraph-level control flow that get_callers and get_call_chain
// trace without any tool change.
// RATIONALE: the procedure division is flat in this grammar (Pitfall 1):
// section_header and paragraph_header are siblings of the statements, so
// membership comes from span order, and header names come from the header
// text because the name token is hidden (Pitfall 2). Only explicit
// transfers become edges (D-04); fall-through is left to cobol_ordinal.
// KEYWORDS: cobol, paragraph, section, perform, cobol_flow, cobol_ordinal.
func cobolProcedure(w *cobolWalk) {
	obs := w.h.Facts.Observations
	occurrences := make(map[string]int) // full symbol path -> nodes seen
	symbolFor := func(base string) string {
		occurrences[base]++
		if k := occurrences[base]; k >= 2 {
			return base + "#" + strconv.Itoa(k)
		}
		return base
	}
	ordinals := make(map[string]int) // program symbol -> headers seen

	for div, o := range obs {
		// Every direct *_division child of a named program_definition.
		p, ok := w.parent(div)
		if !ok || obs[p].Kind != "program_definition" || !strings.HasSuffix(o.Kind, "_division") {
			continue
		}
		prog := w.programs[p]
		root := w.root != "" && prog == w.root
		if prog == "" || (root && o.Kind != "procedure_division") {
			continue
		}
		// The procedure container: the division node, else (root) the file
		// node, whose symbol is prog (w.id).
		divID, divSymbol, labelBase := w.filePath, prog, prog+"/"
		if !root {
			divNode := w.symbolNode(symbolFor(prog+"/@"+o.Kind), o.Kind, graph.KindPackage, "division", div)
			divNode.Meta["cobol_division"] = strings.TrimSuffix(o.Kind, "_division")
			cobolStampRange(divNode, o.Original)
			w.defines(divNode.ID, divNode.StartLine)
			w.memberOf(divNode.ID, w.filePath+"::"+prog, divNode.StartLine)
			divID, divSymbol, labelBase = divNode.ID, divNode.QualName, prog+"/@proc/"
		}
		if o.Kind != "procedure_division" {
			// Data and environment sections are containers (D-01).
			for _, c := range w.children[div] {
				kind := obs[c].Kind
				if !strings.HasSuffix(kind, "_section") {
					continue
				}
				sect := w.symbolNode(symbolFor(prog+"/@"+kind), kind, graph.KindPackage, "section", c)
				sect.Meta["cobol_section_kind"] = strings.TrimSuffix(kind, "_section")
				cobolStampRange(sect, obs[c].Original)
				w.defines(sect.ID, sect.StartLine)
				w.memberOf(sect.ID, divID, sect.StartLine)
				w.sections[c] = sect
			}
			continue
		}
		w.container[div], w.flowFrom[div] = divSymbol, prog

		// An open section or paragraph: its node, header, and last division
		// child so far (whose span covers its own descendants).
		type open struct {
			node         *graph.Node
			header, last int
		}
		var sect, para *open
		finish := func(c *open, parentID string) {
			if c == nil {
				return
			}
			cobolStampExtent(c.node, obs[c.header].Original, obs[c.last].Original)
			w.defines(c.node.ID, c.node.StartLine)
			if parentID != w.filePath {
				w.memberOf(c.node.ID, parentID, c.node.StartLine)
			}
		}
		sectionID := func() string {
			if sect != nil {
				return sect.node.ID
			}
			return divID
		}
		for _, c := range w.children[div] {
			kind := obs[c].Kind
			if kind == "section_header" || kind == "paragraph_header" {
				fields := strings.Fields(w.text(c))
				if len(fields) == 0 {
					continue
				}
				name := strings.TrimSuffix(fields[0], ".")
				finish(para, sectionID())
				para = nil
				base := labelBase + cobolIDSegment(name)
				cobolKind := "paragraph"
				if kind == "section_header" {
					finish(sect, divID)
					sect = nil
					cobolKind = "section"
				} else if sect != nil {
					base = sect.node.QualName + "/" + cobolIDSegment(name)
				}
				node := w.symbolNode(symbolFor(base), name, graph.KindFunction, cobolKind, c)
				ordinals[prog]++
				node.Meta["cobol_ordinal"] = ordinals[prog]
				if kind == "section_header" {
					node.Meta["cobol_section_kind"] = "procedure"
					sect = &open{node: node, header: c, last: c}
				} else {
					para = &open{node: node, header: c, last: c}
				}
			}
			if sect != nil {
				sect.last = c
			}
			if para != nil {
				para.last = c
			}
			container, flow := divSymbol, prog
			if sect != nil {
				container, flow = sect.node.QualName, sect.node.QualName
			}
			if para != nil {
				container, flow = para.node.QualName, para.node.QualName
			}
			for i := c; i <= w.last[c]; i++ {
				w.container[i], w.flowFrom[i] = container, flow
			}
		}
		finish(para, sectionID())
		finish(sect, divID)
	}

	// Statements are registered before any relationship edge is built, so
	// relationEdge can point each edge back to its statement.
	cobolStatements(w)

	// Flow targets are matched after every header is known, since a PERFORM
	// or GO TO may name a paragraph defined later in the program.
	cobolFlow(w)
}
