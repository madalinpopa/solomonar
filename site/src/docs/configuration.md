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
  - internal/cli/runtime.go
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
| `docs/solomonar/extensions/<name>/` | The store: an installed catalog extension, such as `go`, copied with its `extension.json`; or a project extension you or the `extension-authoring` skill wrote. | Catalog extensions managed, `ext:<name>`; project extensions yours |
| `docs/decisions.md`, `docs/stack.md` | One approved decision entry per change to a stack group's extensions, and one stack line per change. Written only when a stack extension is chosen, added, or removed. | Entries appended by `init` and `extension add\|remove`; the files are yours |
| `CLAUDE.md`, `.claude/skills/`, `.claude/agents/` | Claude Code adapter stubs. | Managed runtime files |
| `.agents/skills/`, `.codex/agents/` | Codex adapter stubs. | Managed runtime files |
| `.github/skills/<skill>/SKILL.md`, `.github/agents/<role>.agent.md` | Copilot CLI adapter stubs. | Managed runtime files, `runtime:copilot` |
| `.solomonar/manifest.json` | Your choices and the base hash of every managed file. | CLI only |
| `refs/heads/workflow-state` | The task registry: `state.json` with the schema version and one `tasks/<ID>.json` per task. | `solomonar task` commands only |

## Files your work creates

The skills and roles create these as the project grows. A small project does not need every one on day one.

| Path | Owns |
| --- | --- |
| `prototype/` | Independently runnable behavioral references for CLI, web, module/library, or service flows; approved scenarios define behavior. |
| `prototype/README.md` | Per-part run/check/reset instructions, stable flow IDs, scenario results, scoped approvals, and simulation limits. |
| `docs/specs/` | Current capability behavior: agreed rules, observations, and open questions. |
| `docs/work/` | Active task specifications; completed specs remain in Git history. |
| `docs/architecture.md` | Approved structure, boundaries, and shared contracts. |
| `docs/stack.md` | Chosen technologies and rationale; manifests hold versions. |
| `docs/decisions.md` | Scoped approved, pending, and superseded decisions. |
| `docs/ui.md` | Shared UI and interaction rules when useful. |

New prototype parts can be grouped by type under `prototype/cli/`, `web/`, `module/`, or `service/`; create only what the selected behavior needs. See [Prototype workflow]({{ '/docs/prototype/' | url }}) for outputs and examples. These project-owned artifacts are created through the skill, not by `init`, and are kept for future inspection.

## Runtime adapters

`init --runtime claude`, `--runtime codex`, and `--runtime copilot` install adapters. Each adapter file is a stub: the runtime's frontmatter plus one line that loads the portable file in `docs/solomonar/`. Edit the portable file, never the stub. Body-only edits apply when the runtime next loads that source. The name and description of a core or catalog extension stub come from the release, so editing them in `docs/solomonar/` changes nothing in the stub. After changing the name or description of a project extension, or after deleting a stub by mistake, run `solomonar upgrade`: at the project's version it regenerates the adapter files and restores missing managed files.

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

`--adversarial-model <runtime>=<model>` sets the model of that runtime's adversarial reviewer at `init`. Afterwards, `solomonar runtime add <runtime> --adversarial-model <model>` changes it.

### Changing runtimes

`init` records the runtimes once. [`runtime add`]({{ '/docs/cli/runtime/' | url }}#solomonar-runtime-add) generates another runtime's adapters for the core and the installed extensions, restores a missing adapter of an installed runtime, or updates its adversarial-reviewer model. [`runtime remove`]({{ '/docs/cli/runtime/' | url }}#solomonar-runtime-remove) deletes the runtime's unmodified adapter files and drops it from the manifest; an edited adapter is a conflict you resolve with `--keep` or `--overwrite`. Both keep the other runtimes, extensions, and decisions, and leave unrelated files under `.claude`, `.codex`, `.agents`, and `.github`, a `CLAUDE.md` the CLI did not create, and the top-level adapter directories in place. No runtime command installs, merges, or removes a hook or a runtime settings file; hook management is deferred with `guard`.

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

Every extension the project uses lives in the store, `docs/solomonar/extensions/<name>/`. `init --ext` and [`extension add`]({{ '/docs/cli/extension/#solomonar-extension-add' | url }}) copy a catalog extension there, `extension.json` included, and the manifest tracks each of its files as `ext:<name>`; `upgrade` keeps them current and `extension remove` deletes them. The catalog holds the `go` language extension and the `extension-authoring` workflow skill.

A project extension is a store directory whose name the catalog does not use. It is project-owned: no command writes, tracks, or upgrades a file in it. `extension add <name>` activates it with runtime stubs, an `AGENTS.md` line, a manifest entry with `source: path`, and, in a stack group, a decision; `extension remove <name>` deactivates it and leaves the directory as it was.

`extension-authoring` researches, writes, and validates skills, rules, or specialist reviewers. It supports catalog source in Solomonar and project extensions in adopting repositories. It is user-invoked: its runtime stubs disable model invocation, and `AGENTS.md` lists it under "User-invoked skills", so it runs only when you start it. Its stubs point to `docs/solomonar/extensions/extension-authoring/SKILL.md`; conditional stage references and one authoring-evidence source map live beside that file. Each authored extension keeps its own sources, decisions, and results in a single research record.

The skill creates content; the CLI activates it. In project mode the skill writes the extension in catalog format to the store, validates it, and hands off `solomonar extension add <name>`. It never edits the manifest, the runtime stubs, or `AGENTS.md`.

The catalog holds no specialist-reviewer extension; those arrive as project extensions in the `review` group. `task move --agent` accepts the role of any agent extension the manifest lists as installed, alongside the core roles.

An installed extension is listed in the `## Extensions` section of `AGENTS.md` with its group, its decision for a stack group, its "applies when" condition, and its paths. Agents load a model-invoked skill when a change matches, and cite its rules as `<name>/C-n`; triage requires an installed agent when its condition matches.

A project extension that is not installed contributes no rules, runtime adapters, or required reviewers. `extension list` shows it `available`, or `invalid: <reason>` when it fails the extension checks.

`init`, `upgrade`, `uninstall`, and the `runtime` and `extension` commands preserve project edits. Identical managed content is a no-op; differing content becomes an explicit conflict. A retained project edit does not become the distributed base hash. This allows a later upgrade to detect the difference again.

Copilot adapters use the same file classification and conflict rules, without taking ownership of unrelated `.github` files. The manifest stays at schema version 1; older binaries can reject a newer framework version or the unknown runtime. Do not hand-edit the manifest or lower its version. `init` never changes recorded runtime choices; add Copilot to a project with other recorded runtimes with `runtime add copilot`.

The full catalog and conflict-resolution flags remain in the [CLI interface specification]({{ 'docs/cli/interface.md' | source }}).

## Removing the installation

[`solomonar uninstall`]({{ '/docs/cli/uninstall/' | url }}) follows the ownership in the tables above. It removes the managed files the manifest records and that are still unchanged: the core, catalog extensions in the store, the runtime stubs, a `CLAUDE.md` the CLI created, the managed part of `AGENTS.md`, and the manifest. In `docs/README.md` it removes only the framework index line. An edited managed file is a conflict you resolve with `--keep` or `--overwrite`.

Everything you own stays: the `## Project notes` section of `AGENTS.md`, project extensions in `docs/solomonar/extensions/<name>/`, `docs/decisions.md` and `docs/stack.md` with every entry, the documents your work created, a `CLAUDE.md` you wrote, other files in `.claude/`, `.codex/`, `.agents/`, and `.github/`, and the registry `refs/heads/workflow-state` with its task history. Mentions of Solomonar in those files are yours to edit; the command warns only about the ones it knows of. It does not touch other worktrees, the binary, or user-wide configuration.
