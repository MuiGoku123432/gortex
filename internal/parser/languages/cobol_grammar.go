//go:build !windows

package languages

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime/debug"
	"sync"

	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/handoff"
	"github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor/transform"

	"github.com/zzet/gortex/internal/graph"
	"github.com/zzet/gortex/internal/parser"
)

const (
	// cobolApprovedGrammarID is the Gortex-side approved-grammar pin. The
	// preprocessor's embedded attestation proves the linked grammar is the
	// one that module was generated against; this constant proves that
	// module is the approved one, so a go.mod bump to another fork commit
	// cannot silently move the parser baseline (BASE-01, BASE-02).
	cobolApprovedGrammarID = "f97452e11a2b80b92acb7edf776131470bc2e1ffd7c077f4ade4ce3c3a47503d"
	// cobolGrammarExtractorVersion identifies this handoff-to-graph mapping.
	// Bump it whenever the emitted nodes, edges, or prov_* keys change.
	cobolGrammarExtractorVersion = "gortex-cobol-grammar/1"
	// cobolForestModulePath is the module path the enhanced grammar is
	// linked under; go.mod replaces it with the fork's forest-shim.
	cobolForestModulePath = "github.com/alexaandru/go-sitter-forest/cobol"
)

// errCobolGrammarNotApproved reports a handoff produced by a grammar other
// than the approved one. Extraction fails; nothing falls back (D-05).
var errCobolGrammarNotApproved = errors.New("COBOL handoff grammar is not the approved enhanced grammar")

// cobolAnalyzeSlot serializes Analyze across every extractor instance. A
// single COBOL parse can need a large share of memory on a 16 GB host, so
// COBOL files parse one at a time while other languages stay parallel.
var cobolAnalyzeSlot = make(chan struct{}, 1)

// CobolGrammarExtractor extracts COBOL program definitions from the
// enhanced tree-sitter-cobol-upgrade grammar through its preprocessor
// handoff (cobol-handoff-v1). It emits the file node, one KindFunction node
// per named program with cobol_kind=program, and a file-to-program
// EdgeDefines edge, each carrying the prov_* provenance keys. When the
// enhanced parser is unavailable or unapproved, Extract returns an error
// and no result: there is no regex or stock-grammar fallback.
type CobolGrammarExtractor struct {
	approvedGrammarID string
	once              sync.Once
	analyzer          preprocessor.Analyzer
	parserModule      string
	initErr           error
}

// NewCobolGrammarExtractor returns a grammar extractor pinned to the
// approved grammar. Attestation runs lazily on the first Extract, because
// RegisterAll runs for every gortex command.
func NewCobolGrammarExtractor() *CobolGrammarExtractor {
	return &CobolGrammarExtractor{approvedGrammarID: cobolApprovedGrammarID}
}

// Language returns "cobol".
func (e *CobolGrammarExtractor) Language() string { return "cobol" }

// Extensions returns the COBOL source and copybook extensions, the same set
// the regex CobolExtractor claims, so registering this extractor replaces it.
func (e *CobolGrammarExtractor) Extensions() []string {
	return []string{".cob", ".cbl", ".cpy", ".COB", ".CBL", ".CPY"}
}

// init attests the embedded grammar and records the parser module identity.
// Its error is cached so every later Extract fails the same way.
func (e *CobolGrammarExtractor) init() {
	analyzer, err := preprocessor.NewEmbeddedAnalyzer(transform.NewCatalog(nil))
	if err != nil {
		e.initErr = fmt.Errorf("enhanced COBOL parser unavailable: %w", err)
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		e.initErr = fmt.Errorf("enhanced COBOL parser unavailable: %w", errors.New("binary carries no build info"))
		return
	}
	for _, dep := range info.Deps {
		if dep.Path != cobolForestModulePath {
			continue
		}
		module := dep
		if dep.Replace != nil {
			module = dep.Replace
		}
		e.parserModule = module.Path + "@" + module.Version
	}
	if e.parserModule == "" {
		e.initErr = fmt.Errorf("enhanced COBOL parser unavailable: %w",
			fmt.Errorf("build info lists no %s dependency", cobolForestModulePath))
		return
	}
	e.analyzer = analyzer
}

// Extract analyzes one COBOL document and maps its handoff onto graph nodes
// and edges. filePath is the unprefixed repository-relative path; the
// indexer applies the repository prefix and scope fields afterwards.
func (e *CobolGrammarExtractor) Extract(filePath string, src []byte) (*parser.ExtractionResult, error) {
	e.once.Do(e.init)
	if e.initErr != nil {
		return nil, e.initErr
	}
	sourceID, err := preprocessor.NewSourceID(filePath)
	if err != nil {
		return nil, fmt.Errorf("COBOL source identity for %s: %w", filePath, err)
	}
	var h handoff.Handoff
	func() {
		cobolAnalyzeSlot <- struct{}{}
		defer func() { <-cobolAnalyzeSlot }()
		h, err = e.analyzer.Analyze(context.Background(), sourceID, src)
	}()
	if err != nil {
		return nil, fmt.Errorf("enhanced COBOL analysis of %s: %w", filePath, err)
	}
	if h.Tool.CompiledLanguageID != e.approvedGrammarID {
		return nil, fmt.Errorf("%w: handoff grammar %s, approved grammar %s",
			errCobolGrammarNotApproved, h.Tool.CompiledLanguageID, e.approvedGrammarID)
	}

	docMeta := map[string]any{
		"prov_source_path":            filePath,
		"prov_source_id":              string(h.SourceID),
		"prov_revision_content_id":    string(h.OriginalContentID),
		"prov_parser_tool_id":         h.ToolID,
		"prov_parser_grammar_id":      h.Tool.CompiledLanguageID,
		"prov_parser_module":          e.parserModule,
		"prov_parse_config_id":        h.ParseConfigurationID,
		"prov_transform_config_id":    h.Ledger.ConfigurationID,
		"prov_handoff_schema":         h.SchemaVersion,
		"prov_extractor_version":      cobolGrammarExtractorVersion,
		"prov_evidence_class":         "DETERMINISTIC",
		"prov_origin":                 graph.OriginASTResolved,
		"prov_confidence":             1.0,
		"prov_document_grade":         string(h.Assessment.Grade),
		"prov_document_grade_reasons": append([]string{}, h.Assessment.Reasons...),
		"prov_grade_policy":           h.Assessment.PolicyVersion,
	}
	fileNode := &graph.Node{
		ID: filePath, Kind: graph.KindFile, Name: filePath,
		FilePath: filePath, StartLine: 1, EndLine: bytes.Count(src, []byte("\n")) + 1,
		Language: "cobol", Meta: docMeta,
	}
	result := &parser.ExtractionResult{Nodes: []*graph.Node{fileNode}}

	names := cobolProgramNames(h)
	for i, o := range h.Facts.Observations {
		name, ok := names[i]
		if o.Kind != "program_definition" || !ok {
			continue
		}
		meta := maps.Clone(docMeta)
		meta["cobol_kind"] = "program"
		meta["prov_observation_kind"] = o.Kind
		meta["prov_affected"] = o.Affected
		id := filePath + "::" + name
		node := &graph.Node{
			ID: id, Kind: graph.KindFunction, Name: name, QualName: name,
			FilePath: filePath, Language: "cobol", Meta: meta,
		}
		if r := o.Original; r != nil {
			node.StartLine = int(r.Start.Row) + 1
			node.EndLine = int(r.End.Row) + 1
			node.StartColumn = int(r.Start.Column)
			node.EndColumn = int(r.End.Column)
			meta["prov_start_row"] = int(r.Start.Row)
			meta["prov_start_column"] = int(r.Start.Column)
			meta["prov_end_row"] = int(r.End.Row)
			meta["prov_end_column"] = int(r.End.Column)
			meta["prov_start_byte"] = int(r.Start.Byte)
			meta["prov_end_byte"] = int(r.End.Byte)
			meta["prov_range_exact"] = r.Exact
		} else {
			// Never invent coordinates for a range with no original image.
			meta["prov_range_exact"] = false
			meta["prov_range_absence"] = "no_original_projection"
		}
		result.Nodes = append(result.Nodes, node)
		result.Edges = append(result.Edges, &graph.Edge{
			From: fileNode.ID, To: id, Kind: graph.EdgeDefines,
			FilePath: filePath, Line: node.StartLine,
			Origin: graph.OriginASTResolved, Confidence: 1.0, ConfidenceLabel: "EXTRACTED",
		})
	}
	return result, nil
}

// cobolProgramNames maps each program_definition observation index to its
// verbatim name: the program_name child of the identification_division
// child of that definition. program_name children of end_program markers
// are not definitions and never match.
func cobolProgramNames(h handoff.Handoff) map[int]string {
	obs := h.Facts.Observations
	parentOf := func(i int) (int, bool) {
		p := obs[i].Parent
		return p, p >= 0 && p < len(obs) && p != i
	}
	names := make(map[int]string)
	for i, o := range obs {
		if o.Kind != "program_name" {
			continue
		}
		div, ok := parentOf(i)
		if !ok || obs[div].Kind != "identification_division" {
			continue
		}
		def, ok := parentOf(div)
		if !ok || obs[def].Kind != "program_definition" {
			continue
		}
		if _, seen := names[def]; !seen {
			names[def] = string(h.Generated[o.Generated.Start:o.Generated.End])
		}
	}
	return names
}

var _ parser.Extractor = (*CobolGrammarExtractor)(nil)
