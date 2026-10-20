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
// edge kind: aliases are the statement's alias texts for it in source order,
// and raw is the table name as first written in the statement.
type cobolSQLTable struct {
	to       string
	kind     graph.EdgeKind
	aliases  []string
	location string
	raw      string
}

// cobolSQLOrientation holds the FETCH orientation and option keywords that
// can stand between OPEN, FETCH, or CLOSE and its cursor name.
var cobolSQLOrientation = map[string]bool{
	"NEXT": true, "PRIOR": true, "FIRST": true, "LAST": true, "CURRENT": true,
	"BEFORE": true, "AFTER": true, "ABSOLUTE": true, "RELATIVE": true, "FROM": true,
	"ROWSET": true, "STARTING": true, "AT": true, "WITH": true, "SENSITIVE": true,
	"INSENSITIVE": true, "CONTINUE": true,
}

// cobolSQL emits the DB2 table and dynamic SQL edges of every EXEC SQL
// statement in a named program or a procedure copybook (DET-11, DET-12).
//
// PURPOSE: DB2 table use is the other half of the mainframe dependency
// picture, and a false table would poison every impact query.
// RATIONALE: only sql_source_candidate observations name physical tables,
// and a same-statement CTE name is subtracted first. Each table is a shared
// KindTable db2::<schema key or _unqualified>/<table key> whose schema comes
// only from source (D-13, D-15). Aliases, CTE names, and derived tables are
// statement-scoped Meta, never nodes; EXEC SQL INCLUDE stays on the COPY
// path. OPEN, FETCH, and CLOSE read the tables of the one same-program
// DECLARE of their cursor. Otherwise they are marked ambiguous when the
// cursor is declared more than once (WR-07), else cursor_not_declared, or
// cursor_not_in_copybook in a copybook (D-12).
// PREPARE goes to a per-program dynamic placeholder that is never bound, and
// EXECUTE only marks its statement, so no table is invented for dynamic SQL.
// Every key passes through the SQL clause of cobol-name-norm/1 and
// cobolIDSegment (T-03-28, T-03-29).
// KEYWORDS: cobol, db2, exec sql, KindTable, cursor, dynamic sql, sql_verb.
func cobolSQL(w *cobolWalk, data *cobolDataIndex) {
	obs := w.h.Facts.Observations
	type cursorUse struct {
		stmt                          int
		from, programID, verb, cursor string
	}
	declared := make(map[string][]int)      // program ID + cursor key -> DECLARE statements
	tables := make(map[int][]cobolSQLTable) // DECLARE statement -> its tables
	var uses []cursorUse
	edge := func(from string, t cobolSQLTable, stmt int, verb string) *graph.Edge {
		e := w.relationEdge(from, t.to, t.kind, stmt)
		e.Meta["sql_verb"] = verb
		e.Meta["cobol_name_norm"] = cobolNameNormRule
		e.Meta["cobol_resource_raw"] = t.raw
		if len(t.aliases) > 0 {
			e.Meta["sql_aliases"] = t.aliases
		}
		if t.location != "" {
			e.Meta["sql_location"] = t.location
		}
		return e
	}
	for i, o := range obs {
		prog := w.programSymbol(i)
		if o.Kind != "exec_sql_statement" || prog == "" {
			continue
		}
		// The pinned grammar admits EXEC SQL statements only in a procedure
		// division; outside one the program is the flow source.
		programID := w.id(prog)
		from := programID
		if w.flowFrom[i] != "" {
			from = w.id(w.flowFrom[i])
		}
		verb, at := cobolSQLVerb(w, i)
		n := w.statementNode(i)
		if n != nil && verb != "" {
			n.Meta["sql_verb"] = verb
		}
		sub, rest := "", ""
		if at >= 0 {
			if p := obs[at].Parent; p != i {
				sub = obs[p].Kind
			}
			rest = string(w.h.Generated[obs[at].Generated.End:o.Generated.End])
		}
		found := cobolSQLTables(w, i, n)
		cursor := ""
		switch norm := cobolNameNorm(verb); {
		case sub == "sql_declare_cursor_statement":
			if cursor = cobolSQLCursor(rest); cursor != "" {
				key := programID + "\x00" + cobolSQLName(cursor)
				declared[key] = append(declared[key], i)
				tables[i] = found
			}
		case sub == "" && (norm == "OPEN" || norm == "FETCH" || norm == "CLOSE"):
			uses = append(uses, cursorUse{i, from, programID, verb, cobolSQLCursor(rest)})
		case sub == "" && norm == "EXECUTE":
			if n != nil {
				n.Meta["sql_dynamic"] = true
			}
		case sub == "sql_prepare_statement":
			for _, c := range w.children[obs[at].Parent] {
				if obs[c].Kind != "sql_dynamic_source" {
					continue
				}
				name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(w.text(c)), ":"))
				to := "unresolved::cobol::dynamic::sql::" + programID + "::" + cobolIDSegment(cobolNameNorm(name))
				e := edge(from, cobolSQLTable{to: to, kind: graph.EdgeReferences}, i, verb)
				e.Meta["cobol_unresolved_reason"] = "dynamic"
				cobolDataReference(w, data, from, i, programID, name, nil, "sql_dynamic_source").Meta["sql_verb"] = verb
			}
		}
		for _, t := range found {
			if e := edge(from, t, i, verb); cursor != "" {
				e.Meta["sql_cursor"] = cursor
			}
		}
		if n != nil && cursor != "" {
			n.Meta["sql_cursor"] = cursor
		}
	}

	for _, u := range uses {
		n := w.statementNode(u.stmt)
		if n != nil && u.cursor != "" {
			n.Meta["sql_cursor"] = u.cursor
		}
		decls := declared[u.programID+"\x00"+cobolSQLName(u.cursor)]
		if len(decls) != 1 {
			if n != nil {
				reason := "cursor_not_declared"
				switch {
				case len(decls) > 1:
					// WR-07: the cursor exists, more than once; which DECLARE
					// a use reads is not known statically.
					reason = "ambiguous"
				case w.root != "":
					// A copybook does not declare it; an including program may.
					reason = "cursor_not_in_copybook"
				}
				n.Meta["cobol_unresolved_reason"] = reason
			}
			continue
		}
		for _, t := range tables[decls[0]] {
			read := cobolSQLTable{to: t.to, kind: graph.EdgeReads, location: t.location}
			edge(u.from, read, u.stmt, u.verb).Meta["sql_via_cursor"] = u.cursor
		}
	}
}

// cobolSQLCursor returns the cursor name as written in rest, the statement
// text after its OPEN, FETCH, CLOSE, or DECLARE verb: the first token that
// is no orientation keyword, host variable, or number. It reads the text
// because an underscore name is a hidden token with no observation.
func cobolSQLCursor(rest string) string {
	for _, tok := range strings.Fields(rest) {
		if strings.EqualFold(tok, "END-EXEC") {
			break
		}
		if c := tok[0]; c == ':' || c == '+' || c == '-' || (c >= '0' && c <= '9') || cobolSQLOrientation[strings.ToUpper(tok)] {
			continue
		}
		return tok
	}
	return ""
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
		alias, raw := "", ""
		for _, c := range w.children[j] {
			switch obs[c].Kind {
			case "sql_table_name":
				raw = w.text(c)
				parts = cobolSQLParts(raw)
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
		name := cobolSQLName(table)
		meta := map[string]any{"table": name, "dialect": "db2", "cobol_name_norm": cobolNameNormRule}
		schemaKey, location := "_unqualified", ""
		if len(parts) >= 2 {
			schema := cobolSQLName(parts[len(parts)-2])
			meta["schema"] = schema
			schemaKey = cobolIDSegment(schema)
			location = strings.Join(parts[:len(parts)-2], ".")
		}
		to := w.resourceNode("db2::"+schemaKey+"/"+cobolIDSegment(name), name, graph.KindTable, meta)
		kind, ok := cobolSQLKinds[obs[p].Kind]
		if !ok {
			kind = graph.EdgeReferences
		}
		key := to + "|" + string(kind)
		k, seen := index[key]
		if !seen {
			k = len(tables)
			index[key] = k
			tables = append(tables, cobolSQLTable{to: to, kind: kind, location: location, raw: raw})
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

// cobolSQLAlias returns the identifier of an sql_table_alias text, which
// is always AS followed by the identifier.
func cobolSQLAlias(text string) string {
	if len(text) < 2 {
		return ""
	}
	return strings.TrimSpace(text[2:])
}
