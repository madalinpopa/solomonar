# Go engineering rules: research and adoption

Reviewed 2026-10-22 against Solomonar 0.1.3, local baseline `636af62f`.
Status: adopted into the framework catalog on 2026-10-22. Runtime rules live in
`assets/extensions/language/go/`; this document retains the evidence and migration notes.

## 1. Research summary

Keep Solomonar's language-agnostic core and installed Go extension. Put six common
rules in its entrypoint and load four focused files by impact. Correct the current
absolute rules before adding coverage. Preserve the distinction between mandatory
correctness constraints and recommendations that require judgment.

### Verified baseline and source authority

- The latest stable release is **Go 1.27.3**, released 2026-10-19. This repository
  declares `go 1.27.3`; installed guidance must follow each adopting project's
  supported versions instead. [Release history][releases], [go.mod reference][gomod].
- Current language/API documentation and release notes govern technical behavior.
  Effective Go remains useful for idioms but explicitly says it is not actively
  updated and omits modules, generics, and newer libraries. [Effective Go][effective].
- Go Code Review Comments gives official-project review guidance, not a complete
  language specification. [Code Review Comments][crc].
- Google's Guide and Decisions are normative **at Google**; Best Practices is
  advisory. Their rules are supporting guidance, not universal Go requirements.
  Clarity and simplicity take priority over concision. [Guide][google-guide],
  [Decisions][google-decisions], [Best Practices][google-best].
- Repository-approved policy still governs local choices. The framework already
  owns authority, scope, generated-file ownership, architecture approval, acceptance
  coverage, and review gates. Do not reproduce those procedures in Go rule files.
  [AGENTS template](../../../assets/templates/AGENTS.md),
  [implementer](../../../assets/framework/agents/implementer.md),
  [spec reviewer](../../../assets/framework/agents/spec-reviewer.md).

### Improvements to the previous extension

These are policy and context-design findings, not claims of production defects in
the CLI. The table describes the Go skill at the review baseline above; the
[adopted Go skill](../../../assets/extensions/language/go/SKILL.md) contains its replacement.

| Previous rule or mechanism | Finding | Adopted treatment |
| --- | --- | --- |
| Entire skill loads for every Go change | Networking, transactions, and channels accompany unrelated changes. | Keep a small common set and explicit impact triggers. |
| C-1: nothing newer than `go.mod` | Ignores legitimate version-constrained implementations. | Preserve minimum-version compatibility across supported build configurations; allow guarded implementations. [stdversion][stdversion] |
| C-2: formatting | A formatter/check owns mechanical layout. | Retire the separate rule; retain a fallback formatting check. [Effective Go][effective] |
| C-3: generated files | Already owned by the AGENTS template. | Retire the duplicate Go rule. |
| C-4: every ignored error needs a comment | Encourages noise for already documented, harmless cleanup. | Require meaningful error handling; explain discards whose reason is not evident or documented. Keep this as an explicitly qualified project policy. [Error handling][effective], [transaction example][transactions] |
| C-5: wrapping and inspection combined | Caller inspection and producer exposure are separate choices. | Keep C-5 for inspection; use C-21 for the error contract. Wrapping commits an API to the exposed cause. [Go errors][errors] |
| C-8: contexts | Useful imposed-signature exception; also contains a separate storage decision. | Preserve propagation; move storage to C-31 and add context-value guidance. [Context][context] |
| C-9: every goroutine can be stopped | Finite work can complete naturally; process-lifetime work has a different owner. | Require ownership and an exit/shutdown path, plus completion before resource teardown. [Pipelines][pipelines] |
| C-10: every channel has an owner that closes it | Closing signals protocol completion; it is not resource reclamation. | Require safe closure when used; permit channels that remain unclosed. [Specification][spec-close] |
| C-11: no I/O while locked | Moving I/O can break serialization or an invariant. | Recommend moving it only when the invariant permits. This is project guidance, not a language rule. [Memory model][memory], [simplicity][google-guide] |
| C-12: resources plus `Rows.Err` | Cleanup and iteration failure are independent decisions. | Separate C-12 and C-33; make ownership transfer explicit. [Cleanup][defer], [querying][querying] |
| C-13: transactions | Useful; must permit deferred rollback after commit. | Retain finalization and successful-commit requirement. [Transactions][transactions] |
| C-14: no accumulating defer | Harmless bounded retention is valid. | Trigger the recommendation when live-resource accumulation matters. [Defer semantics][defer] |
| C-15: explicit timeouts everywhere | Existing context bounds can suffice; a total timeout can terminate healthy streams. | Distinguish finite operations, streaming lifetime, and server phase limits. [HTTP Client, Request, and Server][http] |
| C-16: architecture imports | Approved boundaries already belong to architecture and the spec reviewer. | Retire the duplicate, retain architecture enforcement at its owner. |
| C-17: follow a whole style document | Too broad to serve as a focused runtime instruction. | Keep call-site naming guidance; add scoped API/design decisions with evidence below. |
| C-18: race pass plus no flakes | Race evidence and deterministic test design differ. | Keep deterministic design as C-18; put race execution in the check route. Passing `-race` covers executed paths only. [Race detector][race] |
| C-19: helpers and failure messages | Two independent choices. | Keep C-19 for helper attribution; use C-38 for failure diagnostics. [Go Test Comments][test-comments] |
| C-20 with `tdd`: every behavior test uses a table and `t.Run` | Test-first workflow does not imply this structure; different scenarios may need different logic. | Retire C-20; recommend tables only for shared test logic. Subtests are optional. [Google Decisions][google-decisions] |
| Check table maps `go vet` to ignored errors/channel safety | Vet is heuristic, not a proof of these properties. | Remove claimed complete rule coverage; combine tools with code evidence. [Vet][vet] |

### Important tradeoffs

An interface may serve a real test boundary without having two production
implementations. Consumer ownership is a default; protocol packages can legitimately
publish interfaces. Do not impose a numeric implementation count or prohibit
interface return types. [Google interface guidance][google-best].

Use SOLID as reasoning about responsibility and dependency direction, not a demand
for layers. Apply DRY when shared behavior is actually shared; modest duplication
can be clearer. Generic abstractions should solve a current problem. These are
qualified design recommendations. [Google simplicity][google-guide],
[When to use generics][generics].

Prefer existing standard-library capabilities when sufficient, but allow a maintained
dependency to replace substantial custom machinery. The standard-library-first
preference is project policy; the official supply-chain guidance supports the cost
of adding trust relationships, not a universal dependency ban. [Supply chain][supply-chain].

Keep the project's existing assertion library and organization. For example, this
repository already uses `testify` and permits it in `.golangci.yml`; advisory external
style preferences do not justify replacing it. Do not impose `cmd/`, repository
interfaces, file-size limits, or layer templates. Official layout guidance offers
different arrangements for different project needs. [Module layout][layout].

### Version-sensitive recommendations

| Version | Verified behavior | Consequence for agents |
| --- | --- | --- |
| Go 1.18+ | Native fuzzing is available. [Fuzzing][fuzz] | Consider it for suitable properties, not every function. |
| Go 1.22+ language semantics | Loop-declared variables are per iteration. [Release notes][go122] | Do not mechanically add historical `tc := tc` workarounds; reassigned variables still require care. |
| Go 1.24+ | `T.Context` and `B.Loop` are available. [Testing][testing] | Use lifecycle context where appropriate; benchmark APIs remain task-specific references. |
| Go 1.25+ | `testing/synctest` is stable; `WaitGroup.Go` is available. [Release notes][go125] | Consult them when concurrency testing or task tracking needs them. Neither is mandatory for every goroutine. |
| Go 1.27+ | Generic methods are supported; interface methods cannot declare type parameters. [Release notes][go127] | Reject outdated blanket claims that methods cannot be generic; do not add generics without need. |
| Go 1.27+ tooling | `go test` runs the `stdversion` vet check by default. [Release notes][go127] | Avoid duplicate execution where existing commands cover it; check actual configured analyzers. |

Keep this release inventory out of runtime rules. C-1 directs agents to the target
module and supported configurations; consult specific API documentation when needed.

## 2. Rule architecture

Use the existing portable skill location. The requested `rules/go/` lives **inside
the Go extension**, so it remains versioned and owned by that extension rather
than creating a competing repository-root rule system.

```text
assets/templates/AGENTS.md                 existing generic extension route
assets/extensions/language/go/
  extension.json                          existing discovery metadata
  SKILL.md                                common rules, impact routes, checks
  rules/go/
    design.md                             API/type/package/dependency/performance decisions
    io.md                                 contexts, resources, networking, transactions
    concurrency.md                        lifecycle, channels, synchronization
    testing.md                            Go-specific test design
docs/research/go-rules/review.md            evidence and maintenance rationale
```

At installation, the Go directory becomes `docs/solomonar/skills/go/` (2026-10-24:
the install path is now the store, `docs/solomonar/extensions/go/`). Keep the
research outside `assets/` so it is neither installed nor offered as routine task
context. Split topics only where they support an independent loading decision;
do not create separate tiny files for naming, error strings, receivers, or benchmarks.

This fits existing mechanisms:

- `extensionFiles` walks nested skill resources;
  `installedPath` keeps their relative structure.
  [Installer](../../../internal/project/extension.go).
- The existing installer tests cover nested reference copying. Only `SKILL.md`
  receives conditional-section rendering; subordinate files are copied literally.
  Keep `With tdd` directives out of subordinate files.
  [Tests](../../../internal/project/extension_test.go),
  renderer `internal/project/render.go` (removed 2026-10-24).
- Runtime stubs already point to the portable `SKILL.md`. No adapter-specific
  copy of the engineering rules is needed.
  [Runtime adapters](../../../internal/project/runtime.go).
- AGENTS already routes to installed extensions, and the role contracts load
  their required files. Add no Go-specific rule list to AGENTS.

The entrypoint evaluates **all four triggers** against affected contracts as well
as filenames. A cancellation change can require both I/O and concurrency even when
no new `go` statement appears. A dependency upgrade can affect consumer code and
tests without editing them. Running existing tests alone does not require reading
test-authoring guidance.

The spec reviewer groups evidence for loaded mandatory rules and gives one reason
for each unloaded topic. It must not load every file just to enumerate inapplicable
IDs. This preserves the existing requirement to check every **applicable** rule.

Research supports selective retrieval as a context-engineering technique, but does
not establish an optimal file count or prove better coding outcomes for this
proposal. The four-topic split is a framework-specific design judgment.
[Context engineering][context-engineering].

### Precedence

Retain the framework's existing order: scoped user decisions, approved global
rules, approved work spec, then preferences. Conflicting approved requirements
follow the existing decision procedure. Canonical Go documentation determines
technical semantics; external style recommendations do not silently override
project conventions or become approved global requirements.

## 3. Ready-to-use Markdown files

The approved drafts have moved into the catalog. These are the canonical runtime files:

| File | Purpose |
| --- | --- |
| [SKILL.md](../../../assets/extensions/language/go/SKILL.md) | Six common rules and impact-based loading/checks |
| [rules/go/design.md](../../../assets/extensions/language/go/rules/go/design.md) | Ten design rules |
| [rules/go/io.md](../../../assets/extensions/language/go/rules/go/io.md) | Nine context/resource/I/O rules |
| [rules/go/concurrency.md](../../../assets/extensions/language/go/rules/go/concurrency.md) | Six concurrency rules |
| [rules/go/testing.md](../../../assets/extensions/language/go/rules/go/testing.md) | Seven testing rules |

Each rule names its strength and imperative instruction. Its file/route supplies
obvious scope; exceptions appear only when needed. No citations, tutorials, or
extended rationale occur in runtime files.

The existing AGENTS route is already sufficient; retain it rather than adding
another always-loaded section:

```markdown
Load every installed extension skill whose "applies when" matches your change,
before writing or reviewing code. The orchestrator names matching extensions in packets.
```

This is an abbreviated illustration of the existing route, not a replacement for
the full AGENTS template or its authority rules.

### Adoption and retained-ID mapping

The catalog keeps its extension metadata and runtime destinations. Its entrypoint
now routes to four topic files, and spec-reviewer method 7 follows extension
loading triggers while accounting for applicable mandatory rules and skipped topics.
AGENTS remains language-agnostic. No installer, schema, or adapter change is required.

Retained IDs keep their engineering subject; C-2, C-3, C-16, and C-20 are retired
and not reused. C-5/C-12/C-19 retain part of their earlier meaning;
C-21/C-33/C-38 hold the separated decisions. C-18 retains deterministic tests;
race evidence is a check. The full previous-to-current mapping is in section 1.

Go rules no longer render a `tdd`-specific layout requirement. Renderer tests
retain generic conditional-section coverage and check that Go renders the same
rules with and without `tdd`. Installation tests cover all five Go files,
extension ownership, initialization/reruns, and both runtime adapters.

Do not silently change rule meanings in an already installed project. Publish them
as an extension/framework update with a retained-ID mapping and the project's
normal adoption procedure. Inspect the release's actual CLI capabilities: `sync`
and `upgrade` are currently planned commands, not available migration mechanisms.

## 4. Evidence matrix

Classification: **O** = official technical contract or convention; **B** =
established, qualified best practice; **P** = opinionated project recommendation.
Confidence describes evidence strength, not the mandatory/advisory status chosen
for the framework. High = explicit primary guidance; moderate = reasoned adaptation.

### Common rules

| Rule | Source | Class / confidence | Rationale and practical benefit | Exceptions or limits |
| --- | --- | --- | --- | --- |
| C-1 | [go.mod][gomod], [stdversion][stdversion] | O / high | Preserve supported builds and prevent accidental version drift. | Version-constrained implementations are valid with working fallback builds. |
| C-4 | [Handle Errors][crc], [transaction cleanup][transactions] | O+B / high | Prevent false success without redundant discard comments. | Harmless, evident or documented discards; qualify against project error policy. |
| C-5 | [Go errors][errors] | O / high | Inspect wrapped causes correctly. | `err == nil` remains valid; text matching is valid for textual contracts. |
| C-6 | [Google error guidance][google-decisions] | B / high | Make failures attributable without repeated noise. | Direct propagation is adequate when context adds nothing. |
| C-7 | [Effective Go errors][effective], [panic/recover][defer] | O / high | Keep operational failure recoverable by the caller. | Programmer invariants and documented `Must` APIs may panic. |
| C-21 | [Whether to Wrap][errors] | O / high | Preserve intended cause inspection and error abstraction. | `%w` is not universally required; wrapped causes become API commitments. |

### Design

| Rule | Source | Class / confidence | Rationale and practical benefit | Exceptions or limits |
| --- | --- | --- | --- | --- |
| C-17 | [Package names][package-names] | O / high | Make names useful in caller context. | Short locals and conventional receiver names are valid. |
| C-22 | [Generics guidance][generics], [simplicity][google-guide], [interfaces][google-best] | B / high | Require a present benefit before abstraction. | A test boundary or public protocol is a real need; no arbitrary repetition threshold. |
| C-23 | [Interfaces][google-best] | B / high | Keep required method sets small and owned near use. | Shared protocols and intended interface returns are legitimate. |
| C-24 | [Module layout][layout], [API evolution][compatibility] | B / high | Limit maintenance commitments and protect invariants. | Plain data structs can expose fields; do not add getters mechanically. |
| C-25 | [Module compatibility][compatibility] | O+B / high | Avoid breaking existing consumers during local changes. | Deliberate breaking changes follow the task and module versioning policy. |
| C-26 | [Receiver Type][crc] | O / high | Make field updates visible and avoid invalid copies. | Maps, slices, functions, and channels may use value receivers appropriately. |
| C-27 | [Doc comments][doc-comments] | O / high | Tell callers the contract the signature cannot express. | Match documentation depth to API complexity; document why when non-obvious. |
| C-28 | [Diagnostics][diagnostics] | B / high | Optimize an observed cost and verify the gain. | Clearer algorithms need no ritual benchmark; profiling/representative workloads can be better than microbenchmarks. |
| C-29 | [Supply-chain trust][supply-chain], [Effective Go][effective] | P / moderate | Avoid unnecessary maintenance and dependency trust. | Established dependencies can reduce complexity; this is not an official stdlib-only rule. |
| C-30 | [Package design][package-names], [module layout][layout] | B / high | Organize around coherent use and actual dependencies. | No fixed layer scheme, required directory layout, or package-size threshold. |

### Contexts and I/O

| Rule | Source | Class / confidence | Rationale and practical benefit | Exceptions or limits |
| --- | --- | --- | --- | --- |
| C-8 | [Context][context], [Contexts][crc] | O / high | Carry request cancellation through blocking operations. | Imposed callbacks/interfaces retain their signatures; a context cannot cancel arbitrary blocking APIs automatically. |
| C-12 | [Defer][defer], [cancel functions][context], [HTTP bodies][http] | O+B / high | Match cleanup to ownership on every path. | Explicit transfer changes the owner; C-4 governs meaningful cleanup errors. |
| C-13 | [Transactions][transactions] | O / high | Prevent abandoned transactions and false commit success. | Deferred rollback after commit is normal; do not overwrite a useful primary error with harmless cleanup. |
| C-14 | [Defer semantics][defer] | B / high | Avoid retaining resources for an entire loop's lifetime. | Bounded retention can be intentional; a helper function can scope a defer. |
| C-15 | [Client and Request][http] | B+P / moderate | Bound finite work without duplicating existing deadlines. | Streaming uses cancellation and suitable phase/idle bounds; policy does not prescribe timeout values. |
| C-31 | [Context rules][context], [Contexts][crc] | O / high | Keep context lifetime tied to each operation. | An imposed interface can require stored context; do not redesign external contracts. |
| C-32 | [Context values][context] | O / high | Keep dependencies and options explicit. | Request-scoped identity/tracing metadata belongs in context when appropriate. |
| C-33 | [Query errors][querying] | O / high | Distinguish complete results from failed iteration. | A prior scan failure can return directly; the rule applies before reporting success. |
| C-34 | [Server][http] | B+P / moderate | Limit stalled protocol phases without breaking legitimate traffic. | Account for proxy limits, uploads, and streaming; no universal server timeout tuple. |

### Concurrency

| Rule | Source | Class / confidence | Rationale and practical benefit | Exceptions or limits |
| --- | --- | --- | --- | --- |
| C-9 | [Lifetimes][crc], [pipelines][pipelines], [CancelFunc][context] | O+B / high | Prevent leaked work and teardown races. | Natural completion and process-owned workers are valid; cancellation does not itself wait for exit. |
| C-10 | [Close specification][spec-close] | O / high | Prevent sends/double-close panics. | Channels can remain unclosed; a coordinator may close after all producers finish. |
| C-11 | [Memory model][memory], [simplicity][google-guide] | P / moderate | Reduce unnecessary lock duration without weakening invariants. | I/O under a lock can be necessary; do not split a required atomic operation. |
| C-35 | [Memory model][memory] | O / high | Ensure cross-goroutine state has defined synchronization. | Concurrent read-only access needs no extra lock; ownership transfer is another valid design. |
| C-36 | [Bounded parallelism][pipelines] | B / high | Prevent input size from causing resource exhaustion. | Small fixed fan-out needs no worker-pool framework. |
| C-37 | [sync contracts][sync] | O / high | Preserve primitive identity and synchronization state. | Copying channel handles is valid; obey the specific type's non-copy contract. |

### Testing

| Rule | Source | Class / confidence | Rationale and practical benefit | Exceptions or limits |
| --- | --- | --- | --- | --- |
| C-18 | [Stable results][test-comments], [synctest][synctest] | B / high | Make tests repeatable across machines and schedules. | Tests of time still use time; controlled clocks, events, or bounded condition waits avoid guessed sleeps. |
| C-19 | [Mark Test Helpers][test-comments] | O / high | Attribute failures to meaningful callers. | A helper whose internal location is useful need not hide it. |
| C-38 | [Useful failures][crc], [test diagnostics][test-comments] | O / high | Make a failure actionable from one result. | Exact message formats are not required; existing assertion tools can supply useful diffs. |
| C-39 | [Tables and subtests][google-decisions], [test forms][test-comments] | B / high | Share genuinely common logic without a test framework inside a table. | Standalone tests and plain tables without `t.Run` are valid. |
| C-40 | [T.Setenv, T.Chdir, T.Parallel][testing] | O+B / high | Avoid interference between parallel tests. | Parallelism is optional; local fixtures are safe when isolated. |
| C-41 | [T.Context, T.Cleanup, T.TempDir][testing] | O+B / high | Match fixture lifetime to the owning test. | Go 1.24 gates `T.Context`; cancellation-specific tests and cleanup may need separate contexts. |
| C-42 | [Fuzzing][fuzz] | B / high | Explore input spaces with reproducible properties. | Optional, not universal; ordinary tests execute seeds rather than doing a fuzz campaign. |

### Routing and verification policy

| Instruction | Evidence | Classification / limit |
| --- | --- | --- |
| Follow impact triggers and expand for contracts/callers. | [AGENTS](../../../assets/templates/AGENTS.md), [framework design](../../design/framework-v0.1.md), [context engineering][context-engineering] | P / moderate: fits the existing selective-loading contract; no measured quality claim. |
| Account for loaded must rules and unloaded-topic reasons. | [Spec reviewer method 7](../../../assets/framework/agents/spec-reviewer.md) | P / moderate: groups non-applicability without weakening mandatory coverage. |
| Use project checks before fallback commands. | [AGENTS commands route](../../../assets/templates/AGENTS.md), [implementer step 6](../../../assets/framework/agents/implementer.md) | Existing framework policy; it already delegates mechanical checks to tools. |
| Retain race checks and report unsupported execution. | [Current Go checks](../../../assets/extensions/language/go/SKILL.md), [race detector][race] | Existing project policy, not a Go requirement for every edit; passing only checks executed paths. |
| Broaden module verification and inspect vulnerabilities. | [Supply chain][supply-chain], current Go security check | P / moderate: dependency changes have module-wide impact; preserve stronger configured checks. |

## 5. Optimization review

### Keep policy at its owner

| Material | Owner | Go-extension treatment |
| --- | --- | --- |
| Authority, approvals, allowed writes, generated files | AGENTS and roles | Refer to framework policy; retire C-3. |
| Approved architecture/import boundaries | Architecture document, spec review, configured `depguard` | Retire C-16; do not mirror an import graph in prose. |
| Acceptance coverage, minimal implementation scope, first failing test | Implementer/work-spec procedure | Do not add another generic testing strategy checklist. |
| Task/spec IDs, revision evidence, review write restrictions | AGENTS/report/reviewer | Reuse existing contracts. |
| Test-first workflow | Framework implementer and optional `tdd` practice | Remove the Go-specific table mandate. |
| Go-specific lifecycle and semantic choices | Go extension | Keep actionable, scoped instructions. |
| Supporting evidence and version inventory | This research document | Never load during routine implementation/review. |

### Put mechanical work in existing tooling

This repository's Taskfile and `.golangci.yml` already cover formatting, naming
checks, import boundaries, error linting, test helpers, modernization, race tests,
and vulnerabilities. They are implementation-project configuration, **not** tools
automatically available in every project adopting the extension. Preserve the
portable fallback and discover each project's actual commands.

| Concern | Prefer | Human/agent judgment still needed |
| --- | --- | --- |
| Layout/import formatting | Configured formatter; fallback `gofmt -l` | None for mechanical layout; retire C-2. |
| Build/import cycles | Compiler/build | Package responsibility and dependency direction. |
| Version/API compatibility | Compiler, `stdversion`, supported-toolchain CI | Build-tag coverage and documented support matrix. |
| Suspicious constructs/copying locks/cancel leaks | Configured vet/static analysis | General channel lifecycle, error discards, and complete cleanup are not proved. |
| Error wrapping syntax | Configured `errorlint` | Whether a cause should be public. |
| Names/comment form/helper markers | Existing linters | Whether names, documentation, and failure reports are useful. |
| Races | Existing `-race` tests | Unexecuted paths and logical concurrency errors. |
| Vulnerabilities | Existing `govulncheck` command | Suitability and maintenance cost of the dependency. |
| Modern APIs | Configured modernization checks | Avoid churn or unsupported target versions. |

Do not install more linters or copy this repository's complexity/interface limits
into framework rules. No coverage percentage, function-size cap, universal
`t.Parallel`, mandatory mock generator, or forced framework architecture is introduced.

### Context cost and verification limits

Measure the files actually loaded, not just the entrypoint. Report word/byte counts
as transparent proxies; actual tokens depend on the target model/tokenizer.
The previous catalog skill was 650 words / 4,204 bytes before installation rendering;
its default installed form omits the inactive `With tdd` section, so the actual
baseline is smaller. The following counts include the installation provenance
line, modeled with decision ID `P/D-1`, and exclude the inactive `With tdd` section
from the current baseline. Byte counts use UTF-8; longer project decision IDs
slightly increase both entrypoints.

| Loaded Go context | Words | Bytes | Byte difference from current baseline |
| --- | ---: | ---: | ---: |
| Previous installed Go skill, without `tdd` | 630 | 4,081 | baseline |
| Adopted entrypoint only | 368 | 2,565 | 37% less |
| Adopted entrypoint + testing | 555 | 3,900 | 4% less |
| Adopted entrypoint + design + testing | 795 | 5,572 | 37% more |
| Adopted entire runtime corpus | 1,148 | 8,060 | 98% more |

The useful saving is eliminating unrelated topics for narrow tasks.
Design work with tests and cross-cutting changes can cost more
because they gain additional explicit guidance. Treat that as a tradeoff, not a
universal reduction in task context.

The adopted corpus is intentionally larger than the previous skill: it
adds missing design and testing decisions while making them conditional. A
cross-cutting change can load more text than before. This design does **not**
claim universal token savings, an optimal rule count, or demonstrated improvement
in generated-code quality.

### Quality gate

- Every retained/new operational ID maps to evidence above; retired IDs are not reused.
- Exceptions preserve version-gated implementations, protocol interfaces, direct
  errors, channels without closure, lock-held serialization, streams, and standalone tests.
- Files have no cross-topic mandatory read chains; their only common dependency
  is the Go entrypoint and framework instructions already loaded by the role.
- Check commands do not claim to prove semantic rules; project commands supersede
  fallback execution without weakening required coverage.
- Catalog adoption applies to new installations from this source. Existing
  installations retain their approved framework version until deliberately updated.

### Validation performed

Checked local Markdown links, source-reference definitions, heading separation,
balanced fences, trailing whitespace, final newlines, unique rule IDs, retired-ID
exclusion, trigger destinations, and one-to-one evidence coverage. All 38 adopted
rules have matrix entries, and all four topic files have entrypoint routes.

The bundled skill validator could not start because its Python environment lacks
`PyYAML` (`ModuleNotFoundError: No module named 'yaml'`). The proposal's simple
`name`/`description` frontmatter was checked directly instead; no dependency was
installed. The initial research pass changed no Go code or active embedded assets.
Adoption checks passed:

- Baseline: `go test ./internal/project ./internal/cli` passed before edits.
- Updated packaging/rendering tests failed against the previous catalog, then
  passed after adoption. They cover topic installation and manifest ownership,
  Go rendering independently of `tdd`, both runtime stubs, and initialization/reruns.
- `task check` passed: module tidy check, configured lint (zero issues), full
  race-enabled tests, and vulnerability checking (none found).
- `npm run build` and `npm run check` passed in `site/` after updating the
  extension example; the site checker validated all 23 HTML pages.
- Markdown links, rule IDs/evidence coverage, loading routes, and whitespace
  were rechecked against the adopted catalog paths.

The Go checks required access to the existing build cache outside the filesystem
sandbox and passed after that access was approved. No dependencies or release
versions were changed.

## Sources

[releases]: https://go.dev/doc/devel/release
[gomod]: https://go.dev/doc/modules/gomod-ref#go
[effective]: https://go.dev/doc/effective_go
[crc]: https://go.dev/wiki/CodeReviewComments
[google-guide]: https://google.github.io/styleguide/go/guide
[google-decisions]: https://google.github.io/styleguide/go/decisions
[google-best]: https://google.github.io/styleguide/go/best-practices
[stdversion]: https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/stdversion
[errors]: https://go.dev/blog/go1.13-errors
[context]: https://pkg.go.dev/context
[spec-close]: https://go.dev/ref/spec#Close
[memory]: https://go.dev/ref/mem
[defer]: https://go.dev/blog/defer-panic-and-recover
[transactions]: https://go.dev/doc/database/execute-transactions
[querying]: https://go.dev/doc/database/querying
[http]: https://pkg.go.dev/net/http
[race]: https://go.dev/doc/articles/race_detector
[test-comments]: https://go.dev/wiki/TestComments
[vet]: https://pkg.go.dev/cmd/vet
[generics]: https://go.dev/blog/when-generics
[supply-chain]: https://go.dev/blog/supply-chain
[layout]: https://go.dev/doc/modules/layout
[go122]: https://go.dev/doc/go1.22
[testing]: https://pkg.go.dev/testing
[go125]: https://go.dev/doc/go1.25
[go127]: https://go.dev/doc/go1.27
[context-engineering]: https://www.anthropic.com/research/effective-context-engineering-for-ai-agents
[package-names]: https://go.dev/blog/package-names
[compatibility]: https://go.dev/blog/module-compatibility
[doc-comments]: https://go.dev/doc/comment
[diagnostics]: https://go.dev/doc/diagnostics
[pipelines]: https://go.dev/blog/pipelines
[sync]: https://pkg.go.dev/sync
[synctest]: https://pkg.go.dev/testing/synctest
[fuzz]: https://go.dev/doc/security/fuzz/
