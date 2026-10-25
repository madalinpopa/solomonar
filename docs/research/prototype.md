# Prototype workflow: evidence and validation

Researched 2026-10-25 against Solomonar 0.3.2 (`21d94a79`). The
[prototype skill](../../assets/framework/skills/prototype/SKILL.md) owns construction and handoff;
[AGENTS rule 9](../../assets/templates/AGENTS.md) owns the project-wide ordering constraint.

## Sources and rule selection

| Source / authority | Applied guidance and concrete reason |
|---|---|
| [Agent Skills specification](https://agentskills.io/specification) — format specification | Keep valid name/description metadata and direct relative references. Load only the application type needed instead of all four guides. |
| [OpenAI skill authoring](https://learn.chatgpt.com/docs/build-skills) — authoring guidance | Keep instructions imperative, scoped, and instruction-only; retain a compact entrypoint without adding scripts or another template structure. |
| [CLI Design Guidelines](https://clig.dev/) — design recommendations | Demonstrate help, scriptable inputs, stdout/stderr, meaningful exits, and actionable errors so a clickable substitute cannot conceal CLI behavior. |
| [WCAG 2.2](https://www.w3.org/TR/WCAG22/) — W3C Recommendation, selected success criteria | Apply keyboard access (2.1.1), visible focus (2.4.7), labels (3.3.2), and text error identification (3.3.1) to make flows inspectable without a pointer. This is not a claim of full WCAG conformance. |
| [Nielsen usability heuristics](https://www.nngroup.com/articles/ten-usability-heuristics/) — design recommendations | Use clear hierarchy, consistent controls, timely feedback, and understandable errors to reveal state rather than merely display screens. |
| [Python argparse](https://docs.python.org/3/library/argparse.html), [module execution](https://docs.python.org/3/library/__main__.html); [Node parseArgs](https://nodejs.org/api/util.html#utilparseargsconfig), [HTTP](https://nodejs.org/api/http.html), [test runner](https://nodejs.org/api/test.html) — official runtime capabilities | Standard parsers, executable consumers, local servers, and optional behavior tests need no application framework. Check availability in the selected installed runtime; do not mandate a language or newest version. |
| [MDN localStorage](https://developer.mozilla.org/en-US/docs/Web/API/Window/localStorage), [IndexedDB](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API), [browser modules](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Modules) — browser API documentation | Use small/simple or structured storage as needed. Replace the unconditional file-only rule: file-origin storage is undefined and browser modules need HTTP serving. Keep reset confined to prototype data. |
| [RFC 9110](https://www.rfc-editor.org/rfc/rfc9110.html) — IETF HTTP standard | Preserve method semantics, status codes, and media types so API simulations expose realistic contracts. Do not invent REST conventions or an error schema. |

The approval loop, behavioral authority, deterministic local simulations, runtime-first dependency policy,
and README/reset requirements are requested project policy, not external standards or empirically proven
universal development practices. They prevent unreviewed requirements, behavior drift, and setup-dependent
demonstrations. Prototype mechanisms never determine production architecture.

Scope refinement: a prototype may cover one function, component, module, or workflow. Organize independent
parts by type under `prototype/` and give each exact run and behavior-check instructions; preserve existing
runnable layouts. A minimal consumer/harness makes partial code inspectable without building the host app.

## Review coverage

- Preserve existing flow IDs/statuses, revision-pinned handoffs, document ownership, and Git task conventions.
- Check an existing approved flow change: mark affected flows draft, demonstrate/reapprove, then extract specs
  before production changes. Check direct spec requests, in-task decisions, and correction packets against the
  same gate; none substitutes task/plan approval for prototype approval.
- Keep ambiguous behavior and prototype/spec conflicts blocked until resolved in the prototype. Reuse an
  existing approved prototype for defect restoration; technical work preserving behavior needs no new prototype.
- Inspect each application route: CLI plus local JSON; browser plus native storage/local HTTP; module plus
  executable consumer/test doubles; service plus runtime HTTP/in-memory state/curl. None requires production
  credentials, services, or deployment. Shared construction rules stay in SKILL.md; type guides add only interface
  details, with no nested reference chains.

## Validation

- Passed the bundled skill-creator validator for all four core skills, with additional specification checks
  for nonempty descriptions, matching directory/name, length limits, and direct reference routing. The prototype
  entrypoint is 104 lines; application guides are 14–17 lines each.
- Passed Markdown/local-link and whitespace checks. Built and checked the documentation site.
- Built the CLI and installed into a scratch project with Claude/Codex adapters; verified the prototype skill,
  template, and all four references were copied unchanged and adapters route to the store.
- Passed lint, race-enabled unit tests, and vulnerability scanning. Updated existing installation/removal
  snapshots to account for the four added reference files; no CLI implementation changed.
- Integration checks have one remaining failure: `TestScript/upgrade`, `testdata/upgrade.txtar:7`, exits 4
  instead of 2 because its legacy `.claude/skills/go/SKILL.md` conflicts with its manifest hash. Reproduced the
  same failure at untouched baseline `21d94a79`; all other integration scenarios passed.

Instruction walkthroughs check consistency; they do not establish that an agent always follows the policy
or that generated prototypes work. The change adds no CLI-level approval enforcement and does not claim a
live agent workflow trial. The standalone `skills-ref` CLI was unavailable; metadata validation used the
bundled validator and explicit specification checks instead.
