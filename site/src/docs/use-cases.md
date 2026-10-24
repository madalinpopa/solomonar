---
title: AI coding agent workflow examples
navTitle: Workflow examples
description: Explore ten Solomonar workflow examples with agent prompts for product discovery, specifications, bug fixes, parallel tasks, reviews, and recovery.
kind: How-to examples
group: Guides
order: 9
sources:
  - assets/framework/skills/solomonar/SKILL.md
  - assets/framework/skills/prototype/SKILL.md
  - assets/framework/skills/spec/SKILL.md
  - assets/framework/skills/work-spec/SKILL.md
  - assets/framework/agents/orchestrator.md
  - assets/framework/agents/implementer.md
  - assets/framework/rules.md
  - internal/cli/run.go
---

These examples follow **Ledger**, a fictional shared-expenses project. Its expense screens and CSV export are example product features, not features built into Solomonar. Start with the situation that matches your work.

> **Before you try them:** the prompts assume a project prepared with `solomonar init`. If you have not done that yet, follow [Set up a project]({{ '/docs/set-up-a-project/' | url }}). Task IDs below are illustrative.

Paste **agent prompts** into a top-level agent session. The portable wording is “Use the solomonar skill.” Codex also accepts `$solomonar`; Claude Code uses `/solomonar`. **Terminal commands** go in your shell. The binary checks and records work; the agent session coordinates it.

## Explore an idea before choosing a stack

**Situation:** you know the problem, but have not decided how adding an expense should feel.

**Agent prompt:**

```text
Use the solomonar skill. Prototype adding a shared expense to Ledger.
Let me try the empty list, a valid expense, and an invalid amount.
Use fictional data; do not choose a production stack yet.
```

The entry skill routes to `prototype`. It builds a small, clickable flow under `prototype/` that opens directly from the file system. Storage and external services are simulated and documented.

**Try it:** open the supplied entry file, add an expense, submit an invalid amount, and describe what should change. An accidental defect is fixed; an intentional change is recorded. When you confirm the flow is ready, the handoff gives its flow ID and a committed `SHA:path` reference.

**You leave with:** a flow you have tried and a durable handoff to `spec`. The prototype has not approved a database, library, or production architecture.

## Turn a confirmed flow into product rules

**Situation:** the add-expense flow is ready, and you want its behavior to survive beyond the prototype.

Paste the real handoff reference in place of `<full-SHA>` and `<entry-file>`, and use the flow ID returned by your prototype handoff.

**Agent prompt:**

```text
Use the spec skill with <full-SHA>:prototype/<entry-file>, flow add-expense.
Record what the flow demonstrates. I confirm that an invalid amount must
show an error and must not add an expense. Keep undecided behavior open.
```

The `spec` skill reads the committed flow and its handoff notes. It separates **Observed** behavior from **Agreed** rules and **Open** questions. Your confirmation supplies evidence for the agreed validation rule; it does not approve unrelated observations.

**You leave with:** capability rules with stable IDs and sources, plus unresolved questions. For example, expense behavior may belong in `docs/specs/expenses.md`; storage remains a separate decision. See [the book: specifications]({{ '/docs/framework/#the-book-specifications' | url }}).

Already know the behavior of an API or CLI? Give `spec` the decided rules directly. A UI prototype is optional.

## Build the first useful slice

**Situation:** add-expense behavior is agreed, but Ledger has no runnable application yet.

**Agent prompt:**

```text
Use the solomonar skill. Implement the agreed add-expense flow as Ledger's
first runnable slice. I must be able to save an expense and see it after
reloading. Exclude accounts, payments, and recurring expenses.
```

1. This top-level session takes the orchestrator role, registers one task, confirms registration, and creates its worktree.
2. `work-spec` defines the outcome, boundaries, acceptance criteria, and checks. Only the minimum runnable baseline needed for this slice enters scope. Necessary architecture or dependency choices go to you for a decision.
3. Once readiness passes and canonical changes are applied, the orchestrator presents the implementation plan. After you explicitly approve that exact plan, it commits `Status: Approved` before delegating. The implementer follows the plan and returns a committed candidate. Persistence makes this a tier 2 change: required specialists review it, followed by adversarial review.

**You leave with:** a bounded implementation tied to observable checks. A useful criterion here is: after saving a valid expense and reloading, that expense is still listed. A passing implementation handoff alone does not complete the task.

## Fix a bug in an existing product

**Situation:** Ledger already exports CSV, but a description containing a comma becomes two fields.

**Agent prompt:**

```text
Use the solomonar skill. Fix the existing CSV export: the description
"Coffee, beans" must remain one field when the export is parsed.
Keep the existing columns and ordering. Add a regression check.
Use the current application and its existing build tools.
```

The orchestrator prepares a bug-fix task. `work-spec` reads the affected code and relevant rules, captures the reproduction and expected result, and names a regression check. If the expected behavior was never agreed, it is clarified and recorded before implementation.

Missing run instructions do not mean the application is missing. Commands are discovered from repository scripts, manifests, and existing instructions. Existing failures are distinguished from regressions introduced by the change.

**You leave with:** the smallest correction and evidence that the comma case works. The task does not authorize replacing the application skeleton or changing unrelated export behavior.

## Keep maintenance work small

**Situation:** the installation instructions are wrong, while the software already behaves correctly.

**Agent prompt:**

```text
Use the solomonar skill. Correct only the installation instructions in
README.md using the repository's current scripts. Verify the examples
and links. Do not change application behavior or requirements.
```

The implementer may edit documentation explicitly named in the work spec. Documentation-only work without behavior or requirement changes is tier 0: applicable CI checks, with no agent-review stages. It needs no new runnable baseline.

For a **refactor**, make a separate request that names both the structural target and the behavior to preserve:

```text
Use the solomonar skill. Extract the existing CSV formatting into one
module. Preserve column order, quoting, and output for the existing cases.
Add no dependency or new export format.
```

**You leave with:** a change whose checks match its scope. A code refactor receives the applicable code-review path; it does not inherit the documentation-only tier.

## Plan work and run independent tasks in parallel

**Situation:** you want to remember CSV export for later, then work on independent outcomes in separate sessions.

**Terminal, in an initialized project:**

```sh
solomonar task plan --title "Export expenses as CSV"
```

Use the ID returned by the command. In a new top-level session, ask:

**Agent prompt:**

```text
Use the solomonar skill. Work on W-001.
```

Wait for **Registration confirmed** before starting another task in another top-level session. Each orchestrator owns one task and its worktree. Independent tasks can then run concurrently; shared services still need separate ports and test data.

If export needs the first task's code, record that dependency using its real ID:

```sh
solomonar task plan --title "Export expenses as CSV" --depends-on W-001
```

**You leave with:** distinct task records and ownership. A dependent task waits until its prerequisite is merged into `main`; an available worktree is not proof that the dependency is ready.

## Answer a decision without losing independent work

**Situation:** an implementation needs a storage choice that has not been approved.

The orchestrator presents the options and records a decision such as `W-001/D-1`, a blocker, and the next action for you. Dependent work waits; independent authorized work can continue.

**Agent prompt, in the session that requested the decision:**

```text
For W-001/D-1, I approve the proposed SQLite option for local persistence.
Keep cloud synchronization outside this task.
```

This is an illustrative answer: use the option and decision ID actually presented to you. The owning orchestrator records the approval, updates the affected documents, and resumes the dependent work. If its session ended, resume it before answering.

**You leave with:** a durable decision with clear scope. Silence, a prototype's simulated storage, and an answer sent to another task do not approve the choice. Existing approvals remain valid within their recorded scope.

## Recheck a change when main moves

**Situation:** CSV export passed review, but another task changed a shared policy function from returning a string to returning an object. The two tasks edited different files.

**Agent prompt, to the owning orchestrator:**

```text
Main changed the policy result used by this exporter. Integrate current
main and verify the export against the new contract. Reassess the affected
reviews on the resulting revision before merging.
```

The implementer merges current `main` into the task branch. CI reruns. The orchestrator checks changed files **and** the relevant contracts, dependencies, and requirements; disjoint files alone do not justify reusing a verdict.

Required reviewers examine the same stable candidate. A correction creates a new candidate and the affected checks run again. A reused verdict retains its original SHA and a concrete reason why its scope is unaffected; unclear impact requires a rerun.

**You leave with:** evidence for the integrated revision. A conflict-free merge or a passing smoke test alone does not prove the exporter still handles the policy result correctly. See [review and integration]({{ '/docs/workflow/#implement-and-review-one-candidate' | url }}).

## Pause, resume, or hand work to another session

**Situation:** you need to stop while CSV export is being implemented.

**Agent prompt, in its current session:**

```text
Pause W-001. Preserve the current work and record what remains to do.
```

On a later visit, inspect the stored record:

**Terminal, in the same initialized project:**

```sh
solomonar task show W-001
```

**Agent prompt:**

```text
Use the solomonar skill. Resume W-001 from its existing worktree and PR.
Inspect the saved work and actual Git state before continuing.
```

Resuming reconciles the record with the worktree, decisions, and PR. Uncommitted work is preserved. A lost review report without a durable reference must be verified again. If the PR already merged, the orchestrator verifies and closes it instead of rebuilding the feature.

If ownership must change, explicitly request the handoff and confirm the previous execution stopped. Only then does the new session take ownership before writing. Elapsed time alone never transfers ownership.

**Agent prompt, in a new session after the old one stopped:**

```text
Use the solomonar skill. The session that owned W-001 has stopped.
Take over W-001 and continue from its approved plan.
```

The new session runs `task handoff`, then `task status`. Its `plan` section shows, for example, `steps 1 of 2 committed; remaining S-2`, and `dirty` lists an uncommitted file the earlier implementer left. The session inspects that file, runs `task confirm`, and delegates only `S-2`, naming the partial work to reuse. `S-1` is not repeated: its checkpoint commit already holds it. The full terminal output is in the [`task status` example]({{ '/docs/cli/task/#solomonar-task-status' | url }}).

**You leave with:** the same task, worktree, and PR, with a known next action. `task show` reports the stored record, not live agent activity; `task status W-001` compares it with the actual worktree and PR. See [task state]({{ '/docs/task-state/' | url }}) for the distinction between recorded state and Git or forge facts.

## Finish the task and handle a later regression

**Situation:** the export meets its acceptance criteria and every required gate passes.

The orchestrator keeps the final work specification in a dedicated spec commit, records its `spec_ref` in the PR, then makes a separate commit removing the active work-spec file. The PR merges with a **merge commit** so both remain in history. Only after the merge is verified does `task done` write `done / complete`.

**Check the result in the terminal:**

```sh
solomonar task show W-001
```

The compact record keeps the summary, merge commit, PR, and final specification reference. That reference can answer a later question about what was agreed, even though the active work-spec file was removed.

If the merged feature later regresses, create related work using its real task ID:

```sh
solomonar task plan --title "Fix CSV quoting regression" --related W-001
```

**You leave with:** preserved agreement and a separate regression task. The merged task stays `done`; a later defect does not erase its history. Read [closure and recovery]({{ '/docs/workflow/#integrate-and-preserve-the-final-specification' | url }}) for the complete gates.

---

These examples adapt the framework/CLI design review of 21 October 2026 and are checked against the source contracts below. They illustrate the workflow; they are not a claim that the fictional Ledger project or the complete setup path was executed. For adding or removing extensions after `init`, see [`solomonar extension`]({{ '/docs/cli/extension/' | url }}); for upgrades, [`solomonar upgrade`]({{ '/docs/cli/upgrade/' | url }}). For adding or removing runtimes, see [`solomonar runtime`]({{ '/docs/cli/runtime/' | url }}); for synchronization, consult the [planned commands]({{ '/docs/cli/#planned-commands' | url }}) and the [project files reference]({{ '/docs/configuration/' | url }}).
