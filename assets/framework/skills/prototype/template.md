# Prototype

<!-- Framework version: 0.3.7. Fill in English. Delete empty optional sections and all hint comments; never
leave placeholders. -->

Objective: <one line, or "see `docs/README.md`">

## Run, inspect, and reset

<!-- One row per independently runnable part, including a single function/component. Group new parts by type
under cli/, web/, module/, or service/; preserve existing runnable layouts. Create only needed directories.
Give exact commands and working directories, runtime/version needs, and web URLs where applicable.
Test/inspect points to executable checks or numbered scenarios with expected success/failure results in Flows. -->

| Part / type | Location and runtime | Run | Test / inspect | Reset |
|---|---|---|---|---|
| <part / CLI, web, module, or service> | <path relative to prototype/; runtime requirements> | <command; working directory> | <check command, consumer, requests, or browser scenarios> | <command/steps restoring seeds, or stateless> |

## Flows
<!-- ID: stable kebab-case, never renamed or reused once committed. Status: draft | ready for spec | superseded. -->

Entry-file paths are relative to `prototype/`.

| ID | Entry file | Steps | Status | Approval | Notes |
|---|---|---|---|---|---|
| <flow-id> | <entry file> | 1. <input/action> → <output/state>; 2. … | draft | none | <relevant failure/boundary/alternate triggers; replacement ID and approved revision if superseded> |

<!-- Record ready for spec only after explicit approval: user YYYY-MM-DD, covering the listed scenarios.
For longer flows, put numbered scenarios below the table: inputs, outputs/state, validation/constraints,
relevant failures/edges, persistence and integration outcomes. Mark affected approvals stale on behavioral edits. -->

## Simulated behavior
<!-- Every external substitute, matching SIMULATED comments. Include limitations affecting interpretation. -->

- <what>: <local substitute> (`<file>`) · limits: <what is not demonstrated> · flows: <IDs>

## Prototype-only execution choices
<!-- Optional; delete when none. Justify dependencies/setup beyond the runtime; not production approval. -->

- <dependency/version or setup>: <behavioral fidelity or execution need; exact setup command if any>

## Intentional changes
<!-- date (YYYY-MM-DD) · flow ID · change · reason (user decision or observed problem). Not defect fixes. -->

- <YYYY-MM-DD> · <flow-id> · <change> · <reason>

## Open questions
<!-- Use flow IDs, or none when no flow applies. -->

- <question> · flows: <IDs or none> · blocking: yes|no
