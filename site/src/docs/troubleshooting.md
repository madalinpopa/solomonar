---
title: Troubleshooting
description: Understand actionable rejections, missing setup, ownership mistakes, and the limits of the current build.
kind: How-to guide
group: Use the CLI
order: 8
sources:
  - docs/cli/interface.md
  - internal/cli/errors.go
  - internal/cli/repo.go
---

## A command is unknown

First run `solomonar help` to see the command table in your binary. The interface specification includes future commands; the current build does not yet implement `init`, `doctor`, or `task status`.

Use the [availability table]({{ '/docs/cli/#current-commands' | url }}) before following an example from the full design contract. An unknown command exits 2; it does not mean your application needs to be scaffolded again.

## The registry is not initialized

`not_initialized` means `refs/heads/workflow-state` is missing. Task commands depend on that registry. Because `init` is still planned, a normal source checkout is not a complete managed project.

Do not write `state.json` by hand or create a registry branch to bypass initialization. You can still build the CLI, inspect help and versions, and read the framework contracts.

## The command cannot find the repository

`not_a_repository` means the current directory, or the directory selected with `-C`, is outside a Git repository. Run from the intended repository or pass its path with `-C`.

Git must be version 2.40 or later. `gh` is optional; missing or unauthenticated forge access means those facts are unknown, never passing.

## An owner command is rejected

`owner_required` means neither `--as` nor `SOLOMONAR_OWNER` identifies the acting owner. `owner_mismatch` means that identity differs from the task's recorded owner.

Inspect the stored record with `task show`, then act as the correct owner. A handoff requires explicit user instruction and the `task handoff` command. Do not impersonate an owner to work around a stale session.

## A transition or review candidate is rejected

Read the error's `check`, `found`, and `fix` lines before retrying. Typical causes include:

- `revision_not_full_sha`: supply the full 40-character commit hash.
- `revision_not_on_branch`: the candidate must be reachable from the task's feature branch.
- `spec_not_found`: the spec must exist in the task worktree or at the recorded revision.
- `tier_required`: entering a review or integration stage needs the recorded risk tier.
- `blockers_required`: a blocked task needs an open blocker; unblocking the last one must also resume the prior status.

A rejected state update leaves the registry unchanged. A failed remote push after a successful local update is different: the local write remains successful and publication is reported as a warning. Inspect the current record before repeating a write.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success, including no-op and success with a publication warning. |
| `1` | Unexpected I/O or Git failure, or a corrupt object/registry. |
| `2` | Usage error or malformed input. |
| `3` | Rejected validation, ownership, transition, or fact check. |
| `4` | File conflict or exhausted compare-and-swap retries. |
| `5` | Planned `doctor` or `sync --check` found a failing check. |
| `130`, `143` | Interrupted before the local commit point. |

After a successful registry update, interruption during publication reports success with `push_interrupted`. This avoids making a script repeat a write that already happened.

## Report a problem with evidence

Include the CLI version, the command, the error code, and the relevant Git state. Redact secrets and private content. Distinguish an implemented command failure from a planned feature that is not available yet.

The detailed error catalog and corrective actions are in the [CLI interface specification]({{ 'docs/cli/interface.md' | source }}).
