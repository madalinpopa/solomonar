# W-001: Change title
Type: <feature | bug fix | refactor | technical task>

<!-- Framework version: 0.1.0. English. One change only. Cite rule IDs, decision IDs, and paths; never copy
their content. No execution state. Delete empty optional sections, unused optional bullets, and all hints,
including the example below. -->

## Outcome and scope
<Authorized outcome, included behavior, and relevant boundaries.>

## Context
- <Current capability spec and relevant sections.>
- <Affected contracts/code, if they exist.>
- Prototype: <commit:path and flow, when relevant.>
- Spec changes: <rule IDs added or changed, with target statements and sources; optional.>

## Acceptance criteria
<!-- EARS where it fits: "When <trigger>, the system shall <response>" / "If <condition>, then the system
shall <response>". Cite the rule IDs verified or introduced (`<capability>/R-n` or `<capability>/new`). IDs
stable; a removed AC is never reused. Observed rule: `(orders/R-2, confirmed user YYYY-MM-DD)`. -->
- AC-1: <Observable condition and expected result.>
- AC-2: <Relevant failure or boundary case.>

## Constraints and checks
- <Applicable approved rule or dependency decision reference.>
- <Check for the criteria and any known verification limit.>

## Open questions
<!-- Optional. Blocking first: W-001/Q-n · <question> · blocking | non-blocking · needs decision: yes|no -->
- <Only unresolved choices relevant to this work; mark blocking choices.>

<!-- Illustrative only (headings condensed), delete:
# W-001: Cancel a pending order / Type: feature
Outcome: a customer cancels their own "Pending" order from the order list. Refunds are out of scope.
Context: docs/specs/orders.md (orders/R-1, orders/R-2); internal/orders/; docs/ui.md (ui/R-2)
Prototype: <full SHA>:prototype/orders.html, flow cancel-order
Spec changes: orders/R-2 → **Agreed** (user 2026-10-20);
  orders/new: "A cancelled order shows its cancellation date." (user 2026-10-20)
- AC-1: When a customer cancels a "Pending" order, the system shall show "Cancelled" and the cancellation date
  (orders/R-1, orders/new).
- AC-2: If the order is "Shipped", then the system shall reject the cancellation with an error message and keep
  its status (orders/R-1).
- AC-3: When the page is reloaded after cancelling, the system shall still list the order as "Cancelled"
  (orders/R-2, confirmed user 2026-10-20).
Checks: `go test ./internal/orders/...` (AC-1–AC-3); scenario cancel-order on the running app (AC-1–AC-3).
- W-001/Q-1 · Ask for confirmation before cancelling? · non-blocking · needs decision: no -->
