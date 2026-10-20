# {{ .ProjectName }}

## Mission

<!-- User-owned. What the product is for, in one or two lines. -->

## Intended users

<!-- User-owned. Who uses the product and what they need from it. -->

## Current objective

{{ if .Objective }}{{ .Objective }}{{ else }}Not set. Run the prototype skill to define it.{{ end }}

## Documents

<!-- The `spec` skill indexes spec files; the role that creates `docs/architecture.md`, `docs/stack.md`, or
`docs/decisions.md` adds its line. Add a document only when it exists. -->

- `docs/framework/`: framework rules, roles, and skills
- `prototype/README.md`: prototype flows (when it exists)

## Commands

<!-- The implementer records the real commands once known. Never guess; keep "not established" until then. -->

| Purpose | Command |
|---|---|
| Run | not established |
| Build | not established |
| Test | not established |
| Lint | not established |

Next useful action: start the `prototype` skill.
