---
name: extension-authoring
description: Researches, authors, and validates Solomonar extension skills, rules, and specialist reviewers, in the catalog or as project-specific source. Use for extension content; adoption and runtime registration belong to the CLI.
---

# Extension authoring skill

Framework version: {{ .Version }}

## 1. Purpose and boundaries

Create supported extension content through research → authoring → validation.
This is an optional supporting procedure, not a new mandatory process step.

- Catalog mode edits the selected `assets/extensions/<group>/<name>/` package,
  its research, and the related contributor documentation and check fixtures.
- Project mode edits a project extension and its research. It lives in
  `docs/solomonar/extensions/<name>/` in catalog format: `extension.json` plus
  `SKILL.md` or `<name>.md` and support files. Confirm ownership before editing
  an existing directory; installed catalog extensions there belong to the CLI.
- Keep one portable instruction body. The CLI generates runtime stubs that load
  it; source authoring does not activate an extension or approve its adoption.
- Follow the owning task/role when invoked in managed work. Reviewers retain
  their read-only contract; return proposed fixes rather than applying them.
- Do not edit the core, application code, unrelated configuration, or workflow
  state through this skill. Report a needed core process change separately.

## 2. Start and route

1. Read `AGENTS.md` when present and inspect the worktree without changing history.
   Identify the requested outcome, target name/kind, supported versions, and scope.
   Preserve unrelated edits; commit or publish only when the user requests it.
2. Select catalog mode only for an explicitly targeted Solomonar source checkout
   with `assets/framework/` and `assets/extensions/`. Otherwise use project mode.
   An ambiguous target or conflicting ownership needs clarification before edits.
3. Use the existing extension research record, otherwise
   `docs/research/<name>-rules/review.md`. It owns sources, decisions, evaluation
   cases, results, and the next action; do not duplicate these in stage reports.
4. Select the stage below. A request to create an extension covers all three
   stages within its authorized scope. Stop at a stage only when requested or
   when a material unresolved choice/input prevents dependent work. Reuse scoped
   approvals; do not request confirmation again between stages automatically.

Paths in this table are relative to this skill. Load only the current stage;
read another when advancing or when a concrete dependency requires it.

| Stage | Load | Completion/handoff |
| --- | --- | --- |
| New topic, changed requirements, or inadequate/stale evidence | [references/research.md](references/research.md) | Evidence, target/file map, expected evaluation outcomes |
| Supported scope ready to create or update | [references/author.md](references/author.md) | Source candidate, final claim/rule mapping, structural checks |
| Candidate ready to assess or refine | [references/validate.md](references/validate.md) | Packaging and behavior results, fixes, remaining gaps |

The research stage alone loads [references/evidence.md](references/evidence.md)
as a source map for authoring practices; refresh relevant claims rather than
treating it as timeless authority. Ordinary tasks using the authored extension
do not load this skill or its research merely because that extension is installed.

## 3. Common rules

Cite `extension-authoring/C-n`; framework authority and review gates still apply.

- extension-authoring/C-1 (must) — Rule: Trace each substantive domain rule,
  required procedure, and check to its source, scope, exceptions, and verification
  in the research record; distinguish established policy from hypotheses.
- extension-authoring/C-2 (must) — Rule: Leave the manifest, generated runtime
  stubs, `AGENTS.md` activation routes, and decision records to
  `solomonar extension add|remove`; never register an extension by hand.
- extension-authoring/C-3 (must) — Rule: Preserve rule IDs and retirement history;
  assign new IDs after the highest ever used in that extension, without reuse.
- extension-authoring/C-4 (must) — Rule: Report source support, structural checks,
  manual walkthroughs, and observed live behavior separately; missing or failed
  required verification cannot become a completed pass.

## 4. Finish

Update the single research record with the target/mode, files changed, checks and
their limits, unresolved choices, and next action. Report whether content is a
source candidate, validated, or actually installed; cite the actual source
revision when known and distinguish working-copy edits from committed content.
In project mode, end by handing off `solomonar extension add <name>` for the user
to run; name any approval it needs, since adopting a stack-group extension is an
architecture or dependency decision. The CLI owns activation and removal, the
manifest, runtime stubs, and conflict handling; this skill owns the substance,
metadata, and evidence of the instructions.
