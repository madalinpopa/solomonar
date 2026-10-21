<h1 align="center">
  <img src="docs/logo/lockup.svg" alt="solomonar" width="560">
</h1>

solomonar: in Romanian folklore, a wizard who learns every secret from a single book and rides a dragon to command the storms. solomonar gives your project its book (specs), its spells (skills) and its dragons (agents).

**Framework v0.1.0 · Phase 1 complete**

The framework contracts are defined; the CLI is not implemented yet. `main.go` currently prints `Hello World`.
The commands below describe the [Phase 2 interface](docs/cli/interface.md), not a working release.
The [workflow audit and applied changes](docs/reviews/2026-10-21-framework-cli-audit.html) track the design fixes.

## How Solomonar works

From intent to verified code. The prototype discovers the product, the specifications pin down its behavior, and every task runs through an orchestrator that delegates implementation and reviews, writes and checks state only through the `solomonar` CLI, and merges into `main` only what has passed every check.

### A light core, extended by the CLI

Solomonar is a way of working, not a technology stack. The core holds only the minimal skills and agents that enforce the process, and none of them names a language. `solomonar init` and `solomonar extension add` install everything else, following the project's approved stack decisions.

| Part | Contains |
| - | - |
| Core | Skills `solomonar` (your entry point), `prototype`, `spec`, `work-spec`. Agents `orchestrator`, `implementer`, and four language-agnostic reviewers: `security-reviewer`, `correctness-reviewer`, `spec-reviewer`, `adversarial-reviewer`. Always installed. |
| Extensions | Skills with the rules of a language, library, or practice (for example `go`, `htmx`, `templ`, `tdd`), and agents that add a specialist reviewer. Added by the CLI; the process never depends on one. |

### Start every request with the `solomonar` skill

One small skill, invoked only by the user, routes a request to the skill or role that owns it, in the same session. It writes nothing itself. Sessions started without it follow the same routes from `AGENTS.md`.

| Runtime | In the agent session |
| - | - |
| Claude Code | `/solomonar work on W-002` |
| Codex CLI/IDE | `$solomonar work on W-002` (or select it with `/skills`) |
| Portable prompt | `Use the solomonar skill to work on W-002.` |

| Request | Route |
| - | - |
| None | Overview: what waits on you, what is blocked, what can start next. |
| Start a product; explore or change a flow | `prototype` |
| Flows ready for spec, or product behavior you decided outside a task | `spec` |
| A new, planned, or resumed task, e.g. "work on W-002" | This session takes the `orchestrator` role. |
| Adopt or drop a technology | You run `solomonar extension add\|remove`; your command records the decision. |

**The session takes the role; it is never delegated.** The orchestrator delegates the implementer and every reviewer, and a delegated subagent cannot delegate further, so the orchestrator must be the top-level session, one task per session. An answer to a pending decision goes back to the session that asked for it.

### Start a project (Phase 2 CLI contract)

Once the CLI is released, install a published version with `go install github.com/madalinpopa/solomonar@vX.Y.Z`
or use its release binary. Prerequisites: Git 2.40 or later, a selected runtime, and a first commit before task
worktrees. The complete integration path also needs a remote, PR access, and merge commits. Discovery can stay
local while those are being prepared; a managed task cannot finish without them. Increment 1 is an internal
milestone: choosing the stack after discovery needs `extension add` from increment 2.

For a new directory, start without a stack and commit the generated framework files explicitly:

```sh
solomonar init ./ledger --git-init --runtime codex --ext none --yes
cd ledger
git add AGENTS.md .agents .solomonar docs
git commit -m "chore: initialize solomonar 0.1.0"
```

Then open a top-level Codex session in `ledger` and use `$solomonar` with the next useful outcome. In Claude
Code use `/solomonar`; `Use the solomonar skill` works as a portable prompt. For an existing repository, inspect
the init plan and its file conflicts before committing; the implementer discovers the real run/check commands.
Unknown commands do not mean the application must be scaffolded again.

The prototype is useful when behavior needs discovery. A known brief can go directly to a task, for example
`Use the solomonar skill. Add CSV export to the existing CLI; no UI prototype is needed.` Behavior already
decided outside a task goes to `spec`. A prompt should name one outcome and the failure or boundary states
needed to judge it; existing approvals do not need another confirmation.

## 1 / 5 · The full cycle, from prototype to closure

![The Solomonar cycle: prototype and spec discover the product, solomonar task register and work-spec prepare the task, then implementation, tiered reviews, PR with merge, and solomonar task done; CHANGES_REQUIRED sends the task back to implementation.](docs/images/workflow-cycle.svg)

**The colored boxes are the three skills** (`prototype`, `spec`, `work-spec`); the rest are orchestrator steps. Discovery works on a static prototype: `spec` marks every rule *Observed* or *Agreed* and keeps *Open* questions, which go back into the prototype. A task gets a worktree only after `solomonar task register` succeeds. When the work spec is ready, its user-confirmed spec changes go into the canonical specs before any code is written. Any defect found in review sends the task back to implementation.

## 2 / 5 · The task loop: everything goes through the orchestrator

![The orchestrator is the hub: it delegates the work spec to the implementer, sends the same revision to the required language-agnostic specialists (security, correctness, spec, plus installed extension reviewers) in parallel, then to the adversarial reviewer for tier 2 tasks, asks the user to approve decisions, writes state through solomonar task, reads it back with task status against Git, and opens the PR.](docs/images/workflow-task-loop.svg)

**Agents talk only to the orchestrator.** The user starts each orchestrator as a top-level session with the `solomonar` skill, which takes the role in that same session. The implementer writes its approach first and raises every architecture or dependency decision in one `NEEDS_DECISION`, before touching tracked files. Specialists review the same revision in parallel; the triage records a risk tier in state and marks a specialty that does not apply as "not required", never as a fabricated PASS. The adversarial review runs only on tier 2 tasks, after the specialist blockers are closed, on a different model when the runtime allows. No response is not approval. After two rounds with no progress on the same blocker, the orchestrator stops the loop and takes the problem to the user. **The orchestrator decides; the CLI checks.** Each `solomonar task` write is checked against Git: the revision exists on the feature branch, the spec exists, and the stage respects the recorded tier. `task status` reports what Git shows and where the record has drifted. The CLI never launches agents or stores verdicts.

### Core reviewers

| Priority | Reviewer | Checks |
| - | - | - |
| 1 | Security | Bugs and security defects: crashes, error handling, leaks, races, trust boundaries, input, secrets. |
| 2 | Correctness | Business logic: agreed business rules, domain invariants, calculations, state transitions. |
| 3 | Spec | The task matches its work spec (acceptance criteria, scope) and the rules of the installed extensions that apply, such as the language's rules. |
| - | Adversarial | Tier 2 only: tries to break the integrated feature with an input, a sequence, or a failure. |

After every code change, Security, Correctness, and Spec all review the same revision in parallel. Their priority is the order of the table: the orchestrator orders the corrections by it, and when two corrections conflict, the higher-priority one wins. Language rules are not in any reviewer: choosing Go installs the `go` extension, a separate rules file (`go/C-n`) that the Spec reviewer checks. Installed extension agents join the specialists when their "applies when" matches the change.

### Report results

| Result | Meaning |
| - | - |
| `READY_FOR_REVIEW` | The implementer hands off a revision. It approves nothing. |
| `PASS` | The reviewer covered its scope with no open blockers and no missing checks. |
| `CHANGES_REQUIRED` | Confirmed defects; back to the implementer through the orchestrator. |
| `NEEDS_DECISION` | Dependent work waits for a user decision. |
| `INCOMPLETE` | Evidence or the review run is missing. It does not count as a PASS. |

### Review tiers

| Tier | Meaning |
| - | - |
| Tier 0 | Documentation only, no behavior or requirement change. CI only. |
| Tier 1 | Normal code. The required specialists. |
| Tier 2 | Persistence, authentication, concurrency, external integrations, data migration, or an approved architecture or dependency decision. Specialists, then adversarial. |

### Who wins a conflict

1. The user's explicit decision, recorded with its scope.
2. Approved global specs and mandatory rules. Only *Agreed* spec rules count.
3. The task's approved work spec.
4. Agent recommendations, only when compatible and backed by a concrete benefit.

Code, prototype, and *Observed* rules are evidence of existing behavior. They do not change approved requirements.

## 3 / 5 · Sequential starts, parallel execution

![Timeline: orch-01 registers W-001 with solomonar task register, and only then can orch-02 start. The two tasks run in parallel in separate worktrees; merges into main happen one at a time, and W-002 updates from main before its own merge.](docs/images/workflow-parallel.svg)

**The user starts tasks one at a time; the tasks then run in parallel.** The blue arrows are the only registry writes. Each is one `solomonar task` command, which validates ownership, the transition, and the facts in Git, and commits with a compare-and-swap on `refs/heads/workflow-state`, so there is no state checkout and no lock. That branch is never merged into `main`. Merges happen one at a time: before its merge, W-002 takes in the current `main`, CI always reruns, and affected agent reviews rerun for conflicts, overlapping files, changed relevant contracts or dependencies, or uncertain impact. Reused verdicts name their original revision and why the update does not affect their scope. A worktree isolates files, not databases or ports, which are separated explicitly.

## 4 / 5 · Status and stage, two separate axes

![Stage shows where the task is in the workflow, from setup to complete; status shows whether it can advance: planned, reserved, running, done, with the blocked, paused, and cancelled branches. Only a confirmed merge leads to done and complete.](docs/images/workflow-status-stage.svg)

**Status says whether the task can move on; stage says where it has reached.** A blocked or paused task keeps its stage and resumes from there. A tier 0 task goes from implementation straight to integration, and a tier 1 task skips the adversarial review; the CLI enforces both from the tier recorded at triage. A failed review or an unrun check never leads to `done`. `done` is final at merge: `solomonar task done` checks that the merge commit is in `main` and writes the compact record with stage `complete`.

### What the record stores, what Git shows

State holds decisions; Git holds what happened. A fact Git can derive is never stored, so the record cannot drift from it unnoticed: `task status` shows the drift and the commands allowed next.

| Stored in the record | Meaning |
| - | - |
| `status` · `stage` | Whether the task can advance, and where it is. |
| `tier` | Risk tier from triage; the CLI enforces the stage skips. |
| `revision` | The candidate the reviews pin; verified to exist on `feature/<ID>`. |
| `blockers` · `next_action` | What stops the task, and who acts next. |

| Read from Git by `task status` | Meaning |
| - | - |
| worktree · `HEAD` | Where the work is, and whether `HEAD` moved past `revision`. |
| dirty files | Uncommitted work to preserve on resume. |
| merge | Whether `feature/<ID>` is merged into `main`. |
| PR · checks | Through `gh` when available; otherwise `unknown`, never passing. |

## 5 / 5 · Closure inside the feature PR

![On the feature branch, the final spec commit becomes spec_ref and one last commit deletes the work spec; the merge commit keeps both in main's history, and solomonar task done writes the compact record with spec_ref and merge_commit.](docs/images/workflow-closure.svg)

**Current docs stay small; history stays in Git.** After every gate passes, the last `spec(W-001)` commit is the final spec and becomes `spec_ref`; a new one is made only if the work spec changed. One last commit deletes the work spec from `docs/work/`, with no agent review and a fresh CI run. The PR merges with a merge commit, never squash or rebase, so both commits stay in the history of `main`. There is no second PR: `solomonar task done` records the merge and keeps only a summary and the references. Details load on demand with `git show "<spec_ref>"`.

---

Source: [`assets/framework/`](assets/framework/) (installed as `docs/solomonar/`) · [CLI interface](docs/cli/interface.md) · Logo: [`docs/logo/`](docs/logo/README.md) · Solomonar v0.1.0
