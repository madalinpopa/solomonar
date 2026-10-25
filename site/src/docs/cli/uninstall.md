---
title: solomonar uninstall
navTitle: solomonar uninstall
description: Remove the Solomonar framework, catalog extensions, and runtime adapters from one checkout with solomonar uninstall, keeping project extensions, decisions, notes, and task history.
kind: Reference
group: Commands
order: 13.5
sources:
  - internal/cli/uninstall.go
  - internal/project/uninstall.go
  - internal/project/apply.go
  - docs/cli/interface.md
---

Removes the Solomonar installation from the checkout you run it in, or the one `-C` names: the framework core, every installed catalog extension, the runtime adapter files, the generated part of `AGENTS.md`, the framework line in `docs/README.md`, and `.solomonar/manifest.json`. It plans from the manifest alone and removes only files the manifest records as written by the CLI and still unchanged. An edited file stops the command until you decide about it.

It is not extension removal and not binary removal:

- To remove one extension and keep the framework, use [`extension remove`]({{ '/docs/cli/extension/#solomonar-extension-remove' | url }}). The core cannot be removed that way; `uninstall` is the only command that removes it, together with everything else.
- `uninstall` leaves the `solomonar` executable, its Homebrew package, runtime applications such as Claude Code, and user-wide configuration in place. Remove the executable afterwards, if you no longer need it, as described in [Install the CLI]({{ '/docs/install/#remove-the-cli' | url }}).

{% cliUsage "When a project stops using Solomonar, or before installing it again from scratch. Run it in each checkout you want cleaned up; other worktrees are not changed.", "You, from your terminal. Review and commit the removal yourself; agents do not uninstall projects." %}

## Synopsis

```text
solomonar uninstall [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
```

## Flags

| Flag | Meaning |
| --- | --- |
| `--keep` | Leave one conflicting file in place, no longer managed; one path per flag. |
| `--overwrite` | Remove one edited managed file anyway; one path per flag. For `AGENTS.md`, remove the framework part and keep your project notes. |
| `-y`, `--yes` | Accept the plan without the confirmation prompt. Required without a terminal and with `--json`. It never resolves a conflict. |
| `--dry-run` | Run every check, print the plan, and write nothing. |

A path passed to `--keep` or `--overwrite` must be a conflict of this plan; another path exits 2 with `invalid_value`. The global flags work as for every command.

## What is removed and what stays

| Path | `uninstall` |
| --- | --- |
| `docs/solomonar/` core files and installed catalog extensions, including older layouts such as `docs/solomonar/skills/go/` | Removed when unchanged; an edited file is a conflict. |
| Runtime adapter files: the `.claude/`, `.agents/`, `.codex/`, and `.github/` stubs, and a `CLAUDE.md` the CLI created | Removed when unchanged; other files in those directories stay. |
| `AGENTS.md` | The framework part above `## Project notes` is removed. Your notes stay byte for byte; the file is deleted only when the notes section is still the empty template. |
| `docs/README.md` | Only the line `` - `docs/solomonar/`: framework rules, roles, and skills `` in `## Documents` is removed. A changed entry stays and is reported for you to review. |
| `.solomonar/manifest.json` | Removed last. |
| Project extensions in `docs/solomonar/extensions/<name>/` | Kept, with every file in them. Only their adapter stubs and `AGENTS.md` lines go. |
| `docs/decisions.md`, `docs/stack.md`, `docs/architecture.md`, specifications, work specs, reports, prototypes | Kept. Decisions stay as history, also those that name removed paths. |
| The registry `refs/heads/workflow-state` with every task, task branches, other worktrees, `.git`, `.jj` | Kept and never read. |

Directories left empty are removed, including `docs/solomonar/` and `.solomonar/`. The top-level `docs`, `.claude`, `.codex`, `.agents`, and `.github` directories stay.

Because your notes, decisions, and other documents stay, the project can still mention Solomonar after `uninstall`. The command warns about the references it knows of, such as a kept `CLAUDE.md` that still imports `@AGENTS.md`; the rest is yours to edit.

## Examples

These examples come from a run of the 0.2.0 build in the fictional `ledger` project, initialized for Claude Code and Codex with the `go` extension, the `money` project extension activated, one planned task, and a project note in `AGENTS.md`; paths are shortened to `/work/ledger`.

Preview the removal first; `--dry-run` writes nothing:

```sh
solomonar uninstall --dry-run
```

```text
Plan
  remove     .agents/skills/go/SKILL.md
  remove     .agents/skills/prototype/SKILL.md
  ...
  remove     .codex/agents/spec-reviewer.toml
  update     AGENTS.md                        managed part removed; project notes kept
  remove     CLAUDE.md
  update     docs/README.md                   framework index line removed
  remove     docs/solomonar/agents/adversarial-reviewer.md
  ...
  remove     docs/solomonar/extensions/go/rules/go/testing.md
  skip       docs/solomonar/extensions/money  project extension; source kept
  remove     docs/solomonar/report.md
  ...
  remove     docs/solomonar/skills/work-spec/template.md
  remove     .solomonar/manifest.json
Would uninstall solomonar from /work/ledger
  2 files updated, 51 removed, 1 skipped
```

In a terminal, `uninstall` shows the grouped plan on stderr and asks before it writes. Answering `n` prints `Cancelled; nothing was written.` and exits 0:

```sh
solomonar uninstall
```

```text
Plan
  framework 0.3.6
    update     AGENTS.md       managed part removed; project notes kept
    update     docs/README.md  framework index line removed
    remove     docs/solomonar/agents/ (6 roles)
    remove     docs/solomonar/report.md
    remove     docs/solomonar/reviewer.md
    remove     docs/solomonar/rules.md
    remove     docs/solomonar/skills/prototype/ (SKILL.md, references/cli.md, references/module.md, references/service.md, references/web.md, template.md)
    remove     docs/solomonar/skills/solomonar/ (SKILL.md)
    remove     docs/solomonar/skills/spec/ (SKILL.md, template.md)
    remove     docs/solomonar/skills/work-spec/ (SKILL.md, template.md)
  extensions
    remove     docs/solomonar/extensions/go/ (6 files)
    skip       docs/solomonar/extensions/money  project extension; source kept
  runtime claude
    remove     .claude/agents/ (6 role stubs: 5 core, 1 extension)
    remove     .claude/skills/ (5 skill stubs: 4 core, 1 extension)
    remove     CLAUDE.md
  runtime codex
    remove     .agents/skills/ (5 skill stubs: 4 core, 1 extension; solomonar/agents/openai.yaml)
    remove     .codex/agents/ (6 role stubs: 5 core, 1 extension)
  manifest
    remove     .solomonar/manifest.json
Proceed? [Y/n]
```

Without a terminal, pass `--yes`. With `--json`, `--yes` is required as well.

### A conflict

You edited `docs/solomonar/rules.md`. The plan lists every file and the conflict on stdout, and nothing is written (exit 4):

```sh
solomonar uninstall --yes
```

```text
Plan
  remove     .agents/skills/go/SKILL.md
  ...
  conflict   docs/solomonar/rules.md          modified since installation (manifest sha256 differs)
  ...
  remove     .solomonar/manifest.json
error: uninstall found 1 conflict (file_conflict)
  check: no planned file conflicts with the project's files
  found: 1 conflict: docs/solomonar/rules.md
  fix:   rerun with --keep <path> to keep the project's file, or --overwrite <path> to remove it
nothing was written
```

Decide whether you still need the edit, then choose per file. `--keep` leaves the file in place, no longer managed:

```sh
solomonar uninstall --keep docs/solomonar/rules.md --yes
```

```text
Uninstalled solomonar from /work/ledger
  2 files updated, 50 removed, 1 kept, 1 skipped
  kept: AGENTS.md, docs/README.md, docs/solomonar/extensions/money, docs/solomonar/rules.md
  The registry (refs/heads/workflow-state) and other worktrees are unchanged.

Next:
  1. Review the removal with git status, then commit it.
warning: docs/solomonar/rules.md is kept and no longer managed; remove it by hand if unneeded
```

`--overwrite docs/solomonar/rules.md` deletes the edited file instead, and the result reads `2 files updated, 51 removed, 1 skipped`. If `AGENTS.md` has lost its `## Project notes` heading, `--overwrite AGENTS.md` is refused so that your notes are never discarded: restore the heading, or keep the file.

A file that became a symlink, directory, or other non-regular file is never followed or removed; keep it with `--keep` and remove it by hand. A record outside the paths `uninstall` removes can only be kept.

### After the removal

`uninstall` never stages or commits. In the example, `git status` shows 46 deleted files and 3 modified ones: `AGENTS.md`, `docs/README.md`, and the kept `docs/solomonar/rules.md`. What is left in the working tree is project content:

```text
AGENTS.md
docs/README.md
docs/decisions.md
docs/solomonar/extensions/money/extension.json
docs/solomonar/extensions/money/money.md
docs/solomonar/rules.md
docs/stack.md
go.mod
```

`AGENTS.md` now holds only the notes section:

```text
## Project notes

<!-- Project-owned instructions; preserved by framework upgrades. Keep this heading unchanged. -->

Use integer cents.
```

The task history is untouched: the registry tip is the same commit, and `solomonar task show` still reads it:

```text
ID     STATUS     STAGE  OWNER  NEXT
W-001  planned    -      -      -
registry refs/heads/workflow-state @ ff93418 · 2026-10-24T18:26:44Z
```

Running `uninstall` again finds no manifest and does nothing, with exit 0:

```text
No Solomonar manifest; nothing to uninstall automatically.
```

### Known references

A `CLAUDE.md` that you wrote before `init` is not managed and stays. When it still imports `@AGENTS.md`, `uninstall` warns, with `--dry-run` and on success:

```text
warning: CLAUDE.md still imports @AGENTS.md, which uninstall removes; edit it by hand
```

A customized framework entry in `docs/README.md` is skipped with the same `manual_reference` warning code. Other mentions of Solomonar in your documents are not searched for.

### JSON

`--json --yes` prints one envelope; `files` holds every plan line and `preserved` the kept paths and the registry:

```json
{
  "ok": true,
  "command": "uninstall",
  "data": {
    "project": "/work/ledger",
    "files": [
      {
        "path": ".agents/skills/go/SKILL.md",
        "action": "remove",
        "owner": "runtime:codex"
      },
      ...
    ],
    "preserved": [
      "AGENTS.md",
      "docs/README.md",
      "docs/solomonar/extensions/money",
      "refs/heads/workflow-state"
    ],
    "next": [
      "Review the removal with git status, then commit it."
    ]
  },
  "warnings": []
}
```

The absent-manifest no-op has `"already_absent": true` and empty `files`, `preserved`, and `next`.

### Install again

`solomonar init` after `uninstall` reuses the retained registry, the project notes, and the project extension sources; reactivate a project extension with `extension add <name>`. `init` never rewrites an existing `docs/README.md`, so the removed framework index line does not come back; add it by hand if you want it.

## Errors

| Code | Exit | Meaning |
| --- | --- | --- |
| `invalid_value` | 2 | A `--keep` or `--overwrite` path is not a conflict of this plan, or `--overwrite` cannot resolve it. |
| `interactive_required` | 2 | No terminal, or `--json`, and no `--yes`. |
| `cli_outdated` | 3 | A newer release wrote the project; install a newer `solomonar`. |
| `file_conflict` | 4 | An edited or unsafe managed file or manifest, a record outside the removal paths, a hook entry, or a file that changed after the plan was shown; nothing is written. |
| `manifest_invalid` | 1 | `.solomonar/manifest.json` cannot be decoded; inspect it, never edit it by hand. |

A project whose manifest records a runtime hook entry cannot be uninstalled automatically: neither flag resolves that conflict until hook entry removal ships with the planned [`guard`]({{ '/docs/cli/guard/' | url }}) hooks. The current releases install no hook entries.

A handled failure or an interruption restores every file this run changed, and the command ends with `nothing was written`. If restoring itself fails, the error lists the paths under `found: not restored: …`, and the last line reads `rollback incomplete; recover the paths named above`; restore them, for example with `git restore`, then rerun. There is no recovery after a forced kill or power loss during the removal.
