//go:build darwin || linux

package languages

import (
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// cobolIDMSKinds maps an IDMS verb (its observation kind) to the edge kinds
// of its record and set operands (D-14 verb table). FIND counts as a read
// although it only establishes currency; BIND, ACCEPT … CURRENCY, and every
// unlisted verb reference their operands (A7: the raw verb is kept, so a
// later reclassification needs no reparse).
var cobolIDMSKinds = map[string][2]graph.EdgeKind{
	"OBTAIN":     {graph.EdgeReads, graph.EdgeReads},
	"FIND":       {graph.EdgeReads, graph.EdgeReads},
	"GET":        {graph.EdgeReads, graph.EdgeReferences},
	"STORE":      {graph.EdgeWrites, graph.EdgeReferences},
	"MODIFY":     {graph.EdgeWrites, graph.EdgeReferences},
	"ERASE":      {graph.EdgeWrites, graph.EdgeReferences},
	"CONNECT":    {graph.EdgeWrites, graph.EdgeWrites},
	"DISCONNECT": {graph.EdgeWrites, graph.EdgeWrites},
}

// cobolMainframe emits the IDMS and CICS edges of every statement in a
// named program's or a procedure copybook's procedure division
// (DET-07..DET-10). From and owner IDs go through w.id, so a copybook
// statement's edges come From its paragraph or the copybook file node.
//
// PURPOSE: IDMS records and sets and CICS programs, transactions, maps, and
// mapsets are the mainframe half of the dependency picture, and each use
// must say how the statement touches the resource.
// RATIONALE: edges are classified by verb or command and keep it as
// written (D-14). Resources are shared KindResource nodes whose qualifiers
// come only from source, else the explicit _unqualified marker: an IDMS
// subschema is never observed and a WITHIN operand's set-or-area role is
// undecidable, so neither is guessed (D-13, D-15). Every key segment passes
// through cobolNameNorm and cobolIDSegment (T-03-25).
// KEYWORDS: cobol, idms, cics, KindResource, resource_type, reads, writes.
func cobolMainframe(w *cobolWalk, data *cobolDataIndex) {
	for i, o := range w.h.Facts.Observations {
		prog := w.programSymbol(i)
		if prog == "" || w.flowFrom[i] == "" || len(w.children[i]) == 0 {
			continue
		}
		from := w.id(w.flowFrom[i])
		switch o.Kind {
		case "idms_navigation_statement", "idms_update_statement", "idms_session_statement", "idms_accept_statement":
			cobolIDMS(w, from, i)
		case "exec_cics_statement":
			cobolCICS(w, data, from, w.id(prog), i)
		}
	}
}

// cobolIDMS appends the edges of IDMS statement stmt in operand source
// order: idms_record_name and idms_set_name operands go to
// idms::_unqualified/{record,set}/<key> with the verb's edge kind, and an
// idms_scope_name (WITHIN set or area) references
// unresolved::cobol::idms_scope::_unqualified/<key> with reason
// scope_role_unknown. The statement node and every edge carry idms_verb as
// written; an idms_area_name is recorded as idms_area on the statement, and
// an ERASE's observed scope keywords as idms_erase_scope on both.
func cobolIDMS(w *cobolWalk, from string, stmt int) {
	obs := w.h.Facts.Observations
	kids := w.children[stmt]
	verb := obs[kids[0]].Kind
	kinds, ok := cobolIDMSKinds[verb]
	if !ok {
		kinds = [2]graph.EdgeKind{graph.EdgeReferences, graph.EdgeReferences}
	}
	var edges []*graph.Edge
	var erase []string
	area := ""
	for _, c := range kids[1:] {
		key := cobolIDSegment(cobolNameNorm(w.text(c)))
		switch obs[c].Kind {
		case "idms_record_name", "idms_set_name":
			role, kind := "record", kinds[0]
			if obs[c].Kind == "idms_set_name" {
				role, kind = "set", kinds[1]
			}
			to := w.resourceNode("idms::_unqualified/"+role+"/"+key, cobolNameNorm(w.text(c)), graph.KindResource,
				map[string]any{"resource_type": "idms_" + role, "cobol_name_norm": cobolNameNormRule})
			edge := w.relationEdge(from, to, kind, stmt)
			edge.Meta["cobol_resource_raw"] = w.text(c)
			edges = append(edges, edge)
		case "idms_scope_name":
			edge := w.relationEdge(from, "unresolved::cobol::idms_scope::_unqualified/"+key, graph.EdgeReferences, stmt)
			edge.Meta["cobol_unresolved_reason"] = "scope_role_unknown"
			edges = append(edges, edge)
		case "idms_area_name":
			area = w.text(c)
		case "ALL", "PERMANENT", "SELECTIVE", "MEMBERS":
			if verb == "ERASE" {
				erase = append(erase, w.text(c))
			}
		}
	}

	written, scope := w.text(kids[0]), strings.Join(erase, " ")
	for _, e := range edges {
		e.Meta["idms_verb"] = written
		e.Meta["cobol_name_norm"] = cobolNameNormRule
		if scope != "" {
			e.Meta["idms_erase_scope"] = scope
		}
	}
	n := w.statementNode(stmt)
	if n == nil {
		return
	}
	n.Meta["idms_verb"] = written
	if area != "" {
		n.Meta["idms_area"] = area
	}
	if scope != "" {
		n.Meta["idms_erase_scope"] = scope
	}
}

// cobolCICS appends the edges of EXEC CICS statement stmt, in option source
// order, from node ID from in the program with ID programID. The command is
// the first WORD after CICS, kept as written in cics_command on the
// statement node and every edge. A PROGRAM operand goes through
// cobolCallTarget (calls for LINK and XCTL, references otherwise), so a
// CICS program is a call target and never a resource (D-10). TRANSID and
// MAPSET operands are references; a MAP is written by SEND, read by
// RECEIVE, and referenced by any other command (D-14).
//
// A literal TRANSID, MAP, or MAPSET is a shared cics::transaction/<key>,
// cics::map/<qualifier>/<key>, or cics::mapset/<key> resource, where a
// map's qualifier is the literal MAPSET key of the same statement, else
// _unqualified: no mapset is defaulted (D-15). A data-name operand goes to
// the per-program dynamic target
// unresolved::cobol::dynamic::{transaction,map,mapset}::<programID>::<key>
// with reason dynamic plus a cics_<role> data reference, and is never bound
// (T-03-26).
func cobolCICS(w *cobolWalk, data *cobolDataIndex, from, programID string, stmt int) {
	obs := w.h.Facts.Observations
	literal := func(c int) bool { k := obs[c].Kind; return k == "string" || k == "number" }
	command, cics, mapset := "", false, "_unqualified"
	for _, c := range w.children[stmt] {
		switch k := obs[c].Kind; {
		case k == "CICS":
			cics = true
		case k == "WORD" && cics && command == "":
			command = w.text(c)
		case k == "cics_mapset_name" && mapset == "_unqualified" && len(w.children[c]) > 0 && literal(w.children[c][0]):
			mapset = cobolIDSegment(cobolNameNorm(w.text(w.children[c][0])))
		}
	}

	norm := cobolNameNorm(command)
	var edges []*graph.Edge
	for _, c := range w.children[stmt] {
		if len(w.children[c]) == 0 {
			continue
		}
		operand := w.children[c][0]
		role, prefix, kind := "", "", graph.EdgeReferences
		switch obs[c].Kind {
		case "cics_program_name":
			if norm == "LINK" || norm == "XCTL" {
				kind = graph.EdgeCalls
			}
			edges = append(edges, cobolCallTarget(w, data, from, stmt, programID, operand, kind, "cics_program")...)
			continue
		case "cics_transaction_name":
			role, prefix = "transaction", "cics::transaction/"
		case "cics_map_name":
			role, prefix = "map", "cics::map/"+mapset+"/"
			switch norm {
			case "SEND":
				kind = graph.EdgeWrites
			case "RECEIVE":
				kind = graph.EdgeReads
			}
		case "cics_mapset_name":
			role, prefix = "mapset", "cics::mapset/"
		default:
			continue
		}
		raw := w.text(operand)
		key := cobolIDSegment(cobolNameNorm(raw))
		switch {
		case literal(operand):
			to := w.resourceNode(prefix+key, cobolNameNorm(raw), graph.KindResource,
				map[string]any{"resource_type": "cics_" + role, "cobol_name_norm": cobolNameNormRule})
			edge := w.relationEdge(from, to, kind, stmt)
			edge.Meta["cobol_resource_raw"] = raw
			edges = append(edges, edge)
		case obs[operand].Kind == "WORD":
			edge := w.relationEdge(from, "unresolved::cobol::dynamic::"+role+"::"+programID+"::"+key, kind, stmt)
			edge.Meta["cobol_unresolved_reason"] = "dynamic"
			edges = append(edges, edge, cobolDataReference(w, data, from, stmt, programID, raw, nil, "cics_"+role))
		}
	}

	for _, e := range edges {
		e.Meta["cics_command"] = command
		e.Meta["cobol_name_norm"] = cobolNameNormRule
	}
	if n := w.statementNode(stmt); n != nil {
		n.Meta["cics_command"] = command
	}
}
