package resolver

import (
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/zzet/gortex/internal/graph"
)

// Missing-artifact placeholders (D-11, GAP-05). A literal reference whose
// target was never obtained points at one owned node per (referencing
// repository, artifact type, normalized key):
// `<callerRepo>/cobol-missing::<type>/<key>`.
const (
	cobolMissingNamespace    = "cobol-missing::"
	cobolMissingArtifactKind = "missing_artifact"

	cobolArtifactProgram    = "program"
	cobolArtifactCopybook   = "copybook"
	cobolArtifactIncMember  = "inc_member"
	cobolArtifactIDMSMember = "idms_dictionary_member"

	// Edge Meta read by the binding pass: the COPY-family form (copy, inc,
	// idms, sql_include) and the SQLCA/SQLDA flag of an EXEC SQL INCLUDE.
	metaCobolCopyForm     = "cobol_copy_form"
	metaSQLIncludeBuiltin = "sql_include_builtin"
)

// EXTERNAL_UNRESOLVED findings (GAP-02, D-04). Each reference occurrence on
// a placeholder gets one finding of kind external-reference, whose evidence
// is the reference form (edge kind, statement kind, CICS command), the call
// or copy form, IDMS kind, and as-written target, read from the edge Meta
// the extractor stamps (never from source, D-18, D-25).
const (
	cobolExternalKind = "external-reference"

	metaCobolCall           = "cobol_call"
	metaCobolTargetRaw      = "cobol_target_raw"
	metaCobolCopyMemberRaw  = "cobol_copy_member_raw"
	metaCobolCopyIDMSKind   = "cobol_copy_idms_kind"
	metaCICSCommand         = "cics_command"
	metaObservationKind     = "prov_observation_kind"
	metaCobolOccurrences    = "cobol_occurrence_count"
	cobolExternalFindingSeg = "::@finding/external/"
)

// cobolExternalRef is one desired entry of the binding pass: a reference
// occurrence whose edge targets a missing-artifact placeholder, with the
// caller repository, the placeholder ID, its artifact type and key, and the
// edge reason (missing or idms_dictionary_member). reconcileCobol collects
// one per such edge, unchanged edges included.
type cobolExternalRef struct {
	e                                                    *graph.Edge
	callerRepo, placeholderID, artifactType, key, reason string
}

// cobolPlaceholderTarget maps a reference of callerRepo to its placeholder
// ID, artifact type, and key by string alone: a program key to program/,
// an -INC copybook key (cobol_copy_form inc) to inc_member/, any other
// copybook key to copybook/, and an IDMS key to idms_dictionary_member/
// without its idms/ prefix. Keys are never normalized again (D-18).
func cobolPlaceholderTarget(callerRepo string, ref cobolRef) (id, artifactType, key string) {
	switch {
	case ref.idms:
		artifactType, key = cobolArtifactIDMSMember, strings.TrimPrefix(ref.key, "idms/")
	case ref.isProgram:
		artifactType, key = cobolArtifactProgram, ref.key
	default:
		artifactType, key = cobolArtifactCopybook, ref.key
		if form, _ := ref.e.Meta[metaCobolCopyForm].(string); form == "inc" {
			artifactType = cobolArtifactIncMember
		}
	}
	return callerRepo + "/" + cobolMissingNamespace + artifactType + "/" + key, artifactType, key
}

// cobolUnresolvedFromPlaceholder returns the unresolved COPY IDMS target an
// idms_dictionary_member placeholder ID stands for, or "" for any other ID.
// Only IDMS edges need it: every other placeholder-targeted edge keeps its
// unresolved target in cobol_placeholder.
func cobolUnresolvedFromPlaceholder(id string) string {
	_, rel, _ := strings.Cut(id, "/")
	rest, ok := strings.CutPrefix(rel, cobolMissingNamespace+cobolArtifactIDMSMember+"/")
	if !ok || rest == "" {
		return ""
	}
	return graph.UnresolvedMarker + cobolIDMSCopyTargetPrefix + rest
}

// reconcileCobolFindings owns the missing-artifact placeholders and the
// EXTERNAL_UNRESOLVED findings of the binding pass. It evicts the external
// findings no desired entry produces any more, adds the placeholder and the
// finding each desired entry needs, and finally evicts every placeholder of
// the reconcile set repos (every repository when repos is nil) that no code
// reference targets any more.
//
// PURPOSE — a reference whose target was never obtained points at one
// KindResource node per (referencing repository, artifact type, key) with
// ID and FilePath `<callerRepo>/cobol-missing::<type>/<key>`, the caller's
// RepoPrefix, WorkspaceID, and ProjectID, Name the key, and only the D-11
// Meta: cobol_kind missing_artifact, artifact_type, artifact_key,
// artifact_status not_obtained, synthetic, prov_evidence_class
// EXTERNAL_UNRESOLVED, prov_range_exact false, prov_range_absence
// not_obtained, and cobol_name_norm. Each such reference occurrence also
// gets one EXTERNAL_UNRESOLVED finding (GAP-02, D-04), including every
// same-line duplicate the extractor collapsed into one edge: an edge with
// cobol_occurrence_count n yields n findings (D-26). Each finding is owned
// by the edge's file, with a member_of edge to the edge's From node and a
// references edge to the placeholder. Its Meta is the full RESEARCH Pattern
// 3 contract (GAP-04): every prov_* key of the edge verbatim, then
// prov_evidence_class EXTERNAL_UNRESOLVED, finding_kind external-reference,
// finding_reason the edge reason, finding_severity medium (D-12),
// finding_scope unavailable, the artifact type as finding_expected_family
// and finding_artifact_type, finding_container, finding_status active, the
// evidence digest, the rule IDs, and finding_target_key. When a reference
// leaves the desired set (bound to a retrieved target, deleted, or edited
// away) its finding is evicted, and the placeholder goes with its last code
// reference through EvictFile on its own path, as reconcileCobolResources
// collects orphaned shared resources (GAP-07).
//
// RATIONALE — the placeholder carries no source, body, range, parser stamp,
// or as-written spelling (GAP-05, P4 D-18), and it is never a binding
// candidate: candidates are KindFunction programs and KindFile copybooks
// only. A finding ID is graph.FindingSymbol over the From node without its
// repository prefix, the class, the kind, the artifact type, and the digest
// of the reference form (edge kind, statement kind, CICS command), the call
// or copy form, IDMS kind, and as-written target (D-01, D-25), so a line
// shift keeps it and a CALL edited into an XCTL changes it; identical
// references in one container take #k in prov_start_byte, Line, edge Kind,
// then occurrence order. The second and later occurrences of a collapsed
// edge have no range of their own, so they carry prov_range_absence
// same_line_occurrence and no coordinates, never a copied range (D-26).
// Because the container is part of the ID and of finding_container,
// references in two containers never share a ledger pairing group and never
// supersede each other (D-03). The as-written target is hashed, never
// stored, and no cobol_* key is copied.
// Owning both nodes by the referencing repository keeps them scope-safe and
// audited (SEC-01, SEC-08). The existing findings are read from the
// references in-edges of the reconcile set's placeholders, never by
// streaming every finding: each external finding has exactly one, to its
// own repository's placeholder, added with it in one AddBatch. The same
// one in-edge read decides the sweep, because evicting findings removes
// only their references edges and a placeholder added in this pass carries
// the code edge that desired it. Undesired findings are evicted with
// graph.EvictFindingNodesByIDs (kind-guarded, so a parse finding or any
// other node can never go) before the sweep; otherwise the sweep would cut
// their only references edge and strand them. Only absent placeholders and
// findings are added, in one AddBatch, so an unchanged rerun writes
// nothing (ID-07). The sweep evicts only a KindResource node with
// cobol_kind missing_artifact whose FilePath equals its ID, so no node owned
// by a real file is ever removed (T-05-17). A `references` edge from a
// finding never keeps its placeholder; any other in-edge does, including a
// code `references` edge such as a literal EXEC CICS LOAD PROGRAM. Every
// pass ends by adding its repository set and the IDs it added and evicted
// to the change record CobolFindingChanges drains; an unchanged rerun adds
// only its repositories.
//
// KEYWORDS — cobol, missing artifact, placeholder, not obtained,
// external unresolved, finding, external-reference, sweep, orphan, evict,
// retrieval, GAP-02, GAP-04, GAP-05, GAP-07, D-03, D-04, D-11, D-25, D-26.
func (r *Resolver) reconcileCobolFindings(repos map[string]struct{}, desired []cobolExternalRef) {
	existing := make(map[string]struct{})
	var owned []*graph.Node // sweep candidates: placeholders of the reconcile set
	for n := range r.nodesByKindLang(graph.KindResource, "cobol") {
		if n == nil || n.Meta["cobol_kind"] != cobolMissingArtifactKind {
			continue
		}
		existing[n.ID] = struct{}{}
		if n.RepoPrefix == "" || n.FilePath != n.ID {
			continue
		}
		if repos != nil {
			if _, ok := repos[n.RepoPrefix]; !ok {
				continue
			}
		}
		owned = append(owned, n)
	}

	// One in-edge read of the reconcile set's placeholders: their
	// references in-edges from external findings enumerate the existing
	// findings, and the same snapshot decides the sweep below.
	var inEdges map[string][]*graph.Edge
	if len(owned) > 0 {
		ids := make([]string, 0, len(owned))
		for _, n := range owned {
			ids = append(ids, n.ID)
		}
		inEdges = r.graph.GetInEdgesByNodeIDs(ids)
	}
	present := make(map[string]struct{})
	for _, edges := range inEdges {
		for _, e := range edges {
			if e != nil && e.Kind == graph.EdgeReferences && strings.Contains(e.From, cobolExternalFindingSeg) {
				present[e.From] = struct{}{}
			}
		}
	}

	type finding struct {
		d          cobolExternalRef
		node       *graph.Node
		start      int
		occurrence int
	}
	findings := make([]finding, 0, len(desired))
	for _, d := range desired {
		start, ok := metaIntValue(d.e.Meta["prov_start_byte"])
		if !ok {
			start = -1
		}
		// D-26: an edge the extractor collapsed from n same-line duplicates
		// stands for n occurrences. Only the first has a range on the edge;
		// the others keep the shared line and never invent coordinates.
		n, ok := metaIntValue(d.e.Meta[metaCobolOccurrences])
		if !ok || n < 1 {
			n = 1
		}
		for i := range n {
			node := r.newCobolExternalFinding(d)
			if i > 0 {
				for _, k := range []string{"prov_start_row", "prov_start_column", "prov_end_row",
					"prov_end_column", "prov_start_byte", "prov_end_byte"} {
					delete(node.Meta, k)
				}
				node.Meta["prov_range_exact"] = false
				node.Meta["prov_range_absence"] = "same_line_occurrence"
				node.EndLine = node.StartLine
			}
			findings = append(findings, finding{d: d, node: node, start: start, occurrence: i})
		}
	}
	sort.Slice(findings, func(a, b int) bool {
		fa, fb := findings[a], findings[b]
		switch {
		case fa.node.ID != fb.node.ID:
			return fa.node.ID < fb.node.ID
		case fa.start != fb.start:
			return fa.start < fb.start
		case fa.d.e.Line != fb.d.e.Line:
			return fa.d.e.Line < fb.d.e.Line
		case fa.d.e.Kind != fb.d.e.Kind:
			return fa.d.e.Kind < fb.d.e.Kind
		}
		return fa.occurrence < fb.occurrence
	})
	// note: the extractor assigns parse finding ordinals the same way
	// (cobolAddFinding, internal/parser/languages/cobol_grammar_findings.go).
	occurrences := make(map[string]int, len(findings))
	wanted := make(map[string]struct{}, len(findings))
	for _, f := range findings {
		occurrences[f.node.ID]++
		if k := occurrences[f.node.ID]; k >= 2 {
			f.node.ID += "#" + strconv.Itoa(k)
			f.node.QualName += "#" + strconv.Itoa(k)
		}
		wanted[f.node.ID] = struct{}{}
	}

	var undesired []string
	for id := range present {
		if _, ok := wanted[id]; !ok {
			undesired = append(undesired, id)
		}
	}
	var evicted []string
	if len(undesired) > 0 {
		sort.Strings(undesired)
		if _, _, ok := graph.EvictFindingNodesByIDs(r.graph, undesired); ok {
			evicted = undesired
		} else {
			r.logger.Warn("cobol: store cannot evict finding nodes; stale EXTERNAL_UNRESOLVED findings remain",
				zap.Int("findings", len(undesired)))
		}
	}

	var nodes []*graph.Node
	var edges []*graph.Edge
	for _, d := range desired {
		if _, ok := existing[d.placeholderID]; ok {
			continue
		}
		existing[d.placeholderID] = struct{}{}
		n := &graph.Node{
			ID: d.placeholderID, Kind: graph.KindResource, Name: d.key,
			FilePath: d.placeholderID, Language: "cobol", RepoPrefix: d.callerRepo,
			Meta: map[string]any{
				"cobol_kind":            cobolMissingArtifactKind,
				"artifact_type":         d.artifactType,
				"artifact_key":          d.key,
				"artifact_status":       "not_obtained",
				"synthetic":             true,
				graph.MetaEvidenceClass: graph.FindingClassExternal,
				"prov_range_exact":      false,
				"prov_range_absence":    "not_obtained",
				"cobol_name_norm":       "cobol-name-norm/1",
			},
		}
		if caller := r.cachedGetNode(d.e.From); caller != nil {
			n.WorkspaceID, n.ProjectID = caller.WorkspaceID, caller.ProjectID
		}
		nodes = append(nodes, n)
	}
	for _, f := range findings {
		if _, ok := present[f.node.ID]; ok {
			continue
		}
		nodes = append(nodes, f.node)
		edges = append(edges,
			&graph.Edge{
				From: f.node.ID, To: f.d.e.From, Kind: graph.EdgeMemberOf, FilePath: f.node.FilePath,
				Line: f.node.StartLine, Origin: graph.OriginASTResolved, Confidence: 1,
			},
			&graph.Edge{
				From: f.node.ID, To: f.d.placeholderID, Kind: graph.EdgeReferences, FilePath: f.node.FilePath,
				Line: f.node.StartLine, Origin: graph.OriginASTResolved, Confidence: 1,
			})
	}
	if len(nodes) > 0 {
		r.graph.AddBatch(nodes, edges)
	}

	// Placeholders added above carry the edges that desired them, so only
	// the ones found before the add can be orphaned.
	for _, n := range owned {
		referenced := false
		for _, e := range inEdges[n.ID] {
			if e == nil {
				continue
			}
			if e.Kind == graph.EdgeReferences && strings.Contains(e.From, "::@finding/") {
				continue // a finding's own edge to the placeholder
			}
			referenced = true
			break
		}
		if !referenced {
			r.graph.EvictFile(n.FilePath)
			evicted = append(evicted, n.ID)
		}
	}

	r.cobolFindingMu.Lock()
	defer r.cobolFindingMu.Unlock()
	if len(repos) > 0 && r.cobolFindingRepos == nil {
		r.cobolFindingRepos = make(map[string]struct{}, len(repos))
	}
	for repo := range repos {
		r.cobolFindingRepos[repo] = struct{}{}
	}
	if len(nodes) > 0 && r.cobolFindingAdded == nil {
		r.cobolFindingAdded = make(map[string]struct{}, len(nodes))
	}
	for _, n := range nodes {
		r.cobolFindingAdded[n.ID] = struct{}{}
	}
	if len(evicted) > 0 && r.cobolFindingEvicted == nil {
		r.cobolFindingEvicted = make(map[string]struct{}, len(evicted))
	}
	for _, id := range evicted {
		r.cobolFindingEvicted[id] = struct{}{}
	}
}

// CobolFindingChanges returns, each sorted, the repositories the COBOL
// reconcile passes reconciled and the finding and placeholder IDs they added
// and evicted since the last call, and clears the record.
//
// PURPOSE — the indexer's COBOL settle points drain every resolver that
// reconciled before running the finding ledger, so a reconcile that touches
// a dependent repository (the set widened by resolve_from) or, with no set,
// any repository's findings ledgers that repository at the same step (D-03
// across repositories, roadmap SC4); 05-11 syncs search from the same IDs
// (Pitfall 4). A pass over every edge (nil repository set) records no
// repository, only IDs.
//
// RATIONALE — the record accumulates instead of being overwritten, because
// two passes can precede one drain (a resolve's settle-window reconcile and
// then a tail's ReconcileCobol) and the no-op second pass must not hide the
// first one's changes. An ID may appear in both added and evicted when two
// passes created and removed it. The record has its own leaf mutex, taken
// here without mu, so a drain never waits on a running resolve.
//
// KEYWORDS — cobol, finding, placeholder, change record, drain, ledger,
// search sync, resolve_from, dependent repository, D-03, Pitfall 4.
func (r *Resolver) CobolFindingChanges() (repos, added, evicted []string) {
	r.cobolFindingMu.Lock()
	defer r.cobolFindingMu.Unlock()
	repos = slices.Sorted(maps.Keys(r.cobolFindingRepos))
	added = slices.Sorted(maps.Keys(r.cobolFindingAdded))
	evicted = slices.Sorted(maps.Keys(r.cobolFindingEvicted))
	r.cobolFindingRepos, r.cobolFindingAdded, r.cobolFindingEvicted = nil, nil, nil
	return repos, added, evicted
}

// newCobolExternalFinding returns the EXTERNAL_UNRESOLVED finding node of
// desired entry d without its #k ordinal: ID `<edge FilePath>::<symbol>`,
// Name the normalized key, QualName the symbol, owned by the caller's
// repository, workspace, and project, with StartLine the edge Line, EndLine
// prov_end_row + 1 when the edge has a range, and the Pattern 3 Meta.
//
// Its evidence is the fixed six-field list edge kind, prov_observation_kind,
// cics_command, call or copy form, COPY IDMS kind, and as-written target,
// each whitespace-collapsed (the edge kind is a constant), an absent field
// as the empty string, joined with NUL (graph.FindingEvidenceRule). The fixed arity keeps one field from
// shifting into another, and the reference form (edge kind, statement kind,
// CICS command) makes CALL, LINK, XCTL, and LOAD of one target distinct
// findings, so editing one form into another changes its ID (D-25).
func (r *Resolver) newCobolExternalFinding(d cobolExternalRef) *graph.Node {
	e := d.e
	collapse := func(key string) string {
		s, _ := e.Meta[key].(string)
		return strings.Join(strings.Fields(s), " ")
	}
	form, raw := collapse(metaCobolCall), collapse(metaCobolTargetRaw)
	if form == "" {
		form = collapse(metaCobolCopyForm)
	}
	if raw == "" {
		raw = collapse(metaCobolCopyMemberRaw)
	}
	evidence := []string{
		string(e.Kind), collapse(metaObservationKind), collapse(metaCICSCommand),
		form, collapse(metaCobolCopyIDMSKind), raw,
	}
	digest := graph.FindingEvidenceDigest(strings.Join(evidence, "\x00"))
	container := strings.TrimPrefix(e.From, d.callerRepo+"/")
	symbol := graph.FindingSymbol(container, graph.FindingClassExternal, cobolExternalKind, d.artifactType, digest)

	meta := make(map[string]any, len(e.Meta)+14)
	for k, v := range e.Meta {
		if strings.HasPrefix(k, "prov_") {
			meta[k] = v
		}
	}
	meta[graph.MetaEvidenceClass] = graph.FindingClassExternal
	meta[graph.MetaFindingKind] = cobolExternalKind
	meta[graph.MetaFindingReason] = d.reason
	meta[graph.MetaFindingSeverity] = "medium"
	meta[graph.MetaFindingSeverityRule] = graph.FindingSeverityRule
	meta[graph.MetaFindingScope] = "unavailable"
	meta[graph.MetaFindingExpectedFamily] = d.artifactType
	meta[graph.MetaFindingContainer] = container
	meta[graph.MetaFindingStatus] = graph.FindingStatusActive
	meta[graph.MetaFindingEvidenceDigest] = digest
	meta[graph.MetaFindingEvidenceRule] = graph.FindingEvidenceRule
	meta[graph.MetaFindingIDRule] = graph.FindingIDRule
	meta[graph.MetaFindingArtifactType] = d.artifactType
	meta[graph.MetaFindingTargetKey] = d.key

	n := &graph.Node{
		ID: e.FilePath + "::" + symbol, Kind: graph.KindFinding, Name: d.key, QualName: symbol,
		FilePath: e.FilePath, Language: "cobol", RepoPrefix: d.callerRepo, StartLine: e.Line, Meta: meta,
	}
	if row, ok := metaIntValue(e.Meta["prov_end_row"]); ok {
		n.EndLine = row + 1
	}
	if caller := r.cachedGetNode(e.From); caller != nil {
		n.WorkspaceID, n.ProjectID = caller.WorkspaceID, caller.ProjectID
	}
	return n
}
