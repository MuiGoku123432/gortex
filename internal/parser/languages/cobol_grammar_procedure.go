//go:build darwin || linux

package languages

import (
	"slices"
	"strconv"
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// cobolLabel is one procedure section or paragraph a PERFORM can name.
type cobolLabel struct {
	key    string // cobolNameNorm of the name as written
	symbol string // QualName (program-relative node symbol)
}

// cobolProcedure emits each named program's divisions, its procedure
// sections and paragraphs, and the calls edges of single-label
// PERFORM statements (D-01, D-02).
//
// PURPOSE: paragraph-level control flow that get_callers and get_call_chain
// trace without any tool change.
// RATIONALE: the procedure division is flat in this grammar (Pitfall 1):
// section_header and paragraph_header are siblings of the statements, so
// membership comes from span order, and header names come from the header
// text because the name token is hidden (Pitfall 2). Only explicit
// transfers become edges (D-04); fall-through is left to cobol_ordinal.
// Labels bind only to exactly one section or paragraph of the same program
// under cobol-name-norm/1; zero or several matches stay on an unresolved
// label target with a reason (D-12). Qualified labels, THRU, and GO TO are
// Plan 03-06's.
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
	labels := make(map[string][]cobolLabel) // program symbol -> labels in source order
	ordinals := make(map[string]int)        // program symbol -> headers seen
	var divisions []int

	for div, o := range obs {
		// Every direct *_division child of a named program_definition.
		p, ok := w.parent(div)
		if !ok || obs[p].Kind != "program_definition" || !strings.HasSuffix(o.Kind, "_division") {
			continue
		}
		prog := w.programs[p]
		if prog == "" {
			continue
		}
		divNode := w.symbolNode(symbolFor(prog+"/@"+o.Kind), o.Kind, graph.KindPackage, "division", div)
		divNode.Meta["cobol_division"] = strings.TrimSuffix(o.Kind, "_division")
		cobolStampRange(divNode, o.Original)
		w.defines(divNode.ID, divNode.StartLine)
		w.memberOf(divNode.ID, w.filePath+"::"+prog, divNode.StartLine)
		if o.Kind != "procedure_division" {
			continue
		}
		divisions = append(divisions, div)

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
			w.memberOf(c.node.ID, parentID, c.node.StartLine)
		}
		sectionID := func() string {
			if sect != nil {
				return sect.node.ID
			}
			return divNode.ID
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
				base := prog + "/@proc/" + cobolIDSegment(name)
				cobolKind := "paragraph"
				if kind == "section_header" {
					finish(sect, divNode.ID)
					sect = nil
					cobolKind = "section"
				} else if sect != nil {
					base = sect.node.QualName + "/" + cobolIDSegment(name)
				}
				node := w.symbolNode(symbolFor(base), name, graph.KindFunction, cobolKind, c)
				ordinals[prog]++
				node.Meta["cobol_ordinal"] = ordinals[prog]
				labels[prog] = append(labels[prog], cobolLabel{key: cobolNameNorm(name), symbol: node.QualName})
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
			container, flow := divNode.QualName, prog
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
		finish(sect, divNode.ID)
	}

	// PERFORM targets are matched after every header is known, since a
	// PERFORM may name a paragraph defined later in the program.
	for _, div := range divisions {
		prog := w.programSymbol(div)
		for i := div + 1; i <= w.last[div]; i++ {
			if obs[i].Kind == "perform_statement_call_proc" && w.owner[i] == w.owner[div] {
				cobolPerform(w, i, prog, labels[prog])
			}
		}
	}
}

// cobolPerform emits the calls edge of one PERFORM whose perform_procedure
// holds exactly one unqualified label (perform_option siblings are
// ignored). The edge goes to the single same-program section or paragraph
// the label names, else to unresolved::cobol::label::<program ID>::<key>
// with cobol_unresolved_reason label_not_found or ambiguous.
func cobolPerform(w *cobolWalk, stmt int, prog string, labels []cobolLabel) {
	obs := w.h.Facts.Observations
	var procs []int
	for _, c := range w.children[stmt] {
		if obs[c].Kind == "perform_procedure" {
			procs = append(procs, c)
		}
	}
	if len(procs) != 1 {
		return
	}
	var targets []int
	for _, c := range w.children[procs[0]] {
		if obs[c].Kind == "label" {
			targets = append(targets, c)
		}
	}
	if len(targets) != 1 {
		return // PERFORM ... THRU: Plan 03-06
	}
	raw := w.text(targets[0])
	if len(strings.Fields(raw)) != 1 {
		return // label IN/OF section: Plan 03-06
	}
	key := cobolNameNorm(raw)
	var matches []string
	for _, l := range labels {
		if l.key == key {
			matches = append(matches, l.symbol)
		}
	}
	from := w.filePath + "::" + w.flowFrom[stmt]
	if len(matches) == 1 {
		edge := w.relationEdge(from, w.filePath+"::"+matches[0], graph.EdgeCalls, stmt)
		edge.Meta["cobol_flow"] = "perform"
		edge.Meta["cobol_name_norm"] = cobolNameNormRule
		return
	}
	target := "unresolved::cobol::label::" + w.filePath + "::" + prog + "::" + cobolIDSegment(key)
	edge := w.relationEdge(from, target, graph.EdgeCalls, stmt)
	edge.Meta["cobol_flow"] = "perform"
	edge.Meta["cobol_name_norm"] = cobolNameNormRule
	if len(matches) == 0 {
		edge.Meta["cobol_unresolved_reason"] = "label_not_found"
		return
	}
	slices.Sort(matches)
	edge.Meta["cobol_unresolved_reason"] = "ambiguous"
	edge.Meta["cobol_candidate_symbols"] = matches
}
