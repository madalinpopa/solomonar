# Report contract

Framework version: 0.1.0

Every agent report, the consolidated correction list, and the PR Verification section follow this file.
`rules.md` below means `docs/solomonar/rules.md`, which owns orchestration, routing, arbitration, and state.

## 1. General rules

- One report contract for every agent. Write reports in English only.
- Omit empty optional sections. Never pad: there is no minimum finding count, and zero confirmed defects in
  the reviewed scope is a valid result.
- Reference evidence (file:line, command, test name, commit, link) instead of pasting logs. Keep any excerpt
  to about 20 lines.
- Never include secrets or credentials in a report.

## 2. Report template

```markdown
# <Task ID>: <Role>
Revision: <exact candidate commit, full SHA>
Result: <result value>

## Summary
<Outcome or review conclusion.>

## Checks and evidence
- <Criterion/scenario; check performed; result; evidence reference.>

## Findings
- <ID; blocking/suggestion/unverified; location; conditions; impact;
  violated rule/AC; evidence; required correction.>

## Decision needed
- <ID; question; recommended option; alternatives; impact;
  work awaiting the decision.>

## Next action
<Responsible actor and action, if work remains.>
```

- Header: exactly these three lines, in this order; the orchestrator routes on them alone. `<Task ID>`: the
  task ID from the delegation packet (e.g. `W-001`). `Result`: exactly one value from §3.
- `<Role>`: one of `implementer`, `security-reviewer`, `correctness-reviewer`, `spec-reviewer`,
  `adversarial-reviewer`, or an installed extension agent. `<Role>` is a role file in `docs/solomonar/agents/`; `docs/solomonar/reviewer.md` is
  a shared contract, not a role.
- `Revision`: the full SHA of a committed candidate. Implementer: the commit it hands off. Reviewer: the
  exact commit it reviewed. Uncommitted work is never a candidate. A review of one revision says nothing
  about any other revision. If no candidate was committed (e.g. the implementer stops on `NEEDS_DECISION`
  before handoff), give the commit the work started from and state in `Summary` that no candidate exists.
- Required sections: header, `Summary` (what was done or concluded, and the scope covered),
  `Checks and evidence`. Optional: `Findings`, `Decision needed`, `Next action`.
- `Next action` is required whenever `Result` is not `PASS`. Format: `<actor>: <action>`; the actor is
  `user`, `orchestrator`, or a role name.

## 3. Result values

| Value | Emitted by | Meaning | Orchestrator handling |
|---|---|---|---|
| `READY_FOR_REVIEW` | implementer only | Candidate committed and handed off; not an approval | `rules.md` §2 |
| `PASS` | reviewers only | Assigned scope completed; no unresolved blocking finding; no required verification gap | Gate in `rules.md` §6 (Triage and gate) |
| `CHANGES_REQUIRED` | reviewers | At least one confirmed `blocking` finding | `rules.md` §2 |
| `NEEDS_DECISION` | any | A user decision is required before dependent work proceeds | `rules.md` §2 |
| `INCOMPLETE` | any | Required evidence or review execution is missing (check not run, environment failure, scope not covered) | `rules.md` §2 |

- Report every applicable section, but set exactly one `Result`: the most restrictive applicable value.
  Reviewers: `NEEDS_DECISION` > `INCOMPLETE` > `CHANGES_REQUIRED` > `PASS`. Implementer:
  `NEEDS_DECISION` > `INCOMPLETE` > `READY_FOR_REVIEW`. Every finding and decision is still consolidated.
- `PASS` may coexist with `suggestion` findings and with `unverified` findings outside required verification.
- `PASS` is valid only when `Revision` equals the delegated revision and no required check is `not run`.
- A reviewer asked to review a specialty that does not apply to the candidate returns `INCOMPLETE` with the
  reason. Only the orchestrator marks a specialty "not required" (`rules.md` §6).

## 4. Checks and evidence

- Entry: `<criterion or scenario, AC ID when applicable>; <exact command or scenario steps>; <result>;
  <evidence reference>`. Result: `pass` | `fail` | `not run` | `limited`.
- `not run` and `limited` must state the reason and what remains unverified. A required check that is
  `not run` forces `INCOMPLETE`; so does a `limited` check whose unverified part is required.
- Mark a failure present before the candidate as `fail (pre-existing, baseline <ref>)`, using the baseline
  recorded per `rules.md` §11. Never mix pre-existing failures with failures the candidate introduces.
- Record verification limits (environment, unexecutable scenario, missing access) here; reports have no
  separate Limits section.

## 5. Findings

| Classification | Meaning | Blocks? |
|---|---|---|
| `blocking` | Confirmed defect, or violation of a mandatory rule or acceptance criterion | Yes; must name the violated rule/AC (or the demonstrated defect) and carry evidence |
| `suggestion` | Improvement without a violated rule | Never |
| `unverified` | Plausible but unconfirmed hypothesis; states what would confirm or refute it | Only by leaving a required check incomplete (→ `INCOMPLETE`) |

Entry format (omit `Violates` for `suggestion`; `Status` only on rechecks):

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
  `IMPL-1`. The orchestrator maps them to stable task-scoped IDs (`W-001/F-1`).
- On a recheck, reference existing `W-001/F-n` IDs and state each one's `Status` on the new revision:
  `resolved` | `still open` (not fixed on this revision) | `regressed` (resolved on an earlier revision and
  failing again). New findings, including a new defect introduced by a fix, get new provisional IDs.
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

The single list the implementer receives after consolidation (`rules.md` §6):

```markdown
# <Task ID>: Corrections
Revision reviewed: <full SHA>

## Open findings
- W-001/F-n `<classification>`: <location>; violates <rule/AC>; evidence <ref>; correction: <outcome>

## Approved decisions to apply
- W-001/D-n: <selected option; scope reference>

## Rejected (do not act)
- <W-001/F-n or reviewer ID>: <evidence-backed justification>
```

- The implementer addresses only this list, never raw reviewer reports. It must resolve every `blocking`
  entry and applies `suggestion` entries only within `rules.md` §1.
- The next `READY_FOR_REVIEW` report lists each addressed `W-001/F-n` in `Checks and evidence` with the
  check that shows the correction.

## 8. PR Verification section

```markdown
## Verification
- Task: W-001
- Candidate: <full SHA>; integration baseline: <main SHA or integrated candidate>
- Final specification: <spec_ref: final spec commit:path, written before merge>
- Acceptance: <AC IDs, outcomes, and evidence>
- Checks: <commands/scenarios actually run and results>
- Reviews: tier <0 | 1 | 2> and why; <required specialist and adversarial verdicts, revisions, models, report references>
- Decisions: <scoped approval references relevant to the change>
- Limits: <unverified behavior or environment limitations, if any>
```

- The orchestrator writes it in the feature PR description before merge and updates it whenever the
  candidate changes.
- Every verdict names the revision it covers. List specialties marked "not required" with their reason.
  List only checks actually run. Omit `Limits` only when nothing is unverified.
- Report references point to a durable location when one exists; otherwise write `session only (not durable)`.
- This section is the durable evidence kept after closure. Tier 0: `Reviews` states the tier and "CI only".

## 9. Example (illustrative)

```markdown
# W-001: security-reviewer
Revision: <FULL_SHA>
Result: CHANGES_REQUIRED

## Summary
Reviewed login and session handling at <FULL_SHA>. One blocking finding; one suggestion.

## Checks and evidence
- AC-3 locked account cannot log in; `go test ./internal/auth -run TestLocked`; pass; auth_test.go:88
- Login rate limit; 200 req/s load scenario; not run (no load environment; not required); limit unverified

## Findings
- SEC-1 `blocking`: internal/auth/session.go:42
  - Conditions: logout from any client
  - Impact: a stolen session token stays valid until expiry
  - Violates: AC-5 (logout invalidates the session server-side)
  - Evidence: log in, copy token, log out, reuse token → HTTP 200 (temporary test outside the worktree)
  - Correction: a token presented after logout must be rejected
- SEC-2 `suggestion`: internal/auth/cookie.go:12, internal/auth/login.go:30
  - Impact: cookie attributes are set in two places; one helper would prevent drift
  - Correction: a single source for session cookie attributes

## Next action
orchestrator: consolidate SEC-1 and return it to the implementer
```
