---
title: Get started with Solomonar
navTitle: Get started
description: What Solomonar is, what you need before you start, and the four steps from a fresh repository to your first reviewed task with an AI coding agent.
kind: Overview
group: Start here
order: 1
sources:
  - README.md
  - internal/cli/run.go
---

Solomonar is a spec-driven workflow for building software with AI coding agents. You agree on what the software should do, an agent builds one focused task against that agreement, separate agents review it, and a small CLI records every step in Git.

It has two parts:

- **The framework:** Markdown rules, skills, and agent roles that live in your repository under `docs/solomonar/`. Your configured runtime, Claude Code, Codex, or GitHub Copilot CLI, reads them.
- **The CLI:** the `solomonar` binary. It installs the framework into a project and keeps the task record honest. It never calls a model or starts an agent.

## Before you begin

You need:

- macOS or Linux. Windows is not supported in v0.1.
- Git 2.40 or later.
- Go 1.27.3 or later, only to build the CLI from source.
- Claude Code, Codex, or GitHub Copilot CLI, to run the agent sessions.

No model-provider API key is needed by the CLI itself.

Copilot CLI adapter installation is available. Instruction and skill discovery were checked with Copilot 1.0.85, including a feature worktree and all three adapters together. Copilot 1.0.103 sessions run, but prompt mode (`copilot -p`) cannot invoke the entry skill. Delegation, reviewer tools, model selection, and the minimum supported version remain unverified. This covers the CLI only, not IDE or cloud-agent compatibility. See [the acceptance record]({{ 'docs/cli/interface.md' | source }}#copilot-runtime-acceptance).

Start Copilot interactively with `copilot` in the project directory, then invoke `/solomonar` in the top-level session. See [Set up a project]({{ '/docs/set-up-a-project/' | url }}) for installation and the printed commit command.

## Your path through the docs

Read these in order. Each step ends where the next one starts.

1. [Install the CLI]({{ '/docs/install/' | url }}): install the binary with Homebrew or build it, and check it runs.
2. [Set up a project]({{ '/docs/set-up-a-project/' | url }}): run `solomonar init` in a repository and make the first commit.
3. [Run your first task]({{ '/docs/first-task/' | url }}): start an agent session, prototype a flow, and take one task from specification to merge.
4. [How the framework works]({{ '/docs/framework/' | url }}): the ideas behind what you just did.

After that, use the [workflow examples]({{ '/docs/use-cases/' | url }}) for everyday situations and the [CLI reference]({{ '/docs/cli/' | url }}) when you need a flag.

## What works today

The CLI is built in increments. This table describes the source tree; check `solomonar help` and `solomonar init --help` to confirm availability in your installed build:

| Area | Status |
| --- | --- |
| `init`, interactive or with flags, with the core framework and Claude Code, Codex, or Copilot CLI adapters | Available; live Copilot workflow acceptance pending |
| Extensions at `init`: the `go` extension | Available |
| Task commands: `plan`, `register`, `show`, `status`, `move`, `block`, `unblock`, `handoff`, `confirm`, `migrate`, `done` | Available |
| `extension` commands, project extensions, and more catalog extensions | Planned |
| `doctor`, `sync`, `runtime`, `guard`, `upgrade`, `dashboard` | Planned |

The [CLI reference]({{ '/docs/cli/#planned-commands' | url }}) keeps the full list of planned commands.

## How these docs are organized

- **Start here:** the steps above, in order.
- **Learn the framework:** why the workflow is shaped the way it is.
- **Guides:** prompts and commands for specific situations, and fixes when something fails.
- **Reference:** every command, flag, file, and setting.
