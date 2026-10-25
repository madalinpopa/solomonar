---
title: How Solomonar compares with other spec-driven tools
navTitle: Compared with others
description: Compare Solomonar with GitHub Spec Kit, Kiro, OpenSpec, and BMAD-METHOD on specifications, parallel work, review, task state, and agent runtimes.
kind: Explanation
group: Learn the framework
order: 8.5
sources:
  - assets/framework/rules.md
  - assets/framework/skills/prototype/SKILL.md
  - docs/cli/interface.md
  - docs/cli/architecture.md
---

## At a glance

| | Solomonar | [GitHub Spec Kit](https://github.com/github/spec-kit) | [Kiro](https://kiro.dev/) | [OpenSpec](https://github.com/Fission-AI/OpenSpec) | [BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD) |
| --- | --- | --- | --- | --- | --- |
| Form | Framework files and a checking CLI | Templates, agent commands, and a scaffolding CLI | IDE and CLI from AWS | Agent commands and a validating CLI | Agent personas, skills, and workflows |
| Agent runtimes | Claude Code, Codex, and GitHub Copilot CLI | Dozens of coding agents | Kiro itself | 30+ coding agents | Several coding agents, including Claude Code, Codex, and Cursor |
| Calls a model itself | No | No; the agent runs the steps | Yes | No; the agent runs the steps | No; the agent runs the steps |
| Specification flow | Prototype, human approval, agreed specs, then one work spec per task for new/changed behavior | Constitution, specify, plan, tasks, implement | `requirements.md` in EARS notation, `design.md`, `tasks.md` | Change proposals with delta specs, archived into living specs | Product brief, PRD, architecture, epics and stories |
| Parallel work | A worktree per task, by rule | Branches through an optional extension; worktrees through community extensions | Parallel subagents and tasks | Not covered | Worktrees recommended in its guides |
| Review | Risk-tiered specialist reviewers, then adversarial review for tier 2 | Consistency analysis across artifacts | Requirements analysis and property-based tests | A verify step against the change | A multi-lens review command, including an adversarial lens |
| Task state | Git ref outside the working branch, compare-and-swap writes | Files under `specs/` | Files under `.kiro/` | Files under `openspec/`, or a separate store (beta) | Files in the repository, or a separate ticket repository |

## What Solomonar focuses on

Most spec-driven tools concentrate on writing good specifications. Solomonar takes that as a starting point and puts its weight on what happens next: running tasks in parallel safely, reviewing them independently, and proving that each step actually happened.

### Progress that Git can confirm

In the other tools, progress is usually a file the agent edits, such as a task list. Solomonar's CLI accepts a task update only when Git supports it: the task's branch and worktree exist, and the merge commit is reachable from main. A task cannot be reported done before its code is merged.

The CLI checks; it never drives. It calls no model, runs no agent, and starts no daemon, so it adds no cost and no lock-in to the runtime you already use.

### Parallel tasks that do not collide

Every task gets its own `feature/<ID>` branch and worktree, by rule rather than by recommendation. The task registry lives on its own Git ref, `refs/heads/workflow-state`, written with compare-and-swap. Several orchestrators can work on different tasks at the same time without overwriting each other's state, and no feature branch carries progress files that conflict at merge.

Spec Kit offers worktrees only through community extensions, and BMAD recommends them in its guides. The other tools keep task state in files that agents edit, by default in the project's own working tree.

### Review as a gate, not a step

Elsewhere, checking is mostly a command you choose to run: Spec Kit analyzes consistency between documents, OpenSpec verifies a change against its specs, and Kiro tests properties derived from requirements. In Solomonar, a task advances only when every required reviewer returns `PASS`:

- Security, correctness, and spec reviewers work independently, in parallel, on the same committed revision.
- Risky changes, such as persistence, authentication, concurrency, or migrations, then get an adversarial review, on a different model when the runtime allows.
- Any new commit is a new revision. An earlier `PASS` never covers new code.

### Behavior before requirements

New or changed behavior is demonstrated in a runnable prototype and explicitly approved before specification and production implementation. A prototype can cover a CLI, web flow, module/library, or service. The approved scenarios define behavior; prototype implementation choices do not select production architecture. The `spec` skill extracts that agreement and keeps unapproved observations separate. See [Prototype workflow]({{ '/docs/prototype/' | url }}) for practical examples.

### Specifications that stay small

A task's work specification lives only as long as the task. At closure, a separate commit removes it, and the compact task record keeps a reference to its final version in Git history. Like OpenSpec's archive, this keeps the specifications agents read focused on current behavior.

### A small core that grows with the project

The core holds only the process: its rules, four skills, and the roles that implement and review. It is tied to no language, library, or stack, so a new project starts without assumptions it would later have to undo. BMAD ships a full product-planning process and Kiro a complete editor from day one; Solomonar starts with the minimum that keeps a task honest.

Everything else arrives as the project needs it. Extensions add the rules of a language, library, or practice, a supporting procedure, or an extra reviewer. Adopting a stack extension records an architecture decision, so the agents' knowledge follows the stack you approved rather than a default. Your own rules join the same way: approved specifications and the `must` rules of installed extensions bind every implementer and reviewer.

The catalog holds the `go` extension and the `extension-authoring` skill. You choose extensions at `init`, add or remove them later with [`solomonar extension`]({{ '/docs/cli/extension/' | url }}), and write project extensions for your own domain with `extension-authoring`.

## When another tool fits better

- You want an editor that runs the models and manages the specs for you: Kiro.
- You want document templates that work with almost any agent and little process: Spec Kit or OpenSpec.
- You want a full planning process with product roles before any code: BMAD-METHOD.
- You want independent reviews, isolated parallel tasks, and task state that Git can verify: Solomonar. Claude Code, Codex, and GitHub Copilot CLI adapters are available.
