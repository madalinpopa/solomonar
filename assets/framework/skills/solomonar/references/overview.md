# Project overview

Framework version: {{ .Version }}

Read-only route for a general status or next-step request. Do not load task-execution procedures.

1. Run `solomonar task show --json`. On `not_initialized`, report it and suggest `solomonar init`; on other
   failures, report the CLI diagnostic without suggesting reinitialization or repairing state. Stop on failure.
2. List each task once: user `next_action.actor` first, with the action; then remaining `blocked`, `running`,
   and `reserved` tasks with owner, stage, and blockers; `paused` tasks with owner and next action; `planned`
   tasks; counts of `done` and `cancelled` tasks. Records show what orchestrators last wrote, not whether
   their sessions are alive; use `solomonar task status <ID>` when actual Git facts are needed.
3. Find project open questions (`rules.md` section 3) without loading whole documents:
   `grep -nE '^- [a-z0-9-]+/Q-[0-9]+ ·' docs/specs/*.md docs/ui.md` and
   `awk '/^## /{h=$0} /^- Status: pending/ && h ~ /^## P\/D-/{print h}' docs/decisions.md`.
   List them, blocking first, and suggest that the user answer them; answering follows the project
   open-question route.
4. End with at most two next steps consistent with ownership and dependencies. Name the task and owning
   session where applicable. Do not treat an overview request as permission to resume paused or cancelled work.
