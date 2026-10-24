# Prompt 2: Author the extension

Use after [Prompt 1](01-research.md). Supply the research path even in the same
chat so the task can resume without relying on conversation history.

```text
Create or update the Solomonar extension described by:

Research brief: <docs/research/name-rules/review.md>
Requested scope or decisions since research: <changes or none>

Implement the supported scope in repository files. Optimize for accurate,
checkable behavior and economical context. Preserve unrelated work. Do not
commit, publish, or install into an adopting user's project unless requested.

Start from the brief and applicable repository instructions. Recheck the
relevant CONTRIBUTING.md templates and active assets. Compare the brief's source
revision with the current files; refresh affected evidence if contracts,
versions, or scope changed. Load only relevant research rows and references.
An unsupported requirement remains an explicit question or is omitted; it does
not become mandatory because it sounds useful.

Follow the selected form:

- New skill: create assets/extensions/<group>/<name>/extension.json and SKILL.md.
  Use only the current schema fields: name, kind, group, title, requires,
  conflicts, applies_when, paths. Verify allowed groups and unique names from
  the current catalog. Keep activation metadata specific to observable work.
- Existing-skill rules: edit the owning entrypoint/topic files; add a topic and
  its loading trigger only when it creates a useful independent context unit.
  Preserve existing rule IDs, retirements, and references. Allocate the next
  unused C-n across the extension; never renumber or reuse retired IDs.
- Specialist reviewer: create extension.json with kind agent and <name>.md,
  normally under assets/extensions/review/<name>/. Define its specialty,
  activation, method, checks, evidence requirements, and boundaries. Reference
  reviewer.md and report.md for shared contracts. Preserve read-only review of
  one candidate revision, report to the orchestrator, and choose a unique finding
  prefix. Do not duplicate another role's full review or invent new authority.

For entrypoints, use one-line name and description frontmatter and the current
Framework version from assets/framework/rules.md. Do not change release versions.
Combination sections are rendered only in a skill/agent main file; support files
are copied verbatim. Agent support files are not currently installed, so do not
create references to files that the adopting project will not receive.

Apply these writing constraints:

1. Use direct English and one actionable instruction per bullet. State the
   trigger, action, and observable requirement; include a real exception when
   needed. Avoid vague advice such as 'follow best practices'.
2. Use the repository's <name>/C-n (must|should) rule format. Reserve must for
   a supported, applicable requirement whose violation can be verified as a
   defect. A should deviation calls for a reason, not an automatic blocker.
   Do not turn an unapproved preference into policy or prescribe arbitrary
   thresholds, tools, architectures, or dependencies.
3. Keep only broadly applicable rules in the entrypoint. Route other rules by
   impact with explicit when-to-read paths, including indirect callers/contracts.
   All matching topics load before the governed work; missing required files
   remain missing input. An index or link does not load its target automatically.
4. Give each requirement one owner. Reference framework scope, approvals, review,
   and reporting contracts instead of copying them. Remove generic background
   unless evidence or an evaluation case shows that the agent needs it. Keep
   enough detail to identify non-obvious triggers and preserve important limits.
5. Prefer a default with a clear exception over a menu of equivalent methods.
   Add a concise example only when it resolves a demonstrated ambiguity or
   defines a necessary output contract. Do not demand hidden chain-of-thought;
   request conclusions, sources, checks, and concise decision rationale.
6. Keep research, long rationale, and evidence tables in the research brief.
   Runtime text should contain the decisions needed to act. Do not force
   ordinary users of the extension to research its authoring evidence again.
7. Prefer the adopting project's configured checks. Discover commands rather
   than assuming Solomonar's own tools exist there; document portable fallbacks
   and their limits. Mechanical checks belong in existing tooling when suitable.
   Missing verification never becomes a pass, and a tool pass proves only the
   properties it actually checks.

Update the evidence matrix so every substantive domain rule, required procedure,
and check maps to its final location/ID and support. Record changed/retired rule
meanings and exceptions. Update the relevant catalog listings and documentation
required by CONTRIBUTING.md; inspect existing content rather than inventing a
status section. Keep the core language-agnostic and generated adapters untouched.

Perform focused structural checks: metadata, unique IDs, references, Markdown,
trigger destinations, and completeness of the evidence mapping. Prepare the
representative cases for Prompt 3, reusing existing tests where they cover the
contract. Do not create tests that merely assert the new wording exists.

Report changed files, why each runtime instruction earns its context cost,
remaining uncertainties, and the research path needed for validation. This is
an authored candidate; do not claim measured agent improvements yet.
```
