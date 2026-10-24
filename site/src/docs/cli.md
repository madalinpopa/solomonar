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
| [`upgrade`]({{ '/docs/cli/upgrade/' | url }}) | Move the project to this CLI's framework version, keeping project edits and choices. | You | Yes |
| [`uninstall`]({{ '/docs/cli/uninstall/' | url }}) | Remove the framework, catalog extensions, and adapters from one checkout, keeping project extensions, decisions, notes, and task history. | You | Yes |
| [`runtime list`]({{ '/docs/cli/runtime/#solomonar-runtime-list' | url }}) | List Claude Code, Codex, and Copilot CLI with their installation, executable, and adapter locations. | You | No |
| [`runtime add`]({{ '/docs/cli/runtime/#solomonar-runtime-add' | url }}) | Generate one runtime's adapter files and record it, or update its adversarial-reviewer model. | You | Yes |
| [`runtime remove`]({{ '/docs/cli/runtime/#solomonar-runtime-remove' | url }}) | Delete one runtime's unmodified adapter files and drop it from the manifest. | You | Yes |
| [`extension list`]({{ '/docs/cli/extension/#solomonar-extension-list' | url }}) | List the catalog and project extensions with their status, decision, and suggestion; alias `ext`. | You and agents | No |
| [`extension add`]({{ '/docs/cli/extension/#solomonar-extension-add' | url }}) | Install catalog extensions or activate project extensions, recording a stack decision. | You, or an orchestrator with an approved decision | Yes |
| [`extension remove`]({{ '/docs/cli/extension/#solomonar-extension-remove' | url }}) | Remove catalog extensions or deactivate project extensions. | You, or an orchestrator with an approved decision | Yes |
| [`task plan`]({{ '/docs/cli/task/#solomonar-task-plan' | url }}) | Create a planned task with the next free ID. | You | Yes |
| [`task register`]({{ '/docs/cli/task/#solomonar-task-register' | url }}) | Reserve a task for one orchestrator. | Orchestrator agent | Yes |
| [`task show`]({{ '/docs/cli/task/#solomonar-task-show' | url }}) | Read the stored registry or one task. | You and agents | Yes |
| [`task status`]({{ '/docs/cli/task/#solomonar-task-status' | url }}) | Compare one task's record with Git and forge facts; list drift and valid next commands. | You and agents | Yes |
| [`task move`]({{ '/docs/cli/task/#solomonar-task-move' | url }}) | Validate a status, stage, or field update. | Owning orchestrator | Yes |
| [`task block`]({{ '/docs/cli/task/#solomonar-task-block' | url }}) | Add a blocker, optionally updating status and next action atomically. | Owning orchestrator | Yes |
| [`task unblock`]({{ '/docs/cli/task/#solomonar-task-unblock' | url }}) | Clear an open blocker, optionally resuming the prior status. | Owning orchestrator | Yes |
| [`task handoff`]({{ '/docs/cli/task/#solomonar-task-handoff' | url }}) | Transfer ownership on explicit user instruction. | New orchestrator, on your instruction | Yes |
| [`task confirm`]({{ '/docs/cli/task/#solomonar-task-confirm' | url }}) | Confirm a task's continuation after a handoff or migration, once its state is inspected. | Owning orchestrator | Yes |
| [`task migrate`]({{ '/docs/cli/task/#solomonar-task-migrate' | url }}) | Rewrite a schema-1 registry as one file per task, once. | You | Yes |
| [`task done`]({{ '/docs/cli/task/#solomonar-task-done' | url }}) | Verify the merge and final spec reference, then write the compact record. | Owning orchestrator | Yes |
| [`version`]({{ '/docs/cli/version/' | url }}) | Print the CLI and project framework versions. | You | No |
| [`help`]({{ '/docs/cli/help/' | url }}) | Print help; `-h` and `--help` are equivalent. | You and agents | No |

Task commands, `upgrade`, `runtime add|remove`, and `extension add|remove` need a project prepared with `init`; `runtime list` and `extension list` run in any repository. `uninstall` without a manifest is a no-op. Agents run the writing task commands; you mostly use `init`, `upgrade`, `runtime`, `extension`, `uninstall`, `task plan`, `task show`, and `task status`.

## Commands in development

These commands are specified and scheduled for delivery. They are **not in the current build**: running one today exits 2 with an unknown-command error. Their pages carry an <span class="cli-label">In development</span> label and show examples from the interface specification.

| Command | Purpose | Who runs it |
| --- | --- | --- |
| [`doctor`]({{ '/docs/cli/doctor/' | url }}) | Check the repository, registry, files, extensions, adapters, and forge settings. | You, or CI |
| [`sync`]({{ '/docs/cli/sync/' | url }}) | Regenerate runtime adapter files from the portable source. | You, or CI with `--check` |
| [`guard`]({{ '/docs/cli/guard/' | url }}) | Deny direct registry writes from an agent session. | The runtime's hook |

## Global flags

| Flag | Meaning |
| --- | --- |
| `-C <path>` | Run as if started in `<path>`, like `git -C`. Any worktree of the repository works. |
| `--json` | Print machine-readable output on stdout. Never prompts: a command that needs an answer takes `--yes` or fails with `interactive_required`. |
| `--no-color` | Disable color; setting `NO_COLOR` to any value does the same. |
| `-v`, `--verbose` | Print each Git and `gh` command to stderr, as `+ git …` or `+ gh …`, before it runs. |
| `-h`, `--help` | Show command help. |

`solomonar` here means a binary on your `PATH`; see [Install the CLI]({{ '/docs/install/' | url }}). Writing task commands except `task migrate` support `--dry-run`: they run validation and report the proposed record without writing it. Owner commands accept `--as <owner>`, with `SOLOMONAR_OWNER` as the default.

## Output formats

Human-readable results go to stdout. Warnings and errors go to stderr, prefixed `warning:` or `error:`. With `--json`, stdout contains exactly one JSON object; its field names are `snake_case`, timestamps are RFC 3339 UTC, and SHAs are full.

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
| 1 | Unexpected failure: I/O, a Git error, a corrupt object. Nothing is written; only when `uninstall` cannot restore its changes does it name the paths left changed. |
| 2 | Usage error: unknown command or flag, missing or malformed argument. |
| 3 | Rejected: a validation, ownership, transition, or fact check failed. |
| 4 | Conflict: a managed file differs from what the CLI would write, or the registry update kept losing to concurrent writes. |
| 5 | Checks failed: `doctor` found a `fail`, or `sync --check` found stale files. <span class="cli-label">In development</span> |
| 130, 143 | Interrupted by SIGINT or SIGTERM before anything was written. Nothing is written. |

An interruption after a registry update has succeeded exits 0 with the warning `push_interrupted`, so a script never repeats a write that already happened.

## Planned commands

These are interface contracts, not commands available in the current build.

| Delivery increment | Planned |
| --- | --- |
| Increment 2, [in development](#commands-in-development) | `doctor`, `sync`, `guard` and its runtime hooks |
| Increment 3 | `upgrade --diff`, `dashboard` |

The catalog holds `go` and `extension-authoring`; further catalog extensions are added once their framework content is written and reviewed. A project's own rules and reviewers are [project extensions]({{ '/docs/cli/extension/#write-a-project-extension' | url }}).

The read-only dashboard is a planned local view of the registry, not this documentation website. Windows, model calls, a daemon, and independently writable registry clones are outside the v0.1 scope.

## Full interface contract

The command pages are the reference for every command's synopsis, flags, and output. The [interface specification]({{ 'docs/cli/interface.md' | source }}) holds the contract beneath them: check order and exact texts, file planning and ownership, the manifest, JSON data shapes, the task transition rules, and the error code catalog. It also covers planned commands; check the availability tables above before running one.
