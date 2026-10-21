---
title: CLI commands and flags
description: Reference current Solomonar task commands, global flags, dry runs, and JSON output. See which CLI commands work today and which remain planned.
kind: Reference
group: Use the CLI
order: 6
sources:
  - internal/cli/run.go
  - internal/cli/flags.go
  - internal/cli/help.go
  - docs/cli/interface.md
---

The CLI is a guard and an oracle. It validates and records decisions and reports Git facts. The orchestrator runs the workflow; the binary never launches agents or calls model providers.

## Current commands

These commands are present in the current implementation. Use `solomonar help` from your build to confirm its command table.

| Command | Purpose | Requires a registry |
| --- | --- | --- |
| `help` | Print help; `-h` and `--help` are equivalent. | No |
| `version` | Print CLI, embedded framework, and project versions. | No |
| `task plan` | Create a planned task with the next free ID. | Yes |
| `task register` | Reserve a task for one orchestrator. | Yes |
| `task show` | Read the stored registry or one task. | Yes |
| `task move` | Validate a status, stage, or field update. | Yes |
| `task block` | Add a blocker, optionally updating status and next action atomically. | Yes |
| `task unblock` | Clear an open blocker, optionally resuming the prior status. | Yes |
| `task handoff` | Transfer ownership on explicit user instruction. | Yes |
| `task done` | Verify the merge and final spec reference, then write the compact record. | Yes |

> Task commands need an initialized `workflow-state` registry. Initialization is still planned. Do not treat the current task commands as a complete new-project setup path.

## Invocation and help

```sh
solomonar help
solomonar task move --help
solomonar help task block
```

`solomonar` here means a binary on your `PATH`. If you followed [Getting started]({{ '/docs/getting-started/' | url }}), use `./bin/solomonar` instead.

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

| Delivery increment | Planned commands |
| --- | --- |
| Remainder of increment 1 | `init`, `task status` |
| Increment 2 | `doctor`, `sync`, `runtime list/add/remove`, `extension list/show/add/remove/new` (alias `ext`), `guard` |
| Increment 3 | `upgrade`, `dashboard` |

The read-only dashboard is a planned local view of the registry, not this documentation website. Windows, model calls, a daemon, and independently writable registry clones are outside the v0.1 scope.

## Full interface contract

For exhaustive arguments, validation tables, JSON data shapes, rejections, and use cases, read [the version-controlled interface specification]({{ 'docs/cli/interface.md' | source }}). It contains planned examples as well as implemented behavior; check this page's availability table before running a command.
