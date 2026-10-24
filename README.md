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

Solomonar is an open-source workflow for building software with AI coding agents. You agree on what the software should do, an agent builds one focused task against that agreement, separate agents review it, and a small CLI records every step in Git.

It has two parts:

- **The framework:** Markdown rules, skills, and agent roles that `solomonar init` installs into your repository under `docs/solomonar/`. Your agent runtime reads them. The core works with any programming language; extensions add the rules for your stack.
- **The CLI:** the `solomonar` binary. It installs and upgrades the framework, adds and removes extensions, writes the runtime adapters, and validates every task update against Git. It never calls a model, starts an agent, or stores review verdicts.

![The Solomonar framework: the core in docs/solomonar/ holds the agent roles, the skills, and the contracts every role follows; optional extensions, the go and extension-authoring skills from the catalog or your own skills and reviewers, add stack rules, procedures, and extra reviewers, loaded when their applies_when matches the change or when you start them.](docs/images/framework-core.svg)

The name comes from Romanian folklore: a solomonar is a wizard who learns every secret from a single book and rides a dragon to command the storms. Solomonar gives your project its book (specs), its spells (skills), and its dragons (agents). Read [the story](https://solomonar.dev/docs/story/) behind the name.

## Features

- **Specs before code.** Rules are marked *Agreed*, *Observed*, or *Open*. Agents implement and review against agreed rules only.
- **One orchestrator per task.** The top-level session delegates the implementer and every reviewer, and owns the decisions it brings back to you.
- **Independent, risk-based review.** Security, correctness, and spec reviewers check the same committed revision in parallel. Risky changes also face an adversarial reviewer.
- **Parallel tasks in isolated worktrees.** Tasks start one at a time, run in parallel, and merge into `main` one at a time.
- **A task record that cannot drift silently.** State lives on a dedicated Git branch, is written only through checked CLI commands, and is compared with the real branch, worktree, and pull request on demand.
- **Resume from Git, not from a conversation.** You approve each task's implementation plan; every step becomes its own checkpoint commit. A new session continues the remaining steps from the plan and its commits.
- **Small current docs, full history.** Each finished task keeps a compact record; its final work spec stays in Git history.
- **Runtime-agnostic.** Claude Code, Codex, and GitHub Copilot CLI adapters point to one portable source of instructions. Copilot installation and discovery are verified; live workflow acceptance is pending.

## Requirements

- macOS or Linux. Windows is not supported in v0.1.
- Git 2.40 or later.
- Claude Code, Codex, or GitHub Copilot CLI, to run the agent sessions.
- Optional: the GitHub CLI (`gh`), so `task status` can report pull requests and checks.
- To build from source: Go 1.27.3 or later.

The CLI needs no model-provider API key.

Copilot CLI discovery was checked with versions 1.0.85 and 1.0.103. Invoke `/solomonar` in an interactive session; prompt mode (`copilot -p`) cannot invoke the entry skill. Delegation, reviewer tools, model selection, and the minimum supported Copilot version remain unverified. See the [acceptance record](docs/cli/interface.md#copilot-runtime-acceptance). Copilot adapter installation is available in this source tree; check `solomonar init --help` for `copilot` in an installed build. IDE and cloud-agent compatibility are outside this delivery.

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

`--ext go` installs the Go rules. Without `--ext`, `init` selects `extension-authoring` in every project and `go` when a `go.mod` exists; add or remove extensions later with `solomonar extension add|remove`, and move to a newer framework with `solomonar upgrade`.

Make the first commit, because task worktrees branch from `main`. `init` prints the exact command for the runtimes you chose:

```sh
git add AGENTS.md CLAUDE.md .claude .solomonar docs
git commit -m "chore: initialize solomonar 0.2.0"
```

Open a top-level agent session in the project and start every request with the `solomonar` skill:

| Runtime | Example request |
| --- | --- |
| Claude Code | `/solomonar Prototype adding a shared expense.` |
| Codex | `$solomonar Prototype adding a shared expense.` |
| GitHub Copilot CLI | `/solomonar Prototype adding a shared expense.` (live workflow acceptance pending) |
| Any runtime | `Use the solomonar skill to prototype adding a shared expense.` |

To install the Copilot adapter in a project with no recorded Solomonar choices, use `--runtime copilot` instead of `--runtime claude`. Follow the printed commit command: it lists individual Copilot files rather than the whole `.github` directory. Adding Copilot to an already initialized Solomonar project must wait for the planned `runtime add` command; `init` never changes recorded choices.

With no request, the skill reports what is waiting on you and what can start next. Follow the tasks from your shell:

```sh
solomonar task show
solomonar task status W-001
```

[Set up a project](https://solomonar.dev/docs/set-up-a-project/) covers existing repositories and file conflicts. [Run your first task](https://solomonar.dev/docs/first-task/) follows one feature from prototype to merge.

## Documentation

The website at [solomonar.dev](https://solomonar.dev) is the user documentation:

| Start here | Learn the framework | Guides | Reference |
| --- | --- | --- | --- |
| [Get started](https://solomonar.dev/docs/getting-started/) | [How it works](https://solomonar.dev/docs/framework/) | [Workflow examples](https://solomonar.dev/docs/use-cases/) | [CLI commands](https://solomonar.dev/docs/cli/) |
| [Install the CLI](https://solomonar.dev/docs/install/) | [The task lifecycle](https://solomonar.dev/docs/workflow/) | [Troubleshooting](https://solomonar.dev/docs/troubleshooting/) | [Project files](https://solomonar.dev/docs/configuration/) |
| [Set up a project](https://solomonar.dev/docs/set-up-a-project/) | [Task state](https://solomonar.dev/docs/task-state/) | | |
| [Run your first task](https://solomonar.dev/docs/first-task/) | [The story](https://solomonar.dev/docs/story/) | | |

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
| `internal/project` | The desired-state plan for `init`, `upgrade`, and `extension`, the manifest, managed files, adapters, the extension catalog and store. |
| `internal/git` | The Git runner and repository discovery. |
| `assets/framework` | The framework core that `init` installs. |
| `assets/extensions` | The extension catalog. |
| `assets/templates` | Templates for `AGENTS.md`, `docs/README.md`, and the empty registry. |
| `docs/` | Design contracts, diagrams, and the logo. |
| `site/` | The documentation website; see [`site/README.md`](site/README.md). |

## Contributing

Contributions are welcome. [`CONTRIBUTING.md`](CONTRIBUTING.md) covers the development setup, the documents to update with each change, commit conventions, and step-by-step guides with templates for adding framework rules, extension skills, and agents. For anything beyond a small fix, open an issue first to agree on the behavior.

The catalog holds `go` and the
[`extension-authoring` workflow skill](assets/extensions/workflow/extension-authoring/SKILL.md),
which `init` suggests in every project. Start it yourself: it researches, writes, and
validates catalog extensions here, or project extensions in an adopting repository's
`docs/solomonar/extensions/<name>/`, and hands off `solomonar extension add <name>`.
The CLI, not the skill, activates an extension and tracks it in the manifest.

To report a problem, include the output of `solomonar version`, the command, the error code, and the relevant Git state, with secrets redacted. The [troubleshooting guide](https://solomonar.dev/docs/troubleshooting/#report-a-problem-with-evidence) lists what helps. Report security problems privately as [`SECURITY.md`](SECURITY.md) describes.

### Releases

Maintainers release from `main` with `task release:patch`, `task release:minor`, or `task release:major`. The task updates framework version headers and current-build examples in the docs, site, and this README while preserving historical release and compatibility references. It commits those changes to `main`, then tags the commit and pushes the tag; the release workflow reruns CI, publishes the archives with GoReleaser, updates the Homebrew cask, and republishes the website.

## License

Solomonar is released under the [MIT License](LICENSE). Font and icon licenses for the website are listed in [`site/README.md`](site/README.md#design-and-asset-provenance).
