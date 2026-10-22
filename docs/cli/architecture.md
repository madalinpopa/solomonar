# Solomonar CLI v0.1: architecture

Status: accepted, 2026-10-21. How the `solomonar` binary is built. The interface (commands, flags, output, exit codes,
validation) is fixed by [the interface spec](interface.md) (cited as `I §n`); this document does not restate it.
When the two differ, the interface spec wins for behavior and this document wins for structure. Decisions here are
`AD-n`; questions still open are in section 13.

This is an implementation contract. Increment 1 follows it and is released as v0.1.0; [the implementation
tasks](implementation-tasks.md) track its stages. The accepted workflow-audit fixes are
recorded in the [review register](../design/phase1-review-register.md#l-workflow-audit-corrections-2026-10-21), with runtime verification
kept separate from completion of the design changes.

Contents: 1 Scope · 2 Analysis of the interface design · 3 Toolchain · 4 Architectural decisions · 5 Layout and
packages · 6 Key types · 7 Execution flow · 8 Errors, cancellation, and file safety · 9 Code conventions ·
10 Tooling · 11 Testing · 12 Implementation plan · 13 Open questions · 14 Non-goals and deferred work

## 1. Scope

Requirements come from the interface spec and `assets/framework/rules.md` §7, §11. Nothing is added here.

| Requirement | Source | Consequence for the architecture |
|---|---|---|
| Guard and oracle, never a driver | I §1, J1 | No agent, model, daemon, or cache code. The only long-running process is `dashboard`. |
| Registry on `refs/heads/workflow-state`, compare-and-swap writes, no checkout | rules §7, I §6 UC-21 | Git plumbing through the `git` binary; a read-validate-write loop that re-runs validation on a lost swap. |
| Fact checks against Git before every write | I §8.3, J2 | Task validation needs a repository, not only a record. |
| Human and `--json` output with the same content, one envelope | I §3.3, §3.4 | Every command returns data; one renderer chooses the form. |
| Exit codes 0 to 5, actionable errors with check, found, fix | I §3.5, §3.6, §8.4 | One error type carries the code, the exit class, and the fix. |
| Embedded framework and catalog, copied then owned, manifest hashes | I §4.5, §4.7 | `embed.FS` plus a plan/apply engine over project files. |
| Prompts only on a terminal; never with `--json` | I §3.7, CLI-D5 | Prompts read from an injected reader; the multi-select enters raw mode through an injected `Env.RawMode`. |
| Optional network: `git push`, read-only `gh` | I §1, J6 | Time-limited external calls whose failures become warnings. |
| macOS and Linux; `go install` and release binaries | AD-11, AD-12 | No Windows path, quoting, or terminal code in v0.1. |

## 2. Analysis of the interface design

The interface spec is complete and consistent enough to build from: every command has its arguments, output,
rejections, and exit code. The points below are where it affects the build, ordered by impact.

| # | Finding | Proposal | State |
|---|---|---|---|
| F1 | Prior status after `blocked`/`paused` was derived from registry history: a walk of `workflow-state` commits on every `unblock` or resume. | A task is `reserved` exactly while at `setup`, so the prior status follows from the stage. | Decided, CLI-D16; applied to rules §7 and I §8.1. |
| F2 | Positionals come before flags (`task move W-001 --stage …`), and shared flags appear before and after the command. The standard `flag` package stops at the first non-flag argument. | Interspersed parsing helper over `flag.FlagSet` (AD-2); `--json`, `-v`, `--no-color`, `-C` registered on every command's set. | Adopted. |
| F3 | "A partial file write is rolled back" (I §3.5) for commands that write tens of files. | Plan first, apply with a journal, roll back the files this run touched (AD-6). | Adopted. |
| F4 | An unmanaged file identical to what `init` would write is a conflict, so a re-run after a crash fails on the run's own files. | Treat identical content as `unchanged (adopted)` and record it in the manifest. | Decided, CLI-D17. |
| F5 | `upgrade --diff` needs a text diff; the standard library has none. | `git diff --no-index --no-color <tmp-embedded> <path>`. | Adopted. |
| F6 | The guard (UC-22) recognizes registry writes inside arbitrary shell commands. | Best effort: resolve protected Git ref paths for file tools and recognize mutating shell operations on `workflow-state`. Ordinary product `state.json` files and read-only inspection are allowed; test real payloads. | Adopted; narrowed by audit F-06. |
| F7 | `--runtime` defaults to "detected on `PATH`" also with `--yes`. | Keep; the plan and the JSON `choices` show the detected list. | No change. |
| F8 | Interrupt handling is not specified. | The exit code reports what was written: 128 + signal before the swap, 0 with `push_interrupted` after it. | Decided, CLI-D18. |
| F9 | CLI-D8 (binary name) was still "deferred". | Root `main.go` (AD-13). | Decided, CLI-D8 c; applied. |
| F10 | `already_initialized` is reused for a rejected downgrade (UC-9), and with strict decoding (AD-7) any command would report a newer registry as `registry_invalid`. | `cli_outdated` for every command that reads the manifest or the registry, checked before decoding. | Decided, CLI-D19. |

Clarifications the implementation settles this way unless the spec says otherwise:

- "Active" for `owner_in_use` means `reserved`, `running`, `blocked`, `paused`.
- A plan followed by a prompt goes to stderr, so stdout carries only results; `--dry-run` plans go to stdout.
- `spec_ref_not_final` takes the first `spec(<ID>):` subject in `git log --topo-order --format="%H %s" <merge>^2`
  (topological order, not commit date), never `--grep`, which also matches body lines.

## 3. Toolchain

- **Go 1.27.3**, the latest stable patch (go.dev/doc/devel/release, 2026-10-19), as `go.mod` declares.
- **One production dependency** in increments 1 and 2 (AD-1): `golang.org/x/term`, for raw terminal mode in the
  multi-select (CLI-D5; author, 2026-10-21); everything else is the standard library. The only other module
  dependency is test-only: `github.com/stretchr/testify` v1.12.1, packages `assert` and `require` (AD-14), added with
  the first test.
  Development tools are pinned in `Taskfile.yml` (section 10) and never enter `go.mod`.

| Need | Standard library | Note |
|---|---|---|
| Commands and flags | `flag.FlagSet`, custom `flag.Value` | Repeatable and comma-separated flags (`--agent`, `--runtime`, `--ext`) as one `listFlag` type. |
| Embedded assets | `embed`, `io/fs`, `text/template` | The `docs/README.md` template already uses `{{ .ProjectName }}`. |
| Project file I/O confined to the repository | `os.Root` (`OpenRoot`, `WriteFile`, `MkdirAll`, `Rename`, `Remove`) | Manifest paths and extension sources are editable input; `os.Root` rejects `..` and symlink escapes. |
| Git and `gh` | `os/exec`, `context` | `exec.CommandContext` kills the child on cancel. |
| Hashes | `crypto/sha256` | Manifest file hashes. |
| JSON | `encoding/json/v2` for reading `state.json` and the manifest | Rejects duplicate keys by default and can reject unknown members (AD-7). |
| Signals | `os/signal.NotifyContext` | One place, in `main`. |
| Raw terminal mode | `golang.org/x/term` (`MakeRaw`, `Restore`), the one non-standard package | The multi-select only (CLI-D5); set in `main` as `Env.RawMode` (AD-1). |
| Sorted iteration | `slices.Sorted(maps.Keys(...))`, `cmp` | Task IDs sorted by number, not by string. |
| Dashboard (increment 3) | `net/http`, `embed`, `httptest` | Loopback by default. |

Not used, deliberately: generics beyond standard helpers, reflection, goroutines outside the dashboard server and
tests, `log` and `log/slog` (the CLI writes user output, not logs; `-v` traces external commands to stderr).
Section 10 enforces these with lint rules.

## 4. Architectural decisions

| ID | Decision | Rationale | Rejected alternative |
|---|---|---|---|
| AD-1 | One production dependency for increments 1 and 2: `golang.org/x/term`, for raw terminal mode in the multi-select (CLI-D5; author, 2026-10-21); everything else is the standard library. | The standard library covers every need in section 3 except raw terminal mode; the framework treats each dependency as a decision (rules §3). | Cobra or Kong (AD-2); go-git (AD-3); a full-screen prompt library (CLI-D5). |
| AD-2 | `flag.FlagSet` per leaf command, a static command table, hand-written help, and an interspersed-arguments helper (about 40 lines). | The interface fixes every help line, error envelope, and exit code, so a framework's generated help and error handling would be overridden, not used. The tree is two levels deep with one alias. | Cobra: interspersed flags for free, but pflag, `SilenceErrors`/`SilenceUsage`, custom exit mapping, and a dependency for about 20 commands. Reconsider only for shell completion. |
| AD-3 | Git and `gh` through their executables, never a Go Git library. | Atomic `update-ref <new> <old>` and `git worktree list --porcelain` are the contract (rules §7); the user's Git configuration and identity apply; Git ≥ 2.40 is already required. | go-git: no worktree parity, a second implementation of the swap semantics, a large dependency. |
| AD-4 | Commands return data or a typed error; one renderer writes human or JSON form; `main` alone calls `os.Exit`. | One place enforces the envelope (I §3.4) and the exit table (I §3.5); packages stay testable. | Printing inside domain code: the two output forms would drift. |
| AD-5 | Registry writes as `Update(ctx, fn)`: read tip, run `fn` on a copy, build the tree, swap; on a lost swap re-read and re-run `fn`, up to 5 attempts. | Re-running the whole validation on the new tip is exactly UC-21; callers cannot forget it. | Validate once, retry only the swap: wrong after a concurrent claim of the same ID. |
| AD-6 | Every file-writing command computes a `Plan` before writing; `Apply` writes temp + rename inside `os.Root`, journals the previous content, and restores it on handled failure. Include the manifest in that file plan and finalize it before any registry CAS. Keep rollback available until the CAS succeeds. | `--dry-run`, conflict detection, and rollback share one structure; no fallible project-file write remains after the registry commit point (audit F-09). | Writing while deciding, or a manifest write after CAS: no truthful "nothing written" guarantee on failure. |
| AD-7 | `state.json` and the manifest are decoded strictly and validated after decoding; written with stable key order and a trailing newline. Their version fields are read first, and a newer version is `cli_outdated` (CLI-D19) before strict decoding runs. | The registry is the authority; a hand edit or a newer schema fails loudly (`registry_invalid`, `manifest_invalid`) instead of losing fields on the next write. | Lenient decoding: unknown fields vanish on rewrite. |
| AD-8 | Task rules are pure functions over the record; fact checks are a separate step that takes the repository. | The I §8.1 and §8.2 tables are tested without Git; `allowed_next` (I §9.3) comes from the same tables, so validation and the oracle cannot disagree. | Rules and Git calls interleaved: every rule test needs a repository. |
| AD-9 | Process inputs (args, standard streams, environment lookup, clock, terminal detection, embedded assets) reach `cli.Run` as one `Env` value. | In-process tests of every use case, without building the binary or touching `os` globals. | Reading `os.Args`, `os.Getenv`, `time.Now` inside commands. |
| AD-10 | CLI version from `runtime/debug.ReadBuildInfo`: `go install …@vX.Y.Z` records the module version, and `go build` in a checkout stamps it from the Git tag (with `+dirty` for local changes). Framework version from the embedded `rules.md` header. | No version constant and no `-ldflags -X` to forget. | A hand-maintained `const Version`. |
| AD-11 | Platforms: macOS and Linux. Windows is deferred. | Windows changes hook command quoting, path handling in the guard, and terminal detection; no user needs it yet. | Windows from the start. |
| AD-12 | Distribution: `go install github.com/madalinpopa/solomonar@vX.Y.Z` now; release binaries for linux and darwin on amd64 and arm64, built by GoReleaser with `-trimpath` from a `vX.Y.Z` tag on `main`, and a Homebrew cask in `madalinpopa/homebrew-tap` that GoReleaser regenerates on each stable release (section 10). | AD-10 makes both carry the right version without extra flags. A release tool is a build tool and stays out of `go.mod`. The cask comes from the same release, so it never drifts from the published archives. | A hand-written formula or cask in the tap. |
| AD-13 | The entry point is `main.go` at the module root, not `cmd/solomonar/main.go`; it also holds `//go:embed assets` and passes the tree to `cli.Run` in `Env`. | The repository holds one binary, which is the official layout for "a command with supporting packages" (go.dev/doc/modules/layout); the install path shortens to the module path; the `assets` package and its global disappear. | `cmd/solomonar/`: meant for repositories with several binaries. A `cmd` package holding the command handlers (the Cobra convention): in Go, `cmd/<name>` means a `main` package, so it misleads readers and `go install ./cmd/...`. |
| AD-14 | Tests use testify: `require` when the test cannot continue after a failure (errors, setup), `assert` for the remaining checks. `mock` and `suite` are not used. | Author decision, 2026-10-21. Shorter, uniform assertions with readable diffs (`assert.JSONEq` for the `--json` envelopes). Without `mock` and `suite`, tests keep real repositories and plain test functions (section 11). | The standard `testing` package alone. |

## 5. Layout and packages

Layout (AD-13):

```
main.go                  signal context, //go:embed assets, Env.RawMode, cli.Run, os.Exit; nothing else
assets/                  framework/, extensions/, templates/ (existing content)
internal/
  cli/                   command table, flag parsing, help, prompts, rendering, error envelope, exit codes
  task/                  registry model, transition and field rules, fact checks, status and drift, CAS store
  project/               manifest, file plans and apply, catalog and rendering, adapters, decisions
  git/                   git and gh process runner, typed helpers, -v tracing
  dashboard/             HTTP server and embedded page (increment 3 only)
Taskfile.yml  .golangci.yml  .goreleaser.yaml  go.mod
.github/                 CI and release workflows, Dependabot
```

| Package | Owns | Why it is a separate package |
|---|---|---|
| `main` (root) | Process boundary: `signal.NotifyContext`, the embedded `assets` tree, building `cli.Env` from `os` (`RawMode` from `term.MakeRaw`/`term.Restore`), `os.Exit(code)`. | `go install github.com/madalinpopa/solomonar` yields the binary `solomonar`. |
| `internal/cli` | Everything the user sees: argument parsing, help, prompts, human tables, JSON envelope, mapping errors to exit codes, `doctor` and `guard` assembly. One file per command group (`task.go`, `init.go`, `extension.go`, …). | The only package that knows about terminals and output forms; `task` and `project` never print. |
| `internal/task` | The registry's meaning: `Registry`, `Record`, statuses, stages, tiers, blockers; the rules of I §8; fact checks of I §8.3; `Status` facts and drift of I §9; the swap loop. | The core domain with real invariants; most rule tests live here. |
| `internal/project` | The project's files: manifest, ownership classes, plan/apply, catalog loading, extension rendering, adapter stubs, `AGENTS.md` sections, decision entries. | Different data (files, not records), different failures (conflicts, not rejections), different tests (file trees). |
| `internal/git` | Running `git` and `gh`: context, working directory, stdin, trimmed output, errors with stderr, `-v` trace. Typed helpers only where used twice. `OnPath(program string) bool` is the `PATH` lookup for runtime detection (only `git` may import `os/exec`, AD-3). | Both `task` and `project` need it; `exec` details and tracing stay in one place. |
| `internal/dashboard` | `net/http` handlers for `/`, `/api/state`, `/api/status/<ID>`, and the embedded page. | HTTP is a distinct boundary; deferred to increment 3. |

Dependency direction; nothing imports `cli`, `git` imports no internal package, and `.golangci.yml` enforces all
three (section 10):

```
main → cli → task      → git
           → project   → git
           → dashboard → task
```

`task` does not import `project`: `cli` passes the core delegated roles plus extension agents whose manifest
state is `installed` for `--agent` validation (I §8.3) as a `[]string`; draft agent files are excluded.
`project` receives the embedded tree as an `fs.FS`, so its tests use `testing/fstest.MapFS`.

Not created, and why: no `pkg/` (no public API); no `config` package (settings are three manifest fields read by
`project`); no `output` or `ui` package (rendering is `cli`); no package per command; no `doctor` or `guard`
package until either outgrows a file in `cli`.

## 6. Key types

Sketches that fix names and ownership, not final code.

```go
// internal/cli
type Env struct {
	Args           []string
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	Getenv         func(string) string
	Now            func() time.Time
	Interactive    bool
	RawMode        func() (restore func(), err error) // main: term.MakeRaw/term.Restore on stdin
	Assets         fs.FS
}

func Run(ctx context.Context, env Env) int

type Error struct {
	Code              string
	Exit              int
	Message           string // error: <Message> (<Code>); the JSON message
	Check, Found, Fix string
	Err               error
}

type runFunc func(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error)

type command struct {
	name, summary, usage, description string
	bind                              func(fs *flag.FlagSet) runFunc // registers the command's own flags
}
```

```go
// internal/task
type Registry struct {
	SchemaVersion int                `json:"schema_version"`
	Tasks         map[string]*Record `json:"tasks"`
}

func (r *Registry) IDs() []string // numeric order: W-999 before W-1000

type Record struct {
	Title        string      `json:"title"`
	Status       Status      `json:"status"`
	Stage        Stage       `json:"stage,omitempty"`
	Owner        string      `json:"owner,omitempty"`
	Branch       string      `json:"branch,omitempty"`
	Spec         string      `json:"spec,omitempty"`
	Revision     string      `json:"revision,omitempty"`
	Tier         *int        `json:"tier,omitempty"`
	ActiveAgents []Agent     `json:"active_agents,omitempty"`
	Blockers     []Blocker   `json:"blockers,omitempty"`
	NextAction   *NextAction `json:"next_action,omitempty"`
	PR           string      `json:"pr,omitempty"`
	// depends_on, related_task, summary, spec_ref, merge_commit, updated_at …
}

func (r *Record) Move(m Move, roles []string, now time.Time) (changed []string, err error)

func SchemaOf(data []byte) (int, error)

type Store struct {
	Git    *git.Repo
	Remote string
}

func (s *Store) Update(ctx context.Context, fn func(*Registry) (subject string, err error)) (Commit, error)
```

`SchemaOf` reads the registry's `schema_version` with the header validation and no newer-than check; the strict
decoder builds on it, and `version` calls it alone (UC-24).
`Tier` is a pointer because tier 0 is valid and "no tier yet" must stay distinguishable. `task` and `project` each
define their own `Rejection{Code, Check, Found, Fix}`; a shared package for one struct would be a `common` package
in disguise. `cli` recognizes both with `errors.As` and maps codes to exit classes with one table.

```go
// internal/project
type Action string // create, update, append (ActionAppend), unchanged, keep, remove, conflict, skip (I §3.3)

type Change struct {
	Path    string
	Action  Action
	Owner   string
	Content []byte
	Reason  string
}

type Plan struct{ Changes []Change }

func (p Plan) Conflicts() []Change

type Want struct {
	Path, Owner string
	Content     []byte
	CreateOnly  bool
}

func Classify(root *os.Root, m *Manifest, wants []Want, keep, overwrite []string) (Plan, map[string]File, error)

type Choices struct {
	Name, Objective, FrameworkVersion string
	Extensions                        []CatalogExtension
	Documents                         []string // README index lines (I §4.3)
}

func CoreWants(assets fs.FS, c Choices) ([]Want, error)
func RuntimeWants(assets fs.FS, runtimes []string, models map[string]string, framework string,
	extensions []CatalogExtension) ([]Want, error)

func ManifestVersions(data []byte) (schema int, framework string, err error)

type CatalogExtension struct {
	Name, Kind, Group, Title string
	Requires, Conflicts      []string
	AppliesWhen              string
	Paths                    []string
	Dir                      string
}

type Catalog map[string]CatalogExtension

func Groups() []string
func LoadCatalog(assets fs.FS) (Catalog, error)
func (c Catalog) Resolve(names []string) ([]string, error)
func Render(content []byte, self string, installed []string, decision string) ([]byte, []string)
func ExtensionWants(assets fs.FS, installed []CatalogExtension, decisions map[string]string) ([]Want,
	map[string]Extension, error)

type Decision struct {
	ID, Group string
	Selected  []string
}

type Decisions struct {
	Changes []Change
	Files   map[string]File
	Records []Decision
	ByGroup map[string]string
	Index   []string
}

func PlanDecisions(root *os.Root, existing *Manifest, installed []CatalogExtension, date string) (Decisions, error)

type Requested struct {
	Name, MainBranch, Remote string
	Runtimes, Extensions     []string
	AdversarialModels        map[string]string
}

func (m *Manifest) CheckChoices(r Requested) error
func InitPlan(root *os.Root, existing *Manifest, wants []Want, keep, overwrite []string, p Project,
	runtimes map[string]Runtime, extensions map[string]Extension, decisions Decisions,
	framework string) (Plan, *Manifest, error)

type Journal struct{ /* each touched path's previous content and mode, or its absence */ }

func Apply(ctx context.Context, root *os.Root, p Plan) (*Journal, error)
func (j *Journal) Rollback() error
```

`Classify` applies I §4.5's rules and returns the plan with the manifest `files` entries for the wanted paths.
`CoreWants` returns the core files from `framework/` as `docs/solomonar/`, then `AGENTS.md` and `docs/README.md`
from `templates/`, rendered with `text/template`, with the `AGENTS.md` extension lists of I §4.6 from
`Choices.Extensions`.
`RuntimeWants` returns the adapter stubs of I §4.2 per runtime, in the order `claude`, `codex`, extension skills and
agents sorted with the core ones; a `CreateOnly` want (`CLAUDE.md`) present and unmanaged is `skip`, never a conflict,
and `InitPlan` records `runtimes` and `extensions` (nil records `{}`) in the manifest.
`InitPlan` runs `Classify`, puts the `decisions` changes after the classified wants, merges their files into the
manifest built from the entries, and appends the manifest change last (I §4.5); it turns a present `docs/README.md`
`skip` into `append` with `decisions.Index` under `## Documents`, or keeps the `skip` with the reason (I §4.3).
`PlanDecisions` plans one `P/D-<n>` entry per stack group of the selection and the group document lines (I §4.3):
`create` or `append` per document, `unchanged` for the existing manifest's `decision` files on a re-run; `Records`
feeds the JSON `decisions` (I §5 UC-1), `ByGroup` the `ExtensionWants` decisions, `Index` the README lines for the
documents it creates (`Choices.Documents`).
`Groups` lists the extension groups in I §4.3 order.
`LoadCatalog` decodes and checks every `extensions/<group>/<name>/extension.json` (I §4.3, a defect is `read catalog
<dir>: …`), `Resolve` returns the sorted unique selection or the first of its three rejections, and `Render` keeps
the satisfied `## With` sections, adds the provenance line, and returns the kept names (`rendered_with`).
`ExtensionWants` returns the installed files of each extension (I §4.3), rendered for the installed set and ordered
by extension name, then file path, and their manifest entries; only a skill directory's top-level `extension.json` is
excluded. `decisions` maps a stack group to its decision ID: a stack-group extension without a non-empty one is
`render <name>: no decision for group <group>`, and `workflow` and `review` ignore any given one (`Installed by
solomonar`, no manifest `decision`); an unreadable file is `read extension <name>: <err>`, and an extension's
frontmatter error in `RuntimeWants` is `read frontmatter <dir>/<file>: <reason>`.
`ManifestVersions` reads the manifest's `schema_version` and `framework_version` with the header validation and no
newer-than check; the strict decoder builds on it, and `version` calls it alone (UC-24).
`CheckChoices` rejects a given choice that differs from the manifest with `already_initialized` (UC-3), adversarial
models after extensions; an empty string, nil list, or nil map means not given.
`Apply` returns its `Journal` so the caller can still roll back a successful apply until the registry CAS
succeeds (AD-6, F-09).

## 7. Execution flow

**Task write** (`task move W-001 --revision … --tier 2 --stage specialized_review`):

1. `main` builds the signal context and `Env`; `cli.Run` parses shared flags, finds `task move`, lets it register
   its own flags (`bind`) beside the shared ones, and parses the rest with the interspersed helper. Malformed input
   stops here: exit 2.
2. `cli` locates the repository (`git rev-parse --git-common-dir` from `-C` or the working directory), reads the
   manifest for `main_branch` and `remote` (`main` and `origin` when absent; newer or invalid rejects before the
   registry, nothing written), and resolves `--as` from the flag or `SOLOMONAR_OWNER`.
3. `store.Update(ctx, fn)`: read the tip and `state.json`; `fn` looks up the record, rejects a done record
   (`done_final`), checks ownership, applies `Record.Move`, then runs the fact checks (`git merge-base
   --is-ancestor`, worktree spec lookup). A rejection returns at once: exit 3, nothing written. A move that changes
   nothing skips steps 4–5.
4. The store writes `hash-object` → `mktree` → `commit-tree -p <old>` → `update-ref <new> <old>`. A lost swap
   re-reads the tip and re-runs `fn`; five losses give `cas_exhausted` (exit 4).
5. With a remote, `git push <remote> workflow-state` under a time limit; failure is the warning `push_failed`,
   exit stays 0.
6. `cli` renders the UC-13 lines, or the envelope with the UC-13 `data`.

With `--dry-run`, step 3 reads the registry and applies the change to it without `Update`; nothing is committed.

**Task read** (`task status W-001`): read the registry; collect the I §9.1 facts with one Git command each; `gh`
only without `--no-forge` and when installed; compute drift and `allowed_next` from the rule tables; render.
Exit 0 whatever the drift.

**Project write** (`init`, `extension add`, `sync`, `upgrade`, `runtime add|remove`): load the manifest (absent
for `init`); take choices from flags, then prompts when interactive; build the `Plan` from the embedded assets and
the files on disk; show it; stop on conflicts (exit 4) or after `--dry-run`; confirm, only when the plan creates or
updates a file or, for `init`, creates the registry (`interactive_required` without `--yes` and without a terminal);
`Apply` every project file, including the manifest. For `init` create the registry, or for a schema-changing
`upgrade` migrate it, only after all file writes succeed. Keep the file journal until that CAS succeeds, rolling
back this run's changes on a handled failure. With an existing valid init registry, reuse it unchanged; successful
file apply is then the local commit point. Publish only after local success; print `Next:`.

For init, the selection is the requested `--ext` names, else the manifest's, else `go` for a root `go.mod`, after
`Resolve`; `PlanDecisions` feeds `ByGroup` to `ExtensionWants` and `Index` to `CoreWants`, and the core, extension,
and `RuntimeWants` wants go with the decisions to `InitPlan`. Validate an existing registry and its version before
any file apply. A valid registry is reused; an invalid one is rejected without touching either the registry or
project files (I UC-3). On `file_conflict`, `render` still prints the plan text (`Plan` and the conflict lines) on
stdout before the error on stderr; JSON keeps `data` absent.

## 8. Errors, cancellation, and file safety

- Expected failures are values: a `Rejection` for user-correctable checks, `cli.Error` at the boundary.
  Unexpected failures are wrapped with `%w` and context (`read workflow-state: %w`) and become exit 1.
- Each error is reported once, by `cli`; packages never log. `errors.As` is used only in `cli`. No error hierarchy
  beyond these types.
- Git and `gh` failures carry the command, exit status, and trimmed stderr; with `-v` the command line is traced
  before it runs, as `+ git …` or `+ gh …` (I §3.1).
- Cancellation (CLI-D18): one context from `signal.NotifyContext(os.Interrupt, syscall.SIGTERM)`; after it is
  done, `main` calls `stop()` so a second signal ends the process. Local Git commands run without a time limit;
  `git push` and `gh` get 30 seconds each. Before the registry swap, cancellation rolls back `Apply` and exits
  128 + signal with `interrupted`. The swap is the commit point: no project-file write remains after it; only
  optional publication and result rendering remain. An interrupted push is `push_interrupted` with exit 0.
  A project-only command, including init reusing an existing valid registry, commits locally when its entire
  file plan succeeds. Journal rollback covers handled errors/signals, not SIGKILL or machine crashes; repeated
  init recovers by adopting identical files and preserving an existing registry (I UC-3).
- Files: every project path goes through one `os.Root` opened at the worktree root. Nothing outside the repository
  is written except Git's own objects and refs.
  - `Apply` acts only on `create`, `update`, `append`, and `remove`, in plan order, checking the context before each
    change; `append` writes the full new content, as `update` does. A plan that disagrees with the disk is an error:
    `create` on an existing path, or `update`/`append`/`remove` on a missing or non-regular path (checked with
    `Lstat`).
  - Each write is a temp file `.<name>.solomonar-tmp` in the target's directory, chmodded to its exact mode
    (permission bits only, independent of the umask), then renamed over the target; updates keep their mode, new
    files get `0o644`. Missing parent directories are created `0o755` one level at a time, each journaled.
  - The journal keeps each touched path's previous content and mode, or its absence. Errors are wrapped
    `apply <path>: %w` (cancellation wraps the context error); a failed or cancelled `Apply` rolls back
    automatically, joins any rollback failure into the error, and returns no journal.
  - `Journal.Rollback` restores in reverse order through the same temp-file-and-rename, deletes created files, and
    removes a created directory only when empty. Entries that fail to restore stay in the journal for the next
    `Rollback`; after a full rollback the journal is empty, so a repeated call is a no-op.
- `init` creates the registry only when the ref is absent (`update-ref <ref> <new> ""` is create-only); it never
  overwrites one. On any failure before that CAS succeeds (a failed create, cancellation) it calls
  `Journal.Rollback` and returns the error (`interrupted` or `unexpected_error`) with the rollback error joined.
  With `--git-init`, a failure after `git init` rolls back the project files but leaves the new `.git`; a re-run
  adopts the files (I UC-4).

## 9. Code conventions

- Code explains itself through names, small functions, and types. No comment restates what the code does.
- A comment exists only for what the code cannot say: a non-obvious reason or an external constraint, cited
  briefly (`I §8.1`, a Git behavior).
- Doc comments only where the name and signature leave the behavior unclear: one short sentence that starts with
  the name, no further explanation. No package comments are required.
- Errors use lowercase messages without trailing punctuation and name the failed operation: `read manifest: %w`.
- Tests are named for behavior (`TestMoveRejectsShortRevision`); table cases carry a `name` field.

Lint enforces the form, never the presence, of comments: `godoclint` checks existing doc comments (start with
the name, at most 100 characters per line), and nothing requires one (revive `exported` and staticcheck
`ST1000` are off; the `comments` exclusion preset is on). Brevity stays a review rule.

## 10. Tooling

`Taskfile.yml` is the single entry point for local development; it pins the tool versions and installs them into
`.bin/` with `go install`, so they are built with the project's Go toolchain and never enter `go.mod`.

| Task | Runs |
|---|---|
| `task` | Lists the tasks. |
| `task build` | `go build -trimpath -o bin/solomonar`, skipped when sources are unchanged. |
| `task install` | `go install -trimpath`. |
| `task run -- <args>` | The CLI from source. |
| `task test` | `go test -race ./...`. |
| `task cover` | Tests with a coverage profile and the total. |
| `task fmt` | `golangci-lint fmt` (gofumpt, goimports with the module as local prefix). |
| `task lint` | `go mod tidy -diff`, then `golangci-lint run` (includes `go vet`, `gosec`, format checks). |
| `task security` | `govulncheck ./...`. |
| `task check` | `lint`, `test`, `security`: the commands CI runs. |
| `task tools` | Installs golangci-lint v2.15.0 and govulncheck v1.9.0 into `.bin/`; skipped when the pinned versions are there. `task tools --force` rebuilds after a Go upgrade. |
| `task clean` | Removes `bin/` and coverage output. |
| `task release:patch`, `release:minor`, `release:major` | Fetch the tags, bump the latest stable `vX.Y.Z` (none counts as `v0.0.0`; prerelease tags are ignored), and after a confirmation tag `HEAD` and push the tag, which starts the release workflow. They refuse when `HEAD` is not on `origin/main` or the tag exists. `release:next` is `release:patch`. |

`.golangci.yml` (v2 format) starts from the standard linters (errcheck, govet with all analyzers except
`fieldalignment` and `shadow`, ineffassign, staticcheck, unused) and adds rules that encode this document:

| Rule | Encodes |
|---|---|
| depguard `stdlib-only`: only the standard library, this module, and `golang.org/x/term` outside tests | AD-1 |
| depguard `tests`: tests may also import testify `assert` and `require`, never `mock` or `suite` | AD-14 |
| testifylint (all checkers): `require` for errors, argument order, specific assertions | AD-14 |
| depguard `banned`: no `reflect`, `unsafe`, `log`, `log/slog` outside tests | Section 3 |
| depguard `exec-only-in-git`: `os/exec` only in `internal/git` | AD-3 |
| depguard `nothing-imports-cli`, `task-without-project`, `git-is-a-leaf` | Section 5 dependency direction |
| forbidigo: `os.Exit` outside `main.go`; `os.Args`, standard streams, `os.Getenv`, `time.Now` outside `main.go`; `fmt.Print*`, `print`, `panic`; `exec.Command` without context | AD-4, AD-9, section 8 |
| tagliatelle: JSON tags in `snake_case` | I §3.4 |
| musttag: marshaled structs carry tags | AD-7 |
| exhaustive: switches over `Status`, `Stage`, `Action` cover every value | AD-8 |
| iface (`identical`, `unused`, `opaque`), ireturn, interfacebloat (max 5) | Concrete types first; small consumer interfaces |
| gochecknoglobals, gochecknoinits | No global mutable state (the `//go:embed` variable is allowed) |
| errorlint | `%w`, `errors.Is`, `errors.As` |
| usetesting, thelper | `t.TempDir`, `t.Setenv`, `t.Context`, `t.Helper` (section 11) |
| gocognit (20, not in tests), nakedret (none), revive (flow and naming rules), gocritic, modernize, unparam, unconvert, wastedassign, nilerr, predeclared, usestdlibvars, misspell (US) | Readable, modern, self-explaining code (section 9) |
| godoclint (basic, `max-len` 100) | Section 9 |
| gosec, with `G204` excluded in `internal/git` and file modes `0755`/`0644` | Running `git` with arguments is the design (AD-3); project files are ordinary repository files |
| nolintlint: every `//nolint` names its linter and gives a reason | Exceptions stay visible |

### CI and release

GitHub Actions run the `task check` commands through the official actions, with the same pinned tool versions.
Actions are pinned to commit SHAs; Dependabot updates them and the Go modules weekly, after a 7-day cooldown.

| Workflow | Trigger | Jobs |
|---|---|---|
| `ci-solomonar.yml` | pull request, push to `main`, manual, called by release | In parallel: `lint` (`go mod tidy -diff`, golangci-lint), `test` (`go test -race` with coverage), `vulncheck` (govulncheck), `build` (`goreleaser release --snapshot` for every target, then `version` and `help` on linux/amd64). |
| `ci-solomonar.yml` | weekly schedule | `vulncheck` only. |
| `ci-release-solomonar.yml` | push of a `v*.*.*` tag | `ci` reruns every check on the tagged commit; `verify` requires a semver tag, a commit on `main`, and no existing release; then `release` runs GoReleaser. |

`.goreleaser.yaml` builds linux and darwin on amd64 and arm64 with `CGO_ENABLED=0`, `-trimpath`, `-s -w`, and the
commit time as file timestamp, then writes `tar.gz` archives, SHA-256 checksums, and GitHub-generated release notes;
a tag with a prerelease suffix publishes a prerelease. It also writes the `solomonar` cask to
`madalinpopa/homebrew-tap` with the `HOMEBREW_TAP_TOKEN` secret; a prerelease leaves the cask on the last stable
version, and a macOS postflight step removes the quarantine attribute because the binaries are not notarized. `dist/` is ignored by Git, since an untracked file would stamp
the version `+dirty` (AD-10). Only the `release` job can write, and it builds without a Go cache.

## 11. Testing

| Layer | What | How |
|---|---|---|
| Rules | I §8.1 and §8.2 tables, field rules, ID allocation, blocker IDs, `allowed_next` | Table-driven tests in `task`, no Git. One row per allowed and per rejected transition. |
| Registry I/O | Strict decoding, schema validation, stable encoding | Table tests with JSON inputs, including duplicate and unknown keys. |
| Git facts | Fact checks (I §8.3), `task status` facts and drift (I §9) | Real repositories in `t.TempDir()` built by helpers (`newRepo`, `commit`, `worktree`), isolated with `t.Setenv("GIT_CONFIG_GLOBAL", …)` and author and committer variables. |
| Concurrency | The swap loop (UC-21) | Two goroutines register against one repository; both succeed with distinct IDs; run with `-race`. A second test: re-validation rejects a concurrent claim of the same planned ID. |
| Project files | Plans, conflicts, the upgrade table (UC-9), `With` sections, adapters | `fstest.MapFS` as the catalog; temp directories as projects; assert on `Plan` values, then on files after `Apply`; rollback tested with a write that fails midway. |
| Use cases | Every UC of the increment, human and JSON forms, exit codes | In-process `cli.Run` with buffers, a fixed clock, and `os.DirFS` of `assets/`; JSON compared with `assert.JSONEq`; a few human outputs as golden files in `testdata/`, refreshed with `-update`. |
| Prompts | Text, confirmation, multi-select keys and redraws, EOF, Ctrl-C, `interactive_required` | `Env.Interactive`, a no-op `Env.RawMode`, and byte streams as stdin. |

Assertions use testify (AD-14); test helpers that build repositories take `t`, call `t.Helper()`, and use
`require`. Tests require `git` ≥ 2.40 on `PATH` and fail with that message otherwise. `gh` is never called in tests:
`--no-forge`, or a fake `gh` script placed first on `PATH`.

### Accepted workflow-audit scenarios

These are implementation acceptance checks, not claims that the current CLI already passes them. Keep the
contract walkthrough and the runtime smoke results distinct in the audit report.

| Audit | Scenario | Required result |
|---|---|---|
| F-01, F-03 | README-only correction in an empty product repository; maintenance in an existing application with unknown run commands | Assigned documentation is permitted; no new application baseline is added. The first authorized executable outcome still establishes only its needed baseline. |
| F-02 | Offered A, local B, `init --keep`, upgrade C, keep B, repeat C, then upgrade D | Base hashes follow A → C; B is preserved, repeated C keeps B, and D conflicts rather than overwriting B. Cover rendered assets and the managed part of AGENTS. |
| F-04 | Main changes a contract used by a feature in a different file; separate control changes only an unrelated logo | CI reruns in both; affected or uncertain reviews rerun for the contract, while unrelated changes permit justified reuse. Use the audit's exporter/policy counterexample. |
| F-05 | Real Claude Code and Codex sessions after init/runtime add, including a feature worktree | The documented invocation discovers the skill, loads the portable source, takes the orchestrator role in the same top-level session, and delegates with the assigned worktree/revision. Record runtime versions; file-tree assertions alone do not pass this check. |
| F-06 | Real hook payloads for product state.json, read-only registry inspection, task move, and direct protected-ref writes | Product writes and authorized CLI/read calls are allowed; direct registry mutations are denied. Include linked-worktree Git paths and packed refs. |
| F-07 | `new → list/show → sync/doctor → add → repeated add`, for a skill and an agent | Draft is visible but inactive and needs no decision; adoption validates metadata and installs once; a repeated identical adoption is a no-op. Test local-directory flags with no terminal. |
| F-08 | Add htmx, add templ, remove templ | Every remaining installed frontend extension points to the approved decision for the full new selection; old decisions remain in history; doctor does not fail on the valid transition. |
| F-09 | Inject errors writing the manifest and at registry CAS; separately fail push after CAS | Manifest failure creates no new ref; CAS failure restores the manifest and other files; successful CAS plus failed push returns local success with warning. Existing valid init registry is preserved, and invalid registry is rejected before file apply. |
| F-10 | Review an io.Reader implementation and a new project-owned cancellable API | The imposed Read signature is accepted; the new API still propagates context. A signature exception is not a blanket exemption from cancellation discipline. |

Manual runtime smoke checks belong to the runtime adapter's increment; they need not turn rule/fixture tests
into model-provider calls. Use the existing human/JSON, rollback, concurrency, and use-case suites for the other
acceptance scenarios instead of duplicating them in another layer.

Deferred: transcript tests (`rsc.io/script` or `testscript`). The use cases are transcripts, so they would fit,
but they are a dependency; reconsider if the in-process tests read worse than the transcripts.

## 12. Implementation plan (increment 1)

Increment 1 of I §10: `init`, every `task` command, `version`, `help`. Each step ends with `task check` green.

This is an internal milestone. The full new-project journey with a stack chosen after discovery also requires
`extension add` in increment 2; label that limitation in demos and release notes.

| Step | Objective and scope | Acceptance criteria | Tests |
|---|---|---|---|
| 1 | **Skeleton.** `main.go` (AD-13), `cli.Run`, command table, interspersed parsing, shared flags, help, `version`, `cli.Error`, envelope, exit codes; testify enters `go.mod` with the first test. | `solomonar version` prints UC-24 outside a project; `solomonar nope` exits 2; `--json` prints one envelope; `-h` on any command prints its help. | Use-case tests for UC-24, usage errors, envelope shape. |
| 2 | **Git runner.** `internal/git`: run with context and stdin, `-v` trace, repository discovery from `-C` and worktrees, Git version check. | The main checkout and a feature worktree resolve the same common directory; Git < 2.40 is reported. | Temp repositories with a worktree. |
| 3 | **Registry model and rules.** `task` types, strict decoding, validation, I §8.1 and §8.2 as tables, CLI-D16 invariant, field rules, ID allocation. No Git. | Every row of I §8.1 and §8.2 is accepted or rejected with its I §8.4 code. | Table tests only. |
| 4 | **Store, `task plan`, `task show`.** Swap loop, commit subjects (I §8.5), push with warning; registry creation as a function (used by tests now, by `init` in step 8). | UC-11, UC-16 (`show`), UC-21 pass; a lost swap re-validates; a newer `schema_version` is `cli_outdated`; a cancel before the swap writes nothing, a cancel during the push exits 0 with `push_interrupted`. | Concurrency test with `-race`; push to a bare remote and to a missing one; cancellation through the test's context. |
| 5 | **`task register`, `task move`.** Ownership, I §8.3 fact checks, every UC-14 rejection. | UC-12, UC-13, UC-14, UC-18 pass in human and JSON form. | Real repositories with `feature/<ID>` branches and worktrees. |
| 6 | **`task block`, `unblock`, `handoff`, `done`.** Blocker IDs, prior status from the stage, merge and `spec_ref` checks. | UC-15, UC-19, UC-20 pass; a squash merge is rejected with `merge_not_merge_commit`. | Repositories with a merge commit, a squash, and a rebased branch. |
| 7 | **`task status`.** Facts, drift, `allowed_next`, optional `gh`. | UC-16 (`status`) and UC-17 pass; without `gh` the forge facts are `unknown`, never passing. | Fake `gh` on `PATH`; each drift row produced by a repository state. |
| 8 | **`init`, non-interactive, core only.** Manifest, plan/apply with rollback, core files, `AGENTS.md` and `docs/README.md` from templates, registry creation, re-run no-op, conflicts, `--keep`/`--overwrite`, `--dry-run`. | UC-2 with `--runtime none --ext none`, UC-3, and the non-interactive UC-4 cases pass; a re-run over files left without a manifest adopts them (CLI-D17); keep preserves the upstream base; manifest apply precedes registry CAS. | File-tree assertions plus F-02/F-09: a failing file write or CAS rolls back this run's entire file plan, including the manifest. |
| 9 | **`init` with runtimes and catalog extensions.** Adapter stubs, `.claude/settings.json` merge, extension rendering, `requires`/`conflicts`, `P/D-n` entries, `docs/stack.md`. | UC-2 with `--runtime claude,codex --ext go`; a repeated `init` stays a no-op. | `fstest.MapFS` catalog with `With` sections; the settings merge keeps foreign keys. |
| 10 | **Interactive `init`.** Line prompts, key-driven multi-select (`golang.org/x/term`), `AGENTS.md` adoption, `--git-init` prompt, `interactive_required`. | UC-1 and the interactive UC-4 branches pass with scripted stdin. | Prompt tests with `strings.Reader`. |

Increments 2 and 3 follow I §10 and are planned in detail after increment 1, because their main inputs (CLI-D9
for Codex, the guard's real hook payloads) are verified only then.

## 13. Open questions

None. Resolved on 2026-10-21: Q1 as CLI-D16; Q2 as CLI-D17; Q3 as CLI-D18; Q4 as CLI-D19; Q5 as AD-11; Q6 as
AD-12; Q7 by moving the interface spec to `docs/cli/interface.md`; Q8 as AD-13 and CLI-D8 c.

## 14. Non-goals and deferred work

- Not built: agent launching, model calls, verdict storage, a daemon, a cache, multiple writable clones of the
  registry, Jujutsu support (I §1, rules §11).
- Deferred to increment 2: `doctor`, `sync`, `runtime`, `extension` commands, `guard`.
- Deferred to increment 3: `upgrade` (with `--diff` through `git diff --no-index`), `dashboard`.
- Deferred until a concrete need: shell completion (and with it AD-2), transcript tests, Git URL extension sources
  (CLI-D6), Windows (AD-11), a configuration file, any environment variable beyond `SOLOMONAR_OWNER` and
  `NO_COLOR`.
