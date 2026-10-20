# <Capability name>

<!-- Framework version: 0.1.0. English. File name = capability ID (kebab-case). Delete empty optional sections
and all hints; no placeholders. -->

<One or two lines: who uses this area and for what.>

## Rules
<!-- One line per rule: <capability>/R-<n> · **Observed** | **Agreed** · <observable statement> · <source>.
Observed → `<full SHA>:prototype/<file> — flow <flow-id>`. Agreed → `user YYYY-MM-DD` or `W-001/D-n`.
Observable behavior only; no storage, framework, or simulation detail. IDs are stable and never reused.
Illustrative only, delete:
- orders/R-1 · **Agreed** · A customer can cancel an order only while its status is "Pending". · user 2026-10-20
- orders/R-2 · **Observed** · A placed order is still listed after the page is reloaded. · <full SHA>:prototype/orders.html — flow checkout
- orders/Q-1 (Open, under "Open questions") · Where are placed orders stored? · blocks: orders/R-2 · needs decision: yes -->

- <capability>/R-1 · **Agreed** · <observable rule> · <source>

## Flows and UI behavior
<!-- Optional. Per flow ID: steps and the expected observable result, including failure, empty, and permission
states. Cite rule IDs; reference shared rules in `docs/ui.md` (e.g. `ui/R-2`) instead of copying them. -->

### <flow-id>
1. <step> → <observable result>
- <Empty | failure | permission> state: <trigger> → <observable result>

## Open questions
<!-- Optional. Blocking first. needs decision: yes = needs an architecture or dependency decision. -->

- <capability>/Q-1 · <question> · blocks: <rule/flow IDs or "none"> · needs decision: <yes|no>

## References
<!-- Optional. -->

- Prototype: `<full SHA>:prototype/<entry file> — flow <flow-id>`
- Decisions: <W-001/D-n>. Related specs: `docs/specs/<other>.md`, `docs/ui.md`
