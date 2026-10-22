---
title: How the spec-driven framework works
navTitle: How it works
description: Learn how Solomonar connects specifications, agent skills, orchestration, independent code review, and Git integration in a language-agnostic framework.
kind: Explanation
group: Learn the framework
order: 5
sources:
  - assets/framework/rules.md
  - assets/framework/skills/solomonar/SKILL.md
  - assets/framework/agents/orchestrator.md
---

Solomonar is a way of working, rather than a technology stack. It connects product discovery, agreed behavior, focused implementation, independent review, and integration. The core remains language-agnostic; extensions supply knowledge of the project's chosen technologies.

The names come from a Romanian folktale. The mapping below is the project's interpretation of that story. These are the same five diagrams used in the repository README, rendered with this site's theme.

If you followed [Run your first task]({{ '/docs/first-task/' | url }}), you have already used every part described here. This page explains why each part exists.

## The full cycle at a glance

{% diagram "cycle" %}

Discovery and task preparation lead into implementation, risk-based reviews, integration, and confirmed closure. Corrections return to implementation; they do not create another task.

## The book: specifications

Specifications keep the project's intent readable and reviewable. Canonical capability specifications describe current behavior; a work specification defines one change and its acceptance criteria.

The `spec` skill distinguishes three kinds of information:

- **Observed:** behavior seen in a prototype or implementation. It is evidence, not approval.
- **Agreed:** requirements confirmed by a user decision. These guide implementation and review.
- **Open:** unresolved questions that still need an answer.

The `work-spec` skill prepares one task's outcome, boundaries, criteria, context, and checks. It does not rewrite the whole application's specification.

## The spells: skills

A skill is a focused procedure, loaded when it is useful.

| Core skill | Responsibility |
| --- | --- |
| `solomonar` | User-invoked entry point; routes the request to the right skill or role in the same session. |
| `prototype` | Explore a useful flow with static HTML, CSS, JavaScript, and sample data. |
| `spec` | Maintain application behavior, preserving the distinction between observed and agreed information. |
| `work-spec` | Prepare or refine one registered task before implementation. |

The entry skill does not write the project or delegate an orchestrator. A managed-task request makes the current top-level session take the orchestrator role.

## The dragons: agents

An agent has a responsibility; a skill gives it a procedure. The orchestrator delegates implementation and the required reviews, then consolidates the results.

| Role | Responsibility |
| --- | --- |
| Implementer | Build the authorized change, run relevant checks, and hand off a candidate revision. |
| Security reviewer | Check defects, error handling, resources, trust boundaries, input, and secrets. |
| Correctness reviewer | Check business rules, invariants, calculations, transitions, and relevant boundary cases. |
| Spec reviewer | Check the work spec, scope, shared contracts, and applicable installed extension rules. |
| Adversarial reviewer | For tier 2 changes, try to break the integrated feature after specialist blockers are closed. |

The core reviewers are language-agnostic. For example, Go rules belong in the `go` extension, which the implementer follows and the Spec reviewer checks.

## The reins: orchestration

One top-level orchestrator owns one task. It prepares context, delegates the work, gathers reports, manages decisions, and coordinates integration.

The distinction matters: **the orchestrator decides; the CLI checks.** The CLI validates task updates against the record and Git. It does not decide the next task, run agents, call model providers, or store review verdicts.

{% diagram "task-loop" %}

The orchestrator is the hub. Implementation and specialist reviews report back to it; tier 2 adds adversarial review after specialist blockers are closed.

## Sequential starts, parallel work

{% diagram "parallel" %}

Register each task before starting the next one. Independent tasks then work in separate worktrees, while merges into main happen one at a time.

## Two axes: status and stage

{% diagram "status-stage" %}

Status says whether the task can advance; stage says where it has reached. A blocked or paused task keeps its stage. Only a confirmed merge leads to `done / complete`.

## Close the task, keep the history

{% diagram "closure" %}

The final spec has its own commit. A separate closure commit removes the active work-spec file, and a merge commit preserves both in main's history. The compact task record keeps the references.

## The order of authority

When recommendations disagree, use this order:

1. The user's explicit decision, within its recorded scope.
2. Approved global specifications and mandatory rules, including applicable installed extension `must` rules.
3. The task's approved work specification.
4. Compatible agent recommendations supported by a concrete benefit.

Two approved requirements that contradict each other need a user decision. Implementation quirks and accidental prototype behavior do not silently become new requirements.

## A light core, extended deliberately

The core contains the workflow's rules, skills, and required roles. Extensions add language, library, practice, or specialist-review instructions.

`solomonar init` copies the core into your project under `docs/solomonar/` and creates small runtime adapters that point to that portable source. Adopting a stack extension records a decision; it does not install application packages or approve further dependencies.

In v0.1.2, extensions are chosen at `init`, and the catalog holds the `go` extension. The `extension` commands that change them later are planned. The [project files reference]({{ '/docs/configuration/' | url }}) shows what is installed today.
