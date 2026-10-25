---
name: spec
description: Extracts approved prototype behavior into capability specifications and shared UI rules in `docs/specs/` and `docs/ui.md`, and maintains them after approved prototype changes. Use `prototype` before specifying new behavior and `work-spec` for a single task. Does not choose architecture/dependencies or write production code.
---

# Spec skill

Framework version: {{ .Version }}

## 1. Purpose and boundaries

- Express approved prototype behavior as requirements under `AGENTS.md` rule 9. Never finalize new or changed
  behavior from user statements, task decisions, code, or draft prototype changes alone.
- You may edit only `docs/specs/*.md`, `docs/ui.md`, and the document index in `docs/README.md`.
- You must not edit `docs/work/`, `docs/decisions.md`, `docs/architecture.md`, `docs/stack.md`, `prototype/`,
  product code, or workflow state.
- A simulated backend action or a prototype library never approves production architecture or a dependency.
  Reuse applicable approved decisions. Record only unresolved production implications relevant to the selected
  behavior as **Open** items with `needs decision: yes`; do not make architecture or dependency choices here.
- Never rewrite requirements to match an accidental prototype or code defect; report the defect.

## 2. Modes

| Mode | Where | Input | Commits |
|---|---|---|---|
| Discovery | Main checkout | Approved prototype handoff references; existing code for observations only | Yes (section 7) |
| In task | Assigned feature worktree (`feature/<ID>`) | Approved decisions or the work spec's "Spec changes", backed by approved prototype references for behavior | No; the owning orchestrator commits |

- Before editing, inspect `git rev-parse --show-toplevel`, `git branch --show-current`,
  `git worktree list --porcelain`, and `git status --short`; preserve unrelated edits.
- Discovery requires the primary checkout on a named branch, outside `feature/*` and `workflow-state`.
  If repository rules forbid commits on the current branch, use `spec/<topic>` there and tell the user.
- In-task use belongs only to the owning orchestrator. Confirm the registered task, owner, branch, and assigned
  worktree with `solomonar task status <ID>` and the Git inspection above. Edit only the listed approved
  decisions or user-confirmed "Spec changes"; never update the work spec through this skill.
- A location, owner, or input mismatch: stop before editing and report it.

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

## 4. Procedure, discovery mode

1. Start from the user's outcome or handoff references and `docs/README.md`. Load only relevant capability
   specs, shared UI rules, and scoped decision entries. Read architecture/stack entries only when needed to
   resolve an implication; expand context for discovered dependencies or shared contracts.
2. For each handoff, use `git show "<SHA>:prototype/README.md"` to check the selected flow's ID, entry file,
   `ready for spec` status, and explicit approval covering the selected scenarios. Read its entry file and
   supporting files at that same SHA, never from the working copy. Entry-file paths in the README are relative
   to `prototype/`. Select simulations/questions by their flow IDs, plus
   relevant project-wide entries. Missing references, draft/stale approval, or unrepresented behavior: stop that
   part and return it to `prototype`; direct confirmation of a spec rule cannot bypass this check.
3. Extract only demonstrated, approved behavior per flow: actions/inputs, outputs, transitions, validation,
   business constraints, relevant success/failure/edge cases, and persistence/integration outcomes. A
   `SIMULATED:` step yields only its observable outcome (e.g. "a placed order is still listed after reload"); the
   mechanism (e.g. `localStorage`) stays evidence, subject to section 1's decision boundary.
4. Map items to capabilities: reuse existing files; create a file only for an independently loadable area; put
   cross-capability layout and interaction rules in `docs/ui.md`. When the split is unclear, propose one and ask
   per `AGENTS.md` rule 14, with the proposal as the recommended option.
5. Mark faithfully extracted approved behavior **Agreed**, citing its approval and prototype reference. Keep
   existing unconfirmed observations **Observed**; promote them only after representation and prototype approval.
   A readiness label or intentional-change entry alone does not approve inferred rules or additional behavior.
6. On a prototype/spec conflict, leave the affected requirement unchanged and record an **Open** item citing
   both sources. Resolve it through `prototype` first: restore approved behavior for a defect, or demonstrate
   and obtain approval for a change. Resume only with the resolved approved snapshot; never choose silently.
7. Surface ambiguous or missing behavior for a decision, blocking questions first. Send it to `prototype` for
   demonstration/approval before finalizing the related rule; continue independent approved extraction only.
8. Delete an **Open** item that an approved `P/D-n` resolves and cite that decision under "References".
   Create or update files and validate them per section 6. Put prototype references under "References".
9. Commit (section 7) and report: files changed; rules per label per file; open questions, blocking first;
   prototype defects found; next action: `work-spec` for a selected change, `prototype` for blocking unknowns,
   or answering remaining project open questions (`rules.md` section 3).

## 5. Procedure, in-task mode

1. Read the listed decisions in `docs/decisions.md` (only status `approved`; a `pending` entry approves nothing)
   or the work spec's "Spec changes" line, and the spec rules they affect. Apply section 4 step 2's prototype
   check to every behavioral change; without that handoff, report it to the orchestrator and leave it unchanged.
2. Apply each authorized change in the rule's primary document, using section 3's IDs, labels, and sources.
   Assign final IDs to `<capability>/new` entries; delete the **Open** items the change resolves. Use section 4
   step 6 for conflicts; a task decision does not authorize resolving one solely in the specification.
3. Leave everything else unchanged, including **Observed** rules the input does not cover.
4. An input that is unclear for a rule, lacks a decision or user source, or contradicts another **Agreed** rule:
   change nothing for it; report it.
5. Apply section 6's document-creation and validation steps. Report changed files, rule IDs, `new` and split-ID
   mappings, unresolved inputs, and required reference fixes outside scope to the orchestrator. It updates the
   work spec and commits; this mode never commits.

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

- Messages: `docs(<capability>): <summary>`; `docs(ui): <summary>` for `docs/ui.md`. One commit per file when
  practical; include the related `docs/README.md` index change.
- Commit only intended changes allowed by section 1, selected by path, on the branch established at startup.
  Inspect the staged diff; preserve unrelated staged changes. Skip the commit when nothing relevant changed.
- Never push unless the user asks.
