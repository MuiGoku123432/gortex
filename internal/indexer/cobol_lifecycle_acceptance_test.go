//go:build darwin || linux

package indexer

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
)

// The Phase 4 acceptance tests (SEC-03, SEC-08, D-15) run on the 04-01
// lifecycle corpus. Every fixture is invented fixed-format COBOL.

// TestCobolLifecycle_Ownership drives one store through track, an edit, a
// delete, a restore, a config change, and a cross-repository change; the
// strict ownership audit must then find no unowned and no misprefixed node
// (SEC-08, D-15).
func TestCobolLifecycle_Ownership(t *testing.T) {
	env := newCobolIncrementalEnv(t)

	for _, rel := range []string{"cobol/sub.cbl", "cobol/main.cbl"} {
		env.write(t, "lifea", rel, cobolLifecycleEdits[rel])
	}
	env.reindex(t, "lifea")

	env.remove(t, "lifea", "copy/custrec.cpy")
	env.reindex(t, "lifea")

	for _, rel := range []string{"cobol/sub.cbl", "cobol/main.cbl", "copy/custrec.cpy"} {
		env.write(t, "lifea", rel, cobolLifecycleRepoA[rel])
	}
	env.reindex(t, "lifea")

	// Config change: lifea may now bind into lifeb, so CALL 'SUBPGM' sees a
	// candidate in each repository.
	env.reconfigure(t, "lifea", "index:\n  cobol:\n    resolve_from: [lifeb]\n    db2_default_schema: MYSCH\n")

	// Cross-repository change: lifeb's SUBPGM goes away through a reindex of
	// lifeb alone, and lifea's unchanged CALL is decided again.
	env.remove(t, "lifeb", "cobol/sub.cbl")
	env.reindex(t, "lifeb")
	env.requireCleanEquivalent(t)

	d, ok := graph.ReadPrefixDiagnostics(env.store, 5)
	require.True(t, ok, "the SQLite store must answer the ownership audit")
	require.Positive(t, d.OwnedCodeNodes, "sanity: the corpus must produce owned code nodes (%s)", d.Summary())
	require.Zero(t, d.UnownedCodeNodes, "unowned nodes after the lifecycle (%s)", d.Summary())
	require.Zero(t, d.MisprefixedNodes, "misprefixed nodes after the lifecycle: %v (%s)", d.MisprefixedSamples, d.Summary())
	require.True(t, d.Clean(), "repo ownership audit must be clean (%s)", d.Summary())
}

// cobolSameProgram is written at the same relative path in two
// repositories; its dynamic CALL target string is the same in both.
const cobolSameProgram = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SAMEPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-PGM PIC X(8).
       PROCEDURE DIVISION.
       MAIN-PARA.
           CALL WS-PGM.
           GOBACK.
`

// TestCobolLifecycle_PlaceholderCollision proves two repositories with the
// same path, program, and dynamic CALL never merge facts (SEC-03): the
// shared unresolved target is an edge endpoint, not a node, and each edge
// keeps its own repository-prefixed From.
func TestCobolLifecycle_PlaceholderCollision(t *testing.T) {
	env := newCobolLifecycleEnv(t, map[string]map[string]string{
		"samea": {"cobol/same.cbl": cobolSameProgram},
		"sameb": {"cobol/same.cbl": cobolSameProgram},
	})
	env.track(t, "samea", "sameb")

	const target = "unresolved::cobol::dynamic::program::cobol/same.cbl::SAMEPGM::WS-PGM"
	var froms []string
	for _, e := range env.store.AllEdges() {
		if e.To == target {
			froms = append(froms, e.From)
		}
	}
	sort.Strings(froms)
	require.Len(t, froms, 2, "edges to %s", target)
	assert.NotEqual(t, froms[0], froms[1])
	assert.True(t, strings.HasPrefix(froms[0], "samea/"), froms[0])
	assert.True(t, strings.HasPrefix(froms[1], "sameb/"), froms[1])
	assert.Nil(t, env.store.GetNode(target), "no node may exist for the shared unresolved target")
}
