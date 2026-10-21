//go:build darwin || linux

package indexer

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/testutil/graphdump"
)

// The findings half of the canonical-graph harness (GAP-06, D-01, D-07,
// P4 D-04): active findings and missing-artifact placeholders are part of
// the canonical dump, finding history is not. Every fixture below is
// invented fixed-format COBOL. It must never be replaced with
// estate-derived source: this repository is public.

// cobolFindingsMain is repository fcora's cobol/main.cbl. It holds every
// finding source of the phase in one program: a missing COPY, a present
// COPY (REALCPY), a missing -INC, a COPY IDMS member, an EXEC SQL INCLUDE
// SQLCA, an SQL tail on ZQXCOL, a localized ERROR, a missing CALL, an
// ambiguous CALL (DUPPGM is defined twice), and a dynamic CALL.
const cobolFindingsMain = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. MAINPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY NOCPY.
       COPY REALCPY.
-INC NOINC
       COPY IDMS RECORD EMP-REC.
           EXEC SQL INCLUDE SQLCA END-EXEC.
       01 WS-B PIC 9(4).
       01 WS-PGM PIC X(8).
       PROCEDURE DIVISION.
       TAIL-PARA.
           EXEC SQL SELECT ZQXCOL FROM APP.ZQXTAB END-EXEC.
       ERR-PARA.
           COMPUTE WS-B = (1 + 2.
       CALL-PARA.
           CALL 'NOPGM'.
           CALL 'DUPPGM'.
           CALL WS-PGM.
           STOP RUN.
`

// cobolFindingsDup is one of the two DUPPGM definitions that make the CALL
// 'DUPPGM' of cobolFindingsMain ambiguous.
const cobolFindingsDup = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. DUPPGM.
       PROCEDURE DIVISION.
       DUP-PARA.
           GOBACK.
`

// cobolFindingsCorpus returns a fresh copy of the two independent
// repositories (no resolve_from between them). fcora adds a program with a
// MISSING period, a present copybook, and a copybook that cascades; fcorb
// holds its own missing CALL, missing COPY, and SQL tail.
func cobolFindingsCorpus() map[string]map[string]string {
	return map[string]map[string]string{
		"fcora": {
			"cobol/main.cbl": cobolFindingsMain,
			"cobol/dup1.cbl": cobolFindingsDup,
			"cobol/dup2.cbl": cobolFindingsDup,
			"cobol/noper.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. NOPER
       PROCEDURE DIVISION.
       MAIN-PARA.
           STOP RUN.
`,
			"copy/realcpy.cpy": "       01 REAL-REC.\n           05 REAL-ID PIC X(8).\n",
			"copy/casc.cpy": "       01  CUST-REC.\n" +
				"           05 CUST-ID PIC X(8) WHATEVER.\n" +
				"           05  CUST-NAME PIC X(30).\n" +
				"           05  CUST-CITY PIC X(20).\n",
		},
		"fcorb": {
			"cobol/other.cbl": `       IDENTIFICATION DIVISION.
       PROGRAM-ID. OTHPGM.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       COPY NOCPYB.
       PROCEDURE DIVISION.
       OTH-PARA.
           EXEC SQL SELECT ZQXOTH FROM APP.ZQXTAB END-EXEC.
           CALL 'NOPGM'.
           STOP RUN.
`,
		},
	}
}

// cobolDumpNode is one decoded N row of a canonical dump.
type cobolDumpNode struct {
	id, kind string
	meta     map[string]any
}

// cobolDumpNodes decodes the N rows of res whose kind is kind.
func cobolDumpNodes(t *testing.T, res graphdump.Result, kind graph.NodeKind) []cobolDumpNode {
	t.Helper()
	var out []cobolDumpNode
	for _, line := range res.Rows {
		fields := strings.Split(line, "\x1f")
		if fields[0] != "N" || fields[2] != string(kind) {
			continue
		}
		var meta map[string]any
		require.NoError(t, json.Unmarshal([]byte(fields[len(fields)-1]), &meta), line)
		out = append(out, cobolDumpNode{id: fields[1], kind: fields[2], meta: meta})
	}
	return out
}

// cobolDumpIDs returns the IDs of nodes.
func cobolDumpIDs(nodes []cobolDumpNode) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.id)
	}
	return out
}

// cobolDumpFindingIDs returns the sorted KindFinding IDs of res.
func cobolDumpFindingIDs(t *testing.T, res graphdump.Result) []string {
	t.Helper()
	ids := cobolDumpIDs(cobolDumpNodes(t, res, graph.KindFinding))
	slices.Sort(ids)
	return ids
}

// newCobolFindingsEnv tracks a fresh copy of the findings corpus, so the
// sequences can mutate env.repos to mirror the disk.
func newCobolFindingsEnv(t *testing.T) *cobolLifecycleEnv {
	t.Helper()
	env := newCobolLifecycleEnv(t, cobolFindingsCorpus())
	env.track(t, "fcora", "fcorb")
	return env
}

// TestCobolFindings_CleanDeterminism proves GAP-06 idempotency for clean
// builds: the findings corpus gives one dump across rebuilds and opposite
// track orders, and that dump holds parse and external findings and
// missing-artifact placeholders.
func TestCobolFindings_CleanDeterminism(t *testing.T) {
	want := cleanCobolDump(t, cobolFindingsCorpus(), "fcora", "fcorb")
	requireSameDump(t, want, cleanCobolDump(t, cobolFindingsCorpus(), "fcora", "fcorb"))
	requireSameDump(t, want, cleanCobolDump(t, cobolFindingsCorpus(), "fcorb", "fcora"))

	classes := map[string][]string{}
	for _, n := range cobolDumpNodes(t, want, graph.KindFinding) {
		class, _ := n.meta[graph.MetaEvidenceClass].(string)
		kind, _ := n.meta[graph.MetaFindingKind].(string)
		classes[class] = append(classes[class], kind)
		// Ambiguous, dynamic, and SQLCA references stay edge-only (D-09, D-19).
		assert.NotContains(t, []any{"DUPPGM", "WS-PGM", "SQLCA"}, n.meta[graph.MetaFindingTargetKey], n.id)
	}
	t.Logf("finding kinds by class: %v", classes)
	for class, kinds := range map[string][]string{
		graph.FindingClassParse:    {"unparsed-tail", "error", "missing", "uncontained-cascade"},
		graph.FindingClassExternal: {"external-reference"},
	} {
		for _, kind := range kinds {
			assert.Contains(t, classes[class], kind, "the dump holds a %s %s finding", class, kind)
		}
	}

	placeholders := map[string]bool{}
	for _, n := range cobolDumpNodes(t, want, graph.KindResource) {
		if n.meta["cobol_kind"] == "missing_artifact" {
			placeholders[n.id] = true
		}
	}
	for _, id := range []string{
		"fcora/cobol-missing::program/NOPGM", "fcora/cobol-missing::copybook/NOCPY",
		"fcora/cobol-missing::inc_member/NOINC", "fcora/cobol-missing::idms_dictionary_member/RECORD/EMP-REC",
		"fcorb/cobol-missing::program/NOPGM", "fcorb/cobol-missing::copybook/NOCPYB",
	} {
		assert.True(t, placeholders[id], "the dump holds placeholder %s: %v", id, placeholders)
	}
	assert.Len(t, placeholders, 6, "no placeholder for an ambiguous, dynamic, or SQLCA reference: %v", placeholders)
}

// TestCobolFindings_LineShiftKeepsIDs proves D-01: three comment lines
// prepended to a program keep every parse and external finding ID, and the
// incremental dump equals a clean build of the shifted files.
func TestCobolFindings_LineShiftKeepsIDs(t *testing.T) {
	env := newCobolFindingsEnv(t)
	before := cobolDumpFindingIDs(t, env.requireCleanEquivalent(t))
	require.NotEmpty(t, before)

	shifted := "      * SHIFT ONE\n      * SHIFT TWO\n      * SHIFT THREE\n" + cobolFindingsMain
	env.write(t, "fcora", "cobol/main.cbl", shifted)
	env.reindex(t, "fcora")

	after := env.requireCleanEquivalent(t)
	assert.Equal(t, before, cobolDumpFindingIDs(t, after), "a line shift keeps every finding ID")
}

// TestCobolFindings_IncrementalEqualsClean proves P4 D-04 for findings: an
// added missing CALL, a deleted copybook (its COPY becomes missing, with a
// placeholder and a finding), and the restore of both files each end equal
// to a clean build, and the restore equals the original clean build.
func TestCobolFindings_IncrementalEqualsClean(t *testing.T) {
	env := newCobolFindingsEnv(t)
	original := env.requireCleanEquivalent(t)

	env.write(t, "fcora", "cobol/main.cbl", strings.Replace(cobolFindingsMain,
		"           CALL 'NOPGM'.\n", "           CALL 'NOPGM'.\n           CALL 'NEWPGM'.\n", 1))
	env.reindex(t, "fcora")
	got := env.requireCleanEquivalent(t)
	assert.Contains(t, cobolDumpIDs(cobolDumpNodes(t, got, graph.KindResource)), "fcora/cobol-missing::program/NEWPGM")

	env.remove(t, "fcora", "copy/realcpy.cpy")
	env.reindex(t, "fcora")
	got = env.requireCleanEquivalent(t)
	assert.Contains(t, cobolDumpIDs(cobolDumpNodes(t, got, graph.KindResource)), "fcora/cobol-missing::copybook/REALCPY")
	var realcpy int
	for _, n := range cobolDumpNodes(t, got, graph.KindFinding) {
		if n.meta[graph.MetaFindingTargetKey] == "REALCPY" {
			realcpy++
		}
	}
	assert.Equal(t, 1, realcpy, "the deleted copybook's COPY gets one EXTERNAL_UNRESOLVED finding")

	env.write(t, "fcora", "copy/realcpy.cpy", cobolFindingsCorpus()["fcora"]["copy/realcpy.cpy"])
	env.write(t, "fcora", "cobol/main.cbl", cobolFindingsMain)
	env.reindex(t, "fcora")
	requireSameDump(t, original, env.requireCleanEquivalent(t))
}

// TestCobolFindings_HistoryNotCanonical proves D-07: appending a
// finding_history row leaves the canonical dump digest unchanged.
func TestCobolFindings_HistoryNotCanonical(t *testing.T) {
	env := newCobolFindingsEnv(t)
	before := env.dump(t)
	rows := len(cobolHistory(t, env, "fcora"))
	require.Positive(t, rows)

	var appender graph.FindingHistoryAppender = env.store
	require.NoError(t, appender.AppendFindingTransitions("fcora", []graph.FindingTransition{{
		FindingID: "fcora/cobol/main.cbl::@finding/parse/00000000000000000000000000000000", RepoPrefix: "fcora",
		WorkspaceID: "phase4", ProjectID: "phase4", PriorStatus: graph.FindingStatusActive,
		NewStatus: graph.FindingStatusClosed, Class: graph.FindingClassParse, Kind: "error",
		Container: "cobol/main.cbl::MAINPGM", StartByte: -1,
	}}))
	require.Len(t, cobolHistory(t, env, "fcora"), rows+1, "the synthetic row was appended")
	requireSameDump(t, before, env.dump(t))
}

// cobolCrossContainer is repository xcont's one program: A-PARA calls the
// never-obtained NOPGM and B-PARA does not; cobolCrossContainerMoved moves
// the call to B-PARA in one edit.
const cobolCrossContainer = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. XPGM.
       PROCEDURE DIVISION.
       A-PARA.
           CALL 'NOPGM'.
           DISPLAY 'A'.
       B-PARA.
           DISPLAY 'B'.
           GOBACK.
`

const cobolCrossContainerMoved = `       IDENTIFICATION DIVISION.
       PROGRAM-ID. XPGM.
       PROCEDURE DIVISION.
       A-PARA.
           DISPLAY 'A'.
       B-PARA.
           CALL 'NOPGM'.
           DISPLAY 'B'.
           GOBACK.
`

// TestCobolFindings_ExternalCrossContainerNoSupersede proves D-03 and GAP-07
// on the real path: moving CALL 'NOPGM' from one paragraph to another in one
// edit closes the old external finding and opens the new one, and never
// pairs them as a supersede, because findings never match across containers.
func TestCobolFindings_ExternalCrossContainerNoSupersede(t *testing.T) {
	const placeholder = "xcont/cobol-missing::program/NOPGM"
	env := newCobolLifecycleEnv(t, map[string]map[string]string{"xcont": {"cobol/x.cbl": cobolCrossContainer}})
	env.track(t, "xcont")
	oldID := cobolExternalFindingFor(t, env, "xcont", "NOPGM")
	oldContainer, _ := env.store.GetNode(oldID).Meta[graph.MetaFindingContainer].(string)
	require.True(t, strings.HasSuffix(oldContainer, "/A-PARA"), "the call sits in A-PARA: %s", oldContainer)
	before := len(cobolHistory(t, env, "xcont"))

	env.write(t, "xcont", "cobol/x.cbl", cobolCrossContainerMoved)
	env.reindex(t, "xcont")

	newID := cobolExternalFindingFor(t, env, "xcont", "NOPGM")
	require.NotEqual(t, oldID, newID, "the container is part of the finding ID")
	newContainer, _ := env.store.GetNode(newID).Meta[graph.MetaFindingContainer].(string)
	require.True(t, strings.HasSuffix(newContainer, "/B-PARA"), "the call sits in B-PARA: %s", newContainer)
	require.NotNil(t, env.store.GetNode(placeholder), "the placeholder stays")

	rows := cobolHistory(t, env, "xcont")[before:]
	var external []graph.FindingTransition
	for _, row := range rows {
		assert.NotEqual(t, graph.FindingStatusSuperseded, row.NewStatus, "no supersede: %+v", row)
		if row.Class == graph.FindingClassExternal {
			external = append(external, row)
		}
	}
	require.Len(t, external, 2, "exactly the old close and the new open: %+v", rows)
	byID := map[string]graph.FindingTransition{}
	for _, row := range external {
		byID[row.FindingID] = row
	}
	require.Contains(t, byID, oldID)
	require.Contains(t, byID, newID)
	assert.Equal(t, graph.FindingStatusActive, byID[oldID].PriorStatus)
	assert.Equal(t, graph.FindingStatusClosed, byID[oldID].NewStatus)
	assert.Equal(t, oldContainer, byID[oldID].Container)
	assert.Empty(t, byID[oldID].SuccessorID)
	assert.Equal(t, "", byID[newID].PriorStatus)
	assert.Equal(t, graph.FindingStatusActive, byID[newID].NewStatus)
	assert.Equal(t, newContainer, byID[newID].Container)
}
