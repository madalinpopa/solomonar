# Research an extension

Use the target mode and research record selected by `SKILL.md`.

## Establish the contract

1. Inspect existing instructions and the target extension, including its rule
   history. Choose a skill, rules in an existing skill, or a specialist reviewer.
   Rules are skill content, not a third extension kind. Prefer extending an
   existing owner when the outcome does not justify a separate extension.
2. In catalog mode, read the relevant sections of `CONTRIBUTING.md`,
   `assets/templates/AGENTS.md`, and active `assets/framework/` contracts. Use the
   Go extension as a loading/packaging example, not as a mandatory topic layout.
   Inspect `docs/cli/interface.md` sections 4.1–4.3 and `internal/project/` only
   for unresolved schema, rendering, installation, or discovery questions.
   `docs/design/framework-v0.1.md` is historical.
3. In project mode, read the applicable installed extension entrypoints and the
   relevant `docs/solomonar/` contracts. For a reviewer, inspect `report.md`,
   `reviewer.md`, and an adjacent reviewer; for a skill, inspect the nearest
   relevant installed skill. Do not assume a project contains Solomonar's source
   code, contributor guide, Go tools, or repository-specific check commands.
4. Use `solomonar help` when the CLI is available. Distinguish implemented
   commands from planned documentation. Adoption is separate from authoring.
   If source or required framework contracts cannot be found, report that gap;
   keep independent research useful without inventing an installation contract.

Record kind, group, activation condition, governed paths, required/conflicting
extensions, and overlaps. Supporting procedures normally use `workflow`, extra
reviewers `review`; technology choices belong to the relevant stack group.
Confirm catalog metadata against the current schema in catalog mode. Project
metadata is CLI/manifest-owned; keep proposed values in the research record.

## Validate the topic and the instruction design

Open current primary sources: specifications, official versioned documentation,
maintainer guidance, original papers, and reproducible evaluations. Research
the extension's domain, not only how to write skills. Search for counterexamples
and conflicting findings; snippets and recollection are leads, not evidence.

Load [evidence.md](evidence.md) for the authoring source map. Recheck relevant
version-sensitive guidance and investigate applicable questions about context,
instruction clarity, examples, positional bias, and performance. A fresh topic
does not require rereading unrelated papers already covered by a current record.

For each substantive proposed instruction, record a claim ID and:

| Field | Required content |
| --- | --- |
| Proposed requirement | Action, applicability, concrete failure prevented, exceptions |
| Support | Direct URL or repository source/decision, supporting section, version/date |
| Class | Technical requirement, scoped recommendation, empirical result, project policy, or hypothesis |
| Limits | What the source establishes, what remains unproved, conflicting evidence |
| Disposition | Keep, qualify, omit, or investigate; intended verification |

Group equivalent claims without leaving requirements untraced. Evidence that a
domain rule is correct is distinct from evidence that putting it in a prompt
improves an agent. Do not promote preferences, corporate style guides, or a
benchmark result into universal mandatory behavior. Record inaccessible sources
and unsupported effectiveness claims explicitly. Stop searching once the scoped
requirements have adequate support or an explicit unresolved disposition.

## Plan context and validation

Specify discovery metadata, common entrypoint rules, conditional topic triggers,
and the files each task actually needs. Include indirect callers/contracts and
missing-input handling. Important trigger information must remain visible even
when detail is conditional; neither an index nor a link automatically loads it.
Skill support files ship; a catalog agent currently installs only its main file.

Treat the U-shaped lost-in-the-middle effect as a model/task-dependent risk.
Propose positional testing when relevant; Markdown headings alone do not control
the assembled context. Preserve exceptions rather than compressing them away.

Define expected outcomes before authoring: matching/nonmatching requests, a
subtle domain defect, a valid exception, indirect impact, overlapping extensions,
missing input/tools, and misleading source material where relevant. Combine
cases that exercise the same risk. Identify baseline/candidate conditions and
held-out cases; reuse available evaluation tools rather than building a harness
without a demonstrated need.

Update the record with this evidence, the proposed file/loading map, unresolved
choices, source revision, and next action. Preserve prior adoption and retired-ID
history. Continue to authoring when the requested scope is supported; a material
unresolved choice blocks only the work that depends on it.
