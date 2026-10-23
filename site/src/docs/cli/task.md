---
title: solomonar task
navTitle: solomonar task
description: Plan, register, inspect, move, block, hand off, confirm, migrate, and complete Solomonar tasks with the solomonar task commands, with real examples of each.
kind: Reference
group: Commands
order: 14
sources:
  - internal/cli/task.go
  - internal/cli/register.go
  - internal/cli/status.go
  - internal/cli/move.go
  - internal/cli/block.go
  - internal/cli/handoff.go
  - internal/cli/done.go
  - internal/cli/confirm.go
  - internal/cli/migrate.go
  - docs/cli/interface.md
---

The `task` commands read and write the task registry on the branch `refs/heads/workflow-state`: `state.json` with the schema version and one `tasks/<ID>.json` per task. They are the only way the registry is written: each write is validated against the record and Git, and becomes one registry commit. They need a project prepared with [`init`]({{ '/docs/cli/init/' | url }}).

The examples on this page come from runs of the current build in the fictional `ledger` project. They follow task `W-001` from plan to `done`, including a handoff to a new session. Paths are shortened to `/work/ledger`; the SHAs and timestamps are those the runs produced, so they match within each section.

## Conventions

Task IDs have the form `W-NNN`. Every writing task command accepts `--json` and `--dry-run`, and a successful write is one commit on the registry branch. When the project has a remote, the CLI pushes that commit and prints `pushed origin/workflow-state`; without one it warns that the registry stays local.

| Value | Format |
| --- | --- |
| `--as <owner>` | Acting owner, an agent ID such as `orchestrator-1`. Defaults to `SOLOMONAR_OWNER`; a command that needs an owner and has neither fails with `owner_required`. |
| `--next <actor>:<action>` | Next action. The actor is `user`, the owner, or an active agent ID; the action is 1 to 120 characters. |
| `<full-sha>` | A full 40-character hexadecimal commit SHA; short SHAs are rejected. |

The [task state page]({{ '/docs/task-state/' | url }}) explains the statuses and stages.

## solomonar task plan

Adds a `planned` task under the next free `W-NNN`. A planned task has no owner, stage, or branch yet.

{% cliUsage "To queue work you want done later, or to record the order of tasks with `--depends-on` before any orchestrator picks them up.", "You. An orchestrator does not plan tasks; it claims one with `task register`." %}

```text
solomonar task plan --title <title> [--depends-on <ID>]... [--related <ID>] [--json] [--dry-run]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `--title` | Required. The task title; it must not be blank. |
| `--depends-on` | A task this one depends on; repeatable. It must exist. |
| `--related` | One related task, such as the `done` task this one follows up. It must exist. |

### Example

```sh
solomonar task plan --title "Add a shared expense"
solomonar task plan --title "List expenses by month" --depends-on W-001
```

```text
Planned W-001: Add a shared expense
  state(W-001): planned, plan "Add a shared expense"
  registry @ abbc81a0e1f080e6e8bffbc456851c5859cdac95 · pushed origin/workflow-state
```

With `--json --dry-run`, the command runs every check and reports the record it would write:

```sh
solomonar task plan --title "Export to CSV" --json --dry-run
```

```json
{
  "ok": true,
  "command": "task plan",
  "data": {
    "id": "W-003",
    "record": {
      "title": "Export to CSV",
      "status": "planned",
      "updated_at": "2026-10-22T17:21:04Z"
    },
    "dry_run": true
  },
  "warnings": []
}
```

## solomonar task register

Reserves the next free `W-NNN`, or claims the planned `<ID>`, for one orchestrator. The task starts `reserved` at stage `setup`, on branch `feature/<ID>`. This is the registration barrier: the next task may start only after the registration is confirmed.

{% cliUsage "At the start of every task, before the branch and worktree exist.", "The orchestrator agent, when you ask a session for an outcome. You see its result as Registration confirmed." %}

```text
solomonar task register [<ID>] --title <title> --owner <owner> [--depends-on <ID>]... [--related <ID>]
                        [--next <actor>:<action>] [--json] [--dry-run]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<ID>` | Optional. A `planned` task to claim; any other status is `not_planned`. |
| `--title` | Required. On a claim, a different title replaces the planned one with the warning `title_changed`. |
| `--owner` | Required. The orchestrator's agent ID. It must not own another `reserved`, `running`, `blocked`, or `paused` task (`owner_in_use`). |
| `--depends-on`, `--related` | As for `task plan`. On a claim, they replace the planned values when given. A dependency that is not `done` is the warning `dependency_not_done`. |
| `--next` | Initial next action; the actor is `user` or the owner. |

### Example

Claim the planned `W-001`:

```sh
solomonar task register W-001 --title "Add a shared expense" --owner orchestrator-1
```

```text
Registered W-001: Add a shared expense
  status    reserved / setup
  owner     orchestrator-1
  branch    feature/W-001 (create it now)
  next      orchestrator-1: Create feature/W-001 and its worktree
  state(W-001): reserved/setup, registered by orchestrator-1
  registry @ b0260cd4435162e74248f15f290c90fb46b609e8 · pushed origin/workflow-state

Registration confirmed. The next task may start.
Create the worktree:
  git worktree add -b feature/W-001 ../ledger-worktrees/W-001 main
```

## solomonar task show

Read-only. Without an ID it prints every task sorted by ID, with its status, stage, owner, and next action, and the registry commit. With an ID it prints the stored record, the content of `tasks/<ID>.json`, exactly as JSON.

{% cliUsage "To see at a glance what is running, blocked, or waiting on you, or to read one record as the agents see it.", "You, most often. Agents read it too when they resume." %}

```text
solomonar task show [<ID>] [--json]
```

### Example

The registry overview while `W-001` is registered:

```sh
solomonar task show
```

```text
ID     STATUS     STAGE  OWNER           NEXT
W-001  reserved   setup  orchestrator-1  orchestrator-1: Create feature/W-001 and its worktree
W-002  planned    -      -               -
registry refs/heads/workflow-state @ a2b3473 · 2026-10-22T17:21:04Z
```

One record, while `W-001` waits on a decision:

```sh
solomonar task show W-001
```

```json
{
  "title": "Add a shared expense",
  "status": "blocked",
  "stage": "implementation",
  "owner": "orchestrator-1",
  "confirmed": true,
  "branch": "feature/W-001",
  "base": "ac06d8df21b7c78152b780f19e0a0074c0ff8496",
  "worktree": "../ledger-worktrees/W-001",
  "spec": "docs/work/W-001-add-shared-expense.md",
  "active_agents": [
    {
      "id": "impl-1",
      "role": "implementer"
    }
  ],
  "blockers": [
    {
      "id": "B-1",
      "kind": "decision",
      "summary": "Choose local storage for expenses",
      "ref": "W-001/D-1"
    }
  ],
  "last_blocker": 1,
  "next_action": {
    "actor": "user",
    "action": "Approve a storage option for W-001/D-1"
  },
  "updated_at": "2026-10-22T17:21:20Z"
}
```

## solomonar task status

Read-only. It prints the stored record, the Git facts (branch, worktree, HEAD, dirty files, spec, merge state), the pull request and checks from `gh` when available, any drift between them, and the commands valid next. It exits 0 whatever the drift.

{% cliUsage "To check that a task's record matches what Git shows, and when a session ended unexpectedly: the drift and the allowed commands tell the next session where to continue.", "You and agents. An orchestrator runs it before resuming a task." %}

```text
solomonar task status <ID> [--json] [--no-forge]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `--no-forge` | Skip `gh`. Unknown forge facts never count as passing. |

`SOLOMONAR_OWNER`, when set, lets the command report a task owned by another orchestrator as `owned_by_other`.

### Example

A new session has taken the task with `task handoff` and has not confirmed it yet. Step `S-1` is committed, and the implementer left `S-2` uncommitted:

```sh
solomonar task status W-001 --no-forge
```

```text
W-001  Add a shared expense
record
  status    running / implementation   owner orchestrator-2 (unconfirmed)   tier none
  base      ac06d8df21b7c78152b780f19e0a0074c0ff8496
  worktree  ../ledger-worktrees/W-001
  spec      docs/work/W-001-add-shared-expense.md
  revision  none
  agents    impl-1 implementer (last reported, not liveness)
  blockers  none
  next      impl-1: Implement S-1 and S-2
  pr        none
  updated   2026-10-23T16:28:20Z (registry @ 2a80590)
git
  branch    feature/W-001 @ b4c551b27d78956c448001987212fd088e3cf5f6
  worktree  /work/ledger-worktrees/W-001
  HEAD      b4c551b2
  dirty     1 path: expense/split.go (untracked)
  spec      present in worktree only
  merged    no
plan
  approval  approved at b4c551b2
  steps     1 of 2 committed; remaining S-2
  other     2 commits without a Step trailer
forge (gh)
  pr      unknown (--no-forge)
  checks  unknown
drift
  unconfirmed     W-001 is not confirmed by orchestrator-2; inspect, then task confirm
  dirty_worktree  1 uncommitted path; preserve it (rules section 10)
allowed next
  solomonar task confirm W-001 --as orchestrator-2
  solomonar task block W-001 --as orchestrator-2 --kind <kind> --summary <text>
  solomonar task move W-001 --as orchestrator-2 --status paused --next <actor>:<action>
  solomonar task move W-001 --as orchestrator-2 --status cancelled --next <actor>:<action>
```

The `plan` section is the recovery point: `S-1` needs no repetition, `S-2` remains, and the untracked file is partial work to inspect before delegating it. The two other commits are the work spec's draft and approval. A commit that claims a step without matching it appears under `plan` with its code, such as `step_untagged` or `step_subject_mismatch`, and blocks review until resolved. After a merge, `drift` reports `merged_not_done` and `allowed next` lists only `task done`, with the merge commit filled in.

## solomonar task move

Applies every given field flag as one validated update and one registry commit. At least one field flag is required. Only the owner may move a task. `done` and `complete` are written only by `task done`.

{% cliUsage "At each step of a task: starting the specification, entering implementation with the recorded spec, recording the reviewed revision and tier, changing the delegated agents, recording the pull request, pausing, or cancelling.", "The orchestrator that owns the task. You ask for a pause or a cancellation in the session; the orchestrator runs the command." %}

```text
solomonar task move <ID> [--as <owner>]
  [--status <reserved|running|blocked|paused|cancelled>] [--resume-cancelled]
  [--stage <setup|specification|implementation|specialized_review|adversarial_review|integration>]
  [--revision <full-sha>] [--spec <worktree-relative path>] [--tier <0|1|2>]
  [--agent <id>:<role>]... | [--agent none]
  [--pr <url|number>] [--next <actor>:<action>]
  [--base <full-sha>] [--worktree <path relative to the main checkout>]
  [--json] [--dry-run]
```

Leaving `setup` records the task's base commit, its worktree path relative to the main checkout, and the owner's confirmation. Every later stage change, and `task done`, needs a confirmed task (see [`task confirm`](#solomonar-task-confirm)); status-only moves do not.

### Flags

| Flag | Meaning |
| --- | --- |
| `--status` | New status. `running` leaves `setup` together with `--stage specification`; `blocked` needs an open blocker; `paused` and `cancelled` need `--next`. From `blocked` or `paused`, only the prior status or `cancelled` is allowed. |
| `--resume-cancelled` | Required to return a `cancelled` task to its prior status, on the user's explicit instruction. |
| `--stage` | New stage. Forward moves follow the tier: tier 0 goes from `implementation` to `integration`, tier 1 adds `specialized_review`, tier 2 adds `adversarial_review`. Some backward moves are allowed for rework. A `blocked` or `paused` task keeps its stage. |
| `--revision` | The reviewed candidate commit on `feature/<ID>`. Accepted when the current or resulting stage is `implementation` or `integration`. |
| `--spec` | The work specification, a clean path relative to the worktree. Accepted from `specification` on; entering `implementation` needs it. |
| `--tier` | Review tier `0`, `1`, or `2`. Stages after `implementation` need it, passed with `--revision`. |
| `--agent` | An active delegated agent as `<id>:<role>`, such as `impl-1:implementer`; repeatable, replacing the list. `--agent none` clears it. The role is a core delegated role or an installed agent extension, never `orchestrator` or a draft. |
| `--pr` | The pull request, as a positive number or an `https://` URL. Accepted in `integration`. |
| `--next` | New next action. An agent added by `--agent` in the same command may be the actor. |
| `--base` | Fills a missing base, such as after a migration. It must be on `feature/<ID>`; a recorded base never changes. |
| `--worktree` | Records a relocation after `git worktree move`. It must be the path Git lists for `feature/<ID>`. |

Two moves are gated by the work specification's plan. Entering `implementation` from `specification` needs `Status: Approved` and at least one plan step in the spec committed on `feature/<ID>`. Leaving `implementation` for a review stage or `integration` needs the candidate to be the clean tip of the branch, every plan step committed with its `Step:` trailer or closed by an outcome, and no commit claiming a step it does not match.

### Example

Start the specification, then enter implementation with the committed work spec and a delegated implementer:

```sh
solomonar task move W-001 --as orchestrator-1 --status running --stage specification \
  --next "orchestrator-1:Write the work spec"
solomonar task move W-001 --as orchestrator-1 --stage implementation \
  --spec docs/work/W-001-add-shared-expense.md --agent impl-1:implementer \
  --next "impl-1:Implement the add-expense flow"
```

```text
W-001  reserved/setup → running/specification
  confirmed  yes
  base       ac06d8df21b7c78152b780f19e0a0074c0ff8496 (on feature/W-001)
  worktree   ../ledger-worktrees/W-001 (listed by git worktree list)
  next       orchestrator-1: Write the work spec
  state(W-001): running/specification, status running, stage specification, confirmed, base ac06d8d, worktree ../ledger-worktrees/W-001, next orchestrator-1
  registry @ 7059a0eca533755a40da5184dd241bc4d3702c47 · pushed origin/workflow-state
```

Before the plan is approved, implementation cannot start:

```text
error: task move W-001 rejected (plan_not_approved)
  check: the work spec committed on feature/W-001 has an approved implementation plan
  found: status is "In review" at 24455a8b
  fix:   record the user's approval as Status: Approved below the title and commit the work spec
nothing was written
```

After the approval is committed:

```text
W-001  running/specification → running/implementation
  spec    docs/work/W-001-add-shared-expense.md (found in /work/ledger-worktrees/W-001)
  agents  impl-1 implementer
  next    impl-1: Implement the add-expense flow
  state(W-001): running/implementation, stage implementation, spec docs/work/W-001-add-shared-expense.md, agents impl-1, next impl-1
  registry @ 92a4704fdf1d8fc2f06ac039ce85948eeecac8c2 · pushed origin/workflow-state
```

A move the record cannot support is rejected, exits 3, and writes nothing:

```sh
solomonar task move W-001 --as orchestrator-1 --stage integration
```

```text
error: task move W-001 rejected (tier_required)
  check: a stage after implementation needs a recorded tier
  found: implementation -> integration
  fix:   triage the candidate and pass --revision <full-sha> --tier <0|1|2> with the stage move
nothing was written
```

With the reviewed revision and its tier, the same move succeeds:

```sh
solomonar task move W-001 --as orchestrator-1 --stage integration \
  --revision c2eb0409c46f23bf8c89c856b1ac46f175f3bdf0 --tier 0 --agent none \
  --next "orchestrator-1:Open the pull request"
```

```text
W-001  running/implementation → running/integration
  revision  c2eb0409c46f23bf8c89c856b1ac46f175f3bdf0 (on feature/W-001)
  tier      0: no review stages
  agents    none
  next      orchestrator-1: Open the pull request
  state(W-001): running/integration, stage integration, revision c2eb040, tier 0, agents none, next orchestrator-1
  registry @ b6a9d5b2dcb8b2ecdc60f056f5276024d31f8424 · pushed origin/workflow-state
```

## solomonar task block

Adds an open blocker `B-n` to the task. `--status` and `--next` are shorthands for the same `task move` fields, so one blocking event is one registry commit.

{% cliUsage "When part of the work cannot continue: a decision only you can make, a task it depends on, a failing check, or a broken environment.", "The orchestrator that owns the task. For a decision, it then asks you in the session, quoting the decision ID." %}

```text
solomonar task block <ID> --kind <decision|dependency|check|environment> --summary <text> [--ref <ref>]
                     [--next <actor>:<action>] [--status blocked] [--as <owner>] [--json] [--dry-run]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `--kind` | Required. `decision`, `dependency`, `check`, or `environment`. |
| `--summary` | Required. What blocks the task. |
| `--ref` | A reference for the blocker. Required with `--kind decision`, as `<ID>/D-n`. |
| `--status blocked` | Also set the status to `blocked`. |
| `--next` | Also set the next action, typically `user:<decision to make>`. |

### Example

Stop for a storage decision:

```sh
solomonar task block W-001 --as orchestrator-1 --kind decision --ref W-001/D-1 \
  --summary "Choose local storage for expenses" --status blocked \
  --next "user:Approve a storage option for W-001/D-1"
```

```text
W-001  running/implementation → blocked/implementation
  blocker  B-1 decision: Choose local storage for expenses (W-001/D-1)
  next     user: Approve a storage option for W-001/D-1
  state(W-001): blocked/implementation, block B-1 decision
  registry @ 823582e8c0085952f9fbf2777fb4bbb901dc0be6 · pushed origin/workflow-state
```

## solomonar task unblock

Clears one open blocker, given by its ID such as `B-1`.

{% cliUsage "When the blocking cause is resolved: you answered the decision, the dependency is done, or the check or environment works again.", "The orchestrator that owns the task, after it receives your answer." %}

```text
solomonar task unblock <ID> <blocker-id> [--status <prior status>] [--next <actor>:<action>]
                       [--as <owner>] [--json] [--dry-run]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<blocker-id>` | Required. An open blocker of the task; anything else is `blocker_not_found`. |
| `--status` | The prior status to resume: `reserved` at stage `setup`, `running` at any later stage. Required when the last blocker of a `blocked` task is cleared. |
| `--next` | Also set the next action. |

### Example

You approved SQLite for `W-001/D-1`:

```sh
solomonar task unblock W-001 B-1 --as orchestrator-1 --status running \
  --next "impl-1:Implement the add-expense flow with SQLite"
```

```text
W-001  blocked/implementation → running/implementation
  cleared  B-1 decision: Choose local storage for expenses
  next     impl-1: Implement the add-expense flow with SQLite
  state(W-001): running/implementation, unblock B-1
  registry @ 42d70e8472adf2b6e9999cfc381cd4393be5ca2d · pushed origin/workflow-state
```

## solomonar task handoff

Transfers ownership to a new orchestrator. It takes no `--as` and ignores `SOLOMONAR_OWNER`; the previous owner must stop writing the record. Every status except `planned` and `done` accepts a handoff.

{% cliUsage "When the session that owns a task is gone or must stop, and another session takes over the same task.", "The new orchestrator, only on your explicit instruction. Agents never take over a task on their own." %}

```text
solomonar task handoff <ID> --owner <new-owner> [--next <actor>:<action>] [--json] [--dry-run]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `--owner` | Required. The new owner's agent ID. It must differ from the current owner and must not own another active task. |
| `--next` | New next action. Without it, a next action of the previous owner moves to the new owner with the same text. |

### Example

```sh
solomonar task handoff W-001 --owner orchestrator-2
```

```text
W-001  owner orchestrator-1 → orchestrator-2 (running/implementation)
  confirmed  no
  state(W-001): running/implementation, handoff orchestrator-1 -> orchestrator-2
  registry @ 2a80590bb8c4304cb9219be0f45e313dad80184f · pushed origin/workflow-state
  The previous owner must stop writing this record.
  Inspect with solomonar task status W-001, then solomonar task confirm W-001 --as orchestrator-2.
```

The handoff clears the confirmation. Until the new owner runs [`task confirm`](#solomonar-task-confirm), its stage changes and `task done` are rejected with `confirmation_required`.

## solomonar task confirm

Records that the owner inspected the task and continues it from its actual state. It checks the branch, the recorded worktree against Git's listing, the recorded base, and, from `implementation` until `integration`, the approved plan at the branch tip. An already confirmed task is unchanged.

{% cliUsage "After a handoff or a migration, once the new session has read `task status` and preserved any partial work. Leaving `setup` confirms a task, so a task that never changed hands needs no confirmation.", "The orchestrator that owns the task. Confirmation proves neither plan approval, review success, nor that the earlier session stopped; make sure it did." %}

```text
solomonar task confirm <ID> [--as <owner>] [--json] [--dry-run]
```

### Example

```sh
solomonar task confirm W-001 --as orchestrator-2
```

```text
W-001  running/implementation
  confirmed  yes
  state(W-001): running/implementation, confirmed by orchestrator-2
  registry @ 24b1064cc13acb6c0251168ba4f7c578d697c3f0 · pushed origin/workflow-state
```

A rejection names the missing evidence: `worktree_mismatch` when the worktree moved (record it with `task move --worktree`), `base_missing` after a migration (record it with `task move --base`), or `plan_not_approved` when the committed plan is no longer approved.

## solomonar task migrate

Rewrites a registry written by CLI 0.1.6 or earlier, one `state.json` with every task (schema 1), as one file per task (schema 2). The migration is one commit on top of the registry history. It infers nothing: active tasks stay unconfirmed and keep no base or worktree until their owners record them.

{% cliUsage "Once per repository, when any command reports `registry_outdated`. Run it after upgrading every machine that writes the registry: CLI 0.1.6 rejects the migrated registry.", "You. It takes no `--as`." %}

```text
solomonar task migrate [--json]
```

### Example

```sh
solomonar task show
solomonar task migrate
```

```text
error: task show rejected (registry_outdated)
  check: registry schema version
  found: registry schema 1, this CLI uses 2
  fix:   solomonar task migrate
nothing was written
Migrated registry to schema 2: 2 tasks, one tasks/<ID>.json each
  registry @ 21edeae27613f9e3ab8600f34cce9cf0a89e35a9
```

When active tasks are past `setup`, the output lists them as `unconfirmed` with the next step: inspect each with `task status`, then `task confirm`. A registry already at schema 2 prints `Registry already at schema 2; nothing to migrate.` To undo a migration, point the ref back to its parent with `git update-ref refs/heads/workflow-state refs/heads/workflow-state^`; writes made after the migration are lost.

## solomonar task done

Completes a confirmed `running` task at stage `integration` with no open blocker, after its pull request merged. It verifies the facts with Git only, then replaces the record with the compact `done` record.

{% cliUsage "Right after the pull request is merged with a merge commit. `task status` lists the exact command, with the merge commit filled in.", "The orchestrator that owns the task. You read the result with `task show`." %}

```text
solomonar task done <ID> --merge <full-sha> --spec-ref <full-sha>:<path> --summary <text>
                    [--as <owner>] [--json] [--dry-run]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `--merge` | Required. The merge commit: reachable from the local main branch, with two or more parents, and containing the recorded `revision`. A squash or rebase merge is rejected. |
| `--spec-ref` | Required. `<full-sha>:<path>` of the final specification: the last `spec(<ID>):` commit of the merged branch, with the path equal to the recorded `spec`. Copy it from the pull request's "Final specification" line. |
| `--summary` | Required. A one-line summary of 1 to 120 characters. |

### Example

```sh
solomonar task done W-001 --as orchestrator-2 \
  --merge 3b6f524bb592fed2488cae041a0321320218b3b6 \
  --spec-ref 630056c7a51878ee3b978a58d4ade9d71d6e230d:docs/work/W-001-add-shared-expense.md \
  --summary "Add shared expenses split equally"
```

```text
Verified
  merge     3b6f524b is a merge commit reachable from main
  revision  d0751e9b is contained in the merge
  spec_ref  resolves; 630056c7 is the last spec(W-001) commit merged
W-001  running/integration → done/complete
  state(W-001): done/complete, merged 3b6f524
  registry @ 8457273c982a762cfcf7cf12c6b663354ca6c914 · pushed origin/workflow-state
Compact record
  {
    "title": "Add a shared expense",
    "status": "done",
    "stage": "complete",
    "base": "ac06d8df21b7c78152b780f19e0a0074c0ff8496",
    "pr": "12",
    "summary": "Add shared expenses split equally",
    "spec_ref": "630056c7a51878ee3b978a58d4ade9d71d6e230d:docs/work/W-001-add-shared-expense.md",
    "merge_commit": "3b6f524bb592fed2488cae041a0321320218b3b6",
    "updated_at": "2026-10-23T16:29:29Z"
  }
Next: delete the worktree only after confirming it holds no uncommitted or unmerged work:
  git worktree remove ../ledger-worktrees/W-001
```

The registry overview now shows the summary in place of a next action:

```text
ID     STATUS     STAGE     OWNER  NEXT
W-001  done       complete  -      Add shared expenses split equally
W-002  planned    -         -      -
registry refs/heads/workflow-state @ 8457273 · 2026-10-23T16:29:29Z
```
