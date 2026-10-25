# Registry reference

Framework version: 0.3.6

Detail for `rules.md` section 7, loaded only when a command, a CLI rejection, or a record field needs it. The
CLI enforces what this file describes; `rules.md` wins on conflict.

## 1. Storage

- `state.json` holds `{"schema_version": 2}`; each task is `tasks/<ID>.json`. Read one with
  `git show refs/heads/workflow-state:tasks/<ID>.json` when the CLI is unavailable. A schema-1 registry (one
  `state.json` with every task) is migrated once by the user with `solomonar task migrate`; commands reject it
  until then (`registry_outdated`).
- The registry has no filesystem checkout. Compare-and-swap protects updates from concurrent writers; no
  checkout or lock is needed.
- Each state-changing command creates one `state(<ID>): ...` commit. Read-only commands, dry runs, and
  unchanged moves create none. Git history is the transition history; the registry holds no event log.
- The CLI pushes `workflow-state` when a remote is configured and reports publication failures; the local ref
  stays canonical after publication failure. Multiple independently writable clones are out of scope.
- Local Git facts are required. PR and check facts come from `gh` when it is installed and authenticated;
  otherwise `task status` reports them as `unknown`, never as passing. `task done` relies only on Git.

## 2. Commands

| Command | Who | Effect |
|---|---|---|
| `task plan --title <t>` | user | Creates a `planned` record without owner |
| `task register [<ID>] --title <t> --owner <o>` | orchestrator | Claims the `planned` `<ID>` or allocates the next `W-NNN`; sets `reserved` / `setup`, branch `feature/<ID>`, `next_action` |
| `task move <ID> --as <owner> <field flags>` | owner | Validated transition and field update; repeated `--agent <id>:<role>` replaces `active_agents`, `--agent none` clears it |
| `task block <ID> --as <owner> --kind <k> --summary <s>` / `task unblock <ID> <blocker-ID> --as <owner>` | owner | Adds or clears a blocker; combine `--status` and `--next` atomically (`rules.md` section 3) |
| `task done <ID> --as <owner> --merge <sha> --spec-ref <sha:path> --summary <s>` | owner | Verifies the merge commit is reachable from main and `spec_ref` resolves and names the last `spec(<ID>)` commit merged, then writes the compact record (`done` / `complete`) |
| `task handoff <ID> --owner <o>` | new owner, on user instruction | Ownership change; clears the confirmation (`rules.md` section 10) |
| `task confirm <ID> --as <owner>` | owner, after inspection | Checks the branch, recorded worktree and base, and the approved plan; records that this execution continues the task (`rules.md` section 10) |
| `task migrate` | user | Rewrites a schema-1 registry as one file per task, once |
| `task show [<ID>]` | anyone | Reads the stored record |
| `task status <ID> [--json]` | anyone | Reads record, Git and available forge facts, drift, and allowed next commands; writes nothing |

- Owner commands accept `SOLOMONAR_OWNER` as the terminal default for `--as`; shells need not preserve it.
- Discover field flags with `solomonar help task move`.
- The CLI validates schema, ownership, and allowed transitions, and checks facts before writing: `--revision` is
  a full SHA reachable from `feature/<ID>`; `--spec` exists in the feature worktree or at the recorded
  `revision`; a stage move respects the recorded `tier`; stage changes past `setup` and `task done` need a confirmed task (leaving `setup` confirms it; `task
  confirm` applies only past `setup`);
  entering `implementation` needs the approved plan committed; leaving it needs a checkpointed, clean candidate
  (`rules.md` section 5 Checkpoints).

## 3. Status

| Status | Meaning |
|---|---|
| `planned` | Known work; no owner yet |
| `reserved` | Registration committed; worktree may not exist yet |
| `running` | Owner is actively advancing the stage |
| `blocked` | A blocker stops all authorized work on the task |
| `paused` | Stopped on instruction; resumes on instruction |
| `cancelled` | Stopped without merge; work preserved; explicit resumption required |
| `done` | Feature merge into main confirmed; compact record written; final |

- `planned` → `reserved` only through `task register`; registration may also allocate a new task directly.
  `reserved` → `running` pairs with `setup` → `specification` in one update. Only `task done` sets `done` / `complete`.
- `reserved` or `running` → `blocked` or `paused` → back to the prior status at the preserved stage.

## 4. Fields

| Phase | Required fields |
|---|---|
| Planned | `title`, `status`, `updated_at` |
| Reserved | adds `owner`, `branch`, `stage`, `next_action` |
| Past setup | adds `base` (full SHA, never changed), `worktree` (path relative to the main checkout), `confirmed` |
| Before implementation | adds `spec` (worktree-relative path) |
| Review / integration | adds `revision`: latest reported committed candidate, full SHA; `tier` (0, 1, 2) from that candidate's triage (`rules.md` section 6) |
| Done (compact) | `title`, `status`, `stage`, `summary`, `spec_ref`, `merge_commit`, `updated_at`; optional `base`, `pr`, and `related_task`; active fields removed |

- Empty `active_agents` and `blockers` are omitted. `active_agents` entries `{id, role}` record the last
  delegation, not liveness. Optional fields include `pr`, `depends_on`, `related_task`, and `session_ref`;
  omit empty values.
- Blocker: `{id, kind, summary, ref?}`, kind ∈ `decision` | `dependency` | `check` | `environment`. Blocker
  IDs stay stable within the task and are never reused: `last_blocker` holds the highest number issued.
- `next_action`: `{actor, action}`, e.g. `{"actor":"user","action":"Decide W-001/D-1: session storage"}`.
- Use full SHAs and RFC 3339 UTC timestamps. `worktree` is relative to the main checkout, and Git's worktree listing stays authoritative. After `git worktree move`,
  record the relocation with `task move <ID> --as <owner> --worktree <path>`.
- `revision` is stored because reviews pin it, `base` because it bounds the task's checkpoint history; the
  CLI verifies both.
- Resolve active `spec` paths against the feature worktree (`rules.md` section 4 step 4) or its recorded `revision`.
