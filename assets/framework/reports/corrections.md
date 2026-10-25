# Consolidated correction list

Framework version: {{ .Version }}

Load with `docs/solomonar/report.md`; common evidence and result gates remain mandatory.

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
