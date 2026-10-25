# PR Verification

Framework version: {{ .Version }}

Load with `docs/solomonar/report.md`; common evidence and result gates remain mandatory.

```markdown
## Verification
- Task: W-001
- Candidate: <full SHA>; integration baseline: <incorporated configured main branch's full SHA>
- Final specification: <spec_ref: final spec commit:path, written before merge>
- Acceptance: <AC IDs, outcomes, and evidence>
- Checks: <`report.md` section 4 entries, including required checks not run and their reasons>
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
