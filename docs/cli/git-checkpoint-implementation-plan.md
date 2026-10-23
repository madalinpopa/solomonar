# Git checkpoint task recovery: implementation plan

Status: implemented on `feat/git-checkpoint-recovery`; live runtime acceptance (step 5) is pending.
It does not describe behavior already delivered by the CLI.

## Goal

Allow a new orchestrator session to continue an existing task from its approved implementation plan,
Git checkpoints, and preserved working files, without requiring the previous conversation or repeating
completed implementation steps.

Keep the CLI as a guard and an oracle. The selected approach uses validation at managed workflow gates
and a partial execution history. It does not introduce an agent runtime, automatic tool-call recording,
heartbeats, a daemon, SQLite, a separate event journal, or a `checkpoint.json` file.

## Agreed behavior

### Task registry

- Keep the registry on the separate `workflow-state` branch. Never merge it into `main`.
- Store one JSON file per task, such as `W-001.json`, instead of one document containing every task.
  The directory layout and schema-version metadata still need a contract.
- The agreed record fields are the task title, task number, worktree, branch, base commit reference,
  `depends_on`, and `updated_at`. Confirmation and lifecycle fields require the design work below.
- The base commit is the revision before task work starts. It is not the latest checkpoint or the
  candidate submitted for review, and importing subsequent changes from `main` does not change it.
- A task starts unconfirmed. A new session inspects and reconciles its actual state before confirming
  continuation. Confirmation does not establish plan approval, review success, or agent liveness.
- Completed tasks are not reset on inspection. Context compaction within the same execution is not a
  new ownership handoff.

### Task workspace and specification

- Each task has its own branch and worktree. Recovery reuses them rather than allocating a new task.
- The agent in the main session is the orchestrator. Only it edits the task's work specification.
- The work specification contains the objective and scope, acceptance criteria, relevant references,
  constraints and checks, and an implementation plan.
- Put a plan-approval status immediately below the title, with no separate approval section. The agreed
  values in the discussion were `status: In Review` and `status: Aprobat`, meaning approved. Settle the
  canonical spelling for the English framework before implementation; do not silently translate a
  machine-readable value.
- The user explicitly approves the implementation plan before implementation starts. The orchestrator
  records that approval and commits the approved document before delegation. Reuse an existing explicit
  approval when it covers the exact plan; do not ask for a duplicate approval.
- Express each implementation-plan step as its intended commit message, with enough detail to delimit
  the work. The document is the approved plan, not a second execution log.
- The discussion used `W-001.md`; the current framework requires `docs/work/<ID>-<slug>.md`. Resolve the
  filename contract explicitly while preserving existing task references.

### Implementation and checkpoints

- The orchestrator delegates implementation. The implementer executes the approved plan in the assigned
  worktree and creates an actual checkpoint commit for each completed step.
- Use the planned message as the commit subject. Keep the body concise: progress, checks and their
  results, and observations needed to continue.
- Git commits preserve the implementation and its checkpoint notes together. Do not duplicate progress
  in a JSON checkpoint file or a separate Markdown execution checklist.
- A commit is not proof of correctness. Recovery checks its changes as well as its relationship to a
  plan step. Matching a subject alone is insufficient.
- A checkpoint preserves only committed content. Later working-tree changes must be inspected and
  preserved separately; ignored files are not saved automatically.

### Recovery, review, and closure

- A new orchestrator reads the task record, approved specification, checkpoint history since the base,
  and working-tree status. It loads detailed diffs and code only where needed.
- Confirm the previous execution has stopped or isolate its access before allowing replacement work
  in the same worktree. A stored owner or a closed conversation is not proof that execution stopped.
- Continue the plan steps without corresponding valid checkpoint commits. Inspect and reuse justified
  partial work rather than overwriting it or repeating completed steps.
- If all implementation steps have commits, proceed to review. Repeat verification whose evidence
  cannot be recovered; do not treat lost reports as passing.
- Review the cumulative task change on one stable candidate after the complete implementation or a
  complete correction batch, not after every checkpoint commit.
- Preserve the current risk-based review policy: tier 0 uses CI only; tier 1 uses the required specialist
  and applicable extension reviewers; tier 2 adds adversarial review after specialist blockers close.
- The orchestrator consolidates corrections, delegates them, and reruns affected reviews on the new
  candidate. Review success is followed by integration, required CI, PR checks, and merge before closure.

## Current implementation and gaps

| Area | Current behavior | Required change |
| --- | --- | --- |
| Registry | [`registry.go`](../../internal/task/registry.go) decodes schema 1 with a `tasks` map; [`store.go`](../../internal/task/store.go) stores `state.json` | Versioned per-task files, compatible discovery, and migration |
| Concurrent updates | Store updates use Git compare-and-swap and revalidation, with five attempts | Preserve this protection, including global task-number allocation and other task files |
| Setup | Registration reserves an ID before the orchestrator creates the worktree | Record the verified base and complete workspace fields through checked updates |
| Worktree | [`facts.go`](../../internal/task/facts.go) derives the worktree from Git; the rules prohibit storing machine-specific paths | Store the requested location while defining validation, relocation, and portability |
| Implementation entry | Fact checks establish that the specification exists | Validate the approved-plan contract before accepting implementation entry |
| Work specification | The skill excludes execution state and has no implementation plan or plan-approval field | Add the explicitly scoped approval field and commit-based plan |
| Implementer | Writes its own approach and commits authorized changes without a predefined commit sequence | Execute the orchestrator's approved plan and checkpoint protocol |
| Recovery | State, Git facts, worktree, and durable report references support reconciliation | Add plan-to-checkpoint reconciliation and explicit confirmation |
| Closure | [`done.go`](../../internal/task/done.go) writes a compact record and verifies the merge and final `spec(<ID>)` commit | Preserve final-plan and checkpoint traceability under the revised record contract |

The existing store publishes the state branch on a best-effort basis; a publication failure leaves the
local registry canonical. Multiple independently writable clones remain outside this change's scope.
Separate task files do not remove contention on the shared Git reference.

## Implementation work

### 1. Define the data and workflow contracts

Update [the CLI interface](interface.md), [the architecture](architecture.md), and
[the framework design](../design/framework-v0.1.md) before changing behavior.

- [ ] Define the per-task registry layout, version metadata, empty-registry representation, and required
  fields at registration, completed setup, active work, and closure.
- [ ] Distinguish the immutable base, approved-plan version, checkpoint commits, reviewed candidate,
  and final merge. Reuse derivable Git facts rather than adding redundant fields without a need.
- [ ] Define confirmation independently from lifecycle progress. A task can require confirmation while
  already being at implementation or review. Settle field names and values rather than overloading one enum.
- [ ] Define how an execution is identified for preflight and confirmation. The CLI cannot assume that
  the runtime exposes a native session identifier or detect a new session automatically.
- [ ] Define stable plan-step identifiers and how commits identify the task, step, and relevant plan
  version. Recommend structured commit metadata; exact names and syntax are not yet approved.
- [ ] Define corrections, plan amendments, steps that become unnecessary, and verification-only outcomes.
  Do not fabricate empty commits simply to satisfy a checklist.
- [ ] Define workspace relocation and mismatches between the recorded path and Git's worktree listing.
- [ ] Define preservation of the approved plan through closure. The existing deletion-before-merge policy
  may remain only if durable references keep the plan recoverable.

Completion: the normal flow, recovery flow, failure cases, and record invariants are specified without
requiring a particular runtime or claiming that recorded declarations prove external approval.

### 2. Change registry storage and provide migration

Depends on the data contract in step 1.

- [x] Update [`registry.go`](../../internal/task/registry.go),
  [`validate.go`](../../internal/task/validate.go), and [`store.go`](../../internal/task/store.go) for the
  new layout, strict decoding, deterministic writes, and supported schema versions.
- [x] Preserve unrelated task files and registry metadata when constructing a new Git tree. Keep
  compare-and-swap retries and rerun validation against the latest registry.
- [x] Adapt [`init.go`](../../internal/cli/init.go), its plan rendering, and the
  [existing registry template](../../assets/templates/state.json) to initialize and recognize the new layout.
- [x] Provide an explicit migration path. Preserve IDs, dependencies, applicable lifecycle information,
  and the previous registry history. Do not infer missing baselines or past checkpoints as confirmed facts.
- [x] Define handling of legacy completed records, which have already discarded active fields. Define
  read compatibility, downgrade rejection, and a recovery procedure for unsuccessful migration.
- [x] Keep registry migration separate from framework/adaptor updates, but document their compatible
  versions so old instructions cannot silently operate against incompatible state. Framework 0.1.6 and
  earlier read schema 1; these rules require schema 2, and a 0.1.6 CLI rejects schema 2 as `cli_outdated`.

Completion: new repositories and migrated repositories support safe task updates, while unsupported
versions fail clearly without discarding data.

### 3. Add checked setup, progress inspection, and recovery

Depends on steps 1 and 2. Final command names and flags belong in the interface contract; the current
`task plan` command creates a backlog record and must not be confused with implementation-plan approval.

- [x] Extend registration and setup through [`register.go`](../../internal/task/register.go),
  [the CLI registration command](../../internal/cli/register.go), and fact checks. Capture the verified
  base and validate the branch and worktree before accepting setup completion.
- [x] Add plan and checkpoint inspection that correlates approved steps with actual commits. Account
  for specification commits, integration merges, correction commits, and unrelated or duplicate matches.
- [x] Extend [`status.go`](../../internal/task/status.go), [`drift.go`](../../internal/task/drift.go),
  and [CLI status output](../../internal/cli/status.go) with compact recovery information: approval,
  completed and remaining steps, dirty changes, and discrepancies requiring inspection.
- [x] Keep inspection read-only. Add explicit preflight/confirmation and reconcile ownership handoff with
  [`handoff.go`](../../internal/task/handoff.go); do not turn inspection into automatic takeover.
- [x] Preserve partial and unrelated changes. Missing or inconsistent evidence requires investigation,
  not a reset, duplicate worktree, or assumption that a plan step succeeded.
- [x] Specify recovery when a checkpoint commit exists but a subsequent state update failed. A task-branch
  commit and a registry update are separate operations, not one atomic transaction.

Completion: a replacement session can locate the same task, identify remaining work, and confirm a
safe continuation point without the original conversation.

### 4. Enforce managed gates and adapt closure

Depends on the approved-plan and recovery contracts and step 3.

- [x] Update [`move.go`](../../internal/task/move.go), [`movetask.go`](../../internal/task/movetask.go),
  [`transition.go`](../../internal/task/transition.go), and fact checks to reject implementation entry
  without the required approved-plan evidence.
- [x] Require complete checkpoint coverage and a clean, correctly referenced candidate before review
  entry. Validate structural evidence without claiming to prove semantic completion.
- [x] Update allowed-next-command reporting from the same rules, and align human and JSON errors and output.
- [x] Preserve the existing risk tiers, review ordering, and review-reuse conditions. Do not add verdict
  storage to the registry implicitly; missing durable evidence still requires verification to be repeated.
- [x] Adapt [`done.go`](../../internal/task/done.go) and final-spec discovery to the revised plan lifecycle.
  Closure must still verify the correct reviewed work was merged and preserve required historical references.

Completion: the CLI rejects invalid managed transitions while leaving arbitrary file execution outside
its enforcement claims.

### 5. Align framework rules, roles, skills, and adapters

Depends on the settled CLI contract. Deliver these instructions with the compatible implementation.

| Source | Required changes |
| --- | --- |
| [`rules.md`](../../assets/framework/rules.md) | Registry layout, approval gate, checkpoints, confirmation, recovery, and closure |
| [`agents/orchestrator.md`](../../assets/framework/agents/orchestrator.md) | Plan ownership and approval, setup, preflight, remaining-step delegation, stable review handoff, and recovery |
| [`agents/implementer.md`](../../assets/framework/agents/implementer.md) | Follow the approved commit plan; create checkpoint commits; report deviations; preserve partial work; do not edit the specification |
| [`skills/work-spec/SKILL.md`](../../assets/framework/skills/work-spec/SKILL.md) and [`template.md`](../../assets/framework/skills/work-spec/template.md) | Approval field below the title, implementation-plan format, amendment rules, and validation |
| [`skills/solomonar/SKILL.md`](../../assets/framework/skills/solomonar/SKILL.md) | Route new work and resumed work through the correct preflight and delegation path |
| [`reviewer.md`](../../assets/framework/reviewer.md), [`report.md`](../../assets/framework/report.md), and reviewer roles | Cumulative review scope, stable candidate references, recoverable evidence, and correction rounds |
| [`AGENTS.md` template](../../assets/templates/AGENTS.md) and [document-index template](../../assets/templates/docs/README.md) | New registry references and concise ownership/navigation guidance |

- [x] Remove the conflicting requirement that the implementer independently establishes the implementation
  approach after an orchestrator-approved plan already exists.
- [x] Keep framework instructions in English and keep `AGENTS.md` compact. Place detailed procedures in
  the relevant role or skill rather than duplicating the whole workflow.
- [x] Keep checkpoint notes incremental and short. Give implementers the remaining approved steps and
  relevant constraints; give reviewers the specification and candidate diff, not the implementation conversation.
- [x] Inspect [`runtime.go`](../../internal/project/runtime.go) and adapter tests for required changes.
  Prefer the existing portable instruction sources; do not introduce hooks without a separate requirement.
- [ ] Verify instruction discovery and delegation packets for Claude, Codex, and Copilot. Generated-file
  tests do not establish live runtime acceptance, including the currently pending Copilot workflow acceptance.

Completion: no framework path still instructs agents to use the old registry format, bypass plan approval,
duplicate progress, or review each checkpoint independently.

### 6. Update documentation and examples

Depends on the final user-facing contract.

- [x] Update [README](../../README.md), [CONTRIBUTING](../../CONTRIBUTING.md), the design contracts, and
  installed documentation templates.
- [x] Update site pages for [task state](../../site/src/docs/task-state.md),
  [workflow](../../site/src/docs/workflow.md), [the first task](../../site/src/docs/first-task.md), and
  [task commands](../../site/src/docs/cli/task.md).
- [x] Update [configuration](../../site/src/docs/configuration.md),
  [project setup](../../site/src/docs/set-up-a-project.md),
  [troubleshooting](../../site/src/docs/troubleshooting.md), and
  [use cases](../../site/src/docs/use-cases.md) where the old contracts appear.
- [x] Revise affected diagrams in `docs/images/` and references elsewhere in the site. Explain the
  distinction between base, checkpoint, approved plan, reviewed candidate, and confirmed merge.
- [x] Include a complete successful task and a recovery example with committed steps plus unfinished
  working-tree changes. Document migration and workspace relocation.
- [x] Describe commands as available only once implemented. Document managed-gate enforcement limits
  and the absence of automatic session-failure detection.

Completion: examples, command reference, framework assets, and implementation agree on one workflow.

## Verification and acceptance

Use focused tests at the existing boundaries rather than repeating every scenario at every test layer.

- Registry and schema tests: strict decoding, version rejection, per-task writes preserving other files,
  migration without invented history, and legacy completed records.
- Concurrency tests: distinct task-number allocation, conflicting claims, and revalidation after a lost swap.
- Git-fact and recovery tests: wrong base or worktree, moved worktree, partial setup, missing checkpoint,
  duplicate or subject-only matches, dirty partial work, and checkpoint/state-update interruption.
- Transition tests: unapproved plan rejected; confirmation separate from lifecycle; all planned steps
  committed leads to review; correction rounds preserve plan and candidate semantics.
- Closure tests: the correct candidate and recoverable final specification survive merge and completion.
- CLI and transcript tests: compact human/JSON output, actionable rejections, initialization/migration,
  and a representative end-to-end task and resume journey in `testdata/`.
- Framework and adapter tests: source ownership, installed templates, and instruction discovery. Record
  manual runtime acceptance separately from fixture or generated-file checks.
- During implementation, run the relevant package tests and the required checks from `Taskfile.yml`.
  Validate the site with its `npm run build` and `npm run check` scripts, and inspect Markdown links and diffs.

The principal acceptance condition is that a new session can continue the same task from preserved
evidence, without the previous conversation, repeating completed steps, or losing uncommitted work.
Review and final integration remain required even when every planned checkpoint exists.

## Resolved contract

Decided by the author on 2026-10-23 (CLI-D20 to CLI-D26 in [the interface](interface.md#11-open-decisions)).
The interface and architecture carry the normative detail; this section is the summary they implement.

1. **Registry layout (schema 2).** The `workflow-state` tree holds `state.json`, exactly
   `{"schema_version": 2}`, and one `tasks/<ID>.json` per task with the record in canonical form. The file name
   is the ID; no other entry is allowed. An empty registry is `state.json` alone. Keeping the version in
   `state.json` makes a schema-1 CLI reject the new layout as `cli_outdated` instead of failing on a missing file.
   Every write still swaps the whole ref, so allocation and revalidation keep their compare-and-swap guarantees.
2. **Record fields.** New optional fields: `base` (full SHA), `worktree` (clean `/`-separated path relative to
   the main checkout, such as `../ledger-worktrees/W-001`), and `confirmed` (`true`, omitted when false).
   The move from `setup` to `specification` records `base` as `git merge-base <main> feature/<ID>`, `worktree`
   from `git worktree list`, and `confirmed`. `base` never changes once recorded; `--base` only fills a missing
   one. `--worktree` records a relocation and must match Git's listing. A done record keeps `base`.
3. **Confirmation.** An execution is an owner ID; a new session takes the task through `task handoff`, which
   clears `confirmed`. `task confirm <ID> --as <owner>` runs the preflight (branch, recorded worktree matching
   Git, recorded base on the branch, an approved plan from `implementation` on) and sets `confirmed`.
   Stage changes and `task done` need `confirmed`; status-only moves do not. Confirmation proves neither plan
   approval, review success, nor that the previous execution stopped.
4. **Work spec.** The path stays `docs/work/<ID>-<slug>.md`. The line below the title is `Status: In review`
   or `Status: Approved`. `## Implementation plan` lists steps `- S-<n>: <commit subject>` with indented
   detail lines; an indented `Outcome: dropped — <reason>` or `Outcome: verified — <reason>` closes a step without
   a commit. Step IDs are unique and never reused; a changed approved step takes a new ID and the status returns
   to `In review`. Review corrections within the approved scope are appended as new steps without new approval.
5. **Checkpoints.** The implementer commits each step with the planned subject and a `Step: <ID>/S-<n>`
   trailer; there is no plan-version trailer and no CLI commit command. Reconciliation reads
   `git log --no-merges <base>..feature/<ID>`: a commit counts for a step only with the trailer, the exact
   subject, and a non-empty change. Commits without a trailer (specification, decisions, closure) are other
   commits. Discrepancies: `step_untagged` (subject match without trailer), `step_subject_mismatch`,
   `step_unknown`, `step_foreign`, `step_empty`, `step_duplicate`.
6. **Managed gates.** Entering `implementation` needs `Status: Approved` and at least one step in the spec
   committed at the tip of `feature/<ID>`. Leaving `implementation` for review or integration needs `--revision`
   equal to the branch tip, a clean worktree, every step committed or closed by an outcome, and no discrepancy.
7. **Migration.** `solomonar task migrate` rewrites a schema-1 registry as schema 2 in one compare-and-swap
   commit `state: migrate schema 1 -> 2` whose parent is the old tip, so history is kept. It adds no `base`,
   `worktree`, or `confirmed`; migrated active tasks are confirmed explicitly. Other commands reject schema 1 with
   `registry_outdated` (fix `solomonar task migrate`). `upgrade` no longer migrates the registry.
8. **Closure.** The work spec is still deleted before merge; `spec_ref` keeps the final plan recoverable, and
   `base` with `merge_commit^2` bounds the checkpoint history of a done task.
