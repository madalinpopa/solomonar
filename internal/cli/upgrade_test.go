package cli

import (
	"encoding/json/v2"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
)

const (
	legacySkill   = "docs/solomonar/skills/go/SKILL.md"
	legacyStore   = "docs/solomonar/skills/go/"
	upgradedStore = "docs/solomonar/extensions/go/"
	legacyIntro   = "Installed with `solomonar`; draft project extensions are not active. Load every installed " +
		"extension skill whose\n\"applies when\" matches your change, before writing or reviewing code, and cite its " +
		"rules as `<name>/C-n`.\nThe orchestrator names the matching extensions in each packet.\n"
	currentIntro = "Installed with `solomonar`. Before writing or reviewing code, load the extension skills your " +
		"packet names; without\na packet, every installed one whose \"applies when\" matches your change. Cite their " +
		"rules as `<name>/C-n`.\n"
)

// legacyRepo rebuilds what v0.1.7 `init --runtime claude,codex --ext go` wrote: the go skill in
// docs/solomonar/skills/go/ with its provenance line, and the files that name that path.
func legacyRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.NewRepo(t)
	got := runInRepo(t, dir, "init", "--name", "ledger", "--runtime", "claude,codex", "--ext", "go", "--yes")
	require.Equal(t, 0, got.exit, got.stderr)
	m := readTestManifest(t, dir)
	for _, path := range slices.Sorted(maps.Keys(m.Files)) {
		rest, ok := strings.CutPrefix(path, upgradedStore)
		if !ok {
			continue
		}
		delete(m.Files, path)
		if rest == "extension.json" {
			continue
		}
		content := readTestFile(t, dir, path)
		if rest == "SKILL.md" {
			require.Contains(t, content, "# Go language rules\n\n")
			content = strings.Replace(content, "# Go language rules\n\n", "# Go language rules\n\nInstalled by P/D-1.\n\n", 1)
		}
		writeFile(t, dir, legacyStore+rest, content)
		m.Files[legacyStore+rest] = project.File{Owner: "ext:go", SHA256: project.Hash([]byte(content))}
	}
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "docs/solomonar/extensions")))
	for _, path := range []string{
		".claude/skills/go/SKILL.md", ".agents/skills/go/SKILL.md", "docs/solomonar/agents/implementer.md",
		"docs/solomonar/reviewer.md", "docs/decisions.md",
	} {
		content := readTestFile(t, dir, path)
		require.Contains(t, content, "docs/solomonar/extensions/")
		content = strings.ReplaceAll(content, "docs/solomonar/extensions/", "docs/solomonar/skills/")
		writeFile(t, dir, path, content)
		if f := m.Files[path]; f.SHA256 != "" {
			m.Files[path] = project.File{Owner: f.Owner, SHA256: project.Hash([]byte(content))}
		}
	}
	agents := readTestFile(t, dir, "AGENTS.md")
	require.Contains(t, agents, currentIntro)
	require.Contains(t, agents, "`go` (language, P/D-1)")
	agents = strings.Replace(agents, currentIntro, legacyIntro, 1)
	agents = strings.Replace(agents, "`go` (language, P/D-1)", "`go` (language)", 1)
	writeFile(t, dir, "AGENTS.md", agents)
	managed, _, _ := strings.Cut(agents, "## Project notes\n")
	m.Files["AGENTS.md"] = project.File{Owner: "core+notes", SHA256: project.Hash([]byte(managed))}
	m.FrameworkVersion = "0.1.7"
	writeTestManifest(t, dir, m)
	return dir
}

func readTestFile(t *testing.T, dir, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	return string(content)
}

func readTestManifest(t *testing.T, dir string) *project.Manifest {
	t.Helper()
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	defer root.Close()
	m, err := project.ReadManifest(root, "99.0.0")
	require.NoError(t, err)
	require.NotNil(t, m)
	return m
}

func writeTestManifest(t *testing.T, dir string, m *project.Manifest) {
	t.Helper()
	content, err := m.Encode()
	require.NoError(t, err)
	writeFile(t, dir, project.ManifestPath, string(content))
}

// overlayFS serves files from over in place of the same paths in the embedded FS.
type overlayFS struct {
	fs.FS
	over fstest.MapFS
}

func (o overlayFS) Open(name string) (fs.File, error) {
	if _, ok := o.over[name]; ok {
		return o.over.Open(name)
	}
	return o.FS.Open(name)
}

// assetsAt ships the repository assets as framework version.
func assetsAt(t *testing.T, version string) fs.FS {
	t.Helper()
	assets := repoAssets(t)
	rules, err := fs.ReadFile(assets, "framework/rules.md")
	require.NoError(t, err)
	content := strings.Replace(string(rules), "Framework version: "+shipped, "Framework version: "+version, 1)
	return overlayFS{FS: assets, over: fstest.MapFS{"framework/rules.md": {Data: []byte(content)}}}
}

func upgradeIn(t *testing.T, assets fs.FS, dir string, args ...string) result {
	t.Helper()
	return runCLI(t, assets, append([]string{"-C", dir, "upgrade"}, args...)...)
}

func TestUpgradeMovesLegacyProjectToTheStore(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)
	decisions := readTestFile(t, dir, "docs/decisions.md")
	tip := refTip(t, dir)

	got := upgradeIn(t, assetsAt(t, "99.0.0"), dir, "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Upgraded solomonar 0.1.7 → 99.0.0 in "+dir+"\n"+
		"  6 files created, 26 updated, 5 removed, 21 unchanged, 1 skipped\n\n"+
		"Next:\n"+
		"  1. git add AGENTS.md .claude .agents .codex .solomonar docs && "+
		"git commit -m \"chore: upgrade solomonar to 99.0.0\"\n"+
		"  2. Running tasks pick up the new framework after they update from main.\n", got.stdout)
	assert.Empty(t, got.stderr)
	assets := repoAssets(t)
	for _, name := range []string{"extension.json", "SKILL.md", "rules/go/testing.md"} {
		want, err := fs.ReadFile(assets, "extensions/language/go/"+name)
		require.NoError(t, err)
		assert.Equal(t, string(want), readTestFile(t, dir, upgradedStore+name), name)
	}
	assert.NoDirExists(t, filepath.Join(dir, "docs/solomonar/skills/go"))
	assert.Contains(t, readTestFile(t, dir, ".claude/skills/go/SKILL.md"), "Load `docs/solomonar/extensions/go/SKILL.md`")
	assert.Contains(t, readTestFile(t, dir, ".agents/skills/go/SKILL.md"), "Load `docs/solomonar/extensions/go/SKILL.md`")
	agents := readTestFile(t, dir, "AGENTS.md")
	assert.Contains(t, agents, currentIntro)
	assert.Contains(t, agents, "`go` (language, P/D-1)")
	assert.Equal(t, decisions, readTestFile(t, dir, "docs/decisions.md"))
	assert.Equal(t, tip, refTip(t, dir))
	m := readTestManifest(t, dir)
	assert.Equal(t, "99.0.0", m.FrameworkVersion)
	assert.Equal(t, "P/D-1", m.Extensions["go"].Decision)
	assert.NotContains(t, m.Files, legacySkill)
	assert.Equal(t, project.File{Owner: "ext:go", SHA256: project.Hash([]byte(readTestFile(t, dir, upgradedStore+"SKILL.md")))},
		m.Files[upgradedStore+"SKILL.md"])
}

func TestUpgradeRerunIsNoOp(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)
	first := upgradeIn(t, repoAssets(t), dir, "--yes")
	require.Equal(t, 0, first.exit, first.stderr)
	before := tree(t, dir)

	got := upgradeIn(t, repoAssets(t), dir)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Already at solomonar "+shipped+"; nothing to do.\n  53 files unchanged, 1 skipped\n", got.stdout)
	assert.Empty(t, got.stderr)
	assert.Equal(t, before, tree(t, dir))
}

func TestUpgradeAtTheSameVersionRestoresADeletedStub(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)
	first := upgradeIn(t, repoAssets(t), dir, "--yes")
	require.Equal(t, 0, first.exit, first.stderr)
	const stub = ".claude/skills/spec/SKILL.md"
	want := readTestFile(t, dir, stub)
	require.NoError(t, os.Remove(filepath.Join(dir, stub)))

	got := upgradeIn(t, repoAssets(t), dir, "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, "Refreshed solomonar "+shipped+" in "), got.stdout)
	assert.Contains(t, got.stdout, "1 file created")
	assert.Contains(t, got.stdout, "git commit -m \"chore: refresh solomonar "+shipped+" files\"")
	assert.Equal(t, want, readTestFile(t, dir, stub))
}

func TestUpgradeEditedLegacySkillIsConflict(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)
	writeFile(t, dir, legacySkill, "# Go, edited\n")
	before, tip := tree(t, dir), refTip(t, dir)

	got := upgradeIn(t, repoAssets(t), dir, "--yes")

	require.Equal(t, 4, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, "Upgrade framework 0.1.7 → "+shipped+"\n"), got.stdout)
	assert.Contains(t, got.stdout, "  create     docs/solomonar/extensions/go/SKILL.md")
	assert.Contains(t, got.stdout, "  unchanged  docs/solomonar/rules.md")
	assert.Contains(t, got.stdout,
		"  conflict   "+legacySkill+"  modified since installation (manifest sha256 differs)\n")
	assert.Equal(t, "error: upgrade found 1 conflict (file_conflict)\n"+
		"  check: no planned file conflicts with the project's files\n"+
		"  found: 1 conflict: "+legacySkill+"\n"+
		"  fix:   rerun with --keep <path> to keep the project's file, or --take-new <path> to take the new version\n"+
		"nothing was written\n", got.stderr)
	assertNothingWritten(t, dir, before, tip)
}

func TestUpgradeResolvesRemovalConflict(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		flag string
		kept bool
	}{
		{name: "keep untracks the file", flag: "--keep", kept: true},
		{name: "take-new deletes the file", flag: "--take-new"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := legacyRepo(t)
			writeFile(t, dir, legacySkill, "# Go, edited\n")

			got := upgradeIn(t, repoAssets(t), dir, "--yes", tt.flag, legacySkill)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.NoFileExists(t, filepath.Join(dir, legacyStore, "rules/go/testing.md"))
			assert.FileExists(t, filepath.Join(dir, upgradedStore, "SKILL.md"))
			assert.NotContains(t, readTestManifest(t, dir).Files, legacySkill)
			if tt.kept {
				assert.Equal(t, "# Go, edited\n", readTestFile(t, dir, legacySkill))
			} else {
				assert.NoDirExists(t, filepath.Join(dir, legacyStore))
			}
			again := upgradeIn(t, repoAssets(t), dir)
			require.Equal(t, 0, again.exit, again.stderr)
			assert.True(t, strings.HasPrefix(again.stdout, "Already at solomonar "), again.stdout)
		})
	}
}

func TestUpgradeKeepAdvancesTheBase(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)
	writeFile(t, dir, ".claude/skills/go/SKILL.md", "edited stub\n")

	got := upgradeIn(t, repoAssets(t), dir, "--yes", "--keep", ".claude/skills/go/SKILL.md")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "edited stub\n", readTestFile(t, dir, ".claude/skills/go/SKILL.md"))
	entry := readTestManifest(t, dir).Files[".claude/skills/go/SKILL.md"]
	assert.NotEqual(t, project.Hash([]byte("edited stub\n")), entry.SHA256)
	again := upgradeIn(t, repoAssets(t), dir)
	require.Equal(t, 0, again.exit, again.stderr)
	assert.Contains(t, again.stdout, "  1 file kept, ")
}

func TestUpgradeRejectsNewerProject(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*project.Manifest)
	}{
		{name: "framework", mutate: func(m *project.Manifest) { m.FrameworkVersion = "9.0.0" }},
		{name: "schema", mutate: func(m *project.Manifest) { m.SchemaVersion = 2 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := legacyRepo(t)
			m := readTestManifest(t, dir)
			tt.mutate(m)
			writeTestManifest(t, dir, m)
			before := tree(t, dir)

			got := upgradeIn(t, repoAssets(t), dir, "--yes")

			require.Equal(t, 3, got.exit, got.stderr)
			assert.Empty(t, got.stdout)
			assert.True(t, strings.HasPrefix(got.stderr, "error: upgrade rejected (cli_outdated)\n"), got.stderr)
			assert.Contains(t, got.stderr, "  fix:   install a newer solomonar release\n")
			assert.Equal(t, before, tree(t, dir))
		})
	}
}

func TestUpgradeRequiresManifest(t *testing.T) {
	t.Parallel()
	dir := gittest.NewRepo(t)

	got := upgradeIn(t, repoAssets(t), dir, "--yes")

	require.Equal(t, 3, got.exit, got.stderr)
	assert.Empty(t, got.stdout)
	assert.Equal(t, "error: upgrade rejected (not_initialized)\n"+
		"  check: the project is initialized\n"+
		"  found: no .solomonar/manifest.json in "+dir+"\n"+
		"  fix:   run solomonar init\n"+
		"nothing was written\n", got.stderr)
	assert.Empty(t, tree(t, dir))
}

func TestUpgradeDryRunWritesNothing(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := upgradeIn(t, repoAssets(t), dir, "--dry-run")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, "Upgrade framework 0.1.7 → "+shipped+"\n  unchanged  docs/solomonar/agents/"),
		got.stdout)
	assert.Contains(t, got.stdout, "  remove     "+legacySkill+"\n")
	assert.Contains(t, got.stdout, "  update     .solomonar/manifest.json\n")
	assert.True(t, strings.HasSuffix(got.stdout, "Would upgrade solomonar 0.1.7 → "+shipped+" in "+dir+"\n"+
		"  6 files created, 6 updated, 5 removed, 41 unchanged, 1 skipped\n"), got.stdout)
	assertNothingWritten(t, dir, before, tip)
}

func TestUpgradeNeedsConfirmation(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)
	before := tree(t, dir)

	got := upgradeIn(t, repoAssets(t), dir)

	require.Equal(t, 2, got.exit, got.stderr)
	assert.Empty(t, got.stdout)
	assert.Equal(t, "error: upgrade needs a confirmation (interactive_required)\n"+
		"  fix:   pass --yes to accept the plan, or --dry-run to print it\n"+
		"nothing was written\n", got.stderr)
	assert.Equal(t, before, tree(t, dir))
}

func TestUpgradeConfirmsInTerminal(t *testing.T) {
	t.Parallel()
	wantPlan := "Upgrade framework 0.1.7 → " + shipped + "\n" +
		"  framework " + shipped + "\n" +
		"    update     docs/solomonar/agents/ (1 role)\n" +
		"    update     docs/solomonar/reviewer.md\n" +
		"    update     AGENTS.md\n" +
		"  extensions\n" +
		"    create     docs/solomonar/extensions/go/ (6 files)\n" +
		"    remove     docs/solomonar/skills/go/ (5 files)\n" +
		"  runtime claude\n" +
		"    update     .claude/skills/ (1 skill stub: 0 core, 1 extension)\n" +
		"  runtime codex\n" +
		"    update     .agents/skills/ (1 skill stub: 0 core, 1 extension)\n" +
		"  manifest\n" +
		"    update     .solomonar/manifest.json\n" +
		"Proceed? [Y/n] "
	tests := []struct {
		name, input, stdout string
		applied             bool
	}{
		{name: "declined", input: "n\n", stdout: "Cancelled; nothing was written.\n"},
		{name: "accepted", input: "\n", stdout: "Upgraded solomonar 0.1.7 → " + shipped + " in ", applied: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := legacyRepo(t)

			got := runInteractive(t, repoAssets(t), dir, tt.input, "upgrade")

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, wantPlan+"\n", got.stderr)
			assert.True(t, strings.HasPrefix(got.stdout, tt.stdout), got.stdout)
			assert.Equal(t, tt.applied, readTestManifest(t, dir).Files[upgradedStore+"SKILL.md"].Owner != "")
		})
	}
}

func TestUpgradeJSON(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)

	got := upgradeIn(t, repoAssets(t), dir, "--yes", "--json")

	require.Equal(t, 0, got.exit, got.stderr)
	var envelope struct {
		Command string      `json:"command"`
		Data    upgradeData `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(got.stdout), &envelope))
	assert.Equal(t, "upgrade", envelope.Command)
	assert.Equal(t, dir, envelope.Data.Project)
	assert.Equal(t, "0.1.7", envelope.Data.From)
	assert.Equal(t, shipped, envelope.Data.To)
	assert.Contains(t, envelope.Data.Files, initFile{Path: legacySkill, Action: "remove", Owner: "ext:go"})
	assert.Len(t, envelope.Data.Next, 2)
	noOp := upgradeIn(t, repoAssets(t), dir, "--json")
	require.Equal(t, 0, noOp.exit, noOp.stderr)
	assert.Contains(t, noOp.stdout, `"next": []`)
	dryRun := upgradeIn(t, repoAssets(t), dir, "--json", "--dry-run")
	require.Equal(t, 0, dryRun.exit, dryRun.stderr)
	assert.Contains(t, dryRun.stdout, `"dry_run": true`)
	assert.NotContains(t, dryRun.stdout, `"next"`)
}

func TestUpgradeRollsBackOnApplyError(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)
	locked := filepath.Join(dir, legacyStore, "rules/go")
	require.NoError(t, os.Chmod(locked, 0o555))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	before, tip := tree(t, dir), refTip(t, dir)

	got := upgradeIn(t, repoAssets(t), dir, "--yes")

	require.Equal(t, 1, got.exit, got.stderr)
	assert.Contains(t, got.stderr, "(unexpected_error)")
	assertNothingWritten(t, dir, before, tip)
}

func TestUpgradeMessagesNameTakeNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "unresolved conflict",
			args: []string{"--yes"},
			want: "  fix:   rerun with --keep <path> to keep the project's file, or --take-new <path> to take the new version\n",
		},
		{
			name: "path in both flags",
			args: []string{"--yes", "--keep", legacySkill, "--take-new", legacySkill},
			want: "error: upgrade rejected (invalid_value)\n" +
				"  check: --keep and --take-new resolve conflicts\n" +
				"  found: " + legacySkill + " is in both --keep and --take-new\n" +
				"  fix:   pass --keep or --take-new once per conflicting path\n" +
				"nothing was written\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := legacyRepo(t)
			writeFile(t, dir, legacySkill, "# Go, edited\n")

			got := upgradeIn(t, repoAssets(t), dir, tt.args...)

			assert.NotZero(t, got.exit)
			assert.Contains(t, got.stderr, tt.want)
			assert.NotContains(t, got.stderr, "--overwrite")
		})
	}
}
