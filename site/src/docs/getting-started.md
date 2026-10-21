---
title: Getting started
description: Build the current CLI from source, find your way around the project, and see what you can use today.
kind: Tutorial
group: Start here
order: 1
sources:
  - go.mod
  - Taskfile.yml
  - internal/cli/run.go
  - docs/cli/implementation-tasks.md
---

Solomonar is a lightweight, spec-driven workflow for people building software with AI coding agents. It gives a project shared specifications, focused skills, clear agent responsibilities, and review gates.

> **Current status:** the framework contracts are defined at v0.1.0. The CLI is being implemented incrementally. Project initialization and the complete managed-task workflow are not available yet. The examples below build and inspect the working CLI; they do not initialize a managed project.

## Before you begin

- Use macOS or Linux; Windows support is deferred in the v0.1 design.
- Install the Go toolchain required by [`go.mod`]({{ 'go.mod' | source }}), currently Go 1.27.3.
- Install Git 2.40 or later. The CLI uses your Git executable and configuration.

The CLI does not require a model-provider API key. The selected agent runtime runs sessions; the Solomonar binary does not call models or launch agents.

## Build from source

Clone the repository and build the binary:

```sh
git clone https://github.com/madalinpopa/solomonar.git
cd solomonar
go build -trimpath -o bin/solomonar .
```

The output is `bin/solomonar`. Keep the `./bin/` prefix in the examples below; no system-wide installation is needed.

## Check your build

```sh
./bin/solomonar version
./bin/solomonar help
./bin/solomonar task plan --help
```

`version` reports the CLI and embedded framework versions. A development checkout may report `devel` or a source-derived version, with `+dirty` for local changes; outside an initialized project, the project line reads `not initialized`.

For structured output:

```sh
./bin/solomonar version --json
```

You should receive one JSON object with `ok`, `command`, `data`, and `warnings` fields. See [the output contract]({{ '/docs/cli/#output-formats' | url }}).

## What works today

The current command table contains `help`, `version`, and these task commands: `plan`, `register`, `show`, `move`, `block`, `unblock`, `handoff`, and `done`.

Task commands require an initialized registry on `refs/heads/workflow-state`. Because `init` is still planned, these commands are useful for implementation and validation in prepared repositories. Do not create or edit the registry by hand to work around missing initialization.

The [CLI reference]({{ '/docs/cli/' | url }}) separates current commands from the rest of the interface design.

## Choose your next read

- New to the model? Read [the framework]({{ '/docs/framework/' | url }}) and its book, spells, dragons, and reins.
- Want to understand a task from start to finish? Follow [the workflow]({{ '/docs/workflow/' | url }}).
- Need flags or errors? Open [the CLI reference]({{ '/docs/cli/' | url }}) or [troubleshooting]({{ '/docs/troubleshooting/' | url }}).

For contributor checks, the repository's `Taskfile.yml` is authoritative. This website has its own [build and maintenance instructions]({{ 'site/README.md' | source }}).
