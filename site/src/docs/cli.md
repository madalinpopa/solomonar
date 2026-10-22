---
title: CLI commands and flags
navTitle: CLI commands
description: Reference every Solomonar CLI command available today, with the syntax and flags of init, version, and each task command, plus global flags, dry runs, JSON output, and planned commands.
kind: Reference
group: Reference
order: 11
sources:
  - internal/cli/run.go
  - internal/cli/flags.go
  - internal/cli/help.go
  - internal/cli/version.go
  - internal/cli/task.go
  - internal/cli/register.go
  - internal/cli/status.go
  - internal/cli/move.go
  - internal/cli/block.go
  - internal/cli/handoff.go
  - internal/cli/done.go
  - docs/cli/interface.md
---

The CLI is a guard and an oracle. It validates and records decisions and reports Git facts. The orchestrator runs the workflow; the binary never launches agents or calls model providers.

## Current commands

These commands are present in the current implementation. Use `solomonar help` from your build to confirm its command table.

| Command | Purpose | Requires a project |
| --- | --- | --- |
| `help` | Print help; `-h` and `--help` are equivalent. | No |
| `version` | Print the CLI and project framework versions. | No |
| `init` | Install the framework core, runtime adapters, and extensions, and create the registry. | No |
| `task plan` | Create a planned task with the next free ID. | Yes |
| `task register` | Reserve a task for one orchestrator. | Yes |
| `task show` | Read the stored registry or one task. | Yes |
| `task status` | Compare one task's record with Git and forge facts; list drift and valid next commands. | Yes |
| `task move` | Validate a status, stage, or field update. | Yes |
| `task block` | Add a blocker, optionally updating status and next action atomically. | Yes |
| `task unblock` | Clear an open blocker, optionally resuming the prior status. | Yes |
| `task handoff` | Transfer ownership on explicit user instruction. | Yes |
| `task done` | Verify the merge and final spec reference, then write the compact record. | Yes |

Task commands need a project prepared with `init`. Agents run the writing task commands; you mostly use `task plan`, `task show`, and `task status`.

## init

```text
solomonar init [<dir>] [--name <name>] [--objective <text>]
               [--runtime <claude|codex|none>]... [--ext <name|none>]...
               [--adversarial-model <runtime>=<model>]...
               [--main-branch <name>] [--remote <name>]
               [--keep <path>]... [--overwrite <path>]...
               [--git-init] [-y|--yes] [--dry-run] [--json]
```

| Flag | Meaning |
| --- | --- |
| `--name` | Project name; defaults to the directory name. |
| `--objective` | Current objective written to `docs/README.md`. |
| `--runtime` | `claude`, `codex`, or `none`; repeatable or comma-separated. Defaults to the runtimes found on `PATH`. |
| `--ext` | Extensions to install, repeatable or comma-separated, or `none`. The catalog holds `go`. Defaults to `go` when the repository root has a `go.mod`, otherwise none. |
| `--adversarial-model` | Model for the adversarial reviewer of one runtime, such as `claude=<model>`. |
| `--main-branch`, `--remote` | Defaults: `main` and `origin`. |
| `--keep`, `--overwrite` | Resolve one conflicting file; one path per flag. |
| `--git-init` | Run `git init -b <main-branch>` when the directory is not a repository. |
| `-y`, `--yes` | Accept the plan without prompts. Required without a terminal. |
| `--dry-run` | Print the plan and write nothing. |

In a terminal, without `--yes` or `--json`, `init` asks for each choice whose flag you did not pass, shows the plan, and asks `Proceed?` before it writes. Each choice of a stack extension is recorded as your approved decision in `docs/decisions.md` and `docs/stack.md`.

A re-run with the same choices and unchanged files is a no-op. A re-run never changes a recorded choice. See [Set up a project]({{ '/docs/set-up-a-project/' | url }}) for a walkthrough.

## version

```text
solomonar version
```

Prints the CLI version, which is also the version of the framework it installs, and the framework version recorded in the project. Outside a project the project line reads `not initialized`.

## Task command conventions

Task IDs have the form `W-NNN`. Every writing task command accepts `--json` and `--dry-run`, and a successful write is one commit on the registry branch.

| Value | Format |
| --- | --- |
| `--as <owner>` | Acting owner, an agent ID such as `orchestrator-1`. Defaults to `SOLOMONAR_OWNER`; a command that needs an owner and has neither fails with `owner_required`. |
| `--next <actor>:<action>` | Next action. The actor is `user`, the owner, or an active agent ID; the action is 1 to 120 characters. |
| `<full-sha>` | A full 40-character hexadecimal commit SHA; short SHAs are rejected. |

## task plan

```text
solomonar task plan --title <title> [--depends-on <ID>]... [--related <ID>] [--json] [--dry-run]
```

Adds a `planned` task under the next free `W-NNN`.

| Flag | Meaning |
| --- | --- |
| `--title` | Required. The task title; it must not be blank. |
| `--depends-on` | A task this one depends on; repeatable. It must exist. |
| `--related` | One related task, such as the `done` task this one follows up. It must exist. |

## task register

```text
solomonar task register [<ID>] --title <title> --owner <owner> [--depends-on <ID>]... [--related <ID>]
                        [--next <actor>:<action>] [--json] [--dry-run]
```

Reserves the next free `W-NNN`, or claims the planned `<ID>`, for one orchestrator. The task starts `reserved` at stage `setup`. The next task may start only after the registration is confirmed.

| Flag | Meaning |
| --- | --- |
| `<ID>` | Optional. A `planned` task to claim; any other status is `not_planned`. |
| `--title` | Required. On a claim, a different title replaces the planned one with the warning `title_changed`. |
| `--owner` | Required. The orchestrator's agent ID. It must not own another `reserved`, `running`, `blocked`, or `paused` task (`owner_in_use`). |
| `--depends-on`, `--related` | As for `task plan`. On a claim, they replace the planned values when given. A dependency that is not `done` is the warning `dependency_not_done`. |
| `--next` | Initial next action; the actor is `user` or the owner. |

## task show

```text
solomonar task show [<ID>] [--json]
```

Read-only. Without an ID it prints every task sorted by ID and the registry commit. With an ID it prints the stored record exactly as JSON.

## task status

```text
solomonar task status <ID> [--json] [--no-forge]
```

Read-only. It prints the stored record, the Git facts (branch, worktree, HEAD, dirty files, merge state), the pull request and checks from `gh` when available, any drift between them, and the commands valid next. It exits 0 whatever the drift. `--no-forge` skips `gh`; unknown forge facts never count as passing.

## task move

```text
solomonar task move <ID> [--as <owner>]
  [--status <reserved|running|blocked|paused|cancelled>] [--resume-cancelled]
  [--stage <setup|specification|implementation|specialized_review|adversarial_review|integration>]
  [--revision <full-sha>] [--spec <worktree-relative path>] [--tier <0|1|2>]
  [--agent <id>:<role>]... | [--agent none]
  [--pr <url|number>] [--next <actor>:<action>]
  [--json] [--dry-run]
```

Applies every given field flag as one validated update. At least one field flag is required. Only the owner may move a task. `done` and `complete` are written only by `task done`.

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

The [task state page]({{ '/docs/task-state/' | url }}) explains the statuses and stages.

## task block

```text
solomonar task block <ID> --kind <decision|dependency|check|environment> --summary <text> [--ref <ref>]
                     [--next <actor>:<action>] [--status blocked] [--as <owner>] [--json] [--dry-run]
```

Adds an open blocker `B-n` to the task. `--status` and `--next` are shorthands for the same `task move` fields, so one blocking event is one registry commit.

| Flag | Meaning |
| --- | --- |
| `--kind` | Required. `decision`, `dependency`, `check`, or `environment`. |
| `--summary` | Required. What blocks the task. |
| `--ref` | A reference for the blocker. Required with `--kind decision`, as `<ID>/D-n`. |
| `--status blocked` | Also set the status to `blocked`. |
| `--next` | Also set the next action, typically `user:<decision to make>`. |

## task unblock

```text
solomonar task unblock <ID> <blocker-id> [--status <prior status>] [--next <actor>:<action>]
                       [--as <owner>] [--json] [--dry-run]
```

Clears one open blocker, given by its ID such as `B-1`.

| Flag | Meaning |
| --- | --- |
| `<blocker-id>` | Required. An open blocker of the task; anything else is `blocker_not_found`. |
| `--status` | The prior status to resume: `reserved` at stage `setup`, `running` at any later stage. Required when the last blocker of a `blocked` task is cleared. |
| `--next` | Also set the next action. |

## task handoff

```text
solomonar task handoff <ID> --owner <new-owner> [--next <actor>:<action>] [--json] [--dry-run]
```

Transfers ownership to a new orchestrator, on the user's instruction. It takes no `--as` and ignores `SOLOMONAR_OWNER`; the previous owner must stop writing the record. Every status except `planned` and `done` accepts a handoff.

| Flag | Meaning |
| --- | --- |
| `--owner` | Required. The new owner's agent ID. It must differ from the current owner and must not own another active task. |
| `--next` | New next action. Without it, a next action of the previous owner moves to the new owner with the same text. |

## task done

```text
solomonar task done <ID> --merge <full-sha> --spec-ref <full-sha>:<path> --summary <text>
                    [--as <owner>] [--json] [--dry-run]
```

Completes a `running` task at stage `integration` with no open blocker, after its pull request merged. It verifies the facts with Git only, then replaces the record with the compact `done` record.

| Flag | Meaning |
| --- | --- |
| `--merge` | Required. The merge commit: reachable from the local main branch, with two or more parents, and containing the recorded `revision`. A squash or rebase merge is rejected. |
| `--spec-ref` | Required. `<full-sha>:<path>` of the final specification: the last `spec(<ID>):` commit of the merged branch, with the path equal to the recorded `spec`. Copy it from the pull request's "Final specification" line. |
| `--summary` | Required. A one-line summary of 1 to 120 characters. |

## Invocation and help

```sh
solomonar help
solomonar task move --help
solomonar help task block
```

`solomonar` here means a binary on your `PATH`; see [Install the CLI]({{ '/docs/install/' | url }}).

Command-specific help gives the exact accepted arguments. A missing command, a bare command group such as `task`, or an unknown command exits with a usage error.

## Global flags

| Flag | Meaning |
| --- | --- |
| `-C <path>` | Run in another directory, like `git -C`. |
| `--json` | Print machine-readable output on stdout. |
| `--no-color` | Disable color; equivalent to setting `NO_COLOR`. |
| `-v`, `--verbose` | Trace external Git and `gh` commands to stderr. |
| `-h`, `--help` | Show command help. |

Writing task commands support `--dry-run`: they run validation and report the proposed record without writing it. Owner commands accept `--as <owner>`, with `SOLOMONAR_OWNER` as the default.

## Output formats

Human-readable results go to stdout. Warnings and errors go to stderr. With `--json`, stdout contains exactly one JSON object.

A successful command follows this envelope; the `data` fields depend on the command:

```json
{
  "ok": true,
  "command": "help",
  "data": {
    "help": "..."
  },
  "warnings": []
}
```

The shortened help string above illustrates the envelope. Run this working command for the full output:

```sh
solomonar help --json
```

On failure, `ok` is `false`, `error` replaces `data`, and a rejection identifies the failed check, what was found, and how to correct it. Unknown forge or check facts never count as passing.

## Planned commands

These are interface contracts, not commands available in the current build.

| Delivery increment | Planned |
| --- | --- |
| Increment 2 | `doctor`, `sync`, `runtime list/add/remove`, `extension list/show/add/remove/new` (alias `ext`), project extensions from a local path, `guard` and its runtime hooks |
| Increment 3 | `upgrade`, `dashboard` |

The `htmx`, `templ`, and `tdd` extensions of the interface catalog are added once their framework content is written.

The read-only dashboard is a planned local view of the registry, not this documentation website. Windows, model calls, a daemon, and independently writable registry clones are outside the v0.1 scope.

## Full interface contract

This page covers every flag of the current commands. For the full validation tables, JSON data shapes, error codes, and use cases, read [the version-controlled interface specification]({{ 'docs/cli/interface.md' | source }}). It contains planned examples as well as implemented behavior; check this page's availability table before running a command.
