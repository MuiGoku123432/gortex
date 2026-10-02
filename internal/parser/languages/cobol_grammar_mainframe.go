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
// named program's procedure division (DET-07..DET-10).
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
		if w.programSymbol(i) == "" || w.flowFrom[i] == "" || len(w.children[i]) == 0 {
			continue
		}
		from := w.filePath + "::" + w.flowFrom[i]
		switch o.Kind {
		case "idms_navigation_statement", "idms_update_statement", "idms_session_statement", "idms_accept_statement":
			cobolIDMS(w, from, i)
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
			to := w.resourceNode("idms::_unqualified/"+role+"/"+key, w.text(c), graph.KindResource,
				map[string]any{"resource_type": "idms_" + role, "cobol_name_norm": cobolNameNormRule})
			edges = append(edges, w.relationEdge(from, to, kind, stmt))
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
