# Framework rules

Framework version: {{ .Version }}

Orchestrators load this shared contract with `AGENTS.md`, then only the current stage/trigger procedure below.
Project decisions outside a task load section 3's procedure without taking the orchestrator role.
Load `report.md` before delegation or consuming reports; registry detail is conditional (section 7).
An unloaded procedure must be read before its governed action. References are not implicit loads.

“Main” is `.solomonar/manifest.json` → `project.main_branch`, or `main` without a manifest.
Substitute it for `<main>`; prefix `task` commands with `solomonar`. Framework paths are relative to `docs/solomonar/`.

## 1. Scope and authority

- Authority order, highest first:
  1. The user's explicit decision applicable to the subject, recorded with subject and limits.
  2. Approved global specs and mandatory rules, including the `must` rules (`<name>/C-n`) of installed
     extensions whose installing decision is approved.
  3. The approved work spec. A deviation requires an explicit decision and updates to every affected document.
  4. Agent recommendations. Apply only when compatible with 1–3 and backed by a concrete benefit.
     Preferences stay suggestions.
- Two approved requirements that contradict each other require a user decision (section 3).
- Flag every conflict between a global spec and the work spec. Never silently drop an acceptance criterion.
- Apply `AGENTS.md` rule 7 before changing requirements or production behavior. The approved prototype defines
  represented behavior; **Agreed** spec rules express it for implementation. Pause conflicting work rather than
  choosing either artifact silently. **Observed** rules and code do not approve behavior; prototype libraries
  and simulation mechanisms do not approve production architecture or dependencies.

## 2. Orchestrator responsibilities and delegation

- The orchestrator must:
  - exclusively claim its task and maintain the task's state record through `solomonar task` commands (section 7);
  - prepare context and delegate implementation and every review; it must not perform them itself;
  - consolidate findings (section 6) and manage user approvals (section 3);
  - create the PR, and merge when section 8 holds and it has the right to integrate.
- Multiple orchestrators may run in parallel. Each owns different tasks.
- Use the packets in `agents/orchestrator.md` section 4. Reviews pin an exact full SHA and a diff base;
  check evidence names its revision. Pass full conversation history only when necessary.
- The implementer and reviewers report only to the orchestrator. Only the orchestrator starts the next stage.
- Route results: `READY_FOR_REVIEW` → record its committed SHA as `revision` (`task move`), start triage (section 6);
  `CHANGES_REQUIRED` → consolidation (section 6); `NEEDS_DECISION` → section 3; `INCOMPLETE` → resolve the gap, rerun.
- The implementer must receive one consolidated list of corrections, never concurrent or conflicting instructions.

## 3. Approvals and decisions

Load [workflow/decisions.md](workflow/decisions.md) before raising/recording decisions, conflicts, or project open-question answers.
Reuse scoped approvals; never apply a new architecture/dependency choice before approval.

## 4. Starting a task

Load [workflow/setup.md](workflow/setup.md) for registration/worktree setup, then
[workflow/specification.md](workflow/specification.md) before specification or plan approval.
Register before creating a worktree or confirming the next task may start; do not advance without its required gate.

## 5. Task flow

Main flow: implement → triage (section 6) → specialist reviews in parallel on the same revision → consolidate → fix
loop → tier 2: adversarial review once specialist blockers are closed → fix loop → PR, required checks,
integration with main → closure commits (section 9) → merge → `task done` (`done` / `complete`).

| Trigger | Return to | Resume when |
|---|---|---|
| Decision or requirement conflict | User, via orchestrator | Decision recorded and documents updated |
| Specialist defect | Implementer | Fix committed; relevant reviews pass on the new revision |
| Adversarial defect | Implementer | Fix committed; recheck sequence (section 6 Adversarial ordering) passes on the new revision |
| Integration or CI failure after PR | Orchestrator diagnoses; implementer if code changes | Relevant checks pass on the current candidate |
| No progress | Orchestrator reassesses, then user | User selects an option |

- Loop stop: after two consecutive rounds that leave the same blocker unchanged or reintroduce the same
  issue, stop looping, analyze the cause, present the blocker and options to the user. Record it as a
  `decision` blocker with a `W-001/D-n` entry; the user's choice follows section 3. Reaching the threshold is never
  acceptance.
- A transient infrastructure error may be diagnosed and retried without arbitrary code changes.
- A fix after the PR is opened must update the same PR.

### Checkpoints

Plan format and approval: `skills/work-spec/SKILL.md` section 7. Before delegating implementation, load
[workflow/implementation.md](workflow/implementation.md#checkpoints). Its checkpoints govern execution progress.

## 6. Reviews

Before triage, review, consolidation, or verdict reuse, load [workflow/reviews.md](workflow/reviews.md).
Keep the candidate unchanged while reviews run; missing required reviews/checks prevent advancement.

## 7. Shared state (registry)

Storage, the command table, CLI checks, statuses, and record fields: `docs/solomonar/registry.md`. Load it only
when a command, a CLI rejection, or a record field needs it.

- The canonical registry is `refs/heads/workflow-state`. Read it with `solomonar task show [<ID>]`; `task status
  <ID>` adds Git and forge facts, drift, and the allowed next commands.
- Only `solomonar task` commands write it. Agents never edit, commit, or check out the registry, force-push
  it, or merge it into main. These boundaries concern the registry and its Git ref, not ordinary product files
  named `state.json`; do not assume runtime hooks enforce them (section 11).
- The CLI is a guard and an oracle, never a driver: it validates and records what the orchestrator decided and
  reports what Git shows. It never launches agents, calls models, or stores review verdicts. State holds
  decisions; Git holds what happened; `task status` reconciles them.
- The user runs `task plan` (a `planned` record without owner) and `task migrate`. Otherwise each orchestrator
  writes only its own task's record and passes `--as <owner>` on every `move`, `block`, `unblock`, and `done`
  call. Combine related changes in one validated update.
- A rejected update leaves the registry unchanged and names the failed check and the corrective action: report
  the diagnostic, apply the fix, and preserve the last successful state.
- Never store logs, secrets, absolute paths, or facts Git can derive (HEAD, dirty files, checkpoints, PR or
  check state) in state; `task status` reads them. Do not invent schema fields for evidence.

### Stages and transitions

`setup` → `specification` → `implementation` → `specialized_review` → `adversarial_review` → `integration`
(PR, required CI, compatibility with main, closure commits, merge) → `complete`.

- Stage reflects the current activity. Preserve the stage while `blocked` or `paused`.
- Tier 0 goes from `implementation` to `integration`; tier 1 skips `adversarial_review` (section 6). The CLI enforces
  this from the recorded `tier`; no stage after `implementation` without one.
- A task is `reserved` exactly while at stage `setup`: the move out of `setup` sets `running`, and no stage
  returns to `setup`. The prior status of a `blocked`, `paused`, or resumed `cancelled` task is therefore
  `reserved` at `setup` and `running` at any later stage.
- `cancelled` resumes only on explicit user instruction after inspecting preserved work, using
  `task move <ID> --as <owner> --resume-cancelled --status <prior> --next "<actor>:<action>"`.
- A `done` task stays `done`. A regression or revert is a new task linked via `related_task`.
- Corrections return to `implementation` without a new task ID, as new plan steps (section 5 Checkpoints).

## 8. PR and merge

Before integration or merge, load [workflow/integration.md](workflow/integration.md).
Merge only after all required gates pass; merge commits are mandatory. No extra human merge confirmation by default.

## 9. Closure (orchestrator rule, not a skill)

Closure procedure: [workflow/integration.md](workflow/integration.md#closure). Load it before finalization/deletion/merge.
Done is final only after confirmed merge and successful `task done`; cleanup stays separate.

## 10. Recovery

Load [workflow/recovery.md](workflow/recovery.md) for resumed executions, handoff, pause/cancel, or drift.
Preserve dirty work; an old record proves neither liveness nor takeover/resumption authority. Never duplicate worktrees or PRs.

## 11. Bootstrap and runtime assumptions

- Before the first managed task, the following must exist: a local Git baseline and `refs/heads/workflow-state`
  with an empty registry, created by `solomonar init`. No other checkout is needed for state.
- Solomonar requires a Git repository and Git worktrees; Jujutsu and other VCS front ends are out of scope.
- The repository must allow merge commits, and the orchestrator merges only with a merge commit: squash and
  rebase merges break `spec_ref`. Verify repository merge settings before integration.
- Use `solomonar help` to check command availability. `doctor` and guard hooks are planned
  capabilities; do not run missing commands or assume hooks enforce policy. Use `doctor` for project
  health checks only when the installed CLI provides it.
- Integration requires a remote and PR path. Missing access is a concrete blocker (kind `environment`).
- Record real run, build, test, and lint commands in `docs/README.md` once known. Record unknown commands as
  "not established"; never guess them.
- Before changing existing code, record the baseline and known failures. Report pre-existing failures
  separately from regressions.
- The runtime must support: explicit working directory, selective file loading, command execution, isolated
  delegation with task and revision, result collection, pausing for a user decision.
- Give reviewers read-only access where supported. Ownership rules are behavioral contracts; a worktree is
  not a security boundary.
