---
title: solomonar runtime
navTitle: solomonar runtime
description: List, add, and remove the Claude Code, Codex, and GitHub Copilot CLI adapters of a Solomonar project with solomonar runtime list, add, and remove.
kind: Reference
group: Commands
order: 13.6
sources:
  - internal/cli/runtime.go
  - internal/project/init.go
  - internal/project/apply.go
  - docs/cli/interface.md
---

A runtime is the agent product that runs sessions: Claude Code (`claude`), Codex (`codex`), or GitHub Copilot CLI (`copilot`). Each runtime gets small generated adapter files that point to the portable source in `docs/solomonar/`, never a second copy of it.

`init --runtime` records the first choice. `init` never changes recorded runtime choices afterwards; `runtime add` and `runtime remove` do, one runtime at a time, and `runtime list` shows where each stands.

The three commands manage adapter files and the manifest's runtime choices only. None installs, merges, or removes a guard hook, touches a runtime's settings or hook files, runs the runtime, or checks provider access. `solomonar guard` and its hooks remain deferred.

Copilot targets the CLI only. It reads `AGENTS.md` natively and gets `.github/skills/<skill>/SKILL.md` and `.github/agents/<role>.agent.md` stubs, tracked as `runtime:copilot`. Its entry skill is user-invoked with `/solomonar`; delegated roles load the portable role and repository instructions. IDE extensions and the cloud agent are not supported.

The adapter follows GitHub's [CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference) and [custom-agent documentation](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli). Prompt mode (`copilot -p`) cannot load the entry skill; invoke `/solomonar` in an interactive session.

## solomonar runtime list

Read-only. Lists the supported runtimes with whether each is installed in the project, whether its executable is on `PATH`, and its adapter locations. It works in any Git repository, initialized or not: without a manifest every runtime is `available`. Installation comes from the manifest only, never from `PATH` or file health.

{% cliUsage "Before adding or removing a runtime, or to see which adapter files belong to which runtime.", "You." %}

```text
solomonar runtime list [--json]
```

### Example

In a project initialized with `--runtime none`:

```sh
solomonar runtime list
```

```text
RUNTIME  STATUS     ON PATH  ADAPTER
claude   available  yes      CLAUDE.md, .claude/skills, .claude/agents
codex    available  yes      AGENTS.md (native), .agents/skills, .codex/agents
copilot  available  yes      AGENTS.md (native), .github/skills, .github/agents
```

`STATUS` is `installed` once the manifest records the runtime. `ON PATH` only reports whether the `claude`, `codex`, or `copilot` executable is found.

With `--json`, `data.runtimes` lists the same rows in the same order:

```json
{
  "name": "copilot",
  "status": "available",
  "on_path": true,
  "adapters": [
    "AGENTS.md",
    ".github/skills",
    ".github/agents"
  ]
}
```

## solomonar runtime add

Generates the adapter files of one runtime for the core and the installed extensions, and records the runtime in the manifest. Other runtimes, models, extensions, and decisions stay as they are. Adding works even when the runtime's executable is not on `PATH`.

Adding a runtime that is already installed reconciles it: missing adapter files are restored, and `--adversarial-model` updates its recorded model and the adversarial-reviewer stub. When nothing changes, the command is a no-op that needs no confirmation.

{% cliUsage "When you start using another runtime in an initialized project, chose `--runtime none` at `init`, or want to change a runtime's adversarial-reviewer model.", "You. Commit the new files on main afterwards." %}

```text
solomonar runtime add <claude|codex|copilot> [--adversarial-model <model>]
                      [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<claude\|codex\|copilot>` | Required. The runtime to add. |
| `--adversarial-model <model>` | Model for this runtime's adversarial reviewer, as a plain name (not `init`'s `<runtime>=<model>`). Omitted, the recorded model is kept, and a new runtime has none. An empty value is an error (exit 2). |
| `--keep <path>` | Keep one conflicting file's local content; one path per flag. |
| `--overwrite <path>` | Replace one conflicting file with the generated content; one path per flag. |
| `-y`, `--yes` | Accept the plan without the confirmation prompt. |
| `--dry-run` | Print the plan and write nothing. |

### Example

```sh
solomonar runtime add copilot --dry-run
```

```text
Plan
  unchanged  docs/solomonar/agents/adversarial-reviewer.md
  ...
  unchanged  AGENTS.md
  skip       docs/README.md
  create     .github/skills/prototype/SKILL.md
  create     .github/skills/solomonar/SKILL.md
  create     .github/skills/spec/SKILL.md
  create     .github/skills/work-spec/SKILL.md
  create     .github/agents/adversarial-reviewer.agent.md
  create     .github/agents/correctness-reviewer.agent.md
  create     .github/agents/implementer.agent.md
  create     .github/agents/security-reviewer.agent.md
  create     .github/agents/spec-reviewer.agent.md
  update     .solomonar/manifest.json
Would add runtime copilot in /work/ledger
  9 files created, 1 updated, 21 unchanged, 1 skipped
```

```sh
solomonar runtime add copilot --yes
```

```text
Added runtime copilot in /work/ledger
  9 files created, 1 updated, 21 unchanged, 1 skipped

Next:
  1. git add .github/skills/prototype/SKILL.md .github/skills/solomonar/SKILL.md .github/skills/spec/SKILL.md .github/skills/work-spec/SKILL.md .github/agents/adversarial-reviewer.agent.md .github/agents/correctness-reviewer.agent.md .github/agents/implementer.agent.md .github/agents/security-reviewer.agent.md .github/agents/spec-reviewer.agent.md .solomonar && git commit -m "chore: add solomonar runtime copilot"
  2. Start a top-level GitHub Copilot CLI session in /work/ledger:
       /solomonar Explore the next useful flow.
```

The `git add` suggestion names the changed top-level directories, such as `CLAUDE.md .claude .solomonar` for Claude Code, but individual files under `.github`, which may hold unrelated files. The CLI never runs it. With `--json`, `next` holds only the commit suggestion.

Running the same command again changes nothing:

```text
Already installed: copilot; nothing to do.
  31 files unchanged, 1 skipped
```

To change an installed runtime's model:

```sh
solomonar runtime add claude --adversarial-model opus --yes
```

For Copilot, `--adversarial-model <model>` writes only the adversarial-reviewer stub's `model` field. An omitted model inherits runtime defaults. Copilot can override or fall back from the configured model; installation does not authenticate, check account access, or guarantee model isolation.

## solomonar runtime remove

Deletes the runtime's unmodified adapter files and drops the runtime and its model from the manifest. It keeps a `CLAUDE.md` it did not create, unrelated files under `.github`, `.claude`, `.codex`, and `.agents`, the framework, and every extension. Emptied subdirectories are removed, but the top-level `.github`, `.claude`, `.codex`, and `.agents` directories stay. A modified adapter file is a conflict (exit 4) until you resolve it. Removing the last runtime leaves the framework installed.

Removing a runtime that is not installed is a no-op, with `--dry-run` too: `Not installed: copilot; nothing to do.`

{% cliUsage "When the project stops using a runtime.", "You. Commit the removals on main afterwards." %}

```text
solomonar runtime remove <claude|codex|copilot>
                         [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
```

### Flags

| Flag | Meaning |
| --- | --- |
| `<claude\|codex\|copilot>` | Required. The runtime to remove. |
| `--keep <path>` | Untrack one modified adapter file and leave it on disk; one path per flag. |
| `--overwrite <path>` | Delete one modified adapter file; one path per flag. |
| `-y`, `--yes` | Accept the plan without the confirmation prompt. |
| `--dry-run` | Print the plan and write nothing. |

### Example

```sh
solomonar runtime remove copilot --dry-run
```

```text
Plan
  unchanged  docs/solomonar/agents/adversarial-reviewer.md
  ...
  remove     .github/agents/adversarial-reviewer.agent.md
  ...
  remove     .github/skills/work-spec/SKILL.md
  update     .solomonar/manifest.json
Would remove runtime copilot in /work/ledger
  1 file updated, 9 removed, 21 unchanged, 1 skipped
```

```sh
solomonar runtime remove copilot --yes --json
```

```json
{
  "ok": true,
  "command": "runtime remove",
  "data": {
    "project": "/work/ledger",
    "runtime": "copilot",
    "files": [
      {
        "path": ".github/agents/adversarial-reviewer.agent.md",
        "action": "remove",
        "owner": "runtime:copilot"
      }
    ],
    "next": [
      "git add .github/agents/adversarial-reviewer.agent.md .github/agents/correctness-reviewer.agent.md .github/agents/implementer.agent.md .github/agents/security-reviewer.agent.md .github/agents/spec-reviewer.agent.md .github/skills/prototype/SKILL.md .github/skills/solomonar/SKILL.md .github/skills/spec/SKILL.md .github/skills/work-spec/SKILL.md .solomonar && git commit -m \"chore: remove solomonar runtime copilot\""
    ]
  },
  "warnings": []
}
```

`files` lists every planned path; the excerpt above shows one. The suggestion names the deleted files, so `git add` stages their removal.

## Errors and exit codes

`add` and `remove` need an initialized project whose framework matches the CLI; run `solomonar upgrade` first in an older project. `list` also reads an older project. With `--json` or without a terminal, a change needs `--yes`; a dry run or a no-op does not:

```text
error: runtime add needs a confirmation (interactive_required)
  fix:   pass --yes to accept the plan, or --dry-run to print it
nothing was written
```

A conflict plan prints on stdout before the error, as for [`extension`]({{ '/docs/cli/extension/' | url }}). A `--keep` or `--overwrite` path that is not a conflict is rejected, also on a not-installed `remove`.

| Exit | When |
| --- | --- |
| 0 | Success, dry run, no-op, or declined confirmation. |
| 1 | Unexpected I/O or Git failure, an invalid manifest, or failed prompt input. |
| 2 | Unknown runtime name (`none` included), invalid usage, empty `--adversarial-model`, a `--keep` or `--overwrite` path that is not a conflict, or a change without `--yes` where no prompt can be shown. |
| 3 | Outside a Git repository, `add` or `remove` in an uninitialized project, or a framework older or newer than the CLI. |
| 4 | An unresolved conflict; `--yes` never resolves it. |
| 130, 143 | Interrupted before the file plan completes; every change is rolled back. |
