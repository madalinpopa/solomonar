---
title: solomonar runtime
navTitle: solomonar runtime
description: List, add, and remove the Claude Code and Codex adapters of a Solomonar project with the solomonar runtime commands. In development.
kind: Reference
group: In development
order: 19
inDevelopment: true
sources:
  - docs/cli/interface.md
---

> These commands are in development and not in the current build; running one today exits 2 with an unknown-command error. This page describes the planned behavior; the final output may differ in detail.

A runtime is the agent product that runs sessions: Claude Code (`claude`) or Codex (`codex`). Each runtime gets small generated adapter files that point to the portable source in `docs/solomonar/`, never a second copy of it. Until these commands exist, choose runtimes with `init --runtime`.

## solomonar runtime list

Read-only. Lists the supported runtimes with whether each is installed in the project, whether its executable is on `PATH`, and the adapter files it uses.

{% cliUsage "Before adding or removing a runtime, or to see which adapter files belong to which runtime.", "You." %}

```text
solomonar runtime list [--json]
```

### Example

```sh
solomonar runtime list
```

```text
RUNTIME  STATUS     ON PATH  ADAPTER
claude   installed  yes      CLAUDE.md, .claude/skills (8), .claude/agents (5), guard hook
codex    available  yes      AGENTS.md (native), .agents/skills, .codex/agents, guard hook
```

## solomonar runtime add

Generates the adapter files of one runtime: its skill stubs, role stubs, and state guard hook, then records them in the manifest.

{% cliUsage "When you start using a second runtime in a project set up for one, or chose `--runtime none` at `init`.", "You. Commit the new files on main afterwards." %}

```text
solomonar runtime add <claude|codex> [--adversarial-model <model>] [-y] [--dry-run] [--json]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<claude\|codex>` | Required. The runtime to add. |
| `--adversarial-model` | Model for this runtime's adversarial reviewer. |
| `-y`, `--yes` | Accept the plan without prompts. |
| `--dry-run` | Print the plan and write nothing. |

### Example

```sh
solomonar runtime add codex --yes
```

```text
Plan
  create  .agents/skills/ (8 skill stubs: solomonar, prototype, spec, work-spec, go, htmx, templ, tdd)
  create  .agents/skills/solomonar/agents/openai.yaml
  create  .codex/agents/ (5 role stubs)
  create  .codex/hooks.json (state guard hook)
  update  .solomonar/manifest.json
Added runtime codex: 16 files.
  warning: Codex state guard hook not trusted yet; review it with /hooks in Codex.
Next: commit the files on main.
```

## solomonar runtime remove

Deletes the runtime's unmodified adapter files and its hook entry. It keeps a `CLAUDE.md` it did not create. A modified adapter file is a conflict (exit 4).

{% cliUsage "When the project stops using a runtime.", "You." %}

```text
solomonar runtime remove <claude|codex> [--keep <path>]... [--overwrite <path>]... [-y] [--dry-run] [--json]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<claude\|codex>` | Required. The runtime to remove. |
| `--keep` | Untrack one modified adapter file and leave it on disk; one path per flag. |
| `--overwrite` | Delete one modified adapter file; one path per flag. |
| `-y`, `--yes` | Accept the plan without prompts. |
| `--dry-run` | Print the plan and write nothing. |

### Example

```sh
solomonar runtime remove codex --dry-run
```

The output lists each adapter file to delete, as in the `runtime add` plan; nothing is written.
