# Route an extension choice

Framework version: {{ .Version }}

In a task, return to its owning orchestrator. Otherwise check `solomonar help`; if it lists `extension`,
show `solomonar extension list` and propose `solomonar extension add|remove <name>` for the user to run.
If unavailable, report that extension management is unsupported and stop. Never substitute manual registration.
These CLI commands own the adoption decision; a different project decision follows `workflow/decisions.md`.
