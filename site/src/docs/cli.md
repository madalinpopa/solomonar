---
title: CLI commands and flags
navTitle: CLI commands
description: Reference every Solomonar CLI command available today, including init with extensions and the task commands, with global flags, dry runs, JSON output, and planned commands.
kind: Reference
group: Reference
order: 11
sources:
  - internal/cli/run.go
  - internal/cli/flags.go
  - internal/cli/help.go
  - docs/cli/interface.md
---

The CLI is a guard and an oracle. It validates and records decisions and reports Git facts. The orchestrator runs the workflow; the binary never launches agents or calls model providers.

## Current commands

These commands are present in the current implementation. Use `solomonar help` from your build to confirm its command table.

| Command | Purpose | Requires a project |
| --- | --- | --- |
| `help` | Print help; `-h` and `--help` are equivalent. | No |
| `version` | Print CLI, embedded framework, and project versions. | No |
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

## task status

```text
solomonar task status <ID> [--json] [--no-forge]
```

Read-only. It prints the stored record, the Git facts (branch, worktree, HEAD, dirty files, merge state), the pull request and checks from `gh` when available, any drift between them, and the commands valid next. It exits 0 whatever the drift. `--no-forge` skips `gh`; unknown forge facts never count as passing.

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

For exhaustive arguments, validation tables, JSON data shapes, rejections, and use cases, read [the version-controlled interface specification]({{ 'docs/cli/interface.md' | source }}). It contains planned examples as well as implemented behavior; check this page's availability table before running a command.
