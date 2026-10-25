---
title: Prototype workflow
navTitle: Prototype workflow
description: Build runnable CLI, web, module, and service prototypes, check their behavior, approve the result, and carry it into Solomonar specifications and managed tasks.
kind: Workflow guide
group: Guides
order: 9
sources:
  - assets/framework/skills/prototype/SKILL.md
  - assets/framework/skills/prototype/template.md
  - assets/framework/skills/prototype/references/cli.md
  - assets/framework/skills/prototype/references/web.md
  - assets/framework/skills/prototype/references/module.md
  - assets/framework/skills/prototype/references/service.md
  - assets/framework/skills/spec/SKILL.md
  - assets/framework/skills/work-spec/SKILL.md
  - assets/templates/AGENTS.md
---

A prototype is an executable, human-readable agreement about what the application should do. It can cover a whole application or one function, component, module, command, or workflow. Once approved, it defines the behavior represented by those scenarios; production architecture remains a separate decision.

This guide assumes a project prepared with `solomonar init`. For setup, follow [Set up a project]({{ '/docs/set-up-a-project/' | url }}). Put agent prompts in a top-level agent session and run terminal commands in your shell. The examples use Ledger, a fictional shared-expenses application.

For new or changed behavior, follow **prototype → human review and approval → specifications → production implementation**. An existing prototype follows the same loop. A feature request, successful test, or approved implementation plan cannot substitute for reviewing and approving the demonstrated behavior.

## Choose the smallest useful part

Start with the next outcome someone needs to judge. Demonstrate its primary path and relevant alternate, validation, and failure cases. A module needs a small executable consumer; it does not need a website or server just to make it reviewable.

| Interface | Demonstrate | Typical local setup |
| --- | --- | --- |
| CLI | Commands, arguments, flags, help, output streams, and exit codes. | Standard argument parser; local JSON when state must survive commands. |
| Web | Navigation, forms, interactions, state changes, keyboard use, and feedback. | Semantic HTML, CSS, vanilla JavaScript, and a local HTTP server when needed. |
| Module or library | Public calls, inputs, return values, state, invariants, and failures. | Target-language module plus a consumer or harness and deterministic test doubles. |
| Standalone service | Required endpoints, HTTP methods, validation, response bodies/statuses, and state effects. | Runtime HTTP server; in-memory or local JSON state; executable requests. |

Use realistic fictional seed data and simulate external collaborators locally. Keep every exposed action functional. Prefer an available runtime and its standard libraries; justify extra dependencies or setup by the behavior they make possible. A prototype should run without production credentials, infrastructure, or a connection to production code.

Ask the agent in your initialized project:

```text
Use the solomonar skill. Prototype Ledger's expense-splitting function.
Split an integer amount in minor currency units among a positive number
of people. For 1001 among 3, show [334, 334, 333]; preserve the total.
Demonstrate invalid inputs through a runnable consumer. Keep this limited
to the module, and document how I run and check each scenario.
```

The entry skill routes this request to `prototype`. It asks only the next necessary question and builds a draft for you to try. You do not need to settle the production stack first.

## What the prototype produces

The result lives under `prototype/`, alongside a short `prototype/README.md`. Independent parts can be grouped by type. Create only the directories you need and preserve an existing runnable layout.

An example project exploring several interfaces might contain:

```text
prototype/
├── README.md
├── cli/
│   ├── expenses.py
│   ├── seed.json
│   └── state.json
├── web/
│   ├── index.html
│   └── app.js
├── module/
│   ├── split.py
│   ├── demo.py
│   └── test_split.py
└── service/
    └── server.mjs
```

These are illustrative application files the agent might create, not commands or features built into Solomonar. Run the exact commands recorded in your own prototype README.

| README record | What it tells you |
| --- | --- |
| Run, test/inspect, and reset | Each part's working directory, runtime requirements, exact commands or steps, and how to restore seeded state. |
| Flows | Stable flow IDs, entry files relative to `prototype/`, numbered actions and expected outputs/state changes, status, and approval scope. |
| Simulated behavior | Which collaborators are local substitutes, how to select their outcomes, and what their simulation cannot demonstrate. |
| Prototype-only execution choices | Why additional dependencies or setup are needed; these choices approve no production dependency. |
| Intentional changes and open questions | What changed and why, plus unresolved questions marked as blocking or non-blocking. |

Prototype code stays simple enough to read. Its implementation can be replaced when production code is written; keep the runnable behavioral reference available for later inspection.

## Run and check the behavior

Every independent part needs a run entry point and a way to check its observable results. Use a small harness for code that cannot run on its own. A compilation or startup check alone does not show whether the behavior is right.

The following examples assume commands run from the project root and that the named files implement the example interfaces. They use Python 3 or Node.js only as possible runtimes. Use the target language when the interface depends on it.

### CLI example

Start with a scoped request:

```text
Use the solomonar skill. Prototype Ledger's expense CLI.
Show help, adding a valid expense, listing it, and rejecting an invalid
amount without changing state. Document the run, check, and reset commands.
```

For an expense command, the generated README could give this sequence:

```sh
cp prototype/cli/seed.json prototype/cli/state.json
python3 prototype/cli/expenses.py --help
python3 prototype/cli/expenses.py add --amount-minor 1250 --description Lunch
python3 prototype/cli/expenses.py list
python3 prototype/cli/expenses.py add --amount-minor 0 --description Invalid
python3 prototype/cli/expenses.py list
```

Check that help explains the supported commands and flags, the valid expense appears in the list, and the invalid amount changes no state. Successful commands return zero; the invalid command returns the documented nonzero exit code with an actionable error on stderr. Command results go to stdout. The initial copy restores the same sample state for another run, using only files in this prototype.

The README should give the exact expected output and failure code for the chosen interface. Inputs must also be available without mandatory interactive prompts so the sequence can run in a script.

### Web example

To explore adding an expense before choosing a production stack:

```text
Use the solomonar skill. Prototype adding a shared expense to Ledger.
Let me try the empty list, a valid expense, and an invalid amount.
Use fictional data; do not choose a production stack yet.
```

For a browser expense form:

```sh
python3 -m http.server 4300 --bind 127.0.0.1 --directory prototype/web
```

Open `http://127.0.0.1:4300/`. Follow the README's numbered scenarios: inspect the empty list, submit a valid expense, submit an invalid amount, and reach a simulated failure using its documented trigger. Check the resulting records and feedback, then try the controls with the keyboard and verify visible focus and labels.

If persistence is part of the flow, reload and check the result. The reset instructions must restore seeds while affecting only this prototype's storage keys or database. Use `localStorage` for small simple state and IndexedDB when structured persistence is justified. Keep a stable local origin; use direct file opening only when the chosen browser features work reliably there.

### Module example

For the expense-splitting function:

```sh
python3 prototype/module/demo.py
python3 -m unittest discover -s prototype/module -p 'test_*.py'
```

The consumer shows the public call and its results: splitting `1001` among `3` returns `[334, 334, 333]`, with three shares totaling `1001`. It also demonstrates the documented failure when the participant count is zero. The example is stateless, so another run starts fresh.

Here, a small standard-library test file can check the same contract automatically. Automated behavior checks are useful when they add confidence; the skill does not require a production test framework or CI pipeline for every prototype.

### Service example

Start a generated local expense API:

```sh
node prototype/service/server.mjs
```

From another terminal, exercise its documented interface:

```sh
curl -i http://127.0.0.1:4310/expenses
curl -i -X POST http://127.0.0.1:4310/expenses \
  -H 'Content-Type: application/json' \
  -d '{"description":"Lunch","amountMinor":1250}'
curl -i -X POST http://127.0.0.1:4310/expenses \
  -H 'Content-Type: application/json' \
  -d '{"description":"Invalid","amountMinor":0}'
curl -i http://127.0.0.1:4310/expenses
```

In this example, the valid POST returns `201`, the invalid amount returns `422` with a validation message, and the final GET includes the valid new expense without an entry for the rejected request. Those statuses describe this example contract; other APIs must choose statuses matching their own HTTP semantics. If state is in memory, stopping and restarting the process restores its deterministic seed data. Restarting is not a persistence demonstration; the README must state that limitation.

For every interface, the agent runs the documented scenarios before presenting the result, fixes introduced defects, and reports the checks and their limits. Missing browser access, for example, leaves browser interactions unverified; a passing module test cannot stand in for that check.

## Review and approve a specific flow

Try the runnable result and give feedback against its flow ID. If an interaction is wrong, describe the input and expected result. Intentional behavior changes are recorded; defects that differ from already approved behavior are corrected without redefining the agreement.

When the demonstrated behavior is right, approve its scope explicitly:

```text
I approve flow split-expense as demonstrated: integer shares preserve
the total, remainder units go to the first participants, and zero
participants produces the shown validation error. It is ready for spec.
```

The README records the approval source (`user YYYY-MM-DD`) and the covered scenarios. Flow statuses mean:

| Status | Meaning |
| --- | --- |
| `draft` | The current behavior has not been approved; it cannot authorize specs or production behavior. |
| `ready for spec` | Explicit approval covers the demonstrated scenarios, and the verified flow can be handed off. |
| `superseded` | A replacement flow is named; its old record and approved revision remain inspectable. |

Before changing an approved flow's behavior, the agent returns every affected flow to `draft`, including flows affected through shared code. Review the changed result before it becomes ready again. Silence, a commit, or an unrelated approval establishes nothing about that change.

## Carry the agreement into specifications

The handoff identifies a committed approved snapshot, not the working copy. It gives one reference per selected flow:

```text
<full commit SHA>:prototype/module/split.py, flow split-expense
```

The `spec` skill reads that revision's README, approval record, entry file, and supporting files. It extracts actions/inputs, outputs, state transitions, validation, business constraints, relevant success/failure/edge cases, and persistence/integration outcomes. It records approved behavior as **Agreed** rules in `docs/specs/`, adding shared interaction rules to `docs/ui.md` when needed.

Paste the real handoff SHA and flow ID into a request such as:

```text
Use the spec skill with <full-SHA>:prototype/module/split.py, flow split-expense.
Extract the approved splitting and validation scenarios into capability rules.
Keep undecided behavior open.
```

For example, a rule can describe how remainder units are allocated and how invalid participant counts fail. It should not require the prototype's loop, helper functions, JSON fixture format, or mock collaborator. Simulated persistence demonstrates the expected visible outcome; it does not select a production database.

Unrepresented or ambiguous behavior returns to the prototype for a decision, demonstration, and approval before the related rule is finalized. A prototype/spec conflict is resolved there first. The agent cannot silently edit a requirement to fit the code or an accidental prototype defect.

## How this fits a managed task

The prototype skill runs in the primary checkout. It writes `prototype/` and its allowed project-index entries; it does not write production code, canonical specs, or registry state. The rest of the framework carries its handoff forward:

| Owner | Responsibility |
| --- | --- |
| `solomonar` entry skill | Route discovery or behavior changes to `prototype`, and an approved handoff to `spec`. |
| `spec` | Extract approved behavior into canonical rules with approval and prototype references. |
| Owning orchestrator and `work-spec` | Prepare the task, cite the approved flows/rules, and require coverage for new or changed behavior before implementation. |
| Implementer | Follow the approved specs and plan; stop affected work if prototype coverage is missing or conflicting. |
| Spec reviewer | Check that changed behavior is represented, approved, and reflected in the governing specifications. |

Prototype approval and implementation-plan approval cover different things. First approve what the system does; later approve the plan for building the selected task. Neither approval automatically selects production architecture or dependencies.

If a task decision or review introduces a behavior change, the orchestrator holds dependent work, routes the prototype update to the primary checkout, and waits for its approved committed handoff. It brings that snapshot into the task branch, updates canonical specs and work-spec references, then resumes dependent implementation or corrections. Do not edit the prototype from the feature worktree to skip the loop.

A defect fix restoring represented approved behavior reuses its existing approval. A technical change with no behavioral effect needs no new prototype. New behavior always needs representation and approval, including in an existing project.

These are workflow gates followed by the skills and roles. The CLI records and checks task/Git facts; it does not inspect a demo or independently verify human prototype approval. Continue with [Task workflow]({{ '/docs/use-cases/' | url }}) for task prompts, plan approval, reviews, and merge. [The task lifecycle]({{ '/docs/workflow/' | url }}) explains the concepts and gates behind those steps.
