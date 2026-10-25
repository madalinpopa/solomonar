# Project instructions

Framework version: {{ .FrameworkVersion }}. Framework files: `docs/solomonar/`; bare framework paths below are
relative to it.

Write framework artifacts, specs, reports, commit messages, and PR text in English.

## Scope and loading

Applies project-wide. Explicit user instructions override repository guidance; nearer `AGENTS.md` files take
precedence in their subtrees. Roles/skills cannot override approved requirements.

Load only your role/skill, its required files, assigned spec, relevant references, and affected code/tests.
Every role loads `report.md`; reviewers add `reviewer.md`; only orchestrators and the project-decision route
load `rules.md`. Expand for dependencies, shared contracts, or security rules. Indexes do not load targets.
Optional project documents are created by their owning procedure, not merely to satisfy a route.

## Request routes

Load the routed file before acting.

| When | Load |
|---|---|
| User entry point | `skills/solomonar/SKILL.md` (user-only invocation) |
| Prototype a CLI, web, module/library, or service flow; change product behavior | `skills/prototype/SKILL.md` |
| Record product behavior / prototype handoff | `skills/spec/SKILL.md` |
| Start/resume a task | `agents/orchestrator.md`; top-level only, one task per session |
| Specify a registered task | `skills/work-spec/SKILL.md`; orchestrator only |
| Implement an assigned packet | `agents/implementer.md` |
| Review an assigned packet | `agents/<role>.md`: `security-reviewer`, `correctness-reviewer`, `spec-reviewer`, or `adversarial-reviewer` |

Routing exceptions: `skills/solomonar/SKILL.md` section 2. Without a work spec, start with `docs/README.md` and the
user's outcome.

## Extensions

Installed with `solomonar`. Before writing or reviewing code, load the extension skills your packet names; without
a packet, every installed one whose "applies when" matches your change. Cite their rules as `<name>/C-n`.

{{ .Extensions }}

## Context routes

| Need | Source |
|---|---|
| Product behavior | `docs/specs/<capability>.md` |
| Task requirements | Assigned `docs/work/<ID>-<slug>.md` |
| Structure, contracts, dependencies | `docs/architecture.md`; `docs/stack.md` and manifests |
| UI | `docs/ui.md`, capability spec, referenced `prototype/README.md` flow |
| Approvals | Scoped `approved` entries in `docs/decisions.md` |
| Run, build, test, lint | Exact verified commands in `docs/README.md` → Commands |
| Registry / actual task state | `solomonar task show [<ID>]` / `solomonar task status <ID>` |
| Historical spec | Compact task record, then `git show "<spec_ref>"` for a concrete question |

Discover missing commands from manifests/scripts; verify before recording, otherwise keep "not established".
Use configured formatters/linters. Check CLI availability with `solomonar help`; do not run planned commands.

## Mandatory rules

Cite a rule as `AGENTS.md rule n`.

1. Scoped user decisions > approved global rules > work spec > preferences. Escalate contradictions.
   **Agreed** specs express approved prototype behavior; code, unapproved observations, and prototype choices
   establish no requirements or production architecture.
2. New architecture/dependency changes require user approval; investigate first, then follow your role's
   decision procedure. Register decisions and open questions at the level they affect, project or task
   (`rules.md` section 3). Scoped approvals persist; silence approves nothing.
3. Only `solomonar task` writes the registry: orchestrators own their task records; the user owns `task plan`.
   Never directly edit, commit, or check out the registry files on `workflow-state`.
4. Edit only your role/skill's allowed files and assigned worktree. Reviewers are read-only on one stable revision.
5. Orchestrators delegate implementation/reviews and consolidate findings; delegated roles report only to them.
6. Blockers, missing required reviews, or failed/unperformed required checks prevent advancement.
7. New or changed behavior must follow prototype → explicit human review/approval → specs → production code,
   including when a prototype already exists. Resolve prototype/spec conflicts in the prototype first; never
   implement unrepresented or unapproved behavior. Reuse existing prototype approval when restoring represented
   behavior; technical work with no behavioral effect requires no new prototype. Commit the final task spec separately.
8. Never commit or report secrets, credentials, or real personal data.
9. Commit messages follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)
   (`<type>(<scope>)[!]: <summary>`); role/skill defines type and scope. Never credit an AI tool, agent, or model
   in commits or PR text (no `Co-authored-by` trailer or "Generated with" line); review evidence naming a model
   (`report.md` section 8) is not attribution.
10. Tests/comments cite `<capability>/R-n`, never task/AC IDs removed at closure.
11. Ask the user one question at a time through the harness's ask-user tool when available, offering known
    options with the recommended answer first; otherwise ask in chat. Wait for each answer.

Never hand-edit `.solomonar/manifest.json`, generated runtime stubs, or generated product files; change their
source through the owning procedure. Preserve unrelated edits. Project-specific guidance belongs under Project notes.

## Project notes

<!-- Project-owned instructions; preserved by framework upgrades. Keep this heading unchanged. -->
