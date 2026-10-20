package resolver

import (
	"strings"

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

// reconcileCobolFindings adds the missing-artifact placeholder each desired
// entry of the binding pass targets, over the reconcile set repos (every
// repository when repos is nil).
//
// PURPOSE — a reference whose target was never obtained points at one
// KindResource node per (referencing repository, artifact type, key) with
// ID and FilePath `<callerRepo>/cobol-missing::<type>/<key>`, the caller's
// RepoPrefix, WorkspaceID, and ProjectID, Name the key, and only the D-11
// Meta: cobol_kind missing_artifact, artifact_type, artifact_key,
// artifact_status not_obtained, synthetic, prov_evidence_class
// EXTERNAL_UNRESOLVED, prov_range_exact false, prov_range_absence
// not_obtained, and cobol_name_norm.
//
// RATIONALE — the placeholder carries no source, body, range, parser stamp,
// or as-written spelling (GAP-05, P4 D-18), and it is never a binding
// candidate: candidates are KindFunction programs and KindFile copybooks
// only. Owning it by the referencing repository keeps it scope-safe and
// audited (SEC-01, SEC-08). Only placeholders absent from the store are
// added, in one AddBatch, so an unchanged rerun writes nothing (ID-07).
//
// KEYWORDS — cobol, missing artifact, placeholder, not obtained,
// external unresolved, GAP-05, D-11.
func (r *Resolver) reconcileCobolFindings(repos map[string]struct{}, desired []cobolExternalRef) {
	existing := make(map[string]struct{})
	for n := range r.nodesByKindLang(graph.KindResource, "cobol") {
		if n != nil && n.Meta["cobol_kind"] == cobolMissingArtifactKind {
			existing[n.ID] = struct{}{}
		}
	}
	var added []*graph.Node
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
		added = append(added, n)
	}
	if len(added) > 0 {
		r.graph.AddBatch(added, nil)
	}
}
