# Solomonar CLI v0.1: implementation tasks (increment 1)

Tracks the [architecture](architecture.md) §12 plan as reviewable stages of about 500 changed lines each,
tests included; a moderate overrun is acceptable. Each stage is implemented test-first and ends with
`task check` green and the author's review.
Later stages are outlines; each is detailed into tasks when it starts.

Status: `[ ]` open · `[~]` in progress · `[r]` awaiting review · `[x]` accepted

## Step 1: Skeleton

### Stage 1.1 — Run, flags, errors, envelope, `version` `[x]`

- [x] testify v1.12.1 in `go.mod` as a test dependency, `go.sum` committed with it (AD-14)
- [x] `main.go`: `signal.NotifyContext`, `//go:embed assets`, `fs.Sub` to the tree root, `cli.Env` from `os`,
      `os.Exit(cli.Run(...))` (AD-13, AD-9)
- [x] `cli.Env` and `cli.Run`; static command table (AD-2, AD-4)
- [x] Interspersed parsing over `flag.FlagSet`; shared flags `--json`, `-v/--verbose`, `--no-color`, `-C`
      accepted before and after the command (F2, I §3.1)
- [x] `cli.Error` and the exit-code mapping; usage errors exit 2 (I §3.5)
- [x] Human error form on stderr and the JSON envelope on stdout (I §3.4, §3.6)
- [x] `version` outside a project, human and JSON (UC-24, AD-10)

Decisions where the spec is silent (author, 2026-10-21): usage error codes `unknown_command` and
`invalid_usage` (exit 2); unexpected failures are `unexpected_error` (exit 1); `cli.Error` carries `Message`,
shown as `error: <Message> (<Code>)` and as the JSON `message`; an untagged build reports CLI version `devel`;
`-h` is `invalid_usage` until stage 1.2.

### Stage 1.2 — Help `[x]`

Decisions (author, 2026-10-21): `solomonar` without a command stays `unknown_command`; the envelope `command`
is the command for `<command> -h` and `help` otherwise; command help points to `solomonar help` for global
flags.

- [x] `help`, `help <command>`, `-h`/`--help` at the top level and on every command (I §2, §3.1)
- [x] Hand-written help text per registered command; unknown `help <command>` is a usage error

## Step 2: Git runner

### Stage 2.1 — `internal/git` runner and discovery `[x]`

- [x] `git.Repo` and `Run`/`RunInput` with context, stdin, trimmed stdout, errors carrying args, exit status,
      stderr (AD-3, §8)
- [x] `-v` trace of each command line to an injected writer (I §3.1)
- [x] `Discover` from a directory (`-C` or the working directory): worktree root and absolute common
      directory; `ErrNotRepository` (I §3.2)
- [x] Git version parsing and the 2.40 minimum
- [x] Test helpers: isolated temp repositories with commits and a feature worktree (§11)

## Step 3: Registry model and rules (no Git)

### Stage 3.1 — Registry model, decoding, validation, encoding `[x]`

- [x] `Registry`, `Record`, `Status`, `Stage`, `Agent`, `Blocker`, `NextAction` types (rules §7 Fields, §6)
- [x] `task.Rejection{Code, Check, Found, Fix}` (§6)
- [x] `schema_version` read first; newer is `cli_outdated` (CLI-D19, AD-7)
- [x] Strict decoding: unknown and duplicate keys are `registry_invalid` (AD-7)
- [x] Record validation: IDs, enums, formats, required and forbidden fields per status (rules §7)
- [x] Stable encoding: struct field order, sorted task keys, two-space indent, trailing newline (AD-7)

Stage 3.1 does not include the transition rules or the `reserved`-at-`setup` invariant; those are stage 3.2.
### Stage 3.2 — Status and stage transitions `[x]`

- [x] Status transitions of I §8.1 as a table: `transition_not_allowed` names the allowed targets,
      `done_final`, `cancelled_requires_flag`
- [x] Prior status of `blocked`, `paused`, `cancelled` derived from the stage (CLI-D16)
- [x] Stage transitions of I §8.2 by tier as a table: `stage_not_allowed` names the allowed stages,
      `tier_required`; a stage changes only while `running` (`reserved` only at `setup`)
- [x] Invariant: `reserved` exactly while at `setup` (CLI-D16)
- [x] Allowed next statuses and stages from the same tables, for `allowed_next` (AD-8, I §9.3)
### Stage 3.3 — ID allocation and flag values `[x]`

Decision (author, 2026-10-21): blocker IDs are never reused because the record's `last_blocker` holds the
highest number issued (rules §7, I UC-15). `--next` and `--agent` split at the first `:`; `--pr` is a positive
number or an `https://` URL, stored as given; `--agent` roles are checked against installed roles in stage 5.2.

- [x] Next task ID: highest `W-NNN` plus one, at least three digits (I UC-11)
- [x] `last_blocker` in the record: decoding, validation, next blocker ID `B-n` (I UC-15)
- [x] Flag value parsing: `--agent <id>:<role>` and `none`, `--next <actor>:<action>`, `--tier <0|1|2>`,
      `--revision` full SHA, `--pr <url|number>`, blocker `--kind`, `--ref`, `--summary` (I §8.4
      `invalid_value`, `revision_not_full_sha`)

### Stage 3.4 — `Record.Move` and field rules `[x]`

Decisions (author, 2026-10-21): `field_not_allowed` (exit 3) for a field flag outside its stages; `changed` lists
fields in record order (UC-13 example updated).

- [x] `Move` input parsed with the stage 3.3 parsers, applied to a copy, then `checkTransition`; the record
      changes only on success
- [x] `changed` fields in record order; `updated_at` from the clock, RFC 3339 UTC seconds
- [x] Field rules of I §8.2 (`field_not_allowed`) and `spec_required`, `revision_required`, `tier_required`,
      `next_required`, `blockers_required`
- [x] UC-14 check and fix texts for `tier_required` and `stage_not_allowed`

## Step 4: Store, `task plan`, `task show`

### Stage 4.1 — Registry store `[x]`

- [x] Shared Git test helpers in `internal/git/gittest` (standard `testing` only), used by `git` and `task` tests
- [x] `Store.Read`: tip and decoded registry; a missing ref is `not_initialized` (I §8.4)
- [x] `Store.Create`: root commit `state: initialize registry`, create-only `update-ref` (I §8.5, §8)
- [x] `Store.Update`: `hash-object` → `mktree` → `commit-tree -p <old>` → `update-ref <new> <old>`; a lost swap
      re-reads and re-runs `fn`, five attempts, then `cas_exhausted` (AD-5, UC-21)
- [x] Push to the remote with a 30-second limit; failure is `push_failed`, no remote is `no_remote` (I §8.4, §8)
### Stage 4.2 — Cancellation and rejection mapping `[x]`

- [x] Store: a cancel before the swap writes nothing; a cancel during the push keeps the commit and warns
      `push_interrupted`; a push timeout stays `push_failed` (CLI-D18, I §3.5)
- [x] `cli` maps `task.Rejection` codes to exit classes with one table (I §8.4, §3.5)
- [x] `interrupted` exits 128 + signal (130 for SIGINT, 143 for SIGTERM) with fix "rerun"
- [x] Warnings in the output: `warning: <message>` on stderr, `warnings` in the JSON envelope (I §3.3, §3.4)
- [x] `cli_outdated` (exit 3) reaches the user unchanged (CLI-D19)
### Stage 4.3 — `task plan`, `task show` `[x]`

Decisions (author, 2026-10-21): `not_a_repository` exits 3; Git older than 2.40 is `git_too_old` (exit 1, fix
"install git 2.40 or later"); `task plan --dry-run` prints `Would plan <ID>: <title>` and JSON `"dry_run": true`
without a commit; an unpushed write ends with `registry @ <sha>` and the warning; `task show --json` data is
`{"registry", "commit"}`, or `{"id", "record", "commit"}` with an ID.

Decisions confirmed at review (author, 2026-10-21): each command registers its own flags (`bind`) and receives
the shared flags; `task` alone or with an unknown subcommand is `unknown_command`; `help task plan` equals
`task plan -h`; a rejection under `task show <ID>` reads `task show <ID> rejected`; `--dry-run` reads the
registry and applies `Plan` without `Update`; the overview ends with `registry <ref> @ <sha7> · <commit time,
RFC 3339 UTC>`; STATUS is padded to the width of `cancelled`; NEXT shows the summary for `done`,
`<actor>: <action>` otherwise, `-` when empty; IDs sort numerically.

- [x] Repository context in `cli`: discovery from `-C`, Git ≥ 2.40, `-v` trace on stderr, `not_a_repository`;
      remote `origin` and main branch `main` until the manifest exists (I §3.2)
- [x] Two-word commands `task plan` and `task show` in the command table and help
- [x] `task plan`: next `W-NNN`, `--title`, `--depends-on`, `--related`, `--dry-run`; commit subject (I §8.5);
      human and JSON output (UC-11)
- [x] `task show [<ID>]`: overview table sorted by ID, raw record, JSON forms (UC-16)

## Step 5: `task register`, `task move`

### Stage 5.1 — `Registry.Register` (domain) `[x]`

Decisions (author, 2026-10-21): `task register` splits into this domain stage and
the CLI stage 5.2; an owner ID matches `[a-z]+-[0-9]+` like agent IDs, otherwise `invalid_value`; claiming a
planned task keeps its `depends_on` and `related_task` unless the flag is given, which replaces the value; an
unknown ID is `task_not_found`; `--next` names `user` or the owner (I §8.3, no agents exist yet); the default
next action is `<owner>: Create feature/<ID> and its worktree`; a blank or whitespace-only title is
`invalid_value`; an empty `--depends-on` keeps the planned list, so a claim cannot clear it; `owner_in_use` names
the first active task in ID order. The specs mirror every recorded decision (interface and architecture updated
with this stage).

- [x] Allocate the next `W-NNN` or claim a `planned` ID; `reserved` / `setup`, `owner`, `branch feature/<ID>`,
      `next_action`, `updated_at` (I §8.1, rules §4 step 2, §7 Fields)
- [x] Rejections: `not_planned` (the ID exists and is not `planned`), `owner_in_use` (the owner has a
      `reserved`, `running`, `blocked`, or `paused` task; architecture §2), `task_not_found`, `invalid_value`
      (UC-12, I §8.4)
- [x] Warnings: `title_changed` when a claim changes the planned title, `dependency_not_done` per dependency
      that is not `done` (UC-12, I §8.4)
- [x] Commit subject `state(<ID>): reserved/setup, registered by <owner>` (I §8.5)

### Stage 5.2 — `task register` command `[x]`

Decisions (author, 2026-10-21): `--title` and `--owner` are required flags
(`invalid_usage`); a rejection with an ID reads `task register <ID> rejected`; the human form prints the four
UC-12 lines (status, owner, branch, next), the subject, and the registry line; without a push the registry line
ends at the SHA, the warnings follow on stderr, and "Registration confirmed" still prints (rules §4 step 3);
domain warnings come before push warnings; the worktree path is `../<repo>-worktrees/<ID>` with `<repo>` the main
checkout's directory name, relative when run from the main checkout and absolute from another worktree; the
base is the main branch (`main` until the manifest exists); `--dry-run` prints `Would register <ID>: <title>`,
the four lines, and the subject, with no registry line, confirmation, or worktree command; `--json` data is
`{"id", "record", "commit", "pushed"}`, or `{"id", "record", "dry_run": true}` with `--dry-run`; `--dry-run`
reports the domain warnings, never a push warning; with both flags missing the error is `--title is required`.
UC-11, UC-12, and I §8.3 updated to match.

- [x] `task register [<ID>]` in the command table and help; flags `--title`, `--owner`, `--depends-on`,
      `--related`, `--next`, `--dry-run` (UC-12)
- [x] Human output of UC-12, "Registration confirmed" only after the swap, the worktree command; `--dry-run`;
      JSON form; warnings (I §3.3, §3.4)
- [x] `task plan` rejects a blank or whitespace-only title with `checkTitle`, as `task register` does; update
      UC-11 and I §8.3 to match

### Stage 5.3 — Move field checks and ownership (domain, no Git) `[x]`

Decisions (author, 2026-10-21): the I §8.3 checks that need no Git split from
those that do (stage 5.4); `Record.Move` takes the installed roles as `[]string` (architecture §5, §6), passed by
`cli` in stage 5.5; the `--next` length check sits in the shared parser, so `task register` gets it too; `--spec` must be in clean form (`./a`, `a//b`, `a/`, `a/../b` are
rejected); with no installed roles every `--agent` is rejected; `CheckOwner(id, as)` returns `owner_mismatch`
with the UC-14 texts. I §8.3 updated to match.

- [x] `owner_mismatch`: the acting owner differs from the record's owner; check, found, and fix as in UC-14
- [x] `--agent`: ID `[a-z]+-[0-9]+` (`validAgentID`), role among the installed roles, never `orchestrator`
      (`invalid_value`, I §8.3)
- [x] `--next`: action 1 to 120 characters; actor `user`, the owner, or an `active_agents` ID after the move
      (`invalid_value`, I §8.3)
- [x] `--spec`: a relative path under the worktree, no `..` (`invalid_value`, I §8.3)

### Stage 5.4 — Git fact checks (domain) `[x]`

Decision (author, 2026-10-21): the stage move `setup` → `specification` is `branch_missing` without the branch
and `worktree_missing` (exit 3, fix `git worktree add ../<repo>-worktrees/<ID> feature/<ID>`) with the branch but
no worktree; I §8.4 gains the code.

Decisions (author, 2026-10-21): a `--revision` Git does not know is
`revision_not_on_branch`; the spec existence check runs when `--spec` is given or the move enters
`implementation`, and the spec must be a regular file in the worktree or a blob at the recorded `revision`; the
check returns where the spec was found (the worktree path or the short revision) for the UC-13 line; the UC-14
`revision_not_full_sha` fix becomes `git rev-parse <short>`, as I §8.4 states, since the object database is
shared by every worktree; the revision lookup needs a blob (`git cat-file -t`), so a directory does not count;
any `os.Stat` error means the spec is not in the worktree; the `revision_not_on_branch` fix names the real
worktree path; `spec_not_found` found names where it looked. I §3.4, §3.6, §8.3, §8.4, UC-13, UC-14 updated to
match.

- [x] `branch_missing`, `revision_not_on_branch` for `--revision` (I §8.3)
- [x] `branch_missing`, `worktree_missing` for `setup` → `specification` (I §8.2, §8.3)
- [x] `spec_not_found` through `git worktree list --porcelain` or the recorded `revision` (I §8.3)

### Stage 5.5 — `Registry.MoveTask` and the move commit (domain) `[x]`

Decisions (author, 2026-10-21): the move subject lists every changed field in record order (`status running,
stage implementation, spec <path>, agents impl-1, next impl-1`), as I §8.5 states, and the UC-13 examples follow
it; a move that changes nothing writes no commit and exits 0.

Decisions (author, 2026-10-21): `task move` splits into this domain stage and the
CLI stage 5.6; `MoveTask` runs lookup, `done_final`, ownership, `Record.Move`, then the fact checks; `Store.Update`
writes nothing when `fn` returns an empty subject and returns the current tip unpushed; subject parts are `revision
<short>`, `tier <n>`, `agents <id> <id>` or `agents none`, `pr <value>`, `next <actor>`; resuming a cancelled task
warns `cancelled_resumed` with the UC-18 text, when the status leaves `cancelled`; a move that changes nothing
still runs every check; `done_final` found is `done -> <status>`, or `done -> done` without `--status`. UC-13,
UC-18, I §8.4, I §8.5, and architecture §7 updated to match.

- [x] `Registry.MoveTask`: lookup, `done_final`, `owner_mismatch`, `Record.Move`, `CheckFacts`; returns from, to,
      changed, where the spec was found, the subject, and warnings (I §7 step 3 of the architecture, UC-13, UC-18)
- [x] Commit subject of I §8.5 from the changed fields; empty when nothing changed
- [x] `Store.Update` skips the commit for an empty subject
- [x] `cancelled_resumed` warning on `--resume-cancelled` (UC-18)

- Stage 5.6 `[ ]` — `task move` command: `--as` and `SOLOMONAR_OWNER` (`owner_required`), installed roles from
  the embedded core agents, UC-13 and UC-18 output, every UC-14 rejection; rejections name the ID (`task move
  W-001 rejected`) through `rejectedAs`, as I §3.6 states

## Step 6: `block`, `unblock`, `handoff`, `done`

- Stage 6.1 `[ ]` — `task block`, `task unblock`, prior status from the stage (UC-15)
- Stage 6.2 `[ ]` — `task handoff`, `task done` with merge and `spec_ref` checks (UC-19, UC-20)

## Step 7: `task status`

- Stage 7.1 `[ ]` — I §9.1 facts, optional `gh`
- Stage 7.2 `[ ]` — Drift, `allowed_next`, rendering (UC-16 status, UC-17)

## Step 8: `init`, non-interactive, core only

- Stage 8.1 `[ ]` — Manifest model, strict decoding, hashes
- Stage 8.2 `[ ]` — `Plan`/`Apply` with journal and rollback
- Stage 8.3 `[ ]` — Core file plan, `AGENTS.md` and `docs/README.md` templates, adoption (CLI-D17)
- Stage 8.4 `[ ]` — `init` command, registry creation after file apply, `--keep`/`--overwrite`/`--dry-run`,
  F-02/F-09 (UC-2, UC-3, UC-4); outside a repository `init` keeps its own `not_a_repository` fix from UC-4

## Step 9: `init` with runtimes and catalog extensions

- Stage 9.1 `[ ]` — Adapter stubs, `.claude/settings.json` merge
- Stage 9.2 `[ ]` — Extension rendering, `requires`/`conflicts`, `P/D-n` entries, `docs/stack.md`

## Step 10: Interactive `init`

- Stage 10.1 `[ ]` — Line prompts and multi-select (I §3.7)
- Stage 10.2 `[ ]` — Interactive `init`, `AGENTS.md` adoption, `--git-init`, `interactive_required` (UC-1, UC-4)
