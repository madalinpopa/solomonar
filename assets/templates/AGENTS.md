# Project instructions

Framework version: {{ .FrameworkVersion }}. Framework files: `docs/framework/`.

## Language

Write framework artifacts, specs, reports, commit messages, and PR text in English.

## Load your context

- Load your role from `docs/framework/agents/<role>.md` and every file its first lines require. A new or
  resumed managed task takes the `orchestrator` role, which the user starts as a top-level session, never as a
  delegated subagent; other roles start only from an orchestrator packet.
- Load only: the invoked skill, the assigned work spec, the relevant global rules (Context routes), the
  referenced documents, and the affected code and tests. Expand only for a discovered dependency, shared
  contract, or security rule.
- With no work spec, start from `docs/README.md` and the user's selected outcome.
- Orchestrators also load `docs/framework/rules.md`; other roles do not. Every role loads
  `docs/framework/report.md`.
- A document index never loads its targets automatically.

## Skills

Load `docs/framework/skills/<skill>/SKILL.md` before using a skill.

- `prototype`: start a product, or explore or change a flow in `prototype/`.
- `spec`: record a prototype handoff or a user decision in `docs/specs/<capability>.md` and `docs/ui.md`.
- `work-spec`: specify one registered task in `docs/work/<ID>-<slug>.md` (orchestrator, `specification` stage).

## Context routes

- Product behavior: `docs/specs/<capability>.md`; cite rules by ID `<capability>/R-n`.
- Task requirements: `docs/work/<ID>-<slug>.md` in the task worktree.
- Structure and contracts: `docs/architecture.md`.
- Dependencies: `docs/stack.md` and the manifests.
- UI: `docs/ui.md`, the relevant capability, and the prototype flow (`prototype/README.md`).
- Approvals: the relevant `approved` entries in `docs/decisions.md`.
- Run and check commands: `docs/README.md` "Commands".
- Execution state: `solomonar task status <ID>` (record, Git facts, drift), `solomonar task show [<ID>]`, or
  `git show refs/heads/workflow-state:state.json`.
- History: the task's compact record; load its `spec_ref` with `git show` only for a concrete question.

## Mandatory rules

Cite a rule as `AGENTS.md rule n`.

1. Follow applicable user decisions, then approved global rules, then the work spec. Escalate contradictions;
   preferences never override requirements. Code and prototype are evidence, not requirements.
2. New architecture decisions and dependency changes require user approval. Investigate first, report through
   the orchestrator, and wait before applying. No response is not approval.
3. Workflow state is written only through `solomonar task` commands: by orchestrators, each for its own task's
   record, and by the user for `task plan`. Never edit, commit, or check out `state.json`.
4. Register the task (`solomonar task register`) before creating its feature worktree.
5. Modify only the files your role or skill allows. In a task, work only in your assigned worktree. Reviewers
   are read-only and review one stable revision.
6. The orchestrator delegates implementation and every review, and consolidates findings. Other roles report
   only to it.
7. Unresolved blockers, missing required reviews, or failed or unperformed required checks prevent advancement.
8. Mark a task `done` only after a confirmed merge into main.
9. Update the canonical project specs before merge. Commit the final task spec separately.
10. Close the task per `docs/framework/rules.md` §9 and keep verifiable Git references.
11. Never commit or report secrets, credentials, or real personal data.
12. Commit messages: English, `<type>(<scope>): <summary>`. Your role or skill file defines the type and scope.
13. Tests and code comments cite spec rule IDs (`<capability>/R-n`), never task or AC IDs, which leave the tree
    at closure.

## Project notes

<!-- Project-specific instructions go here, e.g. adopted style guides (such as Go Code Review Comments),
branch rules for direct commits, and the test policy. This section is owned by the project and preserved by
framework upgrades. -->
