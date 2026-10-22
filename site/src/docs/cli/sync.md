---
title: solomonar sync
navTitle: solomonar sync
description: Regenerate Solomonar runtime adapter files from the portable source in docs/solomonar, or check them in CI, with solomonar sync. In development.
kind: Reference
group: In development
order: 18
inDevelopment: true
sources:
  - docs/cli/interface.md
---

> This command is in development and not in the current build; running it today exits 2 with an unknown-command error. This page describes the planned behavior; the final output may differ in detail.

Regenerates the runtime adapter files from the portable source in `docs/solomonar/`, re-renders the catalog extensions whose combined rules depend on the installed set, and updates the "Extensions" section of `AGENTS.md`. Draft extensions are excluded.

{% cliUsage "After you edit a role's or skill's frontmatter in `docs/solomonar/`, delete a stub, or merge a branch that changed `docs/solomonar/`. Edit the source, never a runtime stub: a body-only edit needs no sync, because stubs load the source.", "You. CI runs `sync --check`." %}

## Synopsis

```text
solomonar sync [--check] [--overwrite <path>]... [--dry-run] [--json]
```

## Flags

| Flag | Meaning |
| --- | --- |
| `--check` | Write nothing; exit 5 when any file would change. For CI. |
| `--overwrite` | Replace one hand-edited stub, which is otherwise a conflict (exit 4); one path per flag. |
| `--dry-run` | Print the changes and write nothing. |

## Examples

```sh
solomonar sync
```

```text
  update  .claude/skills/spec/SKILL.md      description changed in docs/solomonar/skills/spec/SKILL.md
  update  .agents/skills/spec/SKILL.md      description changed in docs/solomonar/skills/spec/SKILL.md
  create  .claude/agents/spec-reviewer.md   missing
  unchanged  9 files
Synced 3 files.
```

In CI, before the sync:

```sh
solomonar sync --check
```

```text
stale  .claude/skills/spec/SKILL.md   description differs from docs/solomonar/skills/spec/SKILL.md
error: 1 adapter file is stale (checks_failed)
  fix: solomonar sync
```
