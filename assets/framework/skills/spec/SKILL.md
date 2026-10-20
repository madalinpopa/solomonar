---
name: spec
description: Maintains the application specifications (`docs/specs/<capability>.md`, `docs/ui.md`) as the current agreed product behavior. Use it after a prototype handoff, when an intentional prototype change or a user decision changes product behavior, or when a capability spec must be created or split. Do not use it to describe a single requested change (use `work-spec`), for architecture or dependency choices, or for production code.
---

# Spec skill

Framework version: 0.1.0

## 1. Purpose and boundaries

- Specs state the current agreed product behavior. The prototype and code are evidence, never requirements.
- You may edit only `docs/specs/*.md`, `docs/ui.md`, and the document index in `docs/README.md`.
- You must not edit `docs/work/`, `docs/decisions.md`, `docs/architecture.md`, `docs/stack.md`, `prototype/`,
  product code, or workflow state.
- A simulated backend action or a prototype library never approves production architecture or a dependency.
  Record such implications as **Open** items with `needs decision: yes`; never in architecture or stack docs.

## 2. Modes

| Mode | Where | Input | Commits |
|---|---|---|---|
| Discovery | Main checkout | Prototype handoff references or user statements | Yes (section 7) |
| In task | Assigned feature worktree (`feature/<ID>`) | Approved decisions `W-001/D-n`, or the work spec's "Spec changes" line with sources `user YYYY-MM-DD` | No; the invoking role commits |

- In-task mode edits only what the listed decisions or user-confirmed "Spec changes" cover.
- Location and input match no mode (e.g. a feature worktree without either input): stop and tell the invoker.

## 3. Labels and rule IDs

- `<capability>` is the spec file's base name, kebab-case (`orders` for `docs/specs/orders.md`); `ui` for `docs/ui.md`.
- Each rule is one line under "Rules": `<capability>/R-<n> · **<label>** · <statement> · <source>`.
  - `**Observed**`: seen in the prototype, not yet confirmed. Source: `<full SHA>:<path> — flow <flow-id>`.
    Brownfield: existing code at `<full SHA>:<path>` is also a valid source.
  - `**Agreed**`: confirmed. Source: `user YYYY-MM-DD` or a decision ID (`W-001/D-1`).
- **Open** items live only under "Open questions", labeled by that section:
  `<capability>/Q-<n> · <question> · blocks: <rule/flow IDs or "none"> · needs decision: yes|no`.
  `yes` = answering needs an architecture or dependency decision; `no` = the user can answer it directly.
- IDs are stable: a changed rule keeps its ID; an ID is never reused (new `n` = highest ever used + 1; check the
  file's Git history when rules were deleted); a removed rule is deleted, Git keeps history.
- A rule moved to another file (split) gets a new ID there; the old ID is retired. Fix references and list
  `old → new` in the commit body. Active tasks update their work-spec references when they next merge main.
- Rules state observable behavior: what a user or client sees or can do, data shown, validation, states,
  transitions, permissions. Never implementation: no storage mechanism, framework, or simulated backend detail.
- Each rule has one primary document. Other documents cite its ID; they never copy it.

## 4. Procedure, discovery mode

1. Read the handoff references; `prototype/README.md` at the referenced revision (intentional changes, open
   questions, simulated behavior); `docs/README.md`; existing `docs/specs/*.md`, `docs/ui.md`, `docs/decisions.md`.
2. Read each flow at its referenced revision (`git show <sha>:<path>`, plus the files it loads), never the working
   copy. A flow not `ready for spec` at that revision: ask before using it.
3. Extract features and observable behavior per flow, including failure, empty, and permission states. A
   `SIMULATED:` step yields only its observable outcome (e.g. "a placed order is still listed after reload"); the
   mechanism behind it (e.g. `localStorage`) becomes an **Open** item with `needs decision: yes`.
4. Map items to capabilities: reuse existing files; create a file only for an independently loadable area; put
   cross-capability layout and interaction rules in `docs/ui.md`. When the split is unclear, propose one and ask.
5. Label each item. Default **Observed**. **Agreed** only with a user confirmation or a recorded decision. A
   recorded intentional change counts only if its reason cites a user decision. Rules drafted from user
   statements are confirmed in step 7; unconfirmed ones become **Open** items.
6. Conflicts: an observation that contradicts an **Agreed** rule must not overwrite it, even when recorded as an
   intentional change. Add an **Open** item citing the rule ID and the prototype reference, and ask the user. If
   the user calls the difference accidental, it is a prototype defect: report it for the `prototype` skill.
7. Confirm with the user, one decision per message, blocking questions first; wait for each answer. One
   capability's numbered list of **Observed** rules counts as one question (confirm, correct, or drop each).
   Confirmed or corrected → **Agreed**, source `user <today>`. Dropped → delete. Unsure → **Open** item. The
   confirmation lives in the rule's source, never in `docs/decisions.md`.
8. Create or update the files. Read `docs/framework/skills/spec/template.md` only when creating a file (also for
   `docs/ui.md`). Put prototype references under "References". Add new files to the document index in
   `docs/README.md`; if that file is missing, create it with only a `## Documents` index and say so.
9. Commit (section 7) and report: files changed; rules per label per file; open questions, blocking first;
   prototype defects found; next action: `work-spec` for a selected change, or `prototype` for blocking unknowns.

## 5. Procedure, in-task mode

1. Read the listed decisions in `docs/decisions.md` (only status `approved`; a `pending` entry approves nothing)
   or the work spec's "Spec changes" line, and the spec rules they affect.
2. Per decision or user-confirmed change: in the rule's primary document, update the rule (same ID) or add one;
   label **Agreed**; source = the decision ID or `user YYYY-MM-DD`. Assign final IDs (section 3) to
   `<capability>/new` entries. Fix references; delete the **Open** items it resolves.
3. Leave everything else unchanged, including **Observed** rules the input does not cover.
4. An input that is unclear for a rule, lacks a decision or user source, or contradicts another **Agreed** rule:
   change nothing for it; report it.
5. Report the files and rule IDs changed (new IDs mapped from their `new` entries) to the invoking role. Do not commit.

## 6. Updating after intentional changes or user decisions

- Change a rule only in its primary document; other documents keep their ID references.
- An intentional change (user confirmation or approved decision) updates the rule in place: same ID, **Agreed**,
  new source.
- Never edit a spec to match an accidental defect in the prototype or code. Report the defect instead.

## 7. Commits (discovery mode)

- English: `docs(<capability>): <summary>`; `docs(ui): <summary>` for `docs/ui.md`. One commit per file when
  practical; include the related `docs/README.md` index change.
- Stage only the files section 1 allows, by path. Never sweep in unrelated changes.
- Commit on the main checkout's current branch. If `AGENTS.md` or the repository rules forbid direct commits
  there, use a `spec/<topic>` branch and tell the user.
- Never commit in a feature worktree (a `feature/*` branch or a linked worktree) in discovery mode: stop and tell
  the user.
- Never push unless the user asks.

## 8. Never

- Never write a rule without a label and a source.
- Never label a rule **Agreed** without a user confirmation or a recorded decision.
- Never write architecture, stack, or dependency content.
- Never copy a rule into a second document.
- Never batch multiple decisions into one question.
- Never edit files outside section 1's allowed set.
