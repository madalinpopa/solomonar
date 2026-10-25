# Implementer integration mode

Framework version: {{ .Version }}

Load with `agents/implementer.md`; its common preflight, approvals, checks, and handoff gates apply.

- Apply `agents/implementer.md` section 2's preflight. Resolve main from `.solomonar/manifest.json` → `project.main_branch`;
  use `main` only without a manifest. Record `git rev-parse <main>` as the integration baseline.
- If `git merge-base --is-ancestor <main-SHA> HEAD` succeeds, no merge commit is needed. Otherwise run
  `git merge --no-ff -m "chore(<ID>): merge <main> into feature/<ID>" <main-SHA>`. Never amend, rebase, reset,
  or force-update handed-off history, even at a packet's suggestion.
- A delegated merge may import main's changes. Manual conflict resolution still follows `agents/implementer.md` section 1:
  orchestrator-owned documents → `INCOMPLETE`, naming the needed owner action; a product/architecture choice
  → `NEEDS_DECISION`. Preserve unresolved merge state; do not force a checkpoint, claim a candidate, or alter
  protected documents to finish the merge.
- For conflicts within your write scope, resolve and stage explicit paths with `git add -- <resolved-paths>`;
  inspect the complete `git diff --cached`, then finish with
  `git commit -m "chore(<ID>): merge <main> into feature/<ID>"`. Merge completion commits the whole merge
  index, unlike `agents/implementer.md` section 5's path-only commits; stop if it contains unrelated staged work.
- Run affected checks and apply `agents/implementer.md` section 5's result/handoff gates. Report integration baseline, touched areas,
  full resulting SHA, and verification limits; the orchestrator decides review reuse and merge readiness.
- Code changes for an integration or CI failure follow `agents/implementer.md` section 5 or 6, as the packet states.
