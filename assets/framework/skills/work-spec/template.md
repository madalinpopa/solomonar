# <ID>: <Change title>
Status: In review
Type: <feature | bug fix | refactor | technical task>

<!-- Framework version: 0.3.1. English. Delete empty optional sections, unused optional bullets, and all hints.
No execution state or readiness verdicts. Status stays "In review" until the user approves the implementation
plan; the orchestrator then sets "Approved". -->

## Outcome and scope

<Authorized outcome, included behavior, and relevant boundaries.>

## Context

- <Current capability spec and relevant sections.>
- <Affected contracts/code, if they exist.>
- Prototype: `<full SHA>:prototype/<entry file>, flow <flow-id>`
- Spec changes: <rule ID or capability/new> → <target statement or label>;
  source: <user YYYY-MM-DD or approved decision ID>; ACs: <AC IDs>.

## Acceptance criteria
<!-- Rules: <capability>/R-n or <capability>/new; none for purely technical outcomes.
Observed rule confirmation: (<rule ID>, confirmed user YYYY-MM-DD). -->

- AC-1: <condition> → <observable result> (<rule IDs or none>).

## Constraints and checks
<!-- Approved decisions: P/D-n or <ID>/D-n, within their recorded scope. -->

- <Applicable global rule IDs and approved project/task decision IDs>.
- Check: <exact command or named scenario> · ACs: <AC IDs> · limits: <none or verification limits>.

## Open questions
<!-- Optional. Blocking first. Reuse canonical question IDs; otherwise use <ID>/Q-n. -->

- <question ID> · <question> · <blocking | non-blocking> · needs decision: <yes|no>

## Implementation plan
<!-- One step per intended checkpoint commit, in order. The subject is the exact commit subject. Never reuse
a step ID; close an unnecessary step with an indented "Outcome: dropped — <reason>" or "Outcome: verified — <reason>". -->

- S-1: <type>(<ID>): <summary>
  <What the step changes, its ACs, and its checks.>
