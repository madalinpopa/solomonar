---
title: solomonar init
navTitle: solomonar init
description: Set up Solomonar in a repository with solomonar init. Install the framework core, runtime adapters, and extensions, and create the task registry.
kind: Reference
group: Commands
order: 13
sources:
  - internal/cli/init.go
  - internal/cli/init_plan.go
  - internal/cli/init_prompt.go
  - docs/cli/interface.md
---

Installs the framework core into the worktree containing `<dir>` (default: the current directory), generates the adapter files of the selected runtimes, installs the chosen extensions, writes `.solomonar/manifest.json`, and creates the registry branch `refs/heads/workflow-state` when it is absent.

{% cliUsage "Once per repository, before the first task: in a new project or in an existing repository. Run it again to resolve a file conflict; after a CLI update, run `upgrade` instead. A re-run with the same choices and unchanged files is a no-op, and it never changes a recorded choice.", "You, from your terminal. Agents do not set up projects." %}

## Synopsis

```text
solomonar init [<dir>] [--name <name>] [--objective <text>]
               [--runtime <claude|codex|copilot|none>]... [--ext <name|none>]...
               [--adversarial-model <runtime>=<model>]...
               [--main-branch <name>] [--remote <name>]
               [--keep <path>]... [--overwrite <path>]...
               [--git-init] [-y|--yes] [--dry-run] [--json]
```

## Flags

| Flag | Meaning |
| --- | --- |
| `<dir>` | Optional. The directory to set up; defaults to the current directory. |
| `--name` | Project name; defaults to the directory name. |
| `--objective` | Current objective written to `docs/README.md`. |
| `--runtime` | `claude`, `codex`, `copilot`, or `none`; repeatable or comma-separated. For a new project, defaults to runtimes found on `PATH`; a rerun keeps the manifest's choices. Explicit selection works without the executable installed. `none` must be used alone. |
| `--ext` | Catalog extensions to install, repeatable or comma-separated, or `none`. The catalog holds `go` and `extension-authoring`. For a new project, defaults to the suggested extensions: `extension-authoring` always, and `go` when a `go.mod` exists anywhere in the repository, tracked or untracked but not ignored. A rerun keeps the manifest's extensions. Activate project extensions after `init` with [`extension add`]({{ '/docs/cli/extension/#solomonar-extension-add' | url }}). |
| `--adversarial-model` | Model for the adversarial reviewer of a selected runtime, such as `claude=<model>` or `copilot=<model>`. Repeatable; the last value for each runtime wins. |
| `--main-branch`, `--remote` | Defaults: `main` and `origin`. |
| `--keep`, `--overwrite` | Resolve one conflicting file; one path per flag. |
| `--git-init` | Run `git init -b <main-branch>` when the directory is not a repository. |
| `-y`, `--yes` | Accept the plan without prompts. Required without a terminal. |
| `--dry-run` | Print the plan and write nothing. |

## Prompts

In a terminal, without `--yes` or `--json`, `init` asks for each choice whose flag you did not pass, shows the plan, and asks `Proceed?` before it writes. Each choice of a stack extension is recorded as your approved decision in `docs/decisions.md` and `docs/stack.md`. See [Set up a project]({{ '/docs/set-up-a-project/' | url }}) for a walkthrough of the prompts.

The runtime choices are Claude Code, Codex, and GitHub Copilot CLI, in that order. Selected runtimes get an adversarial-model prompt unless you supplied `--adversarial-model`. JSON never prompts; non-interactive writes require `--yes`.

## Examples

These examples come from a run of the current build in the fictional `ledger` project; paths are shortened to `/work/ledger`.

Preview the plan first; `--dry-run` writes nothing:

```sh
solomonar init --name ledger --runtime claude --ext none --git-init --dry-run --yes
```

```text
Plan
  git        init -b main
  create     docs/solomonar/agents/adversarial-reviewer.md
  create     docs/solomonar/agents/correctness-reviewer.md
  ...
  create     AGENTS.md
  create     docs/README.md
  create     CLAUDE.md
  create     .claude/skills/solomonar/SKILL.md
  ...
  create     .solomonar/manifest.json
  registry   create refs/heads/workflow-state with an empty registry
Would initialize solomonar 0.3.2 in /work/ledger
  33 files created
```

Then set up the project for Claude Code and Codex, in a repository with an `origin` remote:

```sh
solomonar init --name ledger --objective "Track shared household expenses" \
  --runtime claude --runtime codex --ext none --yes
```

```text
Initialized solomonar 0.3.2 in /work/ledger
  43 files created
  registry  refs/heads/workflow-state @ 2be8870347892548d1c083c28a0e8c834d1327bb
  pushed    origin/workflow-state

Next:
  1. git add AGENTS.md CLAUDE.md .claude .agents .codex .solomonar docs && git commit -m "chore: initialize solomonar 0.3.2"
  2. Start a top-level session in /work/ledger with your selected outcome:
       Claude Code: /solomonar Explore the next useful flow.
       Codex:       $solomonar Explore the next useful flow.
       Portable:    Use the solomonar skill to explore the next useful flow.
```

Commit the files as shown before any task starts. Without a remote, `init` warns that the registry stays local.

For Copilot CLI alone, preview installation in a Git repository with no recorded Solomonar choices:

```sh
solomonar init --name ledger --runtime copilot --ext none --dry-run --yes
```

Abbreviated stdout; the full plan lists every file:

```text
Plan
  create     docs/solomonar/agents/adversarial-reviewer.md
  ...
  create     .github/skills/prototype/SKILL.md
  create     .github/skills/solomonar/SKILL.md
  create     .github/skills/spec/SKILL.md
  create     .github/skills/work-spec/SKILL.md
  create     .github/agents/adversarial-reviewer.agent.md
  ...
  create     .github/agents/spec-reviewer.agent.md
  create     .solomonar/manifest.json
  registry   create refs/heads/workflow-state with an empty registry
Would initialize solomonar 0.3.2 in /work/ledger
  32 files created
```

Remove `--dry-run` to install, then follow the printed commit command. It names individual Copilot adapter files rather than the whole `.github` directory, keeping unrelated GitHub configuration out of the suggested commit. Start `copilot` at the repository root and enter `/solomonar Explore the next useful flow.` in the top-level session. The invocation and discovery paths follow GitHub's [official CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference). Copilot 1.0.85 passed local discovery checks, but account policy rejected a live session; full runtime acceptance and the minimum supported version remain pending. See the [acceptance record]({{ 'docs/cli/interface.md' | source }}#copilot-runtime-acceptance).

You can combine runtimes, for example `--runtime codex,copilot`. JSON reports the sorted selection in `data.choices.runtimes`, each adapter with owner `runtime:copilot` in `data.files`, and the same staging paths in `data.next`. Add `--adversarial-model copilot=<model>` to request an adversarial model; installation does not authenticate or check model access. Copilot may override or fall back from that preference. No Copilot hook is installed.

An unchanged rerun is a no-op. To change extensions later, use [`extension add` and `extension remove`]({{ '/docs/cli/extension/' | url }}); after a CLI update, move the framework with [`upgrade`]({{ '/docs/cli/upgrade/' | url }}). `init` cannot add Copilot to a project with different recorded runtime choices; use [`runtime add copilot`]({{ '/docs/cli/runtime/' | url }}#solomonar-runtime-add) instead, and [`runtime remove`]({{ '/docs/cli/runtime/' | url }}#solomonar-runtime-remove) to drop a runtime. Copilot adapter collisions use the same `--keep`/`--overwrite` rules as other managed files. See [Project files and configuration]({{ '/docs/configuration/' | url }}).
