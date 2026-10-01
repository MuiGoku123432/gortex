# cobolprobe — is the vendored COBOL grammar usable on a real estate?

Measurement and parser-acceptance harness, not a production extractor test.
Corpus measurements skip unless given a corpus flag. The cascade acceptance
gate skips unless `-enhanced-parser` is set.

The enhanced parser is linked through go.mod: a `require` of the private
fork's `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor` module
and a `replace` of `github.com/alexaandru/go-sitter-forest/cobol` with the
fork's `forest-shim/cobol`, both at the pinned pseudo-version. `GOPRIVATE`
must cover the fork. Run the cascade gate with:

    GOWORK=off go test -race ./internal/parser/forest/cobolprobe/ \
        -run TestErrorCascade -v -enhanced-parser

The corpus measurements remain opt-in and take a local source directory:

    GOWORK=off go test ./internal/parser/forest/cobolprobe/ -v -timeout 20m \
        -corpus      "$COBOL_CORPUS_ROOT" \
        -neut-corpus "$COBOL_CORPUS_ROOT"

To iterate on a local fork checkout, point both fork modules at it through a
temporary modfile instead of editing go.mod (never through go.work):

    tmp="$(mktemp -d)"
    cp go.mod "$tmp/dev.mod"
    cp go.sum "$tmp/dev.sum"
    go mod edit -modfile="$tmp/dev.mod" \
        -replace=github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor="$COBOL_UPGRADE_ROOT/preprocessor" \
        -replace=github.com/alexaandru/go-sitter-forest/cobol="$COBOL_UPGRADE_ROOT/forest-shim/cobol"
    GOWORK=off go test -modfile="$tmp/dev.mod" -race \
        ./internal/parser/forest/cobolprobe/ -run TestErrorCascade -v -enhanced-parser

An edited grammar fails the fork's embedded attestation and Gortex's
approved-grammar pin until the fork regenerates its expected identity and
Gortex re-pins deliberately.

`internal/parser/languages/cobol_grammar.go` is the production COBOL
extractor. The regex extractor in `cobol.go` stays in-tree but unregistered.
The measurements below were taken against the stock
`github.com/alexaandru/go-sitter-forest/cobol` v1.9.1 grammar, which vendors
**`yutaro-sakamoto/tree-sitter-cobol`** (MIT, revision `e99dbdc3`).

## What it measured, on 1,564 files (606 `.cbl`, 958 `.cpy`)

Recall against source truth — level-number lines and area-A paragraph labels
counted directly from the code area:

| | fields in source | grammar | + preprocessing |
|---|---|---|---|
| `.cbl` | 121,457 | 17,905 (15%) | 32,360 (**27%**) |
| `.cpy` | 24,478 | 0 (0%) | 22,800 (**93%**) |

The regex extractor in `internal/parser/languages/cobol.go` produces **zero**
data items, so anything here is a strict gain.

## The four constructs that break it, and why it matters

Each costs exactly one ERROR node, and **the error cascades to the end of its
division**:

| construct | effect |
|---|---|
| `EXEC CICS` / `EXEC SQL` / `EXEC DLI` | 1 error → **1 of 20** following paragraphs survive |
| IDMS DML (`OBTAIN`, `FIND`, `STORE`, …) | same |
| IDMS `SCHEMA SECTION` / `DB x WITHIN y` | 1 error → **0 of 20** data items survive |
| a copybook parsed standalone | no program structure → whole file becomes `comment_entry` |

`TestErrorCascade` records that original failure and, when run with
`-enhanced-parser`, enforces recovery parity for the IDMS, CICS, and SQL forms
now supported by `tree-sitter-cobol-upgrade/main`. The `SCHEMA SECTION` and
invalid DATA DIVISION controls remain diagnostic rather than parity gates.

Ruled out by `TestHypothesisIdentificationParagraphs` — these all parse clean:
`INSTALLATION.` / `AUTHOR.` / `DATE-WRITTEN.` comment-entry paragraphs, banner
comments, change markers in columns 1-6, and every `COPY IDMS` form.

## What preprocessing does

`neutralize()` rewrites the unknown constructs into valid COBOL of the same
shape, preserving line count so line numbers stay usable, and copybooks get a
synthetic program shell. That takes `.cpy` from 0% to **93%** recall and lifts
`.cbl` by 81% in absolute data items. The neutralised constructs are exactly the
ones the island regexes already extract correctly, so nothing is lost.

## Conclusion (stock grammar, before the enhanced parser)

**Adopt for copybooks now** — 93% recall on the files that hold the field
layouts, which is the DATA DIVISION content the regex path cannot produce at
all. **Programs need the grammar extended** (`EXEC CICS`/`EXEC SQL` + IDMS DML)
before tree-sitter beats the island extractor at 27% recall; keep `cobol.go`
for `.cbl` until then. Extending the grammar is upstreamable to
`yutaro-sakamoto/tree-sitter-cobol` — the four families are well-defined
statement forms.
