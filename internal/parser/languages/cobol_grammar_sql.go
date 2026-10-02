//go:build darwin || linux

package languages

import (
	"slices"
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// cobolSQLKinds maps the parent kind of an sql_source_candidate to the edge
// kind of its table (D-14): the target of INSERT, UPDATE, DELETE, and MERGE
// is written, a FROM, JOIN, or MERGE USING source is read, and the table of
// CREATE, ALTER, or DROP TABLE is referenced (DDL is not a data read or
// write). An unlisted parent references its table.
var cobolSQLKinds = map[string]graph.EdgeKind{
	"sql_insert_statement":       graph.EdgeWrites,
	"sql_update_statement":       graph.EdgeWrites,
	"sql_delete_statement":       graph.EdgeWrites,
	"sql_merge_statement":        graph.EdgeWrites,
	"sql_source_reference":       graph.EdgeReads,
	"sql_using_reference":        graph.EdgeReads,
	"sql_create_table_statement": graph.EdgeReferences,
	"sql_alter_table_statement":  graph.EdgeReferences,
	"sql_drop_table_statement":   graph.EdgeReferences,
}

// cobolSQLTable is the one edge a statement gets for one physical table and
// edge kind: aliases are the statement's alias texts for it in source order.
type cobolSQLTable struct {
	to       string
	kind     graph.EdgeKind
	aliases  []string
	location string
}

// cobolSQL emits the DB2 table edges of every EXEC SQL statement in a named
// program (DET-11, DET-12).
//
// PURPOSE: DB2 table use is the other half of the mainframe dependency
// picture, and a false table would poison every impact query.
// RATIONALE: only sql_source_candidate observations name physical tables,
// and a same-statement CTE name is subtracted first. Each table is a shared
// KindTable db2::<schema key or _unqualified>/<table key> whose schema comes
// only from source (D-13, D-15). Aliases, CTE names, and derived tables are
// statement-scoped Meta, never nodes; EXEC SQL INCLUDE stays on the COPY
// path. Every key passes through the SQL clause of cobol-name-norm/1 and
// cobolIDSegment (T-03-28, T-03-29).
// KEYWORDS: cobol, db2, exec sql, KindTable, sql_verb, reads, writes.
func cobolSQL(w *cobolWalk, data *cobolDataIndex) {
	for i, o := range w.h.Facts.Observations {
		prog := w.programSymbol(i)
		if o.Kind != "exec_sql_statement" || prog == "" {
			continue
		}
		// The pinned grammar admits EXEC SQL statements only in a procedure
		// division; outside one the program is the flow source.
		from := w.filePath + "::" + prog
		if w.flowFrom[i] != "" {
			from = w.filePath + "::" + w.flowFrom[i]
		}
		verb, _ := cobolSQLVerb(w, i)
		n := w.statementNode(i)
		if n != nil && verb != "" {
			n.Meta["sql_verb"] = verb
		}
		for _, t := range cobolSQLTables(w, i, n) {
			edge := w.relationEdge(from, t.to, t.kind, i)
			edge.Meta["sql_verb"] = verb
			edge.Meta["cobol_name_norm"] = cobolNameNormRule
			if len(t.aliases) > 0 {
				edge.Meta["sql_aliases"] = t.aliases
			}
			if t.location != "" {
				edge.Meta["sql_location"] = t.location
			}
		}
	}
}

// cobolSQLVerb returns the verb of EXEC SQL statement stmt as written and
// the observation index of the WORD it came from: the first WORD of the
// statement's sql_* subnode (the SELECT after a WITH clause), else the bare
// WORD after SQL that the fallback form keeps (OPEN, FETCH, COMMIT, …).
func cobolSQLVerb(w *cobolWalk, stmt int) (string, int) {
	obs := w.h.Facts.Observations
	for _, c := range w.children[stmt] {
		k := obs[c].Kind
		if k == "WORD" {
			return w.text(c), c
		}
		if strings.HasPrefix(k, "sql_") && k != "sql_cte_clause" && len(w.children[c]) > 0 {
			if first := w.children[c][0]; obs[first].Kind == "WORD" {
				return w.text(first), first
			}
		}
	}
	return "", -1
}

// cobolSQLTables returns the physical table edges of EXEC SQL statement
// stmt, one per table and kind in first-candidate order, emitting each
// table node through resourceNode. A candidate whose one-part name matches a
// CTE defined in the same statement is no table: it is recorded in
// sql_cte_refs, the CTE names in sql_ctes, and the alias of a derived table
// in sql_derived_aliases, all on statement node n when it exists.
func cobolSQLTables(w *cobolWalk, stmt int, n *graph.Node) []cobolSQLTable {
	obs := w.h.Facts.Observations
	var ctes, cteRefs, derived []string
	cteKeys := make(map[string]bool)
	for j := stmt + 1; j <= w.last[stmt]; j++ {
		if obs[j].Kind != "sql_cte_name" {
			continue
		}
		name := strings.TrimSpace(w.text(j))
		ctes = append(ctes, name)
		cteKeys[cobolSQLName(name)] = true
	}

	var tables []cobolSQLTable
	index := make(map[string]int) // table ID and kind -> tables index
	for j := stmt + 1; j <= w.last[stmt]; j++ {
		p, _ := w.parent(j)
		switch {
		case obs[j].Kind == "sql_table_alias" && obs[p].Kind != "sql_source_candidate":
			derived = append(derived, cobolSQLAlias(w.text(j)))
			continue
		case obs[j].Kind != "sql_source_candidate":
			continue
		}
		var parts []string
		alias := ""
		for _, c := range w.children[j] {
			switch obs[c].Kind {
			case "sql_table_name":
				parts = cobolSQLParts(w.text(c))
			case "sql_table_alias":
				alias = cobolSQLAlias(w.text(c))
			}
		}
		if len(parts) == 0 {
			continue
		}
		table := parts[len(parts)-1]
		if len(parts) == 1 && cteKeys[cobolSQLName(table)] {
			cteRefs = append(cteRefs, table)
			continue
		}
		meta := map[string]any{"table": table, "dialect": "db2", "cobol_name_norm": cobolNameNormRule}
		schemaKey, location := "_unqualified", ""
		if len(parts) >= 2 {
			schema := parts[len(parts)-2]
			meta["schema"] = schema
			schemaKey = cobolIDSegment(cobolSQLName(schema))
			location = strings.Join(parts[:len(parts)-2], ".")
		}
		to := w.resourceNode("db2::"+schemaKey+"/"+cobolIDSegment(cobolSQLName(table)), table, graph.KindTable, meta)
		kind, ok := cobolSQLKinds[obs[p].Kind]
		if !ok {
			kind = graph.EdgeReferences
		}
		key := to + "|" + string(kind)
		k, seen := index[key]
		if !seen {
			k = len(tables)
			index[key] = k
			tables = append(tables, cobolSQLTable{to: to, kind: kind, location: location})
		}
		if alias != "" && !slices.Contains(tables[k].aliases, alias) {
			tables[k].aliases = append(tables[k].aliases, alias)
		}
	}

	if n == nil {
		return tables
	}
	if len(ctes) > 0 {
		n.Meta["sql_ctes"] = ctes
	}
	if len(cteRefs) > 0 {
		n.Meta["sql_cte_refs"] = cteRefs
	}
	if len(derived) > 0 {
		n.Meta["sql_derived_aliases"] = derived
	}
	return tables
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

// cobolSQLAlias returns the identifier of an sql_table_alias text, which
// is always AS followed by the identifier.
func cobolSQLAlias(text string) string {
	if len(text) < 2 {
		return ""
	}
	return strings.TrimSpace(text[2:])
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
