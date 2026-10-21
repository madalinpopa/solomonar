---
title: Follow a task
description: Trace the framework's intended path from a selected outcome to a reviewed, integrated change.
kind: How-to guide
group: Understand
order: 3
sources:
  - assets/framework/rules.md
  - assets/framework/skills/work-spec/SKILL.md
  - assets/framework/report.md
  - docs/cli/interface.md
---

> **Framework walkthrough:** this page explains the agreed workflow. The complete CLI path is still under development, including initialization and status inspection. Use [Getting started]({{ '/docs/getting-started/' | url }}) for commands you can run against the current source build.

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

Implementation can start when the outcome is authorized, the scope is understandable, the criteria can be verified, and no unresolved decision blocks the chosen approach. Existing approvals apply within their recorded scope.

## Implement and review one candidate

The implementer returns `READY_FOR_REVIEW` with a committed revision. The orchestrator records the candidate and triages its risk.

| Tier | Change | Required review path |
| --- | --- | --- |
| 0 | Documentation only, without behavior or requirement changes | CI checks; no agent-review stages. |
| 1 | Normal code | Required specialist reviews. |
| 2 | Persistence, authentication, concurrency, external integrations, migration, or an approved architecture or dependency decision | Specialists, then adversarial review. |

Required specialists inspect the same stable revision in parallel. The orchestrator deduplicates findings and orders corrections by Security, Correctness, then Spec priority. An inapplicable specialty is marked not required with a reason; it does not receive a fabricated pass.

A code change creates a new candidate. An earlier review covers it only when its original revision and a specific reason for reusing unaffected evidence are recorded.

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

On restart, inspect the stored record and actual Git, worktree, PR, and check state before repeating an action. Preserve uncommitted changes. Continue the same task rather than creating another worktree or PR.

The planned `task status` command is designed to show facts and drift; it does not infer whether an agent is alive from its last reported assignment. See [task state]({{ '/docs/task-state/' | url }}).
