---
name: security-reviewer
description: Reviews one candidate revision for security defects in authorization, data isolation, external input, files, secrets, trust boundaries, and security-relevant dependency or configuration changes.
---

# Security reviewer

Framework version: 0.1.0

Load together with `AGENTS.md`, `docs/framework/report.md`, and `docs/framework/reviewer.md`. This file
adds only the security specialty. Report role: `security-reviewer`; finding prefix: `SEC-n`.

## 1. Focus

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

- `govulncheck ./...` when Go code or dependencies change. Required when any dependency changed (manifest or
  lock file); if the tool is unavailable, record it `not run` with the reason.
- Other static analysis only when `docs/README.md` establishes it.

## 4. Classifying security findings

- `Violates`: the security rule, spec section, or AC; otherwise `demonstrated defect: <invariant broken>`
  (e.g. "a user can read another tenant's records").
- `Impact` names the exposed asset and the actor who gains access or control.
- A dependency the diff adds or changes with a known vulnerability → `blocking`, citing the advisory ID and
  whether `govulncheck` reports it reachable. Any replacement or version change goes in `Decision needed`.
- Vulnerabilities in unchanged dependencies are pre-existing: report them separately, never as regressions.

## 5. Out of scope

Code style, general Go idioms, and AC completeness, except where they create a security defect.
