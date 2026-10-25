---
title: solomonar guard
navTitle: solomonar guard
description: The solomonar guard runtime hook denies direct registry writes and unregistered task worktrees from an agent session. In development.
kind: Reference
group: In development
order: 21
inDevelopment: true
sources:
  - docs/cli/interface.md
---

> This command is in development and not in the current build. This page describes the planned behavior; the final output may differ in detail.

The hook contract below covers Claude Code and Codex. The Copilot CLI adapter installs no guard hook.

Reads a tool call from stdin and answers in the runtime's hook format. It denies:

- direct writes to the registry branch `workflow-state`, its reflog, or `packed-refs`;
- Git commands that check out, reset, commit to, update, or push `workflow-state` other than through `solomonar`;
- `git worktree add` for a `feature/<ID>` branch whose task is not registered;
- an agent's `extension add` or `extension remove` of a stack extension without `--decision`.

It never writes. An allowed call produces no output and exits 0. Read-only Git commands on `workflow-state` and product files that happen to be named `state.json` are allowed.

{% cliUsage "Never by hand. `init` and `runtime add` register it as the runtime's pre-tool hook.", "The runtime, Claude Code or Codex, before each tool call of an agent. The task commands validate every write regardless; the guard stops a mistake earlier." %}

## Synopsis

```text
solomonar guard --runtime <claude|codex>
```

## Flags

| Flag | Meaning |
| --- | --- |
| `--runtime` | Required. The hook payload format: `claude` or `codex`. |

## Example

The reason an agent sees when it tries to write the registry directly:

```text
solomonar guard: denied. The registry is written only through `solomonar task` commands (AGENTS.md rule 3).
Use: solomonar task move W-001 --as orchestrator-1 --stage <stage>   (see: solomonar task status W-001)
```
