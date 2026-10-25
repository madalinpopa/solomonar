---
title: Run your first task
navTitle: Run your first task
description: Start an agent session with the solomonar skill, prototype a flow, agree its rules, and follow one task from registration through review, merge, and closure.
kind: Tutorial
group: Start here
order: 4
sources:
  - assets/framework/skills/solomonar/SKILL.md
  - assets/framework/skills/prototype/SKILL.md
  - assets/framework/agents/orchestrator.md
  - assets/framework/rules.md
  - internal/cli/run.go
---

You talk to an agent session; the agents do the work; the CLI records it. This page follows one feature of the `ledger` project from [Set up a project]({{ '/docs/set-up-a-project/' | url }}): adding a shared expense.

Two kinds of input appear below. **Agent prompts** go into your configured agent session started in the project directory. **Terminal commands** go into your shell.

## Open a session

Start a new top-level session in the project. The entry point is the `solomonar` skill:

| Runtime | How to call it |
| --- | --- |
| Claude Code | `/solomonar` followed by your request |
| Codex | `$solomonar` followed by your request |
| GitHub Copilot CLI | `/solomonar` followed by your request |
| Any runtime | “Use the solomonar skill.” followed by your request |

With no request, the skill reports what is waiting on you and what can start. With a request, it hands over to the skill or role that owns it, in the same session.

## Try the flow as a prototype

Demonstrate new or changed behavior before specifying and implementing it:

```text
/solomonar Prototype adding a shared expense. Let me try the empty list,
a valid expense, and an invalid amount. Use fictional data.
```

The `prototype` skill builds an independently runnable flow under `prototype/`. For a web flow, use its documented browser URL or entry file; CLI, module/library, and service flows provide commands, consumer examples, or requests. Storage and external services are simulated. Its README gives exact run, inspection, and reset instructions.

A prototype can cover just one function, component, or module. Independent parts can live under `prototype/cli/`, `prototype/web/`, `prototype/module/`, or `prototype/service/`. Each part has an exact run command and tests or inspection scenarios with expected results; a small harness runs code that has no entry point.

See [Prototype workflow]({{ '/docs/prototype/' | url }}) for example layouts, commands, review records, and the handoff into specifications.

Try it and say what should change. Intentional changes are recorded; accidental defects are fixed. The prototype never decides your stack or architecture.

An existing prototype follows the same loop: update it, review and approve the behavior, update specs, then implement. Task or implementation-plan approval cannot replace prototype approval.

## Agree the rules

When the flow feels right, say so. The prototype hands off to the `spec` skill, which writes the behavior to `docs/specs/`:

```text
/solomonar The add-expense flow is ready for the spec. I confirm that an
invalid amount shows an error and adds nothing.
```

The `spec` skill extracts approved behavior as **Agreed** rules, citing the prototype revision and your approval. Existing unconfirmed observations remain **Observed**; anything ambiguous returns to the prototype for a decision and demonstration. Agents implement and review against agreed rules only.

## Start the task

Ask for the outcome, with the boundaries that matter:

```text
/solomonar Implement the agreed add-expense flow as Ledger's first
runnable slice. A saved expense must still be listed after a reload.
Exclude accounts and payments.
```

This session now takes the **orchestrator** role for one task. It registers the task, tells you **Registration confirmed**, and creates a branch and a separate worktree at `../ledger-worktrees/W-001`. You can open another session for an independent task once you see that confirmation.

In the worktree, the `work-spec` skill writes the task's outcome, scope, acceptance criteria, and checks to `docs/work/`. It asks only the questions that block the work, through your agent's question prompt when the harness provides one.

The work spec ends with an implementation plan, one line per intended commit, such as `S-1: feat(W-001): store expenses`. The orchestrator shows it to you and waits:

```text
I approve the implementation plan for W-001.
```

Only then does implementation start. The implementer commits each step separately, so the plan and its commits show exactly what is done.

## Watch it from the terminal

The registry tells you where every task stands. In the project directory:

```sh
solomonar task show
```

```text
ID     STATUS     STAGE          OWNER           NEXT
W-001  running    specification  orchestrator-1  orchestrator-1: Write the work spec
registry refs/heads/workflow-state @ 2d4f6a8 · 2026-10-20T15:12:40Z
```

For one task, `task status` compares the record with the real branch, worktree, and pull request, and lists the commands that are valid next:

```sh
solomonar task status W-001
```

Both commands only read. Agents write the record through the CLI, which refuses any step the record cannot support: implementation cannot start without an approved plan, and a review needs every plan step committed and a clean revision on the task branch. Once implementation starts, `task status` also shows which plan steps are committed and which remain.

## Answer decisions

When the work needs a choice that has not been approved, such as how to store expenses, the orchestrator stops that part of the work and asks you. Each decision has an ID like `W-001/D-1`:

```text
For W-001/D-1, I approve SQLite for local storage.
Keep cloud sync outside this task.
```

Answer in the session that asked. Silence never counts as approval, and neither does a prototype's simulated storage.

## Review and merge

The implementer builds the change and hands over a committed revision. The orchestrator then sends it to reviewers matched to its risk:

- Documentation-only changes need only CI checks.
- Normal code gets the security, correctness, and spec reviewers.
- Persistence, authentication, concurrency, external integrations, and migrations also face an adversarial reviewer. Saving expenses lands here.

Confirmed problems go back to the implementer, and the new revision is reviewed again. Once reviews pass, the orchestrator pushes the branch, opens the pull request, waits for the required checks, and brings the branch up to date with main. It then closes the task: it commits the final work spec if it changed, and a last commit removes the task's file from `docs/work/`. The orchestrator merges with a merge commit, never squash or rebase. It asks for no extra approval, but approvals your repository requires still apply. Then it runs `task done`, which verifies the merge and the final spec before writing the task as `done`.

```sh
solomonar task show W-001
```

The finished record keeps a summary, the merge commit, the pull request, and a reference to the final work spec, which stays in Git history.

## If something interrupts you

Sessions end. Your work does not:

```text
/solomonar Resume W-001.
```

The new session takes the task over on your instruction, reads the record, the approved plan, and its commits, and inspects the worktree and pull request. It confirms the task, then delegates only the steps that remain; uncommitted work is inspected, not overwritten. It does not start a second branch. Make sure the old session has stopped before you ask. To stop on purpose, say “Pause W-001”.

## Next

You have seen the whole loop once. [How the framework works]({{ '/docs/framework/' | url }}) explains why it is shaped this way. Use [Prototype workflow]({{ '/docs/prototype/' | url }}) for runnable behavior examples and [Task workflow]({{ '/docs/use-cases/' | url }}) for implementation, bug fixes, parallel tasks, and recovery.
