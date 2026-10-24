# Extension authoring: evidence and adoption

Initial prompt research: 2026-10-24, parent `3fbe9f5e`. Skill adoption candidate:
2026-10-24, parent `655f0bb` (Solomonar 0.1.7).

The single external authoring source map now lives in
[the skill's evidence reference](../../assets/extensions/workflow/extension-authoring/references/evidence.md).
It preserves the original E-1–E-13 evidence, dates, counterevidence, and limits;
those sources are not duplicated here. It ships with the optional skill so an
adopting project can use it without this repository's `docs/research/` tree.
This document owns the authoring skill's design, local evidence, and validation.

## Outcome and boundaries

The user requested turning the three prompts into a framework skill and selected
**both catalog and project-specific routes** on 2026-10-24. The candidate is the
optional workflow extension
[`extension-authoring`](../../assets/extensions/workflow/extension-authoring/SKILL.md).
It is a supporting procedure, not a new required core role or step.

- Catalog mode researches and edits `assets/extensions/<group>/<name>/` and its
  associated research/documentation/check fixtures.
- Project mode researches and edits project-owned source, using
  `tools/solomonar/extensions/<name>/` for a new unregistered candidate or the
  recorded source path of an existing CLI-created draft.
- Research → authoring → validation is the default for a complete creation
  request. Stage-only requests stop at that stage. Scoped approvals persist;
  stage transitions introduce no new confirmation gate.
- Each authored extension keeps sources, decisions, expected cases, results,
  and the next action in one research record. The standalone prompt wrappers
  were removed after consolidation into the skill. Short stage invocation
  examples live in [CONTRIBUTING.md](../../CONTRIBUTING.md#writing-framework-text);
  the skill's stage references own the procedures.
- Runtime adapters remain generated pointers to one installed portable body.
  No new CLI command or runtime-adapter implementation is introduced.

## Skill and CLI responsibilities

| Concern | Owner | Local evidence / implication |
| --- | --- | --- |
| Domain research, rules, exceptions, reviewer method, evaluation | Skill | Existing prompt workflow; E-9/E-13 in the source map |
| Catalog metadata | Catalog author, checked by CLI | [Catalog parser](../../internal/project/catalog.go); strict `extension.json` schema |
| Project metadata and draft registration | CLI | [Interface UC-7](../cli/interface.md#uc-7-add-workflow-skills-review-agents-or-the-projects-own-extensions); proposed values remain in the research record until registration |
| Installation, ownership, activation, combinations, generated stubs | CLI | [AGENTS template](../../assets/templates/AGENTS.md), [installer](../../internal/project/extension.go), [runtime generation](../../internal/project/runtime.go) |
| Draft scaffold | Planned `extension new` | It creates source **and records a tracked draft**, so authoring content is not a complete replacement |
| Adopting a local source directory | Planned `extension add <path>` | An existing source needs no `new` step; adoption remains a separate supported operation |

Retain the planned lifecycle commands. The skill can replace the manual content
writing portion of a new-extension workflow; `extension new` remains a small
optional deterministic scaffold/registration operation. Removing tracked-draft
semantics would be a separate CLI design change, not necessary for this skill.
The current [command registry](../../internal/cli/run.go) has no extension
management commands. New project source therefore remains unregistered, and an
initialized project cannot change extension choices with another `init` call.

The project-source contract specifically says metadata belongs in the manifest,
not a separate package `extension.json`. The original catalog-only prompts did
not account for that distinction. It is now explicit in both the target map and
adoption handoff, without teaching the agent to modify the manifest itself.

## Context and source ownership

| Load | Purpose |
| --- | --- |
| `SKILL.md` | Scope, mode/stage routing, four common rules, and finish contract |
| `references/research.md` | Current repository/project inspection, topic evidence, expected validation cases |
| `references/author.md` | Target-specific output, writing constraints, integration boundary |
| `references/validate.md` | Structural, packaging, and behavioral validation with explicit limits |
| `references/evidence.md` | One external source map, loaded during research only |

The stage references use relative links so they resolve both in the catalog and
under `docs/solomonar/skills/extension-authoring/`. The installed skill never
requires Solomonar's source checkout or contributor guide for project mode.
Source-only paths are restricted to catalog mode. No core route or default
extension selection is changed; direct invocation and existing extension
activation metadata make this optional capability discoverable after installation.

## Instruction evidence matrix

| Instruction | Governing evidence | Classification / limit |
| --- | --- | --- |
| C-1: trace substantive requirements, distinguishing policy and hypothesis | [Go research precedent](go-rules/review.md), E-4/E-5/E-9/E-13 | Proposed authoring policy; table shape has no measured universal advantage |
| C-2: CLI owns registration/generated output | [AGENTS mandatory rules](../../assets/templates/AGENTS.md), interface 4.3 and UC-7, installer/runtime code | Existing ownership contract; absent CLI support is not permission to bypass it |
| C-3: stable IDs and retirement history | [Contributor rule procedure](../../CONTRIBUTING.md#add-rules-to-an-extension) | Existing rule-identity contract |
| C-4: distinguish evidence levels and incomplete checks | [Report contract](../../assets/framework/report.md), E-13 | Existing verification contract plus evaluation guidance |
| Optional `workflow` placement, not core | [Core/extension boundary](../../CONTRIBUTING.md#contributing-to-the-framework), interface 4.3 | Existing extension classification |
| One portable body, generated stubs for each supported runtime | [Runtime generator](../../internal/project/runtime.go), E-7 | Existing packaging mechanism; live discovery remains distinct |
| Conditional stage references and one evidence map | Existing Go loading routes, E-3/E-6/E-7/E-8 | Design synthesis; no claim that this split is universally faster |
| Direct language, scoped absolutes, relevant examples/defaults | Contributor writing rules, E-6/E-9/E-10 | Authoring guidance; retain detail that addresses demonstrated failures |
| Position, cost, and token behavior require measurement | E-1/E-2/E-3/E-11/E-12 | Evidence is model/task-dependent, not a Solomonar performance promise |
| Baselines, exceptions, misleading inputs, and held-out checks | E-4/E-5/E-9/E-13 | Proposed evaluation plan; no benchmark gain inferred from packaging |

All three stage procedures refine these owners: research owns source quality and
case design; authoring owns target format and writing; validation owns evidence
and measurements. Mandatory role boundaries remain in the existing framework.
No dependencies, architectures, rule-count targets, or arbitrary token caps are
added to adopting projects.

## Evaluation cases and expected outcomes

| Case | Expected outcome |
| --- | --- |
| New catalog skill | Strict catalog metadata, portable main/support files, normal generated adapters, no manual registration code |
| New project skill with extension commands unavailable | Project-owned unregistered source, one evidence record, no manifest/AGENTS/stub edits, explicit adoption gap |
| Project skill with `extension new` available | Use verified tracked-draft path when selected; CLI owns draft metadata and adoption |
| Specialist reviewer | Specialty-only contract with unique prefix and shared report/reviewer boundaries; no unshipped support references |
| Rules added to existing extension | Preserve active/retired IDs; allocate after historical maximum; refresh only affected evidence |
| Unsupported absolute or inaccessible source | Qualify/omit/investigate; no fabricated certainty or blocking policy |
| Request only to install an existing extension | Route to supported CLI ownership; do not author content or infer new policy |
| Candidate packages successfully but has no live evaluation | Separate packaging result from unmeasured behavior; no empirical-quality claim |

## Validation record

Framework verification completed:

- Module tidy/lint passed with zero lint findings; full race-enabled package
  tests passed. The aggregate `task check` reached its security step but could
  not fetch the vulnerability database inside the network sandbox. Rerunning
  only `task security` with network access passed: no vulnerabilities found.
- `task test:integration` passed, including the new
  `testdata/init_extension_authoring.txtar`. It installs the source candidate in
  a scratch project, checks verbatim support-file copies, all three runtimes'
  stubs pointing to the portable body, extension/runtime manifest ownership,
  activation routing, absence of a stack decision, and an unchanged rerun.
- `task build` completed. Its sandboxed Go invocation warned that it could not
  write optional module stat-cache data outside the workspace, but exited zero;
  the built binary successfully initialized the isolated Codex trial project.
- Site build and check passed (24 HTML pages checked). Metadata/frontmatter,
  current version, unique rule IDs, local links/anchors, Markdown, whitespace,
  and final newlines passed focused checks.
- The bundled skill-creator Python validator could not import `yaml` because
  PyYAML is absent. No dependency was installed. Actual CLI installation parsed
  the frontmatter and catalog, and focused checks covered the simple metadata,
  naming, references, and unfinished scaffolding directly; the unavailable
  validator is not reported as having passed.

Adding the Workflow catalog group exposed expected interactive-input and
known-extension diagnostic fixture changes. Updated existing tests for those
observable behaviors; no production Go code or CLI command implementation changed.

### Authoring trials

Fixtures: `/tmp/solomonar-authoring-trial-18fpz3zf/`. Each used a small supplied
release-summary policy and an existing research brief, with authoring-only scope.
No package installation, publication, or history mutation was authorized.

| Trial | Observed result | Limit |
| --- | --- | --- |
| Catalog source | Created strict `extension.json` and a portable workflow `SKILL.md`, mapped five rules to supplied policy, updated the existing record and README, and reported missing fixture site/build files without inventing them. | Reduced checkout; authoring only. The initial trial inherited parent context. |
| Installed project route | Created only project-owned `tools/solomonar/extensions/release-summary/SKILL.md` and updated its research record. Preserved manifest/stubs and reported unsupported adoption. | Proposed metadata incorrectly used its own source location as the governed `paths` value. The initial trial inherited parent context. |

The project trial revealed an ambiguity worth correcting: `paths` is the work
an extension governs, not the location where its own instructions are stored.
The authoring reference now states that distinction and uses an empty list for
request-driven procedures without a relevant governed-file glob. This is based
on the existing CLI field semantics, not a new rule imposed on all extensions.
A fresh-context project trial using the revised reference produced `paths: []`
for this request-driven workflow, created only the project source and its existing
research record, and kept the missing-adoption boundary explicit. Structural
checks and a pre/post SHA-256 comparison of fixture files passed for the authored
scope. Its `bin/solomonar` symlink exposed an external rebuild by the parent
packaging check; the evaluating agent detected and reported that change rather
than modifying or concealing it.

These are limited observations of this authoring skill creating a source
candidate, not execution of the newly authored release-summary skill. Research
and validation stages have not had separate live forwarding trials. No paired
baseline/candidate study, positional experiment, Claude/Copilot live discovery,
or statistically supported accuracy/latency improvement is claimed.

### Context cost

File-content proxies, measured as words and UTF-8 bytes:

| Loaded route | Words | Bytes |
| --- | ---: | ---: |
| Entrypoint only | 670 | 4,974 |
| Entrypoint + research + evidence | 2,362 | 18,213 |
| Entrypoint + authoring | 1,468 | 10,654 |
| Entrypoint + validation | 1,247 | 9,329 |
| Entire portable package | 3,737 | 28,248 |

These counts exclude generated provenance, discovery metadata, shared project
instructions, tool output, and accumulated conversation. Stage routing prevents
unnecessary upfront reads; it does not unload previous stages from the same
chat. A fresh session can resume from the single research record. Actual model
tokens, total run cost/cache use, and latency were not measured; this is not a
claim that the new package is smaller than every earlier prompt invocation.

### Readiness

Technical support and structural checks are complete for the scoped candidate.
Packaging is verified, including all three generated adapter formats and the
recheck after clarifying `paths`. Live authoring observations cover both target
modes; behavior claims remain limited as described above. New installations from
this source can select the optional skill. Existing installed projects retain
their recorded choices until supported lifecycle management becomes available.
All repository changes remain uncommitted.
