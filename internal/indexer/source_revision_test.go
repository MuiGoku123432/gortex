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
func sourceRevisionResult(relPath string) *parser.ExtractionResult {
	return &parser.ExtractionResult{
		Nodes: []*graph.Node{
			{ID: relPath, Kind: graph.KindFile, FilePath: relPath, Meta: map[string]any{"prov_revision_content_id": "content"}},
			{ID: relPath + "::DEMOPGM", Kind: graph.KindFunction, FilePath: relPath, Meta: map[string]any{"prov_revision_content_id": "content"}},
			{ID: relPath + "::MAIN-PARA", Kind: graph.KindFunction, FilePath: relPath},
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

// stampAt runs the stamp for relPath with the indexer rooted at root.
func stampAt(t *testing.T, root, relPath string) *parser.ExtractionResult {
	t.Helper()
	idx := newTestIndexer(graph.New())
	idx.rootPath = root
	result := sourceRevisionResult(relPath)
	idx.stampSourceRevision(relPath, result)
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
		idx.stampSourceRevision("pkg/f.go", result)
		assert.Equal(t, map[string]any{"lang": "go"}, result.Nodes[0].Meta)
		assert.Nil(t, result.Nodes[1].Meta)
		idx.stampSourceRevision("pkg/f.go", nil)
	})
}
