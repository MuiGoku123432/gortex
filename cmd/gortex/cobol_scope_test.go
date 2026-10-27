//go:build darwin || linux

package main

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
)

// isCobolProgram reports whether n is a COBOL PROGRAM-ID node.
func isCobolProgram(n *graph.Node) bool {
	return n.Kind == graph.KindFunction && n.Meta["cobol_kind"] == "program"
}

// TestCobolScope_CrossSurface proves one repository allow-set admits the
// same COBOL programs on the projection snapshot, the graph.Scope predicate
// the reconcile pass applies, and the MCP source gate behind
// get_symbol_source (SEC-01, D-16), and that every bound CALL or COPY edge
// leaving repository A stays inside that allow-set.
func TestCobolScope_CrossSurface(t *testing.T) {
	repos := indexCobolBreadthRepos(t)
	socket, _ := serveCobolTestDaemon(t, repos.dir, repos.store, repos.idx, repos.mi, repos.cm)

	anchor := repos.store.GetNode(repos.prefixA + "/cobol/ordpgm.cbl::ORDPGM")
	require.NotNil(t, anchor, "repository A program ORDPGM")
	require.NotEmpty(t, anchor.WorkspaceID)
	require.NotEmpty(t, anchor.ProjectID)
	scope, err := graph.NewScope(anchor.WorkspaceID, anchor.ProjectID, []string{repos.prefixA})
	require.NoError(t, err)

	allNodes := repos.store.AllNodes()
	var programs, programsB []string
	for _, n := range allNodes {
		if !isCobolProgram(n) {
			continue
		}
		programs = append(programs, n.ID)
		if n.RepoPrefix == repos.prefixB {
			programsB = append(programsB, n.ID)
		}
	}
	require.NotEmpty(t, programsB, "repository B must hold a program")

	// Surface 1: the scoped projection snapshot.
	snap, err := repos.store.OpenScopedProjectionSnapshot(context.Background(), scope)
	require.NoError(t, err)
	var set1 []string
	require.NoError(t, snap.ReadNodePages(context.Background(), func(page []*graph.Node) error {
		for _, n := range page {
			if isCobolProgram(n) {
				set1 = append(set1, n.ID)
			}
		}
		return nil
	}))
	require.NoError(t, snap.Close())

	// Surface 2: the Scope.Allows predicate over every node of both repositories.
	var set2 []string
	for _, n := range allNodes {
		if isCobolProgram(n) && scope.Allows(n) {
			set2 = append(set2, n.ID)
		}
	}

	// Surface 3: the MCP source gate through the daemon relay. The call comes
	// from the directory above both checkouts, which reaches both
	// repositories, so only the explicit repo argument narrows the read.
	var set3 []string
	for _, id := range programs {
		text, isError := callCobolToolResult(t, socket, repos.dir, "get_symbol_source",
			map[string]any{"id": id, "repo": repos.prefixA})
		if !isError {
			set3 = append(set3, id)
			continue
		}
		assert.Equal(t, "symbol not found: "+id, text)
	}

	sort.Strings(set1)
	sort.Strings(set2)
	sort.Strings(set3)
	require.NotEmpty(t, set1)
	assert.Equal(t, set1, set2, "projection vs Scope.Allows")
	assert.Equal(t, set1, set3, "projection vs get_symbol_source")
	for _, id := range programsB {
		assert.NotContains(t, set1, id)
	}

	// Resolution: every bound literal CALL or COPY edge leaving repository A
	// lands on a repository A node.
	bound := 0
	for _, e := range repos.store.AllEdges() {
		if _, ok := e.Meta["cobol_placeholder"]; !ok || strings.HasPrefix(e.To, "unresolved::") {
			continue
		}
		from := repos.store.GetNode(e.From)
		if from == nil || from.RepoPrefix != repos.prefixA {
			continue
		}
		to := repos.store.GetNode(e.To)
		require.NotNil(t, to, "bound %s edge %s -> %s has no target node", e.Kind, e.From, e.To)
		assert.Equal(t, repos.prefixA, to.RepoPrefix, "bound %s edge %s -> %s", e.Kind, e.From, e.To)
		assert.True(t, scope.Allows(to), "bound %s edge %s -> %s", e.Kind, e.From, e.To)
		bound++
	}
	assert.Positive(t, bound, "repository A must hold bound CALL or COPY edges")
}
