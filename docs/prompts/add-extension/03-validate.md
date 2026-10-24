# Prompt 3: Validate and refine the extension

Use after [Prompt 2](02-author.md), preferably in a fresh chat. The canonical
procedure is the [extension-authoring skill](../../../assets/extensions/workflow/extension-authoring/SKILL.md),
with its [validation stage](../../../assets/extensions/workflow/extension-authoring/references/validate.md).

```text
Use the extension-authoring skill to validate and refine this candidate:

Extension name and candidate paths: <name and paths>
Research record: <path from Prompt 1>
Additional constraints: <constraints or none>

If the skill is not installed in this Solomonar source checkout, load
assets/extensions/workflow/extension-authoring/SKILL.md directly and follow its
validation route. Fix established defects within the applicable role's allowed
scope, rerun affected checks, and update the same record. Assess source support,
packaging, and live behavior separately. Leave unavailable checks explicit;
do not register or adopt the extension merely to claim completion.
```
