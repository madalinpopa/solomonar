# Spec discovery mode

Framework version: {{ .Version }}

Load with `skills/spec/SKILL.md`; its boundaries, IDs, approval/conflict checks, and validation apply.

Discovery runs in the primary checkout on a named branch outside `feature/*` and `workflow-state`.
If commits are forbidden there, use `spec/<topic>` and tell the user. Inspect location/status per `SKILL.md` section 2.

1. Start from the user's outcome or handoff references and `docs/README.md`. Load only relevant capability
   specs, shared UI rules, and scoped decision entries. Read architecture/stack entries only when needed to
   resolve an implication; expand context for discovered dependencies or shared contracts.
2. Apply `SKILL.md` section 4’s prototype approval check to each handoff.
3. Extract only demonstrated, approved behavior per flow: actions/inputs, outputs, transitions, validation,
   business constraints, relevant success/failure/edge cases, and persistence/integration outcomes. A
   `SIMULATED:` step yields only its observable outcome (e.g. "a placed order is still listed after reload"); the
   mechanism (e.g. `localStorage`) stays evidence, subject to `SKILL.md` section 1's decision boundary.
4. Map items to capabilities: reuse existing files; create a file only for an independently loadable area; put
   cross-capability layout and interaction rules in `docs/ui.md`. When the split is unclear, propose one and ask
   per `AGENTS.md` rule 11, with the proposal as the recommended option.
5. Mark faithfully extracted approved behavior **Agreed**, citing its approval and prototype reference. Keep
   existing unconfirmed observations **Observed**; promote them only after representation and prototype approval.
   A readiness label or intentional-change entry alone does not approve inferred rules or additional behavior.
6. Apply `SKILL.md` section 4’s conflict procedure.
7. Surface ambiguous or missing behavior for a decision, blocking questions first. Send it to `prototype` for
   demonstration/approval before finalizing the related rule; continue independent approved extraction only.
8. Delete an **Open** item that an approved `P/D-n` resolves and cite that decision under "References".
   Create or update files and validate them per `SKILL.md` section 6. Put prototype references under "References".
9. Commit (Commits below) and report: files changed; rules per label per file; open questions, blocking first;
   prototype defects found; next action: `work-spec` for a selected change, `prototype` for blocking unknowns,
   or answering remaining project open questions (`rules.md` section 3).

## Commits

- Messages: `docs(<capability>): <summary>`; `docs(ui): <summary>` for `docs/ui.md`. One commit per file when
  practical; include the related `docs/README.md` index change.
- Commit only intended changes allowed by `SKILL.md` section 1, selected by path, on the branch established at startup.
  Inspect the staged diff; preserve unrelated staged changes. Skip the commit when nothing relevant changed.
- Never push unless the user asks.
