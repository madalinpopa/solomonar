# <Capability name>

<!-- Framework version: 0.2.0. English. File name = capability ID (kebab-case). Delete empty optional sections
and all hints; no placeholders. -->

<One or two lines: who uses this area and for what.>

## Rules
<!-- Label: Observed | Agreed.
Observed source: `<full SHA>:<path>`; append `, flow <flow-id>` for a prototype.
Agreed source: `user YYYY-MM-DD` or an approved decision ID. -->

- <capability>/R-1 · **<label>** · <observable rule> · <source>

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
<!-- Optional. Keep only sources used. -->

- Prototype: `<full SHA>:prototype/<entry file>, flow <flow-id>`
- Code: `<full SHA>:<path>`
- Decisions: <approved decision ID>
- Related specs: `docs/specs/<other>.md`, `docs/ui.md`
