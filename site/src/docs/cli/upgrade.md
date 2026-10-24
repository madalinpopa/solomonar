---
title: solomonar upgrade
navTitle: solomonar upgrade
description: Move a Solomonar project to the framework embedded in the installed CLI with solomonar upgrade, keeping project edits, choices, decisions, and project extensions.
kind: Reference
group: Commands
order: 13.4
sources:
  - internal/cli/upgrade.go
  - internal/project/apply.go
  - internal/project/classify.go
  - docs/cli/interface.md
---

Updates the managed files to the framework version embedded in the running CLI, with the choices the manifest records, and removes the managed files the new version no longer offers. The runtimes, adversarial models, installed extensions, and each stack group's decision stay as recorded: `upgrade` writes no decision, reads no `docs/decisions.md`, and never adds an extension that became a default or is newly detected.

Project extensions in `docs/solomonar/extensions/`, the decision documents, `docs/README.md`, and the `## Project notes` section of `AGENTS.md` are never touched. The registry is not touched either; a registry schema change is the separate [`task migrate`]({{ '/docs/cli/task/#solomonar-task-migrate' | url }}).

{% cliUsage "After installing a newer CLI release, before the next task starts; `extension add` and `extension remove` reject a project whose framework is older than the CLI with `framework_outdated`.", "You, on main. Running tasks pick up the new framework after they update from main." %}

## Synopsis

```text
solomonar upgrade [--keep <path>]... [--take-new <path>]... [-y|--yes] [--dry-run] [--json]
```

## Flags

| Flag | Meaning |
| --- | --- |
| `--keep` | Keep the project's version of one conflicting file; one path per flag. For a file the new version no longer offers, leave it in place and stop tracking it. |
| `--take-new` | Take the new version of one conflicting file, or delete one that is no longer offered; one path per flag. |
| `-y`, `--yes` | Accept the plan without prompts. Required without a terminal. |
| `--dry-run` | Print the plan and write nothing. |

A path passed to `--keep` or `--take-new` must be a conflict of this plan; another path exits 2 with `invalid_value`.

## How each file is planned

`base` is the hash the manifest records, `current` the file on disk, and `new` the content this CLI offers.

| Condition | Action |
| --- | --- |
| current = base, new = base | `unchanged` |
| current = base, new ≠ base | `update` |
| current ≠ base, new = base | `keep`: your edit stays |
| current = new | `unchanged`; the recorded base moves to new |
| current ≠ base, new ≠ base, current ≠ new | `conflict` |
| no longer offered, current = base | `remove` |
| no longer offered, current ≠ base | `conflict` |

After an upgrade, the recorded base of every managed file is the new version, even for a file you kept. A later upgrade that changes the file again reports a conflict instead of overwriting your edit. In `AGENTS.md`, only the part above `## Project notes` is planned.

## Examples

These examples come from a run of the 0.2.0 build in the fictional `ledger` project, initialized with 0.1.7 for Claude Code and Codex with the `go` extension; paths are shortened to `/work/ledger`.

0.2.0 moves installed extensions to the store `docs/solomonar/extensions/<name>/`, so the 0.1.7 copy under `docs/solomonar/skills/go/` is no longer offered. In a terminal, `upgrade` shows the plan on stderr and asks before it writes:

```sh
solomonar upgrade
```

```text
Upgrade framework 0.1.7 → 0.2.0
  framework 0.3.1
    update     docs/solomonar/agents/ (6 roles)
    update     docs/solomonar/report.md
    update     docs/solomonar/reviewer.md
    update     docs/solomonar/rules.md
    update     docs/solomonar/skills/prototype/ (SKILL.md, template.md)
    update     docs/solomonar/skills/solomonar/ (SKILL.md)
    update     docs/solomonar/skills/spec/ (SKILL.md, template.md)
    update     docs/solomonar/skills/work-spec/ (SKILL.md, template.md)
    update     AGENTS.md
  extensions
    create     docs/solomonar/extensions/go/ (6 files)
    remove     docs/solomonar/skills/go/ (5 files)
  runtime claude
    update     .claude/skills/ (5 skill stubs: 4 core, 1 extension)
    update     .claude/agents/ (5 role stubs; adversarial-reviewer model opus)
  runtime codex
    update     .agents/skills/ (5 skill stubs: 4 core, 1 extension; solomonar/agents/openai.yaml)
    update     .codex/agents/ (5 role stubs)
  manifest
    update     .solomonar/manifest.json
Proceed? [Y/n] y

Upgraded solomonar 0.1.7 → 0.2.0 in /work/ledger
  6 files created, 39 updated, 5 removed, 3 unchanged, 1 skipped

Next:
  1. git add AGENTS.md .claude .agents .codex .solomonar docs && git commit -m "chore: upgrade solomonar to 0.2.0"
  2. Running tasks pick up the new framework after they update from main.
```

Commit the files as shown. `--dry-run` prints the same header and every file line, including `unchanged` and `skip`, on stdout, then `Would upgrade solomonar 0.1.7 → 0.2.0 in /work/ledger` and the counts. `--yes` prints only the result.

A project already at the CLI's version is a no-op and needs no confirmation:

```text
Already at solomonar 0.3.1; nothing to do.
  48 files unchanged, 1 skipped
```

### A conflict

You edited the 0.1.7 copy of the Go rules, which 0.2.0 no longer offers. The plan lists every file and the conflict, and nothing is written (exit 4):

```sh
solomonar upgrade --yes
```

```text
Upgrade framework 0.1.7 → 0.2.0
  update     docs/solomonar/agents/adversarial-reviewer.md
  ...
  conflict   docs/solomonar/skills/go/SKILL.md  modified since installation (manifest sha256 differs)
  remove     docs/solomonar/skills/go/rules/go/concurrency.md
  ...
  update     .solomonar/manifest.json
error: upgrade found 1 conflict (file_conflict)
  check: no planned file conflicts with the project's files
  found: 1 conflict: docs/solomonar/skills/go/SKILL.md
  fix:   rerun with --keep <path> to keep the project's file, or --take-new <path> to take the new version
nothing was written
```

Decide whether you still need the edit, then choose per file. `--take-new` deletes the old file; `--keep` leaves it in place, no longer managed:

```sh
solomonar upgrade --keep docs/solomonar/skills/go/SKILL.md --yes
```

```text
Upgraded solomonar 0.1.7 → 0.2.0 in /work/ledger
  6 files created, 39 updated, 4 removed, 1 kept, 3 unchanged, 1 skipped
```

## Errors

| Code | Exit | Meaning |
| --- | --- | --- |
| `not_initialized` | 3 | No `.solomonar/manifest.json`; run `solomonar init`. |
| `cli_outdated` | 3 | The project was written by a newer release; install a newer `solomonar`. `upgrade` never downgrades. |
| `invalid_value` | 2 | A `--keep` or `--take-new` path is not a conflict of this plan, or an installed project extension no longer passes the extension checks. |
| `file_conflict` | 4 | A managed file was edited and the new version changes it too; nothing is written. |
| `interactive_required` | 2 | No terminal and no `--yes`. |

Decision entries written by 0.1.7 keep their references to `docs/solomonar/skills/<extension>/`, which no longer exist after the upgrade; the next decision for the group names the store paths. `upgrade --diff`, to compare one file with its new version, is planned.
