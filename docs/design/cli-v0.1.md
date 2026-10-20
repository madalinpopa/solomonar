# Solomonar CLI v0.1: interface specification

Status: design candidate, 2026-10-20. Phase 2 reference for the `solomonar` command-line interface: every command,
its arguments, its interaction, and its output, written as use cases. Behavior that the framework already fixes
(`assets/framework/rules.md` §7, register sections G and J) is restated here only as interface; when the two differ,
`rules.md` wins until §11 below is applied. Example SHAs, hashes, and paths are illustrative but have their real
length and shape.

Contents: 1 Principles · 2 Command map · 3 Conventions · 4 Project choices and files · 5 Use cases: project setup ·
6 Use cases: task state · 7 Use cases: runtime integration · 8 Reference: task validation · 9 Reference: `task
status` · 10 Delivery order · 11 Framework changes required · 12 Open decisions

## 1. Principles

- **Guard and oracle, never a driver** (J1). The CLI validates and records what the orchestrator or the user
  decided, and reports what Git shows. It never launches agents, calls model providers, stores review verdicts or
  reports, or runs a daemon. The only long-running command is the read-only `dashboard`.
- **One writer for state.** Only `solomonar task` commands write `refs/heads/workflow-state:state.json`, each
  write one compare-and-swap commit (rules §7).
- **Project files are copied, then owned.** `init` and the commands that add content copy embedded assets into
  the project and record each managed file in a manifest. A repeated operation with identical content is a no-op;
  a differing file is a conflict, never a silent overwrite.
- **Light core, opt-in extensions.** The framework core is only the process: its rules, roles, and skills. Technology
  knowledge, supporting procedures, and extra reviewers are extensions, skills or agents, that the user chooses at
  `init` or adds later (4.1).
- **Portable source, runtime adapters.** `docs/framework/` is the single source for rules, roles, skills, and
  extensions. A runtime (Claude Code, Codex) gets small generated adapter files that point to that source; it
  never gets a second copy of the content.
- **Actionable rejections.** A rejected command names the failed check and the corrective action, writes nothing,
  and exits non-zero.
- **Same data for people and agents.** Every command has a human form and a `--json` form with the same content.
- **No network except Git and `gh`.** Network use is limited to `git push` of `workflow-state` and read-only `gh`
  queries. Both are optional; their failures are warnings with an explicit `unknown`, never a pass.

## 2. Command map

| Command | Purpose | Writes | Use case |
|---|---|---|---|
| `solomonar init [<dir>]` | Install the framework core, the selected runtimes and extensions; create the registry | project files, manifest, `workflow-state` | UC-1 to UC-4 |
| `solomonar runtime list\|add\|remove` | Manage runtime adapters (`claude`, `codex`) | adapter files, manifest, `AGENTS.md` | UC-5 |
| `solomonar extension list\|show\|add\|remove\|new` (alias `ext`) | Manage extensions: skills and agents grouped by decision type; record the decision that adopts a stack extension | `docs/framework/skills/`, `docs/framework/agents/`, `docs/decisions.md`, `docs/stack.md`, `docs/architecture.md`, adapter files, manifest, `AGENTS.md` | UC-6, UC-7 |
| `solomonar sync [--check]` | Regenerate adapter files and rendered sections from the portable source | adapter files, manifest | UC-8 |
| `solomonar upgrade` | Move the project to the framework version embedded in this CLI | managed files, manifest | UC-9 |
| `solomonar doctor` | Check repository, registry, files, adapters, and forge settings | nothing | UC-10 |
| `solomonar task plan` | User: create a `planned` record | `workflow-state` | UC-11 |
| `solomonar task register` | Orchestrator: claim or allocate a task; the registration barrier | `workflow-state` | UC-12 |
| `solomonar task move` | Owner: validated transition and field update | `workflow-state` | UC-13, UC-14 |
| `solomonar task block` / `unblock` | Owner: add or clear a blocker | `workflow-state` | UC-15 |
| `solomonar task show` | Read the stored record or registry | nothing | UC-16 |
| `solomonar task status` | Read the record, Git and forge facts, drift, allowed next commands | nothing | UC-16, UC-17 |
| `solomonar task handoff` | New owner, on user instruction: ownership change | `workflow-state` | UC-19 |
| `solomonar task done` | Owner: verify the merge and write the compact record | `workflow-state` | UC-20 |
| `solomonar guard` | Runtime hook: deny direct registry writes | nothing | UC-22 |
| `solomonar dashboard` | Serve the read-only registry view | nothing | UC-23 |
| `solomonar version` | Print CLI, embedded framework, and project framework versions | nothing | UC-24 |
| `solomonar help [<command>]` | Print help; `-h` on any command is equivalent | nothing | - |

"Runtime" is the agent product that runs sessions (Claude Code, Codex). "Agent" and "role" keep their framework
meaning (orchestrator, implementer, reviewers), so the CLI never uses `agent` for a runtime.

## 3. Conventions

### 3.1 Invocation and global flags

```
solomonar [global flags] <command> [<subcommand>] [flags] [args]
```

| Flag | Default | Meaning |
|---|---|---|
| `-C <path>` | current directory | Run as if started in `<path>`, as `git -C` does. Any worktree of the repository works. |
| `--json` | off | Machine output on stdout (3.4). Never prompts: a command that would prompt needs `--yes` or fails with `interactive_required`. |
| `--no-color` | color when stdout is a TTY | Disable color. `NO_COLOR` set to any value has the same effect. |
| `-v`, `--verbose` | off | Also print each Git and `gh` command run, on stderr. |
| `-h`, `--help` | - | Help for the command. |

Command flags shared by several commands:

| Flag | Commands | Meaning |
|---|---|---|
| `-y`, `--yes` | `init`, `runtime`, `extension`, `upgrade` | Accept the plan without the confirmation prompt; with `init`, take defaults for every unanswered choice. |
| `--dry-run` | every writing command | Run every check and print the plan or the record that would be written; write nothing; exit as the real run would. |
| `--as <owner>` | `task move`, `block`, `unblock`, `done` | Acting orchestrator, checked against the record's `owner` (CLI-D2). Defaults to `SOLOMONAR_OWNER`. Agents pass `--as` on every command, because a runtime may start a new shell per command and lose the variable; the variable is a convenience for the user's terminal. The check guards against mistakes (wrong task, a stale session after a handoff), not against impersonation. |

Environment variables: `SOLOMONAR_OWNER` (default for `--as`), `NO_COLOR`. No other environment variable changes
behavior.

### 3.2 Repository context

- The CLI locates the repository with `git rev-parse --git-common-dir`, so it works from the main checkout or any
  feature worktree. The registry ref is shared by all worktrees.
- Project settings (main branch, remote) come from `.solomonar/manifest.json` in the current worktree; defaults
  are `main` and `origin`.
- Requirements: Git 2.40 or later; `gh` optional. A colocated Jujutsu repository (`.jj/`) is a warning: v0.1 is
  Git-only (C6), and jj commands must not be used on `workflow-state`.

### 3.3 Human output

- Results go to stdout; warnings and errors go to stderr, prefixed `warning:` or `error:`.
- Full SHAs appear wherever a value will be copied into another command; a short SHA (7) appears only beside
  a full one or in tables.
- File plans use one verb per line: `create`, `update`, `unchanged`, `keep` (project-modified, left as is),
  `remove`, `conflict`, `skip` (project-owned, never touched).
- Every writing command ends with what was written (commit, files) and, when useful, a `Next:` block.

### 3.4 JSON output

Every command with `--json` prints exactly one JSON object on stdout:

```json
{
  "ok": true,
  "command": "task move",
  "data": { },
  "warnings": [{"code": "push_failed", "message": "git push origin workflow-state failed: could not resolve host"}]
}
```

On failure `ok` is `false`, `data` is absent, and `error` is present:

```json
{
  "ok": false,
  "command": "task move",
  "error": {
    "code": "revision_not_on_branch",
    "message": "revision 1902e3d6fc4e78a0bcc50ba12b882769afbf4a8c is not reachable from feature/W-001",
    "check": "--revision must be a full SHA reachable from feature/W-001",
    "fix": "Commit the candidate on feature/W-001 and pass its full SHA: git -C <worktree> rev-parse HEAD"
  },
  "warnings": []
}
```

`data` per command is given in its use case. Field names are `snake_case`; timestamps are RFC 3339 UTC; SHAs are full.

### 3.5 Exit codes

| Code | Meaning | Written |
|---|---|---|
| 0 | Success, including a no-op and success with warnings (for example a failed push) | as reported |
| 1 | Unexpected failure: I/O, Git error, corrupt object | nothing; a partial file write is rolled back |
| 2 | Usage error: unknown command or flag, missing or malformed argument | nothing |
| 3 | Rejected: a validation, ownership, transition, or fact check failed | nothing |
| 4 | Conflict: a managed file differs from what the CLI would write, or the registry compare-and-swap kept losing | nothing |
| 5 | Checks failed: `doctor` found a `fail`, or `sync --check` found stale files | nothing |

### 3.6 Errors

Human form, on stderr:

```
error: task move W-001 rejected (revision_not_on_branch)
  check: --revision must be a full SHA reachable from feature/W-001
  found: 1902e3d6fc4e78a0bcc50ba12b882769afbf4a8c is not on feature/W-001
  fix:   commit the candidate on feature/W-001 and pass its full SHA: git -C <worktree> rev-parse HEAD
nothing was written
```

The error code catalog for task commands is in 8.4; project commands use `not_a_repository`, `not_initialized`,
`already_initialized`, `file_conflict`, `unknown_runtime`, `unknown_extension`, `core_item`,
`name_collision`, `invalid_value`, `unknown_group`, `extension_requires`, `extension_conflict`, `decision_not_found`,
`decision_not_approved`, `registry_exists_invalid`, `manifest_invalid`, `interactive_required`,
`checks_failed`.

### 3.7 Prompts

- Prompts appear only when stdin and stdout are terminals and neither `--yes` nor `--json` is set. Otherwise a
  missing required choice is error `interactive_required` (exit 2) naming the flag that supplies it.
- Prompts are line-based (no full-screen UI, CLI-D5): a text prompt shows its default in brackets; a multi-select
  shows a numbered list, accepts numbers separated by commas or spaces to toggle, `all`, `none`, or Enter to accept
  the marked defaults; a confirmation is `[Y/n]` or `[y/N]`, default capitalized.

## 4. Project choices and files

### 4.1 Core and extensions

The framework core holds only the process. Every other capability is an extension, a skill or an agent, that the
user chooses at `init` or adds later. The CLI installs, renders, and removes extensions; the core never depends on
one being present.

| Part | Contains | Installed |
|---|---|---|
| Core: the process | `docs/framework/rules.md`, `report.md`, `reviewer.md`; the process roles `orchestrator`, `implementer`, `code-reviewer`, `security-reviewer`, `spec-reviewer`, `adversarial-reviewer`; the process skills `prototype`, `spec`, `work-spec`; `AGENTS.md`; `docs/README.md`; the empty registry on `refs/heads/workflow-state`; `.solomonar/manifest.json` | always; cannot be removed |
| Runtimes | Adapter files for `claude`, `codex` (4.2) | zero or more; default: those found on `PATH` |
| Extensions | Skills (knowledge of a language, library, or practice; or a supporting procedure) and agents (extra specialist reviewers), grouped by decision type (4.3) | zero or more; default: `go` when `go.mod` exists |

Core boundary (CLI-D14, decided): the core is the current framework in `assets/framework/`, the items `rules.md`
names as a step or a required role of the process. The core carries no technology-specific content: the Go specifics of today's `go-reviewer` move to the `go`
extension, and `code-reviewer` stays language-neutral (CLI-D1).

A project whose stack is not known yet starts with no stack extension: the stack is decided later (rules §3), and
`solomonar extension add` follows that decision (UC-6).

### 4.2 Runtime adapters

An adapter adapts the portable framework to one runtime: it registers the instructions, the skills, and the roles
where the runtime discovers them, and restricts tools where the runtime allows. Adapter files are generated stubs:
the runtime frontmatter plus a one-line body that loads the portable file, so the content lives only in
`docs/framework/` (CLI-D7). Runtime paths and formats in this table must be verified against each runtime's
documentation before implementation (G-register note; CLI-D9).

| Concern | `claude` (Claude Code) | `codex` (Codex) |
|---|---|---|
| Root instructions | `CLAUDE.md` containing `@AGENTS.md`, created only when absent (CLI-D3) | `AGENTS.md`, read natively; nothing generated |
| Skills | `.claude/skills/<skill>/SKILL.md` stub per installed skill | `.agents/skills/<skill>/SKILL.md` stub per installed skill |
| Roles | `.claude/agents/<role>.md` per delegated role; reviewers get a read-only tool list; `adversarial-reviewer` gets the configured model | native role definitions where supported (CLI-D9); otherwise none, and packets name the role file |
| Orchestrator | no native agent file: it is a top-level session (G13) | same |
| State guard | `PreToolUse` hook running `solomonar guard --runtime claude`, merged into `.claude/settings.json` | hook where supported (CLI-D9); otherwise none, and `doctor` reports it |
| Adversarial model | `--adversarial-model claude=<model>` writes `model:` in its role stub | `--adversarial-model codex=<model>` where native roles exist |

Skill stub (Claude Code; Codex identical under `.agents/skills/`):

```markdown
---
name: work-spec
description: Prepares or refines the specification of one registered task (feature, bug fix, refactor, or technical task) in `docs/work/<ID>-<slug>.md`, before or during its implementation. Use it in a task's `specification` stage, or when that task's scope or acceptance criteria must change. Do not use it to maintain the application specifications (use `spec`) or for prototyping (use `prototype`).
---

Generated by solomonar 0.1.0 from `docs/framework/skills/work-spec/SKILL.md`; edit that file, then run `solomonar sync`.
Load `docs/framework/skills/work-spec/SKILL.md` and follow it.
```

Role stub (Claude Code):

```markdown
---
name: security-reviewer
description: Reviews one candidate revision for security defects in authorization, data isolation, external input, files, secrets, trust boundaries, and security-relevant dependency or configuration changes.
tools: Read, Grep, Glob, Bash
---

Generated by solomonar 0.1.0 from `docs/framework/agents/security-reviewer.md`; edit that file, then run `solomonar sync`.
Load `AGENTS.md`, then `docs/framework/agents/security-reviewer.md` and every file its first lines require.
```

Tool lists: `implementer` inherits the session's tools; every reviewer gets `Read, Grep, Glob, Bash` (read-only for
tracked files is a behavioral contract, rules §11; `Bash` is needed to run checks).

### 4.3 Extensions

An extension is one skill or one agent, plus catalog metadata the CLI uses to install and render it. Extensions are
grouped by the type of decision that selects them, so the project's capabilities follow its approved architecture
and stack: choosing htmx and templ for the frontend installs the htmx and templ skills, the templ rules that only
apply together with htmx, and nothing about React.

| Kind | Adds | Installed to | Runtime adapter |
|---|---|---|---|
| `skill` | Knowledge (the rules of a language, library, or practice) or a supporting procedure | `docs/framework/skills/<name>/` | a skill stub per runtime (4.2) |
| `agent` | An extra specialist reviewer, with the condition that makes it required | `docs/framework/agents/<name>.md` | a role stub per runtime, read-only tools |

| Group | Decision type | Records a decision | Document the CLI updates |
|---|---|---|---|
| `language` | Programming languages | yes | `docs/stack.md` |
| `architecture` | Structural style and module boundaries | yes | `docs/architecture.md` |
| `frontend` | UI rendering and interaction | yes | `docs/stack.md` |
| `backend` | HTTP serving and API style | yes | `docs/stack.md` |
| `persistence` | Data store and data access | yes | `docs/stack.md` |
| `testing` | Test practice and tools beyond the language | yes | `docs/stack.md` |
| `operations` | Build, packaging, deployment | yes | `docs/stack.md` |
| `workflow` | Procedures that support the process (for example recording a baseline) | no | - |
| `review` | Extra specialist reviewers (for example accessibility) | no | - |

The groups are fixed. The first seven are stack groups: adopting or dropping one of their extensions is an
architecture or dependency decision. `workflow` and `review` configure how the process runs and record no decision.
A group without catalog extensions takes project extensions (UC-7); prompts and `extension list` show only groups
that have at least one extension.

**Catalog source.** Each catalog extension is a directory embedded in the CLI,
`assets/extensions/<group>/<name>/`, holding `extension.json` and the skill or agent files:

```json
{
  "name": "templ",
  "kind": "skill",
  "group": "frontend",
  "title": "templ components",
  "requires": ["go"],
  "conflicts": [],
  "applies_when": "A change touches *.templ files or the handlers that render them",
  "paths": ["**/*.templ"]
}
```

```markdown
---
name: templ
description: Rules for templ components in this project. Use when a change touches `*.templ` files or the handlers that render them.
---

# templ components

## Rules

- templ/C-1 (must): `*_templ.go` files are generated by `templ generate` and never edited by hand; CI fails when
  they are stale.
- templ/C-2 (must): User data is never rendered through `templ.Raw` or any other unescaped output.
- templ/C-3 (should): Components take view models built for the page, not domain entities or database rows.

## With htmx
<!-- solomonar:with htmx -->

- templ/C-4 (must): A handler that serves both full navigation and htmx requests renders the fragment component
  when the `HX-Request` header is set and the page component otherwise, from the same component tree.
```

| Field | Meaning | CLI behavior |
|---|---|---|
| `name`, `group` | Extension ID and its group; `name` matches the directory and the skill or agent `name` | `unknown_extension`, `unknown_group` |
| `kind` | `skill` or `agent` | decides the install path and the adapter stub |
| `title` | One-line name | shown by `extension list` and `extension show` |
| `requires` | Extensions that must be installed with it | `extension_requires` (exit 3) naming the missing ones; never added silently |
| `conflicts` | Extensions that cannot be installed with it | `extension_conflict` (exit 3) |
| `applies_when` | Skill: when an agent loads it. Agent: when triage requires it | rendered into `AGENTS.md` (4.6) and into the skill's `description` |
| `paths` | Globs of files the extension governs | rendered into `AGENTS.md`; the orchestrator uses them to name extensions in a packet; the CLI never inspects a change |

**Generation.** The CLI renders each skill or agent file for the installed set: a `## With <name>` section marked
`<!-- solomonar:with <name>[,<name>...] -->` is kept only when every named extension is installed, and dropped
otherwise. Adding or removing an extension re-renders every installed one that refers to it. Rendering is
deterministic templating from embedded files; it never calls a model (J1). The rendered file gets one line under its
title: `Installed by P/D-2; rendered with: htmx.`

**Loading.** Runtimes trigger extension skills by their `description`, but the process does not rely on that: the
orchestrator names the installed extensions whose `applies_when` or `paths` match the change in every packet, the
implementer loads them before writing code, and reviewers check their `must` rules. Triage lists installed extension
agents beside the core specialists, required when their `applies_when` matches (rules §6).

**Rule IDs and force** (CLI-D12). Extension rules have stable IDs `<name>/C-n`, never reused across versions. `must`
rules are approved global rules (rules §1 authority 2) once the decision that installed the extension is approved; a
violation is a blocking finding (rules §6). `should` rules are suggestions. Findings cite them as they cite
`<capability>/R-n`.

**What a stack extension approves.** The decision that installs it approves the technology or practice and its
rules. It approves no version (the manifests hold versions) and no further dependency: each one still follows
rules §3.

**Decision recording** (CLI-D11). Every change to the installed set of a stack group is tied to an approved decision
entry:

- With `--decision <ID>`, the CLI checks that `docs/decisions.md` contains that entry with status `approved`
  (`decision_not_found`, `decision_not_approved`, exit 3), and records the ID with the extensions. This is the path
  for an orchestrator applying a task decision such as `W-002/D-1`.
- Without it, the user's command is the decision: the CLI appends a project-scoped entry `P/D-n` per affected group
  with status `approved`. When the group already has an approved `P/D-n`, the new entry lists the full new
  selection and sets the old one to `superseded`, naming its replacement (rules §3).
- In both cases the CLI appends one line per group to the group's document (table above), creating the document
  with a title and an index line in `docs/README.md` when absent. It never edits other entries or lines.

Entry written by the CLI, in the format it also reads for `--decision` (section 11):

```markdown
## P/D-2: Frontend technologies

- Status: approved
- Selected option: htmx, templ
- Rejected options: none
- Scope: project; frontend extensions `htmx`, `templ`
- Rationale: selected by the user with `solomonar init`
- Date: 2026-10-20
- References: `docs/framework/skills/htmx/SKILL.md`, `docs/framework/skills/templ/SKILL.md`
```

**Commands in extensions.** Extensions list typical checks (`go test ./...`, `go vet ./...`, `templ generate`,
`govulncheck ./...`) as suggestions only. No extension writes commands into `docs/README.md`; they stay "not
established" until the implementer records the real ones (rules §11).

### 4.4 Extension catalog v0.1

Decided in CLI-D13. The rule summaries show intent; each extension's content is written and reviewed as framework
content before release.

| Extension | Kind | Group | Requires | Applies when | Main rules | Renders with |
|---|---|---|---|---|---|---|
| `go` | skill | `language` | - | A change touches Go code (`**/*.go`) | Go Code Review Comments; errors wrapped with context and checked; `context.Context` first and never stored; goroutines with an owner and a stop path; transactions closed on every path; `govulncheck` for dependency changes (suggested check) | `tdd`: test layout rules |
| `htmx` | skill | `frontend` | - | A change touches `hx-*` attributes or endpoints that return fragments | Endpoints targeted by `hx-*` return HTML fragments, never JSON; state changes use `hx-post`, `hx-put`, `hx-patch`, `hx-delete`, never `hx-get`, and carry the CSRF token; a URL reached by full navigation or `hx-push-url` renders the whole page without htmx; error responses render a fragment the page can show | `templ` |
| `templ` | skill | `frontend` | `go` | A change touches `*.templ` files or the handlers that render them | Generated `*_templ.go` files never edited by hand and checked current in CI; no `templ.Raw` with user data; components take page view models, not domain entities | `htmx`: one handler renders the fragment on `HX-Request`, the page otherwise |
| `tdd` | skill | `testing` | - | A change adds or changes behavior | A failing test for each behavior AC before the production code, its failing run recorded in "Checks and evidence"; red, green, refactor, with refactoring only on green; a test that weakens or deletes an assertion needs an AC or a decision; tests cite rule IDs (`AGENTS.md` rule 13). Turns "where practical" (G9) into `must`, unless the work spec states why a test first is impossible | `go`: table tests with one `t.Run` per case |

No `workflow` or `review` extension is in the v0.1 catalog (CLI-D10); both groups accept project extensions.

### 4.5 File ownership

| Class | Files | `init` | `upgrade` | `sync` |
|---|---|---|---|---|
| `core` | `docs/framework/**` except extensions | create | update if unmodified, else conflict | - |
| `ext:<name>` | a catalog extension's skill directory or agent file | create, rendered | as `core`, rendered for the installed set | re-render when the installed set changes |
| `project-ext:<name>` | extensions added from a local directory or created with `extension new` | create | never | never |
| `decision` entries | `P/D-n` entries in `docs/decisions.md`; one line per group in `docs/stack.md` or `docs/architecture.md` | append; create the file when absent | never | never |
| `core+notes` | `AGENTS.md` | create, or adopt an existing file (UC-4) | the part above `## Project notes` as `core`; the section from that heading on is project-owned | rendered lists (4.6) |
| `runtime:<name>` | adapter stubs, `CLAUDE.md` when created | create | regenerate | regenerate |
| `runtime:<name>` entry | the guard hook entry inside `.claude/settings.json` | merge the entry; never touch other keys | merge | merge |
| `seed` | `docs/README.md` | create when absent, else `skip`; append a "Documents" index line when the CLI creates a document | never | never |
| `manifest` | `.solomonar/manifest.json` | create | update | update |

"Unmodified" means the file's SHA-256 equals the hash recorded in the manifest.

### 4.6 Rendered `AGENTS.md`

The managed part of `AGENTS.md` is rendered from the choices: the framework version, the core "Skills" list, and an
"Extensions" section grouped like `extension list`:

```markdown
## Extensions

Installed with `solomonar`. Load every extension skill whose "applies when" matches your change, before writing or
reviewing code, and cite its rules as `<name>/C-n`. The orchestrator names the matching extensions in each packet.

- Skills
  - `go` (language): a change touches Go code (`**/*.go`)
  - `htmx` (frontend): `hx-*` attributes or endpoints that return fragments
  - `templ` (frontend): `*.templ` files or the handlers that render them (`**/*.templ`)
  - `tdd` (testing): a change adds or changes behavior
- Agents: none. An installed agent is a specialist reviewer, required by triage when its "applies when" matches.
```

### 4.7 Manifest

`.solomonar/manifest.json` is committed on main, written only by the CLI, and records the choices and every managed
file. `extensions` maps a name to its kind, group, source (`catalog`, or `path` with the repo-relative source
directory), the decision that installed it (stack groups only), and the extensions it was rendered with; `files`
keys are repo-relative paths:

```json
{
  "schema_version": 1,
  "framework_version": "0.1.0",
  "project": {"name": "ledger", "main_branch": "main", "remote": "origin"},
  "runtimes": {
    "claude": {"adversarial_model": "opus"},
    "codex": {}
  },
  "extensions": {
    "go": {"kind": "skill", "group": "language", "source": "catalog", "decision": "P/D-1", "rendered_with": ["tdd"]},
    "htmx": {"kind": "skill", "group": "frontend", "source": "catalog", "decision": "P/D-2", "rendered_with": ["templ"]},
    "templ": {"kind": "skill", "group": "frontend", "source": "catalog", "decision": "P/D-2", "rendered_with": ["htmx"]},
    "tdd": {"kind": "skill", "group": "testing", "source": "catalog", "decision": "P/D-3", "rendered_with": ["go"]}
  },
  "files": {
    "AGENTS.md": {"owner": "core+notes", "sha256": "<64 hex, managed part only>"},
    "docs/framework/rules.md": {"owner": "core", "sha256": "<64 hex>"},
    "docs/framework/skills/templ/SKILL.md": {"owner": "ext:templ", "sha256": "<64 hex>"},
    "docs/decisions.md": {"owner": "decision", "entries": ["P/D-1", "P/D-2", "P/D-3"]},
    ".claude/skills/templ/SKILL.md": {"owner": "runtime:claude", "sha256": "<64 hex>"},
    ".claude/settings.json": {"owner": "runtime:claude", "entry": "hooks.PreToolUse/solomonar-guard"},
    "docs/README.md": {"owner": "seed"}
  }
}
```

## 5. Use cases: project setup

### UC-1 Initialize a new project interactively

Actor: user. Preconditions: a Git repository with at least one commit on `main`; the CLI is installed; no
`.solomonar/manifest.json`.

```
$ cd /work/ledger
$ solomonar init
solomonar 0.1.0 · framework 0.1.0 · /work/ledger (git, branch main)

Project name [ledger]:
Current objective, one line (optional): Track shared household expenses

Runtimes that will run this project's sessions:
  [x] 1  claude   Claude Code: CLAUDE.md import, .claude/skills, .claude/agents, state guard hook
  [ ] 2  codex    Codex: AGENTS.md (native), .agents/skills
Toggle numbers, Enter to accept: 1 2

Model for the adversarial reviewer in Claude Code, different from your main session model
(blank = runtime default): opus

Extensions, by decision type. A choice in a stack group is recorded as your approved decision in
docs/decisions.md. Enter skips a group; add more later with `solomonar extension add`.

Language:
  [ ] 1  go      Go code
Toggle numbers, Enter to accept: 1

Frontend:
  [ ] 1  htmx    Server-rendered HTML fragments driven by hx-* attributes
  [ ] 2  templ   Type-checked Go HTML components (requires go)
Toggle numbers, Enter to accept: 1 2

Testing:
  [ ] 1  tdd     A failing test first for each behavior, then the code
Toggle numbers, Enter to accept: 1

Plan
  framework 0.1.0
    create  AGENTS.md
    create  docs/README.md
    create  docs/framework/rules.md
    create  docs/framework/report.md
    create  docs/framework/reviewer.md
    create  docs/framework/agents/ (6 roles)
    create  docs/framework/skills/prototype/ (SKILL.md, template.md)
    create  docs/framework/skills/spec/ (SKILL.md, template.md)
    create  docs/framework/skills/work-spec/ (SKILL.md, template.md)
  extensions
    create  docs/framework/skills/go/SKILL.md      rendered with: tdd
    create  docs/framework/skills/htmx/SKILL.md    rendered with: templ
    create  docs/framework/skills/templ/SKILL.md   rendered with: htmx
    create  docs/framework/skills/tdd/SKILL.md     rendered with: go
  decisions (approved, selected by you)
    create  docs/decisions.md   P/D-1 language: go · P/D-2 frontend: htmx, templ · P/D-3 testing: tdd
    create  docs/stack.md       language: go (P/D-1) · frontend: htmx, templ (P/D-2) · testing: tdd (P/D-3)
  runtime claude
    create  CLAUDE.md
    create  .claude/skills/ (7 skill stubs: 3 core, 4 extensions)
    create  .claude/agents/ (5 role stubs; adversarial-reviewer model opus)
    create  .claude/settings.json (state guard hook)
  runtime codex
    create  .agents/skills/ (7 skill stubs: 3 core, 4 extensions)
  registry
    create  refs/heads/workflow-state with an empty state.json
  manifest
    create  .solomonar/manifest.json
Proceed? [Y/n] y

Initialized solomonar 0.1.0 in /work/ledger
  45 files created
  registry  refs/heads/workflow-state @ 3a5c0d7e9f1b2a4c6d8e0f1a3b5c7d9e1f2a4b6c
  pushed    origin/workflow-state

Next:
  1. Review and commit the framework files on main:
       git add AGENTS.md CLAUDE.md .claude .agents .solomonar docs
       git commit -m "chore: initialize solomonar 0.1.0"
  2. Start a top-level session (claude or codex) in /work/ledger and ask it to run the prototype skill.
```

Postconditions: the files exist uncommitted on main; each chosen group has an approved `P/D-n` entry; `workflow-state` exists as a root commit
`state: initialize registry` containing `{"schema_version": 1, "tasks": {}}`; the manifest records every choice.
`init` never commits on main (CLI-D4); `doctor` reports uncommitted framework files until they are committed.

`--json` data:

```json
{
  "project": "/work/ledger",
  "framework_version": "0.1.0",
  "choices": {"runtimes": ["claude", "codex"], "extensions": {"language": ["go"], "frontend": ["htmx", "templ"], "testing": ["tdd"]}, "adversarial_model": {"claude": "opus"}},
  "decisions": [{"id": "P/D-1", "group": "language", "selected": ["go"]}, {"id": "P/D-2", "group": "frontend", "selected": ["htmx", "templ"]}, {"id": "P/D-3", "group": "testing", "selected": ["tdd"]}],
  "files": [{"path": "AGENTS.md", "action": "create", "owner": "core+notes"}],
  "registry": {"ref": "refs/heads/workflow-state", "commit": "3a5c0d7e9f1b2a4c6d8e0f1a3b5c7d9e1f2a4b6c", "action": "create", "pushed": true},
  "next": ["git add AGENTS.md CLAUDE.md .claude .agents .solomonar docs", "git commit -m \"chore: initialize solomonar 0.1.0\""]
}
```

### UC-2 Initialize non-interactively

Actor: user or script.

```
solomonar init [<dir>]
  [--name <name>]                      default: directory name
  [--objective <text>]                 default: empty ("Not set" in docs/README.md)
  [--runtime <claude|codex|none>]...   repeatable or comma-separated; default: detected on PATH
  [--ext <name|path|none>]...          extensions of any group, catalog name or local directory, repeatable or
                                       comma-separated; one P/D-n per stack group; default: go when go.mod exists
  [--adversarial-model <runtime>=<model>]   repeatable
  [--main-branch <name>]               default: main
  [--remote <name>]                    default: origin
  [--git-init]                         run git init -b <main-branch> when <dir> is not a repository
  [--overwrite <path>]... [--keep <path>]...   resolve conflicts (UC-3)
  [-y|--yes] [--dry-run] [--json]
```

```
$ solomonar init --runtime claude --ext go,tdd --objective "Track shared household expenses" --yes
Initialized solomonar 0.1.0 in /work/ledger
  34 files created
  registry  refs/heads/workflow-state @ 3a5c0d7e9f1b2a4c6d8e0f1a3b5c7d9e1f2a4b6c
  warning: no remote "origin"; workflow-state stays local (rules §7)

Next:
  1. git add AGENTS.md CLAUDE.md .claude .solomonar docs && git commit -m "chore: initialize solomonar 0.1.0"
  2. Start a top-level Claude Code session in /work/ledger and ask it to run the prototype skill.
```

Without a terminal and without `--yes`:

```
$ solomonar init --runtime claude < /dev/null
error: init needs a confirmation (interactive_required)
  fix: pass --yes to accept the plan, or --dry-run to print it
```

Exit 2.

### UC-3 Re-run init on an initialized project

Same choices, identical files: a no-op.

```
$ solomonar init --yes
Already initialized with solomonar 0.1.0; nothing to do.
  45 files unchanged · registry refs/heads/workflow-state unchanged
  Change choices with `solomonar runtime|extension add|remove`; move versions with `solomonar upgrade`.
```

Exit 0. A choice flag that differs from the manifest is rejected, so that `init` never becomes a second way to
change choices:

```
$ solomonar init --ext go --yes
error: init found different choices (already_initialized)
  found: manifest extensions [go, htmx, templ, tdd]; requested [go]
  fix:   change extensions with `solomonar extension add|remove`; init never changes choices
```

Exit 3. A managed file edited since installation, or an unmanaged file in a target path, is a conflict:

```
$ solomonar init --runtime claude --yes
Plan
  conflict  docs/framework/rules.md      modified since installation (manifest sha256 differs)
  conflict  .claude/agents/implementer.md   exists and is not managed by solomonar
error: 2 conflicts; nothing was written (file_conflict)
  fix: rerun with --keep <path> to keep the project's file, or --overwrite <path> to replace it;
       see the difference with: solomonar upgrade --diff <path>
```

Exit 4. `--keep` records the project's version as the new base in the manifest; `--overwrite` writes the embedded
version. The registry is never recreated: an existing valid `workflow-state` is `unchanged`; an invalid one stops
`init` with `registry_exists_invalid` (exit 3) and the fix "inspect with git show refs/heads/workflow-state:state.json;
the CLI never overwrites a registry".

### UC-4 Initialize an existing repository

Not a Git repository:

```
$ solomonar init
error: /work/ledger is not a Git repository (not_a_repository)
  fix: run `git init -b main` first, or rerun with --git-init
```

Interactively, the same condition asks `Initialize a Git repository here with main branch "main"? [y/N]`.

Existing unmanaged `AGENTS.md`: the default is adoption, the existing content moved under `## Project notes`:

```
AGENTS.md exists and is not managed by solomonar.
  Adopt it: write the framework part above "## Project notes" and move the current content into that section? [Y/n]
```

Non-interactively, adoption is the default with `--yes`; `--overwrite AGENTS.md` discards the old content; `--keep
AGENTS.md` aborts with a conflict, because the framework cannot run without its `AGENTS.md`.

Existing `CLAUDE.md` without `@AGENTS.md`: never modified (G10):

```
  skip      CLAUDE.md   exists; add this line so Claude Code loads the framework:  @AGENTS.md
```

Other cases: an existing `docs/README.md` is `skip` (seed, project-owned); an empty repository with no commit on
`main` initializes, and `doctor` reports that feature worktrees need a first commit on `main`; a `.jj/` directory
prints `warning: Jujutsu detected; v0.1 supports Git only. Never run jj commands on workflow-state.`

### UC-5 Add, list, or remove a runtime

```
solomonar runtime list [--json]
solomonar runtime add <claude|codex> [--adversarial-model <model>] [-y] [--dry-run] [--json]
solomonar runtime remove <claude|codex> [-y] [--dry-run] [--json]
```

```
$ solomonar runtime list
RUNTIME  STATUS     ON PATH  ADAPTER
claude   installed  yes      CLAUDE.md, .claude/skills (7), .claude/agents (5), guard hook
codex    available  yes      AGENTS.md (native), .agents/skills

$ solomonar runtime add codex --yes
Plan
  create  .agents/skills/ (7 skill stubs: prototype, spec, work-spec, go, htmx, templ, tdd)
  update  .solomonar/manifest.json
Added runtime codex: 8 files.
  warning: Codex state guard hook not available (CLI-D9); the task commands still validate every write.
Next: commit the files on main.
```

`remove` deletes the runtime's unmodified adapter files and its hook entry, keeps a `CLAUDE.md` it did not create,
and reports a modified adapter file as a conflict (exit 4) resolved with `--keep <path>` (untrack, leave the file)
or `--overwrite <path>` (delete it).

### UC-6 Choose extensions for an architecture or stack decision

Typical triggers: the user decides part of the stack after the prototype, or an orchestrator applies an approved
task decision that adopts a technology or practice. The examples start from a project initialized with `go` only
(`P/D-1`).

```
solomonar extension list [--group <group>] [--kind <skill|agent>] [--installed] [--json]
solomonar extension show <name> [--json]
solomonar extension add <name | path>... [--decision <ID>] [--rationale <text>] [-y] [--dry-run] [--json]
solomonar extension remove <name>... [--decision <ID>] [--rationale <text>] [-y] [--dry-run] [--json]
solomonar extension new <name> --kind <skill|agent> --group <group> [--title <text>] [--requires <name>]...
                        [--applies-when <text>] [--path <glob>]... [--dry-run] [--json]
```

`ext` is an alias for `extension`. `--decision` and `--rationale` apply to stack groups only.

List, grouped by decision type:

```
$ solomonar extension list
GROUP     NAME   KIND   SOURCE   STATUS     DECISION  REQUIRES  APPLIES WHEN
language  go     skill  catalog  installed  P/D-1     -         A change touches Go code
frontend  htmx   skill  catalog  available  -         -         hx-* attributes or endpoints that return fragments
frontend  templ  skill  catalog  available  -         go        *.templ files or the handlers that render them
testing   tdd    skill  catalog  available  -         -         A change adds or changes behavior
```

The user adopts htmx for the frontend:

```
$ solomonar extension add htmx
Plan
  extensions
    create  docs/framework/skills/htmx/SKILL.md
    create  .claude/skills/htmx/SKILL.md
    create  .agents/skills/htmx/SKILL.md
  decisions
    append  docs/decisions.md   P/D-2 Frontend technologies: htmx (approved, selected by you)
    append  docs/stack.md       frontend: htmx (P/D-2)
  update  AGENTS.md (Extensions: htmx)
  update  .solomonar/manifest.json
Record P/D-2 as your approved decision for the frontend group? [Y/n] y
Added extension htmx (frontend skill) under P/D-2.
  note: P/D-2 approves the technology and its rules; versions stay in the manifests, and every further
        dependency still needs approval (rules §3).
Next: commit on main: git add AGENTS.md .solomonar .claude .agents docs && git commit -m "docs: adopt the htmx extension (P/D-2)"
```

Adding templ to the same group supersedes P/D-2 with the full new selection and generates the rules that apply to
the combination:

```
$ solomonar extension add templ --rationale "Type-checked components for the htmx fragments" --yes
Plan
  extensions
    create  docs/framework/skills/templ/SKILL.md   rendered with: htmx (adds templ/C-4)
    update  docs/framework/skills/htmx/SKILL.md    re-rendered with: templ (adds htmx/C-6)
    create  .claude/skills/templ/SKILL.md
    create  .agents/skills/templ/SKILL.md
  decisions
    append  docs/decisions.md   P/D-3 Frontend technologies: htmx, templ (supersedes P/D-2)
    update  docs/decisions.md   P/D-2 status superseded, replaced by P/D-3
    append  docs/stack.md       frontend: htmx, templ (P/D-3)
  update  AGENTS.md, .solomonar/manifest.json
Added extension templ (frontend skill) under P/D-3.
```

An orchestrator applying an approved task decision, in the task worktree, so the extension merges with the task:

```
$ cd /work/ledger-worktrees/W-002
$ solomonar extension add tdd --decision W-002/D-1 --yes
Verified W-002/D-1: approved in docs/decisions.md (selected option: test-driven development for all behavior changes)
Plan
  extensions
    create  docs/framework/skills/tdd/SKILL.md   rendered with: go (adds tdd/C-5)
    update  docs/framework/skills/go/SKILL.md    re-rendered with: tdd (adds go/C-9)
    create  .claude/skills/tdd/SKILL.md
    create  .agents/skills/tdd/SKILL.md
  decisions
    append  docs/stack.md   testing: tdd (W-002/D-1)
  update  AGENTS.md, .solomonar/manifest.json
Added extension tdd (testing skill) under W-002/D-1.
Next: commit in the worktree as "docs(W-002): adopt extensions for W-002/D-1"; they reach main with the task's merge.
```

Rejections (exit 3):

```
$ solomonar extension add templ          # in a project without the go extension
error: extension add rejected (extension_requires)
  check: templ requires go (language), which is not installed
  fix:   solomonar extension add go templ
nothing was written

$ solomonar extension add tdd --decision W-002/D-2
error: extension add rejected (decision_not_approved)
  check: --decision must name an approved entry in docs/decisions.md
  found: W-002/D-2 has status pending
  fix:   wait for the user's approval (rules §3), then rerun
nothing was written
```

Show one extension as rendered for this project:

```
$ solomonar extension show templ
templ components · frontend skill · catalog · installed under P/D-3 · requires go
Applies when: a change touches *.templ files or the handlers that render them (paths: **/*.templ)
Rendered with: htmx
  templ/C-1  must    *_templ.go files are generated by templ generate and never edited by hand; CI fails when stale
  templ/C-2  must    User data is never rendered through templ.Raw or any other unescaped output
  templ/C-3  should  Components take view models built for the page, not domain entities or database rows
  templ/C-4  must    With htmx: render the fragment component when HX-Request is set, else the page component
File: docs/framework/skills/templ/SKILL.md
```

`extension remove` follows the same decision rules: without `--decision` it records a superseding `P/D-n` with the
reduced selection (or, when the group becomes empty, an entry whose selected option is `none`), deletes the
unmodified extension files and their stubs, re-renders the extensions that referred to the removed one, and rejects
a removal that leaves a `requires` unmet (`extension_requires`: removing `go` while `templ` is installed). A
modified file is a conflict (exit 4), resolved with `--keep <path>` or `--overwrite <path>`. The CLI never checks
whether code still uses a technology; it prints `warning: code governed by htmx loses its rules`. Core items cannot
be removed (`core_item`).

### UC-7 Add workflow skills, review agents, or the project's own extensions

Project extensions cover what the catalog does not: a domain rule set, an in-house library, a supporting procedure,
or an extra reviewer. They use the catalog format and are project-owned; upgrades never change them.

Create a stack-group skill, write its rules, then adopt it; a stack group records a decision as in UC-6:

```
$ solomonar extension new money --kind skill --group backend --title "Money handling"
Created project extension money (backend skill, project-owned)
  create  docs/framework/skills/money/SKILL.md
  update  .solomonar/manifest.json   (metadata: group, requires, applies_when, paths)
Write its rules as money/C-n (must|should), then adopt it: solomonar extension add money

$ solomonar extension add money --yes
Added extension money (backend skill) under P/D-4.
```

Create an extra reviewer; the `review` group records no decision:

```
$ solomonar extension new a11y-reviewer --kind agent --group review --title "Accessibility reviewer" \
    --applies-when "A change touches templates or docs/ui.md" --path "**/*.templ"
Created project extension a11y-reviewer (review agent, project-owned)
  create  docs/framework/agents/a11y-reviewer.md
  update  .solomonar/manifest.json
Write its specialty and finding prefix, then adopt it: solomonar extension add a11y-reviewer

$ solomonar extension add a11y-reviewer --yes
Plan
  create  .claude/agents/a11y-reviewer.md   tools: Read, Grep, Glob, Bash
  update  AGENTS.md (Extensions: agent a11y-reviewer, required when a change touches templates or docs/ui.md)
  update  .solomonar/manifest.json
Added extension a11y-reviewer (review agent). Triage now lists it beside the core specialists.
```

Add a supporting procedure from a local directory; the `workflow` group records no decision:

```
$ solomonar extension add ./tools/skills/release-notes --yes
Plan
  create  docs/framework/skills/release-notes/SKILL.md   (project-ext, from ./tools/skills/release-notes)
  create  .claude/skills/release-notes/SKILL.md
  create  .agents/skills/release-notes/SKILL.md
  update  AGENTS.md (Extensions: release-notes)
  update  .solomonar/manifest.json
Added extension release-notes (workflow skill, project-owned).
```

A local directory needs a `SKILL.md` with `name` and `description` frontmatter, or an agent file with `name` and
`description`. A project extension's metadata lives in the manifest, never in an extra file: the CLI asks for the
group (default `workflow` for a skill, `review` for an agent) and `applies_when`, or takes them from flags. Rejections (exit 3): `core_item` for `extension remove work-spec` or `extension remove
security-reviewer`; `name_collision` when the name exists (fix: `--name <other>` on `new`, or rename the source
directory); `unknown_extension` for a catalog name that does not exist; `invalid_value` naming a missing frontmatter
field. A skill's `name` must match its directory (G11).

### UC-8 Sync adapters after editing the portable source

Trigger: a role's or skill's frontmatter changed in `docs/framework/`, a stub was deleted, or a branch merged
`docs/framework/` changes.

```
solomonar sync [--check] [--dry-run] [--json]
```

```
$ solomonar sync
  update  .claude/skills/spec/SKILL.md      description changed in docs/framework/skills/spec/SKILL.md
  update  .agents/skills/spec/SKILL.md      description changed in docs/framework/skills/spec/SKILL.md
  create  .claude/agents/spec-reviewer.md   missing
  unchanged  9 files
Synced 3 files.

$ solomonar sync --check
stale  .claude/skills/spec/SKILL.md   description differs from docs/framework/skills/spec/SKILL.md
error: 1 adapter file is stale (checks_failed)
  fix: solomonar sync
```

`sync` also re-renders catalog extensions whose `With` sections depend on the installed set and the
"Extensions" section of `AGENTS.md`. `--check` writes nothing and exits 5 when anything would change, for CI. A hand-edited stub is a conflict (exit 4),
resolved with `--overwrite <path>`.

### UC-9 Upgrade the framework

```
solomonar upgrade [--dry-run] [--diff <path>] [--take-new <path>]... [--keep <path>]... [-y] [--json]
```

The target is the framework version embedded in the running CLI. Per managed file, with `base` = manifest hash,
`current` = file on disk, `new` = embedded content:

| Condition | Action |
|---|---|
| current = base, new = base | `unchanged` |
| current = base, new ≠ base | `update` |
| current ≠ base, new = base | `keep` (project edit preserved) |
| current = new | `unchanged` (manifest base moves to new) |
| current ≠ base, new ≠ base, current ≠ new | `conflict` |

```
$ solomonar upgrade
Upgrade framework 0.1.0 → 0.2.0
  update    docs/framework/rules.md
  update    docs/framework/agents/orchestrator.md
  keep      docs/framework/skills/spec/template.md   (project edit, unchanged upstream)
  conflict  docs/framework/agents/implementer.md     (edited here and upstream)
  update    AGENTS.md (framework part; Project notes preserved)
  update    .claude/skills/ (7 stubs)
error: 1 conflict; nothing was written (file_conflict)
  fix: compare with `solomonar upgrade --diff docs/framework/agents/implementer.md`, then rerun with
       --take-new <path> or --keep <path>

$ solomonar upgrade --take-new docs/framework/agents/implementer.md --yes
Upgraded framework 0.1.0 → 0.2.0: 7 files updated, 1 kept.
Next: commit on main as "chore: upgrade solomonar framework to 0.2.0"; running tasks pick it up after updating from main.
```

An upgrade that changes `state.json`'s `schema_version` also migrates the registry with one compare-and-swap commit
`state: migrate schema 1 -> 2`, shown in the plan. Downgrade is rejected (`already_initialized`, fix: install the
matching CLI). The upgrade never commits on main.

### UC-10 Check the project

```
solomonar doctor [--json]
```

```
$ solomonar doctor
git
  ok    git 2.47.1 (2.40 or later)
  ok    main branch "main" has commits
  warn  .jj/ present: v0.1 supports Git only
registry
  ok    refs/heads/workflow-state @ 3a5c0d7, schema_version 1, 4 tasks valid
  ok    origin/workflow-state matches local
files
  ok    framework 0.1.0, manifest valid
  fail  4 framework paths not committed on main: AGENTS.md, CLAUDE.md, .claude/, .solomonar/
  warn  docs/framework/skills/spec/template.md modified since installation (kept by upgrades)
extensions
  ok    4 installed: go (P/D-1), htmx, templ (P/D-2), tdd (P/D-3); requirements met; rendering current
runtimes
  ok    claude: CLAUDE.md imports AGENTS.md; 7 skill stubs, 5 role stubs in sync; guard hook present
  warn  codex: state guard hook not available (CLI-D9)
forge
  ok    gh 2.62.0 authenticated for github.com/acme/ledger
  ok    merge commits allowed (rules §11)
  warn  squash merges also allowed; the orchestrator must use merge commits only
project
  warn  docs/README.md commands: run, build, test, lint not established
1 failure, 4 warnings
```

Exit 5 when any `fail`, else 0. Check catalog:

| Area | Check | Level when it fails |
|---|---|---|
| git | Git version ≥ 2.40; repository; `main` has a commit; worktrees usable | fail |
| git | `.jj/` present | warn |
| registry | ref exists; `state.json` parses; schema valid; every record valid for its status | fail |
| registry | remote configured and local ref published | warn |
| files | manifest valid; managed files present | fail |
| files | managed files committed on main | fail |
| files | managed files modified since installation | warn |
| extensions | every installed stack extension's decision exists in `docs/decisions.md` with status `approved` | fail |
| extensions | `requires` and `conflicts` hold for the installed set; rendered extensions current | fail |
| extensions | the core is complete: every item `rules.md` names is present | fail |
| runtimes | adapter files in sync (as `sync --check`) | fail |
| runtimes | `CLAUDE.md` loads `AGENTS.md`; guard hook present | warn |
| forge | `gh` installed and authenticated | warn (facts become `unknown`) |
| forge | repository allows merge commits | fail |
| forge | squash or rebase merges also allowed | warn |
| project | `docs/README.md` commands established | warn |

## 6. Use cases: task state

All examples continue in `/work/ledger`. Worktrees live at `/work/ledger-worktrees/<ID>` (rules §4 step 4). Each
successful write prints the commit and, when a remote is configured, the push result. Every writing command accepts
`--dry-run` and `--json`.

### UC-11 Plan a task (user)

```
solomonar task plan --title <title> [--depends-on <ID>]... [--related <ID>] [--json] [--dry-run]
```

```
$ solomonar task plan --title "Export expenses as CSV"
Planned W-003: Export expenses as CSV
  state(W-003): planned, plan "Export expenses as CSV"
  registry @ bed97175b06e68eb7ff5722897aa86b8ea75c8ef · pushed origin/workflow-state
```

The ID is the next free `W-NNN` (highest existing number plus one). `--depends-on` and `--related` must name existing
tasks (`task_not_found`). `task plan` takes no `--as`: it is the user's write (G16).

`--json` data: `{"id": "W-003", "record": {"title": "Export expenses as CSV", "status": "planned", "updated_at":
"2026-10-20T14:58:03Z"}, "commit": "bed97175b06e68eb7ff5722897aa86b8ea75c8ef", "pushed": true}`.

### UC-12 Register a task (orchestrator): the registration barrier

```
solomonar task register [<ID>] --title <title> --owner <owner> [--depends-on <ID>]... [--related <ID>]
                        [--next <actor>:<action>] [--json] [--dry-run]
```

```
$ solomonar task register --title "Login" --owner orchestrator-1
Registered W-001: Login
  status    reserved / setup
  owner     orchestrator-1
  branch    feature/W-001 (create it now)
  next      orchestrator-1: Create feature/W-001 and its worktree
  state(W-001): reserved/setup, registered by orchestrator-1
  registry @ e06b95860a6082ae37ef08874f8eb5fade2549da · pushed origin/workflow-state

Registration confirmed. The next task may start.
Create the worktree:
  git worktree add -b feature/W-001 ../ledger-worktrees/W-001 main
```

Claiming a planned task: `solomonar task register W-003 --title "Export expenses as CSV" --owner orchestrator-2`; the
title must match the planned title or is updated with `warning: title changed from "<old>"`. The default
`next_action` is the worktree step shown above.

Rejections (exit 3): `not_planned` (the ID exists and is not `planned`; fix: `task show <ID>`, resume it instead);
`owner_in_use` (the owner already owns an active task; fix: use a new owner such as `orchestrator-3`, since an
orchestrator owns exactly one task); `dependency_not_done` is a warning, not a rejection: register, then `task block
--kind dependency` (rules §4). The success line "Registration confirmed" appears only after the compare-and-swap
commit exists; `--dry-run` never prints it.

### UC-13 Advance a task through its stages (owner)

```
solomonar task move <ID> [--as <owner>]
  [--status <reserved|running|blocked|paused|cancelled>] [--resume-cancelled]
  [--stage <setup|specification|implementation|specialized_review|adversarial_review|integration>]
  [--revision <full-sha>] [--spec <worktree-relative path>] [--tier <0|1|2>]
  [--agent <id>:<role>]... | [--agent none]
  [--pr <url|number>] [--next <actor>:<action>]
  [--json] [--dry-run]
```

At least one field flag is required. All flags apply as one validated update and one commit. `done` and `complete`
are not accepted here: only `task done` writes them.

Worktree ready, specification starts:

```
$ export SOLOMONAR_OWNER=orchestrator-1
$ solomonar task move W-001 --status running --stage specification --next "orchestrator-1:Run work-spec"
W-001  reserved/setup → running/specification
  next  orchestrator-1: Run work-spec
  state(W-001): running/specification, status running, stage specification
  registry @ 5b2e9d0c7a4f1e3b6d8c0a2f4e6b8d0c2a4e6f8b
```

Work spec committed, readiness passed, implementation delegated:

```
$ solomonar task move W-001 --spec docs/work/W-001-login.md --stage implementation \
    --agent impl-1:implementer --next "impl-1:Implement AC-1 to AC-4"
W-001  running/specification → running/implementation
  spec    docs/work/W-001-login.md (found in /work/ledger-worktrees/W-001)
  agents  impl-1 implementer
  next    impl-1: Implement AC-1 to AC-4
  state(W-001): running/implementation, spec docs/work/W-001-login.md, agents impl-1
  registry @ 8e1f3a5c7b9d0e2f4a6c8b0d1e3f5a7c9b1d2e4f
```

`READY_FOR_REVIEW` at a committed SHA, triage recorded tier 2, specialists launched:

```
$ solomonar task move W-001 --revision 7cd18e45bb55c3be9465c9f893683bc05216f529 --tier 2 \
    --stage specialized_review --agent sec-1:security-reviewer --agent code-1:code-reviewer \
    --agent spec-1:spec-reviewer --next "orchestrator-1:Collect specialist reports"
W-001  running/implementation → running/specialized_review
  revision  7cd18e45bb55c3be9465c9f893683bc05216f529 (on feature/W-001)
  tier      2: specialists, then adversarial
  agents    sec-1 security-reviewer, code-1 code-reviewer, spec-1 spec-reviewer
  state(W-001): running/specialized_review, revision 7cd18e4, tier 2
  registry @ 2d4f6a8c0e1b3d5f7a9c1e2b4d6f8a0c2e3b5d7f
```

The remaining moves follow the same shape: `--stage adversarial_review --agent adv-1:adversarial-reviewer`;
`--stage implementation` with a correction round; `--stage integration`; `--pr 42`; `--revision` for each closure
commit (rules §9). `--json` data: `{"id": "W-001", "from": {"status": "running", "stage": "implementation"}, "to":
{"status": "running", "stage": "specialized_review"}, "changed": ["revision", "tier", "stage", "active_agents",
"next_action"], "record": { }, "commit": "<sha>", "pushed": true}`.

### UC-14 A rejected move

Each rejection exits 3, writes nothing, and names the check and the fix (8.4). Typical cases:

```
$ solomonar task move W-001 --stage specialized_review
error: task move W-001 rejected (tier_required)
  check: a stage after implementation needs a recorded tier
  fix:   triage the candidate and pass --revision <full-sha> --tier <0|1|2> with the stage move
nothing was written

$ solomonar task move W-001 --revision 7cd18e4 --tier 1 --stage specialized_review
error: task move W-001 rejected (revision_not_full_sha)
  check: --revision must be a full 40-character SHA
  fix:   git -C /work/ledger-worktrees/W-001 rev-parse 7cd18e4
nothing was written

$ solomonar task move W-001 --stage adversarial_review
error: task move W-001 rejected (stage_not_allowed)
  check: tier 1 skips adversarial_review (rules §7)
  fix:   --stage integration, or record tier 2 if the triage changed: --tier 2
nothing was written

$ solomonar task move W-001 --stage implementation --as orchestrator-2
error: task move W-001 rejected (owner_mismatch)
  check: only the owner writes a task's record
  found: owner is orchestrator-1; acting as orchestrator-2
  fix:   act as orchestrator-1, or ask the user for an explicit handoff: solomonar task handoff W-001 --owner orchestrator-2
nothing was written
```

### UC-15 Wait for a user decision

The implementer returned `NEEDS_DECISION` on session storage. The orchestrator records the blocker and the user as
next actor in one write, then sets `blocked` because no independent work remains (rules §3):

```
solomonar task block <ID> --kind <decision|dependency|check|environment> --summary <text> [--ref <ref>]
                     [--next <actor>:<action>] [--status blocked] [--as <owner>] [--json] [--dry-run]
solomonar task unblock <ID> <blocker-id> [--status <prior status>] [--next <actor>:<action>]
                       [--as <owner>] [--json] [--dry-run]
```

`--next` and `--status` on `block` and `unblock` are shorthands for the same fields of `task move`, so one decision
event is one commit.

```
$ solomonar task block W-001 --kind decision --summary "Session storage" --ref W-001/D-1 \
    --next "user:Decide W-001/D-1: session storage" --status blocked
W-001  running/implementation → blocked/implementation
  blocker  B-1 decision: Session storage (W-001/D-1)
  next     user: Decide W-001/D-1: session storage
  state(W-001): blocked/implementation, block B-1 decision
  registry @ 9c0e2a4b6d8f1a3c5e7b9d0f2a4c6e8b0d1f3a5c
```

After approval:

```
$ solomonar task unblock W-001 B-1 --status running --next "impl-1:Apply W-001/D-1, continue AC-2"
W-001  blocked/implementation → running/implementation
  cleared  B-1 decision: Session storage
  next     impl-1: Apply W-001/D-1, continue AC-2
  state(W-001): running/implementation, unblock B-1
  registry @ 4f6a8c0e2b4d6f8a1c3e5b7d9f0a2c4e6b8d1f3a
```

Rules: blocker IDs are `B-n`, stable within the task, never reused; `--kind decision` requires `--ref <ID>/D-n`;
`--summary` is at most 80 characters; `--status blocked` requires at least one open blocker after the write;
`unblock --status` accepts only the prior status (8.1).

### UC-16 Inspect state

```
solomonar task show [<ID>] [--json]
solomonar task status <ID> [--json] [--no-forge]
```

Registry overview:

```
$ solomonar task show
ID     STATUS     STAGE               OWNER           NEXT
W-001  running    specialized_review  orchestrator-1  orchestrator-1: Collect specialist reports
W-002  blocked    specification       orchestrator-2  user: Decide W-002/D-1: currency rounding
W-003  planned    -                   -               -
registry refs/heads/workflow-state @ 2d4f6a8 · 2026-10-20T15:12:40Z
```

Rows are sorted by ID; a `done` row shows its `summary` in NEXT. `task show <ID>` prints the stored record exactly
as JSON, the raw read (J3); `task show --json` returns the full `state.json` plus the source commit.

The oracle (rules §7, J3):

```
$ solomonar task status W-001
W-001  Login
record
  status    running / specialized_review       owner orchestrator-1   tier 2
  spec      docs/work/W-001-login.md
  revision  7cd18e45bb55c3be9465c9f893683bc05216f529
  agents    sec-1 security-reviewer, code-1 code-reviewer, spec-1 spec-reviewer (last reported, not liveness)
  blockers  none
  next      orchestrator-1: Collect specialist reports
  updated   2026-10-20T15:12:40Z (registry @ 2d4f6a8)
git
  branch    feature/W-001 @ 992051e56493f5bb8abb4faadcdbc5f271195498
  worktree  /work/ledger-worktrees/W-001
  HEAD      992051e5 · 1 commit ahead of revision
  dirty     1 path: internal/auth/session_test.go (modified)
  spec      present in worktree and at revision
  merged    no
forge (gh)
  pr        none for feature/W-001
  checks    unknown (no PR)
drift
  head_ahead_of_revision  HEAD moved past the reviewed revision while reviews run (rules §6 Candidate stability)
  dirty_worktree          uncommitted work during review; preserve it, do not discard
allowed next
  solomonar task move W-001 --stage implementation --next <actor>:<action>
  solomonar task move W-001 --stage adversarial_review
  solomonar task block W-001 --kind <kind> --summary <text>
  solomonar task move W-001 --status paused --next <actor>:<action>
  solomonar task move W-001 --status cancelled --next <actor>:<action>
```

`--no-forge` skips `gh`. Exit 0 whatever the drift: drift is information, not failure. The full JSON shape is in 9.3.

### UC-17 Resume after an unexpected session end

A new orchestrator session for W-001 starts. The previous session merged the PR, then ended before `task done`:

```
$ solomonar task status W-001
W-001  Login
record
  status    running / integration          owner orchestrator-1   tier 2
  revision  fea453f853c8645b085126e6517eab38dfaa022f
  pr        https://github.com/acme/ledger/pull/42
  next      orchestrator-1: Merge PR 42 with a merge commit
git
  branch    feature/W-001 @ fea453f853c8645b085126e6517eab38dfaa022f
  worktree  /work/ledger-worktrees/W-001 (clean)
  merged    yes, by 14091a9f2461267ee7e02525b4f1f2923f1c9849 (merge commit on main)
  spec_ref  derived 10ec7f36405c2d9eae3871433246dfd650c19e81:docs/work/W-001-login.md (last spec(W-001) commit merged)
forge (gh)
  pr        #42 merged
drift
  merged_not_done  feature/W-001 is merged into main but the record is not done
allowed next
  solomonar task done W-001 --merge 14091a9f2461267ee7e02525b4f1f2923f1c9849 \
    --spec-ref <PR "Final specification" line> --summary <one sentence>
```

The derived `spec_ref` is a cross-check (C14 b); the orchestrator takes `spec_ref` from the PR's Verification
section (C14 a), and `task done` rejects a mismatch.

### UC-18 Pause, cancel, and resume

```
$ solomonar task move W-002 --status paused --next "orchestrator-2:Resume at specification; draft AC-3 uncommitted"
W-002  running/specification → paused/specification
  state(W-002): paused/specification, status paused

$ solomonar task move W-002 --status running
W-002  paused/specification → running/specification
  state(W-002): running/specification, status running

$ solomonar task move W-004 --status cancelled --next "user:Superseded by W-005; branch and worktree kept"
W-004  running/implementation → cancelled/implementation
  note: branch feature/W-004, its worktree, and PR are kept untouched (rules §10)

$ solomonar task move W-004 --status running --resume-cancelled
W-004  cancelled/implementation → running/implementation
  warning: resuming a cancelled task requires the user's explicit instruction (rules §7)
```

`--status paused` and `--status cancelled` require `--next` (`next_required`); `--status running` on a cancelled
task without `--resume-cancelled` is `cancelled_requires_flag`.

### UC-19 Hand off ownership

The user tells a new session to take over W-002 after confirming the previous session stopped:

```
solomonar task handoff <ID> --owner <new-owner> [--next <actor>:<action>] [--json] [--dry-run]
```

```
$ solomonar task handoff W-002 --owner orchestrator-3
W-002  owner orchestrator-2 → orchestrator-3 (blocked/specification)
  state(W-002): blocked/specification, handoff orchestrator-2 -> orchestrator-3
  registry @ 0a2c4e6b8d1f3a5c7e9b0d2f4a6c8e1b3d5f7a9c
  The previous owner must stop writing this record.
```

`handoff` takes no `--as`: the new owner runs it on the user's instruction (rules §10). Rejections: `handoff_same_owner`,
`owner_in_use`, `done_final`.

### UC-20 Complete a task after the merge

```
solomonar task done <ID> --merge <full-sha> --spec-ref <full-sha>:<path> --summary <text>
                    [--as <owner>] [--json] [--dry-run]
```

```
$ solomonar task done W-001 --merge 14091a9f2461267ee7e02525b4f1f2923f1c9849 \
    --spec-ref 10ec7f36405c2d9eae3871433246dfd650c19e81:docs/work/W-001-login.md \
    --summary "Password login and logout with server-side sessions"
Verified
  merge     14091a9f is a merge commit reachable from main
  revision  fea453f8 is contained in the merge
  spec_ref  resolves; 10ec7f36 is the last spec(W-001) commit merged
W-001  running/integration → done/complete
  state(W-001): done/complete, merged 14091a9
  registry @ 6c8e0a2d4f6b8c1e3a5d7f9b0c2e4a6d8f1b3c5e
Compact record
  {"title": "Login", "status": "done", "stage": "complete",
   "summary": "Password login and logout with server-side sessions",
   "spec_ref": "10ec7f36405c2d9eae3871433246dfd650c19e81:docs/work/W-001-login.md",
   "merge_commit": "14091a9f2461267ee7e02525b4f1f2923f1c9849",
   "pr": "https://github.com/acme/ledger/pull/42", "updated_at": "2026-10-20T17:03:22Z"}
Next: delete the worktree only after confirming it holds no uncommitted or unmerged work:
  git worktree remove ../ledger-worktrees/W-001
```

`task done` uses only Git (J6). Its checks are in 8.3. The compact record keeps `title`, `status`, `stage`,
`summary`, `spec_ref`, `merge_commit`, `pr` (when recorded), `updated_at`, and the optional `related_task`; every other
field leaves the record and stays in registry history. A failed `task done` writes nothing; the orchestrator fixes
the cause and reruns it, never reverting the merge (rules §9).

### UC-21 Concurrent writes

Two orchestrators write at the same moment. Each command reads the registry tip, validates, builds the new tree,
and runs `update-ref refs/heads/workflow-state <new> <old>`. The loser re-reads the new tip, re-validates against it,
and retries, up to 5 attempts:

```
$ solomonar task register --title "Recurring expenses" --owner orchestrator-4 -v
git update-ref refs/heads/workflow-state 77a1… 2d4f6a8…   (lost: tip moved to 9c0e2a4…)
retry 1: re-read tip 9c0e2a4…, re-validated, next free ID now W-005
Registered W-005: Recurring expenses
```

When the re-validation fails on the new tip (for example the same planned ID was claimed meanwhile), the command
fails with that check's code (exit 3). After 5 lost attempts: `cas_exhausted` (exit 4), fix "retry the command; no
state was written".

## 7. Use cases: runtime integration

### UC-22 The state guard denies a direct registry write

The Claude Code adapter registers a `PreToolUse` hook that runs `solomonar guard --runtime claude`. The guard reads
the tool call from stdin and answers in the runtime's hook format. It denies, with a reason that names the right
command:

- writes or edits to any file named `state.json`;
- shell commands that check out, switch to, reset, commit to, `update-ref`, or push `workflow-state`, other than
  through `solomonar`;
- `git worktree add` for a `feature/<ID>` branch whose task is not registered (rules §4: register first);
- `solomonar extension add` or `extension remove` of a stack-group extension without `--decision`: from an agent
  session the change needs an approved decision entry (rules §3); only the user's own command records a `P/D-n`.

```
solomonar guard --runtime <claude|codex>     reads the hook payload on stdin; never writes
```

Denial reason shown to the agent:

```
solomonar guard: denied. The registry is written only through `solomonar task` commands (AGENTS.md rule 3).
Use: solomonar task move W-001 --stage <stage>   (see: solomonar task status W-001)
```

An allowed call produces no output and exit 0. The guard is a convenience: the task commands validate every write
regardless, and the rules stay behavioral contracts (rules §11).

### UC-23 Watch the registry

```
solomonar dashboard [--addr 127.0.0.1:7420] [--interval 5s] [--no-forge]
```

```
$ solomonar dashboard
Serving the read-only registry view on http://127.0.0.1:7420 (refresh 5s). Ctrl-C to stop.
```

Views per the design note "Live HTML dashboard contract": task list, task detail, needs your decision, data
freshness. The page reads `/api/state` (the `task show --json` data) and `/api/status/<ID>` (the `task status
--json` data). It binds to loopback by default, never mutates state, and shows each task as last reported, not as
alive.

### UC-24 Versions

```
$ solomonar version
solomonar 0.1.0 (embedded framework 0.1.0)
project   framework 0.1.0 · registry schema 1
```

Outside a project the `project` line reads `not initialized`. `--json`: `{"cli": "0.1.0", "framework": "0.1.0",
"project_framework": "0.1.0", "registry_schema": 1}`.

## 8. Reference: task validation

### 8.1 Status transitions

| From | Allowed to | Command | Condition |
|---|---|---|---|
| (none) | `planned` | `task plan` | - |
| (none), `planned` | `reserved` | `task register` | owner not in use |
| `reserved` | `running`, `blocked`, `paused`, `cancelled` | `task move` / `task block` | `blocked` needs an open blocker; `paused` and `cancelled` need `--next` |
| `running` | `blocked`, `paused`, `cancelled` | `task move` / `task block` | as above |
| `running` | `done` | `task done` only | 8.3 |
| `blocked`, `paused` | the prior status (`reserved` or `running`), `cancelled` | `task move` / `task unblock` | the prior status is derived from registry history: the task's last status other than `blocked` or `paused` |
| `cancelled` | `reserved`, `running` | `task move --resume-cancelled` | explicit flag |
| `done` | none | - | `done_final` |

### 8.2 Stage transitions

A stage change needs status `running` after the write (`reserved` only at `setup`); a `blocked` or `paused` task
keeps its stage (rules §7).

| From | Forward | Back | Requires |
|---|---|---|---|
| `setup` | `specification` | - | worktree on `feature/<ID>` exists |
| `specification` | `implementation` | - | `spec` recorded and present in the worktree |
| `implementation` | tier 1, 2: `specialized_review`; tier 0: `integration` | `specification` | `revision` and `tier` recorded or passed in the same command |
| `specialized_review` | tier 2: `adversarial_review`; tier 1: `integration` | `implementation` | - |
| `adversarial_review` | `integration` | `implementation`, `specialized_review` | tier 2 |
| `integration` | `complete` via `task done` only | `implementation`, `specialized_review` (tier 1, 2: rerun after a main update, C12) | - |

Field rules: `--revision` is accepted when the current or resulting stage is `implementation` or `integration`;
`--tier` is accepted with `--revision` or when the recorded tier allows the current stage; `--spec` from
`specification` onward; `--pr` in `integration`; `--agent` and `--next` in any stage before `done`.

### 8.3 Fact checks

| Flag or command | Check | Code |
|---|---|---|
| `--revision` | full 40-hex SHA | `revision_not_full_sha` |
| `--revision` | `feature/<ID>` exists locally | `branch_missing` |
| `--revision` | `git merge-base --is-ancestor <sha> feature/<ID>` | `revision_not_on_branch` |
| `--spec` | relative path under the worktree, no `..` | `invalid_value` |
| `--spec` | exists in the worktree located by `git worktree list --porcelain`, or at the recorded `revision` | `spec_not_found` |
| `--agent` | `<id>:<role>`, id `[a-z]+-[0-9]+`, role installed in `docs/framework/agents/` and not `orchestrator` | `invalid_value` |
| `--next` | `<actor>:<action>`, actor `user`, the owner, or an `active_agents` id; action 1 to 120 characters | `invalid_value` |
| `--pr` | GitHub PR URL of the configured remote, or a number turned into that URL | `invalid_value` |
| `task done --merge` | full SHA, reachable from main, has two or more parents | `merge_not_on_main`, `merge_not_merge_commit` |
| `task done` | recorded `revision` is an ancestor of the merge commit | `revision_not_merged` |
| `task done --spec-ref` | `git show "<spec_ref>"` resolves | `spec_ref_invalid` |
| `task done --spec-ref` | the commit is the last commit with subject `spec(<ID>):` reachable from the merge commit's second parent | `spec_ref_not_final` |
| `task done` | stage `integration`, status `running`, no open blocker | `transition_not_allowed`, `blockers_open` |

### 8.4 Error codes (task commands)

| Code | Exit | Meaning | Fix shown |
|---|---|---|---|
| `not_initialized` | 3 | No `refs/heads/workflow-state` | `solomonar init` |
| `registry_invalid` | 1 | `state.json` fails the schema | inspect with `git show refs/heads/workflow-state:state.json`; never edit it by hand |
| `task_not_found` | 3 | Unknown ID | `solomonar task show` |
| `not_planned` | 3 | `register <ID>` on a non-planned record | resume the task instead |
| `owner_required` | 2 | No `--as` and no `SOLOMONAR_OWNER` | pass `--as <owner>` |
| `owner_mismatch` | 3 | Acting owner differs from the record | act as the owner, or `task handoff` on the user's instruction |
| `owner_in_use` | 3 | The owner owns another active task | choose a new owner ID |
| `transition_not_allowed` | 3 | Status change not in 8.1 | names the allowed targets |
| `stage_not_allowed` | 3 | Stage change not in 8.2 or not allowed by the tier | names the allowed stages |
| `tier_required` | 3 | A stage after `implementation` without a tier | pass `--tier` with `--revision` |
| `revision_required` | 3 | Review stage without a revision | pass `--revision` |
| `revision_not_full_sha` | 3 | Short or malformed SHA | `git rev-parse <short>` |
| `branch_missing` | 3 | `feature/<ID>` does not exist | create the branch and worktree (rules §4) |
| `revision_not_on_branch` | 3 | SHA not reachable from `feature/<ID>` | commit on the feature branch |
| `spec_required` | 3 | `implementation` without `spec` | pass `--spec` |
| `spec_not_found` | 3 | The spec path does not exist | commit the work spec in the worktree |
| `next_required` | 3 | Pause or cancel without `--next` | pass `--next <actor>:<action>` |
| `blockers_required` | 3 | `blocked` without an open blocker | `task block` first, or pass `--status blocked` to it |
| `blocker_not_found` | 3 | Unknown blocker ID | `task show <ID>` |
| `blockers_open` | 3 | `task done` with open blockers | `task unblock` |
| `cancelled_requires_flag` | 3 | Resuming a cancelled task without the flag | `--resume-cancelled`, on the user's instruction |
| `handoff_same_owner` | 3 | Handoff to the current owner | - |
| `done_final` | 3 | Any write to a `done` task | register a related task (`--related`) |
| `merge_not_on_main` | 3 | Merge commit not reachable from main | fetch and update main, then rerun |
| `merge_not_merge_commit` | 3 | The commit has one parent (squash or rebase) | `spec_ref` cannot survive; report to the user (rules §11) |
| `revision_not_merged` | 3 | The reviewed revision is not in the merge | check that the right PR merged |
| `spec_ref_invalid` | 3 | `spec_ref` does not resolve | copy it from the PR "Final specification" line |
| `spec_ref_not_final` | 3 | Not the last `spec(<ID>)` commit merged | shows the derived `spec_ref` |
| `invalid_value` | 2 | Malformed flag value | names the expected format |
| `cas_exhausted` | 4 | Lost the compare-and-swap 5 times | retry |

Warnings (exit 0): `push_failed`, `no_remote`, `dependency_not_done`, `title_changed`, `forge_unavailable`.

### 8.5 Registry commits

One commit per successful command, subject `state(<ID>): <status>/<stage>, <summary>` (rules §7; the `/<stage>`
part is omitted while a planned record has no stage), author and committer from the user's Git identity, parent the
previous tip.

| Command | Summary part |
|---|---|
| `task plan` | `plan "<title>"` |
| `task register` | `registered by <owner>` |
| `task move` | the changed fields, comma-separated: `status running`, `stage implementation`, `revision 7cd18e4`, `tier 2`, `spec <path>`, `agents impl-1`, `pr 42`, `next <actor>` |
| `task block` | `block B-n <kind>` |
| `task unblock` | `unblock B-n` |
| `task handoff` | `handoff <old> -> <new>` |
| `task done` | `merged <short merge sha>` |
| `init` | subject `state: initialize registry` (root commit) |
| `upgrade` | subject `state: migrate schema <n> -> <m>` |

## 9. Reference: `task status`

### 9.1 Facts

| Fact | Source | When unavailable |
|---|---|---|
| branch, tip | `git rev-parse feature/<ID>` | `missing` |
| worktree | `git worktree list --porcelain`, by branch | `missing` |
| HEAD vs `revision` | `git rev-list --count <revision>..HEAD` in the worktree | `unknown` without a worktree |
| dirty paths | `git status --porcelain=v2` in the worktree | `unknown` without a worktree |
| spec present | the worktree file, and `git cat-file -e <revision>:<spec>` | `missing` |
| merged | first merge commit on main whose ancestry contains `revision` (`git rev-list --ancestry-path --merges --reverse <revision>..main`) | `no` |
| derived `spec_ref` | last `spec(<ID>):` commit reachable from the merge's second parent | absent before merge |
| PR, checks | `gh pr view feature/<ID> --json number,url,state,mergeCommit,statusCheckRollup` | `unknown`, never passing (J6) |

### 9.2 Drift

| Code | Condition | Suggested command |
|---|---|---|
| `branch_missing` | status `running` past `setup`, no `feature/<ID>` | `task block --kind environment` |
| `worktree_missing` | branch exists, no worktree | `git worktree add ../<repo>-worktrees/<ID> feature/<ID>` |
| `head_ahead_of_revision` | HEAD has commits after `revision` | record the new candidate with `task move --revision` once reported |
| `revision_not_on_branch` | `revision` no longer reachable (history rewritten) | `task move --stage implementation`; reviews rerun |
| `dirty_worktree` | uncommitted changes | preserve them (rules §10) |
| `spec_missing` | recorded `spec` not found | commit the work spec, or correct `--spec` |
| `blocked_without_blockers` | status `blocked`, no open blocker | `task move --status <prior>` |
| `tier_missing` | stage after `implementation`, no tier | `task move --tier` |
| `merged_not_done` | merge found, status not `done` | `task done` with the derived values |
| `pr_not_recorded` | `gh` shows a PR, record has none | `task move --pr <url>` |
| `pr_mismatch` | recorded PR differs from the branch's PR | `task move --pr <url>` |
| `owned_by_other` | `SOLOMONAR_OWNER` set and differs from the owner | stop; handoff only on the user's instruction |

### 9.3 JSON shape

```json
{
  "id": "W-001",
  "record": { },
  "registry_commit": "2d4f6a8c0e1b3d5f7a9c1e2b4d6f8a0c2e3b5d7f",
  "facts": {
    "branch": {"name": "feature/W-001", "exists": true, "tip": "992051e56493f5bb8abb4faadcdbc5f271195498"},
    "worktree": {"path": "/work/ledger-worktrees/W-001", "head": "992051e56493f5bb8abb4faadcdbc5f271195498",
                 "ahead_of_revision": 1, "dirty": [{"path": "internal/auth/session_test.go", "state": "modified"}]},
    "spec": {"in_worktree": true, "at_revision": true},
    "merge": {"merged": false, "merge_commit": null, "derived_spec_ref": null},
    "forge": {"available": true, "pr": null, "checks": "unknown"}
  },
  "drift": [
    {"code": "head_ahead_of_revision", "message": "HEAD moved past the reviewed revision while reviews run"},
    {"code": "dirty_worktree", "message": "uncommitted work during review; preserve it"}
  ],
  "allowed_next": [
    "solomonar task move W-001 --stage implementation --next <actor>:<action>",
    "solomonar task move W-001 --stage adversarial_review"
  ]
}
```

`allowed_next` lists commands valid from the record under 8.1 and 8.2, filled with known values (`--merge`, the
derived `spec_ref`) when drift supplies them. It never judges which one the orchestrator should run (J1).

## 10. Delivery order

J7 puts `init` and the `task` commands first. The init choices in this spec move part of the adapter work forward:

| Increment | Commands | Completion condition |
|---|---|---|
| 1 | `init` (core, runtimes, catalog extensions), every `task` command, `version`, `help` | UC-1 to UC-4 and UC-11 to UC-21 pass as tests in temporary repositories, including the compare-and-swap retry and repeated `init` as a no-op |
| 2 | `doctor`, `sync`, `runtime` list/add/remove, `extension` list/show/add/remove/new, `guard` | UC-5 to UC-8, UC-10, UC-22 |
| 3 | `upgrade`, `dashboard` | UC-9, UC-23; an upgrade test between two embedded framework versions |

## 11. Framework changes required

This spec assumes the following framework changes. Each needs the author's decision (section 12) and is applied in
`assets/` separately, in the register's usual way.

| Change | Files |
|---|---|
| Core reduced to the process (CLI-D14): Go specifics leave the core; a language-neutral `code-reviewer` replaces `go-reviewer`, and the Go content moves to the `go` extension (CLI-D1) | `agents/go-reviewer.md` → `agents/code-reviewer.md`; rules §6 specialty table ("Code and correctness: code changes; loads the installed extensions that apply"); orchestrator §2 agent prefix `code-n`; `assets/extensions/` new |
| Extension point for review agents: triage lists installed extension agents beside the core specialists, required when their `applies_when` matches | rules §6 specialty table (one row for installed extension agents); orchestrator §3 `specialized_review` |
| Extension skills as approved global rules: `must` rules bind once their decision is approved; IDs `<name>/C-n` cite like spec rule IDs (CLI-D12) | rules §1 authority 2, §6 Consolidation; `report.md` finding `Violates`; implementer and reviewer loading lines; orchestrator packets name the extensions that apply |
| Project-scoped decisions `P/D-n`, appended by the CLI on the user's own command (CLI-D11); a fixed entry format the CLI can read | rules §3 (entry format, project scope); orchestrator §1 (edits only its task's entries); `docs/README.md` template index lines for `decisions.md`, `stack.md`, `architecture.md` |
| `AGENTS.md` template rendered from choices: core skills list, "Extensions" section | `assets/templates/AGENTS.md` |
| Acting identity for owner checks (CLI-D2) | rules §7 command table (`--as <owner>` / `SOLOMONAR_OWNER` on `move`, `block`, `unblock`, `done`); orchestrator §3 (pass `--as <owner>` on every `task` write) |
| `--next` and `--status` shorthands on `task block` and `task unblock` | rules §7 command table |
| `doctor` check list | rules §11 (reference section 5 UC-10 instead of "checks the repository settings") |
| Init choices and the manifest | design note "Phase 2: Go initialization contract" (point to this spec) |
| Adapter stubs instead of installed copies | register G11, G12 resolution |

## 12. Open decisions

| ID | Question | Options | Recommendation | Decision |
|---|---|---|---|---|
| CLI-D1 | How language knowledge reaches reviews | a) one language-neutral `code-reviewer` loads the installed extension skills that apply to the change; b) one `<lang>-reviewer` role per language, rendered into the rules §6 table | a: one role and one rules table, a language is one skill, and mixed-language changes need no extra reviewer | **a**, author, 2026-10-20 |
| CLI-D2 | How the CLI knows the acting orchestrator | a) `--as <owner>` with `SOLOMONAR_OWNER` default; b) no check, ownership stays behavioral | a: rules §7 promises ownership validation; it remains a behavioral contract, not authentication | **a**, author, 2026-10-20 |
| CLI-D3 | Claude Code root instructions (G10) | a) create a one-line `CLAUDE.md` (`@AGENTS.md`) when absent, never modify an existing one; b) nothing, if Claude Code is verified to load `AGENTS.md` natively | a, unless the verification at implementation time shows b | open |
| CLI-D4 | Does `init` commit on main | a) never; print the commit command; `doctor` fails until committed; b) `--commit` flag; c) always | a: commits on main stay the user's; b can be added later | open |
| CLI-D5 | Prompt implementation | a) line-based prompts with the standard library; b) a full-screen prompt library (a new dependency, needs approval) | a: no dependency, scriptable, testable with piped input | open |
| CLI-D6 | Extension sources in v0.1 | a) embedded catalog and local directories; b) a, plus Git URLs | a: remote sources bring pinning and trust questions; add later with a recorded commit and hash | open |
| CLI-D7 | Adapter file form | a) stubs that point to `docs/framework/`; b) full copies; c) symlinks | a: one source of content, small diffs, works where symlinks do not | open |
| CLI-D8 | Binary name: `go install ./cmd/cli` produces `cli` | a) rename the package directory to `cmd/solomonar`; b) keep `cmd/cli`, document `go build -o solomonar ./cmd/cli` and set the name in release builds | a: `go install` then yields `solomonar` | deferred, author, 2026-10-20 |
| CLI-D9 | Codex specifics: skill path, native role definitions, hook support | verify each against Codex documentation before increment 1 | adopt what is verified; fall back to packets naming role files and no hook | open |
| CLI-D10 | `workflow` and `review` extensions in the catalog | a) add `baseline` (records commands and known failures, rules §11) and `decision` (prepares a decision entry, rules §3) as `workflow` skills; b) none in v0.1, mechanism only | b for v0.1, as CLI-D13 fixes the catalog; a is the first candidate after it | open |
| CLI-D11 | How a change of stack extensions is approved and recorded | a) `--decision <ID>` names an approved entry; otherwise the user's own command appends an approved project entry `P/D-n` per group, superseding the previous one, plus a line in `docs/stack.md` or `docs/architecture.md`; b) extensions install without any decision record; c) the CLI only accepts `--decision`, and the user writes every entry by hand | a: choosing a stack extension is an architecture or dependency decision (rules §3); the user's command is that explicit decision, and the record keeps rules §1 authority traceable | open |
| CLI-D12 | Rule IDs and force inside extension skills | a) `<name>/C-n` with `must` (approved global rule, blocking) and `should` (suggestion); b) every extension rule binding; c) extensions are guidance only | a: matches the finding handling in rules §6 and keeps preferences out of the blocking set | open |
| CLI-D13 | v0.1 extension catalog | a) a catalog for every group; b) a small catalog matching the stack in use | b: `go`, `htmx`, `templ`, `tdd`; other groups take project extensions until catalog ones are written | **b**, author, 2026-10-20: `go`, `htmx`, `templ`, `tdd` |
| CLI-D14 | Core boundary | a) core = what `rules.md` names as a process step or required role: `orchestrator`, `implementer`, `code-reviewer`, `security-reviewer`, `spec-reviewer`, `adversarial-reviewer`; `prototype`, `spec`, `work-spec`; b) core = `orchestrator`, `implementer`, `work-spec`; reviewers and discovery skills become extensions | a: the review gates and the prototype-to-spec flow are the process; making them optional turns a required review into a choice | **a**, author, 2026-10-20: the current framework in `assets/framework/` is the core, with `go-reviewer` made language-neutral as `code-reviewer` (CLI-D1) |
