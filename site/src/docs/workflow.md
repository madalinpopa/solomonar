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

This page explains each stage of a managed task and the rule behind it. For a hands-on walk through the same path, start with [Run your first task]({{ '/docs/first-task/' | url }}).

## Start with one useful outcome

When behavior is unclear, explore the next useful flow with `prototype`. Use `spec` to turn intentional decisions into agreed rules. A known brief can go directly to a managed task; no UI prototype is required for a change whose behavior is already understood.

A useful task prompt names the outcome and the boundaries that matter. For example:

```text
Use the solomonar skill. Add CSV export to the existing CLI.
Keep the current text output unchanged. No UI prototype is needed.
```

This is an example of a request, not a built-in export feature.

## Register, then isolate

The orchestrator reads the registry and checks ownership and dependencies. It reserves a task with `task register`; successful local registration is the barrier before creating its branch and worktree.

The user starts tasks one at a time. Once registered, independent tasks can run in parallel in separate worktrees. Separate ports and test data still need explicit care; worktrees isolate files, not shared services.

## Agree the work before implementation

In the task worktree, `work-spec` prepares the selected outcome, scope, acceptance criteria, relevant references, constraints, and checks. User-confirmed changes to canonical behavior go into the relevant specifications before code is written.

The work spec ends with an implementation plan: each step is the commit it will become, such as `S-1: feat(W-001): store expenses`. The orchestrator shows you the plan; implementation starts only after you approve it explicitly, the orchestrator records `Status: Approved`, and commits it. An approval you already gave for the exact plan is reused, never asked twice.

Implementation can start when the outcome is authorized, the scope is understandable, the criteria can be verified, no unresolved decision blocks the chosen approach, and the plan is approved. Existing approvals apply within their recorded scope.

## Implement and review one candidate

The implementer follows the approved plan and commits each step as a checkpoint with its planned subject and a `Step:` trailer. It returns `READY_FOR_REVIEW` once every step is committed, with the branch tip as the candidate. The orchestrator records the candidate and triages its risk. Reviews cover the whole change on that candidate, never one checkpoint at a time.

| Tier | Change | Required review path |
| --- | --- | --- |
| 0 | Documentation only, without behavior or requirement changes | CI checks; no agent-review stages. |
| 1 | Normal code | Required specialist reviews. |
| 2 | Persistence, authentication, concurrency, external integrations, migration, or an approved architecture or dependency decision | Specialists, then adversarial review. |

Required specialists inspect the same stable revision in parallel. The orchestrator deduplicates findings and orders corrections by Security, Correctness, then Spec priority. An inapplicable specialty is marked not required with a reason; it does not receive a fabricated pass.

Corrections are appended to the plan as new steps, so the plan stays complete without a new approval while they stay within the approved scope. A code change creates a new candidate. An earlier review covers it only when its original revision and a specific reason for reusing unaffected evidence are recorded.

## Handle a decision or a failed check

New architecture and dependency decisions go back to the user through the orchestrator. A decision blocker names the pending choice and the next actor. Dependent work waits; independent authorized work may continue.

Review results have precise meanings:

| Result | Meaning |
| --- | --- |
| `READY_FOR_REVIEW` | Implementation handoff; it approves nothing. |
| `PASS` | The reviewer completed its scope without blockers or missing required checks. |
| `CHANGES_REQUIRED` | Confirmed defects return to implementation. |
| `NEEDS_DECISION` | Dependent work waits for a user decision. |
| `INCOMPLETE` | Evidence or an important check is missing. |

No response is approval. After two rounds that leave the same blocker unchanged or reintroduce it, the orchestrator stops the loop, explains the cause, and brings options to the user.

## Integrate and preserve the final specification

Before merging, required checks and reviews must pass, canonical documents must reflect approved changes, blockers must be closed, and compatibility with current `main` must be checked.

The final task specification has its own `spec(<ID>):` commit. A separate final commit removes the active work-spec file. The PR merges with a **merge commit**, preserving both commits in history.

After the merge is confirmed, `task done` verifies the merge and the final `spec_ref`, then writes the compact `done / complete` record. A merge failure or missing check never counts as completion.

## Resume from evidence

On restart, inspect the stored record and actual Git, worktree, PR, and check state before repeating an action. Preserve uncommitted changes. Continue the same task rather than creating another worktree or PR. A new session takes the task with `task handoff` on your instruction, confirms it with `task confirm` after inspection, and continues with the remaining plan steps; completed steps are never repeated.

`solomonar task status <ID>` shows those facts, the plan's committed and remaining steps, any drift from the record, and the commands valid next. It does not infer whether an agent is alive from its last reported assignment. See [task state]({{ '/docs/task-state/' | url }}).
