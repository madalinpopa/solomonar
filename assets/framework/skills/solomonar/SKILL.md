---
name: solomonar
description: The user's entry point to the Solomonar process. Routes a request to the one skill or role that owns it, in this session; with no request, shows what waits on the user and what can start. Invoked only by the user; never by another skill, a role, or a delegated subagent.
---

# Solomonar skill

Framework version: 0.1.0

## 1. Purpose and boundaries

- Classify the user's request and load the one skill or role that owns it (section 2). This skill owns no
  procedure: once a file is loaded, that file governs and this skill adds nothing.
- Only the user invokes it. If you are a delegated subagent, stop and tell the invoker.
- You must not edit files, write workflow state, or run a writing `solomonar` command. You may run the read-only
  `solomonar task show`, `solomonar task status`, and `solomonar extension list`.
- The orchestrator delegates implementation and reviews, so it runs only as the top-level session: you take the
  role in this session (section 4); you never delegate it.

## 2. Route

Read the request from the skill's arguments, else from the user's message. If it fits two routes, ask one
question. Load only the routed file and the files its first lines require.

| Request | Route |
|---|---|
| None, "status", "what's next" | Overview (section 3) |
| Start a product; explore, demonstrate, or change a prototype flow | Load `docs/solomonar/skills/prototype/SKILL.md` |
| Prototype handoff references, flows ready for spec, or a product behavior the user decided outside a task | Load `docs/solomonar/skills/spec/SKILL.md`, discovery mode, in the main checkout |
| A new task, a `planned` ID, or resume, pause, or cancel a task | Take the orchestrator role (section 4) |
| An answer to `W-NNN/D-n`, or a scope or AC change for a running task | The owning orchestrator's session handles it: tell the user to answer there. If that session ended, take the orchestrator role to resume the task (section 4) |
| Adopt or drop a language, library, or practice | Show `solomonar extension list`; propose `solomonar extension add\|remove <name>` for the user to run. The user's command records the decision; never run it yourself |
| A product code change outside any task | Propose a managed task: `solomonar task plan --title <t>` for later, or take the orchestrator role now. Work outside the process only if the user says so |
| A question about the project or the process | Answer it from `AGENTS.md` context routes; no route needed |

## 3. Overview

1. Run `solomonar task show --json`. If it fails because the project is not initialized, say so, propose
   `solomonar init` or `solomonar doctor`, and stop.
2. Report, in this order: tasks whose `next_action.actor` is the user, with the action; `blocked` and `running`
   tasks with owner, stage, and blockers; `planned` tasks; the number of `done` tasks. The record shows what
   orchestrators last wrote, not whether their sessions are alive; `solomonar task status <ID>` shows Git facts.
3. End with at most two next steps, e.g. "Answer W-001/D-1 in orchestrator-1's session" or "Start W-002: open a
   new top-level session and run `/solomonar work on W-002`".

## 4. Taking the orchestrator role

1. One task per orchestrator session. If this session already holds the orchestrator role for another task,
   stop: tell the user to open a new top-level session and invoke this skill there with the same request.
2. Load `docs/solomonar/agents/orchestrator.md` and every file its first lines require.
3. Start at its section 2 "Session start" with the request: the task ID, the title of new work, or the prototype
   selection. The orchestrator procedure governs from here.

## 5. Never

- Never edit a file, write state, or run a writing `solomonar` command.
- Never delegate the orchestrator role, or hold two tasks in one session.
- Never answer, approve, or record a decision on the user's behalf, or relay it to another session as approval.
- Never load more than the routed file and the files it requires.
