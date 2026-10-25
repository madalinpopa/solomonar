# Project instructions

Framework version: {{ .FrameworkVersion }}. Framework files: `docs/solomonar/`.

Solomonar coordinates specification, implementation, review, and verified merge. Roles define responsibilities;
rules define task policy; skills define procedures. The CLI records state; the runtime executes work.

Write framework artifacts, specs, reports, commit messages, and PR text in English.

## Scope and loading

Applies project-wide. Explicit user instructions override repository guidance; nearer `AGENTS.md` files take
precedence in their subtrees. Roles/skills cannot override approved requirements.

Load only your role/skill, its required files, assigned spec, relevant references, and affected code/tests.
Expand for dependencies, shared contracts, or security rules. Indexes do not load targets. Optional project
documents are created by their owning procedure, not merely to satisfy a route.

## Request routes

Bare framework paths below are relative to `docs/solomonar/`. Load the routed file before acting.

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
user's outcome. Every role loads `report.md`; reviewers add `reviewer.md`; only orchestrators and the
project-decision route load `rules.md`.
Load packet-named extension skills; without a packet, use Extensions below.

## Extensions

Installed with `solomonar`. Load every installed extension skill whose "applies when" matches your change, before
writing or reviewing code, and cite its rules as `<name>/C-n`. The orchestrator names the matching extensions in
each packet.

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
   Approved prototypes define represented behavior; **Agreed** specs express it for implementation. Code and
   unapproved observations establish no requirements. Prototype choices never approve production architecture.
2. New architecture/dependency changes require user approval; investigate first, then follow your role's
   decision procedure. Register decisions and open questions at the level they affect, project or task
   (`rules.md` section 3). Scoped approvals persist; silence approves nothing.
3. Only `solomonar task` writes the registry: orchestrators own their task records; the user owns `task plan`.
   Never directly edit, commit, or check out the registry files on `workflow-state`.
4. Run `solomonar task register` before creating the feature worktree; setup: `rules.md` section 4.
5. Edit only your role/skill's allowed files and assigned worktree. Reviewers are read-only on one stable revision.
6. Orchestrators delegate implementation/reviews and consolidate findings; delegated roles report only to them.
7. Blockers, missing required reviews, or failed/unperformed required checks prevent advancement.
8. Mark `done` only after confirmed merge into main.
9. New or changed behavior must follow prototype → explicit human review/approval → specs → production code,
   including when a prototype already exists. Resolve prototype/spec conflicts in the prototype first; never
   implement unrepresented or unapproved behavior. Reuse existing prototype approval when restoring represented
   behavior; technical work with no behavioral effect requires no new prototype. Commit the final task spec separately.
10. Closure and verifiable Git references: `rules.md` section 9.
11. Never commit or report secrets, credentials, or real personal data.
12. Commit messages: `<type>(<scope>): <summary>`; role/skill defines type and scope.
13. Tests/comments cite `<capability>/R-n`, never task/AC IDs removed at closure.
14. Ask the user one question at a time through the harness's ask-user tool when available, offering known
    options with the recommended answer first; otherwise ask in chat. Wait for each answer.

Never hand-edit `.solomonar/manifest.json`, generated runtime stubs, or generated product files; change their
source through the owning procedure. Preserve unrelated edits. Project-specific guidance belongs under Project notes.

## Project notes

<!-- Project-owned instructions; preserved by framework upgrades. Keep this heading unchanged. -->
