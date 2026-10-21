---
title: Configuration and runtime adapters
description: Reference Solomonar environment variables, specification files, project layout, and planned runtime adapters for Claude Code and Codex.
kind: Reference
group: Understand
order: 4
sources:
  - docs/cli/interface.md
  - assets/templates/AGENTS.md
  - assets/framework/rules.md
---

## Current CLI controls

The CLI does not use a general configuration file. Global flags control each invocation; the interface defines two environment variables:

| Variable | Purpose |
| --- | --- |
| `SOLOMONAR_OWNER` | Default acting identity for owner commands such as `task move`, `block`, and `unblock`. |
| `NO_COLOR` | Disable color when set to any value. |

Agents pass `--as` on each owner command because a runtime may start a new shell for each invocation. Ownership validation prevents accidental writes from the wrong session; it is not authentication.

## Portable project layout

The following layout is the **initialization contract**. `init` and runtime adapter installation are planned; these paths describe what they will create in a managed project.

| Path | Owns |
| --- | --- |
| `AGENTS.md` | Compact mandatory instructions and context routes. |
| `docs/README.md` | Project mission, document index, and verified run/check commands. |
| `docs/specs/` | Current capability behavior. |
| `docs/work/` | Active task specifications; completed specs remain in Git history. |
| `docs/architecture.md` | Approved structure, boundaries, and shared contracts. |
| `docs/stack.md` | Chosen technologies and rationale; manifests hold versions. |
| `docs/decisions.md` | Scoped approved, pending, and superseded decisions. |
| `docs/ui.md` | Shared UI and interaction rules when useful. |
| `docs/solomonar/` | Portable framework rules, roles, skills, and extensions. |
| `.solomonar/manifest.json` | Installed choices and the hashes of managed file bases. |

Documents are created when useful. A small project does not need every optional file on day one.

## Runtime adapters

The design targets Claude Code and Codex. The portable source remains in `docs/solomonar/`; small generated adapter files point to it rather than duplicating the instructions.

The entry invocation is `/solomonar` in Claude Code and `$solomonar` in Codex. A portable request is `Use the solomonar skill`. These invocations apply after the appropriate adapters are installed; the current CLI does not install them yet.

The orchestrator is always the top-level session. It is not registered as a delegated native agent, because it must delegate the implementer and reviewers itself.

## Extensions and ownership

The planned extension commands add skills and specialist reviewers. Stack extensions record an approved technology decision; workflow and review extensions configure the process without a stack decision.

Project extensions start as inactive drafts until adopted. Drafts do not contribute rules, runtime adapters, or required reviewers.

Initialization and upgrades preserve project edits. Identical managed content is a no-op; differing content becomes an explicit conflict. A retained project edit does not become the distributed base hash. This allows a later upgrade to detect the difference again.

The full catalog and conflict-resolution flags remain in the [CLI interface specification]({{ 'docs/cli/interface.md' | source }}).
