# Implementation Plan: GitHub Copilot CLI Runtime

Status: planned. Checkboxes track implementation and verification; none of these tasks is complete.

## Objective

Add `copilot` to `solomonar init`, generating runtime adapters that load the existing portable framework.
Deliver the behavior in small increments without adding dependencies or changing the CLI's role as a guard
and oracle.

## Current State

- `go.mod` declares Go 1.27.3. No new dependency is needed.
- `internal/project/runtime.go` generates Claude and Codex adapters from portable skills and roles.
- `internal/project` owns manifest validation, file classification, conflict handling, and rollback.
- `internal/cli` owns runtime selection, prompts, plan output, and session-start instructions.
- `runtime list/add/remove`, `sync`, `doctor`, and `guard` are specified but unimplemented.
- Existing runtime-generation, manifest-decoding, and relevant init regression tests passed during planning.
  Copilot behavior was checked against documentation, not a live session.

## Scope

Confirmed requirements:

- Support GitHub Copilot as another runtime while preserving the existing architecture and portable sources.
- Include relevant specification, README, and site updates in the implementation tasks.

Conservative assumptions:

- The target is **Copilot CLI**. IDE and cloud-agent compatibility are outside this delivery.
- The first delivery supports fresh initialization and reruns with the same recorded choices.
- Runtime management and hooks remain in their existing delivery increment.

Constraints and exclusions:

- `init` never changes recorded runtime choices. Adding Copilot to an initialized project remains deferred
  until `runtime add` exists; do not introduce a bypass or recommend hand-editing the manifest.
- Preserve existing flags, exit codes, output envelopes, registry behavior, file safety, and rollback.
- No agent launching, model calls, credential handling, SDK integration, new environment variables, or
  runtime abstraction layer.
- No changes to task transitions or portable workflow rules.

## Architectural Decisions

The accepted interface currently permits only `claude` and `codex`. TASK-01 explicitly extends that contract;
the decisions below are the proposed Copilot addition, not claims about the current implementation.

| Concern | Decision |
| --- | --- |
| Runtime identifier | `copilot`; executable detection uses `copilot` on `PATH`. |
| Root instructions | Use native `AGENTS.md` discovery; generate no additional root instruction file. |
| Skills | `.github/skills/<skill>/SKILL.md`, pointing to `docs/solomonar/skills/<skill>/SKILL.md`. |
| Entry skill | Set `disable-model-invocation: true` for `solomonar`; invoke it with `/solomonar`. |
| Delegated roles | `.github/agents/<role>.agent.md`, pointing to the portable role; exclude the orchestrator. |
| Repository context | Set `include-custom-instructions: true` in delegated role definitions. |
| Tools | Implementer inherits tools; reviewers get `read`, `search`, and `execute`. Shell access remains subject to the portable read-only contract. |
| Model | Write the configured model only into the adversarial-reviewer adapter; omit it when unspecified. |
| Ownership | `runtime:copilot`, using existing SHA-256 file tracking. |
| Ordering | Preserve Claude before Codex; append Copilot. Sort skills and roles by their existing identifiers. |
| Packages | Extend `internal/project` and `internal/cli`; add no package, interface, provider registry, or dependency. |
| Manifest | Retain schema version 1 because the structure is unchanged. Older binaries reject the new runtime; use the normal framework release version boundary. |

Copilot documents native instructions, skill invocation controls, and model settings in its
[CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference).
The separate skill location follows its
[agent skills documentation](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-skills).
Role tools follow the
[custom-agent configuration reference](https://docs.github.com/en/copilot/reference/custom-agents-configuration);
repository instruction loading follows the
[custom-agent documentation](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli).
Recheck version-sensitive fields when implementing and record the tested Copilot version in TASK-04.

## CLI Behavior

Affected syntax; all other arguments and flags retain their existing meaning:

```text
solomonar init [<dir>] [--runtime <claude|codex|copilot|none>]...
```

Examples:

```sh
solomonar init --runtime copilot --ext none --dry-run
solomonar init --runtime copilot --yes
solomonar init --runtime claude,codex,copilot --yes
solomonar init --runtime copilot --adversarial-model copilot=<model> --yes
```

- Runtime flags remain repeatable and comma-separated. `none` remains exclusive.
- Explicit `--runtime copilot` works without Copilot installed. Without an explicit selection, retain manifest
  choices; for a new project, detect supported executables on `PATH`.
- Model overrides must name a selected runtime; preserve last-value-wins behavior. Installation does not
  authenticate or validate account model access.
- Human results and dry-run plans go to stdout; prompts, warnings, and errors follow existing stderr conventions.
- JSON retains the existing envelope, adding Copilot choices and file records. It never prompts.
- Non-interactive writes require `--yes`; dry runs write nothing.
- Preserve exits: 0 success/no-op/declined confirmation, 1 unexpected failure, 2 usage, 3 rejected choices,
  4 conflicts, and existing interruption handling. Unknown runtime names remain exit 2.
- Write framework, adapter, and manifest files through the current plan/apply engine. Preserve registry
  creation and optional push behavior. Do not invoke Copilot or install hooks.
- Next steps show the Copilot `/solomonar` invocation. Suggested staging commands name only generated adapter
  paths under `.github`; never suggest staging the whole `.github` directory.

## Implementation Tasks

### TASK-01: Extend the runtime contract

**Objective:** Specify Copilot support consistently with the accepted CLI contracts.

**Dependencies:** None.

**Relevant code:** `docs/cli/interface.md`, `docs/cli/architecture.md`, `site/src/docs/cli/runtime.md`.

**Scope and implementation guidance**

- [ ] Extend the interface's runtime names, adapter table, manifest validation, ordering, prompts, error text,
  and next-step contracts to include Copilot.
- [ ] Record native `AGENTS.md` loading, portable-source stubs, delegated roles, entry-skill policy, tool
  restrictions, and model handling without changing the existing runtime contracts.
- [ ] Update the architecture's adapter responsibilities and ordering; retain the existing package boundaries,
  standard-library approach, and manifest schema.
- [ ] Update the planned runtime-management contract and `site/src/docs/cli/runtime.md` to include Copilot,
  explicitly retaining their unavailable status. Do not promise a Copilot hook in this delivery.
- [ ] Document older-binary compatibility and the existing prohibition on changing choices through `init`.

**Acceptance criteria**

- [ ] Specifications agree on Copilot paths, metadata, ownership, CLI behavior, and delivery boundaries.
- [ ] Available initialization is distinguished from deferred management and hooks.

**Verification**

- [ ] Review the documentation diff against the behavior and decisions above.
- [ ] Confirm version-sensitive adapter fields against the official sources before implementing them.

**Out of scope:** Command implementation and changes to existing architectural decisions.

### TASK-02: Generate and track Copilot adapters

**Objective:** Make the project layer produce Copilot adapters and accept their manifest records.

**Dependencies:** TASK-01.

**Relevant code:** `internal/project/runtime.go`, `internal/project/validate.go`,
`internal/project/manifest.go`, `internal/project/runtime_test.go`, `internal/project/manifest_test.go`,
`internal/project/classify.go`, `site/src/docs/configuration.md`.

**Scope and implementation guidance**

- [ ] Extend runtime generation and runtime-name validation using the existing portable discovery, extension
  inclusion, stub generation, and `Want` ownership patterns.
- [ ] Generate core and installed extension skill/role adapters with the agreed policies, source references,
  tool restrictions, and optional adversarial model. Exclude the orchestrator.
- [ ] Serialize Copilot metadata safely, including descriptions and model values containing YAML-significant
  characters; do not promote a new YAML dependency to production.
- [ ] Keep generated paths distinct from Claude and Codex paths, with `runtime:copilot` ownership and hashes.
- [ ] Reuse classification and apply behavior without refactoring the filesystem engine.
- [ ] Update `site/src/docs/configuration.md`: file inventory, adapter comparison, native instructions,
  ownership, entry-skill policy, reviewer limitations, model configuration, and edit-preservation behavior.

**Acceptance criteria**

- [ ] Copilot-only and mixed-runtime generation are deterministic and have no overlapping managed paths.
- [ ] Representative core and extension adapters have correct metadata and portable-source references.
- [ ] Copilot manifests round-trip; unsupported names remain rejected.
- [ ] No orchestrator adapter or extra root instruction file is generated.
- [ ] Existing Claude and Codex adapter bytes remain unchanged.

**Verification**

- [ ] Extend existing project tests for ownership, ordering, extension inclusion, metadata escaping, and model
  omission/selection; avoid duplicating file-engine coverage.
- [ ] Run `rtk go test ./internal/project -run 'Test(RuntimeWants|DecodeManifest)' -count=1`.
- [ ] Compare the configuration reference with the generated files and manifest entries.

**Out of scope:** Generic runtime abstractions and filesystem-engine changes.

### TASK-03: Deliver Copilot initialization end to end

**Objective:** Let users select, preview, and install Copilot through `init`.

**Dependencies:** TASK-02.

**Relevant code:** `internal/cli/init.go`, `internal/cli/init_prompt.go`, `internal/cli/init_plan.go`,
`internal/cli/init_test.go`, `internal/cli/init_prompt_test.go`, `internal/cli/init_plan_test.go`,
`internal/cli/init_ext_test.go`, `site/src/docs/cli/init.md`, `site/src/docs/set-up-a-project.md`.

**Scope and implementation guidance**

- [ ] Extend parsing, validation messages, executable detection, help, runtime choices, and model prompts.
- [ ] Extend grouped plans to recognize `.github/skills/` and `.github/agents/`; account for `.agent.md` when
  identifying extension roles and counting stubs.
- [ ] Update JSON choices, generated-file records, and next steps for Copilot alone and mixed selections.
- [ ] Preserve existing output for selections without Copilot; handle any supported runtime combination
  without assuming exactly two runtimes.
- [ ] Limit suggested staging paths to the generated adapters; do not include unrelated GitHub configuration.
- [ ] Preserve confirmation, dry-run, conflict resolution, rollback, and same-choice rerun behavior.
- [ ] Update `site/src/docs/cli/init.md`: synopsis, flags, defaults, examples, output, and session invocation.
- [ ] Update `site/src/docs/set-up-a-project.md`: runtime selection, generated-file inventory, Copilot startup,
  and a representative setup example. Keep examples consistent with actual output.

**Acceptance criteria**

- [ ] Explicit selection works without Copilot installed; new-project defaults detect it on `PATH`.
- [ ] Interactive selection and model prompts identify GitHub Copilot CLI correctly.
- [ ] Plans, counts, JSON, and next steps are accurate for Copilot and mixed selections.
- [ ] Dry runs write nothing; identical reruns are no-ops; collisions follow existing conflict rules.
- [ ] Changing recorded runtime choices remains rejected, including attempts to add Copilot through `init`.

**Verification**

- [ ] Extend init tests for representative runtime combinations, isolated `PATH` detection, prompts,
  non-interactive behavior, model validation, conflicts, and reruns. Reuse existing rollback coverage.
- [ ] Run `rtk go test ./internal/cli -run TestInit -count=1`.
- [ ] Check documented examples against command output; label abbreviated output clearly.

**Out of scope:** `runtime add/remove`, `sync`, `doctor`, and `guard` implementation.

### TASK-04: Validate discovery and complete public documentation

**Objective:** Establish live Copilot compatibility and publish accurate support boundaries.

**Dependencies:** TASK-03.

**Relevant code:** `main_test.go`, `testdata/init_new_repo.txtar`, `testdata/init_conflict.txtar`,
`README.md`, `site/src/docs/getting-started.md`, `site/src/docs/configuration.md`,
`site/src/docs/troubleshooting.md`, `site/src/docs/cli.md`, `Taskfile.yml`, `site/package.json`.

**Scope and implementation guidance**

- [ ] Add one focused Copilot transcript under `testdata/` for installation, stream separation, generated
  ownership, and an unchanged rerun. Keep detailed adapter assertions in project tests.
- [ ] Perform runtime acceptance in a disposable repository using an authenticated Copilot installation;
  record the exact version, steps, results, and limitations in `docs/cli/plan.md`.
- [ ] Confirm `/solomonar` loads the portable source and keeps orchestration in the top-level session.
- [ ] Confirm delegated roles load repository instructions and respect the assigned worktree/revision.
- [ ] Confirm reviewer tools permit required checks without exposing direct editing tools; verify the
  portable read-only contract remains explicit where shell access is allowed.
- [ ] Observe the configured adversarial model, including runtime fallback or override behavior; do not
  claim model isolation based only on generated metadata.
- [ ] Confirm discovery in a feature worktree and with Claude, Codex, and Copilot adapters installed together.
  Check that Copilot selects the intended `.github` definitions despite compatible discovery locations.
- [ ] Update `README.md` and `site/src/docs/getting-started.md`: runtime support, requirements, availability,
  and the Copilot entry invocation, using the tested version as the compatibility baseline.
- [ ] Update `site/src/docs/troubleshooting.md`: missing discovery, personal-agent shadowing, runtime/model
  overrides, and the restriction on adding a runtime to an initialized project.
- [ ] Update `site/src/docs/cli.md` and affected site descriptions to include Copilot without marking deferred
  commands as available. Search README, `docs/cli`, and `site/src` for remaining two-runtime assumptions;
  change only statements affected by this feature.
- [ ] Ensure public documentation and future-command pages do not imply installed Copilot guard enforcement.

**Acceptance criteria**

- [ ] Process-level tests pass and live discovery/delegation checks have recorded evidence.
- [ ] The tested Copilot version and any observed limitations are documented.
- [ ] README, specifications, command references, setup guidance, and configuration tables agree.
- [ ] Documentation does not claim IDE/cloud compatibility or unimplemented runtime-management support.

**Verification**

- [ ] Complete the final automated checks below and record outcomes separately from live smoke results.
- [ ] If live acceptance cannot run, leave its checkboxes open and report the exact blocker; generated-file
  tests alone do not establish runtime compatibility.

**Out of scope:** Provider-backed CI, credential setup, and IDE/cloud compatibility testing.

## Dependency Order

`TASK-01 -> TASK-02 -> TASK-03 -> TASK-04`

TASK-03 delivers the first complete CLI workflow. TASK-04 establishes actual runtime compatibility.
Documentation changes accompany the task that owns the behavior rather than waiting for a separate cleanup.

## Final Verification

Run after implementation; these checks have not been completed for the proposed feature:

- [ ] Run `rtk task check` from the repository root.
- [ ] Run `rtk task test:integration` from the repository root.
- [ ] Run `rtk go vet ./...` from the repository root.
- [ ] Run `rtk npm run build` from `site/`.
- [ ] Run `rtk npm run check` from `site/`.
- [ ] Inspect the final diff for unintended changes, broken local links, and inconsistent runtime availability.
- [ ] Record successful live Copilot acceptance separately from automated test results before claiming support.

## Open Questions

No question blocks implementation under the scope assumptions above.

- [ ] Establish the minimum supported Copilot CLI version from TASK-04 evidence.

Hooks and installation into already initialized projects remain in the existing runtime-management increment.
