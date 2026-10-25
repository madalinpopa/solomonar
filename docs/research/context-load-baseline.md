# Framework context-load baseline

Measured: 2026-10-25. Proposal: [request-scoped framework context](../design/context-loading.md).

## Scope and method

- Input: 36 files under `assets/framework/`, `assets/templates/`, and `assets/extensions/` in the working tree.
  The snapshot includes existing uncommitted framework edits; the parent commit is `7ef616a1`.
- Input fingerprint: `3145d6edfb5251f11bacf63e4a82e47d320072b49b489691c9993059798e6be9`.
  Reproduce by SHA-256 hashing each sorted relative path, NUL, raw file bytes, NUL, consecutively.
- Tokenizer: `tiktoken` 0.14.0; reference encodings `o200k_base` and `cl100k_base`. These are exact counts for
  those text encodings, not confirmed tokenizers for every supported runtime/model.
- Normalize `{{ .Version }}` and `{{ .FrameworkVersion }}` to `0.0.0`, matching
  [release stamping](../../internal/release/release.go). Render AGENTS with no extensions using
  [extensionsBlock](../../internal/project/extension.go); use no project notes.
- Render the seeded docs README with project name `Example` and objective `Not established.`. JSON files are
  unchanged. Count complete UTF-8 text, including frontmatter, comments, Markdown, and examples.
- Route totals sum each required file once, encoded independently. Message framing, file labels, tool output
  formatting, generated stubs, discovery catalogs, and cross-message boundaries are excluded.
- A conditional source is added only when its trigger applies. Merely citing a section does not require
  loading that whole file. Project documents, code, decisions, and command outputs have variable additional cost.
- Tokenizer installation and measurement scripts were temporary, outside the repository. No model calls were
  run to simulate framework workflows; no live per-call usage is claimed.

## File inventory

Paths are relative to the repository root. `o200k` is the primary comparison unit throughout the proposal.

| File | o200k | cl100k |
| --- | ---: | ---: |
| `assets/templates/AGENTS.md` | 1,259 | 1,258 |
| `assets/templates/docs/README.md` | 145 | 146 |
| `assets/templates/state.json` | 10 | 10 |
| `assets/framework/rules.md` | 6,959 | 6,966 |
| `assets/framework/report.md` | 2,765 | 2,766 |
| `assets/framework/reviewer.md` | 1,464 | 1,461 |
| `assets/framework/registry.md` | 1,605 | 1,606 |
| `assets/framework/agents/orchestrator.md` | 4,534 | 4,539 |
| `assets/framework/agents/implementer.md` | 3,083 | 3,083 |
| `assets/framework/agents/security-reviewer.md` | 1,351 | 1,344 |
| `assets/framework/agents/correctness-reviewer.md` | 1,049 | 1,042 |
| `assets/framework/agents/spec-reviewer.md` | 1,434 | 1,433 |
| `assets/framework/agents/adversarial-reviewer.md` | 1,575 | 1,576 |
| `assets/framework/skills/solomonar/SKILL.md` | 1,274 | 1,278 |
| `assets/framework/skills/prototype/SKILL.md` | 1,793 | 1,789 |
| `assets/framework/skills/prototype/references/cli.md` | 211 | 211 |
| `assets/framework/skills/prototype/references/module.md` | 189 | 189 |
| `assets/framework/skills/prototype/references/service.md` | 231 | 232 |
| `assets/framework/skills/prototype/references/web.md` | 261 | 259 |
| `assets/framework/skills/prototype/template.md` | 585 | 583 |
| `assets/framework/skills/spec/SKILL.md` | 2,307 | 2,304 |
| `assets/framework/skills/spec/template.md` | 369 | 368 |
| `assets/framework/skills/work-spec/SKILL.md` | 3,164 | 3,157 |
| `assets/framework/skills/work-spec/template.md` | 456 | 452 |
| `assets/extensions/language/go/SKILL.md` | 561 | 560 |
| `assets/extensions/language/go/rules/go/design.md` | 358 | 358 |
| `assets/extensions/language/go/rules/go/io.md` | 333 | 331 |
| `assets/extensions/language/go/rules/go/concurrency.md` | 210 | 210 |
| `assets/extensions/language/go/rules/go/testing.md` | 308 | 308 |
| `assets/extensions/language/go/extension.json` | 93 | 93 |
| `assets/extensions/workflow/extension-authoring/SKILL.md` | 1,061 | 1,061 |
| `assets/extensions/workflow/extension-authoring/references/research.md` | 1,013 | 1,017 |
| `assets/extensions/workflow/extension-authoring/references/evidence.md` | 1,745 | 1,750 |
| `assets/extensions/workflow/extension-authoring/references/author.md` | 1,338 | 1,340 |
| `assets/extensions/workflow/extension-authoring/references/validate.md` | 891 | 893 |
| `assets/extensions/workflow/extension-authoring/extension.json` | 95 | 95 |

- Core framework files: 36,659 o200k tokens; templates: 1,414; catalog extensions: 8,006.
- Entire inventory: 46,079; Markdown subset: 45,881. These are availability totals, not startup loads.
- Extension manifests and registry seed are data, not mandatory agent instructions. Historical design docs,
  contributor/site docs, images, source code, and generated runtime configuration are outside these totals.

## Current request costs

Aliases: **A** = rendered AGENTS; **S** = entry skill; **O** = orchestrator; **R** = rules;
**P** = report; **V** = reviewer; **W** = work-spec skill; **C** = capability-spec skill.
All counts below use o200k. They describe fresh framework text or its retained union, not whole model requests.

| Request | Required file set | Tokens |
| --- | --- | ---: |
| Direct project/process question | A; add question-specific excerpts as needed | 1,259 before excerpts |
| Status/overview through entry | A + S; add actual CLI/project output | 2,533 before output |
| Existing CLI prototype, direct | A + prototype skill + CLI reference | 3,263 |
| New CLI prototype README, direct | Previous + prototype template | 3,848 |
| Existing capability spec, discovery, direct | A + C | 3,566 |
| New capability spec, discovery, direct | A + C + spec template | 3,935 |
| Orchestrator, direct role selection | A + O + R + P | 15,517 |
| Orchestrator through entry | A + S + O + R + P | 16,791 |
| Work-spec in that orchestrator context | Previous + W | 19,955 |
| New work-spec file | Previous + work-spec template | 20,411 |
| Work-spec followed by canonical spec changes | A + S + O + R + P + W + C; no templates | 22,262 |
| Delegated implementer | A + P + implementer | 7,107 |
| Security reviewer | A + P + V + security role | 6,839 |
| Correctness reviewer | A + P + V + correctness role | 6,537 |
| Spec reviewer | A + P + V + spec-reviewer role | 6,922 |
| Adversarial reviewer | A + P + V + adversarial role | 7,063 |

- Explicit entry invocation before a standalone prototype/spec request retains another 1,274 tokens.
- Project decision through entry: 3,928 with only `rules.md` section 3, versus 9,492 if the whole file is read.
  This illustrates why section references and mandatory full-file loads must be distinguished.
- Registry detail adds 1,605 only when its command/rejection/field trigger applies.
- Go entry + testing = 869; all Go Markdown = 1,770, excluding extension registration/discovery overhead.
- Spec-reviewer verification cites work-spec section 4. Reading that section adds 540; reading the whole skill
  instead adds 3,164. The route table excludes this conditional reference read, not the verification obligation.
- Three specialists total 20,298 across separate agents. A tier-2 pass with a 22,262-token orchestrator,
  one implementer, and all four reviewers totals 56,730 framework tokens loaded once across six agents.
  This excludes templates, extensions, repeated calls, retries, inherited history, and returned reports.

## Largest removable unconditional loads

Section counts include their headings and following content, stopping at the next numbered second-level heading.
Moving text defers its cost; it does not eliminate its cost when later required.

| Current content | Tokens | Trigger that should own it |
| --- | ---: | --- |
| Rules §3: decisions | 1,395 | Actual decision or conflict; project decision route |
| Rules §4: starting a task | 634 | Setup |
| Rules §6: reviews | 949 | Triage, review, consolidation, reuse assessment |
| Rules §§8–9: merge and closure | 896 | Integration/closure |
| Rules §10: recovery | 739 | Resume, handoff, pause/cancel, drift |
| Report §7: correction list | 263 | Consolidation or correction packet |
| Report §8: PR Verification | 310 | PR/integration |
| Implementer §§6–7: correction/integration | 636 | Assigned mode |
| Orchestrator §3: all stage procedures | 2,076 | Current stage plus discovered dependencies |
| Orchestrator §4: all delegation packets | 899 | Preparing the applicable delegation |

These rows overlap in responsibility and do not constitute an additive savings forecast. Shared invariants,
transition gates, redirect text, and necessary cross-stage dependencies must remain in the required closure.

## Per-call accounting

The [CLI's registry contract](../../assets/framework/rules.md) states that it never launches agents or calls
models. Repository files cannot reveal the actual prompts and usage of every model call.

For an agent and model call, distinguish:

- **New framework text:** content first loaded by this call's preceding reads, plus any duplicate reads.
- **Retained framework text:** prior framework content still present, plus new text; compaction may replace it.
- **Full input:** runtime/system instructions, tool definitions/catalogs, retained framework text, task/code
  context, user messages, prior responses, and tool results.
- **Usage:** observed input, cached input, output, and reasoning tokens where the host exposes them. Cached
  input is a subset of input; reasoning is a subset of output. Missing telemetry is not zero.

Illustrative progression with no compaction, duplicate reads, templates, or extensions:

| Context after action | Newly loaded framework text | Retained framework text |
| --- | ---: | ---: |
| Project instructions available | 1,259 | 1,259 |
| Read entry skill for status | 1,274 | 2,533 |
| Later take orchestrator role | 14,258 | 16,791 |
| Read work-spec skill | 3,164 | 19,955 |
| Next call, no new framework reads | 0 | 19,955 |
| Read capability-spec skill | 2,307 | 22,262 |
| Start isolated implementer | 7,107 in child | Child: 7,107; parent retains its own context |

Each subsequent model call can still carry the retained text. Ten calls retaining 16,791 framework tokens
account for 167,910 framework-token input occurrences before framing; they are not ten new file loads.
Actual cache charges and compaction change the cost model, so this arithmetic is not a billing estimate.

For a measured before/after walkthrough, record runtime/version, model, session/agent, call identifier,
request/stage/mode, source revision, loaded files/sections, duplicate reads, compaction, and available usage.
Use existing harness traces; Solomonar needs no new model integration. Compare equivalent completed work,
including all child calls and retries; do not double-count cumulative usage as per-call deltas.

## Runtime evidence and limits

- Codex discovers skill names/descriptions/paths before selecting a skill, then reads its full instructions.
  Keep required details in conditional references rather than relying on headings to prevent a full initial
  skill read. See [official skills documentation](https://learn.chatgpt.com/docs/build-skills).
- Applicable AGENTS files contribute persistent instruction context. Nested files do not remove root text;
  use root instructions for invariants and explicit references for optional detail. See
  [official instruction discovery](https://learn.chatgpt.com/docs/agent-configuration/agents-md).
- The four core skill metadata field payloads total 281 tokens when each name, description, and relative
  adapter path is separated by newlines. This is a comparison count, not the host-rendered discovery message.
  Extension/agent metadata, absolute paths, wrappers, and user-installed plugins add separate overhead.
- Model input also includes history and tool results; child agents consume their own calls. Caching reuses
  processing of matching context rather than removing that context. See
  [official usage accounting](https://developers.openai.com/api/docs/guides/agents-api/observability#model-usage-and-cost).
- The actual Claude/Copilot instruction assembly, model tokenizers, history inheritance, and cache behavior
  were not measured. Test each supported adapter's loading behavior before claiming equivalent savings.
