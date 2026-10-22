# Go contexts and I/O

- go/C-8 (must) — Rule: Accept `context.Context` first in project-controlled
  cancellable APIs and propagate it to blocking work.
  Exception: Preserve imposed signatures; obtain context where their contract
  permits and document cancellation limits.
- go/C-12 (must) — Rule: Arrange release of owned resources and derived-context
  cancel functions on every exit path; make ownership transfers explicit.
- go/C-13 (must) — Rule: Ensure owned transactions reach commit or rollback;
  report success only after `Commit` succeeds. Deferred rollback is valid cleanup.
- go/C-14 (should) — Rule: Release per-iteration resources within the iteration
  when retaining them until function return would accumulate live resources.
- go/C-15 (must) — Rule: Bound finite network operations with effective deadlines
  or timeouts, including caller-provided bounds.
  Exception: Long-lived streams require lifecycle cancellation and appropriate
  phase or idle limits instead of a finite total duration.
- go/C-31 (must) — Rule: Pass context per operation rather than storing it in
  structs. Exception: An imposed interface may require storing it.
- go/C-32 (must) — Rule: Use context values for request-scoped metadata, not
  dependencies or optional function parameters.
- go/C-33 (must) — Rule: Check `sql.Rows.Err()` after iteration before reporting success.
- go/C-34 (should) — Rule: Choose server phase and idle timeouts for the protocol
  and deployment, accounting for limits already enforced by infrastructure.
