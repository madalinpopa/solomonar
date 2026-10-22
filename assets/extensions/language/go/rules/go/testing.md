# Go testing

Framework procedures own acceptance coverage, test-first workflow, and verification gates.

- go/C-18 (must) — Rule: Use controlled inputs and event synchronization so
  results do not depend on wall-clock delays, random fixtures, or scheduling.
- go/C-19 (should) — Rule: Mark test helpers with `t.Helper()` when failure
  locations should point to their callers.
- go/C-38 (should) — Rule: Make failures identify the scenario, actual result,
  and expected result without requiring a second run.
- go/C-39 (should) — Rule: Use tables for cases sharing test logic; use subtests
  when independent names, lifecycle, or filtering help. Separate cases whose
  setup or assertions would require branch-heavy table machinery.
- go/C-40 (must) — Rule: Isolate mutable state before parallelizing tests;
  do not call `t.Setenv` or `t.Chdir` under parallel ancestry.
- go/C-41 (should) — Rule: Tie fixtures to test lifetime with `t.TempDir()` and
  `t.Cleanup()`; use `t.Context()` for test-lifetime cancellation on Go 1.24+.
  Exception: Use a separately scoped context when exercising cancellation or
  when cleanup must continue after the test context is canceled.
- go/C-42 (should) — Rule: Consider fuzzing input-heavy code with a meaningful
  property; keep targets deterministic, isolated, and inexpensive.
