package indexer

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
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

// classifySourceRevision returns the HEAD commit for rootPath/relPath when
// the git blob of src equals the blob HEAD records for that path, otherwise
// an empty commit and one absence reason: git_unavailable,
// not_a_git_worktree, no_head_commit, not_under_version_control,
// working_tree_differs_from_head, or indexed_bytes_differ_from_file (src is
// not the file on disk, for example after a BOM strip or a command
// transform, or because the file changed after it was read). It never
// claims a commit for modified, staged-only, untracked, or ignored bytes
// (T-02-16). Git runs without a shell and every path goes after `--` with
// literal pathspecs (T-02-17).
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
	top, err := gitcmd.Output(ctx, root, "rev-parse", "--show-toplevel")
	if ctx.Err() != nil {
		return "", "git_unavailable"
	}
	if err != nil || top == "" {
		return "", "not_a_git_worktree"
	}
	if resolved, err := filepath.EvalSymlinks(top); err == nil {
		top = resolved
	}

	// Resolve HEAD to one commit first, so the tree lookup below and the
	// returned claim refer to the same commit even if HEAD moves meanwhile.
	head, err := gitcmd.Output(ctx, top, "rev-parse", "-q", "--verify", "HEAD^{commit}")
	if ctx.Err() != nil {
		return "", "git_unavailable"
	}
	if err != nil || head == "" {
		return "", "no_head_commit"
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
	out, err := gitcmd.Run(ctx, top, "--literal-pathspecs", "ls-tree", "-z", head, "--", rel)
	if ctx.Err() != nil || err != nil {
		return "", "git_unavailable"
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
		if ctx.Err() != nil {
			return "", "git_unavailable"
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
