# Review coordination

Framework version: {{ .Version }}

Orchestrator review/triage procedure. Load `report.md`; load `reports/corrections.md` before consolidation.

## Triage and gate

Before launching reviews, record each candidate's risk tier and which specialties apply, with reasons. The tier
goes into state with `task move --tier` (`rules.md` section 7); specialties and reasons stay in the session, then the PR
Verification section (`report.md` section 8).

| Tier | Candidate | Agent reviews |
|---|---|---|
| 0 | Documentation only, no behavior or requirement change | None; CI only |
| 1 | Changes outside tiers 0 and 2, including documentation that changes behavior or requirements | The required specialists per the table below |
| 2 | Persistence, authentication or authorization, concurrency, external integrations, data migration, or any approved architecture or dependency decision in the task | Specialists plus adversarial |

| Priority | Specialty | Role | Focus | Required when |
|---|---|---|---|---|
| 1 | Security | `security-reviewer` | Bugs and security defects | Every code change; documentation affecting security requirements or contracts (tier 1, 2) |
| 2 | Correctness | `correctness-reviewer` | Business logic | Every code change; documentation affecting business rules or contracts (tier 1, 2) |
| 3 | Spec | `spec-reviewer` | Work spec, canonical documents, `must` rules of applicable extensions | Every tier 1 or 2 change |
| - | Extension agent | the installed agent | Its specialty | Its `applies_when` matches the change |
| - | Adversarial | `adversarial-reviewer` | The integrated feature | Tier 2 (see Adversarial ordering) |

- The core reviewers are language-agnostic. Name in each packet the installed extensions whose `applies_when`
  or `paths` match the change; their `must` rules (`<name>/C-n`) are approved global rules (`rules.md` section 1).
- All required specialists run in parallel on the same revision. When the runtime cannot run them in
  parallel, or their checks interfere, run them in priority order.
- Mark an inapplicable specialty "not required" with a reason. Never fabricate a `PASS`.
- Use a different model for the adversarial reviewer when the runtime allows; record the model in the
  Verification section (`report.md` section 8).
- Every required reviewer must return a valid `PASS` on the current revision before the stage advances,
  unless its earlier verdict is reused under Candidate stability.
- A missing result or an important unrun check makes the stage `INCOMPLETE`, not `PASS`.

## Candidate stability

- Code and specs must stay unchanged while a review set runs.
- Reviewers follow `reviewer.md`'s read-only contract; run interfering checks sequentially.
- A changed candidate gets a new revision and an explicit assessment of affected checks. Reuse unaffected
  evidence only with its original revision and a recorded reason. An earlier `PASS` never covers new code.
- Updating the candidate from main follows `rules.md` section 8: CI always reruns; agent reviews rerun only on its conditions.

## Consolidation

- Deduplicate into stable task-scoped finding IDs (`W-001/F-1`). Clarify contradictions between reviewers.
- Finding fields and correction-list format: `report.md` sections 5 and 7.
- Rejecting a finding requires evidence-backed justification.
- Order the correction list by review priority (Security, Correctness, Spec, then extension agents); within one
  reviewer, by severity. A merged duplicate takes the highest-priority reviewer's position.
- Priority orders the list, not the choice between conflicting corrections. Resolve conflicts against evidence
  and the authority order (`rules.md` section 1) before delegation; conflicting requirements need a user decision (`rules.md` section 3).
- Classify findings and verification gaps per `report.md` sections 3–5: confirmed defects and mandatory
  violations block; suggestions do not; missing required verification makes the review incomplete.
- Disputed defects require reproduction, a test, or precise analysis; record the conclusion. Missing or
  contradictory policy, unclear product choices, and architecture/dependency changes follow `rules.md` section 3.
- Severity sets urgency by impact; blocking follows violated rules and acceptance conditions. Specialty labels
  and vote counts never override evidence. Apply the same arbitration to adversarial findings.

## Adversarial ordering

- Tier 2 only: start adversarial review after specialist blockers are closed.
- After an adversarial fix: implementer → relevant specialists recheck → adversarial recheck, all on the new revision.

## Stage procedures

### `specialized_review`
- **Entry:** `revision` recorded; candidate committed and unchanged. **Delegate:** required specialists, in parallel.
- **Do:** launch required specialists whose verdicts cannot be reused under `rules.md` section 6, including
  applicable installed extension agents. Delegate the same `revision`; wait for every launched report and
  consolidate (`workflow/reviews.md` → Consolidation checklist). Follow the rules' sequential fallback when parallel checks would interfere.
- **State:** maintain the full delegation set and next action; record blockers atomically when needed.
- **Exit:** every required specialist has a current `PASS` or justified reuse of one → move to
  `adversarial_review` for tier 2, `integration` for tier 1; update the next action.
- **Failure:** `CHANGES_REQUIRED` → correction list → `implementation`, appending one plan step per coherent
  correction commit (`fix(<ID>): …`) and committing the work spec first. `INCOMPLETE` → close the gap under
  `agents/orchestrator.md` section 3's retry rule. `NEEDS_DECISION` → `rules.md` section 3.

### `adversarial_review`
- **Entry:** tier 2; specialist gate valid for the candidate (`rules.md` section 6). **Delegate:** `adversarial-reviewer`,
  on a different model when the runtime allows (`rules.md` section 6).
- **Do:** unless a verdict can validly be reused, send the adversarial packet (Adversarial packet below), wait, and consolidate
  (`workflow/reviews.md` → Consolidation checklist).
- **State:** maintain delegation state and next action; record blockers when needed.
- **Exit:** current adversarial `PASS` or justified reuse of one under `rules.md` sections 6 and 8 → `integration`.
- **Failure:** `CHANGES_REQUIRED` → correction list → `implementation`, then relevant specialists and
  adversarial recheck on the new revision. `INCOMPLETE` → close the gap under `agents/orchestrator.md` section 3's retry rule.
  `NEEDS_DECISION` → `rules.md` section 3.

## Specialist packet

```text
Task: W-001, <title>    Role: <security-reviewer | correctness-reviewer | spec-reviewer | extension agent>; docs/solomonar/agents/<role>.md (extension agent: docs/solomonar/extensions/<name>/<name>.md)
Worktree: <path>; read-only: create or change no files here; reproductions in <external temp dir> with isolated test data
Revision: <full SHA>           Diff: <base SHA>..<full SHA>; on a recheck, base = last reviewed revision
Objective: review the candidate for <specialty>; recheck <none, or each W-001/F-n entry in report.md section 7 form>
Work spec: <exact path/spec_ref>; References: <spec sections, AC IDs, decisions, contracts>
Area of responsibility: <packages, files, flows>; Extensions: <matching installed extensions or none>
Checks available: <exact command; result; actual checked SHA; source>
Baseline: <full SHA>; known failures: <refs or none>   Constraints: ports <…>; test data <…>
Output: report per docs/solomonar/report.md; Revision must equal the SHA above
```

## Adversarial packet

```text
Task: W-001, <title>           Role: adversarial-reviewer; docs/solomonar/agents/adversarial-reviewer.md
<same fields as the specialist packet>
Form your own hypotheses from the requirements and the code BEFORE reading the block below.
=== EARLIER REVIEW SUMMARIES (read last) ===
<per reviewer: role, original revision, Result, mapped W-001/F-n IDs with one-line summaries; reuse reason if applicable>
=== END EARLIER REVIEW SUMMARIES ===
```

## Packet requirements

Apply `agents/orchestrator.md` section 4. Missing reviewer input is `INCOMPLETE` (`reviewer.md` section 2).

## Consolidation checklist

1. Apply `agents/orchestrator.md` section 6's report validation and missing-report gates.
2. Assess reuse under Candidate stability and `workflow/integration.md`; retain original evidence and reasons.
3. Map provisional IDs (`SEC-1`, `COR-1`, …) to `W-001/F-n`; merge duplicates; on rechecks keep IDs and `Status`.
   Order the correction list by review priority (`rules.md` section 6 Consolidation).
4. Resolve conflicting corrections against evidence and authority before delegation (`rules.md` section 6);
   priority orders the list, not the decision.
5. Apply finding classifications and verification gaps per `report.md` sections 3–5; required gaps block advancement.
6. Decision needs: assign the ID at its level (`P/D-n` or `W-001/D-n`) and follow `rules.md` section 3;
   hold only the corrections that depend on them.
7. Put rejected findings under "Rejected" with evidence-backed justification; produce one correction list
   (`report.md` section 7); keep the triage for the PR Verification section.
