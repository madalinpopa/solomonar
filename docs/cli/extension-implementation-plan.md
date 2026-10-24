# Extension store, minimal upgrade, and extension commands: implementation plan

Status: approved plan, 2026-10-24; not started. TASK-01 moves the agreed design into the
[interface spec](interface.md) and the [architecture](architecture.md); after that, those documents own the
normative detail and this plan only orders the work.

## Objective

Keep installed and project extensions in one local store, `docs/solomonar/extensions/<name>/`. The CLI activates
an extension by copying catalog files (catalog only), generating runtime stubs, the `AGENTS.md` line, the decision,
and the manifest entry. A minimal `upgrade` moves v0.1.7 projects to the new layout before the extension commands
ship.

## Current state

- Catalog extensions install only at `init`, copied and rendered (`## With` sections, provenance line) into
  `docs/solomonar/skills/<name>/` (`internal/project/extension.go`, `render.go`).
- Stubs read frontmatter from the embedded assets (`runtime.go`); user-only invocation is hard-coded on
  `s.id == "solomonar"` (`runtime.go:76`, `:143`).
- `init` defaults to `go` when `go.mod` exists (hard-coded) and, with a manifest present, recomputes the whole
  wanted state (`internal/cli/init.go:422`). It cannot plan the removal of managed paths that are no longer wanted.
- Manifest validation accepts only `P/D-<n>` as a decision (`validate.go:110`).
- No `upgrade` or `extension` command exists (`internal/cli/run.go`).
- The spec (`interface.md` 4.3–4.7, UC-6, UC-7, UC-9) describes the previous design; 4.4 is stale (the real
  catalog is `go` and `extension-authoring`).

## Scope

In scope: the store layout; the `detect`, `default`, and `invocation` fields of `extension.json`; removal of
Generation; desired-state planning with removals; a minimal `upgrade`; `extension list|add|remove` (alias `ext`);
updates to the `extension-authoring` skill, the spec, the site, and the documentation.

Out of scope: `sync`, `runtime`, `doctor`, `guard`, `dashboard`, `upgrade --diff`, `extension show|new`, Git
sources (CLI-D6), Windows. The registry format and `task migrate` do not change.

## Architectural decisions

| ID | Decision |
|---|---|
| X-1 | Flat store `docs/solomonar/extensions/<name>/`: `extension.json` plus `SKILL.md` or `<name>.md` and support files. It holds **only** installed catalog extensions and project extensions; a repository never carries the whole catalog. |
| X-2 | An installed catalog extension has all its files in the store, `extension.json` included, copied verbatim with owner `ext:<name>` and a hash. A project extension is project-owned: the CLI never writes or tracks a file in its directory. |
| X-3 | Generation is removed: no `## With` sections, `rendered_with`, or provenance line. The decision appears in the `AGENTS.md` line, `` `go` (language, P/D-1): … ``. |
| X-4 | `extension.json` adds `detect` (globs, default `[]`), `default` (bool), and `invocation` (`model` or `user`, default `model`), the same for catalog and project, through one strict loader (`readExtension`). |
| X-5 | Detection suggests and never installs: at `init` it preselects `default` and detected extensions; afterwards `list` marks them `suggested`. Matching runs over `git ls-files --cached --others --exclude-standard`, with an own `**` matcher and no new dependency. |
| X-6 | `invocation: user` adds `disable-model-invocation: true` to the Claude and Copilot stubs and `allow_implicit_invocation: false` for Codex. User-invoked skills get a separate `AGENTS.md` line, not the "load when applies" list. |
| X-7 | One desired-state planner (core, extensions, stubs, `AGENTS.md`, decisions, manifest), used by `init`, `upgrade`, and `extension add/remove`. A managed manifest path that is no longer wanted is `remove` when unmodified and `conflict` when modified; `--keep` untracks it, `--overwrite` deletes it. |
| X-8 | The manifest stays at `schema_version: 1`. `state` is always `installed`; an installed project extension has `source: path` and `path` to its store directory; `rendered_with` and project metadata are no longer written; `decision` also accepts `<task-ID>/D-<n>`. |
| X-9 | `extension add/remove` reject a project whose `framework_version` is older than the CLI with the new code `framework_outdated` (exit 3, fix `run solomonar upgrade`). |
| X-10 | Decision rules stay CLI-D11: `P/D-n` per group, supersession, `--decision` checked as `approved`, lines in `docs/stack.md` and `docs/architecture.md`, rebinding in the manifest and `AGENTS.md`. |

## CLI behavior

```
solomonar upgrade [--keep <path>]... [--take-new <path>]... [-y] [--dry-run] [--json]
solomonar extension list [--group <g>] [--kind <skill|agent>] [--installed] [--json]
solomonar extension add <name>... [--decision <ID>] [--rationale <text>] [--keep <path>]... [--overwrite <path>]... [-y] [--dry-run] [--json]
solomonar extension remove <name>... [--decision <ID>] [--rationale <text>] [--keep <path>]... [--overwrite <path>]... [-y] [--dry-run] [--json]
```

- **`upgrade`**: targets the embedded framework. The plan follows the UC-9 table plus `remove` for old paths;
  project extension files and decision documents are never touched. A downgrade is `cli_outdated`; a missing
  manifest is `not_initialized` (fix `solomonar init`); an unresolved conflict is `file_conflict` (exit 4) and
  nothing is written. A current project with nothing to change is a no-op, exit 0.
- **`list`**: the embedded catalog plus the store plus the manifest. Columns `GROUP NAME KIND SOURCE STATUS
  DECISION SUGGESTED APPLIES WHEN`; `STATUS` is `installed`, `available`, or `invalid: <reason>` (project
  extensions only). Read-only.
- **`add`**: names only, never paths. A name resolves in the catalog, then in the store; present in both is
  `name_collision`, in neither `unknown_extension` (exit 2). An invalid project extension is `invalid_value`. Check
  order follows 4.3: `extension_requires`, `extension_conflict`, `decision_not_found`, `decision_not_approved`. An
  identical selection is a no-op; `add` of an installed extension refreshes its stubs.
- **`remove`**: `core_item`; `extension_requires` on the resulting set. Catalog: delete its store files. Project:
  deactivate only. An emptied group records the selection `none`. Prints
  `warning: code governed by <name> loses its rules`.
- **Common to writing commands**: the plan precedes `Proceed?`, on stderr when a prompt follows (AD-2); without a
  terminal and without `--yes` it is `interactive_required`; `--dry-run` prints the plan to stdout; a `Next:` line
  gives the commit command. No environment variable beyond `SOLOMONAR_OWNER` and `NO_COLOR`.

## Implementation tasks

### TASK-01: Contract in the spec and architecture

- **Objective**: `interface.md` and `architecture.md` describe X-1 to X-10 and the commands above, before any code.
- **Dependencies**: none.
- **Scope**: interface sections 2, 3.1, 3.6 (`framework_outdated`), 4.3–4.7; UC-6 and UC-7 rewritten; UC-9 marked
  with the delivered subset; sections 10 and 11 (new decisions from `CLI-D27`: store, detection, invocation,
  removal of Generation, minimal `upgrade` first; CLI-D10 and CLI-D13 rewritten for the real catalog).
  Architecture section 12 (partial increment 2) and section 5 (`upgrade.go`, `extension.go` in `cli`).
- **Relevant code**: `docs/cli/interface.md`, `docs/cli/architecture.md`.
- **Guidance**: delete the draft, `show`, `new`, metadata-flag, `add <path>`, and Generation text rather than
  marking it deprecated. Keep CLI-D11 unchanged. Examples show real output shapes.
- **Acceptance**: no reference to `state: draft`, `rendered_with`, `solomonar:with`, or `extension new` remains; the
  manifest example validates under X-8.
- **Verification**: `rg -n 'draft|rendered_with|solomonar:with|extension new|extension show' docs/cli` returns
  nothing that describes the previous design; spec review.

### TASK-02: Store layout and removal of Generation

- **Objective**: `init` installs catalog extensions into `docs/solomonar/extensions/<name>/` without rendering;
  stubs and `AGENTS.md` point to the store.
- **Dependencies**: TASK-01.
- **Scope**: `ExtensionWants` copies every file verbatim, `extension.json` included, owner `ext:<name>`; delete
  `Render`, `render.go`, `rendered_with`, and their tests; `extensionsBlock` writes the decision into the line;
  `RuntimeWants` points to the new path; `init` stops reporting `rendered with`. Core assets that name the old
  install path move with it: `assets/framework/reviewer.md:20` and `assets/framework/agents/implementer.md:10-11`
  (`docs/solomonar/skills/<name>/SKILL.md` → `docs/solomonar/extensions/<name>/SKILL.md`), and the
  `## Extensions` text of `assets/templates/AGENTS.md` (no draft sentence; the decision in each line; the
  user-invoked line of X-6 lands in TASK-05).
- **Relevant code**: `internal/project/extension.go`, `render.go`, `runtime.go`, `core.go`,
  `assets/templates/AGENTS.md`, `assets/framework/reviewer.md`, `assets/framework/agents/implementer.md`,
  `internal/cli/init.go`, `init_plan.go` and their tests, `testdata/init_*.txtar` (every transcript that
  installs `go` or `extension-authoring` checks the new paths).
- **Guidance**: an agent's files also live in a directory, `extensions/<name>/<name>.md`. Plan order stays by
  name, then path.
- **Acceptance**: `init --ext go` creates `docs/solomonar/extensions/go/{extension.json,SKILL.md,rules/go/*.md}`;
  the Claude, Codex, and Copilot stubs point to `docs/solomonar/extensions/go/SKILL.md`; the `AGENTS.md` line holds
  `P/D-1`; `SKILL.md` is byte-identical to the catalog source.
- **Verification**: `go test ./internal/project/... ./internal/cli/...`, `task test:integration`, `go vet ./...`.

### TASK-03: Desired-state planner with removals

- **Objective**: one planner in `project` produces the full plan from the choices; managed paths no longer wanted
  become `remove` or `conflict` (X-7).
- **Dependencies**: TASK-02.
- **Scope**: extend `Classify`/`InitPlan`, or add a neighboring function, to classify `m.Files` paths absent from
  the wants; `--keep`/`--overwrite` rules for removed paths; `init` uses the planner with unchanged behavior.
- **Relevant code**: `internal/project/classify.go`, `init.go`, `apply.go` (`ActionRemove` and empty-directory
  cleanup exist), `internal/cli/init.go:422`.
- **Guidance**: `decision`, `seed`, and `manifest` files are never removed. A path missing on disk simply drops
  from the manifest. No interface: a function taking the root, the manifest, the wants, and the flags is enough.
- **Acceptance**: an unmodified, no longer wanted `docs/solomonar/skills/go/SKILL.md` is `remove`; modified, it is
  `conflict` with a reason; existing `init` tests pass unchanged.
- **Verification**: table-driven tests in `internal/project/classify_test.go` with `t.TempDir()`;
  `go test ./internal/project/...`.

### TASK-04: Minimal `upgrade` command

- **Objective**: `solomonar upgrade` moves a v0.1.7 project to the current framework and the new layout.
- **Dependencies**: TASK-03.
- **Scope**: new command in `internal/cli/upgrade.go` with the flags above (`--take-new`, not `--overwrite`); the
  `Upgrade framework <old> → <new>` header, plan lines, counts, `Next:`; command help.
- **Relevant code**: `internal/cli/init.go` (flow, confirmation, journal apply), `prompt.go`, `run.go`,
  `internal/project/manifest.go` (`ReadManifest`).
- **Guidance**: choices come from the manifest; each group's recorded decision is reused without reading
  `docs/decisions.md` (the carry-over in `PlanDecisions` exists). The registry is never touched. Any apply error
  rolls back through the journal.
- **Acceptance**: on a v0.1.7 fixture (schema 1 manifest, `skills/go` with provenance, old stubs),
  `upgrade --yes` moves `go` into the store, updates the stubs and `AGENTS.md`, and writes the current
  `framework_version`; a second run is a no-op; an edited `go/SKILL.md` is a conflict, and `--keep` preserves the
  content while advancing the base; a newer manifest version is `cli_outdated`.
- **Verification**: tests in `internal/cli/upgrade_test.go` on a fixture built in the test with computed hashes;
  process transcript `testdata/upgrade.txtar`; `task check`.

### TASK-05: `detect`, `default`, and `invocation` metadata

- **Objective**: the catalog declares its own preselection and invocation rules.
- **Dependencies**: TASK-02; independent of TASK-04.
- **Scope**: the new fields, strictly decoded in `CatalogExtension`, with the spec's validation messages; the `**`
  matcher and the Git file list; `init` preselection from `default` and `detect` replaces the hard-coded `go.mod`
  rule, and `--ext none` excludes everything; `invocation: user` in stubs. Update
  `assets/extensions/language/go/extension.json` (`detect: ["**/go.mod"]`) and
  `workflow/extension-authoring/extension.json` (`default: true`, `invocation: user`).
- **Relevant code**: `internal/project/catalog.go`, `runtime.go:70-170`, `internal/cli/init.go:264-305`,
  `init_prompt.go`, `internal/git/` (`ls-files`; only `git` imports `os/exec`).
- **Guidance**: the matcher lives in `project`; the file list comes from `cli` as `[]string`, so `project` does not
  depend on Git for detection. Multi-select order is unchanged; only the initial marks change.
- **Acceptance**: `init --yes` in a repository with `go.mod` selects `go` and `extension-authoring`; without it,
  only `extension-authoring`. The Claude stub for `extension-authoring` holds `disable-model-invocation: true` and
  the Codex one `allow_implicit_invocation: false`. An unknown member in `extension.json` stays an error.
- **Verification**: table-driven matcher tests (`**/*.ts`, `go.mod` at the root and in a subdirectory, ignored
  files); `init_ext_test.go`, `init_prompt_test.go`; `task test:integration`.

### TASK-06: `extension list` and store discovery

- **Objective**: the user sees available, installed, invalid, and suggested extensions.
- **Dependencies**: TASK-05.
- **Scope**: the `extension` command group with alias `ext` (through `commandName`); `list` with filters, table,
  and `--json`; a store loader: directories without a manifest entry that the catalog does not own are project
  extensions, validated with `readExtension` over `root.FS()`.
- **Relevant code**: `internal/cli/run.go` (`commandName`, `commands`), `help.go`, `internal/project/catalog.go`.
- **Guidance**: an invalid entry never stops the listing; its reason goes in `STATUS`. No writes, no prompt.
- **Acceptance**: a valid `docs/solomonar/extensions/money/` lists as `available · project`; without
  `applies_when` it lists as `invalid: applies_when is missing`; an uninstalled `go` shows `suggested (go.mod)`;
  `ext list` equals `extension list`.
- **Verification**: `go test ./internal/cli/... -run Extension`; `help_test.go`.

### TASK-07: `extension add` for the `workflow` and `review` groups

- **Objective**: adopt a catalog or project extension that records no decision.
- **Dependencies**: TASK-04, TASK-06.
- **Scope**: name resolution; `requires`/`conflicts` on the resulting set; catalog copy or project activation
  through the TASK-03 planner; confirmation, `--dry-run`, `--keep`/`--overwrite`; `framework_outdated` (X-9);
  `name_collision`, `unknown_extension`. An installed project extension is written to the manifest with
  `source: path` and `path`.
- **Relevant code**: `internal/cli/init.go` (confirmation and apply), `internal/project/catalog.go` (`Resolve`),
  `extension.go`, `runtime.go`.
- **Guidance**: `RuntimeWants` and `ExtensionWants` must also read from `root.FS()` for project extensions; attach
  an `fs.FS` to each resolved extension rather than adding a type if `CatalogExtension` suffices. Never plan a file
  inside a project extension's directory.
- **Acceptance**: `extension add money` (a review agent in the store) creates its stubs and `AGENTS.md` line and
  leaves `docs/solomonar/extensions/money/` unchanged; a second run is a no-op; a v0.1.7 project is rejected with
  `framework_outdated`.
- **Verification**: temporary-repository tests; `testdata/extension_add.txtar` including `--dry-run`;
  `task check`.

### TASK-08: Stack-group decisions on `add`

- **Objective**: full CLI-D11 on `add`: new `P/D-n`, supersession, `--decision`/`--rationale`, rebinding.
- **Dependencies**: TASK-07.
- **Scope**: `PlanDecisions` writes a new decision only when a group's selection differs from the recorded one;
  supersession sets the old entry's `Status: superseded` and names the replacement; entry-status parsing in
  `docs/decisions.md` for `--decision`; manifest validation accepts `<task-ID>/D-<n>` (`validate.go:110`); every
  installed extension of the group rebinds to the new decision; `--decision`/`--rationale` on a non-stack group is
  `invalid_usage`.
- **Relevant code**: `internal/project/decision.go`, `validate.go`, `manifest_test.go`, `decision_test.go`.
- **Guidance**: entry format and group order follow 4.3. Only the replaced entry's `Status` line is edited.
- **Acceptance**: the UC-6 `htmx` + `templ` example, on a test catalog, produces `P/D-3` and supersedes `P/D-2`;
  `--decision W-002/D-2` with status `pending` is `decision_not_approved` (exit 3); a manifest with `W-002/D-1`
  validates.
- **Verification**: table-driven parsing and supersession tests (`fstest.MapFS` catalog);
  `go test ./internal/project/...`.

### TASK-09: `extension remove`

- **Objective**: deactivate an extension with the matching decision.
- **Dependencies**: TASK-08.
- **Scope**: `core_item`, `extension_requires` on the resulting set; catalog store files deleted through hashes;
  project extensions deactivated only; the `none` decision for an emptied group; the stderr warning.
- **Relevant code**: the TASK-03 planner, `internal/cli/extension.go`.
- **Acceptance**: `remove go` with an edited `go/SKILL.md` is a conflict, and `--keep` untracks the file and leaves
  it on disk; `remove money` leaves the project directory intact and `list` shows it `available` again;
  `remove work-spec` is `core_item`.
- **Verification**: temporary-repository tests; `testdata/extension_remove.txtar`; `task check`.

### TASK-10: The `extension-authoring` skill and framework text

- **Objective**: the skill writes project extensions into the store in catalog format and hands off with
  `solomonar extension add <name>`.
- **Dependencies**: TASK-07 (the commands must exist for the text to be true).
- **Scope**: `SKILL.md` project mode targets `docs/solomonar/extensions/<name>/` with `extension.json`, no longer
  `tools/solomonar/extensions/`; C-2 stays ("no hand registration"). `references/author.md` and `validate.md` drop
  the manifest-metadata branch and cover an empty or custom `detect`. Framework text:
  `assets/framework/rules.md:74-77` drops "Drafts approve nothing…"; `assets/framework/skills/solomonar/SKILL.md`
  (lines 15, 33) keeps the `solomonar help` check and names `extension list|add|remove` only;
  `orchestrator.md:27-30` stays (it already checks availability). Repository docs:
  - `CONTRIBUTING.md`: repository table (lines 57-65), "Add an extension skill" step 3 (the `detect`, `default`,
    `invocation` fields and their table), delete step 5 (combination sections), step 6 (the listings to update
    match TASK-11), "Add an extension agent" (store directory per agent), the verification steps (lines 416-426:
    `docs/solomonar/extensions/`), and the extension-authoring request examples (project requests end with
    `solomonar extension add <name>`).
  - `docs/research/extension-authoring.md`: the outcome bullets, the responsibility table, and the evaluation
    cases (no "commands unavailable" or draft case; a store-validation case instead).
  - `docs/design/framework-v0.1.md`: FW-4 reads "installs and removes" (no rendering); the parts table keeps
    `solomonar extension add`.
  - `docs/research/go-rules/review.md:121`: one dated note that the install path is now
    `docs/solomonar/extensions/go/`; the rest stays as the historical record.
  - Unchanged historical records: `docs/design/phase1-validation.md`, `docs/cli/git-checkpoint-implementation-plan.md`.
- **Acceptance**: no text says project source stays unregistered for lack of commands; relative links resolve
  from the store; no current-state document names `docs/solomonar/skills/<extension>/`, drafts, or combination
  sections.
- **Verification**: `rg -n 'unregistered|extension new|tools/solomonar|solomonar:with|## With|draft project' assets
  CONTRIBUTING.md docs/research docs/design`; a scratch `init --ext extension-authoring` and a read of the
  generated stubs.

### TASK-11: Site pages, examples, outputs, and README

- **Objective**: every public page describes the delivered behavior, with outputs captured from the binary.
- **Dependencies**: TASK-04, TASK-09, TASK-10.
- **Scope**: the pages of the [documentation alignment inventory](#documentation-alignment-inventory) marked
  TASK-11. Delivered commands lose `inDevelopment`, `group: In development`, and the banner, and move to the
  `Commands` group (`site/src/_data/site.js`). In-development pages that describe extension behavior (`doctor`,
  `sync`) drop rendering and drafts but stay in development. Version-only text is left to `task version:set` in
  TASK-13.
- **Guidance**: capture each output example by running the built binary in a scratch repository (the `ledger`
  example project: `go` installed, then `extension-authoring`, then a project reviewer); keep illustrative SHAs
  and paths in their real shape. The site examples use only real catalog names; `htmx`, `templ`, and `tdd` stay
  only in the spec's illustrative test catalog.
- **Acceptance**: every example matches a real run except for paths, dates, and SHAs; "in development" no longer
  appears for `upgrade` or `extension`; no page names drafts, `extension new`, `extension show`, or rendering;
  the getting-started availability table and the home page note list the delivered commands.
- **Verification**: site build and check (`site/package.json`, as `ci-site.yml` runs it);
  `rg -n -i 'extension new|extension show|draft|in development|rendered with|htmx|templ|tdd' site/src README.md`
  returns only intended hits.

### TASK-12: Diagrams

- **Objective**: the framework diagrams show the real catalog and the extension lifecycle.
- **Dependencies**: TASK-11.
- **Scope**: `docs/images/framework-blocks.svg` and `framework-core.svg`: replace `htmx · templ · tdd` with the
  real catalog (`go · extension-authoring`), let the CLI block read "installs the core, adapters, extensions;
  upgrades the framework", and update both `aria-label`s; the matching alt text in `README.md:32`. Add one
  diagram, `docs/images/extension-lifecycle.svg`: embedded catalog and `extension-authoring` → store
  `docs/solomonar/extensions/<name>/` → `extension add` → stubs, `AGENTS.md`, decision, manifest; `remove`
  reverses activation. Register it in the `diagrams` map of `site/eleventy.config.js` and embed it in
  `site/src/docs/cli/extension.md`.
- **Guidance**: follow the existing SVG conventions (the `dg` class, theme classes, text halo, `role="img"` with
  an `aria-label` that states the content). The workflow diagrams (`workflow-*.svg`) do not mention extensions
  and stay unchanged.
- **Acceptance**: both themes render the diagrams legibly in the built site; the labels match the delivered
  behavior.
- **Verification**: site build and check; a visual check of the extension page and the framework page in light
  and dark mode.

### TASK-13: Version and final verification

- **Objective**: a verified `0.2.0` release candidate.
- **Dependencies**: TASK-01 to TASK-12.
- **Scope**: `task version:set` to `0.2.0` (it updates current-release text in `assets`, `README.md`,
  `site/src`, and `interface.md`, such as `Initialized solomonar 0.1.7`, `In v0.1.7`, and `The v0.1.7 catalog`);
  then review the diff, since "In v0.1.7, extensions are chosen at `init`" sentences must already be rewritten by
  TASK-11 rather than re-versioned. Updated transcripts; a manual smoke test: v0.1.7 project → `upgrade` →
  `extension add`/`remove` with Claude Code.
- **Verification**: `task check`, `task test:integration`, `task build`, `go vet ./...`, the site build and check.
  CI needs no workflow change: `ci-solomonar.yml` runs the new tests and transcripts, and `ci-site.yml` already
  triggers on `site/**` and `docs/images/**`. Report the live result separately from the automated checks.

## Dependency order

```
01 → 02 → 03 → 04 ─────────┐
          02 → 05 → 06 ────┴→ 07 → 08 → 09 → 10 → 11 → 12 → 13
```

`upgrade` (TASK-04) can ship on its own after TASK-03; its site page (`cli/upgrade.md`) and the `upgrade` rows of
`cli.md` and `getting-started.md` may ship with it, ahead of TASK-11.

## Documentation alignment inventory

Every current-state mention found on 2026-10-24 (`git grep` over tracked files, excluding `site/_site`). The
owning task updates it; historical records stay unchanged.

| File | What changes | Task |
|---|---|---|
| `docs/cli/interface.md`, `docs/cli/architecture.md` | The contract (TASK-01 scope) | 01 |
| `assets/framework/reviewer.md:20`, `assets/framework/agents/implementer.md:10-11` | Extension install path | 02 |
| `assets/templates/AGENTS.md:35-43` | No draft sentence; decision per line; user-invoked line | 02, 05 |
| `testdata/init_*.txtar` | New paths; preselection from `default`/`detect` | 02, 05 |
| `assets/extensions/*/*/extension.json` | `detect`, `default`, `invocation` | 05 |
| `assets/extensions/workflow/extension-authoring/**` | Project mode targets the store; handoff to `extension add` | 10 |
| `assets/framework/rules.md:74-77`, `assets/framework/skills/solomonar/SKILL.md:15,33` | No drafts; `extension list`, `add`, and `remove` only | 10 |
| `CONTRIBUTING.md` | Repository table, extension skill/agent guides, combination step removed, verification paths, request examples | 10 |
| `docs/research/extension-authoring.md` | Responsibilities, outcome, evaluation cases | 10 |
| `docs/design/framework-v0.1.md` (FW-4) | "installs and removes", no rendering | 10 |
| `docs/research/go-rules/review.md:121` | One dated note on the new install path | 10 |
| `site/src/docs/cli/extension.md` | Rewrite: `list`, `add`, `remove`; store; authoring handoff; real outputs; lifecycle diagram | 11, 12 |
| `site/src/docs/cli/upgrade.md` (new) | Command page with real outputs | 11 |
| `site/src/docs/cli.md` | Command table, "Commands in development", "Planned commands" (increment 2 row), exit code 5 note | 11 |
| `site/src/docs/cli/init.md:38` | `--ext` default from `default` and `detect`; real catalog | 11 |
| `site/src/docs/cli/doctor.md:32,60`, `site/src/docs/cli/sync.md:15` | No rendering, drafts, or `htmx`/`templ`/`tdd` example (still in development) | 11 |
| `site/src/docs/cli/guard.md:22` | Unchanged (guard still planned; rule stays valid) | — |
| `site/src/docs/configuration.md:28,94-110` | Store path, ownership, `extension-authoring` handoff, no drafts, user-invoked skills | 11 |
| `site/src/docs/set-up-a-project.md:58-64,111,142` | Choosing extensions later with `extension add`; store path; recorded choices | 11 |
| `site/src/docs/getting-started.md:53-56` | Availability table | 11 |
| `site/src/docs/framework.md:116`, `site/src/docs/comparison.md:65` | Extensions after `init` are available | 11 |
| `site/src/docs/troubleshooting.md:16,47` | Commands now available; `framework_outdated`; `already_initialized` points to `extension add` | 11 |
| `site/src/docs/use-cases.md:251` | Pointer to delivered commands | 11 |
| `site/src/index.njk:125` | "Available today" note | 11 |
| `site/src/_data/site.js` | Delivered pages leave the "In development" group | 11 |
| `README.md:29-32,89,162-177` | Project status, catalog, `extension-authoring` paragraph, diagram alt text | 11, 12 |
| `docs/images/framework-blocks.svg`, `framework-core.svg` | Real catalog, CLI block, `aria-label` | 12 |
| `docs/images/extension-lifecycle.svg` (new), `site/eleventy.config.js` | New diagram and its registration | 12 |
| Version text across `assets`, `README.md`, `site/src`, `interface.md` | `task version:set 0.2.0` | 13 |
| `docs/design/phase1-validation.md`, `docs/cli/git-checkpoint-implementation-plan.md`, `SECURITY.md`, `site/src/docs/install.md` | Unchanged: historical records or unrelated `upgrade` wording | — |
| `.github/workflows/*.yml` | Unchanged; verified in TASK-13 | — |

## Final verification

The checks of TASK-13; every spec and site example compared with real output; the inventory greps of TASK-10
and TASK-11 rerun on the release candidate.

## Open questions

1. Version `0.2.0` for the layout change (assumed).
2. A missing manifest on `upgrade` is `not_initialized`, with no new code (assumed).
3. User-invoked skills in `AGENTS.md`: a separate line (X-6) or none. Recommendation: a separate line, so they stay
   discoverable.
4. Copilot: `invocation: user` blocks prompt-mode invocation (see the
   [Copilot acceptance record](interface.md#copilot-runtime-acceptance)); it stays a known, documented limitation.
