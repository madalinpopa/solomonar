# Prototype-first workflow design addendum

Recorded 2026-10-25 for the prototype skill revision based on Solomonar 0.3.2. This addendum supersedes the
prototype scope, behavioral authority, and readiness assumptions in [the historical v0.1 design](framework-v0.1.md).
Keep that original note as history. Current executable instructions remain owned by
[`assets/framework/`](../../assets/framework/skills/prototype/SKILL.md) and
[the AGENTS template](../../assets/templates/AGENTS.md); this note explains their integration.

## Design changes

| Earlier assumption | Current design |
| --- | --- |
| A prototype is a static, clickable browser flow. | A prototype can cover a CLI, web interface, function, component, module/library, workflow, or standalone service. Select only the part needed for a behavioral decision. |
| Every prototype opens through `file://`. | Every independent part has exact runtime, run, test/inspection, expected-result, and reset instructions; use a minimal harness or local HTTP server when needed. |
| Prototypes are evidence that can become requirements through a separate spec confirmation. | Explicit approval makes represented prototype behavior authoritative. Specs extract that agreement; direct spec/task/plan approval cannot establish missing prototype behavior. |
| Canonical spec updates can follow a behavior decision directly. | New or changed behavior follows prototype → human review/approval → specifications → production implementation, including during an active task. Resolve prototype/spec conflicts in the prototype first. |

Keep independent parts under `prototype/cli/`, `web/`, `module/`, or `service/` when creating a new layout;
preserve existing runnable layouts. The shared `prototype/README.md` indexes run/check/reset instructions,
flow IDs/statuses, scoped approval sources, simulated collaborators and their limitations. Local deterministic
substitutes keep the behavioral reference usable without production credentials or infrastructure.

An approved prototype is authoritative only for demonstrated behavior. Its libraries, data structures,
persistence mechanisms, and hosting choices do not approve production architecture. Runtime examples in
conditional references illustrate execution options; they do not select a project stack or require an extension.

## Integration and ownership

- Keep shared constraints and handoff in the prototype skill, with four directly routed interface references.
  The [public workflow guide](../../site/src/docs/prototype.md) owns user-facing output and execution examples;
  [the research record](../research/prototype.md) owns sources and validation evidence.
- Keep site explanations under Learn the framework conceptual. Split practical guidance into Prototype workflow
  (discovery through specification handoff) and [Task workflow](../../site/src/docs/use-cases.md) (implementation,
  review, merge, and recovery) under Guides; link to those procedures instead of duplicating them in explanations.
- Preserve stable flow IDs, `draft` / `ready for spec` / `superseded` statuses, and committed SHA:path handoffs.
  Approval is scoped to demonstrated scenarios. Behavioral edits invalidate affected readiness; preserve earlier
  approved revisions for inspection. A defect restoration can reuse an existing approved prototype.
- Ask the user the next necessary question, including each blocking open question before handoff, through the
  harness's ask-user tool when available, with known options and a recommendation; otherwise ask in chat.
- Run prototype work in the primary checkout. Keep canonical extraction in `spec`, task preparation in the
  owning orchestrator's feature worktree through `work-spec`, and production edits with the implementer.
- When an active task needs different behavior, hold dependent work for the approved prototype handoff, bring
  its snapshot into the task branch, update canonical/work specs, then resume. Apply the same gate to correction
  packets; independent authorized work can continue while the changed behavior waits.
- Make missing representation/approval and prototype/spec conflicts stop affected implementation. Have the Spec
  reviewer verify the behavior's prototype and specification references. Keep implementation-plan approval
  separate from behavioral approval; do not reconfirm an approval covering unchanged behavior.

This adds no task status, registry field, CLI command, or mechanical approval detector. Skills and roles enforce
the agreement; the CLI continues to validate recorded task and Git facts. Technical work preserving behavior
does not require a new prototype. Retain untouched legacy spec rules; representation and approval are required
when behavior is introduced or changed.
