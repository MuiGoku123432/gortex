package store_sqlite

import (
	"database/sql"
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// PrefixDiagnostics answers graph.PrefixDiagnosticsReader natively.
//
// The predicates below are the SQL transcription of
// graph.IsAuditableRepoSourceNode and graph.ClassifyNodePrefix. The
// exemption terms are generated once from graph.OwnershipExemptKinds,
// graph.OwnershipExemptPathPrefixes and graph.OwnershipGlobalIDNamespaces, so
// adding an exemption in Go changes both backends; the storetest
// testPrefixDiagnostics fence is the parity test that proves they agree.
//
// Each population is answered by one aggregate plus one bounded sample
// query rather than a node pull: on a warm multi-repo store the node table
// runs to millions of rows, and this is called from health and boot paths
// that must not materialise it.
var (
	// The audited population. Prefix checks use instr(...)=1 rather than
	// LIKE so `_` and `%` in a path or ID are never wildcards. The list
	// values are compile-time constants from package graph, never user
	// input; every runtime value still binds as a parameter.
	auditableRepoSourceNodePredicate = buildAuditableRepoSourceNodePredicate()

	// A code node no repository claims.
	unownedCodeNodePredicate = `repo_prefix = '' AND (` + auditableRepoSourceNodePredicate + `)`

	// An owned node whose minted identity does not carry the prefix its
	// repo_prefix column claims. instr(x, y) is 1 when y starts x.
	misprefixedNodePredicate = `repo_prefix <> '' AND (` + auditableRepoSourceNodePredicate + `) AND (` +
		`instr(file_path, repo_prefix || '/') <> 1 OR instr(id, repo_prefix || '/') <> 1)`

	// A code node whose repository claim and minted identity agree. The
	// negation of misprefixedNodePredicate over the same owned population,
	// so owned + misprefixed + unowned partitions every audited source node.
	ownedCodeNodePredicate = `repo_prefix <> '' AND (` + auditableRepoSourceNodePredicate + `) AND ` +
		`instr(file_path, repo_prefix || '/') = 1 AND instr(id, repo_prefix || '/') = 1`
)

// buildAuditableRepoSourceNodePredicate renders graph.IsAuditableRepoSourceNode
// as SQL from the exported Ownership* exemption lists.
func buildAuditableRepoSourceNodePredicate() string {
	kinds := make([]string, 0, len(graph.OwnershipExemptKinds))
	for _, k := range graph.OwnershipExemptKinds {
		kinds = append(kinds, quoteSQLLiteral(string(k)))
	}
	terms := []string{`kind NOT IN (` + strings.Join(kinds, ", ") + `)`}
	for _, p := range graph.OwnershipExemptPathPrefixes {
		terms = append(terms, `instr(file_path, `+quoteSQLLiteral(p)+`) <> 1`)
	}
	namespaces := make([]string, 0, len(graph.OwnershipGlobalIDNamespaces))
	for _, ns := range graph.OwnershipGlobalIDNamespaces {
		namespaces = append(namespaces, `instr(id, `+quoteSQLLiteral(ns)+`) = 1`)
	}
	terms = append(terms, `NOT (file_path = '' AND (`+
		`(repo_prefix = '' AND (`+strings.Join(namespaces, " OR ")+`)) OR `+
		`(repo_prefix <> '' AND instr(id, repo_prefix || '::') = 1)))`)
	return strings.Join(terms, " AND ")
}

func (s *Store) PrefixDiagnostics(sampleLimit int) graph.PrefixDiagnostics {
	var d graph.PrefixDiagnostics
	d.OwnedCodeNodes, _ = s.countAndSampleNodes(ownedCodeNodePredicate, 0)
	d.UnownedCodeNodes, d.UnownedSamples = s.countAndSampleNodes(unownedCodeNodePredicate, sampleLimit)
	d.MisprefixedNodes, d.MisprefixedSamples = s.countAndSampleNodes(misprefixedNodePredicate, sampleLimit)
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM nodes WHERE view_gen = ?`, s.viewGen).Scan(&d.Scanned); err != nil {
		panicOnFatal(err)
	}
	return d
}

// countAndSampleNodes returns the number of nodes matching predicate plus up
// to sampleLimit of their IDs. A query failure reports zero rather than
// panicking on a non-fatal error: this is a diagnostic, and a health probe
// that takes the daemon down would be worse than one that stays quiet.
func (s *Store) countAndSampleNodes(predicate string, sampleLimit int) (count int, samples []string) {
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM nodes WHERE `+predicate+` AND view_gen = ?`, s.viewGen).Scan(&count); err != nil {
		panicOnFatal(err)
		return 0, nil
	}
	if count == 0 || sampleLimit <= 0 {
		return count, nil
	}
	rows, err := s.db.Query(`SELECT id FROM nodes WHERE `+predicate+` AND view_gen = ? LIMIT ?`, s.viewGen, sampleLimit)
	if err != nil {
		panicOnFatal(err)
		return count, nil
	}
	defer rows.Close()
	samples = collectStringColumn(rows, sampleLimit)
	return count, samples
}

func collectStringColumn(rows *sql.Rows, capHint int) []string {
	out := make([]string, 0, capHint)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			panicOnFatal(err)
			return out
		}
		out = append(out, v)
	}
	return out
}
