# Implementation Plan: Runtime Commands

Status: planned, 2026-10-24. This document plans implementation; it does not make the commands available.

## Objective

Deliver the minimum runtime management workflow: `solomonar runtime list`, `runtime add`, and `runtime remove`
for `claude`, `codex`, and `copilot`, using the existing adapters and project-file planner.

The user explicitly chose commands only. Guard implementation, hook installation, and hook removal are deferred.
Update the contracts and website to distinguish this delivery from the remaining increment 2 work.

## Current State

| Area | Existing implementation and reference |
| --- | --- |
| CLI | [run.go](../../internal/cli/run.go) registers `init`, `upgrade`, `extension`, and task commands; no runtime command is registered. [The runtime page](../../site/src/docs/cli/runtime.md) describes planned commands. |
| Adapters | [runtime.go](../../internal/project/runtime.go), `RuntimeWants`, generates all three runtimes' core and installed-extension stubs, Claude's create-only `CLAUDE.md`, and Codex invocation policies. It generates no guard hooks. |
| Choices | [manifest.go](../../internal/project/manifest.go) already records runtimes and their optional adversarial model under schema 1. `Manifest.CheckChoices` in [init.go](../../internal/project/init.go) rejects changed runtime choices and points users to the future commands. |
| Project writes | [init.go](../../internal/cli/init.go), `initRun.planFiles`, assembles the complete desired file set. `project.InitPlan`, `Classify`, and `Apply` handle hashes, conflicts, removals, the manifest, and rollback. [extension.go](../../internal/cli/extension.go) demonstrates changing choices through that flow. |
| Confirmation/output | [upgrade.go](../../internal/cli/upgrade.go), `confirmed`, handles interactive and non-interactive confirmation. [init_plan.go](../../internal/cli/init_plan.go) provides grouped plans; [output.go](../../internal/cli/output.go) owns rendering and JSON envelopes. |
| Documentation conflict | [Interface UC-5 and section 4.2](interface.md#uc-5-add-list-or-remove-a-runtime) and the runtime page couple Claude/Codex management to hooks. The user's commands-only scope supersedes that delivery coupling; TASK-01 records the change explicitly. |

The module declares Go **1.27.3**. Dependencies already cover the implementation and tests; no addition is needed.
The architecture is [accepted](architecture.md); the older [framework design](../design/framework-v0.1.md) is
historical, with current framework behavior owned by `assets/framework/`.

## Scope

**Confirmed requirements**

- List the three supported runtimes, installation status, executable presence, and adapter locations.
- Add or remove one runtime in an initialized project, including stubs for its installed extensions.
- Preserve explicit conflict resolution, dry runs, confirmation, human/JSON output, and existing filesystem safety.
- Retain the specified `runtime add --adversarial-model <model>` option; use it to update an installed runtime's model.
- Include interface references, design, decisions, and site documentation in the delivery.

**Out of scope**

- `guard`, hook merging/removal/trust checks, `doctor`, `sync`, and dashboard work. Never install a hook pointing
  to an unavailable command; leave runtime settings/hook files alone.
- Agent launching, runtime installation, authentication, model availability checks, network calls, automatic
  commits, or registry writes.
- New runtime names, bulk selection, model-reset flags, manifest/schema migrations, or adapter format redesign.
- Changing extension selections, stack decisions, or `init`'s refusal to change recorded choices.

**Conservative defaults to document in TASK-01**

- `list` works in any Git repository, without initialization; absent manifest means all runtimes are available.
  Installation is determined by the manifest, not by executable presence or inferred file health.
- `add|remove` require the project's framework to match the CLI, as `extension add|remove` already do. An older
  project must run `upgrade`; `list` can read an older valid manifest. Newer/invalid manifests use existing errors.
- Adding an installed runtime reconciles its desired state; an identical result is a no-op. An omitted model
  preserves its recorded value, or leaves it unspecified for a newly added runtime.
- Removing a supported but uninstalled runtime is a no-op. No-op writes need no confirmation. Unknown names
  remain usage errors; `none` is an `init` selection value, not a runtime name.

These defaults fill gaps in UC-5 rather than asserting previously approved behavior. No critical question
remains after the user's scope choice; record these defaults before implementation.

## Architectural Decisions

1. **Keep existing boundaries.** `internal/cli` owns parsing, choice changes, orchestration, and presentation;
   `internal/project` owns adapter generation, classification, manifest encoding, and transactional file writes.
   Use the static command table and `flag.FlagSet` (AD-2), injected `Env` (AD-9), and shared renderer (AD-4).
2. **Reuse the complete desired-state plan.** Start from the manifest's choices, change only the named runtime
   or its model, then reuse `initRun.planFiles`, `project.InitPlan`, `Classify`, and `Apply` (AD-6). Passing only
   one runtime's wants to `Classify` with the full manifest would incorrectly remove other managed files.
3. **Preserve extension provenance.** Resolve the currently installed catalog and project extensions using
   `LoadCatalog`, `LoadStore`, and the existing selection helpers. Do not select newly detected/default extensions
   or create a new stack decision. Keep project-owned extension sources and decision records intact.
4. **Keep existing reconciliation semantics.** The shared plan may restore missing managed files or refresh
   generated stubs under existing ownership rules; include every resulting action in its output. There is no
   implicit framework upgrade. `AGENTS.md` has no runtime-specific selection field and normally stays unchanged;
   do not introduce one. Preserve its project notes through the current planner.
5. **Keep schema and adapters unchanged.** Reuse schema 1, `runtime:<name>` ownership, SHA-256 bases, ordering,
   create-only `CLAUDE.md`, and existing serializers (AD-7). Generate no hook file or hook entry. No new package,
   interface, runtime registry abstraction, or production dependency is justified.
6. **Keep project writes local.** Use one `os.Root` for the worktree and the existing journaled apply. The entire
   file plan, including the manifest last, is the local completion point; handled failure/cancellation rolls it
   back. Discover Git through `openRepo`; use `git.OnPath` for detection. Never call a task store or publish a ref.

TASK-01 records the scope adjustment in the interface's `CLI-D` decision table and updates architecture delivery
notes. Existing AD decisions remain in force; this feature needs no replacement architecture.

## CLI Behavior

### Syntax and inputs

```text
solomonar runtime list [--json]
solomonar runtime add <claude|codex|copilot> [--adversarial-model <model>]
                      [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
solomonar runtime remove <claude|codex|copilot>
                         [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]
```

All commands accept the existing global flags, including `-C`. Flags remain interspersed with arguments.
`list` accepts no positional argument; each writing command requires exactly one supported name. A bare group
or unknown subcommand is `unknown_command`; missing/extra arguments and unknown flags are `invalid_usage`.

`add` takes a plain model string, unlike `init`'s `<runtime>=<model>` syntax. An explicitly empty model is
`invalid_value`; omission preserves the existing selection. No runtime/model prompt or new model default is added.
Explicit addition works even when the runtime executable is absent. Model strings go through existing adapter
serialization; the command does not authenticate or test provider access.

`--keep` and `--overwrite` use the existing section 4.5 rules, including rejection of non-conflict paths and paths
in both flags. Their availability on `add` comes from the shared-flags contract; add them to its incomplete
command-page synopsis. `--overwrite` cannot delete a non-regular removal path.

### Observable results

| Command | Result and side effects |
| --- | --- |
| `list` | Stable `claude`, `codex`, `copilot` order. Columns: `RUNTIME`, `STATUS` (`installed`/`available`), `ON PATH`, `ADAPTER`. Show native `AGENTS.md` loading for Codex/Copilot and each runtime's adapter locations. No hook claims, health checks, runtime execution, or writes. |
| `add` | Include the named runtime and optional model in the resulting manifest; generate its existing core and installed-extension adapters through the shared plan. Preserve other runtime/model choices. An installed runtime can have its model updated or missing adapters restored. |
| `remove` | Exclude the runtime and its model; remove its managed files when unmodified, drop already absent paths, and conflict on modified files. `--keep` untracks a conflicting removal and leaves it on disk; `--overwrite` deletes a regular conflicting file. Removing the last runtime leaves the framework and extensions installed. |

Removal preserves an unmanaged `CLAUDE.md`, unrelated files under `.github`, `.claude`, `.codex`, and `.agents`,
portable framework content, and project extension directories. Apply's existing empty-parent cleanup boundaries
remain unchanged. No runtime command changes `workflow-state`, branches, commits, or remotes.

Human results and dry-run plans go to stdout. Interactive plans and confirmation go to stderr; warnings and
errors use the existing prefixes there. Conflict plans still print to stdout before the stderr rejection.
Successful writes report the runtime and file-action counts, then suggest committing the affected paths;
add also gives the runtime's established skill invocation. Stage suggestions must include removed tracked paths
and name individual Copilot files rather than the whole `.github` tree. The CLI never runs the suggestion.

For JSON, reuse exactly one envelope with canonical `command` names (`runtime list`, `runtime add`,
`runtime remove`). Define the following data in UC-5 before implementing it:

- `list`: `runtimes`, an ordered array with `name`, `status`, `on_path` (boolean), and `adapters` (adapter locations).
- `add|remove`: `project`, `runtime`, `files` (existing per-file action/path/owner/reason form), optional
  `adversarial_model` for the resulting selection, `next`, and `dry_run` when true. A no-op has empty `next`;
  a dry run reports the proposed plan. Failures omit `data` through the existing renderer.

Writing validation order: parse/validate arguments and model → discover repository/check Git → read and validate
manifest → require matching framework → resolve installed extensions → plan and resolve conflicts → dry-run/no-op
→ confirmation → apply. An already absent runtime can return its validated no-op without reconciling project files.
Resolution flags must still identify actual conflicts; they cannot be silently ignored on that no-op.

| Exit | Runtime behavior |
| --- | --- |
| 0 | Success, valid dry run, no-op, or declined confirmation. |
| 1 | Existing unexpected I/O/Git failure, invalid manifest, or prompt input failure. |
| 2 | Usage/name/model/resolution error, or `interactive_required` when a write needs confirmation without `--yes`. |
| 3 | Outside a repository, uninitialized writing command, or incompatible framework/CLI version. |
| 4 | Unresolved managed-file conflict, including a removal conflict. `--yes` never resolves it. |
| 130/143 | Existing handled SIGINT/SIGTERM behavior before successful file-plan completion; rollback leaves no changes. |

`--json` never prompts. In a pipe/CI, a real change requires `--yes`; `--dry-run` and no-ops do not. There are no
new environment variables or configuration options: executable detection uses `PATH`, and existing global output
controls still apply. `SOLOMONAR_OWNER` does not affect runtime commands.

Example workflow in an initialized project:

```sh
solomonar runtime list
solomonar runtime add copilot --dry-run
solomonar runtime add copilot --yes
solomonar runtime add claude --adversarial-model opus --yes
solomonar runtime remove copilot --dry-run
solomonar runtime remove copilot --yes --json
```

## Implementation Tasks

### TASK-01: Align the runtime contract, design, and decision record

**Objective**
Make the commands-only scope and conservative defaults authoritative before code changes.

**Dependencies**
None.

**Scope**
Update the behavioral contract, structural delivery notes, and planned command reference without claiming availability.

**Relevant code**
[interface.md](interface.md) sections 2, 3.1/3.6, 4.2/4.5/4.7, UC-5, 10, 11;
[architecture.md](architecture.md) sections 1, 7, 11, 12, 14;
[runtime command page](../../site/src/docs/cli/runtime.md);
[historical design](../design/framework-v0.1.md).

**Implementation guidance**
Record the user's hook deferral and the defaults above in the next unused `CLI-D` entry, distinguishing the
confirmed scope from newly specified behavior. Expand UC-5 with validation order, no-ops, model updates, JSON
data, and file effects. Explain that the shared planner also serves runtime mutations. Separate runtime command
delivery from the remaining `guard`/hook work; retain future hook ownership/schema design as explicitly deferred.
Keep the historical design intact; add a short dated amendment pointing to UC-5 and the current architecture.
Remove planned hook output from the runtime page but keep its in-development label until implementation lands.

**Acceptance criteria**
No current-delivery section promises hook installation/removal or makes guard a dependency. The three commands,
errors, and defaults agree across UC-5 and the command page. Future hook design remains distinguishable from
current behavior. This plan is linked from architecture section 12.

**Verification**
Review the documentation diff and local links. From `site/`, run `npm run build` and `npm run check`.

**Out of scope**
Changing adapter formats or framework workflow policy.

### TASK-02: Deliver read-only runtime listing

**Objective**
Users can inspect runtime installation and executable presence before changing the project.

**Dependencies**
TASK-01.

**Scope**
Register `runtime list`, add its handler/help and human/JSON forms, and cover repository/manifest boundaries.

**Relevant code**
[run.go](../../internal/cli/run.go), [help_test.go](../../internal/cli/help_test.go),
[extension.go](../../internal/cli/extension.go) (`extensionListFlags.run`),
[repo.go](../../internal/cli/repo.go), [manifest.go](../../internal/project/manifest.go),
[git.go](../../internal/git/git.go).

**Implementation guidance**
Keep the runtime command group together in `internal/cli`. Use `openRepo`, `ReadManifest`, and `git.OnPath`.
List known locations without loading the extension catalog/store, generating adapters, or invoking runtimes.
Keep any shared runtime-name validation concrete and small; reject `none` for writing commands when added.

**Acceptance criteria**
All three rows appear in stable order with independent installed/on-PATH values. A repository without a manifest
lists all as available; older valid manifests remain readable. Newer/invalid manifests and outside-repository
calls use existing error classes. Human and JSON contain the same information; help works through both forms.
No files or refs change, including from a nested directory or `-C` feature worktree.

**Verification**
Add focused in-process cases for these boundaries and a controlled `PATH`; run
`go test ./internal/cli -run 'TestRuntimeList|TestHelp|TestCommandHelp'`.
From the source checkout, inspect `go run . -C <repo> runtime list` and
`go run . -C <repo> runtime list --json`, pointing `<repo>` at a disposable repository.

### TASK-03: Add a runtime through the existing project transaction

**Objective**
An initialized project can install another runtime or update its recorded adversarial model safely.

**Dependencies**
TASK-02.

**Scope**
Implement `runtime add`, conflict flags, confirmation, dry-run/no-op, result data, and commit/session suggestions.

**Relevant code**
[extension.go](../../internal/cli/extension.go) (`extensionFlags.plan`, `currentFramework`),
[init.go](../../internal/cli/init.go) (`selectRuntimes`, `selectModels`, `initRun.planFiles`, `generatedPaths`),
[upgrade.go](../../internal/cli/upgrade.go) (`initializedManifest`, `confirmed`),
[runtime.go](../../internal/project/runtime.go), [init.go](../../internal/project/init.go),
[classify.go](../../internal/project/classify.go), [apply.go](../../internal/project/apply.go),
[init_copilot_test.go](../../internal/cli/init_copilot_test.go).

**Implementation guidance**
Copy the manifest's runtime/model choices before changing the named entry; retain existing extension selections
and decision IDs. Reuse the complete planner and `project.Apply`; do not call init's registry-creation path.
Add no hook wants. Track whether the model flag was supplied so omission preserves a value and explicit empty
input fails. Reuse the existing output/confirmation helpers without building a generic mutation framework.

**Acceptance criteria**
Each runtime can be added from `--runtime none` and alongside another runtime, without its executable installed.
Core skills/roles and installed catalog/project extension skills/agents get the existing adapter forms and owners,
including user-invoked Codex policy files. Other choices and decision documents stay intact. Repeated identical
add is a no-op; a changed model updates only the named model choice and its affected stub under normal planning.
Unmanaged `CLAUDE.md` stays untracked. Conflicts, kept base hashes, dry runs, decline, and JSON/non-interactive
confirmation follow the contract. No settings/hook file or registry ref is changed.

**Verification**
Add in-process CLI cases for the behavior above; use temporary repositories and existing fixture helpers. Assert
manifest choices, file content/ownership, unchanged extension sources/decisions, and unchanged registry tip.
Run `go test ./internal/cli -run 'TestRuntimeAdd|TestHelp|TestCommandHelp'`; if shared planner code changes,
also run `go test ./internal/project`. Use `runtime add copilot --dry-run` then `--yes` in a disposable project.

### TASK-04: Remove a runtime without losing project files

**Objective**
Users can remove one adapter, preserve edited files explicitly, and keep the rest of the project usable.

**Dependencies**
TASK-03.

**Scope**
Implement `runtime remove` using the same choice/planning/apply flow and the existing removal classifications.

**Relevant code**
[extension.go](../../internal/cli/extension.go) (remove and output patterns),
[classify.go](../../internal/project/classify.go) (`classifyRemovals`, `Resolutions.removal`),
[apply.go](../../internal/project/apply.go), [classify_test.go](../../internal/project/classify_test.go),
[extension_test.go](../../internal/cli/extension_test.go), [init.go](../../internal/cli/init.go) (`generatedPaths`).

**Implementation guidance**
Remove only the named runtime choice/model from the resulting choices; give the planner the full remaining set.
Let manifest ownership and base hashes identify removed files. Do not remove directories recursively or infer
ownership from a runtime directory name. Reuse removal conflict resolution and journal cleanup. Ensure printed
staging guidance includes deleted tracked paths even when an entire adapter directory disappears.

**Acceptance criteria**
Removing each runtime deletes its unmodified managed adapters and model record, including policies and a
CLI-created `CLAUDE.md`. Missing managed files are untracked. Edited files conflict; keep leaves them untracked,
overwrite deletes regular files, and non-regular deletion is rejected. Other runtimes, unrelated `.github` files,
unmanaged `CLAUDE.md`, installed extensions, project notes, and registry tip remain intact. Removing the last
runtime leaves an initialized framework; repeated removal succeeds without prompting or writing.

**Verification**
Run `go test ./internal/cli -run 'TestRuntimeRemove|TestHelp|TestCommandHelp'` and
`go test ./internal/project -run 'TestClassify|TestApply'` if those implementations changed. Cover removal with
all three runtimes installed, last-runtime removal, keep/overwrite, absent files, preserved user files, and a
handled late apply failure restoring removed files/directories and the old manifest. Reuse existing apply failure
fixtures rather than introduce an I/O abstraction. Preview and apply `runtime remove` in a disposable project.

### TASK-05: Verify the command journey and publish matching documentation

**Objective**
Finish with verified process behavior and consistent repository/site references.

**Dependencies**
TASK-04.

**Scope**
Add one runtime-management transcript, finish reference/design status updates, and update existing website pages.

**Relevant code**
[main_test.go](../../main_test.go), [extension_add.txtar](../../testdata/extension_add.txtar),
[extension_remove.txtar](../../testdata/extension_remove.txtar), [init_copilot.txtar](../../testdata/init_copilot.txtar),
[Taskfile.yml](../../Taskfile.yml), [site/README.md](../../site/README.md),
and the documentation paths listed below.

**Implementation guidance**
The transcript should cover a single real-process journey: initialize with no runtimes, list, preview/add a
runtime, repeat add as a no-op, preview/remove, and verify the final manifest/files and unchanged registry tip.
Exercise JSON, stdin without a terminal, exit statuses, and deleted paths in commit guidance at the process
boundary. Keep exhaustive validation and conflict combinations in the in-process tests.

Update these existing documentation surfaces together:

| Surface | Required update |
| --- | --- |
| `docs/cli/interface.md` | Finalize UC-5, the command map, shared flags/errors, ownership/manifest notes, scope decision, and delivery status. Keep hook behavior explicitly deferred and retain Copilot's existing acceptance limitations. |
| `docs/cli/architecture.md` | Finalize runtime execution/reconciliation notes, link this plan, and separate completed command management from deferred hooks/guard in implementation and acceptance sections. |
| `docs/design/framework-v0.1.md` | Complete the dated delivery amendment/cross-reference; preserve historical decisions and process design. |
| `site/src/docs/cli/runtime.md` | Remove `inDevelopment` and its banner, move to the Reference group, add implementation sources, and replace speculative output/counts with verified examples. Document flags, model updates, no-ops, conflicts, and commands-only scope. |
| `site/src/docs/cli.md` | Move all three commands into Current commands, state which require initialization, and remove runtime management from planned tables while keeping hooks/guard deferred. |
| `site/src/docs/getting-started.md` | Update the availability table without implying other increment 2 commands shipped. |
| `site/src/docs/set-up-a-project.md`, `site/src/docs/cli/init.md` | Replace instructions to wait for `runtime add` with add/remove examples and links. Preserve `init`'s recorded-choice restriction. |
| `site/src/docs/configuration.md` | Explain runtime/model changes, existing adapter ownership/removal, and absent hook management. Keep `sync` deferred. |
| `site/src/docs/troubleshooting.md` | Update unknown-command guidance and runtime-choice recovery; retain actual runtime compatibility caveats. |
| `README.md` | Replace the Copilot/runtime-add deferral with the verified command workflow and reference link. |

Only update availability after the commands exist. Do not mark Copilot delegation/model compatibility or a
minimum supported version verified by file-generation tests. Existing runtime adapter smoke acceptance stays
separate; this delivery does not change runtime discovery formats or require provider sessions in automated tests.

**Acceptance criteria**
The transcript passes, documentation matches actual help and output, and local site links/navigation are valid.
No current page claims installed guard hooks. Remaining planned commands and Copilot acceptance limits retain
accurate labels. Record commands actually run and any remaining acceptance limits; do not equate planning with
verified implementation.

**Verification**
Run `INTEGRATION=true go test . -run '^TestScript/runtime_management$'` for the proposed transcript name, then the
final checks below. From `site/`, run `npm run build` and `npm run check`; inspect the generated runtime page,
overview, and navigation. Search affected docs for stale `wait for`, `planned runtime`, and hook-install claims.

**Out of scope**
Site redesign, deployment, release tagging, or unrelated documentation cleanup.

## Dependency Order

`TASK-01 → TASK-02 → TASK-03 → TASK-04 → TASK-05`.

This gives a read-only command first, then the first writing workflow, then its inverse. Reuse the existing
project transaction throughout; no infrastructure-only implementation step or guard dependency is needed.

## Final Verification

After implementation, run the repository's existing checks from the root:

```sh
task check
task test:integration
go vet ./...
```

From `site/`:

```sh
npm run build
npm run check
```

Use Node.js 24+ and the locked site dependencies, as documented in `site/README.md`. Use Go 1.27.3 and the
existing pinned tools from `Taskfile.yml`. If a shared helper changes, include its existing callers in regression
verification. No new test framework, coverage target, or CI job is required.

Finally review the diff for accidental hooks, schema/dependency changes, extra commands/flags, or unrelated
refactoring. Confirm `init` still rejects changed choices, `upgrade`/`extension` still preserve runtime choices,
and runtime operations leave registry history unchanged. Suggested commits remain suggestions.

## Open Questions

None blocking. The user resolved the delivery scope in favor of runtime management only. TASK-01 records the
conservative command defaults above; hooks and the existing runtime live-acceptance gaps remain separately
deferred rather than implied complete by this plan.
