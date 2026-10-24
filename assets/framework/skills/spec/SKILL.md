---
name: spec
description: Maintains capability specifications and shared UI rules in `docs/specs/` and `docs/ui.md` after prototype handoffs or explicit behavior decisions. Use `work-spec` to describe a single task; this skill does not choose architecture or dependencies or write production code.
---

# Spec skill

Framework version: 0.3.1

## 1. Purpose and boundaries

- Specs state the current agreed product behavior. The prototype and code are evidence, never requirements.
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
| Discovery | Main checkout | Prototype handoff references or user statements | Yes (section 7) |
| In task | Assigned feature worktree (`feature/<ID>`) | Approved decisions `W-001/D-n`, or the work spec's "Spec changes" line with sources `user YYYY-MM-DD` | No; the owning orchestrator commits |

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
  - `**Observed**`: seen in the prototype, not yet confirmed. Source: `<full SHA>:<path>, flow <flow-id>`.
    Brownfield: existing code at `<full SHA>:<path>` is also a valid source.
  - `**Agreed**`: explicitly confirmed by the user or covered by a scoped approved decision. Source:
    `user YYYY-MM-DD` or the decision ID (e.g. `W-001/D-1`). Reuse existing confirmation sources and dates.
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
   and status. Read its entry file and loaded assets at that same SHA, never from the working copy. Entry-file
   paths in the README are relative to `prototype/`. Select simulations/questions by their flow IDs, plus
   relevant project-wide entries. Missing references: report and stop for that flow; not `ready for spec`:
   ask before using it.
3. Extract features and observable behavior per flow, including failure, empty, and permission states. A
   `SIMULATED:` step yields only its observable outcome (e.g. "a placed order is still listed after reload"); the
   mechanism (e.g. `localStorage`) stays evidence, subject to section 1's decision boundary.
4. Map items to capabilities: reuse existing files; create a file only for an independently loadable area; put
   cross-capability layout and interaction rules in `docs/ui.md`. When the split is unclear, propose one and ask.
5. Label observations **Observed** by default. Clear user statements and applicable approved decisions can
   establish **Agreed** rules directly; do not reconfirm them. An intentional-change entry counts only when its
   reason cites a user decision. A flow's readiness alone does not approve every inferred rule.
6. Conflicts: an observation that contradicts an **Agreed** rule must not overwrite it, even when recorded as an
   intentional change. Add an **Open** item citing the rule ID and the prototype reference, and ask the user. If
   the user calls the difference accidental, it is a prototype defect: report it for the `prototype` skill.
7. Confirm only newly inferred or ambiguous behavior, one decision per message, blocking questions first;
   wait for each answer. One capability's numbered list of **Observed** rules counts as one question
   (confirm, correct, or drop each).
   Confirmed or corrected → **Agreed**, source `user <today>`. Dropped → delete. Unsure → **Open** item. The
   confirmation lives in the rule's source, never in `docs/decisions.md`.
8. Create or update files and validate them per section 6. Put prototype references under "References".
9. Commit (section 7) and report: files changed; rules per label per file; open questions, blocking first;
   prototype defects found; next action: `work-spec` for a selected change, or `prototype` for blocking unknowns.

## 5. Procedure, in-task mode

1. Read the listed decisions in `docs/decisions.md` (only status `approved`; a `pending` entry approves nothing)
   or the work spec's "Spec changes" line, and the spec rules they affect.
2. Apply each authorized change in the rule's primary document, using section 3's IDs, labels, and sources.
   Assign final IDs to `<capability>/new` entries; delete the **Open** items the change resolves.
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
