package cli

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
)

// planned is one change per path, all with the same action and owner.
func planned(action project.Action, owner string, paths ...string) []project.Change {
	changes := make([]project.Change, 0, len(paths))
	for _, p := range paths {
		changes = append(changes, project.Change{Path: p, Action: action, Owner: owner})
	}
	return changes
}

func extension(name, group string) project.CatalogExtension {
	return project.CatalogExtension{Name: name, Kind: "skill", Group: group}
}

func uc1Run() initRun {
	const c = project.ActionCreate
	var changes []project.Change
	for _, part := range [][]project.Change{
		planned(c, "core", "docs/solomonar/agents/adversarial-reviewer.md", "docs/solomonar/agents/implementer.md",
			"docs/solomonar/agents/orchestrator.md", "docs/solomonar/rules.md", "docs/solomonar/skills/solomonar/SKILL.md",
			"docs/solomonar/skills/spec/SKILL.md", "docs/solomonar/skills/spec/template.md"),
		planned(c, "core+notes", "AGENTS.md"),
		planned(c, "seed", "docs/README.md"),
		planned(c, "ext:go", "docs/solomonar/skills/go/SKILL.md"),
		planned(c, "ext:htmx", "docs/solomonar/skills/htmx/SKILL.md"),
		planned(c, "ext:templ", "docs/solomonar/skills/templ/SKILL.md"),
		planned(c, "runtime:claude", "CLAUDE.md", ".claude/skills/go/SKILL.md", ".claude/skills/htmx/SKILL.md",
			".claude/skills/solomonar/SKILL.md", ".claude/skills/spec/SKILL.md", ".claude/skills/templ/SKILL.md",
			".claude/agents/adversarial-reviewer.md", ".claude/agents/implementer.md", ".claude/agents/orchestrator.md"),
		planned(c, "runtime:codex", ".agents/skills/go/SKILL.md", ".agents/skills/htmx/SKILL.md",
			".agents/skills/solomonar/SKILL.md", ".agents/skills/solomonar/agents/openai.yaml",
			".agents/skills/spec/SKILL.md", ".agents/skills/templ/SKILL.md", ".codex/agents/adversarial-reviewer.toml",
			".codex/agents/implementer.toml", ".codex/agents/orchestrator.toml"),
		planned(c, "decision", "docs/decisions.md", "docs/stack.md"),
		planned(c, "manifest", ".solomonar/manifest.json"),
	} {
		changes = append(changes, part...)
	}
	return initRun{
		framework: "0.1.0", mainBranch: "main", plan: project.Plan{Changes: changes},
		runtimes: []string{"claude", "codex"}, models: map[string]string{"claude": "opus"},
		extensions: []project.CatalogExtension{
			extension("go", "language"), extension("htmx", "frontend"), extension("templ", "frontend"),
		},
		renderedWith: map[string][]string{"htmx": {"templ"}, "templ": {"htmx"}},
		decisions: []project.Decision{
			{ID: "P/D-1", Group: "language", Selected: []string{"go"}},
			{ID: "P/D-2", Group: "frontend", Selected: []string{"htmx", "templ"}},
		},
	}
}

const uc1Plan = "  framework 0.1.0\n" +
	"    create     docs/solomonar/agents/ (3 roles)\n" +
	"    create     docs/solomonar/rules.md\n" +
	"    create     docs/solomonar/skills/solomonar/ (SKILL.md)\n" +
	"    create     docs/solomonar/skills/spec/ (SKILL.md, template.md)\n" +
	"    create     AGENTS.md\n" +
	"    create     docs/README.md\n" +
	"  extensions\n" +
	"    create     docs/solomonar/skills/go/SKILL.md\n" +
	"    create     docs/solomonar/skills/htmx/SKILL.md   rendered with: templ\n" +
	"    create     docs/solomonar/skills/templ/SKILL.md  rendered with: htmx\n" +
	"  decisions (approved, selected by you)\n" +
	"    create     docs/decisions.md  P/D-1 language: go · P/D-2 frontend: htmx, templ\n" +
	"    create     docs/stack.md      language: go (P/D-1) · frontend: htmx, templ (P/D-2)\n" +
	"  runtime claude\n" +
	"    create     CLAUDE.md\n" +
	"    create     .claude/skills/ (5 skill stubs: 2 core, 3 extensions)\n" +
	"    create     .claude/agents/ (3 role stubs; adversarial-reviewer model opus)\n" +
	"  runtime codex\n" +
	"    create     .agents/skills/ (5 skill stubs: 2 core, 3 extensions; solomonar/agents/openai.yaml)\n" +
	"    create     .codex/agents/ (3 role stubs)\n" +
	"  registry\n" +
	"    create     refs/heads/workflow-state with an empty state.json\n" +
	"  manifest\n" +
	"    create     .solomonar/manifest.json\n"

func TestGroupedPlanSummarizesUC1(t *testing.T) {
	t.Parallel()

	assert.Equal(t, uc1Plan, uc1Run().groupedPlan())
}

func TestGroupedPlanKeepsSetsWithReasonsOrMixedActions(t *testing.T) {
	t.Parallel()

	const claudeReason = "exists; add this line so Claude Code loads the framework:  @AGENTS.md"
	changes := []project.Change{
		{Path: "docs/solomonar/agents/implementer.md", Action: project.ActionUnchanged, Owner: "core"},
		{Path: "docs/solomonar/agents/orchestrator.md", Action: project.ActionUpdate, Owner: "core"},
		{Path: "docs/solomonar/skills/spec/SKILL.md", Action: project.ActionKeep, Owner: "core", Reason: "kept by --keep"},
		{Path: "docs/solomonar/skills/spec/template.md", Action: project.ActionKeep, Owner: "core"},
		{Path: "docs/solomonar/skills/work-spec/SKILL.md", Action: project.ActionUnchanged, Owner: "core"},
		{Path: "AGENTS.md", Action: project.ActionUnchanged, Owner: "core+notes"},
		{Path: "CLAUDE.md", Action: project.ActionSkip, Owner: "runtime:claude", Reason: claudeReason},
		{Path: ".claude/skills/go/SKILL.md", Action: project.ActionCreate, Owner: "runtime:claude"},
		{Path: ".claude/skills/spec/SKILL.md", Action: project.ActionUnchanged, Owner: "runtime:claude"},
		{Path: ".claude/agents/implementer.md", Action: project.ActionUnchanged, Owner: "runtime:claude"},
		{Path: "docs/decisions.md", Action: project.ActionUnchanged, Owner: "decision"},
		{Path: ".solomonar/manifest.json", Action: project.ActionUpdate, Owner: "manifest"},
	}
	r := initRun{
		framework: "0.1.0", tip: "abc", plan: project.Plan{Changes: changes}, runtimes: []string{"claude"},
		extensions: []project.CatalogExtension{extension("go", "language")},
	}

	assert.Equal(t, "  framework 0.1.0\n"+
		"    unchanged  docs/solomonar/agents/implementer.md\n"+
		"    update     docs/solomonar/agents/orchestrator.md\n"+
		"    keep       docs/solomonar/skills/spec/SKILL.md  kept by --keep\n"+
		"    keep       docs/solomonar/skills/spec/template.md\n"+
		"    unchanged  docs/solomonar/skills/work-spec/ (SKILL.md)\n"+
		"    unchanged  AGENTS.md\n"+
		"  decisions (approved, selected by you)\n"+
		"    unchanged  docs/decisions.md\n"+
		"  runtime claude\n"+
		"    skip       CLAUDE.md  "+claudeReason+"\n"+
		"    create     .claude/skills/go/SKILL.md\n"+
		"    unchanged  .claude/skills/spec/SKILL.md\n"+
		"    unchanged  .claude/agents/ (1 role stub)\n"+
		"  registry\n"+
		"    unchanged  refs/heads/workflow-state\n"+
		"  manifest\n"+
		"    update     .solomonar/manifest.json\n", r.groupedPlan())
}

func TestGroupedPlanOmitsEmptyGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  initRun
		want string
	}{
		{
			name: "no runtimes",
			run: initRun{framework: "0.1.0", plan: project.Plan{Changes: slices.Concat(
				planned(project.ActionCreate, "core+notes", "AGENTS.md"),
				planned(project.ActionCreate, "manifest", ".solomonar/manifest.json"),
			)}},
			want: "  framework 0.1.0\n    create     AGENTS.md\n" +
				"  registry\n    create     refs/heads/workflow-state with an empty state.json\n" +
				"  manifest\n    create     .solomonar/manifest.json\n",
		},
		{
			name: "git init",
			run: initRun{framework: "0.1.0", mainBranch: "trunk", gitInit: true, plan: project.Plan{
				Changes: planned(project.ActionCreate, "manifest", ".solomonar/manifest.json"),
			}},
			want: "  git\n    init       -b trunk\n" +
				"  registry\n    create     refs/heads/workflow-state with an empty state.json\n" +
				"  manifest\n    create     .solomonar/manifest.json\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.run.groupedPlan())
		})
	}
}

func TestInteractiveInitShowsTheGroupedPlan(t *testing.T) {
	dir := committedRepo(t)
	gittest.NoGH(t)
	assets := catalogAssets(t)
	assets["extensions/frontend/templ/SKILL.md"] = &fstest.MapFile{Data: []byte(
		"---\nname: templ\ndescription: D\n---\n\n# templ\n\n## With htmx\n<!-- solomonar:with htmx -->\nSwap.\n")}

	got := runInteractive(t, assets, dir, "\ny\n", "init", "--name", "ledger", "--objective", "o",
		"--runtime", "claude,codex", "--adversarial-model", "claude=opus", "--ext", "go,htmx,templ")

	require.Equal(t, 0, got.exit, got.stderr)
	_, plan, ok := strings.Cut(got.stdout, "Plan\n")
	require.True(t, ok, got.stdout)
	plan, _, ok = strings.Cut(plan, "Proceed? [Y/n] ")
	require.True(t, ok, got.stdout)
	assert.Equal(t,
		"  framework 0.1.0\n"+
			"    create     docs/solomonar/agents/ (6 roles)\n"+
			"    create     docs/solomonar/report.md\n"+
			"    create     docs/solomonar/reviewer.md\n"+
			"    create     docs/solomonar/rules.md\n"+
			"    create     docs/solomonar/skills/prototype/ (SKILL.md, template.md)\n"+
			"    create     docs/solomonar/skills/solomonar/ (SKILL.md)\n"+
			"    create     docs/solomonar/skills/spec/ (SKILL.md, template.md)\n"+
			"    create     docs/solomonar/skills/work-spec/ (SKILL.md, template.md)\n"+
			"    create     AGENTS.md\n"+
			"    create     docs/README.md\n"+
			"  extensions\n"+
			"    create     docs/solomonar/skills/go/SKILL.md\n"+
			"    create     docs/solomonar/skills/htmx/SKILL.md\n"+
			"    create     docs/solomonar/skills/templ/SKILL.md  rendered with: htmx\n"+
			"  decisions (approved, selected by you)\n"+
			"    create     docs/decisions.md  P/D-1 language: go · P/D-2 frontend: htmx, templ\n"+
			"    create     docs/stack.md      language: go (P/D-1) · frontend: htmx, templ (P/D-2)\n"+
			"  runtime claude\n"+
			"    create     CLAUDE.md\n"+
			"    create     .claude/skills/ (7 skill stubs: 4 core, 3 extensions)\n"+
			"    create     .claude/agents/ (5 role stubs; adversarial-reviewer model opus)\n"+
			"  runtime codex\n"+
			"    create     .agents/skills/ (7 skill stubs: 4 core, 3 extensions; solomonar/agents/openai.yaml)\n"+
			"    create     .codex/agents/ (5 role stubs)\n"+
			"  registry\n"+
			"    create     refs/heads/workflow-state with an empty state.json\n"+
			"  manifest\n"+
			"    create     .solomonar/manifest.json\n", plan)
	assert.Contains(t, got.stdout, "Proceed? [Y/n] \nInitialized solomonar 0.1.0 in "+dir+"\n")
}
