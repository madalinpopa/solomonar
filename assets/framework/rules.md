# Framework rules

Framework version: 0.1.6

These rules apply to every managed task. Orchestrators load this file together with `AGENTS.md`. Report
format, result values, finding fields, and the PR Verification section: `docs/solomonar/report.md`. Role
procedures: `docs/solomonar/agents/<role>.md`. Skill procedures: `docs/solomonar/skills/<skill>/SKILL.md`.

Here, "main" means the integration branch named by `.solomonar/manifest.json` → `project.main_branch`
(`main` without a manifest). Substitute it for `<main>` in commands; prefix `task` commands with `solomonar`.

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
- Code and prototypes are evidence of existing behavior. They must not change approved requirements.
- Only **Agreed** spec rules are approved requirements. **Observed** rules, prototype libraries, and
  simulations are evidence and approve nothing.

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

- Any new architecture decision or dependency change requires user approval, including when a reviewer
  proposes it as a fix. Applying an approved decision within its recorded scope needs no new approval.
- The agent that identifies the need may investigate and must prepare: question, recommended option,
  alternatives, impact, blocked work. It must not apply the choice before approval.
- Orchestrator procedure:
  1. Assign a task-scoped decision ID (`W-001/D-1`).
  2. Record the blocker and next action together:
     `task block <ID> --as <owner> --kind decision --summary <s> --ref <decision-ID> --next "user:<action>"`.
     Add `--status blocked` when no independent authorized work can continue. Write the `pending` entry in
     `docs/decisions.md` in the task worktree; wait for every running review to report before changing the
     candidate (section 6 Candidate stability).
  3. Present question, recommendation, alternatives, impact.
  4. Keep the recovery evidence current (section 10) and wait. No response is not approval. Never repeat the question or retry automatically.
- While waiting, affected and dependent work must wait. Independent authorized work may continue.
- On approval: set the entry `approved` with the selected option, update affected specs via `spec` in-task
  mode, and commit before clearing the blocker. Use
  `task unblock <ID> <blocker-ID> --as <owner> --next "<actor>:<action>"`; when removing the last blocker from
  a `blocked` task, also pass `--status reserved` at `setup`, otherwise `--status running`. Keep it blocked
  while other blockers stop all work; paused or cancelled tasks still require explicit resumption (section 10).
- On rejection: keep the entry `pending` and list the rejected option with its reason in it. Re-delegate the
  role that identified the need when the revised option needs investigation; otherwise present a listed
  alternative. Present the revised option under the same decision ID.
- Commit each entry change, with the spec updates it requires, in the task worktree:
  `docs(<ID>): record W-001/D-n`.
- Entry format: `## <ID>: <question>`, then named bullets `Status`, `Selected option` (once approved),
  `Rejected options` (`none` when empty), `Scope`, `Rationale`, `Date`, and `References`. Keep the labels exact
  so the CLI can read the status (`pending` | `approved` | `superseded`); only `approved` authorizes work.
  References identify any superseded decision and its replacement.
- Reuse a decision only within its scope. A superseding decision must name the decision it replaces, set
  that entry's status to `superseded`, and update the affected canonical rules.
- Stack extensions follow the same approval rule. When available, the user's `solomonar extension add|remove`
  records `P/D-n` for each affected group's complete selection, using the entry format above; the orchestrator
  instead passes `--decision <approved-ID>`. The CLI updates installed extensions' decision references and
  supersession links together. Drafts approve nothing and are excluded from loading, rendering, and triage.
- Check availability with `solomonar help` (section 11). If extension management is unavailable, report it;
  never substitute manual edits to the manifest or generated adapters.

## 4. Starting a task

The user starts tasks one at a time. Started tasks run in parallel. Work selected from a prototype enters at step 1.

1. Read the registry (`solomonar task show`, section 7) and the task brief or existing spec. Check ownership and
   dependencies. A complete work spec is not required to reserve.
2. Run `solomonar task register [<ID>] --title <t> --owner <o>`. It claims the `planned` ID or allocates the
   next `W-NNN` and records `reserved` / `setup`, branch `feature/<ID>`, and `next_action` (section 7).
3. The successful `task register` is the registration barrier. Confirm it to the user. Only then may the next
   task start. A push is not required.
4. Verify the intended current main baseline. If a prerequisite merged remotely, update it first and verify
   it contains the prerequisite's recorded `merge_commit`. From the main checkout, run
   `git worktree add -b feature/<ID> ../<repo>-worktrees/<ID> <main>`.
5. Run `task move <ID> --as <owner> --status running --stage specification`. It records the task's `base`
   (the commit it branched from), its worktree path relative to the main checkout, and the confirmation
   (section 7). Create or refine the work spec in the worktree with `work-spec`. Commit it separately; record it
   with `task move <ID> --as <owner> --spec <path>`.
6. When the readiness gate owned by `work-spec` passes, apply the work spec's user-confirmed "Spec changes"
   with the `spec` skill (in-task mode), commit them as `docs(<ID>): apply spec changes`, and cite the assigned
   rule IDs in the work spec (`spec(<ID>): refine task specification`).
7. Present the work spec's implementation plan (section 5 Checkpoints) to the user. On explicit approval of that
   exact plan, set `Status: Approved` below the title and commit it as `spec(<ID>): approve implementation plan`;
   reuse an earlier explicit approval that covers the exact plan instead of asking again. Then set stage
   `implementation` and delegate. The CLI rejects `implementation` without the approved plan committed.

- If `task register` fails, do not create the worktree.
- If worktree setup fails, keep the registration: blocker kind `environment` (`task block`), status `blocked`,
  stage `setup`.
- Each task has its own branch and worktree. Agents must modify only the assigned worktree.
- When services are shared, assign separate ports and test data. A worktree does not isolate databases or
  external services.
- A task that needs another task's code must wait until that code is merged into main (blocker kind `dependency`).
- Same-file changes on different branches are reconciled at integration.

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

- The work spec's `## Implementation plan` lists steps `- S-<n>: <type>(<ID>): <summary>`, each the intended
  commit subject with enough detail to delimit the work. Its `Status` line is `In review` until the user approves
  the plan, then `Approved`. Only the orchestrator edits the plan; it is the approved plan, not an execution log.
- The implementer commits each completed step as its checkpoint: the planned subject, a short body (progress,
  checks and results, observations needed to continue), and the trailer `Step: <ID>/S-<n>`. Progress lives in
  these commits only; never duplicate it in the work spec, the registry, or another file.
- Step IDs are never reused. Changing an approved step's scope gives it a new ID and returns `Status` to
  `In review` for a new approval. Review corrections within the approved scope are appended as new steps without
  a new approval. A step that becomes unnecessary or is satisfied by verification alone gets an indented
  `Outcome: dropped — <reason>` or `Outcome: verified — <reason>`; never create an empty commit for it.
- A checkpoint preserves only committed content. A commit is not proof of correctness: `task status` reconciles
  steps with commits (`plan` section) and reviews still verify the change.
- Leaving `implementation` for review or integration needs the clean branch tip as candidate, every step
  committed or closed by an outcome, and no reconciliation discrepancy; the CLI enforces it. Review the cumulative
  change on that one candidate after the complete implementation or correction batch, never per checkpoint.

## 6. Reviews

### Triage and gate

Before launching reviews, record each candidate's risk tier and which specialties apply, with reasons. The tier
goes into state with `task move --tier` (section 7); specialties and reasons stay in the session, then the PR
Verification section (`report.md` section 8).

| Tier | Candidate | Agent reviews |
|---|---|---|
| 0 | Documentation only, no behavior or requirement change | None; CI only |
| 1 | Changes outside tiers 0 and 2, including documentation that changes behavior or requirements | The required specialists per the table below |
| 2 | Persistence, authentication or authorization, concurrency, external integrations, data migration, or any approved architecture or dependency decision in the task | Specialists plus adversarial |

| Priority | Specialty | Role | Focus | Required when |
|---|---|---|---|---|
| 1 | Security | `security-reviewer` | Bugs and security defects | Every code change; documentation affecting security requirements or contracts (tier 1, 2) |
| 2 | Correctness | `correctness-reviewer` | Business logic | Every code change; documentation affecting business rules or contracts (tier 1, 2) |
| 3 | Spec | `spec-reviewer` | Work spec, canonical documents, `must` rules of applicable extensions | Every tier 1 or 2 change |
| - | Extension agent | the installed agent | Its specialty | Its `applies_when` matches the change |
| - | Adversarial | `adversarial-reviewer` | The integrated feature | Tier 2 (see Adversarial ordering) |

- The core reviewers are language-agnostic. Name in each packet the installed extensions whose `applies_when`
  or `paths` match the change; their `must` rules (`<name>/C-n`) are approved global rules (section 1).
- All required specialists run in parallel on the same revision. When the runtime cannot run them in
  parallel, or their checks interfere, run them in priority order.
- Mark an inapplicable specialty "not required" with a reason. Never fabricate a `PASS`.
- Use a different model for the adversarial reviewer when the runtime allows; record the model in the
  Verification section (`report.md` section 8).
- Every required reviewer must return a valid `PASS` on the current revision before the stage advances,
  unless its earlier verdict is reused under Candidate stability.
- A missing result or an important unrun check makes the stage `INCOMPLETE`, not `PASS`.

### Candidate stability

- Code and specs must stay unchanged while a review set runs.
- Reviewers follow `reviewer.md`'s read-only contract; run interfering checks sequentially.
- A changed candidate gets a new revision and an explicit assessment of affected checks. Reuse unaffected
  evidence only with its original revision and a recorded reason. An earlier `PASS` never covers new code.
- Updating the candidate from main follows section 8: CI always reruns; agent reviews rerun only on its conditions.

### Consolidation

- Deduplicate into stable task-scoped finding IDs (`W-001/F-1`). Clarify contradictions between reviewers.
- Finding fields and correction-list format: `report.md` sections 5 and 7.
- Rejecting a finding requires evidence-backed justification.
- Order the correction list by review priority (Security, Correctness, Spec, then extension agents); within one
  reviewer, by severity. A merged duplicate takes the highest-priority reviewer's position.
- Priority orders the list, not the choice between conflicting corrections. Resolve conflicts against evidence
  and the authority order (section 1) before delegation; conflicting requirements need a user decision (section 3).
- Classify findings and verification gaps per `report.md` sections 3–5: confirmed defects and mandatory
  violations block; suggestions do not; missing required verification makes the review incomplete.
- Disputed defects require reproduction, a test, or precise analysis; record the conclusion. Missing or
  contradictory policy, unclear product choices, and architecture/dependency changes follow section 3.
- Severity sets urgency by impact; blocking follows violated rules and acceptance conditions. Specialty labels
  and vote counts never override evidence. Apply the same arbitration to adversarial findings.

### Adversarial ordering

- Tier 2 only: start adversarial review after specialist blockers are closed.
- After an adversarial fix: implementer → relevant specialists recheck → adversarial recheck, all on the new revision.

## 7. Shared state (registry)

### Location and writes

- Canonical registry on `refs/heads/workflow-state`: `state.json` holds `{"schema_version": 2}` and each task is
  `tasks/<ID>.json`. Read it with `solomonar task show [<ID>]` or `git show refs/heads/workflow-state:tasks/<ID>.json`.
  A schema-1 registry (one `state.json` with every task) is migrated once by the user with `solomonar task migrate`;
  commands reject it until then (`registry_outdated`).
- Only `solomonar task` commands write it. Agents never edit, commit, or check out the registry, force-push
  it, or merge it into main. It has no filesystem checkout. These boundaries concern the registry and its
  Git ref, not ordinary product files named `state.json`; do not assume runtime hooks enforce them (section 11).
- The CLI is a guard and an oracle, never a driver: it validates and records what the orchestrator decided and
  reports what Git shows. It never launches agents, calls models, or stores review verdicts. State holds
  decisions; Git holds what happened; `task status` reconciles them.

| Command | Who | Effect |
|---|---|---|
| `task plan --title <t>` | user | Creates a `planned` record without owner |
| `task register [<ID>] --title <t> --owner <o>` | orchestrator | Claims the `planned` `<ID>` or allocates the next `W-NNN`; sets `reserved` / `setup`, branch `feature/<ID>`, `next_action` |
| `task move <ID> --as <owner> <field flags>` | owner | Validated transition and field update; repeated `--agent <id>:<role>` replaces `active_agents`, `--agent none` clears it |
| `task block <ID> --as <owner> --kind <k> --summary <s>` / `task unblock <ID> <blocker-ID> --as <owner>` | owner | Adds or clears a blocker; combine `--status` and `--next` atomically (section 3) |
| `task done <ID> --as <owner> --merge <sha> --spec-ref <sha:path> --summary <s>` | owner | Verifies the merge commit is reachable from main and `spec_ref` resolves and names the last `spec(<ID>)` commit merged, then writes the compact record (`done` / `complete`) |
| `task handoff <ID> --owner <o>` | new owner, on user instruction | Ownership change; clears the confirmation (section 10) |
| `task confirm <ID> --as <owner>` | owner, after inspection | Checks the branch, recorded worktree and base, and the approved plan; records that this execution continues the task (section 10) |
| `task migrate` | user | Rewrites a schema-1 registry as one file per task, once |
| `task show [<ID>]` | anyone | Reads the stored record |
| `task status <ID> [--json]` | anyone | Reads record, Git and available forge facts, drift, and allowed next commands; writes nothing |

- `planned` records are created by the user with `task plan`; they have no owner. This is the only state write
  not made by an orchestrator. Otherwise each orchestrator writes only its own task's record.
- Owner commands accept `SOLOMONAR_OWNER` as the terminal default for `--as`. Agents pass `--as <owner>` on
  every `move`, `block`, `unblock`, and `done` call; shells need not preserve environment variables.
- Discover field flags with `solomonar help task move`. Combine related changes in one validated update.
- The CLI validates schema, ownership, and allowed transitions, and checks facts before writing: `--revision` is
  a full SHA reachable from `feature/<ID>`; `--spec` exists in the feature worktree or at the recorded
  `revision`; a stage move respects the recorded `tier`; stage changes and `task done` need a confirmed task;
  entering `implementation` needs the approved plan committed; leaving it needs a checkpointed, clean candidate
  (section 5 Checkpoints). A rejection names the failed check and the corrective
  action. Compare-and-swap protects registry updates from concurrent writers; no checkout or lock is needed.
- Each state-changing command creates one `state(<ID>): ...` commit. Read-only commands, dry runs, and
  unchanged moves create none. Git history is the transition history; the registry holds no event log.
  Rejected updates leave the registry unchanged: report the diagnostic and preserve its last successful state.
- The CLI pushes `workflow-state` when a remote is configured and reports publication failures; the local ref
  stays canonical after publication failure. Multiple independently writable clones are out of scope.
- Local Git facts are required. PR and check facts come from `gh` when it is installed and authenticated;
  otherwise `task status` reports them as `unknown`, never as passing. `task done` relies only on Git.

### Status

| Status | Meaning |
|---|---|
| `planned` | Known work; no owner yet |
| `reserved` | Registration committed; worktree may not exist yet |
| `running` | Owner is actively advancing the stage |
| `blocked` | A blocker stops all authorized work on the task |
| `paused` | Stopped on instruction; resumes on instruction |
| `cancelled` | Stopped without merge; work preserved; explicit resumption required |
| `done` | Feature merge into main confirmed; compact record written; final |

### Stages and transitions

`setup` → `specification` → `implementation` → `specialized_review` → `adversarial_review` → `integration`
(PR, required CI, compatibility with main, closure commits, merge) → `complete`.

- Stage reflects the current activity. Preserve the stage while `blocked` or `paused`.
- Tier 0 goes from `implementation` to `integration`; tier 1 skips `adversarial_review` (section 6). The CLI enforces
  this from the recorded `tier`; no stage after `implementation` without one.
- `planned` → `reserved` only through `task register`; registration may also allocate a new task directly.
  `reserved` → `running` pairs with `setup` → `specification` in one update. Only `task done` sets `done` / `complete`.
- `reserved` or `running` → `blocked` or `paused` → back to the prior status at the preserved stage.
- A task is `reserved` exactly while at stage `setup`: the move out of `setup` sets `running`, and no stage
  returns to `setup`. The prior status of a `blocked`, `paused`, or resumed `cancelled` task is therefore
  `reserved` at `setup` and `running` at any later stage.
- `cancelled` resumes only on explicit user instruction after inspecting preserved work, using
  `task move <ID> --as <owner> --resume-cancelled --status <prior> --next "<actor>:<action>"`.
- A `done` task stays `done`. A regression or revert is a new task linked via `related_task`.
- Corrections return to `implementation` without a new task ID, as new plan steps (section 5 Checkpoints).

### Fields

| Phase | Required fields |
|---|---|
| Planned | `title`, `status`, `updated_at` |
| Reserved | adds `owner`, `branch`, `stage`, `next_action` |
| Past setup | adds `base` (full SHA, never changed), `worktree` (path relative to the main checkout), `confirmed` |
| Before implementation | adds `spec` (worktree-relative path) |
| Review / integration | adds `revision`: latest reported committed candidate, full SHA; `tier` (0, 1, 2) from that candidate's triage (section 6) |
| Done (compact) | `title`, `status`, `stage`, `summary`, `spec_ref`, `merge_commit`, `updated_at`; optional `base`, `pr`, and `related_task`; active fields removed |

- Empty `active_agents` and `blockers` are omitted. `active_agents` entries `{id, role}` record the last
  delegation, not liveness. Optional fields include `pr`, `depends_on`, `related_task`, and `session_ref`;
  omit empty values and do not invent schema fields for evidence.
- Blocker: `{id, kind, summary, ref?}`, kind ∈ `decision` | `dependency` | `check` | `environment`. Blocker
  IDs stay stable within the task and are never reused: `last_blocker` holds the highest number issued.
- `next_action`: `{actor, action}`, e.g. `{"actor":"user","action":"Decide W-001/D-1: session storage"}`.
- Use full SHAs and RFC 3339 UTC timestamps. Never store logs, secrets, or absolute paths in state; `worktree`
  is relative to the main checkout, and Git's worktree listing stays authoritative. After `git worktree move`,
  record the relocation with `task move <ID> --as <owner> --worktree <path>`.
- Never store a fact Git can derive (HEAD, dirty files, checkpoints, PR or check state); `task status` reads
  it. `revision` is stored because reviews pin it, `base` because it bounds the task's checkpoint history; the
  CLI verifies both.
- Resolve active `spec` paths against the feature worktree (section 4 step 4) or its recorded `revision`.

## 8. PR and merge

- Merge only when all hold: mandatory criteria met; affected specs reflect approved decisions; required
  reviews complete and valid for the current candidate; no blocking defects or open decisions; mandatory
  checks and integration with main succeeded; closure commits made (section 9); repository rules satisfied.
- No extra human merge confirmation by default. Approvals required by the repository stay mandatory. An
  informal approval message never replaces CI or repository rules.
- Record every verdict with the revision checked in the PR Verification section.
- Staleness: before merge the feature branch contains current main (merged by the implementer,
  `implementer.md` section 7). CI always reruns. Compare main's changes with the candidate, its referenced contracts,
  dependencies, and requirements. With `<base>` = `git merge-base <reviewed-revision> <main>`, overlapping paths
  from `git diff --name-only <base>..<reviewed-revision>` and `git diff --name-only <base>..<main>`, merge
  conflicts, or a changed relevant contract/dependency require the affected reviews to rerun. Disjoint paths
  alone are not a reuse reason. Reuse a verdict only with its original SHA and a concrete reason why main's
  changes do not affect its scope; if impact is unclear, rerun that review. A rerun moves tier 1 or 2 back to
  `specialized_review` and through the required stages to `integration`; tier 0 keeps its CI-only gate.
- A spec rule ID that collides with one already on main after merging main: renumber the task's rule with the
  `spec` skill, update the work spec, and treat it as a changed candidate (section 6).
- Merge only with a merge commit (section 11), never squash or rebase, so the final spec commit survives in main's
  history. Merge one task at a time. Run `task done` only after the merge is confirmed (section 9).

## 9. Closure (orchestrator rule, not a skill)

Before merge, after every required gate passes on the candidate:

1. Canonical documents must reflect approved changes. Never rewrite requirements to match implementation omissions.
2. The final spec commit must match the spec used for final validation. If the work spec is unchanged since its
   last `spec(<ID>)` commit inside the reviewed revision, that commit is the final spec commit; no new revision.
   Only when content changed, commit that file alone as `spec(<ID>): finalize task specification` and treat it
   as a changed candidate (section 6 Candidate stability). The final spec commit and the work spec path form `spec_ref`;
   write it on the PR Verification line `Final specification` (`report.md` section 8).
3. One last commit deletes `docs/work/<ID>-<slug>.md`: `docs(<ID>): remove completed work spec`. It needs no
   agent review (docs-only deletion of the task spec); CI reruns. A later fix goes on top of it; packets then
   name `Work spec: <spec_ref>`, read with `git show "<spec_ref>"`. A fix that changes the work spec first
   reverts the deletion commit, follows the spec flow, and repeats steps 2–3.

At confirmed merge:
- Run `task done <ID> --as <owner> --merge <sha> --spec-ref <sha:path> --summary <s>`, with `spec_ref` from the
  PR's `Final specification` line. It verifies the merge commit is reachable from main and `git show
  "<spec_ref>"` resolves to the last `spec(<ID>)` commit merged, then writes the compact record (section 7) with
  `done` / `complete`. `done` is final at merge.
- If `task done` fails, report the cause and rerun it once fixed. Never revert the merge.
- Delete a worktree only as a separate deliberate step, after confirming it holds no uncommitted or unmerged work.
- History lookup: read the compact record first. Run `git show "<spec_ref>"` only for a concrete question.

## 10. Recovery

Before acting on any event, inspect actual state: `task status <ID>` (section 7) reports Git state, worktree, dirty
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
4. Run `task confirm <ID> --as <owner>`. Confirmation proves neither plan approval nor review success.
5. Delegate only the remaining steps, with the partial work to inspect and reuse. With every step committed,
   proceed to triage and review; repeat verification whose evidence cannot be recovered.

A discrepancy (a commit matching a step only by subject, a wrong trailer, an empty or duplicate checkpoint) needs
investigation, never a reset, a duplicate worktree, or an assumption that the step succeeded. Commit amendments
to the plan rather than rewriting checkpoint history.

| Event | Orchestrator behavior |
|---|---|
| Pause | Stop delegating; settle or stop active work; `task move <ID> --as <owner> --status paused --next "<actor>:<outstanding work>"`; preserve stage and recorded revision |
| Unexpected session end | Nothing automatic; the next session follows Resume |
| Resume | Inspect and reconcile; restore the prior status at the preserved stage (section 7), subject to remaining blockers; reuse the worktree and PR. If merged, reconcile to `running` / `integration` and run `task done` (section 9) with `spec_ref` from the PR, without reimplementing |
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

## 11. Bootstrap and runtime assumptions

- Before the first managed task, the following must exist: a local Git baseline and `refs/heads/workflow-state`
  with an empty registry, created by `solomonar init`. No other checkout is needed for state.
- v0.1 requires a Git repository and Git worktrees; Jujutsu and other VCS front ends are out of scope.
- The repository must allow merge commits, and the orchestrator merges only with a merge commit: squash and
  rebase merges break `spec_ref`. Verify repository merge settings before integration.
- Use `solomonar help` to check command availability. Extension management, `doctor`, and guard hooks are
  planned capabilities; do not run missing commands or assume hooks enforce policy. Use `doctor` for project
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
