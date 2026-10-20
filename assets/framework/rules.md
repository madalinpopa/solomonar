# Framework rules

Framework version: 0.1.0

These rules apply to every managed task. Orchestrators load this file together with `AGENTS.md`. Report
format, result values, finding fields, and the PR Verification section: `docs/framework/report.md`. Role
procedures: `docs/framework/agents/<role>.md`. Skill procedures: `docs/framework/skills/<skill>/SKILL.md`.

## 1. Scope and authority

- Authority order, highest first:
  1. The user's explicit decision applicable to the subject, recorded with subject and limits.
  2. Approved global specs and mandatory rules.
  3. The approved work spec. A deviation requires an explicit decision and updates to every affected document.
  4. Agent recommendations. Apply only when compatible with 1–3 and backed by a concrete benefit.
     Preferences stay suggestions.
- Two approved requirements that contradict each other require a user decision (§3).
- Flag every conflict between a global spec and the work spec. Never silently drop an acceptance criterion.
- Code and prototypes are evidence of existing behavior. They must not change approved requirements.
- Only **Agreed** spec rules are approved requirements. **Observed** rules, prototype libraries, and
  simulations are evidence and approve nothing.

## 2. Orchestrator responsibilities and delegation

- The orchestrator must:
  - exclusively claim its task and maintain the task's state record through `solomonar task` commands (§7);
  - prepare context and delegate implementation and every review; it must not perform them itself;
  - consolidate findings (§6) and manage user approvals (§3);
  - create the PR, and merge when §8 holds and it has the right to integrate.
- Multiple orchestrators may run in parallel. Each owns different tasks.
- Delegation packet: objective, work spec path, relevant references, area of responsibility. Reviewers
  additionally receive the exact revision (full SHA) or diff and the check results. Pass full conversation
  history only when necessary.
- The implementer and reviewers report only to the orchestrator. Only the orchestrator starts the next stage.
- Route results: `READY_FOR_REVIEW` → record its committed SHA as `revision` (`task move`), start triage (§6);
  `CHANGES_REQUIRED` → consolidation (§6); `NEEDS_DECISION` → §3; `INCOMPLETE` → resolve the gap, rerun.
- The implementer must receive one consolidated list of corrections, never concurrent or conflicting instructions.

## 3. Approvals and decisions

- Any new architecture decision or dependency change requires user approval, including when a reviewer
  proposes it as a fix. Applying an approved decision within its recorded scope needs no new approval.
- The agent that identifies the need may investigate and must prepare: question, recommended option,
  alternatives, impact, blocked work. It must not apply the choice before approval.
- Orchestrator procedure:
  1. Assign a task-scoped decision ID (`W-001/D-1`).
  2. Record a `decision` blocker (`task block`) and set `next_action` to the user (`task move --next`, §7). Write the
     decision entry in `docs/decisions.md` in the task worktree with status `pending`; while a review set runs
     on the current revision, write it only after every running review has reported (§6 Candidate stability).
  3. Present question, recommendation, alternatives, impact.
  4. If no independent authorized work can continue, set status `blocked` (`task move --status`).
  5. Save a checkpoint (§10) and wait. No response is not approval. Never repeat the question; never retry automatically.
- While waiting, affected and dependent work must wait. Independent authorized work may continue.
- On approval: set the entry `approved` with the selected option, update affected specs, clear the blocker
  (`task unblock`), resume dependent work on the approval event.
- On rejection: keep the entry `pending` and list the rejected option with its reason in it. Re-delegate the
  role that identified the need when the revised option needs investigation; otherwise present a listed
  alternative. Present the revised option under the same decision ID.
- Commit each entry change, with the spec updates it requires, in the task worktree:
  `docs(<ID>): record W-001/D-n`.
- Decision entry fields (English): ID, question, status (`pending` | `approved` | `superseded`), selected
  option (once approved), rejected options with reasons, scope, rationale, date, references. Only an
  `approved` entry authorizes work.
- Reuse a decision only within its scope. A superseding decision must name the decision it replaces, set
  that entry's status to `superseded`, and update the affected canonical rules.

## 4. Starting a task

The user starts tasks one at a time. Started tasks run in parallel. Work selected from a prototype enters at step 1.

1. Read the registry (`solomonar task show`, §7) and the task brief or existing spec. Check ownership and
   dependencies. A complete work spec is not required to reserve.
2. Run `solomonar task register [<ID>] --title <t> --owner <o>`. It claims the `planned` ID or allocates the
   next `W-NNN` and records `reserved` / `setup`, branch `feature/<ID>`, and `next_action` (§7).
3. The successful `task register` is the registration barrier. Confirm it to the user. Only then may the next
   task start. A push is not required.
4. Create branch `feature/<ID>` and its worktree at `../<repo>-worktrees/<ID>` (relative to the main checkout)
   from the intended current main revision. The path is not recorded; locate it with `git worktree list` by
   branch `feature/<ID>`. If a prerequisite merged remotely, update the baseline first and verify it contains
   the prerequisite's recorded `merge_commit`.
5. Set status `running`, stage `specification` (`task move`). Create or refine the work spec in the worktree
   with the `work-spec` skill. Commit it separately. Record `spec` (`task move --spec`).
6. When the readiness gate owned by `work-spec` passes, apply the work spec's user-confirmed "Spec changes"
   with the `spec` skill (in-task mode), commit them as `docs(<ID>): apply spec changes`, and cite the assigned
   rule IDs in the work spec (`spec(<ID>): refine task specification`). Then set stage `implementation` and delegate.

- If `task register` fails, do not create the worktree.
- If worktree setup fails, keep the registration: blocker kind `environment` (`task block`), status `blocked`,
  stage `setup`.
- Each task has its own branch and worktree. Agents must modify only the assigned worktree.
- When services are shared, assign separate ports and test data. A worktree does not isolate databases or
  external services.
- A task that needs another task's code must wait until that code is merged into main (blocker kind `dependency`).
- Same-file changes on different branches are reconciled at integration.

## 5. Task flow

Main flow: implement → triage (§6) → specialist reviews in parallel on the same revision → consolidate → fix
loop → tier 2: adversarial review once specialist blockers are closed → fix loop → PR, required checks,
integration with main → closure commits (§9) → merge → `task done` (`done` / `complete`).

| Trigger | Return to | Resume when |
|---|---|---|
| Decision or requirement conflict | User, via orchestrator | Decision recorded and documents updated |
| Specialist defect | Implementer | Fix committed; relevant reviews pass on the new revision |
| Adversarial defect | Implementer | Fix committed; recheck sequence (§6 Adversarial ordering) passes on the new revision |
| Integration or CI failure after PR | Orchestrator diagnoses; implementer if code changes | Relevant checks pass on the current candidate |
| No progress | Orchestrator reassesses, then user | User selects an option |

- Loop stop: after two consecutive rounds that leave the same blocker unchanged or reintroduce the same
  issue, stop looping, analyze the cause, present the blocker and options to the user. Record it as a
  `decision` blocker with a `W-001/D-n` entry; the user's choice follows §3. Reaching the threshold is never
  acceptance.
- A transient infrastructure error may be diagnosed and retried without arbitrary code changes.
- A fix after the PR is opened must update the same PR.

## 6. Reviews

### Triage and gate

Before launching reviews, record each candidate's risk tier and which specialties apply, with reasons.

| Tier | Candidate | Agent reviews |
|---|---|---|
| 0 | Documentation only, no behavior or requirement change | None; CI only |
| 1 | Normal code | The required specialists per the table below |
| 2 | Persistence, authentication or authorization, concurrency, external integrations, data migration, or any approved architecture or dependency decision in the task | Specialists plus adversarial |

| Specialty | Role | Required when |
|---|---|---|
| Security | `security-reviewer` | Authorization, data isolation, external input, files, secrets, trust boundaries, relevant dependency or configuration changes |
| Go and correctness | `go-reviewer` | Go code changes |
| Spec compliance | `spec-reviewer` | A requirement is implemented or changed |
| Adversarial | `adversarial-reviewer` | Tier 2 (see Adversarial ordering) |

- Functional correctness of non-Go code stays an explicit concern: name the reviewer responsible for it in
  the delegation packet.
- Mark an inapplicable specialty "not required" with a reason. Never fabricate a `PASS`.
- Use a different model for the adversarial reviewer when the runtime allows; record the model in the
  Verification section (`report.md` §8).
- Every required reviewer must return a valid `PASS` on the current revision before the stage advances,
  unless its earlier verdict is reused under Candidate stability.
- A missing result or an important unrun check makes the stage `INCOMPLETE`, not `PASS`.

### Candidate stability

- Code and specs must stay unchanged while a review set runs.
- Reviewers must not modify tracked files. Reproductions use temporary locations and isolated test data.
- Run interfering checks sequentially.
- A changed candidate gets a new revision and an explicit assessment of affected checks. Reuse unaffected
  evidence only with its original revision and a recorded reason. An earlier `PASS` never covers new code.
- Updating the candidate from main follows §8: CI always reruns; agent reviews rerun only on its conditions.

### Consolidation

- Deduplicate into stable task-scoped finding IDs (`W-001/F-1`). Clarify contradictions between reviewers.
- Keep per finding: revision checked, rule or acceptance criterion, impact, evidence, resolution.
- Rejecting a finding requires evidence-backed justification.

| Finding | Handling |
|---|---|
| Demonstrated defect or mandatory-criterion violation | Fix before completion |
| Style or organization without a violated rule | Non-blocking suggestion |
| Disagreement on whether a defect exists | Request reproduction, test, or precise analysis; record the conclusion |
| Contradictory requirements or unclear product choice | User decision (§3) |
| Fix needs new architecture or dependency | Approval first (§3) |
| Unrun check or missing reviewer result | Stage `INCOMPLETE` |

- Severity sets urgency by impact (unauthorized access, data loss, blocked essential flow). Blocking depends
  on the violated rule and acceptance conditions.
- Specialty labels and reviewer vote counts never override valid evidence. The same arbitration applies to
  adversarial findings.

### Adversarial ordering

- Tier 2 only: start adversarial review after specialist blockers are closed.
- After an adversarial fix: implementer → relevant specialists recheck → adversarial recheck, all on the new revision.

## 7. Shared state (registry)

### Location and writes

- Canonical registry: `state.json` on `refs/heads/workflow-state`: `{"schema_version": 1, "tasks": {"<ID>": <record>}}`.
  Read it with `solomonar task show [<ID>]` or `git show refs/heads/workflow-state:state.json`.
- Only `solomonar task` commands write it. Agents never edit, commit, or check out the registry. A runtime hook
  that denies direct writes to `state.json` is a Phase 2 adapter concern.

| Command | Who | Effect |
|---|---|---|
| `task plan --title <t>` | user | Creates a `planned` record without owner |
| `task register [<ID>] --title <t> --owner <o>` | orchestrator | Claims the `planned` `<ID>` or allocates the next `W-NNN`; sets `reserved` / `setup`, branch `feature/<ID>`, `next_action` |
| `task move <ID> [--status] [--stage] [--revision] [--spec] [--agent <id>:<role>]... [--pr] [--next <actor>:<action>]` | owner | Validated transition and field update; repeated `--agent` replaces `active_agents`, `--agent none` clears it |
| `task block <ID> --kind <k> --summary <s> [--ref <r>]` / `task unblock <ID> <blocker-id>` | owner | Adds or clears a blocker |
| `task done <ID> --merge <sha> --spec-ref <sha:path> --summary <s>` | owner | Verifies the merge commit is reachable from main and `spec_ref` resolves and names the last `spec(<ID>)` commit merged, then writes the compact record (`done` / `complete`) |
| `task handoff <ID> --owner <o>` | new owner, on user instruction | Ownership change (§10) |
| `task show [<ID>]` | anyone | Read |

- `planned` records are created by the user with `task plan`; they have no owner. This is the only state write
  not made by an orchestrator. Otherwise each orchestrator writes only its own task's record.
- The CLI validates schema, ownership, and allowed transitions, and writes with Git plumbing compare-and-swap:
  `hash-object` → `mktree` → `commit-tree -p <old>` → `update-ref refs/heads/workflow-state <new> <old>`,
  retrying on mismatch. No checkout, no lock.
- Each successful command is one commit: `state(<ID>): <status>/<stage>, <summary>`. Git history is the
  transition history; state.json holds no event log. A failed command writes nothing: report it. Consumers read
  the last successful commit.
- The CLI pushes `workflow-state` when a remote is configured and reports publication failures; the local ref
  stays canonical. Never force-push it; never merge it into main. Multiple independently writable clones are
  out of scope.

### Status

| Status | Meaning |
|---|---|
| `planned` | Known work; no owner yet |
| `reserved` | Registration committed; worktree may not exist yet |
| `running` | Owner is actively advancing the stage |
| `blocked` | A blocker stops all authorized work on the task |
| `paused` | Stopped on instruction; resumes on instruction |
| `cancelled` | Stopped without merge; work preserved; never `done` |
| `done` | Feature merge into main confirmed; compact record written; final |

### Stages and transitions

`setup` → `specification` → `implementation` → `specialized_review` → `adversarial_review` → `integration`
(PR, required CI, compatibility with main, closure commits, merge) → `complete`.

- Stage reflects the current activity. Preserve the stage while `blocked` or `paused`.
- Tier 0 goes from `implementation` to `integration`; tier 1 skips `adversarial_review` (§6).
- `planned` or `reserved` → `running` → `done`. `done` comes only with stage `complete`, set by `task done`.
- `reserved` or `running` → `blocked` or `paused` → back to the prior status at the preserved stage.
- `cancelled` resumes only on explicit user instruction after inspecting preserved work.
- A `done` task stays `done`. A regression or revert is a new task linked via `related_task`.
- Corrections return to `implementation` without a new task ID.

### Fields

| Phase | Required fields |
|---|---|
| Planned | `title`, `status`, `updated_at` |
| Reserved | adds `owner`, `branch`, `stage`, `next_action` |
| Before implementation | adds `spec` (worktree-relative path) |
| Review | adds `revision`: latest reported committed candidate, full SHA |

- Always present once reserved: `active_agents` `[{id, role}]` (last set by `task move --agent`, not proof of
  liveness), `blockers`, `pr` once opened.
- Optional: `depends_on`, `related_task`, `session_ref`, evidence references. Omit empty optional fields.
- Blocker: `{id, kind, summary, ref?}`, kind ∈ `decision` | `dependency` | `check` | `environment`. Blocker
  IDs stay stable within the task.
- `next_action`: `{actor, action}`, e.g. `{"actor":"user","action":"Decide W-001/D-1: session storage"}`.
- Use full SHAs and RFC 3339 UTC timestamps. Never store logs, secrets, or machine-specific paths in state.
- Resolve active `spec` paths against the feature worktree (§4 step 4) or its recorded `revision`.

Active record (illustrative):
```json
"W-001": {
  "title": "Login", "status": "blocked", "stage": "implementation",
  "owner": "orchestrator-1", "branch": "feature/W-001",
  "spec": "docs/work/W-001-login.md", "revision": "<FULL_SHA>",
  "active_agents": [{"id": "impl-1", "role": "implementer"}],
  "blockers": [{"id": "B-1", "kind": "decision", "summary": "Session storage", "ref": "W-001/D-1"}],
  "next_action": {"actor": "user", "action": "Decide W-001/D-1: session storage"},
  "updated_at": "2026-01-15T10:00:00Z"
}
```

Compact record (illustrative):
```json
"W-001": {
  "title": "Login", "status": "done", "stage": "complete", "summary": "Password login with sessions",
  "spec_ref": "<FULL_SHA>:docs/work/W-001-login.md", "merge_commit": "<FULL_SHA>",
  "pr": "<PR_URL>", "updated_at": "2026-01-20T09:00:00Z"
}
```

## 8. PR and merge

- Merge only when all hold: mandatory criteria met; affected specs reflect approved decisions; required
  reviews complete and valid for the current candidate; no blocking defects or open decisions; mandatory
  checks and integration with main succeeded; closure commits made (§9); repository rules satisfied.
- No extra human merge confirmation by default. Approvals required by the repository stay mandatory. An
  informal approval message never replaces CI or repository rules.
- Record every verdict with the revision checked in the PR Verification section.
- Staleness: before merge the feature branch contains current main (merged by the implementer,
  `implementer.md` §7). CI always reruns. Agent reviews rerun only if updating from main had conflicts, or if
  main's changes since the branch base touch files in the candidate's diff: with `<base>` =
  `git merge-base <reviewed revision> main`, `git diff --name-only <base>..<reviewed revision>` and
  `git diff --name-only <base>..main` share a file. A rerun moves the task back to `specialized_review` and
  through the review stages to `integration` again. Otherwise reuse the verdicts with that reason (§6).
- A spec rule ID that collides with one already on main after merging main: renumber the task's rule with the
  `spec` skill, update the work spec, and treat it as a changed candidate (§6).
- Merge only with a merge commit (§11), never squash or rebase, so the final spec commit survives in main's
  history. Merge one task at a time. Run `task done` only after the merge is confirmed (§9).

## 9. Closure (orchestrator rule, not a skill)

Before merge, after every required gate passes on the candidate:
1. Canonical documents must reflect approved changes. Never rewrite requirements to match implementation omissions.
2. The final spec commit must match the spec used for final validation. If the work spec is unchanged since its
   last `spec(<ID>)` commit inside the reviewed revision, that commit is the final spec commit; no new revision.
   Only when content changed, commit that file alone as `spec(<ID>): finalize task specification` and treat it
   as a changed candidate (§6 Candidate stability). The final spec commit and the work spec path form `spec_ref`;
   write it on the PR Verification line `Final specification` (`report.md` §8).
3. One last commit deletes `docs/work/<ID>-<slug>.md`: `docs(<ID>): remove completed work spec`. It needs no
   agent review (docs-only deletion of the task spec); CI reruns. A later fix goes on top of it; packets then
   name `Work spec: <spec_ref>`, read with `git show "<spec_ref>"`. A fix that changes the work spec first
   reverts the deletion commit, follows the spec flow, and repeats steps 2–3.

At confirmed merge:
- Run `solomonar task done <ID> --merge <sha> --spec-ref <sha:path> --summary <s>`, with `spec_ref` from the
  PR's `Final specification` line. It verifies the merge commit is reachable from main and `git show
  "<spec_ref>"` resolves to the last `spec(<ID>)` commit merged, then writes the compact record (§7) with
  `done` / `complete`. `done` is final at merge.
- If `task done` fails, report the cause and rerun it once fixed. Never revert the merge.
- Delete a worktree only as a separate deliberate step, after confirming it holds no uncommitted or unmerged work.
- History lookup: read the compact record first. Run `git show "<spec_ref>"` only for a concrete question.

## 10. Recovery

Before acting on any event, inspect actual Git state, worktree, dirty files, checks, decisions, and PR,
then reconcile the record with what exists (`task move`). Never duplicate worktrees or PRs. Preserve
uncommitted changes.

| Event | Orchestrator behavior |
|---|---|
| Pause | Stop delegating; settle or stop active work; set status `paused` with the latest committed `revision` and a `next_action` naming outstanding work |
| Unexpected session end | Nothing automatic; the next session follows Resume |
| Resume | Inspect and reconcile; continue the recorded stage; reuse the existing worktree and PR; if the PR is merged, verify the merge commit is in main's history and run `task done` (§9) with `spec_ref` from the PR, without reimplementing |
| Ownership handoff | Only on explicit user instruction; stop or confirm the previous execution stopped; `task handoff` before any further write; the previous owner stops writing |
| Cancel | Stop execution; set status `cancelled` with a `next_action` naming the reason and remaining work; keep branch, worktree, and PR untouched |
| Regression or revert after merge | Keep the merged task `done`; register a new task linked via `related_task` |

- Checkpoint = the task's state record + its worktree + Git history. A brief note of uncommitted work may go
  in `next_action`. Approvals persist within their recorded scope.
- No heartbeats, no time-based ownership takeover, no automatic destructive cleanup.
- A report lost on resume without a durable reference counts as unverified. Repeat the verification.

## 11. Bootstrap and runtime assumptions

- Before the first managed task, the following must exist: a local Git baseline and `refs/heads/workflow-state`
  with an empty registry, created by `solomonar init`. No other checkout is needed for state.
- v0.1 requires a Git repository and Git worktrees; Jujutsu and other VCS front ends are out of scope.
- The repository must allow merge commits, and the orchestrator merges only with a merge commit: squash and
  rebase merges break `spec_ref`. `solomonar doctor` checks the repository settings (Phase 2).
- Integration requires a remote and PR path. Missing access is a concrete blocker (kind `environment`).
- Record real run, build, test, and lint commands in `docs/README.md` once known. Record unknown commands as
  "not established"; never guess them.
- Before changing existing code, record the baseline and known failures. Report pre-existing failures
  separately from regressions.
- The runtime must support: explicit working directory, selective file loading, command execution, isolated
  delegation with task and revision, result collection, pausing for a user decision.
- Give reviewers read-only access where supported. Ownership rules are behavioral contracts; a worktree is
  not a security boundary.
