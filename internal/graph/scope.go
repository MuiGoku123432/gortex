package graph

import (
	"fmt"
	"sort"
	"strings"
)

// Scope is the one fail-closed source boundary (D-16): a workspace, a
// project, and an exact, non-empty repository allow-set.
//
// PURPOSE — Projection and export use it today; the source read/write gate
// and the COBOL reconcile pass reuse it, so every surface answers "may this
// caller see this node?" the same way (SEC-01).
//
// RATIONALE — Unlike repoNarrowAdmits and QueryOptions.ScopeAllows, which
// treat an empty allow-set or an empty RepoPrefix as "no narrow", Scope never
// admits an empty prefix and never treats the zero value as "everything".
// Build values with NewScope so the repository set is trimmed, deduped, and
// sorted.
//
// KEYWORDS — scope, allow-set, repository boundary, fail closed, projection.
type Scope struct {
	Workspace    string
	Project      string
	Repositories []string
}

// NewScope trims the workspace, project, and every repository entry, rejects
// an empty repository set or a blank entry, and returns the deduped, sorted
// allow-set. It never modifies repos.
func NewScope(workspace, project string, repos []string) (Scope, error) {
	if len(repos) == 0 {
		return Scope{}, fmt.Errorf("scope: repository allow-set is required")
	}
	seen := make(map[string]struct{}, len(repos))
	allowed := make([]string, 0, len(repos))
	for _, repo := range repos {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			return Scope{}, fmt.Errorf("scope: repository allow-set contains an empty value")
		}
		if _, ok := seen[repo]; ok {
			continue
		}
		seen[repo] = struct{}{}
		allowed = append(allowed, repo)
	}
	sort.Strings(allowed)
	return Scope{
		Workspace:    strings.TrimSpace(workspace),
		Project:      strings.TrimSpace(project),
		Repositories: allowed,
	}, nil
}

// RequireWorkspaceProject is the projection-only rule: a projection owner is
// keyed by workspace and project, so both must be set.
func (s Scope) RequireWorkspaceProject() error {
	if s.Workspace == "" || s.Project == "" {
		return fmt.Errorf("scope: workspace and project are required")
	}
	return nil
}

// AllowsRepo reports whether repo is in the allow-set. An empty repo or an
// empty allow-set is never allowed. Repositories must be sorted (NewScope).
func (s Scope) AllowsRepo(repo string) bool {
	if repo == "" || len(s.Repositories) == 0 {
		return false
	}
	i := sort.SearchStrings(s.Repositories, repo)
	return i < len(s.Repositories) && s.Repositories[i] == repo
}

// Allows reports whether n is inside the scope: its repository must be in the
// allow-set, and when Workspace or Project is set the node's WorkspaceID or
// ProjectID must match it exactly. A nil node is never allowed.
func (s Scope) Allows(n *Node) bool {
	if n == nil || !s.AllowsRepo(n.RepoPrefix) {
		return false
	}
	if s.Workspace != "" && n.WorkspaceID != s.Workspace {
		return false
	}
	if s.Project != "" && n.ProjectID != s.Project {
		return false
	}
	return true
}
