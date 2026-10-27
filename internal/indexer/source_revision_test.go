package indexer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// sourceRevisionResult builds the shape a COBOL extraction hands the
// indexer: a file node and a program node that both carry a content
// revision, plus one node without it that the stamp must leave alone.
// Its edges mirror that: a relationship edge carrying the revision and a
// structural edge without Meta.
func sourceRevisionResult(relPath string) *parser.ExtractionResult {
	return &parser.ExtractionResult{
		Nodes: []*graph.Node{
			{ID: relPath, Kind: graph.KindFile, FilePath: relPath, Meta: map[string]any{"prov_revision_content_id": "content"}},
			{ID: relPath + "::DEMOPGM", Kind: graph.KindFunction, FilePath: relPath, Meta: map[string]any{"prov_revision_content_id": "content"}},
			{ID: relPath + "::MAIN-PARA", Kind: graph.KindFunction, FilePath: relPath},
		},
		Edges: []*graph.Edge{
			{From: relPath + "::DEMOPGM/@proc/MAIN-PARA", To: relPath + "::DEMOPGM/@proc/WORK-PARA", Kind: graph.EdgeCalls, FilePath: relPath, Meta: map[string]any{"prov_revision_content_id": "content"}},
			{From: relPath, To: relPath + "::DEMOPGM", Kind: graph.EdgeDefines, FilePath: relPath},
		},
	}
}

// requireGit skips a subtest on machines without a git binary; every
// subtest except git_unavailable needs a real repository.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available in PATH")
	}
}

// writeRepoFile writes content at root/rel, creating parent directories.
func writeRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	writeFile(t, path, content)
}

// headCommit returns `git rev-parse HEAD` for dir.
func headCommit(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}

// stampAt runs the stamp for relPath with the indexer rooted at root, as
// if the file's current bytes on disk (none when it does not exist) had
// been extracted.
func stampAt(t *testing.T, root, relPath string) *parser.ExtractionResult {
	t.Helper()
	src, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	return stampBytes(t, root, relPath, src)
}

// stampBytes runs the stamp for relPath as if src had been extracted.
func stampBytes(t *testing.T, root, relPath string, src []byte) *parser.ExtractionResult {
	t.Helper()
	idx := newTestIndexer(graph.New())
	idx.rootPath = root
	result := sourceRevisionResult(relPath)
	idx.stampSourceRevision(relPath, src, result)
	return result
}

// assertCommit checks that every revision-carrying node records commit
// and no absence reason, and that the node without a revision is bare.
func assertCommit(t *testing.T, result *parser.ExtractionResult, commit string) {
	t.Helper()
	for _, n := range result.Nodes[:2] {
		assert.Equal(t, commit, n.Meta["prov_vcs_commit"], "node %s commit", n.ID)
		assert.NotContains(t, n.Meta, "prov_vcs_commit_absence", "node %s must not carry both keys", n.ID)
		assert.Equal(t, "content", n.Meta["prov_revision_content_id"], "node %s keeps its content revision", n.ID)
	}
	assert.Nil(t, result.Nodes[2].Meta, "a node without prov_revision_content_id is untouched")
}

// assertAbsence checks that every revision-carrying node records reason
// and no commit, and that the node without a revision is bare.
func assertAbsence(t *testing.T, result *parser.ExtractionResult, reason string) {
	t.Helper()
	for _, n := range result.Nodes[:2] {
		assert.Equal(t, reason, n.Meta["prov_vcs_commit_absence"], "node %s absence reason", n.ID)
		assert.NotContains(t, n.Meta, "prov_vcs_commit", "node %s must not carry both keys", n.ID)
		assert.Equal(t, "content", n.Meta["prov_revision_content_id"], "node %s keeps its content revision", n.ID)
	}
	assert.Nil(t, result.Nodes[2].Meta, "a node without prov_revision_content_id is untouched")
}

// committedRepo returns a repository whose first commit holds
// cobol/a.cbl.
func committedRepo(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo")
	gitInitRepo(t, root)
	writeRepoFile(t, root, "cobol/a.cbl", "       PROGRAM-ID. A.\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "init")
	return root
}

// TestSourceRevisionStamp covers D-09: a node carrying a content revision
// also carries the HEAD commit only when the file's bytes are exactly
// what HEAD holds, and otherwise one precise absence reason.
func TestSourceRevisionStamp(t *testing.T) {
	const rel = "cobol/a.cbl"

	t.Run("clean_committed", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		// An unrelated dirty file proves the classification is per file.
		writeRepoFile(t, root, "cobol/other.cbl", "untracked\n")
		assertCommit(t, stampAt(t, root, rel), headCommit(t, root))
	})

	t.Run("modified", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		writeRepoFile(t, root, rel, "       PROGRAM-ID. B.\n")
		assertAbsence(t, stampAt(t, root, rel), "working_tree_differs_from_head")
	})

	t.Run("staged_new", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		writeRepoFile(t, root, "cobol/new.cbl", "       PROGRAM-ID. N.\n")
		runGit(t, root, "add", "cobol/new.cbl")
		assertAbsence(t, stampAt(t, root, "cobol/new.cbl"), "working_tree_differs_from_head")
	})

	t.Run("untracked", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		writeRepoFile(t, root, "cobol/new.cbl", "       PROGRAM-ID. N.\n")
		assertAbsence(t, stampAt(t, root, "cobol/new.cbl"), "not_under_version_control")
	})

	t.Run("ignored", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		writeRepoFile(t, root, ".gitignore", "cobol/gen/\n")
		runGit(t, root, "add", ".gitignore")
		runGit(t, root, "commit", "-q", "-m", "ignore")
		writeRepoFile(t, root, "cobol/gen/g.cbl", "       PROGRAM-ID. G.\n")
		assertAbsence(t, stampAt(t, root, "cobol/gen/g.cbl"), "not_under_version_control")
	})

	t.Run("unborn", func(t *testing.T) {
		requireGit(t)
		root := filepath.Join(t.TempDir(), "repo")
		gitInitRepo(t, root)
		writeRepoFile(t, root, rel, "       PROGRAM-ID. A.\n")
		runGit(t, root, "add", ".")
		assertAbsence(t, stampAt(t, root, rel), "no_head_commit")
	})

	t.Run("not_git", func(t *testing.T) {
		requireGit(t)
		root := t.TempDir()
		// Keep git from discovering an enclosing repository above the
		// temp dir (for example a checkout that itself lives under TMPDIR).
		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(root))
		writeRepoFile(t, root, rel, "       PROGRAM-ID. A.\n")
		assertAbsence(t, stampAt(t, root, rel), "not_a_git_worktree")
	})

	t.Run("git_unavailable", func(t *testing.T) {
		root := t.TempDir()
		writeRepoFile(t, root, rel, "       PROGRAM-ID. A.\n")
		t.Setenv("PATH", t.TempDir())
		assertAbsence(t, stampAt(t, root, rel), "git_unavailable")
	})

	t.Run("subdirectory_root", func(t *testing.T) {
		requireGit(t)
		// t.TempDir is under /var on macOS, a symlink to /private/var, so
		// the unresolved root differs from git's resolved top level.
		top := filepath.Join(t.TempDir(), "repo")
		gitInitRepo(t, top)
		writeRepoFile(t, top, "estate/"+rel, "       PROGRAM-ID. A.\n")
		runGit(t, top, "add", ".")
		runGit(t, top, "commit", "-q", "-m", "init")
		assertCommit(t, stampAt(t, filepath.Join(top, "estate"), rel), headCommit(t, top))
	})

	// CR-01: the claim follows the hashed bytes, never the worktree state.
	t.Run("bom_stripped", func(t *testing.T) {
		requireGit(t)
		root := filepath.Join(t.TempDir(), "repo")
		gitInitRepo(t, root)
		raw := "\xEF\xBB\xBF       PROGRAM-ID. A.\n"
		writeRepoFile(t, root, rel, raw)
		runGit(t, root, "add", ".")
		runGit(t, root, "commit", "-q", "-m", "init")
		// The always-on pre-ingestion pipeline strips the BOM, so the
		// extracted bytes are not the committed blob even though git
		// reports the file clean.
		src := newTestIndexer(graph.New()).transforms.run(rel, []byte(raw))
		require.NotEqual(t, raw, string(src), "the BOM strip transform must have run")
		assertAbsence(t, stampBytes(t, root, rel, src), "indexed_bytes_differ_from_file")
	})

	t.Run("reverted_after_hashing", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		// Modified bytes were read and hashed; the file was reverted to
		// HEAD's content before the stamp ran.
		assertAbsence(t, stampBytes(t, root, rel, []byte("       PROGRAM-ID. B.\n")), "indexed_bytes_differ_from_file")
	})

	t.Run("assume_unchanged", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		writeRepoFile(t, root, rel, "       PROGRAM-ID. B.\n")
		// git status no longer reports the modification.
		runGit(t, root, "update-index", "--assume-unchanged", rel)
		assertAbsence(t, stampAt(t, root, rel), "working_tree_differs_from_head")
	})

	t.Run("sha256_repository", func(t *testing.T) {
		requireGit(t)
		root := filepath.Join(t.TempDir(), "repo")
		require.NoError(t, os.MkdirAll(root, 0o755))
		if err := exec.Command("git", "-C", root, "init", "-q", "--object-format=sha256").Run(); err != nil {
			t.Skip("git without SHA-256 repository support")
		}
		runGit(t, root, "config", "user.email", "test@example.com")
		runGit(t, root, "config", "user.name", "Test")
		runGit(t, root, "config", "commit.gpgsign", "false")
		writeRepoFile(t, root, rel, "       PROGRAM-ID. A.\n")
		runGit(t, root, "add", ".")
		runGit(t, root, "commit", "-q", "-m", "init")
		head := headCommit(t, root)
		require.Len(t, head, 64)
		assertCommit(t, stampAt(t, root, rel), head)
	})

	// WR-05: a git failure is never reported as a fact about the file.
	t.Run("corrupt_repository", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD^{tree}").Output()
		require.NoError(t, err)
		tree := strings.TrimSpace(string(out))
		require.NoError(t, os.Remove(filepath.Join(root, ".git", "objects", tree[:2], tree[2:])))
		assertAbsence(t, stampAt(t, root, rel), "vcs_query_failed")
	})

	t.Run("submodule", func(t *testing.T) {
		requireGit(t)
		sub := committedRepo(t)
		root := filepath.Join(t.TempDir(), "super")
		gitInitRepo(t, root)
		runGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", "-q", sub, "vendor/sub")
		runGit(t, root, "commit", "-q", "-m", "add submodule")
		// The file is versioned by the submodule, so its commit is the
		// submodule's HEAD, not "not under version control".
		assertCommit(t, stampAt(t, root, "vendor/sub/"+rel), headCommit(t, filepath.Join(root, "vendor/sub")))
	})

	t.Run("untouched", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		idx := newTestIndexer(graph.New())
		idx.rootPath = t.TempDir()
		result := &parser.ExtractionResult{
			Nodes: []*graph.Node{
				{ID: "pkg/f.go", Kind: graph.KindFile, FilePath: "pkg/f.go", Meta: map[string]any{"lang": "go"}},
				{ID: "pkg/f.go::Run", Kind: graph.KindFunction, FilePath: "pkg/f.go"},
			},
		}
		idx.stampSourceRevision("pkg/f.go", nil, result)
		assert.Equal(t, map[string]any{"lang": "go"}, result.Nodes[0].Meta)
		assert.Nil(t, result.Nodes[1].Meta)
		idx.stampSourceRevision("pkg/f.go", nil, nil)
	})
}

// TestSourceRevisionStampEdges covers the carried-forward provenance
// contract: a relationship edge that carries a content revision records
// the same commit or absence reason as its file's nodes, never both.
func TestSourceRevisionStampEdges(t *testing.T) {
	const rel = "cobol/a.cbl"

	t.Run("clean_committed", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		result := stampAt(t, root, rel)
		e := result.Edges[0]
		assert.Equal(t, headCommit(t, root), e.Meta["prov_vcs_commit"])
		assert.NotContains(t, e.Meta, "prov_vcs_commit_absence", "an edge must not carry both keys")
		assert.Equal(t, "content", e.Meta["prov_revision_content_id"])
		assert.Nil(t, result.Edges[1].Meta, "an edge without prov_revision_content_id is untouched")
	})

	t.Run("modified", func(t *testing.T) {
		requireGit(t)
		root := committedRepo(t)
		writeRepoFile(t, root, rel, "       PROGRAM-ID. B.\n")
		result := stampAt(t, root, rel)
		e := result.Edges[0]
		assert.Equal(t, "working_tree_differs_from_head", e.Meta["prov_vcs_commit_absence"])
		assert.NotContains(t, e.Meta, "prov_vcs_commit", "an edge must not carry both keys")
		assert.Nil(t, result.Edges[1].Meta, "an edge without prov_revision_content_id is untouched")
	})
}

// byteRangeResult builds a node and an edge that carry a content revision
// and an extractor-written range: bytes 10..40 of the indexed bytes, row 1.
func byteRangeResult(relPath string) *parser.ExtractionResult {
	meta := func() map[string]any {
		return map[string]any{
			"prov_revision_content_id": "content",
			"prov_start_byte":          10,
			"prov_end_byte":            40,
			"prov_start_row":           1,
			"prov_end_row":             1,
			"prov_start_column":        7,
			"prov_end_column":          37,
			"prov_range_exact":         true,
		}
	}
	return &parser.ExtractionResult{
		Nodes: []*graph.Node{{ID: relPath + "::DEMOPGM", Kind: graph.KindFunction, FilePath: relPath, StartLine: 2, EndLine: 2, Meta: meta()}},
		Edges: []*graph.Edge{{From: relPath + "::DEMOPGM/@proc/MAIN-PARA", To: relPath + "::DEMOPGM/@proc/WORK-PARA", Kind: graph.EdgeCalls, FilePath: relPath, Line: 2, Meta: meta()}},
	}
}

// TestSourceRevisionBOMShift covers the CR-01 residual (PROV-03): byte
// offsets written against the BOM-stripped bytes become file offsets
// when the file is exactly a UTF-8 BOM plus those bytes, any other
// difference marks the range inexact, and rows and columns never move.
func TestSourceRevisionBOMShift(t *testing.T) {
	const rel = "cobol/a.cbl"
	src := []byte("       IDENTIFICATION DIVISION.\n       PROGRAM-ID. DEMOPGM.\n")

	stamp := func(t *testing.T, disk []byte) *parser.ExtractionResult {
		t.Helper()
		root := t.TempDir()
		if disk != nil {
			writeRepoFile(t, root, rel, string(disk))
		}
		idx := newTestIndexer(graph.New())
		idx.rootPath = root
		result := byteRangeResult(rel)
		idx.stampSourceRevision(rel, src, result)
		return result
	}
	metas := func(result *parser.ExtractionResult) []map[string]any {
		return []map[string]any{result.Nodes[0].Meta, result.Edges[0].Meta}
	}
	assertUnmoved := func(t *testing.T, result *parser.ExtractionResult, start, end int) {
		t.Helper()
		for _, m := range metas(result) {
			assert.Equal(t, start, m["prov_start_byte"])
			assert.Equal(t, end, m["prov_end_byte"])
			assert.Equal(t, 1, m["prov_start_row"])
			assert.Equal(t, 1, m["prov_end_row"])
			assert.Equal(t, 7, m["prov_start_column"])
			assert.Equal(t, 37, m["prov_end_column"])
		}
		assert.Equal(t, 2, result.Nodes[0].StartLine)
		assert.Equal(t, 2, result.Nodes[0].EndLine)
		assert.Equal(t, 2, result.Edges[0].Line)
	}
	assertExact := func(t *testing.T, result *parser.ExtractionResult) {
		t.Helper()
		for _, m := range metas(result) {
			assert.Equal(t, true, m["prov_range_exact"])
			assert.NotContains(t, m, "prov_range_absence")
		}
	}

	t.Run("bom", func(t *testing.T) {
		result := stamp(t, append([]byte{0xEF, 0xBB, 0xBF}, src...))
		assertUnmoved(t, result, 13, 43)
		assertExact(t, result)
	})

	t.Run("equal", func(t *testing.T) {
		result := stamp(t, src)
		assertUnmoved(t, result, 10, 40)
		assertExact(t, result)
	})

	t.Run("differ", func(t *testing.T) {
		result := stamp(t, []byte("       IDENTIFICATION DIVISION.\n       PROGRAM-ID. OTHER.\n"))
		assertUnmoved(t, result, 10, 40)
		for _, m := range metas(result) {
			assert.Equal(t, false, m["prov_range_exact"])
			assert.Equal(t, "indexed_bytes_differ_from_file", m["prov_range_absence"])
		}
	})

	t.Run("unreadable", func(t *testing.T) {
		result := stamp(t, nil)
		assertUnmoved(t, result, 10, 40)
		assertExact(t, result)
	})
}
