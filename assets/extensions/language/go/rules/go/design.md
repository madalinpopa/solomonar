# Go design

- go/C-17 (should) — Rule: Choose names that explain their purpose at the call
  site, accounting for package and receiver context.
- go/C-22 (should) — Rule: Introduce interfaces, generic APIs, or helper layers
  only for a current caller, test boundary, or concrete simplification; retain
  small duplication when extraction obscures independent behavior.
- go/C-23 (should) — Rule: Define needed interfaces around the consumer's used
  methods. Exception: Shared protocols may own interfaces; returning an interface
  is appropriate when it is the intended contract.
- go/C-24 (should) — Rule: Export only the API consumers need; keep state private
  when direct mutation would bypass its invariants.
- go/C-25 (must) — Rule: Preserve consumed API signatures and observable contracts,
  including errors and serialized values, unless the task requires their change.
- go/C-26 (must) — Rule: Use pointer receivers when field assignments must update
  the original instance or the receiver must not be copied.
- go/C-27 (should) — Rule: Document new or changed exported APIs and packages,
  including non-obvious ownership, cancellation, and error contracts.
- go/C-28 (should) — Rule: Justify performance-specific complexity with a relevant
  measurement and compare the change against its baseline.
- go/C-29 (should) — Rule: Prefer the standard library or an established project
  dependency when it meets the requirements without substantial custom machinery.
- go/C-30 (should) — Rule: Place behavior in a cohesive package with a meaningful
  API; introduce a package split only for a concrete responsibility or dependency boundary.
