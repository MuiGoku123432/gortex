package store_sqlite

import (
	"database/sql"
	"fmt"
)

// unprefixedFileBackedNodePredicate is the audit's unowned population
// narrowed to rows that name a source file.
var unprefixedFileBackedNodePredicate = `repo_prefix = '' AND file_path <> '' AND (` + auditableRepoSourceNodePredicate + `)`

// purgeUnprefixedRepoRows is the v6→v7 in-place step that retires
// single-repo (unprefixed) mode's on-disk residue.
//
// WHY A MIGRATION IS REQUIRED. Before v7 a workspace's only repo was indexed
// WITHOUT a prefix: its nodes carried an empty repo_prefix and raw
// repo-relative ids ("internal/foo.go::Bar"). The indexer now always
// prefixes, so on the first warmup after upgrading the same repo is
// re-indexed under "<repo>/internal/foo.go::Bar". Nothing removes the old
// rows — they belong to no tracked prefix, so PurgeRepo never sees them and
// OrphanRepoPrefixes deliberately exempts the empty prefix. The store would
// carry both copies: roughly double the nodes, doubled search hits, per-repo
// counts disagreeing with whole-store counts, and index_health reporting
// green over all of it. That is a silent wrong answer, which is why it is a
// schema boundary and not a best-effort cleanup.
//
// WHY IN-PLACE RATHER THAN rebuild:true. A rebuild is three lines and always
// correct, but planSchemaMigrationWith reads the flag statically — there is
// no "rebuild only if this store has unprefixed rows" strategy — so every
// multi-repo workspace would eat a full cold index (hours on a large one)
// for a change that cannot affect it. A solo workspace must cold-index
// either way, since its ids all change; this targets the cost at the stores
// that actually need to pay it.
//
// WHAT IS DELETED, AND WHAT IS DELIBERATELY KEPT. The predicate selects
// exactly the file-backed rows the ownership audit counts as unowned: an
// empty repo_prefix, a non-empty file_path, and auditable under
// auditableRepoSourceNodePredicate. The file_path guard keeps synthetic
// GLOBAL EXTERNALS with no path (dep::…, non-Go module::…, legacy bare
// builtin::…); the auditable predicate keeps the populations that carry an
// empty repo_prefix legitimately even WITH a path — semantic externals and
// their module::go:… attribution under external::/external-call:: paths,
// and the exempt kinds (a workspace-scoped memory projects a rationale node
// with no repo). All of them model things owned by no repository, and
// deleting them would drop live attribution on every store this runs on.
//
// v22 re-runs this step for the D-14 rollout, so it must be safe on a
// current store, not only on a pre-v7 one.
func purgeUnprefixedRepoRows(tx *sql.Tx) error {
	unowned := `SELECT id FROM nodes WHERE ` + unprefixedFileBackedNodePredicate
	// Edges first, while the nodes they reference still exist. An edge is
	// dropped when EITHER endpoint is an unprefixed code node: a half-
	// dangling edge is worse than none, because the resolver would keep
	// re-materialising its missing target as an unresolved stub.
	if _, err := tx.Exec(`DELETE FROM edges WHERE from_id IN (` + unowned + `) OR to_id IN (` + unowned + `)`); err != nil {
		return fmt.Errorf("delete unprefixed edges: %w", err)
	}

	// Embeddings are keyed by node_id alone — the vectors table has no
	// repo_prefix column — so they can only be reached through node
	// membership, and only while the nodes are still present.
	if _, err := tx.Exec(`DELETE FROM vectors WHERE node_id IN (` + unowned + `)`); err != nil {
		return fmt.Errorf("delete unprefixed vectors: %w", err)
	}

	res, err := tx.Exec(`DELETE FROM nodes WHERE ` + unprefixedFileBackedNodePredicate)
	if err != nil {
		return fmt.Errorf("delete unprefixed nodes: %w", err)
	}
	// A store with no unowned file-backed node holds no unprefixed repository
	// residue, so its empty-prefix sidecar rows belong to the populations kept
	// above. v22 runs on every current store; leave them alone.
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("count unprefixed nodes: %w", err)
	} else if n == 0 {
		return nil
	}

	// Sidecar residue written under the empty prefix. Both table sets are
	// dropped rather than relabelled: the re-index rewrites index-time
	// sidecars under the new ids and paths, and the node_id-keyed enrichment
	// tables are dangling against the nodes just deleted. Reuses the same
	// table lists RekeyRepoPrefix maintains so a new sidecar table cannot be
	// added to one path and forgotten in the other.
	//
	// file_mtimes is in rekeyMoveTables and is deleted here on purpose:
	// leaving it would let the next warm restart find empty-prefix mtimes
	// describing nodes no longer in the store, and skip the re-index that
	// has to happen.
	for _, table := range append(append([]string{}, rekeyMoveTables...), rekeyDropTables...) {
		if _, err := tx.Exec(`DELETE FROM ` + table + ` WHERE repo_prefix = ''`); err != nil {
			return fmt.Errorf("delete unprefixed rows from %s: %w", table, err)
		}
	}
	return nil
}
