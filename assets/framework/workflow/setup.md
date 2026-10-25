# Task setup

Framework version: {{ .Version }}

Orchestrator setup only. `task` commands mean `solomonar task`; policy references retain `rules.md` section numbers.

The user starts tasks one at a time. Started tasks run in parallel. Work selected from a prototype enters at step 1.

1. Read the registry (`solomonar task show`, `rules.md` section 7) and the task brief or existing spec. Check ownership and
   dependencies. A complete work spec is not required to reserve.
2. Run `solomonar task register [<ID>] --title <t> --owner <o>`. It claims the `planned` ID or allocates the
   next `W-NNN` and records `reserved` / `setup`, branch `feature/<ID>`, and `next_action` (`rules.md` section 7).
3. The successful `task register` is the registration barrier. Confirm it to the user. Only then may the next
   task start. A push is not required.
4. Verify the intended current main baseline. If a prerequisite merged remotely, update it first and verify
   it contains the prerequisite's recorded `merge_commit`. From the main checkout, run
   `git worktree add -b feature/<ID> ../<repo>-worktrees/<ID> <main>`.


- If `task register` fails, do not create the worktree.
- If worktree setup fails, keep the registration: blocker kind `environment` (`task block`), status `blocked`,
  stage `setup`.
- Each task has its own branch and worktree. Agents must modify only the assigned worktree.
- When services are shared, assign separate ports and test data. A worktree does not isolate databases or
  external services.
- A task that needs another task's code must wait until that code is merged into main (blocker kind `dependency`).
- Same-file changes on different branches are reconciled at integration.

## Stage procedure

### `setup`
- **Entry:** selected new/planned work, or an owned task resuming at `setup`. **Delegate:** none.
- **Do:** check registry, ownership, and dependencies; register new/planned work and confirm the registration to the user (`rules.md`
  section 4 steps 1–3). Verify the baseline and create the worktree per step 4. On resume, reuse completed
  setup; never re-register the task or duplicate its worktree.
- **State:** registration → `reserved` / `setup`; after worktree verification,
  `task move <ID> --as <owner> --status running --stage specification --next "<owner>:Run work-spec"`.
- **Exit:** the move to `running` / `specification` succeeded.
- **Failure:** failed registration → report, no worktree or confirmation. Setup failure or unmet prerequisite →
  `task block` with kind `environment` or `dependency`, `--status blocked`, and `--next`; preserve registration.
