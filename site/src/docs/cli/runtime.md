---
title: solomonar runtime
navTitle: solomonar runtime
description: Planned runtime commands for Claude Code, Codex, and GitHub Copilot CLI adapters. List, add, or remove runtimes once runtime management is available.
kind: Reference
group: In development
order: 19
inDevelopment: true
sources:
  - docs/cli/interface.md
---

> These commands are in development and not in the current build; running one today exits 2 with an unknown-command error. This page describes the planned behavior; the final output may differ in detail.

A runtime is the agent product that runs sessions: Claude Code (`claude`), Codex (`codex`), or GitHub Copilot CLI (`copilot`). Each runtime gets small generated adapter files that point to the portable source in `docs/solomonar/`, never a second copy of it.

The current build accepts `claude`, `codex`, and `copilot` through `init --runtime`. The [Copilot implementation plan]({{ 'docs/cli/plan.md' | source }}) tracks the remaining runtime acceptance. This delivery covers initialization only; the commands below remain unavailable. `init` never changes recorded runtime choices, so adding Copilot to an already initialized project must wait for `runtime add`.

Copilot targets the CLI only. It reads `AGENTS.md` natively and will get `.github/skills/<skill>/SKILL.md` and `.github/agents/<role>.agent.md` stubs, tracked as `runtime:copilot`. No extra root instruction file or Copilot guard hook is generated. Its entry skill is user-invoked with `/solomonar`; delegated roles load the portable role and repository instructions. IDE and cloud-agent compatibility are outside this delivery.

The planned adapter follows GitHub's [CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference) and [custom-agent documentation](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli), checked on 2026-10-23. Live acceptance and the minimum supported Copilot version are still pending.

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
copilot  available  yes      AGENTS.md (native), .github/skills, .github/agents; no guard hook
```

## solomonar runtime add

Generates the adapter files of one runtime and records them in the manifest. Claude and Codex also get their planned state guard hooks; Copilot gets skill and role stubs only.

{% cliUsage "When you start using a second runtime in a project set up for one, or chose `--runtime none` at `init`.", "You. Commit the new files on main afterwards." %}

```text
solomonar runtime add <claude|codex|copilot> [--adversarial-model <model>] [-y] [--dry-run] [--json]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<claude\|codex\|copilot>` | Required. The runtime to add. |
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

For Copilot, `--adversarial-model <model>` writes only the adversarial-reviewer stub's `model` field. An omitted model inherits runtime defaults. Copilot can override or fall back from the configured model; installation does not authenticate, check account access, or guarantee model isolation. There is no Copilot hook entry to install.

## solomonar runtime remove

Deletes the runtime's unmodified adapter files and its hook entry when one exists. Copilot has no hook entry. It keeps a `CLAUDE.md` it did not create and leaves unrelated `.github` files alone. A modified adapter file is a conflict (exit 4).

{% cliUsage "When the project stops using a runtime.", "You." %}

```text
solomonar runtime remove <claude|codex|copilot> [--keep <path>]... [--overwrite <path>]... [-y] [--dry-run] [--json]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<claude\|codex\|copilot>` | Required. The runtime to remove. |
| `--keep` | Untrack one modified adapter file and leave it on disk; one path per flag. |
| `--overwrite` | Delete one modified adapter file; one path per flag. |
| `-y`, `--yes` | Accept the plan without prompts. |
| `--dry-run` | Print the plan and write nothing. |

### Example

```sh
solomonar runtime remove codex --dry-run
```

The output lists each adapter file to delete, as in the `runtime add` plan; nothing is written.
