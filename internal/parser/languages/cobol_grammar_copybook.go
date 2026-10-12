//go:build darwin || linux

package languages

import (
	"maps"
	"path/filepath"
	"strings"

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
// written in the member (D-11, D-19), so impact queries reach them.
// RATIONALE: the producer's copybook mode wraps the member in a synthetic
// header (D-16). Its first program_definition is that wrapper, registered as
// the walk root so cobolReferences and cobolCopy name the file node instead.
// The wrapper's program, divisions, and sections are never emitted. The
// walk gets its own docMeta copy, so COPY marks and binding keys written on
// the file node never leak into item or edge Meta.
// KEYWORDS: cobol, copybook, wrapper, COPY, imports, data item.
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
}
