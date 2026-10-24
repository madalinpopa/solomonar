---
name: prototype
description: Builds and updates the static HTML, CSS, and JavaScript prototype in `prototype/` for product discovery through use. Use it to start a new product, explore or demonstrate a flow, or change an existing prototype flow after feedback. Do not use it for production code, specs, or managed tasks.
---

# Prototype skill

Framework version: 0.3.2

## 1. Purpose and boundaries

- The prototype is a discovery tool: evidence of intended behavior, never production code or approved requirements.
- Edit only `prototype/**` and these parts of `docs/README.md`: an unset Current objective; user-confirmed
  Mission and Intended users; the Documents entry for `prototype/README.md` after it exists.
- No production code, real backend or service calls, integration with product code, build tooling, package
  manifests, or installed packages. Prototype choices never approve production architecture or dependencies.
- Use fictional data; never use real personal data, credentials, or secrets.
- Ask only the next necessary question, one at a time. Reuse existing answers and approvals within their scope.

## 2. Start

1. Before editing, inspect `git rev-parse --show-toplevel`, `git branch --show-current`,
   `git worktree list --porcelain`, and `git status --short`. Require the primary checkout on a named branch;
   stop in linked worktrees or on `feature/*` or `workflow-state`. Preserve unrelated edits. If repository rules
   forbid commits on the current branch, use `prototype/<topic>` in the primary checkout and tell the user.
2. Read `docs/README.md` and `prototype/README.md` if present.
3. Treat absent/empty objectives and unset placeholders (including "Not established.") as missing. Reuse an
   objective supplied by the user or recorded in `prototype/README.md`; ask only if still unknown. Fill the
   unset Current objective in `docs/README.md`; if that file is missing, record it in `prototype/README.md` and
   say so. Mission and Intended users remain user-owned: propose them if useful, record only after confirmation.
4. Select the next useful flow; ask only if unclear. Unknown architecture, stack, or layout does not block discovery.
5. Load `docs/solomonar/skills/prototype/template.md` only when creating `prototype/README.md`. When that README
   exists, add its Documents entry to `docs/README.md` if present; do not duplicate an existing entry.

## 3. Build rules

- Vanilla HTML, CSS, and JS, opened directly through `file://` without a server. Use classic `<script src>` tags;
  no ES modules or `fetch` of local files.
- Keep shared files minimal: `index.html` (flow index), `styles.css`, `app.js`, `data.js`, plus one HTML file
  per flow screen only when needed.
- Start each flow with the smallest useful version: the main path plus the states needed to judge it.
- Avoid libraries. If one is truly useful and the user agrees, vendor a single pinned file
  (`prototype/vendor/<name>-<version>.js`) and list it as prototype-only in the README. It is never an approved
  dependency.
- Simulate every backend or external service in memory or `localStorage`. Mark each with a `SIMULATED: <what>`
  code comment and list it under "Simulated behavior". A request for a real integration (e.g. a payment API) gets
  a simulation instead; record the real integration as a non-blocking open question.
- Every flow must be clickable end to end from `index.html`, including the failure, empty, or permission states
  that matter to it. Give each such state a visible trigger (e.g. a sample record or a toggle).
- Use semantic HTML (`form`, `label`, `button`, headings). Keep styling plain unless the user asks for visual
  exploration.
- UI copy follows the product's intended language; if unknown, use the user's language and record it as a
  non-blocking open question. Framework artifacts follow `AGENTS.md`'s language policy.

## 4. Discovery loop

After each change, verify before presenting a preview or committing:

- Open `prototype/index.html` through `file://`; exercise the flow's documented steps and relevant failure,
  empty, and permission states. Recheck flows affected by shared-file changes.
- Check navigation, local asset loading, and browser errors; fix introduced defects.
- Report checks and verification limits with the preview or handoff. If browser access is unavailable, state
  what remains unverified; do not claim browser verification.

1. Build or adjust the smallest change that lets the user try the current flow.
2. Tell the user how to try it: file path, flow ID, numbered steps, and how to reach the non-happy states.
3. Collect feedback.
4. Classify each item:
   - intentional change → apply it and record it under "Intentional changes";
   - question that blocks the current flow → ask it, one per message, and wait;
   - accidental defect (the prototype differs from what was agreed) → fix it; do not record it as a change;
   - unrelated idea or unknown → record it under "Open questions"; do not ask now.
5. Update `prototype/README.md`: flows, simulated behavior, libraries, changes, questions.
6. Commit at each checkpoint (section 6): the first runnable version of a flow, a feedback round the user
   accepts, or when the user asks.
7. Repeat from step 1, or hand off (section 7) when the user says flows are ready.

## 5. Flow IDs and records

- Each flow has a stable kebab-case ID (e.g. `checkout`), an entry file, numbered steps, and a status:
  `draft` | `ready for spec` | `superseded`.
- An ID never changes or is reused once it appears in a commit. A replaced flow gets a new ID; mark the old one
  `superseded` and name its replacement in Notes.
- Record each intentional change as: date (YYYY-MM-DD) · flow ID · change · reason (user decision or observed
  problem). Defect fixes and internal refactoring are not recorded.
- Set `ready for spec` only after the user confirms the flow. `draft` flows may live on the main branch.
- Before changing a ready flow's behavior, including through shared files, reset it to `draft`. Preserve IDs.
  Restore readiness only after confirmation of the changed behavior.
- Files of a `superseded` flow may be deleted; its README row stays.

## 6. Commits

- Messages: `prototype(<flow-id>): <summary>`. Keep one flow per commit when practical. README-only or
  multi-flow commits: `prototype: <summary>`.
- Commit only intended changes allowed by section 1, selected by path, on the branch established at startup.
  Inspect the staged diff; preserve unrelated staged changes. Skip the commit when nothing relevant changed.
- Never push unless the user asks.

## 7. Handoff to `spec`

1. When the user is ready, confirm which flows are ready (one question if unclear).
2. Verify the selected flows (section 4), set them to `ready for spec`, and update `prototype/README.md`.
   Commit relevant changes (section 6); when unchanged, reuse the existing committed snapshot.
3. Resolve the full SHA with `git rev-parse HEAD`. Check `git show "<SHA>:prototype/README.md"` for each selected
   flow's `ready for spec` record and `git show "<SHA>:prototype/<entry file>"` for its entry file. Do not hand
   off uncommitted prototype changes. Give one reference per flow:
   `<full commit SHA>:prototype/<entry file>, flow <flow-id>`.
4. List the open questions for those flows, blocking ones first, and any verification limits.
5. State the next action: run the `spec` skill with these references. Do not write specs yourself.
