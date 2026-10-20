---
name: prototype
description: Builds and updates the static HTML, CSS, and JavaScript prototype in `prototype/` for product discovery through use. Use it to start a new product, explore or demonstrate a flow, or change an existing prototype flow after feedback. Do not use it for production code, specs, or managed tasks.
---

# Prototype skill

Framework version: 0.1.0

## 1. Purpose and boundaries

- The prototype is a discovery tool: evidence of intended behavior, never production code or approved requirements.
- You may edit only `prototype/**`, the "Current objective" line of `docs/README.md` when no objective exists, and
  its Mission and Intended users once the user confirms them (section 2).
- You must not edit `docs/specs/`, `docs/work/`, `docs/decisions.md`, `docs/architecture.md`, `docs/stack.md`,
  product code, or workflow state.
- You must not decide production architecture or dependencies. No prototype choice approves either.

## 2. Start

1. Read `docs/README.md` and `prototype/README.md` if present.
2. No project objective: ask for one short objective (one question) and record it in `docs/README.md`
   "Current objective". If `docs/README.md` is missing, record it in `prototype/README.md` and say so.
   Mission and Intended users are user-owned: you may propose them here; record them only after the user confirms.
3. Agree on the next useful flow; ask one question only if it is unclear. Architecture, stack, and layout may be
   unknown: do not ask about them.
4. Read `docs/framework/skills/prototype/template.md` only when creating `prototype/README.md` for the first time.

## 3. Build rules

- Vanilla HTML, CSS, and JS. It must open directly from the file system (`file://`): no build step, no package
  install, no server. Use classic `<script src>` tags; no ES modules or `fetch` of local files.
- Keep shared files minimal: `index.html` (flow index), `styles.css`, `app.js`, `data.js`, plus one HTML file
  per flow screen only when needed.
- Start each flow with the smallest useful version: the main path plus the states needed to judge it.
- Avoid libraries. If one is truly useful and the user agrees, vendor a single pinned file
  (`prototype/vendor/<name>-<version>.js`) and list it as prototype-only in the README. It is never an approved
  dependency.
- Sample data must be fictional. Never use real personal data, credentials, or secrets.
- Simulate every backend or external service in memory or `localStorage`. Mark each with a `SIMULATED: <what>`
  code comment and list it under "Simulated behavior". A request for a real integration (e.g. a payment API) gets
  a simulation instead; record the real integration as a non-blocking open question.
- Every flow must be clickable end to end from `index.html`, including the failure, empty, or permission states
  that matter to it. Give each such state a visible trigger (e.g. a sample record or a toggle).
- Use semantic HTML (`form`, `label`, `button`, headings). Keep styling plain unless the user asks for visual
  exploration.
- Framework artifacts (README, commits, open questions) are English. UI copy follows the product's intended
  language; if unknown, use the user's language and record it as a non-blocking open question.

## 4. Discovery loop

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
- Files of a `superseded` flow may be deleted; its README row stays.

## 6. Commits

- English messages: `prototype(<flow-id>): <summary>`. Keep one flow per commit when practical. README-only or
  multi-flow commits: `prototype: <summary>`.
- Stage only the files section 1 allows, by path. Never sweep in unrelated changes.
- Commit in the main checkout's current branch. If `AGENTS.md` or the repository rules forbid direct commits
  there, use a `prototype/<topic>` branch and tell the user.
- Never commit inside a feature worktree (a `feature/*` branch or a linked worktree): stop and tell the user.
- Never push unless the user asks.

## 7. Handoff to `spec`

1. When the user is ready, confirm which flows are ready (one question if unclear).
2. Set them to `ready for spec`, update `prototype/README.md`, and commit.
3. Give one reference per flow, using the full SHA of that commit (`git rev-parse HEAD`):
   `<full commit SHA>:prototype/<entry file>, flow <flow-id>`.
4. List the open questions for those flows, blocking ones first.
5. State the next action: run the `spec` skill with these references. Do not write specs yourself.

## 8. Never

- Never write production code or a real backend, or wire the prototype into product code.
- Never make real external calls or use real credentials, keys, or personal data.
- Never add build tooling, package manifests, or installed packages.
- Never ask more than one question at a time.
- Never treat a prototype library, simulation, or structure as approved architecture or a dependency.
- Never edit files outside section 1's allowed set.
