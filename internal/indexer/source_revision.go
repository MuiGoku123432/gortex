package indexer

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zzet/gortex/internal/gitcmd"
	"github.com/zzet/gortex/internal/gitstate"
	"github.com/zzet/gortex/internal/parser"
)

// stampSourceRevision completes the revision half of the provenance
// contract (D-09, PROV-04). Every node whose Meta carries
// prov_revision_content_id gains exactly one of prov_vcs_commit (the HEAD
// commit, written only when the file's bytes are what HEAD holds) or
// prov_vcs_commit_absence (why no commit can be claimed).
//
// PURPOSE — extractors see only a relative path, so only the indexer,
// which knows rootPath, can relate a file to its git state. It is called
// once from applyCoverageDomains, which covers the bulk and incremental
// index paths (D-18).
//
// RATIONALE — git is sampled on every call, so the snapshot reflects the
// worktree when this file is stamped; only files that carry a content
// revision pay for it. Other languages return after one loop. Files that
// the bulk path quarantines or times out skip applyCoverageDomains and so
// get no stamp, which is consistent: they produce no program node either.
//
// KEYWORDS — provenance, vcs, git, revision, prov_vcs_commit
func (idx *Indexer) stampSourceRevision(relPath string, result *parser.ExtractionResult) {
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

	commit, absence := classifySourceRevision(idx.rootPath, relPath)
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
// the file is tracked and has no difference from HEAD, otherwise an empty
// commit and one absence reason: git_unavailable, not_a_git_worktree,
// no_head_commit, not_under_version_control, or
// working_tree_differs_from_head. It never claims a commit for modified,
// staged-only, untracked, or ignored bytes (T-02-16). Git runs without a
// shell and every path goes after `--` with literal pathspecs (T-02-17).
func classifySourceRevision(rootPath, relPath string) (commit, absence string) {
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

	sampler, err := gitstate.NewDirtySampler(top, "", "")
	if err != nil {
		return "", "git_unavailable"
	}
	snap, err := sampler.Sample(ctx)
	if err != nil {
		return "", "git_unavailable"
	}
	if snap.HeadCommit == "" {
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
	for _, e := range snap.Entries {
		if e.Path != rel {
			continue
		}
		if e.Kind == gitstate.DirtyUntracked {
			return "", "not_under_version_control"
		}
		return "", "working_tree_differs_from_head"
	}

	// Ignored files never appear in the status snapshot, so the index is
	// the last word on whether HEAD can have supplied these bytes.
	_, err = gitcmd.Output(ctx, top, "--literal-pathspecs", "ls-files", "--error-unmatch", "--", rel)
	if ctx.Err() != nil {
		return "", "git_unavailable"
	}
	if err != nil {
		return "", "not_under_version_control"
	}
	return snap.HeadCommit, ""
}
