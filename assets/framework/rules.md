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
  - exclusively claim its task and maintain the task's state record (§7);
  - prepare context and delegate implementation and every review; it must not perform them itself;
  - consolidate findings (§6) and manage user approvals (§3);
  - create the PR, and merge when §8 holds and it has the right to integrate.
- Multiple orchestrators may run in parallel. Each owns different tasks.
- Delegation packet: objective, work spec path, relevant references, area of responsibility. Reviewers
  additionally receive the exact revision (full SHA) or diff and the check results. Pass full conversation
  history only when necessary.
- The implementer and reviewers report only to the orchestrator. Only the orchestrator starts the next stage.
- Route results: `READY_FOR_REVIEW` → record its committed SHA as `revision`, start triage (§6);
  `CHANGES_REQUIRED` → consolidation (§6); `NEEDS_DECISION` → §3; `INCOMPLETE` → resolve the gap, rerun.
- The implementer must receive one consolidated list of corrections, never concurrent or conflicting instructions.

## 3. Approvals and decisions

- Any new architecture decision or dependency change requires user approval, including when a reviewer
  proposes it as a fix. Applying an approved decision within its recorded scope needs no new approval.
- The agent that identifies the need may investigate and must prepare: question, recommended option,
  alternatives, impact, blocked work. It must not apply the choice before approval.
- Orchestrator procedure:
  1. Assign a task-scoped decision ID (`W-001/D-1`).
  2. Add a blocker of kind `decision`, set `next_action` to the user, commit the record (§7). Write the
     decision entry in `docs/decisions.md` in the task worktree with status `pending`.
  3. Present question, recommendation, alternatives, impact.
  4. If no independent authorized work can continue, set status `blocked`.
  5. Save a checkpoint (§10) and wait. No response is not approval. Never repeat the question; never retry automatically.
- While waiting, affected and dependent work must wait. Independent authorized work may continue.
- On approval: set the entry `approved` with the selected option, update affected specs, clear the blocker,
  resume dependent work on the approval event.
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

1. Read the committed registry (§7) and the task brief or existing spec. Check ownership and dependencies.
   A complete work spec is not required to reserve.
2. Write the record (§7 write protocol): task ID, title, owner, status `reserved`, stage `setup`, planned
   branch `feature/<ID>`, planned worktree, `next_action`. Commit `state.json` in the coordination checkout.
3. The successful local commit is the registration barrier. Confirm it to the user. Only then may the next
   task start. A push is not required.
4. Create branch `feature/<ID>` and its worktree from the intended current main revision. If a prerequisite
   merged remotely, update the baseline first and verify it contains the prerequisite's recorded `merge_commit`.
5. Set status `running`, stage `specification`. Create or refine the work spec in the worktree with the
   `work-spec` skill. Commit it separately. Record `spec`.
6. When the readiness gate owned by `work-spec` passes, set stage `implementation` and delegate.

- If registration or its commit fails, do not create the worktree.
- If worktree setup fails, keep the registration: status `blocked`, stage `setup`, blocker kind `environment`.
- Each task has its own branch and worktree. Agents must modify only the assigned worktree.
- When services are shared, assign separate ports and test data. A worktree does not isolate databases or
  external services.
- A task that needs another task's code must wait until that code is merged into main (blocker kind `dependency`).
- Same-file changes on different branches are reconciled at integration.

## 5. Task flow

Main flow: implement → specialist reviews in parallel on the same revision → consolidate → fix loop →
adversarial review once specialist blockers are closed → fix loop → PR, required checks, integration with
main → merge → `done` → compaction (§9).

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

Decide which specialties apply to each candidate and record why.

| Specialty | Role | Required when |
|---|---|---|
| Security | `security-reviewer` | Authorization, data isolation, external input, files, secrets, trust boundaries, relevant dependency or configuration changes |
| Go and correctness | `go-reviewer` | Go code changes |
| Spec compliance | `spec-reviewer` | A requirement is implemented or changed |
| Adversarial | `adversarial-reviewer` | Every task (see Adversarial ordering) |

- Functional correctness of non-Go code stays an explicit concern: name the reviewer responsible for it in
  the delegation packet.
- Mark an inapplicable specialty "not required" with a reason. Never fabricate a `PASS`.
- Every required reviewer must return a valid `PASS` on the current revision before the stage advances,
  unless its earlier verdict is reused under Candidate stability.
- A missing result or an important unrun check makes the stage `INCOMPLETE`, not `PASS`.

### Candidate stability

- Code and specs must stay unchanged while a review set runs.
- Reviewers must not modify tracked files. Reproductions use temporary locations and isolated test data.
- Run interfering checks sequentially.
- A changed candidate gets a new revision and an explicit assessment of affected checks. Reuse unaffected
  evidence only with its original revision and a recorded reason. An earlier `PASS` never covers new code.

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

- Start adversarial review only after specialist blockers are closed.
- After an adversarial fix: implementer → relevant specialists recheck → adversarial recheck, all on the new revision.

## 7. Shared state (registry)

### Location and writes

- Canonical registry: `state.json` on branch `workflow-state`, read as `refs/heads/workflow-state:state.json`:
  `{"schema_version": 1, "tasks": {"<ID>": <record>}}`.
- Write it only in one persistent coordination checkout (e.g. `<repo>-state/`). Never merge it into main.
  Never force-push it. Feature-local copies are not coordination sources.
- Only orchestrators write, each only its own task's record, except during an explicit ownership handoff (§10).
- Write protocol: acquire a short local serialization lock → read current → validate owner and transition →
  update → commit → release. Never hold the lock during implementation, review, or a user wait.
- On write failure: report it and resolve any uncommitted registry change before another writer proceeds.
  Consumers read the last successful commit.
- Commit every change to stage, status, agent assignment, blockers, decisions, or next action as
  `state(<ID>): <status>/<stage> — <summary>`. Git history is the transition history; state.json holds no event log.
- Push to a configured remote when available and surface publication failures. The local branch stays
  canonical. Multiple independently writable clones are out of scope.

### Status

| Status | Meaning |
|---|---|
| `planned` | Known work; no owner yet |
| `reserved` | Registration committed; worktree may not exist yet |
| `running` | Owner is actively advancing the stage |
| `blocked` | A blocker stops all authorized work on the task |
| `paused` | Stopped on instruction; resumes on instruction |
| `cancelled` | Stopped without merge; work preserved; never `done` |
| `done` | Feature merge into main confirmed; compaction may be pending |

### Stages and transitions

`setup` → `specification` → `implementation` → `specialized_review` → `adversarial_review` → `integration`
(PR, required CI, compatibility with main, merge) → `compaction` → `complete`.

- Stage reflects the current activity. Preserve the stage while `blocked` or `paused`.
- `planned` or `reserved` → `running` → `done`.
- `reserved` or `running` → `blocked` or `paused` → back to the prior status at the preserved stage.
- `cancelled` resumes only on explicit user instruction after inspecting preserved work.
- A `done` task stays `done`. A regression or revert is a new task linked via `related_task`.
- Corrections return to `implementation` without a new task ID.

### Fields

| Phase | Required fields |
|---|---|
| Planned | `title`, `status`, `updated_at` |
| Reserved | adds `owner`, `branch`, `worktree`, `stage`, `next_action` |
| Before implementation | adds `spec` (worktree-relative path) |
| Review | adds `revision`: latest reported committed candidate, full SHA |

- Always present once reserved: `active_agents` `[{id, role}]` (last reported, not proof of liveness),
  `blockers`, `pr` once opened.
- Optional: `depends_on`, `related_task`, `session_ref`, `cleanup_pr`, evidence references. Omit empty
  optional fields.
- Blocker: `{id, kind, summary, ref?}`, kind ∈ `decision` | `dependency` | `check` | `environment`. Blocker
  IDs stay stable within the task.
- `next_action`: `{actor, action}`, e.g. `{"actor":"user","action":"Decide W-001/D-1: session storage"}`.
- Use full SHAs and RFC 3339 UTC timestamps. Never store logs or secrets in state.
- Resolve active `spec` paths against the feature worktree or its recorded `revision`, never the state checkout.

Active record (illustrative):
```json
"W-001": {
  "title": "Login", "status": "blocked", "stage": "implementation",
  "owner": "orchestrator-1", "branch": "feature/W-001", "worktree": "../<repo>-W-001",
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
  checks and integration with main succeeded; repository rules satisfied.
- No extra human merge confirmation by default. Approvals required by the repository stay mandatory. An
  informal approval message never replaces CI or repository rules.
- Record every verdict with the revision checked in the PR Verification section.
- If main changes relevantly or the candidate changes, recheck integration and affected reviews.
- Use a merge commit, never squash, so the final spec commit survives in main's history. Merge one task at
  a time. Set `done` only after the merge is confirmed.

## 9. Closure and compaction (orchestrator rule, not a skill)

Before merge:
- Canonical documents must reflect approved changes. Never rewrite requirements to match implementation omissions.
- The final spec commit must match the spec used for final validation. If the work spec is unchanged since its
  last `spec(<ID>)` commit inside the reviewed revision, that commit is the final spec commit; no new revision.
  Only when content changed, commit that file alone as `spec(<ID>): finalize task specification` and treat it
  as a changed candidate (§6 Candidate stability).

After confirmed merge:
1. Set status `done`, stage `compaction`, and the real `merge_commit`.
2. Set `spec_ref` to `<final spec commit SHA>:<spec path>`. Verify `git show "<spec_ref>"` opens the final
   document and the commit is in main's history.
3. Open a cleanup PR on branch `cleanup/<ID>` from current main: docs only; remove `docs/work/<ID>-*.md`, fix
   references. Same task ID, no new work spec, commit `docs(W-001): compact completed task`. No specialist or
   adversarial review; required repository checks and rules apply. Record `cleanup_pr`.
4. After the cleanup PR merges, set stage `complete` and replace the record with the compact record (§7).

- Cleanup failure: keep `done` / `compaction`, add a blocker (kind `check` or `environment`, by cause) and
  a `next_action`. Never revert the feature; never block dependents. Keep the full spec until cleanup succeeds.
- Delete a worktree only as a separate deliberate step, after confirming it holds no uncommitted or unmerged work.
- History lookup: read the compact record first. Run `git show "<spec_ref>"` only for a concrete question.

## 10. Recovery

Before acting on any event, inspect actual Git state, worktree, dirty files, checks, decisions, and PR,
then reconcile the record with what exists. Never duplicate worktrees or PRs. Preserve uncommitted changes.

| Event | Orchestrator behavior |
|---|---|
| Pause | Stop delegating; settle or stop active work; commit status `paused` with the latest committed `revision` and a `next_action` naming outstanding work |
| Unexpected session end | Nothing automatic; the next session follows Resume |
| Resume | Inspect and reconcile; continue the recorded stage; reuse the existing worktree and PR; if the PR is merged, set `done` / `compaction` with the real `merge_commit` and continue §9 without reimplementing |
| Ownership handoff | Only on explicit user instruction; stop or confirm the previous execution stopped; commit the new `owner` before any further write; the previous owner stops writing |
| Cancel | Stop execution; commit status `cancelled` with a `next_action` naming the reason and remaining work; keep branch, worktree, and PR untouched |
| Regression or revert after merge | Keep the merged task `done`; register a new task linked via `related_task` |

- If only compaction is paused, cancelled, or blocked, status stays `done`.
- Checkpoint = the task's state record + its worktree + Git history. A brief note of uncommitted work may go
  in `next_action`. Approvals persist within their recorded scope.
- No heartbeats, no time-based ownership takeover, no automatic destructive cleanup.
- A report lost on resume without a durable reference counts as unverified. Repeat the verification.

## 11. Bootstrap and runtime assumptions

- Before the first managed task, the following must exist: a local Git baseline, branch `workflow-state`
  with an empty registry, and the coordination checkout.
- v0.1 requires a Git repository and Git worktrees; Jujutsu and other VCS front ends are out of scope.
- Integration requires a remote and PR path. Missing access is a concrete blocker (kind `environment`).
- Record real run, build, test, and lint commands in `docs/README.md` once known. Record unknown commands as
  "not established"; never guess them.
- Before changing existing code, record the baseline and known failures. Report pre-existing failures
  separately from regressions.
- The runtime must support: explicit working directory, selective file loading, command execution, isolated
  delegation with task and revision, result collection, pausing for a user decision.
- Give reviewers read-only access where supported. Ownership rules are behavioral contracts; a worktree is
  not a security boundary.
