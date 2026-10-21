# Solomonar CLI v0.1: implementation tasks (increment 1)

Tracks the [architecture](architecture.md) §12 plan as reviewable stages of at most 500 changed lines each,
tests included. Each stage is implemented test-first and ends with `task check` green and the author's review.
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

- Stage 2.1 `[ ]` — `internal/git`: run with context and stdin, `-v` trace, errors with stderr, repository
  discovery from `-C` and worktrees, Git ≥ 2.40 check

## Step 3: Registry model and rules (no Git)

- Stage 3.1 `[ ]` — `Registry`/`Record` types, strict decoding, version check before decoding, validation,
  stable encoding
- Stage 3.2 `[ ]` — I §8.1 status and §8.2 stage transitions as tables, CLI-D16 invariant
- Stage 3.3 `[ ]` — Field rules, ID allocation, blocker IDs

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
