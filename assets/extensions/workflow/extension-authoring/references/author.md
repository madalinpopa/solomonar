# Author an extension

Read the research record and inspect the current target before editing. Refresh
only the evidence affected by changed contracts, versions, or scope. Unsupported
requirements remain questions or are omitted, rather than becoming mandatory.

## Choose the output

| Target | Source to create/edit | Metadata |
| --- | --- | --- |
| Catalog skill | `assets/extensions/<group>/<name>/SKILL.md` and routed support files | `extension.json` beside it, using the current strict schema |
| Catalog reviewer | `assets/extensions/<group>/<name>/<name>.md`, normally group `review` | `extension.json` beside it, `kind: agent` |
| New project source | `tools/solomonar/extensions/<name>/SKILL.md` or `<name>.md` | Proposed CLI metadata in the research record, not an extra `extension.json` |
| Existing project source/draft | Its verified project-owned source path | Preserve recorded identity; CLI owns changes to registered metadata |

Catalog `extension.json` fields are `name`, `kind`, `group`, `title`, `requires`,
`conflicts`, `applies_when`, and `paths`. Verify the allowed groups and unique
name against the current catalog. Name/group match directories; do not silently
add dependencies. `paths` matches work governed by the extension, not the
extension's own source or installation path. Use an empty list for a
request-driven procedure with no relevant governed-file glob. A project source
directory is not a registered draft and has
no installed rule authority or reviewer-triage role.

Use the main file's one-line `name` and `description` frontmatter. Descriptions
say what the skill/reviewer does and when it applies. Use the applicable
framework's `Framework version` under the title; do not bump release versions.
In catalog mode follow `CONTRIBUTING.md`'s current templates. In project mode
use the corresponding installed contracts and adjacent source as the format.

## Write the content

- Use direct English, one actionable instruction per bullet. State the trigger,
  action, observable requirement, and a real exception when needed. Omit vague
  advice such as “follow best practices.”
- For domain rules use `<name>/C-n (must|should) — Rule: ...`, with `Exception:`
  where needed. A `must` violation must be supported and checkable as a defect;
  a `should` deviation needs a reason, not an automatic blocker.
- Keep broadly applicable rules and essential boundaries in the entrypoint.
  Route conditional topics by impact, including affected callers/contracts.
  Load matching topics before the governed work; absent required files remain
  missing input. Add a topic file only when it is a useful independent unit.
- Give each requirement one owner. Reference framework authority, approval,
  reporting, and review contracts rather than copying them into the extension.
  Supporting procedures should specify inputs, allowed edits, steps, and finish
  conditions, following the same ownership boundaries as other skills.
- Prefer a default with a clear exception over a menu of equivalent choices.
  Keep an example only when it resolves an ambiguity or defines a required
  output format. Request evidence and concise rationale, not hidden reasoning.
- Preserve non-obvious trigger information and exceptions when shortening text.
  Keep literature reviews and long rationale in the research record; ordinary
  use of the authored extension does not research its authoring method again.
- Prefer verified project check commands. Document portable fallbacks and their
  limits only where useful; do not assume this repository's tooling exists in
  another project. Existing tools own mechanical checks where suitable.

For a specialist reviewer, add only its specialty to the shared `reviewer.md`
and `report.md` contracts: focus, method, checks, evidence, blocking conditions,
and neighboring owners. Keep review read-only on one stable revision and report
to the orchestrator. Select a finding prefix not used by another installed role.

Combination sections belong only in the skill/agent main file; support files are
copied verbatim. A catalog reviewer installs only `<name>.md`: do not reference
unshipped support files. Recheck current CLI support before assuming that a
project extension offers additional packaging capabilities.

## Integrate without registering by hand

In catalog mode, update the affected catalog listings and contributor docs
named by `CONTRIBUTING.md`. Let the existing installer generate stubs from this
portable source; no new registration code or copied runtime body is needed.

In project mode, leave new source unregistered until supported adoption. When
`extension new` exists, the user can choose its tracked-draft path before
authoring; it is optional when a local source directory already exists. Prepare
the appropriate `extension add <path>` or `add <draft-name>` metadata handoff
only after confirming the command/flags exist. Follow framework authority for
the actual adoption; authoring alone supplies no adoption approval. If lifecycle
commands are unavailable, report that boundary and retain the source candidate.
Do not use `init` to change recorded extension choices or hand-write a manifest,
runtime stub, activation entry, or rendered combination.

Map final rules/procedures/checks to the supported claim IDs in the single
research record. Preserve stable IDs and prior retirement/adoption history.
Check metadata, links, Markdown, IDs, trigger destinations, exceptions, and
evidence coverage. Prepare validation cases using existing coverage; avoid
tests that merely assert headings or the instruction wording exists. Continue
to validation unless the user requested authoring only.
