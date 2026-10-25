# Approvals and decisions

Framework version: {{ .Version }}

Load before raising/recording a decision or resolving project open questions. In a task, the owning orchestrator acts;
outside a task, the user’s top-level session acts. `task` means `solomonar task`; section references name `rules.md`.

- Any new architecture decision or dependency change requires user approval, including when a reviewer
  proposes it as a fix. Applying an approved decision within its recorded scope needs no new approval.
- The agent that identifies the need may investigate and must prepare a decision request: question,
  recommended option, alternatives, impact, blocked work. It must not apply the choice before approval.
- Register each decision and open question at the level it affects:
  - Project level: it affects or constrains the whole project beyond one task, such as an architecture style,
    a shared contract, or a dependency other work would use. Decisions take `P/D-n`. Project open questions are
    the open questions in `docs/specs/*.md` and `docs/ui.md` (`<capability>/Q-n`) plus `pending` `P/D-n` entries.
  - Task level: it affects or constrains only one task. Decisions take `<ID>/D-n`; open questions take
    `<ID>/Q-n` in the work spec. A task-level decision never binds another task.
  - When the level is unclear, propose one with the question and let the user choose. A task-level choice that
    later must bind other work needs a project decision that supersedes it.
- Where entries live:
  - Task entries: in the task worktree. Commit each entry change with the spec updates it requires:
    `docs(<ID>): record <ID>/D-n`.
  - Project entries: only on `<main>` in the main checkout, never in a task worktree or on another branch, so
    every session numbers from the same file. If the main checkout is not on `<main>`, ask the user before
    switching it. `<n>` is one more than the highest `P/D-<n>` in `docs/decisions.md` and
    `.solomonar/manifest.json`; never reuse an ID. Commit each entry change alone, right after numbering it:
    `docs(decisions): record P/D-n`. `Scope` starts with `project; `. Create `docs/decisions.md` with the title
    `# Decisions` and index it in `docs/README.md` when absent.
- Orchestrator procedure:
  1. Assign the decision ID at its level: the next `P/D-n` or the next `W-001/D-n`.
  2. Record the blocker and next action together:
     `task block <ID> --as <owner> --kind decision --summary <s> --ref <decision-ID> --next "user:<action>"`.
     Add `--status blocked` when no independent authorized work can continue. Write the `pending` entry where
     its level requires. Wait for every running review to report before changing the candidate (`rules.md` section 6
     Candidate stability).
  3. Present the decision request.
  4. Keep the recovery evidence current (`rules.md` section 10) and wait. No response is not approval. Never repeat the question or retry automatically.
- While waiting, affected and dependent work must wait. Independent authorized work may continue.
- On approval: set the entry `approved` with the selected option. If it changes behavior, first complete
  `AGENTS.md` rule 7 through `prototype` in the primary checkout; keep dependent work blocked. A project entry
  or a prototype handoff reaches the task branch only through main; a handoff on `prototype/<topic>` waits
  until the user merges it. Once it is on main, delegate the merge to the implementer (packet objective
  "merge main into `feature/<ID>`", `agents/implementer.md` section 7). Then update affected specs via `spec` in-task
  mode, and commit before clearing the blocker. Use
  `task unblock <ID> <blocker-ID> --as <owner> --next "<actor>:<action>"`; when removing the last blocker from
  a `blocked` task, also pass `--status reserved` at `setup`, otherwise `--status running`. Keep it blocked
  while other blockers stop all work; paused or cancelled tasks still require explicit resumption (`rules.md` section 10).
- On rejection: keep the entry `pending` and list the rejected option with its reason in it. Re-delegate the
  role that identified the need when the revised option needs investigation; otherwise present a listed
  alternative. Present the revised option under the same decision ID.
- Entry format: `## <ID>: <question>`, then named bullets `Status`, `Selected option` (once approved),
  `Rejected options` (`none` when empty), `Scope`, `Rationale`, `Date`, and `References`. Keep the labels exact
  so the CLI can read the status (`pending` | `approved` | `superseded`); only `approved` authorizes work.
  References identify any superseded decision and its replacement.
- Reuse a decision only within its scope. A superseding decision must name the decision it replaces, set
  that entry's status to `superseded`, and update the affected canonical rules.
- Outside any task, the user's top-level session, routed by the `solomonar` skill, records project decisions:
  present the decision request; write nothing until the user explicitly approves an option, then append the
  entry `approved`. Record it `pending` only when the user defers it. No response is not approval. A decision
  that changes behavior then completes `AGENTS.md` rule 7 through `prototype` and `spec`. The task that first
  applies a project decision updates `docs/architecture.md` or `docs/stack.md`.
- When a session reports project open questions, suggest that the user answer them, blocking first, and wait.
  Only after the user agrees, ask them per `AGENTS.md` rule 11. Record an approved decision as above; route a
  behavior answer through `prototype`, then `spec`; leave an unanswered question open.
- Stack extensions follow the same approval rule; adopting or dropping one is a project decision. When
  available, the user's `solomonar extension add|remove` records `P/D-n` for each affected group's complete
  selection, using the entry format above; never hand-write that entry outside a task. The orchestrator instead
  records the project decision and passes `--decision <approved-ID>`. The CLI updates installed extensions'
  decision references and supersession links together.
- Check availability with `solomonar help` (`rules.md` section 11). If extension management is unavailable, report it;
  never substitute manual edits to the manifest or generated adapters.
