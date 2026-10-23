---
title: Project files and configuration
navTitle: Project files
description: Reference Solomonar files, ownership, Claude Code and Codex adapters, the Copilot CLI adapter contract, environment variables, and edit preservation.
kind: Reference
group: Reference
order: 12
sources:
  - docs/cli/interface.md
  - assets/templates/AGENTS.md
  - assets/framework/rules.md
  - internal/cli/move.go
  - internal/project/runtime.go
  - internal/project/validate.go
---

## Files init creates

`solomonar init` writes these files. The core and the adapters are managed: the manifest records their hashes so a later run can tell your edits from its own.

`init --runtime copilot` installs the Copilot adapters below. Copilot 1.0.85 discovered the instructions and all four core skills in a main checkout and feature worktree with all three adapters installed. Copilot 1.0.103 sessions run, but prompt mode cannot invoke the entry skill; delegation, reviewer tool behavior, model selection, and the minimum supported version remain unverified. See the [acceptance record]({{ 'docs/cli/interface.md' | source }}#copilot-runtime-acceptance).

| Path | Owns | Owner |
| --- | --- | --- |
| `AGENTS.md` | Compact mandatory instructions and context routes; your notes go under `## Project notes`. | Framework section managed; notes yours |
| `docs/README.md` | Project mission, objective, document index, and verified run and check commands. | You (seeded once) |
| `docs/solomonar/` | Framework rules, report format, the four skills, and the six agent roles. | Managed core |
| `docs/solomonar/skills/<extension>/` | The rules of an installed extension, such as `go`. | Managed extension |
| `docs/decisions.md`, `docs/stack.md` | One approved `P/D-n` entry per extension group you chose, and one stack line per group. Written only when you chose an extension. | Entries appended by `init`; the files are yours |
| `CLAUDE.md`, `.claude/skills/`, `.claude/agents/` | Claude Code adapter stubs. | Managed runtime files |
| `.agents/skills/`, `.codex/agents/` | Codex adapter stubs. | Managed runtime files |
| `.github/skills/<skill>/SKILL.md`, `.github/agents/<role>.agent.md` | Copilot CLI adapter stubs. | Managed runtime files, `runtime:copilot` |
| `.solomonar/manifest.json` | Your choices and the base hash of every managed file. | CLI only |
| `refs/heads/workflow-state` | The task registry: `state.json` with the schema version and one `tasks/<ID>.json` per task. | `solomonar task` commands only |

## Files your work creates

The skills and roles create these as the project grows. A small project does not need every one on day one.

| Path | Owns |
| --- | --- |
| `prototype/` | Clickable discovery flows; evidence, never approved requirements. |
| `docs/specs/` | Current capability behavior: agreed rules, observations, and open questions. |
| `docs/work/` | Active task specifications; completed specs remain in Git history. |
| `docs/architecture.md` | Approved structure, boundaries, and shared contracts. |
| `docs/stack.md` | Chosen technologies and rationale; manifests hold versions. |
| `docs/decisions.md` | Scoped approved, pending, and superseded decisions. |
| `docs/ui.md` | Shared UI and interaction rules when useful. |

## Runtime adapters

`init --runtime claude`, `--runtime codex`, and `--runtime copilot` install adapters. Each adapter file is a stub: the runtime's frontmatter plus one line that loads the portable file in `docs/solomonar/`. Edit the portable file, never the stub.

| Concern | Claude Code | Codex |
| --- | --- | --- |
| Root instructions | `CLAUDE.md` containing `@AGENTS.md`; created only when absent | `AGENTS.md`, read natively |
| Skills | `.claude/skills/<skill>/SKILL.md` | `.agents/skills/<skill>/SKILL.md` |
| Agent roles | `.claude/agents/<role>.md`; reviewers get a read-only tool list | `.codex/agents/<role>.toml` |
| Entry invocation | `/solomonar` | `$solomonar` |

The Copilot CLI adapter uses the following format. These fields were checked against GitHub's [CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference), [skill locations](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-skills), and [custom-agent configuration](https://docs.github.com/en/copilot/reference/custom-agents-configuration) on 2026-10-23. Live compatibility and the minimum supported Copilot version remain unverified.

| Concern | GitHub Copilot CLI |
| --- | --- |
| Root instructions | Native `AGENTS.md` loading; no additional root instruction file |
| Skills | `.github/skills/<skill>/SKILL.md`, pointing to the portable skill |
| Agent roles | `.github/agents/<role>.agent.md`, pointing to the portable role; `include-custom-instructions: true` opts delegated roles into repository instructions |
| Reviewer tools | `read`, `search`, `execute`; the implementer inherits available tools |
| Entry invocation | `/solomonar` in an interactive session; its stub sets `disable-model-invocation: true` so invocation remains yours, which also prevents prompt-mode (`-p`) invocation |
| Adversarial model | Optional `model` field on the adversarial reviewer only; omitted when unspecified |
| Ownership | `runtime:copilot` and a SHA-256 base hash for each adapter; no hook entry |

Copilot stubs cover core skills and delegated roles plus installed extension skills and reviewers. The orchestrator stays in the top-level session. Metadata is quoted as YAML strings so descriptions and model values preserve punctuation and cannot introduce extra fields. Portable instruction bodies remain in `docs/solomonar/`.

Reviewer tool restrictions exclude direct editing tools, but shell commands can still write. Reviewers must follow the portable read-only contract, including running checks that write files in an external temporary directory. No Copilot guard hook is generated, and IDE/cloud-agent support is outside this delivery.

The `solomonar` entry skill is invoked only by you, never by the model on its own. A portable request is “Use the solomonar skill.” The orchestrator has no adapter file: it is always the top-level session, because it must delegate the implementer and reviewers itself.

`--adversarial-model <runtime>=<model>` sets the model of that runtime's adversarial reviewer.

`--adversarial-model copilot=<model>` configures the Copilot adversarial-reviewer stub. This is a runtime preference: overrides or unavailable models can cause a different model to run. Installation does not authenticate or check account model access. Live acceptance must observe the actual model rather than treating the configuration as proof of model isolation.

## Environment variables

The CLI does not use a general configuration file. Global flags control each invocation; the interface defines two environment variables:

| Variable | Purpose |
| --- | --- |
| `SOLOMONAR_OWNER` | Default acting identity for owner commands such as `task move`, `block`, and `unblock`. |
| `NO_COLOR` | Disable color when set to any value. |

Agents pass `--as` on each owner command because a runtime may start a new shell for each invocation. Ownership validation prevents accidental writes from the wrong session; it is not authentication.

## Extensions and ownership

Extensions add skills and specialist reviewers. Stack extensions record an approved technology decision; workflow and review extensions configure the process without a stack decision.

In v0.1.6, `init --ext` installs extensions from the embedded catalog, which holds the `go` language extension. The `extension` commands that add or remove an extension later, and project extensions from a local directory, are planned.

The v0.1.6 catalog holds no specialist-reviewer extension; those arrive as project extensions. `task move --agent` already accepts the role of any agent extension the manifest lists as installed, alongside the core roles.

An installed extension is listed in the `## Extensions` section of `AGENTS.md` with its "applies when" condition and paths. Agents load it when a change matches, and cite its rules as `<name>/C-n`.

Project extensions will start as inactive drafts until adopted. Drafts do not contribute rules, runtime adapters, or required reviewers.

`init` and upgrades preserve project edits. Identical managed content is a no-op; differing content becomes an explicit conflict. A retained project edit does not become the distributed base hash. This allows a later upgrade to detect the difference again.

Copilot adapters use the same file classification and conflict rules, without taking ownership of unrelated `.github` files. The manifest stays at schema version 1; older binaries can reject a newer framework version or the unknown runtime. Do not hand-edit the manifest or lower its version. Adding Copilot to a project with other recorded runtimes remains deferred until `runtime add` is available; `init` never changes recorded runtime choices.

The full catalog and conflict-resolution flags remain in the [CLI interface specification]({{ 'docs/cli/interface.md' | source }}).
