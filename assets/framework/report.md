# Report contract

Framework version: 0.3.5

Delegated agent reports, the consolidated correction list, and the PR Verification section follow this file.
`rules.md` below means `docs/solomonar/rules.md`, which owns orchestration, routing, arbitration, and state.
Its references are for the orchestrator; delegated roles do not load it.

## 1. General rules

- Write reports in English.
- Omit empty optional sections. Never pad: there is no minimum finding count, and zero confirmed defects in
  the reviewed scope is a valid result.
- Reference evidence (file:line, command, test name, commit, link) instead of pasting logs. Keep any excerpt
  to about 20 lines.
- Never include secrets, credentials, or real personal data.

## 2. Report template

```markdown
# <Task ID>: <Role>
Revision: <full SHA, as defined below>
Result: <result value>

## Summary
<Outcome or review conclusion.>

## Checks and evidence
- <Entry per section 4, including revision and evidence source.>

## Findings
- <Entry per section 5.>

## Decision needed
- <Entry per section 6.>

## Next action
<Responsible actor and action, if work remains.>
```

- Header: exactly these three lines, in this order. Use the packet's task ID and one result from section 3.
  The header routes the report; acceptance also requires its evidence and the governing gates.
- `<Role>`: one of `implementer`, `security-reviewer`, `correctness-reviewer`, `spec-reviewer`,
  `adversarial-reviewer`, or an installed extension agent. `<Role>` is a role file in `docs/solomonar/agents/` (an extension agent's is `docs/solomonar/extensions/<name>/<name>.md`); `docs/solomonar/reviewer.md` is
  a shared contract, not a role.
- `Revision` is a full SHA:
  - Implementer handoff: committed candidate at worktree `HEAD`; an unchanged outcome may use existing `HEAD`.
  - Implementer with no candidate: verified starting commit. State "no candidate" in `Summary`; list any
    checkpoint SHA (step checkpoints or a `wip` commit) and preserved dirty/merge state separately. A checkpoint
    is not a reviewed candidate.
  - Reviewer: delegated SHA. If preflight fails, report that SHA, the observed `HEAD` when available, and
    that review did not run; never substitute the unexpected `HEAD` as the reviewed revision.
  Uncommitted work is never a candidate. An unestablished task SHA is an input failure: report the missing
  context without inventing a value; it cannot form a valid handoff and the orchestrator treats it as incomplete.
- Required sections: header, `Summary` (what was done or concluded, and the scope covered),
  `Checks and evidence`. Optional: `Findings`, `Decision needed`, `Next action`.
- Include `Findings` for `CHANGES_REQUIRED`, `Decision needed` for `NEEDS_DECISION`, and `Next action` for
  every non-`PASS` result. Next-action format: `<user | orchestrator | role name>: <action>`. Reports go to the
  orchestrator; suggested actions grant no approval, delegation, or state write. The orchestrator maps actors
  to actual owner/agent IDs before using CLI next-action fields.

## 3. Result values

| Value | Emitted by | Meaning |
|---|---|---|
| `READY_FOR_REVIEW` | implementer only | Assigned outcome verified, committed handoff clean, required checks pass; not an approval |
| `PASS` | reviewers only | Assigned scope completed; no unresolved blocking finding or required verification gap |
| `CHANGES_REQUIRED` | reviewers | At least one unresolved confirmed `blocking` finding |
| `NEEDS_DECISION` | any | A user decision is required before dependent work proceeds |
| `INCOMPLETE` | any | Required input, scope, or verification incomplete; also failed required checks or unresolved introduced failures for implementers |

Orchestrator routing and gates: `rules.md` sections 2 and 6.

- Report every applicable section, but set exactly one `Result`: the most restrictive applicable value.
  Reviewers: `NEEDS_DECISION` > `INCOMPLETE` > `CHANGES_REQUIRED` > `PASS`. Implementer:
  `NEEDS_DECISION` > `INCOMPLETE` > `READY_FOR_REVIEW`. Every finding and decision is still consolidated.
- `PASS` may coexist with `suggestion` findings and with `unverified` findings outside required verification.
- A failed required check prevents both `PASS` and `READY_FOR_REVIEW`, including a pre-existing failure.
  Implementer: `INCOMPLETE`. Reviewer: confirmed unresolved defect → `CHANGES_REQUIRED`; inability to complete
  required verification, including an environment/baseline failure without a defect finding → `INCOMPLETE`.
- A fresh `PASS` requires the delegated SHA and complete required verification. A `limited` check may count
  only when its unverified portion is outside required verification (section 4).
- A reviewer asked to review a specialty that does not apply to the candidate returns `INCOMPLETE` with the
  reason. Only the orchestrator marks a specialty "not required" (`rules.md` section 6).

## 4. Checks and evidence

- Entry: `<criterion/scenario and AC IDs>; <required | optional>; <exact command/steps>; <result>;
  <checked SHA, or target SHA if not run>; <performed | supplied, with source/evidence reference>`.
  Result: `pass` | `fail` | `not run` | `limited`. Requiredness follows the packet, spec, and role, not a report's waiver.
- Distinguish your checks from supplied evidence. Preserve the source and actual checked SHA; baseline
  results are not fresh candidate results. Older evidence retains its original revision; the orchestrator
  assesses reuse under `rules.md` sections 6 and 8, recording the report reference, scope, and reason.
- `not run` and `limited` must state the reason and what remains unverified. A required check that is
  `not run` forces `INCOMPLETE`; so does a `limited` check whose unverified part is required.
- Mark a failure reproduced at the packet's baseline as `fail (pre-existing, baseline <full SHA>)`, with
  evidence. Without that evidence, state that attribution is unverified. The label waives no required check.
- Record verification limits (environment, unexecutable scenario, missing access) here; reports have no
  separate Limits section.

## 5. Findings

| Classification | Meaning | Blocks? |
|---|---|---|
| `blocking` | Confirmed defect, or violation of a mandatory rule or acceptance criterion | While unresolved; name the violated rule/AC (or demonstrated defect) and evidence |
| `suggestion` | Improvement without a violated rule | Never |
| `unverified` | Plausible but unconfirmed hypothesis; states what would confirm or refute it | Only by leaving a required check incomplete (→ `INCOMPLETE`) |

Entry format (`Status` only on rechecks; classification-specific omissions below):

```markdown
- <ID> `<classification>`: <location: file:line, endpoint, or flow step>
  - Conditions: <when it occurs>
  - Impact: <consequence>
  - Violates: <rule or AC ID (`AC-n`, `<capability>/R-n`, `<name>/C-n`); when none exists, `demonstrated defect: <invariant broken>`>
  - Evidence: <reproduction steps, test, command; or reasoning plus the verification limit>
  - Correction: <the outcome that must hold, not a preferred implementation>
  - Status: <resolved | still open | regressed>
```

- IDs: the reporting role assigns provisional IDs with its prefix: `SEC-1`, `COR-1`, `SPEC-1`, `ADV-1`,
  `IMPL-1`. Extension agents use their role's declared prefix, otherwise `<role-name>-1`. Provisional IDs are
  report-local; identify them by report reference plus local ID. The reference identifies one emission
  (message/permalink or agent instance and round). The orchestrator maps them to stable task-scoped finding
  IDs (`W-001/F-1`); never assign an existing finding ID to a different issue.
- On a recheck, reference existing `W-001/F-n` IDs and state each one's `Status` on the new revision:
  `resolved` | `still open` (not fixed on this revision) | `regressed` (resolved on an earlier revision and
  failing again). Resolved findings retain their classification and ID but do not count as open blockers.
  If status cannot be established, omit `Status` and record the required verification gap; never claim
  `resolved`. New findings get new provisional IDs.
- Omit `Violates` for suggestions. For `unverified`, omit `Violates` and `Correction`; `Evidence` names the
  suspected rule/invariant and the check that would confirm or refute it. Never call an unconfirmed hypothesis
  a demonstrated defect or require its speculative fix.
- If the only valid correction requires a new architecture decision or dependency change, also add a
  `Decision needed` entry. Never prescribe the dependency as the fix.
- Never turn a new feature request or a preference into a defect. When the governing policy is missing or
  contradictory, report a `Decision needed` entry with the affected scenario instead of a `blocking` finding.

## 6. Decision needed

```markdown
- <ID>: <question>
  - Recommended: <option and why>; Alternatives: <other options>
  - Impact: <consequence of each option>
  - Waiting: <work awaiting the decision>; Independent: <work that can continue>
```

- IDs: provisional `D-1`, `D-2`, … within the report. The orchestrator assigns `W-001/D-n`; later reports
  reference that ID.
- The reporting agent may investigate but must not apply the choice.

## 7. Consolidated correction list (orchestrator → implementer)

The single list the implementer receives after consolidation (`rules.md` section 6):

```markdown
# <Task ID>: Corrections
Revision reviewed: <full SHA>

## Open findings
- W-001/F-n `<classification>`: <location>; [violates <rule/AC>;] evidence <ref>; correction: <outcome or verification needed>

## Approved decisions to apply
- <approved P/D-n or W-001/D-n>: <selected option; approval and scope reference>

## Rejected (do not act)
- <W-001/F-n or reviewer ID>: <evidence-backed justification>
```

- The implementer addresses only this list, never raw reviewer reports. It must resolve every `blocking`
  entry and applies suggestions only within `AGENTS.md`'s authority and the assigned scope. Omit `violates`
  for suggestions; unverified entries request verification, not speculative code changes. A decision listed
  here must already be approved within scope; listing it does not grant approval.
- The next `READY_FOR_REVIEW` report lists each addressed `W-001/F-n` in `Checks and evidence` with the
  check that shows the correction.

## 8. PR Verification section

```markdown
## Verification
- Task: W-001
- Candidate: <full SHA>; integration baseline: <incorporated configured main branch's full SHA>
- Final specification: <spec_ref: final spec commit:path, written before merge>
- Acceptance: <AC IDs, outcomes, and evidence>
- Checks: <section 4 entries, including required checks not run and their reasons>
- Reviews: tier <0 | 1 | 2> and why; <verdicts, original SHAs, models when known, report references, reuse reasons>
- Decisions: <scoped approval references relevant to the change>
- Limits: <unverified behavior or environment limitations, if any>
```

- The orchestrator writes it in the feature PR description before merge and updates it whenever the
  candidate changes.
- Every verdict names its original revision; reused verdicts also name the unaffected scope and reason.
  List specialties marked "not required" with their reason. Never turn missing results into passing checks
  or invent a model name. Omit `Limits` only when nothing is unverified; reference check entries to avoid duplication.
- Report references point to a durable location when one exists; otherwise write `session only (not durable)`.
- This summary remains after closure; it does not make a lost report durable or replace missing evidence.
  Resume verification follows `rules.md` section 10. Tier 0: `Reviews` states the tier and "CI only".
