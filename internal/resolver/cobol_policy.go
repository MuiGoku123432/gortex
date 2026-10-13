package resolver

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

// cobolPolicyFor returns repo's policy, or the zero policy when no lookup is
// installed.
//
//nolint:unused // read by the reconcile pass once 04-14 lands the allowlist.
func (r *Resolver) cobolPolicyFor(repo string) CobolPolicy {
	if r.cobolPolicyLookup == nil {
		return CobolPolicy{}
	}
	return r.cobolPolicyLookup(repo)
}
