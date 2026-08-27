# cobolprobe — is the vendored COBOL grammar usable on a real estate?

Measurement harness, not a unit test. Every test skips unless given a corpus
flag, so it never runs in CI.

    go test ./internal/parser/forest/cobolprobe/ -v -timeout 20m \
        -corpus      ~/repos/mine/cobolCode/cam-corpus-dcc/DCC \
        -neut-corpus ~/repos/mine/cobolCode/cam-corpus-dcc/DCC

The grammar under test is `github.com/alexaandru/go-sitter-forest/cobol`
v1.9.1, which vendors **`yutaro-sakamoto/tree-sitter-cobol`** (MIT, revision
`e99dbdc3`). It is already a `go.mod` dependency and already compiled in; no
extractor is registered for it, because `cobol.go` claims `.cbl`/`.cpy`.

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

`TestErrorCascade` pins that behaviour. It is why a 5,352-line program yields
4 ERROR nodes but only 5 of its 77 paragraphs: recovery is not local.

Ruled out by `TestHypothesisIdentificationParagraphs` — these all parse clean:
`INSTALLATION.` / `AUTHOR.` / `DATE-WRITTEN.` comment-entry paragraphs, banner
comments, change markers in columns 1-6, and every `COPY IDMS` form.

## What preprocessing does

`neutralize()` rewrites the unknown constructs into valid COBOL of the same
shape, preserving line count so line numbers stay usable, and copybooks get a
synthetic program shell. That takes `.cpy` from 0% to **93%** recall and lifts
`.cbl` by 81% in absolute data items. The neutralised constructs are exactly the
ones the island regexes already extract correctly, so nothing is lost.

## Conclusion

**Adopt for copybooks now** — 93% recall on the files that hold the field
layouts, which is the DATA DIVISION content the regex path cannot produce at
all. **Programs need the grammar extended** (`EXEC CICS`/`EXEC SQL` + IDMS DML)
before tree-sitter beats the island extractor at 27% recall; keep `cobol.go`
for `.cbl` until then. Extending the grammar is upstreamable to
`yutaro-sakamoto/tree-sitter-cobol` — the four families are well-defined
statement forms.
