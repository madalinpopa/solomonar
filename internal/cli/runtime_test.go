package cli

import (
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
)

const runtimeListHeader = "RUNTIME  STATUS     ON PATH  ADAPTER\n"

func TestRuntimeListWithoutManifest(t *testing.T) {
	gittest.PathWith(t, "codex")
	dir := gittest.NewRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runInRepo(t, dir, "runtime", "list")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, runtimeListHeader+
		"claude   available  no       CLAUDE.md, .claude/skills, .claude/agents\n"+
		"codex    available  yes      AGENTS.md (native), .agents/skills, .codex/agents\n"+
		"copilot  available  no       AGENTS.md (native), .github/skills, .github/agents\n", got.stdout)
	assert.Empty(t, got.stderr)
	assertNothingWritten(t, dir, before, tip)
}

// runtimeListRepo has claude and copilot installed while claude and codex are on PATH.
func runtimeListRepo(t *testing.T) string {
	t.Helper()
	gittest.PathWith(t, "claude", "codex")
	dir := gittest.NewRepo(t)
	got := runInRepo(t, dir, runtimeArgs("claude,copilot")...)
	require.Equal(t, 0, got.exit, got.stderr)
	return dir
}

func TestRuntimeListSeparatesInstallationFromPath(t *testing.T) {
	dir := runtimeListRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runInRepo(t, dir, "runtime", "list")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, runtimeListHeader+
		"claude   installed  yes      CLAUDE.md, .claude/skills, .claude/agents\n"+
		"codex    available  yes      AGENTS.md (native), .agents/skills, .codex/agents\n"+
		"copilot  installed  no       AGENTS.md (native), .github/skills, .github/agents\n", got.stdout)
	assertNothingWritten(t, dir, before, tip)
}

func TestRuntimeListJSON(t *testing.T) {
	dir := runtimeListRepo(t)

	got := runInRepo(t, dir, "runtime", "list", "--json")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "runtime list",
		"data": {"runtimes": [
			{"name": "claude", "status": "installed", "on_path": true,
			 "adapters": ["CLAUDE.md", ".claude/skills", ".claude/agents"]},
			{"name": "codex", "status": "available", "on_path": true,
			 "adapters": ["AGENTS.md", ".agents/skills", ".codex/agents"]},
			{"name": "copilot", "status": "installed", "on_path": false,
			 "adapters": ["AGENTS.md", ".github/skills", ".github/agents"]}
		]},
		"warnings": []
	}`, got.stdout)
	assert.Empty(t, got.stderr)
}

func TestRuntimeListAcceptsAnOlderFramework(t *testing.T) {
	dir := runtimeListRepo(t)
	m := readTestManifest(t, dir)
	m.FrameworkVersion = "0.1.7"
	writeTestManifest(t, dir, m)

	got := runInRepo(t, dir, "runtime", "list")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "copilot  installed  no ")
}

func TestRuntimeListFromNestedDirectoryAndWorktree(t *testing.T) {
	dir := runtimeListRepo(t)
	gittest.Git(t, dir, "add", "-A")
	gittest.Git(t, dir, "commit", "-m", "chore: initialize solomonar")
	nested := filepath.Join(dir, "docs", "solomonar")
	worktree := gittest.Worktree(t, dir, "feature")
	want := runInRepo(t, dir, "runtime", "list")
	require.Equal(t, 0, want.exit, want.stderr)

	for _, from := range []string{nested, worktree} {
		before, tip := tree(t, from), refTip(t, dir)
		head := gittest.Git(t, from, "rev-parse", "HEAD")

		got := runInRepo(t, from, "runtime", "list")

		require.Equal(t, 0, got.exit, got.stderr)
		assert.Equal(t, want.stdout, got.stdout)
		assertNothingWritten(t, from, before, tip)
		assert.Equal(t, head, gittest.Git(t, from, "rev-parse", "HEAD"))
	}
}

func TestRuntimeListRejections(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	fix := "  fix:   run solomonar help\nnothing was written\n"
	tests := []struct {
		name string
		dir  func(t *testing.T) string
		args []string
		exit int
		want string
	}{
		{
			name: "group alone", dir: func(*testing.T) string { return outside }, args: []string{"runtime"}, exit: 2,
			want: "error: unknown command \"runtime\" (unknown_command)\n" + fix,
		},
		{
			name: "unknown subcommand", dir: func(*testing.T) string { return outside }, args: []string{"runtime", "nope"},
			exit: 2, want: "error: unknown command \"runtime nope\" (unknown_command)\n" + fix,
		},
		{
			name: "argument", dir: gittest.NewRepo, args: []string{"runtime", "list", "claude"}, exit: 2,
			want: "error: unexpected argument \"claude\" (invalid_usage)\n" + fix,
		},
		{
			name: "unknown flag", dir: gittest.NewRepo, args: []string{"runtime", "list", "--all"}, exit: 2,
			want: "error: flag provided but not defined: -all (invalid_usage)\n" + fix,
		},
		{
			name: "outside a repository", dir: func(*testing.T) string { return outside }, args: []string{"runtime", "list"},
			exit: 3, want: "error: " + outside + " is not a Git repository (not_a_repository)\n" +
				"  fix:   run inside a Git repository or pass -C <path>\nnothing was written\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := runInRepo(t, tt.dir(t), tt.args...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Equal(t, tt.want, got.stderr)
		})
	}
}

func TestRuntimeListChecksTheManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		edit  func(t *testing.T, dir string)
		exit  int
		error string
	}{
		{name: "newer", exit: 3, error: "error: runtime list rejected (cli_outdated)\n", edit: func(t *testing.T, dir string) {
			t.Helper()
			m := readTestManifest(t, dir)
			m.FrameworkVersion = "99.0.0"
			writeTestManifest(t, dir, m)
		}},
		{name: "malformed", exit: 1, error: "error: runtime list failed (manifest_invalid)\n", edit: func(t *testing.T, dir string) {
			t.Helper()
			writeFile(t, dir, project.ManifestPath, "{")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := initializedRepo(t)
			tt.edit(t, dir)
			before := tree(t, dir)

			got := runInRepo(t, dir, "runtime", "list")

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, tt.error)
			assert.Equal(t, before, tree(t, dir))
		})
	}
}

func runtimeAddIn(t *testing.T, dir string, args ...string) result {
	t.Helper()
	return runInRepo(t, dir, append([]string{"runtime", "add"}, args...)...)
}

func runtimeCommit(verb, name string) string {
	return fmt.Sprintf(" && git commit -m \"chore: %s solomonar runtime %s\"", verb, name)
}

func TestRuntimeAddFromNone(t *testing.T) {
	// Explicit addition must work without any runtime executable.
	gittest.PathWith(t)
	tests := []struct {
		name, paths, session string
		created              int
	}{
		{
			name: "claude", paths: "CLAUDE.md .claude .solomonar", created: 10,
			session: "  2. Start a top-level Claude Code session in %s:\n       /solomonar Explore the next useful flow.\n",
		},
		{
			name: "codex", paths: ".agents .codex .solomonar", created: 10,
			session: "  2. Start a top-level Codex session in %s:\n       $solomonar Explore the next useful flow.\n",
		},
		{
			name: "copilot", paths: copilotCoreFiles + " .solomonar", created: 9,
			session: "  2. Start a top-level GitHub Copilot CLI session in %s:\n       /solomonar Explore the next useful flow.\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initializedRepo(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runtimeAddIn(t, dir, tt.name, "--yes")

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, "Added runtime "+tt.name+" in "+dir+"\n"+
				fmt.Sprintf("  %d files created, 1 updated, 17 unchanged, 1 skipped\n", tt.created)+
				"\nNext:\n  1. git add "+tt.paths+runtimeCommit("add", tt.name)+"\n"+
				fmt.Sprintf(tt.session, dir), got.stdout)
			assert.Empty(t, got.stderr)
			manifest := manifestOf(t, dir)
			assert.Equal(t, map[string]project.Runtime{tt.name: {}}, manifest.Runtimes)
			after := tree(t, dir)
			for path, content := range before {
				if path != project.ManifestPath {
					assert.Equal(t, content, after[path], path)
				}
			}
			for path, content := range after {
				if _, existed := before[path]; !existed && content != "/" {
					assert.Equal(t, "runtime:"+tt.name, manifest.Files[path].Owner, path)
				}
			}
			assert.Equal(t, tip, refTip(t, dir))
		})
	}
}

func TestRuntimeAddBesideAnotherRuntime(t *testing.T) {
	gittest.PathWith(t)
	dir := claudeRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runtimeAddIn(t, dir, "codex", "--yes", "--json")

	require.Equal(t, 0, got.exit, got.stderr)
	var envelope struct {
		Command string         `json:"command"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(got.stdout), &envelope))
	assert.Equal(t, "runtime add", envelope.Command)
	assert.Equal(t, dir, envelope.Data["project"])
	assert.Equal(t, "codex", envelope.Data["runtime"])
	assert.NotContains(t, envelope.Data, "adversarial_model")
	assert.NotContains(t, envelope.Data, "dry_run")
	assert.Equal(t, []any{"git add .agents .codex .solomonar" + runtimeCommit("add", "codex")}, envelope.Data["next"])
	assert.Contains(t, envelope.Data["files"], map[string]any{
		"path": ".agents/skills/solomonar/agents/openai.yaml", "action": "create", "owner": "runtime:codex",
	})
	assert.Equal(t, map[string]project.Runtime{"claude": {AdversarialModel: "opus"}, "codex": {}},
		manifestOf(t, dir).Runtimes)
	after := tree(t, dir)
	for path, content := range before {
		if path != project.ManifestPath {
			assert.Equal(t, content, after[path], path)
		}
	}
	assert.Equal(t, tip, refTip(t, dir))
}

func TestRuntimeAddKeepsExtensionsAndDecisions(t *testing.T) {
	gittest.PathWith(t)
	dir := gittest.NewRepo(t)
	got := runInRepo(t, dir, "init", "--runtime", "claude", "--ext", "go,extension-authoring", "--yes")
	require.Equal(t, 0, got.exit, got.stderr)
	writeFile(t, dir, "docs/solomonar/extensions/money/extension.json", moneyReviewerJSON)
	writeFile(t, dir, "docs/solomonar/extensions/money/money.md", moneyReviewer)
	got = addIn(t, dir, "money", "--yes")
	require.Equal(t, 0, got.exit, got.stderr)
	before, tip, recorded := tree(t, dir), refTip(t, dir), manifestOf(t, dir)

	got = runtimeAddIn(t, dir, "codex", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	after, manifest := tree(t, dir), manifestOf(t, dir)
	assert.Equal(t, recorded.Extensions, manifest.Extensions)
	for path, content := range before {
		if path != project.ManifestPath {
			assert.Equal(t, content, after[path], path)
		}
	}
	for _, path := range []string{
		".agents/skills/go/SKILL.md", ".agents/skills/extension-authoring/SKILL.md",
		".agents/skills/extension-authoring/agents/openai.yaml", ".codex/agents/money.toml",
	} {
		assert.Contains(t, after, path)
		assert.Equal(t, "runtime:codex", manifest.Files[path].Owner, path)
	}
	assert.NotContains(t, after, ".agents/skills/go/agents/openai.yaml")
	assert.NotContains(t, after, ".codex/hooks.json")
	assert.NotContains(t, after, ".claude/settings.json")
	assert.Equal(t, tip, refTip(t, dir))
}

func TestRuntimeAddUpdatesTheModel(t *testing.T) {
	dir := claudeRepo(t)
	before := tree(t, dir)

	got := runtimeAddIn(t, dir, "claude", "--adversarial-model", "sonnet", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  2 files updated, 26 unchanged, 1 skipped\n")
	assert.Contains(t, got.stdout, "git add .claude .solomonar"+runtimeCommit("add", "claude"))
	after := tree(t, dir)
	assert.Contains(t, after[".claude/agents/adversarial-reviewer.md"], "\nmodel: sonnet\n")
	for path, content := range before {
		if path != project.ManifestPath && path != ".claude/agents/adversarial-reviewer.md" {
			assert.Equal(t, content, after[path], path)
		}
	}
	assert.Equal(t, map[string]project.Runtime{"claude": {AdversarialModel: "sonnet"}}, manifestOf(t, dir).Runtimes)
}

func TestRuntimeAddRepeatedIsNoOp(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"claude"}, {"claude", "--adversarial-model", "opus"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			dir := claudeRepo(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runtimeAddIn(t, dir, args...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, "Already installed: claude; nothing to do.\n  28 files unchanged, 1 skipped\n", got.stdout)
			assertNothingWritten(t, dir, before, tip)

			asJSON := runtimeAddIn(t, dir, append(args, "--json")...)
			require.Equal(t, 0, asJSON.exit, asJSON.stderr)
			assert.Contains(t, asJSON.stdout, `"next": []`)
			assert.Contains(t, asJSON.stdout, `"adversarial_model": "opus"`)
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestRuntimeAddLeavesAnUnmanagedClaudeFile(t *testing.T) {
	t.Parallel()
	dir := initializedRepo(t)
	writeFile(t, dir, "CLAUDE.md", "project notes\n")

	got := runtimeAddIn(t, dir, "claude", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "git add .claude .solomonar"+runtimeCommit("add", "claude"))
	assert.Equal(t, "project notes\n", tree(t, dir)["CLAUDE.md"])
	assert.NotContains(t, manifestOf(t, dir).Files, "CLAUDE.md")
}

func TestRuntimeAddDryRun(t *testing.T) {
	t.Parallel()
	dir := initializedRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runtimeAddIn(t, dir, "copilot", "--dry-run")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, "Plan\n"), got.stdout)
	assert.Contains(t, got.stdout, "  create     .github/skills/spec/SKILL.md\n")
	assert.Contains(t, got.stdout, "  update     .solomonar/manifest.json\n")
	assert.True(t, strings.HasSuffix(got.stdout, "Would add runtime copilot in "+dir+"\n"+
		"  9 files created, 1 updated, 17 unchanged, 1 skipped\n"), got.stdout)
	assertNothingWritten(t, dir, before, tip)

	asJSON := runtimeAddIn(t, dir, "copilot", "--dry-run", "--json", "--adversarial-model", "gpt-5")
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(asJSON.stdout), &envelope))
	assert.Equal(t, true, envelope.Data["dry_run"])
	assert.Equal(t, "gpt-5", envelope.Data["adversarial_model"])
	assert.NotContains(t, envelope.Data, "next")
	assertNothingWritten(t, dir, before, tip)
}

func TestRuntimeAddConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("terminal", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name, input, stdout string
			applied             bool
		}{
			{name: "declined", input: "n\n", stdout: "Cancelled; nothing was written.\n"},
			{name: "accepted", input: "\n", stdout: "Added runtime copilot in ", applied: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := initializedRepo(t)
				before, tip := tree(t, dir), refTip(t, dir)

				got := runInteractive(t, repoAssets(t), dir, tt.input, "runtime", "add", "copilot")

				require.Equal(t, 0, got.exit, got.stderr)
				assert.Equal(t, "Plan\n"+
					"  runtime copilot\n"+
					"    create     .github/skills/ (4 skill stubs: 4 core)\n"+
					"    create     .github/agents/ (5 role stubs)\n"+
					"  manifest\n"+
					"    update     .solomonar/manifest.json\n"+
					"Proceed? [Y/n] \n", got.stderr)
				assert.True(t, strings.HasPrefix(got.stdout, tt.stdout), got.stdout)
				if tt.applied {
					assert.Contains(t, manifestOf(t, dir).Runtimes, "copilot")
				} else {
					assertNothingWritten(t, dir, before, tip)
				}
			})
		}
	})
	for _, args := range [][]string{{"copilot"}, {"copilot", "--json"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			dir := initializedRepo(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runtimeAddIn(t, dir, args...)

			assert.Equal(t, 2, got.exit)
			assert.Contains(t, got.stdout+got.stderr, "interactive_required")
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestRuntimeAddConflicts(t *testing.T) {
	t.Parallel()

	const file = ".github/skills/spec/SKILL.md"
	dir := initializedRepo(t)
	writeFile(t, dir, file, "project instructions\n")
	before, tip := tree(t, dir), refTip(t, dir)

	conflict := runtimeAddIn(t, dir, "copilot", "--yes")
	assert.Equal(t, 4, conflict.exit)
	assert.True(t, strings.HasPrefix(conflict.stdout, "Plan\n"), conflict.stdout)
	assert.Contains(t, conflict.stdout, file)
	assert.Contains(t, conflict.stderr, "error: runtime add found 1 conflict (file_conflict)\n")
	assertNothingWritten(t, dir, before, tip)

	notConflict := runtimeAddIn(t, dir, "copilot", "--yes", "--keep", file, "--keep", "AGENTS.md")
	assert.Equal(t, 2, notConflict.exit)
	assert.Contains(t, notConflict.stderr, "invalid_value")
	assertNothingWritten(t, dir, before, tip)

	clean := initializedRepo(t)
	require.Equal(t, 0, runtimeAddIn(t, clean, "copilot", "--yes").exit)
	offer := tree(t, clean)[file]
	for _, resolution := range []string{"--keep", "--overwrite"} {
		t.Run(resolution, func(t *testing.T) {
			t.Parallel()
			dir := initializedRepo(t)
			writeFile(t, dir, file, "project instructions\n")

			got := runtimeAddIn(t, dir, "copilot", "--yes", resolution, file)

			require.Equal(t, 0, got.exit, got.stderr)
			entry := manifestOf(t, dir).Files[file]
			assert.Equal(t, "runtime:copilot", entry.Owner)
			want := offer
			if resolution == "--keep" {
				want = "project instructions\n"
			}
			assert.Equal(t, want, tree(t, dir)[file])
			assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256([]byte(offer))), entry.SHA256)
		})
	}
}

func TestRuntimeAddRejections(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	fix := "  fix:   run solomonar help\nnothing was written\n"
	outdated := func(t *testing.T) string {
		t.Helper()
		dir := initializedRepo(t)
		m := readTestManifest(t, dir)
		m.FrameworkVersion = "0.1.7"
		writeTestManifest(t, dir, m)
		return dir
	}
	tests := []struct {
		name string
		dir  func(t *testing.T) string
		args []string
		exit int
		want string
	}{
		{
			name: "missing name", dir: initializedRepo, args: nil, exit: 2,
			want: "error: runtime name is required (invalid_usage)\n" + fix,
		},
		{
			name: "extra name", dir: initializedRepo, args: []string{"claude", "codex"}, exit: 2,
			want: "error: unexpected argument \"codex\" (invalid_usage)\n" + fix,
		},
		{
			name: "unknown flag", dir: initializedRepo, args: []string{"claude", "--model", "x"}, exit: 2,
			want: "error: flag provided but not defined: -model (invalid_usage)\n" + fix,
		},
		{
			name: "none", dir: func(*testing.T) string { return outside }, args: []string{"none"}, exit: 2,
			want: "error: runtime add rejected (unknown_runtime)\n" +
				"  check: <runtime> names a known runtime\n" +
				"  found: \"none\" is not a runtime; known: claude, codex, copilot\n" +
				"  fix:   pass claude, codex, or copilot\nnothing was written\n",
		},
		{
			name: "unknown name", dir: initializedRepo, args: []string{"gemini", "--adversarial-model", ""}, exit: 2,
			want: "error: runtime add rejected (unknown_runtime)\n" +
				"  check: <runtime> names a known runtime\n" +
				"  found: \"gemini\" is not a runtime; known: claude, codex, copilot\n" +
				"  fix:   pass claude, codex, or copilot\nnothing was written\n",
		},
		{
			name: "empty model", dir: func(*testing.T) string { return outside },
			args: []string{"claude", "--adversarial-model="}, exit: 2,
			want: "error: runtime add rejected (invalid_value)\n" +
				"  check: --adversarial-model names a model\n" +
				"  found: empty model\n" +
				"  fix:   pass a model or omit --adversarial-model\nnothing was written\n",
		},
		{
			name: "outside a repository", dir: func(*testing.T) string { return outside }, args: []string{"claude"},
			exit: 3, want: "error: " + outside + " is not a Git repository (not_a_repository)\n" +
				"  fix:   run inside a Git repository or pass -C <path>\nnothing was written\n",
		},
		{
			name: "not initialized", dir: gittest.NewRepo, args: []string{"claude"}, exit: 3,
			want: "error: runtime add rejected (not_initialized)\n",
		},
		{
			name: "older framework", dir: outdated, args: []string{"claude", "--yes"}, exit: 3,
			want: "error: runtime add rejected (framework_outdated)\n" +
				"  check: the project's framework matches this CLI\n" +
				"  found: manifest framework 0.1.7, this CLI ships " + shipped + "\n" +
				"  fix:   run solomonar upgrade\nnothing was written\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.dir(t)
			before := tree(t, dir)

			got := runtimeAddIn(t, dir, tt.args...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			if strings.HasSuffix(tt.want, "\nnothing was written\n") {
				assert.Equal(t, tt.want, got.stderr)
			} else {
				assert.Contains(t, got.stderr, tt.want)
			}
			assert.Equal(t, before, tree(t, dir))
		})
	}
}

func runtimeRemoveIn(t *testing.T, dir string, args ...string) result {
	t.Helper()
	return runInRepo(t, dir, append([]string{"runtime", "remove"}, args...)...)
}

// allRuntimesRepo initializes a project with every runtime and a Claude model.
func allRuntimesRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.NewRepo(t)
	got := runInRepo(t, dir, runtimeArgs("claude,codex,copilot", "--adversarial-model", "claude=opus")...)
	require.Equal(t, 0, got.exit, got.stderr)
	return dir
}

func TestRuntimeRemoveFromAllRuntimes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, paths string
		left        map[string]project.Runtime
	}{
		{
			name: "claude", paths: ".claude CLAUDE.md .solomonar",
			left: map[string]project.Runtime{"codex": {}, "copilot": {}},
		},
		{
			name: "codex", paths: ".agents .codex .solomonar",
			left: map[string]project.Runtime{"claude": {AdversarialModel: "opus"}, "copilot": {}},
		},
		{
			name: "copilot", paths: ".github/agents/adversarial-reviewer.agent.md .github/agents/correctness-reviewer.agent.md " +
				".github/agents/implementer.agent.md .github/agents/security-reviewer.agent.md " +
				".github/agents/spec-reviewer.agent.md .github/skills/prototype/SKILL.md .github/skills/solomonar/SKILL.md " +
				".github/skills/spec/SKILL.md .github/skills/work-spec/SKILL.md .solomonar",
			left: map[string]project.Runtime{"claude": {AdversarialModel: "opus"}, "codex": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := allRuntimesRepo(t)
			before, tip, recorded := tree(t, dir), refTip(t, dir), manifestOf(t, dir)
			var owned []string
			for path, file := range recorded.Files {
				if file.Owner == "runtime:"+tt.name {
					owned = append(owned, path)
				}
			}
			require.NotEmpty(t, owned)

			got := runtimeRemoveIn(t, dir, tt.name, "--yes")

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Empty(t, got.stderr)
			assert.True(t, strings.HasPrefix(got.stdout, "Removed runtime "+tt.name+" in "+dir+"\n"), got.stdout)
			assert.Contains(t, got.stdout, fmt.Sprintf("\n  1 file updated, %d removed, ", len(owned)))
			assert.True(t, strings.HasSuffix(got.stdout,
				"\nNext:\n  1. git add "+tt.paths+runtimeCommit("remove", tt.name)+"\n"), got.stdout)
			manifest, after := manifestOf(t, dir), tree(t, dir)
			assert.Equal(t, tt.left, manifest.Runtimes)
			for _, path := range owned {
				assert.NotContains(t, after, path)
				assert.NotContains(t, manifest.Files, path)
			}
			for path, content := range before {
				if path != project.ManifestPath && !slices.Contains(owned, path) && content != "/" {
					assert.Equal(t, content, after[path], path)
				}
			}
			assert.Equal(t, tip, refTip(t, dir))
		})
	}
}

func TestRuntimeRemoveNotInstalledIsNoOp(t *testing.T) {
	t.Parallel()
	dir := claudeRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runtimeRemoveIn(t, dir, "codex")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Not installed: codex; nothing to do.\n", got.stdout)
	assert.Empty(t, got.stderr)
	assertNothingWritten(t, dir, before, tip)

	preview := runtimeRemoveIn(t, dir, "codex", "--dry-run")
	require.Equal(t, 0, preview.exit, preview.stderr)
	assert.Equal(t, got.stdout, preview.stdout)

	asJSON := runtimeRemoveIn(t, dir, "codex", "--json")
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	var envelope struct {
		Command string         `json:"command"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(asJSON.stdout), &envelope))
	assert.Equal(t, "runtime remove", envelope.Command)
	assert.Equal(t, map[string]any{"project": dir, "runtime": "codex", "files": []any{}, "next": []any{}}, envelope.Data)
	assertNothingWritten(t, dir, before, tip)
}

func TestRuntimeRemoveRejections(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	fix := "  fix:   run solomonar help\nnothing was written\n"
	outdated := func(t *testing.T) string {
		t.Helper()
		dir := claudeRepo(t)
		m := readTestManifest(t, dir)
		m.FrameworkVersion = "0.1.7"
		writeTestManifest(t, dir, m)
		return dir
	}
	notConflict := func(path string) string {
		return "error: runtime remove rejected (invalid_value)\n" +
			"  check: --keep and --overwrite resolve conflicts\n" +
			"  found: " + path + " is not a conflict\n"
	}
	tests := []struct {
		name string
		dir  func(t *testing.T) string
		args []string
		exit int
		want string
	}{
		{
			name: "missing name", dir: claudeRepo, args: nil, exit: 2,
			want: "error: runtime name is required (invalid_usage)\n" + fix,
		},
		{
			name: "extra name", dir: claudeRepo, args: []string{"claude", "codex"}, exit: 2,
			want: "error: unexpected argument \"codex\" (invalid_usage)\n" + fix,
		},
		{
			name: "model flag", dir: claudeRepo, args: []string{"claude", "--adversarial-model", "x"}, exit: 2,
			want: "error: flag provided but not defined: -adversarial-model (invalid_usage)\n" + fix,
		},
		{
			name: "none", dir: func(*testing.T) string { return outside }, args: []string{"none"}, exit: 2,
			want: "error: runtime remove rejected (unknown_runtime)\n" +
				"  check: <runtime> names a known runtime\n" +
				"  found: \"none\" is not a runtime; known: claude, codex, copilot\n" +
				"  fix:   pass claude, codex, or copilot\nnothing was written\n",
		},
		{
			name: "outside a repository", dir: func(*testing.T) string { return outside }, args: []string{"claude"},
			exit: 3, want: "error: " + outside + " is not a Git repository (not_a_repository)\n" +
				"  fix:   run inside a Git repository or pass -C <path>\nnothing was written\n",
		},
		{
			name: "not initialized", dir: gittest.NewRepo, args: []string{"codex"}, exit: 3,
			want: "error: runtime remove rejected (not_initialized)\n",
		},
		{
			name: "older framework", dir: outdated, args: []string{"codex", "--keep", "AGENTS.md"}, exit: 3,
			want: "error: runtime remove rejected (framework_outdated)\n",
		},
		{
			name: "keep on the no-op", dir: claudeRepo, args: []string{"codex", "--keep", "AGENTS.md"}, exit: 2,
			want: notConflict("AGENTS.md"),
		},
		{
			name: "overwrite on the no-op", dir: claudeRepo, args: []string{"codex", "--overwrite", ".codex/x"}, exit: 2,
			want: notConflict(".codex/x"),
		},
		{
			name: "keep without a conflict", dir: claudeRepo, args: []string{"claude", "--keep", "CLAUDE.md"}, exit: 2,
			want: notConflict("CLAUDE.md"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.dir(t)
			before := tree(t, dir)

			got := runtimeRemoveIn(t, dir, tt.args...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			if strings.HasSuffix(tt.want, "\nnothing was written\n") {
				assert.Equal(t, tt.want, got.stderr)
			} else {
				assert.Contains(t, got.stderr, tt.want)
			}
			assert.Equal(t, before, tree(t, dir))
		})
	}
}

func TestRuntimeRemoveConflicts(t *testing.T) {
	t.Parallel()

	const file = ".claude/skills/spec/SKILL.md"
	edited := func(t *testing.T) string {
		t.Helper()
		dir := allRuntimesRepo(t)
		writeFile(t, dir, file, "project instructions\n")
		return dir
	}

	t.Run("unresolved", func(t *testing.T) {
		t.Parallel()
		dir := edited(t)
		before, tip := tree(t, dir), refTip(t, dir)

		got := runtimeRemoveIn(t, dir, "claude", "--yes")

		assert.Equal(t, 4, got.exit)
		assert.True(t, strings.HasPrefix(got.stdout, "Plan\n"), got.stdout)
		assert.Contains(t, got.stdout, "  conflict   "+file)
		assert.Contains(t, got.stderr, "error: runtime remove found 1 conflict (file_conflict)\n")
		assertNothingWritten(t, dir, before, tip)
	})
	t.Run("--keep", func(t *testing.T) {
		t.Parallel()
		dir := edited(t)

		got := runtimeRemoveIn(t, dir, "claude", "--yes", "--keep", file)

		require.Equal(t, 0, got.exit, got.stderr)
		assert.Equal(t, "project instructions\n", tree(t, dir)[file])
		assert.NotContains(t, manifestOf(t, dir).Files, file)
		assert.NotContains(t, manifestOf(t, dir).Runtimes, "claude")
		assert.Contains(t, got.stdout, " 1 kept, ")
	})
	t.Run("--overwrite", func(t *testing.T) {
		t.Parallel()
		dir := edited(t)

		got := runtimeRemoveIn(t, dir, "claude", "--yes", "--overwrite", file)

		require.Equal(t, 0, got.exit, got.stderr)
		assert.NotContains(t, tree(t, dir), file)
		assert.NotContains(t, tree(t, dir), ".claude/skills")
		assert.NotContains(t, manifestOf(t, dir).Files, file)
	})
	t.Run("--overwrite of a directory", func(t *testing.T) {
		t.Parallel()
		dir := allRuntimesRepo(t)
		require.NoError(t, os.Remove(filepath.Join(dir, file)))
		writeFile(t, dir, file+"/notes.md", "kept\n")
		before, tip := tree(t, dir), refTip(t, dir)

		got := runtimeRemoveIn(t, dir, "claude", "--yes", "--overwrite", file)

		assert.Equal(t, 2, got.exit)
		assert.Contains(t, got.stderr, "  found: "+file+" is not a regular file\n")
		assertNothingWritten(t, dir, before, tip)

		kept := runtimeRemoveIn(t, dir, "claude", "--yes", "--keep", file)
		require.Equal(t, 0, kept.exit, kept.stderr)
		assert.Equal(t, "kept\n", tree(t, dir)[file+"/notes.md"])
	})
}

func TestRuntimeRemoveUntracksAbsentFiles(t *testing.T) {
	t.Parallel()
	const absent = ".github/agents/implementer.agent.md"
	dir := allRuntimesRepo(t)
	require.NoError(t, os.Remove(filepath.Join(dir, absent)))

	got := runtimeRemoveIn(t, dir, "copilot", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "\n  1 file updated, 8 removed, ")
	assert.NotContains(t, got.stdout, absent)
	assert.NotContains(t, manifestOf(t, dir).Files, absent)
	assert.NotContains(t, tree(t, dir), absent)
}

func TestRuntimeRemoveKeepsUserFilesAndTheFramework(t *testing.T) {
	t.Parallel()
	dir := gittest.NewRepo(t)
	userFiles := map[string]string{
		"CLAUDE.md":                      "project notes\n",
		".github/workflows/ci.yml":       "on: push\n",
		".github/skills/mine/SKILL.md":   "mine\n",
		".claude/settings.json":          "{}\n",
		".codex/config.toml":             "model = \"x\"\n",
		".agents/skills/mine/SKILL.md":   "mine\n",
		".github/agents/mine.agent.md":   "mine\n",
		".claude/agents/mine.md":         "mine\n",
		".codex/agents/mine.toml":        "name = \"mine\"\n",
		"docs/solomonar/extensions/x.md": "unrelated\n",
	}
	for path, content := range userFiles {
		writeFile(t, dir, path, content)
	}
	got := runInRepo(t, dir, "init", "--runtime", "claude,codex,copilot", "--ext", "go", "--yes")
	require.Equal(t, 0, got.exit, got.stderr)
	agents := tree(t, dir)["AGENTS.md"] + "\nOur conventions.\n"
	writeFile(t, dir, "AGENTS.md", agents)
	recorded, tip := manifestOf(t, dir), refTip(t, dir)
	require.NotContains(t, recorded.Files, "CLAUDE.md")

	for _, name := range []string{"copilot", "claude", "codex"} {
		got := runtimeRemoveIn(t, dir, name, "--yes")
		require.Equal(t, 0, got.exit, got.stderr)
		assert.NotContains(t, got.stdout, "CLAUDE.md")
	}

	after, manifest := tree(t, dir), manifestOf(t, dir)
	for path, content := range userFiles {
		assert.Equal(t, content, after[path], path)
	}
	assert.Equal(t, agents, after["AGENTS.md"])
	assert.Empty(t, manifest.Runtimes)
	assert.Equal(t, recorded.Extensions, manifest.Extensions)
	assert.Equal(t, recorded.FrameworkVersion, manifest.FrameworkVersion)
	for path, file := range recorded.Files {
		if !strings.HasPrefix(file.Owner, "runtime:") {
			assert.Equal(t, file, manifest.Files[path], path)
			assert.Contains(t, after, path)
		} else {
			assert.NotContains(t, manifest.Files, path)
			assert.NotContains(t, after, path)
		}
	}
	assert.Equal(t, tip, refTip(t, dir))

	repeated := runtimeRemoveIn(t, dir, "codex")
	require.Equal(t, 0, repeated.exit, repeated.stderr)
	assert.Equal(t, "Not installed: codex; nothing to do.\n", repeated.stdout)
	assertNothingWritten(t, dir, after, tip)
	status := runInRepo(t, dir, "runtime", "list")
	require.Equal(t, 0, status.exit, status.stderr)
	assert.NotContains(t, status.stdout, "installed ")
}

func TestRuntimeRemoveDryRunAndJSON(t *testing.T) {
	t.Parallel()
	dir := allRuntimesRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runtimeRemoveIn(t, dir, "claude", "--dry-run")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, "Plan\n"), got.stdout)
	assert.Contains(t, got.stdout, "  remove     CLAUDE.md\n")
	assert.Contains(t, got.stdout, "  remove     .claude/agents/adversarial-reviewer.md\n")
	assert.Contains(t, got.stdout, "  update     .solomonar/manifest.json\n")
	assert.Contains(t, got.stdout, "Would remove runtime claude in "+dir+"\n  1 file updated, 10 removed, ")
	assertNothingWritten(t, dir, before, tip)

	preview := runtimeRemoveIn(t, dir, "claude", "--dry-run", "--json")
	require.Equal(t, 0, preview.exit, preview.stderr)
	var envelope struct {
		Command string         `json:"command"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(preview.stdout), &envelope))
	assert.Equal(t, true, envelope.Data["dry_run"])
	assert.NotContains(t, envelope.Data, "next")
	assertNothingWritten(t, dir, before, tip)

	applied := runtimeRemoveIn(t, dir, "claude", "--yes", "--json")
	require.Equal(t, 0, applied.exit, applied.stderr)
	envelope.Data = nil
	require.NoError(t, json.Unmarshal([]byte(applied.stdout), &envelope))
	assert.Equal(t, "runtime remove", envelope.Command)
	assert.Equal(t, "claude", envelope.Data["runtime"])
	assert.NotContains(t, envelope.Data, "adversarial_model")
	assert.NotContains(t, envelope.Data, "dry_run")
	assert.Contains(t, envelope.Data["files"], map[string]any{
		"path": "CLAUDE.md", "action": "remove", "owner": "runtime:claude",
	})
	assert.Equal(t, []any{"git add .claude CLAUDE.md .solomonar" + runtimeCommit("remove", "claude")},
		envelope.Data["next"])
}

func TestRuntimeRemoveConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("declined", func(t *testing.T) {
		t.Parallel()
		dir := allRuntimesRepo(t)
		before, tip := tree(t, dir), refTip(t, dir)

		got := runInteractive(t, repoAssets(t), dir, "n\n", "runtime", "remove", "codex")

		require.Equal(t, 0, got.exit, got.stderr)
		assert.True(t, strings.HasPrefix(got.stderr, "Plan\n"), got.stderr)
		assert.True(t, strings.HasSuffix(got.stderr, "Proceed? [Y/n] \n"), got.stderr)
		assert.Equal(t, "Cancelled; nothing was written.\n", got.stdout)
		assertNothingWritten(t, dir, before, tip)
	})
	for _, args := range [][]string{{"codex"}, {"codex", "--json"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			dir := allRuntimesRepo(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runtimeRemoveIn(t, dir, args...)

			assert.Equal(t, 2, got.exit)
			assert.Contains(t, got.stdout+got.stderr, "interactive_required")
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestRuntimeRemoveRollsBackAFailedApply(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := allRuntimesRepo(t)
	solomonar := filepath.Join(dir, ".solomonar")
	require.NoError(t, os.Chmod(solomonar, 0o500)) // the manifest, the last change, cannot be written
	t.Cleanup(func() { _ = os.Chmod(solomonar, 0o755) })
	before, tip, modes := tree(t, dir), refTip(t, dir), dirModes(t, dir)

	got := runtimeRemoveIn(t, dir, "claude", "--yes")

	assert.Equal(t, 1, got.exit)
	assert.Empty(t, got.stdout)
	assert.Contains(t, got.stderr, "error: apply .solomonar/manifest.json: ")
	assert.True(t, strings.HasSuffix(got.stderr, "nothing was written\n"), got.stderr)
	assertNothingWritten(t, dir, before, tip)
	assert.Equal(t, modes, dirModes(t, dir))
	assert.Contains(t, before, ".claude/skills/spec")
}
