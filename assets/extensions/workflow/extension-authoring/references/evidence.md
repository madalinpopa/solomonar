# Evidence for extension authoring

Reviewed 2026-10-24. This is the single source map for the authoring method.
Load it during extension research, then open only sources relevant to the current
question. It is supporting research, not additional mandatory runtime rules.
The stage procedures own how to apply it; results for each authored extension
belong in that extension's research record.

## Findings and limits from external sources

Evidence classes: **empirical** means an observed result under the study's
conditions; **guidance** means maintainer/vendor advice or a documented mechanism;
**policy** means a Solomonar decision. None alone proves agent gains here.

| ID | Evidence and concrete result | Authoring decision and limit |
| --- | --- | --- |
| E-1 | **Empirical.** [Liu et al., Lost in the Middle](https://arxiv.org/abs/2307.03172v3), TACL publication 2024: multi-document QA and key-value retrieval often perform best when relevant information is near either end and worse in the middle. | Keep scope and essential constraints easy to locate; test placement in realistic contexts. These tasks and older models do not establish the optimum layout of every current agent prompt. |
| E-2 | **Empirical preprint.** [Gupte et al., GM-Extract](https://arxiv.org/abs/2511.13900v1), 2025: experiments on 7–8B models did not consistently show a distinct U-shaped curve; mitigation effects depended on setup and could be negative. | Treat the U effect as a testable risk. Do not mandate duplicated beginning/end instructions or claim a universal fix. Generalization to other model sizes and tasks is limited. |
| E-3 | **Empirical technical report.** [Chroma, Context Rot](https://www.trychroma.com/research/context-rot), 2025-07-14: evaluation across 18 models found nonuniform effects of input length and sensitivity to distractors and semantic matching. | Evaluate long and distracting contexts, not just lexical retrieval or short ideal examples. Synthetic retrieval/copying results are not a coding-agent acceptance test. |
| E-4 | **Empirical preprint.** [Gloaguen et al., Evaluating AGENTS.md](https://arxiv.org/abs/2602.11988v3), revised 2026-09-29: context files did not generally improve task success in their repository-task evaluations and raised average inference cost by over 20%; repository overviews were unhelpful. | Require useful task-specific information and a baseline. This does not establish that every context file or specialized extension harms performance. |
| E-5 | **Empirical preprint.** [SkillsBench v4](https://arxiv.org/abs/2602.12670v4), 2026-06-14: curated skills raised mean pass rate from 33.9% to 50.5% across 87 tasks, eight domains, and 18 model/harness configurations. Focused skills outperformed larger bundles in that study. | Skills can help when they supply relevant expertise; measure the proposed extension. Neither the aggregate gain nor its module-count finding is a universal target or structural limit. |
| E-6 | **Guidance and reported internal evaluation.** [OpenAI current model guidance](https://developers.openai.com/api/docs/guides/latest-model#favor-leaner-prompts), checked 2026-10-24: remove instruction groups incrementally and evaluate; a sample of internal coding-agent runs favored leaner prompts. | State an instruction once, then test removals. Keep examples and constraints that encode requirements or correct observed failures. The documentation's gains are directional, not a Solomonar forecast; the page is mutable. |
| E-7 | **Guidance / mechanism.** [OpenAI Build skills](https://learn.chatgpt.com/docs/build-skills), checked 2026-10-24, describes metadata discovery followed by loading the selected skill body. [Anthropic Agent Skills](https://www.anthropic.com/research/equipping-agents-for-the-real-world-with-agent-skills), 2025-10-16, describes further conditional references and recommends evaluating real tasks. | Make activation metadata specific; put conditional detail behind explicit triggers. Metadata still costs context. These runtime mechanisms do not establish that Solomonar's generated adapters are discovered correctly without installation checks. |
| E-8 | **Engineering guidance.** [Anthropic context engineering](https://www.anthropic.com/research/effective-context-engineering-for-ai-agents), 2025-09-29, recommends targeted retrieval and persistent notes, while acknowledging runtime exploration can be slower than precomputed retrieval. | Use small handoffs and direct references; avoid loading unrelated research during runtime work. Keep critical trigger information visible. More fragmentation can add tool calls and missed-context risks. |
| E-9 | **Maintainer guidance.** [Agent Skills best practices](https://agentskills.io/skill-creation/best-practices), checked 2026-10-24: ground instructions in actual expertise, omit already-known background, calibrate prescription to fragility, and refine using execution traces. | Use concrete actions, defaults, relevant exceptions, and examples that resolve ambiguity. These are design recommendations, not controlled proof that a particular bullet style improves all models. |
| E-10 | **Model-specific guidance.** [OpenAI reasoning best practices](https://developers.openai.com/api/docs/guides/reasoning-best-practices), checked 2026-10-24, recommends simple direct prompts, clear delimiters, and avoiding unnecessary chain-of-thought requests for reasoning models. | Ask for checkable conclusions, evidence, and concise rationale. Do not demand hidden reasoning or prescribe an elaborate thought ritual. Adapt if the actual model/runtime differs. |
| E-11 | **Performance guidance.** [OpenAI latency optimization](https://developers.openai.com/api/docs/guides/latency-optimization#use-fewer-input-tokens), checked 2026-10-24: halving input can yield only a 1–5% latency improvement; output generation and request overhead also matter. | Measure elapsed time, output, and calls independently of input size. This is a heuristic, not a guaranteed latency curve for local tools or every workload. |
| E-12 | **Runtime mechanism.** [OpenAI prompt caching](https://developers.openai.com/api/docs/guides/prompt-caching), checked 2026-10-24: reuse requires matching rendered prefixes and eligible cache boundaries/settings; stable instructions can help. | Record actual cache/cost data when available. Cache reuse does not remove instructions from the logical context or validate their quality. Markdown alone does not control cache configuration or prompt assembly. |
| E-13 | **Evaluation guidance.** [OpenAI evaluation best practices](https://developers.openai.com/api/docs/guides/evaluation-best-practices), checked 2026-10-24: use task-specific tests, typical/edge/adversarial cases, calibrated grading, and human feedback; avoid judging only by apparent plausibility. | Define expected outcomes before drafting, use matched baselines and held-out cases, and separate packaging from behavior. No mandatory hosted evaluation service or new harness is needed. |

E-4 and E-5 examine different interventions and task distributions. Their results
support testing whether a specific extension contributes useful expertise; they
do not justify choosing one universal verdict about instructional files.

## Advice intentionally excluded

- Fixed universal token, file-length, or rule-count limits. Published guidelines
  and benchmark-specific patterns are not Solomonar accuracy thresholds.
- Turning every idiom or style preference into a blocking `must`.
- Loading the whole framework, catalog, source corpus, or research on every task.
- Splitting every paragraph into a separate reference file. Loading overhead and
  trigger recognition matter alongside entrypoint size.
- Applying repository-overview findings to conclude that useful domain skills
  cannot help, or applying SkillsBench gains as a promise for every new skill.
- Treating repeated instructions, role personas, or chain-of-thought requests as
  established universal accuracy improvements.
- Claiming a shorter file is faster, counting words as measured tokens, or
  treating a successful build/installation as evidence of better agent behavior.

