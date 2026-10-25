# Implementation coordination

Framework version: {{ .Version }}

Orchestrator procedure. Delegates implementer work; never implement here. Load `report.md` before delegation.

## Checkpoints

- The work spec's `## Implementation plan` lists steps `- S-<n>: <type>(<ID>)[!]: <summary>`, each the intended
  commit subject with enough detail to delimit the work. Its `Status` line is `In review` until the user approves
  the plan, then `Approved`. Only the orchestrator edits the plan; it is the approved plan, not an execution log.
- The implementer commits each completed step as its checkpoint: the planned subject, a short body (progress,
  checks and results, observations needed to continue), and the trailer `Step: <ID>/S-<n>`. Progress lives in
  these commits only; never duplicate it in the work spec, the registry, or another file.
- Step IDs are never reused. Changing an approved step's scope gives it a new ID and returns `Status` to
  `In review` for a new approval. Review corrections within the approved scope are appended as new steps without
  a new approval. A step that becomes unnecessary or is satisfied by verification alone gets an indented
  `Outcome: dropped — <reason>` or `Outcome: verified — <reason>`; never create an empty commit for it.
- A checkpoint preserves only committed content. A commit is not proof of correctness: `task status` reconciles
  steps with commits (`plan` section) and reviews still verify the change.
- Leaving `implementation` for review or integration needs the clean branch tip as candidate, every step
  committed or closed by an outcome, and no reconciliation discrepancy; the CLI enforces it. Review the cumulative
  change on that one candidate after the complete implementation or correction batch, never per checkpoint.

## Stage procedure

### `implementation`
- **Entry:** spec ready, a consolidated correction list, or an approved decision to apply. **Delegate:** `implementer`.
- **Do:** send the implementer or correction-round packet (Delegation packet below) with the remaining steps from
  `task status`; the implementer executes them in order and commits one checkpoint per step
  (`agents/implementer.md` section 5). Wait for the report; validate its header (`agents/orchestrator.md` section 6); route per
  `rules.md` section 2. A reported deviation from the plan is a plan amendment you decide: append or close
  steps, or change scope with a new approval (`rules.md` section 5 Checkpoints).
- **State:** maintain delegation state per `agents/orchestrator.md` section 3. On `READY_FOR_REVIEW`, verify in the assigned worktree that
  `feature/<ID>` is checked out, `git rev-parse HEAD` equals the handoff's full SHA, and
  `git status --porcelain` is empty, and `task status` shows every step committed or closed with no
  discrepancy. Triage tier, required specialties, and reuse reasons per `rules.md`
  section 6; record `--revision <SHA> --tier <n>` with
  the move to `specialized_review`, or `integration` for tier 0, and update `--next`. Specialties and reasons
  stay in the session and PR Verification section. **Exit:** verified handoff and successful stage update.
- **Failure:** `NEEDS_DECISION` → `rules.md` section 3 procedure, send *Decision request*. `INCOMPLETE` → resolve the gap,
  re-delegate. Two consecutive rounds with the same blocker → loop stop (`rules.md` section 5), send *Blocked*.

## Delegation packet

```text
Task: W-001, <title>           Role: implementer; follow docs/solomonar/agents/implementer.md
Worktree: <path> on feature/W-001; modify nothing outside it
Objective: <outcome to deliver>
Work spec: <worktree-relative path>    References: <spec sections, AC IDs, P/D-n or W-001/D-n, contracts>
Plan steps: <remaining S-n IDs, in order>; partial work to inspect: <dirty paths or none>
Area of responsibility: <packages, files, flows>
Extensions: <applicable installed extensions; or none>
Baseline: <full SHA>; known failures: <refs or none>; checks available: <command; result; SHA>
Constraints: ports <…>; test data <…>; commands per docs/README.md
Output: report per docs/solomonar/report.md; Revision = the committed candidate you hand off
```

Packet requirements: `agents/orchestrator.md` section 4.

## Correction packet

Load `reports/corrections.md` before preparing this packet.

```text
Task: W-001, <title>           Role: implementer; follow docs/solomonar/agents/implementer.md
Mode: correction
Worktree: <path> on feature/W-001. Objective: resolve exactly the list below; nothing else
Plan steps: <the correction steps S-n appended for this round>
Work spec / References / Area of responsibility / Extensions: <as in the implementer packet, current for this round>
Baseline / Known failures / Checks available / Constraints: <as in the implementer packet, current for this round>
<consolidated correction list per report.md section 7, verbatim>
Output: READY_FOR_REVIEW lists each addressed W-001/F-n with the check that shows the correction
```
