//go:build darwin || linux

package languages

import (
	"slices"
	"strconv"
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// cobolLabel is one procedure section or paragraph a PERFORM or GO TO can
// name, in the program's source order.
type cobolLabel struct {
	key       string // cobolNameNorm of the name as written
	symbol    string // QualName (program-relative node symbol)
	isSection bool
	in        int // index of the enclosing section label, or -1
}

// cobolLabelRef is a label as written: its name key, its optional section
// qualifier key (X OF S / X IN S), and its raw text.
type cobolLabelRef struct {
	key, section, raw string
}

// id is the ID-safe key of an unresolved label target: <key>, or
// <section key>/<key> when qualified (T-03-20).
func (r cobolLabelRef) id() string {
	if r.section != "" {
		return cobolIDSegment(r.section) + "/" + cobolIDSegment(r.key)
	}
	return cobolIDSegment(r.key)
}

// cobolFlow emits the calls edges of every PERFORM and GO TO in a named
// program's procedure division (D-02, D-03, D-04).
//
// PURPOSE: paragraph execution paths that get_callers and get_call_chain
// trace without any tool change.
// RATIONALE: labels resolve program-local only, never across programs,
// nested programs included (A4), under cobol-name-norm/1: X OF S names
// paragraph X of section S; an unqualified X prefers the statement's own
// section, else the unique section or paragraph X in the program. Zero or
// several matches go to an unresolved label target with a reason (D-12).
// A copybook's labels resolve over its own headers only, the same way.
// A THRU range expands exactly, or not at all (T-03-19). Only explicit
// transfers become edges: no fall-through, ALTER, STOP RUN, GOBACK, or
// EXIT edge is ever emitted (deferred by CONTEXT).
// KEYWORDS: cobol, perform, thru, goto, depending, cobol_flow, label.
func cobolFlow(w *cobolWalk) {
	obs := w.h.Facts.Observations
	for div, o := range obs {
		prog := w.programSymbol(div)
		if o.Kind != "procedure_division" || prog == "" || w.container[div] == "" {
			continue
		}
		labels := cobolProcedureLabels(w, div)
		for i := div + 1; i <= w.last[div]; i++ {
			kind := obs[i].Kind
			if w.owner[i] != w.owner[div] || (kind != "perform_statement_call_proc" && kind != "goto_statement") {
				continue
			}
			own := -1 // the statement's own section label
			for j, l := range labels {
				if l.symbol == w.flowFrom[i] {
					own = j
					if !l.isSection {
						own = l.in
					}
				}
			}
			if kind == "goto_statement" {
				cobolGoTo(w, i, prog, labels, own)
				continue
			}
			cobolPerform(w, i, prog, labels, own)
		}
	}
}

// cobolProcedureLabels lists the sections and paragraphs headed directly in
// procedure division div, in source order.
func cobolProcedureLabels(w *cobolWalk, div int) []cobolLabel {
	obs := w.h.Facts.Observations
	var labels []cobolLabel
	section := -1
	for _, c := range w.children[div] {
		kind := obs[c].Kind
		if kind != "section_header" && kind != "paragraph_header" {
			continue
		}
		// note: same header-name rule as cobolProcedure (two uses).
		fields := strings.Fields(w.text(c))
		if len(fields) == 0 {
			continue
		}
		l := cobolLabel{key: cobolNameNorm(strings.TrimSuffix(fields[0], ".")), symbol: w.container[c], isSection: kind == "section_header", in: section}
		if l.isSection {
			l.in = -1
			section = len(labels)
		}
		labels = append(labels, l)
	}
	return labels
}

// cobolLabelOf reads label observation i: a qualified_word whose first WORD
// is the name and second the section qualifier, or one or two number
// children read the same way.
func cobolLabelOf(w *cobolWalk, i int) cobolLabelRef {
	parts := w.children[i]
	if len(parts) == 1 && w.h.Facts.Observations[parts[0]].Kind == "qualified_word" {
		parts = w.children[parts[0]]
	}
	ref := cobolLabelRef{key: cobolNameNorm(w.text(i)), raw: w.text(i)}
	if len(parts) > 0 {
		ref.key = cobolNameNorm(w.text(parts[0]))
	}
	if len(parts) > 1 {
		ref.section = cobolNameNorm(w.text(parts[1]))
	}
	return ref
}

// cobolResolve returns the index of the one label ref names for a statement
// whose own section is labels[own] (-1 for none), else -1 with reason
// label_not_found, or ambiguous and the sorted candidate symbols.
func cobolResolve(labels []cobolLabel, ref cobolLabelRef, own int) (int, string, []string) {
	find := func(keep func(cobolLabel) bool) []int {
		var hits []int
		for i, l := range labels {
			if l.key == ref.key && keep(l) {
				hits = append(hits, i)
			}
		}
		return hits
	}
	var hits []int
	switch {
	case ref.section != "":
		hits = find(func(l cobolLabel) bool { return l.in >= 0 && labels[l.in].key == ref.section })
	case own >= 0:
		hits = find(func(l cobolLabel) bool { return l.in == own })
	}
	if len(hits) == 0 && ref.section == "" {
		hits = find(func(cobolLabel) bool { return true })
	}
	switch len(hits) {
	case 0:
		return -1, "label_not_found", nil
	case 1:
		return hits[0], "", nil
	}
	var candidates []string
	for _, h := range hits {
		candidates = append(candidates, labels[h].symbol)
	}
	slices.Sort(candidates)
	return -1, "ambiguous", candidates
}

// cobolFlowEdge appends one calls edge for statement stmt with cobol_flow
// flow, plus the unresolved reason and candidates when reason is set. In a
// copybook a missing label is label_not_in_copybook (D-12).
func cobolFlowEdge(w *cobolWalk, from, to string, stmt int, flow, reason string, candidates []string) *graph.Edge {
	edge := w.relationEdge(from, to, graph.EdgeCalls, stmt)
	edge.Meta["cobol_flow"] = flow
	edge.Meta["cobol_name_norm"] = cobolNameNormRule
	if reason == "label_not_found" && w.root != "" {
		// A copybook does not define the label; an including program may.
		reason = "label_not_in_copybook"
	}
	if reason != "" {
		edge.Meta["cobol_unresolved_reason"] = reason
	}
	if len(candidates) > 0 {
		edge.Meta["cobol_candidate_symbols"] = candidates
	}
	return edge
}

// cobolPerform emits the calls edges of one PERFORM with exactly one
// perform_procedure (perform_option siblings are ignored). One label is a
// cobol_flow=perform edge. Two labels (A THRU C) expand to one
// perform_thru edge per procedure of the endpoints' kind from A through C
// in source order (D-03); a missing or ambiguous endpoint, mixed kinds, or
// C before A instead yields exactly one edge to
// unresolved::cobol::label::<program ID>::<start>::thru::<end> (the program
// ID is the copybook path in a copybook).
func cobolPerform(w *cobolWalk, stmt int, prog string, labels []cobolLabel, own int) {
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
	var refs []cobolLabelRef
	for _, c := range w.children[procs[0]] {
		if obs[c].Kind == "label" {
			refs = append(refs, cobolLabelOf(w, c))
		}
	}
	from := w.id(w.flowFrom[stmt])
	unresolved := "unresolved::cobol::label::" + w.id(prog) + "::"
	switch len(refs) {
	case 1:
		// note: same single-label target rule as cobolGoTo (two uses).
		hit, reason, candidates := cobolResolve(labels, refs[0], own)
		to := unresolved + refs[0].id()
		if hit >= 0 {
			to = w.filePath + "::" + labels[hit].symbol
		}
		cobolFlowEdge(w, from, to, stmt, "perform", reason, candidates)
	case 2:
		start, end := refs[0], refs[1]
		a, reason, candidates := cobolResolve(labels, start, own)
		c := -1
		if reason == "" {
			c, reason, candidates = cobolResolve(labels, end, own)
		}
		if reason == "" && (labels[a].isSection != labels[c].isSection || c < a) {
			reason = "thru_range_invalid"
		}
		var edges []*graph.Edge
		if reason != "" {
			to := unresolved + start.id() + "::thru::" + end.id()
			edges = append(edges, cobolFlowEdge(w, from, to, stmt, "perform_thru", reason, candidates))
		} else {
			for i := a; i <= c; i++ {
				if labels[i].isSection == labels[a].isSection {
					edges = append(edges, cobolFlowEdge(w, from, w.filePath+"::"+labels[i].symbol, stmt, "perform_thru", "", nil))
				}
			}
		}
		for _, e := range edges {
			e.Meta["cobol_thru_start"] = start.raw
			e.Meta["cobol_thru_end"] = end.raw
		}
	}
}

// cobolGoTo emits one cobol_flow=goto calls edge per distinct target listed
// in a GO TO, in order of first listing (D-04). With DEPENDING ON (a direct
// qualified_word child, not wrapped in label) each edge also carries
// cobol_goto_depending, cobol_depending_positions (the 1-based positions
// where its target is listed), and cobol_goto_depending_on (the identifier
// as written). An alterable GO TO, with no label, emits no edge and marks
// its statement node cobol_goto_alterable.
func cobolGoTo(w *cobolWalk, stmt int, prog string, labels []cobolLabel, own int) {
	obs := w.h.Facts.Observations
	var refs []cobolLabelRef
	depending := ""
	for _, c := range w.children[stmt] {
		switch obs[c].Kind {
		case "label":
			refs = append(refs, cobolLabelOf(w, c))
		case "qualified_word":
			depending = w.text(c)
		}
	}
	if len(refs) == 0 {
		if n := w.statementNode(stmt); n != nil {
			n.Meta["cobol_goto_alterable"] = true
		}
		return
	}
	from := w.id(w.flowFrom[stmt])
	byTarget := make(map[string]*graph.Edge)
	for pos, ref := range refs {
		// note: same single-label target rule as cobolPerform (two uses).
		hit, reason, candidates := cobolResolve(labels, ref, own)
		to := "unresolved::cobol::label::" + w.id(prog) + "::" + ref.id()
		if hit >= 0 {
			to = w.filePath + "::" + labels[hit].symbol
		}
		edge := byTarget[to]
		if edge == nil {
			edge = cobolFlowEdge(w, from, to, stmt, "goto", reason, candidates)
			byTarget[to] = edge
		}
		if depending != "" {
			positions, _ := edge.Meta["cobol_depending_positions"].([]string)
			edge.Meta["cobol_depending_positions"] = append(positions, strconv.Itoa(pos+1))
			edge.Meta["cobol_goto_depending"] = true
			edge.Meta["cobol_goto_depending_on"] = depending
		}
	}
}
