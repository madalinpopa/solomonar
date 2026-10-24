---
name: go
description: Engineering rules for writing or reviewing Go source, module changes, and affected tests.
---

# Go language rules

Framework version: 0.3.0

Cite `go/C-n`; apply framework authority and review gates.

## Common rules

- go/C-1 (must) — Rule: Keep source and standard-library usage compatible with
  the module's declared Go version and supported build configurations.
  Exception: Newer-version files may use appropriate version build constraints
  when older supported builds retain a working implementation.
- go/C-4 (must) — Rule: Handle or return errors that can affect the operation's
  result; justify intentional discards when the reason is not evident or documented.
- go/C-5 (must) — Rule: Use `errors.Is` or `errors.As` for cause checks that must
  survive wrapping. Exception: Match error text when that text is the contract.
- go/C-6 (should) — Rule: Add error context only when it identifies useful
  operation or input information absent from the cause.
- go/C-7 (must) — Rule: Return errors for expected operational failures.
  Exception: Documented `Must` APIs may panic on invalid programmer-supplied input.
- go/C-21 (must) — Rule: Preserve cause inspection required by the API; use `%w`
  when adding text. Do not expose a cause that the API contract hides.

## Load by impact

Evaluate every trigger against changed code and affected callers/contracts.
Load matching files before writing or reviewing; expand as impact emerges.
Paths are relative to this file. Do not load research during ordinary tasks.

| Impact | Load |
| --- | --- |
| API/type contracts, package structure, exported names, dependency selection, or optimization | `rules/go/design.md` |
| Contexts, blocking operations, resources, networking, or transactions | `rules/go/io.md` |
| Goroutines, channels, shared mutable state, or synchronization | `rules/go/concurrency.md` |
| Choosing Go test coverage or writing/reviewing affected Go tests | `rules/go/testing.md` |

Spec reviewers account for loaded `must` rules and give one non-applicability
reason per unloaded file. A missing required file is missing input.

## Checks

Use project commands without duplicating covered checks. Fallback: build affected
packages, `gofmt -l` changed files, `go vet`, and `go test -race`; include affected
consumers. Unsupported `-race`: run ordinary tests and report the verification gap.
Module changes require module-wide build/test and `govulncheck ./...`.

Run configured linters only. Bound targeted fuzz/benchmark runs.
