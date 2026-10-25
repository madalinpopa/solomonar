---
name: spec
description: Extracts approved prototype behavior into capability specifications and shared UI rules in `docs/specs/` and `docs/ui.md`, and maintains them after approved prototype changes. Use `prototype` before specifying new behavior and `work-spec` for a single task. Does not choose architecture/dependencies or write production code.
---

# Spec skill

Framework version: {{ .Version }}

## 1. Purpose and boundaries

- Express approved prototype behavior as requirements under `AGENTS.md` rule 7. Never finalize new or changed
  behavior from user statements, task decisions, code, or draft prototype changes alone.
- You may edit only `docs/specs/*.md`, `docs/ui.md`, and the document index in `docs/README.md`.
- You must not edit `docs/work/`, `docs/decisions.md`, `docs/architecture.md`, `docs/stack.md`, `prototype/`,
  product code, or workflow state.
- A simulated backend action or a prototype library never approves production architecture or a dependency.
  Reuse applicable approved decisions. Record only unresolved production implications relevant to the selected
  behavior as **Open** items with `needs decision: yes`; do not make architecture or dependency choices here.
- Never rewrite requirements to match an accidental prototype or code defect; report the defect.

## 2. Modes

Before editing, inspect `git rev-parse --show-toplevel`, `git branch --show-current`,
`git worktree list --porcelain`, and `git status --short`; preserve unrelated edits.
Load exactly the selected mode before acting; location, owner, or input mismatch stops editing.

| Mode | Load | Input |
|---|---|---|
| Discovery, primary checkout | [references/discovery.md](references/discovery.md) | Approved prototype handoffs; code supports observations only |
| In task, owning orchestrator's worktree | [references/in-task.md](references/in-task.md) | Approved decisions/Spec changes backed by approved behavior; no commits |


## 3. Labels and rule IDs

- `<capability>` is the spec file's base name, kebab-case (`orders` for `docs/specs/orders.md`); `ui` for `docs/ui.md`.
- Each rule is one line under "Rules": `<capability>/R-<n> · **<label>** · <statement> · <source>`.
  - `**Observed**`: existing behavior not yet approved; never an implementation requirement. Source:
    `<full SHA>:<path>`, appending `, flow <flow-id>` for prototype evidence. Do not infer missing behavior.
  - `**Agreed**`: represented in an explicitly approved prototype flow. Source: `user YYYY-MM-DD` or the scoped
    decision ID, plus `<full SHA>:prototype/<entry file>, flow <flow-id>`. Reuse existing approval sources/dates;
    retain legacy agreed rules unchanged until their behavior changes.
- **Open** items live only under "Open questions", labeled by that section:
  `<capability>/Q-<n> · <question> · blocks: <rule/flow IDs or "none"> · needs decision: yes|no`.
  `yes` = answering needs an architecture or dependency decision; `no` = the user can answer it directly.
- IDs are stable: a changed rule keeps its ID; an ID is never reused (new `n` = highest ever used + 1; check the
  file's Git history when rules were deleted); a removed rule is deleted, Git keeps history.
- A rule moved to another file (split) gets a new ID there; retire the old ID. Report `old → new` and include
  the mapping in the discovery commit body or pass it to the orchestrator in-task. Active tasks update their
  work-spec references when they next merge main. Reference fixes stay within section 6's scope.
- Rules state observable behavior: what a user or client sees or can do, data shown, validation, states,
  transitions, permissions. Never implementation: no storage mechanism, framework, or simulated backend detail.
  New rules use EARS form where it fits: "When <trigger>, the system shall <response>" or "If <condition>, then
  the system shall <response>".
- Each rule has one primary document. Other documents cite its ID; they never copy it.

## 4. Prototype approval and conflicts

### Approval check (both modes)

For each handoff, use `git show "<SHA>:prototype/README.md"` to check the selected flow's ID, entry file,
   `ready for spec` status, and explicit approval covering the selected scenarios. Read its entry file and
   supporting files at that same SHA, never from the working copy. Entry-file paths in the README are relative
   to `prototype/`. Select simulations/questions by their flow IDs, plus
   relevant project-wide entries. Missing references, draft/stale approval, or unrepresented behavior: stop that
   part and return it to `prototype`; direct confirmation of a spec rule cannot bypass this check.

### Conflict procedure (both modes)

On a prototype/spec conflict, leave the affected requirement unchanged and record an **Open** item citing
   both sources. Resolve it through `prototype` first: restore approved behavior for a defect, or demonstrate
   and obtain approval for a change. Resume only with the resolved approved snapshot; never choose silently.

Discovery procedure: [references/discovery.md](references/discovery.md), loaded only in that mode.

## 5. Procedure, in-task mode

Load [references/in-task.md](references/in-task.md) only in that mode.

## 6. Writing and validation (both modes)

1. Load `docs/solomonar/skills/spec/template.md` only when creating a document, including `docs/ui.md`.
   Add each new document to `docs/README.md`'s Documents index once; if the README is absent, create it with
   only that index and say so. Preserve existing content.
2. Repair references only in allowed files and within the authorized scope. Report needed changes in code,
   tests, work specs, or other documents to their owner; do not expand your write scope.
3. Check the affected documents:
   - IDs are unique in their namespace, stable, and never reused (section 3).
   - Each rule has a valid label and supporting source; source references resolve at their recorded revision,
     and **Agreed** sources authorize the stated behavior.
   - Question `blocks` entries resolve to rule/flow IDs or `none`; resolved questions and changed references agree.
   - No unresolved contradiction remains between affected agreed rules; behavior stays within authorized scope.
   - New documents are indexed; no template hints or unfinished placeholders remain.
4. Run `git diff --check` and inspect the final diff. Report unresolved validation gaps before handoff or commit.

## 7. Commits (discovery mode)

Discovery commits follow [references/discovery.md](references/discovery.md#commits). In-task mode never commits.
