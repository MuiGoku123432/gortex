package graph

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/zeebo/blake3"
)

// Finding resolution classes (GAP-03). The COBOL extractor writes the parse
// class and the COBOL resolver the external class, into the ID segment and
// prov_evidence_class; the finding ledger and index_health read them.
const (
	FindingClassParse    = "PARSE_UNRESOLVED"
	FindingClassExternal = "EXTERNAL_UNRESOLVED"
)

// Finding lifecycle statuses (D-03, D-06). Finding nodes only ever carry
// active; the finding ledger writes closed and superseded into history rows.
const (
	FindingStatusActive     = "active"
	FindingStatusClosed     = "closed"
	FindingStatusSuperseded = "superseded"
)

// Versioned finding rule IDs. Every finding writer records them in Meta so
// a later change to the ID, evidence, or severity rule is visible; nothing
// reads them for matching.
const (
	FindingIDRule       = "gortex-finding-id/1"
	FindingEvidenceRule = "gortex-finding-evidence/1"
	FindingSeverityRule = "gortex-finding-severity/1"
)

// Finding Meta keys (GAP-04). The COBOL extractor and resolver write them on
// KindFinding nodes; the finding ledger, index_health, and Phase 6 queries
// read them.
const (
	MetaEvidenceClass          = "prov_evidence_class"
	MetaFindingKind            = "finding_kind"
	MetaFindingReason          = "finding_reason"
	MetaFindingSeverity        = "finding_severity"
	MetaFindingSeverityRule    = "finding_severity_rule"
	MetaFindingScope           = "finding_scope"
	MetaFindingNodeKind        = "finding_node_kind"
	MetaFindingExpectedFamily  = "finding_expected_family"
	MetaFindingContainer       = "finding_container"
	MetaFindingStatus          = "finding_status"
	MetaFindingEvidenceDigest  = "finding_evidence_digest"
	MetaFindingEvidenceRule    = "finding_evidence_rule"
	MetaFindingIDRule          = "finding_id_rule"
	MetaFindingProducerReasons = "finding_producer_reasons"
	MetaFindingArtifactType    = "finding_artifact_type"
	MetaFindingTargetKey       = "finding_target_key"
)

// FindingEvidenceDigest returns the 32-hex digest of a finding's normalized
// evidence text.
//
// PURPOSE — Finding IDs and Meta carry this digest instead of the evidence
// itself, so protected estate source never leaks through a finding's ID,
// search terms, or Neo4j projection.
//
// RATIONALE — The value is length-framed after FindingEvidenceRule, so the
// empty string still has a distinct digest and a future normalization rule
// cannot collide with this one. Callers normalize first; this function
// hashes bytes as given.
//
// KEYWORDS — finding, evidence, digest, blake3, privacy, GAP-04.
func FindingEvidenceDigest(evidence string) string {
	return findingDigest(FindingEvidenceRule, evidence)
}

// FindingSymbol returns the QualName of a finding,
// `@finding/<parse|external>/<32 hex>`. Callers prepend `<path>::` and
// append `#k` for the k-th (k >= 2) identical symbol in one document.
// class must be FindingClassParse or FindingClassExternal; any other value
// panics, because only the COBOL extractor and resolver call it.
//
// PURPOSE — One identity function shared byte for byte by the extractor
// (parse findings) and the resolver (external findings), anchored to the
// repository-relative containing symbol ID, the resolution class, the gap
// kind, the node kind or expected family, and the evidence digest (D-01).
//
// RATIONALE — No line, byte offset, producer gap ID, or parser or extractor
// version is hashed (D-01, D-02), so line shifts and parser bumps keep the
// ID and only an edit of the gap's own text changes it. Every value is
// length-framed, so adjacent inputs cannot alias through concatenation, and
// source text cannot forge an ID or a path segment. '@' cannot begin a
// COBOL word, so no written name collides with the @finding root.
//
// KEYWORDS — finding, identity, ID, blake3, PARSE_UNRESOLVED,
// EXTERNAL_UNRESOLVED, D-01, D-02, GAP-06.
func FindingSymbol(container, class, kind, family, evidenceDigest string) string {
	var segment string
	switch class {
	case FindingClassParse:
		segment = "parse"
	case FindingClassExternal:
		segment = "external"
	default:
		panic("graph.FindingSymbol: unknown finding class " + class)
	}
	return "@finding/" + segment + "/" +
		findingDigest(FindingIDRule, container, class, kind, family, evidenceDigest)
}

// findingDigest returns the first 16 bytes, hex-encoded, of the BLAKE3
// digest of values, each framed by its 8-byte big-endian length (the
// projectionDigest framing).
func findingDigest(values ...string) string {
	h := blake3.New()
	var size [8]byte
	for _, value := range values {
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = h.Write(size[:])
		_, _ = h.Write([]byte(value))
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}
