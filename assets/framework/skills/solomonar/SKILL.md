---
name: solomonar
description: User-invoked entry point to Solomonar. Routes requests to the owning skill or role; with no request, reports task priorities and next steps. Never invoked by another skill, role, or delegated subagent.
---

# Solomonar skill

Framework version: 0.3.5

## 1. Purpose and boundaries

- Only the user invokes it. If you are a delegated subagent, stop and tell the invoker.
- While routing or reporting status, do not edit files, write workflow state, or run writing commands. Use
  read-only commands: `solomonar help`, `solomonar task show`, `solomonar task status`, and, if available,
  `solomonar extension list`.
- Never answer, approve, or record a decision on the user's behalf, or relay it to another session as approval.
- Load only the routed file and the files it requires. Once selected, that skill or role governs its procedure
  and write scope; this router adds no further instructions.

## 2. Route

Read the request from the skill's arguments, else from the user's message. Use explicit intent and the current
task/session context to resolve overlapping routes. Ask one question only if the owning route remains unclear.

| Request | Route |
|---|---|
| None, or general "status" / "what's next" | Overview (section 3) |
| A task ID without an action, or a request for its status | Run `solomonar task status <ID>`; report facts without starting or resuming work |
| Start a product; explore, demonstrate, or change CLI, web, module/library, or service behavior | Load `docs/solomonar/skills/prototype/SKILL.md`; for an owned task, its orchestrator coordinates the handoff under `AGENTS.md` rule 9 |
| Approved prototype handoff references or flows ready for spec | Load `docs/solomonar/skills/spec/SKILL.md`, discovery mode, in the main checkout |
| A new task, starting `planned` work, or resume, pause, or cancel a task | Take the orchestrator role (section 4) |
| Task requirements, an answer to `W-NNN/D-n`, or a scope or AC change | Continue in this session if it owns the task; otherwise direct the user to the owning session. An explicit request to resume here follows section 4 |
| Adopt or drop a language, library, or practice | In-task: owning orchestrator. Otherwise check `solomonar help`; if it lists `extension`, show `solomonar extension list` and propose `solomonar extension add\|remove <name>` for the user to run. Otherwise report that extension management is unsupported by this CLI and stop |
| A product code change outside any task | Propose a managed task: `solomonar task plan --title <t>` for later, or take the orchestrator role now. Work outside the process only if the user says so |
| A question about the project or the process | Answer it from `AGENTS.md` context routes; no route needed |

## 3. Overview

1. Run `solomonar task show --json`. On `not_initialized`, report it and suggest `solomonar init`; on other
   failures, report the CLI diagnostic without suggesting reinitialization or repairing state. Stop on failure.
2. List each task once: user `next_action.actor` first, with the action; then remaining `blocked`, `running`,
   and `reserved` tasks with owner, stage, and blockers; `paused` tasks with owner and next action; `planned`
   tasks; counts of `done` and `cancelled` tasks. Records show what orchestrators last wrote, not whether
   their sessions are alive; use `solomonar task status <ID>` when actual Git facts are needed.
3. End with at most two next steps consistent with ownership and dependencies. Name the task and owning
   session where applicable. Do not treat an overview request as permission to resume paused or cancelled work.

## 4. Taking the orchestrator role

1. One task per orchestrator session. If this session already holds the orchestrator role for another task,
   stop: tell the user to open a new top-level session and invoke this skill there with the same request.
2. Take the role in this top-level session; never delegate it. Load `docs/solomonar/agents/orchestrator.md` and
   its required files, or continue the loaded role when this session already owns the task.
3. For a new or resumed session, start at its section 2 "Session start" with the task ID, new work's title, or
   prototype selection. Follow its ownership and recovery rules; another recorded owner requires explicit
   user handoff. An ended session or an old record alone never authorizes takeover.
4. Resumed work continues from the approved plan and its checkpoint commits, never from the previous
   conversation: after the handoff, inspect, `task confirm`, and delegate only the remaining steps
   (`rules.md` section 10).
