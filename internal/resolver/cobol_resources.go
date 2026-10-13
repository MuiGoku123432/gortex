package resolver

import (
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

// metaCobolQualifierRule is the edge Meta key on a D-11 link edge; its value
// is cobolQualifierRule. Only alias edges carrying it are owned by this pass.
const metaCobolQualifierRule = "cobol_qualifier_rule"

// reconcileCobolResources collects orphaned COBOL shared resources and keeps
// the configured default-qualifier link edges in step, over the reconcile
// set repos (every repository when repos is nil) and the pass's policies.
//
// PURPOSE — a shared resource (an IDMS or CICS resource, a DB2 table) owns
// its own virtual path (FilePath equals its ID), so evicting the last file
// that names it never removes it (CR-02). Here every shared resource of a
// repository in repos whose in-edges are none or only EdgeAliases is
// evicted through EvictFile on that virtual path, which also drops its alias
// edges (D-05). Then, for each surviving `_unqualified` DB2 or IDMS resource
// whose repository policy names a default schema or subschema, exactly one
// EdgeAliases edge links it to the qualified node of the same repository
// with Meta cobol_qualifier_rule when that node exists (D-11); alias edges
// carrying the rule that the policy no longer implies are removed.
//
// RATIONALE — a node is evicted only when its FilePath equals its ID, so a
// node owned by a real source file (indexed before resources got their own
// path) is never deleted (ID-07). A configured default only adds or removes
// a link edge: no node is created, renamed, or merged (ID-04, A8). Alias
// edges without the rule key (a Go type alias) are never touched. The pass
// writes nothing when every link is already in place, so a second run is a
// no-op.
//
// KEYWORDS — cobol, shared resource, orphan, gc, evict, db2, idms,
// default schema, subschema, alias, qualifier.
func (r *Resolver) reconcileCobolResources(repos map[string]struct{}, policies map[string]CobolPolicy) {
	var shared []*graph.Node
	for _, kind := range []graph.NodeKind{graph.KindResource, graph.KindTable} {
		for n := range r.nodesByKindLang(kind, "cobol") {
			if n == nil || n.RepoPrefix == "" || n.FilePath != n.ID {
				continue
			}
			if absence, _ := n.Meta["prov_range_absence"].(string); absence != "shared_resource" {
				continue
			}
			if repos != nil {
				if _, ok := repos[n.RepoPrefix]; !ok {
					continue
				}
			}
			shared = append(shared, n)
		}
	}
	if len(shared) == 0 {
		return
	}

	ids := make([]string, 0, len(shared))
	for _, n := range shared {
		ids = append(ids, n.ID)
	}
	inEdges := r.graph.GetInEdgesByNodeIDs(ids)
	// Each surviving `_unqualified` resource and the qualified ID its
	// policy's default names ("" when the policy names none).
	unqualified := make(map[string]string)
	for _, n := range shared {
		referenced := false
		for _, e := range inEdges[n.ID] {
			if e != nil && e.Kind != graph.EdgeAliases {
				referenced = true
				break
			}
		}
		if !referenced {
			r.graph.EvictFile(n.FilePath)
			continue
		}
		policy := r.cobolPolicyFor(policies, n.RepoPrefix)
		rel := strings.TrimPrefix(n.ID, n.RepoPrefix+"/")
		if rest, ok := strings.CutPrefix(rel, "db2::_unqualified/"); ok {
			unqualified[n.ID] = cobolQualifiedID(n.RepoPrefix, "db2", policy.DB2DefaultSchema, rest)
		} else if rest, ok := strings.CutPrefix(rel, "idms::_unqualified/"); ok {
			unqualified[n.ID] = cobolQualifiedID(n.RepoPrefix, "idms", policy.IDMSDefaultSubschema, rest)
		}
	}
	if len(unqualified) == 0 {
		return
	}

	from := make([]string, 0, len(unqualified))
	for id := range unqualified {
		from = append(from, id)
	}
	outEdges := r.graph.GetOutEdgesByNodeIDs(from)
	var added []*graph.Edge
	for _, id := range from {
		want := unqualified[id]
		if want != "" && r.graph.GetNode(want) == nil {
			want = "" // never create the qualified node (A8)
		}
		linked := false
		for _, e := range outEdges[id] {
			if e == nil || e.Kind != graph.EdgeAliases {
				continue
			}
			if _, ours := e.Meta[metaCobolQualifierRule]; !ours {
				continue
			}
			if e.To == want && !linked {
				linked = true
				continue
			}
			r.graph.RemoveEdge(e.From, e.To, e.Kind)
		}
		if want != "" && !linked {
			added = append(added, &graph.Edge{
				From: id, To: want, Kind: graph.EdgeAliases, FilePath: id,
				Origin: graph.OriginASTResolved, Confidence: 1,
				Meta: map[string]any{metaCobolQualifierRule: cobolQualifierRule},
			})
		}
	}
	if len(added) > 0 {
		r.graph.AddBatch(nil, added)
	}
}

// cobolQualifiedID returns the qualified node ID a configured default names
// for an `_unqualified` resource of repo, or "" when no default is set.
func cobolQualifiedID(repo, system, qualifier, rest string) string {
	qualifier = strings.TrimSpace(qualifier)
	if qualifier == "" {
		return ""
	}
	return repo + "/" + system + "::" + qualifier + "/" + rest
}
