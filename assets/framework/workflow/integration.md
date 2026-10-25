# Integration and closure

Framework version: {{ .Version }}

Orchestrator integration only. Load `reports/pr-verification.md`; delegation additionally loads `workflow/implementation.md`.

## PR and merge

- Merge only when all hold: mandatory criteria met; affected specs reflect approved decisions; required
  reviews complete and valid for the current candidate; no blocking defects or open decisions; mandatory
  checks and integration with main succeeded; closure commits made (`rules.md` section 9); repository rules satisfied.
- No extra human merge confirmation by default. Approvals required by the repository stay mandatory. An
  informal approval message never replaces CI or repository rules.
- Record every verdict with the revision checked in the PR Verification section.
- Staleness: before merge the feature branch contains current main (merged by the implementer,
  `agents/implementer.md` section 7). CI always reruns. Compare main's changes with the candidate, its referenced contracts,
  dependencies, and requirements. With `<base>` = `git merge-base <reviewed-revision> <main>`, overlapping paths
  from `git diff --name-only <base>..<reviewed-revision>` and `git diff --name-only <base>..<main>`, merge
  conflicts, or a changed relevant contract/dependency require the affected reviews to rerun. Disjoint paths
  alone are not a reuse reason. Reuse a verdict only with its original SHA and a concrete reason why main's
  changes do not affect its scope; if impact is unclear, rerun that review. A rerun moves tier 1 or 2 back to
  `specialized_review` and through the required stages to `integration`; tier 0 keeps its CI-only gate.
- A spec rule ID that collides with one already on main after merging main: renumber the task's rule with the
  `spec` skill, update the work spec, and treat it as a changed candidate (`rules.md` section 6).
- Merge only with a merge commit (`rules.md` section 11), never squash or rebase, so the final spec commit survives in main's
  history. Merge one task at a time. Run `task done` only after the merge is confirmed (`rules.md` section 9).

## Closure

Before merge, after every required gate passes on the candidate:

1. Canonical documents must reflect approved changes. Never rewrite requirements to match implementation omissions.
2. The final spec commit must match the spec used for final validation. If the work spec is unchanged since its
   last `spec(<ID>)` commit inside the reviewed revision, that commit is the final spec commit; no new revision.
   Only when content changed, commit that file alone as `spec(<ID>): finalize task specification` and treat it
   as a changed candidate (`rules.md` section 6 Candidate stability). The final spec commit and the work spec path form `spec_ref`;
   write it on the PR Verification line `Final specification` (`report.md` section 8).
3. One last commit deletes `docs/work/<ID>-<slug>.md`: `docs(<ID>): remove completed work spec`. It needs no
   agent review (docs-only deletion of the task spec); CI reruns. A later fix goes on top of it; packets then
   name `Work spec: <spec_ref>`, read with `git show "<spec_ref>"`. A fix that changes the work spec first
   reverts the deletion commit (`docs(<ID>): restore work spec`), follows the spec flow, and repeats steps 2–3.

At confirmed merge:
- Run `task done <ID> --as <owner> --merge <sha> --spec-ref <sha:path> --summary <s>`, with `spec_ref` from the
  PR's `Final specification` line. It verifies the merge commit is reachable from main and `git show
  "<spec_ref>"` resolves to the last `spec(<ID>)` commit merged, then writes the compact record
  (`registry.md` section 4) with `done` / `complete`. `done` is final at merge.
- If `task done` fails, report the cause and rerun it once fixed. Never revert the merge.
- Delete a worktree only as a separate deliberate step, after confirming it holds no uncommitted or unmerged work.
- History lookup: read the compact record first. Run `git show "<spec_ref>"` only for a concrete question.

## Stage procedure

### `integration`
- **Entry:** every required review gate passes for the candidate (tier 0: none). **Delegate:**
  `implementer` for code changes or for merging main into `feature/<ID>`; otherwise none.
- **Do:** push; open/update the same PR with Verification (`report.md` section 8); monitor required checks.
  Delegate integration with current main; assess review reuse and spec rule-ID collisions per `rules.md`
  section 8. Rerun affected or uncertain reviews through `specialized_review` (tiers 1/2); tier 0 stays CI-only.
  Once gates pass, perform closure per `rules.md` section 9 and record `spec_ref` on the PR's
  `Final specification` line. For new commits, record the new revision, assess verdict validity, and rerun
  required checks. Merge only when `rules.md` sections 8 and 11 hold; confirm the merge, then run `task done`
  with `spec_ref` from the PR and report completion.
- **State:** update `--revision`, `--pr`, and `--next` with `task move`; record `check`/`environment` blockers;
  `task done` → `done` / `complete`.
- **Exit:** `task done` succeeded. `complete` has no further action.
- **Failure:** diagnose; retry transient errors; code changes → `implementation`, same PR (`rules.md` section 5); no remote → `rules.md` section 11;
  `task done` fails → report and rerun once fixed, never revert (`rules.md` section 9).
