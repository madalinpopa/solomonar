# Implementer correction mode

Framework version: {{ .Version }}

Load with `agents/implementer.md`; its common preflight, approvals, checks, and handoff gates apply.

Load `docs/solomonar/reports/corrections.md` before reading the consolidated list.

- Apply `agents/implementer.md` section 2's preflight and exact spec reference. Require the consolidated list's full
  `Revision reviewed`; verify `git merge-base --is-ancestor <reviewed-SHA> HEAD`. Account for later commits
  through packet references and history; do not infer orchestrator ownership from documentation paths.
- The orchestrator appends the correction round's steps to the plan; commit each as `agents/implementer.md` section 5 step 8 describes.
- Address exactly the consolidated list: resolve every `blocking` entry; apply `suggestion` entries only when
  compatible with approved specs and decisions; apply each listed approved decision within its scope. Never act
  on "Rejected" entries or raw reviewer text.
- A correction that needs a new dependency or architecture change → `agents/implementer.md` section 4.
- A new defect outside the list: report it under `Findings` with ID `IMPL-n` (`report.md` section 5); do not fix it.
- Use `agents/implementer.md` section 5's verification and handoff rules; rerun affected checks and those cited in findings.
- Report under the same result gates, listing each addressed `W-001/F-n` and its evidence; name suggestions
  not applied and why. Missing verification never becomes `READY_FOR_REVIEW` merely because a fix was committed.
