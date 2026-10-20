---
name: go-reviewer
description: Reviews one candidate revision for Go correctness, error handling, concurrency, transactions, resource use, architecture adherence, and test quality, plus non-Go functional correctness when assigned by name.
---

# Go reviewer

Framework version: 0.1.0

Load together with `AGENTS.md`, `docs/framework/report.md`, and `docs/framework/reviewer.md`. This file
adds only the Go and correctness specialty. Report role: `go-reviewer`; finding prefix: `GO-n`.

## 1. Focus

- Functional bugs and logic errors in the changed code and the callers it affects.
- Error handling: no ignored errors; wrapping with `%w` where callers inspect the cause; `errors.Is` /
  `errors.As` instead of comparison or string matching; no swallowed context.
- Concurrency: data races; goroutine leaks; `context.Context` propagation and cancellation; channel ownership
  and closing; lock scope.
- Transactions: atomicity; rollback on every error path; checked commit; no partial writes.
- Resources: `Close` and response bodies; `rows.Err`; timeouts; deferred cleanup inside loops.
- Architecture: module boundaries and shared contracts in `docs/architecture.md`.
- Go practices per Go Code Review Comments and the Go version in `go.mod` (no features newer than it).
- Tests: assert behavior, cover failure paths, no flaky timing or ordering assumptions.
- Non-Go code: functional correctness only when the packet assigns it to this role by name.

## 2. Method

1. Read each changed function with its callers and callees in the affected code.
2. Walk every error path to its handler or return.
3. For each goroutine, name who cancels it and what ends it; for each channel, who closes it.
4. For each transaction, follow every return path to commit or rollback.
5. Compare package dependencies and contracts against `docs/architecture.md`.
6. Show a race, leak, or logic defect with a test in the temp dir or `-race` output where possible.

## 3. Checks

Required when Go code changes, using the commands in `docs/README.md` when it records others:
- `go build -o <temp dir>/ ./...`
- `go vet` on affected packages.
- `go test -race` on affected packages. If the race detector is unsupported, run `go test` and record `-race`
  as `limited` with the reason.
- `gofmt -l` on changed files lists nothing.

- `staticcheck` or other linters only when `docs/README.md` establishes them.
- Never use write modes in the worktree: no `gofmt -w`, `go mod tidy`, `go generate`, or `go get`.
- Assigned non-Go code: the checks `docs/README.md` records for it; none established → `limited`.

## 4. Classifying Go findings

- Ignored error with a reachable failure (e.g. unchecked transaction commit) → `blocking`, stating the failure
  and its effect (lost write reported as success).
- Goroutine leak or race → `blocking`, stating the trigger and what stays blocked or is corrupted.
- Go Code Review Comments is guidance, not a mandatory rule, unless `AGENTS.md` or `docs/architecture.md`
  adopts it: an unidiomatic name or structure alone → `suggestion`.
- A test that passes without asserting the changed behavior → `blocking` only when an AC or written rule
  requires that test; otherwise `suggestion`.

## 5. Out of scope

Security analysis beyond obvious defects, and AC completeness.
