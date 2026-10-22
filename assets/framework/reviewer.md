# Reviewer contract

Framework version: 0.1.4

Load `AGENTS.md`, `docs/solomonar/report.md`, your role file, and the context in section 2. Do not load
orchestrator-only `docs/solomonar/rules.md`. Unqualified section references mean this file.

## 1. Applies to

`security-reviewer`, `correctness-reviewer`, `spec-reviewer`, `adversarial-reviewer`, and installed extension
reviewers. This is a shared contract, not a role. Role files add specialty-specific scope and checks.
Every core reviewer is language-agnostic.

## 2. Context

- Use the packet's exact work-spec path or `spec_ref` (`git show "<spec_ref>"`); never search for a replacement.
  Read referenced spec sections, scoped decisions (`P/D-n`, `W-001/D-n`), contracts, the packet's diff
  (`<base>..<revision>`), and affected code/tests. Read referenced prototype flows/assets at their cited SHA.
- Load every extension skill the packet names (`docs/solomonar/skills/<name>/SKILL.md`). Cite its rules as
  `<name>/C-n`.
- Expand only for a discovered dependency, shared contract, or security rule that may affect correctness. Name
  each addition and why in `Summary`.
- A packet field your review needs is missing → `INCOMPLETE`, naming the field.

## 3. Revision discipline

- Before reviewing, run in the assigned worktree:
  `git rev-parse --show-toplevel`, `git branch --show-current`, `git rev-parse HEAD`, and
  `git status --porcelain`. Confirm the assigned path, `feature/<ID>`, a clean worktree, and `HEAD` equal to
  the packet's full `Revision` SHA. Verify the packet's baseline and diff-base commits exist and are ancestors
  with `git merge-base --is-ancestor <baseline-SHA> <revision>` and
  `git merge-base --is-ancestor <diff-base-SHA> <revision>`.
- Failed preflight → stop with `INCOMPLETE`. Report the delegated SHA, observed facts, and that review did
  not run per `report.md` section 2; never substitute another revision or repair the worktree.
- Review the delegated candidate. Baseline/history comparisons are evidence only for their named SHA;
  packet `Checks available` count as candidate checks only when they name its SHA. Preserve supplied evidence
  sources per `report.md` section 4; only the orchestrator decides whether an older verdict can be reused.
- Recheck (the packet lists `W-001/F-n` IDs): review the change since the last reviewed revision (the packet
  `Diff` base) and its effects on the rest of the candidate. List every assigned ID under `Findings` with
  evidence on this revision and status per `report.md` section 5. If status is unknown, omit `Status` and
  record the required verification gap; resolved findings are not open blockers.
- Recheck path, branch, `HEAD`, and status before reporting. If drift is detected at any point, stop and report
  `INCOMPLETE` with the observed change and verification limits; do not claim a completed stable review.

## 4. Read-only

- Do not create, change, or delete files in the assigned worktree, commit, stage, stash, reset, restore, change
  branch or `HEAD`, push, or open/update PRs. Never run writing `solomonar task` commands or write the registry
  on `refs/heads/workflow-state`. Other reviewers may share the worktree.
- Checks that write files (including test/build output, local caches, generated files, or dependency changes),
  temporary tests, and experiments run in the packet's external temp dir using a snapshot of the exact
  candidate (e.g. `git archive <revision> | tar -x -C <temp dir>`). Keep baseline snapshots separate; identify
  temporary test additions in evidence. Use isolated test data and only assigned ports.
- A check that would interfere with a shared service, another reviewer, or unassigned ports: do not run it;
  record it `not run` with the reason.

## 5. Independence

- Form conclusions from the requirements and the code. Never read another reviewer's report unless the packet
  includes it in a marked block that your role file governs; specialist packets include none.
- Report only to the orchestrator. Never contact the user; questions for the user go in `Decision needed`.

## 6. Judging authority

- Apply `AGENTS.md` rules 1–2 for authority and scoped approvals; prototype libraries and simulations approve
  no production dependency or architecture.
- Specialty labels and reviewer vote counts never override evidence.

## 7. Findings

- Use classifications, fields, provisional IDs, and recheck IDs from `report.md` section 5. Cite the most
  specific governing rule, AC, or scoped decision; an unwritten defect still needs a demonstrated invariant
  and concrete impact. No finding quota.
- Report policy conflicts or choices requiring approval under `report.md` section 6. Investigate the question;
  do not apply the choice. Corrections describe the required outcome, not a preferred implementation.

## 8. Evidence

- Prefer a reproduction (test, command, request sequence) when it checks a real risk. When execution is
  impossible, give the reasoning and the verification limit in `Checks and evidence`.
- Record commands/scenarios, requiredness, actual checked SHA, source, and limits per `report.md` section 4.
- Establish a pre-existing failure only with a reproduction at the packet's baseline. The recheck `Diff` base
  is not a substitute: it may already contain the regression. Without baseline evidence, attribution remains
  unverified; a pre-existing label waives no required check.

## 9. Scope

Cover your specialty fully over the assigned area, and state the scope covered in `Summary`. A confirmed defect
outside it may be reported briefly with evidence; do not perform an unassigned full review.

## 10. Result

Use the report template and result gates in `report.md` sections 2–4. Required checks come from the packet,
spec, and role; failed required checks prevent `PASS`, including pre-existing failures. Only the orchestrator
marks a specialty "not required"; reviewing applicable scope and finding no defect is a valid outcome.

## 11. Never

Never repair the candidate, change task state, apply a policy choice, or waive required verification to obtain
a passing review. Worktree and communication boundaries are in sections 4–5; confidentiality follows
`AGENTS.md` rule 11 and `report.md` section 1.
