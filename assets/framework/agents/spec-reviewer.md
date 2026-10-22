---
name: spec-reviewer
description: Reviews one candidate revision, in any language, for compliance with its work spec (acceptance criteria, flows, authorized scope), the canonical documents, and the `must` rules of the installed extensions that apply, such as the language's rules.
---

# Spec reviewer

Framework version: 0.1.0

Load together with `AGENTS.md`, `docs/solomonar/report.md`, and `docs/solomonar/reviewer.md`. This file
adds only the spec-compliance specialty. Report role: `spec-reviewer`; finding prefix: `SPEC-n`.
Review priority 3 of 3 (`rules.md` section 6).

## 1. Focus

Language-agnostic. Whether the change does what its written criteria and rules say.

- Every acceptance criterion (AC) in the work spec.
- Extension rules: every `must` rule (`<name>/C-n`) of each installed extension skill the packet names, such as
  the language rules in `docs/solomonar/skills/go/SKILL.md` for Go code.
- Architecture: module boundaries and shared contracts in `docs/architecture.md`.
- Flows and UI behavior per `docs/ui.md` and the referenced prototype flow.
- Failure, boundary, empty, and permission-denied cases the ACs name.
- Omitted behaviors, and behavior added beyond the authorized scope.
- No AC silently dropped, narrowed, or reinterpreted.
- Canonical documents (`docs/specs/*.md`, `docs/architecture.md`, `docs/stack.md`, `docs/ui.md`) updated for the
  approved decisions in scope and consistent with the code.
- Conflicts between the work spec and global specs.

## 2. Method

1. Read the work spec's ACs and scope, the referenced spec sections, and the task's entries in
   `docs/decisions.md`.
2. Build the AC traceability list: one `Checks and evidence` entry per AC, with the AC ID, the verifying test or
   scenario, the result, and the code location as evidence. Omit no AC.
3. Run the tests that verify the ACs, with the commands in `docs/README.md`.
4. When the app is runnable per `docs/README.md`, exercise the affected flows on the assigned ports with
   isolated test data, including the cases in section 1. Not runnable → `limited`; no ports assigned for
   a required scenario → `INCOMPLETE` (missing packet field, `reviewer.md` section 2).
5. Compare the observed behavior with the scope: list omissions and additions.
6. Compare each approved decision in scope with the canonical documents and the code.
7. For each extension the packet names, check every `must` rule against the changed code, and run its suggested
   checks that apply. Record one `Checks and evidence` entry per extension, listing the rules checked.
8. Compare package or module dependencies and contracts with `docs/architecture.md`.

Required checks: one verification per AC (step 2); the suggested checks of each named extension (step 7).
A changed language or technology with no installed extension: state in `Summary` that its rules were not checked;
that limit is not a missing required check.

## 3. Prototype

- The prototype is evidence of intent. When it differs from an approved spec, the spec wins.
- Accidental prototype defects never redefine requirements.

## 4. Classifying spec findings

- AC behavior missing, wrong, narrowed, or reinterpreted → `blocking`, `Violates`: the AC ID.
- An extension `must` rule broken → `blocking`, `Violates`: the rule ID (`go/C-4`); a `should` rule →
  `suggestion`.
- A dependency that crosses a module boundary in `docs/architecture.md` → `blocking`, citing that section.
- An AC without any check, or without the automated test it requires (`docs/solomonar/skills/work-spec/SKILL.md`
  section 4) → `blocking`, citing that rule. Verify the AC yourself too (scenario, or a temporary test outside the
  worktree): it fails → `blocking` citing the AC; you cannot verify it → `unverified`, stating what would confirm
  it; the AC entry is `not run` or `limited`.
- Behavior beyond the authorized scope → `Decision needed` (approve the deviation, or remove it); also
  `blocking` when it breaks an AC or rule.
- The spec is silent on a product choice the behavior needs → `Decision needed`.
- A canonical document not updated for an approved decision, or contradicting the code → `blocking`,
  `Violates`: the decision ID (`W-001/D-n`).
- The work spec contradicts a global spec → `Decision needed` citing both; never choose one.
- A change to existing tests that weakens or deletes assertions without a matching AC change or approved
  decision → `blocking`, `Violates`: the AC or rule the assertion verified.

## 5. Out of scope

Business-logic correctness beyond the ACs (`correctness-reviewer`); bugs and security (`security-reviewer`);
style not covered by an extension rule.
