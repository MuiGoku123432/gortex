package graph

import (
	"strconv"
	"strings"
)

// Repo-prefix diagnostics.
//
// A node's repo prefix is the graph's ownership key: it keys the byRepo
// bucket, the per-repo memory estimates, the SQLite `nodes_by_repo` partial
// index, and every repo-scoped query. When ownership is wrong, nothing
// errors — the graph simply answers with a subset, or with two copies of
// the same repo under different IDs. Both failures read as "green" to
// index_health, so they need an explicit detector rather than an assertion
// that would never fire.
//
// Single-repo mode is a workspace of one (D-14): every audited node must be
// owned, whether one repository is tracked or many. The only nodes outside
// the audit are the ones the three enumerated exemption lists below name —
// OwnershipExemptKinds, OwnershipExemptPathPrefixes and
// OwnershipGlobalIDNamespaces. There is no catch-all: an empty-FilePath node
// in an unlisted namespace is audited like any file-backed node, and the
// SQLite predicates are generated from the same lists so the two backends
// cannot drift.
//
// Every audited node with an empty prefix is an UNOWNED CODE NODE: a symbol
// no repository claims.

// PrefixDiagnostics is a bounded ownership audit of the node set.
type PrefixDiagnostics struct {
	// OwnedCodeNodes / UnownedCodeNodes split the auditable repository-source
	// node set by whether a repository claims it.
	//
	// Any unowned node is a defect (D-14: single-repo mode is a workspace of
	// one). BOTH being non-zero is the sharpest form of it: a stale
	// unprefixed population survived alongside a freshly prefixed one and
	// every count, scope filter and per-repo bucket now sees half the graph.
	// See Mixed.
	OwnedCodeNodes   int
	UnownedCodeNodes int

	// MisprefixedNodes counts owned nodes (RepoPrefix != "") whose ID or
	// FilePath does not actually start with "<RepoPrefix>/". The prefix is
	// applied by string concatenation in Indexer.applyRepoPrefix, so a
	// divergence here means the stamped field and the minted identity
	// disagree and repo-scoped lookups will miss the node.
	MisprefixedNodes int

	// UnownedSamples / MisprefixedSamples hold up to sampleLimit node IDs
	// from each population, so a log line can name the offenders instead of
	// only counting them.
	UnownedSamples     []string
	MisprefixedSamples []string

	// Scanned is the number of nodes actually examined. A backend may cap
	// the walk; Truncated records that the counts are lower bounds.
	Scanned   int
	Truncated bool
}

// Mixed reports the ghost-graph signature: the same store holding both
// owned and unowned code nodes. It always means one population is stale
// residue that every repo-scoped read is silently splitting on.
func (d PrefixDiagnostics) Mixed() bool {
	return d.OwnedCodeNodes > 0 && d.UnownedCodeNodes > 0
}

// Clean reports whether the audit found no ownership problems: no unowned
// and no misprefixed node. A wholly unowned graph is not clean (D-14).
func (d PrefixDiagnostics) Clean() bool {
	return d.UnownedCodeNodes == 0 && d.MisprefixedNodes == 0
}

// Summary renders the audit as a short human-readable clause for a log line
// or a recommendation string.
func (d PrefixDiagnostics) Summary() string {
	var b strings.Builder
	b.WriteString("owned=")
	b.WriteString(strconv.Itoa(d.OwnedCodeNodes))
	b.WriteString(" unowned=")
	b.WriteString(strconv.Itoa(d.UnownedCodeNodes))
	if d.MisprefixedNodes > 0 {
		b.WriteString(" misprefixed=")
		b.WriteString(strconv.Itoa(d.MisprefixedNodes))
	}
	if len(d.UnownedSamples) > 0 {
		b.WriteString(" e.g. ")
		b.WriteString(strings.Join(d.UnownedSamples, ", "))
	}
	return b.String()
}

// PrefixDiagnosticsReader is the optional store capability that answers the
// audit natively. The SQLite store implements it with two aggregate queries;
// callers that probe and miss fall back to AuditNodePrefixes over a node
// slice.
type PrefixDiagnosticsReader interface {
	PrefixDiagnostics(sampleLimit int) PrefixDiagnostics
}

// NodePrefixOwnership classifies one node for the audit. It is the single
// definition both the in-memory walk and the SQLite predicates encode, so
// the two backends cannot drift.
//
// Returns ("", true) for a node that is fine or out of scope (synthetic
// global externals), and a reason string for a node that is not.
const (
	// PrefixIssueUnowned marks a code node no repository claims.
	PrefixIssueUnowned = "unowned_code_node"
	// PrefixIssueMisprefixed marks an owned node whose identity does not
	// carry the prefix its RepoPrefix field claims.
	PrefixIssueMisprefixed = "misprefixed_identity"
)

// OwnershipExemptKinds are node kinds whose identity deliberately does not
// carry the owning repository's prefix, so the ownership audit skips them.
// The SQLite predicates are generated from this list.
var OwnershipExemptKinds = []NodeKind{
	// Contract identities live in a global namespace (contract::http::…)
	// so producers and consumers in different repos meet on one node.
	KindContract,
	// Contract bridges use the virtual contracts://bridges path.
	KindContractBridge,
	// Topic identities (topic::nats::…) are global for the same reason as
	// contracts.
	KindTopic,
	// The memory projection mints `rationale::<memory-id>` over the virtual
	// .gortex/rationale path while stamping the memory's repo, so the
	// identity can never carry the prefix.
	KindRationale,
}

// OwnershipExemptPathPrefixes are FilePath prefixes that name a synthetic
// attribution bucket rather than repository source. The SQLite predicates
// are generated from this list.
var OwnershipExemptPathPrefixes = []string{
	// Semantic externals: symbols of a dependency, not of a tracked repo.
	"external::",
	// Synthesized external call targets.
	"external-call::",
}

// OwnershipGlobalIDNamespaces are the ID namespaces an UNOWNED node with an
// empty FilePath may live in without being audited. An owned empty-FilePath
// node is exempt only under its own "<RepoPrefix>::" namespace; every other
// empty-FilePath node is audited. Add a namespace here, with its reason,
// only when it is a real graph-wide namespace — never replace the list with
// a catch-all empty-path rule. The SQLite predicates are generated from it.
var OwnershipGlobalIDNamespaces = []string{
	// Dependency stubs (dep::<module>::<symbol>) shared by every repo that
	// imports the module.
	"dep::",
	// External module nodes, including the shared module::go:stdlib node
	// go/types attribution groups stdlib references under.
	"module::",
	// Standard-library stubs, global per language.
	"stdlib::",
	// External symbol stubs with no attribution path.
	"external::",
	// Language builtins (builtin::go::len), global per language.
	"builtin::",
	// String-literal nodes (string::<context>::<value>), keyed by context
	// and value rather than by the file they appear in.
	"string::",
	// Extractor sentinel for a reference the resolver has not bound yet
	// (UnresolvedMarker); its multi-repo form is "<repo>::unresolved::…".
	UnresolvedMarker,
}

// IsAuditableRepoSourcePath reports whether filePath names repository source
// rather than a synthetic attribution bucket (OwnershipExemptPathPrefixes).
// An empty path is not repository source.
func IsAuditableRepoSourcePath(filePath string) bool {
	if filePath == "" {
		return false
	}
	for _, p := range OwnershipExemptPathPrefixes {
		if strings.HasPrefix(filePath, p) {
			return false
		}
	}
	return true
}

// IsAuditableRepoSourceNode reports whether n participates in the repository
// ownership invariant. A node is exempt only through one of the enumerated
// lists: its kind is in OwnershipExemptKinds, its FilePath starts with an
// OwnershipExemptPathPrefixes entry, or its FilePath is empty and it is
// either unowned under an OwnershipGlobalIDNamespaces namespace or owned
// under its own "<RepoPrefix>::" namespace. Every other node is audited.
func IsAuditableRepoSourceNode(n *Node) bool {
	if n == nil {
		return false
	}
	for _, k := range OwnershipExemptKinds {
		if n.Kind == k {
			return false
		}
	}
	if n.FilePath != "" {
		return IsAuditableRepoSourcePath(n.FilePath)
	}
	if n.RepoPrefix != "" {
		return !strings.HasPrefix(n.ID, n.RepoPrefix+"::")
	}
	for _, ns := range OwnershipGlobalIDNamespaces {
		if strings.HasPrefix(n.ID, ns) {
			return false
		}
	}
	return true
}

// ClassifyNodePrefix returns "" when n's ownership is consistent or outside
// the repository-source audit, and one of the PrefixIssue* constants otherwise.
// For audited nodes, both ID and FilePath are produced by
// Indexer.applyRepoPrefix concatenating `repoPrefix + "/"`, so both must carry
// it.
func ClassifyNodePrefix(n *Node) string {
	if !IsAuditableRepoSourceNode(n) {
		return ""
	}
	if n.RepoPrefix == "" {
		return PrefixIssueUnowned
	}
	want := n.RepoPrefix + "/"
	if !strings.HasPrefix(n.FilePath, want) || !strings.HasPrefix(n.ID, want) {
		return PrefixIssueMisprefixed
	}
	return ""
}

// accumulate folds one node into the audit.
func (d *PrefixDiagnostics) accumulate(n *Node, sampleLimit int) {
	if n == nil {
		return
	}
	d.Scanned++
	switch ClassifyNodePrefix(n) {
	case PrefixIssueUnowned:
		d.UnownedCodeNodes++
		if len(d.UnownedSamples) < sampleLimit {
			d.UnownedSamples = append(d.UnownedSamples, n.ID)
		}
	case PrefixIssueMisprefixed:
		d.MisprefixedNodes++
		if len(d.MisprefixedSamples) < sampleLimit {
			d.MisprefixedSamples = append(d.MisprefixedSamples, n.ID)
		}
	default:
		// Healthy. Only audited repository-source nodes count toward
		// ownership; synthetic attribution paths and identity namespaces do
		// not belong to either population.
		if IsAuditableRepoSourceNode(n) {
			d.OwnedCodeNodes++
		}
	}
}

// AuditNodePrefixes runs the ownership audit over an explicit node slice.
// It is the backend-independent fallback for stores that do not implement
// PrefixDiagnosticsReader.
func AuditNodePrefixes(nodes []*Node, sampleLimit int) PrefixDiagnostics {
	var d PrefixDiagnostics
	for _, n := range nodes {
		d.accumulate(n, sampleLimit)
	}
	return d
}

// PrefixDiagnostics walks every shard and audits node ownership. The
// in-memory store is the reference implementation; it holds each shard's
// read lock only for that shard's pass.
func (g *Graph) PrefixDiagnostics(sampleLimit int) PrefixDiagnostics {
	var d PrefixDiagnostics
	for _, s := range g.shards {
		s.mu.RLock()
		for _, n := range s.nodes {
			d.accumulate(n, sampleLimit)
		}
		s.mu.RUnlock()
	}
	return d
}

// ReadPrefixDiagnostics runs the audit against any store, preferring the
// native capability and falling back to a full node pull only when the
// backend cannot answer it directly. Returns ok=false when neither path is
// available, so a caller can stay silent rather than report a false clean.
func ReadPrefixDiagnostics(store any, sampleLimit int) (PrefixDiagnostics, bool) {
	if store == nil {
		return PrefixDiagnostics{}, false
	}
	if r, capable := store.(PrefixDiagnosticsReader); capable {
		return r.PrefixDiagnostics(sampleLimit), true
	}
	if all, capable := store.(interface{ AllNodes() []*Node }); capable {
		return AuditNodePrefixes(all.AllNodes(), sampleLimit), true
	}
	return PrefixDiagnostics{}, false
}
