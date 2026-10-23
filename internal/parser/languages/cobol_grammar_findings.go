//go:build darwin || linux

package languages

import (
	"bytes"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/grade"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/handoff"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

// cobolFindings maps every producer PARSE_UNRESOLVED gap of the walked
// document onto one KindFinding node with a member_of edge to its innermost
// statement, else its anchor, then adds one uncontained-cascade finding when
// the producer graded the document with a cascade-family reason.
//
// PURPOSE — GAP-01: each gap the producer reports (error, missing,
// unparsed-tail, lost-capture, unmapped-capture, and the document-scope
// no-tree, budget-exceeded, invocation-failure) becomes a persisted,
// retrievable finding (D-10). Every gap is one finding: per-token SQL tails
// are never aggregated (D-18) and nothing is filtered or merged. A cascaded
// document also gets one high-severity document finding carrying the
// producer's reasons, alongside its localized gaps (D-13, GAP-08).
//
// RATIONALE — EXTERNAL_UNRESOLVED gaps are skipped: the analyzer runs with an
// empty -INC catalog, so the producer reports every -INC member as not
// obtained; -INC findings come only from the resolver (D-17). The ID and
// finding_container, the ledger pairing key, hash the name-stable anchor
// (paragraph, section, procedure division, data or environment section,
// program, or, for document scope, the file node), never a statement
// ordinal, while member_of keeps the precise statement (D-21). Gaps are
// visited by generated span (start, end, list index), so the #k ordinal of
// identical evidence in one anchor is the same on every rerun. Severity,
// scope, and absence reasons are the producer's verbatim (D-12); the ID
// never hashes a line, offset, gap ID, version, or producer reason (D-01,
// D-02). Name is the gap kind and Meta holds only the evidence digest and
// range, never source text. Programs call it before the document flags are
// written to docMeta, and copybooks after their unmapped check on a cloned
// docMeta, so neither leaks into finding Meta.
//
// KEYWORDS — cobol, finding, PARSE_UNRESOLVED, handoff gaps, cascade,
// uncontained-cascade, GAP-01, GAP-08, D-10, D-13, D-17, D-18, member_of.
func cobolFindings(w *cobolWalk) {
	order := make([]int, 0, len(w.h.Gaps))
	for i, g := range w.h.Gaps {
		if g.ResolutionClass == handoff.ResolutionParseUnresolved {
			order = append(order, i)
		}
	}
	sort.SliceStable(order, func(a, b int) bool {
		ga, gb := w.h.Gaps[order[a]].Generated, w.h.Gaps[order[b]].Generated
		switch {
		case ga == nil || gb == nil:
			return ga == nil && gb != nil
		case ga.Start != gb.Start:
			return ga.Start < gb.Start
		case ga.End != gb.End:
			return ga.End < gb.End
		}
		return order[a] < order[b]
	})
	occurrences := make(map[string]int, len(order)+1)
	for _, i := range order {
		g := w.h.Gaps[i]
		cobolAddFinding(w, occurrences, g, cobolFindingEvidence(w, g))
	}
	if !slices.ContainsFunc(w.h.Assessment.Reasons, func(r string) bool {
		return slices.Contains(cobolCascadeFamily, r)
	}) {
		return
	}
	// D-13: one document finding per cascaded document. Its evidence is the
	// constant kind, so the producer reasons, re-stamped in Meta, never
	// change the ID of a persisting cascade (D-01, D-02).
	cascade := cobolAddFinding(w, occurrences, handoff.Gap{
		Kind: cobolCascadeKind, Reason: cobolCascadeKind,
		Severity: handoff.SeverityHigh, Scope: handoff.ScopeDocument,
		Container: -1, AbsenceReason: "document_scope",
	}, cobolCascadeKind)
	cascade.Meta[graph.MetaFindingProducerReasons] = append([]string{}, w.h.Assessment.Reasons...)
}

// cobolCascadeKind is the kind, reason, and evidence of the Gortex-derived
// uncontained-cascade finding (D-13).
const cobolCascadeKind = "uncontained-cascade"

// cobolAnalysisFailedKind is the kind and evidence of the WR-06 finding on a
// copybook whose analysis failed; cobolAnalysisFailedReason is its coarse
// reason class. The producer's error text is never persisted (T-05-32).
const (
	cobolAnalysisFailedKind   = "analysis-failed"
	cobolAnalysisFailedReason = "analysis-error"
)

// cobolCopybookAnalysisFailed returns the identity-only result for the
// copybook at filePath whose analysis failed.
//
// PURPOSE — WR-06: a copybook that exists but could not be analyzed must not
// look missing. Its file node keeps the COPY binding keys, so every COPY of
// it still binds, and one high-severity document PARSE_UNRESOLVED finding
// records the failure (D-10); it never becomes EXTERNAL_UNRESOLVED.
//
// RATIONALE — no handoff exists, so the file node carries only identity and
// the stamps known without one (source, content revision, approved grammar,
// modules, extractor version) plus
// prov_analysis_absence=copybook_analysis_failed; no handoff-derived key is
// invented. The finding goes through cobolAddFinding with a constant
// evidence, so its ID is stable. Programs get no such fallback (D-20):
// Gortex never derives a program identity from a filename.
func (e *CobolGrammarExtractor) cobolCopybookAnalysisFailed(filePath string, src []byte, sourceID preprocessor.SourceID) *parser.ExtractionResult {
	docMeta := map[string]any{
		"prov_source_path":         filePath,
		"prov_source_id":           string(sourceID),
		"prov_revision_content_id": string(preprocessor.NewContentID(src)),
		"prov_parser_grammar_id":   e.approvedGrammarID,
		"prov_parser_module":       e.parserModule,
		"prov_preprocessor_module": e.preprocessorModule,
		"prov_extractor_version":   cobolGrammarExtractorVersion,
		"prov_evidence_class":      "DETERMINISTIC",
		"prov_origin":              graph.OriginASTResolved,
		"prov_confidence":          1.0,
	}
	// note: the same binding keys as cobolCopybook writes on an analyzed
	// copybook; keep the two in step.
	fileMeta := maps.Clone(docMeta)
	base := filepath.Base(filePath)
	fileMeta["cobol_kind"] = "copybook"
	fileMeta["cobol_name_key"] = cobolIDSegment(cobolNameNorm(strings.TrimSuffix(base, filepath.Ext(base))))
	fileMeta["cobol_name_norm"] = cobolNameNormRule
	fileMeta["prov_analysis_absence"] = "copybook_analysis_failed"
	result := &parser.ExtractionResult{Nodes: []*graph.Node{{
		ID: filePath, Kind: graph.KindFile, Name: filePath,
		FilePath: filePath, StartLine: 1, EndLine: bytes.Count(src, []byte("\n")) + 1,
		Language: "cobol", Meta: fileMeta,
	}}}
	w := newCobolWalk(handoff.Handoff{}, src, filePath, docMeta, nil, result)
	cobolAddFinding(w, map[string]int{}, handoff.Gap{
		Kind: cobolAnalysisFailedKind, Reason: cobolAnalysisFailedReason,
		Severity: handoff.SeverityHigh, Scope: handoff.ScopeDocument,
		Container: -1, AbsenceReason: "document_scope",
	}, cobolAnalysisFailedKind)
	return result
}

// cobolCascadeFamily lists the producer grade reasons that mark an
// uncontained recovery cascade (D-13). Gortex reads them from the
// assessment and never re-derives a cascade from gaps.
var cobolCascadeFamily = []string{
	grade.ReasonDivisionScaleRecovery,
	grade.ReasonRetainedStructureCollapse,
	grade.ReasonMissingRequiredAnchor,
	grade.ReasonLostCapture,
}

// cobolAddFinding appends the PARSE_UNRESOLVED finding for g, whose
// normalized evidence text is evidence, with its member_of edge, and returns
// it. The ID is graph.FindingSymbol over the name-stable anchor, kind, node
// kind or expected family, and evidence digest, plus #k for the k-th
// (k >= 2) identical symbol counted in occurrences. Severity, scope, and
// absence reason come from g verbatim.
//
// The ID and finding_container (the ledger pairing key) hash the anchor of
// cobolFindingAnchor, never a statement node, whose ID carries a per-kind
// ordinal; the member_of edge keeps the precise innermost statement when
// one exists, else the anchor (D-21).
func cobolAddFinding(w *cobolWalk, occurrences map[string]int, g handoff.Gap, evidence string) *graph.Node {
	anchor := cobolFindingAnchor(w, g.Container)
	family := g.NodeKind
	if family == "" {
		family = g.ExpectedFamily
	}
	digest := graph.FindingEvidenceDigest(evidence)
	symbol := graph.FindingSymbol(anchor, graph.FindingClassParse, string(g.Kind), family, digest)
	// note: the resolver assigns external finding ordinals the same way.
	occurrences[symbol]++
	if k := occurrences[symbol]; k >= 2 {
		symbol += "#" + strconv.Itoa(k)
	}

	meta := maps.Clone(w.docMeta)
	meta[graph.MetaEvidenceClass] = graph.FindingClassParse
	meta[graph.MetaFindingKind] = string(g.Kind)
	meta[graph.MetaFindingReason] = g.Reason
	meta[graph.MetaFindingSeverity] = string(g.Severity)
	meta[graph.MetaFindingSeverityRule] = graph.FindingSeverityRule
	meta[graph.MetaFindingScope] = string(g.Scope)
	if g.NodeKind != "" {
		meta[graph.MetaFindingNodeKind] = g.NodeKind
	}
	if g.ExpectedFamily != "" {
		meta[graph.MetaFindingExpectedFamily] = g.ExpectedFamily
	}
	meta[graph.MetaFindingContainer] = anchor
	meta[graph.MetaFindingStatus] = graph.FindingStatusActive
	meta[graph.MetaFindingEvidenceDigest] = digest
	meta[graph.MetaFindingEvidenceRule] = graph.FindingEvidenceRule
	meta[graph.MetaFindingIDRule] = graph.FindingIDRule
	node := &graph.Node{
		ID: w.filePath + "::" + symbol, Kind: graph.KindFinding, Name: string(g.Kind),
		QualName: symbol, FilePath: w.filePath, Language: "cobol", Meta: meta,
	}
	cobolStampRange(node, g.Original)
	if g.AbsenceReason != "" {
		meta["prov_range_exact"] = false
		meta["prov_range_absence"] = g.AbsenceReason
	}
	w.result.Nodes = append(w.result.Nodes, node)
	member := anchor
	if s := w.statementSymbol(g.Container); s != "" {
		member = w.filePath + "::" + s
	}
	w.memberOf(node.ID, member, node.StartLine)
	return node
}

// cobolFindingAnchor returns the repository-relative node ID of the
// name-stable anchor of observation c: the procedure membership container
// (paragraph, else section, else procedure division), else a data or
// environment section, each found by walking Parent links; else the owning
// named program, else the walk root. A document-scope gap (c < 0), an
// out-of-range c, or one nothing contains anchors to the file node.
//
// RATIONALE — D-21: a statement node is never the anchor, because its ID
// carries a per-kind ordinal that an unrelated same-kind statement inserted
// or deleted above it shifts. Document scope never anchors to "the single
// program", because a second program would then move it (WR-07).
func cobolFindingAnchor(w *cobolWalk, c int) string {
	if c < 0 || c >= len(w.h.Facts.Observations) {
		return w.filePath
	}
	levels := []func(i int) string{
		func(i int) string {
			if s := w.container[i]; s != "" {
				return w.id(s)
			}
			return ""
		},
		func(i int) string {
			if n := w.sections[i]; n != nil {
				return n.ID
			}
			return ""
		},
	}
	for _, level := range levels {
		for i := c; ; {
			if id := level(i); id != "" {
				return id
			}
			p, ok := w.parent(i)
			if !ok {
				break
			}
			i = p
		}
	}
	if s := w.programSymbol(c); s != "" {
		return w.id(s)
	}
	if w.root != "" {
		return w.id(w.root)
	}
	return w.filePath
}

// cobolFindingEvidence returns the normalized evidence text of g under
// graph.FindingEvidenceRule: the imaged original bytes with the
// fixed-format sequence and identification areas dropped, else the
// generated bytes, whitespace-collapsed, else the gap kind and its absence
// reason. A zero-width span yields the empty string; the ID still differs by
// kind and node kind.
func cobolFindingEvidence(w *cobolWalk, g handoff.Gap) string {
	if o := g.Original; o != nil && !o.NoImage && !o.Synthetic &&
		o.Start.Byte <= o.End.Byte && uint64(o.End.Byte) <= uint64(len(w.src)) {
		return cobolFixedEvidenceText(w.src[o.Start.Byte:o.End.Byte], int(o.Start.Column))
	}
	if s := g.Generated; s != nil && s.Start <= s.End && uint64(s.End) <= uint64(len(w.h.Generated)) {
		return strings.Join(strings.Fields(string(w.h.Generated[s.Start:s.End])), " ")
	}
	return string(g.Kind) + "\x00" + g.AbsenceReason
}

// cobolFixedEvidenceText keeps only the bytes of b at 0-based columns 6
// through 71 of each physical line, b starting at column, then collapses
// whitespace runs to one space and trims. Renumbering columns 1-6 or
// stamping columns 73-80 therefore never changes a finding ID. Case is kept.
func cobolFixedEvidenceText(b []byte, column int) string {
	kept := make([]byte, 0, len(b))
	for _, c := range b {
		if c == '\n' {
			kept = append(kept, ' ')
			column = 0
			continue
		}
		if column >= 6 && column < 72 {
			kept = append(kept, c)
		}
		column++
	}
	return strings.Join(strings.Fields(string(kept)), " ")
}
