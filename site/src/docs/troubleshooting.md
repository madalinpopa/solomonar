---
title: CLI troubleshooting
navTitle: Troubleshooting
description: Diagnose Solomonar CLI errors, init conflicts and confirmations, missing initialization, task ownership conflicts, rejected transitions, and exit codes.
kind: How-to guide
group: Guides
order: 10
sources:
  - docs/cli/interface.md
  - internal/cli/errors.go
  - internal/cli/repo.go
---

## A command is unknown

First run `solomonar help` to see the command table in your binary. The interface specification includes future commands; the current build does not yet implement `doctor`, `sync`, `runtime`, `extension`, or `upgrade`.

Use the [availability table]({{ '/docs/cli/#current-commands' | url }}) before following an example from the full design contract. An unknown command exits 2; it does not mean your application needs to be scaffolded again.

## The registry is not initialized

`not_initialized` means `refs/heads/workflow-state` is missing. Task commands depend on that registry. Run `solomonar init` in the repository; see [Set up a project]({{ '/docs/set-up-a-project/' | url }}).

Do not write `state.json` by hand or create a registry branch to bypass initialization.

## init asks for a confirmation

`interactive_required` means `init` had something to write but no terminal to ask. Review the plan with `--dry-run`, then rerun with `--yes`. A run that would change nothing needs no confirmation.

## init reports a conflict

`file_conflict` (exit 4) means a file in a target path differs from what `init` would write: you edited a managed file, or the path held your own file before. The plan lists each conflict, and nothing was written. Decide per file:

```sh
solomonar init --keep docs/solomonar/rules.md --yes
solomonar init --overwrite .claude/agents/implementer.md --yes
```

`--keep` preserves your bytes; `--overwrite` writes the framework's version. An existing `AGENTS.md` is never a conflict: it is adopted, with your content moved under `## Project notes`.

## init rejects a choice

`already_initialized` means the project's manifest records a different choice than the one you passed, such as another runtime. `init` never changes recorded choices; the planned `runtime` and `extension` commands will. Rerun without the differing flag.

`unknown_extension` (exit 2) means an `--ext` value is not in the embedded catalog; the error lists the known names. `--ext none` cannot be combined with another name.

`not_a_repository` from `init` means the directory is not inside a Git repository. Run `git init -b main` first, or pass `--git-init`.

To add Copilot to a project that already records different Solomonar runtimes, wait for the planned `runtime add` command. Do not hand-edit `.solomonar/manifest.json`, lower its version, or rerun `init` with different runtime choices.

## Copilot does not discover the framework

Check `solomonar init --help` for `copilot`; an older build may not include the adapter. Initialize an eligible project with `--runtime copilot`, then start `copilot` at its worktree root. The adapter uses `.github/skills/<skill>/SKILL.md` and `.github/agents/<role>.agent.md`, while root instructions come from `AGENTS.md`.

On Copilot CLI 1.0.85, these read-only commands report instruction and skill discovery:

```sh
copilot instruction list --json
copilot skill list --json
```

The Solomonar skills should point into the selected worktree's `.github/skills/`. Copilot also reads compatible `.agents` and `.claude` skill directories, so inspect the resolved paths. Custom-agent discovery requires a live session: use `/agent` there. A personal agent in `~/.copilot/agents/` takes precedence over a project agent with the same ID. Check the filename-derived ID, not just its display name. These rules follow GitHub's [CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference) and [custom-agent documentation](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli).

Invoke `/solomonar` yourself in an interactive top-level session. The entry skill disables automatic model invocation, and Copilot 1.0.103 does not expand `/solomonar` in prompt mode (`copilot -p`), so a prompt-mode session cannot load it. For a non-interactive session, ask Copilot to load `docs/solomonar/skills/solomonar/SKILL.md` and follow it. Delegated roles opt into repository instructions with `include-custom-instructions: true`; a session using `--no-custom-instructions` disables that loading. Do not edit generated stubs to work around discovery; inspect the portable source and use only commands your Solomonar build provides.

## Copilot rejects a session or uses another model

`Access denied by policy settings` means Copilot rejected the session under account, subscription, or organization policy. Follow the error's settings link or contact the account administrator. Regenerating Solomonar adapters does not resolve that rejection. The 1.0.85 acceptance attempt reached this error; later 1.0.103 sessions ran, but live workflow compatibility remains unverified; see [the acceptance record]({{ 'docs/cli/interface.md' | source }}#copilot-runtime-acceptance).

`--adversarial-model copilot=<model>` writes a model preference, not proof of which model ran. Session settings, per-agent overrides, automatic selection, and account model access can change the effective model. Inspect the running session and its subagent settings before recording a model in review evidence. Installation performs no account/model access check.

In Copilot 1.0.103, `--allow-tool 'shell(...)'` rules match only the first word: `shell(solomonar:*)` allows every `solomonar` command, while `shell(solomonar task show)` matches nothing.

No Copilot guard hook is installed. Reviewer shell access still follows the portable read-only contract; generated tool lists do not establish a filesystem security boundary.

## The command cannot find the repository

`not_a_repository` means the current directory, or the directory selected with `-C`, is outside a Git repository. Run from the intended repository or pass its path with `-C`.

Git must be version 2.40 or later. `gh` is optional; missing or unauthenticated forge access means those facts are unknown, never passing.

## An owner command is rejected

`owner_required` means neither `--as` nor `SOLOMONAR_OWNER` identifies the acting owner. `owner_mismatch` means that identity differs from the task's recorded owner.

Inspect the stored record with `task show`, then act as the correct owner. A handoff requires explicit user instruction and the `task handoff` command. Do not impersonate an owner to work around a stale session.

## A transition or review candidate is rejected

Read the error's `check`, `found`, and `fix` lines before retrying. Typical causes include:

- `revision_not_full_sha`: supply the full 40-character commit hash.
- `revision_not_on_branch`: the candidate must be reachable from the task's feature branch.
- `spec_not_found`: the spec must exist in the task worktree or at the recorded revision.
- `tier_required`: entering a review or integration stage needs the recorded risk tier.
- `blockers_required`: a blocked task needs an open blocker; unblocking the last one must also resume the prior status.

A rejected state update leaves the registry unchanged. A failed remote push after a successful local update is different: the local write remains successful and publication is reported as a warning. Inspect the current record before repeating a write.

## Exit codes

The [exit code table]({{ '/docs/cli/#exit-codes' | url }}) lists every code. Any non-zero code means nothing was written, so fix the cause and rerun. A `0` with a warning such as `push_failed` or `push_interrupted` means the local write succeeded and only publication failed: do not repeat the write.

## Report a problem with evidence

Include the CLI version, the command, the error code, and the relevant Git state. Redact secrets and private content. Distinguish an implemented command failure from a planned feature that is not available yet.

The detailed error catalog and corrective actions are in the [CLI interface specification]({{ 'docs/cli/interface.md' | source }}).
