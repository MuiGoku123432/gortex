package resolver

import (
	"maps"
	"strings"
)

// cobolCopySearchRule names the D-10 ordered copybook search rule. A COPY
// bound by library rank records it in cobol_copy_search_rule next to the
// configured library index in cobol_copy_search_rank.
const cobolCopySearchRule = "cobol-copy-search/1"

// cobolQualifierRule names the D-11 default-qualifier rule. Every link edge
// from an `_unqualified` shared resource to the qualified node its
// repository's default schema or subschema names carries it in
// cobol_qualifier_rule.
const cobolQualifierRule = "cobol-default-qualifier/1"

// CobolLibrary is one entry of a repository's ordered copybook search list
// (D-10): a library Name as a COPY ... OF/IN clause spells it, the repository
// that holds it (empty means the caller's own), and the directory Path inside
// that repository.
type CobolLibrary struct {
	Name string
	Repo string
	Path string
}

// CobolPolicy is one repository's COBOL binding policy, as configured per
// tracked repository.
//
// ResolveFrom is the directed list of other repositories whose programs and
// copybooks this repository may bind to (D-09); a repository never appears in
// another's list implicitly. CopybookLibraries is the ordered copybook search
// list (D-10). DB2DefaultSchema and IDMSDefaultSubschema qualify unqualified
// shared-resource names (D-11). The zero value means same-repository binding
// only.
type CobolPolicy struct {
	ResolveFrom          []string
	CopybookLibraries    []CobolLibrary
	DB2DefaultSchema     string
	IDMSDefaultSubschema string
}

// CobolPolicyLookup returns the COBOL policy for a caller repository prefix.
// It is called at most once per repository per reconcile pass.
type CobolPolicyLookup func(repoPrefix string) CobolPolicy

// SetCobolPolicyLookup installs the per-repository COBOL policy source the
// reconcile pass consults. A nil lookup (the default) means no policy for
// any repository, so every COBOL CALL and COPY binds inside its caller's
// repository only. The lookup persists across passes until reset, mirroring
// the other Set* configuration setters.
func (r *Resolver) SetCobolPolicyLookup(fn CobolPolicyLookup) {
	r.cobolPolicyLookup = fn
}

// cobolPolicyFor returns repo's policy from the pass cache, consulting the
// lookup at most once per repository per pass. With no lookup installed it
// is the zero policy.
func (r *Resolver) cobolPolicyFor(policies map[string]CobolPolicy, repo string) CobolPolicy {
	if p, ok := policies[repo]; ok {
		return p
	}
	var p CobolPolicy
	if r.cobolPolicyLookup != nil {
		p = r.cobolPolicyLookup(repo)
	}
	policies[repo] = p
	return p
}

// cobolReconcileRepos returns changed plus every repository in policies whose
// ResolveFrom names a changed repository (D-09, Pitfall 5): a change in
// repository B re-decides the edges of every repository allowed to resolve
// against B. It never modifies changed.
func cobolReconcileRepos(changed map[string]struct{}, policies map[string]CobolPolicy) map[string]struct{} {
	repos := maps.Clone(changed)
	for repo, p := range policies {
		for _, from := range p.ResolveFrom {
			if _, ok := changed[strings.TrimSpace(from)]; ok {
				repos[repo] = struct{}{}
				break
			}
		}
	}
	return repos
}
