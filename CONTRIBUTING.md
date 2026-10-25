# Contributing to Solomonar

Thank you for helping improve Solomonar. This guide covers the development setup, the rules every change follows,
and step-by-step procedures, with templates, for extending the framework: rules, skills, and agents.

- [Before you start](#before-you-start)
- [Development setup](#development-setup)
- [How the project is organized](#how-the-project-is-organized)
- [Specs lead the code](#specs-lead-the-code)
- [Contributing to the framework](#contributing-to-the-framework)
  - [Writing framework text](#writing-framework-text)
  - [Change a framework rule](#change-a-framework-rule)
  - [Add an extension skill](#add-an-extension-skill)
  - [Add rules to an extension](#add-rules-to-an-extension)
  - [Add an extension agent](#add-an-extension-agent)
  - [Add a core skill or agent](#add-a-core-skill-or-agent)
  - [Verify a framework change](#verify-a-framework-change)
- [Commits and pull requests](#commits-and-pull-requests)
- [Reporting problems](#reporting-problems)
- [License](#license)

## Before you start

- **Small fixes** (typos, broken links, an obvious bug with a test) can go straight to a pull request.
- **Anything larger**, including every new rule, skill, agent, or extension, starts with an issue. Describe the
  problem, the behavior you propose, and who it affects, so the behavior is agreed before you write it.
- **Security problems** are never reported in a public issue. Follow [`SECURITY.md`](SECURITY.md).

## Development setup

You need Go 1.27.3 or later, Git 2.40 or later, and [Task](https://taskfile.dev). Linters are pinned and installed
into `.bin/` on first use.

```sh
git clone https://github.com/madalinpopa/solomonar.git
cd solomonar
task build        # builds bin/solomonar
task check        # lint, tests with the race detector, and govulncheck, as CI runs them
```

| Task | Purpose |
| --- | --- |
| `task build` | Build `bin/solomonar`. |
| `task run -- <args>` | Build and run the CLI, for example `task run -- task show`. |
| `task test` | Run all tests with the race detector. |
| `task test:integration` | Run all tests, including the transcript tests in `testdata/`. |
| `task fmt` | Format the code. |
| `task check` | Run lint, tests, and security checks. |

The documentation website lives in `site/`. Its setup and checks are in [`site/README.md`](site/README.md).

## How the project is organized

| Path | Contents |
| --- | --- |
| `assets/framework/` | The framework core: `rules.md`, `report.md`, `reviewer.md`, the agent roles in `agents/`, and the skills in `skills/`. `init` installs it as `docs/solomonar/`. |
| `assets/extensions/<group>/<name>/` | The extension catalog. Each extension is one skill or one agent, plus `extension.json`. An installed extension is copied as is to the project's store, `docs/solomonar/extensions/<name>/`, beside the project's own extensions. |
| `assets/templates/` | Templates for `AGENTS.md`, `docs/README.md`, and the empty task registry header (`state.json`). |
| `internal/` | The CLI: commands, the task registry, the init plan, adapters, and the extension catalog loader. |
| `docs/cli/` | The CLI interface specification and architecture. |
| `docs/design/`, `docs/research/` | Framework design notes, validation, and the research behind adopted rules. |
| `site/` | The documentation website. |

`main.go` embeds `assets/` into the binary, so every change under `assets/` ships with the next build. No
registration code is needed: the CLI discovers core skills, core roles, and catalog extensions from the directory
layout.

## Specs lead the code

Behavior is specified before it is built, and the documents always describe the code:

- [`site/src/docs/cli/`](site/src/docs/cli/) is the CLI reference: each command's synopsis, flags, output, and exit codes.
- [`docs/cli/interface.md`](docs/cli/interface.md) is the contract beneath it: check order, file planning, the
  manifest, JSON data, the extension catalog, and task validation.
- [`docs/cli/architecture.md`](docs/cli/architecture.md) records how the binary is built.
- [`assets/framework/`](assets/framework/) is the specification of framework behavior.

When your change departs from any of them, update the affected documents in the same pull request. A pull request
that changes behavior without its documents is incomplete.

## Contributing to the framework

The framework is Markdown that coding agents read in other people's repositories. Treat it as a contract: every
sentence is an instruction some agent will follow.

**Core or extension?** The core holds only the process: the items `rules.md` names as a step or a required role.
It never names a language, library, or tool. Anything technology-specific (Go rules, a templating library, an
accessibility reviewer) is an extension. When you are unsure, it is an extension.

| You want to | Change |
| --- | --- |
| Change how the process works | A framework rule in `assets/framework/` |
| Teach agents the rules of a language, library, or practice | An extension skill |
| Add rules to an existing stack | That extension's `SKILL.md` or topic rule files |
| Add a specialist reviewer | An extension agent, usually in the `review` group |
| Add a new process step or required role | A core skill or agent, only after a recorded design decision |

### Writing framework text

For evidence-based authoring, use the optional
[`extension-authoring` skill](assets/extensions/workflow/extension-authoring/SKILL.md).
It supports catalog contributions here and project-specific source in adopting
repositories, through research, authoring, and validation. Its stage references
own the detailed procedures.
The skill writes content; the CLI owns adoption, manifests, and runtime stubs.

Use these requests with the installed skill. In this source checkout, if it is
not installed, ask the agent to load
`assets/extensions/workflow/extension-authoring/SKILL.md` first.

| Intent | Example request |
| --- | --- |
| Complete workflow | `Use the extension-authoring skill to create a catalog skill for <topic>.` |
| Research only | `Use the extension-authoring skill to research a catalog extension for <topic> only.` |
| Author only | `Use the extension-authoring skill to author the extension described in <research-record>; authoring only.` |
| Validate only | `Use the extension-authoring skill to validate and refine <candidate-paths>, using <research-record>.` |
| Project extension | `Use the extension-authoring skill to create a project skill for <topic> in docs/solomonar/extensions/<name>/, then hand off solomonar extension add <name>.` |

In an adopting project, ask for a project extension instead of a catalog one: the
skill writes it to `docs/solomonar/extensions/<name>/`, `extension.json`
included, and the request ends with `solomonar extension add <name>`, which you
run to activate it. Include scope and supported versions when known. Stage-only
requests reuse the same research record; a complete creation request runs all
three stages.

- Write in English, in the imperative, one instruction per bullet. Prefer `must` and `never` only where they
  are meant literally.
- Keep files short. Agents load them into limited context; put rarely needed detail in a separate file and load it
  by trigger.
- Give each rule one owner. Reference a rule (`report.md` section 5, `go/C-4`) instead of copying it.
- Every core file and every extension `SKILL.md` or agent file carries the line `Framework version: <current>`
  under its title. Use the version in `assets/framework/rules.md`; the release task updates every copy.
- Skill and agent frontmatter holds `name` and `description`, each on **one line**. The CLI copies them into the
  runtime adapters as they are, so the description must say what the skill or role does and when to use it.

### Change a framework rule

Framework rules live in the file that owns their subject:

| Subject | File |
| --- | --- |
| Orchestration, approvals, task flow, review triage, state, merge, closure, recovery | `assets/framework/rules.md` |
| What every reviewer does (preflight, isolation, evidence, result) | `assets/framework/reviewer.md` |
| Report format, result values, findings, the PR Verification section | `assets/framework/report.md` |
| One role's specialty | `assets/framework/agents/<role>.md` |
| One skill's procedure | `assets/framework/skills/<skill>/SKILL.md` |
| What agents load and when | `assets/templates/AGENTS.md` |

Steps:

1. Open an issue that states the problem the rule solves and the case where today's rules fail.
2. Find the owning file and section. Search the framework for rules on the same subject
   (`grep -rn "<keyword>" assets/framework assets/templates`) and change those instead of adding a competing one.
3. Write the rule as a bullet in the owning section. Name who acts, what they do, and the observable result:

   ```markdown
   - <Actor> must <action> when <condition>; <consequence or exception>. See <owner reference> for <detail>.
   ```

4. Update every file that references the changed behavior: the role files that follow it, the
   [framework guide](site/src/docs/framework.md), [the task lifecycle](site/src/docs/workflow.md), and
   `docs/cli/interface.md` when the CLI checks it.
5. [Verify the change](#verify-a-framework-change).

### Add an extension skill

An extension skill carries the rules of a language, library, or practice. It installs to
`docs/solomonar/extensions/<name>/`, every file copied as is, and gets a skill stub in each runtime.

1. **Choose the group.** Use the group of the decision that selects the extension: `language`, `architecture`,
   `frontend`, `backend`, `persistence`, `testing`, `operations`, or `workflow`. The groups are fixed. Adopting an
   extension in the first seven groups is an architecture or dependency decision in the user's project.
2. **Create the directory** `assets/extensions/<group>/<name>/`. The name matches `[a-z][a-z0-9-]*` and is unique
   across the catalog.
3. **Add `extension.json`.** The catalog decodes it strictly, so unknown fields are rejected:

   ```json
   {
     "name": "<name>",
     "kind": "skill",
     "group": "<group>",
     "title": "<One-line name, shown by extension list>",
     "requires": [],
     "conflicts": [],
     "applies_when": "A change touches <files or behavior>",
     "paths": ["**/*.<ext>"],
     "detect": ["**/<marker file>"]
   }
   ```

   | Field | Rule |
   | --- | --- |
   | `name`, `group` | Equal to the directory names. |
   | `kind` | `skill`. |
   | `title`, `applies_when` | Required, not blank. `applies_when` is written into the project's `AGENTS.md`, and agents load the skill when it matches. |
   | `requires`, `conflicts` | Names of other catalog extensions. Never add a dependency silently: a missing one is an error the user sees. |
   | `paths` | Globs of the files the extension governs; the orchestrator uses them to name the extension in review packets. |
   | `detect` | Optional. Globs of repository files that suggest the extension, such as `**/go.mod`; `init` preselects it and `extension list` marks it `suggested`. Omit it when no file reliably signals the need. |
   | `default` | Optional, `false` by default. `true` suggests the extension in every project; keep it for procedures most projects use. |
   | `invocation` | Optional, `model` by default. `user` means only the user starts the skill: its stubs disable model invocation and `AGENTS.md` lists it on a separate line. |

   Neither `detect` nor `default` installs anything; the user still chooses at `init` or with `extension add`.

4. **Add `SKILL.md`.** Its frontmatter `name` equals the extension name:

   ```markdown
   ---
   name: <name>
   description: Engineering rules for writing or reviewing <subject>. Use when a change touches <files or behavior>.
   ---

   # <Title>

   Framework version: <current>

   Cite `<name>/C-n`; apply framework authority and review gates.

   ## Common rules

   - <name>/C-1 (must) — Rule: <the rule, stated so a reviewer can check it>.
     Exception: <the case where it does not apply, if any>.
   - <name>/C-2 (should) — Rule: <a recommendation that needs judgment>.

   ## Load by impact

   Evaluate every trigger against changed code and affected callers/contracts.
   Load matching files before writing or reviewing; expand as impact emerges.
   Paths are relative to this file.

   | Impact | Load |
   | --- | --- |
   | <When this kind of change happens> | `rules/<name>/<topic>.md` |

   Spec reviewers account for loaded `must` rules and give one non-applicability
   reason per unloaded file. A missing required file is missing input.

   ## Checks

   <The commands that verify the rules when the project names none, for example build, format check, lint, tests.>
   ```

   Keep only the rules that apply to almost every change in `Common rules`. Move topic rules to files under
   `rules/<name>/`, routed from the `Load by impact` table. Drop the table when there are no topic files.

5. **Document it.** Add the extension to the catalog listings on the site (`site/src/docs/cli/init.md`,
   `site/src/docs/cli/extension.md`, `site/src/docs/configuration.md`) and in the README's project status.
   Put research and sources for a substantial rule set in `docs/research/<name>-rules/`, as
   [`docs/research/go-rules/`](docs/research/go-rules/) does.
6. [Verify the change](#verify-a-framework-change).

The [`go` extension](assets/extensions/language/go/) is the reference implementation.

### Add rules to an extension

1. **Pick the file.** A rule that applies to almost every change goes in `SKILL.md` under `Common rules`. Any other
   rule goes in the topic file whose `Load by impact` trigger matches it. Create a new topic file only for a new
   trigger, and add its row to the table.
2. **Assign the next unused ID.** Rule IDs are stable references that reports and reviews cite. Take the highest
   ID in the extension plus one; never renumber, and never reuse a retired ID.
3. **Choose the strength.**
   - `must`: a violation is a defect that blocks the review. Only for rules a reviewer can verify from the change.
   - `should`: a recommendation; a deviation needs a reason, not a fix.
4. **Write the rule** with this shape. Add an `Exception:` only for a real, recurring case:

   ```markdown
   - <name>/C-<n> (must|should) — Rule: <one checkable statement>.
     Exception: <the specific case where the rule does not apply>.
   ```

5. **Template for a new topic file** `rules/<name>/<topic>.md`:

   ```markdown
   # <Language or library> <topic>

   <One line: what this file covers and what it leaves to the framework or other files.>

   - <name>/C-<n> (must) — Rule: <rule>.
   - <name>/C-<n> (should) — Rule: <rule>.
   ```

6. **Changing or retiring a rule** changes its meaning in projects that already installed it. Say so in the pull
   request, and record the previous-to-current mapping in the extension's research notes.

### Add an extension agent

An extension agent is an extra specialist reviewer, for example for accessibility or a regulated domain. It
installs to its own store directory, `docs/solomonar/extensions/<name>/` with `<name>.md` and `extension.json`,
runs in parallel with the core reviewers when its `applies_when` matches the change, and gets read-only tools in
runtimes that support a tool list.

1. **Create the directory** `assets/extensions/review/<name>/`. Use another group only when the reviewer belongs to a
   stack decision.
2. **Add `extension.json`** with `"kind": "agent"`. Here `applies_when` is the condition under which triage
   requires this reviewer, so make it precise. `detect` and `default` work as for a skill; `invocation` stays
   unset, because an agent is never user-invoked:

   ```json
   {
     "name": "<name>",
     "kind": "agent",
     "group": "review",
     "title": "<Specialty> reviewer",
     "requires": [],
     "conflicts": [],
     "applies_when": "A change touches <files or behavior that need this specialty>",
     "paths": ["<glob>"]
   }
   ```

3. **Add `<name>.md`**, following the structure of the core reviewers. It inherits preflight, isolation, evidence,
   and results from `reviewer.md` and `report.md`; it adds only its specialty:

   ```markdown
   ---
   name: <name>
   description: Reviews one candidate revision for <specialty>: <the defects it looks for>.
   ---

   # <Specialty> reviewer

   Framework version: <current>

   Load `AGENTS.md`, `docs/solomonar/report.md`, and `docs/solomonar/reviewer.md`; follow the shared preflight,
   isolation, evidence, and result rules. This file adds only the <specialty> specialty.
   Report role: `<name>`; provisional finding prefix: `<PREFIX>-n`. The orchestrator owns review priority
   and routing; do not load `rules.md`.

   ## 1. Focus

   - <What this reviewer checks, one subject per bullet.>

   ## 2. Method

   1. <The steps that establish each finding, in order.>

   ## 3. Checks

   | Trigger | Required verification |
   |---|---|
   | <Kind of change> | <Check or command> |

   - No command established or tool unavailable → record `not run` with the reason. Never turn missing required
     verification into `PASS`.

   ## 4. Classifying findings

   - Classify under `report.md` section 5. <What makes a finding in this specialty blocking.>

   ## 5. Out of scope

   <Subjects owned by other reviewers, with the role that owns each.>
   ```

   Choose a finding prefix no other role uses (`SEC`, `COR`, `SPEC`, and `ADV` are taken).
4. Document it as for a skill (step 5 above), then [verify the change](#verify-a-framework-change).

### Add a core skill or agent

The core changes only when the process itself gains a step or a required role. Every project gets it and cannot
remove it while the framework stays installed: `extension remove` rejects core items, and only the
whole-project `solomonar uninstall` (CLI-D32) removes the core with the rest of the installation. Open an issue
first and record the decision in `docs/cli/interface.md` (the core boundary is decision CLI-D14).

**A core skill** lives in `assets/framework/skills/<name>/`:

1. Add `SKILL.md`, with an optional `template.md` for the document it produces:

   ```markdown
   ---
   name: <name>
   description: <What it produces and when>. Use it to <cases>. Do not use it for <cases owned by another skill>.
   ---

   # <Name> skill

   Framework version: <current>

   ## 1. Purpose and boundaries

   - <What the skill produces, which files it may edit, and what it must never do.>

   ## 2. Start

   1. <Preflight checks and the context to read before acting.>

   ## 3. <Procedure>

   - <The steps and rules of the procedure.>

   ## 4. Finish

   - <Verification, commit, and what to report.>
   ```

2. Add its route to the table in `assets/templates/AGENTS.md`, and the step that uses it to `rules.md` or the
   `solomonar` skill.

**A core agent** is a file `assets/framework/agents/<role>.md`, following the [extension agent
template](#add-an-extension-agent). A reviewer also needs:

1. A row in the triage table in `rules.md` section 6, with its priority and the condition that requires it.
2. Its name in the role lists of `report.md` section 2 and `reviewer.md` section 1, and in the review route of
   `assets/templates/AGENTS.md`.
3. Its packet in `agents/orchestrator.md` section 4, and an `Out of scope` reference from the reviewers it borders.

For either, update the core table in `docs/cli/interface.md` section 4.1, the file listings in
`site/src/docs/cli/init.md` and `site/src/docs/set-up-a-project.md`, and the transcript tests in `testdata/` that
list installed files.

The CLI picks up new core skills and roles from the directory layout; runtime adapters for Claude Code and Codex
are generated without code changes. Reviewers get the read-only tool list automatically, and the orchestrator
never gets an adapter, because it runs as the top-level session.

### Verify a framework change

1. Run `task check`, then `task test:integration`. The tests read the embedded catalog and fail on a malformed
   `extension.json`, a missing `SKILL.md` or agent file, or frontmatter without `name` and `description`.
2. Install the change into a scratch project and read what an agent will read:

   ```sh
   task build
   tmp=$(mktemp -d) && cd "$tmp"
   /path/to/solomonar/bin/solomonar init --git-init --runtime claude,codex --ext <name> \
     --objective "Scratch project" --yes
   ```

   Check the installed files in `docs/solomonar/extensions/<name>/`, the `## Extensions` section of `AGENTS.md`,
   and the stubs in `.claude/` and `.agents/` or `.codex/`. For an extension that is not default, also run
   `solomonar extension list` and `solomonar extension add <name>` in a project initialized without it.
3. Run a real task with the changed rules in Claude Code or Codex when the change affects agent behavior, and
   describe what you observed in the pull request.

## Commits and pull requests

- Write commit messages in [Conventional Commits](https://www.conventionalcommits.org/) form, with the area as the
  scope: `feat(cli): ...`, `fix(task): ...`, `feat(agents): ...`, `feat(go): ...`, `docs(site): ...`,
  `docs(research): ...`.
- Keep one logical change per commit. A rule change and the documents it updates may share a commit; unrelated
  edits may not.
- Add or update tests for every behavior change, and run `task check` until it passes.
- In the pull request, link the issue, summarize the behavior change, and list the documents you updated. For a
  framework change, describe how you verified it.
- Do not write the current release version in the tree; the release tag sets it (AD-10). Framework and extension
  Markdown files write `{{ .Version }}`, site pages `{{ release.number }}`, and the specs and `README.md`
  `X.Y.Z`. Keep historical delivery versions and compatibility boundaries as literal versions.

## Reporting problems

Open an issue with the output of `solomonar version`, the command you ran, the error code, and the relevant Git
state, with secrets redacted. The [troubleshooting guide](https://solomonar.dev/docs/troubleshooting/#report-a-problem-with-evidence)
lists what helps. Report security problems privately, as [`SECURITY.md`](SECURITY.md) describes.

## License

By contributing, you agree that your contributions are released under the project's [MIT License](LICENSE).
