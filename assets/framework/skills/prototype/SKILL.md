---
name: prototype
description: Builds and updates independently runnable behavioral prototypes in `prototype/` for whole applications or selected functions, components, modules/libraries, workflows, and services. Supports CLI and web interfaces. Use for discovery, feedback, or new or changed behavior before specs and production implementation, including existing prototypes. Does not write specs or production code.
---

# Prototype skill

Framework version: 0.3.7

## 1. Purpose and boundaries

- Treat the prototype as an executable, human-readable agreement: once explicitly approved, it is the source
  of truth for represented behavior, not production architecture. Keep it independently runnable for inspection;
  its implementation remains disposable.
- Enforce **prototype → human review and approval → specifications → production implementation** for every
  new or changed behavior, including changes to an existing prototype. A request, spec edit, task/plan approval,
  or production implementation never substitutes for approval of the demonstrated behavior.
- Resolve behavioral conflicts in the prototype first; route ambiguous or unrepresented behavior for a decision
  and demonstration before specification. Unapproved changes establish no requirements.
- Edit only `prototype/**` and these parts of `docs/README.md`: an unset Current objective; user-confirmed
  Mission and Intended users; the Documents entry for `prototype/README.md`. Do not edit specs, production code,
  or task state. An in-task request returns here in the primary checkout; the orchestrator owns task coordination.
- Reuse existing answers and scoped approvals; ask only the next necessary question, per `AGENTS.md` rule 14.
  A defect fix restoring already approved prototype behavior or an internal refactor needs verification, not
  renewed behavioral approval.

## 2. Start and select guidance

1. Inspect `git rev-parse --show-toplevel`, `git branch --show-current`, `git worktree list --porcelain`, and
   `git status --short`. Require the primary checkout on a named branch outside `feature/*` and `workflow-state`.
   Preserve unrelated edits. If repository rules forbid commits there, use `prototype/<topic>` in that checkout.
2. Read `docs/README.md`, `prototype/README.md`, and relevant existing flow files/specs. Reuse the objective
   supplied by the user or prototype README; ask only if unknown. Fill an unset Current objective (including
   "Not established."); if `docs/README.md` is absent, record it in the prototype README and report that limit.
   Record Mission and Intended users only after user confirmation.
3. Select the next useful behavior and interface type; a single function, component, module, or workflow is
   enough. Do not build the whole application to demonstrate one part. Organize independent parts by type under
   `prototype/cli/`, `prototype/web/`, `prototype/module/`, or `prototype/service/`; create only needed directories
   and preserve existing runnable layouts. Index each part and its run/check instructions in `prototype/README.md`.
   Unknown production architecture does not block discovery. Load only matching references below; combine them
   only for flows crossing those interfaces.
4. When creating `prototype/README.md`, use [template.md](template.md); index it once in `docs/README.md` if present.

| Prototype interface | Read |
|---|---|
| Commands, flags, terminal output | [references/cli.md](references/cli.md) |
| Browser screens and interactions | [references/web.md](references/web.md) |
| Public module/library interface | [references/module.md](references/module.md) |
| Standalone service or API | [references/service.md](references/service.md) |

## 3. Build constraints

- Use clear names, simple structure, short cohesive functions, and straightforward control flow. Avoid
  production layers, speculative features, design patterns without a concrete need, and premature optimization.
- Demonstrate primary workflows end to end with relevant alternate, failure, empty, permission, and boundary cases.
  Preserve realistic inputs, validation, constraints, state transitions, outputs, and feedback. Make every
  exposed action functional; omit controls or operations implying unsupported behavior.
- Use realistic fictional, deterministic seed data. Control time/randomness when they affect results. Simulate
  external collaborators locally, without production code, services, credentials, personal data, or infrastructure.
- Keep mock infrastructure identifiable apart from observable behavior, without abstraction layers just for mocks.
  Mark substitutes with `SIMULATED: <what>` comments and document their limits and scenario triggers in the README.
- Prefer standard libraries and native platform APIs, with one local run command using an available runtime.
  Use Node.js built-ins when Node.js fits. Add dependencies, manifests, package managers, build steps, bundlers,
  or additional services only when execution or behavioral fidelity requires them; record why and pin dependencies.
  Keep them prototype-local and subject to repository approval rules. Use Docker/Compose only when its portability
  or reproducibility benefit materially outweighs setup. Prototype choices never approve production dependencies.
- Give each independently runnable part exact run and test/inspection commands or steps, working directory,
  runtime requirements, scenario triggers, and expected outputs/state changes, including relevant failures.
  Use a minimal harness for code without an entry point; never leave behavior as an unexecutable code fragment.
  Document simulation limitations and, where state matters, a deterministic reset to seeded prototype data.
  Use product-facing language for interactions; if unknown, use the user's language and record the question.

## 4. Review and approval loop

1. Before changing approved behavior, mark every affected flow `draft`, including flows affected by shared files.
   Keep the previous approved snapshot available. Build the smallest useful change; record intentional changes
   as date (YYYY-MM-DD) · flow ID · change · reason. Exclude defect fixes and internal refactors from that log.
2. Run each affected part using its README command, then its documented tests/inspection scenarios and relevant
   alternate/failure cases; verify expected outputs, state changes, and reset.
   Recheck affected shared behavior. Use lightweight automated behavior checks where useful; do not introduce
   production testing infrastructure. Fix introduced defects; report exact checks and execution limits.
3. Present the runnable result with flow IDs, numbered inspection steps, expected results, and alternate/failure
   triggers. Collect human feedback; never treat silence, a successful check, or a commit as approval.
4. Apply feedback and repeat verification. Before handoff, ask each blocking open question (`AGENTS.md`
   rule 14); fold answers into the prototype and remove resolved questions. Keep unrelated
   ideas and non-blocking unknowns under Open questions. Set `ready for spec` only after explicit approval covering the
   demonstrated behavior; record `user YYYY-MM-DD` and its scope per flow. Later behavioral edits invalidate it.
5. Commit checkpoints: the first runnable flow, an accepted feedback round, or when requested. Use
   `prototype(<flow-id>): <summary>`; README-only/multi-flow: `prototype(readme): <summary>`. Select only
   allowed paths, inspect the staged diff, preserve unrelated staged work, skip unchanged commits, and never
   push unless asked.

## 5. Records and specification handoff

- Keep stable kebab-case flow IDs, entry-file paths relative to `prototype/`, numbered scenarios, and status
  `draft` | `ready for spec` | `superseded`. Never rename/reuse a committed ID; a replacement gets a new ID.
  Retain superseded rows with their replacement and approved revision reference so prior behavior stays inspectable.
- Before handoff, ensure each selected flow exposes actions/external inputs; observable outputs and state
  transitions; validation and business constraints; relevant success/failure/edge cases; persistence and integration
  outcomes. Do not fill gaps by inference. Resolve ambiguity in the prototype and obtain approval first.
- Commit the verified, approved flows and README; reuse an unchanged committed snapshot. Resolve the full SHA
  with `git rev-parse HEAD`; use `git show "<SHA>:prototype/README.md"` and `git show "<SHA>:prototype/<entry file>"`
  to verify status, approval, and entry files at that revision. Never hand off uncommitted or draft behavior.
- Give `spec` one reference per flow: `<full commit SHA>:prototype/<entry file>, flow <flow-id>`, plus questions
  and verification limits. Specifications extract only approved observable behavior, never mock implementation
  details. The owning orchestrator brings updated prototype/spec references into a waiting task before production
  work resumes. Do not write specifications or production code through this skill.
