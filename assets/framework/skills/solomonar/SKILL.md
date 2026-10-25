---
name: solomonar
description: User-invoked entry point to Solomonar. Routes requests to the owning skill or role; with no request, reports task priorities and next steps. Never invoked by another skill, role, or delegated subagent.
---

# Solomonar skill

Framework version: {{ .Version }}

## 1. Purpose and boundaries

- User invocation only; a delegated subagent stops and tells its invoker.
- Routing/status is read-only: no file edits, workflow writes, or automatic resumption. Use `solomonar help`,
  `solomonar task show`, `solomonar task status`, or available `solomonar extension list`.
- Never answer, approve, record, or relay a decision as approval on the user's behalf.

## 2. Route

Read skill arguments, else the user's message. Follow `AGENTS.md` → Request routes, using explicit intent and
current ownership. Ask one question only if ownership/scope remains ambiguous. Load only the selected route
and its dependencies; its procedure and write scope govern after routing.

## 3. Overview

For no arguments or a general status/next-step request, load [references/overview.md](references/overview.md).
A named task status needs only `solomonar task status <ID>`, not the overview or orchestrator.

## 4. Taking the orchestrator role

Take the role in this top-level session, never a subagent. Already owning another task: direct the user to a
new session. Load `docs/solomonar/agents/orchestrator.md` and start at Session start; reuse the loaded role
for this task. Another owner or paused/cancelled work requires explicit handoff/resumption. An ended session
or old record authorizes no takeover. Recovery follows `docs/solomonar/workflow/recovery.md`.
