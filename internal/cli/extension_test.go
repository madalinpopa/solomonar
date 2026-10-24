package cli

import (
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
)

const (
	moneyJSON = `{"name": "money", "kind": "skill", "group": "backend", "title": "Money rules", ` +
		`"applies_when": "A change handles monetary amounts", "paths": ["**/money/*.go"]}`
	moneySkill   = "---\nname: money\ndescription: Use for monetary amounts.\n---\n# Money\n"
	reviewerJSON = `{"name": "a11y-reviewer", "kind": "agent", "group": "review", "title": "Accessibility"}`
)

// storeRepo is a repository without a manifest, with go.mod, a valid and an invalid project extension.
func storeRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.NewRepo(t)
	writeFile(t, dir, "go.mod", "module example.com/ledger\n")
	writeFile(t, dir, "docs/solomonar/extensions/money/extension.json", moneyJSON)
	writeFile(t, dir, "docs/solomonar/extensions/money/SKILL.md", moneySkill)
	writeFile(t, dir, "docs/solomonar/extensions/a11y-reviewer/extension.json", reviewerJSON)
	writeFile(t, dir, "docs/solomonar/extensions/a11y-reviewer/a11y-reviewer.md", "# a11y\n")
	return dir
}

func TestExtensionListWithoutManifest(t *testing.T) {
	t.Parallel()

	got := runInRepo(t, storeRepo(t), "extension", "list")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, ""+
		"GROUP     NAME                 KIND   SOURCE   STATUS                            DECISION  SUGGESTED  APPLIES WHEN\n"+
		"language  go                   skill  catalog  available                         -         go.mod     A change touches Go code (*.go, go.mod, go.sum)\n"+
		"backend   money                skill  project  available                         -         -          A change handles monetary amounts\n"+
		"workflow  extension-authoring  skill  catalog  available                         -         default    A request creates, changes, or validates a Solomonar extension skill, its rules, or a specialist reviewer\n"+
		"review    a11y-reviewer        agent  project  invalid: applies_when is missing  -         -          -\n",
		got.stdout)
}

func TestExtensionListAliasIsTheSameCommand(t *testing.T) {
	t.Parallel()

	dir := storeRepo(t)

	full := runInRepo(t, dir, "--json", "extension", "list")
	alias := runInRepo(t, dir, "--json", "ext", "list")

	require.Equal(t, 0, full.exit, full.stderr)
	assert.Contains(t, full.stdout, `"command": "extension list"`)
	assert.Equal(t, full.stdout, alias.stdout)
}

func TestExtensionListShowsInstalledExtensions(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	init := runInRepo(t, dir, "init", "--runtime", "none", "--ext", "go", "--yes")
	require.Equal(t, 0, init.exit, init.stderr)

	got := runInRepo(t, dir, "extension", "list", "--installed")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, ""+
		"GROUP     NAME  KIND   SOURCE   STATUS     DECISION  SUGGESTED  APPLIES WHEN\n"+
		"language  go    skill  catalog  installed  P/D-1     -          A change touches Go code (*.go, go.mod, go.sum)\n",
		got.stdout)
}

func TestExtensionListFilters(t *testing.T) {
	t.Parallel()

	dir := storeRepo(t)
	tests := []struct {
		name  string
		args  []string
		names []string
	}{
		{name: "group", args: []string{"--group", "backend"}, names: []string{"money"}},
		{name: "kind", args: []string{"--kind", "agent"}, names: []string{"a11y-reviewer"}},
		{name: "installed", args: []string{"--installed"}, names: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := runInRepo(t, dir, append([]string{"--json", "extension", "list"}, tt.args...)...)

			require.Equal(t, 0, got.exit, got.stderr)
			var envelope struct {
				Data struct {
					Extensions []struct {
						Name string `json:"name"`
					} `json:"extensions"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal([]byte(got.stdout), &envelope))
			names := []string{}
			for _, e := range envelope.Data.Extensions {
				names = append(names, e.Name)
			}
			assert.Equal(t, tt.names, names)
		})
	}
}

func TestExtensionListEmptyFilterPrintsTheHeader(t *testing.T) {
	t.Parallel()

	got := runInRepo(t, storeRepo(t), "extension", "list", "--installed")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "GROUP  NAME  KIND  SOURCE  STATUS  DECISION  SUGGESTED  APPLIES WHEN\n", got.stdout)
}

func TestExtensionListJSON(t *testing.T) {
	t.Parallel()

	got := runInRepo(t, storeRepo(t), "--json", "extension", "list", "--group", "backend")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "extension list",
		"data": {"extensions": [{
			"name": "money", "group": "backend", "kind": "skill", "source": "project", "status": "available",
			"title": "Money rules", "applies_when": "A change handles monetary amounts", "paths": ["**/money/*.go"],
			"requires": [], "conflicts": [], "invocation": "model", "path": "docs/solomonar/extensions/money"
		}]},
		"warnings": []
	}`, got.stdout)
}

func TestExtensionListJSONInvalidAndSuggested(t *testing.T) {
	t.Parallel()

	got := runInRepo(t, storeRepo(t), "--json", "extension", "list", "--group", "review")
	lang := runInRepo(t, storeRepo(t), "--json", "extension", "list", "--group", "language")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, `"status": "invalid"`)
	assert.Contains(t, got.stdout, `"reason": "applies_when is missing"`)
	assert.Contains(t, lang.stdout, `"suggested": "go.mod"`)
	assert.NotContains(t, lang.stdout, `"path"`)
}

func TestExtensionListMarksACatalogNameInTheStore(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	writeFile(t, dir, "docs/solomonar/extensions/go/extension.json", `{"name": "go", "kind": "skill", `+
		`"group": "language", "title": "Our Go", "applies_when": "Go code"}`)
	writeFile(t, dir, "docs/solomonar/extensions/go/SKILL.md", "---\nname: go\ndescription: Go.\n---\n")

	got := runInRepo(t, dir, "extension", "list", "--group", "language")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, ""+
		"GROUP     NAME  KIND   SOURCE   STATUS                                          DECISION  SUGGESTED  APPLIES WHEN\n"+
		"language  go    skill  catalog  available                                       -         -          A change touches Go code (*.go, go.mod, go.sum)\n"+
		"language  go    skill  project  invalid: name \"go\" is also a catalog extension  -         -          Go code\n",
		got.stdout)
}

func TestExtensionListRejections(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	tests := []struct {
		name string
		dir  string
		args []string
		exit int
		want string
	}{
		{
			name: "unknown group", dir: storeRepo(t), args: []string{"--group", "nope"}, exit: 2,
			want: "error: extension list rejected (unknown_group)\n" +
				"  check: --group names a group\n" +
				"  found: \"nope\" is not a group; known: language, architecture, frontend, backend, persistence, " +
				"testing, operations, workflow, review\n" +
				"  fix:   pass --group with a known group\n" +
				"nothing was written\n",
		},
		{
			name: "kind", dir: storeRepo(t), args: []string{"--kind", "rule"}, exit: 2,
			want: "error: extension list rejected (invalid_value)\n" +
				"  check: --kind is skill or agent\n" +
				"  found: \"rule\" is not skill or agent\n" +
				"  fix:   pass --kind skill or --kind agent\n" +
				"nothing was written\n",
		},
		{
			name: "argument", dir: storeRepo(t), args: []string{"go"}, exit: 2,
			want: "error: unexpected argument \"go\" (invalid_usage)\n  fix:   run solomonar help\nnothing was written\n",
		},
		{
			name: "outside a repository", dir: outside, exit: 3,
			want: "error: " + outside + " is not a Git repository (not_a_repository)\n" +
				"  fix:   run inside a Git repository or pass -C <path>\nnothing was written\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := runInRepo(t, tt.dir, append([]string{"extension", "list"}, tt.args...)...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Equal(t, tt.want, got.stderr)
		})
	}
}

func TestExtensionListAcceptsAnOlderFramework(t *testing.T) {
	t.Parallel()

	dir := initializedRepo(t)
	m := readTestManifest(t, dir)
	m.FrameworkVersion = "0.1.7"
	writeTestManifest(t, dir, m)

	got := runInRepo(t, dir, "extension", "list")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "extension-authoring")
}

const (
	moneyReviewerJSON = `{"name": "money", "kind": "agent", "group": "review", "title": "Money reviewer", ` +
		`"applies_when": "A change handles monetary amounts"}`
	moneyReviewer = "---\nname: money\ndescription: Reviews monetary amounts.\n---\n# Money reviewer\n"
	moneyCommit   = `git add AGENTS.md .claude .agents .codex .solomonar docs && ` +
		`git commit -m "chore: add solomonar extension money"`
)

// moneyRepo is a current project with the claude and codex runtimes, no extension, and the project
// review agent money in the store.
func moneyRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.NewRepo(t)
	got := runInRepo(t, dir, "init", "--runtime", "claude,codex", "--ext", "none", "--yes")
	require.Equal(t, 0, got.exit, got.stderr)
	writeFile(t, dir, "docs/solomonar/extensions/money/extension.json", moneyReviewerJSON)
	writeFile(t, dir, "docs/solomonar/extensions/money/money.md", moneyReviewer)
	return dir
}

func addIn(t *testing.T, dir string, args ...string) result {
	t.Helper()
	return runInRepo(t, dir, append([]string{"extension", "add"}, args...)...)
}

func TestExtensionAddActivatesAProjectAgent(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)
	stored := tree(t, filepath.Join(dir, "docs/solomonar/extensions/money"))

	got := addIn(t, dir, "money", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Regexp(t, `^Added extension money \(review agent, project\) in `+regexp.QuoteMeta(dir)+`\n`+
		`  2 files created, 2 updated, \d+ unchanged, 1 skipped\n\nNext:\n  1\. `+regexp.QuoteMeta(moneyCommit)+"\n$", got.stdout)
	assert.Empty(t, got.stderr)
	assert.Contains(t, readTestFile(t, dir, ".claude/agents/money.md"),
		"from `docs/solomonar/extensions/money/money.md`")
	assert.Contains(t, readTestFile(t, dir, ".codex/agents/money.toml"), `description = "Reviews monetary amounts."`)
	assert.Contains(t, readTestFile(t, dir, "AGENTS.md"), "  - `money` (review): A change handles monetary amounts\n")
	assert.Equal(t, stored, tree(t, filepath.Join(dir, "docs/solomonar/extensions/money")))
	m := readTestManifest(t, dir)
	assert.Equal(t, project.Extension{
		State: "installed", Kind: "agent", Group: "review", Source: "path", Path: "docs/solomonar/extensions/money",
	}, m.Extensions["money"])
	for path := range m.Files {
		assert.NotContains(t, path, "docs/solomonar/extensions/money")
	}
}

func TestExtensionAddRerunIsNoOp(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)
	require.Equal(t, 0, addIn(t, dir, "money", "--yes").exit)
	before, tip := tree(t, dir), refTip(t, dir)

	got := addIn(t, dir, "money")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Regexp(t, `^Already installed: money; nothing to do.\n  \d+ files unchanged, 1 skipped\n$`, got.stdout)
	assertNothingWritten(t, dir, before, tip)
	rerun := runInRepo(t, dir, "init", "--yes")
	require.Equal(t, 0, rerun.exit, rerun.stderr)
	assert.Contains(t, rerun.stdout, "nothing to do")
}

func TestExtensionAddCopiesACatalogWorkflowExtension(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)

	got := addIn(t, dir, "extension-authoring", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout,
		"Added extension extension-authoring (workflow skill, catalog) in "+dir+"\n  "), got.stdout)
	m := readTestManifest(t, dir)
	assert.Equal(t, project.Extension{State: "installed", Kind: "skill", Group: "workflow", Source: "catalog"},
		m.Extensions["extension-authoring"])
	var stored []string
	for path, f := range m.Files {
		if strings.HasPrefix(path, "docs/solomonar/extensions/extension-authoring/") {
			assert.Equal(t, "ext:extension-authoring", f.Owner)
			stored = append(stored, path)
		}
	}
	assert.Len(t, stored, 6)
	assert.Equal(t, readTestFile(t, "../../assets", "extensions/workflow/extension-authoring/SKILL.md"),
		readTestFile(t, dir, "docs/solomonar/extensions/extension-authoring/SKILL.md"))
	assert.Contains(t, readTestFile(t, dir, ".agents/skills/extension-authoring/agents/openai.yaml"),
		"allow_implicit_invocation: false")
}

func TestExtensionAddDryRunWritesNothing(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := addIn(t, dir, "money", "--dry-run")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, "Plan\n  unchanged  docs/solomonar/agents/"), got.stdout)
	assert.Contains(t, got.stdout, "  create     .claude/agents/money.md\n")
	assert.Contains(t, got.stdout, "  update     .solomonar/manifest.json\n")
	assert.Regexp(t, `Would add extension money in `+regexp.QuoteMeta(dir)+`\n  2 files created, 2 updated, \d+ unchanged, 1 skipped\n$`,
		got.stdout)
	assertNothingWritten(t, dir, before, tip)
}

func TestExtensionAddNeedsConfirmation(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)
	before := tree(t, dir)

	got := addIn(t, dir, "money")

	require.Equal(t, 2, got.exit, got.stderr)
	assert.Empty(t, got.stdout)
	assert.Equal(t, "error: extension add needs a confirmation (interactive_required)\n"+
		"  fix:   pass --yes to accept the plan, or --dry-run to print it\n"+
		"nothing was written\n", got.stderr)
	assert.Equal(t, before, tree(t, dir))
}

func TestExtensionAddConfirmsInTerminal(t *testing.T) {
	t.Parallel()
	const wantPlan = "Plan\n" +
		"  framework " + "%s" + "\n" +
		"    update     AGENTS.md\n" +
		"  runtime claude\n" +
		"    create     .claude/agents/ (1 role stub: 0 core, 1 extension)\n" +
		"  runtime codex\n" +
		"    create     .codex/agents/ (1 role stub: 0 core, 1 extension)\n" +
		"  manifest\n" +
		"    update     .solomonar/manifest.json\n" +
		"Proceed? [Y/n] \n"
	tests := []struct {
		name, input, stdout string
		applied             bool
	}{
		{name: "declined", input: "n\n", stdout: "Cancelled; nothing was written.\n"},
		{name: "accepted", input: "\n", stdout: "Added extension money (review agent, project) in ", applied: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := moneyRepo(t)

			got := runInteractive(t, repoAssets(t), dir, tt.input, "extension", "add", "money")

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, fmt.Sprintf(wantPlan, shipped), got.stderr)
			assert.True(t, strings.HasPrefix(got.stdout, tt.stdout), got.stdout)
			_, installed := readTestManifest(t, dir).Extensions["money"]
			assert.Equal(t, tt.applied, installed)
		})
	}
}

func TestExtensionAddJSON(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)

	got := runInRepo(t, dir, "--json", "ext", "add", "money", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	var envelope struct {
		Command string        `json:"command"`
		Data    extensionData `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(got.stdout), &envelope))
	assert.Equal(t, "extension add", envelope.Command)
	assert.Equal(t, dir, envelope.Data.Project)
	assert.Equal(t, []string{"money"}, envelope.Data.Added)
	assert.Empty(t, envelope.Data.Decisions)
	assert.Contains(t, envelope.Data.Files, initFile{Path: ".claude/agents/money.md", Action: "create", Owner: "runtime:claude"})
	assert.Equal(t, []string{moneyCommit}, envelope.Data.Next)
	noOp := runInRepo(t, dir, "--json", "extension", "add", "money")
	require.Equal(t, 0, noOp.exit, noOp.stderr)
	assert.Contains(t, noOp.stdout, `"next": []`)
	assert.Contains(t, noOp.stdout, `"decisions": []`)
	dryRun := runInRepo(t, dir, "--json", "extension", "add", "money", "--dry-run")
	require.Equal(t, 0, dryRun.exit, dryRun.stderr)
	assert.Contains(t, dryRun.stdout, `"dry_run": true`)
	assert.NotContains(t, dryRun.stdout, `"next"`)
}

func TestExtensionAddRejectsAnOlderFramework(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)
	m := readTestManifest(t, dir)
	m.FrameworkVersion = "0.1.7"
	writeTestManifest(t, dir, m)
	before := tree(t, dir)

	got := runCLI(t, assetsAt(t, "0.1.8"), "-C", dir, "extension", "add", "nope", "--yes")

	assert.Equal(t, 3, got.exit)
	assert.Empty(t, got.stdout)
	assert.Equal(t, "error: extension add rejected (framework_outdated)\n"+
		"  check: the project's framework matches this CLI\n"+
		"  found: manifest framework 0.1.7, this CLI ships 0.1.8\n"+
		"  fix:   run solomonar upgrade\n"+
		"nothing was written\n", got.stderr)
	assert.Equal(t, before, tree(t, dir))
}

func TestExtensionAddRejections(t *testing.T) {
	t.Parallel()

	rejected := func(code, check, found, fix string) string {
		return "error: extension add rejected (" + code + ")\n  check: " + check + "\n  found: " + found +
			"\n  fix:   " + fix + "\nnothing was written\n"
	}
	storeFiles := func(name, extra string) map[string]string {
		dir := "docs/solomonar/extensions/" + name + "/"
		return map[string]string{
			dir + "extension.json": `{"name": "` + name + `", "kind": "skill", "group": "review", "title": "T", ` +
				`"applies_when": "A change"` + extra + `}`,
			dir + "SKILL.md": "---\nname: " + name + "\ndescription: Use for " + name + ".\n---\n",
		}
	}
	tests := []struct {
		name  string
		files map[string]string
		bare  bool
		args  []string
		exit  int
		want  string
	}{
		{
			name: "no name", exit: 2,
			want: "error: extension name is required (invalid_usage)\n  fix:   run solomonar help\nnothing was written\n",
		},
		{
			name: "not initialized", bare: true, args: []string{"money"}, exit: 3,
			want: rejected("not_initialized", "the project is initialized", "no .solomonar/manifest.json in %s",
				"run solomonar init"),
		},
		{
			name: "unknown", args: []string{"nope"}, exit: 2,
			want: rejected("unknown_extension", "extension add names a catalog or project extension",
				`"nope" is not in the catalog or docs/solomonar/extensions/; known: extension-authoring, go, money`,
				"pass a name from solomonar extension list"),
		},
		{
			name: "collision", files: storeFiles("go", ""), args: []string{"go"}, exit: 3,
			want: rejected("name_collision", "the name is not both a catalog and a project extension",
				"go is in the catalog and in docs/solomonar/extensions/go",
				"rename or remove docs/solomonar/extensions/go/"),
		},
		{
			name: "core skill name", files: storeFiles("work-spec", ""), args: []string{"work-spec"}, exit: 2,
			want: rejected("invalid_value", "docs/solomonar/extensions/work-spec is a valid extension",
				`name "work-spec" is a core skill`,
				"fix docs/solomonar/extensions/work-spec/ and rerun; the extension-authoring skill validates it"),
		},
		{
			name: "core role name", args: []string{"implementer"}, exit: 2,
			files: map[string]string{
				"docs/solomonar/extensions/implementer/extension.json": `{"name": "implementer", "kind": "agent", ` +
					`"group": "review", "title": "T", "applies_when": "A change"}`,
				"docs/solomonar/extensions/implementer/implementer.md": "---\nname: implementer\n" +
					"description: Implements.\n---\n",
			},
			want: rejected("invalid_value", "docs/solomonar/extensions/implementer is a valid extension",
				`name "implementer" is a core role`,
				"fix docs/solomonar/extensions/implementer/ and rerun; the extension-authoring skill validates it"),
		},
		{
			name: "invalid", files: map[string]string{"docs/solomonar/extensions/bad/SKILL.md": "# bad\n"},
			args: []string{"money", "bad"}, exit: 2,
			want: rejected("invalid_value", "docs/solomonar/extensions/bad is a valid extension",
				"extension.json is missing",
				"fix docs/solomonar/extensions/bad/ and rerun; the extension-authoring skill validates it"),
		},
		{
			name: "requires", files: merge(storeFiles("audit", `, "requires": ["money", "trace"]`), storeFiles("trace", "")),
			args: []string{"audit"}, exit: 3,
			want: rejected("extension_requires", "every required extension is installed",
				"audit requires money, trace", "add them in the same call: solomonar extension add audit money trace"),
		},
		{
			name: "conflict", files: storeFiles("audit", `, "conflicts": ["money"]`),
			args: []string{"money", "audit"}, exit: 3,
			want: rejected("extension_conflict", "no installed extensions conflict", "audit conflicts with money",
				"remove money first, or drop audit"),
		},
		{
			name: "decision for a workflow extension", args: []string{"money", "--decision", "P/D-1"}, exit: 2,
			want: "error: --decision and --rationale apply to stack groups only (invalid_usage)\n" +
				"  fix:   run solomonar help\nnothing was written\n",
		},
		{
			name: "rationale for a workflow extension", args: []string{"money", "--rationale", "why"}, exit: 2,
			want: "error: --decision and --rationale apply to stack groups only (invalid_usage)\n" +
				"  fix:   run solomonar help\nnothing was written\n",
		},
		{
			name: "requires before the decision", files: storeFiles("audit", `, "requires": ["money"]`),
			args: []string{"audit", "go", "--decision", "W-002/D-9"}, exit: 3,
			want: rejected("extension_requires", "every required extension is installed",
				"audit requires money", "add them in the same call: solomonar extension add audit go money"),
		},
		{
			name: "malformed decision", args: []string{"go", "--decision", "D-1"}, exit: 2,
			want: rejected("invalid_value", "--decision is P/D-<n> or <task-ID>/D-<n>",
				`"D-1" is not P/D-<n> or <task-ID>/D-<n>`, "pass the ID of an approved entry in docs/decisions.md"),
		},
		{
			name: "decision not found", args: []string{"go", "--decision", "W-002/D-2"}, exit: 3,
			want: rejected("decision_not_found", "--decision must name an approved entry in docs/decisions.md",
				"W-002/D-2 is not in docs/decisions.md", "record the decision first (rules section 3), then rerun"),
		},
		{
			name: "decision not approved", files: map[string]string{"docs/decisions.md": taskDecisions},
			args: []string{"go", "--decision", "W-002/D-2"}, exit: 3,
			want: rejected("decision_not_approved", "--decision must name an approved entry in docs/decisions.md",
				"W-002/D-2 has status pending", "wait for the user's approval (rules section 3), then rerun"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := gittest.NewRepo(t)
			if !tt.bare {
				dir = moneyRepo(t)
			}
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}
			before := tree(t, dir)

			got := addIn(t, dir, append(tt.args, "--yes")...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			want := tt.want
			if strings.Contains(want, "%s") {
				want = fmt.Sprintf(want, dir)
			}
			assert.Equal(t, want, got.stderr)
			assert.Equal(t, before, tree(t, dir))
		})
	}
}

func merge(parts ...map[string]string) map[string]string {
	all := map[string]string{}
	for _, part := range parts {
		maps.Copy(all, part)
	}
	return all
}

func TestExtensionAddRefreshesAnInstalledExtension(t *testing.T) {
	t.Parallel()
	dir := gittest.NewRepo(t)
	init := runInRepo(t, dir, "init", "--runtime", "claude", "--ext", "go", "--yes")
	require.Equal(t, 0, init.exit, init.stderr)
	stub := readTestFile(t, dir, ".claude/skills/go/SKILL.md")
	require.NoError(t, os.Remove(filepath.Join(dir, ".claude/skills/go/SKILL.md")))

	got := addIn(t, dir, "go", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout,
		"Added extension go (language skill, catalog, under P/D-1) in "+dir+"\n  1 file created, "), got.stdout)
	assert.Equal(t, stub, readTestFile(t, dir, ".claude/skills/go/SKILL.md"))
	assert.Equal(t, "P/D-1", readTestManifest(t, dir).Extensions["go"].Decision)
}

func TestExtensionAddConflictNeedsAResolution(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)
	writeFile(t, dir, ".claude/agents/money.md", "# ours\n")
	before := tree(t, dir)

	got := addIn(t, dir, "money", "--yes")

	assert.Equal(t, 4, got.exit)
	assert.Contains(t, got.stdout, "  conflict   .claude/agents/money.md  exists and is not managed by solomonar\n")
	assert.Contains(t, got.stderr, "error: extension add found 1 conflict (file_conflict)\n")
	assert.Contains(t, got.stderr, "  fix:   rerun with --keep <path> to keep the project's file, or "+
		"--overwrite <path> to replace it\n")
	assert.Equal(t, before, tree(t, dir))
	kept := addIn(t, dir, "money", "--yes", "--keep", ".claude/agents/money.md")
	require.Equal(t, 0, kept.exit, kept.stderr)
	assert.Equal(t, "# ours\n", readTestFile(t, dir, ".claude/agents/money.md"))
}

func TestExtensionAddRejectsAnInstalledProjectExtensionThatNoLongerLoads(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)
	require.Equal(t, 0, addIn(t, dir, "money", "--yes").exit)
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "docs/solomonar/extensions/money")))

	got := addIn(t, dir, "extension-authoring", "--yes")

	assert.Equal(t, 2, got.exit)
	assert.Contains(t, got.stderr, "error: extension add rejected (invalid_value)\n"+
		"  check: docs/solomonar/extensions/money is a valid extension\n"+
		"  found: extension.json is missing\n")
}

const taskDecisions = "# Decisions\n\n## W-002/D-1: Language\n\n- Status: approved\n- Selected option: go\n\n" +
	"## W-002/D-2: Frontend\n\n- Status: pending\n"

// frontendAssets adds the illustrative htmx and templ extensions to the catalog (I section 4.4).
func frontendAssets(t *testing.T) fs.FS {
	t.Helper()
	assets := fstest.MapFS{}
	require.NoError(t, fs.WalkDir(repoAssets(t), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(repoAssets(t), name)
		assets[name] = &fstest.MapFile{Data: data}
		return err
	}))
	for name, requires := range map[string]string{"htmx": "", "templ": `"go"`} {
		dir := "extensions/frontend/" + name + "/"
		assets[dir+"extension.json"] = &fstest.MapFile{Data: []byte(`{"name": "` + name + `", "kind": "skill", ` +
			`"group": "frontend", "title": "` + name + ` rules", "requires": [` + requires + `], ` +
			`"applies_when": "A change touches ` + name + ` code", "paths": ["**/*.` + name + `"]}`)}
		assets[dir+"SKILL.md"] = &fstest.MapFile{Data: []byte("---\nname: " + name + "\ndescription: Use for " +
			name + ".\n---\n# " + name + "\n")}
	}
	return assets
}

func TestExtensionAddRecordsTheGroupDecision(t *testing.T) {
	t.Parallel()
	dir := gittest.NewRepo(t)
	init := runInRepo(t, dir, "init", "--runtime", "claude", "--ext", "none", "--yes")
	require.Equal(t, 0, init.exit, init.stderr)

	got := addIn(t, dir, "go", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout,
		"Added extension go (language skill, catalog, under P/D-1) in "+dir+"\n"), got.stdout)
	assert.Equal(t, "# Decisions\n\n## P/D-1: Programming languages\n\n- Status: approved\n"+
		"- Selected option: go\n- Rejected options: none\n- Scope: project; language extensions `go`\n"+
		"- Rationale: selected by the user with `solomonar extension add`\n- Date: 2026-10-21\n"+
		"- References: `docs/solomonar/extensions/go/SKILL.md`\n", readTestFile(t, dir, "docs/decisions.md"))
	assert.Equal(t, "# Stack\n\n- language: go (P/D-1)\n", readTestFile(t, dir, "docs/stack.md"))
	assert.Contains(t, readTestFile(t, dir, "docs/README.md"),
		"- `docs/decisions.md`: decision entries and their status\n"+
			"- `docs/stack.md`: technologies and practices, with the decisions that adopted them\n")
	assert.Contains(t, readTestFile(t, dir, "AGENTS.md"), "  - `go` (language, P/D-1): A change touches Go code")
	m := readTestManifest(t, dir)
	assert.Equal(t, "P/D-1", m.Extensions["go"].Decision)
	assert.Equal(t, project.File{Owner: "decision", Entries: []string{"P/D-1"}}, m.Files["docs/decisions.md"])
	assert.Equal(t, project.File{Owner: "decision", Entries: []string{"P/D-1"}}, m.Files["docs/stack.md"])
	refresh := addIn(t, dir, "go", "--yes")
	require.Equal(t, 0, refresh.exit, refresh.stderr)
	assert.Contains(t, refresh.stdout, "nothing to do")
}

func TestExtensionAddSupersedesTheGroupDecision(t *testing.T) {
	t.Parallel()
	assets := frontendAssets(t)
	dir := gittest.NewRepo(t)
	init := runCLI(t, assets, "-C", dir, "init", "--runtime", "claude", "--ext", "go,htmx", "--yes")
	require.Equal(t, 0, init.exit, init.stderr)
	htmx := tree(t, filepath.Join(dir, "docs/solomonar/extensions/htmx"))
	args := []string{"extension", "add", "templ", "--rationale", "Type-checked components"}

	declined := runInteractive(t, assets, dir, "n\n", args...)
	got := runCLI(t, assets, append([]string{"-C", dir}, append(args, "--yes")...)...)

	require.Equal(t, 0, declined.exit, declined.stderr)
	assert.Contains(t, declined.stderr, "  decisions (approved, selected by you)\n"+
		"    update     docs/decisions.md  P/D-3 frontend: htmx, templ · P/D-2 superseded\n"+
		"    append     docs/stack.md      frontend: htmx, templ (P/D-3)\n")
	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout,
		"Added extension templ (frontend skill, catalog, under P/D-3) in "+dir+"\n"), got.stdout)
	decisions := readTestFile(t, dir, "docs/decisions.md")
	assert.Contains(t, decisions, "## P/D-2: Frontend technologies\n\n- Status: superseded by P/D-3\n")
	assert.Equal(t, 1, strings.Count(decisions, "- Status: approved\n- Selected option: go\n"))
	assert.True(t, strings.HasSuffix(decisions, "\n\n## P/D-3: Frontend technologies\n\n- Status: approved\n"+
		"- Selected option: htmx, templ\n- Rejected options: none\n"+
		"- Scope: project; frontend extensions `htmx`, `templ`\n- Rationale: Type-checked components\n"+
		"- Date: 2026-10-21\n- References: supersedes P/D-2; `docs/solomonar/extensions/htmx/SKILL.md`, "+
		"`docs/solomonar/extensions/templ/SKILL.md`\n"), decisions)
	assert.True(t, strings.HasSuffix(readTestFile(t, dir, "docs/stack.md"),
		"- frontend: htmx (P/D-2)\n- frontend: htmx, templ (P/D-3)\n"))
	agents := readTestFile(t, dir, "AGENTS.md")
	assert.Contains(t, agents, "  - `htmx` (frontend, P/D-3): ")
	assert.Contains(t, agents, "  - `templ` (frontend, P/D-3): ")
	m := readTestManifest(t, dir)
	assert.Equal(t, "P/D-3", m.Extensions["htmx"].Decision)
	assert.Equal(t, "P/D-3", m.Extensions["templ"].Decision)
	assert.Equal(t, "P/D-1", m.Extensions["go"].Decision)
	assert.Equal(t, []string{"P/D-1", "P/D-2", "P/D-3"}, m.Files["docs/decisions.md"].Entries)
	assert.Equal(t, htmx, tree(t, filepath.Join(dir, "docs/solomonar/extensions/htmx")))
}

func TestExtensionAddRecordsAnApprovedTaskDecision(t *testing.T) {
	t.Parallel()
	dir := gittest.NewRepo(t)
	init := runInRepo(t, dir, "init", "--runtime", "none", "--ext", "none", "--yes")
	require.Equal(t, 0, init.exit, init.stderr)
	writeFile(t, dir, "docs/decisions.md", taskDecisions)

	got := addIn(t, dir, "go", "--decision", "W-002/D-1", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout,
		"Added extension go (language skill, catalog, under W-002/D-1) in "+dir+"\n"), got.stdout)
	assert.Equal(t, taskDecisions, readTestFile(t, dir, "docs/decisions.md"))
	assert.Equal(t, "# Stack\n\n- language: go (W-002/D-1)\n", readTestFile(t, dir, "docs/stack.md"))
	assert.Contains(t, readTestFile(t, dir, "AGENTS.md"), "  - `go` (language, W-002/D-1): ")
	m := readTestManifest(t, dir)
	assert.Equal(t, "W-002/D-1", m.Extensions["go"].Decision)
	assert.Equal(t, project.File{Owner: "decision", Entries: []string{"W-002/D-1"}}, m.Files["docs/stack.md"])
	_, recorded := m.Files["docs/decisions.md"]
	assert.False(t, recorded)
}

const goRemoveCommit = `git add AGENTS.md .claude .solomonar docs && ` +
	`git commit -m "chore: remove solomonar extension go"`

// claudeGoRepo is a current project with the claude runtime and go installed under P/D-1.
func claudeGoRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.NewRepo(t)
	got := runInRepo(t, dir, "init", "--runtime", "claude", "--ext", "go", "--yes")
	require.Equal(t, 0, got.exit, got.stderr)
	return dir
}

func removeIn(t *testing.T, dir string, args ...string) result {
	t.Helper()
	return runInRepo(t, dir, append([]string{"extension", "remove"}, args...)...)
}

func TestExtensionRemoveDeletesACatalogExtension(t *testing.T) {
	t.Parallel()
	dir := claudeGoRepo(t)

	got := removeIn(t, dir, "go", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Regexp(t, `^Removed extension go \(language skill, catalog, under P/D-2\) in `+regexp.QuoteMeta(dir)+`\n`+
		`  4 files updated, 7 removed, \d+ unchanged, 1 skipped\n\nNext:\n  1\. `+regexp.QuoteMeta(goRemoveCommit)+"\n$",
		got.stdout)
	assert.Equal(t, "warning: code governed by go loses its rules\n", got.stderr)
	assert.NoDirExists(t, filepath.Join(dir, "docs/solomonar/extensions/go"))
	assert.NoDirExists(t, filepath.Join(dir, ".claude/skills/go"))
	assert.NotContains(t, readTestFile(t, dir, "AGENTS.md"), "`go`")
	decisions := readTestFile(t, dir, "docs/decisions.md")
	assert.Contains(t, decisions, "## P/D-1: Programming languages\n\n- Status: superseded by P/D-2\n")
	assert.True(t, strings.HasSuffix(decisions, "\n\n## P/D-2: Programming languages\n\n- Status: approved\n"+
		"- Selected option: none\n- Rejected options: none\n- Scope: project; language extensions none\n"+
		"- Rationale: selected by the user with `solomonar extension remove`\n- Date: 2026-10-21\n"+
		"- References: supersedes P/D-1\n"), decisions)
	assert.Equal(t, "# Stack\n\n- language: go (P/D-1)\n- language: none (P/D-2)\n",
		readTestFile(t, dir, "docs/stack.md"))
	m := readTestManifest(t, dir)
	assert.Empty(t, m.Extensions)
	for path := range m.Files {
		assert.NotContains(t, path, "/go/")
	}
	assert.Equal(t, []string{"P/D-1", "P/D-2"}, m.Files["docs/decisions.md"].Entries)
}

func TestExtensionRemoveEditedStoreFileIsAConflict(t *testing.T) {
	t.Parallel()
	const skill = "docs/solomonar/extensions/go/SKILL.md"
	tests := []struct {
		name, flag string
		kept       bool
	}{
		{name: "keep", flag: "--keep", kept: true},
		{name: "overwrite", flag: "--overwrite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := claudeGoRepo(t)
			writeFile(t, dir, skill, "# our go rules\n")
			before, tip := tree(t, dir), refTip(t, dir)

			conflict := removeIn(t, dir, "go", "--yes")
			assertNothingWritten(t, dir, before, tip)
			got := removeIn(t, dir, "go", "--yes", tt.flag, skill)

			assert.Equal(t, 4, conflict.exit)
			assert.Contains(t, conflict.stdout, "  conflict   "+skill)
			assert.Contains(t, conflict.stderr, "error: extension remove found 1 conflict (file_conflict)\n")
			assert.NotContains(t, conflict.stderr, "warning:")
			require.Equal(t, 0, got.exit, got.stderr)
			if tt.kept {
				assert.Equal(t, "# our go rules\n", readTestFile(t, dir, skill))
			} else {
				assert.NoFileExists(t, filepath.Join(dir, skill))
			}
			assert.NoFileExists(t, filepath.Join(dir, "docs/solomonar/extensions/go/extension.json"))
			_, tracked := readTestManifest(t, dir).Files[skill]
			assert.False(t, tracked)
		})
	}
}

func TestExtensionAddReinstallsAfterAKeptRemoval(t *testing.T) {
	t.Parallel()
	const skill = "docs/solomonar/extensions/go/SKILL.md"
	dir := claudeGoRepo(t)
	writeFile(t, dir, skill, "# our go rules\n")
	require.Equal(t, 0, removeIn(t, dir, "go", "--yes", "--keep", skill).exit)

	list := runInRepo(t, dir, "extension", "list")
	conflict := addIn(t, dir, "go", "--yes")
	got := addIn(t, dir, "go", "--yes", "--overwrite", skill)

	require.Equal(t, 0, list.exit, list.stderr)
	assert.NotContains(t, list.stdout, "invalid")
	assert.Equal(t, 4, conflict.exit)
	assert.Contains(t, conflict.stdout, "  conflict   "+skill+"  exists and is not managed by solomonar")
	require.Equal(t, 0, got.exit, got.stderr)
	assert.NotEqual(t, "# our go rules\n", readTestFile(t, dir, skill))
	m := readTestManifest(t, dir)
	assert.Equal(t, "catalog", m.Extensions["go"].Source)
	assert.Contains(t, m.Files, skill)
}

func TestExtensionAddRefillsAnEmptiedGroup(t *testing.T) {
	t.Parallel()
	dir := claudeGoRepo(t)
	require.Equal(t, 0, removeIn(t, dir, "go", "--yes").exit)

	got := addIn(t, dir, "go", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	decisions := readTestFile(t, dir, "docs/decisions.md")
	assert.Contains(t, decisions, "## P/D-2: Programming languages\n\n- Status: superseded by P/D-3\n")
	assert.True(t, strings.HasSuffix(decisions, "\n\n## P/D-3: Programming languages\n\n- Status: approved\n"+
		"- Selected option: go\n- Rejected options: none\n- Scope: project; language extensions `go`\n"+
		"- Rationale: selected by the user with `solomonar extension add`\n- Date: 2026-10-21\n"+
		"- References: supersedes P/D-2; `docs/solomonar/extensions/go/SKILL.md`\n"), decisions)
	assert.Equal(t, "# Stack\n\n- language: go (P/D-1)\n- language: none (P/D-2)\n- language: go (P/D-3)\n",
		readTestFile(t, dir, "docs/stack.md"))
	assert.Equal(t, "P/D-3", readTestManifest(t, dir).Extensions["go"].Decision)
}

func TestExtensionRemoveRejectsOverwritingANonRegularFile(t *testing.T) {
	t.Parallel()
	const skill = "docs/solomonar/extensions/go/SKILL.md"
	dir := claudeGoRepo(t)
	require.NoError(t, os.Remove(filepath.Join(dir, skill)))
	require.NoError(t, os.Symlink("extension.json", filepath.Join(dir, skill)))
	before, tip := tree(t, dir), refTip(t, dir)

	got := removeIn(t, dir, "go", "--yes", "--overwrite", skill)

	assert.Equal(t, 2, got.exit)
	assert.Equal(t, "error: extension remove rejected (invalid_value)\n"+
		"  check: --keep and --overwrite resolve conflicts\n  found: "+skill+" is not a regular file\n"+
		"  fix:   remove it by hand or pass --keep "+skill+"\nnothing was written\n", got.stderr)
	assertNothingWritten(t, dir, before, tip)
}

func TestExtensionRemoveDeactivatesAProjectExtension(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)
	require.Equal(t, 0, addIn(t, dir, "money", "--yes").exit)
	stored := tree(t, filepath.Join(dir, "docs/solomonar/extensions/money"))

	got := removeIn(t, dir, "money", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout,
		"Removed extension money (review agent, project) in "+dir+"\n"), got.stdout)
	assert.Contains(t, got.stdout, `git commit -m "chore: remove solomonar extension money"`)
	assert.Equal(t, "warning: code governed by money loses its rules\n", got.stderr)
	assert.Equal(t, stored, tree(t, filepath.Join(dir, "docs/solomonar/extensions/money")))
	assert.NoFileExists(t, filepath.Join(dir, ".claude/agents/money.md"))
	assert.NoFileExists(t, filepath.Join(dir, ".codex/agents/money.toml"))
	assert.NotContains(t, readTestFile(t, dir, "AGENTS.md"), "`money`")
	assert.NotContains(t, readTestManifest(t, dir).Extensions, "money")
	assert.NoFileExists(t, filepath.Join(dir, "docs/decisions.md"))
	list := runInRepo(t, dir, "extension", "list", "--group", "review")
	assert.Contains(t, list.stdout, "review  money  agent  project  available")
}

func TestExtensionRemoveRecordsAnApprovedTaskDecision(t *testing.T) {
	t.Parallel()
	dir := claudeGoRepo(t)
	writeFile(t, dir, "docs/decisions.md", readTestFile(t, dir, "docs/decisions.md")+"\n"+
		strings.TrimPrefix(taskDecisions, "# Decisions\n\n"))
	decisions := readTestFile(t, dir, "docs/decisions.md")

	got := removeIn(t, dir, "go", "--decision", "W-002/D-1", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout,
		"Removed extension go (language skill, catalog, under W-002/D-1) in "+dir+"\n"), got.stdout)
	assert.Equal(t, decisions, readTestFile(t, dir, "docs/decisions.md"))
	assert.True(t, strings.HasSuffix(readTestFile(t, dir, "docs/stack.md"), "- language: none (W-002/D-1)\n"))
}

func TestExtensionRemoveDryRunAndConfirmation(t *testing.T) {
	t.Parallel()
	dir := claudeGoRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	dryRun := removeIn(t, dir, "go", "--dry-run")
	unconfirmed := removeIn(t, dir, "go")
	declined := runInteractive(t, repoAssets(t), dir, "n\n", "extension", "remove", "go")

	require.Equal(t, 0, dryRun.exit, dryRun.stderr)
	assert.Contains(t, dryRun.stdout, "  remove     docs/solomonar/extensions/go/SKILL.md\n")
	assert.Regexp(t, `Would remove extension go in `+regexp.QuoteMeta(dir)+`\n  .*\n$`, dryRun.stdout)
	assert.Empty(t, dryRun.stderr)
	assert.Equal(t, 2, unconfirmed.exit)
	assert.Equal(t, "error: extension remove needs a confirmation (interactive_required)\n"+
		"  fix:   pass --yes to accept the plan, or --dry-run to print it\n"+
		"nothing was written\n", unconfirmed.stderr)
	require.Equal(t, 0, declined.exit, declined.stderr)
	assert.Contains(t, declined.stderr, "  decisions (approved, selected by you)\n"+
		"    update     docs/decisions.md  P/D-2 language: none · P/D-1 superseded\n"+
		"    append     docs/stack.md      language: none (P/D-2)\n")
	assert.Contains(t, declined.stderr, "    remove     docs/solomonar/extensions/go/ (")
	assert.Equal(t, "Cancelled; nothing was written.\n", declined.stdout)
	assertNothingWritten(t, dir, before, tip)
}

func TestExtensionRemovePlanLabelsTheRemovedStubs(t *testing.T) {
	t.Parallel()
	goDir := claudeGoRepo(t)
	moneyDir := gittest.NewRepo(t)
	got := runInRepo(t, moneyDir, "init", "--runtime", "claude", "--adversarial-model", "claude=opus", "--ext", "none",
		"--yes")
	require.Equal(t, 0, got.exit, got.stderr)
	writeFile(t, moneyDir, "docs/solomonar/extensions/money/extension.json", moneyReviewerJSON)
	writeFile(t, moneyDir, "docs/solomonar/extensions/money/money.md", moneyReviewer)
	require.Equal(t, 0, addIn(t, moneyDir, "money", "--yes").exit)
	tests := []struct{ name, dir, ext, line string }{
		{name: "skill", dir: goDir, ext: "go", line: "    remove     .claude/skills/ (1 skill stub: 0 core, 1 extension)\n"},
		{name: "agent", dir: moneyDir, ext: "money", line: "    remove     .claude/agents/ (1 role stub: 0 core, 1 extension)\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			declined := runInteractive(t, repoAssets(t), tt.dir, "n\n", "extension", "remove", tt.ext)

			require.Equal(t, 0, declined.exit, declined.stderr)
			assert.Contains(t, declined.stderr, "  runtime claude\n"+tt.line)
		})
	}
}

func TestExtensionRemoveJSON(t *testing.T) {
	t.Parallel()
	dir := claudeGoRepo(t)

	got := runInRepo(t, dir, "--json", "ext", "remove", "go", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	var envelope struct {
		Command  string        `json:"command"`
		Data     extensionData `json:"data"`
		Warnings []warning     `json:"warnings"`
	}
	require.NoError(t, json.Unmarshal([]byte(got.stdout), &envelope))
	assert.Equal(t, "extension remove", envelope.Command)
	assert.Equal(t, []string{"go"}, envelope.Data.Removed)
	assert.Nil(t, envelope.Data.Added)
	assert.Equal(t, []project.Decision{{ID: "P/D-2", Group: "language", Selected: []string{}}}, envelope.Data.Decisions)
	assert.Contains(t, envelope.Data.Files,
		initFile{Path: "docs/solomonar/extensions/go/SKILL.md", Action: "remove", Owner: "ext:go"})
	assert.Equal(t, []string{goRemoveCommit}, envelope.Data.Next)
	assert.Equal(t, []warning{{Code: "rules_removed", Message: "code governed by go loses its rules"}},
		envelope.Warnings)
}

func TestExtensionRemoveRejections(t *testing.T) {
	t.Parallel()
	rejected := func(code, check, found, fix string) string {
		return "error: extension remove rejected (" + code + ")\n  check: " + check + "\n  found: " + found +
			"\n  fix:   " + fix + "\nnothing was written\n"
	}
	audit := map[string]string{
		"docs/solomonar/extensions/audit/extension.json": `{"name": "audit", "kind": "skill", "group": "review", ` +
			`"title": "T", "applies_when": "A change", "requires": ["money"]}`,
		"docs/solomonar/extensions/audit/SKILL.md": "---\nname: audit\ndescription: Use for audit.\n---\n",
	}
	tests := []struct {
		name  string
		bare  bool
		files map[string]string
		add   []string
		args  []string
		exit  int
		want  string
	}{
		{
			name: "no name", exit: 2,
			want: "error: extension name is required (invalid_usage)\n  fix:   run solomonar help\nnothing was written\n",
		},
		{
			name: "not initialized", bare: true, args: []string{"money"}, exit: 3,
			want: rejected("not_initialized", "the project is initialized", "no .solomonar/manifest.json in %s",
				"run solomonar init"),
		},
		{
			name: "core skill", args: []string{"work-spec"}, exit: 3,
			want: rejected("core_item", "extension remove names an extension", "work-spec is part of the core",
				"the core cannot be removed (4.1)"),
		},
		{
			name: "core reviewer", args: []string{"security-reviewer"}, exit: 3,
			want: rejected("core_item", "extension remove names an extension", "security-reviewer is part of the core",
				"the core cannot be removed (4.1)"),
		},
		{
			name: "not installed", add: []string{"money"}, args: []string{"go"}, exit: 2,
			want: rejected("unknown_extension", "extension remove names an installed extension",
				`"go" is not installed; installed: money`, "pass a name from solomonar extension list --installed"),
		},
		{
			name: "nothing installed", args: []string{"money"}, exit: 2,
			want: rejected("unknown_extension", "extension remove names an installed extension",
				`"money" is not installed; installed: none`, "pass a name from solomonar extension list --installed"),
		},
		{
			name: "decision for a review extension", add: []string{"money"},
			args: []string{"money", "--decision", "P/D-1"}, exit: 2,
			want: "error: --decision and --rationale apply to stack groups only (invalid_usage)\n" +
				"  fix:   run solomonar help\nnothing was written\n",
		},
		{
			name: "required", files: audit, add: []string{"money", "audit"}, args: []string{"money"}, exit: 3,
			want: rejected("extension_requires", "every required extension is installed", "audit requires money",
				"remove both: solomonar extension remove money audit"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := gittest.NewRepo(t)
			if !tt.bare {
				dir = moneyRepo(t)
			}
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}
			if len(tt.add) > 0 {
				require.Equal(t, 0, addIn(t, dir, append(tt.add, "--yes")...).exit)
			}
			before := tree(t, dir)

			got := removeIn(t, dir, append(tt.args, "--yes")...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			want := tt.want
			if strings.Contains(want, "%s") {
				want = fmt.Sprintf(want, dir)
			}
			assert.Equal(t, want, got.stderr)
			assert.Equal(t, before, tree(t, dir))
		})
	}
}

func TestExtensionRemoveRejectsAnOlderFramework(t *testing.T) {
	t.Parallel()
	dir := claudeGoRepo(t)
	m := readTestManifest(t, dir)
	m.FrameworkVersion = "0.1.7"
	writeTestManifest(t, dir, m)
	before := tree(t, dir)

	got := runCLI(t, assetsAt(t, "0.1.8"), "-C", dir, "extension", "remove", "work-spec", "--yes")

	assert.Equal(t, 3, got.exit)
	assert.Contains(t, got.stderr, "error: extension remove rejected (framework_outdated)\n")
	assert.Equal(t, before, tree(t, dir))
}

func TestExtensionRemoveDeactivatesAnInvalidProjectExtension(t *testing.T) {
	t.Parallel()
	dir := moneyRepo(t)
	require.Equal(t, 0, addIn(t, dir, "money", "--yes").exit)
	writeFile(t, dir, "docs/solomonar/extensions/money/extension.json", "{}")

	got := removeIn(t, dir, "money", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.NotContains(t, readTestManifest(t, dir).Extensions, "money")
	assert.Equal(t, "{}", readTestFile(t, dir, "docs/solomonar/extensions/money/extension.json"))
}
