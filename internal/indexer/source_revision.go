package indexer

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zzet/gortex/internal/gitcmd"
	"github.com/zzet/gortex/internal/parser"
)

// stampSourceRevision completes the revision half of the provenance
// contract (D-09, PROV-04). Every node whose Meta carries
// prov_revision_content_id gains exactly one of prov_vcs_commit (the HEAD
// commit, written only when src, the exact bytes that were extracted and
// hashed, are the blob HEAD holds for the file) or prov_vcs_commit_absence
// (why no commit can be claimed).
//
// PURPOSE — extractors see only a relative path, so only the indexer,
// which knows rootPath, can relate a file to its git state. It is called
// once from applyCoverageDomains, which covers the bulk and incremental
// index paths (D-18).
//
// RATIONALE — the claim is decided from src, not from the worktree, so a
// pre-ingestion transform (the always-on BOM strip, a command transform),
// a checkout between reading and stamping, and assume-unchanged or
// skip-worktree index flags can never pair a commit with bytes it does not
// hold. Only files that carry a content revision pay for the git queries.
// Other languages return after one loop. Files that the bulk path
// quarantines or times out skip applyCoverageDomains and so get no stamp,
// which is consistent: they produce no program node either.
//
// KEYWORDS — provenance, vcs, git, revision, prov_vcs_commit
func (idx *Indexer) stampSourceRevision(relPath string, src []byte, result *parser.ExtractionResult) {
	if result == nil {
		return
	}
	carries := false
	for _, n := range result.Nodes {
		if n == nil {
			continue
		}
		if _, ok := n.Meta["prov_revision_content_id"]; ok {
			carries = true
			break
		}
	}
	if !carries {
		return
	}

	commit, absence := classifySourceRevision(idx.rootPath, relPath, src)
	for _, n := range result.Nodes {
		if n == nil {
			continue
		}
		if _, ok := n.Meta["prov_revision_content_id"]; !ok {
			continue
		}
		if commit != "" {
			n.Meta["prov_vcs_commit"] = commit
			delete(n.Meta, "prov_vcs_commit_absence")
			continue
		}
		n.Meta["prov_vcs_commit_absence"] = absence
		delete(n.Meta, "prov_vcs_commit")
	}
}

// classifySourceRevision returns the HEAD commit of the git repository
// that contains rootPath/relPath (a submodule's own repository for a file
// inside one) when the git blob of src equals the blob HEAD records for
// that path, otherwise an empty commit and one absence reason:
// git_unavailable (no git binary), not_a_git_worktree, no_head_commit,
// not_under_version_control, working_tree_differs_from_head,
// indexed_bytes_differ_from_file (src is not the file on disk, for example
// after a BOM strip or a command transform, or because the file changed
// after it was read), or vcs_query_failed (a git query timed out or failed
// for a reason that says nothing about the file, such as a corrupt
// repository or a safe.directory refusal). It never claims a commit for
// modified, staged-only, untracked, or ignored bytes (T-02-16). Git runs
// without a shell and every path goes after `--` with literal pathspecs
// (T-02-17).
func classifySourceRevision(rootPath, relPath string, src []byte) (commit, absence string) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", "git_unavailable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if rootPath == "" {
		return "", "not_a_git_worktree"
	}
	root, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", "not_a_git_worktree"
	}
	// One query, run from the file's own directory so a file inside a
	// submodule resolves to that submodule, prints the top level and then
	// HEAD resolved to one commit. The tree lookup below and the returned
	// claim use that commit even if HEAD moves meanwhile. An unborn HEAD
	// prints only the top level and exits 1.
	dir := filepath.Dir(filepath.Join(root, filepath.FromSlash(relPath)))
	out, err := gitcmd.Run(ctx, dir, "rev-parse", "--show-toplevel", "-q", "--verify", "HEAD^{commit}")
	if ctx.Err() != nil {
		return "", "vcs_query_failed"
	}
	top, head, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "not a git repository"):
			return "", "not_a_git_worktree"
		case gitExitedWithOne(err) && top != "":
			return "", "no_head_commit"
		}
		return "", "vcs_query_failed"
	}
	if top == "" || head == "" {
		return "", "vcs_query_failed"
	}
	if resolved, err := filepath.EvalSymlinks(top); err == nil {
		top = resolved
	}

	rel, err := filepath.Rel(top, filepath.Join(root, relPath))
	if err != nil {
		return "", "not_under_version_control"
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", "not_under_version_control"
	}

	// One entry, "<mode> SP <type> SP <oid> TAB <path> NUL", when HEAD
	// holds rel as a file.
	out, err = gitcmd.Run(ctx, top, "--literal-pathspecs", "ls-tree", "-z", head, "--", rel)
	if err != nil {
		return "", "vcs_query_failed"
	}
	var headBlob string
	if info, path, ok := strings.Cut(strings.TrimSuffix(string(out), "\x00"), "\t"); ok && path == rel {
		if fields := strings.Fields(info); len(fields) == 3 && fields[1] == "blob" {
			headBlob = fields[2]
		}
	}
	if headBlob == "" {
		// HEAD does not hold rel. The index says whether it is staged
		// (tracked, but not what HEAD holds) or not versioned at all.
		_, err = gitcmd.Output(ctx, top, "--literal-pathspecs", "ls-files", "--error-unmatch", "--", rel)
		if ctx.Err() != nil || (err != nil && !gitExitedWithOne(err)) {
			return "", "vcs_query_failed"
		}
		if err != nil {
			return "", "not_under_version_control"
		}
		return "", "working_tree_differs_from_head"
	}
	if gitBlobOID(src, len(head)) == headBlob {
		return head, ""
	}
	if disk, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath))); err != nil || !bytes.Equal(disk, src) {
		return "", "indexed_bytes_differ_from_file"
	}
	return "", "working_tree_differs_from_head"
}

// gitExitedWithOne reports whether err is git exiting with status 1, the
// "not found" answer of `rev-parse -q --verify` and `ls-files
// --error-unmatch`, as opposed to a fatal error (status 128) or a kill.
func gitExitedWithOne(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == 1
}

// gitBlobOID returns the git object ID of data as a blob: the hex hash of
// "blob <len>\x00" + data, SHA-256 when the repository's object IDs are 64
// hex digits long and SHA-1 otherwise.
// note: internal/mcp's gitBlobSHA computes the SHA-1 form for overlay drift
// checks; two uses with different needs, so it stays duplicated.
func gitBlobOID(data []byte, oidHexLen int) string {
	h := sha1.New()
	if oidHexLen == 2*sha256.Size {
		h = sha256.New()
	}
	fmt.Fprintf(h, "blob %d\x00", len(data))
	_, _ = h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
