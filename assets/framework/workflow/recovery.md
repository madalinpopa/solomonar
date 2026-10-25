# Task recovery

Framework version: {{ .Version }}

Orchestrator procedure for resumed executions, handoff, pause/cancel, or observed drift. Load `rules.md` first;
`task` means `solomonar task`. After inspection, load the procedure for the stage being continued.

Before acting on any event, inspect actual state: `task status <ID>` (`rules.md` section 7) reports Git state, worktree, dirty
files, merge state, and drift; check decisions yourself, and PR and checks when it reports them `unknown`.
Then reconcile the record with what exists (`task move`). Never duplicate worktrees or PRs. Preserve
uncommitted changes.

A new session continues a task from its record, approved work spec, checkpoint commits, and preserved working
files, without the previous conversation:

1. Take the task only through `task handoff` on the user's instruction; it clears the confirmation. Context
   compaction within the same execution is not a handoff.
2. Confirm the previous execution stopped, or isolate its access, before replacement work in the same worktree.
   A recorded owner or a closed conversation is not proof that execution stopped.
3. Read `task status <ID>`: the `plan` section lists committed, closed, and remaining steps and any discrepancy;
   load detailed diffs and code only where needed. Inspect dirty files and keep justified partial work.
4. Past `setup`, run `task confirm <ID> --as <owner>`. Confirmation proves neither plan approval nor review success.
5. Delegate only the remaining steps, with the partial work to inspect and reuse. With every step committed,
   proceed to triage and review; repeat verification whose evidence cannot be recovered.

A discrepancy (a commit matching a step only by subject, a wrong trailer, an empty or duplicate checkpoint) needs
investigation, never a reset, a duplicate worktree, or an assumption that the step succeeded. Commit amendments
to the plan rather than rewriting checkpoint history.

| Event | Orchestrator behavior |
|---|---|
| Pause | Stop delegating; settle or stop active work; `task move <ID> --as <owner> --status paused --next "<actor>:<outstanding work>"`; preserve stage and recorded revision |
| Unexpected session end | Nothing automatic; the next session follows Resume |
| Resume | Inspect and reconcile; restore the prior status at the preserved stage (`rules.md` section 7), subject to remaining blockers; reuse the worktree and PR. If merged, reconcile to `running` / `integration` and run `task done` (`rules.md` section 9) with `spec_ref` from the PR, without reimplementing |
| Ownership handoff | Only on explicit user instruction; stop or confirm the previous execution stopped; `task handoff` before any further write, then inspect and `task confirm`; the previous owner stops writing |
| Cancel | Stop execution; `task move <ID> --as <owner> --status cancelled --next "<actor>:<reason and remaining work>"`; keep branch, worktree, and PR untouched |
| Regression or revert after merge | Keep the merged task `done`; register a new task linked via `related_task` |

- Recovery evidence = the task's state record + its approved work spec + its checkpoint commits + its worktree.
  A brief note of uncommitted work may go in `next_action`; ignored files are never preserved automatically.
  Approvals persist within their recorded scope.
- Set a new `revision` only for a committed candidate when the current or resulting stage is `implementation`
  or `integration`; pausing or cancelling alone does not require one. Earlier stages keep their existing fields.
- No heartbeats, no time-based ownership takeover, no automatic destructive cleanup.
- A report lost on resume without a durable reference counts as unverified. Repeat the verification.
