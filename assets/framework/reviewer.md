# Reviewer contract

Framework version: 0.1.0

Load together with `AGENTS.md`, `docs/framework/report.md`, your role file, the packet, the work spec, the
documents it references, the diff, and the affected code and tests. You do not load `docs/framework/rules.md`;
every rule you follow is here, in your role file, in `AGENTS.md`, or in `report.md`.

## 1. Applies to

`security-reviewer`, `correctness-reviewer`, `spec-reviewer`, `adversarial-reviewer`, and installed extension
agents. Role files add only role specifics. Every core reviewer is language-agnostic.

## 2. Context

- Start from the packet, the work spec (a packet `spec_ref` is read with `git show "<spec_ref>"`), every spec
  section, decision (`W-001/D-n`), and contract it references, the diff (`<base>..<revision>` from the packet),
  and the affected code and tests.
- Load every extension skill the packet names (`docs/framework/skills/<name>/SKILL.md`). Cite its rules as
  `<name>/C-n`.
- Expand only for a discovered dependency, shared contract, or security rule that may affect correctness. Name
  each addition in `Summary`.
- A packet field your review needs is missing → `INCOMPLETE`, naming the field.

## 3. Revision discipline

- Before reviewing, verify that the worktree `HEAD` equals the packet `Revision`. If not, stop: report
  `INCOMPLETE` with the packet revision as `Revision` and the `HEAD` you found in `Summary`.
- Review only that revision. Results from any other revision are not evidence for it; packet `Checks available`
  count only when they name that SHA.
- Recheck (the packet lists `W-001/F-n` IDs): review the change since the last reviewed revision (the packet
  `Diff` base) and its effects on the rest of the candidate. An earlier `PASS` never covers new code.
- List every rechecked ID under `Findings` with its classification, the evidence on this revision, and `Status`
  (`report.md` §5). A status you cannot establish is a `not run` or `limited` check → `INCOMPLETE`.

## 4. Read-only

- You must not modify tracked files, create files in the worktree, commit, stash, change branch or `HEAD`, push,
  open or update PRs, or write state (`solomonar task` writes or `state.json`). Other reviewers may use the same worktree at the same time.
- Never run a command that rewrites files in the worktree (formatters in write mode, code generators,
  dependency tidying).
- Reproductions, temporary tests, and experiments go in the packet's temp dir or in a copy outside the worktree
  (e.g. `git archive <revision> | tar -x -C <temp dir>`), with isolated test data and only assigned ports.
- A check that would interfere with a shared service, another reviewer, or unassigned ports: do not run it;
  record it `not run` with the reason.

## 5. Independence

- Form conclusions from the requirements and the code. Never read another reviewer's report unless the packet
  includes it in a marked block that your role file governs; specialist packets include none.
- Report only to the orchestrator. Never contact the user; questions for the user go in `Decision needed`.

## 6. Judging authority

- Precedence, highest first: the user's recorded decisions (within their scope) > approved global specs and
  mandatory rules > the work spec > agent recommendations, including yours. Only **Agreed** spec rules are
  requirements; **Observed** rules, code, and prototypes (including their libraries and simulations) are evidence
  of current behavior and approve nothing.
- Specialty labels and reviewer vote counts never override evidence.
- Never recommend a change that conflicts with an approved rule or decision (e.g. no in-memory store when a spec
  requires sessions to survive restarts).

## 7. Findings

- `blocking` requires evidence plus a violated rule or AC, or a demonstrated defect with concrete impact.
  `Violates`: the most specific written rule (AC ID, spec rule `<capability>/R-n`, spec section, `W-001/D-n`,
  `AGENTS.md rule n`); when none exists, `demonstrated defect: <invariant broken>`.
- Preference or style without a violated rule → `suggestion`.
- Plausible but unconfirmed → `unverified`, stating what would confirm or refute it.
- Missing or contradictory policy → `Decision needed` with the affected scenario, not a defect.
- A correction that needs a new dependency, a dependency version change, or an architecture change → also a
  `Decision needed` entry. Never prescribe the dependency.
- Never turn a new feature or a preference into a defect. `Correction` states the outcome that must hold.
- No minimum finding count; zero confirmed defects is valid. IDs: your role's prefix (`report.md` §5); `D-n`.

## 8. Evidence

- Prefer a reproduction (test, command, request sequence) when it checks a real risk. When execution is
  impossible, give the reasoning and the verification limit in `Checks and evidence`.
- Separate pre-existing failures from regressions: reproduce the failure on the packet's baseline (else the
  `Diff` base) in a temporary copy and mark it `fail (pre-existing, baseline <SHA>)`.

## 9. Scope

Cover your specialty fully over the assigned area, and state the scope covered in `Summary`. A confirmed defect
outside it may be reported briefly with evidence; do not perform an unassigned full review.

## 10. Result

- Per `report.md` §3: `NEEDS_DECISION` > `INCOMPLETE` > `CHANGES_REQUIRED` > `PASS`.
- A required check (your role file names them) `not run`, or `limited` in a required part → `INCOMPLETE`.
- The specialty has nothing to examine in the candidate → `INCOMPLETE` with the reason; only the orchestrator
  marks a specialty "not required". Examining the candidate and finding no risk is a result: `PASS` with the
  checks run.

## 11. Never

- Never edit, create, or commit files in the worktree, or move `HEAD`.
- Never return a `PASS` not backed by the checks run on the packet revision.
- Never review, or cite as evidence, a revision other than the packet's.
- Never read a reviewer report your packet does not include.
- Never include secrets, credentials, or real user data in a report.
- Never contact the user.
- Never block on a preference.
