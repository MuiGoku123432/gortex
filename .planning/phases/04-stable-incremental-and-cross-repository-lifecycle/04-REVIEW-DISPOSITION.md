---
phase: 04
review: 04-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: open
    title: "The nested-root walk seed comes from the wrong repository list: project-defined repos are still double-indexed, and inactive entries make files disappear"
  - id: WR-01
    severity: warning
    disposition: open
    title: "The seed never refreshes, so after untracking the nested repository the outer repository keeps excluding its files"
  - id: WR-02
    severity: warning
    disposition: open
    title: "Tracking a nested repository later freezes the outer repository's stale duplicates, because the watcher now refuses their update and delete events"
  - id: WR-03
    severity: warning
    disposition: open
    title: "Nested-root containment is lexical only, so a symlinked or alternate spelling of the nested root is not pruned"
  - id: WR-04
    severity: warning
    disposition: open
    title: "The explore artifact lane emits matched source lines without any source gate, so the physical-owner fix is not applied everywhere"
  - id: WR-05
    severity: warning
    disposition: open
    title: "`find_declaration`'s regex branch still caps before filtering, and the new physical-owner filter makes false \"no use sites matched\" answers more likely (prior WR-03)"
  - id: IN-01
    severity: info
    disposition: open
    title: "Scope is re-resolved and `EvalSymlinks` runs once per unique match path"
  - id: IN-02
    severity: info
    disposition: open
    title: "View-routed `search_text` matches are anchored to the canonical root, not the view's checkout"
  - id: IN-03
    severity: info
    disposition: open
    title: "Graph-backed reads now require the linked worktree's prefix to be in scope when the worktree fallback re-roots"
  - id: IN-04
    severity: info
    disposition: open
    title: "Test gaps let the CR-01, WR-01, WR-02 and WR-03 defects pass"
  - id: IN-05
    severity: info
    disposition: open
    title: "The \"caller must not hold mi.mu\" invariant on walk and watcher admission is documented but not enforced"
  - id: IN-06
    severity: info
    disposition: open
    title: "Carried-forward accepted risk: the `rename_symbol` refusal now also reveals nested physical owners (prior WR-04)"
  - id: CR-02
    severity: critical
    disposition: fixed
    title: "`search_text` and `find_declaration` return source lines from every tracked repository with no scope"
  - id: CR-03
    severity: critical
    disposition: fixed
    title: "Untracking a repository deletes other repositories' bound COBOL CALL/COPY edges"
open: 12
total: 14
recorded: 2026-10-13T19:25:05.920Z
---

# Phase 04: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | open | - |
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| WR-03 | warning | open | - |
| WR-04 | warning | open | - |
| WR-05 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |
| IN-06 | info | open | - |
| CR-02 | critical | fixed | 04-REVIEW-FIX.md (not in the current review) |
| CR-03 | critical | fixed | 04-REVIEW-FIX.md (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
