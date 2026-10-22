---
name: work-spec
description: Prepares or refines one registered task's work specification in `docs/work/`, before implementation or when its scope or acceptance criteria change. Use from the orchestrator in the task worktree; use `spec` for application-wide behavior and `prototype` for discovery.
---

# Work-spec skill

Framework version: 0.1.0

## 1. Purpose and boundaries

- A work spec describes one change: authorized outcome, context, acceptance criteria (ACs), constraints, checks.
  It never describes the whole application. Requirements and context live here; execution state lives only in
  `state.json`.
- Requires a registered task ID and its feature worktree (`feature/<ID>`). Without them, stop and say a task must
  be registered by an orchestrator first.
- You may edit only `docs/work/<ID>-<slug>.md` in that worktree. You must not edit `docs/specs/`, `docs/ui.md`,
  `docs/decisions.md`, `docs/architecture.md`, `docs/stack.md`, `prototype/`, product code, or workflow state.

## 2. Inputs to read

Load only what the change touches:
- the user's selected outcome or task brief; prototype handoff references, if any (read each flow at its
  referenced revision with `git show <SHA>:<path>`, never the working copy);
- `docs/README.md`: document index, run and check commands;
- the relevant `docs/specs/*.md` and `docs/ui.md`: rule IDs, labels, open questions;
- the `docs/architecture.md`, `docs/stack.md`, and `docs/decisions.md` entries in scope;
- the affected code and tests (required for a bug fix or refactor);
- for a regression of a merged task: that task's compact record (`solomonar task show <ID>`).
  Load its `spec_ref` with `git show "<spec_ref>"` only for a concrete question.

## 3. Procedure

1. Confirm the task ID, title, and worktree (current branch `feature/<ID>`). Derive `docs/work/<ID>-<slug>.md`
   (slug: kebab-case of the title, e.g. `docs/work/W-001-login.md`). If `docs/work/<ID>-*.md` exists, refine it
   instead of replacing it. Read `docs/solomonar/skills/work-spec/template.md` only when creating the file.
2. Determine the change type and the authorized outcome; state it in one or two sentences under "Outcome and
   scope", with its boundaries. No authorized outcome is a blocking question. For the first executable outcome,
   apply section 5's baseline rule; documentation-only work does not need a runnable application.
3. Identify the context: rule IDs, sections, contracts, code paths, prototype references
   (`<full commit SHA>:prototype/<entry file>, flow <flow-id>`). Cite them; never copy their content.
4. Draft the acceptance criteria (section 4).
5. List the constraints: approved decisions (`W-001/D-n`) and global rules that bind the change, by ID or path.
   List the checks that verify the ACs, by AC ID: commands from `docs/README.md` or named scenarios, with known
   verification limits. Discover missing commands from repository scripts, manifests, or existing instructions;
   cite the source and the check needed to verify them. Until verified, a command remains "not established",
   a verification limit rather than a guessed result or evidence that the application does not exist.
6. List open questions relevant to this change only:
   `W-001/Q-n · <question> · blocking | non-blocking · needs decision: yes|no`. Reuse an existing canonical
   `<capability>/Q-n` ID instead of a new one. Blocking = the answer changes the scope, an AC, or the chosen
   implementation. `needs decision: yes` = an architecture or dependency decision, routed by the orchestrator.
   A conflict with an **Agreed** rule or an approved decision is blocking, cites both, and is routed by the
   orchestrator (`rules.md` section 1, section 3); never resolve it in the work spec. Unknowns outside the scope are not
   listed; they stay in the canonical specs.
7. Ask the user only the blocking questions that need no architecture or dependency decision and are not
   conflicts, one per message; wait for each answer. Fold each answer into the spec and delete the question;
   answers to `needs decision: no` questions live in the work spec only.
   Once the orchestrator records an approved decision, cite its `W-001/D-n` and delete the question it resolves.
8. Evaluate the readiness gate (section 5).
9. Commit the file alone (`git add -- <path>`): `spec(<ID>): draft task specification`; later revisions:
   `spec(<ID>): refine task specification`. Skip the commit when nothing changed. Never push. The final commit
   belongs to the orchestrator per `rules.md` section 9. Report the path, the commit SHA, the open questions (blocking
   first), and the readiness verdict as the last line.

## 4. Acceptance criteria rules

- IDs `AC-1`, `AC-2`, …, stable for the task's life. A removed AC is never reused (new `n` = highest ever + 1).
- Each AC is observable: a condition and the expected result, verified by a check named under "Constraints and
  checks". That check is an automated test unless the work spec states why one is impractical; then it is a named
  manual scenario plus its verification limit.
- Each AC cites the rule IDs it verifies or introduces: `<capability>/R-n`, or `<capability>/new` until the
  `spec` skill assigns one. A behavior AC always maps to a rule (add one under "Spec changes"); only a purely
  technical outcome cites `none`. Preferred form (EARS): "When <trigger>, the system shall <response>" or "If
  <condition>, then the system shall <response>".
- Include the relevant failure and boundary cases.
- Per type:
  - feature: new or changed behavior;
  - bug fix: reproduction steps → expected behavior (cite the rule it violates), plus a regression check;
  - refactor: listed existing behaviors stay unchanged (cite rule IDs), plus the structural target; no new behavior;
  - technical task: the observable or verifiable technical outcome.
- An AC that relies on an **Observed** rule needs user confirmation. Ask it with the outcome question when
  possible, once, and record it on the AC: `(orders/R-2, confirmed user YYYY-MM-DD)`.
- Behavior the change adds or alters, including confirmed **Observed** rules and expected behavior no rule states
  yet, goes into the "Spec changes" line: existing rule IDs with their target statement or label, new rules as
  `<capability>/new: <statement>` (the `spec` skill assigns IDs), each with its source (`user YYYY-MM-DD` or
  `W-001/D-n`). Only the orchestrator applies it, via the `spec` skill in-task mode, when the readiness gate
  passes and before implementation (`rules.md` section 4 step 6); the work spec then cites the assigned IDs.
- Never weaken or drop an AC to match an implementation. A change of an AC after implementation starts requires
  a user decision or an approved decision; cite it on the AC (`changed: W-001/D-2`).

## 5. Readiness gate

Implementation is ready when all hold:
- [ ] Authorized: the selected outcome is authorized (brief, selected prototype flow, or approved decision).
  Starting a `planned` record authorizes its recorded outcome.
- [ ] Scope: the outcome and its boundaries are understandable.
- [ ] Verifiable: every AC is observable and has a named check (section 4).
- [ ] Context: the relevant specs, decisions, code, and prototype references are cited.
- [ ] Decisions: no unresolved blocking question or decision prevents the chosen implementation.

- The gate can pass from existing instructions and approvals. Never ask for a confirmation the user has already
  given; never add a ceremonial approval. Non-blocking questions do not fail the gate.
- When the authorized outcome introduces the first executable functionality and no runnable application baseline
  exists, include only the smallest baseline needed for that outcome. Any architecture or dependency approval it
  needs is a blocking `needs decision: yes` question. Unknown run commands alone do not mean the baseline is
  absent: inspect the existing application and record command discovery and verification limits. Do not add a new
  application baseline to brownfield maintenance or documentation-only work. A missing check blocks readiness
  only when it leaves an AC without a viable verification method.
- Output, at the end of the run: `Readiness: READY` or
  `Readiness: NOT READY (<failing criteria; blocking question IDs>)` (e.g.
  `Readiness: NOT READY (Decisions; W-001/Q-1)`). The verdict is never written into the work spec.

## 6. Never

- Never copy a canonical rule into the work spec; cite its ID.
- Never write execution state (status, stage, owner, blockers, revisions, verdicts); `state.json` holds it.
- Never ask about anything outside the change.
- Never batch questions; ask one per message.
- Never report `READY` with a blocking question open.
- Never edit any file other than the work spec.
