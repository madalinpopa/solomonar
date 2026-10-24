---
title: Task state and workflow tracking
navTitle: Task state
description: Understand Solomonar task status, workflow stages, blockers, and Git-backed state validation so AI coding sessions can resume from recorded evidence.
kind: Explanation
group: Learn the framework
order: 7
sources:
  - assets/framework/rules.md
  - docs/cli/interface.md
  - internal/task/transition.go
  - internal/task/checkpoints.go
---

## Status and stage

{% diagram "status-stage" %}

**Status** says whether work can advance. **Stage** says where it has reached. A blocked or paused task preserves its stage.

| Status | Meaning |
| --- | --- |
| `planned` | Known work without an execution owner. |
| `reserved` | Registration committed; worktree setup may still be pending. |
| `running` | The owner is advancing the task. |
| `blocked` | A blocker prevents all authorized work on the task. |
| `paused` | Stopped on instruction; resumed on instruction. |
| `cancelled` | Stopped without a merge, with work preserved. |
| `done` | Merge into main confirmed; compact record written; final. |

The stages are `setup`, `specification`, `implementation`, `specialized_review`, `adversarial_review`, `integration`, and `complete`. Tier 0 skips agent-review stages; tier 1 skips adversarial review.

A task is `reserved` exactly while at `setup`. On leaving setup, it becomes `running`, and the CLI records the task's base commit, its worktree path, and the owner's confirmation. A blocked or paused task resumes to `reserved` at setup and `running` at a later stage.

## One registry, checked writes

{% diagram "registry" %}

The registry holds decisions; the task branch holds progress. Every state change is one `state(<ID>): …` commit on `workflow-state`. Once the task enters `implementation`, the plan steps exist only as commits on `feature/<ID>`.

The registry lives on `refs/heads/workflow-state`, created empty by `solomonar init`: `state.json` holds only the schema version, and each task is one file, `tasks/<ID>.json`. It has no filesystem checkout. Only `solomonar task` commands write it, using Git compare-and-swap updates. A write changes only the files of the tasks it touches, but every writer still swaps the same ref.

A registry written by CLI 0.1.5 or earlier keeps every task in one `state.json` (schema 1). Commands reject it until you run [`solomonar task migrate`]({{ '/docs/cli/task/#solomonar-task-migrate' | url }}) once; the migration is one commit on top of the registry history.

A command reads the current tip, validates the record and required Git facts, and attempts an atomic ref update. If another writer wins, it re-reads and re-validates before retrying. The local ref is canonical; remote publication can warn without undoing a successful local write.

Never check out, hand-edit, force-push, or merge `workflow-state` into `main`. In a colocated Jujutsu repository, the framework's v0.1 registry contract is Git-only.

## What is stored and what is derived

| Stored in a task record | Read from Git or the forge |
| --- | --- |
| Status, stage, owner, reviewed revision, risk tier | Branch tip, current HEAD, the worktree Git lists |
| Base commit, worktree path relative to the main checkout, owner confirmation | Dirty paths, spec presence, merge ancestry |
| Blockers and next actor/action | Plan approval, committed and remaining plan steps |
| Last reported agent assignments | PR state and checks when `gh` is available |

The base is the commit the task branched from; it never changes, even after `main` is merged into the branch. The recorded worktree path must match what Git lists; after `git worktree move`, the owner records the new path with `task move --worktree`.

Agent assignments are last reported information, not proof of liveness. A review points to one committed candidate, not uncommitted changes.

## Plans and checkpoints

The work specification's line below the title is `Status: In review` until you approve its implementation plan, then `Status: Approved`. The plan lists the intended commits:

```markdown
## Implementation plan

- S-1: feat(W-001): store expenses
  The expense type and its storage; AC-1.
- S-2: feat(W-001): split an expense equally
  The equal split and its tests; AC-1.
```

The implementer commits each step as a checkpoint with the planned subject and a `Step: W-001/S-1` trailer. Those commits are the only progress record. `task status` reconciles the plan with the task branch's own commits since the base and reports committed and remaining steps, plus any commit that claims a step without matching it.

The CLI enforces two gates: implementation cannot start without the approved plan committed, and review cannot start until the candidate is the clean tip of the branch with every step committed or closed by an `Outcome:` line. A checkpoint is not proof of correctness; reviews still check the whole change on one candidate.

## A new session takes over

A session that stops suddenly loses only what it held in memory. Committed steps, the approved plan, and the registry record stay in Git; uncommitted files stay in the worktree, unverified. The record still names the stopped session as the confirmed owner, and nothing takes over automatically.

{% diagram "session-stop" %}

The CLI cannot see sessions. A new session takes a task through `task handoff` on your instruction, which clears the confirmation. It inspects `task status`, preserves uncommitted work, and runs `task confirm`; until then, stage changes are rejected. It then delegates only the remaining steps. Confirmation proves neither that the plan was approved nor that the earlier session stopped; make sure it did.

{% diagram "takeover" %}

The new session reads progress from Git, not from the earlier conversation, so it does not repeat committed steps.

## Blockers

Blocker kinds are `decision`, `dependency`, `check`, and `environment`. IDs such as `B-1` stay stable within the task and are never reused.

A decision blocker needs a scoped decision reference such as `W-001/D-1`. Setting a task to `blocked` requires an open blocker. Clearing the last blocker of a blocked task must also name the prior status; the CLI does not choose it for the owner.

## Closure and history

The `task done` command checks that the recorded revision is in a merge commit on main and that `spec_ref` resolves to the final task-spec commit. It then writes `done / complete` and keeps a compact summary, merge reference, final spec reference, base, and PR when recorded. The final spec reference keeps the approved plan, and the base with the merge bounds the task's checkpoint commits.

The full work specification leaves the current file tree but remains in reachable Git history. A regression after merge is a new related task; it does not reopen the original `done` record.
