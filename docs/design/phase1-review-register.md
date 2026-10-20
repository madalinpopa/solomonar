# Phase 1 review register — framework v0.1.0

Consolidates every gap, conflict, and question raised by the agents that wrote the framework files (prompts 3–10)
and by the Phase 1 validation (`docs/design/phase1-validation.md` §6). Each item has a single resolution.
Section B and D items are resolved by this register. Section C records the author's decisions; C1 and C2 apply the recommendation unless the author objects;
C7–C11 are open.

Sources: `orch` = orchestrator.md agent, `impl` = implementer.md agent, `rev` = reviewer files agent,
`adv` = adversarial-reviewer.md agent, `proto` / `spec` / `ws` = skill agents, `scaf` = project scaffold agent,
`val Qn` = validation report question n. File abbreviations follow the validation report.

## A. Already resolved — no action

| ID | Source | Issue | Resolved by |
|---|---|---|---|
| A1 | spec #1, impl #5 | O §3 said "update affected specs via `work-spec`" | O §3 now says `spec` skill (in-task mode) |
| A2 | impl #4 | P §5 had no `IMPL-n` prefix | validation F1 |
| A3 | rev #1 | P §5 `Violates` rejected a demonstrated defect without a written rule | validation F2 |
| A4 | val C2 | R §10 Pause omitted settling work and the revision | validation F3 |
| A5 | impl #3 | `Revision` of an implementer report without a candidate | P §2 and validation F6 |
| A6 | ws #2 | O §3 routed every product question through R §3 | validation F8 |
| A7 | rev #4 | `AGENTS.md` did not route reviewers to `reviewer.md` | A "every file its first lines require" + role first lines |
| A8 | proto #2–3 | No "Current objective" line for `prototype` to fill | `assets/templates/docs/README.md` |
| A9 | val C5 | O §1 allowed edits omitted canonical specs | refuted: O §1 lists them |

## B. Fixes to apply — no design choice involved

| ID | Source | File · section | Change |
|---|---|---|---|
| B1 | impl #1 | O §4 correction packet | Add `Work spec / References / Baseline / Constraints: as in the implementer packet`. |
| B2 | rev #2, adv #1 | O §4 specialist packet | Add `Baseline: <SHA>; known failures` and `Constraints: ports; test data`. A required scenario without assigned ports is a missing packet field (Rv §2 → `INCOMPLETE`). |
| B3 | rev #3 | O §4 specialist packet | On a recheck, `Diff` base = last reviewed revision; the recheck list carries each `W-001/F-n` entry in `report.md` §7 form, not only its ID. |
| B4 | adv #2 | O §4 adversarial packet | Summaries use mapped `W-001/F-n` IDs. |
| B5 | impl #2 | O §4 | One line: integration work uses the implementer packet with objective "merge main into `feature/<ID>`" or the integration fix. |
| B6 | rev #1, adv Q1 | P §5 | Define recheck statuses: `still open` = not fixed on this revision; `regressed` = resolved on an earlier revision and failing again. A new defect introduced by a fix gets a new ID. |
| B7 | rev #5 | P §2 | `<Role>` excludes the shared reviewer contract (wording depends on C5). |
| B8 | adv #3 | Rv §6 | Add: specialty labels and reviewer vote counts never override evidence. |
| B9 | spec #2–3 | Rv §7; SPC §1, §4 | `Violates` examples include `<capability>/R-n`; spec-reviewer cites rule IDs. |
| B10 | spec #3, #5; proto #1 | R §1; Rv §6 | Only **Agreed** spec rules are approved requirements. **Observed** rules, prototype libraries, and simulations are evidence and approve nothing. |
| B11 | scaf | R §7 | One line: registry = `{"schema_version": 1, "tasks": {"<ID>": <record>}}`. |
| B12 | orch #4, #6, Q4 | R §9 step 3 | Cleanup PR: branch `cleanup/<ID>` from main; docs-only; no specialist or adversarial review; required repository checks and rules apply. |
| B13 | impl Q2 | I §3 | An AC that cannot be met: needs a choice (architecture, dependency, AC change) → `NEEDS_DECISION`; blocked by environment or access → `INCOMPLETE`. |
| B14 | val Q2 | R §3 On rejection; O §3 | The orchestrator re-delegates the identifying role when the revised option needs investigation; otherwise it presents a listed alternative. |
| B15 | val Q5, orch #2, Q2 | R §9 Before merge; O §3 `integration` | If the work spec is unchanged since its last `spec(<ID>)` commit inside the reviewed revision, that commit is the final spec commit; no new revision. Create `spec(<ID>): finalize task specification` only when content changed, and treat it as a changed candidate (R §6 Candidate stability). |
| B16 | val Q6 | R §5 Loop stop | Record a loop stop as a `decision` blocker with a `W-001/D-n` entry; the user's choice follows R §3. |
| B17 | val Q7 | R §10 | Cancel: stop execution; record reason and remaining work. Ownership handoff: stop or confirm the previous execution stopped before the owner change. |
| B18 | val Q8 | 3 × skill `template.md` | Version line inside the first hint comment: `<!-- Framework version: 0.1.0 … -->` (removed when filled). |
| B19 | val Q9, scaf Q3 | README template; sk:prototype §2 | Mission and Intended users are user-owned; `prototype` may propose them at Start and records them only after the user confirms. The role that creates `architecture.md`, `stack.md`, or `decisions.md` adds its index line. |
| B20 | scaf Q1 | A Mandatory rules | Numbered list, cited as `AGENTS.md rule n` (Rv §7 example). |
| B21 | scaf Q4 | A Project notes hint | Name the extension points: adopted style guides (e.g. Go Code Review Comments), branch rules for direct commits, test policy. |
| B22 | proto Q2–4 | sk:prototype §5–6 | README-only or multi-flow commits: `prototype: <summary>`. Files of a `superseded` flow may be deleted; its README row stays. `draft` flows may live on the main branch. |
| B23 | spec Q2–4 | sk:spec §3–4 | Brownfield: existing code at `<sha>:<path>` is a valid **Observed** source. Discovery confirmations live in the rule's source, not in `docs/decisions.md`. After a split, active tasks update their work-spec references when they next merge main (commit body maps old → new). |
| B24 | ws Q2–3 | sk:work-spec §3, §5 | Starting a `planned` record authorizes its recorded outcome. Answers to `needs decision: no` questions live in the work spec only. |

## C. Author decisions

| ID | Source | Question | Options | Recommendation | Decision |
|---|---|---|---|---|---|
| C1 | val Q1, val C1 | Does `docs/decisions.md` hold pending choices? | a) approval-only entries; b) `pending` → `approved` / `superseded`, rejected options listed in the entry; c) b + `rejected` status | b | proposed: b (apply unless the author objects) |
| C2 | val Q4, ws #1, spec #4 | Who applies work-spec "Spec changes" confirmed by the user rather than by a decision? | a) `spec` in-task mode also accepts the work spec's "Spec changes" (source `user YYYY-MM-DD`), run by the orchestrator before integration; b) implementer edits specs directly | a | proposed: a (apply unless the author objects) |
| C3 | val Q3, scaf #1 | Commit formats not yet defined | a) state `state(<ID>): <status>/<stage> — <summary>`; decision `docs(<ID>): record W-001/D-n`; implementer types = Conventional Commits (`feat fix refactor test docs build chore perf`) + `wip`; b) `chore(<ID>)` for state commits | a | **a** — author, 2026-10-20 |
| C4 | rev Q1 | Must every AC have an automated test? | a) every AC needs a named check; automated test required unless the work spec says why it is impractical; b) any named check suffices; c) automated test always | a | **a** — author, 2026-10-20 |
| C5 | rev Q2 | `reviewer.md` sits in `agents/` with frontmatter, so runtimes may register it as an invocable agent | a) move to `docs/framework/reviewer.md` without frontmatter, beside `report.md`; b) keep it, adapters skip it | a | **a** — author, 2026-10-20 |
| C6 | proto Q1 | Support Jujutsu (jj) repositories? | a) v0.1 is Git-only (worktrees, `refs/heads/workflow-state`); jj later; b) support colocated jj now | a | **a** — author, 2026-10-20 |
| C7 | E trace S4, S8 · 2 | Commit formats for the orchestrator's `spec` in-task commit of user-confirmed "Spec changes" (O §3 `integration`) and for follow-up cleanup commits | a) `docs(<ID>): apply spec changes` and `docs(<ID>): fix compaction` (R §9); b) reuse `docs(<ID>): compact completed task` for every cleanup commit and `spec(<ID>): …` for spec changes; c) leave to AGENTS.md rule 12 | a: same `docs(<ID>)` shape as C3; `spec(<ID>)` stays reserved for the work spec | **a** — author, 2026-10-20 |
| C8 | E trace S2 | R §3 step 2 commits the `pending` entry in the task worktree, which breaks R §6 Candidate stability while other reviewers still run on the same revision | a) write the `pending` entry only after every running review on that revision has reported; the state blocker is committed at once; b) record `pending` entries outside the worktree (coordination checkout); c) accept the commit and rerun the affected reviews | a: state stays immediate and durable; the worktree changes only between review sets | **a** — author, 2026-10-20 |
| C9 | E trace S4 | After `spec` in-task mode assigns final IDs to `<capability>/new` entries, does the work spec cite them? | a) yes: the orchestrator updates the "Spec changes" line via `work-spec` before the final spec commit (content changed → `finalize`, B15); b) no: the `spec` commit body maps `new → ID`; the work spec keeps `new` | a: `spec_ref` then names real rule IDs, which the compact history needs | **a** — author, 2026-10-20 |
| C10 | E trace S4 | "Spec changes" applied at `integration` change `docs/specs/` after the spec-reviewer ran | a) R §6 Candidate stability names `spec-reviewer` as affected by any `docs/specs/` or `docs/ui.md` change, so it rechecks that diff; b) apply user-confirmed "Spec changes" before `specialized_review` instead of at `integration` (revises C2); c) keep the assessment case by case | b: the change is reviewed with the code once, and no extra recheck round follows integration | **d** — author, 2026-10-20: apply user-confirmed "Spec changes" when the readiness gate passes, before implementation (revises C2 timing); an ID collision after merging main is renumbered as a changed candidate |
| C11 | E trace S3 | SPC §2 step 4 maps a missing port to `limited`; O §4 (B2) calls it a missing packet field | a) SPC §2 step 4: not runnable → `limited`; ports missing for a required scenario → `INCOMPLETE` (missing packet field, Rv §2); b) leave as is, since both end in `INCOMPLETE` for required scenarios | a: one term for one cause; no change in outcome | **a** — author, 2026-10-20 |

## D. Answers without a file change

| ID | Source | Question | Answer |
|---|---|---|---|
| D1 | orch Q1 | Where is the triage kept durably? | In the session, then in the PR Verification section. On resume the orchestrator re-runs triage; it is cheap and derived from the diff. No state field. |
| D2 | orch Q3 | Owner identity across sessions | The user naming the task in a resume request establishes continuation (O §2). No heartbeat; `session_ref` is optional when the runtime offers one. |
| D3 | orch Q5 | Move ID allocation and the cleanup branch name into `rules.md`? | ID allocation stays in O §2: orchestrators always load both files. The cleanup branch moves to R §9 with B12. |
| D4 | impl Q1 | `wip(<ID>)` commits remain in main's history | Accepted: merge commits preserve history by design; rewriting reviewed history stays forbidden. |
| D5 | adv Q2 | No ports assigned for an HTTP scenario | Required scenario → `INCOMPLETE` (missing packet field); optional scenario → `not run` with the reason. B2 makes ports a packet field. |
| D6 | all | `docs/design/framework-v0.1.md` missing | Author action: copy `~/Downloads/AI-Assisted Software Development Framework.md` to that path. |

## E. Applied changes

Applied 2026-10-20 to the working copy (not committed). File abbreviations as above; Rv = `reviewer.md`, now at
`assets/framework/reviewer.md` (installed `docs/framework/reviewer.md`).

| ID | Files · sections changed |
|---|---|
| B1 | O §4 correction packet: `Work spec / References / Baseline / Constraints` line |
| B2 | O §4 specialist packet: `Baseline` and `Constraints` line; O §4 bullet: required scenario without ports = missing packet field |
| B3 | O §4 specialist packet: recheck `Diff` base = last reviewed revision; recheck list in P §7 entry form |
| B4 | O §4 adversarial packet: summaries use mapped `W-001/F-n` IDs |
| B5 | O §4 bullet: integration work uses the implementer packet |
| B6 | P §5: `still open` and `regressed` defined; a new defect from a fix gets a new provisional ID |
| B7 | P §2 `<Role>`: C5 wording |
| B8 | Rv §6: specialty labels and vote counts never override evidence |
| B9 | SPC §4: breaking an **Agreed** rule → `blocking`, `Violates` the rule ID. Rv §7 already listed `<capability>/R-n`; unchanged |
| B10 | R §1: only **Agreed** rules are approved requirements; Rv §6: same as a short phrase in the authority bullet |
| B11 | R §7 Location and writes: registry shape |
| B12 | R §9 step 3: `cleanup/<ID>` branch, docs only, no specialist or adversarial review, required checks apply; O §3 `compaction`: reference to §9 step 3 (`e.g.` branch removed) |
| B13 | I §3: AC that cannot be met → `NEEDS_DECISION` or `INCOMPLETE` by cause |
| B14 | R §3 On rejection: re-delegate the identifying role or present a listed alternative; O §3 preamble: short reference |
| B15 | R §9 Before merge: last unchanged `spec(<ID>)` commit is final; `finalize` only on change, as a changed candidate; O §3 `integration` Do |
| B16 | R §5 Loop stop: `decision` blocker with a `W-001/D-n` entry, then §3 |
| B17 | R §10 Cancel (stop execution; reason and remaining work in `next_action`) and Ownership handoff (stop or confirm stopped) |
| B18 | 3 × skill `template.md`: `Framework version: 0.1.0` inside the first hint comment |
| B19 | README template: Mission, Intended users marked user-owned; Documents hint names who indexes `architecture.md`, `stack.md`, `decisions.md`; sk:prototype §1, §2: propose and record after confirmation; I §1 and O §1: index line of a document the role creates |
| B20 | A Mandatory rules: numbered, `Cite a rule as AGENTS.md rule n`; Rv §7 example `AGENTS.md rule n` |
| B21 | A Project notes hint: style guides, branch rules for direct commits, test policy |
| B22 | sk:prototype §5: `draft` flows on main; `superseded` flow files deletable, README row stays; §6: `prototype: <summary>` |
| B23 | sk:spec §3: brownfield code as **Observed** source; split → active tasks update references on their next merge of main; §4 step 7: confirmations live in the rule's source |
| B24 | sk:work-spec §3 step 7: `needs decision: no` answers live in the work spec only; §5 Authorized: starting a `planned` record authorizes its outcome |
| C1 | R §3 step 2 (`pending` entry), On approval (`approved` + selected option), On rejection (stays `pending`, rejected option and reason), entry fields; sk:spec §5 step 1 (`approved` only); O §1, §3 preamble; A Context routes (`approved` entries). P names no decision status; unchanged |
| C2 | sk:spec §2 (mode table, bullets), §5 (user-confirmed "Spec changes", final IDs for `<capability>/new`); O §1, §3 `integration`; sk:work-spec §4 (sources on each entry; only the orchestrator applies them); I §1 and §5 step 5 (spec files removed from implementer edits); work-spec template (sources in the "Spec changes" hint and example) |
| C3 | R §7 (`state(<ID>): <status>/<stage> — <summary>`); R §3 (`docs(<ID>): record W-001/D-n`); I §5 step 8 (type set + `wip`); O §3 preamble reference |
| C4 | sk:work-spec §4 (named check; automated test unless impractical, then manual scenario + limit), §5 Verifiable reference; I §5 step 4 reference; SPC §4 (missing check or required automated test → `blocking`) |
| C5 | `assets/framework/agents/reviewer.md` → `assets/framework/reviewer.md`, frontmatter removed; references in SEC, GO, SPC, ADV updated. O, P, and A held no `agents/reviewer.md` reference |
| C6 | R §11: Git and Git worktrees required; Jujutsu and other VCS front ends out of scope |
| D6 | `docs/design/framework-v0.1.md` copied unchanged from `~/Downloads/AI-Assisted Software Development Framework.md` |

Skipped: none.

Remaining gaps found by the spot-traces, each an open author decision in section C:
- Commit format for the orchestrator's `spec` in-task commit of user-confirmed "Spec changes" at `integration`
  (C2) is undefined; C3 covers only decision commits. Follow-up cleanup commits (S8 · 2) also have no format → C7.
- R §3 step 2 commits the `pending` entry in the task worktree; when `NEEDS_DECISION` comes from a reviewer while
  other reviewers still run on the same revision, that commit conflicts with R §6 Candidate stability → C8.
- After `spec` in-task mode assigns final IDs to `<capability>/new` entries, nothing says whether the work spec is
  updated to cite them; if it is not, the final spec (`spec_ref`) keeps `<capability>/new` → C9.
- "Spec changes" applied at `integration` change `docs/specs/` after the spec-reviewer ran; re-review depends only
  on the R §6 Candidate stability assessment → C10.
- SPC §2 step 4 maps a missing port to `limited`, while O §4 (B2) calls it a missing packet field; both end in
  `INCOMPLETE` for required scenarios, but the wording differs → C11.

## F. Applied after the author's decisions on C7–C11

Applied 2026-10-20 in the working copy.

| ID | Files · sections changed |
|---|---|
| C7 | R §4 step 6 (`docs(<ID>): apply spec changes`); R §9 Cleanup failure (`docs(<ID>): fix compaction`) |
| C8 | R §3 step 2: the `pending` entry waits until every running review on the current revision has reported |
| C9, C10 | R §4 step 6: user-confirmed "Spec changes" applied when the readiness gate passes, assigned IDs cited in the work spec; R §8: rule ID collision after merging main → renumber, changed candidate; O §3 `specification` Exit and `integration` Do; sk:work-spec §4 |
| C11 | SPC §2 step 4: not runnable → `limited`; no ports for a required scenario → `INCOMPLETE` |
