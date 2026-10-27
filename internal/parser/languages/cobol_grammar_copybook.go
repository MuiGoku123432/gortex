//go:build darwin || linux

package languages

import (
	"maps"
	"path/filepath"
	"strings"

	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/grade"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/handoff"

	"github.com/zzet/gortex/internal/parser"
)

// cobolCopybookRoot is the root symbol a copybook walk registers for the
// producer's synthetic wrapper program. The copybook file node stands in for
// it (cobolWalk.id), so the wrapper is never a node or an edge endpoint, and
// copybook procedure IDs start with it. '@' cannot begin a COBOL word, so no
// written name collides with it.
const cobolCopybookRoot = "@proc"

// cobolCopybook maps a standalone copybook's handoff onto result, whose only
// node so far is the copybook file node with docMeta.
//
// PURPOSE: the copybook file node owns the member's data items (D-07), its
// procedure sections, paragraphs, statements, and PERFORM/GO TO/CALL edges
// (D-01, D-02, D-05), its IDMS, CICS, and DB2 edges (D-13, D-14, D-15),
// and every COPY, -INC, COPY IDMS, and EXEC SQL INCLUDE
// written in the member (D-11, D-19), so impact queries reach them. Its
// producer parse gaps and any uncontained cascade become findings, as for
// a program (GAP-01, D-13).
// RATIONALE: the producer's copybook mode wraps the member in a synthetic
// header (D-16). Its first program_definition is that wrapper, registered as
// the walk root so cobolReferences and cobolCopy name the file node instead.
// The wrapper's program, divisions, and sections are never emitted. The
// walk gets its own docMeta copy, so COPY marks and binding keys written on
// the file node never leak into item or edge Meta.
// WINDOW (review WR-04): the producer picks the data or procedure header
// from the member's first code line, so a member opening with EXEC CICS,
// FD/SD, a leading COPY, or -INC can get the wrong one and degrade. Gortex
// cannot pick or re-wrap the header without a producer change (that would
// invent coordinates), so the file node records the producer's choice as
// cobol_copybook_wrapper and, when a non-green analysis mapped no member
// node or edge, prov_analysis_absence=copybook_content_unmapped.
// KEYWORDS: cobol, copybook, wrapper, COPY, imports, data item, WR-04.
func cobolCopybook(h handoff.Handoff, src []byte, filePath string, docMeta map[string]any, result *parser.ExtractionResult) {
	var programs []cobolProgramSymbol
	for r, o := range h.Facts.Observations {
		if o.Kind == "program_definition" {
			programs = []cobolProgramSymbol{{index: r, symbol: cobolCopybookRoot}}
			break
		}
	}
	w := newCobolWalk(h, src, filePath, maps.Clone(docMeta), programs, result)
	w.root = cobolCopybookRoot
	// Procedure members: sections, paragraphs, statements, and flow (no-op
	// for a data member, whose wrapper has no procedure division).
	cobolProcedure(w)
	idx := &cobolDataIndex{seen: make(map[string]int), taken: make(map[string]bool)}
	for s, o := range h.Facts.Observations {
		if p, ok := w.parent(s); ok && h.Facts.Observations[p].Kind == "data_division" && strings.HasSuffix(o.Kind, "_section") {
			cobolDataSection(w, idx, filePath, "", "", s)
		}
	}
	cobolDataBind(w, idx)
	cobolReferences(w, idx)
	// IDMS, CICS, and DB2 use in a procedure member, in Extract's order.
	cobolMainframe(w, idx)
	cobolSQL(w, idx)
	// Set after the walk so only the file node carries the binding keys
	// resolveCobolReferences matches a COPY on.
	base := filepath.Base(filePath)
	docMeta["cobol_kind"] = "copybook"
	docMeta["cobol_name_key"] = cobolIDSegment(cobolNameNorm(strings.TrimSuffix(base, filepath.Ext(base))))
	docMeta["cobol_name_norm"] = cobolNameNormRule
	if len(programs) > 0 {
		for _, c := range w.children[programs[0].index] {
			if k := h.Facts.Observations[c].Kind; k == "data_division" || k == "procedure_division" {
				docMeta["cobol_copybook_wrapper"] = strings.TrimSuffix(k, "_division")
				break
			}
		}
	}
	if len(result.Nodes) == 1 && len(result.Edges) == 0 && h.Assessment.Grade != grade.Green {
		docMeta["prov_analysis_absence"] = "copybook_content_unmapped"
	}
	// Last, so the unmapped check above counts only member nodes and edges.
	cobolFindings(w)
}
