# Prompt 1: Research and scope an extension

Replace the bracketed inputs, then send the prompt below. Continue with
[Prompt 2](02-author.md) when you want the agent to create the extension.

```text
Research a Solomonar extension for the following request:

Topic and desired user outcome: <topic and outcome>
Requested form: <new skill | rules for an existing skill | specialist reviewer | infer>
Proposed name or existing extension: <name or infer>
Target technology versions and runtimes: <constraints or discover>
Explicit exclusions: <exclusions or none>

Produce a repository-grounded research and authoring brief. The goal is accurate
agent behavior with only the context needed for the task. This stage may write
research notes, but does not change active framework assets. Preserve unrelated
work; do not commit or publish anything unless requested.

1. Establish the current contract.
Read applicable repository instructions, README.md, and the relevant framework
authoring sections of CONTRIBUTING.md. Inspect assets/templates/AGENTS.md and
the relevant active assets under assets/framework/. For a reviewer, read
report.md, reviewer.md, and an adjacent reviewer. For a skill or rule change,
inspect assets/extensions/language/go/ and the target extension, if any.

Use docs/cli/interface.md sections 4.1-4.3 and internal/project/{catalog,
extension,render,runtime}.go only as needed to establish packaging, rendering,
and discovery. Check implementation/tests when documentation describes planned
behavior. docs/design/framework-v0.1.md is historical, not the current contract.
Search first; load relevant sections rather than the entire repository.

Choose the smallest form that serves the outcome. Catalog kinds are skill and
agent; rules belong to an existing or new skill. Technology knowledge stays in
extensions. A new required process role or step is a separate core design change.
Explain the group, activation condition, paths, owners, and overlap with existing
rules. Do not invent catalog fields, groups, commands, or runtime capabilities.

2. Research the topic and the instruction design.
Find current primary sources: specifications, official versioned documentation,
maintainer guidance, original papers, and reproducible evaluations. Open the
sources and read the supporting passages; snippets and model recollection are
leads, not evidence. Seek counterexamples, exceptions, and conflicting findings.

Use docs/research/extension-authoring.md as a source map, not timeless authority.
Recheck version-sensitive claims and investigate practices relevant to this
extension: precise instructions, selective loading, context growth, positional
bias / the U-shaped lost-in-the-middle effect, examples, verification, and the
tradeoffs among accuracy, tokens, tool calls, cost, and latency. Research the
extension's domain as carefully as the agent-writing practices.

For every substantive proposed rule, procedure, check, or authoring practice,
record a claim ID and:
- Proposed instruction, applicability, exceptions, and concrete failure prevented.
- Direct source URL and supporting section; version/date checked.
- Evidence class: technical requirement, scoped maintainer recommendation,
  empirical result, existing project policy, or unvalidated design hypothesis.
- What the evidence establishes, what it does not, and conflicting evidence.
- Verification method and disposition: keep, qualify, omit, or investigate.

Group genuinely equivalent claims, but leave no substantive requirement untraced.
Separate evidence that a domain rule is correct from evidence that writing it
into a prompt improves agents. Mark unsupported effectiveness claims unvalidated.
A corporate style guide is not a universal requirement; a benchmark result is
not proof for Solomonar or every current model. If sources are inaccessible,
record the gap rather than manufacturing validation. Stop searching when the
proposed requirements have adequate support or explicit unresolved dispositions.

3. Design the minimum useful context and evaluation.
Specify what discovery metadata exposes, what the entrypoint always loads, and
which impact triggers load each topic file. Include affected callers/contracts,
and specify what happens when required input is missing. Keep research outside
installed assets. Account for actual installer limits: skill support files are
copied, but an extension agent currently installs only its main Markdown file.

Keep critical scope and prerequisites visible. Do not assume Markdown headings
control a file's position in the runtime's assembled context, or that repeating
everything at both ends solves positional bias. Propose a beginning/middle/end
evaluation if placement matters. Preserve exceptions when shortening text.

Define representative cases and expected outcomes before authoring: matching
and nonmatching requests, a subtle domain defect, a valid exception, indirect
impact, overlapping extensions, missing input, and misleading source material.
Combine cases where appropriate. Define baseline and candidate conditions using
the same model, runtime, task inputs, tools, and budget. Choose an existing
evaluation method; do not build a new harness merely for this extension.

Create or update docs/research/<name>-rules/review.md with the scope, repository
references, evidence matrix, rejected advice, proposed file/loading map, and
evaluation cases with expected outcomes. Preserve prior adoption and rule
retirement history. End with a compact handoff naming the brief, source revision,
proposed files, retained claim IDs, and unresolved items.
Ask only for missing information that materially prevents a sound decision.
```
