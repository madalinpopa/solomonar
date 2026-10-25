---
title: Set up a project
navTitle: Set up a project
description: Set up Solomonar with Claude Code, Codex, or Copilot CLI adapters, extensions, and the task registry in a Git repository, then make the first commit.
kind: Tutorial
group: Start here
order: 3
sources:
  - internal/cli/init.go
  - internal/project/core.go
  - docs/cli/interface.md
---

`solomonar init` installs the framework into one repository. It copies the rules, skills, and agent roles, writes small adapter files for your agent runtime, installs the extensions you choose, and creates the task registry. This page uses a new project called `ledger`; the same steps work in an existing repository.

## Answer the prompts, or pass flags

In a terminal, `solomonar init` with no flags asks for the project name, a one-line objective, the runtimes, a model for each runtime's adversarial reviewer, and the extensions for each decision group. It then shows a grouped plan and asks `Proceed?` before writing anything. Any choice you pass as a flag is not asked again.

The rest of this page passes every choice as a flag, so the same commands also work in a script or an agent's shell.

## Preview the plan

Start in an empty directory. `--git-init` creates the Git repository for you; inside an existing repository it is ignored.

```sh
mkdir ledger && cd ledger
solomonar init --git-init --runtime claude --runtime codex \
  --objective "Shared expenses for small groups" --dry-run
```

`--dry-run` prints every file `init` would write and writes nothing:

```text
Plan
  git        init -b main
  create     docs/solomonar/agents/adversarial-reviewer.md
  create     docs/solomonar/agents/correctness-reviewer.md
  ...
  create     .codex/agents/spec-reviewer.toml
  create     .solomonar/manifest.json
  registry   create refs/heads/workflow-state with an empty registry
Would initialize solomonar 0.3.7 in /work/ledger
  53 files created
```

Choose `--runtime claude`, `--runtime codex`, `--runtime copilot`, any combination, or `--runtime none` alone. Without the flag, a new project's defaults select the runtimes found on your `PATH`. Explicit selection does not require the executable installed; a rerun preserves the manifest's choices.

To follow this walkthrough with Copilot CLI alone, replace the two runtime flags in both the preview and initialization commands with `--runtime copilot`:

```sh
solomonar init --git-init --runtime copilot --ext none \
  --name ledger --objective "Shared expenses for small groups" --dry-run --yes
```

This plan creates 28 files, including `.github/skills/` and `.github/agents/` adapters. The [init reference]({{ '/docs/cli/init/' | url }}) shows abbreviated Copilot output. Run the preview in a project that has not recorded another Solomonar runtime selection.

## Choose extensions

Extensions add the rules of your stack, supporting procedures, and extra reviewers. The catalog holds `go`, the Go language rules, and `extension-authoring`, the skill that writes your project's own extensions. Without `--ext`, `init` selects the suggested ones: `extension-authoring` in every project, and `go` when a `go.mod` exists anywhere in the repository. The empty `ledger` directory therefore gets only `extension-authoring`. Pass `--ext go` to choose Go yourself, or `--ext none` for no extension.

Choosing a stack extension is an approved decision: `init` records it as `P/D-1` in `docs/decisions.md` and lists it in `docs/stack.md`. The extension installs instructions for the agents, never application packages.

A new project can also start with no stack extension and choose its stack after discovery: `solomonar extension list` shows what is available and suggested, and `solomonar extension add go` records your decision when you make it. See [`solomonar extension`]({{ '/docs/cli/extension/' | url }}).

## Initialize

Run the same command with `--yes` instead of `--dry-run`. In a script or an agent's shell, `init` refuses to write without `--yes`.

```sh
solomonar init --git-init --runtime claude --runtime codex \
  --objective "Shared expenses for small groups" --yes
```

```text
Initialized solomonar 0.3.7 in /work/ledger
  53 files created
  registry  refs/heads/workflow-state @ e8f817f0693ebfb7967375390242914ae750a2dd

Next:
  1. First commit required before task worktrees:
       git add AGENTS.md CLAUDE.md .claude .agents .codex .solomonar docs && git commit -m "chore: initialize solomonar 0.3.7"
  2. Start a top-level session in /work/ledger with your selected outcome:
       Claude Code: /solomonar Explore the next useful flow.
       Codex:       $solomonar Explore the next useful flow.
       Portable:    Use the solomonar skill to explore the next useful flow.
warning: no remote "origin"; the registry stays local
```

The warning is expected in a new repository. Once you add an `origin` remote, the CLI publishes the registry there after each task update.

## Make the first commit

Task worktrees branch from `main`, so `main` needs a commit. Run the command from step 1 of the output:

```sh
git add AGENTS.md CLAUDE.md .claude .agents .codex .solomonar docs
git commit -m "chore: initialize solomonar 0.3.7"
```

For a Copilot selection, use the printed command: it names each generated adapter file under `.github/skills/` and `.github/agents/`. Do not substitute `git add .github`, which can stage unrelated project files.

Start your selected runtime at the repository root. For Copilot CLI, run `copilot`, then enter `/solomonar Explore the next useful flow.` in its top-level session. The entry skill stays user-invoked; the orchestrator is intended to continue in that session and delegate the other roles. Prompt mode (`copilot -p`) cannot load the entry skill, so use an interactive session. See the [official CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference).

## What init created

| Path | What it is |
| --- | --- |
| `AGENTS.md` | Instructions every agent session reads first. |
| `docs/README.md` | Your project's mission, objective, document index, and run and check commands. You own it. |
| `docs/solomonar/` | The framework: rules, the four skills, and the six agent roles. Installed extensions live in `docs/solomonar/extensions/<name>/`. |
| `docs/decisions.md`, `docs/stack.md` | Your approved stack decisions, written only when you chose a stack extension such as `go`. |
| `CLAUDE.md`, `.claude/` | Claude Code adapters: an `@AGENTS.md` import, skill stubs, and agent stubs. |
| `.agents/`, `.codex/` | Codex adapters: skill stubs and agent definitions. |
| `.github/skills/<skill>/SKILL.md`, `.github/agents/<role>.agent.md` | Copilot CLI skill and delegated-role adapters, when selected; no extra root instruction file or guard hook. |
| `.solomonar/manifest.json` | Your choices and the hashes of the files `init` manages. |
| `refs/heads/workflow-state` | The task registry: a Git branch with `state.json` and one `tasks/<ID>.json` per task, never checked out. |

The adapter files are short stubs that point to `docs/solomonar/`. The real instructions live in one place, whatever runtime you use. The [configuration reference]({{ '/docs/configuration/' | url }}) lists every file and who owns it.

Open `docs/README.md` and fill in the mission and intended users. Agents read it at the start of each task.

## In an existing repository

Run `init` from anywhere inside the repository; it writes at the worktree root. Your files are safe:

- An existing `AGENTS.md` is adopted: the framework section goes on top and your content moves under `## Project notes`. Pass `--overwrite AGENTS.md` to replace it instead.
- An existing `CLAUDE.md` is never changed. If it lacks an `@AGENTS.md` line, `init` tells you to add one.
- An existing `docs/README.md` is left alone.
- Any other file that differs from what `init` would write stops the run with a conflict. Choose per file with `--keep <path>` or `--overwrite <path>`.

If the repository already has an application, agents discover its build and test commands from your scripts and manifests. You do not need to describe them up front.

## Running init again

`init` is safe to repeat. With the same choices and unchanged files, it does nothing:

```text
Already initialized with solomonar 0.3.7; nothing to do.
```

It never changes a recorded choice, such as the runtimes or the extensions. Change extensions with [`extension add` and `extension remove`]({{ '/docs/cli/extension/' | url }}); after a CLI update, move the framework with [`upgrade`]({{ '/docs/cli/upgrade/' | url }}). Change runtimes with [`runtime add` and `runtime remove`]({{ '/docs/cli/runtime/' | url }}); do not edit the manifest or pass different runtimes to `init`. For example, to add Copilot to a project initialized for Claude Code and Codex, or drop it again:

```sh
solomonar runtime add copilot --dry-run
solomonar runtime add copilot --yes
solomonar runtime remove copilot --yes
```

Each write prints the `git add` and `git commit` command for the changed adapter files and the manifest.

## Remove Solomonar from a project

[`solomonar uninstall`]({{ '/docs/cli/uninstall/' | url }}) removes what `init` installed from one checkout: the framework core, the catalog extensions, the adapters, the framework part of `AGENTS.md`, the framework line in `docs/README.md`, and the manifest. Your project notes, project extensions, decisions, specifications, and the task registry stay. Preview it with `--dry-run`, then review and commit the removal yourself.

## Next

The project is ready. [Run your first task]({{ '/docs/first-task/' | url }}).
