---
title: Task state and workflow tracking
description: Understand Solomonar task status, workflow stages, blockers, and Git-backed state validation so AI coding sessions can resume from recorded evidence.
kind: Reference
group: Use the CLI
order: 7
sources:
  - assets/framework/rules.md
  - docs/cli/interface.md
  - internal/task/transition.go
---

## Status and stage

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

A task is `reserved` exactly while at `setup`. On leaving setup, it becomes `running`. A blocked or paused task resumes to `reserved` at setup and `running` at a later stage.

## One registry, checked writes

The registry is `state.json` stored on `refs/heads/workflow-state`. It has no filesystem checkout. Only `solomonar task` commands write it, using Git compare-and-swap updates.

A command reads the current tip, validates the record and required Git facts, and attempts an atomic ref update. If another writer wins, it re-reads and re-validates before retrying. The local ref is canonical; remote publication can warn without undoing a successful local write.

Never check out, hand-edit, force-push, or merge `workflow-state` into `main`. In a colocated Jujutsu repository, the framework's v0.1 registry contract is Git-only.

## What is stored and what is derived

| Stored in a task record | Read from Git or the forge |
| --- | --- |
| Status, stage, owner, reviewed revision, risk tier | Branch tip, worktree location, current HEAD |
| Blockers and next actor/action | Dirty paths, spec presence, merge ancestry |
| Last reported agent assignments | PR state and checks when `gh` is available |

Agent assignments are last reported information, not proof of liveness. A review points to one committed candidate, not uncommitted changes.

## Blockers

Blocker kinds are `decision`, `dependency`, `check`, and `environment`. IDs such as `B-1` stay stable within the task and are never reused.

A decision blocker needs a scoped decision reference such as `W-001/D-1`. Setting a task to `blocked` requires an open blocker. Clearing the last blocker of a blocked task must also name the prior status; the CLI does not choose it for the owner.

## Closure and history

The `task done` command checks that the recorded revision is in a merge commit on main and that `spec_ref` resolves to the final task-spec commit. It then writes `done / complete` and keeps a compact summary, merge reference, final spec reference, and PR when recorded.

The full work specification leaves the current file tree but remains in reachable Git history. A regression after merge is a new related task; it does not reopen the original `done` record.
