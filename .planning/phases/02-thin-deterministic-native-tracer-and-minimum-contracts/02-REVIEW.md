---
phase: 02-thin-deterministic-native-tracer-and-minimum-contracts
reviewed: 2026-10-01T17:35:00Z
depth: standard
files_reviewed: 26
files_reviewed_list:
  - .github/workflows/bench-arm.yml
  - .github/workflows/ci.yml
  - .github/workflows/init-smoke.yml
  - .github/workflows/publish-claude-plugin.yml
  - .github/workflows/release.yml
  - .github/workflows/security.yml
  - .github/workflows/skill-drift.yml
  - cmd/gortex/cobol_tracer_test.go
  - cmd/gortex/neo4j.go
  - go.mod
  - internal/graph/store_sqlite/scoped_projection.go
  - internal/indexer/indexer.go
  - internal/indexer/source_revision.go
  - internal/indexer/source_revision_test.go
  - internal/neo4jprojection/neo4j.go
  - internal/neo4jprojection/neo4j_test.go
  - internal/neo4jprojection/properties_test.go
  - internal/neo4jprojection/service_test.go
  - internal/parser/forest/cobolprobe/hypo_test.go
  - internal/parser/languages/cobol_grammar.go
  - internal/parser/languages/cobol_grammar_identity.go
  - internal/parser/languages/cobol_grammar_test.go
  - internal/parser/languages/cobol_grammar_windows.go
  - internal/parser/languages/cobol_grammar_windows_test.go
  - internal/parser/languages/register.go
  - internal/testutil/graphfixture/fixture.go
  - <fork>/preprocessor/analyze.go
  - <fork>/preprocessor/analyze_test.go
  - <fork>/preprocessor/pipeline.go
  - <fork>/preprocessor/internal/tsadapter/parser.go
  - <fork>/preprocessor/internal/tsadapter/parser_test.go
findings:
  critical: 1
  warning: 11
  info: 10
  total: 22
status: issues_found
---

# Phase 2: Code Review Report

**Reviewed:** 2026-10-01T17:35:00Z
**Depth:** standard
**Files Reviewed:** 26 Gortex files (diff `3238e524..HEAD`), plus the fork commit `f9eaf99` (5 files under `<fork>/preprocessor/`, where `<fork>` = the tree-sitter-cobol-upgrade checkout)
**Status:** issues_found

## Summary

The grammar extractor fails closed as intended. Attestation errors are cached by `sync.Once` and returned on every `Extract`, including copybooks. There is no regex or stock-grammar fallback. The Windows stub refuses all input. Workflows never use `pull_request_target`, and the release container really is kept away from the fork credential: a read-only module cache, `GOPROXY=off`, and no PAT passed in. No estate names, private absolute paths, or run IDs show up in the added code or fixtures.

Problems found:

1. **Revision provenance can be false (BLOCKER).** `prov_vcs_commit` is decided from the git state of the file on disk at stamp time, not from the bytes that were hashed into `prov_revision_content_id`. The always-on BOM stripper, user command transforms, and the gap between reading the file and sampling git all let a HEAD commit be claimed for content HEAD never held.
2. **Containment identity can be confidently wrong.** The END PROGRAM stack decides by a name-count lookahead and matches names case-sensitively. It can produce wrong IDs without setting `prov_containment_unresolved`. A scratch-module probe that runs the real `cobolProgramSymbols` confirmed each case below.
3. **Concurrency and timeout gaps.** The COBOL analyze slot only limits one process, and waiting for it counts against the per-file extraction budget.
4. **CI credential scope.** The PAT stays in `~/.gitconfig` for the whole job. Fork PRs and Dependabot runs fail at the credential step.
5. **Fork attestation identity is weaker than it claims.** `AttestEmbedded` reports a checkout identity it never checked. `ToolIdentity` records the upstream (stock) module version instead of the `replace` target.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: `prov_vcs_commit` is claimed for bytes HEAD does not hold (BOM strip, transforms, TOCTOU)

**File:** `internal/indexer/source_revision.go:33-67`, `internal/indexer/source_revision.go:76-141`; call site `internal/indexer/indexer.go:1721`; byte producers `internal/indexer/file_delta.go:137`, `internal/indexer/indexer.go:3577`, `internal/indexer/transform.go:72,274-299`; content ID `internal/parser/languages/cobol_grammar.go:135,158`

**Issue:** The contract (D-09, T-02-16, the doc comment at `source_revision.go:15-19`) says `prov_vcs_commit` is written "only when the file's bytes are what HEAD holds". The code never compares those bytes. It asks `git status` and `git ls-files` about the file on disk at stamp time and pairs the answer with a content ID hashed from different bytes. There are three provable ways for the claim to be false:

1. **Always-on BOM strip.** `newTransformPipeline` always adds `bomStripTransform` (`matches` returns true for every path). `src` loses its leading UTF-8 or UTF-16 BOM before extraction (`file_delta.go:137`, `indexer.go:3577`). `prov_revision_content_id` (`h.OriginalContentID`, or `NewContentID(src)` for `.cpy`) is then the hash of the stripped bytes. `git status` reports the committed BOM file as clean, so the stamp writes the HEAD commit next to a content ID that no blob in that commit has. On the same files, `prov_start_byte`/`prov_end_byte` (`cobol_grammar.go:201-202`) are off by the BOM length from the real file, which breaks D-10's "original coordinates". User-configured command transforms (`transform.go:79`) cause the same mismatch for any file they rewrite.
2. **TOCTOU between read and sample.** Bytes are read and hashed first. Then `Extract` waits for `cobolAnalyzeSlot` and runs `Analyze` (up to the fork's 2-minute `parseWallClockCeiling`). Only after that does `applyCoverageDomains` call `classifySourceRevision`, which samples git. A `git checkout`, `pull`, `stash`, `reset --hard`, or an editor revert during that window gets the *new* HEAD stamped as clean onto the *old* bytes. A long bulk index of an estate can easily run into a branch switch. Because git is sampled per file, one index pass can also stamp different HEADs on different files.
3. **Index flags.** Files marked `assume-unchanged` or `skip-worktree` never appear in `git status` but do pass `ls-files --error-unmatch`, so modified bytes get stamped with HEAD.

No test covers any of these. Every `source_revision_test.go` case writes the file before stamping and never changes the bytes in between.

**Fix:** Decide the commit from the exact bytes that were hashed. One `ls-tree` against the sampled commit replaces the dirty-entry scan and closes all three paths:

```go
// stampSourceRevision(relPath, hashed []byte, rawChanged bool, result)
// rawChanged: the transform pipeline altered the on-disk bytes.
if rawChanged {
    return "", "indexed_bytes_differ_from_file" // BOM strip / command transform
}
out, err := gitcmd.Run(ctx, top, "--literal-pathspecs", "ls-tree", "-z", snap.HeadCommit, "--", rel)
// parse "<mode> blob <oid>\t<path>\x00"; no entry -> not_under_version_control
want := gitBlobOID(hashed, objectFormat) // sha1/sha256 of "blob <len>\x00"+bytes
if oid != want {
    return "", "working_tree_differs_from_head"
}
return snap.HeadCommit, ""
```

`applyCoverageDomains` already has `src`. Pass it in, along with a flag saying whether `transforms.run` changed the bytes; the raw bytes are available at the read sites. Clean or eol filters then fail toward "differs", which is the safe direction. Add tests for: a BOM-prefixed committed file, a file reverted after hashing, and an `assume-unchanged` file.

## Warnings

### WR-01: The END PROGRAM lookahead produces wrong containment IDs without flagging them

**File:** `internal/parser/languages/cobol_grammar_identity.go:68-113`

**Issue:** `remaining` counts END PROGRAM markers by name across the whole file. Any open frame whose name has a marker anywhere later stays open, even when that marker belongs to a different program. A scratch-module probe running the real `cobolProgramSymbols` on synthetic handoffs gave:

| Observation sequence | Result | `containmentUnresolved` |
|---|---|---|
| `A` (no END), `B`, `END B`, `A`, `END A` | `[A A/B A/A]` (expected `A`, `B`, `A#2`) | `false` |
| `A`, `B`, `END A`, `B`, `END B` | `[A A/B B]` (the `A/B` frame is closed implicitly by `END A`) | `false` |

These IDs are persisted and projected (D-10 is marked "costly" to reverse). The design rule is "no guess", but both cases are guesses reported as resolved.

**Fix:** Mark containment unresolved whenever a frame closes without its own marker. That covers frames skipped by an END PROGRAM pop, and frames still open at end of file that received children:

```go
case "end_program":
    ...
    if top < len(open)-1 { // frames above top close without their own END PROGRAM
        containmentUnresolved = true
    }
    open = open[:top]
...
// after the loop:
for _, f := range open {
    if f.hadChildren { containmentUnresolved = true }
}
```

Set `hadChildren` on the parent frame when a program is pushed under it. Add both sequences above to `cobol_grammar_test.go`.

### WR-02: END PROGRAM matching is case-sensitive, so legal COBOL loses its nesting

**File:** `internal/parser/languages/cobol_grammar_identity.go:69-70,82,104`

**Issue:** COBOL user-defined words are case-insensitive, so `PROGRAM-ID. Outer.` … `END PROGRAM OUTER.` is valid. Because `remaining` and the stack match on verbatim bytes, `remaining["Outer"]` is 0. The next program definition then pops `Outer`, and the nested program gets the top-level ID. Probe: `Outer`, `INNER`, `END INNER`, `END OUTER` gives `[Outer INNER]` instead of `Outer/INNER`. The unresolved flag is set, but the persisted ID is wrong. D-20 only says IDs keep the verbatim name; it does not require verbatim matching. The same issue applies to a literal `PROGRAM-ID. "X".` closed by `END PROGRAM X.`.

**Fix:** Match on a normalized key and keep the verbatim name for the symbol:

```go
key := func(n string) string { return strings.ToUpper(strings.Trim(n, `"'`)) }
remaining[key(name)]++                                  // counting
for len(open) > 0 && remaining[key(open[len(open)-1].name)] == 0 { ... }
for top >= 0 && key(open[top].name) != key(name) { top-- }
```

### WR-03: Coordinates are emitted for no-image and synthetic projections

**File:** `internal/parser/languages/cobol_grammar.go:192-208`

**Issue:** The comment says "Never invent coordinates for a range with no original image", but the code only checks `o.Original != nil`. The fork documents that `Original` "is nil only when the map projection itself errors" (`<fork>/preprocessor/parsefacts/facts.go:70-73`). A projection over inserted or no-image bytes therefore arrives non-nil with `NoImage` or `Synthetic` set (`sourcemap/lookup.go:81,237,297`). Its points are written as `StartLine`/`EndLine`/`prov_start_*`, and the only hint is `prov_range_exact=false`.

**Fix:**

```go
if r := o.Original; r != nil && !r.NoImage && !r.Synthetic {
    ... // as today
} else {
    meta["prov_range_exact"] = false
    meta["prov_range_absence"] = "no_original_projection" // or "synthetic_projection"
}
```

### WR-04: `cobolAnalyzeSlot` neither bounds memory under crash isolation nor stays out of the timeout budget

**File:** `internal/parser/languages/cobol_grammar.go:43-46,141-146`; interacting with `internal/indexer/skip_telemetry.go:114-155`, `internal/indexer/crash_isolation.go:35-57`

**Issue:**
- **Crash isolation.** When `index.crash_isolation` or `GORTEX_PARSER_ISOLATION=1` is set, every `gortex __parse-worker` subprocess has its own `cobolAnalyzeSlot`. N workers then run N COBOL analyses at once, which defeats the "one COBOL parse at a time on a 16 GB host" guarantee the comment promises.
- **Budget accounting.** When `max_extract_millis` is set, the timer in `extractWithTimeoutDone` starts before `Extract` blocks on the slot. COBOL files waiting behind one slow parse use up their budget while queued and are recorded as `skipped_due_to_timeout`. Each abandoned goroutine still takes the slot later and runs `Analyze` with `context.Background()`, up to the 2-minute ceiling, for a result that is thrown away. That stretches the queue for every following file.

**Fix:** Acquire the slot with a deadline and give `Analyze` a context, so a caller that has given up never takes the slot. For example, use `context.WithTimeout` from the extraction options, `select` on the slot send against `ctx.Done()`, and pass that ctx to `Analyze`. Under crash isolation, either send COBOL to one designated worker or document that the memory bound does not hold.

### WR-05: Revision absence reasons are misclassified under load and in common repo setups

**File:** `internal/indexer/source_revision.go:80-139`

**Issue:** Each COBOL file builds a fresh `DirtySampler` with no seed. That costs 4 git processes per file: `rev-parse --show-toplevel`, a full-repo `status --porcelain=v2 --untracked-files=all`, `rev-parse <commit>^{tree}`, and `ls-files`. All four share one 30 s deadline and also wait on the process-wide `gitcmd` semaphore. On a large estate with parallel workers the deadline expires and the file is labelled `git_unavailable` while git is working fine. Other wrong labels:
- `safe.directory` "dubious ownership" failures from `rev-parse` become `not_a_git_worktree`.
- Files inside a submodule fail `ls-files` in the superproject and become `not_under_version_control`.
- A non-timeout `ls-files` failure (`:137-139`) is also reported as `not_under_version_control`.

**Fix:** The CR-01 fix (one `ls-tree` per file against a snapshot) removes most of this. Sample `rev-parse --show-toplevel` and HEAD once per index pass, or cache them by `top` and fingerprint, instead of once per file. Return a distinct `git_timeout` (or `vcs_query_failed`) when `ctx.Err() != nil` or git exits non-zero, so a failure is not passed off as a fact about the file.

### WR-06: The fork PAT stays in `~/.gitconfig` for the rest of every job

**File:** `.github/workflows/ci.yml:33-46` (and the matching steps at 123-136, 181-194, 228-241, 283-296), `.github/workflows/release.yml:43-56,197-215,544-557`, `.github/workflows/bench-arm.yml:27-40,82-95`, `.github/workflows/security.yml:31-43`, `.github/workflows/init-smoke.yml:37-50`, `.github/workflows/skill-drift.yml:45-58`, `.github/workflows/publish-claude-plugin.yml:90-102`

**Issue:** `git config --global url."https://x-access-token:${PAT}@…".insteadOf` writes the PAT in plain text to the runner's `~/.gitconfig`, where it stays for the rest of the job. Every later step can read it: the full `go test -race ./...` suite (which includes code from same-repo PR branches), and third-party actions such as `codecov/codecov-action`, `golangci-lint-action`, `govulncheck-action`, cosign, the SLSA and homebrew steps, and the plugin push. Pinning actions by SHA reduces the risk but does not remove it, and the org's least-privilege rule applies. Log masking only protects logs, not the file.

**Fix:** Use the credential for module download only, then remove it:

```yaml
- name: Download Go modules (fork credential scoped to this step)
  env:
    GIT_CONFIG_COUNT: "1"
    GIT_CONFIG_KEY_0: url.https://x-access-token:${{ secrets.COBOL_PARSER_READ_PAT }}@github.com/MuiGoku123432/tree-sitter-cobol-upgrade.insteadOf
    GIT_CONFIG_VALUE_0: https://github.com/MuiGoku123432/tree-sitter-cobol-upgrade
  run: go mod download
# later steps: env GOPROXY=off / GOFLAGS=-mod=readonly — the cache already holds every module
```

The env-scoped config is never written to disk. If you keep `--global`, add a final `if: always()` step that runs `git config --global --unset-all …insteadOf`, and move the download step before any third-party action.

### WR-07: Fork PRs and Dependabot can no longer pass CI

**File:** `.github/workflows/ci.yml:3-8,33-46`; `.github/dependabot.yml:38` (the gomod ecosystem, which has no `registries:`)

**Issue:** `pull_request` runs from forks get no repository secrets, and Dependabot-triggered runs only get Dependabot secrets. In both cases `COBOL_PARSER_READ_PAT` is empty, so every job exits at `::error::COBOL_PARSER_READ_PAT is not configured`. Dependabot's gomod updater also cannot resolve `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/...` without a `registries:` entry, so Go dependency updates stop. D-13 chose "no skip-when-missing", but neither consequence is recorded in CONTEXT or the summaries. For a public repo, that means no external contributions can pass CI.

**Fix:** Record the decision explicitly. If Dependabot should keep working, add a Dependabot secret plus a `registries: { cobol-fork: { type: git, url: https://github.com, username: x-access-token, password: ${{secrets.COBOL_PARSER_READ_PAT}} } }` entry referenced by the gomod update. Document in CONTRIBUTING that fork PRs need a maintainer re-run.

### WR-08: The approved-grammar pin and provenance do not cover the preprocessor module

**File:** `internal/parser/languages/cobol_grammar.go:24-37,87-106,150-153,161`; `go.mod:6`

**Issue:** `cobolApprovedGrammarID` is the same value as the fork's `expectedCompiledLanguageID`, which `NewEmbeddedAnalyzer` already enforces. The per-handoff check at `:150` therefore only trips if the fork regenerates `expected_identity.go`. A `go.mod` bump of only the `preprocessor` module passes silently, yet that bump can change the transform catalog, grade policy, handoff shape, or observation semantics, which is exactly what the mapping depends on. That contradicts the BASE-01 claim in the comment at `:25-29`. `prov_parser_module` only records the forest-shim (`:92-101`), and `ToolID` does not include the preprocessor version (see FK-02), so the provenance cannot tell two preprocessor versions apart either. The check also runs after a full `Analyze` of every file instead of once at init.

**Fix:** In `init()`, read the build-info entry for `github.com/MuiGoku123432/tree-sitter-cobol-upgrade/preprocessor`. Pin it next to the grammar ID (an approved `path@version` constant). Emit it as `prov_preprocessor_module`. Fail init on a mismatch, so no file is parsed under an unapproved producer. Bump `cobolGrammarExtractorVersion` with the new key.

### WR-09: The `!windows` build tag breaks every non-darwin/linux build

**File:** `internal/parser/languages/cobol_grammar.go:1`, `cobol_grammar_identity.go:1`, `cobol_grammar_windows.go:1`, `cobol_grammar_test.go:1`, `cmd/gortex/cobol_tracer_test.go:1`

**Issue:** The stub's own comment (`cobol_grammar_windows.go:11-24`) says the preprocessor's securefs builds "only on darwin and linux". `!windows` also selects freebsd, openbsd, netbsd, illumos, and so on. Those builds now fail to compile instead of getting the fail-closed stub.

**Fix:** Use `//go:build darwin || linux` on the real extractor and its tests, `//go:build !darwin && !linux` on the stub and its test, and rename the stub file (for example `cobol_grammar_unsupported.go`). Update the error text to say the platform is unsupported rather than naming Windows.

## Info

### IN-01: `h.Generated` is sliced without a bounds check

**File:** `internal/parser/languages/cobol_grammar_identity.go:51`
**Issue:** `h.Generated[o.Generated.Start:o.Generated.End]` panics on an inconsistent span. `safeExtractWithOptions` recovers the panic, but the whole file is then lost with a panic error instead of a clear one.
**Fix:** Check `Start <= End && End <= uint64(len(h.Generated))` and skip the observation or return a wrapped error.

### IN-02: Symbol delimiters are not escaped

**File:** `internal/parser/languages/cobol_grammar_identity.go:87,92`
**Issue:** If `program_name` bytes for a literal `PROGRAM-ID. "A/B".` come through without quotes, a top-level `A/B` collides with program `B` nested in `A`, and a literal `"X#2"` collides with the second `X`. This is not verified against the grammar's literal handling.
**Fix:** Percent-escape `/`, `#` and `%` in the name segment, or add a fixture proving quotes are kept.

### IN-03: The pin is checked after a full parse instead of at attestation

**File:** `internal/parser/languages/cobol_grammar.go:150-153`
**Issue:** With a wrong pin, every COBOL file is fully analyzed before it is rejected.
**Fix:** Run a one-off `Analyze` of an empty document in `init()` to read `Tool.CompiledLanguageID`, or check it against the analyzer's identity, and cache the result in `initErr`.

### IN-04: Git subprocesses inherit `GIT_DIR`, `GIT_INDEX_FILE` and `GIT_WORK_TREE`

**File:** `internal/indexer/source_revision.go:90,133` (through `gitcmd.Run`, which uses `env=nil`)
**Issue:** When gortex runs from a git hook (pre-commit uses a temporary `GIT_INDEX_FILE`), status and ls-files read the hook's index or repo, not the checkout. A provenance claim now depends on that.
**Fix:** Clear those variables for these calls (a scrubbed env like `noLazyGitEnv`).

### IN-05: Stamps may go stale when only git state changes (not verified)

**File:** `internal/indexer/source_revision.go:26-30`
**Issue:** The stamp is computed only when the file is re-extracted. A dirty file that is then committed without its bytes changing may keep `working_tree_differs_from_head` until something re-extracts it. I did not check whether the checkout lifecycle re-extracts files when HEAD moves.
**Fix:** Confirm that a HEAD-change re-extracts COBOL files, or re-stamp revision keys in the HEAD-change handler.

### IN-06: Dead `cache-dependency-path`

**File:** `.github/workflows/publish-claude-plugin.yml:88`
**Issue:** `cache-dependency-path` has no effect when `cache: false`.
**Fix:** Remove it.

### IN-07: The stock-grammar negative step greps a fork-internal error string

**File:** `.github/workflows/ci.yml:70,75`
**Issue:** If the fork rewords `errors.New("compiled forest language identity mismatch")` (`<fork>/preprocessor/internal/tsadapter/parser.go`), the step reports "failed for the wrong reason" even though the gate still works.
**Fix:** Have the fork export a sentinel error (for example `preprocessor.ErrGrammarIdentityMismatch`). Assert it with `errors.Is` in the test, and grep for the test's own failure marker.

### IN-08: A private absolute path stays in public git history

**File:** `internal/parser/forest/cobolprobe/README.md` (the removed line in this diff)
**Issue:** The scrub removes `/Users/<user>/repos/mine/devDeps/tree-sitter-cobol-upgrade/...` from the tree. The line came from a commit before `3238e524`, so it is still in history if that commit was pushed.
**Fix:** Accept it, or rewrite history before the branch goes public. No action is needed in code.

## Fork commit `f9eaf99` (producer side, `<fork>/preprocessor/`)

### FK-01 (WARNING): `AttestEmbedded` reports a checkout identity it never checked

**File:** `<fork>/preprocessor/internal/tsadapter/parser.go` (`AttestEmbedded`), `<fork>/preprocessor/analyze.go` (`NewEmbeddedAnalyzer`)
**Issue:** `AttestEmbedded` returns `CheckoutArtifactID: expectedCheckoutArtifactID`, a generated constant, without hashing any shim artifact. `ToolIdentity.ID()` hashes that field, so an embedded-attested run produces the same `ToolID` as a run that actually fingerprinted the shim files. The test `TestNewEmbeddedAnalyzerMatchesCheckoutAnalyzer` requires this equality. That contradicts the `Analyzer` doc ("a caller cannot claim a parser identity it did not attest", T-08-01). The commit message relies on go.sum pinning the shim, but that only holds for a versioned `replace`. A local-directory `replace` has no go.sum pin and would still report the attested checkout ID.
**Fix:** Record how attestation was done. Either add `AttestationMode` (`"checkout"` / `"embedded"`) to `ToolIdentity` and fold it into `ID()`, or leave `CheckoutArtifactID` empty for embedded attestation. Gortex then pins whichever `ToolID` the embedded path produces.

### FK-02 (WARNING): `ToolIdentity.ForestModuleVersion` ignores `replace`, so it records the stock version

**File:** `<fork>/preprocessor/pipeline.go` (`validateShim` → `dependencyVersion`)
**Issue:** `dependencyVersion` returns `dependency.Version` and ignores `dependency.Replace`. A module consumer can only link the fork grammar through a `replace`, and Gortex's `go.mod:29` keeps `require …/go-sitter-forest/cobol v1.9.1`. So `ForestModuleVersion` is `v1.9.1`, the stock grammar's version, inside the fork grammar's identity, and `ToolID` does not change when the `replace` target moves. Gortex works around this with `prov_parser_module`, but the handoff's own `Tool` and `ToolID` are wrong for every module consumer, and the embedded path in this commit is what makes such consumers possible.
**Fix:**

```go
version := dependency.Version
if dependency.Replace != nil {
    path, version = dependency.Replace.Path, dependency.Replace.Version
}
```

Record the replacement path too, and treat an empty replace version (a local directory) as unattested for the embedded path.

### FK-03 (INFO): No negative test for an embedded compiled-identity mismatch

**File:** `<fork>/preprocessor/internal/tsadapter/parser_test.go` (`TestAttestEmbeddedRejectsUnavailableLanguage`)
**Issue:** Only the nil-language path is tested. The mismatch branch is exercised only by Gortex's CI stock-grammar step.
**Fix:** Stub `languagePointer` to another tree-sitter language and assert the mismatch error and a zero `Identity`.

### FK-04 (INFO): Attestation only sees scanner behaviour through sample parses

**File:** `<fork>/preprocessor/internal/tsadapter/parser.go` (`compiledIdentity`)
**Issue:** The compiled ID covers parse tables, queries, `Info()`, and the `identityObservationSources` parses. A changed `scanner.c` whose effect those samples do not trigger passes both the fork attestation and Gortex's pin. The embedded path no longer hashes `scanner.c` from a checkout.
**Fix:** Add external-scanner-sensitive inputs (continuation lines, sequence area, comment indicators) to `identityObservationSources`, or embed a `scanner.c` hash in the shim module and check it in `AttestEmbedded`.

---

_Reviewed: 2026-10-01T17:35:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
