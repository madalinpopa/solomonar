# Project context routes

Framework version: {{ .Version }}

Load this lookup only when selecting project sources; indexes do not load their targets.

| Need | Source |
|---|---|
| Product behavior | Relevant `docs/specs/<capability>.md` sections |
| Task requirements | Packet's exact `docs/work/<ID>-<slug>.md` or historical `spec_ref` |
| Structure/contracts | Relevant `docs/architecture.md` entries |
| Dependencies | `docs/stack.md` entries and manifests |
| UI | `docs/ui.md`, capability spec, cited `prototype/README.md` flow |
| Approvals | Scoped `approved` entries in `docs/decisions.md` |
| Run/build/test/lint | Verified `docs/README.md` → Commands |
| Stored/actual task state | `solomonar task show [<ID>]` / `solomonar task status <ID>` |
| History | Compact task record, then `git show "<spec_ref>"` for a concrete question |

Discover missing commands from manifests/scripts and verify before recording; otherwise use "not established".
Use configured formatters/linters; check CLI availability with `solomonar help`, never run planned commands.
Without a work spec, start with `docs/README.md` and the outcome. Only the owning procedure creates optional documents.
