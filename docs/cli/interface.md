# Solomonar CLI v0.1: interface specification

Status: design candidate, 2026-10-20. Phase 2 reference for the `solomonar` command-line interface: every command,
its arguments, its interaction, and its output, written as use cases. Behavior that the framework already fixes
(`assets/framework/rules.md` section 7 and `registry.md`, register sections G and J) is restated here only as
interface; when they differ, the framework files win. Example SHAs, hashes, and paths are illustrative but have their real
length and shape.

Increment 1 (section 10) is implemented and released as v0.1.0; this spec keeps the final v0.1 behavior. The
accepted workflow-audit corrections are applied to these contracts and the framework assets on 2026-10-21. Runtime and fault-injection acceptance tests remain implementation work.

The Copilot CLI contract was added on 2026-10-23 (acceptance record: [section 10](#copilot-runtime-acceptance)):
adapter generation, manifest tracking, and `init --runtime copilot` are implemented. The live Copilot workflow was
confirmed on 2026-10-25; the minimum supported runtime version is not established. Runtime management is a separate delivery (UC-5), now
implemented, and hooks remain deferred; this addition defines no Copilot guard hook. Existing Claude and Codex contracts are unchanged.

The extension store, the minimal `upgrade`, and `extension list|add|remove` were specified on 2026-10-24 (CLI-D27
to CLI-D31) and are
implemented. They replace the earlier extension design: no rendered combinations, no project-extension metadata
in the manifest, and only the `list`, `add`, and `remove` subcommands.

`uninstall` was specified on 2026-10-24 (UC-27, CLI-D32 to CLI-D35, architecture AD-16) and is implemented. It
removes the project installation from one checkout. Removing an
extension stays `extension remove`, and removing the executable stays a package-manager or manual step.

`runtime list|add|remove` were specified on 2026-10-24 (UC-5, CLI-D36) as a commands-only delivery and are
implemented. Guard hooks, their installation, and their removal stay deferred; the hook design in 4.2, 4.5,
4.7, and UC-22 is future work, not current behavior.

The command pages in `site/src/docs/cli/` are the reference for each command's synopsis, flags, when to use it, who
runs it, and its typical output; this spec does not repeat them. It holds the contract beneath those pages: check
order, exact texts, file planning, the manifest, JSON data, and the task rules. A use case cites its command page and
keeps only the examples the page does not show. When a command changes, update its page and this spec together.

Contents: 1 Principles · 2 Command map · 3 Conventions · 4 Project choices and files · 5 Use cases: project setup ·
6 Use cases: task state · 7 Use cases: runtime integration · 8 Reference: task validation · 9 Reference: `task
status` · 10 Delivery order · 11 Open decisions

## 1. Principles

- **Guard and oracle, never a driver** (J1). The CLI validates and records what the orchestrator or the user
  decided, and reports what Git shows. It never launches agents, calls model providers, stores review verdicts or
  reports, or runs a daemon. The only long-running command is the read-only `dashboard`.
- **One writer for state.** Only `solomonar task` commands write the registry on `refs/heads/workflow-state`
  (`state.json` and `tasks/<ID>.json`, 8.6), each write one compare-and-swap commit (rules section 7).
- **Project files are copied, then owned.** `init` and the commands that add content copy embedded assets into
  the project and record each managed file in a manifest. A repeated operation with identical content is a no-op;
  a differing file is a conflict, never a silent overwrite.
- **Light core, opt-in extensions.** The framework core is only the process: its rules, roles, and skills. Technology
  knowledge, supporting procedures, and extra reviewers are extensions, skills or agents, that the user chooses at
  `init` or adds later (4.1).
- **Portable source, runtime adapters.** `docs/solomonar/` is the single source for rules, roles, skills, and
  extensions. A runtime (Claude Code, Codex, Copilot CLI) gets small generated adapter files that point to that source; it
  never gets a second copy of the content.
- **Actionable rejections.** A rejected command names the failed check and the corrective action, writes nothing,
  and exits non-zero.
- **A clear local commit point.** When `init` creates the registry, it finalizes every project file, including
  the manifest, before the registry compare-and-swap. A handled failure before that swap rolls back this run's
  file changes; after it, optional publication may warn but cannot turn the local write into "nothing written".
  Registry migration is a separate `task migrate` operation; `upgrade` changes project files only.
- **Same data for people and agents.** Every command has a human form and a `--json` form with the same content.
- **No network except Git and `gh`.** Network use is limited to `git push` of `workflow-state` and read-only `gh`
  queries. Both are optional; their failures are warnings with an explicit `unknown`, never a pass.

## 2. Command map

Purpose, who runs each command, and availability: [the CLI overview](../../site/src/docs/cli.md). This map adds what
each command writes and its use cases.

| Command | Writes | Use case |
|---|---|---|
| `init` | project files, manifest, `workflow-state` | UC-1 to UC-4 |
| `runtime list\|add\|remove` | `list` nothing; `add` and `remove`: `runtime:<name>` adapter files, any managed file the shared plan restores (4.5), manifest; never a hook or runtime settings file (CLI-D36) | UC-5 |
| `extension list\|add\|remove` (alias `ext`) | `list` nothing; `add` and `remove`: catalog extension files in `docs/solomonar/extensions/<name>/` (never a project extension's directory), `docs/decisions.md`, `docs/stack.md`, `docs/architecture.md`, adapter files, manifest, `AGENTS.md` | UC-6, UC-7 |
| `upgrade` | managed files (create, update, remove), manifest | UC-9 |
| `uninstall` | managed files (remove), the managed part of `AGENTS.md`, the generated index line of `docs/README.md`, manifest (remove); never `workflow-state` | UC-27 |
| `doctor` | nothing | UC-10 |
| `task plan`, `register`, `move`, `block`, `unblock`, `handoff`, `done` | `workflow-state` | UC-11 to UC-15, UC-18 to UC-21 |
| `task migrate` | `workflow-state` | UC-25 |
| `task confirm` | `workflow-state` | UC-26 |
| `task show`, `task status` | nothing | UC-16, UC-17 |
| `guard` (deferred) | nothing | UC-22 |
| `dashboard` | nothing | UC-23 |
| `version`, `help` | nothing | UC-24 |

"Runtime" is the agent product that runs sessions (Claude Code, Codex, Copilot CLI). "Agent" and "role" keep their framework
meaning (orchestrator, implementer, reviewers), so the CLI never uses `agent` for a runtime.

## 3. Conventions

### 3.1 Invocation and global flags

```
solomonar [global flags] <command> [<subcommand>] [flags] [args]
```

No command, a command group alone (`task`), and an unknown command or subcommand are `unknown_command`; an
unknown flag, a missing required flag, or an unexpected argument is `invalid_usage`. Both exit 2 with the fix "run
solomonar help". `solomonar help <command>` prints the same text as `<command> -h`: the usage line, the
description, and a pointer to `solomonar help` for the global flags. `ext` is an alias for `extension`:
`solomonar ext list` runs `solomonar extension list`, and help and JSON `command` name the full form.

Global flags: [CLI overview, Global flags](../../site/src/docs/cli.md#global-flags). Command flags shared by several
commands:

| Flag | Commands | Meaning |
|---|---|---|
| `-y`, `--yes` | `init`, `runtime`, `extension`, `upgrade`, `uninstall` | Accept the plan without the confirmation prompt; with `init`, take defaults for every unanswered choice. |
| `--dry-run` | writing commands except `task migrate` | Run every check and print the plan or the record that would be written; write nothing; exit as the real run would. |
| `--keep <path>`, `--overwrite <path>` | `init`, `runtime add\|remove`, `extension add\|remove`, `uninstall` (UC-27); `upgrade` uses `--take-new` instead of `--overwrite` | Resolve a managed-file conflict explicitly; `--keep` preserves local content without making it the distributed base (4.5). On a removal conflict, keep untracks the file and leaves it on disk; overwrite (`--take-new`) deletes it. |
| `--decision <ID>`, `--rationale <text>` | `extension add\|remove` | Record the change of a stack group under an approved decision, or give the rationale of the `P/D-n` the CLI writes (4.3); on a call that changes no stack group, either is `invalid_usage`. |
| `--as <owner>` | `task move`, `block`, `unblock`, `done` | Acting orchestrator, checked against the record's `owner` (CLI-D2). Defaults to `SOLOMONAR_OWNER`; neither is `owner_required`. Agents pass `--as` on every command, because a runtime may start a new shell per command and lose the variable; the variable is a convenience for the user's terminal. The check guards against mistakes (wrong task, a stale session after a handoff), not against impersonation. |

Environment variables: `SOLOMONAR_OWNER` (default for `--as`), `NO_COLOR`. No other environment variable changes
behavior.

### 3.2 Repository context

- The CLI locates the repository with `git rev-parse --git-common-dir`, so it works from the main checkout or any
  feature worktree. The registry ref is shared by all worktrees.
- Project settings (main branch, remote) come from `.solomonar/manifest.json` in the current worktree root;
  defaults are `main` and `origin`, used only when the manifest is absent. Task commands read it after locating
  the repository and checking Git, before the registry: a newer manifest is `cli_outdated` (exit 3), an invalid
  one `manifest_invalid` (exit 1), under the command's usual label (`task move W-001 rejected` or `failed`), and
  nothing is written.
- A command that needs the repository fails outside one with `not_a_repository` (exit 3), message
  `<absolute path> is not a Git repository`, fix "run inside a Git repository or pass -C <path>".
- Requirements: Git 2.40 or later, otherwise `git_too_old` (exit 1, fix "install git 2.40 or later"); `gh`
  optional. A colocated Jujutsu repository (`.jj/`) is a warning: v0.1 is Git-only (C6), and jj commands must not
  be used on `workflow-state`. In v0.1 only `init` reports it (`jujutsu_detected`, UC-4) until `doctor` exists.

### 3.3 Human output

- Results go to stdout; warnings and errors go to stderr, prefixed `warning:` or `error:`.
- Full SHAs appear wherever a value will be copied into another command; a short SHA (7) appears only beside
  a full one or in tables.
- File plans use one verb per line: `create`, `update`, `append` (content added at the end of an existing file; 4.3),
  `unchanged`, `keep` (project-modified, left as is; 4.5), `remove`, `conflict`, `skip` (project-owned, never
  touched). An unmanaged file whose content equals what the CLI
  would write is adopted into the manifest and shown as `unchanged  <path>  adopted` (CLI-D17).
- A plan line is `  <action padded to 9>  <path>`; when the change has a reason, two spaces and the reason follow,
  with paths padded to the widest path that has a reason. `init`'s registry line is `  registry   create
  refs/heads/workflow-state with an empty registry` or `  registry   unchanged refs/heads/workflow-state`; with
  `--git-init` the plan's first line is `  git        init -b <main-branch>` (UC-4).
- The interactive `init` plan shown before `Proceed?` groups its lines and collapses uniform file sets (UC-1); `init
  --dry-run` and the `file_conflict` plan list every file in this form.
- A counts line is `<n> file(s) <first action>, <n> <action>, …` over the non-zero actions in the order created,
  updated (`append` included), removed, kept, unchanged, skipped; the manifest counts. Only the first count carries `file`
  (one) or `files` (`1 file created, 17 unchanged, 1 skipped`); a conflict count is `1 conflict` or `<n> conflicts`.
- Every writing command ends with what was written (commit, files) and, when useful, a `Next:` block.

### 3.4 JSON output

The envelope, on success and on failure: [CLI overview, Output formats](../../site/src/docs/cli.md#output-formats).
A warning is `{"code", "message"}`, such as `{"code": "push_failed", "message": "git push origin workflow-state
failed: could not resolve host"}`; on failure `data` is absent. `command` names the command run; `help` and a top-level `-h` report `help`, `<command> -h` reports the command,
and the help data is `{"help": "<text>"}`. `data` per command is given in its use case. Field names are
`snake_case`; timestamps are RFC 3339 UTC; SHAs are full.

### 3.5 Exit codes

The codes 0 to 5, 130, and 143: [CLI overview, Exit codes](../../site/src/docs/cli.md#exit-codes). Every non-zero
code means nothing was written; for 1, 130, and 143 a partial file write is rolled back. 130 and 143 are SIGINT and
SIGTERM (128 + signal) before the commit point, with the code `interrupted`.

The exit code reports what was written, not only that a signal arrived (CLI-D18). A signal after the registry
compare-and-swap succeeded leaves the write in place: the command exits 0 with the warning `push_interrupted`, so a
retry never repeats a write such as `task plan`. A second signal ends the process at once. For `dashboard`, SIGINT
is the normal stop and exits 0.

### 3.6 Errors

Human form, on stderr:

```
error: task move W-001 rejected (revision_not_on_branch)
  check: --revision must be a full SHA reachable from feature/W-001
  found: 1902e3d6fc4e78a0bcc50ba12b882769afbf4a8c is not on feature/W-001
  fix:   commit the candidate on feature/W-001 and pass its full SHA: git -C <worktree path> rev-parse HEAD
nothing was written
```

The first line is `error: <message> (<code>)`, and `message` carries the same text in JSON. A rejection's message
is `<command> rejected`, with the positional task ID when the command takes one (`task show W-009 rejected`), or
`<command> failed` for an exit-1 code. An unexpected failure is `unexpected_error` (exit 1) with the underlying
error as the message.

Every command may report `unknown_command`, `invalid_usage`, `unexpected_error`, and `interrupted`; a command that
needs the repository also `not_a_repository` and `git_too_old` (3.2). The error code catalog for task commands is
in 8.4; project commands use `not_a_repository`, `not_initialized`,
`already_initialized`, `file_conflict`, `unknown_runtime`, `unknown_extension`, `core_item`,
`name_collision`, `invalid_value`, `unknown_group`, `extension_requires`, `extension_conflict`, `decision_not_found`,
`decision_not_approved`, `registry_exists_invalid`, `manifest_invalid`, `interactive_required`,
`checks_failed`, `cli_outdated`, `framework_outdated`, `interrupted`.

`not_initialized` (exit 3) for a project command: `upgrade`, `extension add|remove`, and `runtime add|remove` need
`.solomonar/manifest.json`; without it they reject with check `the project is initialized`, found `no
.solomonar/manifest.json in <root>`, and the fix "run solomonar init". (The task commands' `not_initialized`, 8.4,
names the missing registry instead.) `extension list` and `runtime list` work without a manifest, and `uninstall` without one is
a no-op (UC-27).

`cli_outdated` (exit 3, CLI-D19): the manifest's `framework_version` or `schema_version`, or the registry's
`schema_version`, is newer than this CLI supports. The CLI checks the versions before decoding either file
strictly, so a newer project is never reported as `manifest_invalid` or `registry_invalid`. Every command that reads
the manifest or the registry rejects with it and writes nothing, with the fix "install a newer solomonar release".
`version` prints the project's versions and exits 0 (UC-24), and `doctor` reports it as a `fail`.

`framework_outdated` (exit 3, CLI-D31): the manifest's `framework_version` is older than this CLI's embedded
framework. `extension add` and `extension remove` reject with it after the manifest validates and before any
extension is resolved, and write nothing: check `the project's framework matches this CLI`, found `manifest
framework <v>, this CLI ships <v>`, fix "run solomonar upgrade". `runtime add` and `runtime remove` reject with
it at the same point (UC-5). `extension list`, `runtime list`, `upgrade`, and `uninstall` accept an older project; `init` keeps its own rules (UC-3).

`manifest_invalid` (exit 1, like `registry_invalid`): `.solomonar/manifest.json` fails strict decoding or the 4.7
validation. The message names the field, and the fix is "inspect .solomonar/manifest.json; never edit it by hand".

### 3.7 Prompts

- Prompts appear only when stdin and stdout are terminals and neither `--yes` nor `--json` is set. Otherwise a
  missing required choice is error `interactive_required` (exit 2) naming the flag that supplies it.
- Text prompts and confirmations are line-based; the multi-select redraws its own lines in place (no full-screen UI,
  CLI-D5).
- A text prompt is `<label> [<default>]: `, or `<label>: ` without a default; the answer is trimmed, and empty gives
  the default. In a text prompt or a confirmation, a final line without a newline at end of input counts as input
  closed.
- A confirmation is `<question> [Y/n] ` or `[y/N] `, default capitalized; `y`, `yes`, `n`, `no` in any case; empty
  gives the default; any other answer prints `  answer y or n` and asks again. Declining the confirmation that
  precedes a write prints `Cancelled; nothing was written.` on stdout and exits 0.
- A multi-select lists one row per item, `  <pointer><box> <name>  <description>`: the pointer is `> ` on the
  cursor row and two spaces elsewhere, the box `[x]` or `[ ]`, the name padded to the widest; trailing spaces are
  trimmed, so an empty description leaves no padding. The cursor row reads `  > [x] claude  …`, the others
  `    [ ] codex   …`. The help line `  ↑/↓ move · Space toggle · Enter accept` follows. Keys: ↑/↓ or `k`/`j` move the
  cursor (clamped, no wrap); Space toggles the cursor row; `a` selects all; `n` selects none; Enter accepts the
  marked items; any other key is ignored without a redraw, as is `ESC` followed by a byte other than `[`, or `ESC [`
  followed by anything but `A` or `B`. On accept the list is redrawn once with the pointer column as two spaces and
  without the help line: `    [x] <name>  <description>`.
- End of input on any prompt is `read answer: input closed` (`unexpected_error`, exit 1); any other read error is
  `read answer: <err>`. Ctrl-C in the multi-select prints nothing more, restores the terminal, and ends the command
  `interrupted` (exit 130, 3.5).

## 4. Project choices and files

### 4.1 Core and extensions

The framework core holds only the process. Every other capability is an extension, a skill or an agent, that the
user chooses at `init` or adds later. The CLI installs and removes extensions; the core never depends on one being
present.

| Part | Contains | Installed |
|---|---|---|
| Core: the process | `docs/solomonar/rules.md`, `registry.md`, `report.md`, `reviewer.md`, `context.md`, conditional procedures in `workflow/` and `reports/`; the process roles `orchestrator`, `implementer`, `security-reviewer`, `correctness-reviewer`, `spec-reviewer`, `adversarial-reviewer` and their references; the process skills `solomonar` (user entry point, CLI-D15), `prototype`, `spec`, `work-spec` and their references; `AGENTS.md`; `docs/README.md`; the empty registry on `refs/heads/workflow-state`; `.solomonar/manifest.json` | always; no part can be removed while the framework stays installed (`extension remove` rejects a core item with `core_item`); only `uninstall` removes the core, together with the whole installation (UC-27, CLI-D32), and it keeps the registry |
| Runtimes | Adapter files for `claude`, `codex`, `copilot` (4.2) | zero or more; default: those found on `PATH` |
| Extensions | Skills (knowledge of a language, library, or practice; or a supporting procedure) and agents (extra specialist reviewers), grouped by decision type (4.3) | zero or more; default at `init`: the catalog extensions marked `default` or detected in the repository (4.3) |

Core boundary (CLI-D14, decided): the core is the current framework in `assets/framework/`, the items `rules.md`
names as a step or a required role of the process. The core carries no technology-specific content: Go rules live in the `go`
extension, and the four core reviewers stay language-neutral (CLI-D1; design note "Light core and extensions").

A project whose stack is not known yet starts with no stack extension: the stack is decided later (rules section 3), and
`solomonar extension add` follows that decision (UC-6).

Installation makes these sources available; it does not load them all into agent context. `AGENTS.md` owns
request routing, shared entry contracts name their required inputs, and stage/mode procedures load only when
their trigger applies. The CLI copies nested references recursively and generates the same thin runtime stubs;
selective loading changes no task schema or lifecycle gate.

### 4.2 Runtime adapters

An adapter adapts the portable framework to one runtime: it registers the instructions, the skills, and the roles
where the runtime discovers them, and restricts tools where the runtime allows. Adapter files are generated stubs:
the runtime frontmatter plus a one-line body that loads the portable file, so the content lives only in
`docs/solomonar/` (CLI-D7). Claude and Codex forms were checked against documentation on 2026-10-21 (CLI-D9);
Copilot CLI fields were checked on 2026-10-23 against its
[CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference),
[skills documentation](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-skills),
[agent configuration](https://docs.github.com/en/copilot/reference/custom-agents-configuration), and
[delegated instruction loading](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli).
These are documentation checks; live Copilot acceptance and the minimum supported version remain TASK-04.

| Concern | `claude` (Claude Code) | `codex` (Codex) |
|---|---|---|
| Root instructions | `CLAUDE.md` containing `@AGENTS.md`, create-only (CLI-D3): Claude Code reads `AGENTS.md` natively only when no `CLAUDE.md` exists, so the import stays | `AGENTS.md`, read natively; nothing generated |
| Skills | `.claude/skills/<skill>/SKILL.md` stub per installed skill | `.agents/skills/<skill>/SKILL.md` stub per installed skill |
| Roles | `.claude/agents/<role>.md` per delegated role; reviewers get a read-only tool list; `adversarial-reviewer` gets the configured model | `.codex/agents/<role>.toml` per delegated role (`name`, `description`, `developer_instructions`, `model` for `adversarial-reviewer`); no tool allowlist, so read-only reviewers are a behavioral contract (rules section 11) |
| Orchestrator | no native agent file: it is a top-level session (G13) | same |
| State guard (deferred, CLI-D36) | future: `PreToolUse` hook running `solomonar guard --runtime claude`, merged into `.claude/settings.json`; not generated today | future: `PreToolUse` hook running `solomonar guard --runtime codex`, merged into `.codex/hooks.json`; Codex runs it only after the user trusts the project's `.codex/` layer and reviews the hook (`doctor` warns until then); not generated today |
| Adversarial model | `--adversarial-model claude=<model>` writes `model: <model>` in its role stub | `--adversarial-model codex=<model>` writes `model = "<model>"` in its role stub |

| Concern | `copilot` (GitHub Copilot CLI; live acceptance pending) |
|---|---|
| Root instructions | `AGENTS.md`, read natively; no additional root instruction file generated |
| Skills | `.github/skills/<skill>/SKILL.md` stub per installed skill; `solomonar` and every `invocation: user` extension skill add `disable-model-invocation: true` |
| Roles | `.github/agents/<role>.agent.md` per delegated role; `name`, `description`, and `include-custom-instructions: true`; reviewers add `tools: [read, search, execute]` |
| Orchestrator | no native agent file: it remains the top-level session (G13) |
| State guard | none, current or planned; no hook file or manifest hook entry generated |
| Adversarial model | `--adversarial-model copilot=<model>` writes the `model` field only in the adversarial-reviewer stub; absent when unspecified |

The state guard rows are the deferred hook design (CLI-D36): no current command, `init` and `runtime add|remove`
included, writes a hook, a settings file, or a manifest `entry`. They ship with `guard` (UC-22), after its real hook
payloads are verified.

Copilot skill and role bodies load the same portable paths as the existing adapters. Role bodies explicitly load
`AGENTS.md`, the portable role, and the files its first lines require. Copilot metadata is serialized as valid YAML
strings preserving the source values, including quotes and YAML-significant characters; this does not change
Claude or Codex serialization. Copilot file ownership is `runtime:copilot`, with SHA-256 tracking under 4.5.

Copilot's `model` is a preference: runtime overrides, unavailable models, and session model selection can affect
the actual model. Solomonar does not authenticate, check model availability, or enforce a different model;
record actual runtime behavior during acceptance. No additional model-policy fields are generated.

Delegated roles are every core role in `docs/solomonar/agents/` except `orchestrator`, plus the installed extension
agents; skills are the core skills and the installed extension skills (4.3). Each stub preserves the `name` and
`description` values from the portable file's frontmatter: for a catalog extension the embedded catalog file
(byte-identical to its store copy), for a project extension its file in the store; its path uses the skill directory
or role file name instead (the two match for every shipped file). An extension stub loads its main file in the store:
`docs/solomonar/extensions/<name>/SKILL.md` for a skill, `docs/solomonar/extensions/<name>/<name>.md` for an agent.

Body-only edits apply when the runtime next loads the portable source. Core and catalog extension metadata comes
from the release, so an edit to it in `docs/solomonar/` never reaches a stub; it is a project edit that `upgrade`
keeps or reports as a conflict (UC-9). A project extension's metadata is read from the store, so after changing it
the user runs `solomonar upgrade`, which regenerates the stubs at the same version (CLI-D37). The stub's second line
says which case applies: `Body edits apply directly; the name and description come from the solomonar release.` for
core and catalog extension stubs, `` Body edits apply directly; run `solomonar upgrade` after changing the name or
description. `` for project extension stubs, and the same text without backticks after `# ` in Codex TOML and YAML.
Never hand-edit a generated stub. An extension agent takes
the reviewer form: claude `tools: Read, Grep, Glob, Bash`; codex the role TOML without `model`;
copilot `tools: [read, search, execute]` without `model`.

Plan order: runtimes always in the order `claude`, `codex`, `copilot`, whatever the flag order. `claude`: `CLAUDE.md`, skill
stubs, role stubs; `codex`: skill stubs, the `.agents/skills/<skill>/agents/openai.yaml` policy of each user-invoked
skill sorted by skill, role TOMLs; `copilot`: skill stubs, role stubs. Skills and roles,
core and extension together, are sorted by directory or file name.

Skill stub (the same portable-source body for every runtime; the claude and copilot stubs of a user-invoked skill
add `disable-model-invocation: true`; Copilot YAML strings preserve the same metadata values; an extension stub names
its store path, such as `docs/solomonar/extensions/go/SKILL.md`):

```markdown
---
name: work-spec
description: Prepares or refines one registered task's work specification in `docs/work/`, before implementation or when its scope or acceptance criteria change. Use from the orchestrator in the task worktree; use `spec` for application-wide behavior and `prototype` for discovery.
---

Generated by solomonar X.Y.Z from `docs/solomonar/skills/work-spec/SKILL.md`; edit that file.
Body edits apply directly; the name and description come from the solomonar release.
Load `docs/solomonar/skills/work-spec/SKILL.md` and follow it.
```

Role stub (Claude Code; `implementer` has no `tools` line and inherits the session's tools, `adversarial-reviewer`
adds `model: <model>` when configured):

```markdown
---
name: security-reviewer
description: Reviews one candidate revision for security defects in authorization, data isolation, external input, files, secrets, trust boundaries, and security-relevant dependency or configuration changes.
tools: Read, Grep, Glob, Bash
---

Generated by solomonar X.Y.Z from `docs/solomonar/agents/security-reviewer.md`; edit that file.
Body edits apply directly; the name and description come from the solomonar release.
Load `AGENTS.md`, then `docs/solomonar/agents/security-reviewer.md` and every file its first lines require.
```

Role stub (Codex; `adversarial-reviewer` adds `model = "<model>"` when configured; basic strings escape `\` and `"`
only; a control character in a value, tab included, is an error: `render <path>: control character in "<value>"`):

```toml
# Generated by solomonar X.Y.Z from docs/solomonar/agents/security-reviewer.md; edit that file.
# Body edits apply directly; the name and description come from the solomonar release.
name = "security-reviewer"
description = "Reviews one candidate revision for security defects in authorization, data isolation, external input, files, secrets, trust boundaries, and security-relevant dependency or configuration changes."
developer_instructions = "Load AGENTS.md, then docs/solomonar/agents/security-reviewer.md and every file its first lines require."
```

`.agents/skills/solomonar/agents/openai.yaml` (Codex; a user-invoked extension skill gets the same file at
`.agents/skills/<name>/agents/openai.yaml`, whose first line names `docs/solomonar/extensions/<name>/SKILL.md`):

```yaml
# Generated by solomonar X.Y.Z; edit docs/solomonar/skills/solomonar/SKILL.md.
# Body edits apply directly; the name and description come from the solomonar release.
policy:
  allow_implicit_invocation: false
```

Entry skill (CLI-D15): the `solomonar` skill is invoked only by the user. Its `claude` stub adds
`disable-model-invocation: true` to the frontmatter, so the model never loads it on its own and its description
stays out of the session context; the "Route a request" section of `AGENTS.md` covers sessions started without
it. The `codex` stub is accompanied by `.agents/skills/solomonar/agents/openai.yaml` with `policy:
allow_implicit_invocation: false`; Codex's documentation does not say whether the description then stays out of
the context, so the skill's own boundary (invoked only by the user) remains a behavioral contract.

User-invoked skills (CLI-D29): `solomonar` and every installed extension skill whose `extension.json` sets
`invocation: user` (4.3) get the same treatment: `disable-model-invocation: true` in the claude and copilot stubs, and
one codex `.agents/skills/<id>/agents/openai.yaml` per user-invoked skill with `allow_implicit_invocation: false`.
The core `solomonar` skill has no `extension.json`; it is user-invoked by CLI-D15. They are listed in their own `AGENTS.md`
line, not in the skills the model loads when they apply (4.6). No other skill is user-invoked; the CLI never decides
it by an extension's name.

The `copilot` entry stub sets `disable-model-invocation: true` and retains the default user invocation through
`/solomonar`; other Copilot skills remain eligible for normal skill discovery. No Codex policy file is generated
under `.github/skills/`. Copilot CLI 1.0.103 does not expand `/solomonar` in prompt mode (`-p`), and the flag
blocks the model's skill tool, so prompt-mode sessions cannot invoke the entry skill; the flag is retained and
interactive sessions remain the supported entry. The same known limitation applies to every user-invoked extension
skill, such as `extension-authoring`: invoke it in an interactive Copilot session.

`CLAUDE.md` is create-only (CLI-D3, G10): absent, it is created and managed from then on; present and unmanaged, it
is `skip`, never a conflict, and is not recorded in the manifest, even when it is not a regular file. The reason is
`exists; add this line so Claude Code loads the framework:  @AGENTS.md` when it is not a regular file or has no line
exactly `@AGENTS.md`, else `exists and imports AGENTS.md`. A managed `CLAUDE.md` follows the 4.5 table.

Tool lists (Claude Code): `implementer` inherits the session's tools; every reviewer gets `Read, Grep, Glob, Bash`
(read-only for tracked files is a behavioral contract, rules section 11; `Bash` is needed to run checks).

Tool lists (Copilot CLI): `implementer` omits `tools` and inherits available tools; every reviewer, extension
reviewers included, gets `read`, `search`, and `execute`. Direct editing and delegation tools are excluded;
shell commands can still write, so `reviewer.md`'s read-only contract applies. `include-custom-instructions: true`
opts delegated roles into repository instructions; a session started with custom instructions disabled does
not establish acceptance of that discovery behavior.

**Starting a session.** Print the selected runtime's invocation in `Next`: Claude Code uses `/solomonar`, Codex
CLI/IDE uses `$solomonar` or selection from `/skills`; Copilot CLI uses `/solomonar`. The portable prompt is
"Use the solomonar skill". A new
managed task takes the orchestrator role in that same top-level session. Adapter acceptance includes a real
skill-discovery and delegation smoke test, not just checking generated paths (architecture section 11).
Copilot acceptance covers a feature worktree and coexistence with Claude and Codex adapters, including which
definitions win discovery. Copilot IDE and cloud-agent compatibility are outside this addition.

### 4.3 Extensions

An extension is one skill or one agent, plus the metadata in its `extension.json` that the CLI uses to install,
activate, and suggest it. Extensions are grouped by the type of decision that selects them, so the project's
capabilities follow its approved architecture and stack: choosing Go installs the Go rules and nothing about another
language.

| Kind | Adds | Lives in | Runtime adapter |
|---|---|---|---|
| `skill` | Knowledge (the rules of a language, library, or practice) or a supporting procedure | `docs/solomonar/extensions/<name>/`: `extension.json`, `SKILL.md`, support files | a skill stub per runtime (4.2) |
| `agent` | An extra specialist reviewer, with the condition that makes it required | `docs/solomonar/extensions/<name>/`: `extension.json`, `<name>.md`, support files | a role stub per runtime, read-only tools |

| Group | Decision type | Records a decision | Entry title | Document the CLI updates |
|---|---|---|---|---|
| `language` | Programming languages | yes | Programming languages | `docs/stack.md` |
| `architecture` | Structural style and module boundaries | yes | Architecture style | `docs/architecture.md` |
| `frontend` | UI rendering and interaction | yes | Frontend technologies | `docs/stack.md` |
| `backend` | HTTP serving and API style | yes | Backend technologies | `docs/stack.md` |
| `persistence` | Data store and data access | yes | Persistence technologies | `docs/stack.md` |
| `testing` | Test practice and tools beyond the language | yes | Testing practice | `docs/stack.md` |
| `operations` | Build, packaging, deployment | yes | Build and deployment | `docs/stack.md` |
| `workflow` | Procedures that support the process (for example authoring an extension) | no | - | - |
| `review` | Extra specialist reviewers (for example accessibility) | no | - | - |

The groups are fixed. The first seven are stack groups: adopting or dropping one of their extensions is an
architecture or dependency decision. `workflow` and `review` configure how the process runs and record no decision.
A group without catalog extensions takes project extensions (UC-7); the `init` prompt shows only groups that have
at least one catalog extension, and `extension list` only groups with at least one listed extension.

**The store** (CLI-D27). `docs/solomonar/extensions/<name>/` is the one local home of extensions, one flat directory
per name, whatever the group or kind. It holds **only** the installed catalog extensions and the project's own
extensions; a repository never carries the whole catalog.

- An installed **catalog extension** has every file of its catalog directory in the store, `extension.json`
  included, copied verbatim with owner `ext:<name>` and a hash (4.5). Removing it deletes those files.
- A **project extension** is a store directory whose name the catalog does not use and that the manifest does not
  record as a catalog extension. It is project-owned: the CLI reads it but never writes, hashes, or tracks a file in
  its directory. The `extension-authoring` skill or the user writes it; `extension add <name>` activates it, and
  `extension remove <name>` deactivates it, leaving the directory as it was.

Both kinds use the same format and go through one strict loader. An extension is **installed** when the manifest has
an entry for it (4.7), and **available** otherwise.

**`extension.json`.** The catalog's copy lives in the CLI, `assets/extensions/<group>/<name>/extension.json`; a project
extension's in its store directory. The real `go` entry:

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

| Field | Meaning | CLI behavior |
|---|---|---|
| `name`, `group` | Extension ID and its group; `name` matches the directory and the skill or agent `name` | `unknown_extension`, `unknown_group` |
| `kind` | `skill` or `agent` | decides the main file and the adapter stub |
| `title` | One-line name | shown by the `init` prompt and `extension list --json` |
| `requires` | Extensions that must be installed with it | `extension_requires` (exit 3) naming the missing ones; never added silently |
| `conflicts` | Extensions that cannot be installed with it | `extension_conflict` (exit 3) |
| `applies_when` | Skill: when an agent loads it. Agent: when triage requires it | written into `AGENTS.md` (4.6); the skill or agent `description` is authored in its main file and copied as is, never generated from it |
| `paths` | Globs of files the extension governs | written into `AGENTS.md`; the orchestrator uses them to name extensions in a packet; the CLI never inspects a change |
| `detect` | Globs of repository files that suggest the extension; optional, default `[]` | `init` preselects a detected extension; `extension list` marks it `suggested`; never installs it (CLI-D28) |
| `default` | `true` suggests the extension in every project; optional, default `false` | `init` preselects it; `extension list` marks it `suggested` |
| `invocation` | `model` (the model loads the skill when it applies) or `user` (only the user starts it); optional, default `model` | user-invoked stub fields (4.2) and a separate `AGENTS.md` line (4.6) (CLI-D29) |

`detect` and `default` only change what is offered; nothing is installed without the user's choice at `init` or an
`extension add`.

**Detection.** The CLI lists the worktree's files with `git ls-files --cached --others --exclude-standard`, run at
the worktree root (repo-relative, slash-separated paths; ignored files excluded, untracked ones included); outside a
repository, as before `init --git-init` runs `git init`, the list is empty. A glob matches a path segment by segment:
`**` matches zero or more whole segments, and every other segment matches one segment as `path.Match` does (`*`,
`?`, `[...]`, never across `/`). So `go.mod` matches only the root file, and `**/go.mod` matches it at the root and in
any directory. An extension is detected when one of its `detect` globs matches a listed file; its reason is the first
matching path in sorted order. An extension with `default: true` that is not detected has the reason `default`.

**Loading and checks.** Catalog entries are the files `extensions/*/*/extension.json` of the embedded tree; a
directory without `extension.json` is ignored and a missing `extensions` directory is an empty catalog. One loader
reads catalog and project extensions alike, the first failure being the reason:

1. `extension.json` decodes strictly: unknown members, duplicate keys, and wrong types are rejected (`decode
   extension.json: <err>`).
2. `name` equals its directory (`name "<n>" does not match the directory`) and matches `[a-z][a-z0-9-]*` (`name
   "<n>" is not a name of [a-z][a-z0-9-]*`).
3. Catalog only: `group` equals its directory (`group "<g>" does not match the directory`). Both: `group` is one of
   the nine groups (`group "<g>" is not one of <groups>`).
4. `kind` is `skill` or `agent` (`kind "<k>" is not skill or agent`); the directory holds the regular file
   `SKILL.md` for a skill or `<name>.md` for an agent (`kind <k> has no <file>`).
5. `title`, then `applies_when`, is not empty or whitespace-only (`title is missing`, `applies_when is missing`).
6. Every `detect` entry is a valid glob: not empty, not starting with `/`, no empty, `.`, or `..` segment, `**` only
   as a whole segment, every other segment a valid `path.Match` pattern (`detect "<g>" is not a valid glob`).
7. `invocation` is absent, empty, `model`, or `user` (`invocation "<v>" is not model or user`); an agent is never
   user-invoked (`invocation user is only for a skill`).
8. Project only: the main file's frontmatter has `name` equal to the directory and a non-empty `description`
   (`frontmatter name "<n>" does not match the directory`, `frontmatter description is missing`).

Catalog only, after all entries load: the name is in no earlier entry (`name "<n>" is also in <dir>`); then each entry
in sorted name order has every `requires` entry (`requires "<n>" is not a catalog extension`), then every `conflicts`
entry (`conflicts "<n>" is not a catalog extension`), naming a catalog extension. A catalog defect is an
embedded-asset bug, so it is an unexpected error (exit 1) `read catalog <dir>: <reason>`. A project extension's
`requires` and `conflicts` may name catalog or project extensions (`requires "<n>" is not a known extension`,
`conflicts "<n>" is not a known extension`), checked after every store directory loads, in sorted name order. A
project failure never stops `extension list`, which shows it as `invalid: <reason>`; `extension add` of it is
`invalid_value` (UC-7). The project extensions are every store directory that the manifest does not record as a
catalog extension and that is not a catalog extension's leftover, plus every manifest entry with `source: path`: a
store directory without `extension.json`, and an installed `path` entry whose directory is gone, both fail with
`extension.json is missing`. A leftover is a directory named like a catalog extension and without `extension.json`
(what a kept or untracked removal leaves): it is ignored, has no `extension list` row, and `extension add <n>`
installs the catalog extension over it, its files classified as usual (4.5: identical is `unchanged (adopted)`,
different a conflict `exists and is not managed by solomonar`, resolved with `--keep`/`--overwrite`). A store
directory named like a catalog extension and holding `extension.json`, while the manifest does not record it as
that catalog extension, lists as `invalid: name "<n>" is also a catalog extension` whatever its other failures, and
`extension add <n>` is `name_collision`. A store directory named like a core skill or role (4.1) lists as
`invalid: name "<n>" is a core skill` (or `core role`), winning over every other failure, and `extension add <n>`
is `invalid_value`.

Selection checks run on the sorted unique names of the resulting installed set, in this order, each over the sorted
names; the first failure wins. `init` reports them in the flag form:

| Code | Exit | check | found | fix |
|---|---|---|---|---|
| `unknown_extension` | 2 | `--ext names a catalog extension` | `"<v>" is not in the catalog; known: <sorted names>` | `pass --ext with a known name, or none` |
| `extension_requires` | 3 | `every required extension is selected` | `<name> requires <missing, comma-joined>` | `add --ext <missing, comma-joined>` |
| `extension_conflict` | 3 | `no selected extensions conflict` | `<a> conflicts with <b>` | `drop --ext <a> or --ext <b>` |

With an empty catalog the `unknown_extension` found ends `known: none`. `extension_requires` joins the missing
names with `, ` in found and with `,` in fix, so the fix is a valid flag value (`templ requires go, sql`, `add --ext
go,sql`). A conflict declared on one side only is still found: the check walks each selected name, then its
sorted `conflicts`, and reports the first pair from the side that declares it (`<a>` declares `<b>`). The `extension`
commands use the same checks with their own texts (UC-6, UC-7).

**Installing.** A catalog extension installs every file of its catalog directory, `extension.json` included, to
`docs/solomonar/extensions/<name>/`, keeping relative paths and bytes; every installed file has owner `ext:<name>`
(4.5). Files are planned by extension name, then path. A project extension installs no file: its activation is the
manifest entry, the adapter stubs, the `AGENTS.md` line, and the decision. The decision appears in the `AGENTS.md`
line and the manifest, never inside an extension's files (CLI-D30). Installing an extension that is already
installed refreshes its stubs and `AGENTS.md` line; an identical result is a no-op, never a second decision.

**Loading.** Runtimes trigger extension skills by their `description`, but the process does not rely on that: the
orchestrator names the installed extensions whose `applies_when` or `paths` match the change in every packet, the
implementer loads them before writing code, and `spec-reviewer` checks their `must` rules. Triage lists installed extension
agents beside the core specialists, required when their `applies_when` matches (rules section 6). A user-invoked skill
is loaded only when the user starts it.

**Rule IDs and force** (CLI-D12). Extension rules have stable IDs `<name>/C-n`, never reused across versions. `must`
rules are approved global rules (rules section 1 authority 2) once the decision that installed the extension is approved; a
violation is a blocking finding (rules section 6). `should` rules are suggestions. Findings cite them as they cite
`<capability>/R-n`.

**What a stack extension approves.** The decision that installs it approves the technology or practice and its
rules. It approves no version (the manifests hold versions) and no further dependency: each one still follows
rules section 3.

**Decision recording** (CLI-D11). Every change to the installed set of a stack group is tied to an approved decision
entry:

- With `--decision <ID>`, the CLI checks that `docs/decisions.md` contains that entry with status `approved`
  (`decision_not_found`, `decision_not_approved`, exit 3), and records the ID with the extensions. This is the path
  for an orchestrator applying the project decision it recorded, such as `P/D-3`, or an earlier task decision such
  as `W-002/D-1`; the CLI edits no entry, and the named decision records any supersession itself (rules section 3).
- Without it, the user's command is the decision: the CLI appends a project-scoped entry `P/D-n` per affected group
  with status `approved`, in table order, `<n>` counting on from the highest `<n>` of a line starting exactly
  `## P/D-`, digits, `:` in `docs/decisions.md` and of a `P/D-<n>` the manifest records (an extension `decision` or a
  decision file's `entries`), 0 when there is none, so an ID is never reused; the numbering reads the file only when some group
  needs a new decision. When the group already has a recorded decision (for an emptied group, its `none` entry,
  4.5), the new entry lists the full new selection
  and sets the old one to `superseded`, naming its replacement (rules section 3); a recorded task decision such as
  `W-002/D-1` is superseded the same way. When two groups recording the same old ID both change in one call, the old
  entry names only the later group's new ID.
- In both cases the CLI appends one line per group to the group's document (table above), `- <group>: <a>, <b>
  (<ID>)` in table order (`- <group>: none (<ID>)` for an emptied group), creating the document with a title and an
  index line in `docs/README.md` when absent. It never edits other entries or lines.
- The decision covers the complete resulting installed selection in its group. In the same plan, rebind
  **every remaining installed extension in that group** to the new approved decision, including ones not named
  in this add/remove call: their manifest `decision` and their `AGENTS.md` line change; their files do not.
  Superseding an old decision must not leave any installed extension referring to it.
  A no-op does not supersede or append a decision.

Entry written by the CLI, in the fixed format from rules section 3 that it also reads for `--decision`:

```markdown
## P/D-1: Programming languages

- Status: approved
- Selected option: go
- Rejected options: none
- Scope: project; language extensions `go`
- Rationale: selected by the user with `solomonar init`
- Date: 2026-10-20
- References: `docs/solomonar/extensions/go/SKILL.md`
```

The title is the group's entry title (table above). The CLI writes Rejected options `none`, Scope ``project; <group>
extensions `<a>`, `<b>` `` (`project; <group> extensions none` for an emptied group), Date `YYYY-MM-DD`, and
References the installed main files (each extension's `SKILL.md` or `<name>.md` in the store), sorted by name;
selected names are sorted too, and an emptied group selects `none`. Rationale is ``selected by the user with
`solomonar init` `` (or `` `solomonar extension add` ``, `` `solomonar extension remove` ``), or the `--rationale`
text. A superseding entry's References start with `supersedes <old ID>; ` (just `supersedes <old ID>` for an
emptied group, which has no main files). The superseded entry is edited on one line only: its `- Status: ` line
becomes `- Status: superseded by <new ID>` (the status the CLI reads is the first word); when `docs/decisions.md` has
no `## <old ID>:` heading, or the entry has no Status line, only the Status edit is skipped: the new entry keeps its
`supersedes <old ID>` reference. With `--decision`, nothing in `docs/decisions.md` is edited. The entry status the
CLI reads for `--decision` is the first word after `- Status: ` in the `## <ID>:` entry, `none` when the entry has
no Status line before the next `## ` heading.

An emptied group's entry, as `extension remove go` writes it after `P/D-1`:

```markdown
## P/D-2: Programming languages

- Status: approved
- Selected option: none
- Rejected options: none
- Scope: project; language extensions none
- Rationale: selected by the user with `solomonar extension remove`
- Date: 2026-10-24
- References: supersedes P/D-1
```

Its stack line is `- language: none (P/D-2)`, its plan note `P/D-2 language: none · P/D-1 superseded`, and its JSON
decision `{"id": "P/D-2", "group": "language", "selected": []}`.

A created document starts with its title and a blank line (`# Decisions`, `# Stack`, `# Architecture`) and is
`create`. An existing regular file is `append`, and the full new content is written: a missing final newline is added
first; in `docs/decisions.md` one more blank line follows unless the file already ends with a blank line, so each
entry is preceded by one blank line; stack and architecture lines follow directly. A `docs/decisions.md` whose plan
also edits a superseded entry's Status line is `update`. An empty existing file gets no separator. A document with nothing to add gets no
change unless the manifest records it (4.5). A present document that is not a regular file is an unexpected error,
`read <path>: not a regular file`.

The index lines for the documents a plan creates, in the order decisions, stack, architecture, are
``- `docs/decisions.md`: decision entries and their status``, ``- `docs/stack.md`: technologies and practices, with
the decisions that adopted them``, and ``- `docs/architecture.md`: structure and contracts, with the decisions that
adopted them``. A created `docs/README.md` lists them after its "Documents" items. A present one is `append` when it
has a line `## Documents`; the section runs to the next line starting `## ` or the end of the file. With items, the
lines go right after the last `- ` line, with no blank lines added. Without items, they go after the blank line that
follows the heading (inserting one first when there is none, as at the end of the file), and a blank line follows
them when anything follows in the file. Without the section it stays `skip` with reason
`exists; add under "## Documents": <line>; <line>`. A non-regular `docs/README.md` is the unexpected error `read docs/README.md: not a
regular file` when index lines are to be added, and `skip` otherwise.

**Commands in extensions.** Extensions list typical checks (`go test ./...`, `go vet ./...`, `govulncheck ./...`) as
suggestions only. No extension writes commands into `docs/README.md`; they stay "not established" until the
implementer records the real ones (rules section 11).

Installing an extension installs its instructions and adapters, not application packages, tooling binaries, or
manifest/lockfile dependencies. Print this distinction in stack-extension plans; approved package/version changes
are applied by the implementer within their recorded scope.

### 4.4 Extension catalog

Decided in CLI-D13 and CLI-D10. Each extension's content is written and reviewed as framework content before release.

| Extension | Kind | Group | Requires | Applies when | `detect` | `default` | `invocation` | Main rules |
|---|---|---|---|---|---|---|---|---|
| `go` | skill | `language` | - | A change touches Go code (*.go, go.mod, go.sum) | `**/go.mod` | `false` | `model` | Common version/error rules in `SKILL.md`; design, I/O, concurrency, and testing rules in `rules/go/`, loaded by impact on code and contracts; reviewers account for applicable `must` rules and skipped topic triggers; project checks precede portable fallbacks |
| `extension-authoring` | skill | `workflow` | - | A request creates, changes, or validates a Solomonar extension skill, its rules, or a specialist reviewer | - | `true` | `user` | Research, authoring, and validation stages loaded separately; catalog contributions and project extensions written to the store in catalog format; evidence in one record; activation through `solomonar extension add <name>` |

So `init --yes` in a repository with a `go.mod` selects `go` and `extension-authoring`, and without one only
`extension-authoring`. No `review` extension is in the catalog; `review` and every group without a catalog extension
accept project extensions.

Illustrative test catalog: some examples (UC-6) use `htmx` and `templ` (`frontend` skills, `templ` requiring `go`).
They are not shipped; tests build them as a `fstest.MapFS` catalog to exercise multi-extension groups and
supersession.

### 4.5 File ownership

| Class | Files | `init` | `upgrade` | `uninstall` (UC-27) |
|---|---|---|---|---|
| `core` | `docs/solomonar/**` except `docs/solomonar/extensions/` | create | update if unmodified, else conflict; remove when no longer offered | remove if unmodified, else conflict |
| `ext:<name>` | every file of an installed catalog extension's store directory, `docs/solomonar/extensions/<name>/` | create, verbatim | as `core` | as `core` |
| project extension (no class) | `docs/solomonar/extensions/<name>/` of a project extension | never planned, written, or recorded | never | never; the source stays |
| `decision` entries | `P/D-n` entries in `docs/decisions.md`; one line per group in `docs/stack.md` or `docs/architecture.md` | `append`; `create` the file when absent (4.3); a re-run plans recorded files `unchanged` | never | never |
| `core+notes` | `AGENTS.md` | create, or adopt an existing file (UC-4) | the part above `## Project notes` as `core`, its rendered lists (4.6) included; the section from that heading on is project-owned | remove the verified managed part; keep the notes part, or remove the file when the notes part holds only the template heading and comment |
| `runtime:<name>` | adapter stubs, `CLAUDE.md` when created | create | regenerate | as `core` |
| `runtime:<name>` entry (deferred, CLI-D36) | the future guard hook entry inside `.claude/settings.json` or `.codex/hooks.json`; no current command writes one | future: merge the entry; never touch other keys | future: merge | conflict: entry removal is not supported yet |
| `seed` | `docs/README.md` | create when absent, else `skip`; when the CLI creates a document, a present file with `## Documents` gets the index lines (`append`), else `skip` with the reason (4.3) | never | remove only the exact generated framework index line (CLI-D35) |
| `manifest` | `.solomonar/manifest.json` | create | update | remove, last |

`extension add|remove` follow the `init` column for the paths they plan: `add` creates a catalog extension's files,
`remove` removes them, and both regenerate stubs and `AGENTS.md`. `runtime add|remove` (UC-5) do the same for one
runtime's `runtime:<name>` files. `uninstall` is not a desired-state plan: it
plans removals from the manifest's records and the files on disk alone (UC-27).

"Unmodified" means the file's SHA-256 equals the manifest's `sha256`: the hash of the last distributed or
generated base offered by the CLI, **not the hash of a kept project edit**. For `core+notes`, hash only the managed
part. A kept differing file stays different from that base.
`init --keep` records the offered embedded base while leaving local bytes untouched. At upgrade, even when
`--keep` resolves a conflict, advance the base to the offered upstream version, never to the kept local content.
The next upgrade can therefore preserve the edit or report a conflict instead of overwriting it as unmodified.

**Desired-state planning** (CLI-D27). `init`, `upgrade`, `extension add|remove`, and `runtime
add|remove` (UC-5, CLI-D36) share one planner; `uninstall` does not use it (UC-27, AD-16). From the choices (runtimes, models, installed extensions, each stack group's decision) it computes every
wanted path: core files, catalog extension files, stubs, `AGENTS.md`, decision documents, the manifest. Each wanted
path is classified from `disk` (the current file), `base` (the manifest `sha256`), and `offer` (the content the CLI
would write); the recorded base is always the hash of `offer`. A path is managed when the manifest records a `sha256`
for it; every owner except `seed` is hashed. The first matching row applies, so a present `seed` path is `skip` even
when it is not a regular file, and a managed file whose `disk` = `offer` is `unchanged`:

| Path | Condition | Action |
|---|---|---|
| `seed` owner | absent / present | `create` / `skip` (project-owned; no hash recorded; index lines, 4.3) |
| create-only (`CLAUDE.md`), unmanaged | absent / present, regular or not | `create` / `skip`, never a conflict, not recorded; reason `exists; add this line …` unless a line is exactly `@AGENTS.md` (4.2) |
| any other | absent | `create` |
| any other | not a regular file (directory, symlink) | `conflict`, `exists and is not a regular file` |
| managed (`base` recorded) | `disk` = `offer` | `unchanged` |
| managed | `disk` = `base` | `update` |
| managed | `offer` = `base` | `keep` (the project's edit stays; F-02) |
| managed | otherwise | `conflict`, `modified since installation (manifest sha256 differs)` |
| unmanaged | `disk` = `offer` | `unchanged (adopted)` (CLI-D17) |
| unmanaged | otherwise | `conflict`, `exists and is not managed by solomonar` |

A managed path that no wanted path names, such as a removed extension's file or a v0.1.7 path under
`docs/solomonar/skills/<extension>/`, is classified after the wanted paths, in sorted path order, from `disk` and
`base` alone. `decision`, `seed`, and `manifest` files and runtime `entry` files are never in this set:

| Path | Condition | Action |
|---|---|---|
| managed, no longer wanted | absent | no plan line; dropped from the manifest |
| managed, no longer wanted | `disk` = `base` | `remove`; dropped from the manifest |
| managed, no longer wanted | otherwise (edited, or not a regular file) | `conflict`, `modified since installation (manifest sha256 differs)` or `exists and is not a regular file` |

A removal deletes the file and then every parent directory it leaves empty, never a top-level directory (`docs`,
`.claude`, `.codex`, `.github`, `.agents`), `docs/solomonar`, or `docs/solomonar/extensions`: removing `go` deletes
`docs/solomonar/extensions/go/` and `.claude/skills/go/`, and `.claude/skills/` too when it is left empty. A rollback
recreates the removed directories with their modes. Only `uninstall` also removes `docs/solomonar/extensions`,
`docs/solomonar`, and `.solomonar` when they are left empty (UC-27). Nothing under a project extension's directory is
ever wanted or managed, so it is never in either table.

A path that cannot be inspected for a reason other than absence (for example, a parent that is a file) is an
unexpected error, not `absent`. Only conflicts carry a reason, `adopted` aside.

`--keep <path>` turns a conflict into a plain `keep`, `--overwrite <path>` (`--take-new` for `upgrade`) into a plain
`update`; both record the offered base. On a removal conflict, `--keep` is `keep` with the reason `untracked; left in
place` and drops the path from the manifest, and `--overwrite` (`--take-new`) is `remove`. A path in both flags is
checked first; then each `--keep` path, then each `--overwrite` path, in the given order, must be a conflict of this
plan (a path the plan does not list is also not a conflict). The first offending path is `invalid_value` (exit 2):
check `--keep and --overwrite resolve conflicts`, found `<path> is in both --keep and --overwrite` or `<path> is not a
conflict`, fix `pass --keep or --overwrite once per conflicting path` (with `--take-new` in place of `--overwrite`
for `upgrade`). The plan lists the wanted paths in the given order, then the removals. The manifest entries returned
for unresolved conflicts already carry the offered base, and an unresolved removal conflict keeps its recorded entry;
nothing is written while the plan has a conflict. `--overwrite` (`--take-new`) on a removal conflict whose path is
not a regular file is `invalid_value` (exit 2): check `--keep and --overwrite resolve conflicts` (`--take-new` for
`upgrade`), found `<path> is not a regular file`, fix `remove it by hand or pass --keep <path>`; it is checked while
the removals are planned, after the in-both check and before the `is not a conflict` check.

`AGENTS.md` is split at the first line exactly equal to `## Project notes`: the managed part is everything before
it, the notes part is that line onward (none without such a line). Only the managed part is hashed and classified by
the rows above; a non-regular `AGENTS.md` keeps the `exists and is not a regular file` conflict. A written managed
file is the offered managed part followed by the file's own notes part (the template's when it has none), so
`--overwrite AGENTS.md` on a managed conflict replaces only the managed part and keeps the notes. An unmanaged
`AGENTS.md` is compared on its managed part too: when it equals the offered managed part, the file is
`unchanged (adopted)` whatever its notes; otherwise it is adopted (UC-4): `update`, reason `adopted; existing content
moved under "## Project notes"`, written as the offered managed part, `## Project notes`, a blank line, and the old
content unchanged. An adoption counts as a conflict for the flags: `--overwrite AGENTS.md` writes the full template
and discards the old content, and `--keep AGENTS.md` is `invalid_value` (exit 2): check `--keep and --overwrite
resolve conflicts`, found `AGENTS.md cannot be kept`, fix `adopt it (default) or pass --overwrite AGENTS.md`
(for `upgrade`, `--take-new` replaces `--overwrite` in the check and fix: `adopt it (default) or pass --take-new
AGENTS.md`).

The manifest is planned last, as one change for `.solomonar/manifest.json` with owner `manifest`, and is never
classified by the tables above (it has no hash): `create` when absent, `unchanged` when its bytes equal the canonical
encoding of the new manifest, `update` otherwise. When it exists but cannot be read as a file (for example a
directory), the command stops with an unexpected error (`read manifest: …`, exit 1) instead of planning a conflict.
The new manifest records `files` for every wanted path, each `decision` file as `{"owner": "decision", "entries":
[...]}` listing the IDs the CLI wrote to it in this and earlier runs (below), and `.solomonar/manifest.json`
`{"owner": "manifest"}`, with empty `runtimes` and `extensions` for a core-only `init`; a removed or untracked path
has no entry. It is built even when the plan has conflicts, but nothing is written then.

Decision files are planned by 4.3, before the manifest, not by the tables above. A plan that records no new decision
(an `init` re-run, UC-3; every `upgrade`) carries the existing manifest's `decision` files over and plans them
`unchanged` without changing them, and a group whose decision the manifest records keeps that ID and adds nothing: the
ID of the group's first installed extension, by name, whose manifest entry has a `decision`, of any form (a task
decision such as `W-002/D-1` too). An emptied group has no installed extension; its recorded decision is the ID of
its last line in its group document when that line is `- <group>: none (<ID>)` and the manifest records the
document with owner `decision`, so the documents the manifest records are read to find it. `extension add|remove` record a new decision only for a group whose selection
differs from the recorded one. A document's `entries` are the carried entries followed by the new IDs.

### 4.6 Rendered `AGENTS.md`

The managed part of `AGENTS.md` is rendered from the choices: the framework version, the core "Skills" list, and an
"Extensions" section grouped like `extension list`. For a project with `go` (`P/D-1`), the project skill `money`
(`W-002/D-1`), and `extension-authoring`:

```markdown
## Extensions

Installed with `solomonar`. Load every installed extension skill whose "applies when" matches your change, before
writing or reviewing code, and cite its rules as `<name>/C-n`. The orchestrator names the matching extensions in
each packet.

- Skills
  - `go` (language, P/D-1): A change touches Go code (*.go, go.mod, go.sum); paths: `**/*.go`, `**/go.mod`, `**/go.sum`
  - `money` (backend, W-002/D-1): A change handles monetary amounts; paths: `internal/money/**`
- Agents: none. An installed agent is a specialist reviewer, required by triage when its "applies when" matches.
- User-invoked skills: load one only when the user starts it, never because it applies.
  - `extension-authoring` (workflow): A request creates, changes, or validates a Solomonar extension skill, its rules, or a specialist reviewer
```

An extension line is `` - `<name>` (<group>, <decision>): <applies_when> `` for a stack group and
`` - `<name>` (<group>): <applies_when> `` for `workflow` and `review` (CLI-D30), with `applies_when` verbatim,
followed by `` ; paths: `<glob>`, `<glob>` `` when `paths` is not empty; `<decision>` is the extension's manifest
`decision`. Catalog and project extensions are listed alike. Installed model-invoked skills render a line
`- Skills`, then one nested line each; installed agents render a line `- Agents: an installed agent is a specialist
reviewer, required by triage when its "applies when" matches.`, then one nested line each. Each list is ordered by
group (4.3 table order), then name. An empty list renders `- Skills: none.` or `- Agents: none. An installed agent is
a specialist reviewer, required by triage when its "applies when" matches.` Installed skills with `invocation: user`
render, after the agents, the line `- User-invoked skills: load one only when the user starts it, never because it
applies.` and one nested line each, in the same order and form (CLI-D29); with none, the line is omitted.

### 4.7 Manifest

Adding the `copilot` runtime keeps `schema_version` at 1: no field shape changes. Release it with the normal
framework version increase. Older binaries reject a newer framework before strict decoding; an older binary
given an otherwise readable manifest with `copilot` rejects the unknown runtime. Do not lower the version or
hand-edit runtime choices to make an old binary accept it. Existing Claude/Codex manifests need no migration.

The store layout also keeps `schema_version` at 1 (CLI-D27): `state` is always `installed`, an installed project
extension adds `source: path` and `path`, and a manifest written by 0.1.7 or earlier already decodes under these
rules. Its framework version moves only through `upgrade` (UC-9), which also moves the files to the store.

`.solomonar/manifest.json` is committed on main, written only by the CLI, and records the choices and every managed
file. `extensions` maps each installed extension's name to an entry holding exactly `state` (`installed`), `kind`,
`group`, `source` (`catalog`, or `path` for a project extension), `path` (a `path` source only: its store directory,
`docs/solomonar/extensions/<name>`), and `decision` (stack groups only: the decision for its group's current
selection, `P/D-<n>` or `<task-ID>/D-<n>`). An available extension has no entry; a project extension's metadata
lives only in its `extension.json`. `files` keys are repo-relative paths, and a project extension has none:

```json
{
  "schema_version": 1,
  "framework_version": "X.Y.Z",
  "project": {"name": "ledger", "main_branch": "main", "remote": "origin"},
  "runtimes": {
    "claude": {"adversarial_model": "opus"},
    "codex": {}
  },
  "extensions": {
    "extension-authoring": {"state": "installed", "kind": "skill", "group": "workflow", "source": "catalog"},
    "go": {"state": "installed", "kind": "skill", "group": "language", "source": "catalog", "decision": "P/D-1"},
    "money": {"state": "installed", "kind": "skill", "group": "backend", "source": "path", "path": "docs/solomonar/extensions/money", "decision": "W-002/D-1"}
  },
  "files": {
    "AGENTS.md": {"owner": "core+notes", "sha256": "<64 hex, managed part only>"},
    "docs/solomonar/rules.md": {"owner": "core", "sha256": "<64 hex>"},
    "docs/solomonar/extensions/go/SKILL.md": {"owner": "ext:go", "sha256": "<64 hex>"},
    "docs/solomonar/extensions/go/extension.json": {"owner": "ext:go", "sha256": "<64 hex>"},
    "docs/decisions.md": {"owner": "decision", "entries": ["P/D-1"]},
    ".claude/skills/money/SKILL.md": {"owner": "runtime:claude", "sha256": "<64 hex>"},
    ".claude/settings.json": {"owner": "runtime:claude", "entry": "hooks.PreToolUse/solomonar-guard"},
    "docs/README.md": {"owner": "seed"},
    ".solomonar/manifest.json": {"owner": "manifest"}
  }
}
```

`files` is abridged: a real manifest lists every managed path, such as the other `docs/solomonar/extensions/go/`
files and every stub. The `.claude/settings.json` record shows the reserved `entry` shape of the deferred guard hook
(4.5, CLI-D36); no current command writes one, and validation still accepts it.

After `init` records `runtimes`, only `runtime add|remove` (UC-5) change it, setting or dropping one key and its
`adversarial_model`; an `init` re-run rejects a change (UC-3), and `upgrade` and `extension add|remove` carry it over.

Validation, in order, reporting the first failure:

- Versions first, before strict decoding: `schema_version` must be present and a positive integer; above 1 it is
  `cli_outdated` (3.6), found `manifest schema <n>, this CLI supports 1`. Only then is `framework_version` checked:
  it must be `major.minor.patch`, and newer than the CLI's embedded framework it is `cli_outdated`, found
  `manifest framework <v>, this CLI ships <v>`. Versions compare numerically per part (`0.10.0` is newer than
  `0.2.0`). So a newer schema wins over a malformed `framework_version` or unknown members; a malformed version is
  `manifest_invalid`. An older `framework_version` is valid; `extension add|remove` reject it afterwards with
  `framework_outdated` (3.6).
- Strict decoding: unknown members and duplicate keys are rejected.
- `project.name`, `main_branch`, and `remote` are non-empty; `runtimes`, `extensions`, and `files` are present
  (`{}` is allowed, `null` or missing is rejected).
- `runtimes` keys are `claude`, `codex`, or `copilot`, each value `{}` or `{"adversarial_model": ...}`.
  Copilot adapters use `owner: "runtime:copilot"` and `sha256`; they have no hook `entry`.
- Extension names match `[a-z][a-z0-9-]*`; `state` is `installed` (`extensions.<n>.state "<v>" is not installed`),
  `kind` `skill` or `agent`, `group` one of the nine 4.3 groups, `source` `catalog` or `path`; only a `path` source
  has `path`, and it is `docs/solomonar/extensions/<name>` (`extensions.<n>.path "<v>" is not
  docs/solomonar/extensions/<n>`). A stack-group extension has `decision` `P/D-<n>` or `<task-ID>/D-<n>`, with a task
  ID `W-` and three or more digits (`extensions.<n>.decision "<v>" is not P/D-<n> or <task-ID>/D-<n>`); a `workflow` or `review`
  extension has none.
- `files` keys are clean repo-relative paths. `owner` is `core`, `ext:<name>`, `decision`, `core+notes`,
  `runtime:<name>`, `seed`, or `manifest`, with `<name>` a known extension or runtime. `core`, `ext:`, `core+notes`,
  and `runtime:` files have `sha256` (64 lowercase hex), except a runtime entry file, which has `entry` instead;
  `decision` files have `entries`, each `P/D-<n>` or `<task-ID>/D-<n>` (`docs/stack.md` records `W-002/D-1` after
  `--decision W-002/D-1`; `files.<path>.entries "<v>" is not P/D-<n> or <task-ID>/D-<n>`); `seed` and `manifest`
  files have neither.

Runtimes, extensions, and files are each checked in sorted key order. "Known" means a key in this manifest. Not
validated: whether an `ext:` owner names a `catalog` source, whether a project extension's directory still exists or
validates (`extension list` reports it, 4.3), duplicate `entries`, and an empty `adversarial_model`.

Any other failure is `manifest_invalid` (exit 1) with check `manifest schema`, a found text naming the field (e.g.
`project.remote is missing`, `extensions.go.decision is missing`, `extensions.go.group "web" is not one of
language, ...`, `files.<path>.owner "<v>" is not an ownership class`, `files.<path>.sha256 is missing`), and the fix
"inspect .solomonar/manifest.json; never edit it by hand". The CLI writes the manifest canonically: struct field
order, sorted map keys, two-space indent, trailing newline.

## 5. Use cases: project setup

### UC-1 Initialize a new project interactively

Actor: user. Preconditions: a Git repository with at least one commit on `main`; the CLI is installed; no
`.solomonar/manifest.json`.

```
$ cd /work/ledger
$ solomonar init
solomonar X.Y.Z · /work/ledger (git, branch main)

Project name [ledger]:
Current objective, one line (optional): Track shared household expenses

Runtimes that will run this project's sessions:
    [x] claude   Claude Code: CLAUDE.md import, .claude/skills, .claude/agents
    [x] codex    Codex: AGENTS.md (native), .agents/skills, .codex/agents
    [ ] copilot  GitHub Copilot CLI: AGENTS.md (native), .github/skills, .github/agents

Model for the adversarial reviewer in Claude Code, different from your main session model
(blank = runtime default): opus
Model for the adversarial reviewer in Codex, different from your main session model
(blank = runtime default):

Extensions, by decision type. A choice in a stack group is recorded as your approved decision in
docs/decisions.md. Enter with nothing selected skips a group; add more later with
`solomonar extension add`.

Language:
    [x] go  Go language rules

Workflow:
    [x] extension-authoring  Evidence-based extension authoring

Plan
  framework X.Y.Z
    create     docs/solomonar/agents/ (6 roles)
    create     docs/solomonar/registry.md
    create     docs/solomonar/report.md
    create     docs/solomonar/reviewer.md
    create     docs/solomonar/rules.md
    create     docs/solomonar/skills/prototype/ (SKILL.md, references/cli.md, references/module.md, references/service.md, references/web.md, template.md)
    create     docs/solomonar/skills/solomonar/ (SKILL.md)
    create     docs/solomonar/skills/spec/ (SKILL.md, template.md)
    create     docs/solomonar/skills/work-spec/ (SKILL.md, template.md)
    create     AGENTS.md
    create     docs/README.md
  extensions
    create     docs/solomonar/extensions/extension-authoring/ (6 files)
    create     docs/solomonar/extensions/go/ (6 files)
  decisions (approved, selected by you)
    create     docs/decisions.md  P/D-1 language: go
    create     docs/stack.md      language: go (P/D-1)
  runtime claude
    create     CLAUDE.md
    create     .claude/skills/ (6 skill stubs: 4 core, 2 extensions)
    create     .claude/agents/ (5 role stubs; adversarial-reviewer model opus)
  runtime codex
    create     .agents/skills/ (6 skill stubs: 4 core, 2 extensions; extension-authoring/agents/openai.yaml; solomonar/agents/openai.yaml)
    create     .codex/agents/ (5 role stubs)
  registry
    create     refs/heads/workflow-state with an empty registry
  manifest
    create     .solomonar/manifest.json
Proceed? [Y/n] y

Initialized solomonar X.Y.Z in /work/ledger
  63 files created
  registry  refs/heads/workflow-state @ 3a5c0d7e9f1b2a4c6d8e0f1a3b5c7d9e1f2a4b6c
  pushed    origin/workflow-state

Next:
  1. Review and commit the framework files on main:
       git add AGENTS.md CLAUDE.md .claude .agents .codex .solomonar docs
       git commit -m "chore: initialize solomonar X.Y.Z"
  2. Start a top-level session in /work/ledger with your selected outcome:
       Claude Code: /solomonar Explore the next useful flow.
       Codex:       $solomonar Explore the next useful flow.
       Portable:    Use the solomonar skill to explore the next useful flow.
```

Prompts: `init` prompts when stdin and stdout are terminals and neither `--yes` nor `--json` is set (3.7). It first
prints the header `solomonar <cli version> · <dir> (git, branch <b>)` and a blank line: `<cli
version>` as `solomonar version` prints it, `<dir>` the worktree root (outside a repository, the absolute target
directory as given, symlinks not resolved, the path `not_a_repository` names), `<b>` the current branch (also
unborn); `(git, detached HEAD)` when detached, `(not a Git repository)` outside one. The header is printed on every
prompting run, including a re-run that ends `Already initialized …`, but not when opening the repository fails with
an error other than `not_a_repository`. Choices are prompted only without a manifest (a re-run takes them from it and asks none), each only when its flag was not
given, in this order and with the texts above; each prompt's default is the one the flag would take (UC-2). One blank
line separates each asked block: the name and objective, the runtimes, the models, the extensions paragraph, and
each group.

- `Project name [<default>]: `, then `Current objective, one line (optional): `;
- `Runtimes that will run this project's sessions:` and a multi-select over `claude`, `codex`, and `copilot`,
  in that order. Claude and Codex use the descriptions above; Copilot uses
  `GitHub Copilot CLI: AGENTS.md (native), .github/skills, .github/agents`. Runtimes found on `PATH` are marked;
- without `--adversarial-model`, per selected runtime in the order claude, codex, copilot: `Model for the adversarial
  reviewer in <Claude Code|Codex|GitHub Copilot CLI>, different from your main session model`, then `(blank = runtime default): `;
  blank records no model; all selected runtime model prompts form one block, with no blank line between them;
- when the catalog is not empty and `--ext` was not given: the extensions paragraph, then per group that has catalog
  extensions, in 4.3 order, `<Group>:` (first letter capitalized) and a multi-select over its extensions sorted by
  name, each described by its title plus ` (requires <a>, <b>)`, in catalog order, when it requires others; the
  extensions of the UC-2 default (`default` or detected, 4.3) are marked. An empty selection means no extension, not
  the default. A selection that fails the 4.3 checks prints `  <found>; <fix>` (the flag form, such as `add --ext go`) right after
  the last list, then asks every group again, each with its blank line and the previous selection marked, without
  repeating the extensions paragraph.

When the plan writes, has no conflict, and `--dry-run` is not set, `init` prints `Plan`, the grouped plan below, and
`Proceed? [Y/n] `, also on a re-run with a manifest; yes applies as with `--yes`, and no prints
`Cancelled; nothing was written.` and exits 0, keeping warnings such as `jujutsu_detected`. One blank line follows
the answer, before `Initialized …` or the cancel line. `Plan` is preceded by a blank line only when
a choice was asked, also before an interactive `--dry-run` plan and the `file_conflict` plan text. `--dry-run` still
prompts for choices, never for the confirmation. `--json` without `--yes` never prompts, even on a
terminal, and stays `interactive_required`.

The grouped plan is shown only before `Proceed?`; `--dry-run` (interactive or not), the `file_conflict` plan, and
JSON list every file (3.3). Its groups, in this order with empty groups omitted, are `git` (`init       -b
<main-branch>`), `framework <fw>` (owners `core`, `core+notes`, `seed`), `extensions` (`ext`),
`decisions (approved, selected by you)`, `runtime claude`, `runtime codex`, `runtime copilot`, `registry` (`create     refs/heads/workflow-state
with an empty registry` or `unchanged  refs/heads/workflow-state`), and `manifest`. A header is indented two spaces and its lines four; a line is `<action padded to 9>  <path>`, and a note
follows two spaces after the path padded to the widest noted path in its group. Within a group the plan order is
kept, and a collapsed line takes its set's first position. A set collapses into one line only when all its lines
share one action and none has a reason; otherwise each file gets its own line. The sets are
`docs/solomonar/agents/ (<n> roles)`; `docs/solomonar/skills/<s>/ (<files, comma-joined, sorted>)` per core skill;
`docs/solomonar/extensions/<name>/ (<n> files)` per extension store directory, and likewise any other extension
directory whose files share one action (a v0.1.7 `docs/solomonar/skills/<name>/` that `upgrade` removes, UC-9);
`.claude/skills/`, `.agents/skills/`, and `.github/skills/` as `(<n> skill stubs: <c> core, <e> extensions)`, `: <c> core` always shown,
the codex one adding `; <skill>/agents/openai.yaml` per user-invoked skill, sorted; and `.claude/agents/`, `.codex/agents/`, and `.github/agents/` as `(<n> role
stubs[: <c> core, <e> extensions][; adversarial-reviewer model <m>])`, the split only when an extension role exists
(the stubs of an extension that `extension remove` deactivates count as extension stubs) and the model only when the
set includes the `adversarial-reviewer` stub and the runtime has a model. A count of 1 is singular (`1 role`, `1 role stub`, `1 extension`). Notes join
with `; `, the reason first;
`docs/decisions.md` is noted `<ID> <group>: <names> · …`, each followed by ` · <old ID> superseded` when the entry
supersedes one, and each group document `<group>: <names> (<ID>) · …` for the entries this run records (`none` for
an emptied group); an `unchanged` document gets no note.
Copilot role identifiers strip the full `.agent.md` suffix when counting core versus extension roles.

Postconditions: the files exist uncommitted on main; each chosen group has an approved `P/D-n` entry; `workflow-state` exists as a root commit
`state: initialize registry` containing `{"schema_version": 1, "tasks": {}}`; the manifest records every choice.
`init` never commits on main (CLI-D4); `doctor` reports uncommitted framework files until they are committed.

`--json` data:

```json
{
  "project": "/work/ledger",
  "framework_version": "X.Y.Z",
  "choices": {"runtimes": ["claude", "codex"], "extensions": {"language": ["go"], "workflow": ["extension-authoring"]}, "adversarial_model": {"claude": "opus"}},
  "decisions": [{"id": "P/D-1", "group": "language", "selected": ["go"]}],
  "files": [{"path": "AGENTS.md", "action": "create", "owner": "core+notes"}],
  "registry": {"ref": "refs/heads/workflow-state", "commit": "3a5c0d7e9f1b2a4c6d8e0f1a3b5c7d9e1f2a4b6c", "action": "create", "pushed": true},
  "git_init": false,
  "first_commit_required": false,
  "next": ["git add AGENTS.md CLAUDE.md .claude .agents .codex .solomonar docs", "git commit -m \"chore: initialize solomonar X.Y.Z\""]
}
```

`choices.runtimes` is sorted; `choices.adversarial_model` maps each runtime with a model to it (`{runtime: model}`).
`choices.extensions` maps each group with a selection to its sorted names (`{}` when none). `decisions` holds
`{"id", "group", "selected"}` for the entries this run records (`[]` when none, a re-run included). `reason`
appears in a `files` item only when the change has one.
`git_init` (the plan runs `git init`, UC-4) and `first_commit_required` (empty repository, UC-4) are always present.
`registry.commit` is the tip and `registry.pushed` is `false` when the registry is `unchanged`. With `--dry-run` the
data adds `"dry_run": true`, `registry` is `{"ref", "action"}`, and `next` is absent; a no-op (UC-3) has
`"next": []`.

### UC-2 Initialize non-interactively

Actor: user or script. Synopsis, flags, defaults, and examples: [`solomonar init`](../../site/src/docs/cli/init.md).
An omitted `--objective` writes "Not set" in `docs/README.md`. Each stack group gets one `P/D-n`. `--ext` takes
catalog names only; a project extension is activated after `init` with `extension add` (UC-7).

`--runtime` and `--ext` split their values on commas; `--keep` and `--overwrite` take one path per flag, so a path
may contain commas. A relative `<dir>` is joined to `-C`, an absolute one is used as is; the files are written at the
root of the worktree containing `<dir>`. The help's usage ends `[--git-init] [-y|--yes] [--dry-run] [--json]`, and its
description ends `--git-init runs git init -b <main-branch> when <dir> is not a repository.`

`--runtime` takes `claude`, `codex`, `copilot`, or `none`, repeatable or comma-separated. The usage fragment is
`[--runtime <claude|codex|copilot|none>]...`. Another value is `unknown_runtime` (exit 2, `init rejected`), check
`--runtime names a known runtime`, found `"<v>" is not a runtime; known: claude, codex, copilot`, fix `pass --runtime claude,
codex, copilot, or none`; `none` beside another value is `invalid_value`, check `--runtime names runtimes or none`, found `none
cannot be combined with <v>`, fix `pass --runtime none alone`. Without `--runtime`, a project with a manifest keeps
its runtimes; otherwise the default is each of `claude`, `codex`, `copilot` whose executable is found on `PATH`.
Explicit selection does not require the runtime executable. No Copilot process or authentication check runs.

`--ext` takes extension names or `none`, which requests no extension. `none` beside another value is
`invalid_value`, check `--ext names extensions or none`, found `none cannot be combined with <v>`, fix `pass --ext
none alone`; the names then go through the 4.3 selection checks (`unknown_extension`, `extension_requires`,
`extension_conflict`, `init rejected`). Without `--ext`, a project with a manifest keeps its extensions, an empty
set included: nothing is detected again on a re-run; otherwise the default is every catalog extension with `default: true` or a `detect` glob that matches a listed file (4.3
Detection), else none; `--ext none` excludes them all. The default goes through the same selection checks. The plan
lists core files, extension files, runtime stubs, decision documents, then the manifest.

`--adversarial-model <runtime>=<model>` takes one value per flag (no comma splitting) and is recorded as
`runtimes.<r>.adversarial_model` (4.7); a runtime given twice keeps the last value. A value without `=` or with an
empty side is `invalid_value`, check `--adversarial-model is <runtime>=<model>`, found `"<v>" has no "="`, `"<v>" has
an empty runtime`, or `"<v>" has an empty model`, fix `pass --adversarial-model claude=<model>`. A runtime not
selected, an unknown name included, is `invalid_value`, check `--adversarial-model names a selected runtime`, found
`<r> is not selected`, fix `add --runtime <r> or drop the model` naming that runtime. Models start from the
manifest's models for the selected runtimes, with the requested ones applied on top. This includes
`--adversarial-model copilot=<model>`; model access is not validated during installation.

Warnings go to stderr. The success form is `Initialized solomonar <fw> in <root>`, `  <counts>` (3.3),
`  registry  refs/heads/workflow-state @ <sha>` (`  registry  refs/heads/workflow-state @ <tip> unchanged`, with no
`pushed` line, when an existing registry is reused), `  pushed    <remote>/workflow-state` when pushed, a blank
line, and `Next:`: step 1 is `git add <generated paths> && git commit -m "chore: initialize solomonar <fw>"`, step 2
the top-level session start. The paths are `AGENTS.md`, then `CLAUDE.md` (only when this run creates it), `.claude`
(claude), `.agents` and `.codex` (codex), the Copilot adapter file paths sorted within skills then roles
(copilot), then `.solomonar docs`. Name individual Copilot files rather than staging `.github` or its whole
subdirectories, which may contain unrelated files. Apply the same rule to JSON `next`. Step 2 depends on the runtimes:

- claude only: `  2. Start a top-level Claude Code session in <root>:`, then
  `       /solomonar Explore the next useful flow.`;
- codex only: `  2. Start a top-level Codex session in <root>:`, then
  `       $solomonar Explore the next useful flow.`;
- copilot only: `  2. Start a top-level GitHub Copilot CLI session in <root>:`, then
  `       /solomonar Explore the next useful flow.`;
- two or more: `  2. Start a top-level session in <root> with your selected outcome:`, then one line per
  selected runtime in claude, codex, copilot order: `       Claude Code: /solomonar Explore the next useful flow.`,
  `       Codex:       $solomonar Explore the next useful flow.`, or
  `       GitHub Copilot CLI: /solomonar Explore the next useful flow.`, followed by
  `       Portable:    Use the solomonar skill to explore the next useful flow.`;
- none: `  2. Start a top-level session in <root>:`, then
  `       Use the solomonar skill to explore the next useful flow.`
A failed push is a warning, exit 0 (F-09).

Without a terminal and without `--yes`:

```
$ solomonar init --runtime claude < /dev/null
error: init needs a confirmation (interactive_required)
  fix: pass --yes to accept the plan, or --dry-run to print it
nothing was written
```

Exit 2. A no-op (UC-3) needs no confirmation; every other run does, including one where every file matches but the
registry is absent. `--dry-run` prints `Plan`, every file line, the registry line (`  registry   create
refs/heads/workflow-state with an empty registry`, or `  registry   unchanged refs/heads/workflow-state` with an
existing registry), then, with no blank line, `Would initialize solomonar <fw> in <root>` and `  <counts>`, and
writes nothing; on a terminal it still prompts for the choices (UC-1) but never for the confirmation.

Checks run in this order: usage and values (`--runtime`: `unknown_runtime`, then `none` combined; the
`--adversarial-model` syntax; `--ext` (`none` combined, then the 4.3 selection checks)); the repository
(`not_a_repository`, UC-4) and the Git version; the manifest (`cli_outdated`, `manifest_invalid`); on a terminal
without a manifest, the choice prompts (UC-1); the runtime selection (requested, else the manifest's, else `PATH`) and the
unselected-runtime model check; the recorded choices (`already_initialized`, UC-3), after which
choices not given come from the manifest, else the worktree root's base name, `main`, and `origin`; the registry
(absent: create; valid: reuse; invalid: `registry_exists_invalid`; newer: `cli_outdated`); the plan (`invalid_value`
from `--keep`/`--overwrite`) and its conflicts (`file_conflict`); `--dry-run`; the confirmation; the file apply; the
registry creation on the chosen remote when absent. A failure before the registry commit succeeds (a failed
creation, cancellation) restores every file this run wrote (architecture section 8).

### UC-3 Re-run init on an initialized project

Same choices, identical files: a no-op.

```
$ solomonar init --yes
Already initialized with solomonar X.Y.Z; nothing to do.
  64 files unchanged, 1 skipped · registry refs/heads/workflow-state unchanged
  Change choices with `solomonar runtime|extension add|remove`; move versions with `solomonar upgrade`.
```

Exit 0. The no-op form applies only when the registry exists and the plan creates, updates, appends, or removes nothing;
when every file matches but the registry is absent, `init` takes the normal path (UC-2) and needs the confirmation.
The second line is `  <counts> · registry refs/heads/workflow-state unchanged` (3.3). The CLI fills the
choices not given from the manifest; a manifest `catalog` extension missing from the catalog is an unexpected error
(exit 1) `read extension <name>: not in the catalog`, and a `path` extension is read from its store directory (4.3),
one that fails the loader (its directory gone included) being `invalid_value` (UC-7); a manifest that records no
extensions keeps that empty choice. A given choice that differs from the manifest is rejected, so that
`init` never becomes a second way to change choices:

```
$ solomonar init --ext go --yes
error: init rejected (already_initialized)
  found: manifest extensions [extension-authoring, go]; requested [go]
  fix:   change extensions with `solomonar extension add|remove`; init never changes choices
nothing was written
```

Exit 3. The first differing choice is reported, in the order name, main branch, remote, runtimes, extensions,
adversarial models, with no check line. Found is `manifest <field> <value>; requested <value>` with the field `name`,
`main_branch`, `remote`, `runtimes`, or `extensions`; strings are quoted (`manifest name "ledger"; requested
"billing"`), lists are sorted (`[extension-authoring, go]`), and an empty list is `none`. Runtimes and extensions are compared as
sets, sorted with duplicates dropped; a list given as `none` counts as given and is compared (`manifest extensions
[go]; requested none`); only a choice not given is filled from the manifest. A given adversarial model that differs
from the manifest's is found as `manifest adversarial_model <r> "<v>"; requested "<v>"`, with `none` for an absent
model. The fix is `rerun without --name; init never changes choices` (likewise `--main-branch`, `--remote`),
``change runtimes with `solomonar runtime add|remove`; init never changes choices``, ``change extensions with
`solomonar extension add|remove`; init never changes choices``, or ``change models with `solomonar runtime add
<runtime> --adversarial-model <model>`; init never changes choices``. `--objective` is not a
recorded choice and is ignored on a re-run (`docs/README.md` is a seed).

A managed file edited since installation whose offered content is unchanged is shown as `keep` and is not a
conflict (4.5). It becomes a conflict only when the offered content changes too, as when a newer CLI re-runs `init`
after `docs/solomonar/rules.md` was edited. An unmanaged file in a target path whose content differs from what the
CLI would write is a conflict; an unmanaged file with identical content is adopted (3.3), so a re-run after an
interrupted `init` completes it:

```
$ solomonar init --runtime claude --yes
Plan
  conflict  docs/solomonar/rules.md        modified since installation (manifest sha256 differs)
  conflict  .claude/agents/implementer.md  exists and is not managed by solomonar
error: init found 2 conflicts (file_conflict)
  check: no planned file conflicts with the project's files
  found: 2 conflicts: docs/solomonar/rules.md, .claude/agents/implementer.md
  fix:   rerun with --keep <path> to keep the project's file, or --overwrite <path> to replace it
nothing was written
```

Exit 4. `Plan` and the conflict lines go to stdout, the error to stderr; JSON carries no `data`. `--keep` preserves
the project's bytes and records the offered distributed base hash, never the kept content's hash (4.5);
`--overwrite` writes the embedded version and records that same base. The registry is never recreated: an existing
valid `workflow-state` is reused, `unchanged`, and its tip is reported; an invalid one stops `init` with
`registry_exists_invalid` (exit 3), check `an existing workflow-state holds a valid registry`, the registry's own
found text, and the fix "inspect with git show refs/heads/workflow-state:state.json; the CLI never overwrites a
registry".

### UC-4 Initialize an existing repository

Not a Git repository:

```
$ solomonar init
error: /work/ledger is not a Git repository (not_a_repository)
  fix: run `git init -b main` first, or rerun with --git-init
```

Interactively, without `--git-init`, the same condition asks `Initialize a Git repository here with main branch
"<main-branch>"? [y/N]` (default No): yes continues exactly as `--git-init`; no ends its block with a blank line,
then prints `Cancelled; nothing was written.` and exits 0 (3.7). This question is its own block.

Non-interactively, `--git-init` plans against `<dir>` as the root (it must exist), with no manifest check against
Git and the registry `create`; the plan's first line is `  git        init -b <main-branch>` (`"git_init": true`).
The plan counts as a write, so it needs the confirmation or `--yes`; `--dry-run` never runs `git init`. After the
confirmation `init` runs `git init -b <main-branch> <dir>`, then applies the files and creates the registry as in
UC-2. A failure after `git init` rolls back the files but leaves the new `.git`; a re-run adopts the files
(architecture section 8). A failing `git init` is `unexpected_error` reading `git init: <git error>` (exit 1), with nothing
written. A `file_conflict` under `--git-init` prints only the conflict lines, with no `git` line, like any conflict.
Inside a repository `--git-init` is ignored.

Existing unmanaged `AGENTS.md`: the default is adoption, the existing content moved under `## Project notes`:

```
AGENTS.md exists and is not managed by solomonar.
  Adopt it: write the framework part above "## Project notes" and move the current content into that section? [Y/n]
```

Interactively, an unmanaged `AGENTS.md` planned for adoption and not named by `--overwrite` asks this question
(default Yes) before the plan display, unless the plan has other conflicts: then `file_conflict` is reported as
non-interactively. No asks `  Replace it with the framework template, discarding its content? [y/N]` in the same
block, with no blank line between them: yes plans again as with `--overwrite AGENTS.md`; no ends the block with a
blank line, then prints `Cancelled; nothing was written.` and exits 0, keeping warnings such as
`jujutsu_detected`. `--dry-run` asks neither and shows the adoption. The adoption block and the `git init` question
are separate blocks, each separated by a blank line (UC-1).

Non-interactively, adoption is the default with `--yes`; `--overwrite AGENTS.md` on this unmanaged file writes the
full template and discards the old content; `--keep AGENTS.md` is rejected, because the framework cannot run without
its `AGENTS.md` (I section 4.5):

```
error: init rejected (invalid_value)
  check: --keep and --overwrite resolve conflicts
  found: AGENTS.md cannot be kept
  fix:   adopt it (default) or pass --overwrite AGENTS.md
nothing was written
```

Existing `CLAUDE.md` without `@AGENTS.md`: never modified (G10):

```
  skip       CLAUDE.md  exists; add this line so Claude Code loads the framework:  @AGENTS.md
```

Other cases: an existing `docs/README.md` is `skip` (seed, project-owned). An empty repository, where
`refs/heads/<main-branch>` has no commit (always after `--git-init`), initializes; `Next:` step 1 is
`  1. First commit required before task worktrees:` followed by the command lines indented seven spaces, committing
only the generated paths on the main branch (`"first_commit_required": true`; `next` lists the same commands):

```
Next:
  1. First commit required before task worktrees:
       git switch --orphan main
       git add AGENTS.md .solomonar docs && git commit -m "chore: initialize solomonar X.Y.Z"
```

`git switch --orphan <main-branch>` comes first only when HEAD is not on `<main-branch>`; a detached HEAD counts as
not on it. `doctor` reports the same
missing baseline. An existing application whose commands are not recorded is not an empty repository: discover them
without adding a new application baseline. A `.jj/` directory at the worktree root adds the warning
`jujutsu_detected`, `warning: Jujutsu detected; v0.1 supports Git only. Never run jj commands on workflow-state.`
(stderr, or the envelope's warnings), on every successful outcome, including `--dry-run` and the no-op; it comes
before the registry's warnings (such as the no-remote warning).

### UC-5 Add, list, or remove a runtime

Specified on 2026-10-24 (CLI-D36) and implemented. Command page: [`solomonar runtime`](../../site/src/docs/cli/runtime.md). The commands manage only the
4.2 adapter files and the manifest's runtime choices; none installs, merges, or removes a guard hook or touches a
runtime settings or hook file, and none depends on `guard` (CLI-D36).

```
solomonar runtime list [--json]
solomonar runtime add <claude|codex|copilot> [--adversarial-model <model>]
                      [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
solomonar runtime remove <claude|codex|copilot>
                         [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
```

Global flags, `-C` included, apply, and flags may be interspersed with arguments (3.1). `runtime` alone or an
unknown subcommand is `unknown_command`; an argument to `list`, a missing or extra name for `add` or `remove`, or an
unknown flag is `invalid_usage`. Another name, `none` included (an `init` selection value, not a runtime), is
`unknown_runtime` (exit 2): check `<runtime> names a known runtime`, found `"<v>" is not a runtime; known: claude,
codex, copilot`, fix `pass claude, codex, or copilot`.

**`runtime list`** is read-only, never prompts, and works in any Git repository, initialized or not. It reads the
manifest when present; an older valid manifest is read, a newer one is `cli_outdated` and an invalid one
`manifest_invalid` (3.6). Without a manifest every runtime is `available`. It loads no catalog or store, generates
no adapter, runs no runtime, and checks no file health. One row per runtime in the order `claude`, `codex`,
`copilot`; columns, two spaces apart and padded to the widest cell, the last unpadded:

| Column | Value |
|---|---|
| `RUNTIME` | the name |
| `STATUS` | `installed` when the manifest's `runtimes` has the name, else `available`; executable presence never changes it |
| `ON PATH` | `yes` or `no`, from a `PATH` lookup of the executable (`claude`, `codex`, `copilot`) |
| `ADAPTER` | the 4.2 locations: `CLAUDE.md, .claude/skills, .claude/agents`; `AGENTS.md (native), .agents/skills, .codex/agents`; `AGENTS.md (native), .github/skills, .github/agents` |

`--json` data: `{"runtimes": [{"name": "claude", "status": "installed", "on_path": true, "adapters": ["CLAUDE.md",
".claude/skills", ".claude/agents"]}, …]}`, in the same order; Codex and Copilot list `AGENTS.md` first, as in the
human column without `(native)`.

**`runtime add` and `runtime remove`** reuse the desired-state planner of 4.5: the choices are the manifest's,
with only the named runtime, or its model, changed; the installed catalog and project extensions resolve through the
store as in UC-9, each stack group keeps its recorded decision, and no default or detected extension is added. The
complete wanted set goes to the planner, never one runtime's files alone, so other runtimes' files stay managed.
`add` creates the runtime's core and installed-extension stubs, Codex policy files, and a create-only `CLAUDE.md`
(4.2), owned `runtime:<name>`; `remove` makes them no longer wanted, so each is `remove` when unmodified, dropped from
the manifest when already absent, and `conflict` when edited (4.5). The plan may also restore a missing managed file
or refresh a generated stub under the 4.5 rows, and every resulting action is reported; there is no implicit
framework upgrade. `AGENTS.md` has no runtime field and normally stays `unchanged`.

Removal keeps an unmanaged `CLAUDE.md`, unrelated files under `.github`, `.claude`, `.codex`, and `.agents`, the
portable framework, and project extension directories; empty parents are pruned as in 4.5, so the top-level
`.claude`, `.codex`, `.agents`, and `.github` directories stay, even when left empty. Removing the last
runtime leaves the framework and extensions installed. No runtime command writes `workflow-state`, a branch, a
commit, or a remote.

`--adversarial-model <model>` takes a plain model string, not `init`'s `<runtime>=<model>`. Given, it becomes the
named runtime's `adversarial_model`, also for an installed runtime, whose affected role stub is then updated by the
planner; an explicitly empty value is `invalid_value` (exit 2), check `--adversarial-model names a model`, found
`empty model`, fix `pass a model or omit --adversarial-model`. Omitted, an installed runtime keeps its recorded
model and a new one has none. There is no runtime or model prompt and no model default. The model is serialized by
the 4.2 rules; the CLI never authenticates or checks model availability. Adding works when the executable is not on
`PATH`. `--keep` and `--overwrite` follow 4.5, including the in-both, not-a-conflict, and non-regular removal
checks.

Checks run in this order, the first failure ending the command with nothing written: arguments and the model;
the repository and Git (3.2); the manifest (absent is `not_initialized`, newer `cli_outdated`, invalid
`manifest_invalid`); the framework (`framework_outdated` when older than this CLI, 3.6); the installed extensions;
the plan and its resolutions (`file_conflict`, exit 4, never resolved by `--yes`); `--dry-run` or the no-op;
the confirmation; the apply.

No-ops: `add` of an installed runtime whose plan changes nothing, and `remove` of a supported runtime not in the
manifest. The second returns without planning project files, with `--dry-run` too, but a `--keep` or `--overwrite`
path on it is still `invalid_value` (`<path> is not a conflict`). A no-op writes nothing, needs no confirmation, and
exits 0.

**Writing output.** As for `extension add|remove` (UC-7): on a terminal without `--yes` the grouped plan and
`Proceed? [Y/n] ` go to stderr; without a terminal, or with `--json`, and without `--yes`, a write is
`interactive_required` (exit 2); `--dry-run` prints `Plan` and every file line to stdout, then `Would add runtime
<name> in <root>` (or `Would remove`) and the counts; the `file_conflict` plan lists every file on stdout before the
error. The success form is `Added runtime <name> in <root>` (`Removed runtime …`), `  <counts>`, a blank line, and
`Next:` with `  1. git add <paths> && git commit -m "chore: add solomonar runtime <name>"` (`remove`: `"chore:
remove solomonar runtime <name>"`); `add` follows with `  2. ` and the runtime's session invocation (4.2, Starting a
session). The paths include removed tracked paths and name individual files under `.github` rather than the whole
directory, `CLAUDE.md` only when created or removed. The CLI never runs the suggestion. The no-op forms follow
UC-7: `Already installed: <name>; nothing to do.` and the counts for `add`, `Not installed: <name>; nothing to do.`
for `remove`.

`--json` data: `{"project", "runtime", "adversarial_model", "files": [...], "next": [...]}`, with `files` entries as
for `init` (UC-2), `adversarial_model` the resulting selection and absent when there is none (always for
`remove`), `"dry_run": true` and no `next` for `--dry-run`, and `"next": []` for a no-op. `next` holds only the
commit suggestion; the session step of `add` is human output only, as for `init`. A failure has no `data`
(3.4).

| Exit | Result |
|---|---|
| 0 | success, `--dry-run`, no-op, or declined confirmation |
| 1 | unexpected I/O or Git failure, `manifest_invalid`, prompt input failure |
| 2 | `unknown_command`, `invalid_usage`, `unknown_runtime`, `invalid_value`, `interactive_required` |
| 3 | `not_a_repository`, `not_initialized` (`add`, `remove`), `cli_outdated`, `framework_outdated` |
| 4 | `file_conflict`, a removal conflict included |
| 130, 143 | `interrupted` before the file plan completes; the journal rolls back every change (3.5) |

No environment variable or configuration option is added: detection uses `PATH`, the global output controls apply,
and `SOLOMONAR_OWNER` has no effect.

### UC-6 Choose extensions for an architecture or stack decision

Typical triggers: the user decides part of the stack after the prototype, or an orchestrator applies an approved
task decision that adopts a technology or practice. Command page: [`solomonar
extension`](../../site/src/docs/cli/extension.md). The store, the loader, and the decision rules are 4.3; this use
case adds the command contract.

```
solomonar extension list [--group <group>] [--kind <skill|agent>] [--installed] [--json]
solomonar extension add <name>... [--decision <ID>] [--rationale <text>] [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
solomonar extension remove <name>... [--decision <ID>] [--rationale <text>] [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
```

`ext` is an alias for `extension` (3.1); `ext` alone, with no subcommand, is `unknown_command` naming `"extension"`.
The commands take names, never paths, and have no metadata flags: an extension's metadata is its `extension.json`.
`--decision` and `--rationale` apply to stack groups only.

**`extension list`** is read-only and never prompts. It merges the embedded catalog, the store, and the manifest:
one row per catalog extension and per project extension (4.3 Loading), ordered by group (4.3 order, an unknown or
unreadable group last), then name, then source. A catalog name that is also a store directory with `extension.json`, not recorded as
that catalog extension, gives two rows: the catalog one, then the project one, `invalid: name "<n>" is also a catalog
extension`. Columns, two spaces apart and padded to the widest cell, the last unpadded:

| Column | Value |
|---|---|
| `GROUP`, `NAME`, `KIND` | from `extension.json`; for an invalid entry whose metadata cannot be read, `-` for group and kind |
| `SOURCE` | `catalog` or `project` |
| `STATUS` | `installed` (a manifest entry), `available`, or, for a project extension only, `invalid: <reason>` (the 4.3 loader's first failure; an installed project extension that no longer validates too) |
| `DECISION` | the manifest `decision`, else `-` |
| `SUGGESTED` | for a catalog or valid project extension that is not installed, its bare detection reason (the first matching path, such as `go.mod`, or `default`, 4.3); else `-` (an invalid row is never suggested) |
| `APPLIES WHEN` | `applies_when` verbatim, `-` when it cannot be read |

`--group` keeps one group (another value is `unknown_group`, exit 2: check `--group names a group`, found `"<v>" is
not a group; known: <groups>`, fix `pass --group with a known group`); `--kind` keeps one kind (`invalid_value`,
check `--kind is skill or agent`, found `"<v>" is not skill or agent`, fix `pass --kind skill or --kind agent`);
`--installed` keeps installed rows. Flag values are checked before the repository. A filter that matches nothing
prints only the header line. An invalid entry never stops the listing. Without a manifest every row is `available`
(`list` needs no `init`); outside a repository the command is `not_a_repository`; a newer manifest is
`cli_outdated`.

```
$ solomonar extension list
GROUP     NAME                 KIND   SOURCE   STATUS                            DECISION  SUGGESTED  APPLIES WHEN
language  go                   skill  catalog  installed                         P/D-1     -          A change touches Go code (*.go, go.mod, go.sum)
backend   money                skill  project  available                         -         -          A change handles monetary amounts
workflow  extension-authoring  skill  catalog  available                         -         default    A request creates, changes, or validates a Solomonar extension skill, its rules, or a specialist reviewer
review    a11y-reviewer        agent  project  invalid: applies_when is missing  -         -          -
```

`--json` data is
`{"extensions": [...]}`, one item per row: `name`, `group`, `kind`, `source`, `status` (`installed`, `available`,
`invalid`), `reason` (only when invalid), `decision` (only when installed in a stack group), `suggested` (only when
suggested), `title`, `applies_when`, `paths`, `requires`, `conflicts`, `invocation`, and `path` (the store directory
of a project extension, and of an installed catalog extension). An invalid row keeps the fields its `extension.json`
yields; a field it cannot read is an empty string (`-` in the table).

**`extension add`** activates extensions. Each sorted unique name resolves in the catalog, then in the store:

| Code | Exit | When | check | found | fix |
|---|---|---|---|---|---|
| `name_collision` | 3 | the name is a catalog extension and a store directory with `extension.json` that the manifest does not record as that catalog extension | `the name is not both a catalog and a project extension` | `<n> is in the catalog and in docs/solomonar/extensions/<n>` | `rename or remove docs/solomonar/extensions/<n>/` |
| `unknown_extension` | 2 | neither | `extension add names a catalog or project extension` | `"<v>" is not in the catalog or docs/solomonar/extensions/; known: <sorted available names: catalog and valid project extensions not installed, or none>` | `pass a name from solomonar extension list` |
| `invalid_value` | 2 | a project extension fails the 4.3 loader | `docs/solomonar/extensions/<n> is a valid extension` | the loader's reason | `fix docs/solomonar/extensions/<n>/ and rerun; the extension-authoring skill validates it` |

Checks run in this order: usage (no name is `invalid_usage`, `extension name is required`); the repository and the
Git version; the manifest (`not_initialized`, `cli_outdated`, `manifest_invalid`); `framework_outdated` (3.6); name
resolution, per sorted name; `--decision` or `--rationale` when no named extension is in a stack group
(`invalid_usage`, `--decision and --rationale apply to stack groups only`); the 4.3 selection checks over the
resulting installed set, with the texts below; a malformed `--decision` (`invalid_value`: check `--decision is
P/D-<n> or <task-ID>/D-<n>`, found `"<v>" is not P/D-<n> or <task-ID>/D-<n>`, fix `pass the ID of an approved entry in
docs/decisions.md`; a task ID is `W-` and three or more digits); `decision_not_found`, then `decision_not_approved`
(an entry without a Status line has status `none`). `--decision` is checked whenever it is given with a stack-group
name, even when the plan turns out a no-op. Then the plan
(`invalid_value` from `--keep`/`--overwrite`) and its conflicts (`file_conflict`, exit 4); `--dry-run`; the
confirmation; the apply. Any apply error rolls back through the journal (architecture section 8).

| Code | check | found | fix |
|---|---|---|---|
| `extension_requires` | `every required extension is installed` | `<name> requires <missing, comma-joined>` | `add them in the same call: solomonar extension add <names> <missing>` |
| `extension_conflict` | `no installed extensions conflict` | `<a> conflicts with <b>` | `remove <b> first, or drop <a>` |

A catalog extension's files are created in the store; a project extension gets no file (4.3). Both get their stubs
(4.2), their `AGENTS.md` line (4.6), their manifest entry, and, in a stack group, the decision (4.3). An already
installed name is valid: its stubs and line are refreshed. When the plan creates, updates, or removes nothing, the
command is a no-op that needs no confirmation and exits 0:

```
$ solomonar extension add go
Already installed: go; nothing to do.
  62 files unchanged, 1 skipped
```

**`extension remove`** deactivates installed extensions. Checks, in order after the manifest and
`framework_outdated`: per sorted name, a core skill or role (`solomonar`, `prototype`, `spec`, `work-spec`,
`orchestrator`, `implementer`, and the four core reviewers) is `core_item` (exit 3): check `extension remove names
an extension`, found `<n> is part of the core`, fix `the core cannot be removed (4.1)`; a name that is not installed
is `unknown_extension` (exit 2): check `extension remove names an installed extension`, found `"<v>" is not installed;
installed: <sorted names, or none>`, fix `pass a name from solomonar extension list --installed`; each name is checked
for both before the next. Then `--decision`/`--rationale` as for `add`; the remaining installed extensions resolve as
in UC-3, so another installed project extension that fails the loader is `invalid_value` (UC-7), while a removed one
need not load; then `extension_requires` on the resulting set: check `every required extension is installed`, found
`<other> requires <missing, comma-joined>`, fix `remove both: solomonar extension remove <names> <other>` (the
removed names, then `<other>`). The decision checks, the plan, conflicts (the `file_conflict` fix also reads
`--overwrite <path> to replace it`), `--dry-run`, the confirmation, and the apply follow as for `add`.

A catalog extension's store files become no longer wanted, so they are `remove`, or `conflict` when edited (4.5); a
project extension's directory is never touched, and `extension list` shows it `available` again. Stubs, the
`AGENTS.md` line, and the manifest entry go in both cases. A group left empty records the selection `none` (4.3).
Only after a successful apply (never for `--dry-run` or a rejection), each removed name prints `warning: code governed
by <name> loses its rules` on stderr (`{"code": "rules_removed", ...}` in JSON); the CLI never checks whether code
still uses a technology. Removing `go` from the UC-1 project:

```
$ solomonar extension remove go --yes
Removed extension go (language skill, catalog, under P/D-2) in /work/ledger
  4 files updated, 8 removed, 50 unchanged, 1 skipped

Next:
  1. git add AGENTS.md .claude .agents .codex .solomonar docs && git commit -m "chore: remove solomonar extension go"
warning: code governed by go loses its rules
```

`P/D-2` is the emptied group's entry of 4.3; `extension list` shows `go` `available` again, suggested by `go.mod`.

**Writing commands' output.** `add` and `remove` follow `init`: on a terminal without `--yes` they print `Plan`, the
grouped plan of UC-1 (with the `decisions (approved, selected by you)` group; `unchanged` lines and `skip` lines
without a reason are left out before sets collapse, and the counts still include them), and `Proceed? [Y/n] `, the plan going
to stderr because a prompt follows (architecture section 2); without a terminal and without `--yes` they are
`interactive_required` (fix `pass --yes to accept the plan, or --dry-run to print it`); `--yes` prints no plan;
`--dry-run` prints `Plan` and every file line to stdout, then `Would add extension <names> in <root>` (or `Would
remove`) and the counts; the `file_conflict` plan lists every file. The success form is `Added extension <names>
(<group> <kind>, <source>[, under <ID>]) in <root>` (one name; with several, `Added extensions <a>, <b> in <root>`;
`<ID>` is the decision this run records for the group, else the manifest's recorded one, as when an installed name
is refreshed),
`  <counts>`, a blank line, and `Next:` with `  1. git add <paths> && git commit -m "chore: add solomonar extension
<names>"` (`remove`: `Removed …`, `"chore: remove solomonar extension <names>"`); the paths follow the UC-2 rule,
including `CLAUDE.md` only when created. `--json` data: `{"project", "added"` (or `"removed"`) `: [names],
"decisions": [{"id", "group", "selected"}], "files": [...], "next": [...]}`, with `"dry_run": true` and no `next` for
`--dry-run`, and `"next": []` for a no-op.

Adding `templ` to a frontend group that holds `htmx` under `P/D-2` (the illustrative test catalog of 4.4) supersedes
`P/D-2` with the full new selection and rebinds `htmx`:

```
$ solomonar extension add templ --rationale "Type-checked components for the htmx fragments"
Plan
  framework X.Y.Z
    update     AGENTS.md
  extensions
    create     docs/solomonar/extensions/templ/ (2 files)
  decisions (approved, selected by you)
    update     docs/decisions.md  P/D-3 frontend: htmx, templ · P/D-2 superseded
    append     docs/stack.md      frontend: htmx, templ (P/D-3)
  runtime claude
    create     .claude/skills/ (1 skill stub: 0 core, 1 extension)
  runtime codex
    create     .agents/skills/ (1 skill stub: 0 core, 1 extension)
  manifest
    update     .solomonar/manifest.json
Proceed? [Y/n] y

Added extension templ (frontend skill, catalog, under P/D-3) in /work/ledger
  4 files created, 4 updated, 62 unchanged, 1 skipped

Next:
  1. git add AGENTS.md .claude .agents .codex .solomonar docs && git commit -m "chore: add solomonar extension templ"
```

The `htmx` and `templ` lines of `AGENTS.md` both read `(frontend, P/D-3)`, and both manifest entries hold `P/D-3`;
`htmx`'s files are unchanged. Removing `templ` again records `P/D-4` with the selection `htmx` and supersedes
`P/D-3`; removing both records an entry whose selected option is `none`. Re-filling an emptied group supersedes its
`none` entry: `init --ext go` records `P/D-1`, `extension remove go` `P/D-2` (`none`, superseding `P/D-1`), and
`extension add go` `P/D-3`, superseding `P/D-2`.

A `--decision` that is not approved is rejected (exit 3):

```
$ solomonar extension add templ --decision W-002/D-2
error: extension add rejected (decision_not_approved)
  check: --decision must name an approved entry in docs/decisions.md
  found: W-002/D-2 has status pending
  fix:   wait for the user's approval (rules section 3), then rerun
nothing was written
```

A `--decision` absent from `docs/decisions.md` is `decision_not_found`: found `W-002/D-2 is not in
docs/decisions.md`, fix `record the decision first (rules section 3), then rerun`.

### UC-7 Add workflow skills, review agents, or the project's own extensions

Project extensions cover what the catalog does not: a domain rule set, an in-house library, a supporting procedure,
or an extra reviewer. They use the catalog format in the store, `docs/solomonar/extensions/<name>/`, and are
project-owned: no command writes, tracks, or upgrades their files (4.3, 4.5).

The `extension-authoring` skill writes a project extension, `extension.json` included, and validates it; it ends by
handing off `solomonar extension add <name>`. A `workflow` or `review` extension records no decision; a project
extension in a stack group records one as in UC-6. Adding the `review` agent `money-reviewer`, written to
`docs/solomonar/extensions/money-reviewer/`:

```
$ solomonar extension add money-reviewer --yes
Added extension money-reviewer (review agent, project) in /work/ledger
  2 files created, 2 updated, 60 unchanged, 1 skipped

Next:
  1. git add AGENTS.md .claude .agents .codex .solomonar docs && git commit -m "chore: add solomonar extension money-reviewer"
```

The two created files are the role stubs `.claude/agents/money-reviewer.md` and `.codex/agents/money-reviewer.toml`
(the reviewer form, 4.2); `AGENTS.md` and the manifest are updated; `docs/solomonar/extensions/money-reviewer/` is
unchanged, and triage now lists the agent beside the core specialists. Adding the catalog `workflow` skill
`extension-authoring` to a project initialized without it creates its six store files, its stubs, and, for codex,
`.agents/skills/extension-authoring/agents/openai.yaml` (user-invoked, 4.2).

An extension name is unique across the catalog and the store (`name_collision`, UC-6), and a skill's `name` must
match its directory (G11). Rejections: `core_item` for `extension remove work-spec` or `extension remove
security-reviewer`; `unknown_extension` for a name in neither the catalog nor the store; `invalid_value` for a project
extension that fails the 4.3 loader, naming its reason, such as `applies_when is missing`. An installed project
extension whose directory later fails the loader, or is gone, stops `init`, `upgrade`, and every `extension add|remove`
that keeps it installed with the same `invalid_value`; `extension remove <name>` still deactivates it.

### UC-8 Regenerate adapters (withdrawn)

Withdrawn on 2026-10-25 (CLI-D37); the number stays unused. `upgrade` at the project's version regenerates stubs
and restores missing managed files (UC-9), and `doctor` reports stale adapters for CI (UC-10).

### UC-9 Upgrade the framework

Delivered first as a minimal `upgrade` (CLI-D31), before the `extension` commands; `--diff` is deferred:

```
solomonar upgrade [--keep <path>]... [--take-new <path>]... [-y|--yes] [--dry-run] [--json]
```

The target is the framework version embedded in the running CLI. The choices (runtimes, models, installed catalog and
project extensions) come from the manifest, and each stack group keeps its recorded decision: no decision is written
and no decision entry is edited. An extension that became `default` or is newly detected is never added, and an
installed `path` extension resolves through the store as in UC-3. The desired-state planner (4.5) computes the wanted paths; per managed file, with
`base` = manifest hash, `current` = hash of the file on disk, `new` = hash of the offered embedded content:

| Condition | Action |
|---|---|
| current = base, new = base | `unchanged` |
| current = base, new ≠ base | `update` |
| current ≠ base, new = base | `keep` (project edit preserved) |
| current = new | `unchanged` (manifest base moves to new) |
| current ≠ base, new ≠ base, current ≠ new | `conflict` |
| no longer offered, current = base | `remove` |
| no longer offered, current ≠ base | `conflict` |

After a successful upgrade, the manifest base is `new` for every managed upstream file, including one resolved
with `--keep`; its local content is left intact. For example: init offers A, keeps local B, and records base A.
An upgrade offering C sees current B ≠ base A and new C ≠ base A, so it conflicts instead of overwriting B.
Keeping B records base C; a later unchanged C preserves B, and a later D conflicts again. `--take-new` writes the
offered content, or deletes a no-longer-offered file; `--keep` on a no-longer-offered file untracks it and leaves it
in place (4.5). Project extension files and decision documents are never touched; `AGENTS.md` changes only in its
managed part. Known limitation: after moving a 0.1.7 project to the store, its decision entries keep their 0.1.7
References, such as `` `docs/solomonar/skills/go/SKILL.md` ``, which no longer exist; the next decision for the group
names the store paths. Known limitation: a newer catalog that no longer ships an installed catalog extension makes
`upgrade` stop with the unexpected error `read extension <name>: not in the catalog`; 0.2.0 does not handle it.

Checks run in this order: usage; the repository and the Git version; the manifest: absent is `not_initialized`
(3.6), newer is `cli_outdated` (a downgrade; fix "install a newer solomonar release"), then `manifest_invalid`; the
plan: a manifest `catalog` extension missing from the catalog is an unexpected error `read extension <name>: not in
the catalog`, an installed project extension that fails the loader is `invalid_value` (UC-7), then `invalid_value`
from `--keep`/`--take-new`; conflicts (`file_conflict`, exit 4, nothing written); `--dry-run`; the confirmation
(`interactive_required` without a terminal and without `--yes`); the apply, rolled back through the journal on any
error. A project at the current version whose plan creates, updates, or removes nothing is a no-op, exit 0, with no
confirmation:

```
$ solomonar upgrade
Already at solomonar X.Y.Z; nothing to do.
  53 files unchanged, 1 skipped
```

Moving a 0.1.7 project with `go` and the claude and codex runtimes to the store layout of the next release
(illustratively 0.2.0); the 0.1.7 install under `docs/solomonar/skills/go/` is no longer offered:

```
$ solomonar upgrade
Upgrade framework 0.1.7 → 0.2.0
  framework X.Y.Z
    update     docs/solomonar/agents/ (6 roles)
    update     docs/solomonar/report.md
    update     docs/solomonar/reviewer.md
    update     docs/solomonar/rules.md
    update     docs/solomonar/skills/prototype/ (SKILL.md, template.md)
    update     docs/solomonar/skills/solomonar/ (SKILL.md)
    update     docs/solomonar/skills/spec/ (SKILL.md, template.md)
    update     docs/solomonar/skills/work-spec/ (SKILL.md, template.md)
    update     AGENTS.md
  extensions
    create     docs/solomonar/extensions/go/ (6 files)
    remove     docs/solomonar/skills/go/ (5 files)
  runtime claude
    update     .claude/skills/ (5 skill stubs: 4 core, 1 extension)
    update     .claude/agents/ (5 role stubs; adversarial-reviewer model opus)
  runtime codex
    update     .agents/skills/ (5 skill stubs: 4 core, 1 extension; solomonar/agents/openai.yaml)
    update     .codex/agents/ (5 role stubs)
  manifest
    update     .solomonar/manifest.json
Proceed? [Y/n] y

Upgraded solomonar 0.1.7 → 0.2.0 in /work/ledger
  6 files created, 39 updated, 5 removed, 3 unchanged, 1 skipped

Next:
  1. git add AGENTS.md .claude .agents .codex .solomonar docs && git commit -m "chore: upgrade solomonar to 0.2.0"
  2. Running tasks pick up the new framework after they update from main.
```

At the project's version, `upgrade` is a refresh (CLI-D37): it applies the same plan, which regenerates a stub
whose project extension metadata changed, recreates a deleted stub or managed file, and restores a stub left stale by
a merge, with the same conflict rules. The header is then `Refresh framework <version>`, `--dry-run` ends with
`Would refresh solomonar <version> in <root>`, the success line is `Refreshed solomonar <version> in <root>`, and
step 1 commits with `chore: refresh solomonar <version> files`. The JSON data keeps `from` = `to`.

The header is `Upgrade framework <old> → <new>`, followed by the
grouped plan of UC-1 without `git`, `decisions`, and `registry` groups and without `unchanged` lines or `skip` lines
that give no reason (as for `extension add`), then `Proceed? [Y/n] `, all on stderr because a prompt follows
(architecture section 2; unlike `init`, which prints its plan and prompt on stdout). `--yes` prints neither the
header nor the plan, only the result. `--dry-run` prints the header and every file line to stdout, then `Would upgrade
solomonar <old> → <new> in <root>` and the counts. A conflict prints the header and every file line, then:

```
error: upgrade found 1 conflict (file_conflict)
  check: no planned file conflicts with the project's files
  found: 1 conflict: docs/solomonar/skills/go/SKILL.md
  fix:   rerun with --keep <path> to keep the project's file, or --take-new <path> to take the new version
nothing was written
```

The success form is `Upgraded solomonar <old> → <new> in <root>`, `  <counts>`, a blank line, and `Next:` with step 1
`git add <paths> && git commit -m "chore: upgrade solomonar to <new>"` (paths by the UC-2 rule) and step 2 as above.
`--json` data: `{"project", "from", "to", "files": [...], "next": [...]}`, with `"dry_run": true` and no `next` for
`--dry-run`, and `"next": []` for a no-op.

`upgrade` never touches the registry: a registry schema change is the separate `task migrate` (UC-25, CLI-D26),
so framework files and registry state are updated by different commands. The upgrade never commits on main.
`upgrade --diff <path>` (architecture section 2, F5) remains increment 3.

### UC-10 Check the project

Synopsis, the check areas with their `fail` and `warn` levels, and the example:
[`solomonar doctor`](../../site/src/docs/cli/doctor.md). A newer project is a `fail` (`cli_outdated`, 3.6), and
every installed stack extension must point to an approved decision for its group's complete current selection
(4.3), and every installed project extension must pass the 4.3 loader. A missing `gh` makes the forge
facts `unknown`, never passing.

### UC-27 Uninstall Solomonar from a project

Specified on 2026-10-24 (CLI-D32 to CLI-D35, architecture AD-16) and implemented; the command page is
[`solomonar uninstall`](../../site/src/docs/cli/uninstall.md).

```
solomonar uninstall [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
```

`uninstall` removes the project installation from the checkout selected by the working directory or `-C`, a linked
worktree included: the framework core, every installed catalog extension, the runtime adapters, the generated
`AGENTS.md` part and index line, and the manifest. It is the only command that removes the core (4.1, CLI-D32);
`extension remove` still rejects a core item with `core_item`, and there is no `remove` alias and no selective core
removal. It does not remove the executable, a Homebrew package, runtime applications, or user-wide configuration,
and it changes no other worktree, branch, ref, or remote. It takes no positional argument, adds no environment
variable (`SOLOMONAR_OWNER` is irrelevant here), and uses no network.

**Checks**, in order: usage; the repository and the Git version (3.2); the manifest. An absent manifest is a no-op,
not `not_initialized`: exit 0, no prompt, no `--yes` needed, and the line `No Solomonar manifest; nothing to
uninstall automatically.` The CLI never searches for installation files without the manifest. `--keep` and
`--overwrite` still need real conflicts, so with an absent manifest any such path is `invalid_value`. A manifest
that is a symlink, directory, or special file, or lies under a symlinked `.solomonar`, is never read:
`file_conflict` (exit 4, no plan), check `the manifest is a regular file`, found `.solomonar/manifest.json exists
and is not a regular file`, fix `restore .solomonar/manifest.json as a regular file, then rerun solomonar
uninstall`; neither flag resolves it. A newer
manifest is `cli_outdated` (3.6); a malformed one is `manifest_invalid`. An older framework that the strict decoder
reads, the 0.1.7 layout included, is accepted without `framework_outdated` and without a prior `upgrade`. Then the
plan and `invalid_value` from `--keep`/`--overwrite` (4.5 rules: a path in both flags, an unsafe path, or a path that
is not a resolvable conflict of this plan; repeating a path adds nothing); conflicts (`file_conflict`, exit 4,
nothing written); `--dry-run`; the confirmation (`interactive_required`, exit 2, without a terminal or with `--json`
and without `--yes`); the input recheck; the apply, rolled back through the journal on any handled error. `--yes`
never resolves a conflict.

**Planning** (CLI-D33, AD-16). The plan comes from the decoded manifest and the current files, never from today's
catalog, store, or templates: no core or adapter file is rendered, no `extension.json` is loaded or validated, and
no decision is written. Each recorded path is planned by the first matching row:

| Record or path | Condition | Action |
|---|---|---|
| a project extension's directory, a legacy `project-ext:<name>` record, or a hashed file under a `path` extension's directory | any | `skip`, `project extension; source kept`; source wins over the hash record |
| `runtime:<name>` entry (`entry` instead of `sha256`) | any | `conflict`, `hook entry removal is not supported`; the containing JSON file is never deleted and its record is never dropped |
| a hashed `core`, `ext:<name>`, `core+notes`, or whole-file `runtime:<name>` record outside the paths `uninstall` may remove (below) | any | `conflict`, `outside the paths solomonar uninstall removes`; `--overwrite` cannot resolve it |
| `core`, `ext:<name>`, whole-file `runtime:<name>` (`CLAUDE.md` included), legacy paths such as `docs/solomonar/skills/go/` | absent | no plan line |
| same | regular file, hash = recorded `sha256` | `remove` |
| same | edited | `conflict`, `modified since installation (manifest sha256 differs)` |
| same | a symlink, directory, or special file, or reached through a symlinked parent directory (inside or outside the checkout) | `conflict`, `exists and is not a regular file`; never followed or removed recursively |
| `AGENTS.md` (`core+notes`) | see below | `remove`, `update`, or `conflict` |
| `docs/README.md` | see below | `update` or `skip` |
| `decision` files and other documents | any | not planned; kept |
| `.solomonar/manifest.json` | a regular file (otherwise rejected before planning, see Checks) | `remove`, the last file change |

Each `path` extension of the manifest, a draft included, also gets one `skip` line for its directory (owner
`project-ext:<name>`), with the same reason, so the plan shows the kept sources. The plan lists every line in path
order, then the manifest.

The paths `uninstall` may remove are `docs/solomonar/**` (core and catalog extension files, legacy locations
included) except a project extension's directory, the adapter stub directories of 4.2 (`.claude/skills/`,
`.claude/agents/`, `.agents/skills/`, `.codex/agents/`, `.github/skills/`, `.github/agents/`), `CLAUDE.md`,
`AGENTS.md`, and the manifest. Nothing in `.git`, `.jj`, or Git metadata is a removal target.

`AGENTS.md` (CLI-D34) is split at the first `## Project notes` line, as in 4.5. When the managed part's hash equals
the record: with a notes part holding project content (adopted instructions included), the file is `update`, reason
`managed part removed; project notes kept`, and is rewritten as exactly the notes part, byte for byte, from its
heading on; with no notes part, or one holding only whitespace and the template's heading and comment, the file is
`remove`; the template comment is the current one or the 0.1.0 and 0.1.1 one. A changed managed part is `conflict`,
`modified since installation (manifest sha256 differs)`. No `## Project notes` line, so the managed part cannot be
verified, is `conflict`, `no "## Project notes" boundary`. The pre-install formatting of an adopted file cannot be reconstructed (the manifest does not record it).

`docs/README.md` (CLI-D35): when it is a regular project file, the one standalone line
`` - `docs/solomonar/`: framework rules, roles, and skills `` inside its `## Documents` section is removed
(`update`, reason `framework index line removed`); every other byte, line endings included, stays. A missing file
is not created. A customized entry, or the exact line outside `## Documents`, is left unchanged (`skip`, reason
`framework index entry customized; review by hand`) and reported for manual review with the warning
`manual_reference`; so is an updated file that still mentions `docs/solomonar`. A file without any `docs/solomonar`
mention gets no plan line. A non-regular file or a symlinked `docs` is never followed or rewritten: `skip`, reason
`exists and is not a regular file`, with the same warning.

Conflicts resolve as in 4.5. `--keep <path>` keeps the whole file as it is, untracked, with reason `untracked; left
in place`; this includes an unsafe target and a record outside the removal paths. `--overwrite <path>` authorizes
the removal of an edited regular file. For `AGENTS.md`, `--overwrite` removes the managed part and keeps the notes
part, removing the file when the notes part holds only the template; with no boundary, `--overwrite AGENTS.md` is
`invalid_value`: found `AGENTS.md has no "## Project notes" boundary`, fix `restore the boundary by hand or pass
--keep AGENTS.md`, so notes are never discarded. `--overwrite` on a non-regular path or a path outside the removal
paths is `invalid_value` with the 4.5 texts (found `<path> is not a regular file` or `<path> is outside the paths
solomonar uninstall removes`, fix `remove it by hand or pass --keep <path>`). A hook entry conflict cannot
be resolved by either flag until hook removal ships with the guard hooks; until then such a project cannot be
uninstalled automatically: `--keep` or `--overwrite` on it is `invalid_value`, found `<path> holds a hook entry, which
uninstall cannot remove yet`, fix `keep the installation until hook entry removal ships with the guard hooks`.

**Retained**: project extension sources and their extra files, `docs/decisions.md` and every decision entry (also
those naming removed extension paths), `docs/stack.md`, `docs/architecture.md`, specifications, work specs,
reports, prototypes, the rest of `docs/README.md`, the `AGENTS.md` notes, an unmanaged `CLAUDE.md`, untracked runtime
files and unrelated configuration, every kept path, `refs/heads/workflow-state` and its remote counterpart, task
branches, worktrees, `.git`, and `.jj`. `uninstall` writes no `- <group>: none` decision, because the project's
technology does not change. A retained `CLAUDE.md` that still contains `@AGENTS.md` after `AGENTS.md` is removed is
reported with the warning `manual_reference`.

**Directories.** A removal deletes the parent directories it leaves empty, as in 4.5. Only `uninstall` also removes
`docs/solomonar/extensions`, `docs/solomonar`, and `.solomonar` when empty, through the journal. The top-level
`docs`, `.claude`, `.codex`, `.agents`, and `.github` directories, and every nonempty directory, stay.

**Output.** In a terminal without `--yes`, `Plan`, the file plan grouped as in UC-9 (with `skip` lines that carry a
reason), and `Proceed? [Y/n] ` go to stderr; declining prints `Cancelled; nothing was written.` on stdout, exit 0.
`--dry-run` runs every check, prints `Plan`, every file line, and `Would uninstall solomonar from <root>` with the
counts on stdout, never prompts, needs no `--yes`, and writes no file, directory, or ref. The success form is:

```
Uninstalled solomonar from <root>
  <counts>
  kept: <retained paths>
  The registry (refs/heads/workflow-state) and other worktrees are unchanged.

Next:
  1. Review the removal with git status, then commit it.
```

The `kept:` line lists the plan's `skip`, `keep`, and `update` paths and the `manual_reference` paths, and is left
out when there are none. It never stages, commits, or pushes, and never claims that every mention of Solomonar is
gone. On `file_conflict` the plan goes to stdout and the error to stderr (UC-9's form; the fix reads `rerun with
--keep <path> to keep the project's file, or --overwrite <path> to remove it`). A second run is the
absent-manifest no-op. The `manual_reference` warnings (stderr, or the envelope's `warnings`, with `--dry-run` and on
success) read `docs/README.md still mentions docs/solomonar; review it by hand`, `CLAUDE.md still imports
@AGENTS.md, which uninstall removes; edit it by hand`, and, for each `--keep` path, `<path> is kept and no longer
managed; remove it by hand if unneeded`.

`--json` data: `{"project", "files": [...], "preserved": [...], "next": [...]}`, with `files` entries as for
UC-9 (`path`, `action`, `owner`, `reason`), `preserved` the `kept:` paths and `refs/heads/workflow-state`, sorted,
`next` the `Next:` step, `"dry_run": true` and no
`next` for `--dry-run`, and `{"project", "already_absent": true, "files": [], "preserved": [], "next": []}` for the
absent-manifest no-op. A failure uses the error envelope without `data`; warnings stay in the envelope.

**Apply.** After the confirmation and before the first write, the CLI checks that the manifest and every planned
file still have the content and type it planned from, by rereading the manifest bytes and planning again without and
with the resolutions; any change is `file_conflict`, message `uninstall found a change since the plan was shown`,
check `the planned files are unchanged since the plan was shown`, found `<path> changed since the plan was shown`
(the manifest's path when the manifest changed), fix `rerun solomonar uninstall`, and nothing is written. The apply then follows AD-6: handled
failures and signals (exit 1, 130, 143) roll back the removals, the `AGENTS.md` and `docs/README.md` rewrites, the
manifest, and the removed directories with their modes. When a rollback itself fails, the error keeps its code and
exit, with found `not restored: <paths>`, fix `restore these paths by hand, for example with git restore, then rerun
solomonar uninstall`, and the last stderr line `rollback incomplete; recover the paths named above` instead of
`nothing was written`. There is no cross-process lock and no recovery after
SIGKILL, power loss, or a concurrent write during the apply. Once every change has succeeded the command reports
success.

| Exit | Condition |
|---|---|
| 0 | Removal, dry run, absent-manifest no-op, or declined confirmation |
| 1 | `manifest_invalid`, `git_too_old`, an unexpected I/O or Git failure, or a failed apply (rolled back) |
| 2 | `invalid_usage`, `invalid_value`, `interactive_required` |
| 3 | `not_a_repository`, `cli_outdated` |
| 4 | `file_conflict`: an edited or unsafe managed file or manifest, a record outside the removal paths, a hook entry, or changed plan inputs |
| 130, 143 | `interrupted` before the removal completed; rolled back |

## 6. Use cases: task state

Synopsis, flags, the conventions shared by the task commands, and one example of each command:
[`solomonar task`](../../site/src/docs/cli/task.md). The examples below cover what that page does not show; they continue in `/work/ledger`.
Worktrees live at `/work/ledger-worktrees/<ID>` (rules section 4 step 4).

### UC-11 Plan a task (user)

The ID is the next free `W-NNN` (highest existing number plus one). `--depends-on` and `--related` must name existing
tasks (`task_not_found`). A missing `--title` is `invalid_usage`; a blank or whitespace-only title is
`invalid_value`. `task plan` takes no `--as`: it is the user's write (G16). Without a successful push the
last line ends at `registry @ <sha>` and the warning (`no_remote`, `push_failed`, `push_interrupted`) follows on
stderr.

`--json` data: `{"id": "W-003", "record": {"title": "Export expenses as CSV", "status": "planned", "updated_at":
"2026-10-20T14:58:03Z"}, "commit": "bed97175b06e68eb7ff5722897aa86b8ea75c8ef", "pushed": true}`.

`--dry-run` reads the registry, runs the same checks, and writes no commit:

```
$ solomonar task plan --title "Export expenses as CSV" --dry-run
Would plan W-003: Export expenses as CSV
  state(W-003): planned, plan "Export expenses as CSV"
```

`--dry-run --json` data: `{"id": "W-003", "record": { }, "dry_run": true}`.

### UC-12 Register a task (orchestrator): the registration barrier

Without an ID the task gets the next free `W-NNN`. Claiming a planned task: `solomonar task register W-003 --title
"Export expenses as CSV" --owner orchestrator-2`; an unknown ID is `task_not_found`. A title that differs from the
planned title replaces it with the warning `title_changed`: `title changed from "<old>"`. The claim keeps the
planned `depends_on` and `related_task`; `--depends-on` or `--related`, when given, replaces the planned value, so a
claim cannot clear planned dependencies. A blank or whitespace-only `--title`, and an `--owner` that is not an
agent ID `[a-z]+-[0-9]+` (fix "pass --owner orchestrator-1"), are `invalid_value`. The default `next_action` is
`<owner>: Create feature/<ID> and its worktree`; a given `--next` names `user` or the owner (8.3).

Rejections (exit 3), with the message `task register <ID> rejected` when an ID is given (3.6): `not_planned` (the
ID exists and is not `planned`; found `<ID> is <status>`; fix: `solomonar task show <ID>; resume the task
instead`); `owner_in_use` (the owner has a `reserved`, `running`, `blocked`, or
`paused` task; found `<owner> owns <ID> (<status>)`; fix: choose a new owner ID such as `orchestrator-3`, since an
orchestrator owns exactly one task). `dependency_not_done` is a warning per dependency that is not `done`, not a
rejection: `<ID> is not done; register, then block with task block --kind dependency` (rules section 4). The success line
"Registration confirmed" appears only after the compare-and-swap commit exists; `--dry-run` never prints it.

`--title` and `--owner` are required; a missing one is `invalid_usage` (`--title is required`, `--owner is
required`); when both are missing, the error is `--title is required`. The worktree path is
`../<repo>-worktrees/<ID>`, where `<repo>` is the main checkout's directory name: relative when the command runs in
the main checkout, absolute when it runs in another worktree. The base is the main branch
(`main` until the manifest names it). Without a successful push the registry line ends at `registry @ <sha>`, the
warnings follow on stderr, and "Registration confirmed" still prints, since a push is not required (rules section 4
step 3). Warnings come in the order `title_changed`, `dependency_not_done`, then the push warning (`no_remote`,
`push_failed`, `push_interrupted`); JSON `warnings` keep that order.

`--json` data: `{"id": "W-001", "record": { }, "commit": "e06b95860a6082ae37ef08874f8eb5fade2549da", "pushed":
true}`.

`--dry-run` runs the same checks and writes nothing; it reports `title_changed` and `dependency_not_done` on stderr
and in JSON `warnings`, never a push warning, and prints no registry line, confirmation, or worktree command:

```
$ solomonar task register --title "Login" --owner orchestrator-1 --dry-run
Would register W-001: Login
  status    reserved / setup
  owner     orchestrator-1
  branch    feature/W-001 (create it now)
  next      orchestrator-1: Create feature/W-001 and its worktree
  state(W-001): reserved/setup, registered by orchestrator-1
```

`--dry-run --json` data: `{"id": "W-001", "record": { }, "dry_run": true}`.

### UC-13 Advance a task through its stages (owner)

At least one field flag is required (UC-14); `--as`, `--dry-run`, and `--resume-cancelled` are not field flags.
All flags apply as one validated update and one commit. `done` and `complete` are not accepted here: only `task
done` writes them.

The header is `<ID>  <from status/stage> → <to>`, or `<ID>  <status>/<stage>` when the position does not change.
A detail line follows for every changed field other than status and stage, in record order, labels aligned two
spaces past the longest label: `confirmed yes` or `confirmed no`, `base <sha> (on feature/<ID>)`, `worktree <path>
(listed by git worktree list)`, `spec <path> (found in <worktree>)`, `revision <sha> (on feature/<ID>)`, `tier <n>:
<meaning>` (0 `no review stages`, 1 `specialists only`, 2 `specialists, then adversarial`), `agents <id> <role>, …`
or `agents none`, `next <actor>: <action>`, `pr <value>`; then the subject and the registry line. The `spec` line
reads `(found in <worktree>)` when the spec is in the worktree, and `(found at <short revision>)` when it is found
only at the recorded revision.

Leaving `setup` (`--status running --stage specification`) also records `confirmed`, `base` (`git merge-base <main>
feature/<ID>`), and `worktree` (the path Git lists for `feature/<ID>`, relative to the main worktree, the first
entry of `git worktree list`), so its output lists them before `next`:

```
$ solomonar task move W-001 --as orchestrator-1 --status running --stage specification --next "orchestrator-1:Run work-spec"
W-001  reserved/setup → running/specification
  confirmed  yes
  base       5e1f0c3a7b9d2e4f6a8c0b1d3e5f7a9c2b4d6e8f (on feature/W-001)
  worktree   ../ledger-worktrees/W-001 (listed by git worktree list)
  next       orchestrator-1: Run work-spec
  state(W-001): running/specification, status running, stage specification, confirmed, base 5e1f0c3, worktree ../ledger-worktrees/W-001, next orchestrator-1
  registry @ 9b3d5f7a1c2e4a6b8d0f1e3c5a7b9d2f4e6a8c0b
```

`--base <full-sha>` fills a missing base only (a migrated record, UC-25): the commit must be on `feature/<ID>`
(`base_not_on_branch`), and a different value than the recorded one is `base_recorded`. `--worktree <path>` records
a relocation after `git worktree move`: a clean `/`-separated path relative to the main checkout that must resolve
to the path Git lists for `feature/<ID>` (`worktree_mismatch` otherwise). Both are accepted from `specification` on
and need no confirmation. Every stage change except leaving `setup`, and `task done`, needs a confirmed task
(`confirmation_required`, fix `inspect with solomonar task status <ID>, then solomonar task confirm <ID> --as
<owner>`); status-only moves (block, pause, cancel, resume) do not.

`READY_FOR_REVIEW` at a committed SHA, triage recorded tier 2, specialists launched:

```
$ solomonar task move W-001 --as orchestrator-1 --revision 7cd18e45bb55c3be9465c9f893683bc05216f529 --tier 2 \
    --stage specialized_review --agent sec-1:security-reviewer --agent cor-1:correctness-reviewer \
    --agent spec-1:spec-reviewer --next "orchestrator-1:Collect specialist reports"
W-001  running/implementation → running/specialized_review
  revision  7cd18e45bb55c3be9465c9f893683bc05216f529 (on feature/W-001)
  tier      2: specialists, then adversarial
  agents    sec-1 security-reviewer, cor-1 correctness-reviewer, spec-1 spec-reviewer
  next      orchestrator-1: Collect specialist reports
  state(W-001): running/specialized_review, stage specialized_review, revision 7cd18e4, tier 2, agents sec-1 cor-1 spec-1, next orchestrator-1
  registry @ 2d4f6a8c0e1b3d5f7a9c1e2b4d6f8a0c2e3b5d7f
```

The remaining moves follow the same shape: `--stage adversarial_review --agent adv-1:adversarial-reviewer`;
`--stage implementation` with a correction round; `--stage integration`; `--pr 42`; `--revision` for each closure
commit (rules section 9). `--json` data: `{"id": "W-001", "from": {"status": "running", "stage": "implementation"}, "to":
{"status": "running", "stage": "specialized_review"}, "changed": ["stage", "revision", "tier", "active_agents",
"next_action"], "record": { }, "commit": "<sha>", "pushed": true}`; `changed` lists the fields in record order.
`--dry-run` prints `Would move ` before the header, then the detail lines, the note of a cancel move (UC-18), and
the subject, with no registry line; `--dry-run --json` data is the same without `commit` and `pushed`, with
`"dry_run": true`.

A move whose flags change nothing writes no commit and exits 0: the human output is only the header line
`W-001  running/specification (unchanged)` (`Would move W-001  running/specification (unchanged)` with
`--dry-run`), with no subject or registry line, and `--json` data has `changed: []`, `commit` the current tip, and
`pushed: false`. It still runs every check (ownership, the move rules, and the Git fact checks, so a repeated `--spec`
is verified again) and is rejected like any other move when one fails.

### UC-14 A rejected move

Each rejection exits 3, writes nothing, and names the check and the fix (8.4). Typical cases:

```
$ solomonar task move W-001 --as orchestrator-1 --revision 7cd18e4 --tier 1 --stage specialized_review
error: task move W-001 rejected (revision_not_full_sha)
  check: --revision must be a full 40-character SHA
  found: 7cd18e4
  fix:   git rev-parse 7cd18e4
nothing was written

$ solomonar task move W-001 --as orchestrator-1 --stage adversarial_review
error: task move W-001 rejected (stage_not_allowed)
  check: tier 1 skips adversarial_review (rules section 7)
  found: specialized_review -> adversarial_review
  fix:   --stage integration, or record tier 2 if the triage changed: --tier 2
nothing was written

$ solomonar task move W-001 --stage implementation --as orchestrator-2
error: task move W-001 rejected (owner_mismatch)
  check: only the owner writes a task's record
  found: owner is orchestrator-1; acting as orchestrator-2
  fix:   act as orchestrator-1, or ask the user for an explicit handoff: solomonar task handoff W-001 --owner orchestrator-2
nothing was written
```

Usage errors exit 2 and write nothing: a missing ID is `invalid_usage` (`task ID is required`), and so is a move
without a field flag (`at least one field flag is required`); no `--as` and no `SOLOMONAR_OWNER` is
`owner_required` (`task move needs an acting owner`, fix `pass --as <owner>`).

### UC-15 Wait for a user decision

When an implementer returns `NEEDS_DECISION`, the orchestrator records the blocker, the user as next actor, and
`blocked` once no independent work remains (rules section 3). `--next` and `--status` on `block` and `unblock` are shorthands for the same fields of `task move`, so one decision
event is one commit.

The human output is the `task move` form (UC-13) with a first detail line `blocker B-<n> <kind>: <summary> (<ref>)`
(no parenthetical without a ref) on `block`, or `cleared B-<n> <kind>: <summary>` on `unblock`, then the other
changed fields, labels aligned.

`--json` data has the `task move` shape: `{"id": "W-001", "from": {"status": "running", "stage": "implementation"},
"to": {"status": "blocked", "stage": "implementation"}, "changed": ["status", "blockers", "next_action"], "record":
{ }, "commit": "<sha>", "pushed": true}` for the `task block` example of the command page; `changed` lists `blockers` (never `last_blocker`) in
record order, with `status` and `next_action` when they change. `--dry-run` prints `Would block ` or `Would unblock `
before the header, then the detail lines and the subject, with no registry line; `--dry-run --json` data is the same
without `commit` and `pushed`, with `"dry_run": true`.

Rules: blocker IDs are `B-n`, stable within the task, never reused (the record's `last_blocker` holds the
highest `n` issued); `--kind decision` requires `--ref P/D-n` or `--ref <ID>/D-n`; `--summary` is at most 80 characters;
`block --status` accepts only `blocked` (anything else is `invalid_value`, fix `pass --status blocked, or use task
move`); `--status blocked` requires at least one open blocker after the write; `unblock --status` accepts only the
prior status (8.1); unblocking the last open blocker of a `blocked` task without `--status`, or with `--status
blocked`, is `blockers_required` (exit 3, check `a blocked task needs an open blocker`, found `no open blocker`, fix
`pass --status <prior status>`, e.g. `pass --status running`): the CLI never picks the status. An ID
that is not an open blocker of the task, unknown or malformed, is `blocker_not_found` (exit 3, found `<blocker ID>
is not an open blocker of <ID>`, fix `solomonar task show <ID>`). Both commands check, in order, that the task
exists, `done_final`, and ownership (`--as` / `SOLOMONAR_OWNER`, as `task move`), then apply the blocker change and
the `--status`/`--next` shorthand as one validated write. Rejections read `task block <ID> rejected` and `task
unblock <ID> rejected` (3.6).

Usage errors exit 2 and write nothing, checked before anything is read: a missing ID is `invalid_usage` (`task ID is
required`), and so are `unblock` without the blocker ID (`blocker ID is required`) and `block` without `--kind` or
`--summary` (`--kind is required`, `--summary is required`); no `--as` and no `SOLOMONAR_OWNER` is `owner_required`
(`task block needs an acting owner` or `task unblock needs an acting owner`, fix `pass --as <owner>`).

### UC-16 Inspect state

`task show` without an ID prints one row per task, sorted by ID number. STATUS is as wide as `cancelled`; NEXT shows the `summary` for a `done` row,
`<actor>: <action>` otherwise, and `-` when empty, as do STAGE and OWNER. The last line is `registry <ref> @ <sha7>
· <commit time>`, the time in RFC 3339 UTC. `task show <ID>` prints the stored record exactly as JSON, the raw read
(J3).

`--json` data: `{"registry": {"schema_version": 2, "tasks": {"<ID>": <record>, …}}, "commit": "<sha>"}`, the
registry files (8.6) joined into one document; with an ID, `{"id": "W-001", "record": { },
"commit": "<sha>"}`.

The oracle (rules section 7, J3):

```
$ solomonar task status W-001
W-001  Login
record
  status    running / specialized_review   owner orchestrator-1 (confirmed)   tier 2
  base      5e1f0c3a7b9d2e4f6a8c0b1d3e5f7a9c2b4d6e8f
  worktree  ../ledger-worktrees/W-001
  spec      docs/work/W-001-login.md
  revision  7cd18e45bb55c3be9465c9f893683bc05216f529
  agents    sec-1 security-reviewer, cor-1 correctness-reviewer, spec-1 spec-reviewer (last reported, not liveness)
  blockers  none
  next      orchestrator-1: Collect specialist reports
  pr        none
  updated   2026-10-20T15:12:40Z (registry @ 2d4f6a8)
git
  branch    feature/W-001 @ 992051e56493f5bb8abb4faadcdbc5f271195498
  worktree  /work/ledger-worktrees/W-001
  HEAD      992051e5 · 1 commit ahead of revision
  dirty     1 path: internal/auth/session_test.go (modified)
  spec      present in worktree and at revision
  merged    no
forge (gh)
  pr      none for feature/W-001
  checks  unknown (no PR)
drift
  head_ahead_of_revision  HEAD is 1 commit past the recorded revision
  dirty_worktree          1 uncommitted path; preserve it (rules section 10)
allowed next
  solomonar task move W-001 --as orchestrator-1 --stage implementation --next <actor>:<action>
  solomonar task move W-001 --as orchestrator-1 --stage adversarial_review
  solomonar task block W-001 --as orchestrator-1 --kind <kind> --summary <text>
  solomonar task move W-001 --as orchestrator-1 --status paused --next <actor>:<action>
  solomonar task move W-001 --as orchestrator-1 --status cancelled --next <actor>:<action>
```

The human form has a `<ID>  <title>` header and the sections `record`, `git`, `plan` (not for a `planned` or `done`
record), `forge (gh)`, `drift`, and `allowed next`, each line indented two spaces with its label padded to the widest label of the section. `status` is `<status>
/ <stage>   owner <owner>   tier <tier>` (`-` for a missing stage or owner, `tier none` without a tier); past `setup`
the owner is followed by ` (confirmed)` or ` (unconfirmed)`. `base`, `worktree` (as recorded), `spec`,
`revision`, `agents` (`<id> <role>, …` then `(last reported, not liveness)`; plain `none` when empty), `blockers`
(`B-n <kind>: <summary> (<ref>)`, or `B-n <kind>: <summary>` without a ref, joined by `; `), `next`, and `pr` always
appear, `none` when empty, then `updated   <updated_at> (registry @ <sha7>)`; a `done` record shows `status` (same
format, e.g. `done / complete   owner -   tier none`), `summary`, `spec_ref`, `merge`, `pr`, `updated` instead.
`git`: `branch` is `feature/<ID> @ <tip>` or `feature/<ID> missing`; `worktree` a path or `missing`; `HEAD` `<head8>
· <n> commit(s) ahead of revision`, `<head8>` alone when the revision is unknown or not recorded, or `unknown (no
worktree)`; `dirty` `clean`, `1 path: <p> (<state>)`, `<n> paths: <p> (<state>), …`, or `unknown (no worktree)`;
`spec` `present in worktree and at revision`, `present in worktree only`, `present at revision only`, `missing`, or
`none recorded`; `merged` `yes, by <merge sha> (merge commit on <main>)` whenever the facts have a merge commit,
`no` otherwise; `spec_ref` `derived <spec_ref> (last spec(<ID>) commit merged)` only when derived and the record is
not `done` (a `done` record's recorded `spec_ref` is in the record section). `plan` (8.7): `approval <state> at
<tip8>`, with `: <problem>` when `invalid`, or `unknown (no spec, base, or branch recorded)`; with steps, `steps <n> of
<m> committed`, then `; remaining S-a, S-b` when any remain, `closed S-c dropped, S-d verified` when any step has an
outcome, `other <n> commit(s) without a Step trailer`, and one `<code>  <message>` line per discrepancy. `forge (gh)`: `pr` `#<n> <state
lowercased>`, `none for feature/<ID>`, `unknown (gh unavailable)`, or `unknown (--no-forge)`; `checks` the check
state, or `unknown (no PR)` without a PR. `drift` lists `<code>  <message>` with codes padded; `allowed next` one
command per line, unwrapped; each is `none` when empty.

`--no-forge` skips `gh`. A missing ID is `invalid_usage` (`task ID is required`); an unknown ID is `task_not_found`
as on `task show <ID>`. `SOLOMONAR_OWNER` feeds `owned_by_other`. `forge_unavailable` goes to stderr as a warning;
`--no-forge` warns nothing, and its `--json` forge is `{"available": false, "pr": null, "checks": "unknown"}`. Exit
0 whatever the drift: drift is information, not failure. `--json` data is the 9.3 shape `{"id", "record",
"registry_commit", "facts", "drift", "allowed_next"}`.

### UC-17 Resume after an unexpected session end

A new orchestrator session starts after the previous one merged the PR and ended before `task done`. `task status`
reports `merged_not_done` and lists only `task done` as allowed next, with the merge commit filled in: the example
of [`solomonar task status`](../../site/src/docs/cli/task.md#solomonar-task-status).

The derived `spec_ref` is a cross-check (C14 b); the orchestrator takes `spec_ref` from the PR's Verification
section (C14 a), and `task done` rejects a mismatch.

### UC-18 Pause, cancel, and resume

```
$ solomonar task move W-002 --as orchestrator-2 --status paused --next "orchestrator-2:Resume at specification; draft AC-3 uncommitted"
W-002  running/specification → paused/specification
  next  orchestrator-2: Resume at specification; draft AC-3 uncommitted
  state(W-002): paused/specification, status paused, next orchestrator-2
  registry @ 5b5e0e33f8e29da3ea8d00d07be6c74719046616

$ solomonar task move W-002 --as orchestrator-2 --status running
W-002  paused/specification → running/specification
  state(W-002): running/specification, status running
  registry @ 5ae4ac5f86fa01730a5e37fe7eaa70cc0f5ffdd4

$ solomonar task move W-004 --as orchestrator-4 --status cancelled --next "user:Superseded by W-005; branch and worktree kept"
W-004  running/implementation → cancelled/implementation
  next  user: Superseded by W-005; branch and worktree kept
  note: branch feature/W-004, its worktree, and PR are kept untouched (rules section 10)
  state(W-004): cancelled/implementation, status cancelled, next user
  registry @ 811967098dec97801a7bfb982e1165f9946ff5a6

$ solomonar task move W-004 --as orchestrator-4 --status running --resume-cancelled
W-004  cancelled/implementation → running/implementation
  state(W-004): running/implementation, status running
  registry @ c473d7af3f586cf94fc8d9877fc10a73c9895ab9
warning: resuming a cancelled task requires the user's explicit instruction (rules section 7)
```

The warning goes to stderr. Without a recorded PR the note is `note: branch feature/<ID> and its worktree are kept
untouched (rules section 10)`.

`--status paused` and `--status cancelled` require `--next` (`next_required`); `--status running` on a cancelled
task without `--resume-cancelled` is `cancelled_requires_flag`.

### UC-19 Hand off ownership

The user tells a new session to take over a task after confirming the previous session stopped. `handoff` takes no `--as`: the new owner runs it on the user's instruction (rules section 10). Rejections: `handoff_same_owner`,
`owner_in_use`, `done_final`.

Checks run in order: the lookup (`task_not_found`), `done_final`, `not_registered` (a `planned` task; found `<ID>
is planned`), the `--owner` format (`invalid_value`, as `task register`: an agent ID `[a-z]+-[0-9]+`, fix "pass
--owner orchestrator-1"), `handoff_same_owner` (found `<ID> is owned by <owner>`), `owner_in_use` (the texts of
`task register`), then `--next` (actor `user`, the new owner, or an `active_agents` id). `SOLOMONAR_OWNER` is
ignored. Rejections read `task handoff <ID> rejected` (3.6). Usage errors exit 2 and write nothing, checked before
the registry is read: a missing ID is `invalid_usage` (`task ID is required`), and so is a missing `--owner`
(`--owner is required`).

Every status except `planned` and `done` accepts a handoff, including `cancelled`. Without `--next`, a
`next_action` whose actor is the previous owner is retargeted to the new owner with the same action text; any other
`next_action` is kept. `active_agents` and blockers are kept; `updated_at` is always set. The handoff clears
`confirmed` (CLI-D22): the new owner inspects the task and runs `task confirm` (UC-26) before the next stage change.

A `confirmed no` detail line follows the header when the record was confirmed, then a `next <actor>: <action>` line
when `next_action` changes; the closing lines `The previous owner must stop writing this record.` and `Inspect with
solomonar task status <ID>, then solomonar task confirm <ID> --as <new owner>.` print only after a real write. The subject is always `handoff <old>
-> <new>`, also with `--next`. `--dry-run` prints `Would hand off ` before the header, then the detail lines and the
subject, with no registry line and no closing line.

`--json` data has the `task move` shape: `{"id": "W-002", "from": {"status": "blocked", "stage": "specification"},
"to": {"status": "blocked", "stage": "specification"}, "changed": ["owner"], "record": { }, "commit": "<sha>",
"pushed": true}`; `changed` lists `owner` (in record order, after `stage`), `confirmed` when it was set, and
`next_action` when it changes.
`--dry-run --json` data is the same without `commit` and `pushed`, with `"dry_run": true`.

### UC-20 Complete a task after the merge

`task done` uses only Git (J6). Its checks are in 8.3. The compact record keeps `title`, `status` `done`, `stage`
`complete`, `summary` (trimmed), `spec_ref`, `merge_commit`, `pr` (when recorded), `related_task` (when recorded), and
`updated_at`; every other field leaves the record and stays in registry history. A failed `task done` writes
nothing; the orchestrator fixes the cause and reruns it, never reverting the merge (rules section 9).

Checks run in order: the lookup (`task_not_found`), `done_final`, ownership (`owner_required` / `owner_mismatch`, as
`task move`), status `running` and stage `integration` (`transition_not_allowed`, found `<ID> is <status>/<stage>`,
fix `solomonar task status <ID>`), `blockers_open` (found `<ID> has open blockers B-1, B-2`, fix `solomonar task
unblock <ID> <blocker-id>`), then the flag values (`invalid_value`, exit 2): `--summary` 1 to 120 characters (runes)
after trimming, stored trimmed; `--merge` a full 40-hex SHA (fix `git rev-parse <short>`); `--spec-ref` `<full
SHA>:<path>` with a clean relative path as for `--spec` (fix `copy it from the PR "Final specification" line`).
The Git checks follow in this order: `merge_not_on_main`, `merge_not_merge_commit` (parents from `git rev-list
--parents -n 1 <merge>`), `revision_not_merged`, `spec_ref_invalid`, `spec_ref_not_final`; their found texts are in
8.4. The derived `spec_ref` is `<commit>:<recorded spec>`, where `<commit>` is the first commit in `git log
--topo-order --format="%H %s" <merge>^2` (topological order, not commit date) whose subject (never a body line;
`--grep` is not used) starts with `spec(<ID>):`; `--spec-ref` must name that commit and the recorded `spec`.
Rejections read `task done <ID> rejected` (3.6).

`--as` resolves as on `task move`: `SOLOMONAR_OWNER` is the fallback, and neither is `owner_required` (exit 2).
Usage errors come before the registry is read, in this order: `task ID is required`, `--merge is required`,
`--spec-ref is required`, `--summary is required` (`invalid_usage`, exit 2).

The human output starts with the `Verified` block, its labels aligned: `merge <merge short 8> is a merge commit
reachable from <main>`, `revision <revision short 8> is contained in the merge`, `spec_ref resolves; <spec commit
short 8> is the last spec(<ID>) commit merged`. Then come the `task move` header (UC-13), the subject, and the
registry line, then `Compact record` and the compact record as indented JSON (two-space indent, record field order),
each line prefixed by two spaces. The `Next:` lines follow only when a worktree has `feature/<ID>` checked out:
`Next: delete the worktree only after confirming it holds no uncommitted or unmerged work:` and `  git worktree
remove <path>`. `<path>` is the conventional path in the `task register` form (`../<repo>-worktrees/<ID>` from the
main checkout, absolute from another worktree) when the worktree is there, otherwise the worktree's absolute path.

`--dry-run` prints the `Verified` block, then `Would complete ` before the header, the subject, and `Compact
record` with the record, with no registry line and no `Next:` lines.

`--json` data has the `task move` shape with `record` the compact record: `{"id": "W-001", "from": {"status":
"running", "stage": "integration"}, "to": {"status": "done", "stage": "complete"}, "changed": [...], "record": { },
"commit": "<sha>", "pushed": true}`. `changed` lists, in record order, every field that differs between the record
before and the compact record, whether added, changed, or removed, except the bookkeeping fields `updated_at` and
`last_blocker`; for a record that still lists active agents: `["status", "stage", "owner", "branch", "spec", "revision", "tier",
"active_agents", "next_action", "summary", "spec_ref", "merge_commit"]`. `--dry-run --json` data is the same without
`commit` and `pushed`, with `"dry_run": true`.

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

Each task is its own file (8.6), so a write changes only the files of the records it touches; the other files keep
their blobs. The swap still covers the whole ref, so separate files do not remove contention between writers.

### UC-25 Migrate the registry

`solomonar task migrate [--json]` is the user's command (no `--as`). Every other command that reads the registry
rejects a schema-1 registry, the single `state.json` of CLI 0.1.5 and earlier, with `registry_outdated` (exit 3),
found `registry schema 1, this CLI uses 2`, fix `solomonar task migrate`; `init` keeps such a registry
`unchanged`, and `version` reports it (UC-24).

```
$ solomonar task migrate
Migrated registry to schema 2: 4 tasks, one tasks/<ID>.json each
  unconfirmed  W-001, W-004
  next         inspect each with solomonar task status <ID>, then solomonar task confirm <ID>
  registry @ 4f0c2b7d9e1a3c5f7b9d1e3a5c7f9b1d3e5a7c9f
```

The migration decodes the schema-1 registry strictly (8.6 validation; `registry_invalid` otherwise), writes the
same records as schema-2 files in one compare-and-swap commit `state: migrate schema 1 -> 2` whose parent is the
old tip, and pushes as any write. It copies records unchanged: it never infers a `base`, `worktree`, or
`confirmed`, and a done record without `base` stays valid. `unconfirmed` lists, in ID order, every record with an
owner that is not `done`, is past `setup`, and is not confirmed; the line is omitted when none. A registry already
at schema 2 writes nothing and prints `Registry already at schema 2; nothing to migrate.` (exit 0); a newer one is
`cli_outdated`. A failed migration writes nothing. A schema-1 CLI reading a migrated registry rejects it as
`cli_outdated`, since `state.json` keeps the version; to return to it, point the ref back at the previous commit:
`git update-ref refs/heads/workflow-state refs/heads/workflow-state^` (losing writes made after the migration).

`--json` data: `{"migrated": true, "schema": 2, "tasks": 4, "unconfirmed": ["W-001", "W-004"], "commit": "<sha>",
"pushed": true}`; without a migration `migrated` is `false` and `commit` the current tip.

### UC-26 Confirm a resumed task

`solomonar task confirm <ID> [--as <owner>] [--json] [--dry-run]` records that the owner inspected the task and
continues it from its actual state (CLI-D22). A new session takes a task with `task handoff` (UC-19), which clears
`confirmed`; a migrated task starts unconfirmed (UC-25). Leaving `setup` confirms a task, so the normal flow never
runs this command. Confirmation does not prove plan approval, review success, or that the previous execution stopped.

```
$ solomonar task status W-001 --no-forge
…
drift
  unconfirmed  W-001 is not confirmed by orchestrator-2; inspect, then task confirm
allowed next
  solomonar task confirm W-001 --as orchestrator-2
  …
$ solomonar task confirm W-001 --as orchestrator-2
W-001  running/specification
  state(W-001): running/specification, confirmed by orchestrator-2
  registry @ 6c8e0a2b4d6f8a1c3e5b7d9f0a2c4e6b8d1f3a5c
```

Checks run in order: the lookup (`task_not_found`), `done_final`, `not_registered` (a `planned` task), ownership
(`owner_required`, `owner_mismatch`), `setup_not_complete` (stage `setup`; fix `solomonar task move <ID> --as
<owner> --status running --stage specification`), then the preflight: `branch_missing`, `worktree_unrecorded` (fix
`solomonar task move <ID> --as <owner> --worktree <path relative to the main checkout>`), `worktree_missing` or
`worktree_mismatch` (8.3), `base_missing` (fix `solomonar task move <ID> --as <owner> --base <full sha>, after
inspecting the branch history`), `base_not_on_branch`, and, from `implementation` until `integration` (closure deletes
the work spec there), the approved plan at the branch tip (`plan_not_approved`, `plan_invalid`). An already confirmed task is unchanged: the header ends
with ` (unchanged)`, no commit is written, and `--json` has `changed: []` and the current tip, as on `task move`.
`--json` and `--dry-run` follow `task move` (UC-13); `changed` is `["confirmed"]`.

## 7. Use cases: runtime integration

### UC-22 The state guard denies a direct registry write

Deferred (CLI-D36): `guard` is not implemented, and no current adapter registers its hook. Once it ships, the
Claude Code adapter registers a `PreToolUse` hook that runs `solomonar guard --runtime claude`. The guard reads
the tool call from stdin and answers in the runtime's hook format. It denies, with a reason that names the right
command:

- direct writes to Git's resolved `refs/heads/workflow-state` storage, its reflog, or `packed-refs`, including
  those under the common Git directory of a linked worktree; use `git rev-parse --git-path` to resolve them;
- shell commands that check out, switch to, reset, commit to, `update-ref`, or push `workflow-state`, other than
  through `solomonar`;
- `git worktree add` for a `feature/<ID>` branch whose task is not registered (rules section 4: register first);
- `solomonar extension add` or `extension remove` of a stack-group extension without `--decision`: from an agent
  session the change needs an approved decision entry (rules section 3); only the user's own command records a `P/D-n`.

Synopsis and an example denial reason: [`solomonar guard`](../../site/src/docs/cli/guard.md). It never writes; an
allowed call produces no output and exit 0. The guard is a convenience: the task commands validate every write
regardless, and the rules stay behavioral contracts (rules section 11).

The registry is a Git blob with no filesystem checkout. A product file or fixture such as
`internal/testdata/state.json`, or a product's root `state.json`, is not the registry and is allowed by this
guard. Read-only Git commands that inspect `workflow-state` are allowed. Recognize the mutating operation,
not just a token or basename; use real runtime payloads to test both denials and allowed calls.

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

Output inside and outside a project: [`solomonar version`](../../site/src/docs/cli/version.md); help:
[`solomonar help`](../../site/src/docs/cli/help.md).

The CLI version is the release tag the binary was built from, rendered into the embedded framework (AD-10), so `cli` and `framework` in `--json` are always equal. `version` reads only the version fields of the
manifest and the registry, never decoding either strictly, so a newer project prints its versions and exits 0
(CLI-D19). It discovers the repository from `-C` like other commands: outside a repository, or inside one without
`.solomonar/manifest.json`, the `project` line reads `not initialized`; `git_too_old` and other discovery errors stay
errors. With a manifest: `registry missing` replaces `registry schema <n>` when the registry ref is absent; an
unreadable manifest version makes the line only `manifest invalid (inspect .solomonar/manifest.json)`, skipping the
registry, and an unreadable registry version reads `· registry invalid`; each case exits 0. A manifest read failure
other than absence, a failure opening the worktree, and a registry commit whose tree has no `state.json` are
unexpected errors (exit 1), not `invalid`. A suffix follows whichever registry part is shown (`registry schema <n>`,
`registry missing`, or `registry invalid`): a project framework newer than the embedded one adds ` (newer than this
CLI; install solomonar <project fw> or later)`; otherwise a manifest or registry schema newer than this CLI supports
adds ` (newer than this CLI; install a newer solomonar release)`, and a registry schema older than this CLI's adds `
(older than this CLI; run solomonar task migrate)` (UC-25). `--json`: `{"cli": "X.Y.Z", "framework": "X.Y.Z",
"project_framework": "X.Y.Z", "registry_schema": 2}`; a value not shown (not initialized, missing, invalid) is `null`,
so an invalid manifest leaves both project values `null`.

## 8. Reference: task validation

### 8.1 Status transitions

| From | Allowed to | Command | Condition |
|---|---|---|---|
| (none) | `planned` | `task plan` | - |
| (none), `planned` | `reserved` | `task register` | owner not in use |
| `reserved` | `running`, `blocked`, `paused`, `cancelled` | `task move` / `task block` | `running` only with the stage move out of `setup`; `blocked` needs an open blocker; `paused` and `cancelled` need `--next` |
| `running` | `blocked`, `paused`, `cancelled` | `task move` / `task block` | as above |
| `running` | `done` | `task done` only | 8.3 |
| `blocked`, `paused` | the prior status, `cancelled` | `task move` / `task unblock` | the prior status is derived from the record: `reserved` at stage `setup`, `running` at any later stage |
| `cancelled` | the prior status | `task move --resume-cancelled` | explicit flag; prior status as above |
| `done` | none | - | `done_final` |

Invariant (CLI-D16): a task is `reserved` exactly while at stage `setup`. A write that leaves `running` at `setup`
or `reserved` past it is `transition_not_allowed`, with the fix `--status running --stage specification`.

### 8.2 Stage transitions

A stage change needs status `running` after the write (`reserved` only at `setup`); a `blocked` or `paused` task
keeps its stage (rules section 7).

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
| `--revision` | `git merge-base --is-ancestor <sha> feature/<ID>`; a SHA Git does not know fails too | `revision_not_on_branch` |
| `--stage specification` from `setup` | `feature/<ID>` exists and a worktree has it checked out (`git worktree list --porcelain`); `git merge-base <main> feature/<ID>` finds the base | `branch_missing`, `worktree_missing`, `base_not_found` |
| `--base` | full SHA; equal to the recorded base when one exists; `git merge-base --is-ancestor <base> feature/<ID>` | `invalid_value`, `base_recorded`, `base_not_on_branch` |
| `--worktree` | clean `/`-separated relative path (`..` allowed); resolved against the main worktree, it is the path Git lists for `feature/<ID>` (symbolic links resolved) | `invalid_value`, `worktree_mismatch` |
| a stage change other than leaving `setup`, `task done` | the record is `confirmed` | `confirmation_required` |
| `--stage implementation` from `specification` | the spec committed at the tip of `feature/<ID>` (not the worktree file) parses (8.7), reads `Status: Approved`, and lists a step | `plan_not_approved`, `plan_invalid` |
| a move from `implementation` to `specialized_review` or `integration` | `base` recorded; `--revision` (or the recorded revision) is the tip of `feature/<ID>`; `git status --porcelain` in its worktree is empty; the plan at the revision is approved; reconciliation (8.7) finds no discrepancy and every step committed or closed by an outcome | `base_missing`, `candidate_not_tip`, `candidate_dirty`, `plan_not_approved`, `plan_invalid`, `checkpoint_discrepancy`, `plan_incomplete` |
| `task confirm` | branch, recorded worktree as for `--worktree`, recorded base as for `--base`; from `implementation` up to, not including, `integration`, the approved plan at the branch tip | `branch_missing`, `worktree_unrecorded`, `worktree_missing`, `worktree_mismatch`, `base_missing`, `base_not_on_branch`, `plan_not_approved`, `plan_invalid` |
| `--spec` | clean path relative to the worktree: not absolute, no `..` element, not `.`, already in clean form (`./a`, `a//b`, `a/`, `a/../b` are rejected) | `invalid_value` |
| `--spec` given, or the move enters `implementation` | `spec` is a regular file in the worktree on `feature/<ID>` (located by `git worktree list --porcelain`), or a blob at the recorded `revision` (`git cat-file -t <revision>:<spec>` is `blob`; a directory does not count) | `spec_not_found` |
| `--agent` | `<id>:<role>` split at the first `:`, id `[a-z]+-[0-9]+`, a core delegated role or an extension agent installed in the manifest, never an available one or `orchestrator`; without a manifest, the core agents `adversarial-reviewer`, `correctness-reviewer`, `implementer`, `security-reviewer`, `spec-reviewer`; the fix lists the allowed roles | `invalid_value` |
| `--next` | `<actor>:<action>` split at the first `:`, actor `user`, the owner, or an `active_agents` id after the move, so an agent added by `--agent` in the same `task move` may act (`user` or the owner on `task register`); the fix lists the allowed actors; action 1 to 120 characters (runes) after trimming, on every command that takes `--next`, including `task register` | `invalid_value` |
| `task register --owner` | agent ID `[a-z]+-[0-9]+` | `invalid_value` |
| `task plan --title`, `task register --title` | not blank or whitespace-only | `invalid_value` |
| `--pr` | a positive PR number or an `https://` URL, stored as given | `invalid_value` |
| `task done` | status `running`, stage `integration`, then no open blocker, checked before the flag values | `transition_not_allowed`, `blockers_open` |
| `task done --summary` | 1 to 120 characters (runes) after trimming; stored trimmed | `invalid_value` |
| `task done --merge` | full 40-hex SHA | `invalid_value` |
| `task done --spec-ref` | `<full SHA>:<path>`, the path clean and relative as for `--spec` | `invalid_value` |
| `task done --merge` | `git merge-base --is-ancestor <merge> <main>` against the local main branch; a SHA Git does not know fails too | `merge_not_on_main` |
| `task done --merge` | two or more parents, read with `git rev-list --parents -n 1 <merge>` | `merge_not_merge_commit` |
| `task done` | recorded `revision` is an ancestor of the merge commit | `revision_not_merged` |
| `task done --spec-ref` | `git cat-file -e <spec_ref>` succeeds | `spec_ref_invalid` |
| `task done --spec-ref` | the commit is the first line of `git log --topo-order --format="%H %s" <merge>^2` (topological order, not commit date) whose subject starts with `spec(<ID>):`; only subjects count, never body lines (`--grep` is not used); and the path equals the recorded `spec` | `spec_ref_not_final` |

For `task done`, exit 1 or 128 from `git merge-base --is-ancestor` or `git cat-file -e` fails the check (the
rejection) unless the command was cancelled; any other Git failure is `unexpected_error`.

### 8.4 Error codes (task commands)

| Code | Exit | Meaning | Fix shown |
|---|---|---|---|
| `not_a_repository` | 3 | `-C` or the working directory is not in a Git repository | run inside a Git repository or pass `-C <path>` |
| `not_initialized` | 3 | No `refs/heads/workflow-state` | `solomonar init` |
| `registry_invalid` | 1 | A registry file fails the schema (8.6) | inspect with `git show refs/heads/workflow-state:state.json`; never edit it by hand |
| `manifest_invalid` | 1 | `.solomonar/manifest.json` fails strict decoding or the 4.7 validation | inspect .solomonar/manifest.json; never edit it by hand |
| `task_not_found` | 3 | Unknown ID | `solomonar task show` |
| `not_planned` | 3 | `register <ID>` on a non-planned record | `solomonar task show <ID>; resume the task instead` |
| `not_registered` | 3 | `task handoff` on a `planned` task; found `<ID> is planned` | `solomonar task register <ID> --title <title> --owner <new-owner>` |
| `owner_required` | 2 | No `--as` and no `SOLOMONAR_OWNER` | pass `--as <owner>` |
| `owner_mismatch` | 3 | Acting owner differs from the record | act as the owner, or `task handoff` on the user's instruction |
| `owner_in_use` | 3 | The owner owns a `reserved`, `running`, `blocked`, or `paused` task | choose a new owner ID |
| `transition_not_allowed` | 3 | Status change not in 8.1; on `task done`, found `<ID> is <status>/<stage>` | names the allowed targets; on `task done`, `solomonar task status <ID>` |
| `stage_not_allowed` | 3 | Stage change not in 8.2 or not allowed by the tier | names the allowed stages |
| `field_not_allowed` | 3 | A field flag outside the stages that accept it (8.2 field rules) | names the stages that accept it |
| `tier_required` | 3 | A stage after `implementation` without a tier | pass `--tier` with `--revision` |
| `revision_required` | 3 | Review stage without a revision | pass `--revision` |
| `revision_not_full_sha` | 3 | Short or malformed SHA | `git rev-parse <short>` |
| `branch_missing` | 3 | `feature/<ID>` does not exist | create the branch and worktree (rules section 4) |
| `confirmation_required` | 3 | A stage change or `task done` on an unconfirmed task; found `<ID> is not confirmed by <owner>` | `inspect with solomonar task status <ID>, then solomonar task confirm <ID> --as <owner>` |
| `plan_not_approved` | 3 | The committed work spec is not `Status: Approved`; found `status is "<value>" at <sha8>`, or `<spec> is not committed at <sha8>` | record the user's approval as `Status: Approved` below the title and commit the work spec |
| `plan_invalid` | 3 | The work spec's Status line or plan does not parse (8.7), or lists no step | correct the work spec and commit it |
| `candidate_not_tip` | 3 | The candidate is not the tip of `feature/<ID>`; found `revision <sha8>; feature/<ID> is at <sha8>` | pass `--revision` with the full SHA of the branch tip |
| `candidate_dirty` | 3 | The `feature/<ID>` worktree has uncommitted changes; found `<n> uncommitted path(s)` | commit the work as a checkpoint, or preserve it outside the candidate (rules section 10) |
| `plan_incomplete` | 3 | Steps without a checkpoint commit or outcome; found `S-2, S-3 have no checkpoint commit` | delegate the remaining steps, or record an outcome in the plan |
| `checkpoint_discrepancy` | 3 | A commit claims a step it does not checkpoint; found `<n> discrepancies; first: <message>` | inspect with `solomonar task status <ID>`; correct the commits or the plan |
| `setup_not_complete` | 3 | `task confirm` at stage `setup` | `solomonar task move <ID> --as <owner> --status running --stage specification` |
| `base_not_found` | 3 | Leaving `setup` without a common commit of main and `feature/<ID>` | create `feature/<ID>` from the current main (rules section 4) |
| `base_recorded` | 3 | `--base` differs from the recorded base | omit `--base`; a wrong base needs the user's decision |
| `base_missing` | 3 | `task confirm` without a recorded base | `solomonar task move <ID> --as <owner> --base <full sha>`, after inspecting the branch history |
| `base_not_on_branch` | 3 | The base is unknown to Git or not on `feature/<ID>` | inspect the branch history |
| `worktree_unrecorded` | 3 | `task confirm` without a recorded worktree | `solomonar task move <ID> --as <owner> --worktree <path>` |
| `worktree_mismatch` | 3 | Git lists the `feature/<ID>` worktree elsewhere; found `recorded <path>; Git lists <path>` | `record the relocation: solomonar task move <ID> --worktree <listed path>` |
| `worktree_missing` | 3 | `feature/<ID>` exists but no worktree has it checked out | `git worktree add ../<repo>-worktrees/<ID> feature/<ID>`, `<repo>` the main checkout's directory name |
| `revision_not_on_branch` | 3 | SHA unknown to Git or not reachable from `feature/<ID>` | commit the candidate on `feature/<ID>` and pass its full SHA: `git -C <worktree path> rev-parse HEAD`, or `git rev-parse feature/<ID>` without a worktree |
| `spec_required` | 3 | `implementation` without `spec` | pass `--spec` |
| `spec_not_found` | 3 | The spec is neither a regular file in the `feature/<ID>` worktree nor a blob at the recorded revision; found names where it looked (`<spec> is not in the feature/<ID> worktree or at the recorded revision`, or only the one present), or `no worktree on feature/<ID> and no recorded revision to look for <spec>` | commit the work spec in the worktree |
| `next_required` | 3 | Pause or cancel without `--next` | pass `--next <actor>:<action>` |
| `blockers_required` | 3 | `blocked` without an open blocker, including after the last `unblock` | `task block` first, or pass `--status blocked` to it; on `unblock`, pass `--status <prior status>` |
| `blocker_not_found` | 3 | The ID is not an open blocker of the task (unknown or malformed); found `<blocker ID> is not an open blocker of <ID>` | `solomonar task show <ID>` |
| `blockers_open` | 3 | `task done` with open blockers; found `<ID> has open blockers B-1, B-2` | `solomonar task unblock <ID> <blocker-id>` |
| `cancelled_requires_flag` | 3 | Resuming a cancelled task without the flag | `--resume-cancelled`, on the user's instruction |
| `handoff_same_owner` | 3 | Handoff to the current owner; found `<ID> is owned by <owner>` | - |
| `done_final` | 3 | Any write to a `done` task, checked before ownership | register a related task (`--related`) |
| `merge_not_on_main` | 3 | Merge commit unknown to Git or not reachable from the local main branch; found `<merge short 8> is not reachable from <main>` | `fetch and update <main>, then rerun` |
| `merge_not_merge_commit` | 3 | The commit has fewer than two parents (squash or rebase); found `<merge short 8> has one parent`, or `<merge short 8> has no parent` for a root commit | `spec_ref` cannot survive; report to the user (rules section 11) |
| `revision_not_merged` | 3 | The reviewed revision is not in the merge; found `<revision short 8> is not contained in <merge short 8>` | check that the right PR merged |
| `spec_ref_invalid` | 3 | `spec_ref` does not resolve; found `<spec_ref> does not resolve` | copy it from the PR "Final specification" line |
| `spec_ref_not_final` | 3 | Not the last `spec(<ID>)` commit merged, or the path differs from the recorded `spec`; found `<commit short 8>:<path> is not the last spec(<ID>) commit merged`, or `no spec(<ID>) commit is reachable from <merge short 8>^2` when none exists | `pass --spec-ref <derived full SHA>:<recorded spec>`; none when no `spec(<ID>)` commit exists |
| `invalid_value` | 2 | Malformed flag value | names the expected format |
| `unknown_command` | 2 | No command, a command group alone, or an unknown command or subcommand | run solomonar help |
| `invalid_usage` | 2 | Unknown flag, missing required flag, or unexpected argument | run solomonar help |
| `cas_exhausted` | 4 | Lost the compare-and-swap 5 times | retry |
| `cli_outdated` | 3 | Registry or manifest written by a newer CLI | install a newer solomonar release |
| `registry_outdated` | 3 | Registry at schema 1 (UC-25) | `solomonar task migrate` |
| `git_too_old` | 1 | Git older than 2.40 | install git 2.40 or later |
| `unexpected_error` | 1 | I/O, Git error, corrupt object (3.5) | - |
| `interrupted` | 130, 143 | Signal before the compare-and-swap | rerun (the error form already states nothing was written) |

Warnings (exit 0): `push_failed`, `push_interrupted`, `no_remote`, `dependency_not_done`, `title_changed`, `forge_unavailable`
(`task status`, 9.1), `cancelled_resumed`.

### 8.5 Registry commits

One commit per successful command, subject `state(<ID>): <status>/<stage>, <summary>` (rules section 7; the `/<stage>`
part is omitted while a planned record has no stage), author and committer from the user's Git identity, parent the
previous tip.

| Command | Summary part |
|---|---|
| `task plan` | `plan "<title>"` |
| `task register` | `registered by <owner>` |
| `task move` | every changed field, comma-separated, in record order (the order of the JSON `changed` list): `status <status>`, `stage <stage>`, `spec <path>`, `revision <short 7>`, `tier <n>`, `agents <id> <id>…` (space-separated) or `agents none` when cleared, `next <actor>`, `pr <value as stored>`; `confirmed`, `base <short 7>`, and `worktree <path>` in record order after `stage` |
| `task block` | `block B-n <kind>` |
| `task unblock` | `unblock B-n` |
| `task handoff` | `handoff <old> -> <new>`, also with `--next` |
| `task confirm` | `confirmed by <owner>` |
| `task done` | `merged <short merge sha>` |
| `init` | subject `state: initialize registry` (root commit) |
| `task migrate` | subject `state: migrate schema 1 -> 2` |

### 8.6 Registry layout

Schema 2 (CLI-D20). The tree at the tip of `refs/heads/workflow-state` holds exactly:

| Path | Content |
|---|---|
| `state.json` | `{"schema_version": 2}`, nothing else |
| `tasks/<ID>.json` | one record (rules section 7), the file name its ID (`W-` and three or more digits) |

An empty registry is `state.json` alone. Every file is written in canonical form: struct field order, two-space
indent, trailing newline. Reading checks `state.json`'s `schema_version` first (`cli_outdated` above 2,
`registry_outdated` at 1), then decodes every file strictly: unknown members, duplicate keys, a `tasks` member in
`state.json`, any other path, a subtree other than `tasks`, or a record failing validation is `registry_invalid`.

Record fields added by schema 2 (CLI-D21):

| Field | Value | Written by |
|---|---|---|
| `confirmed` | `true`; omitted when false | the move out of `setup`, `task confirm`; cleared by `task handoff` |
| `base` | full SHA of the commit the task branched from; never changed once recorded | the move out of `setup`, or `--base` when missing |
| `worktree` | clean `/`-separated path relative to the main checkout, `..` allowed, never absolute | the move out of `setup`, or `--worktree` after a relocation |

A `planned` record has none of them. A `done` record keeps `base` when it had one and drops the other two.

### 8.7 Work-spec plan and checkpoints

The work spec (`docs/work/<ID>-<slug>.md`, CLI-D23) carries the plan approval on the line below the title and the
plan as a list of intended commits:

```
# W-001: Login
Status: Approved
Type: feature
…
## Implementation plan

- S-1: feat(W-001): add the session store
  Store sessions server-side; AC-1, AC-2.
- S-2: feat(W-001): add remember me
  Outcome: dropped — the user removed AC-4 (W-001/D-2)
- S-3: test(W-001): cover session expiry
```

- `Status:` is `In review` or `Approved`; any other value is invalid, and a spec without the line has no approval.
- A step is a line `- S-<n>: <commit subject>` under `## Implementation plan`, up to the next `##` heading; `<n>` has
  no leading zero, IDs are unique and never reused. Indented lines are its detail; an indented `Outcome: dropped —
  <reason>` or `Outcome: verified — <reason>` closes the step without a commit. Any other `- ` line there is invalid.
- Framework checkpoint subjects use `<type>(<ID>): <summary>` (rules section 5). The parser accepts any
  non-empty subject; the CLI checks exact matching, while the framework governs type and task-ID scope.
- A checkpoint is a commit whose subject equals the step's subject and whose trailers include `Step: <ID>/S-<n>`
  (CLI-D24), for example `git commit -m "feat(W-001): add the session store" -m "Checks: go test ./... passed." -m
  "Step: W-001/S-1"`. The CLI never creates commits (CLI-D25).

Reconciliation reads the spec committed at the branch tip (or, at the review gate, at the candidate) and the commits
of `git log --first-parent --no-merges --reverse <base>..<tip>`: only the task branch's own commits count, never a
merge or the commits a merge from main brought in (other tasks' checkpoints among them). Per commit, oldest first:

| Case | Result |
|---|---|
| no `Step` trailer, subject unlike any step | counted in `other_commits` (spec, decisions, closure commits) |
| no `Step` trailer, subject equal to a step's | `step_untagged` |
| trailer of another task | `step_foreign` |
| trailer naming a step the plan does not list | `step_unknown` |
| subject differs from the claimed step's | `step_subject_mismatch` |
| no changed file (`--shortstat` empty) | `step_empty` |
| a step already checkpointed by an earlier commit | `step_duplicate` |
| otherwise | the step is `committed` by that commit |

A step is `committed`, else `dropped` or `verified` from its outcome, else `remaining`. Approval is `approved` only
for `Status: Approved` with at least one step, `in_review`, `missing` (no Status line, no step, or no spec at the
commit), or `invalid` with the parse problem. A commit is never proof of correctness: review and the required checks
still run. A checkpoint commit and a registry write are separate operations; a checkpoint committed before a failed
registry update is found again by the next reconciliation, since the registry holds no checkpoint state.

## 9. Reference: `task status`

### 9.1 Facts

| Fact | Source | When unavailable |
|---|---|---|
| branch, tip | `git rev-parse feature/<ID>` | `exists` false, `tip` null |
| branch contains `revision` | `git merge-base --is-ancestor <revision> feature/<ID>` | null without a revision or branch |
| worktree | `git worktree list --porcelain`, by branch | `worktree` null |
| HEAD vs `revision` | `git rev-list --count <revision>..HEAD` in the worktree | null without a worktree or revision |
| dirty paths | `git status --porcelain=v2` in the worktree | `worktree` null |
| spec present | the worktree file, and `git cat-file -t <revision>:<spec>` is `blob` | false |
| plan | the spec at the branch tip and `git log --first-parent --no-merges --reverse --shortstat <base>..<tip>` with `Step` trailers (8.7) | `plan` null without a spec, base, or branch, and for `done` |
| merged | first merge commit on main whose ancestry contains `revision` (`git rev-list --ancestry-path --merges --reverse <revision>..main`) | `merged` false, also without a revision |
| derived `spec_ref` | first commit in `git log --topo-order --format="%H %s" <merge>^2` whose subject (never a body line) starts with `spec(<ID>):`, with the recorded `spec` path | null before merge |
| PR, checks | `gh pr view feature/<ID> --json number,url,state,mergeCommit,statusCheckRollup` | `pr` null, `checks` `unknown`, never passing (J6) |

Branch containment feeds the drift `revision_not_on_branch` (9.2). A recorded `revision` Git does not know (exit
128, e.g. rewritten or garbage-collected history) is not an error: `contains_revision` and `at_revision` are false,
`ahead_of_revision` is null, and `merged` is false. A `done` record's merge facts come from its recorded
`merge_commit` and `spec_ref`, not from a search: `merged` is true, and an empty `merge_commit` or `spec_ref` gives
null for that fact.

Dirty paths come from `git status --porcelain=v2` entries, in Git's order, with paths as Git prints them (quoting
included); entries other than `1`, `2`, `u`, and `?` (ignored ones among them) are skipped. States: `modified`,
`added`, `deleted`, `type_changed` (entries `1`, the worktree side of XY, else the index side), `copied` (entries
`2` whose XY contains `C`) and otherwise `renamed` (entries `2`), both with the new path, `untracked` (`?`), and
`unmerged` (`u`).

`checks` summarizes `statusCheckRollup`: `none` when it is empty, otherwise the first match wins:

- `failing`: any check concluded `FAILURE`, `CANCELLED`, `TIMED_OUT`, `ACTION_REQUIRED`, `ERROR`, or
  `STARTUP_FAILURE` (a check run's `conclusion`, a status context's `state`).
- `pending`: any check not completed: a check run whose `status` is not `COMPLETED`, or a status `PENDING` or
  `EXPECTED`.
- `unknown`: any completed check outside `SUCCESS`, `NEUTRAL`, and `SKIPPED`, or an entry with neither `status` nor
  `state`; also no PR, no `gh`, and `--no-forge`.
- `passing`: every check is `SUCCESS`, `NEUTRAL`, or `SKIPPED`; never reached through an unrecognised value (J6).

The forge is unavailable when `gh pr view` cannot be used: warning `forge_unavailable` (exit 0), message
`gh unavailable: gh not found` when `gh` is not on `PATH`, `gh unavailable: <first stderr line>` when it fails,
`gh unavailable: exit status <n>` when it fails with empty stderr, and `gh unavailable: unreadable gh pr view
output` for malformed JSON. Stderr containing "no pull requests found" is not a failure, whatever the exit code:
the forge is available with no PR. A cancelled run (e.g. Ctrl-C) is no warning: the command fails with
`interrupted`. `--no-forge` skips `gh` and warns nothing; the forge facts are those of an unavailable forge.

### 9.2 Drift

| Code | Condition | Message | Suggested command |
|---|---|---|---|
| `branch_missing` | status `running` past `setup`, no `feature/<ID>` | `feature/<ID> does not exist` | `task block --kind environment` |
| `worktree_missing` | branch exists, no worktree | `feature/<ID> has no worktree` | `git worktree add ../<repo>-worktrees/<ID> feature/<ID>` |
| `worktree_mismatch` | recorded `worktree` differs from the listed one, relative to the main worktree | `recorded <path>; Git lists <path>` | `task move --worktree <listed path>` |
| `base_missing` | past `setup`, no `base` | `<ID> has no recorded base` | `task move --base <full sha>` after inspection |
| `unconfirmed` | past `setup`, not `confirmed` | `<ID> is not confirmed by <owner>; inspect, then task confirm` | `task confirm` |
| `head_ahead_of_revision` | HEAD has commits after `revision` | `HEAD is <n> commits past the recorded revision` | record the new candidate with `task move --revision` once reported |
| `revision_not_on_branch` | `contains_revision` false (history rewritten) | `<revision short 8> is not on feature/<ID>` | `task move --stage implementation`; reviews rerun |
| `dirty_worktree` | uncommitted changes | `<n> uncommitted paths; preserve them (rules section 10)` | preserve them (rules section 10) |
| `spec_missing` | recorded `spec` neither in the worktree nor at `revision` | `<spec> is not in the worktree or at the revision` | commit the work spec, or correct `--spec` |
| `plan_not_approved` | stage `implementation`, `specialized_review`, or `adversarial_review`; plan approval not `approved` | `the work spec at feature/<ID> has no approved plan (<approval>)` | approve and commit the plan, or return to `specification` |
| `checkpoint_discrepancy` | reconciliation found discrepancies | `<n> commit claim(s) to inspect` | inspect the `plan` section |
| `blocked_without_blockers` | status `blocked`, no open blocker | `blocked without an open blocker` | `task move --status <prior>` (guidance only; not in `allowed_next`) |
| `tier_missing` | stage `specialized_review`, `adversarial_review`, or `integration`, no tier | `stage <stage> needs a tier` | `task move --tier` with `--revision` |
| `merged_not_done` | merge found, status not `done` | `feature/<ID> is merged into <main> but the record is not done` | `task done` with `--merge`; `spec_ref` from the PR (C14 a) |
| `pr_not_recorded` | `gh` shows a PR, record has none | `gh shows PR #<n> for feature/<ID>; the record has none` | `task move --pr <url>` |
| `pr_mismatch` | recorded PR differs from the branch's PR | `the record has PR <pr>; gh shows #<n> for feature/<ID>` | `task move --pr <url>` |
| `owned_by_other` | `SOLOMONAR_OWNER` set and differs from the owner | `SOLOMONAR_OWNER is <x>; <ID> is owned by <owner>` | stop; handoff only on the user's instruction |

Drift rows apply to `reserved`, `running`, `blocked`, and `paused` records, except `owned_by_other`, which applies
to every status whenever the record has an owner; `planned`, `cancelled`, and `done` get only `owned_by_other`.
Entries appear in table order. With one, the messages read `HEAD is 1 commit past the recorded revision` and `1
uncommitted path; preserve it (rules section 10)`. A recorded `pr` equal to the PR's number or URL is no `pr_mismatch`.
`<revision short 8>` is the first 8 characters of the recorded `revision`; `<main>` is the main branch name.

Each drift entry carries only `code` and `message`. The suggested command is guidance for the orchestrator, not
output; runnable commands appear in `allowed_next` (9.3).

### 9.3 JSON shape

```json
{
  "id": "W-001",
  "record": { },
  "registry_commit": "2d4f6a8c0e1b3d5f7a9c1e2b4d6f8a0c2e3b5d7f",
  "facts": {
    "branch": {"name": "feature/W-001", "exists": true, "tip": "992051e56493f5bb8abb4faadcdbc5f271195498",
               "contains_revision": true},
    "worktree": {"path": "/work/ledger-worktrees/W-001", "relative": "../ledger-worktrees/W-001",
                 "head": "992051e56493f5bb8abb4faadcdbc5f271195498",
                 "ahead_of_revision": 1, "dirty": [{"path": "internal/auth/session_test.go", "state": "modified"}]},
    "spec": {"in_worktree": true, "at_revision": true},
    "plan": {"approval": "approved", "at": "992051e56493f5bb8abb4faadcdbc5f271195498", "other_commits": 2,
             "steps": [{"id": "S-1", "subject": "feat(W-001): add the session store", "state": "committed",
                        "commit": "7cd18e45bb55c3be9465c9f893683bc05216f529"},
                       {"id": "S-2", "subject": "test(W-001): cover expiry", "state": "remaining", "commit": null}],
             "discrepancies": []},
    "merge": {"merged": false, "merge_commit": null, "derived_spec_ref": null},
    "forge": {"available": true, "pr": null, "checks": "unknown"}
  },
  "drift": [
    {"code": "head_ahead_of_revision", "message": "HEAD is 1 commit past the recorded revision"},
    {"code": "dirty_worktree", "message": "1 uncommitted path; preserve it (rules section 10)"}
  ],
  "allowed_next": [
    "solomonar task move W-001 --as orchestrator-1 --stage implementation --next <actor>:<action>",
    "solomonar task move W-001 --as orchestrator-1 --stage adversarial_review",
    "solomonar task block W-001 --as orchestrator-1 --kind <kind> --summary <text>",
    "solomonar task move W-001 --as orchestrator-1 --status paused --next <actor>:<action>",
    "solomonar task move W-001 --as orchestrator-1 --status cancelled --next <actor>:<action>"
  ]
}
```

Facts never omit a key; an unavailable fact is null, `false`, or `unknown` as 9.1 states:

- `branch`: `{"name", "exists", "tip", "contains_revision"}`; `tip` null when the branch is missing.
- `worktree`: null without a worktree, otherwise `{"path", "relative", "head", "ahead_of_revision", "dirty"}`;
  `relative` is the worktree path relative to the main checkout, using `/` separators;
  `ahead_of_revision` null without a revision or with one Git does not know, `dirty` `[]` when clean, each entry
  `{"path", "state"}`.
- `spec`: `{"in_worktree", "at_revision"}`.
- `plan`: null without a recorded spec, base, or branch, and for `done`; otherwise
  `{"approval", "at", "steps", "other_commits", "discrepancies"}`, plus `problem` only for a parse error.
  Each step is `{"id", "subject", "state", "commit"}`; `commit` is null without a checkpoint. Each discrepancy
  is `{"code", "commit", "message"}`. Approval and step states follow 8.7.
- `merge`: `{"merged", "merge_commit", "derived_spec_ref"}`; both null while `merged` is false, and each null for
  a `done` record that left it empty.
- `forge`: `{"available", "pr", "checks"}`, `pr` null or `{"number", "url", "state", "merge_commit"}`, `state` as
  `gh` prints it (`OPEN`, `MERGED`, `CLOSED`) and `merge_commit` null until `gh` reports one. Without
  `gh`, on a `gh` failure, or with `--no-forge`: `{"available": false, "pr": null, "checks": "unknown"}`; with no
  PR for the branch: `{"available": true, "pr": null, "checks": "unknown"}`.

`allowed_next` lists the commands valid from the record under 8.1 and 8.2, one string each, by status:

| Status | Commands |
|---|---|
| `planned` | `task register <ID> --title <title> --owner <owner>` |
| `reserved` | `task move <ID> --as <owner> --status running --stage specification`, block, pause, cancel |
| `running` | the stage moves 8.2 allows for the tier; at `integration`, `task done`; block, pause, cancel |
| `blocked` | `task unblock <ID> --as <owner> <B-n>` per open blocker, with `--status <prior>` appended only when exactly one blocker is open (`task unblock W-001 --as orchestrator-1 B-1 --status running`); with two or more, no line carries `--status`; block, cancel. With no open blocker, only block and cancel |
| `paused` | `task move <ID> --as <owner> --status <prior>`, cancel |
| `cancelled` | `task move <ID> --as <owner> --status <prior> --resume-cancelled` |
| `done` | none |

With plan facts (8.7), a forward move a managed gate would reject is left out: `specification` → `implementation`
unless the plan is `approved`, and the forward moves out of `implementation` unless it is `approved` with no
discrepancy and no remaining step.

An unconfirmed `reserved` or `running` record past `setup` lists `task confirm <ID> --as <owner>` instead of its
stage moves and `task done`, then block, pause, cancel; an unconfirmed `paused` one lists `task confirm` before its
resume and cancel lines.

Order: stage moves in stage order, then `task done`, then block, pause, cancel. The forms, each prefixed
`solomonar`:

- block: `task block <ID> --as <owner> --kind <kind> --summary <text>`
- pause: `task move <ID> --as <owner> --status paused --next <actor>:<action>`
- cancel: `task move <ID> --as <owner> --status cancelled --next <actor>:<action>`
- stage move: `task move <ID> --as <owner> --stage <stage>`; a back move adds `--next <actor>:<action>`
- done:
  `task done <ID> --as <owner> --merge <full-sha> --spec-ref <PR "Final specification" line> --summary <one sentence>`

`--as` is the recorded owner, written after the ID. Placeholders are `<actor>:<action>`, `<kind>`, `<text>`,
`<path>`, `<full-sha>`, `<n>`, `<title>`, `<owner>`, and `<one sentence>`; the prior status (`<prior>`, 8.1) and
blocker IDs (`<B-n>`, in record order) are filled from the record. A flag a move requires (8.2) appears as a
placeholder only when the record lacks it: `--spec <path>` only on the forward move from `specification` into
`implementation` (back moves into `implementation` carry only `--next`), and `--revision <full-sha>` and
`--tier <n>` on forward moves out of `implementation`. At `implementation` without a tier, both forward targets are
listed: `specialized_review` with `--tier <n>`, `integration` with `--tier 0`. At `specialized_review` or later
without a tier, only the back moves are listed, since `--tier` without `--revision` is not accepted there (8.2
field rules). `task done` fills `--merge` with the merge commit when merged and the commit is known, otherwise
`<full-sha>`; `--spec-ref` stays the placeholder,
since the orchestrator takes `spec_ref` from the PR's "Final specification" line (C14 a) and the derived one is a
cross-check (C14 b). When the branch is merged and the record is not done, `allowed_next` lists only `task done`.
`task handoff` is never listed: it runs only on the user's instruction. `allowed_next` never judges which command
the orchestrator should run (J1).

## 10. Delivery order

J7 puts `init` and the `task` commands first; the init choices in this spec move part of the adapter work forward.
What each increment makes available to users, including the new-product path that needs `extension add`:
[CLI overview, Planned commands](../../site/src/docs/cli.md#planned-commands). This table keeps the completion
conditions.

| Increment | Commands | Completion condition |
|---|---|---|
| 1 | `init` (core, runtimes, catalog extensions), every `task` command, `version`, `help` | UC-1 to UC-4 and UC-11 to UC-21 pass as tests in temporary repositories, including the compare-and-swap retry and repeated `init` as a no-op |
| 2, partial (0.2.0) | the extension store, a minimal `upgrade` (no `--diff`), `extension` list/add/remove (CLI-D27 to CLI-D31) | UC-6, UC-7, and the delivered UC-9 subset pass as tests in temporary repositories, including an upgrade from a v0.1.7 fixture, a repeated upgrade as a no-op, and a repeated `extension add` as a no-op |
| runtime (delivered, CLI-D36) | `runtime` list/add/remove, without hooks | UC-5 passes as tests in temporary repositories, including each runtime added from `--runtime none` and beside another, a repeated `add` and `remove` as no-ops, a model update, a removal conflict resolved by `--keep` and `--overwrite`, a rollback that restores the old manifest and files, and unchanged registry tips; the `runtime_management` transcript covers the process boundary |
| 2, rest | `doctor`, `guard` with hook installation and removal | UC-10, UC-22 |
| uninstall (delivered) | `uninstall` (CLI-D32 to CLI-D35) | UC-27 passes as tests in temporary repositories, including current, core-only, and v0.1.7 layouts, a conflict and a rollback that leave the installation intact, a linked worktree whose neighbors and registry tips stay unchanged, and a repeated `uninstall` as a no-op |
| 3 | `upgrade --diff`, `dashboard` | the rest of UC-9, UC-23 |

`uninstall` is an additional delivery that needs only the existing plan, apply, and journal; it was delivered
without waiting for the rest of increment 2. Hook entry removal ships with `guard` (UC-27). Runtime management is
likewise delivered apart from `guard` (CLI-D36): it installs no hook, so it needs none of the guard work.

Copilot is an additional adapter delivery, after increment 1. Its first release
adds generation and `init` selection only; runtime management is the separate runtime delivery (UC-5), and no
Copilot hook is specified by this addition. Support is established only after automated checks and a recorded live CLI smoke
test, including a feature worktree and coexistence with the other adapters. Adapter generation, manifest tracking,
and CLI initialization are implemented. The Copilot installation transcript passes, and CLI 1.0.85 discovers
instructions and core skills in main and feature worktrees with all three adapters installed. CLI 1.0.103
sessions run, but prompt mode cannot invoke the entry skill. The live interactive workflow was confirmed on
2026-10-25; the minimum supported version is not established. See [the acceptance record](#copilot-runtime-acceptance).

### Copilot runtime acceptance

The Copilot delivery ran in four steps: contract (section 4.2), project-layer generation and `runtime:copilot`
manifest tracking, `init --runtime copilot`, and validation with public documentation. All four steps are implemented.
Automated coverage: project tests for ownership, ordering, extension inclusion, metadata escaping, and model
selection; init tests for runtime combinations, `PATH` detection, prompts, conflicts, and reruns; the
`testdata/init_copilot.txtar` process transcript; `task check`, `task test:integration`, `go vet ./...`, and the
site build and check.

Live checks on 2026-10-23, in disposable repositories initialized with all three adapters:

| Version | Check | Result |
| --- | --- | --- |
| 1.0.85 | `copilot instruction list --json`, `copilot skill list --json` in main and feature worktrees | `AGENTS.md` and the four core skills resolve to the checkout's `.github/skills/`, ahead of `.agents` and `.claude` |
| 1.0.85 | Live `-p` session | Rejected: `Access denied by policy settings` (account policy) |
| 1.0.103 | Live `-p` session | Runs; skills load with `source: project` |
| 1.0.103 | `-p '/solomonar'` | Not expanded; `disable-model-invocation: true` also blocks the model's skill tool, so prompt mode cannot invoke the entry skill. Removing the flag in a fixture made the model call `skill: solomonar` |
| 1.0.103 | `-p` delegation on a registered task and worktree | No delegation happened (entry skill unavailable); registry unchanged |
| 1.0.103 | `--allow-tool 'shell(...)'` | Rules match only the first word: `shell(solomonar:*)` allows, `shell(solomonar task show)` matches nothing |
| 1.0.103 | `auto` model routing | One model available per session (`mai-code-1.1-flash`, `gpt-6-luna`); `--model claude-sonnet-5.5` rejected as unavailable |

Decision 2026-10-23 (user): keep `disable-model-invocation: true` on the Copilot entry stub. Interactive
`/solomonar` is the supported entry; prompt-mode sessions ask Copilot to load
`docs/solomonar/skills/solomonar/SKILL.md`. Confirmation 2026-10-25 (user): the live interactive Copilot workflow is confirmed. The minimum supported Copilot
version remains open; version 1.0.85 is a discovery baseline, not a supported floor.

## 11. Open decisions

| ID | Question | Options | Recommendation | Decision |
|---|---|---|---|---|
| CLI-D1 | How language knowledge reaches reviews | a) language-neutral core reviewers; `spec-reviewer` checks the `must` rules of the installed extension skills that apply to the change; b) one `<lang>-reviewer` role per language, rendered into the rules section 6 table | a: one rules table, a language is one skill, and mixed-language changes need no extra reviewer | **a**, author, 2026-10-20; revised 2026-10-21: core reviewers are `security-reviewer` (bugs, security), `correctness-reviewer` (business logic), `spec-reviewer` (work spec and language rules), `adversarial-reviewer` |
| CLI-D2 | How the CLI knows the acting orchestrator | a) `--as <owner>` with `SOLOMONAR_OWNER` default; b) no check, ownership stays behavioral | a: rules section 7 promises ownership validation; it remains a behavioral contract, not authentication | **a**, author, 2026-10-20 |
| CLI-D3 | Claude Code root instructions (G10) | a) create a one-line `CLAUDE.md` (`@AGENTS.md`) when absent, never modify an existing one; b) nothing, if Claude Code is verified to load `AGENTS.md` natively | a, unless the verification at implementation time shows b | **a**, author, 2026-10-22: implemented in v0.1.0 (4.2) |
| CLI-D4 | Does `init` commit on main | a) never; print the commit command; `doctor` fails until committed; b) `--commit` flag; c) always | a: commits on main stay the user's; b can be added later | **a**, author, 2026-10-22: implemented in v0.1.0 (UC-1) |
| CLI-D5 | Prompt implementation | a) line-based prompts with the standard library; b) a full-screen prompt library (a new dependency, needs approval) | a: no dependency, scriptable, testable with piped input | **decided**, author, 2026-10-21: text prompts and confirmations line-based with the standard library; the multi-select key-driven (↑/↓, Space, Enter) in raw terminal mode through `golang.org/x/term` (architecture AD-1) |
| CLI-D6 | Extension sources in v0.1 | a) the embedded catalog and the project's store (CLI-D27); b) a, plus Git URLs | a: remote sources bring pinning and trust questions; add later with a recorded commit and hash | open |
| CLI-D7 | Adapter file form | a) stubs that point to `docs/solomonar/`; b) full copies; c) symlinks | a: one source of content, small diffs, works where symlinks do not | **a**, author, 2026-10-22: implemented in v0.1.0 (4.2) |
| CLI-D8 | Binary name: `go install ./cmd/cli` produces `cli` | a) rename the package directory to `cmd/solomonar`; b) keep `cmd/cli`, document `go build -o solomonar ./cmd/cli` and set the name in release builds; c) `main.go` at the module root, so `go install github.com/madalinpopa/solomonar` yields `solomonar` | c: the repository holds one binary | **c**, author, 2026-10-21 (architecture AD-13) |
| CLI-D9 | Codex specifics: skill path, native role definitions, hook support | verify each against Codex documentation before increment 1 | adopt what is verified; fall back to packets naming role files and no hook | **decided**, author, 2026-10-21, from the official docs (I section 4.2): skills `.agents/skills/<skill>/SKILL.md`, the `solomonar` skill with `agents/openai.yaml` `allow_implicit_invocation: false`; roles `.codex/agents/<role>.toml` (`name`, `description`, `developer_instructions`, `model`), no tool allowlist; guard a `PreToolUse` hook in `.codex/hooks.json`, run once the user trusts it; Claude Code forms confirmed |
| CLI-D10 | `workflow` and `review` extensions in the catalog | a) `extension-authoring` as the first `workflow` skill, `default: true` and `invocation: user`, with no `review` extension yet; b) none, mechanism only; c) add `baseline` (records commands and known failures, rules section 11) and `decision` (prepares a decision entry, rules section 3) as `workflow` skills | a: the skill that writes project extensions is useful in every project, and only the user should start it; `baseline` and `decision` remain candidates | **a**, author, 2026-10-24 (4.4) |
| CLI-D11 | How a change of stack extensions is approved and recorded | a) `--decision <ID>` names an approved entry; otherwise the user's own command appends an approved project entry `P/D-n` per group, superseding the previous one, plus a line in `docs/stack.md` or `docs/architecture.md`; b) extensions install without any decision record; c) the CLI only accepts `--decision`, and the user writes every entry by hand | a: choosing a stack extension is an architecture or dependency decision (rules section 3); the user's command is that explicit decision, and the record keeps rules section 1 authority traceable | **a**, author, 2026-10-21: accepted workflow audit F-08; rules section 3 and register L |
| CLI-D12 | Rule IDs and force inside extension skills | a) `<name>/C-n` with `must` (approved global rule, blocking) and `should` (suggestion); b) every extension rule binding; c) extensions are guidance only | a: matches the finding handling in rules section 6 and keeps preferences out of the blocking set | **a**, author, 2026-10-22: implemented in v0.1.0 (4.3) |
| CLI-D13 | Extension catalog | a) a catalog for every group; b) a small catalog of reviewed content | b: only content written and reviewed as framework content ships; other groups take project extensions until catalog ones are written | **b**, author, 2026-10-20; revised 2026-10-24 for the shipped catalog: `go` (`language`, `detect: ["**/go.mod"]`) and `extension-authoring` (`workflow`, CLI-D10); `htmx` and `templ` remain only an illustrative test catalog (4.4) |
| CLI-D15 | The user's entry point to the process | a) a core skill `solomonar`, invoked only by the user, that routes a request to `prototype`, `spec`, or the orchestrator role taken in the same top-level session, plus a short "Route a request" section in `AGENTS.md` for sessions started without it; b) the orchestrator rules move into `AGENTS.md`, so every session is an orchestrator; c) a skill that delegates the orchestrator to a subagent | a: b loads about 600 lines into every session and every delegated role, and contradicts the implementer and reviewer contracts; c fails because a delegated subagent cannot delegate the implementer and reviewers | **a**, author, 2026-10-21; framework files installed under `docs/solomonar/` instead of `docs/framework/` |
| CLI-D14 | Core boundary | a) core = what `rules.md` names as a process step or required role: `orchestrator`, `implementer`, `security-reviewer`, `correctness-reviewer`, `spec-reviewer`, `adversarial-reviewer`; `prototype`, `spec`, `work-spec`; b) core = `orchestrator`, `implementer`, `work-spec`; reviewers and discovery skills become extensions | a: the review gates and the prototype-to-spec flow are the process; making them optional turns a required review into a choice | **a**, author, 2026-10-20: the current framework in `assets/framework/` is the core, with language-neutral core reviewers and Go rules in the `go` extension (CLI-D1, revised 2026-10-21) |
| CLI-D16 | How the prior status of a `blocked`, `paused`, or `cancelled` task is known | a) derived from registry history: the last status other than `blocked` or `paused`; b) derived from the record: a task is `reserved` exactly while at `setup`, so the prior status is `reserved` at `setup` and `running` later | b: no walk of registry history on every `unblock` or resume, and the rule is a pure check on one record | **b**, author, 2026-10-21 |
| CLI-D17 | An unmanaged file whose content equals what the CLI would write | a) conflict, as any unmanaged file; b) adopted into the manifest, shown as `unchanged (adopted)` | b: a re-run after an `init` killed before its manifest write completes instead of failing on the run's own files; nothing is lost, since the recorded base is the file's content | **b**, author, 2026-10-21 |
| CLI-D18 | Exit code of an interrupted command | a) 1, as an unexpected failure; b) 130 or 143 whenever a signal arrives; c) the exit code reports what was written: 128 + signal with `interrupted` when nothing was written, 0 with `push_interrupted` after the compare-and-swap | c: after a successful write, a failure exit makes a script or agent retry and repeat a non-idempotent write such as `task plan` | **c**, author, 2026-10-21 |
| CLI-D19 | A project or registry written by a newer CLI | a) `already_initialized` for a downgrade only; b) `cli_outdated` (exit 3) for every command that reads the manifest or the registry, checked before strict decoding; `version` and `doctor` report instead of rejecting | b: with strict decoding, a newer schema would otherwise read as `registry_invalid` with the wrong fix | **b**, author, 2026-10-21 |
| CLI-D20 | Registry layout | a) one `state.json` holding every task; b) `state.json` with the schema only, plus `tasks/<ID>.json` per task | b: a write changes one task's file, history per task reads with `git log -- tasks/<ID>.json`, and keeping the version in `state.json` makes a schema-1 CLI reject the layout as `cli_outdated` | **b**, author, 2026-10-23 (8.6) |
| CLI-D21 | Workspace fields | a) derive the worktree from Git and store no base; b) store `base` (immutable) and `worktree` (relative to the main checkout), with Git's listing authoritative | b: recovery needs the base to bound checkpoints; a relative path stays portable across machines with the same layout | **b**, author, 2026-10-23 (8.6) |
| CLI-D22 | How an execution is identified for confirmation | a) the owner ID: a new session takes the task with `task handoff`, which clears `confirmed`, and runs `task confirm`; b) a separate opaque execution field | a: no second identity beside the owner; the CLI cannot detect a new session, so the handoff is the boundary | **a**, author, 2026-10-23 |
| CLI-D23 | Work-spec plan contract | a) `Status: In review` / `Status: Approved` below the title, steps `S-<n>: <commit subject>` under `## Implementation plan`, IDs never reused; b) a separate approval section | a: one line to check, one list to reconcile; the path stays `docs/work/<ID>-<slug>.md` | **a**, author, 2026-10-23 |
| CLI-D24 | Checkpoint commit metadata | a) the planned subject and a `Step: <ID>/S-<n>` trailer; b) a, plus a `Plan: <sha>` trailer | a: step IDs are never reused, so a plan version adds noise without disambiguating | **a**, author, 2026-10-23 |
| CLI-D25 | Who creates checkpoints | a) the implementer, with Git; the CLI only inspects; b) a CLI commit command | a: the decision to use real Git commits needs no wrapper | **a**, author, 2026-10-23 |
| CLI-D26 | Registry migration | a) an explicit `task migrate`; b) automatic in `init` or `upgrade` | a: registry migration stays separate from framework updates and never runs implicitly | **a**, author, 2026-10-23 (UC-25) |
| CLI-D27 | Where installed and project extensions live | a) one flat store `docs/solomonar/extensions/<name>/` holding only the installed catalog extensions, copied verbatim with `extension.json` and owner `ext:<name>`, and the project's own extensions in the same format, project-owned and never written or tracked by the CLI; b) installed files split into `docs/solomonar/skills/` and `docs/solomonar/agents/`, with project sources elsewhere and their metadata in the manifest; c) the whole catalog copied into every repository | a: one place and one format for both kinds, one loader, and a manifest that records only choices and CLI-written files; the planner removes catalog files that are no longer wanted (4.5) | **a**, author, 2026-10-24 (4.3, 4.5, 4.7) |
| CLI-D28 | How the CLI suggests extensions | a) `detect` globs and `default` in `extension.json`, matched over `git ls-files --cached --others --exclude-standard` with an own `**` matcher; `init` preselects, `extension list` marks `suggested`, nothing installs on its own; b) hard-coded rules in the CLI (`go.mod` → `go`); c) install what is detected | a: the catalog states its own rules, project extensions can use them too, and a suggestion never replaces the user's choice | **a**, author, 2026-10-24 (4.3) |
| CLI-D29 | Which skills only the user starts | a) an `invocation` field (`model` or `user`) in `extension.json` that adds `disable-model-invocation: true` to the Claude and Copilot stubs and the Codex `allow_implicit_invocation: false` policy, with a separate `AGENTS.md` line; b) a hard-coded name list; c) no user-only extensions | a: metadata, not names, decides; a separate line keeps user-invoked skills discoverable without the model loading them; Copilot prompt mode cannot start them (4.2, known limitation) | **a**, author, 2026-10-24 (4.2, 4.6) |
| CLI-D30 | Combination rules and provenance inside extension files | a) none: extension files install byte-identical, and the decision appears in the `AGENTS.md` line and the manifest; b) sections kept or dropped by the installed set, plus a provenance line in each main file | a: no shipped extension needed combinations, verbatim copies keep hashes and upgrades simple, and project-owned files cannot be re-rendered anyway | **a**, author, 2026-10-24 (4.3, 4.6) |
| CLI-D31 | Order of the increment 2 extension work | a) a minimal `upgrade` (no `--diff`; `--keep`, `--take-new`, `-y`, `--dry-run`, `--json`) first, so v0.1.7 projects move to the store; then `extension list`, `add`, `remove`, which reject an older framework with `framework_outdated`; b) all of increment 2 at once; c) the `extension` commands with an automatic layout migration | a: the commands work on one layout only, and the migration stays one explicit, reviewable command | **a**, author, 2026-10-24 (UC-9, 3.6) |
| CLI-D32 | How the project installation is removed | a) one top-level `uninstall` that removes the core, catalog extensions, and adapters together; while the framework stays installed, no core item can be removed, and `extension remove` keeps rejecting one (`core_item`); b) core items removable one by one through `extension remove`; c) a `remove` alias or an `init --remove` mode | a: the core is the process (CLI-D14) and is either installed whole or not at all; one name keeps project uninstall distinct from extension removal and from removing the executable | **a**, author, 2026-10-24 (4.1, UC-27) |
| CLI-D33 | What `uninstall` deletes and keeps | a) only manifest-recorded files whose hash equals the recorded base, legacy locations included, with no prior `upgrade`; project extension sources, decision and seeded documents, and `workflow-state` stay; no `none` decision is written; b) regenerate today's wanted set and remove it; c) also delete decisions and the registry | a: recorded ownership is the only authority for deletion; regeneration misses legacy paths and would load the catalog; decisions and task history are evidence, and the installed technology does not change | **a**, author, 2026-10-24 (4.5, UC-27) |
| CLI-D34 | How `uninstall` treats `AGENTS.md` | a) remove only the verified managed part and keep the `## Project notes` part byte for byte, adopted instructions and heading included; remove the file only when the notes hold just the template; never discard notes to resolve a missing boundary; b) delete the whole file; c) restore the pre-install content | a: the notes are project-owned; c is impossible because the manifest does not record the original content | **a**, author, 2026-10-24 (UC-27) |
| CLI-D35 | The framework entry in the seeded `docs/README.md` | a) `uninstall` removes only the exact generated line `` - `docs/solomonar/`: framework rules, roles, and skills `` from `## Documents`, keeps every other byte, and reports a customized entry for manual review; b) never touch the seed and leave a dangling entry; c) delete the file or every framework mention | a: a narrow, explicit exception to the rule that the CLI never modifies a present seed except to append index lines (4.3, 4.5) that removes the only active generated reference without rewriting project prose | **a**, author, 2026-10-24 (4.5, UC-27) |
| CLI-D36 | Scope of the runtime delivery and the defaults UC-5 left open | a) `runtime list\|add\|remove` only, for `claude`, `codex`, `copilot`, through the shared planner; hook installation, removal, and `guard` deferred; b) runtime commands together with guard hooks for Claude and Codex | a: no hook may point to an unavailable `guard`, and the adapter files are useful without one | **a**, user, 2026-10-24 (confirmed scope). Newly specified defaults, author, 2026-10-24 (UC-5): `list` works in any Git repository and reports installation from the manifest only, never from `PATH` or file health; `add\|remove` need an initialized project whose framework matches the CLI (`framework_outdated` otherwise), while `list` reads an older valid manifest; `add` of an installed runtime reconciles it and is a no-op when nothing changes; an omitted `--adversarial-model` keeps the recorded model, an explicit empty one is `invalid_value`; `remove` of an uninstalled supported runtime is a no-op; no-ops need no confirmation; `none` and other names are `unknown_runtime` |
| CLI-D37 | Whether `sync` ships | a) withdraw `sync` (UC-8): `upgrade` at the project's version refreshes the stubs and `doctor` reports stale adapters with exit 5; b) a separate `sync` that regenerates only adapters, with `--check` for CI | a: stubs take core and catalog metadata from the release, so a metadata edit in `docs/solomonar/` can never reach them; the remaining cases (project extension metadata, a deleted or stale stub) are a plan `upgrade` already computes and applies at the same version, and the CI check belongs with the other `doctor` checks | **a**, user, 2026-10-25 |
