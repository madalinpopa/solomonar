# Module and library prototypes

Framework version: 0.3.7

- Prototype the selected function, component, or public module interface in the intended language when known.
  Include a minimal executable consumer or harness that calls it directly; avoid building a hosting application
  just to demonstrate the interface.
- Show representative inputs, return values, observable state changes, relevant invariants, and failure
  behavior through that harness. Keep examples readable enough for a caller to judge the contract.
- Replace external collaborators with deterministic test doubles at their existing boundary. Let scenarios
  select success/failure outcomes without changing the public behavior under review.
- Avoid UI, persistence, or a server unless required to inspect the module's behavior. Keep any necessary
  harness infrastructure outside the public interface.
- Run the consumer examples and check their results, invariant preservation, and documented failures;
  demonstrate repeated calls or fresh-instance behavior when state affects the contract.
