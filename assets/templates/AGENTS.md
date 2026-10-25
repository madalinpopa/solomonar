# Project instructions

Framework version: {{ .FrameworkVersion }}. Framework paths: `docs/solomonar/`.
Framework artifacts/commit/PR text: English.

## Scope and loading

User instructions win; nearer `AGENTS.md` wins locally. Roles/skills cannot override requirements.
Load selected role/skill, required spec/references, affected code/tests; expand for contracts/dependencies/security.
Indexes/citations load nothing. Reuse same-revision instructions; recover in fresh/compacted contexts.
Missing required input blocks dependent work.

## Request routes

| Request | Load / action |
|---|---|
| User entry / overview | `skills/solomonar/SKILL.md`; user-only |
| Task ID / task status | `solomonar task status <ID>`; no resumption |
| Discover/change behavior | `skills/prototype/SKILL.md`; orchestrator coordinates in-task |
| Approved prototype → specs | `skills/spec/SKILL.md` |
| New/start/resume/pause/cancel task | `agents/orchestrator.md`; top-level, one task/session |
| Task scope/AC/decision answer | Owning session; takeover requires explicit resumption |
| Task spec | `skills/work-spec/SKILL.md`; registered task's orchestrator only |
| Assigned implementation/review | Packet's role file |
| Project decisions/open answers | Owning task, else `workflow/decisions.md` |
| Adopt/drop stack/practice | Owning task, else `skills/solomonar/references/extensions.md` |
| Project/process question | Context below; no execution role |

Product code uses managed tasks unless the user opts out.

## Extensions

Before code work/review, load packet-named extensions, else matching installed ones.
Cite `<name>/C-n`; load topic rules by impact. User-invoked skills require explicit invocation.

{{ .Extensions }}

## Context routes

Unknown source paths: `context.md`; no work spec: `docs/README.md` and outcome.

## Mandatory rules

Cite `AGENTS.md rule n`.

1. Scoped user decisions > approved global rules > work spec > preferences. Escalate conflicts.
   **Agreed** rules express approved prototype behavior; code/observations/prototype choices approve no architecture/requirements.
2. New architecture/dependencies need prior user approval; investigate and record project/task decisions via
   your role (`rules.md` section 3). Scoped approvals persist; silence approves nothing.
3. Only `solomonar task` writes registry; orchestrators own records, users own `task plan`.
   Never edit/commit/check out `workflow-state` registry files.
4. Edit allowed files/worktree only; review read-only on one stable revision.
5. Orchestrators delegate implementation/reviews, consolidate; delegates report only to them.
6. Blockers, missing reviews, failed/unperformed required checks block advancement.
7. New/changed behavior requires prototype → explicit human approval → specs → production; resolve conflicts in prototype.
   Never implement unrepresented/unapproved behavior. Restore approved behavior using its approval; technical work
   preserving behavior needs no new prototype. Commit final task spec separately.
8. Never commit/report secrets, credentials, or real personal data.
9. Conventional Commits; role defines type/scope. No AI commit/PR attribution; review model evidence allowed.
10. Tests/comments cite `<capability>/R-n`, never removed task/AC IDs.
11. Ask one question via ask-user tool, else chat; await answer; recommended known option first.

Edit owning sources, never `.solomonar/manifest.json` or generated files. Preserve unrelated work.

## Project notes

<!-- Project-owned instructions; preserved by framework upgrades. Keep this heading unchanged. -->
