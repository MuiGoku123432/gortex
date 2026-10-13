// Package graphdump renders the scoped active subgraph as a canonical,
// sorted dump plus a digest, for the Phase 4 equivalence proofs
// (incremental equals clean rebuild). It is a test-only helper: production
// packages must not import it, and no CLI or MCP verb exposes it.
package graphdump

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/zeebo/blake3"

	"github.com/zzet/gortex/internal/graph"
)

// Volatile is a versioned allowlist of Meta keys left out of the dump.
// Keys maps each key to the justification for treating it as volatile.
type Volatile struct {
	Version string
	Keys    map[string]string
}

// DeterministicVolatile is the allowlist the Phase 4 harness uses. Add a key
// only when a harness diff proves it volatile, together with its
// justification; never add one only to make a failing comparison pass.
//
// v2 (04-08): the source_*_fingerprint keys are the incremental delta-probe
// cache. The first incremental patch of a file stamps them on its file node
// and cold indexing deliberately does not (indexer.prepareFileDelta), so
// their presence records index history, not the inputs; an incremental dump
// could never equal a clean one with them in.
//
// v3 (04-16): a Meta left empty once those keys are stripped renders as null,
// like a node with no Meta, so a non-COBOL file added on the incremental path
// equals its clean build instead of dumping {} against null.
var DeterministicVolatile = Volatile{Version: "graphdump-volatile/3", Keys: map[string]string{
	"source_semantic_fingerprint":         fingerprintCache,
	"source_metadata_fingerprint":         fingerprintCache,
	"source_core_fingerprint":             fingerprintCache,
	"source_derived_decl_fingerprint":     fingerprintCache,
	"source_derived_import_fingerprint":   fingerprintCache,
	"source_derived_runtime_fingerprint":  fingerprintCache,
	"source_derived_artifact_fingerprint": fingerprintCache,
}}

const fingerprintCache = "incremental delta-probe cache stamped only by an incremental patch, never by a cold index"

// Result is one canonical dump: the sorted rows, the lowercase hex BLAKE3
// digest of the rows joined by newlines, and the allowlist version used.
type Result struct {
	Rows            []string
	Digest          string
	VolatileVersion string
}

// Dump reads the scoped active subgraph through opener and returns its
// canonical dump.
//
// PURPOSE — give every Phase 4 lifecycle test one comparable fingerprint of
// a scope, so "incremental equals clean rebuild" is a digest comparison
// with diagnosable row diffs.
// RATIONALE — rows come only from the scoped projection snapshot (never
// rowids or raw SQL: edge rowids are insertion-ordered), are sorted, and
// carry every Meta key except the justified Volatile ones, rendered as
// encoding/json (which sorts map keys). Node rows are N, ID, Kind, Name,
// FilePath, StartLine:StartColumn-EndLine:EndColumn, and Meta; edge rows are
// E, From, To, Kind, FilePath, Line, and Meta; fields are joined with the
// 0x1F unit separator. A Meta that cannot be marshalled is an error, never a
// skipped row.
// KEYWORDS — graphdump, canonical dump, digest, blake3, volatile allowlist,
// incremental equals clean, scoped projection.
func Dump(ctx context.Context, opener graph.ScopedProjectionSnapshotOpener, scope graph.ProjectionScope, v Volatile) (Result, error) {
	if v.Version == "" {
		return Result{}, fmt.Errorf("graphdump: volatile allowlist has no version")
	}
	for key, why := range v.Keys {
		if strings.TrimSpace(why) == "" {
			return Result{}, fmt.Errorf("graphdump: volatile key %q has no justification", key)
		}
	}
	snap, err := opener.OpenScopedProjectionSnapshot(ctx, scope)
	if err != nil {
		return Result{}, fmt.Errorf("graphdump: open snapshot: %w", err)
	}
	defer snap.Close()

	var rows []string
	err = snap.ReadNodePages(ctx, func(nodes []*graph.Node) error {
		for _, n := range nodes {
			meta, err := canonicalMeta(n.Meta, v)
			if err != nil {
				return fmt.Errorf("graphdump: node %s: %w", n.ID, err)
			}
			span := fmt.Sprintf("%d:%d-%d:%d", n.StartLine, n.StartColumn, n.EndLine, n.EndColumn)
			rows = append(rows, strings.Join([]string{"N", n.ID, string(n.Kind), n.Name, n.FilePath, span, meta}, "\x1f"))
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	err = snap.ReadEdgePages(ctx, func(page []graph.ScopedEdgeRow) error {
		for _, r := range page {
			e := r.Edge
			meta, err := canonicalMeta(e.Meta, v)
			if err != nil {
				return fmt.Errorf("graphdump: edge %s -> %s: %w", e.From, e.To, err)
			}
			rows = append(rows, strings.Join([]string{"E", e.From, e.To, string(e.Kind), e.FilePath, strconv.Itoa(e.Line), meta}, "\x1f"))
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	sort.Strings(rows)
	sum := blake3.Sum256([]byte(strings.Join(rows, "\n")))
	return Result{Rows: rows, Digest: hex.EncodeToString(sum[:]), VolatileVersion: v.Version}, nil
}

// canonicalMeta marshals a copy of meta without the Volatile keys. An empty
// result renders as null, the same as a nil Meta.
func canonicalMeta(meta map[string]any, v Volatile) (string, error) {
	m := maps.Clone(meta)
	for key := range v.Keys {
		delete(m, key)
	}
	if len(m) == 0 {
		return "null", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FirstDiff walks two sorted row slices and reports up to n differing rows,
// each with its index and a "-" (only in want) or "+" (only in got) marker,
// printing the 0x1F field separator as " | ". It returns "" when the slices
// are equal.
func FirstDiff(want, got []string, n int) string {
	var b strings.Builder
	found := 0
	i, j := 0, 0
	for (i < len(want) || j < len(got)) && found < n {
		switch {
		case j >= len(got) || (i < len(want) && want[i] < got[j]):
			fmt.Fprintf(&b, "- [%d] %s\n", i, strings.ReplaceAll(want[i], "\x1f", " | "))
			i++
		case i >= len(want) || got[j] < want[i]:
			fmt.Fprintf(&b, "+ [%d] %s\n", j, strings.ReplaceAll(got[j], "\x1f", " | "))
			j++
		default:
			i, j = i+1, j+1
			continue
		}
		found++
	}
	return b.String()
}
