# Phase 1 validation — framework v0.1.0

Abbreviations: R = `rules.md`, P = `report.md`, O = `orchestrator.md`, I = `implementer.md`, Rv = `reviewer.md`,
SEC/COR/SPC/ADV = specialist role files, A = `AGENTS.md`, sk:x = skill `x` `SKILL.md`. `§n` is the file's section n.
Source of intent: [`framework-v0.1.md`](framework-v0.1.md). Role names follow the current framework; the B8 line
counts are those measured at validation.

## 1. Verdict

**Coherent after fixes.** Eight confirmed cross-file contradictions or list omissions were repaired (section 5).
The five required scenarios trace end to end; the remaining gaps are design choices (commit formats for state and
decision commits, writer of non-decision "Spec changes", empty final spec commit, decision entry status, loop-stop
blocker kind) and are listed in section 6. No gap blocks a walkthrough; each needs one author decision.

## 2. Scenario traces

Feature: product "Notes", flow `login`, task `W-001` "Login", decision `W-001/D-1`, defect `SEC-1` → `W-001/F-1`.

**S1 — A static flow becomes selected work**

| # | Actor | Governing § | State change | Artifacts | Result | Gap |
|---|---|---|---|---|---|---|
| 1 | user + `prototype` | sk:prototype §3–6 | flow `login` `draft` | `prototype/login.html`, `prototype/README.md`; `prototype(login): add login flow` | – | none |
| 2 | user + `prototype` | sk:prototype §5, §7 | `login` → `ready for spec` | `prototype(login): mark login ready for spec`; ref `<SHA>:prototype/login.html — flow login` | – | none |
| 3 | `spec` (discovery) | sk:spec §4.1–4.6 | `auth/R-1..R-3` **Observed**; `SIMULATED` session → `auth/Q-1`, needs decision: yes | `docs/specs/auth.md` | – | none |
| 4 | user + `spec` | sk:spec §4.7–4.9, §7 | R-1..R-3 **Agreed** (`user 2026-10-20`); Q-1 **Open** | `docs(auth): add login rules` (+ index in `docs/README.md`) | – | none |
| 5 | orchestrator-1 | R §4.1–4.3, §7; O §3 `setup` | `W-001` `reserved`/`setup`, owner, branch, worktree, `next_action` | `state.json` commit on `workflow-state`; *Registration confirmed* | – | no rule (message format) → Q3 |
| 6 | orchestrator-1 | R §4.4–4.5 | `running`/`specification` | `feature/W-001`, worktree `../notes-W-001`; state commit | – | none |
| 7 | orchestrator-1 + `work-spec` | sk:work-spec §3–4 | `spec: docs/work/W-001-login.md` | AC-1..AC-5 (AC-5: logout invalidates the session server-side); Q-1 answered by `docs/architecture.md`; "Spec changes: auth/new: …"; `spec(W-001): draft task specification` | – | none |
| 8 | orchestrator-1 | sk:work-spec §5; R §4.6 | stage `implementation` | `Readiness: READY`; state commit | – | none |

**S2 — Implementation needs an unapproved dependency**

| # | Actor | Governing § | State change | Artifacts | Result | Gap |
|---|---|---|---|---|---|---|
| 1 | orchestrator-1 | O §3 `implementation`, §4 | `active_agents` [impl-1] | implementer packet; state commit | – | none |
| 2 | impl-1 | I §4 | – | needs a session library (new dependency); `wip(W-001): add login handler without session store` | `NEEDS_DECISION` (Revision = baseline, `D-1`) | none |
| 3 | orchestrator-1 | O §6.1 (F6), R §2, §3 steps 1–5 | `W-001/D-1`; blocker `B-1` `decision` ref D-1; `next_action` user; `blocked`/`implementation` | state commit; *Decision needed — W-001/D-1* | – | none |
| 4 | user | R §3 | approves option A, scope "W-001 session storage" | – | – | none |
| 5 | orchestrator-1 + `spec` in-task | R §3 On approval; O §3; sk:spec §5 | blocker cleared; back to `running`/`implementation` | `docs/decisions.md` entry `approved`; `auth/Q-1` deleted, rule source `W-001/D-1`; one worktree commit; work spec cites D-1 (`spec(W-001): refine task specification`); state commit | – | no rule (commit format) → Q3 |
| 6 | impl-2 | I §5 | – | `go.mod`/`go.sum`, `docs/stack.md`; `feat(W-001): add server-side sessions` | `READY_FOR_REVIEW` | none |
| 7 | orchestrator-1 | O §3 `implementation` State | `revision` = SHA; `specialized_review` | state commit | – | none |

**S3 — A reviewer proves a defect**

| # | Actor | Governing § | State change | Artifacts | Result | Gap |
|---|---|---|---|---|---|---|
| 1 | orchestrator-1 | R §6 Triage; O §3 | security, correctness, spec required; `active_agents` sec-1, cor-1, spec-1 | three packets, same `revision`; state commit | – | none |
| 2 | sec-1 | SEC §2, §4; Rv §7 | – | `SEC-1` `blocking`, Violates AC-5, request-sequence evidence | `CHANGES_REQUIRED` | none |
| 3 | spec-1 / cor-1 | SPC §4 / COR §3 | – | `SPEC-1` (AC-5) / none | `CHANGES_REQUIRED` / `PASS` | none |
| 4 | orchestrator-1 | O §6; R §6 Consolidation; P §7 | stage `implementation` | SEC-1 + SPEC-1 → `W-001/F-1`; one correction list | – | none |
| 5 | impl-3 | I §6 | – | `fix(W-001): invalidate session on logout` | `READY_FOR_REVIEW` (lists F-1) | none |
| 6 | orchestrator-1 | R §6 Candidate stability; O §3 | new `revision`; `specialized_review` | affected verdicts: security, correctness, spec (code changed) | – | none |
| 7 | sec-2, cor-2, spec-2 | Rv §3 recheck | – | F-1 `Status: resolved` | `PASS` ×3 | none |
| 8 | adv-1 | ADV §3; O §4 adversarial packet | `adversarial_review` → `integration` | scenarios: reuse token after logout, concurrent logout, restart | `PASS` | none |

**S4 — The feature merges**

| # | Actor | Governing § | State change | Artifacts | Result | Gap |
|---|---|---|---|---|---|---|
| 1 | orchestrator-1 | R §9 Before merge; sk:work-spec §4 | – | apply "Spec changes" `auth/new` to `docs/specs/auth.md` | – | conflicting rules → Q4 |
| 2 | orchestrator-1 | R §9; O §3 `integration` | new `revision`; earlier verdicts reused with recorded reason (F7) | `spec(W-001): finalize task specification` | – | ambiguous (empty commit) → Q5 |
| 3 | orchestrator-1 | O §3; P §8 | `pr` | push; PR with Verification; state commit | – | none |
| 4 | orchestrator-1 | R §8 | – | required checks pass; merge commit, no squash | – | none |
| 5 | orchestrator-1 | R §9.1; O §5 | `done`/`compaction`, `merge_commit` | state commit; *Task merged* | – | none |
| 6 | orchestrator-1 | R §9.2; O §3 | `spec_ref` = `<final spec SHA>:docs/work/W-001-login.md` | `git show`, `git merge-base --is-ancestor` | – | none |
| 7 | orchestrator-1 | R §9.3 | `cleanup_pr` | branch `cleanup/W-001` off main; `docs(W-001): compact completed task`; PR | – | none |
| 8 | orchestrator-1 | R §9.4, §7 compact record | `done`/`complete`; compact record | state commit; *Task complete* | – | none |

**S5 — The orchestrator restarts after merge**

| # | Actor | Governing § | State change | Artifacts | Result | Gap |
|---|---|---|---|---|---|---|
| 1 | new session (user names W-001) | O §2; R §10 | – | inspect record (`running`/`integration`, `pr`), worktree, `HEAD` vs `revision`, dirty files, PR = merged | – | none |
| 2 | orchestrator-1 (owner kept) | R §10 Resume | `done`/`compaction`, `merge_commit` from PR; `active_agents` treated as not running | state commit | – | none |
| 3 | orchestrator-1 | O §2 Resume inspection | – | `cleanup_pr` absent → R §9.2–9.3; present → reuse | – | none |
| 4 | orchestrator-1 | R §9.2–9.4 | as S4 rows 6–8 | – | – | none |
| 5 | orchestrator-1 | O §7 | – | no implementer packet, no second worktree or PR | – | none |

**S6–S10 — Additional scenarios**

| Sc · # | Actor | Governing § | State change | Artifacts | Result | Gap |
|---|---|---|---|---|---|---|
| S6 · 1 | user → orchestrator-2 | R §4.1–4.3; O §2 | starts only after W-001's registration was confirmed | – | – | none |
| S6 · 2 | orchestrator-2 | R §7 write protocol | `W-002` `reserved`/`setup`; ID = highest + 1 under lock; own record only | state commit | – | none |
| S6 · 3 | orchestrator-2 | R §4 | needs W-001 code → `depends_on`, blocker `dependency`, waits for `merge_commit`; else worktree `feature/W-002` from main | – | – | none |
| S6 · 4 | all agents | R §4, §8; A | own worktree, ports, test data; shared files reconciled at integration; merges one at a time | – | – | none |
| S7 · 1 | user | R §3 On rejection | rejects option A | – | – | none |
| S7 · 2 | orchestrator-1 | R §3; O §3 | `B-1` stays; `next_action` updated; still `blocked` | state commit; no decision entry yet (written on approval) | – | ambiguous → Q1, Q2 |
| S7 · 3 | implementer or orchestrator | R §3 ("the agent that identifies the need") | – | revised option B under the same `W-001/D-1` | `NEEDS_DECISION` | ambiguous → Q2 |
| S7 · 4 | orchestrator-1, user | R §3 On approval; O §5 | as S2 rows 5–7; entry `approved`, option B | *Decision needed* (revised option, not a repeat) | – | none |
| S8 · 1 | orchestrator-1 | R §9 Cleanup failure; §10 | stays `done`/`compaction`; blocker `B-2` `check` or `environment`; `next_action` | state commit; full spec kept | – | none |
| S8 · 2 | orchestrator-1 | R §5, §9 | dependents unaffected; fix → cleanup merges → `complete` | follow-up cleanup commit | – | no rule (minor) → Q3 |
| S9 · 1 | reviewers | Rv §3 | `W-001/F-1` `still open` in two consecutive rounds | – | `CHANGES_REQUIRED` ×2 | none |
| S9 · 2 | orchestrator-1 | R §5 Loop stop; O §3, §5 | stop looping; analyze; options | *Blocked* | – | none |
| S9 · 3 | orchestrator-1 | R §7 Blocker | status `blocked`; blocker kind? decision entry? | state commit | – | ambiguous → Q6 |
| S9 · 4 | user | R §5 table; sk:work-spec §4 | option chosen; an AC change cites `changed: W-001/D-2` | – | – | none |
| S10 · 1 | cor-1 | – | session ends; no report | – | – | none |
| S10 · 2 | orchestrator-1 | O §6.2; R §6 gate | stays `specialized_review`; gate `INCOMPLETE`, never `PASS` | – | `INCOMPLETE` | none |
| S10 · 3 | cor-2 | O §3 `specialized_review` Failure | `active_agents` cor-2; rerun on same `revision` | state commit | `PASS` | none |

## 3. Audit tables

**B1 — Vocabulary**

| Term | Values | Defined | Used in | Mismatch |
|---|---|---|---|---|
| Task status | `planned` `reserved` `running` `blocked` `paused` `cancelled` `done` | R §7 | R §3–4, §9–10; O §2–3, §7; A | none |
| Stage | `setup` … `compaction` `complete` | R §7 | O §3; A; sk:work-spec | none |
| Result | `READY_FOR_REVIEW` `PASS` `CHANGES_REQUIRED` `NEEDS_DECISION` `INCOMPLETE` | P §3 | R §2, §6; O §3, §6; I; Rv §10; role files | "stage `INCOMPLETE`" (R §6, O §6) reuses a result value for the gate; same in design; accepted |
| Readiness | `READY`, `NOT READY — …` | sk:work-spec §5 | R §4; O §3 | none (skill verdict) |
| Finding class | `blocking` `suggestion` `unverified` | P §5 | R §6; O §6; I §6; Rv §7; role files | `Violates` without a rule — fixed F2 |
| Recheck status | `resolved` `still open` `regressed` | P §5 | Rv §3; ADV §4 | none |
| Check result | `pass` `fail` `not run` `limited`; `fail (pre-existing, baseline <ref>)` | P §4 | I §5; Rv §8, §10; COR §3; ADV §3 | none |
| Roles | orchestrator, implementer, 4 reviewers; `reviewer` = shared contract | agents/*.md; P §2 | A; R §6; O §2, §4 | none |
| Agent / owner IDs | `orchestrator-n`, `impl-n`, `sec-n`, `cor-n`, `spec-n`, `adv-n` | O §2 | R §7 example | none |
| Blocker kind | `decision` `dependency` `check` `environment` | R §7 | R §3–4, §9, §11; O §3 | loop stop has no kind → Q6 |
| Labels | **Observed** **Agreed** **Open** | sk:spec §3 | SPC §1; sk:work-spec §2, §4; spec template | none |
| Flow status | `draft` `ready for spec` `superseded` | sk:prototype §5 | prototype template; sk:spec §4 | none |
| Decision status | `approved` `superseded` | R §3 | sk:spec §5 | no pending/rejected → C1, Q1 |
| `W-NNN` | next free = highest + 1 | O §2 | all files | none |
| `W-001/D-n`, `W-001/F-n` | task-scoped | R §3, §6 | P §5–7; O; I; Rv; SPC; ADV; skills | none |
| `W-001/Q-n` | task question | sk:work-spec §3 | work-spec template | none |
| `B-n` | blocker ID | R §7 example only | O §5 `<blocker ID>` | format illustrative only (minor) |
| `SEC/COR/SPEC/ADV-n`, `IMPL-n` | provisional finding IDs | role files; I §6 | P §5; O §6 | `IMPL-n` missing in P §5 — fixed F1 |
| `D-n`, `AC-n` | provisional decision; AC | P §6; sk:work-spec §4 | Rv §7; I §4; SPC; P §9 | none |
| `<capability>/R-n`, `/Q-n`, `ui/R-n` | spec rule, open item | sk:spec §3 | A; Rv §7; SPC §1; sk:work-spec §3; templates | none |
| Flow ID | kebab-case, never reused | sk:prototype §5 | sk:spec §3; sk:work-spec §3 | none |
| User messages | *Registration confirmed*, *Decision request*, *Blocked*, *Task merged*, *Task complete* | O §5 | O §3 | blocks are unlabeled; mapping unambiguous (minor) |

**B2 — Commit message formats**

| Format | Defined | Actor · location | Issue |
|---|---|---|---|
| `prototype(<flow-id>): <summary>` | sk:prototype §6 | prototype · main or `prototype/<topic>` | none |
| `docs(<capability>): …`, `docs(ui): …` | sk:spec §7 | spec discovery · main or `spec/<topic>` | none |
| `spec(<ID>): draft\|refine task specification` | sk:work-spec §3 | orchestrator · feature worktree | none |
| `spec(<ID>): finalize task specification` | R §9 | orchestrator · feature worktree | may be empty → Q5 |
| `<type>(<ID>): <summary>` (e.g. `feat`) | I §5 | implementer · feature | type set undefined → Q3 |
| `wip(<ID>): <summary>` | I §4 | implementer · feature | none (checkpoint, not candidate) |
| `chore(<ID>): merge main into feature/<ID>` | I §7 | implementer · feature | none |
| `docs(<ID>): compact completed task` | R §9 | orchestrator · `cleanup/<ID>` | none |
| state.json commits | — | orchestrator · coordination checkout | undefined → Q3 |
| decision + spec-update commit | — (R §3, O §3 "commit both") | orchestrator · feature | undefined → Q3 |

Collisions: `docs(<ID>)` vs `docs(<capability>)` differ only by scope shape (`W-NNN` vs kebab-case); none in practice.

**B3 — Write ownership**

| File / area | Writers (file §) | Flag |
|---|---|---|
| `AGENTS.md` | CLI init (Phase 2); "Project notes" is project-owned | no framework writer (intended) |
| `docs/README.md` Mission, Intended users | — | no writer → Q9 |
| `docs/README.md` Current objective | prototype (sk:prototype §1–2) | none |
| `docs/README.md` Documents | spec (sk:spec §1, §4.8); orchestrator via spec in-task (O §1) | `architecture.md`, `stack.md`, `decisions.md` never indexed → Q9 |
| `docs/README.md` Commands | implementer (I §1, §5.5) | none |
| `docs/specs/*`, `docs/ui.md` | spec discovery; orchestrator via spec in-task (O §1, §3); implementer for approved decisions (I §1, §5.5) | overlap resolved by I §5.5 "unless the worktree already reflects them"; non-decision "Spec changes" have no writer → Q4 |
| `docs/work/*` | orchestrator via work-spec (sk:work-spec §1; O §1); final commit and removal (R §9) | none |
| `docs/decisions.md` | orchestrator, own entries (R §3; O §1) | superseding another task's entry was not allowed — fixed F5 |
| `docs/architecture.md`, `docs/stack.md` | implementer (I §1) | none |
| `prototype/**` | prototype (sk:prototype §1) | none |
| product code, tests | implementer (I §1) | none |
| `state.json` | orchestrators, own record (R §7; A); empty registry at bootstrap (R §11) | none |
| PR description, cleanup PR | orchestrator (P §8; O §1) | none |
| Orchestrator commits seen by implementer | I §6 | listed only `docs/work/`, `docs/decisions.md` — fixed F4 |

**B4 — Load chains**

| File | Loads | vs `AGENTS.md` | Rules needed without `rules.md` |
|---|---|---|---|
| orchestrator | A, R, P (+ SKILL.md when used) | matches | – |
| implementer | A, P, packet, work spec, refs, code | matches | authority I §3; approvals I §4 + A; worktree I §1; baseline I §5.1; commits I §5.8 |
| reviewer + 4 roles | A, P, Rv, role file, packet, diff | matches | authority Rv §6; read-only Rv §4; evidence Rv §8 |
| skills | none; A "Skills" loads SKILL.md | matches | self-contained; sk:work-spec cites R §1, §3, §9 (run by the orchestrator) |

P cites R §1 (P §7) and R §11 (P §4); both rules are restated in I §3, §6 and I §5.1 / Rv §8, so they stay reachable.

**B5 — References.** All `§n` and "section n" references resolve to the intended section: R 19, P 10, O 53 + 10
section/step refs, I 4 + 4, Rv 3, ADV 1, SPC 2, skills 16, A 1. Every `docs/framework/...` path maps to
`assets/framework/...`; `AGENTS.md`, `docs/README.md` map to `assets/templates/`; `state.json` is installed on
`workflow-state`. Other paths (`docs/specs/`, `docs/work/`, `docs/ui.md`, `docs/decisions.md`, `docs/architecture.md`,
`docs/stack.md`, `prototype/**`, `go.mod`) are project documents named by the framework. Design typo
`refs/heads/workflow-state.json` (design line 335) is not propagated.

**B6 — Single primary location**

| Rule | Primary | Restated in | Assessment |
|---|---|---|---|
| Authority order | R §1 | A, I §3, Rv §6 | full restatement required by the load chain; keep in sync |
| On-approval procedure | R §3 | O §3 preamble (adds "commit both", "restore the prior status") | operational detail; acceptable |
| Approval triggers | I §4 | A, R §3, Rv §7 | short phrase + reference |
| Result precedence | P §3 | I §5, Rv §10 | short phrase + reference |
| Missing reviewer ≠ `PASS` | R §6 | P §3, O §6–7 | short phrase |
| Pre-existing failures | P §4 | I §5, Rv §8 | short phrase |

**B7 — Design coverage**

| Design section | Status | Covered by |
|---|---|---|
| Intro, Delivery phases | covered / Phase 2 | R header; CLI deferred |
| Language policy | covered | A; P §1; O §5; sk:prototype §3; templates (dashboard labels: Phase 2) |
| Agreed decisions; Agent roles | covered | skills; agents/*.md; R §2–5; O §4 |
| Project and framework files | covered, except `decisions.md` "pending choices" | A routes; templates; C1 → Q1; runtime adapter: Phase 2 |
| Context loading; Example AGENTS.md | covered | A; I §2; Rv §2 |
| work-spec skill and template; Prototype handoff and readiness | covered | sk:work-spec; sk:prototype §7; sk:spec §4 |
| Review specializations; Final adversarial review | covered | R §5–6; SEC; COR; SPC; ADV |
| Priorities and arbitration | covered | R §1, §6; Rv §6 |
| Execution loop, approval loop, return points | covered | R §3, §5; O §3 (rejection actor → Q2) |
| Conditions for PR and merge | covered | R §8 |
| Parallel execution, starting a task, isolation, shared state | covered | R §4, §7 |
| Active state record; State semantics | covered | R §7 (`"pr": null` in design vs "omit empty" in R; R wins) |
| Compaction, final spec commit, compact record | covered | R §7, §9 (Q5; `pr` number vs URL: Phase 2 schema) |
| Agent reports and verification | covered | P |
| Bootstrap and recovery | partial | R §10–11; Pause fixed F3; Cancel and Ownership handoff details missing → Q7 |
| Live HTML dashboard | deferred Phase 2 | needs decision recommendation/impact in the entry → Q1 |
| Phase 2: Go initialization contract | deferred Phase 2 | section 7 |
| Phase 1 validation; Proposed name | this report; out of scope | – |

**B8 — Mechanical checks.** Throwaway module outside the repo: `<scratchpad>/check/main.go` (Go 1.27.2), run as
`cd <scratchpad>/check && go run . /Users/madalinpopa/Projects/solomonar/assets`. It checks the version line, YAML
frontmatter keys (exactly `name`, `description`; `name` = file or skill name), `json.Unmarshal` of `state.json`, and
`text/template` parse + execute (`missingkey=error`, `Objective` set and empty). Result: `failures: 3`.

| File | Lines (after) | Version line | Frontmatter |
|---|---|---|---|
| rules.md / report.md | 301 (300) / 197 | OK / OK | none (correct) |
| agents: orchestrator / implementer / reviewer | 206 (202) / 127 (126) / 104 | OK | OK |
| agents: security / correctness / spec / adversarial | 55 / 61 / 62 / 80 | OK | OK |
| skills SKILL.md: prototype / spec / work-spec | 98 / 106 / 108 | OK | OK |
| skills template.md: prototype / spec / work-spec | 34 / 35 / 40 | **FAIL** (absent) → Q8 | none (correct) |
| templates: AGENTS.md / docs/README.md / state.json | 62 / 33 / 4 | renders `Framework version: 0.1.0 —` / n/a / n/a | parse + execute OK / OK / valid JSON (`schema_version`, `tasks`) |

## 4. Known issues (Part C)

| # | Claim | Verdict | Evidence | Action |
|---|---|---|---|---|
| 1 | R §3 decision status lacks pending/rejected | Confirmed vs design; internally consistent | Design: `decisions.md` holds "pending choices"; record "selected option when approved"; dashboard reads recommendation/impact from the entry. R §3 writes the entry only on approval | Not fixed (new status value) → Q1 |
| 2 | R §10 Pause omits settling active work, revision, outstanding work | Confirmed | Design Pause row vs R §10 | Fixed F3 |
| 3 | P §5 prefixes omit `IMPL-n` | Confirmed | I §6 cites P §5 for `IMPL-n` | Fixed F1 |
| 4 | P §5 `Violates` omits `demonstrated defect` | Confirmed (contradiction) | P §5 required a rule/AC; Rv §7, SEC §4 allow the defect form | Fixed F2 |
| 5 | O §1 allowed edits omit canonical specs | Refuted | O §1 already lists `docs/specs/*.md`, `docs/ui.md`, the index (via spec in-task) | none |

## 5. Fixes applied

| ID | File · section | Before → after | Reason |
|---|---|---|---|
| F1 | report.md §5 IDs | "a reviewer assigns … `SEC-1`, `COR-1`, `SPEC-1`, `ADV-1`" → "the reporting role assigns … `ADV-1`, `IMPL-1`" | C3; I §6 uses `IMPL-n` |
| F2 | report.md §5 table + entry format | "must name the violated rule/AC" → "… (or the demonstrated defect)"; `Violates: <rule or AC ID>` → "…; when none exists, `demonstrated defect: <invariant broken>`" | C4; contradiction with Rv §7, SEC §4 |
| F3 | rules.md §10 Pause | "Commit status `paused` and a `next_action`; stop delegating" → "Stop delegating; settle or stop active work; commit status `paused` with the latest committed `revision` and a `next_action` naming outstanding work" | C2; design Pause row |
| F4 | implementer.md §6 | orchestrator commits touch "only `docs/work/` or `docs/decisions.md`" → also `docs/specs/`, `docs/ui.md`, the `docs/README.md` index | O §1, §3 commit spec updates; a decision during a correction round made the implementer report `INCOMPLETE` |
| F5 | orchestrator.md §1 | "your task's entries in `docs/decisions.md`" → "… (plus the `superseded` status of an entry your decision replaces)" | R §3 requires setting the replaced entry `superseded` |
| F6 | orchestrator.md §6 item 1 | "`Revision` equals the delegated revision" → "… (implementer: the handoff commit, or the start commit when no candidate exists; `report.md` §2)" | O §3 validates implementer headers with item 1; P §2 defines a different `Revision` for the implementer |
| F7 | rules.md §6 gate; orchestrator.md §7 | "valid `PASS` on the current revision" → "…, unless its earlier verdict is reused under Candidate stability"; "a verdict on another revision" → "… without a recorded reuse reason" | O §3 `integration` and R §6 Candidate stability reuse verdicts after the final spec commit; the gate forbade it |
| F8 | orchestrator.md §3 `specification` | "route product choices and requirement conflicts through §3" → "route `needs decision: yes` questions and requirement conflicts through §3" | sk:work-spec §3.6–3.7 asks product questions directly and routes only decision-needing questions and conflicts |

## 6. Questions for the author

| # | Question | Options | Recommendation |
|---|---|---|---|
| Q1 | Should `docs/decisions.md` hold pending (and rejected) choices, as the design says? | a) keep approval-only entries; b) add `pending`, written at R §3 step 2; c) b + `rejected` | b: `pending`, `approved`, `superseded`; a rejection keeps `pending` and adds the rejected option to the entry; enables the dashboard view |
| Q2 | Who prepares the revised option after a rejection? | a) orchestrator re-delegates the implementer; b) orchestrator chooses from listed alternatives | a when investigation is needed, else b; say it in O §3 |
| Q3 | Commit formats for state commits, the decision + spec-update commit, follow-up cleanup commits, and the implementer's `<type>` set | a) `chore(<ID>): <state change>` and `docs(<ID>): record W-001/D-n`; b) dedicated types `state(<ID>)`, `decision(<ID>)` | a, plus Conventional Commits types for the implementer; define them in R §7, R §3, I §5 |
| Q4 | Who applies work-spec "Spec changes" that cite a user confirmation, not a decision? | a) spec in-task mode also accepts the work spec's "Spec changes" line (source `user YYYY-MM-DD`); b) implementer applies them | a: the spec skill owns IDs and labels; update sk:spec §2, §5 and O §1 |
| Q5 | Final spec commit when the work spec is unchanged since its last `spec(<ID>)` commit | a) that commit is the final one; b) empty `finalize` commit; c) always re-commit | a: `spec_ref` points to it; keep `finalize` only when content changed |
| Q6 | How is a loop stop recorded? | a) blocker `decision` + `W-001/D-n` (full R §3); b) blocker `check`; c) new kind | a: the user selects an option; it stays durable and resumable |
| Q7 | Cancel and Ownership handoff omit design details ("stop execution", reason and remaining work; "stop or confirm the previous execution stopped") | a) add to R §10 rows like F3; b) leave | a |
| Q8 | Skill `template.md` files have no version line; adding one would copy into project docs | a) exempt templates; b) version inside the first hint comment (deleted on fill) | b: the CLI can still detect managed-asset versions |
| Q9 | Mission, Intended users, and index entries for `architecture.md`, `stack.md`, `decisions.md` have no writer | a) user-owned, state it in the template; b) prototype fills Mission at Start; implementer/orchestrator add index lines | a for Mission/Users; b for index lines |

## 7. Phase 2 inputs

- Managed (verbatim, version-checked): `assets/framework/**` → `docs/framework/**` (rules, report, 7 agent files,
  3 skills with templates). Version line `Framework version: 0.1.0` after the title or frontmatter (templates: Q8).
- Rendered once with `text/template` (`missingkey=error`): `AGENTS.md` (`.FrameworkVersion`; "Project notes" section is
  project-owned and preserved on upgrade) and `docs/README.md` (`.ProjectName`, `.Objective` optional — empty renders
  "Not set…"; `.FrameworkVersion` is not used there). Both are project-owned after init.
- Project-owned, never written by init: `docs/specs/`, `docs/work/`, `docs/ui.md`, `docs/decisions.md`,
  `docs/architecture.md`, `docs/stack.md`, `prototype/`.
- Registry bootstrap (R §11): local Git baseline; branch `workflow-state` with `state.json` =
  `{"schema_version": 1, "tasks": {}}` (plain JSON, never merged into main); persistent coordination checkout
  (e.g. `<repo>-state/`). Records live under `tasks.<ID>`; R §7 shows only the record shape.
- Runtime adapters: agent and skill frontmatter has only `name` and `description`; `reviewer` is a shared contract,
  not an invocable role. Registration must load A plus each file's "Load together with" chain.
- Repeated init: identical assets are a no-op; a changed managed asset is a conflict, not an overwrite; upgrades are
  a separate operation keyed on the version line.
