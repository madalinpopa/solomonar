package cli

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
)

func goArgs(extra ...string) []string {
	return append([]string{"init", "--runtime", "none", "--ext", "go", "--yes"}, extra...)
}

// goRepo is a committed repository initialized with the go extension and no runtime.
func goRepo(t *testing.T) string {
	t.Helper()
	dir := committedRepo(t)
	got := runInRepo(t, dir, goArgs()...)
	require.Equal(t, 0, got.exit, got.stderr)
	return dir
}

func goEntry(decision string) map[string]project.Extension {
	return map[string]project.Extension{"go": {
		State: "installed", Kind: "skill", Group: "language", Source: "catalog", Decision: decision,
	}}
}

func TestInitUC2WithGo(t *testing.T) {
	t.Parallel()

	dir := committedRepo(t)

	got := runInRepo(t, dir, goArgs()...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  22 files created\n")
	assert.Contains(t, got.stdout, "\nNext:\n"+commitStep("AGENTS.md .solomonar docs"))
	files := tree(t, dir)
	assert.Contains(t, files["docs/solomonar/skills/go/SKILL.md"], "\n# Go language rules\n\nInstalled by P/D-1.\n\n")
	assert.Contains(t, files["docs/decisions.md"], "## P/D-1: Programming languages\n")
	assert.Contains(t, files["docs/decisions.md"], "- Date: 2026-10-21\n")
	assert.Equal(t, "# Stack\n\n- language: go (P/D-1)\n", files["docs/stack.md"])
	assert.Contains(t, files["docs/README.md"],
		"- `docs/decisions.md`: decision entries and their status\n"+
			"- `docs/stack.md`: technologies and practices, with the decisions that adopted them\n")
	assert.Contains(t, files["AGENTS.md"], "- Skills\n  - `go` (language): A change touches Go code "+
		"(*.go, go.mod, go.sum); paths: `**/*.go`, `**/go.mod`, `**/go.sum`\n")
	manifest := manifestOf(t, dir)
	assert.Equal(t, goEntry("P/D-1"), manifest.Extensions)
	assert.Equal(t, "decision", manifest.Files["docs/decisions.md"].Owner)
}

func TestInitWithGoJSON(t *testing.T) {
	t.Parallel()

	dir := committedRepo(t)

	got := runInRepo(t, dir, goArgs("--json")...)

	require.Equal(t, 0, got.exit, got.stderr)
	var envelope struct {
		Data struct {
			Choices   jsontext.Value `json:"choices"`
			Decisions jsontext.Value `json:"decisions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(got.stdout), &envelope))
	assert.JSONEq(t, `{"runtimes": [], "extensions": {"language": ["go"]}, "adversarial_model": {}}`,
		string(envelope.Data.Choices))
	assert.JSONEq(t, `[{"id": "P/D-1", "group": "language", "selected": ["go"]}]`, string(envelope.Data.Decisions))
}

func TestInitWithGoAndClaudeAddsTheSkillStub(t *testing.T) {
	t.Parallel()

	dir := committedRepo(t)

	got := runInRepo(t, dir, "init", "--runtime", "claude", "--ext", "go", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	stub := tree(t, dir)[".claude/skills/go/SKILL.md"]
	assert.True(t, strings.HasPrefix(stub, "---\nname: go\n"), stub)
	assert.True(t, strings.HasSuffix(stub, "Load `docs/solomonar/skills/go/SKILL.md` and follow it.\n"), stub)
}

func TestInitSelectsExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		goMod bool
		args  []string
		want  map[string]project.Extension
	}{
		{name: "go from go.mod", goMod: true, args: []string{"init", "--runtime", "none", "--yes"}, want: goEntry("P/D-1")},
		{name: "none without go.mod", args: []string{"init", "--runtime", "none", "--yes"}, want: map[string]project.Extension{}},
		{name: "none requested beside go.mod", goMod: true, args: initArgs(), want: map[string]project.Extension{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := committedRepo(t)
			if tt.goMod {
				writeFile(t, dir, "go.mod", "module example.com/ledger\n")
			}

			got := runInRepo(t, dir, tt.args...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, tt.want, manifestOf(t, dir).Extensions)
		})
	}
}

func TestInitIgnoresGoModDirectory(t *testing.T) {
	t.Parallel()

	dir := committedRepo(t)
	writeFile(t, dir, "go.mod/keep", "")

	got := runInRepo(t, dir, "init", "--runtime", "none", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Empty(t, manifestOf(t, dir).Extensions)
}

// catalogAssets is the repository's assets plus frontend extensions that require and conflict.
func catalogAssets(t *testing.T) fstest.MapFS {
	t.Helper()
	assets := fstest.MapFS{}
	err := fs.WalkDir(repoAssets(t), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := fs.ReadFile(repoAssets(t), name)
		assets[name] = &fstest.MapFile{Data: content}
		return err
	})
	require.NoError(t, err)
	for name, relations := range map[string]string{
		"templ": `"requires": ["htmx"], "conflicts": []`,
		"htmx":  `"requires": [], "conflicts": ["react"]`,
		"react": `"requires": [], "conflicts": []`,
	} {
		dir := "extensions/frontend/" + name + "/"
		assets[dir+"extension.json"] = &fstest.MapFile{Data: []byte(`{"name": "` + name +
			`", "kind": "skill", "group": "frontend", "title": "T", "applies_when": "A", "paths": [], ` +
			relations + `}`)}
		assets[dir+"SKILL.md"] = &fstest.MapFile{Data: []byte("---\nname: " + name + "\ndescription: D\n---\n\n# " + name + "\n")}
	}
	return assets
}

func TestInitRejectsIncompleteExtensionSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ext    string
		stderr string
	}{
		{
			name: "missing requirement", ext: "templ",
			stderr: "error: init rejected (extension_requires)\n" +
				"  check: every required extension is selected\n" +
				"  found: templ requires htmx\n" +
				"  fix:   add --ext htmx\n",
		},
		{
			name: "conflict", ext: "react,htmx",
			stderr: "error: init rejected (extension_conflict)\n" +
				"  check: no selected extensions conflict\n" +
				"  found: htmx conflicts with react\n" +
				"  fix:   drop --ext htmx or --ext react\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := gittest.NewRepo(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runCLI(t, catalogAssets(t), "-C", dir, "init", "--runtime", "none", "--ext", tt.ext, "--yes")

			assert.Equal(t, 3, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, tt.stderr)
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestInitRerunWithGoIsNoOp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "same choices", args: goArgs()},
		{name: "extensions from the manifest", args: []string{"init", "--yes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := goRepo(t)
			gittest.NoGH(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runInRepo(t, dir, tt.args...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Contains(t, got.stdout, "Already initialized with solomonar "+shipped+"; nothing to do.\n"+
				"  21 files unchanged, 1 skipped · registry")
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestInitRerunPlansDecisionDocumentsUnchanged(t *testing.T) {
	t.Parallel()

	dir := goRepo(t)

	got := runInRepo(t, dir, goArgs("--dry-run")...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  unchanged  docs/decisions.md\n  unchanged  docs/stack.md\n")
}

func TestInitRejectsDifferentExtensionChoice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		repo  func(*testing.T) string
		args  []string
		found string
	}{
		{name: "none over go", repo: goRepo, args: initArgs(), found: "manifest extensions [go]; requested none"},
		{name: "go over none", repo: initializedRepo, args: goArgs(), found: "manifest extensions none; requested [go]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := tt.repo(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runInRepo(t, dir, tt.args...)

			assert.Equal(t, 3, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, "error: init rejected (already_initialized)\n  found: "+tt.found+"\n"+
				"  fix:   change extensions with `solomonar extension add|remove`; init never changes choices\n")
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestInitAppendsToExistingDecisions(t *testing.T) {
	t.Parallel()

	dir := committedRepo(t)
	writeFile(t, dir, "docs/decisions.md", "# Decisions\n\n## P/D-3: Storage\n\n- Status: approved\n")

	got := runInRepo(t, dir, goArgs()...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  21 files created, 1 updated\n")
	decisions := tree(t, dir)["docs/decisions.md"]
	assert.True(t, strings.HasPrefix(decisions,
		"# Decisions\n\n## P/D-3: Storage\n\n- Status: approved\n\n## P/D-4: Programming languages\n"), decisions)
	assert.Equal(t, goEntry("P/D-4"), manifestOf(t, dir).Extensions)
}

func TestInitFailsOnManifestExtensionMissingFromCatalog(t *testing.T) {
	t.Parallel()

	dir := goRepo(t)
	assets := catalogAssets(t)
	delete(assets, "extensions/language/go/extension.json")
	before, tip := tree(t, dir), refTip(t, dir)

	got := runCLI(t, assets, "-C", dir, "init", "--yes")

	assert.Equal(t, 1, got.exit)
	assert.Contains(t, got.stderr, "read extension go: not in the catalog")
	assertNothingWritten(t, dir, before, tip)
}
