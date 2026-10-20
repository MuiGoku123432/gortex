package resolver

import (
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
// is the edge's call or copy form, IDMS kind, and as-written target, read
// from the edge Meta the extractor stamps (never from source, D-18).
const (
	cobolExternalKind = "external-reference"

	metaCobolCall           = "cobol_call"
	metaCobolTargetRaw      = "cobol_target_raw"
	metaCobolCopyMemberRaw  = "cobol_copy_member_raw"
	metaCobolCopyIDMSKind   = "cobol_copy_idms_kind"
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
// gets one EXTERNAL_UNRESOLVED finding (GAP-02, D-04), owned by the edge's
// file, with a member_of edge to the edge's From node and a references edge
// to the placeholder. Its Meta is the full RESEARCH Pattern 3 contract
// (GAP-04): every prov_* key of the edge verbatim, then prov_evidence_class
// EXTERNAL_UNRESOLVED, finding_kind external-reference, finding_reason the
// edge reason, finding_severity medium (D-12), finding_scope unavailable,
// the artifact type as finding_expected_family and finding_artifact_type,
// finding_container, finding_status active, the evidence digest, the rule
// IDs, and finding_target_key. When a reference leaves the desired set
// (bound to a retrieved target, deleted, or edited away) its finding is
// evicted, and the placeholder goes with its last code reference through
// EvictFile on its own path, as reconcileCobolResources collects orphaned
// shared resources (GAP-07).
//
// RATIONALE — the placeholder carries no source, body, range, parser stamp,
// or as-written spelling (GAP-05, P4 D-18), and it is never a binding
// candidate: candidates are KindFunction programs and KindFile copybooks
// only. A finding ID is graph.FindingSymbol over the From node without its
// repository prefix, the class, the kind, the artifact type, and the digest
// of the call or copy form, IDMS kind, and as-written target (D-01), so a
// line shift keeps it; identical references in one container take #k in
// prov_start_byte, Line, then edge Kind order. Because the container is part
// of the ID and of finding_container, references in two containers never
// share a ledger pairing group and never supersede each other (D-03). The
// as-written target is hashed, never stored, and no cobol_* key is copied.
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
// code `references` edge such as a literal EXEC CICS LOAD PROGRAM.
//
// KEYWORDS — cobol, missing artifact, placeholder, not obtained,
// external unresolved, finding, external-reference, sweep, orphan, evict,
// retrieval, GAP-02, GAP-04, GAP-05, GAP-07, D-03, D-04, D-11.
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
		d     cobolExternalRef
		node  *graph.Node
		start int
	}
	findings := make([]finding, 0, len(desired))
	for _, d := range desired {
		start, ok := metaIntValue(d.e.Meta["prov_start_byte"])
		if !ok {
			start = -1
		}
		findings = append(findings, finding{d: d, node: r.newCobolExternalFinding(d), start: start})
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
		}
		return fa.d.e.Kind < fb.d.e.Kind
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
	if len(undesired) > 0 {
		sort.Strings(undesired)
		if _, _, ok := graph.EvictFindingNodesByIDs(r.graph, undesired); !ok {
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
		}
	}
}

// newCobolExternalFinding returns the EXTERNAL_UNRESOLVED finding node of
// desired entry d without its #k ordinal: ID `<edge FilePath>::<symbol>`,
// Name the normalized key, QualName the symbol, owned by the caller's
// repository, workspace, and project, with StartLine the edge Line, EndLine
// prov_end_row + 1 when the edge has a range, and the Pattern 3 Meta.
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
	evidence := []string{form}
	if kind := collapse(metaCobolCopyIDMSKind); kind != "" {
		evidence = append(evidence, kind)
	}
	digest := graph.FindingEvidenceDigest(strings.Join(append(evidence, raw), "\x00"))
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
