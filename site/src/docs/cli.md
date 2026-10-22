---
title: CLI commands and flags
navTitle: CLI overview
description: Overview of the Solomonar CLI. Every command with who runs it, the commands in development, global flags, JSON output, and exit codes.
kind: Reference
group: Reference
order: 11
sources:
  - internal/cli/run.go
  - internal/cli/flags.go
  - internal/cli/output.go
  - docs/cli/interface.md
---

The CLI is a guard and an oracle. It validates and records decisions and reports Git facts. The orchestrator runs the workflow; the binary never launches agents or calls model providers.

Each command has its own page with what it does, when to use it, who runs it, its flags, and examples.

## Current commands

These commands are present in the current implementation. Use `solomonar help` from your build to confirm its command table.

| Command | Purpose | Who runs it | Requires a project |
| --- | --- | --- | --- |
| [`init`]({{ '/docs/cli/init/' | url }}) | Install the framework core, runtime adapters, and extensions, and create the registry. | You | No |
| [`task plan`]({{ '/docs/cli/task/#solomonar-task-plan' | url }}) | Create a planned task with the next free ID. | You | Yes |
| [`task register`]({{ '/docs/cli/task/#solomonar-task-register' | url }}) | Reserve a task for one orchestrator. | Orchestrator agent | Yes |
| [`task show`]({{ '/docs/cli/task/#solomonar-task-show' | url }}) | Read the stored registry or one task. | You and agents | Yes |
| [`task status`]({{ '/docs/cli/task/#solomonar-task-status' | url }}) | Compare one task's record with Git and forge facts; list drift and valid next commands. | You and agents | Yes |
| [`task move`]({{ '/docs/cli/task/#solomonar-task-move' | url }}) | Validate a status, stage, or field update. | Owning orchestrator | Yes |
| [`task block`]({{ '/docs/cli/task/#solomonar-task-block' | url }}) | Add a blocker, optionally updating status and next action atomically. | Owning orchestrator | Yes |
| [`task unblock`]({{ '/docs/cli/task/#solomonar-task-unblock' | url }}) | Clear an open blocker, optionally resuming the prior status. | Owning orchestrator | Yes |
| [`task handoff`]({{ '/docs/cli/task/#solomonar-task-handoff' | url }}) | Transfer ownership on explicit user instruction. | New orchestrator, on your instruction | Yes |
| [`task done`]({{ '/docs/cli/task/#solomonar-task-done' | url }}) | Verify the merge and final spec reference, then write the compact record. | Owning orchestrator | Yes |
| [`version`]({{ '/docs/cli/version/' | url }}) | Print the CLI and project framework versions. | You | No |
| [`help`]({{ '/docs/cli/help/' | url }}) | Print help; `-h` and `--help` are equivalent. | You and agents | No |

Task commands need a project prepared with `init`. Agents run the writing task commands; you mostly use `init`, `task plan`, `task show`, and `task status`.

## Commands in development

These commands are specified and scheduled for the next delivery increment. They are **not in the current build**: running one today exits 2 with an unknown-command error. Their pages carry an <span class="cli-label">In development</span> label and show examples from the interface specification.

| Command | Purpose | Who runs it |
| --- | --- | --- |
| [`doctor`]({{ '/docs/cli/doctor/' | url }}) | Check the repository, registry, files, extensions, adapters, and forge settings. | You, or CI |
| [`sync`]({{ '/docs/cli/sync/' | url }}) | Regenerate runtime adapter files from the portable source. | You, or CI with `--check` |
| [`runtime list/add/remove`]({{ '/docs/cli/runtime/' | url }}) | Manage the Claude Code and Codex adapters. | You |
| [`extension list/show/add/remove/new`]({{ '/docs/cli/extension/' | url }}) | Choose the stack and add skills or reviewers; alias `ext`. | You, or an orchestrator with an approved decision |
| [`guard`]({{ '/docs/cli/guard/' | url }}) | Deny direct registry writes from an agent session. | The runtime's hook |

Increment 2 also completes the new-product path: a project initialized with `--ext none` chooses its stack later with `extension add`.

## Global flags

| Flag | Meaning |
| --- | --- |
| `-C <path>` | Run in another directory, like `git -C`. |
| `--json` | Print machine-readable output on stdout. |
| `--no-color` | Disable color; equivalent to setting `NO_COLOR`. |
| `-v`, `--verbose` | Trace external Git and `gh` commands to stderr. |
| `-h`, `--help` | Show command help. |

`solomonar` here means a binary on your `PATH`; see [Install the CLI]({{ '/docs/install/' | url }}). Writing task commands support `--dry-run`: they run validation and report the proposed record without writing it. Owner commands accept `--as <owner>`, with `SOLOMONAR_OWNER` as the default.

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

On failure, `ok` is `false`, `error` replaces `data`, and the error names the failed check, what was found, and how to correct it:

```json
{
  "ok": false,
  "command": "task move",
  "error": {
    "code": "tier_required",
    "message": "task move W-001 rejected",
    "check": "a stage after implementation needs a recorded tier",
    "found": "implementation -> integration",
    "fix": "triage the candidate and pass --revision <full-sha> --tier <0|1|2> with the stage move"
  },
  "warnings": []
}
```

Unknown forge or check facts never count as passing.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success, including a no-op, a declined confirmation, and success with warnings such as a failed push. |
| 1 | Unexpected failure: I/O, a Git error, a corrupt object. Nothing is written. |
| 2 | Usage error: unknown command or flag, missing or malformed argument. |
| 3 | Rejected: a validation, ownership, transition, or fact check failed. |
| 4 | Conflict: a managed file differs from what the CLI would write, or the registry update kept losing to concurrent writes. |
| 5 | Checks failed: `doctor` found a `fail`, or `sync --check` found stale files. <span class="cli-label">In development</span> |

## Planned commands

These are interface contracts, not commands available in the current build.

| Delivery increment | Planned |
| --- | --- |
| Increment 2, [in development](#commands-in-development) | `doctor`, `sync`, `runtime list/add/remove`, `extension list/show/add/remove/new` (alias `ext`), project extensions from a local path, `guard` and its runtime hooks |
| Increment 3 | `upgrade`, `dashboard` |

The `htmx`, `templ`, and `tdd` extensions of the interface catalog are added once their framework content is written.

The read-only dashboard is a planned local view of the registry, not this documentation website. Windows, model calls, a daemon, and independently writable registry clones are outside the v0.1 scope.

## Full interface contract

The command pages cover every flag of the current commands. For the full validation tables, JSON data shapes, error codes, and use cases, read [the version-controlled interface specification]({{ 'docs/cli/interface.md' | source }}). It contains planned examples as well as implemented behavior; check the availability tables above before running a command.
