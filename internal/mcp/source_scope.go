package mcp

// PURPOSE — the fail-closed source gate (D-13, D-20). With more than one
// tracked repository, no tool may read or write source bytes for a request
// that carries no resolved scope, and a repository outside the resolved
// scope answers exactly like a missing file. Graph-metadata tools are not
// gated here; they keep the all-repositories default for unbound sessions.
//
// RATIONALE — the gate runs at the source-byte choke points
// (resolveFilePath, resolveNodePath, the committed-view reads) rather than
// per tool, so every read and write caller inherits it. The request is
// carried in ctx by wrapToolHandlerMode so the gate can see explicit
// repo / project / workspace / scope narrowing without changing any
// handler signature. Membership is decided with graph.Scope (D-16) so this
// surface and later ones (AI context, claim storage) share one rule.
//
// KEYWORDS — source scope, fail closed, unbound session, multi-repo,
// requireSourceScope, SEC-02, D-13, D-20, graph.Scope.

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/zzet/gortex/internal/graph"
)

// errSourceScopeRequired refuses a source read or write that has no resolved
// scope while several repositories are tracked. It names how to narrow and
// never names a repository.
var errSourceScopeRequired = errors.New(
	"several repositories are tracked, so reading or writing source needs a scope: " +
		"pass repo, project, or workspace, or call from a session bound to a repository")

// errSourceOutOfScope answers a source request for a repository outside the
// resolved scope. It carries the not-found text and no repository name or
// path, so the refusal does not reveal whether the target exists.
var errSourceOutOfScope = fmt.Errorf("%w: no such file or directory", errPathUnresolved)

// refsOutOfScopeResult refuses a refactor with an indexed reference outside
// the resolved scope: a refactor that cannot reach every reference must not
// be applied to the rest. The text names no repository or path, so it only
// reveals that such a reference exists (D-13). action completes "widen it ...
// to <action>".
func refsOutOfScopeResult(tool, action string) *mcp.CallToolResult {
	return mcp.NewToolResultError(tool + ": some references lie outside the resolved scope; " +
		"widen it with repo, project, or workspace to " + action)
}

type sourceScopeRequestKey struct{}

// withSourceScopeRequest stores the tool request so the source gate can
// resolve the caller's explicit narrowing.
func withSourceScopeRequest(ctx context.Context, req mcp.CallToolRequest) context.Context {
	return context.WithValue(ctx, sourceScopeRequestKey{}, req)
}

func sourceScopeRequestFromContext(ctx context.Context) (mcp.CallToolRequest, bool) {
	req, ok := ctx.Value(sourceScopeRequestKey{}).(mcp.CallToolRequest)
	return req, ok
}

// requireSourceScope returns nil when the request may touch source bytes in
// repoPrefix. It is a no-op without a MultiIndexer or with one tracked
// repository. Otherwise a request with no resolved scope gets
// errSourceScopeRequired, and a repository outside the scope gets
// errSourceOutOfScope.
func (s *Server) requireSourceScope(ctx context.Context, repoPrefix string) error {
	if s.multiIndexer == nil || len(s.multiIndexer.RepoPrefixes()) <= 1 {
		return nil
	}
	// No stored request means no explicit narrowing; a bound session still
	// resolves to its own scope through the empty request.
	req, _ := sourceScopeRequestFromContext(ctx)
	// Reach, not Locate: a bound session keeps its whole workspace for
	// source, as it had before this gate; only the unbound fallback changes.
	resolved, err := s.resolveScopeForRequest(ctx, req, IntentReach)
	if err != nil {
		return err
	}
	allow := resolved.RepoAllow
	if len(allow) == 0 && resolved.WorkspaceID != "" {
		// A workspace argument, a workspace-bound session, or the server
		// default workspace resolves to that workspace's repositories.
		allow = s.multiIndexer.ReposInWorkspace(resolved.WorkspaceID)
	}
	repos := make([]string, 0, len(allow))
	for repo, allowed := range allow {
		if allowed {
			repos = append(repos, repo)
		}
	}
	if len(repos) == 0 {
		return errSourceScopeRequired
	}
	scope, err := graph.NewScope("", "", repos)
	if err != nil {
		return errSourceScopeRequired
	}
	if !scope.AllowsRepo(repoPrefix) {
		return errSourceOutOfScope
	}
	return nil
}

// requirePathSourceScope is the source gate for a resolved on-disk path.
// lexical is the repository the caller's spelling names. Root confinement
// accepts a location inside any tracked root, so a symlink or a nested
// checkout can put another repository's bytes under that spelling; the
// repository that physically owns the symlink-resolved location must be in
// scope too (D-13, D-16, D-20). With one tracked repository no second owner
// can exist, so the symlink resolution is skipped; resolveNodePath runs this
// for every withAbsPath listing.
func (s *Server) requirePathSourceScope(ctx context.Context, lexical, absPath string) error {
	if err := s.requireSourceScope(ctx, lexical); err != nil {
		return err
	}
	if s.multiIndexer == nil || !s.multiIndexer.IsMultiRepo() {
		return nil
	}
	real, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		real = resolveNearestExistingAncestor(absPath)
	}
	physical := s.multiIndexer.RepoForFile(real)
	if view := requestViewPathRoot(ctx); view.contains(real) {
		// The request's selected checkout owns its own files, as in
		// resolveFilePath's lexical attribution.
		physical = view.repoPrefix
	}
	if physical == "" || physical == lexical {
		return nil
	}
	return s.requireSourceScope(ctx, physical)
}
