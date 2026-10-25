# Task specification

Framework version: {{ .Version }}

Load before specification work, with `skills/work-spec/SKILL.md`; the first numbered step continues setup steps 1–4.
`task` commands mean `solomonar task`; framework paths are relative to `docs/solomonar/`.

5. Run `task move <ID> --as <owner> --status running --stage specification`. It records the task's `base`
   (the commit it branched from), its worktree path relative to the main checkout, and the confirmation
   (`rules.md` section 7). Create or refine the work spec in the worktree with `work-spec`. Commit it separately; record it
   with `task move <ID> --as <owner> --spec <path>`.
6. When the readiness gate owned by `work-spec` passes, apply its "Spec changes" from the approved prototype
   with the `spec` skill (in-task mode), commit them as `docs(<ID>): apply spec changes`, and cite the assigned
   rule IDs in the work spec (`spec(<ID>): refine task specification`).
7. Present the work spec's implementation plan (`rules.md` section 5 Checkpoints) to the user. On explicit approval of that
   exact plan, set `Status: Approved` below the title and commit it as `spec(<ID>): approve implementation plan`;
   reuse an earlier explicit approval that covers the exact plan instead of asking again. Then set stage
   `implementation` and delegate. The CLI rejects `implementation` without the approved plan committed.

## Stage procedure

### `specification`
- **Entry:** `running` / `specification`; worktree verified. **Delegate:** none.
- **Do:** load `skills/work-spec/SKILL.md` and run it in the worktree with the selected outcome and prototype
  references, if any. It owns questions, readiness, and isolated spec commits; consume its path, verdict,
  resolved canonical questions, and remaining blockers. Route decision needs/conflicts through `rules.md`
  section 3. When ready, load `skills/spec/SKILL.md` for approved canonical changes; commit them and refine
  the work spec with final rule IDs per `rules.md` section 4 step 6. Skip unchanged commits. Then present the
  implementation plan (`work-spec` section 7) and obtain the user's explicit approval per `rules.md` section 4
  step 7.
- **State:** record `--spec <worktree-relative path>` with `task move`; keep blocker and next-action state
  current while waiting.
- **Exit:** readiness passes, canonical changes are applied, final rule IDs replace temporary references,
  the spec path is recorded, and the approved plan is committed → move to `implementation` with the next action
  naming delegation.
- **Failure:** remain in `specification` until its blockers and readiness gaps are resolved.
