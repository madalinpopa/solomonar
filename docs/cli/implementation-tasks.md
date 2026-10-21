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

### Stage 3.4 — `Record.Move` and field rules `[ ]`

- [ ] `Move` input applied to a copy, then `checkTransition`; `changed` fields; `updated_at` from the clock
- [ ] Field rules of I §8.2 and `spec_required`, `revision_required`, `next_required`, `blockers_required`

## Step 4: Store, `task plan`, `task show`

- Stage 4.1 `[ ]` — `Store.Update` swap loop, registry creation, commit subjects (I §8.5), push with warning
- Stage 4.2 `[ ]` — Cancellation (`interrupted`, `push_interrupted`), `cli_outdated`
- Stage 4.3 `[ ]` — `task plan` and `task show` commands (UC-11, UC-16 show, UC-21)

## Step 5: `task register`, `task move`

- Stage 5.1 `[ ]` — `task register` and ownership (UC-12, UC-18)
- Stage 5.2 `[ ]` — `Record.Move` and the I §8.3 fact checks
- Stage 5.3 `[ ]` — `task move` command and every UC-14 rejection (UC-13, UC-14)

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
  F-02/F-09 (UC-2, UC-3, UC-4)

## Step 9: `init` with runtimes and catalog extensions

- Stage 9.1 `[ ]` — Adapter stubs, `.claude/settings.json` merge
- Stage 9.2 `[ ]` — Extension rendering, `requires`/`conflicts`, `P/D-n` entries, `docs/stack.md`

## Step 10: Interactive `init`

- Stage 10.1 `[ ]` — Line prompts and multi-select (I §3.7)
- Stage 10.2 `[ ]` — Interactive `init`, `AGENTS.md` adoption, `--git-init`, `interactive_required` (UC-1, UC-4)
