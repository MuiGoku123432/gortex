---
phase: 05-explicit-unresolved-evidence
plan: 17
subsystem: cobol-findings
tags: [cobol, docs, findings, placeholder, phase-gate, race-suite, gap-closure, d-21, d-22, d-23, d-24, d-25, d-26, d-27]

requires:
  - phase: 05-explicit-unresolved-evidence
    provides: "identity and evidence rules v2 (05-13); settle and master-drain ledger timing (05-14); kind-guarded resource evicter (05-15); placeholder namespace, by-ID sweep, and refusal (05-16)"
provides:
  - "docs/languages.md COBOL row: D-21 (name-stable container, #k adjacency qualification, parser upgrades kept), D-22 (unkeyed digest, guess confirmation by hashing), D-23/D-24 (history timing), D-25 (reference form in external evidence), D-26 (one finding per occurrence, mixed-form same-line window), D-27 (<repo>::cobol-missing::<type>/<key>); D-20 sentence kept"
  - "COBOL row additions: old-store rebuild note, reserved repository names, LOAD in the external-finding list"
  - "Gate evidence in the phase directory: run 3 log at the plan path (exit=1), run 1 log (exit=1), cmd/gortex-alone log (exit=0)"
  - "deferred-items.md entry for the cmd/gortex TestMain query-log guard flaw; open ledger entry #8 (unmet-truth) in .planning/WINDOWS.md"
affects: [05-verification, 06-native-deterministic-acceptance-and-pre-ai-release-gate]

# Phase gate disposition. The gate did NOT pass as written.
gate:
  verify_met: false
  disposition: "accepted on split evidence by user decision 2026-10-26"
  logs:
    - "05-17-full-suite.log (run 3): exit=1"
    - "05-17-full-suite.run1.log (run 1, every package executed): exit=1"
    - "05-17-cmd-gortex-isolated.log (cmd/gortex alone after run 2): exit=0"

actuals:
  tokens: 22841   # chars/4 over lines added in 0e6221d0..a4b19971 (docs row 2335; the rest is the two full-suite logs, the isolated log, and the deferred entry); this SUMMARY and the ledger edit excluded
  tasks: 2
  commits: 2      # task commits 3abe472d and a4b19971; plus 2 metadata commits

tech-stack:
  added: []
  patterns:
    - "A gate that needs more than one run keeps every run's log in the phase directory, not only the one at the plan path"
    - "A cmd/gortex FAIL whose guard report names only ~/.gortex/cache/query-log.jsonl while every test printed PASS is an outside writer until shown otherwise; confirm by running cmd/gortex alone with -count=1"

key-files:
  created:
    - .planning/phases/05-explicit-unresolved-evidence/05-17-full-suite.log
    - .planning/phases/05-explicit-unresolved-evidence/05-17-full-suite.run1.log
    - .planning/phases/05-explicit-unresolved-evidence/05-17-cmd-gortex-isolated.log
  modified:
    - docs/languages.md
    - .planning/phases/05-explicit-unresolved-evidence/deferred-items.md
    - .planning/WINDOWS.md

key-decisions:
  - "User decision 2026-10-26: Task 2's phase gate is accepted on split evidence. Run 1 executed every package and failed only cmd/gortex's end-of-run query-log guard; cmd/gortex passed alone twice with the guard intact. The gate verify (last line exit=0, no FAIL line) is not met literally; this is a recorded deviation, not a pass."
  - "The cmd/gortex TestMain guard (cmd/gortex/main_test.go) fails the package whenever anything on the machine writes the real ~/.gortex/cache/query-log.jsonl during go test ./...; pre-existing since 34e1f482 (2026-08-08), logged in deferred-items.md, not fixed in Phase 5."
  - "The COBOL row also documents the old-store rebuild note: gortex-cobol-grammar/6 is kept, so a pre-gap-closure store keeps old placeholders and the old parse-finding IDs of unchanged files until a clean re-index."
  - "GAP-02, GAP-04, GAP-05, GAP-06, GAP-07 not marked complete; the phase verifier re-runs next."

patterns-established:
  - "Gate evidence split across runs is committed with the plan; the SUMMARY states which verify clauses failed instead of calling the gate passed"

requirements-completed: []  # Plan requirements are [GAP-02, GAP-04, GAP-05, GAP-06, GAP-07]; left unmarked per orchestrator instruction (phase verifier re-runs next).

coverage:
  - id: D1
    description: "COBOL row holds every required gap-closure phrase (D-21 with the #k qualification and the kept parser-upgrade statement, D-22, D-23/D-24, D-25, D-26 with the mixed-form same-line window, D-27), keeps the D-20 sentence, and drops the old placeholder spelling and the old stability parenthetical; the D-16 writer inventory still passes"
    requirement: GAP-04
    verification:
      - kind: other
        ref: "Task 1 <verify> grep chain on docs/languages.md and internal/graph/finding.go, re-run read-only at close-out (go test part omitted per instruction; docs unchanged since 3abe472d)"
        status: pass
      - kind: unit
        ref: "internal/graph/finding_writers_test.go#TestFindingWriters_Inventory (internal/graph ok in run 1, executed, 32.978s, and in run 3, executed, 21.671s; no skip condition in the file)"
        status: pass
    human_judgment: false
  - id: D2
    description: "COBOL row wording matches the shipped behavior of 05-13, 05-14, and 05-16, including the four Rule 2 additions (old-store note, reserved repository names, LOAD, 'Retrieve a finding')"
    verification: []
    human_judgment: true
    rationale: "Doc accuracy is judged by review; the greps prove phrases exist but cannot judge wording, and no grep covers the four Rule 2 additions."
  - id: D3
    description: "Gap-closure phase gate: GOWORK=off go test -race -timeout 60m ./... on 3abe472d with at least 10 GiB free, log ending exit=0 with no FAIL line"
    verification:
      - kind: integration
        ref: ".planning/phases/05-explicit-unresolved-evidence/05-17-full-suite.log (run 3: 156 ok, 1 FAIL cmd/gortex on its query-log guard, 0 --- FAIL, last line exit=1)"
        status: fail
      - kind: integration
        ref: ".planning/phases/05-explicit-unresolved-evidence/05-17-full-suite.run1.log (run 1, every package executed: 156 ok, 1 FAIL cmd/gortex on its query-log guard, 0 --- FAIL, last line exit=1)"
        status: fail
      - kind: integration
        ref: ".planning/phases/05-explicit-unresolved-evidence/05-17-cmd-gortex-isolated.log (GOWORK=off go test -race -timeout 60m -count=1 ./cmd/gortex/ after run 2: ok 123.082s, exit=0)"
        status: pass
    human_judgment: true
    rationale: "Task 2's verify is not met literally: every full-suite run ended exit=1 because cmd/gortex's end-of-run guard saw outside Gortex writes to the real query log, while every test passed. The user accepted the gate on split evidence (2026-10-26); the verifier decides whether a clean full-suite run is still required (ledger entry #8)."

duration: ~4h   # wall clock from 05-16 close (14:56:46Z) to this close-out; exact 05-17 start not recorded
completed: 2026-10-26
status: complete
---

# Phase 5 Plan 17: Gap-Closure Docs and Phase Gate Summary

**The COBOL row of `docs/languages.md` now documents every gap-closure rule (D-21 to D-27), including the D-22 digest risk and the D-26 mixed-form same-line window. The phase gate did not pass as written: all three full-suite runs ended `exit=1`. Each time, every test passed and the only failure was `cmd/gortex`'s end-of-run guard, tripped by outside Gortex writes to the real query log. The user accepted the gate on split evidence.**

## Performance

- **Duration:** about 4 h wall clock. Most of it went to a disk block, run 1 (about 1400 s), waits for outside Gortex writes, and two `cmd/gortex`-alone runs.
- **Started:** 2026-10-26, after 05-16 closed at 14:56:46Z (exact start not recorded); Task 1 committed at 16:24:30Z.
- **Completed:** 2026-10-26, close-out at about 19:00Z.
- **Tasks:** 2 of 2. Task 2 is complete with a user-approved deviation (gate verify not met literally).
- **Files modified:** 2 modified (`docs/languages.md`, `deferred-items.md`), 3 logs created, plus planning artifacts (`.planning/WINDOWS.md`, this file, STATE, ROADMAP).

## Accomplishments

- **Docs (Task 1):** the COBOL row covers:
  - D-27: placeholder spelling `<repo>::cobol-missing::<type>/<key>`, no file path, repository owned, removed by ID, never written over another node;
  - D-21: name-stable container, the `#k` adjacency qualification, and "parser upgrades" kept per D-02;
  - D-25: the reference form in external evidence;
  - D-26: one finding per occurrence and the mixed-form same-line window;
  - D-22: unkeyed digest, guess confirmation by hashing;
  - D-23/D-24: history timing.
  The D-20 sentence is kept.
- **Gate (Task 2):** three full-suite runs on `3abe472d`. In each, 155 of 156 tested packages passed, and `cmd/gortex`'s tests printed `PASS` but its guard failed the package. Run 1 executed every package. `cmd/gortex` passed alone twice. Accepted on split evidence by the user. Not a pass.
- **Evidence kept:** the run 3 log at the plan path, run 1's full log, and the `cmd/gortex`-alone log are committed in the phase directory.
- **Guard flaw recorded:** `deferred-items.md` (From Plan 05-17) and open ledger entry #8 (`unmet-truth`) in `.planning/WINDOWS.md`.

## Task Commits

1. **Task 1 (tracer): COBOL row documents every gap-closure behavior** - `3abe472d` (docs)
2. **Task 2: gap-closure phase gate** - `a4b19971` (test: the three gate logs and the deferred entry). Status: complete with a user-approved deviation (gate verify not met literally).

**Plan metadata:** the docs(05-17) commits that add this file (with the ledger entry) and update STATE and ROADMAP.

## Gate Evidence

All runs were on HEAD `3abe472d` (Task 1's docs commit). Only `docs/languages.md` changed since 05-16 (`git diff --name-only 0e6221d0..3abe472d -- . ':!.planning'`).

| Run | Started (Z) | Free (GiB) | Result | Cause of the `cmd/gortex` FAIL | Query-log fingerprint |
|---|---|---|---|---|---|
| 1 (fresh build cache, every package executed, about 1400 s) | 16:25:36 | 12 | 156 ok, 1 FAIL (`cmd/gortex`, 264.195s), 40 no test files, 0 `--- FAIL` | The previous 05-17 executor's own timed-out Gortex `change detect` call finished in the daemon and wrote `detect_changes` at 16:29:34Z | `c82740ec4b141473 -> 498fb90c86bd6aa3` |
| `cmd/gortex` alone (`-count=1`) | after run 1 | - | ok, 126.6 s, query log unchanged | none | - |
| 2 (141 s) | 18:13:28 | 174 | 156 ok (155 cached), 1 FAIL (`cmd/gortex`, 129.041s), 40 no test files, 0 `--- FAIL` | pricing-engine-service session `detect_changes` at 18:15:09Z | `4d6643ddcde6cdd9 -> 1e8597102ae321e5` |
| `cmd/gortex` alone (`-count=1`) | after run 2 | 173 | ok, 123.082s, `exit=0`, query log unchanged | none | - |
| 3 (135 s) | 18:40:12 | 172 | 156 ok (155 cached; `internal/graph` executed in 21.671s), 1 FAIL (`cmd/gortex`, 127.271s), 40 no test files, 0 `--- FAIL` | kinetic-sales-quest session `explore` at 18:41:38Z and `search_text` at 18:41:43Z | `1e8597102ae321e5 -> 5bb399cafc5c2c1d` |

- In every run all `cmd/gortex` tests printed `PASS`. Each guard report named only the query log.
- The other guarded paths never changed: `~/.config/gortex/config.yaml`, `~/.claude/settings.json`, `~/.claude/CLAUDE.md`, and `~/.claude/skills`. `~/.config/gortex/servers.toml`, `~/.codex/config.toml`, and `~/.codex/AGENTS.md` stayed absent.
- Run 1's cause is corroborated by the query log itself: its only entry in run 1's window is a `detect_changes` call at 16:29:34.13Z.
- An `ok` from `cmd/gortex` proves its guard passed: `TestMain` turns any fingerprint change into a FAIL.
- **Conclusion (user decision):** run 1 executed every package, and `cmd/gortex` passed alone twice with the guard intact, so every package has passed on `3abe472d`. This is split evidence; no single log shows a green suite.

**Run 1's slowest packages:**

| Package | Time |
|---|---|
| internal/indexer | 1283.728s |
| internal/graph/store_sqlite | 816.090s |
| internal/mcp | 627.122s |
| cmd/gortex | 264.195s (FAIL, guard only) |
| internal/parser/languages | 88.887s |
| internal/resolver | 84.998s |

**Free-space readings** (`df -g /System/Volumes/Data`, Available column):

| When | GiB |
|---|---|
| Run 1, before | 12 (98% used) |
| Run 2, before / after | 174 / 173 |
| `cmd/gortex` alone after run 2 | 173 |
| Run 3, before / after | 172 / 173 |

Every reading is at or above the 10 GiB floor (T-05-75). Execution stopped once on that floor before the gate could run, and resumed once space was available.

### The guard

`cmd/gortex/main_test.go`, `TestMain` (line 30):
- it fingerprints `realUserStatePaths()` (line 70; the query log is line 74) before `m.Run()`;
- `diffUserState` (line 90) re-hashes them afterward;
- any change fails the package.

Run 3's guard output (lines 183-192 of `05-17-full-suite.log`, verbatim; tabs shown as spaces):

```
PASS

cmd/gortex: a test wrote through to the developer's real per-user state:
  /Users/e1001547-mbp-it/.gortex/cache/query-log.jsonl (1e8597102ae321e5 -> 5bb399cafc5c2c1d)

Isolate the test with internal/testenv (Sandbox / SandboxAt / UnifiedHome / SetHome)
instead of t.Setenv("HOME", ...), which does not move os.UserHomeDir on Windows.
If a gortex daemon was running and tracked a repository during this run, that is a
false positive — re-run with the daemon stopped to confirm.
FAIL    github.com/zzet/gortex/cmd/gortex    127.271s
```

### Task 2 `<verify>`, run verbatim on the run 3 log at close-out (log read only)

| Clause | Result |
|---|---|
| log present and non-empty | pass |
| `/System/Volumes/Data` Available at least 10 | pass (172) |
| last line is exactly `exit=0` | **fail** (`exit=1`) |
| no line starting `FAIL` | **fail** (`FAIL github.com/zzet/gortex/cmd/gortex 127.271s` and the bare `FAIL` summary line) |
| no `--- FAIL` line | pass (0) |
| at least one `ok` line | pass (156) |
| **Whole command** | **exit 1: not met** |

Run 1's log gives the same result: Available 12 passes, `exit=1` and the two `FAIL` lines fail.

<details>
<summary>Run 3 log, last 40 lines (tabs shown as spaces)</summary>

```
ok   github.com/zzet/gortex/internal/reconcile (cached)
ok   github.com/zzet/gortex/internal/releases (cached)
ok   github.com/zzet/gortex/internal/resolver (cached)
ok   github.com/zzet/gortex/internal/respbudget (cached)
ok   github.com/zzet/gortex/internal/review (cached)
?    github.com/zzet/gortex/internal/routeguard [no test files]
ok   github.com/zzet/gortex/internal/runtimeactivity (cached)
ok   github.com/zzet/gortex/internal/savings (cached)
ok   github.com/zzet/gortex/internal/search (cached)
ok   github.com/zzet/gortex/internal/search/packstrategy (cached)
ok   github.com/zzet/gortex/internal/search/rerank (cached)
ok   github.com/zzet/gortex/internal/search/trigram (cached)
ok   github.com/zzet/gortex/internal/semantic (cached)
ok   github.com/zzet/gortex/internal/semantic/goanalysis (cached)
ok   github.com/zzet/gortex/internal/semantic/lsp (cached)
ok   github.com/zzet/gortex/internal/semantic/scip (cached)
ok   github.com/zzet/gortex/internal/semantic/tstypes (cached)
ok   github.com/zzet/gortex/internal/server (cached)
ok   github.com/zzet/gortex/internal/server/hub (cached)
ok   github.com/zzet/gortex/internal/serverstack (cached)
ok   github.com/zzet/gortex/internal/skills (cached)
ok   github.com/zzet/gortex/internal/sql (cached)
ok   github.com/zzet/gortex/internal/telemetry (cached)
ok   github.com/zzet/gortex/internal/testenv (cached)
ok   github.com/zzet/gortex/internal/testpath (cached)
?    github.com/zzet/gortex/internal/testutil/gitpromisor [no test files]
ok   github.com/zzet/gortex/internal/testutil/graphdump (cached)
ok   github.com/zzet/gortex/internal/testutil/graphfixture (cached)
ok   github.com/zzet/gortex/internal/thirdparty/fswatcher (cached)
ok   github.com/zzet/gortex/internal/todos (cached)
ok   github.com/zzet/gortex/internal/tokens (cached)
ok   github.com/zzet/gortex/internal/toolref (cached)
?    github.com/zzet/gortex/internal/tui [no test files]
ok   github.com/zzet/gortex/internal/version (cached)
ok   github.com/zzet/gortex/internal/viewmetrics (cached)
ok   github.com/zzet/gortex/internal/wiki (cached)
ok   github.com/zzet/gortex/internal/workspace (cached)
ok   github.com/zzet/gortex/pkg/gortex (cached)
FAIL
exit=1
```

</details>

## Files Created/Modified

- `docs/languages.md`: the COBOL row rewrite (Task 1, `3abe472d`).
- `.planning/phases/05-explicit-unresolved-evidence/05-17-full-suite.log`: run 3 at the plan path; ends `exit=1`.
- `.planning/phases/05-explicit-unresolved-evidence/05-17-full-suite.run1.log`: run 1 (every package executed); ends `exit=1`. Copied from `$TMPDIR/opencode/05-17-full-suite.run1.log`.
- `.planning/phases/05-explicit-unresolved-evidence/05-17-cmd-gortex-isolated.log`: `cmd/gortex` alone after run 2; ends `exit=0`. Copied from `$TMPDIR/opencode/05-17-cmd-gortex-isolated.run2.log`.
- `.planning/phases/05-explicit-unresolved-evidence/deferred-items.md`: From Plan 05-17 (the guard flaw).
- `.planning/WINDOWS.md`: entry #8, `unmet-truth`, open.

## Decisions Made

See key-decisions. The gate disposition is the user's decision of 2026-10-26 ("Accept split evidence now").

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing info] Old-store rebuild note**
- **Found during:** Task 1
- **Issue:** the planned sentences did not tell users what a store indexed before the gap-closure changes keeps.
- **Fix:** the row says such a store keeps its old placeholders (not migrated) and the old parse-finding IDs of every unchanged file. The reason: the extractor version stays `gortex-cobol-grammar/6`, and an unchanged file is not re-extracted. It tells users to rebuild the store (a clean re-index) to get the new IDs. Wording checked against 05-13 and 05-16.
- **Files modified:** `docs/languages.md`
- **Verification:** `rebuild the store` count 0 at `0e6221d0`, 1 at HEAD.
- **Committed in:** `3abe472d`

**2. [Rule 2 - Missing info] Reserved repository names**
- **Found during:** Task 1 (05-16-SUMMARY flagged it for 05-17)
- **Issue:** a repository named like a global ID namespace makes its `<repo>::` IDs, placeholders included, read as global names.
- **Fix:** the row says not to name a repository `unresolved`, `dep`, `module`, `stdlib`, `builtin`, `external`, or `string`.
  - Six of these are `graph.OwnershipGlobalIDNamespaces` (`internal/graph/prefix_diagnostics.go:150`).
  - `unresolved` is the unresolved-target marker (`graph.UnresolvedMarker`, `graph.IsUnresolvedTarget`), as 05-16 recorded.
- **Files modified:** `docs/languages.md`
- **Verification:** `Do not name a repository` count 0 at `0e6221d0`, 1 at HEAD.
- **Committed in:** `3abe472d`

**3. [Rule 2 - Doc accuracy] LOAD in the external-finding list**
- **Found during:** Task 1
- **Issue:** the D-25 sentence names `LOAD` as a distinct form, but the list of references that produce external findings left it out.
- **Fix:** the list reads `CALL`, `LINK`, `XCTL`, `LOAD`, `COPY`, `-INC`, or `EXEC SQL INCLUDE`.
- **Files modified:** `docs/languages.md`
- **Committed in:** `3abe472d`

**4. [Rule 2 - Doc clarity] "Retrieve one" changed to "Retrieve a finding"**
- **Found during:** Task 1
- **Issue:** "Retrieve one" was ambiguous in the rewritten row.
- **Fix:** "Retrieve a finding with `get_symbol` by ID or find them with `search_symbols kind:finding`."
- **Files modified:** `docs/languages.md`
- **Committed in:** `3abe472d`

**5. [Rule 3 - Blocking] Gate re-runs and a widened pre-flight**
- **Found during:** Task 2
- **Issue:** environment failures, not code. The disk floor stopped the gate once, then outside Gortex writes failed `cmd/gortex`'s guard in run 1 and run 2.
- **Fix:** Task 2 ran three times, plus two `cmd/gortex`-alone runs (`-count=1`). The pre-flight was widened to all 8 guarded paths, with content hashes. Run 3's before/after fingerprints show only the query log changed.
- **Files modified:** none in the repository; the helper scripts and run metadata stayed in `$TMPDIR/opencode/`.
- **Committed in:** `a4b19971` (the logs)

**6. [User decision, 2026-10-26] Gate accepted on split evidence**
- **Found during:** Task 2
- **Issue:** Task 2's verify needs a last line of `exit=0` and no `FAIL` line. All three runs ended `exit=1` on `cmd/gortex`'s guard.
- **Decision:** the user chose "Accept split evidence now". Run 1 executed every package, and every package but `cmd/gortex` passed. `cmd/gortex` passed alone twice with the guard intact.
- **Not met literally:** the `exit=0` and no-`FAIL` clauses fail on the run 3 log and on run 1's (table above). The gate is not described as passed anywhere in this plan.
- **Follow-up:** the guard flaw is in `deferred-items.md` (From Plan 05-17), with open ledger entry #8 (`unmet-truth`) in `.planning/WINDOWS.md`.
- **Committed in:** `a4b19971`

**7. [Rule 1 - Bug, close-out] Ledger rewrite dropped `milestone` fields**
- **Found during:** close-out, when appending entry #8 with `gsd-tools windows append`
- **Issue:** the tool rewrote `.planning/WINDOWS.md` and dropped `"milestone": "v1.0"` from entries 2-7.
- **Fix:** restored the six fields by edit and gave entry #8 the same field. The JSON block parses (8 entries, 4 open). The diff against HEAD removes only the three frontmatter counters.
- **Files modified:** `.planning/WINDOWS.md`
- **Committed in:** the SUMMARY commit

---

**Total deviations:** 7: four Rule 2 doc additions (Task 1), one Rule 3 environment workaround (Task 2), one user decision (Task 2), and one Rule 1 close-out fix.
**Impact on plan:** no source code changed. The gate's success criterion ("the full race suite is green ... run once") is not met as written; it rests on split evidence the user accepted.

## Issues Encountered

- **Disk floor:** execution stopped once on the 10 GiB check (05-16 had ended with 6.5 GiB free). Run 1 started with 12 GiB available; runs 2 and 3 had 172 to 174 GiB.
- **Outside Gortex writes:** three runs, three different writers (Gate Evidence). The guard cannot tell a test's write from another session's. Its own false-positive hint mentions only a daemon that tracked a repository, but any tool call is enough.
- **Evidence gaps:**
  - No `05-17-gate/` folder existed; run 1's log was flat in `$TMPDIR/opencode/`.
  - The `cmd/gortex`-alone run after run 1 left no log file. Its result (ok, 126.6 s, query log unchanged) comes from the orchestrator's record. The committed isolated log is the run after run 2.
  - Run 1's start time and its about-1400 s duration also come from the orchestrator's record; run 1's log has no timing lines.
- **Close-out:** no `go test` run and no Gortex calls. `GOWORK=off go build ./...` passed (14 s).

## Resume Note

05-17 resumed after a disk block and after outside Gortex writes:
1. Task 1 committed at 16:24:30Z (`3abe472d`).
2. The gate then failed three times on `cmd/gortex`'s query-log guard (runs at 16:25:36Z, 18:13:28Z, and 18:40:12Z). `cmd/gortex` passed alone after runs 1 and 2.
3. After run 3, the user chose "Accept split evidence now" (2026-10-26).
4. This close-out committed the evidence, logged the guard flaw, and wrote this SUMMARY without re-running the suite.

## Tracer Gate

- Auto mode is off (`_auto_chain_active` and `auto_advance` are false). `human_verify_mode` is end-of-phase, mode is yolo, and the plan is autonomous, so the tracer did not halt the plan (05-13, 05-15, and 05-16 precedent).
- At close-out, the tracer's grep checks pass. `TestFindingWriters_Inventory` passed inside `internal/graph` in run 1 and run 3.

## Plan-Level Verification

| Check | Result |
|---|---|
| Task 1 `<verify>` | grep chain passes (re-run at close-out); the inventory test passed in the suite runs (go test not re-run per instruction) |
| Task 2 `<verify>` | **not met** (exit 1: last line `exit=1`, two `FAIL` lines); accepted by user decision |
| Phase-gate log ends `exit=0` | **not met** (`exit=1`) |
| `GOWORK=off go build ./...` | ok (14 s, at close-out) |
| `git diff --name-only 0e6221d0..HEAD` | `docs/languages.md`, the three logs, `deferred-items.md`, plus planning artifacts (this SUMMARY, WINDOWS, STATE, ROADMAP); no Rule 1 source fixes |

## Threat Flags

None. T-05-75 holds: every run's free-space reading was at least 10 GiB, recorded in its log. T-05-71 and T-05-74 are covered by the Task 1 greps (`unkeyed`, `by hashing it`, `mixed-form same-line`).

## Known Stubs

None.

## User Setup Required

None. Developer stores indexed before the gap closure keep old placeholders and old parse-finding IDs until a clean re-index; the docs say so.

## Next Phase Readiness

- All 17 Phase 5 plans now have SUMMARYs. The phase verifier re-runs next.
- GAP-02, GAP-04, GAP-05, GAP-06, and GAP-07 stay unmarked for the verifier.
- **If the verifier re-runs the full suite:** stop all Gortex use on the machine first (other agent sessions, in-flight daemon calls). Alternatively, treat a `cmd/gortex` FAIL whose guard report names only `query-log.jsonl`, with every test `PASS` and 0 `--- FAIL`, as this known flaw, and confirm by running `cmd/gortex` alone with `-count=1`.
- A clean full-suite run closes ledger entry #8 (`gsd-tools windows fixed 8`).
- Open deferred items: 05-05 (gofmt of a C# resolver test), 05-16 (shared DB2/IDMS resources keep the WR-05 shape), 05-17 (`cmd/gortex` query-log guard).

---
*Phase: 05-explicit-unresolved-evidence*
*Completed: 2026-10-26*

## Self-Check: PASSED

- FOUND: `05-17-full-suite.log`, `05-17-full-suite.run1.log`, `05-17-cmd-gortex-isolated.log`, `deferred-items.md`, `docs/languages.md`
- FOUND commits: `3abe472d`, `a4b19971`
- The committed logs match their sources byte for byte (`cmp`).
- The `.planning/config.json` and `.planning/state.json` hashes are unchanged.
- The gate is recorded as not met, accepted on split evidence; nothing here claims it passed.
