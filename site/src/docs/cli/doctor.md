---
title: solomonar doctor
navTitle: solomonar doctor
description: Check a Solomonar project's repository, registry, managed files, extensions, runtime adapters, and forge settings with solomonar doctor. In development.
kind: Reference
group: In development
order: 17
inDevelopment: true
sources:
  - docs/cli/interface.md
---

> This command is in development and not in the current build; running it today exits 2 with an unknown-command error. This page describes the planned behavior; the final output may differ in detail.

Read-only. Checks the repository, the registry, the managed files, the installed extensions, the runtime adapters, the forge settings, and the project commands, then prints `ok`, `warn`, or `fail` for each check. It exits 5 when any check fails, else 0.

{% cliUsage "After `init` and its first commit, after a CLI update, when an agent reports something odd about the project setup, and in CI.", "You, or your CI. Agents may run it to diagnose an environment blocker." %}

## Synopsis

```text
solomonar doctor [--json]
```

## Checks

| Area | What fails (exit 5) | What only warns |
| --- | --- | --- |
| git | Git older than 2.40, no repository, no commit on main, unusable worktrees | A `.jj/` directory |
| registry | A missing or invalid registry, or a schema or framework newer than this CLI | No remote, or an unpublished local registry |
| files | An invalid manifest, missing managed files, or managed files not committed on main | Managed files modified since installation |
| extensions | A stack extension without an approved decision for its group's full selection, unmet `requires` or `conflicts`, an installed project extension that fails the extension checks or whose directory is gone, an incomplete core | - |
| runtimes | Adapter files that differ from what `upgrade` would write; `upgrade` at the project's version fixes them | `CLAUDE.md` not loading `AGENTS.md`, or a missing Claude/Codex guard hook |
| forge | A repository that does not allow merge commits | `gh` missing or not authenticated, squash or rebase merges also allowed |
| project | - | Commands in `docs/README.md` not established |

Copilot has no expected guard hook in this delivery. Its absence is not a warning or failure; check its instruction, skill, and role adapters instead. `doctor` remains unavailable in the current build.

## Example

The specification's `ledger` project, with three extensions installed:

```sh
solomonar doctor
```

```text
git
  ok    git 2.47.1 (2.40 or later)
  ok    main branch "main" has commits
  warn  .jj/ present: v0.1 supports Git only
registry
  ok    refs/heads/workflow-state @ 3a5c0d7, schema_version 2, 4 tasks valid
  ok    origin/workflow-state matches local
files
  ok    framework 0.3.3, manifest valid
  fail  4 framework paths not committed on main: AGENTS.md, CLAUDE.md, .claude/, .solomonar/
  warn  docs/solomonar/skills/spec/template.md modified since installation (kept by upgrades)
extensions
  ok    3 installed: go (P/D-1), extension-authoring, money-reviewer (project); requirements met; project extensions valid
runtimes
  ok    claude: CLAUDE.md imports AGENTS.md; 6 skill stubs, 6 role stubs in sync; guard hook present
  warn  codex: state guard hook not trusted yet; review it with /hooks
forge
  ok    gh 2.62.0 authenticated for github.com/acme/ledger
  ok    merge commits allowed (rules section 11)
  warn  squash merges also allowed; the orchestrator must use merge commits only
project
  warn  docs/README.md commands: run, build, test, lint not established
1 failure, 4 warnings
```
