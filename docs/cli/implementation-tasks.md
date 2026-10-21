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

### Stage 8.6 — `init` command (`internal/cli`; UC-2, UC-3, non-interactive UC-4) `[x]`

Decisions (author, 2026-10-21):
- usage `solomonar init [<dir>] [--name <name>] [--objective <text>] [--runtime none] [--ext none] [--main-branch
  <name>] [--remote <name>] [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]`; `<dir>`
  resolves against `-C`; files are written at the worktree root found from it; `--runtime`/`--ext` are repeatable
  and comma-separated; `--adversarial-model` (9.1) and `--git-init` (8.7) are not registered yet;
- step-8 values: any `--runtime` value but `none` is `unknown_runtime`, any `--ext` value but `none` is
  `unknown_extension`, both exit 2, message `init rejected`, check `--runtime names an available runtime` / `--ext
  names an available extension`, found `"<v>" is not available; available: none`, fix `pass --runtime none` /
  `pass --ext none`; given `none` is an empty requested list, not given is nil;
- check order: usage and values; repository (`not_a_repository`, fix ``run `git init -b <main-branch>` first``,
  completed in 8.7), Git version; manifest (`cli_outdated`, `manifest_invalid`); `CheckChoices`
  (`already_initialized`), then unspecified name, main branch, remote come from the manifest, else the worktree root's
  base name, `main`, `origin`; the registry: absent → `create`, valid → `unchanged`, `registry_invalid` →
  `registry_exists_invalid` (exit 3, check `an existing workflow-state holds a valid registry`, the registry's found,
  fix `inspect with git show refs/heads/workflow-state:state.json; the CLI never overwrites a registry`),
  `cli_outdated` as is; `CoreWants` and `InitPlan` (`invalid_value` from `--keep`/`--overwrite`); conflicts
  (`file_conflict`); `--dry-run`; confirmation; `Apply`; `task.Store.Create` with the chosen remote when the
  registry is absent; on any failure before the CAS succeeds (a failed create, cancellation) `Journal.Rollback` and
  the error (`interrupted` 130/143 or `unexpected_error`), the rollback error joined; a failed push is the store's
  warning, exit 0 (F-09);
- `file_conflict` (exit 4): stdout carries `Plan` and one line per conflict; message `init found <n> conflict(s)`,
  check `no planned file conflicts with the project's files`, found `<n> conflict(s): <p1>, <p2>`, fix `rerun with
  --keep <path> to keep the project's file, or --overwrite <path> to replace it`; `render` writes `out.text` on
  failure too (JSON keeps `data` absent);
- confirmation: needed only when the plan creates or updates a file or creates the registry; without `--yes` that is
  `interactive_required` (exit 2), message `init needs a confirmation`, fix `pass --yes to accept the plan, or
  --dry-run to print it`;
- human output, plan lines `  <action padded to 9>  <path>` plus, when the change has a reason, two spaces and the
  reason with paths padded to the widest path that has a reason; the registry line `  registry   create
  refs/heads/workflow-state with an empty state.json` or `  registry   unchanged refs/heads/workflow-state`; counts
  `<n> file(s) <first action>, <n> <action>, …` over the non-zero actions in the order created, updated, kept,
  unchanged, skipped (the manifest counts);
  - success: `Initialized solomonar <fw> in <root>`, `  <counts>`, `  registry  refs/heads/workflow-state @ <sha>`
    (`… unchanged` when reused), `  pushed    <remote>/workflow-state` when pushed, a blank line, `Next:`, `  1. git add
    AGENTS.md .solomonar docs && git commit -m "chore: initialize solomonar <fw>"`, `  2. Start a top-level session in
    <root>:`, `       Use the solomonar skill to explore the next useful flow.`; warnings on stderr;
  - nothing to write and the registry reused: `Already initialized with solomonar <fw>; nothing to do.`, `  <counts> ·
    registry refs/heads/workflow-state unchanged`, ``  Change choices with `solomonar runtime|extension add|remove`;
    move versions with `solomonar upgrade`.``;
  - `--dry-run`: `Plan`, every file line, the registry line, then `Would initialize solomonar <fw> in <root>` and
    `  <counts>`;
- JSON data `{"project", "framework_version", "choices": {"runtimes": [], "extensions": {}, "adversarial_model": {}},
  "decisions": [], "files": [{"path", "action", "owner", "reason"?}], "registry": {"ref", "commit", "action",
  "pushed"}, "next": ["git add AGENTS.md .solomonar docs", "git commit -m \"chore: initialize solomonar <fw>\""]}`;
  `registry.commit` is the tip when unchanged; with `--dry-run` `"dry_run": true`, `registry` `{"ref", "action"}`,
  no `next`; a no-op has `next` `[]`.
- also: a reused registry reads `  registry  refs/heads/workflow-state @ <tip> unchanged` with no `pushed` line and
  JSON `pushed` `false`; `--dry-run` shows `  registry   unchanged refs/heads/workflow-state` for an existing
  registry, no blank line before `Would initialize …`; counts are singular for one and only the first carries
  `file`/`files`; the no-op form needs the registry to exist and no create, update, or remove (otherwise the normal
  path and the confirmation); `--runtime`/`--ext` split on commas, `--keep`/`--overwrite` take one path per flag; a
  relative `<dir>` joins `-C`, an absolute one is used as is; `openRepo(ctx, env, flags, dir, fix)` is shared with
  `openStore`; `registry_exists_invalid` is a `task.Rejection`, `unknown_*` a `project.Rejection`; the stage ran to
  about 900 lines, accepted by the author. I §3.3, UC-1, UC-2, UC-3, architecture §7, §8 updated to match.

- [x] `init` in the command table and help; flags, values, check order, the registry, apply, rollback until the CAS
- [x] human and JSON output, `file_conflict` with the plan on stdout, `interactive_required`
- [x] tests: UC-2 core-only, UC-3 no-op and `already_initialized`, an interrupted init completed by adoption, F-02
  `keep`, conflicts with `--keep`/`--overwrite`, `registry_exists_invalid`, `--dry-run`, F-09 (a failing manifest
  write creates no ref; a failing CAS restores every file; a failing push warns, exit 0)

### Stage 8.7 — `--git-init`, empty repository, `.jj/` (`internal/cli`; UC-4) `[x]`

Decisions (author, 2026-10-21):
- `not_a_repository` from `init` keeps the message `<absolute dir> is not a Git repository` with the fix ``run `git
  init -b <main-branch>` first, or rerun with --git-init`` (UC-4);
- `--git-init` (usage `[--git-init]`, help line updated): when the directory is not a repository, `init` plans
  against that directory as the root (it must exist; otherwise the existing unexpected error), with no manifest
  check against Git and the registry `create`; the plan's first line is `  git        init -b <main-branch>` (JSON
  `"git_init": true` in `data`, `false` otherwise); the plan counts as a write, so it needs `--yes`; after the
  confirmation `init` runs `git init -b <main-branch> <dir>`, discovers the new repository, then applies and creates
  the registry as before; a failure after `git init` rolls back the files but leaves the new `.git` (a re-run
  adopts the files; architecture §8); `--git-init` inside a repository is ignored; `--dry-run` never runs `git init`;
- empty repository: when `refs/heads/<main-branch>` has no commit (always after `--git-init`), the success text's
  `Next:` step 1 reads `  1. First commit required before task worktrees:` followed by the command lines indented
  seven spaces (`git switch --orphan <main-branch>` first when HEAD is not on `<main-branch>`, then `git add AGENTS.md
  .solomonar docs && git commit -m "chore: initialize solomonar <fw>"`); JSON `"first_commit_required": true`
  (`false` otherwise) and `next` lists the same commands;
- `.jj/` at the worktree root: `init` adds the warning `jujutsu_detected`, message `Jujutsu detected; v0.1 supports
  Git only. Never run jj commands on workflow-state.` (UC-4; stderr / envelope), also with `--dry-run`; other
  commands do not warn yet (`doctor` reports it later).
- also: a detached HEAD counts as not on the main branch (`git switch --orphan` first); `jujutsu_detected` comes
  before the registry's warnings; any `.jj` `Lstat` error means no warning; the `--git-init` root is `absDir(dir)`,
  not symlink-resolved; a `file_conflict` under `--git-init` lists only the conflicts; a failing `git init` reads
  `git init: %w` (exit 1), and a failing discovery after it returns as is (nothing written yet); help line
  `--git-init runs git init -b <main-branch> when <dir> is not a repository.`; `checkVersion` and `traceWriter` split
  from `openRepo`; tests use `GIT_CEILING_DIRECTORIES` and `GIT_TEMPLATE_DIR` through `t.Setenv`. I §3.2, I §3.3,
  UC-1, UC-4, architecture §8 updated to match.

- [x] `not_a_repository` fix; `--git-init` planning, confirmation, `git init`, rollback (UC-4)
- [x] empty repository `Next:` and `first_commit_required`; the `jujutsu_detected` warning

### Stage 8.8 — Task commands read the manifest (`internal/cli`; I §3.2, architecture §7) `[x]`

Decisions (author, 2026-10-21):
- every task command (`task plan`, `show`, `register`, `move`, `block`, `unblock`, `handoff`, `done`, `status`)
  reads `.solomonar/manifest.json` from the current worktree root (`repo.Dir`) with `project.ReadManifest` after
  repository discovery and the Git version check, before the registry; the store's remote and the main branch come
  from `project.remote` and `project.main_branch`, else `origin` and `main` when the manifest is absent (I §3.2);
- a newer manifest is `cli_outdated` (exit 3, `<command> rejected`), an invalid one `manifest_invalid` (exit 1,
  `<command> failed`), with the command label used today (`task move W-001 rejected`); nothing is written;
- the main branch replaces `defaultMainBranch` in `task register` (the worktree command), `task done` (merge
  check and its line), and `task status` (facts, drift, the `merged` line); `defaultRemote`/`defaultMainBranch`
  remain only as the fallbacks;
- `version` reading the project is Stage 8.9.
- also: `openStore(ctx, env, flags) (*task.Store, string, error)` returns the main branch beside the store, read by
  `readProject(env, dir)`; every `openStore` error goes through `rejectedAs` with the command's label (`task show`
  builds its label once); the drift text follows the main branch (`merged into trunk`); the `task status` JSON is
  unchanged; an I/O failure opening the worktree reads `open worktree: %w`. I §3.2, I §8.4, architecture §7 updated
  to match.

- [x] `openStore` reads the manifest; remote and main branch from it; `cli_outdated`/`manifest_invalid`
- [x] tests: a manifest with another remote and main branch drives push, `register`'s worktree line, `done`'s merge
  check, and `status`; a newer and an invalid manifest reject with nothing written; no manifest keeps the defaults


### Stage 8.9 — `version` reads the project (`internal/project`, `internal/task`, `internal/cli`; UC-24) `[x]`

Decisions (author, 2026-10-21):
- `version` never decodes the manifest or the registry strictly: it reads only their version fields
  (`project.ManifestVersions(data) (schema int, framework string, err error)` and `task.SchemaOf(data) (int, error)`,
  sharing the existing header checks of `checkVersions` and `checkSchemaVersion` without the newer-than check), so a
  newer project still prints its versions and exits 0 (CLI-D19, UC-24);
- context: repository discovery from `-C` as other commands; `not_a_repository` → `project   not initialized`
  (JSON `null`s); `git_too_old` and other discovery errors stay errors; inside a repository without
  `.solomonar/manifest.json` → `not initialized`;
- with a manifest: `project   framework <project fw> · registry schema <n>`, `registry missing` when the ref is absent
  (JSON `registry_schema` `null`); a manifest whose version fields are unreadable → `project   manifest invalid
  (inspect .solomonar/manifest.json)`, an unreadable registry → `· registry invalid`, both `null` in JSON, exit 0;
- newer: when the project framework is newer than the embedded one, the line ends ` (newer than this CLI; install
  solomonar <project fw> or later)`; when only the manifest or registry schema is newer, ` (newer than this CLI;
  install a newer solomonar release)`;
- the help description's "Outside a project the project line reads \"not initialized\"." stays.
- also: `project.NewerFramework(project, shipped string) (bool, error)` keeps the malformed-embedded-version error;
  `(*task.Store).ReadSchema(ctx) (schema int, present bool, err error)` shares the blob read with `Read`; an invalid
  manifest skips the registry (both JSON values `null`); the framework suffix wins over the schema suffix and follows
  whichever registry part is shown; a registry commit without `state.json`, a manifest read failure other than
  absence, and a failure opening the worktree are unexpected errors (exit 1); any value not shown is `null` in JSON.
  UC-24, I §3.6, architecture §6 updated to match.

- [x] `ManifestVersions`, `SchemaOf` (domain), with the existing strict paths unchanged
- [x] `version` project line and JSON; tests for every row above, including a newer framework and a newer schema

## Step 9: `init` with runtimes and catalog extensions

Decisions (author, 2026-10-21): CLI-D9 is decided from the official documentation (Claude Code skills, subagents,
memory, hooks; Codex AGENTS.md, skills, subagents, hooks): Codex skills live in `.agents/skills/<skill>/SKILL.md`
(`name`, `description` required), the user-only `solomonar` skill gets `.agents/skills/solomonar/agents/openai.yaml`
with `policy: allow_implicit_invocation: false`, roles are `.codex/agents/<role>.toml` (`name`, `description`,
`developer_instructions`, optional `model`), Codex has no tool allowlist so read-only reviewers stay a behavioral
contract (rules §11); the Claude Code forms of I §4.2 are confirmed. Step-9 limitation (task list only, the specs
keep the final behavior): no state-guard hook for either runtime (`.claude/settings.json`, `.codex/hooks.json`)
until `solomonar guard` exists in increment 2, because a hook running a missing command would block every tool
call.

### Stage 9.1 — Runtime adapter stubs (`internal/project`; I §4.2) `[x]`

Decisions (author, 2026-10-21):
- `RuntimeWants(assets fs.FS, runtimes []string, models map[string]string, framework string) ([]Want, error)`
  returns, per runtime in the order `claude`, `codex`, the wants below; skills are the core skills under
  `framework/skills/` and roles the files under `framework/agents/` except `orchestrator` (top-level session, G13),
  both sorted; `name` and the `description` line are copied verbatim from each portable file's frontmatter (a
  missing one is `read frontmatter <path>: …`);
- `claude` (owner `runtime:claude`): `CLAUDE.md` with `@AGENTS.md` (create-only, below); `.claude/skills/<skill>/SKILL.md`;
  `.claude/agents/<role>.md`;
- `codex` (owner `runtime:codex`): `.agents/skills/<skill>/SKILL.md`; `.agents/skills/solomonar/agents/openai.yaml`;
  `.codex/agents/<role>.toml`;
- skill stub: `---`, `name: <name>`, `description: <description>`, `disable-model-invocation: true` only in the
  claude `solomonar` stub, `---`, blank line, ``Generated by solomonar <fw> from `docs/solomonar/skills/<skill>/SKILL.md`;
  edit that file, then run `solomonar sync`.``, ``Load `docs/solomonar/skills/<skill>/SKILL.md` and follow it.``;
- claude role stub: `---`, `name`, `description`, `tools: Read, Grep, Glob, Bash` for every role but `implementer`
  (which inherits the session's tools), `model: <model>` for `adversarial-reviewer` when configured, `---`, blank
  line, ``Generated by solomonar <fw> from `docs/solomonar/agents/<role>.md`; edit that file, then run `solomonar sync`.``,
  ``Load `AGENTS.md`, then `docs/solomonar/agents/<role>.md` and every file its first lines require.``;
- codex role stub (TOML): `# Generated by solomonar <fw> from docs/solomonar/agents/<role>.md; edit that file, then run
  solomonar sync.`, `name = "<role>"`, `description = "<description>"`, `developer_instructions = "Load AGENTS.md, then
  docs/solomonar/agents/<role>.md and every file its first lines require."`, `model = "<model>"` for
  `adversarial-reviewer` when configured; basic strings escape `\` and `"` only;
- `openai.yaml`: `# Generated by solomonar <fw>; edit docs/solomonar/skills/solomonar/SKILL.md, then run solomonar
  sync.`, `policy:`, `  allow_implicit_invocation: false`;
- `CLAUDE.md` is create-only (CLI-D3, G10): `Want.CreateOnly`; absent → `create` (hashed, managed from then on);
  managed → the I §4.5 table; present and unmanaged → `skip`, not recorded, reason `exists; add this line so Claude
  Code loads the framework:  @AGENTS.md` when it has no line exactly `@AGENTS.md`, else `exists and imports
  AGENTS.md`; never a conflict;
- `InitPlan` gains `runtimes map[string]Runtime` (the manifest `runtimes`, `{"adversarial_model"?}`; nil → `{}`).
- also: `runtimes` is a set validated before reading assets (`unknown runtime %q`); plan order claude: `CLAUDE.md`,
  skills, roles; codex: skills, `openai.yaml`, roles; stub paths use the directory or file name, `name` the
  frontmatter value; a skill is a directory with `SKILL.md`, a role a `.md` file other than `orchestrator.md`;
  frontmatter errors `read frontmatter <path>: no frontmatter|missing name|missing description`; an empty value is
  missing; TOML escapes `\` and `"`, any control character (tab included) is `render <path>: control character in
  %q`; YAML values are written raw; a non-regular unmanaged `CLAUDE.md` is `skip` too; the import line must be
  exactly `@AGENTS.md`; the stage ran to about 580 lines. I §4.2, I §4.5, UC-4, UC-5, UC-10, CLI-D9, architecture
  §6 updated to match (§6 landed in the author's commit kspzztts).

- [x] `RuntimeWants` with the stub forms above, frontmatter reading, the role and skill sets
- [x] `Want.CreateOnly` in `Classify`; `InitPlan` records the runtimes

### Stage 9.2 — `init` with runtimes (`internal/cli`, `internal/project`; UC-2, I §4.2) `[x]`

Decisions (author, 2026-10-21):
- `--runtime` takes `claude`, `codex`, or `none` (repeatable, comma-separated); another value is `unknown_runtime`
  (exit 2, `init rejected`), check `--runtime names a known runtime`, found `"<v>" is not a runtime; known: claude,
  codex`, fix `pass --runtime claude, codex, or none`; `none` beside another value is `invalid_value`, check
  `--runtime names runtimes or none`, found `none cannot be combined with <v>`, fix `pass --runtime none alone`;
  `--ext` keeps its step-8 limitation until 9.3;
- default without `--runtime` and without a manifest: each of `claude`, `codex` whose executable is found on `PATH`
  (`exec.LookPath`); with a manifest, the manifest's runtimes;
- `--adversarial-model <runtime>=<model>` (repeatable): malformed is `invalid_value`, check `--adversarial-model is
  <runtime>=<model>`, found `"<v>" has no "="` (or an empty side), fix `pass --adversarial-model claude=<model>`; a
  runtime that is not selected is `invalid_value`, check `--adversarial-model names a selected runtime`, found
  `<runtime> is not selected`, fix `add --runtime <runtime> or drop the model`; a runtime given twice keeps the last;
  recorded as `runtimes.<r>.adversarial_model`;
- `Requested.AdversarialModels map[string]string` (nil = not given) joins `CheckChoices` after extensions: a given
  model differing from the manifest is `already_initialized`, found `manifest adversarial_model <r> "<v>"; requested
  "<v>"` (`none` for an absent model), fix `change models with \`solomonar runtime add <runtime>
  --adversarial-model <model>\`; init never changes choices`;
- the wants are `CoreWants` then `RuntimeWants`; plan lines stay one per file;
- `Next:` step 1 adds, in this order, `CLAUDE.md` (only when it is created), `.claude` (claude), `.agents` and `.codex`
  (codex) to `git add AGENTS.md … .solomonar docs`; step 2: with only claude `  2. Start a top-level Claude Code
  session in <root>:` and `       /solomonar Explore the next useful flow.`; with only codex `  2. Start a top-level
  Codex session in <root>:` and `       $solomonar Explore the next useful flow.`; with both `  2. Start a top-level
  session in <root> with your selected outcome:`, `       Claude Code: /solomonar Explore the next useful flow.`,
  `       Codex:       $solomonar Explore the next useful flow.`, `       Portable:    Use the solomonar skill to explore
  the next useful flow.`; with none, today's portable line;
- JSON `choices.runtimes` (sorted) and `choices.adversarial_model` (`{runtime: model}`); `next` follows step 1.
- also: the PATH lookup is `git.OnPath(program) bool` because only `internal/git` may import `os/exec` (AD-3);
  `gittest.PathWith(t, names...)` puts fake executables on `PATH`; malformed models read `"<v>" has no "="`, `"<v>"
  has an empty runtime`, `"<v>" has an empty model`; one value per flag; an unknown runtime in a model is `<r> is not
  selected`; the model fix names the runtime; check order: runtime values, model syntax, `--ext`, then after the
  manifest the runtime selection, the unselected-runtime check, `CheckChoices`; models start from the manifest's for
  the selected runtimes, requested ones on top; every init test controls `PATH`. UC-1, UC-2, UC-3, architecture §5,
  §6 updated to match.

- [x] runtime values, PATH default, `--adversarial-model`, the manifest runtimes, `CheckChoices` on models
- [x] wants with `RuntimeWants`; `Next:` paths and invocations; JSON choices; tests (UC-2 with `--runtime claude`,
  both runtimes, PATH detection, models, a re-run no-op, `already_initialized` on runtimes and models)
Decision (author, 2026-10-21): the extension engine is generic and tested with `fstest.MapFS` catalogs; the
embedded catalog holds only `go` until the author writes `htmx`, `templ`, and `tdd` as framework content. Step-9
limitation (task list only): `--ext` takes catalog names and `none`; a local path (project extension, UC-7) waits
for increment 2. Orchestrator split: 9.3 catalog, resolution, rendering; 9.4 extension files, `AGENTS.md`
extensions block, adapter stubs, manifest entries; 9.5 decisions and documents; 9.6 the `init --ext` command.

### Stage 9.3 — Catalog, resolution, rendering (`internal/project`; I §4.3) `[x]`

Decisions (author, 2026-10-21):
- `LoadCatalog(assets fs.FS) (Catalog, error)` with `Catalog map[string]CatalogExtension`; each
  `extensions/<group>/<name>/extension.json` decodes strictly (`encoding/json/v2`, unknown members rejected) into
  `CatalogExtension{Name, Kind, Group, Title string; Requires, Conflicts []string; AppliesWhen string; Paths
  []string}` plus `Dir` (the catalog directory); checks: `name` equals the directory and matches `[a-z][a-z0-9-]*`,
  `group` equals its directory and is one of the nine I §4.3 groups, `kind` `skill` (directory holds `SKILL.md`) or
  `agent` (directory holds `<name>.md`), `title` and `applies_when` non-empty, `requires`/`conflicts` name catalog
  extensions; any failure is an error `read catalog <dir>: <reason>` (an embedded-asset defect, exit 1);
- `(c Catalog) Resolve(names []string) ([]string, error)`: returns the sorted unique names; checks in order, each
  over the sorted names: unknown → `unknown_extension` (exit 2), check `--ext names a catalog extension`, found
  `"<v>" is not in the catalog; known: <sorted names>`, fix `pass --ext with a known name, or none`; a missing
  requirement → `extension_requires` (exit 3), check `every required extension is selected`, found `<name> requires
  <missing, comma-joined>`, fix `add --ext <missing, comma-joined>`; a conflict → `extension_conflict` (exit 3),
  check `no selected extensions conflict`, found `<a> conflicts with <b>`, fix `drop --ext <a> or --ext <b>`;
  `unknown_extension`, `extension_requires`, `extension_conflict` are `project.Rejection`s;
- `Render(content []byte, self string, installed []string, decision string) (rendered []byte, with []string)`: a
  section starts at a line `## With <...>` followed by the line `<!-- solomonar:with <a>[,<b>...] -->` and runs to the
  next line starting `## ` or the end; it is kept, without its marker line, when every named extension is installed,
  and dropped otherwise; `with` is the sorted union of the names in kept sections (`rendered_with`); one provenance
  line goes under the first `# ` title line, separated by blank lines: `Installed by <decision>; rendered with: <a>,
  <b>.`, `Installed by <decision>.` when nothing was kept, and `solomonar` in place of `<decision>` when it is empty
  (workflow and review groups); content without a title gets no line;
- the step-8 `unknown_extension` texts in `internal/cli` stay until 9.6 replaces them with `Resolve`.
also: catalog entries are `extensions/*/*/extension.json`, a directory without one is ignored and a missing
`extensions` directory is an empty catalog (`known: none`); per-entry check order and reason texts as in I §4.3
(duplicate names `name "<n>" is also in <dir>`; `requires`, then `conflicts`, checked after all entries load); a
conflict declared on one side only is still found, reported from the declaring side; `extension_requires` joins
with `, ` in found and `,` in fix; a `## With` heading without a valid marker line is ordinary text; a dropped
section takes its trailing blank lines (and, at the end, the blank lines before it); `rendered_with` excludes the
extension itself; the provenance line has exactly one blank line on each side. Specs updated: I §4.3 (catalog
checks, selection table, Generation), UC-7 (`unknown_extension` exit 2), architecture §6.

- [x] `LoadCatalog`, `CatalogExtension`, validation; the embedded catalog (`go`) loads
- [x] `Resolve` with the three rejections; `Render` with sections, markers, provenance, `rendered_with`

### Stage 9.4 — Extension files, `AGENTS.md` block, adapter stubs, manifest entries (`internal/project`; I §4.2, §4.3, §4.5, §4.6, §4.7) `[x]`

Decision (author, 2026-10-21): an `AGENTS.md` extension line is `` - `<name>` (<group>): <applies_when verbatim> ``,
followed by `` ; paths: `<glob>`, `<glob>` `` when `paths` is not empty.

Decisions (author, 2026-10-21):
- `ExtensionWants(assets fs.FS, installed []CatalogExtension, decisions map[string]string) ([]Want,
  map[string]Extension, error)`: `decisions` maps a stack group to its decision ID (9.5 computes it; tests pass
  `P/D-n`); `workflow` and `review` take none. Per extension, sorted by name: a skill installs every file of its
  catalog directory except `extension.json` to `docs/solomonar/skills/<name>/<rel>` (sorted by path), an agent only
  `<name>.md` to `docs/solomonar/agents/<name>.md`; owner `ext:<name>`; `SKILL.md` and the agent file go through
  `Render(content, name, installed names, decision)`, other files are copied verbatim. The manifest entry is
  `{state installed, kind, group, source catalog, decision (stack groups only), rendered_with}` from `Render`;
- the catalog `description` in the skill or agent frontmatter is authored content and is copied as is (the CLI does
  not generate it from `applies_when`);
- `Choices` gains `Extensions []CatalogExtension`; `CoreWants` renders the `AGENTS.md` block: `- Skills` with one
  nested line per installed skill and `- Agents: an installed agent is a specialist reviewer, required by triage when
  its "applies when" matches.` with one nested line per installed agent, each list ordered by group (I §4.3 table
  order), then name; an empty list keeps the existing `- Skills: none.` / `- Agents: none. …` line;
- `RuntimeWants` gains `extensions []CatalogExtension`: an installed skill gets a skill stub, an installed agent a
  role stub, per runtime, merged with the core skills and roles in name order (I §4.2 plan order); extension agent
  stubs get the reviewer form (claude `tools: Read, Grep, Glob, Bash`; codex TOML as for core roles); stub source
  paths are the installed `docs/solomonar/...` paths; `name` and `description` come from the catalog file's
  frontmatter;
- `InitPlan` gains `extensions map[string]Extension` (nil records `{}`), recorded in the manifest;
- `internal/cli` passes no extensions until 9.6 (call sites updated only).
also (implementer, confirmed): wants ordered by extension name, then path; only the
top-level `extension.json` is excluded; a stack-group extension without a non-empty decision is an error `render
<name>: no decision for group <group>`; a decision given for `workflow`/`review` is ignored; unreadable files are
`read extension <name>: <err>`; `emptyExtensions` folded into `extensionsBlock`. Known gap (not handled): an
extension name equal to a core skill or role id would give two stubs the same path; the embedded catalog avoids it.
Specs updated: I §4.2, §4.3 (installing, `applies_when`/`description`), §4.4 (`go` row), §4.6, §4.7; architecture §6.

- [x] `ExtensionWants`: files, owners, rendering, manifest entries
- [x] `AGENTS.md` extensions block in `CoreWants`
- [x] extension skill and agent stubs in `RuntimeWants`; `InitPlan` records `extensions`

### Stage 9.5 — Decisions and their documents (`internal/project`; I §4.3 Decision recording, §4.5, rules §3) `[x]`

Decisions (author, 2026-10-21):
- entry titles are fixed per group: `language` Programming languages, `architecture` Architecture style, `frontend`
  Frontend technologies, `backend` Backend technologies, `persistence` Persistence technologies, `testing` Testing
  practice, `operations` Build and deployment;
- step-9 limitation (task list only): `init` only appends; numbering continues after the highest `P/D-n` already in
  `docs/decisions.md`, and an approved entry for the same group is not superseded (superseding comes with
  `extension add|remove`, increment 2).

Decisions (author, 2026-10-21; orchestrator proposals confirmed):
- `PlanDecisions(root *os.Root, existing *Manifest, installed []CatalogExtension, date string) (Decisions, error)`
  with `Decisions{Changes []Change; Files map[string]File; Records []Decision; ByGroup map[string]string; Index
  []string}` and `Decision{ID, Group string; Selected []string}` (JSON `id`, `group`, `selected`, I §5 UC-1);
- one new entry per stack group of the selection, in I §4.3 group order, `P/D-<n>` numbered from the highest
  `## P/D-<n>:` heading in `docs/decisions.md` (0 when absent) plus one; a group whose decision the existing
  manifest already records takes that ID and adds nothing (re-run, UC-3);
- entry (rules §3 format), separated from what precedes it by one blank line:
  `## P/D-<n>: <title>` / blank / `- Status: approved` / `- Selected option: <a>, <b>` / `- Rejected options: none` /
  ``- Scope: project; <group> extensions `<a>`, `<b>` `` / ``- Rationale: selected by the user with `solomonar init` ``
  / `- Date: <date>` / ``- References: `<installed main file>`, …`` (names sorted; the skill `SKILL.md` or the agent
  file path from 9.4);
- group documents: `architecture` → `docs/architecture.md`, every other stack group → `docs/stack.md`; one line per
  group `- <group>: <a>, <b> (P/D-<n>)`, in group order;
- a created document starts with a title and a blank line (`# Decisions`, `# Stack`, `# Architecture`) and is
  `create`; an existing regular file gets the new content appended (after a newline when it lacks one) and is the new
  action `append` (`ActionAppend`, applied like `update` with the full new content); a present non-regular file is an
  unexpected error `read <path>: not a regular file`; owner `decision`, manifest `{"owner": "decision", "entries":
  [IDs it holds from this and earlier runs of the manifest]}`; on a re-run the existing manifest's `decision` files are
  carried over and planned `unchanged` without reading them;
- `Index` lists the README lines for the documents this plan creates, in the order decisions, stack, architecture:
  ``- `docs/decisions.md`: decision entries and their status``, ``- `docs/stack.md`: technologies and practices, with
  the decisions that adopted them``, ``- `docs/architecture.md`: structure and contracts, with the decisions that
  adopted them``; `Choices.Documents []string` adds them to the created README's "Documents" list (after the existing
  items); for a present README (seed `skip`), `InitPlan` turns the skip into `append` when the file has a line
  `## Documents`, inserting the lines after the section's last `- ` line (or after the heading and a blank line when
  it has none), and otherwise keeps `skip` with reason `exists; add under "## Documents": <line>; <line>`;
- `InitPlan` takes the `Decisions` value (after `extensions`), puts its changes after the classified wants and before
  the manifest, and merges its files into the manifest.
also (implementer, confirmed): appending adds a missing final newline, then (decisions.md
only) a blank line unless the file already ends with one; stack/architecture lines follow directly; README insertion
adds no blank lines after existing items, and in an item-less section ensures a blank line before and (when text
follows) after the lines; a non-regular `docs/README.md` with index lines is `read docs/README.md: not a regular
file`; on a re-run a group's ID comes from the first extension (name order) whose manifest entry has a `decision`, any
ID accepted; a document with nothing to add and no record gets no change; `entries` are carried IDs then new ones;
numbering counts only `## P/D-<digits>:` lines; `installedPath` shared with `ExtensionWants`. Open for 9.6: the
counts line (I §3.3) has no bucket for `append` (proposal: count it as updated). Size: ~630 lines (moderate overrun).
Specs updated: I §3.3 (`append`), §4.3 (titles, numbering, entries, documents, README index), §4.5 (`decision`,
`seed`, re-run), UC-3; architecture §6, §8.

- [x] `PlanDecisions`: numbering, entries, documents, re-run carry-over, `ActionAppend`
- [x] README index lines (created and present README); `InitPlan` integration

### Stage 9.6 — `init --ext` (`internal/cli`; UC-2, UC-3, I §4.3) `[x]`

Decisions (author, 2026-10-21): an `append` change counts as `updated` in the counts line (I §3.3).

Decisions (author, 2026-10-21; orchestrator proposals confirmed):
- `--ext` values split on commas; `none` requests an empty selection; `none` beside another value is
  `invalid_value` (exit 2), check `--ext names extensions or none`, found `none cannot be combined with <v>`, fix
  `pass --ext none alone`; otherwise the names go through `Catalog.Resolve` (`unknown_extension` exit 2,
  `extension_requires`/`extension_conflict` exit 3, `init rejected`), in the usage-and-values phase after the
  `--adversarial-model` syntax (UC-2 check order); a path value is `unknown_extension` (step-9 limitation);
  `noExtensions` and its step-8 texts go;
- selection: the requested names, else the manifest's extension names (re-run), else `go` when the worktree root has
  a regular file `go.mod` and `go` is in the catalog, else none; a manifest name missing from the catalog is an
  unexpected error `read extension <name>: not in the catalog`;
- the plan: `PlanDecisions(root, existing, selected, <env.Now() as YYYY-MM-DD>)`; `CoreWants` with `Extensions`
  and `Documents: decisions.Index`; wants = core, `ExtensionWants(…, decisions.ByGroup)`, `RuntimeWants(…,
  selected)`; `InitPlan(…, extension entries, decisions, …)`; plan order core, extensions, runtime stubs, decision
  documents, manifest (a flat list; UC-1's grouped plan comes with step 10);
- JSON: `choices.extensions` maps each group with a selection to its sorted names (`{}` when none); `decisions` is
  `Decisions.Records` (`[]` when none) as `{"id", "group", "selected"}`;
- no new human lines: success, dry-run, no-op, and `Next:` keep their forms (`docs` already covers the decision
  documents); the I §4.3 note distinguishing instructions from packages belongs to `extension add` plans.
also (implementer, confirmed): the no-op check counts `append` as a write (UC-3); the
help usage shows `[--ext <name|none>]...` (step-9 limitation; the spec keeps `<name|path|none>`); `go.mod` is checked
with `Lstat` (a symlink does not count); only requested names go through `Resolve`, not the manifest selection or the
`go` default; the catalog loads right after the `--adversarial-model` syntax check and is passed to `initialize`.
Specs updated: I §3.3 (counts), UC-1 (JSON), UC-2 (`--ext`, defaults, check order, plan order), UC-3; architecture §7.

- [x] `--ext` values, `none`, `Resolve`; default selection (`go.mod`); manifest selection on a re-run
- [x] plan wiring (decisions, extension files, stubs, `AGENTS.md`, README index); counts `append` as updated
- [x] JSON `choices.extensions` and `decisions`; tests (UC-2 with `--ext go`, default via `go.mod`, `none`, unknown,
  requires/conflict with a test catalog if the CLI env allows it, a re-run no-op, `already_initialized` on extensions,
  an existing `docs/decisions.md` appended)

## Step 10: Interactive `init`

### Stage 10.1 — Prompts: text, confirmation, key-driven multi-select (`internal/cli`, `main`; I §3.7, CLI-D5) `[x]`

Decisions (author, 2026-10-21):
- CLI-D5: text prompts and confirmations stay line-based; the multi-select is key-driven: ↑/↓ move, Space toggles,
  Enter accepts (the number-typing form is dropped);
- raw terminal mode comes from `golang.org/x/term` (`term.MakeRaw`/`term.Restore`), an AD-1 exception: the only
  production dependency besides the standard library; depguard allows it.

Decisions (author, 2026-10-21; orchestrator proposals confirmed):
- `Env` gains `RawMode func() (restore func(), err error)`; `main` sets it from `term.MakeRaw(int(os.Stdin.Fd()))`;
  tests inject a no-op; prompts run only when `Env.Interactive` (unchanged rule, I §3.7);
- a `prompter` over `Env` (one `bufio.Reader` on `Stdin` shared by every prompt, `Stdout` for output):
  - `text(label, def)`: prints `<label> [<def>]: `, or `<label>: ` when `def` is empty; the answer is trimmed, empty
    gives `def`;
  - `confirm(question, def)`: prints `<question> [Y/n] ` or `[y/N] `; `y`, `yes`, `n`, `no` in any case; empty gives
    the default; anything else prints `  answer y or n` and asks again;
  - `choose(items, selected)` with items `{name, description}`: in raw mode, lines `  > [x] <name>  <description>`
    (`>` on the cursor row, two spaces elsewhere; names padded to the widest), then `  ↑/↓ move · Space toggle · Enter
    accept`; keys: `ESC [ A`/`k` up, `ESC [ B`/`j` down (clamped, no wrap), Space toggles, `a` selects all, `n`
    selects none, Enter (`\r` or `\n`) accepts, any other key is ignored; output uses `\r\n` (raw mode turns off
    output processing); a redraw moves up over the list (`ESC [ <n> A`, `\r`, `ESC [ J`) and reprints; on accept the
    list is redrawn once without the cursor marker and without the help line; the terminal is restored on every
    return path;
- end of input on any prompt is an unexpected error `read answer: input closed` (exit 1); Ctrl-C (byte 3) in the
  multi-select returns `context.Canceled`, so the command ends `interrupted` (exit 130, I §3.5); in line prompts the
  terminal is not raw and SIGINT follows CLI-D18;
- no command uses the prompts until 10.2.
also (implementer, confirmed): `golang.org/x/term v0.47.0` (`x/sys v0.49.0` indirect);
`choose(items []choice, selected []string) ([]string, error)` works on names and returns the marked ones in item order
(`[]` when none); rows keep the pointer column (two spaces when not on the cursor, so accepted rows read `    [x] …`)
and trim trailing spaces; malformed escape sequences are ignored; a final line without a newline is input closed;
Ctrl-C prints nothing; other read errors are `read answer: <err>`; `RawMode` errors pass through unwrapped (`main`
adds `raw terminal:`); `choose` requires `Env.RawMode` (10.2 calls it only when interactive).
Specs updated: I §3.7, UC-1 transcript, CLI-D5; architecture AD-1, §2, §3, §5 (`main`, `Env`), lint and test tables.

- [x] `golang.org/x/term`, depguard, `Env.RawMode` in `main`
- [x] `text`, `confirm`, `choose` with tests over byte streams (keys, redraws, defaults, re-ask, EOF, Ctrl-C)
### Stage 10.2 — Interactive `init`: choices and confirmation (`internal/cli`; UC-1, I §3.7) `[x]`

Decisions (author, 2026-10-21):
- declining a confirmation prints `Cancelled; nothing was written.` on stdout and exits 0;
- an interactive extension selection that fails `Resolve` prints `  <found>; <fix>` and asks every group again, with
  the previous selection marked;
- the grouped, summarized plan of UC-1 is stage 10.4; until then the interactive plan is the flat list.

Decisions (author, 2026-10-21; orchestrator proposals confirmed):
- `init` prompts when `Env.Interactive` and neither `--yes` nor `--json` (I §3.7); otherwise behavior is unchanged
  (`interactive_required` without `--yes`); `--dry-run` still prompts for choices, never for the confirmation;
- choices are prompted only without a manifest (a re-run takes them from the manifest), each only when its flag was
  not given, in this order, with UC-1's texts and blank lines:
  - `Project name [<default>]: `; `Current objective, one line (optional): `;
  - `Runtimes that will run this project's sessions:` then `choose` over `claude` (`Claude Code: CLAUDE.md import,
    .claude/skills, .claude/agents, state guard hook`) and `codex` (`Codex: AGENTS.md (native), .agents/skills,
    .codex/agents, state guard hook`), the runtimes found on `PATH` marked;
  - per selected runtime without `--adversarial-model`, in the order claude, codex: `Model for the adversarial
    reviewer in <Claude Code|Codex>, different from your main session model` then `(blank = runtime default): `;
    blank records no model;
  - when the catalog is not empty and `--ext` was not given: the paragraph `Extensions, by decision type. A choice
    in a stack group is recorded as your approved decision in` / ``docs/decisions.md. Enter with nothing selected
    skips a group; add more later with`` / `` `solomonar extension add`. ``, then per group with catalog extensions,
    in I §4.3 order, `<Group>:` (first letter capitalized) and `choose` over its extensions (`<title>`, plus ` (requires
    <a>, <b>)` when `requires` is not empty), `go` marked when the 9.6 default applies; then `Resolve`;
- after the plan (when it writes and has no conflict, and not `--dry-run`): `Plan`, the flat plan lines, the registry
  line, then `Proceed? [Y/n] `; yes applies as with `--yes`, no is the cancel above;
- the prompts take the defaults the flags would take (name from the root, runtimes from `PATH`, `go` from `go.mod`).
also (implementer, confirmed): choices are asked after the manifest checks and before
the runtime selection, and answers are written back into the flag fields; both model prompts form one block; a blank
line separates asked blocks, precedes `Plan` only when something was asked, and follows the `Proceed?` answer; the
re-ask prints the rejection right after the last list and repeats every group but not the paragraph; extensions are
listed by name; an empty interactive selection means none; a re-run asks only `Proceed?`; `project.Groups()` exported;
the test catalog's `SKILL.md` files gained frontmatter.
Specs updated: I §3.5, §3.7, UC-1 (transcript model prompts, Prompts paragraph), UC-2 (check order, `--dry-run`);
architecture §6 (`Groups`).

- [x] prompt wiring for name, objective, runtimes, models, extensions (with the `Resolve` re-ask)
- [x] `Proceed?` confirmation and cancel; tests with `Env.Interactive`, scripted stdin, and a no-op `RawMode`

### Stage 10.3 — Interactive `init`: header, `git init` question, `AGENTS.md` adoption (`internal/cli`; UC-1, UC-4) `[x]`

Decisions (author, 2026-10-21; orchestrator proposals confirmed):
- when `init` prompts (10.2 rule), it first prints the header `solomonar <cli version> · framework <fw> · <dir>
  (git, branch <b>)` and a blank line: `<dir>` is the worktree root (the target directory when it is not a
  repository), `<b>` the current branch (also unborn), `(git, detached HEAD)` when detached, `(not a Git repository)`
  outside one; `<cli version>` as `solomonar version` prints it;
- outside a repository without `--git-init`: `confirm("Initialize a Git repository here with main branch
  \"<main-branch>\"?", false)`; yes continues exactly as `--git-init`; no is the 10.2 cancel (`Cancelled; nothing
  was written.`, exit 0); non-interactively `not_a_repository` is unchanged;
- an unmanaged `AGENTS.md` planned for adoption (reason `adopted; existing content moved under "## Project notes"`)
  and not named by `--overwrite`: before the plan display, `AGENTS.md exists and is not managed by solomonar.` then
  `confirm("  Adopt it: write the framework part above \"## Project notes\" and move the current content into that
  section?", true)`; no asks `confirm("  Replace it with the framework template, discarding its content?", false)`;
  yes plans again as with `--overwrite AGENTS.md`; no cancels; `--dry-run` asks neither (it shows the adoption);
- each question is its own block (blank-line rule of 10.2).
also (implementer, confirmed): Adopt and Replace form one block; a declined question
ends its block with a blank line before the cancel line; the adoption question is skipped when the plan has other
conflicts (`file_conflict` as before); the header prints on every prompting run, a no-op re-run included, and not when
opening the repository fails otherwise; outside a repository `<dir>` is the absolute target as given; `git.Repo.Branch`
(`symbolic-ref --short --quiet HEAD`, "" when detached) also serves `unbornMain`; `project.Change.Adopted()`; the
framework version is read before the repository is opened; `openTarget`/`beforePlan` extracted for the complexity limit.
Specs updated: UC-1 (header), UC-4 (`git init` question, adoption and replace questions).

- [x] header line; `git init` question
- [x] `AGENTS.md` adoption and replace questions; tests (scripted stdin, a non-repository directory, an existing
  `AGENTS.md`, declines, `--dry-run`)

- Stage 10.4 `[ ]` — The grouped, summarized interactive plan (UC-1)
