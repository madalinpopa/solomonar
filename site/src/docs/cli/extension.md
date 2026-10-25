---
title: solomonar extension
navTitle: solomonar extension
description: List, add, and remove the catalog and project extensions of a Solomonar project with solomonar extension list, add, and remove, recording stack decisions as you go.
kind: Reference
group: Commands
order: 13.7
sources:
  - internal/cli/extension.go
  - internal/project/extension.go
  - internal/project/catalog.go
  - internal/project/detect.go
  - docs/cli/interface.md
---

An extension is one skill or one agent: the rules of a language, library, or practice, a supporting procedure, or an extra reviewer. Extensions are grouped by the type of decision that selects them, so the project's capabilities follow its approved stack. `ext` is an alias for `extension`.

The catalog embedded in the CLI holds two extensions:

| Extension | Kind | Group | Suggested | Invocation |
| --- | --- | --- | --- | --- |
| `go` | skill | `language` | when a `go.mod` exists anywhere in the repository | The model loads it when a change touches Go code |
| `extension-authoring` | skill | `workflow` | in every project | Only you start it |

`init` preselects the suggested extensions; afterwards, `extension list` marks the ones not installed as suggested. A suggestion never installs anything: an extension is installed only by your choice at `init` or by `extension add`. The catalog has no `review` extension; the project's own reviewers are project extensions.

## The store

Every extension a project uses lives in one flat directory per name, `docs/solomonar/extensions/<name>/`, whatever its group or kind. A repository never carries the whole catalog: the store holds only the installed catalog extensions and the project's own extensions.

- **Catalog extension.** `extension add` copies every file of its catalog directory, `extension.json` included, into the store, byte for byte, and records each file's hash in the manifest. `upgrade` keeps them current; `extension remove` deletes them.
- **Project extension.** A store directory whose name the catalog does not use. It is yours: the CLI reads it but never writes, hashes, or upgrades a file in it. `extension add <name>` activates it, and `extension remove <name>` deactivates it, leaving the directory as it was.

Both kinds use the same format: `extension.json`, then `SKILL.md` for a skill or `<name>.md` for an agent, then any support files. Activating an extension creates its runtime stubs, its line in the `## Extensions` section of `AGENTS.md`, its manifest entry, and, in a stack group, its decision. The decision appears in the `AGENTS.md` line and the manifest, never inside the extension's files.

{% diagram "extension-lifecycle" %}

### extension.json

The `go` entry of the catalog:

```json
{
  "name": "go",
  "kind": "skill",
  "group": "language",
  "title": "Go language rules",
  "requires": [],
  "conflicts": [],
  "applies_when": "A change touches Go code (*.go, go.mod, go.sum)",
  "paths": ["**/*.go", "**/go.mod", "**/go.sum"],
  "detect": ["**/go.mod"]
}
```

| Field | Meaning |
| --- | --- |
| `name` | The extension ID; matches the directory and the `name` in the main file's frontmatter. |
| `kind`, `group` | `skill` or `agent`, and one of the nine groups below. |
| `title` | One-line name, shown by the `init` prompt and `extension list --json`. |
| `requires`, `conflicts` | Extensions that must, or must not, be installed with it. |
| `applies_when` | For a skill, when an agent loads it; for an agent, when triage requires the reviewer. Written into `AGENTS.md`. |
| `paths` | Globs of the files the extension governs. Written into `AGENTS.md`. |
| `detect` | Optional. Globs of repository files that suggest the extension, such as `**/go.mod`. |
| `default` | Optional. `true` suggests the extension in every project. |
| `invocation` | Optional. `model` (the default) lets the model load the skill when it applies; `user` means only you start it. A user-invoked skill gets its own line in `AGENTS.md`, and its runtime stubs disable model invocation. Agents are always `model`. |

Detection runs over the files Git lists in the worktree, tracked and untracked but not ignored. `**` matches any number of directories, so `**/go.mod` matches a `go.mod` at the root or in any subdirectory.

## Groups

| Group | Selects | Records a decision |
| --- | --- | --- |
| `language`, `architecture`, `frontend`, `backend`, `persistence`, `testing`, `operations` | The stack | Yes: adopting or dropping one is an architecture or dependency decision, recorded in `docs/decisions.md` and `docs/stack.md` (`docs/architecture.md` for `architecture`) |
| `workflow` | Procedures that support the process | No |
| `review` | Extra specialist reviewers | No |

A stack extension's decision approves the technology and its rules. It approves no version and no further dependency; each one still needs its own approval.

## Write a project extension

Project extensions cover what the catalog does not: a domain rule set, an in-house library, a supporting procedure, or an extra reviewer. Install `extension-authoring` (it is suggested in every project), then ask in a top-level session:

> Use the extension-authoring skill to create a project reviewer for money handling.

The skill is user-invoked, so it runs only when you start it. Research, authoring, and validation are separate stages that share one evidence record. It writes the extension in catalog format to `docs/solomonar/extensions/<name>/`, `extension.json` included, validates it, and hands off the activation command, `solomonar extension add <name>`. In Solomonar's own source checkout, the same skill writes catalog contributions instead.

The skill owns the content; the CLI owns activation. The skill never edits the manifest, the stubs, or `AGENTS.md`.

## solomonar extension list

Read-only. Lists the catalog extensions and the project extensions in the store, by group, with their status, the decision that adopted them, and why one is suggested. It works before `init`, when every row is `available`.

{% cliUsage "Before you choose part of the stack, after writing a project extension, or to see which rules and reviewers apply to the project.", "You and agents." %}

```text
solomonar extension list [--group <group>] [--kind <skill|agent>] [--installed] [--json]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `--group` | Only one group. |
| `--kind` | Only skills or only agents. |
| `--installed` | Only installed extensions. |

`STATUS` is `installed`, `available`, or, for a project extension that fails the extension checks, `invalid: <reason>`. `SUGGESTED` shows, for an extension that is not installed, the first file that matched its `detect` globs or `default`.

### Example

These examples come from a run of the 0.2.0 build in the fictional `ledger` project, upgraded from 0.1.7 with `go` installed under `P/D-1`, for Claude Code and Codex; paths are shortened to `/work/ledger`.

```sh
solomonar extension list
```

```text
GROUP     NAME                 KIND   SOURCE   STATUS     DECISION  SUGGESTED  APPLIES WHEN
language  go                   skill  catalog  installed  P/D-1     -          A change touches Go code (*.go, go.mod, go.sum)
workflow  extension-authoring  skill  catalog  available  -         default    A request creates, changes, or validates a Solomonar extension skill, its rules, or a specialist reviewer
```

A project extension that fails the checks is listed with its first failure and cannot be added until you fix it:

```text
GROUP     NAME                 KIND   SOURCE   STATUS                            DECISION  SUGGESTED  APPLIES WHEN
language  go                   skill  catalog  installed                         P/D-1     -          A change touches Go code (*.go, go.mod, go.sum)
workflow  extension-authoring  skill  catalog  installed                         -         -          A request creates, changes, or validates a Solomonar extension skill, its rules, or a specialist reviewer
review    a11y-reviewer        agent  project  invalid: applies_when is missing  -         -          -
review    money-reviewer       agent  project  available                         -         -          A change computes, stores, converts, or displays monetary amounts
```

## solomonar extension add

Installs catalog extensions into the store, or activates project extensions already there, with their runtime stubs, `AGENTS.md` line, and manifest entry. It takes names, never paths: a name resolves in the catalog, then in the store. For a stack group, without `--decision` your command is the decision: the CLI appends a `P/D-n` entry to `docs/decisions.md` and a line to `docs/stack.md`, superseding the group's previous decision. Adding an installed extension refreshes its stubs; an identical result is a no-op.

{% cliUsage "When you decide part of the stack; when a task's approved decision adopts a technology; after the extension-authoring skill hands off a project extension; or to add a workflow procedure or an extra reviewer.", "You, to record a project decision. An orchestrator only with `--decision` naming an approved task decision." %}

```text
solomonar extension add <name>... [--decision <ID>] [--rationale <text>] [--keep <path>]... [--overwrite <path>]...
                        [-y|--yes] [--dry-run] [--json]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<name>` | Required. Catalog or project extension names; several are allowed. |
| `--decision` | An approved entry in `docs/decisions.md` that adopts the extension, such as `P/D-3`. Stack groups only. |
| `--rationale` | The rationale recorded with the new `P/D-n`. Stack groups only. |
| `--keep`, `--overwrite` | Resolve one conflicting file; one path per flag. |
| `-y`, `--yes` | Accept the plan without prompts. Required without a terminal. |
| `--dry-run` | Print the plan and write nothing. |

A project initialized with an older framework is rejected with `framework_outdated` (exit 3) until you run [`solomonar upgrade`]({{ '/docs/cli/upgrade/' | url }}):

```text
error: extension add rejected (framework_outdated)
  check: the project's framework matches this CLI
  found: manifest framework 0.1.7, this CLI ships 0.2.0
  fix:   run solomonar upgrade
nothing was written
```

### Examples

Add the catalog workflow skill `extension-authoring`. In a terminal the plan goes to stderr before the prompt:

```sh
solomonar extension add extension-authoring
```

```text
Plan
  framework 0.3.7
    update     AGENTS.md
  extensions
    create     docs/solomonar/extensions/extension-authoring/ (6 files)
  runtime claude
    create     .claude/skills/ (1 skill stub: 0 core, 1 extension)
  runtime codex
    create     .agents/skills/ (1 skill stub: 0 core, 1 extension; extension-authoring/agents/openai.yaml)
  manifest
    update     .solomonar/manifest.json
Proceed? [Y/n] y

Added extension extension-authoring (workflow skill, catalog) in /work/ledger
  9 files created, 2 updated, 51 unchanged, 1 skipped

Next:
  1. git add AGENTS.md .claude .agents .codex .solomonar docs && git commit -m "chore: add solomonar extension extension-authoring"
```

The skill then writes the project reviewer `money-reviewer` to `docs/solomonar/extensions/money-reviewer/`:

```text
docs/solomonar/extensions/money-reviewer/
  extension.json
  money-reviewer.md
```

```json
{
  "name": "money-reviewer",
  "kind": "agent",
  "group": "review",
  "title": "Money handling reviewer",
  "requires": [],
  "conflicts": [],
  "applies_when": "A change computes, stores, converts, or displays monetary amounts",
  "paths": ["internal/money/**"]
}
```

`extension list` shows it `available`. Activate it:

```sh
solomonar extension add money-reviewer --yes
```

```text
Added extension money-reviewer (review agent, project) in /work/ledger
  2 files created, 2 updated, 60 unchanged, 1 skipped

Next:
  1. git add AGENTS.md .claude .agents .codex .solomonar docs && git commit -m "chore: add solomonar extension money-reviewer"
```

The two created files are the role stubs `.claude/agents/money-reviewer.md` and `.codex/agents/money-reviewer.toml`; the store directory is unchanged, and the suggested `git add` includes it under `docs`. `AGENTS.md` now lists every installed extension, the user-invoked skill on its own line, and triage requires the reviewer when its condition matches:

```markdown
- Skills
  - `go` (language, P/D-1): A change touches Go code (*.go, go.mod, go.sum); paths: `**/*.go`, `**/go.mod`, `**/go.sum`
- Agents: an installed agent is a specialist reviewer, required by triage when its "applies when" matches.
  - `money-reviewer` (review): A change computes, stores, converts, or displays monetary amounts; paths: `internal/money/**`
- User-invoked skills: load one only when the user starts it, never because it applies.
  - `extension-authoring` (workflow): A request creates, changes, or validates a Solomonar extension skill, its rules, or a specialist reviewer
```

Adding an extension that is already installed changes nothing:

```text
Already installed: go; nothing to do.
  64 files unchanged, 1 skipped
```

In a project that has dropped Go (see [`extension remove`](#solomonar-extension-remove)), an orchestrator applying the project decision that adopts it again passes `--decision`; the CLI checks that the entry is approved and edits no entry itself:

```sh
solomonar extension add go --decision P/D-3
```

```text
error: extension add rejected (decision_not_approved)
  check: --decision must name an approved entry in docs/decisions.md
  found: P/D-3 has status pending
  fix:   wait for the user's approval (rules section 3), then rerun
nothing was written
```

Once you approve `P/D-3`, the same command with `--yes` installs `go` under it and appends `- language: go (P/D-3)` to `docs/stack.md`.

Other rejections: `unknown_extension` (exit 2) for a name in neither the catalog nor the store, `invalid_value` (exit 2) for a project extension that fails the checks, `name_collision` (exit 3) for a store directory that reuses a catalog name, and `extension_requires` or `extension_conflict` (exit 3) for an unmet `requires` or `conflicts`.

```text
error: extension add rejected (invalid_value)
  check: docs/solomonar/extensions/a11y-reviewer is a valid extension
  found: applies_when is missing
  fix:   fix docs/solomonar/extensions/a11y-reviewer/ and rerun; the extension-authoring skill validates it
nothing was written
```

## solomonar extension remove

Removes installed catalog extensions, deleting their store files, or deactivates project extensions, leaving their directory untouched. Both lose their stubs, their `AGENTS.md` line, and their manifest entry. For a stack group it follows the same decision rules as `extension add`; a group left empty records the selection `none`. A file you edited is a conflict (exit 4) until you pass `--keep` or `--overwrite`.

{% cliUsage "When the project drops a technology, practice, procedure, or reviewer. The CLI does not check whether code still uses the technology; it warns that the code loses its rules.", "You, or an orchestrator with an approved task decision, as for `extension add`." %}

```text
solomonar extension remove <name>... [--decision <ID>] [--rationale <text>] [--keep <path>]... [--overwrite <path>]...
                           [-y|--yes] [--dry-run] [--json]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<name>` | Required. Installed extensions to remove. Core skills and roles are rejected with `core_item`. |
| `--decision`, `--rationale` | As for `extension add`. Stack groups only. |
| `--keep` | Keep one edited store file in place and stop tracking it; one path per flag. |
| `--overwrite` | Delete one edited store file; one path per flag. |
| `-y`, `--yes` | Accept the plan without prompts. Required without a terminal. |
| `--dry-run` | Print the plan and write nothing. |

### Examples

Deactivate the project reviewer:

```sh
solomonar extension remove money-reviewer
```

```text
Plan
  framework 0.3.7
    update     AGENTS.md
  runtime claude
    remove     .claude/agents/ (1 role stub: 0 core, 1 extension)
  runtime codex
    remove     .codex/agents/ (1 role stub: 0 core, 1 extension)
  manifest
    update     .solomonar/manifest.json
Proceed? [Y/n] y

Removed extension money-reviewer (review agent, project) in /work/ledger
  2 files updated, 2 removed, 60 unchanged, 1 skipped

Next:
  1. git add AGENTS.md .claude .agents .codex .solomonar docs && git commit -m "chore: remove solomonar extension money-reviewer"
warning: code governed by money-reviewer loses its rules
```

`docs/solomonar/extensions/money-reviewer/` stays as it was, and `extension list` shows it `available` again.

Drop Go, with the reason recorded in the new decision:

```sh
solomonar extension remove go --rationale "The ledger moves to a hosted spreadsheet backend"
```

```text
Plan
  framework 0.3.7
    update     AGENTS.md
  extensions
    remove     docs/solomonar/extensions/go/ (6 files)
  decisions (approved, selected by you)
    update     docs/decisions.md  P/D-2 language: none · P/D-1 superseded
    append     docs/stack.md      language: none (P/D-2)
  runtime claude
    remove     .claude/skills/ (1 skill stub: 0 core, 1 extension)
  runtime codex
    remove     .agents/skills/ (1 skill stub: 0 core, 1 extension)
  manifest
    update     .solomonar/manifest.json
Proceed? [Y/n] y

Removed extension go (language skill, catalog, under P/D-2) in /work/ledger
  4 files updated, 8 removed, 50 unchanged, 1 skipped

Next:
  1. git add AGENTS.md .claude .agents .codex .solomonar docs && git commit -m "chore: remove solomonar extension go"
warning: code governed by go loses its rules
```

The new entry selects `none` and supersedes `P/D-1`, whose status becomes `superseded by P/D-2`:

```markdown
## P/D-2: Programming languages

- Status: approved
- Selected option: none
- Rejected options: none
- Scope: project; language extensions none
- Rationale: The ledger moves to a hosted spreadsheet backend
- Date: 2026-10-24
- References: supersedes P/D-1
```

`extension list` shows `go` `available` again, suggested by `go.mod`. A core skill or role cannot be removed:

```text
error: extension remove rejected (core_item)
  check: extension remove names an extension
  found: work-spec is part of the core
  fix:   the core cannot be removed (4.1)
nothing was written
```

The core is removed only with the whole installation, by [`solomonar uninstall`]({{ '/docs/cli/uninstall/' | url }}).
