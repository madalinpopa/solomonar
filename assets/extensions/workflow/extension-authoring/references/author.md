# Author an extension

Read the research record and inspect the current target before editing. Refresh
only the evidence affected by changed contracts, versions, or scope. Unsupported
requirements remain questions or are omitted, rather than becoming mandatory.

## Choose the output

| Target | Source to create/edit | Metadata |
| --- | --- | --- |
| Catalog skill | `assets/extensions/<group>/<name>/SKILL.md` and routed support files | `extension.json` beside it, using the current strict schema |
| Catalog reviewer | `assets/extensions/<group>/<name>/<name>.md`, normally group `review` | `extension.json` beside it, `kind: agent` |
| New project extension | `docs/solomonar/extensions/<name>/SKILL.md` or `<name>.md` and support files | `extension.json` beside it, same schema as the catalog |
| Existing project extension | Its verified store directory | Preserve its name and rule IDs; edit its `extension.json` in place |

`extension.json` fields are `name`, `kind`, `group`, `title`, `requires`,
`conflicts`, `applies_when`, and `paths`, plus optional `detect`, `default`, and
`invocation`; the strict loader rejects unknown fields. Verify the allowed
groups and that the name is unique across the catalog and the store. `name`
matches the directory and the main file's frontmatter; a catalog `group` matches
its directory. Do not silently add dependencies; a project extension may require
catalog or project extensions. `paths` matches work governed by the extension,
not the extension's own location. Use an empty list for a request-driven
procedure with no relevant governed-file glob.

- `detect`: repository-file globs whose presence suggests the extension, such as
  `**/go.mod`. Globs are repository-relative, never start with `/`, and use `**`
  only as a whole segment. Omit it or use `[]` when no file reliably signals the
  need; never match the extension's own directory.
- `default`: `true` suggests the extension in every project; reserve it for
  procedures most projects use. Omit it otherwise.
- `invocation`: `user` when only the user should start the skill, such as a
  procedure with side effects or a long workflow; omit it for `model`. An agent
  is never user-invoked.

Detection and defaults only suggest; nothing installs without the user's choice.
A project extension has no rule authority or triage role until it is installed.

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

Every file in the extension directory is used verbatim. Link support files
relative to the file that references them and keep them inside that directory.
Recheck current CLI support before assuming further packaging capabilities.

## Integrate without registering by hand

In catalog mode, update the affected catalog listings and contributor docs
named by `CONTRIBUTING.md`. Let the existing installer copy the package and
generate stubs; no new registration code or copied runtime body is needed.

In project mode, write only the store directory and the research record. The
handoff is `solomonar extension add <name>`, run by the user, or by the owning
orchestrator with `--decision <approved-ID>`. Follow framework authority for the
actual adoption; authoring alone supplies no adoption approval. Body edits to an
installed extension apply directly; after changing its `extension.json` or
frontmatter, rerunning `extension add <name>` refreshes its stubs and
`AGENTS.md` line. Do not use `init` to change recorded extension choices or
hand-write a manifest, runtime stub, or activation entry.

Map final rules/procedures/checks to the supported claim IDs in the single
research record. Preserve stable IDs and prior retirement/adoption history.
Check metadata, links, Markdown, IDs, trigger destinations, exceptions, and
evidence coverage. Prepare validation cases using existing coverage; avoid
tests that merely assert headings or the instruction wording exists. Continue
to validation unless the user requested authoring only.
