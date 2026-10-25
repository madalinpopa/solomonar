# Spec in-task mode

Framework version: {{ .Version }}

Load with `skills/spec/SKILL.md`; its boundaries, IDs, approval/conflict checks, and validation apply.

Only the owning orchestrator uses this mode. Confirm the registered task/owner, `feature/<ID>` branch,
and assigned worktree with `solomonar task status <ID>` and `SKILL.md` section 2’s inspection. Edit only the
listed approved decisions or user-confirmed Spec changes; never update the work spec. This mode never commits.

1. Read the listed decisions in `docs/decisions.md` (only status `approved`; a `pending` entry approves nothing)
   or the work spec's "Spec changes" line, and the spec rules they affect. Apply `SKILL.md` section 4's prototype
   approval check to every behavioral change; without that handoff, report it to the orchestrator and leave it unchanged.
2. Apply each authorized change in the rule's primary document, using `SKILL.md` section 3's IDs, labels, and sources.
   Assign final IDs to `<capability>/new` entries; delete the **Open** items the change resolves. Use `SKILL.md` section 4's conflict procedure for conflicts; a task decision does not authorize resolving one solely in the specification.
3. Leave everything else unchanged, including **Observed** rules the input does not cover.
4. An input that is unclear for a rule, lacks a decision or user source, or contradicts another **Agreed** rule:
   change nothing for it; report it.
5. Apply `SKILL.md` section 6's document-creation and validation steps. Report changed files, rule IDs, `new` and split-ID
   mappings, unresolved inputs, and required reference fixes outside scope to the orchestrator. It updates the
   work spec and commits; this mode never commits.
