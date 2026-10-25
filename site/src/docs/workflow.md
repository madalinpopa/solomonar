---
title: From specification to reviewed code
navTitle: The task lifecycle
description: Follow a Solomonar task from acceptance criteria through AI implementation, independent review, pull request integration, and verified closure.
kind: Explanation
group: Learn the framework
order: 6
sources:
  - assets/framework/rules.md
  - assets/framework/skills/work-spec/SKILL.md
  - assets/framework/report.md
  - docs/cli/interface.md
---

This page explains the stages of a managed task and the gates between them. For concrete prompts and terminal examples, use [Task workflow]({{ '/docs/use-cases/' | url }}); for discovery and approval, use [Prototype workflow]({{ '/docs/prototype/' | url }}).

## Behavioral agreement

For new or changed behavior, use `prototype`, obtain human approval of the demonstrated flow, then use `spec` before production implementation. Update an existing prototype through the same loop. CLI commands, modules/libraries, and services use executable commands, consumers, or requests instead of a browser UI. Technical work preserving existing behavior and defect fixes restoring approved behavior reuse that agreement.

Behavioral approval establishes what the selected part does. Implementation-plan approval establishes the approach to building it. Neither substitutes for the other or selects production architecture by implication.

## Task ownership and isolation

The orchestrator reads the registry and checks ownership and dependencies. It reserves a task with `task register`; successful local registration is the barrier before creating its branch and worktree.

The user starts tasks one at a time. Once registered, independent tasks can run in parallel in separate worktrees. Separate ports and test data still need explicit care; worktrees isolate files, not shared services.

## Specification and plan approval

In the task worktree, `work-spec` prepares the selected outcome, scope, acceptance criteria, relevant references, constraints, and checks. New or changed behavior must be represented in an approved prototype; canonical behavior changes go into the relevant specifications before code is written.

The work spec ends with an implementation plan whose steps correspond to coherent, verifiable commits. Implementation requires explicit approval of that plan, recorded as a committed `Status: Approved`. Approval already covering the exact plan remains valid within its scope.

Implementation can start when the outcome is authorized, the scope is understandable, the criteria can be verified, approved prototype coverage exists for changed behavior, no unresolved decision blocks the chosen approach, and the plan is approved. Existing approvals apply within their recorded scope.

## Candidates and review tiers

The implementer follows the approved plan and commits each step as a checkpoint with its planned subject and a `Step:` trailer. It returns `READY_FOR_REVIEW` once every step is committed, with the branch tip as the candidate. The orchestrator records the candidate and triages its risk. Reviews cover the whole change on that candidate, never one checkpoint at a time.

| Tier | Change | Required review path |
| --- | --- | --- |
| 0 | Documentation only, without behavior or requirement changes | CI checks; no agent-review stages. |
| 1 | Normal code | Required specialist reviews. |
| 2 | Persistence, authentication, concurrency, external integrations, migration, or an approved architecture or dependency decision | Specialists, then adversarial review. |

Required specialists inspect the same stable revision in parallel. The orchestrator deduplicates findings and orders corrections by Security, Correctness, then Spec priority. An inapplicable specialty is marked not required with a reason; it does not receive a fabricated pass.

Corrections are appended to the plan as new steps, so the plan stays complete without a new approval while they stay within the approved scope. A code change creates a new candidate. An earlier review covers it only when its original revision and a specific reason for reusing unaffected evidence are recorded.

## Decisions and blockers

New architecture and dependency decisions go back to the user through the orchestrator. A decision blocker names the pending choice and the next actor. Dependent work waits; independent authorized work may continue.

Each decision and open question is registered at the level it affects. A choice that affects or constrains the whole project, such as an architecture style, a shared contract, or a dependency, is a project decision `P/D-n`. A choice that affects only one task is a task decision such as `W-001/D-1`, and its open questions stay in that task's work spec. A task decision never binds another task.

Project entries are written only on `main` in the main checkout, so parallel tasks never take the same ID. A task that raises one waits until you approve it and then merges `main`. Outside a task, the `solomonar` skill records a project decision after you approve an option; stack extensions record theirs through `solomonar extension add|remove`. The first task that applies a project decision updates `docs/architecture.md` or `docs/stack.md`.

Project open questions are the open items in the specifications and the pending `P/D-n` entries. When they exist, the agent suggests answering them; once you agree, it goes through them one at a time with your runtime's question prompt, recommended option first.

If a decision or review changes behavior, the orchestrator first routes it through the prototype in the primary checkout, waits for the approved committed handoff, and brings its updated references into the task branch. Specs and the work spec are updated before dependent production work resumes. Approval of a task decision or plan cannot replace that prototype review.

Review results have precise meanings:

| Result | Meaning |
| --- | --- |
| `READY_FOR_REVIEW` | Implementation handoff; it approves nothing. |
| `PASS` | The reviewer completed its scope without blockers or missing required checks. |
| `CHANGES_REQUIRED` | Confirmed defects return to implementation. |
| `NEEDS_DECISION` | Dependent work waits for a user decision. |
| `INCOMPLETE` | Evidence or an important check is missing. |

Silence never grants approval. After two rounds that leave the same blocker unchanged or reintroduce it, the orchestrator stops the loop, explains the cause, and brings options to the user.

## Integration and specification history

Before merging, required checks and reviews must pass, canonical documents must reflect approved changes, blockers must be closed, and compatibility with current `main` must be checked.

The final task specification has its own `spec(<ID>):` commit. A separate final commit removes the active work-spec file. The PR merges with a **merge commit**, preserving both commits in history.

After the merge is confirmed, `task done` verifies the merge and the final `spec_ref`, then writes the compact `done / complete` record. A merge failure or missing check never counts as completion.

## Recovery from recorded evidence

Recovery reconciles the stored record with actual Git, worktree, PR, and check state. Uncommitted work is preserved, and the task retains its worktree and PR. Ownership transfers only on explicit user instruction; confirmation follows inspection, and only the remaining plan steps continue. Completed steps are never repeated.

The CLI can compare recorded ownership and progress with current facts; it cannot infer whether an agent is alive from its last reported assignment. See [task state]({{ '/docs/task-state/' | url }}) for the model and [Task workflow]({{ '/docs/use-cases/#pause-resume-or-hand-work-to-another-session' | url }}) for recovery examples.
