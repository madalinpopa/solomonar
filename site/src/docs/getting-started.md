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
  - docs/cli/implementation-tasks.md
---

Solomonar is a spec-driven workflow for building software with AI coding agents. You agree on what the software should do, an agent builds one focused task against that agreement, separate agents review it, and a small CLI records every step in Git.

It has two parts:

- **The framework:** Markdown rules, skills, and agent roles that live in your repository under `docs/solomonar/`. Your agent runtime, Claude Code or Codex, reads them.
- **The CLI:** the `solomonar` binary. It installs the framework into a project and keeps the task record honest. It never calls a model or starts an agent.

## Before you begin

You need:

- macOS or Linux. Windows is not supported in v0.1.
- Git 2.40 or later.
- Go 1.27.3 or later, to build the CLI from source.
- Claude Code or Codex, to run the agent sessions.

No model-provider API key is needed by the CLI itself.

## Your path through the docs

Read these in order. Each step ends where the next one starts.

1. [Install the CLI]({{ '/docs/install/' | url }}): install the binary with Homebrew or build it, and check it runs.
2. [Set up a project]({{ '/docs/set-up-a-project/' | url }}): run `solomonar init` in a repository and make the first commit.
3. [Run your first task]({{ '/docs/first-task/' | url }}): start an agent session, prototype a flow, and take one task from specification to merge.
4. [How the framework works]({{ '/docs/framework/' | url }}): the ideas behind what you just did.

After that, use the [workflow examples]({{ '/docs/use-cases/' | url }}) for everyday situations and the [CLI reference]({{ '/docs/cli/' | url }}) when you need a flag.

## What works today

The framework contracts are defined at v0.1.0. The CLI is built in increments:

| Area | Status |
| --- | --- |
| `init` with the core framework and Claude Code or Codex adapters | Available |
| Task commands: `plan`, `register`, `show`, `status`, `move`, `block`, `unblock`, `handoff`, `done` | Available |
| Interactive `init` prompts | Planned; use `--yes` or `--dry-run` |
| Extensions at `init` and `extension` commands | Planned |
| `doctor`, `sync`, `runtime`, `guard`, `upgrade`, `dashboard` | Planned |

The [CLI reference]({{ '/docs/cli/#planned-commands' | url }}) keeps the full list of planned commands.

## How these docs are organized

- **Start here:** the steps above, in order.
- **Learn the framework:** why the workflow is shaped the way it is.
- **Guides:** prompts and commands for specific situations, and fixes when something fails.
- **Reference:** every command, flag, file, and setting.
