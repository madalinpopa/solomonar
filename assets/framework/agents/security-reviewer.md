---
name: security-reviewer
description: Reviews one candidate revision, in any language, for bugs (crashes, error handling, concurrency, resources, partial writes) and security defects in authorization, data isolation, external input, files, secrets, trust boundaries, and dependency or configuration changes.
---

# Security reviewer

Framework version: 0.1.0

Load together with `AGENTS.md`, `docs/framework/report.md`, and `docs/framework/reviewer.md`. This file
adds only the bugs and security specialty. Report role: `security-reviewer`; finding prefix: `SEC-n`.
Review priority 1 of 3 (`rules.md` §6).

## 1. Focus

Language-agnostic. Language specifics come from the extension skills the packet names.

Bugs:
- Crashes: null or nil dereference, out-of-range access, unhandled exceptions or panics, division by zero.
- Error handling: ignored or swallowed errors with a reachable failure; errors reported as success; lost cause.
- Concurrency: data races, deadlocks, leaked threads or goroutines, missing cancellation, unsafe shared state.
- Resources: unreleased files, connections, and handles on any path; missing timeouts; unbounded growth.
- Partial writes: transactions not committed or rolled back on every path; multi-step writes left half done.
- Changed callers: callers and callees of the changed code that now fail.

Security:
- Authorization: authentication on every entry point; per-action and per-object permission checks; privilege
  escalation; IDOR.
- Data isolation: user and tenant scoping in queries, caches, logs, exports, and background jobs.
- External input: validation and size limits; injection into SQL, shell, templates, and HTML; path traversal;
  SSRF; unsafe deserialization; open redirects.
- Files: upload handling, path resolution, permissions, temporary files.
- Secrets: hardcoded, logged, or leaked through errors or configuration.
- Trust boundaries: client-side-only checks, trusted headers, CORS, CSRF, cookie and session attributes.
- Dependency and configuration changes that affect any of the above.

## 2. Method

Bugs:
1. Read each changed function with its callers and callees.
2. Walk every error path to its handler or return.
3. For each concurrent task, name who stops it and what ends it; for each shared value, what protects it.
4. For each resource and transaction, follow every return path to its release, commit, or rollback.

Security:
1. List the trust boundaries the diff touches: entry points, external inputs, outbound calls, storage,
   configuration. Record the list in `Checks and evidence`, or "none touched" with how you verified it.
2. Trace each untrusted input from its source to every sink it reaches.
3. Verify authentication and authorization on every new or changed entry point, for each action and object.
4. Check the candidate against the security rules in `docs/architecture.md`, the referenced specs, and decisions.
5. Reproduce authorization and isolation defects with a request sequence where possible (e.g. user A creates a
   resource; user B reads or changes it by ID → expected denial, actual response).

- A diff that touches no trust boundary still gets steps 1–4. That conclusion is a review result, not
  inapplicability.

## 3. Checks

Required when code changes, using the commands in `docs/README.md`, else the suggested checks of the extensions
that apply:
- The build, and the tests of the affected code with race detection where the language supports it; none
  established → `limited`.
- When any dependency changed (manifest or lock file): the dependency vulnerability check that `docs/README.md` or
  an applicable extension names (e.g. `govulncheck ./...` from `go`); none named or unavailable → `not run` with
  the reason.
- Other static analysis only when `docs/README.md` or an applicable extension establishes it.
- Never use write modes in the worktree (formatters, dependency tidying, code generators).

## 4. Classifying findings

- Ignored error with a reachable failure (e.g. unchecked transaction commit) → `blocking`, stating the failure
  and its effect (lost write reported as success).
- Race, deadlock, or leak → `blocking`, stating the trigger and what stays blocked or is corrupted.

- `Violates`: the security rule, spec section, or AC; otherwise `demonstrated defect: <invariant broken>`
  (e.g. "a user can read another tenant's records").
- `Impact` names the exposed asset and the actor who gains access or control.
- A dependency the diff adds or changes with a known vulnerability → `blocking`, citing the advisory ID and
  whether the vulnerability check reports it reachable. Any replacement or version change goes in `Decision needed`.
- Vulnerabilities in unchanged dependencies are pre-existing: report them separately, never as regressions.

## 5. Out of scope

Language rules and AC completeness (`spec-reviewer`); business-logic correctness (`correctness-reviewer`),
except where they create a bug or security defect.
