package indexer

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/gitcmd"
	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// stampSourceRevision completes the revision half of the provenance
// contract (D-09, PROV-04). Every node and relationship edge whose Meta
// carries prov_revision_content_id gains exactly one of prov_vcs_commit
// (the HEAD commit, written only when src, the exact bytes that were
// extracted and hashed, are the blob HEAD holds for the file) or
// prov_vcs_commit_absence (why no commit can be claimed). It then aligns
// provenance byte offsets with the file (alignProvByteOffsets).
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
		if n != nil {
			applySourceRevision(n.Meta, commit, absence)
		}
	}
	for _, e := range result.Edges {
		if e != nil {
			applySourceRevision(e.Meta, commit, absence)
		}
	}
	idx.alignProvByteOffsets(relPath, src, result)
}

// applySourceRevision gives meta, when it carries prov_revision_content_id,
// exactly one of prov_vcs_commit (commit is non-empty) or
// prov_vcs_commit_absence, deleting the other, and reports whether meta
// changed.
func applySourceRevision(meta map[string]any, commit, absence string) bool {
	if _, ok := meta["prov_revision_content_id"]; !ok {
		return false
	}
	set, drop, value := "prov_vcs_commit_absence", "prov_vcs_commit", absence
	if commit != "" {
		set, drop, value = "prov_vcs_commit", "prov_vcs_commit_absence", commit
	}
	_, stale := meta[drop]
	if cur, ok := meta[set].(string); ok && cur == value && !stale {
		return false
	}
	meta[set] = value
	delete(meta, drop)
	return true
}

// restampSourceRevisions re-decides the revision stamp of the repository's
// COBOL nodes and edges after HEAD moved (D-17): prov_vcs_commit when the
// indexed bytes are the blob the new HEAD holds, prov_vcs_commit_absence
// otherwise. Nothing is re-extracted; only rows whose stamp changed are
// written.
//
// PURPOSE — stampSourceRevision runs only when a file is extracted, so a
// commit that leaves a file's bytes unchanged left the old commit (or a
// stale absence) on it, while a clean build stamps the new HEAD.
//
// RATIONALE — the stamp is decided per document named by prov_source_path,
// from the file's bytes passed through idx.transforms.run, the same
// preparation the extraction paths apply before stampSourceRevision, so a
// commit is never claimed for bytes the commit does not hold. Files the
// mtime ledger reports stale are skipped: their next extraction stamps them.
// COBOL is the only extractor that writes prov_revision_content_id today,
// so only its nodes are read; shared resource nodes carry no content ID and
// are left alone. Nodes are upserted, and edges are persisted through
// EdgeMetaBatchPersister (the in-memory store hands out live edges, so the
// in-place change is already durable there). The log carries counts only.
//
// KEYWORDS — provenance, vcs, git, head move, prov_vcs_commit, restamp
func (idx *Indexer) restampSourceRevisions(rootAbs string) {
	nodes := idx.graph.GetRepoNodesByLanguage(idx.repoPrefix, "cobol")
	if len(nodes) == 0 {
		return
	}
	type revision struct {
		commit, absence string
		ok              bool
	}
	revisions := make(map[string]revision)
	revisionOf := func(meta map[string]any) revision {
		rel, _ := meta["prov_source_path"].(string)
		if rel == "" {
			return revision{}
		}
		if r, seen := revisions[rel]; seen {
			return r
		}
		var r revision
		if !idx.IsStale(rel) {
			if src, err := os.ReadFile(filepath.Join(rootAbs, filepath.FromSlash(rel))); err == nil {
				r.commit, r.absence = classifySourceRevision(rootAbs, rel, idx.transforms.run(rel, src))
				r.ok = true
			}
		}
		revisions[rel] = r
		return r
	}

	ids := make([]string, 0, len(nodes))
	var changedNodes []*graph.Node
	for _, n := range nodes {
		ids = append(ids, n.ID)
		if _, ok := n.Meta["prov_revision_content_id"]; !ok {
			continue
		}
		r := revisionOf(n.Meta)
		if !r.ok {
			continue
		}
		c := *n
		c.Meta = maps.Clone(n.Meta)
		if applySourceRevision(c.Meta, r.commit, r.absence) {
			changedNodes = append(changedNodes, &c)
		}
	}
	var changedEdges []*graph.Edge
	for _, edges := range idx.graph.GetOutEdgesByNodeIDs(ids) {
		for _, e := range edges {
			if _, ok := e.Meta["prov_revision_content_id"]; !ok {
				continue
			}
			if r := revisionOf(e.Meta); r.ok && applySourceRevision(e.Meta, r.commit, r.absence) {
				changedEdges = append(changedEdges, e)
			}
		}
	}

	if len(changedNodes) > 0 {
		idx.graph.AddBatch(changedNodes, nil)
	}
	if len(changedEdges) > 0 {
		if p, ok := idx.graph.(graph.EdgeMetaBatchPersister); ok {
			p.PersistEdgeAttributesBatch(changedEdges)
		}
	}
	idx.logger.Debug("indexer: re-stamped source revisions after HEAD move",
		zap.Int("documents", len(revisions)),
		zap.Int("nodes", len(changedNodes)),
		zap.Int("edges", len(changedEdges)))
}

// alignProvByteOffsets makes prov_start_byte / prov_end_byte true file
// offsets (PROV-03, the CR-01 residual). Extractors write them against
// src, the bytes after the pre-ingestion transforms. When the file on
// disk is exactly a UTF-8 BOM followed by src, every int byte offset on a
// node or edge moves by 3; rows, columns, and line fields stay
// BOM-excluded, as for every other language. Any other difference (a
// command transform, a UTF-16 BOM, a concurrent edit) marks each
// byte-bearing entry prov_range_exact=false with prov_range_absence
// indexed_bytes_differ_from_file instead of presenting its bytes as file
// offsets. Equal bytes, or a file that cannot be read, leave the ranges
// as the extractor wrote them; the revision stamp already records why
// unreadable bytes cannot be tied to the file.
//
// PURPOSE — the BOM strip runs before extraction, so only the indexer,
// which knows rootPath and the indexed bytes, can relate the two.
//
// RATIONALE — offsets shift only on an exact BOM-plus-src match, so a
// shift can never be applied to bytes it does not describe. The file is
// read once, only for files that carry byte offsets, and its bytes are
// never logged or returned.
//
// KEYWORDS — provenance, byte offset, bom, prov_start_byte, prov_range_exact
func (idx *Indexer) alignProvByteOffsets(relPath string, src []byte, result *parser.ExtractionResult) {
	var metas []map[string]any
	for _, n := range result.Nodes {
		if n == nil {
			continue
		}
		if _, ok := n.Meta["prov_start_byte"]; ok {
			metas = append(metas, n.Meta)
		}
	}
	for _, e := range result.Edges {
		if e == nil {
			continue
		}
		if _, ok := e.Meta["prov_start_byte"]; ok {
			metas = append(metas, e.Meta)
		}
	}
	if len(metas) == 0 || idx.rootPath == "" {
		return
	}
	disk, err := os.ReadFile(filepath.Join(idx.rootPath, filepath.FromSlash(relPath)))
	if err != nil || bytes.Equal(disk, src) {
		return
	}
	if bytes.HasPrefix(disk, []byte{0xEF, 0xBB, 0xBF}) && bytes.Equal(disk[3:], src) {
		for _, meta := range metas {
			for _, key := range []string{"prov_start_byte", "prov_end_byte"} {
				if v, ok := meta[key].(int); ok {
					meta[key] = v + 3
				}
			}
		}
		return
	}
	for _, meta := range metas {
		meta["prov_range_exact"] = false
		meta["prov_range_absence"] = "indexed_bytes_differ_from_file"
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
