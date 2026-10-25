# Standalone service and API prototypes

Framework version: 0.3.5

- Expose only endpoints needed for agreed use cases, using a runtime's built-in HTTP server when sufficient
  (for example Node.js `node:http`). Keep external services and persistence in memory or local JSON files.
- Follow HTTP semantics for methods and status codes (RFC 9110): keep safe methods read-only in intent,
  preserve idempotent method semantics, and return status codes and media types matching the result.
  Do not require REST naming, versioning, or a uniform error schema unless part of the agreed interface.
- Demonstrate request validation and representative success/failure responses with observable state effects.
  Keep simulated collaborator failures selectable locally and separate from the public API under review.
- Provide executable `curl` or similarly common request examples, including headers/payloads and expected
  status/body results. Cover state across requests and after restart when persistence is part of the contract.
- Run on a documented loopback address and port. Avoid deployment, distributed systems, real infrastructure,
  or auxiliary services that do not change the behavior being demonstrated.
