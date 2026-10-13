//go:build darwin || linux

package languages

import (
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// cobolReferences emits the cross-file reference edges of one document:
// every CALL in a named program's procedure division (D-09, D-10) and every
// COPY, Endevor -INC, COPY IDMS, and EXEC SQL INCLUDE (D-11, D-19).
//
// PURPOSE: CALL and COPY are the estate graph's cross-file edges, and each
// must say what kind of evidence it rests on, so a literal target, a dynamic
// identifier, a copybook, and an IDMS dictionary member stay distinct.
// RATIONALE: literal program and copybook targets go to the owned
// unresolved::cobol::program:: and ::copybook:: namespaces, keyed under
// cobol-name-norm/1, and only resolveCobolReferences binds them (exactly one
// same-repository candidate). Dynamic and IDMS targets carry a reason and
// are never bound in Phase 3: no VALUE clause or other heuristic is read.
// Every key segment passes through cobolIDSegment (T-03-22), and the
// regex-era import:: and dyncall:: namespaces are never emitted (T-03-23).
// KEYWORDS: cobol, call, copy, -INC, COPY IDMS, SQL INCLUDE, copybook.
func cobolReferences(w *cobolWalk, data *cobolDataIndex) {
	obs := w.h.Facts.Observations
	for i, o := range obs {
		switch o.Kind {
		case "call_statement":
			prog := w.programSymbol(i)
			if prog == "" || w.flowFrom[i] == "" || len(w.children[i]) == 0 {
				continue
			}
			cobolCallTarget(w, data, w.id(w.flowFrom[i]), i, w.id(prog),
				w.children[i][0], graph.EdgeCalls, "call_target")
		case "copy_statement", "exec_sql_include":
			cobolCopy(w, i)
		}
	}
}

// cobolCallTarget appends the edges for program operand target of statement
// occurrence, from node ID from, in the program with ID programID. A string
// or number literal is a cobol_call=literal edge of kind to
// unresolved::cobol::program::<key>. A qualified_word or bare WORD
// identifier is a cobol_call=dynamic edge to
// unresolved::cobol::dynamic::program::<programID>::<key>[/<qualifier key>…]
// with reason dynamic, plus a cobolDataReference with cobol_ref_role role. A
// function_ is dynamic too, keyed by its text, with no data reference. Every
// call edge records cobol_target_raw (as written) and cobol_name_norm. A
// literal edge also keeps its original target in cobol_placeholder, so the
// resolver can re-decide and the indexer restub it without re-normalizing
// the name (D-18). CICS LINK, XCTL, and PROGRAM operands reuse it (03-08).
func cobolCallTarget(w *cobolWalk, data *cobolDataIndex, from string, occurrence int, programID string, target int, kind graph.EdgeKind, role string) []*graph.Edge {
	raw := w.text(target)
	dynamic := "unresolved::cobol::dynamic::program::" + programID + "::"
	call, to, name := "dynamic", "", ""
	var qualifiers []string
	switch w.h.Facts.Observations[target].Kind {
	case "string", "number":
		call, to = "literal", "unresolved::cobol::program::"+cobolIDSegment(cobolNameNorm(raw))
	case "function_":
		to = dynamic + cobolIDSegment(cobolNameNorm(raw))
	case "qualified_word":
		name, qualifiers = w.qualifiedName(target)
	case "WORD":
		name = raw
	}
	if name != "" {
		to = dynamic + cobolIDSegment(cobolNameNorm(name))
		for _, q := range qualifiers {
			to += "/" + cobolIDSegment(cobolNameNorm(q))
		}
	}
	if to == "" {
		return nil
	}
	edge := w.relationEdge(from, to, kind, occurrence)
	edge.Meta["cobol_call"] = call
	edge.Meta["cobol_target_raw"] = raw
	edge.Meta["cobol_name_norm"] = cobolNameNormRule
	if call == "dynamic" {
		edge.Meta["cobol_unresolved_reason"] = "dynamic"
	} else {
		// The resolver binds a nested program only from its container.
		edge.Meta["cobol_caller_program"] = programID
		edge.Meta["cobol_placeholder"] = to
	}
	edges := []*graph.Edge{edge}
	if name != "" {
		edges = append(edges, cobolDataReference(w, data, from, occurrence, programID, name, qualifiers, role))
	}
	return edges
}

// cobolCopy appends the imports edge of copy_statement or exec_sql_include
// observation i to unresolved::cobol::copybook::<key>, where <key> is
// cobol-name-norm/1 of the member with one trailing .CPY removed. The form
// is copy, inc (the original bytes begin with -INC, which the Endevor
// transform rewrote to COPY), idms (COPY IDMS [kind] name, keyed
// idms/<kind key or _>/<key> with reason idms_dictionary_member), or
// sql_include (SQLCA and SQLDA set sql_include_builtin). Every form but
// idms keeps its original target in cobol_placeholder, and an OF/IN library
// adds cobol_copy_library_key (cobol-name-norm/1), so the resolver re-decides
// and the indexer restubs from the original target (D-18, D-10).
//
// A COPY is a grammar extra whose Parent is arbitrary (Pitfall 3), so its
// program is the innermost named program whose generated span contains its
// start. From is the procedure flow source when it sits in a procedure
// division, else that program, else the file node. A copybook's wrapper
// program is its walk root, so its COPYs come from the file node (D-16).
func cobolCopy(w *cobolWalk, i int) {
	obs := w.h.Facts.Observations
	o := obs[i]
	var words []string // COPY's WORD and string children: member, then library
	var include []string
	for _, c := range w.children[i] {
		switch obs[c].Kind {
		case "WORD", "string":
			words = append(words, w.text(c))
		case "sql_include_name":
			include = append(include, w.text(c))
		}
	}
	form, idmsKind := "copy", ""
	switch {
	case o.Kind == "exec_sql_include":
		// Its INCLUDE keyword is aliased WORD; the member is sql_include_name.
		form, words = "sql_include", include
	case cobolCopyIDMS(w.text(i)):
		form = "idms"
		if len(words) > 1 {
			idmsKind, words = words[0], words[1:]
		}
	case o.Original != nil && int(o.Original.Start.Byte) < len(w.src) &&
		cobolHasPrefixFold(string(w.src[o.Original.Start.Byte:]), "-INC"):
		form = "inc"
	}
	if len(words) == 0 {
		return
	}
	member := words[0]
	key := cobolIDSegment(strings.TrimSuffix(cobolNameNorm(member), ".CPY"))
	if key == "" {
		return
	}
	to := "unresolved::cobol::copybook::" + key
	if form == "idms" {
		kindKey := "_"
		if idmsKind != "" {
			kindKey = cobolIDSegment(cobolNameNorm(idmsKind))
		}
		to = "unresolved::cobol::copybook::idms/" + kindKey + "/" + key
	}

	best := -1
	for p := range w.programs {
		g := obs[p].Generated
		if o.Generated.Start >= g.Start && o.Generated.Start < g.End && (best < 0 || g.Start > obs[best].Generated.Start) {
			best = p
		}
	}
	from := w.filePath
	switch {
	case w.flowFrom[i] != "":
		from = w.id(w.flowFrom[i])
	case best >= 0:
		from = w.id(w.programs[best])
	}

	edge := w.relationEdge(from, to, graph.EdgeImports, i)
	edge.Meta["cobol_copy_form"] = form
	edge.Meta["cobol_copy_member_raw"] = member
	edge.Meta["cobol_name_norm"] = cobolNameNormRule
	if form != "idms" {
		edge.Meta["cobol_placeholder"] = to
	}
	switch {
	case form == "idms":
		edge.Meta["cobol_unresolved_reason"] = "idms_dictionary_member"
		if idmsKind != "" {
			edge.Meta["cobol_copy_idms_kind"] = idmsKind
		}
	case form == "sql_include":
		if key == "SQLCA" || key == "SQLDA" {
			edge.Meta["sql_include_builtin"] = true
		}
	case len(words) > 1:
		edge.Meta["cobol_copy_library"] = words[1]
		edge.Meta["cobol_copy_library_key"] = cobolNameNorm(words[1])
	}
}

// cobolCopyIDMS reports whether a copy observation's text begins with the
// grammar's _COPY_IDMS token, COPY [ \t]+ IDMS, folding ASCII letters only.
func cobolCopyIDMS(text string) bool {
	if !cobolHasPrefixFold(text, "COPY") {
		return false
	}
	rest := strings.TrimLeft(text[len("COPY"):], " \t")
	return len(rest) < len(text)-len("COPY") && cobolHasPrefixFold(rest, "IDMS")
}

// cobolHasPrefixFold reports whether s begins with upper, an upper-case
// ASCII prefix, folding ASCII letters only (never Unicode case folding).
func cobolHasPrefixFold(s, upper string) bool {
	if len(s) < len(upper) {
		return false
	}
	for i := 0; i < len(upper); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if c != upper[i] {
			return false
		}
	}
	return true
}
