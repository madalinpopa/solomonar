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

### Stage 5.6 — `task move` command `[x]`

Decisions (author, 2026-10-21): `--as` falls back to `SOLOMONAR_OWNER`, otherwise
`owner_required` (exit 2, message `task move needs an acting owner`, fix `pass --as <owner>`); a missing ID is
`invalid_usage`, and so is a move without a field flag (UC-13); the installed roles are the embedded core agents
(`framework/agents/*.md`) without `orchestrator` until the manifest exists (step 8); the header is `<ID>
<from> → <to>`, or `<ID>  <status>/<stage>` when the position does not change; detail lines follow for every
changed field other than status and stage, in record order, labels aligned: `spec <path> (found in <worktree>)`
or `(found at <short>)`, `revision <sha> (on feature/<ID>)`, `tier <n>: <meaning>` (0 `no review stages`, 1
`specialists only`, 2 `specialists, then adversarial`), `agents <id> <role>, …` or `none`, `pr <value>`, `next
<actor>: <action>`; a move to `cancelled` adds `note: branch feature/<ID>, its worktree, and PR are kept untouched
(rules §10)`, without `, and PR` when none is recorded; then the subject and the registry line; `--dry-run` prints
`Would move ` before the header, the details, and the subject; `--json` data is `{"id", "from", "to", "changed",
"record", "commit", "pushed"}`, or with `"dry_run": true` instead of `commit` and `pushed`; usage checks run before the roles are read and the store opens; the roles are read with
`fs.ReadDir`, a read error is `unexpected_error`; an unchanged move prints only its header; the cancel note sits
before the subject. UC-13, UC-14, UC-18, I §3.1, I §8.3 updated to match.

- [x] `task move <ID>` in the command table and help; flags of UC-13; `--as` and `SOLOMONAR_OWNER`
      (`owner_required`); roles from the embedded core agents
- [x] Human output of UC-13 and UC-18, the unchanged form, `--dry-run`, JSON, warnings
- [x] Every UC-14 rejection end to end, named `task move <ID> rejected` (I §3.6)

## Step 6: `block`, `unblock`, `handoff`, `done`

### Stage 6.1 — `BlockTask`, `UnblockTask` (domain) `[x]`

Decision (author, 2026-10-21): unblocking the last blocker of a `blocked` task without `--status` is
`blockers_required` (exit 3, fix `pass --status <prior status>`); the CLI never picks the status.

Decisions (author, 2026-10-21): `task block` and `task unblock` split into this
domain stage and the CLI stage 6.2; both run lookup, `done_final`, ownership, then the blocker change and the
`--status`/`--next` shorthand through `Record.Move` on the same copy, and return a `MoveResult`; `changed` lists
`blockers` (not `last_blocker`, which is bookkeeping) in record order with `status` and `next_action`; the subject
is always `block B-<n> <kind>` or `unblock B-<n>` (I §8.5); an unknown or malformed blocker ID is
`blocker_not_found`; block checks kind, summary, and ref, then `--status`, then the move rules; a successful
block or unblock always sets `updated_at`; `--status blocked` on the last unblock gets the same rejection;
`changed` and the move subject follow the record order exactly (`next_action` before `pr`, stage 3.4). UC-13,
UC-15, I §8.4, I §8.5 updated to match.

- [x] `BlockTask`: new blocker `B-<n>` from `last_blocker`, `--kind`, `--summary`, `--ref` checks, optional
      `--status blocked` and `--next` in the same write (UC-15, rules §7)
- [x] `UnblockTask`: remove the blocker, `blocker_not_found`, optional `--status <prior>` and `--next`; the
      last-blocker rule above (UC-15, I §8.1)
- [x] Subjects `block B-<n> <kind>` and `unblock B-<n>` (I §8.5)

### Stage 6.2 — `task block`, `task unblock` commands `[x]`

Decisions (author, 2026-10-21): both resolve `--as` like `task move` (`owner_required`
names the command); usage errors before the store opens: `task ID is required`, `blocker ID is required` (unblock),
`--kind is required` and `--summary is required` (block); rejections read `task block <ID> rejected` and `task
unblock <ID> rejected`; the human form is the `task move` form with a first detail line `blocker B-<n> <kind>:
<summary> (<ref>)` (no parenthetical without a ref) or `cleared B-<n> <kind>: <summary>`, then the other changed
fields, labels aligned; `--dry-run` prints `Would block ` or `Would unblock ` before the header; `--json` has the
`task move` shape; `actingOwner` and `changeTask` are shared with `task move`; `invalid_value` stays exit 2
(I §8.4). UC-15 updated to match.

- [x] `task block <ID>` and `task unblock <ID> <blocker-id>` in the command table and help, flags of UC-15
- [x] UC-15 human output, `--dry-run`, JSON, rejections end to end

### Stage 6.3 — `task handoff` (UC-19) `[x]`

Decisions (author, 2026-10-21): one stage for domain and command (small domain part); a `planned` task is
`not_registered` (exit 3, check `the task is registered`, found `<ID> is planned`, fix `solomonar task register
<ID> --title <title> --owner <new-owner>`); without `--next`, a `next_action` whose actor is the old owner is
retargeted to the new owner with the same action text, and `changed` lists `next_action`.

Decisions (author, 2026-10-21): check order lookup (`task_not_found`), `done_final`,
`not_registered`, `--owner` format (`invalid_value`, the `task register` wording), `handoff_same_owner` (found `<ID>
is owned by <owner>`, no fix), `owner_in_use` (the `task register` check), then `--next` (actor `user`, the new
owner, or an `active_agents` id; I §8.3); `SOLOMONAR_OWNER` is ignored; usage errors `task ID is required` and
`--owner is required` before the store opens; rejections read `task handoff <ID> rejected`; the subject is always
`handoff <old> -> <new>`; `changed` lists `owner` in record order (after `stage`) and `next_action` when it changes;
a handoff always sets `updated_at` and keeps `active_agents`; every status except `planned` and `done` accepts a
handoff; human output is the UC-19 form with a `next` detail line when `next_action` changes; `--dry-run` prints
`Would hand off ` before the header, then the detail lines and the subject, without the registry line and without
the closing `The previous owner must stop writing this record.`; `--json` has the `task move` shape; `task
handoff` sits between `task block` and `task move` in the alphabetical command table and help; `Record.Move` runs
only with `--next`, the retarget is applied directly; `changeTask` takes the header from the caller
(`positionHeader` for move, block, unblock). UC-19, I §8.4, I §8.5 updated to match.

- [x] `Registry.HandoffTask`: checks above, owner change, `next_action` retarget or `--next`, `MoveResult`
      (UC-19, I §8.4, rules §10)
- [x] `task handoff <ID> --owner <o> [--next] [--json] [--dry-run]` in the command table and help
- [x] UC-19 human output, `--dry-run`, JSON, rejections end to end

### Stage 6.4 — `Registry.DoneTask` with merge and `spec_ref` checks (domain) `[x]`

Decisions (author, 2026-10-21): `task done` splits into this domain stage and the CLI stage 6.5 (the domain stage
may overrun the budget moderately); the `--spec-ref` path must equal the recorded `spec`, otherwise
`spec_ref_not_final`; `--summary` takes 1 to 120 characters (runes) after trimming, else `invalid_value`.

Decisions (author, 2026-10-21): check order lookup, `done_final`, ownership, status
`running` and stage `integration` (`transition_not_allowed`, check `task done needs running/integration`, found
`<ID> is <status>/<stage>`, fix `solomonar task status <ID>`), `blockers_open` (found `<ID> has open blockers B-1,
B-2`, fix `solomonar task unblock <ID> <blocker-id>`), then flag values (`invalid_value`, exit 2): `--summary`,
`--merge` a full SHA, `--spec-ref` `<full SHA>:<clean path>`; then Git: `merge_not_on_main` (`git merge-base
--is-ancestor <merge> <main>` against the local main branch; an unknown SHA fails too), `merge_not_merge_commit`
(fewer than two parents), `revision_not_merged`, `spec_ref_invalid` (`git cat-file -e <spec_ref>`),
`spec_ref_not_final` (the last commit reachable from the merge's second parent whose subject starts with
`spec(<ID>):`, compared with the `--spec-ref` commit, and the path compared with the recorded `spec`; fix `pass
--spec-ref <derived>` with `<derived>` = `<that commit>:<recorded spec>`; no such commit: found `no spec(<ID>)
commit is reachable from <merge short 8>^2`, no fix); the stored summary is trimmed; the compact record keeps
`title`, `status` `done`, `stage` `complete`, `summary`, `spec_ref`, `merge_commit`, `pr` when recorded,
`related_task` when recorded, and `updated_at`; subject `merged <short 7>` (I §8.5); `changed` is left to stage 6.5;
a root commit as `--merge` reads `<merge short 8> has no parent`; Git exits 1 and 128 of `merge-base --is-ancestor`
and `cat-file -e` fail the check unless cancelled, other Git failures are `unexpected_error`; the last `spec(<ID>):`
commit is the first in `git log --topo-order --format="%H %s" <merge>^2`; the `--summary` found shows the raw value;
the `--merge` fix is `git rev-parse <value>`. UC-20, I §8.3, I §8.4, I §9.1, architecture §2 updated to match.

- [x] Merge checks: `merge_not_on_main`, `merge_not_merge_commit`, `revision_not_merged` (I §8.3), tested on a
      merge commit, a squash, and a rebased branch (architecture §12 step 6)
- [x] `spec_ref` checks: `spec_ref_invalid`, `spec_ref_not_final` with the derived `spec_ref` (I §8.3, C14)
- [x] `Registry.DoneTask`: state and flag checks above, compact record, subject (UC-20, rules §7, §9)

### Stage 6.5 — `task done` command (UC-20) `[x]`

Decisions (author, 2026-10-21): JSON `changed` lists every record field that differs between the record before and
the compact record (added, changed, or removed), in record order; the human `Compact record` block is the record
as indented JSON (two spaces, record field order), each line prefixed by two spaces.

Decisions (author, 2026-10-21): `--as` resolves like `task move`; usage errors before
the store opens, in this order: `task ID is required`, `--merge is required`, `--spec-ref is required`, `--summary
is required`; rejections read `task done <ID> rejected`; the human form is the UC-20 example: `Verified` with the
lines `merge     <m8> is a merge commit reachable from <main>`, `revision  <r8> is contained in the merge`,
`spec_ref  resolves; <s8> is the last spec(<ID>) commit merged`, then the `task move` header, the subject, the
registry line, `Compact record`, and, when a worktree has `feature/<ID>` checked out, `Next: delete the worktree
only after confirming it holds no uncommitted or unmerged work:` and `  git worktree remove <path>` (the
conventional sibling path in the `task register` form, otherwise the worktree's absolute path); `--dry-run` prints
`Verified`, `Would complete ` before the header, the subject, and `Compact record`, without the registry line and
the `Next:` lines; `--json` has the `task move` shape (`record` is the compact record), with the `changed` above; `changed` leaves out `last_blocker` and `updated_at` like the other
commands; a `done` record gets no detail lines; the worktree lookup for `Next:` runs before the write; `task.Worktree`
exports the worktree lookup. UC-20 updated to match.

- [x] `task done <ID> --merge --spec-ref --summary [--as] [--json] [--dry-run]` in the command table and help
- [x] UC-20 human output, `--dry-run`, JSON, rejections end to end, `merge_not_merge_commit` on a squash

## Step 7: `task status`

Decision (author, 2026-10-21): three stages: 7.1 Git and forge facts (may overrun the budget moderately), 7.2 drift
and `allowed_next` (domain), 7.3 the `task status` command.

### Stage 7.1 — Git and forge facts (domain) `[x]`

Decisions (author, 2026-10-21): `checks` is `failing` when any check concluded `FAILURE`, `CANCELLED`, `TIMED_OUT`,
`ACTION_REQUIRED`, `ERROR`, or `STARTUP_FAILURE`, else `pending` when any is not completed, else `passing` when all
are `SUCCESS`, `NEUTRAL`, or `SKIPPED`, `none` for an empty rollup, and `unknown` without a PR or without `gh` (J6);
`forge_unavailable` (warning, exit 0) when `gh` is not on `PATH` or `gh pr view` fails, the forge facts then
`unknown`; `gh` reporting no PR for the branch is not a failure (pr `none`); `--no-forge` gives no warning.

Decisions (author, 2026-10-21): the facts live in `internal/task` as one
`StatusFacts` value whose JSON is the I §9.3 `facts` object; `gh` runs through `internal/git` (AD-3) with the `-v`
trace `+ gh …`; the branch object adds `contains_revision` (`git merge-base --is-ancestor <revision>
feature/<ID>`, null without a revision or branch) for the drift `revision_not_on_branch`; `worktree` is null
without a worktree, `ahead_of_revision` null without a revision, `dirty` `[]` when clean; dirty states from `git
status --porcelain=v2`: `modified`, `added`, `deleted`, `type_changed` (worktree side, else index side), `renamed`
or `copied` (entries `2`), `untracked` (`?`), `unmerged` (`u`), ignored entries skipped; without a revision,
`merged` is `false` (I §9.1); a `done` record's merge facts come from its `merge_commit` and `spec_ref`;
`gh` "no pull requests found" is detected from its stderr; `forge` is `{"available": false, "pr": null, "checks":
"unknown"}` without `gh` or with `--no-forge`, and `pr` is `{"number", "url", "state", "merge_commit"}` otherwise;
the function is `Forge` (the type is `ForgeFacts`), `NoForge()` the `--no-forge` value; a revision Git does not know
(exit 128) gives `ahead_of_revision` null and `merged`, `contains_revision`, `at_revision` false; the PR `state`
passes through as `gh` prints it; `checks` precedence `failing` > `pending` > `unknown` > `passing`, an entry
without `status` and `state` is unknown; a `2` entry with `C` in XY is `copied`, else `renamed`, paths as Git
prints them, Git's order; `forge_unavailable` messages `gh unavailable: gh not found`, `gh unavailable: <first
stderr line>`, `gh unavailable: exit status <n>`, `gh unavailable: unreadable gh pr view output`; a cancelled `gh`
is `interrupted`; `merge_commit` and `derived_spec_ref` are null while `merged` is false; `git.ErrNotFound`,
`Error.Program`, `RunGH`, `gittest.FakeGH` (a `PATH` holding only the fake `gh`). I §3.1, I §8.4, I §9.1, I §9.3,
architecture §8 updated to match.

- [x] Git facts of I §9.1: branch and tip, `contains_revision`, worktree, HEAD vs `revision`, dirty paths, spec in
      the worktree and at the revision, merged and the derived `spec_ref`
- [x] Forge facts: `gh pr view feature/<ID> --json number,url,state,mergeCommit,statusCheckRollup`, the `checks`
      summary, `forge_unavailable`, `--no-forge`; tests with a fake `gh` on `PATH` (architecture §11)

### Stage 7.2 — Drift and `allowed_next` (domain) `[x]`

Decisions (author, 2026-10-21): `allowed_next` follows a per-status rule, every command with `--as <owner>` from the
record, never `task handoff` (user instruction only): `planned` → `task register <ID> --title <title> --owner
<owner>`; `reserved` (stage `setup`) → `task move --status running --stage specification`, block, pause, cancel;
`running` → the stage moves of I §8.2 allowed by the tier in stage order (back moves with `--next
<actor>:<action>`, required flags that are not recorded as placeholders: `--spec <path>`, `--revision <full-sha>`,
`--tier <n>`), at `integration` `task done`, then block, pause, cancel; `blocked` → `task unblock <ID> <B-n>` per
open blocker (the last one with `--status <prior>`), block, cancel; `paused` → `task move --status <prior>`, cancel;
`cancelled` → `task move --status <prior> --resume-cancelled`; `done` → none. `task done` fills `--merge` with the
merge commit when merged and keeps `--spec-ref <PR "Final specification" line>` (C14 a; I §9.3 corrected); when the
branch is merged and the record is not done, `allowed_next` lists only `task done`.

Decisions (author, 2026-10-21): drift entries are `{"code", "message"}` with the
messages below; the I §9.2 suggested commands become `allowed_next` entries or stay as guidance in the table, not
in the output; drift applies to `reserved`, `running`, `blocked`, `paused`; `planned`, `cancelled`, and `done` get
only `owned_by_other` when they have an owner; drift order follows the I §9.2 table. Messages:
`branch_missing` `feature/<ID> does not exist` (status `running` past `setup`); `worktree_missing` `feature/<ID> has
no worktree`; `head_ahead_of_revision` `HEAD is <n> commit(s) past the recorded revision`;
`revision_not_on_branch` `<revision short 8> is not on feature/<ID>` (`contains_revision` false);
`dirty_worktree` `<n> uncommitted path(s); preserve them (rules §10)`; `spec_missing` `<spec> is not in the
worktree or at the revision` (both spec facts false); `blocked_without_blockers` `blocked without an open blocker`;
`tier_missing` `stage <stage> needs a tier`; `merged_not_done` `feature/<ID> is merged into <main> but the record is
not done`; `pr_not_recorded` `gh shows PR #<n> for feature/<ID>; the record has none`; `pr_mismatch` `the record
has PR <pr>; gh shows #<n> for feature/<ID>` (a recorded number or URL that matches the PR's number or URL is no
mismatch); `owned_by_other` `SOLOMONAR_OWNER is <x>; <ID> is owned by <owner>`. Placeholders are written
`<actor>:<action>`, `<kind>`, `<text>`, `<path>`, `<full-sha>`, `<n>`, `<prior>` resolved to the prior status,
`<one sentence>`; a blocker ID and the prior status are filled from the record; `task unblock <ID> --as <owner>
<B-n>` (`--as` after the ID, as UC-15), `--status <prior>` only on the single open blocker's line; `blocked` without
an open blocker lists block and cancel; a missing tier at `specialized_review` or later lists only back moves;
`--spec <path>` only on the forward move into `implementation`; `integration` without a tier gets `--tier 0`;
`--merge` filled only when the merge commit is known; the function is `DriftOf` (the type is `Drift`);
`compareStages` is shared with `transition.go`. I §9.2, I §9.3, UC-16 updated to match.

- [x] `DriftOf(id, main, record, facts, envOwner)`: the I §9.2 rows with the messages above
- [x] `AllowedNext(id, record, facts)`: the rule above, as command strings (I §9.3)
### Stage 7.3 — `task status` command (UC-16 status, UC-17) `[x]`

Decisions (author, 2026-10-21): usage `solomonar task status <ID> [--json]
[--no-forge]`; `task ID is required`; an unknown ID is `task_not_found` as on `task show <ID>`; `SOLOMONAR_OWNER`
feeds `owned_by_other`; exit 0 whatever the drift; `forge_unavailable` goes to stderr as a warning; `--json` data
is the I §9.3 shape `{"id", "record", "registry_commit", "facts", "drift", "allowed_next"}`. Human form, sections
`record`, `git`, `forge (gh)`, `drift`, `allowed next`, two-space indent, labels padded to the widest label of the
section:
- header `<ID>  <title>`;
- `record`: `status    <status> / <stage>   owner <owner>   tier <tier>` (`-` for a missing stage or owner,
  `tier none` without a tier), then always `spec`, `revision`, `agents` (`<id> <role>, …` followed by `(last
  reported, not liveness)`), `blockers` (`B-n <kind>: <summary> (<ref>)` joined by `; `), `next` (`<actor>:
  <action>`), `pr`, each `none` when empty, and `updated   <updated_at> (registry @ <sha7>)`; a `done` record shows
  `status`, `summary`, `spec_ref`, `merge`, `pr`, `updated` instead;
- `git`: `branch    feature/<ID> @ <tip>` or `feature/<ID> missing`; `worktree  <path>` or `missing`; `HEAD
  <head8> · <n> commit(s) ahead of revision` (`<head8>` alone without a revision, `unknown (no worktree)` without a
  worktree); `dirty     clean`, `1 path: <p> (<state>)`, `<n> paths: <p> (<state>), …`, or `unknown (no
  worktree)`; `spec      present in worktree and at revision`, `present in worktree only`, `present at revision
  only`, `missing`, or `none recorded`; `merged    no` or `yes, by <merge sha> (merge commit on <main>)`;
  `spec_ref  derived <spec_ref> (last spec(<ID>) commit merged)` only when derived;
- `forge (gh)`: `pr        #<n> <state lowercased>`, `none for feature/<ID>`, `unknown (gh unavailable)`, or
  `unknown (--no-forge)`; `checks    <value>`, with `unknown (no PR)` when there is no PR;
- `drift`: `<code>  <message>` with codes padded, or `none`;
- `allowed next`: one command per line, unwrapped, or `none`.
UC-16 and UC-17 examples aligned to this form. Also: labels are padded per section (`forge (gh)` pads to
`checks`); a `done` record's `git` section omits the derived `spec_ref` line; empty `agents` reads `none` without
the liveness suffix; a blocker without a ref omits ` (<ref>)`; `HEAD` shows `<head8>` alone when the revision is
unknown or not recorded; a `done` status line keeps the format (`done / complete   owner -   tier none`);
`writeDetails` is shared with `task move`; `gittest.NoGH` puts only a `git` wrapper on `PATH` and `FakeGH` adds the
fake `gh` there. UC-16, UC-17 updated to match.

- [x] `task status <ID> [--json] [--no-forge]` in the command table and help; facts, forge, drift, `allowed_next`
- [x] UC-16 status and UC-17 human output and JSON end to end; fake `gh` and `--no-forge`

## Step 8: `init`, non-interactive, core only

### Stage 8.1 — Manifest model, strict decoding, hashes (`internal/project`) `[x]`

Decisions (author, 2026-10-21): a new package `internal/project` (architecture §5)
holds `Manifest` with the I §4.7 fields and JSON keys; `DecodeManifest(data, framework)` reads `schema_version` and
`framework_version` first, and a newer one than this CLI (`ManifestSchemaVersion = 1`, `framework` the embedded
framework version, compared as `major.minor.patch`) is `cli_outdated` with the registry's texts, before decoding
strictly (`encoding/json/v2`, unknown members and duplicate keys rejected; CLI-D19, AD-7); every other failure is
`manifest_invalid` (exit 1, like `registry_invalid`; fix `inspect .solomonar/manifest.json; never edit it by hand`)
naming the field; `Encode` writes struct order, sorted map keys, two-space indent, trailing newline; validation:
`project.name`, `main_branch`, `remote` non-empty; runtime keys `claude` or `codex`, value `{"adversarial_model"?}`;
extension names `[a-z][a-z0-9-]*`, `state` `installed` or `draft`, `kind` `skill` or `agent`, `group` one of the
nine I §4.3 groups, `source` `catalog` or `path` (with a clean repo-relative `path`, which only `path` sources
carry), an installed stack-group extension has `decision` `P/D-<n>`, a `workflow`/`review` or draft extension has
none, drafts have no `rendered_with`, `rendered_with` names known extensions; `files` keys are clean repo-relative
paths, `owner` one of `core`, `ext:<name>`, `project-ext:<name>`, `decision`, `core+notes`, `runtime:<name>`,
`seed`, `manifest` with `<name>` a known extension or runtime; `sha256` (64 lowercase hex) is required for `core`,
`ext:`, `project-ext:`, `core+notes`, and `runtime:` files except a runtime `entry` file, which has `entry` instead;
`decision` files have `entries` (`P/D-<n>`), `seed` and `manifest` files have neither; `Hash(content)` is the
lowercase hex SHA-256; `ReadManifest(root *os.Root, framework)` returns nil without error when
`.solomonar/manifest.json` is absent; `cli_outdated` keeps the fix `install a newer solomonar release` for the
registry and the manifest (specs aligned); `project.Rejection` mirrors `task.Rejection` and the CLI maps both, with
`manifest_invalid` exit 1 like `registry_invalid`; `project` keeps its own clean-path check; draft lists use
`omitzero`; `runtimes`, `extensions`, `files` must be present; `schema_version` is checked before
`framework_version`, numerically per part; owner sources, `requires`/`conflicts` names, duplicate `entries`, and an
empty `adversarial_model` are not validated. I §3.6, I §4.7, I §8.4 updated to match.

- [x] `Manifest` types, `DecodeManifest` with the version check first, `Encode` (I §4.7, AD-7, CLI-D19)
- [x] Validation above, `manifest_invalid` texts, `Hash`, `ReadManifest`

Decision (orchestrator, 2026-10-21, split further per the plan's rule): 8.2 apply and journal, 8.3 plan
classification, 8.4 core file set and templates, 8.5 the `init` command.

### Stage 8.2 — `Apply` with journal and rollback (`internal/project`) `[x]`

Decisions (author, 2026-10-21): `Apply(ctx, root, plan) (*Journal, error)` instead of
the architecture's `Apply(root, p) error`, because rollback must stay available until the registry CAS (AD-6,
F-09); it writes only `create`, `update`, and `remove` changes, in plan order; each write is a temp file in the
target's directory (`.<name>.solomonar-tmp`) renamed over the target; an update keeps the target's mode, a new file
gets `0o644`; missing parent directories are created (`0o755`) and journaled; the journal holds each touched path's
previous content and mode, or its absence; a failed or cancelled `Apply` (context checked before each change) rolls
back what it applied and returns the error; `Journal.Rollback()` restores in reverse order, deletes created files and
the directories it created when empty, and is safe to call twice; every path goes through the `os.Root` (I §4.5,
architecture §8); `Plan.Conflicts()` returns the `conflict` changes in plan order; a plan that disagrees with the
disk (`create` on an existing path, `update`/`remove` on a missing or non-regular one) is an error that rolls back;
errors read `apply <path>: %w`, a failed rollback is joined, a failed `Apply` returns no journal; the written file
gets its exact permission bits regardless of the umask; parent directories are created one level at a time, each
journaled; entries that fail to restore stay in the journal for the next `Rollback`. Architecture §6, §8 updated to
match.

- [x] `Action`, `Change`, `Plan`, `Conflicts` (architecture §6)
- [x] `Apply` with temp + rename, modes, directories, the journal, rollback on failure or cancellation
- [x] `Journal.Rollback`; tests including a failing write midway and a rollback after a successful apply (F-09)

### Stage 8.3 — Plan classification (`internal/project`) `[x]`

Decision (author, 2026-10-21): a managed file modified locally (disk differs from the manifest base) while the
offered content equals that base is `keep`, not a conflict (F-02: a repeated offer keeps the edit); it is a conflict
only when the offered content also differs from the base; the UC-3 example is aligned (the conflict follows a
framework change).

Decisions (author, 2026-10-21): `Classify(root, manifest, wants, keep, overwrite)`
returns the `Plan` and the manifest `files` entries for the wanted paths; a `Want` is a path, an owner, and the
offered content; the recorded base is always the hash of the offered content (I §4.5); rules, per path, with `disk`
the current file, `base` the manifest `sha256`, `offer` the offered content:
- `seed` owner: absent → `create`; present → `skip` (project-owned), no hash recorded;
- absent → `create`;
- not a regular file (directory, symlink) → `conflict`, reason `exists and is not a regular file`;
- managed (`base` recorded): `disk` = `offer` → `unchanged`; `disk` = `base` → `update`; `offer` = `base` → `keep`;
  otherwise `conflict`, reason `modified since installation (manifest sha256 differs)`;
- unmanaged: `disk` = `offer` → `unchanged`, reason `adopted` (CLI-D17); otherwise `conflict`, reason `exists and
  is not managed by solomonar`;
- a conflict on a path given to `--keep` becomes `keep`, to `--overwrite` becomes `update`; both record the offered
  base;
- `--keep`/`--overwrite` naming a path that is not a conflict, or the same path in both, is `invalid_value` (exit 2):
  check `--keep and --overwrite resolve conflicts`, found `<path> is not a conflict` or `<path> is in both --keep and
  --overwrite`, fix `pass --keep or --overwrite once per conflicting path`;
- the plan lists the wanted paths in the given order; `core+notes` (`AGENTS.md` managed part) is handled in 8.4.
- also: only conflicts and `adopted` carry a reason; a resolved conflict is a plain `keep` or `update`; `Content` is
  set for `create` and `update` only; entries for unresolved conflicts carry the offered base and nothing is written
  while the plan has a conflict; the both-flags check runs first, then `--keep` and `--overwrite` paths in order (a
  path the plan does not list is "not a conflict"); a present `seed` is `skip` even when not regular; managed `disk` =
  `offer` is tested first; an inspection failure other than absence is an unexpected error. I §3.3, I §4.5, UC-3,
  architecture §6 updated to match.

- [x] `Want`, `Classify` with the rules above, the manifest entries, `--keep`/`--overwrite`
- [x] Tests for every row, F-02 (offered A, local B, keep, offer C, keep B, offer C again, offer D), CLI-D17

### Stage 8.4 — Core file set, templates, `AGENTS.md` adoption (`internal/project`) `[x]`

Decisions (author, 2026-10-21): `CoreWants(assets fs.FS, c Choices) ([]Want, error)`
with `Choices{Name, Objective, FrameworkVersion}` returns, in this order: every file under `framework/` copied to
`docs/solomonar/` (owner `core`, path order sorted), `AGENTS.md` (owner `core+notes`) rendered from
`templates/AGENTS.md`, and `docs/README.md` (owner `seed`) rendered from `templates/docs/README.md`, with
`text/template` (`FrameworkVersion`, `Extensions`, `ProjectName`, `Objective`); with no extensions the
`{{ .Extensions }}` block renders `- Skills: none.` and `- Agents: none. An installed agent is a specialist reviewer,
required by triage when its "applies when" matches.` (I §4.6); the manifest itself is added by `init` (8.5).
`AGENTS.md` (I §4.5 `core+notes`): the managed part is everything before the line `## Project notes`, the notes
part is that line and everything after it; only the managed part is hashed and classified by the I §4.5 rules, and
a written file is the offered managed part followed by the file's own notes part (the template's when the file has
none); an unmanaged `AGENTS.md` whose content differs from the offer is adopted (UC-4, default with `--yes`): action
`update`, reason `adopted; existing content moved under "## Project notes"`, written as the offered managed part,
`## Project notes`, a blank line, and the old content unchanged; `--overwrite AGENTS.md` writes the template and
discards the old content; `--keep AGENTS.md` on an unmanaged file is `invalid_value`
(the framework needs its `AGENTS.md`), check `--keep and --overwrite resolve conflicts`, found `AGENTS.md cannot be
kept`, fix `adopt it (default) or pass --overwrite AGENTS.md`. Author decision: `--overwrite AGENTS.md` on a
managed conflict replaces only the managed part and keeps the file's notes; only an unmanaged file's overwrite
writes the full template. Also: the split is at the first line exactly `## Project notes`; an unmanaged file is
compared on its managed part (equal → `unchanged (adopted)` whatever its notes); a non-regular `AGENTS.md` keeps the
generic conflict; framework files are sorted by target path; render errors read `render <name>: %w`. I §4.5, I §4.6,
UC-4, architecture §6 updated to match.

- [x] `Choices`, `CoreWants`, template rendering, the empty extensions block (I §4.1, I §4.6)
- [x] `AGENTS.md` managed part, notes preservation, adoption, `--keep`/`--overwrite` (I §4.5, UC-4)
Decision (orchestrator, 2026-10-21, split further per the plan's rule): 8.5 the init plan and manifest (domain),
8.6 the `init` command, 8.7 the UC-4 edge cases, 8.8 task commands read the manifest. Author decisions
(2026-10-21): until steps 9 and 10, `--runtime` and `--ext` accept only `none` (another value is
`unknown_runtime`/`unknown_extension`, exit 2), both default to none, and without `--yes` a writing plan is
`interactive_required` even on a terminal; these step-8 limitations are recorded here only, the specs keep the final
behavior; a `file_conflict` prints the conflict lines of the plan on stdout before the error (`render` writes
`out.text` on failure too).

### Stage 8.5 — Init plan and manifest (`internal/project`) `[x]`

Decisions (author, 2026-10-21):
- `InitPlan(root *os.Root, existing *Manifest, wants []Want, keep, overwrite []string, p Project, framework string)
  (Plan, *Manifest, error)` runs `Classify` with `existing` (nil before the first init) and builds the manifest:
  `schema_version` 1, `framework_version` `framework`, `project` `p`, empty `runtimes` and `extensions`, `files` the
  `Classify` entries plus `.solomonar/manifest.json` `{"owner": "manifest"}` (I §4.5, I §4.7, AD-6);
- the plan is the `Classify` plan followed by one change for `.solomonar/manifest.json`, owner `manifest`, content
  the encoded manifest: `create` when absent, `unchanged` when the file's bytes equal the encoding, `update`
  otherwise; the manifest is never classified by the I §4.5 table (it carries no hash) and is built even when the plan
  has conflicts (nothing is written then);
- `(m *Manifest) CheckChoices(requested Requested) error` with `Requested{Name, MainBranch, Remote string; Runtimes,
  Extensions []string}` (empty string or nil list = not given) rejects the first differing choice, in the order
  name, main branch, remote, runtimes, extensions, as `already_initialized` (exit 3, UC-3), no check, found
  `manifest <field> <value>; requested <value>` (`name "ledger"`, `main_branch "main"`, `remote "origin"`; lists as
  `[go, tdd]` in sorted order, an empty list as `none`), fix `rerun without --name; init never changes choices` (same
  for `--main-branch`, `--remote`), `change runtimes with \`solomonar runtime add|remove\`; init never changes
  choices`, `change extensions with \`solomonar extension add|remove\`; init never changes choices`; the CLI fills
  unspecified choices from the manifest; `--objective` is not a recorded choice.
- also: requested runtimes and extensions compare as sets (sorted, duplicates dropped); a list given as `none` counts
  as given and is compared, only a nil list is not given; a nil manifest map counts as `none`; a
  `.solomonar/manifest.json` that cannot be read as a file (a directory) is `read manifest: …` (exit 1), not a
  conflict. I §4.5, UC-3, architecture §6 updated to match.

- [x] `InitPlan`: `Classify`, the manifest built from its entries, the manifest change (I §4.7, AD-6)
- [x] `CheckChoices` and `already_initialized` (UC-3)
- Stage 8.6 `[ ]` — `init` command: flags, check order, registry check and creation after file apply, rollback until
  the CAS (F-09), `--keep`/`--overwrite`/`--dry-run`/`--yes`, `interactive_required`, `file_conflict`,
  `registry_exists_invalid`, human and JSON output, `Next:` (UC-2, UC-3, non-interactive UC-4)
- Stage 8.7 `[ ]` — `--git-init` (moved from 10.2), the `not_a_repository` fix of UC-4, an empty repository's
  `Next:`, the `.jj/` warning (UC-4)
- Stage 8.8 `[ ]` — task commands read `main_branch` and `remote` from the manifest (architecture §7 step 2)

## Step 9: `init` with runtimes and catalog extensions

- Stage 9.1 `[ ]` — Adapter stubs, `.claude/settings.json` merge
- Stage 9.2 `[ ]` — Extension rendering, `requires`/`conflicts`, `P/D-n` entries, `docs/stack.md`

## Step 10: Interactive `init`

- Stage 10.1 `[ ]` — Line prompts and multi-select (I §3.7)
- Stage 10.2 `[ ]` — Interactive `init`, `AGENTS.md` adoption, the `git init` question, `interactive_required`
  (UC-1, UC-4)
