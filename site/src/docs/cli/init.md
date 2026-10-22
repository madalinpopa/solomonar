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

{% cliUsage "Once per repository, before the first task: in a new project or in an existing repository. Run it again after a CLI update or to resolve a file conflict; a re-run with the same choices and unchanged files is a no-op, and it never changes a recorded choice.", "You, from your terminal. Agents do not set up projects." %}

## Synopsis

```text
solomonar init [<dir>] [--name <name>] [--objective <text>]
               [--runtime <claude|codex|none>]... [--ext <name|none>]...
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
| `--runtime` | `claude`, `codex`, or `none`; repeatable or comma-separated. Defaults to the runtimes found on `PATH`. |
| `--ext` | Extensions to install, repeatable or comma-separated, or `none`. The catalog holds `go`. Defaults to `go` when the repository root has a `go.mod`, otherwise none. |
| `--adversarial-model` | Model for the adversarial reviewer of one runtime, such as `claude=<model>`. |
| `--main-branch`, `--remote` | Defaults: `main` and `origin`. |
| `--keep`, `--overwrite` | Resolve one conflicting file; one path per flag. |
| `--git-init` | Run `git init -b <main-branch>` when the directory is not a repository. |
| `-y`, `--yes` | Accept the plan without prompts. Required without a terminal. |
| `--dry-run` | Print the plan and write nothing. |

## Prompts

In a terminal, without `--yes` or `--json`, `init` asks for each choice whose flag you did not pass, shows the plan, and asks `Proceed?` before it writes. Each choice of a stack extension is recorded as your approved decision in `docs/decisions.md` and `docs/stack.md`. See [Set up a project]({{ '/docs/set-up-a-project/' | url }}) for a walkthrough of the prompts.

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
  registry   create refs/heads/workflow-state with an empty state.json
Would initialize solomonar 0.1.3 in /work/ledger
  29 files created
```

Then set up the project for both runtimes, in a repository with an `origin` remote:

```sh
solomonar init --name ledger --objective "Track shared household expenses" \
  --runtime claude --runtime codex --ext none --yes
```

```text
Initialized solomonar 0.1.3 in /work/ledger
  39 files created
  registry  refs/heads/workflow-state @ 2be8870347892548d1c083c28a0e8c834d1327bb
  pushed    origin/workflow-state

Next:
  1. git add AGENTS.md CLAUDE.md .claude .agents .codex .solomonar docs && git commit -m "chore: initialize solomonar 0.1.3"
  2. Start a top-level session in /work/ledger with your selected outcome:
       Claude Code: /solomonar Explore the next useful flow.
       Codex:       $solomonar Explore the next useful flow.
       Portable:    Use the solomonar skill to explore the next useful flow.
```

Commit the files as shown before any task starts. Without a remote, `init` warns that the registry stays local.
