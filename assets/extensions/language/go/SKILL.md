---
name: go
description: Rules for Go code in this project. Use when a change touches Go code (`*.go`, `go.mod`, `go.sum`), before writing it (implementer) or reviewing it (spec-reviewer checks every `must` rule).
---

# Go language rules

Framework version: 0.1.3

Installed by `solomonar` when the project adopts Go (`language` group). The implementer follows these rules; the
`spec-reviewer` checks every `must` rule on each candidate that touches Go code. Cite a rule as `go/C-n`.
`must` rules are approved global rules: a violation is a `blocking` finding. `should` rules are suggestions.

## Rules

### Language version and formatting

- go/C-1 (must): Use no language feature or standard library API newer than the `go` directive in `go.mod`.
- go/C-2 (must): Every changed `.go` file is `gofmt`-formatted.
- go/C-3 (must): Generated files are never edited by hand; their generator is the source.

### Errors

- go/C-4 (must): No ignored error return, unless a comment states why the error cannot matter.
- go/C-5 (must): An error returned to a caller that inspects its cause is wrapped with `%w`; callers use
  `errors.Is` or `errors.As`, never `==` on wrapped errors or string matching.
- go/C-6 (should): A wrapped error adds the context of the failed operation, not a repeat of the cause.
- go/C-7 (must): No `panic` for an expected failure; a library package returns errors.

### Context and concurrency

Context convention: [Go Code Review Comments: Contexts](https://go.dev/wiki/CodeReviewComments#contexts).

- go/C-8 (must): A project-controlled API that does cancellable I/O or blocking work takes `context.Context`
  as its first parameter and propagates it. Preserve signatures imposed by interfaces, callbacks, and entry
  points (e.g. `io.Reader.Read` or `main`); obtain or pass context where that contract permits and explain any
  cancellation limit. Do not store context in a struct unless an imposed interface signature requires it.
- go/C-9 (must): Every goroutine has an owner that can stop it and a path by which it ends.
- go/C-10 (must): Each channel has one owner that closes it; no send on a channel that may be closed.
- go/C-11 (should): A lock protects the smallest scope that keeps the invariant; no lock held across I/O.

### Resources and transactions

- go/C-12 (must): Every opened resource (`Close`, HTTP response bodies, `sql.Rows`) is released on every path;
  `rows.Err()` is checked after iteration.
- go/C-13 (must): A transaction is committed or rolled back on every return path, and the `Commit` error is
  checked.
- go/C-14 (should): No `defer` that accumulates inside a loop; release per iteration.
- go/C-15 (must): Outbound network calls and servers set explicit timeouts.

### Packages and style

- go/C-16 (must): Package imports respect the module boundaries in `docs/architecture.md`; no import cycles
  worked around with globals.
- go/C-17 (should): Names, comments, and structure follow
  [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). An unidiomatic name alone is a suggestion.

### Tests

- go/C-18 (must): Tests run with `-race` pass; tests make no timing or ordering assumptions that cause flakes.
- go/C-19 (should): Test helpers call `t.Helper()`; failures name the input and the expected and actual values.

## With tdd
<!-- solomonar:with tdd -->

- go/C-20 (must): Behavior tests are table-driven with one `t.Run` per case, named after the behavior.

## Suggested checks

Run with the commands in `docs/README.md` when it records others. Never use write modes in the worktree
(`gofmt -w`, `go mod tidy`, `go generate`, `go get`) during review.

| Check | Rules | When |
|---|---|---|
| `go build -o <temp dir>/ ./...` | build | Go code changes |
| `gofmt -l <changed files>` lists nothing | go/C-2 | Go code changes |
| `go vet <affected packages>` | go/C-4, go/C-10 | Go code changes |
| `go test -race <affected packages>`; unsupported race detector → `go test`, record `-race` as `limited` | go/C-18 | Go code changes |
| `govulncheck ./...` | dependency vulnerabilities (security-reviewer) | `go.mod` or `go.sum` changes |
| `staticcheck` or other linters | - | only when `docs/README.md` establishes them |
