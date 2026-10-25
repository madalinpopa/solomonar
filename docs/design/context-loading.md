# Request-scoped framework context

Status: Approved and implemented. Source verification and measured results:
[context-load verification](../research/context-load-verification.md).

## Outcome and scope

Reduce framework instructions loaded before each request can proceed. Preserve authority, approvals,
ownership, verification gates, and recovery behavior.

- Scope: portable framework sources, instruction templates, loading references, and generated-adapter compatibility.
- Deliverable: request/stage-specific loading contracts with measurable token budgets.
- Excluded: application code, new lifecycle stages, provider integrations, registry schema changes, and weaker reviews.
- Measurement and source inventory: [context-load baseline](../research/context-load-baseline.md).
- Current authority: [framework sources](../../assets/framework/rules.md), rather than the historical
  [v0.1 design](framework-v0.1.md).

## Findings

Baseline counts below use `o200k_base` on normalized installed text before implementation. They exclude runtime instructions, discovery
metadata, wrappers, task documents, code, command output, and conversation history.

| Priority | Finding | Evidence | Proposed change |
| --- | --- | --- | --- |
| 1 | Orchestration loads future stages immediately: 16,791 tokens through the entry skill, rising to 19,955 with work-spec. | [Orchestrator load declaration](../../assets/framework/agents/orchestrator.md); `rules.md` = 6,959, orchestrator = 4,534. | Load shared invariants plus the current stage and concrete dependencies. |
| 1 | Every delegated role loads all 2,765 report tokens, including correction and PR formats. | [Global loading rule](../../assets/templates/AGENTS.md); [report sections 7–8](../../assets/framework/report.md). | Keep the common report contract; load correction and PR procedures by consumer. |
| 2 | Normal implementation loads correction and integration procedures before either applies. | [Implementer sections 6–7](../../assets/framework/agents/implementer.md) = 636 tokens. | Route normal, correction, and integration modes separately. |
| 2 | Request routing is maintained in two places; the entry skill also carries overview and takeover procedures. | [AGENTS request routes](../../assets/templates/AGENTS.md); [entry skill](../../assets/framework/skills/solomonar/SKILL.md) = 1,274 tokens. | Keep one canonical route table and defer overview/takeover detail. |
| 2 | Spec discovery and in-task procedures share a 2,307-token entry file. | [Spec modes](../../assets/framework/skills/spec/SKILL.md). | Extract mode procedures; keep shared approval and conflict checks reachable from both. |
| Preserve | Go rules already load by impact; extension authoring already loads by stage. | [Go routing](../../assets/extensions/language/go/SKILL.md); [authoring routing](../../assets/extensions/workflow/extension-authoring/SKILL.md). | Reuse these patterns; retain their trigger and missing-input checks. |

The four core skills' discovery fields total only 281 tokens before host formatting. Shortening metadata has
lower priority than fixing the required load chains. Installed files are available context, not automatically
loaded context; the complete asset inventory must not be presented as an every-call charge.

## Requirements: what and why

The user approved these loading changes on 2026-10-25. Existing workflow guarantees remain mandatory.

| ID | Required behavior | Reason |
| --- | --- | --- |
| CL-1 | Persistent project instructions contain project-wide invariants, ownership boundaries, and concise routes. Read-only questions/status do not activate task execution or load its procedures. | Keep routine requests small and preserve explicit resumption. |
| CL-2 | Select the owning route from explicit intent and current task context. Load only its required dependency closure; a citation or index entry alone does not load its target. | Prevent broad traversal of unrelated instructions. |
| CL-3 | Orchestrators load shared gates and the current stage; roles/skills load shared contracts and their current mode. Load another module before performing the action it governs. | Defer procedures until they can affect a decision. |
| CL-4 | Preserve existing rule semantics, scoped approvals, result values, evidence fields, review independence, and recovery guarantees. Every delegated role remains safe without orchestration history. | Context reduction must not remove safeguards. |
| CL-5 | Select extensions using packet assignments or existing applicability triggers. Load matching topic rules; load user-invoked authoring procedures only on explicit invocation. | Installation alone must not pull in domain research or all topic rules. |
| CL-6 | Reuse instructions already present only for the same content/revision and execution. Fresh agents and compacted/resumed contexts must recover their required contracts. | Avoid duplicate reads without relying on missing history. |
| CL-7 | Verification reports distinguish normalized file tokens, newly loaded text, retained framework text, and observed model usage. Unknown usage remains unknown. | Static counts cannot establish actual per-call billing. |
| CL-8 | Missing required modules/inputs retain the owning procedure's failure behavior. Dependency, shared-contract, and security impacts expand context before dependent work proceeds. | A token target never justifies incomplete context or a passing result. |

## Technical design: how

### Persistent entry and routing

- Keep `assets/templates/AGENTS.md` as the canonical request table and source of project-wide invariants.
- Compact wording while preserving mandatory rule numbers and the exact `## Project notes` ownership boundary.
- Make `skills/solomonar/SKILL.md` a small user-only dispatcher referencing that table. Load overview detail
  only for overview requests; task status uses the CLI without taking the orchestrator role.
- Keep project context routes as lookups: capability sections, scoped decisions, relevant contracts, and exact
  verified commands. An absent optional project document does not authorize creating it.
- For mixed requests, load the union of applicable routes as their actions become necessary. Ask only when
  ambiguity changes ownership, scope, or the permitted action.

### Orchestration by stage

- Keep `rules.md` responsible for authority and cross-stage gates, with a short stage/trigger index.
- Extract topic procedures under `assets/framework/workflow/`: setup, decisions, implementation, reviews,
  integration/closure, and recovery. Combine policy and its corresponding procedural detail in one owner.
- Keep `agents/orchestrator.md` focused on role boundaries, session entry, stage dispatch, and delegation inputs.
  Move stage-specific packet formats beside their procedures; load only the packet being prepared.
- Load recovery on a new resumed execution, handoff, pause, cancellation, or observed drift; load decisions
  when a decision/conflict arises. Their prohibitions and activation triggers remain in the shared contract.
- Retain existing numbered headings as short redirects where needed for references. Redirects identify the
  required module; they must not imply that its full procedure was already loaded.

### Reports and role modes

- Keep report sections 1–6 as the common format, result/evidence gates, findings, and decision-request contract.
- Move section 7's format to `reports/corrections.md`, loaded for consolidation and correction packets.
- Move section 8's format to `reports/pr-verification.md`, loaded by the orchestrator during integration.
- Keep short redirects under the original headings and update role/packet references together.
- Extract implementer correction/integration procedures behind mode triggers. Keep preflight, approval
  triggers, checkpoint discipline, and handoff gates visible in every mode.
- Extract spec discovery/in-task procedures behind a common contract. Move reused prototype approval and
  conflict checks into that contract: in-task mode currently depends on discovery steps 2 and 6.
- Leave work-spec's AC rules, readiness, and plan approval together initially; they are needed for its normal request.

### Packaging and context lifetime

- Change portable sources, then regenerate adapters through the existing CLI; never hand-edit generated stubs.
- [CoreWants](../../internal/project/core.go) already copies framework files recursively. New Markdown
  references need no new loader, provider client, or registry field.
- [RuntimeWants](../../internal/project/runtime.go) already generates thin stubs that load portable sources.
  Preserve frontmatter names, descriptions, explicit-only entry invocation, and model configuration.
- Treat files read earlier in a session as retained until the host compacts or omits them. Stage changes do
  not unload prior text; splitting primarily saves early calls and requests that never need later stages.
- Preserve the top-level orchestrator session and existing recovery rules. Do not restart sessions per stage
  merely to satisfy a budget; give delegated agents bounded packets without full history where supported.
- Use bounded searches/reads and existing output filtering. A filtered or truncated required source is incomplete
  input until its relevant content is recovered. Avoid repeated full status listings and pasted reports.

## Request loading contract and budgets

Budgets cover the complete currently required framework text, including applicable stage/mode references.
They exclude project-specific inputs and runtime metadata as in the baseline. They are validation targets;
they are not measured savings or permission to omit required content.

| Request in a fresh relevant context | Required framework scope | Baseline tokens | Target maximum |
| --- | --- | ---: | ---: |
| Project/process question, direct | Project instructions; relevant source excerpts only if needed | 1,259 before question-specific excerpts | 800 before those excerpts |
| Status through entry skill | Instructions, dispatcher, requested status/overview procedure | 2,533 | 1,600 |
| Existing CLI prototype | Instructions, prototype contract, CLI reference | 3,263 | 2,800 |
| Existing capability spec, discovery | Instructions, spec common contract and discovery mode | 3,566 | 3,000 |
| Start task through entry skill | Instructions, dispatcher, orchestration common contract and setup | 16,791 | 8,000 |
| Prepare existing work spec | Previous applicable context plus specification procedure and work-spec | 19,955 | 10,000 |
| Normal implementation | Instructions, implementer common/normal mode, common report | 7,107 | 5,500 |
| Reviewer, before extensions | Instructions, reviewer contract, assigned specialty, common report | 6,537–7,063 | 6,000 |

- New documents add their creation template; mixed interfaces add their matching references.
- Project decisions load the decisions procedure; status-only requests do not load it merely to list questions.
- Normal Go test work currently adds 869 tokens for the Go entry and testing topic; all Go Markdown rules add
  1,770. Preserve conditional loading instead of replacing it with a single language bundle.
- Measure extension registration text and discovery metadata separately; their exact host contribution is runtime-dependent.

## Implementation order

| Task | Change | Dependency | Verification |
| --- | --- | --- | --- |
| T-1 | Freeze baseline and enumerate required modules for each request/stage/mode, separating loads from citations. | None; baseline recorded in this proposal. | Reproduce [inventory and route sums](../research/context-load-baseline.md); review dependencies against current source. |
| T-2 | Compact global instructions and entry dispatcher; establish the single route owner. | T-1 | Trace question, overview, task status, new task, and explicit resume. Preserve user-only invocation and Project notes. |
| T-3 | Extract orchestration topics, report consumers, and implementer modes; update references atomically. | T-2 | Check packaging, rendered versions, all reference targets, required closure, and affected existing project/runtime/upgrade tests. |
| T-4 | Separate spec modes while retaining their shared approval/conflict checks. | T-3 | Trace approved handoff, stale approval, prototype/spec conflict, in-task changes, and new-document creation. |
| T-5 | Compare budget totals and run fresh-session workflow walkthroughs through supported adapters. | T-4 | Record loaded files, role behavior, and available usage for the same scenarios before/after. Report unsupported live-runtime checks separately. |

Implementation follows the user's approval; this document grants no authority to create commits or publish changes.

## Acceptance and verification

- CL-1–3: question/status traces load no task-execution procedure; stage/mode traces contain every governing
  module and no unrelated procedural module. Direct and entry-skill invocation reach the same owner.
- CL-4: walk through missing plan approval, pending dependency approval, wrong ownership, dirty reviewer
  worktree, failed required check, candidate drift, review reuse, and resumed work. Outcomes match current rules.
- CL-5: a Go test-only change selects testing; a concurrency change adds concurrency and other affected topics.
  Ordinary Go work loads neither extension-authoring nor its research.
- CL-6: a fresh delegated agent works from its packet and required sources; a resumed orchestrator verifies
  durable state without previous conversation. Repeated requests do not reread unchanged required files unnecessarily.
- CL-7: rerun tokenization with the same normalization and encodings; compare identical routes and report
  pre/post totals, actual usage availability, and any compaction. Check every proposed budget.
- CL-8: a missing required reference is reported under the existing failure contract; a discovered shared
  contract expands context. Neither case is suppressed to meet a token budget.
- Inspect final Markdown/diff, relative links, retained heading references, adapter file sets, and Project notes
  preservation. Live traces supplement packaging tests; file-tree assertions alone do not prove selective loading.

## Conflicts, assumptions, and open questions

- Coordinated load declarations were updated with the extraction: delegated roles retain the common report
  contract; orchestrators defer it until delegation/report intake and load workflow procedures by stage/trigger.
- Historical design and phase-1 validation notes retain older load chains. Current contributor/site guidance
  and affected packaging fixtures now describe the implemented structure.
- The baseline includes the earlier uncommitted framework edits; implementation preserved them. Remeasure
  the full required file sets when later framework changes affect these counts.
- Assumption: reference-token counts are sufficient to prioritize structural changes. They do not establish
  model-specific usage, billing, or equal tokenization across Claude, Codex, and Copilot.
- No blocking implementation-design question remains. Exact per-call usage is unmeasured: record the target
  runtime/model/version and use its available call-level usage during T-5; do not infer it from asset sizes.
