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

- `docs/solomonar/`: framework rules, roles, and skills
- `prototype/README.md`: prototype flows (when it exists)

## Commands

<!-- The implementer discovers commands from the repository and records them once verified. Never guess;
keep "not established" until then. Unknown commands do not imply an absent application. -->

| Purpose | Command |
|---|---|
| Run | not established |
| Build | not established |
| Test | not established |
| Lint | not established |

Next useful action: use the `solomonar` skill with your selected outcome. For product discovery it routes to
`prototype`; an existing brief or a CLI/API task can start directly. In Claude Code use `/solomonar`; in Codex
CLI/IDE use `$solomonar`; the portable prompt is "Use the solomonar skill".
