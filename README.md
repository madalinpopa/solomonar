<h1 align="center">
  <img src="docs/logo/lockup.svg" alt="solomonar" width="560">
</h1>

<p align="center">
  Spec-driven development for AI coding agents.<br>
  Shared specifications, focused tasks, independent reviews, and verified code.
</p>

<p align="center">
  <a href="https://github.com/madalinpopa/solomonar/actions/workflows/ci-solomonar.yml"><img src="https://github.com/madalinpopa/solomonar/actions/workflows/ci-solomonar.yml/badge.svg?branch=main" alt="CI"></a>
  <a href="https://github.com/madalinpopa/solomonar/releases/latest"><img src="https://img.shields.io/github/v/release/madalinpopa/solomonar" alt="Latest release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/madalinpopa/solomonar" alt="MIT license"></a>
</p>

<p align="center">
  <a href="https://solomonar.dev">Website</a> ·
  <a href="https://solomonar.dev/docs/getting-started/">Documentation</a> ·
  <a href="https://solomonar.dev/docs/cli/">CLI reference</a> ·
  <a href="https://github.com/madalinpopa/solomonar/releases">Releases</a>
</p>

## What is Solomonar?

Solomonar is an open-source workflow for building software with AI coding agents such as Claude Code and Codex. You agree on what the software should do, an agent builds one focused task against that agreement, separate agents review it, and a small CLI records every step in Git.

It has two parts:

- **The framework:** Markdown rules, skills, and agent roles that `solomonar init` installs into your repository under `docs/solomonar/`. Your agent runtime reads them. The core names no language; extensions add the rules of your stack.
- **The CLI:** the `solomonar` binary. It installs the framework, writes the runtime adapters, and validates every task update against Git. It never calls a model, starts an agent, or stores review verdicts.

The name comes from Romanian folklore: a solomonar is a wizard who learns every secret from a single book and rides a dragon to command the storms. Solomonar gives your project its book (specs), its spells (skills), and its dragons (agents). Read [the story](https://solomonar.dev/docs/story/) behind the name.

## Features

- **Specs before code.** Rules are marked *Agreed*, *Observed*, or *Open*. Agents implement and review against agreed rules only.
- **One orchestrator per task.** The top-level session delegates the implementer and every reviewer, and owns the decisions it brings back to you.
- **Independent, risk-based review.** Security, correctness, and spec reviewers check the same committed revision in parallel. Risky changes also face an adversarial reviewer.
- **Parallel tasks in isolated worktrees.** Tasks start one at a time, run in parallel, and merge into `main` one at a time.
- **A task record that cannot drift silently.** State lives on a dedicated Git branch, is written only through checked CLI commands, and is compared with the real branch, worktree, and pull request on demand.
- **Small current docs, full history.** Each finished task keeps a compact record; its final work spec stays in Git history.
- **Runtime-agnostic.** Adapters for Claude Code and Codex point to one portable source of instructions.

## How it works

![The Solomonar cycle: prototype and spec discover the product, solomonar task register and work-spec prepare the task, then implementation, tiered reviews, PR with merge, and solomonar task done; CHANGES_REQUIRED sends the task back to implementation.](docs/images/workflow-cycle.svg)

1. **Discover.** The `prototype` skill builds a small clickable flow with simulated data, so you can try behavior before you specify it.
2. **Specify.** The `spec` skill records the behavior you confirm in `docs/specs/`.
3. **Prepare one task.** The orchestrator registers the task with the CLI, creates its branch and worktree, and the `work-spec` skill writes its outcome, scope, and acceptance criteria.
4. **Implement and review.** The implementer hands over a committed revision. Reviewers matched to the task's risk tier check it; confirmed defects go back to implementation.
5. **Integrate and close.** The pull request merges with a merge commit, and `solomonar task done` verifies the merge before writing the final record.

**The orchestrator decides; the CLI checks.** The [framework guide](https://solomonar.dev/docs/framework/) explains each part with the full set of diagrams, and [the task lifecycle](https://solomonar.dev/docs/workflow/) follows one task stage by stage.

## Requirements

- macOS or Linux. Windows is not supported in v0.1.
- Git 2.40 or later.
- Claude Code or Codex, to run the agent sessions.
- Optional: the GitHub CLI (`gh`), so `task status` can report pull requests and checks.
- To build from source: Go 1.27.3 or later.

The CLI needs no model-provider API key.

## Installation

With Homebrew, on macOS and Linux:

```sh
brew install --cask madalinpopa/tap/solomonar
```

With Go:

```sh
go install github.com/madalinpopa/solomonar@latest
```

Prebuilt archives for macOS and Linux (amd64, arm64) are attached to each [GitHub release](https://github.com/madalinpopa/solomonar/releases). To build from source, see [Development](#development).

Check the installation:

```sh
solomonar version
```

See [Install the CLI](https://solomonar.dev/docs/install/) for upgrades and the JSON output.

## Quick start

Initialize a project. In a terminal, `solomonar init` asks for the project's choices and shows the plan before it writes anything. In a script, pass the choices as flags with `--yes`:

```sh
mkdir ledger && cd ledger
solomonar init --git-init --runtime claude --ext go \
  --objective "Shared expenses for small groups" --yes
```

Make the first commit, because task worktrees branch from `main`. `init` prints the exact command for the runtimes you chose:

```sh
git add AGENTS.md CLAUDE.md .claude .solomonar docs
git commit -m "chore: initialize solomonar 0.1.3"
```

Open a top-level agent session in the project and start every request with the `solomonar` skill:

| Runtime | Example request |
| --- | --- |
| Claude Code | `/solomonar Prototype adding a shared expense.` |
| Codex | `$solomonar Prototype adding a shared expense.` |
| Any runtime | `Use the solomonar skill to prototype adding a shared expense.` |

With no request, the skill reports what is waiting on you and what can start next. Follow the tasks from your shell:

```sh
solomonar task show
solomonar task status W-001
```

[Set up a project](https://solomonar.dev/docs/set-up-a-project/) covers existing repositories and file conflicts. [Run your first task](https://solomonar.dev/docs/first-task/) follows one feature from prototype to merge.

## CLI commands

| Command | Purpose |
| --- | --- |
| `init` | Install the framework, runtime adapters, and extensions, and create the task registry. |
| `version` | Print the CLI and project framework versions. |
| `task plan` | Create a planned task under the next free ID. |
| `task register` | Reserve a task for one orchestrator. |
| `task show` | Print the registry overview or one stored record. |
| `task status` | Compare a task's record with Git and forge facts, and list the valid next commands. |
| `task move` | Apply a validated status, stage, or field change. |
| `task block`, `task unblock` | Add or clear a blocker. |
| `task handoff` | Transfer a task to a new owner on explicit instruction. |
| `task done` | Verify the merge and final spec, then write the compact record. |

Agents run the writing commands. Every command accepts `--json` for one machine-readable object on stdout, and `solomonar help <command>` prints its flags. See the [CLI reference](https://solomonar.dev/docs/cli/) and [troubleshooting](https://solomonar.dev/docs/troubleshooting/).

## Project status

Version 0.1.3 is the current release. The framework contracts are defined, and the CLI ships increment 1: `init` (interactive or with flags, with Claude Code and Codex adapters and the `go` extension) and the full set of task commands.

Planned for later increments: `doctor`, `sync`, `runtime`, `extension` with project extensions from a local directory, and `guard` with its runtime hooks, then `upgrade` and a read-only `dashboard`. The `htmx`, `templ`, and `tdd` extensions join the catalog once their framework content is written. Their [command pages](https://solomonar.dev/docs/cli/#commands-in-development) describe them today; a command is available only when `solomonar help` shows it.

## Documentation

The website at [solomonar.dev](https://solomonar.dev) is the user documentation:

| Start here | Learn the framework | Guides | Reference |
| --- | --- | --- | --- |
| [Get started](https://solomonar.dev/docs/getting-started/) | [How it works](https://solomonar.dev/docs/framework/) | [Workflow examples](https://solomonar.dev/docs/use-cases/) | [CLI commands](https://solomonar.dev/docs/cli/) |
| [Install the CLI](https://solomonar.dev/docs/install/) | [The task lifecycle](https://solomonar.dev/docs/workflow/) | [Troubleshooting](https://solomonar.dev/docs/troubleshooting/) | [Project files](https://solomonar.dev/docs/configuration/) |
| [Set up a project](https://solomonar.dev/docs/set-up-a-project/) | [Task state](https://solomonar.dev/docs/task-state/) | | |
| [Run your first task](https://solomonar.dev/docs/first-task/) | [The story](https://solomonar.dev/docs/story/) | | |

The design contracts live in this repository:

| Document | Owns |
| --- | --- |
| [`assets/framework/`](assets/framework/) | Framework behavior: rules, skills, and agent roles, installed as `docs/solomonar/`. |
| [`site/src/docs/cli/`](site/src/docs/cli/) | The CLI reference: each command's synopsis, flags, output, and exit codes. |
| [`docs/cli/interface.md`](docs/cli/interface.md) | The contract beneath the CLI reference: check order, file planning, the manifest, JSON data, and task validation. |
| [`docs/cli/architecture.md`](docs/cli/architecture.md) | How the binary is built: packages, decisions, and conventions. |
| [`docs/design/`](docs/design/) | Framework design notes and validation. |

## Development

You need Go 1.27.3 or later and [Task](https://taskfile.dev). Linters are pinned and installed into `.bin/` on first use.

```sh
git clone https://github.com/madalinpopa/solomonar.git
cd solomonar
task build        # builds bin/solomonar
task check        # lint, tests with the race detector, and govulncheck, as CI runs them
```

| Task | Purpose |
| --- | --- |
| `task build` | Build `bin/solomonar`. |
| `task run -- <args>` | Run the CLI from source, for example `task run -- task show`. |
| `task test` | Run all tests with the race detector. |
| `task cover` | Run the tests and print total coverage. |
| `task fmt` | Format the code. |
| `task lint` | Check that `go.mod` is tidy and run golangci-lint. |
| `task security` | Report known vulnerabilities reachable from the code. |
| `task check` | Run lint, tests, and security checks. |

Without Task, `go build -trimpath -o bin/solomonar .` and `go test -race ./...` work directly.

### Repository layout

| Path | Contents |
| --- | --- |
| `main.go` | Entry point; embeds `assets/` into the binary. |
| `internal/cli` | Commands, flags, prompts, help, and output rendering. |
| `internal/task` | The task registry model, transitions, and fact checks. |
| `internal/project` | The init plan, manifest, managed files, adapters, and extension catalog. |
| `internal/git` | The Git runner and repository discovery. |
| `assets/framework` | The framework core that `init` installs. |
| `assets/extensions` | The extension catalog. |
| `assets/templates` | Templates for `AGENTS.md`, `docs/README.md`, and the empty registry. |
| `docs/` | Design contracts, diagrams, and the logo. |
| `site/` | The documentation website; see [`site/README.md`](site/README.md). |

### Specs lead the code

Behavior is specified before it is built. The interface specification fixes what the CLI does, the architecture document fixes how it is built, and the implementation plan breaks the work into reviewed stages. When an implementation decision departs from a specification, the same change updates the specification, so the documents always describe the code.

## Contributing

Contributions are welcome. Before you open a pull request:

1. For anything beyond a small fix, open an issue first to agree on the behavior.
2. Check the [command page](site/src/docs/cli/) and the [interface specification](docs/cli/interface.md) for the behavior you change, and update them, the [architecture](docs/cli/architecture.md), or the rest of the [site documentation](site/src/docs/) in the same pull request when they no longer match.
3. Add or update tests, and run `task check` until it passes.
4. Write commit messages in [Conventional Commits](https://www.conventionalcommits.org/) form, such as `feat(cli): ...`, `fix(task): ...`, or `docs(site): ...`, with one logical change per commit.

To report a problem, include the output of `solomonar version`, the command, the error code, and the relevant Git state, with secrets redacted. The [troubleshooting guide](https://solomonar.dev/docs/troubleshooting/#report-a-problem-with-evidence) lists what helps. Report security problems privately as [`SECURITY.md`](SECURITY.md) describes.

### Releases

Maintainers release from `main` with `task release:patch`, `task release:minor`, or `task release:major`. The task sets the next version in the framework assets, docs, site, and this README, commits that to `main`, then tags it and pushes the tag; the release workflow reruns CI, publishes the archives with GoReleaser, updates the Homebrew cask, and republishes the website.

## License

Solomonar is released under the [MIT License](LICENSE). Font and icon licenses for the website are listed in [`site/README.md`](site/README.md#design-and-asset-provenance).
