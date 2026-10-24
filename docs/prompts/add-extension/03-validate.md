# Prompt 3: Validate and refine the extension

Use after [Prompt 2](02-author.md), preferably in a fresh chat for a review that
does not depend on the author's conversation. This prompt permits local fixes;
formal delegated reviewers retain their framework read-only contract.

```text
Validate and refine this Solomonar extension:

Extension name and candidate paths: <name and paths>
Research brief: <docs/research/name-rules/review.md>
Additional constraints: <constraints or none>

Determine whether the candidate is technically supported, integrates correctly,
and helps agents on representative tasks. Fix scoped defects you establish,
then rerun affected checks. Preserve unrelated work. Do not commit or publish.
Follow the applicable framework workflow; do not bypass its review or approval
contracts. Use available authorized runtimes, not invented evaluations.

1. Audit the instructions and evidence.
Read applicable repository instructions, relevant CONTRIBUTING.md requirements,
the candidate, and the brief. Verify every substantive rule/procedure/check
against its supporting source and scope. Reopen affected external sources where
wording, versions, or applicability are uncertain. Seek at least one realistic
counterexample to each proposed absolute, grouping shared exceptions where valid.
Correct unsupported universals, contradictions, vague triggers, missing
exceptions, duplicate ownership, ID reuse, and unapproved policy. Verify that
recommendations and verification gaps cannot be reported as confirmed defects.

2. Validate what is actually installed.
Follow CONTRIBUTING.md's framework verification: task check, task
test:integration, task build, and installation into an isolated scratch project
using the candidate binary's supported init options. Run them through any
applicable command wrapper. Inspect help/source first; extension management or
upgrade commands described in design documents may still be planned.

Inspect the installed entrypoint and topic files, AGENTS.md activation routes,
manifest ownership, and generated runtime adapters. Check supported runtimes
and relevant combination-section presence/absence. Reuse packaging tests; add
coverage only for a real missing behavior contract. Never hand-edit generated
output to make the candidate work. Record unavailable checks and runtime
acceptance separately; successful packaging does not demonstrate agent quality.

3. Evaluate behavior against the predeclared cases.
Use the brief's expected outcomes, grounded in domain sources and project
requirements, rather than the candidate's own wording as the answer key. Compare
the previous extension, or no extension if new, with the candidate under matched
model/version, runtime, tools, task inputs, budgets, and isolation. Keep mandatory
project/framework constraints in both conditions. Repeat cases where practical,
record the run count, and keep held-out cases for the final comparison.

Include relevant activation and nonactivation, subtle defects, valid exceptions,
indirect impact, overlapping extensions, missing evidence/tools, and misleading
material. Test actual behavior and loaded files, not merely whether the final
answer mentions the rule. A manual walkthrough is useful but is not a live run.

For skills, assess missed requirements, false rule violations, unnecessary
changes, and correct handling of exceptions. For reviewers, measure confirmed
defects found, false positives, missed defects, evidence quality, and calibrated
incomplete outcomes. For both, check routing, authority, and required coverage.
Keep research-only documents out of ordinary task context.

If positional sensitivity is relevant, vary the same critical fact/instruction
across the beginning, middle, and end of realistic assembled contexts while
holding content and task constant. Test long contexts and plausible distractors
where feasible. A U-shaped result is a hypothesis to measure, not a universal
property or a reason to repeat the whole policy.

4. Measure efficiency without weakening correctness.
Record actual loaded files, input/output tokens, tool calls, elapsed time, and
cost/cache usage where exposed. Otherwise report words/UTF-8 bytes as proxies
and mark unavailable metrics; do not label bytes/4 as measured tokens. Include
discovery metadata and shared context overhead, narrow-task routes, and a
cross-cutting route. Count total work across agents if delegation was required.

Remove or simplify one instruction group at a time and rerun the affected cases.
Prefer the simpler candidate when it preserves required outcomes and acceptable
false-positive/false-negative rates. Keep a longer candidate when its additional
context addresses a demonstrated failure. Report regressions and tradeoffs;
fewer input tokens do not guarantee lower end-to-end latency or cost. Caching
and prompt assembly are runtime concerns that extension text alone cannot prove.

5. Close the evidence record.
Update the brief with commands, tested revisions/configurations, expected and
actual results, fixes, measured/proxy context costs, rejected optimizations, and
remaining gaps. For each quality claim, say whether it is source-supported,
structurally checked, manually exercised, or observed in live agent runs.

Report readiness separately for technical evidence, packaging, and agent
behavior. Failed or missing required verification prevents a complete pass.
If a runtime or evaluation facility is unavailable, finish all independent
checks and provide the exact remaining cases/commands without fabricating a
result. Do not describe the extension as empirically validated until live
evidence supports that statement.
```
