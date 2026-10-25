---
name: work-spec
description: Prepares or refines one registered task's work specification in `docs/work/`, before implementation or when its scope or acceptance criteria change. Use from the orchestrator in the task worktree; use `spec` for application-wide behavior and `prototype` for discovery.
---

# Work-spec skill

Framework version: {{ .Version }}

## 1. Purpose and boundaries

- A work spec describes one change: authorized outcome, context, acceptance criteria (ACs), constraints, checks,
  and the implementation plan with its approval status. Registry state (status, stage, owner, blocker records,
  revisions, verdicts) and execution progress stay out of this document; progress lives in checkpoint commits.
- Only the owning orchestrator uses this skill, for a registered task in its assigned `feature/<ID>` worktree.
  Registration, lifecycle transitions, and review stability remain its responsibility outside this skill.
- Apply `AGENTS.md` rule 9 before specifying new or changed behavior. Inspect `prototype/README.md` if present;
  require approved flow references covering that behavior. Missing coverage returns to `prototype` through the
  orchestrator, even for an existing prototype; a task brief or decision cannot replace this handoff.
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

Before editing, run `solomonar task status <ID>`, `git rev-parse --show-toplevel`, `git branch --show-current`,
`git worktree list --porcelain`, and `git status --short`. Confirm registration, title, session owner, and actual
assigned worktree on `feature/<ID>`. Stop on mismatch; preserve unrelated edits.

1. Use the task record's `spec` path when set; require `docs/work/<ID>-<slug>.md`. Preserve it when the title
   changes. With no recorded path, refine a single existing `docs/work/<ID>-*.md`; only when none exists,
   derive a kebab-case slug from the title. Multiple matches, an invalid path, or a missing recorded file:
   report to the orchestrator rather than creating a replacement. Historical-only specs follow `rules.md`
   section 9's restoration procedure. Load `docs/solomonar/skills/work-spec/template.md` only for a new file.
2. Determine the change type and the authorized outcome; state it in one or two sentences under "Outcome and
   scope", with its boundaries. No authorized outcome is a blocking question. For the first executable outcome,
   apply section 5's baseline rule; documentation-only work does not need a runnable application.
3. Identify the context: rule IDs, sections, contracts, code paths, prototype references
   (`<full commit SHA>:prototype/<entry file>, flow <flow-id>`). Cite them; never copy their content.
4. Draft the acceptance criteria (section 4).
5. List applicable approved project decisions (`P/D-n`), task decisions (`W-001/D-n`), and global rules by ID
   or path. Reuse approvals and confirmation sources within their scope; pending/superseded decisions approve
   nothing. Ask only about unresolved choices.
   List the checks that verify the ACs, by AC ID: commands from `docs/README.md` or named scenarios, with known
   verification limits. Discover missing commands from repository scripts, manifests, or existing instructions;
   cite the source and the check needed to verify them. Until verified, a command remains "not established",
   a verification limit rather than a guessed result or evidence that the application does not exist.
6. List open questions relevant to this change only:
   `W-001/Q-n · <question> · blocking | non-blocking · needs decision: yes|no`. Reuse an existing canonical
   `<capability>/Q-n` ID instead of a new one. Blocking = the answer changes the scope, an AC, or the chosen
   implementation. `needs decision: yes` = an architecture or dependency decision, routed by the orchestrator.
   A conflict with an **Agreed** rule or an approved decision is blocking, cites both, and is routed by the
   orchestrator (`rules.md` sections 1 and 3); never resolve it in the work spec. Unknowns outside the scope stay
   in the canonical specs. A question whose answer affects the whole project is a project question
   (`rules.md` section 3): cite its canonical ID, or report it to the orchestrator; never list it as `W-001/Q-n`.
7. Ask the user only the blocking questions that need no architecture or dependency decision and are not
   conflicts, per `AGENTS.md` rule 14. Fold answers into the spec and remove resolved questions there. Technical clarifications can stay task-local; product behavior answers enter
   "Spec changes" (section 4). Product answers changing demonstrated behavior require a prototype update and
   approval before entering "Spec changes"; leave that part blocked until its handoff arrives. Report resolved
   canonical question IDs and their answer/prototype sources for the orchestrator's `spec` update.
   Once an applicable approved decision exists, cite its ID and remove the question it resolves.
8. Validate the document (section 6) and evaluate the readiness gate (section 5). When it passes, write the
   implementation plan (section 7) with `Status: In review` and validate it.
9. Commit only the work spec: `git add -- <path>`, inspect `git diff --cached -- <path>`, then
   `git commit --only -m "spec(<ID>): draft task specification" -- <path>`; later revisions use
   `spec(<ID>): refine task specification`. Preserve unrelated staged changes; skip unchanged commits. Never
   push. Keep the `spec(<ID>):` prefix: the CLI uses it to locate the final specification. Finalization belongs
   to the orchestrator per `rules.md` section 9. Report the path, commit SHA, resolved canonical questions,
   remaining questions (blocking first), and validation limits; end with the readiness verdict.

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
- An AC relying on an **Observed** rule stays blocked until an approved prototype represents it and `spec`
  can promote it. Record the approval and flow reference; do not seek spec-only confirmation.
- Behavior the change adds or alters, including confirmed **Observed** rules and expected behavior no rule states
  yet, goes into the "Spec changes" line: existing rule IDs with their target statement or label, new rules as
  `<capability>/new: <statement>` (the `spec` skill assigns IDs), each with its source (`user YYYY-MM-DD` or
  an approved decision ID, plus the approved prototype reference). When multiple proposed rules share
  `<capability>/new`, distinguish each by its target statement and the AC IDs it supports so final-ID mappings
  are unambiguous. Only the orchestrator
  applies changes via the `spec` skill in-task mode, after readiness and before implementation (`rules.md`
  section 4 step 6); the work spec then cites the assigned IDs.
- Never weaken or drop an AC to match an implementation. A change of an AC after implementation starts requires
  a user decision or an approved decision; cite it on the AC (`changed: W-001/D-2`).

## 5. Readiness gate

The work spec is ready when all hold:

- [ ] Authorized: the selected outcome is authorized (brief, selected prototype flow, or approved decision).
  Starting a `planned` record authorizes its recorded outcome.
- [ ] Scope: the outcome and its boundaries are understandable.
- [ ] Verifiable: every AC is observable and has a named check (section 4).
- [ ] Context: the relevant specs, decisions, code, and prototype references are cited.
- [ ] Behavior: every new or changed behavior is demonstrated in an explicitly approved prototype snapshot;
  no conflicting specification or stale approval remains. Technical work with no behavioral effect cites the
  existing contract it preserves and does not require a new prototype or renewed approval.
- [ ] Decisions: no unresolved blocking question or decision prevents the chosen implementation.

- `READY` reports planning readiness, not permission to start implementation or a CLI transition. Before
  delegation, the orchestrator applies approved canonical changes via `spec`, replaces temporary rule
  references with assigned IDs, and records the work-spec path through `solomonar task` commands.
- Readiness requires viable verification methods; implementation tests need not already pass. Never imply
  planned checks were run. A blocking question or unresolved validation gap affecting a gate criterion
  prevents `READY`.
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

## 6. Validation

Before the verdict and commit, check:

- The document matches the registered task ID and selected path; AC IDs are unique, stable, and not reused.
- Existing source references resolve; planned paths and proposed rules are explicit. ACs relying on
  **Observed** rules have approved prototype coverage pending promotion; approval sources cover the proposed behavior.
- Every AC maps to canonical rules or distinguishable proposed rules (technical outcomes may cite `none`),
  and to a named check with any verification limit.
- Question markers match their impact; unresolved blockers remain visible. Behavior answers appear in "Spec
  changes"; resolved canonical questions are reported for the orchestrator to update.
- `Status:` is `In review` or `Approved`; plan steps are `- S-<n>: <type>(<ID>): <summary>` with unique,
  never-reused IDs, and outcomes carry a reason.
- No copied unchanged canonical rules, execution state, template hints, or unused placeholders remain. Pending
  `<capability>/new` entries remain explicit until the orchestrator replaces them.

Run `git diff --check -- <path>` and inspect `git diff -- <path>` for unintended changes. Report validation gaps;
apply section 5's gate without editing files outside the work spec.

## 7. Implementation plan and approval

- Under `## Implementation plan`, list the intended commits in order: `- S-<n>: <type>(<ID>): <summary>`, the
  exact checkpoint commit subject (type ∈ `feat`, `fix`, `refactor`, `test`, `docs`, `build`, `chore`, `perf`).
  Indented lines below a step delimit its work: files or areas, the ACs it serves, its checks. Each step is one
  coherent, independently checkable change; no step only records progress.
- The line below the title is `Status: In review` or `Status: Approved`, exactly. A new or changed plan is
  `In review`. Only the orchestrator sets `Approved`, after the user explicitly approves the exact plan
  (`rules.md` section 4 step 7), and commits it as `spec(<ID>): approve implementation plan`. Reuse an existing
  explicit approval that covers the exact plan; never ask twice.
- Step IDs are never reused. A change to an approved step's scope takes a new ID and returns `Status` to
  `In review`. Corrections within the approved scope are appended as new steps without a new approval. Close a
  step that became unnecessary or was satisfied by verification with an indented
  `Outcome: dropped — <reason>` or `Outcome: verified — <reason>`; never delete a step that has a checkpoint.
- The plan is the approved plan, not an execution log: never mark steps done in it. `solomonar task status`
  reconciles it with the checkpoint commits.
