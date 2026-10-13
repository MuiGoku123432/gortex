---
phase: 04
review: 04-REVIEW.md
titles: json
findings:
  - id: CR-04
    severity: critical
    disposition: open
    title: "Per-entry walk gate admits symlinks whose target is outside every tracked root, so generate_skill copies arbitrary file bytes into the repository"
  - id: WR-06
    severity: warning
    disposition: open
    title: "Declared roots over-prune when a declared repository fails to register, and nothing re-admits the files"
  - id: WR-07
    severity: warning
    disposition: open
    title: "Enclosing-reconcile failures are only logged: track and untrack report success while the outer repository keeps duplicates or misses files"
  - id: WR-08
    severity: warning
    disposition: open
    title: "Scoped move_symbol and inline_symbol now silently leave out-of-scope callers unrewritten and report success"
  - id: WR-09
    severity: warning
    disposition: open
    title: "View attribution overrides a more specific registered owner in the physical-owner gate, and generate_skill's 04-22 deviation copies the same rule"
  - id: WR-10
    severity: warning
    disposition: open
    title: "preview_edit and simulate_chain still read through ungated `resolveOverlayAbsPath`, which also accepts any absolute path (04-19 residual)"
  - id: IN-07
    severity: info
    disposition: open
    title: "`DeclarePendingRoots` was inserted between `indexMultiRepo`'s doc line and its function"
  - id: IN-08
    severity: info
    disposition: open
    title: "The source-channel inventory test is name-keyed and covers only `Grep*` and `Walk*` selector calls"
  - id: IN-09
    severity: info
    disposition: open
    title: "Explore's artifact lane still caps the text probe before the scope filter when `RepoAllow` is nil"
  - id: IN-10
    severity: info
    disposition: open
    title: "`indexMultiRepo` registers a nested repository without evicting an already-registered enclosing repository outside the batch"
  - id: WR-03
    severity: warning
    disposition: open
    title: "Nested-root containment is lexical only; it now also covers declared roots and the enclosing reconcile"
  - id: WR-05
    severity: warning
    disposition: open
    title: "`find_declaration`'s regex branch still caps each repository before the physical-owner filter (narrowed)"
  - id: IN-01
    severity: info
    disposition: open
    title: "Scope is re-resolved and `EvalSymlinks` runs once per path, and now once per walk entry"
  - id: IN-02
    severity: info
    disposition: open
    title: "View-routed `search_text` matches are anchored to the canonical root, not the view's checkout"
  - id: IN-03
    severity: info
    disposition: open
    title: "Graph-backed reads require the linked worktree's prefix to be in scope when the worktree fallback re-roots"
  - id: IN-04
    severity: info
    disposition: open
    title: "Test gaps (partially resolved)"
  - id: IN-05
    severity: info
    disposition: open
    title: "The \"caller must not hold mi.mu\" invariant on walk and watcher admission is documented but not enforced"
  - id: IN-06
    severity: info
    disposition: open
    title: "Carried-forward accepted risk: the `rename_symbol` refusal reveals nested physical owners"
  - id: CR-01
    severity: critical
    disposition: fixed
    title: "The source gate assigns a repository from the lexical path, so a symlink or nested checkout bypasses the scope"
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "COBOL reconcile passes are not serialised, and the warm-restart pass ignores `deferResolve`"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "The v22 sidecar sweep deletes every empty-prefix sidecar row on every v21 store"
  - id: WR-04
    severity: warning
    disposition: skipped
    title: "The `rename_symbol` refusal reveals that out-of-scope references exist"
  - id: CR-02
    severity: critical
    disposition: fixed
    title: "`search_text` and `find_declaration` return source lines from every tracked repository with no scope"
  - id: CR-03
    severity: critical
    disposition: fixed
    title: "Untracking a repository deletes other repositories' bound COBOL CALL/COPY edges"
open: 18
total: 24
recorded: 2026-10-13T22:41:45.082Z
---

# Phase 04: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-04 | critical | open | - |
| WR-06 | warning | open | - |
| WR-07 | warning | open | - |
| WR-08 | warning | open | - |
| WR-09 | warning | open | - |
| WR-10 | warning | open | - |
| IN-07 | info | open | - |
| IN-08 | info | open | - |
| IN-09 | info | open | - |
| IN-10 | info | open | - |
| WR-03 | warning | open | - |
| WR-05 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |
| IN-06 | info | open | - |
| CR-01 | critical | fixed | 04-REVIEW-FIX.md (not in the current review) |
| WR-01 | warning | fixed | 04-REVIEW-FIX.md (not in the current review) |
| WR-02 | warning | fixed | 04-REVIEW-FIX.md (not in the current review) |
| WR-04 | warning | skipped | 04-REVIEW-FIX.md (not in the current review) |
| CR-02 | critical | fixed | 04-REVIEW-FIX.md (not in the current review) |
| CR-03 | critical | fixed | 04-REVIEW-FIX.md (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
