package cli

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
)

const noManifest = "No Solomonar manifest; nothing to uninstall automatically.\n"

func uninstallIn(t *testing.T, dir string, args ...string) result {
	t.Helper()
	return runInRepo(t, dir, append([]string{"uninstall"}, args...)...)
}

func TestHelpForUninstall(t *testing.T) {
	t.Parallel()

	viaHelp := runCLI(t, repoAssets(t), "help", "uninstall")
	viaFlag := runCLI(t, repoAssets(t), "uninstall", "-h")

	require.Equal(t, 0, viaHelp.exit, viaHelp.stderr)
	assert.Equal(t, "Usage: solomonar uninstall [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]\n\n"+
		"Remove the framework core, the installed catalog extensions, the runtime adapters, the managed\n"+
		"part of AGENTS.md, the docs/README.md index line, and the manifest from this checkout. --keep\n"+
		"leaves a conflicting file in place, --overwrite removes it; --dry-run prints the plan and writes\n"+
		"nothing. Project extensions, decisions, project notes, and the registry are kept.\n\n"+
		"Global flags: see 'solomonar help'.\n", viaHelp.stdout)
	assert.Equal(t, viaHelp.stdout, viaFlag.stdout)
}

func TestUninstallWithoutManifestIsANoOp(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{nil, {"--dry-run"}, {"--yes"}} {
		dir := gittest.NewRepo(t)
		writeFile(t, dir, "docs/solomonar/rules.md", "rules\n")
		before := tree(t, dir)

		got := uninstallIn(t, dir, args...)

		require.Equal(t, 0, got.exit, got.stderr)
		assert.Equal(t, noManifest, got.stdout)
		assert.Empty(t, got.stderr)
		assert.Equal(t, before, tree(t, dir))
	}
}

func TestUninstallWithoutManifestJSON(t *testing.T) {
	t.Parallel()
	dir := gittest.NewRepo(t)

	got := uninstallIn(t, dir, "--json")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "uninstall",
		"data": {"project": "`+dir+`", "already_absent": true, "files": [], "preserved": [], "next": []},
		"warnings": []
	}`, got.stdout)
	assert.Empty(t, got.stderr)
}

func TestUninstallRejectsUsage(t *testing.T) {
	t.Parallel()
	dir := gittest.NewRepo(t)

	tests := []struct {
		name  string
		args  []string
		exit  int
		error string
	}{
		{
			name: "argument", args: []string{"extra"}, exit: 2,
			error: "error: unexpected argument \"extra\" (invalid_usage)\n  fix:   run solomonar help\n",
		},
		{
			name: "keep without manifest", args: []string{"--keep", "AGENTS.md"}, exit: 2,
			error: "error: uninstall rejected (invalid_value)\n" +
				"  check: --keep and --overwrite resolve conflicts\n" +
				"  found: AGENTS.md is not a conflict\n" +
				"  fix:   pass --keep or --overwrite once per conflicting path\n",
		},
		{
			name: "keep and overwrite", args: []string{"--keep", "a.md", "--overwrite", "a.md"}, exit: 2,
			error: "error: uninstall rejected (invalid_value)\n" +
				"  check: --keep and --overwrite resolve conflicts\n" +
				"  found: a.md is in both --keep and --overwrite\n" +
				"  fix:   pass --keep or --overwrite once per conflicting path\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := uninstallIn(t, dir, tt.args...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Equal(t, tt.error+"nothing was written\n", got.stderr)
		})
	}
}

func TestUninstallOutsideARepository(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	got := runCLI(t, repoAssets(t), "-C", dir, "uninstall")

	assert.Equal(t, 3, got.exit)
	assert.Contains(t, got.stderr, "(not_a_repository)")
}

const (
	indexLine   = "- `docs/solomonar/`: framework rules, roles, and skills\n"
	notes       = "## Project notes\n\n<!-- Project-owned instructions; preserved by framework upgrades. Keep this heading unchanged. -->\nUse tabs.\n"
	moneySource = "docs/solomonar/extensions/money"
)

// uninstallRepo is a project with the claude and codex runtimes, the catalog extension go, the
// project review agent money, project notes, an unrelated runtime file, and a registry.
func uninstallRepo(t *testing.T) string {
	t.Helper()
	dir := moneyRepo(t)
	got := runInRepo(t, dir, "extension", "add", "go", "money", "--yes")
	require.Equal(t, 0, got.exit, got.stderr)
	writeFile(t, dir, "AGENTS.md", readTestFile(t, dir, "AGENTS.md")+"Use tabs.\n")
	writeFile(t, dir, ".claude/settings.local.json", "{}\n")
	return dir
}

// uninstalledTree is what a full uninstall leaves of the before tree of uninstallRepo.
func uninstalledTree(before map[string]string) map[string]string {
	want := map[string]string{
		".agents": "/", ".claude": "/", ".codex": "/", "docs": "/", "docs/solomonar": "/",
		"docs/solomonar/extensions": "/", moneySource: "/",
		"AGENTS.md":      notes,
		"docs/README.md": strings.Replace(before["docs/README.md"], indexLine, "", 1),
	}
	for _, name := range []string{
		".claude/settings.local.json", "docs/decisions.md", "docs/stack.md",
		moneySource + "/extension.json", moneySource + "/money.md",
	} {
		want[name] = before[name]
	}
	return want
}

func uninstalled(dir string) string {
	return "Uninstalled solomonar from " + dir + "\n" +
		"  2 files updated, 52 removed, 1 skipped\n" +
		"  kept: AGENTS.md, docs/README.md, " + moneySource + "\n" +
		"  The registry (refs/heads/workflow-state) and other worktrees are unchanged.\n\n" +
		"Next:\n  1. Review the removal with git status, then commit it.\n"
}

func TestUninstallRemovesTheInstallation(t *testing.T) {
	t.Parallel()
	dir := uninstallRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)
	require.NotEmpty(t, tip)

	got := uninstallIn(t, dir, "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, uninstalled(dir), got.stdout)
	assert.Empty(t, got.stderr)
	assert.Equal(t, uninstalledTree(before), tree(t, dir))
	assert.Equal(t, tip, refTip(t, dir))

	again := uninstallIn(t, dir, "--yes")
	require.Equal(t, 0, again.exit, again.stderr)
	assert.Equal(t, noManifest, again.stdout)
}

func TestUninstallFindsTheRootFromASubdirectory(t *testing.T) {
	t.Parallel()
	dir := uninstallRepo(t)
	before := tree(t, dir)

	got := runCLI(t, repoAssets(t), "-C", filepath.Join(dir, "docs"), "uninstall", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, uninstalled(dir), got.stdout)
	assert.Equal(t, uninstalledTree(before), tree(t, dir))
}

func TestUninstallPrunesTheEmptyFrameworkDirectories(t *testing.T) {
	t.Parallel()
	dir := initializedRepo(t)
	want := map[string]string{}
	for name, content := range tree(t, dir) {
		if name == "docs" || strings.HasPrefix(name, "docs/") && !strings.HasPrefix(name, "docs/solomonar") {
			want[name] = content
		}
	}
	want["docs/README.md"] = strings.Replace(want["docs/README.md"], indexLine, "", 1)

	got := uninstallIn(t, dir, "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, want, tree(t, dir))
	assert.Contains(t, got.stdout, "  kept: docs/README.md\n")
}

func TestUninstallDryRunWritesNothing(t *testing.T) {
	t.Parallel()
	dir := uninstallRepo(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := uninstallIn(t, dir, "--dry-run")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, "Plan\n  remove     .agents/skills/go/SKILL.md\n"), got.stdout)
	for _, line := range []string{
		"  update     AGENTS.md                        managed part removed; project notes kept\n",
		"  update     docs/README.md                   framework index line removed\n",
		"  skip       " + moneySource + "  project extension; source kept\n",
		"  remove     docs/solomonar/rules.md\n",
	} {
		assert.Contains(t, got.stdout, line)
	}
	assert.True(t, strings.HasSuffix(got.stdout, "  remove     .solomonar/manifest.json\n"+
		"Would uninstall solomonar from "+dir+"\n  2 files updated, 52 removed, 1 skipped\n"), got.stdout)
	assert.Empty(t, got.stderr)
	assertNothingWritten(t, dir, before, tip)
}

func TestUninstallJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		dryRun bool
	}{
		{name: "apply", args: []string{"--yes"}},
		{name: "dry run", args: []string{"--dry-run"}, dryRun: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := uninstallRepo(t)

			got := uninstallIn(t, dir, append([]string{"--json"}, tt.args...)...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Empty(t, got.stderr)
			var envelope struct {
				OK       bool           `json:"ok"`
				Command  string         `json:"command"`
				Data     map[string]any `json:"data"`
				Warnings []any          `json:"warnings"`
			}
			require.NoError(t, json.Unmarshal([]byte(got.stdout), &envelope))
			assert.True(t, envelope.OK)
			assert.Equal(t, "uninstall", envelope.Command)
			assert.Empty(t, envelope.Warnings)
			assert.Equal(t, dir, envelope.Data["project"])
			assert.Equal(t, []any{"AGENTS.md", "docs/README.md", moneySource, "refs/heads/workflow-state"},
				envelope.Data["preserved"])
			files := envelope.Data["files"].([]any)
			assert.Len(t, files, 55)
			assert.Equal(t, map[string]any{
				"path": "AGENTS.md", "action": "update", "owner": "core+notes",
				"reason": "managed part removed; project notes kept",
			}, files[slices.IndexFunc(files, func(f any) bool {
				return f.(map[string]any)["path"] == "AGENTS.md"
			})])
			assert.Equal(t, map[string]any{"path": ".solomonar/manifest.json", "action": "remove", "owner": "manifest"},
				files[len(files)-1])
			if tt.dryRun {
				assert.Equal(t, true, envelope.Data["dry_run"])
				assert.NotContains(t, envelope.Data, "next")
			} else {
				assert.NotContains(t, envelope.Data, "dry_run")
				assert.Equal(t, []any{"Review the removal with git status, then commit it."}, envelope.Data["next"])
			}
		})
	}
}

func TestUninstallNeedsConfirmation(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{nil, {"--json"}} {
		dir := uninstallRepo(t)
		before, tip := tree(t, dir), refTip(t, dir)

		got := uninstallIn(t, dir, args...)

		assert.Equal(t, 2, got.exit)
		if args == nil {
			assert.Empty(t, got.stdout)
			assert.Equal(t, "error: uninstall needs a confirmation (interactive_required)\n"+
				"  fix:   pass --yes to accept the plan, or --dry-run to print it\nnothing was written\n", got.stderr)
		} else {
			assert.Contains(t, got.stdout, `"code": "interactive_required"`)
			assert.NotContains(t, got.stdout, `"data"`)
			assert.Empty(t, got.stderr)
		}
		assertNothingWritten(t, dir, before, tip)
	}
}

func TestUninstallConfirmsInTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, input, stdout string
		applied             bool
	}{
		{name: "declined", input: "n\n", stdout: "Cancelled; nothing was written.\n"},
		{name: "accepted", input: "\n", applied: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := uninstallRepo(t)
			before := tree(t, dir)

			got := runInteractive(t, repoAssets(t), dir, tt.input, "uninstall")

			require.Equal(t, 0, got.exit, got.stderr)
			assert.True(t, strings.HasPrefix(got.stderr, "Plan\n  framework "+shipped+"\n"), got.stderr)
			for _, line := range []string{
				"    remove     docs/solomonar/agents/ (6 roles)\n",
				"    update     AGENTS.md       managed part removed; project notes kept\n",
				"    skip       " + moneySource + "  project extension; source kept\n",
				"    remove     .claude/skills/ (5 skill stubs: 4 core, 1 extension)\n",
				"  manifest\n    remove     .solomonar/manifest.json\nProceed? [Y/n] \n",
			} {
				assert.Contains(t, got.stderr, line)
			}
			if tt.applied {
				assert.Equal(t, uninstalled(dir), got.stdout)
				assert.Equal(t, uninstalledTree(before), tree(t, dir))
			} else {
				assert.Equal(t, tt.stdout, got.stdout)
				assert.Equal(t, before, tree(t, dir))
			}
		})
	}
}

// onRead runs hook once, when the confirmation reads its answer.
type onRead struct {
	hook   func()
	answer io.Reader
}

func (r *onRead) Read(p []byte) (int, error) {
	if r.hook != nil {
		r.hook()
		r.hook = nil
	}
	return r.answer.Read(p)
}

func confirmWith(ctx context.Context, t *testing.T, dir string, hook func(), args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	exit := Run(ctx, Env{
		Args:        append([]string{"-C", dir, "uninstall"}, args...),
		Stdin:       &onRead{hook: hook, answer: strings.NewReader("y\n")},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Getenv:      func(string) string { return "" },
		Now:         func() time.Time { return time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC) },
		Interactive: true,
		RawMode:     func() (func(), error) { return func() {}, nil },
		Assets:      repoAssets(t),
	})
	return result{exit: exit, stdout: stdout.String(), stderr: stderr.String()}
}

func TestUninstallRejectsAChangeDuringTheConfirmation(t *testing.T) {
	t.Parallel()
	const rules = "docs/solomonar/rules.md"

	tests := []struct {
		name string
		args []string
		edit func(t *testing.T, dir string)
		path string
	}{
		{name: "edited file", path: rules, edit: func(t *testing.T, dir string) {
			t.Helper()
			writeFile(t, dir, rules, "mine\n")
		}},
		{name: "restored kept file", args: []string{"--keep", rules}, path: rules, edit: func(t *testing.T, dir string) {
			t.Helper()
			gittest.Git(t, dir, "checkout", "--", rules)
		}},
		{name: "edited notes", args: []string{"--overwrite", "AGENTS.md"}, path: "AGENTS.md", edit: func(t *testing.T, dir string) {
			t.Helper()
			writeFile(t, dir, "AGENTS.md", readTestFile(t, dir, "AGENTS.md")+"More.\n")
		}},
		{name: "new managed file", path: "docs/solomonar/report.md", edit: func(t *testing.T, dir string) {
			t.Helper()
			writeFile(t, dir, "docs/solomonar/report.md", "report\n")
		}},
		{name: "manifest", path: ".solomonar/manifest.json", edit: func(t *testing.T, dir string) {
			t.Helper()
			writeFile(t, dir, ".solomonar/manifest.json", readTestFile(t, dir, ".solomonar/manifest.json")+"\n")
		}},
		{name: "manifest symlink", path: ".solomonar/manifest.json", edit: func(t *testing.T, dir string) {
			t.Helper()
			manifest := filepath.Join(dir, ".solomonar/manifest.json")
			require.NoError(t, os.Rename(manifest, filepath.Join(dir, "manifest.json")))
			require.NoError(t, os.Symlink("../manifest.json", manifest))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := uninstallRepo(t)
			switch tt.name {
			case "restored kept file":
				gittest.Git(t, dir, "add", rules)
				writeFile(t, dir, rules, "mine\n")
			case "edited notes":
				agents := readTestFile(t, dir, "AGENTS.md")
				writeFile(t, dir, "AGENTS.md", strings.Replace(agents, "# ", "# Edited ", 1))
			case "new managed file":
				require.NoError(t, os.Remove(filepath.Join(dir, "docs/solomonar/report.md")))
			}
			var edited map[string]string
			got := confirmWith(t.Context(), t, dir, func() {
				tt.edit(t, dir)
				edited = tree(t, dir)
			}, tt.args...)

			assert.Equal(t, 4, got.exit)
			assert.Empty(t, got.stdout)
			assert.True(t, strings.HasSuffix(got.stderr, "error: uninstall found a change since the plan was shown (file_conflict)\n"+
				"  check: the planned files are unchanged since the plan was shown\n"+
				"  found: "+tt.path+" changed since the plan was shown\n"+
				"  fix:   rerun solomonar uninstall\nnothing was written\n"), got.stderr)
			assert.Equal(t, edited, tree(t, dir))
		})
	}
}

func TestUninstallInterruptedDuringTheConfirmationWritesNothing(t *testing.T) {
	t.Parallel()
	dir := uninstallRepo(t)
	before := tree(t, dir)
	ctx, cancel := context.WithCancel(t.Context())

	got := confirmWith(ctx, t, dir, cancel)

	assert.Equal(t, 130, got.exit)
	assert.Contains(t, got.stderr, "error: interrupted (interrupted)\n")
	assert.Equal(t, before, tree(t, dir))
}

func TestUninstallRollsBackAFailedApply(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := uninstallRepo(t)
	solomonar := filepath.Join(dir, ".solomonar")
	require.NoError(t, os.Chmod(filepath.Join(dir, "docs/solomonar/agents"), 0o700))
	require.NoError(t, os.Chmod(solomonar, 0o500)) // the manifest, the last change, cannot be removed
	t.Cleanup(func() { _ = os.Chmod(solomonar, 0o755) })
	before := tree(t, dir)
	modes := dirModes(t, dir)

	got := uninstallIn(t, dir, "--yes")

	assert.Equal(t, 1, got.exit)
	assert.Empty(t, got.stdout)
	assert.Contains(t, got.stderr, "apply .solomonar/manifest.json: ")
	assert.True(t, strings.HasSuffix(got.stderr, "nothing was written\n"), got.stderr)
	assert.Equal(t, before, tree(t, dir))
	assert.Equal(t, modes, dirModes(t, dir))
}

// dirModes maps each directory under dir, outside .git, to its permission bits.
func dirModes(t *testing.T, dir string) map[string]fs.FileMode {
	t.Helper()
	modes := map[string]fs.FileMode{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if d.Name() == ".git" {
			return filepath.SkipDir
		}
		info, err := d.Info()
		modes[path] = info.Mode().Perm()
		return err
	})
	require.NoError(t, err)
	return modes
}

func TestUninstallFailureNamesThePathsARollbackLeft(t *testing.T) {
	t.Parallel()
	err := errors.Join(errors.New("apply b.md: denied"),
		&project.RollbackError{Paths: []string{"a.md", "docs"}, Err: errors.New("restore a.md: denied\nrestore docs: denied")})

	got := renderResult(t.Context(), false, output{}, rollbackFailure(t.Context(), err))

	assert.Equal(t, 1, got.exit)
	assert.Equal(t, "error: apply b.md: denied\nrestore a.md: denied\nrestore docs: denied (unexpected_error)\n"+
		"  found: not restored: a.md, docs\n"+
		"  fix:   restore these paths by hand, for example with git restore, then rerun solomonar uninstall\n"+
		"rollback incomplete; recover the paths named above\n", got.stderr)
	assert.NotContains(t, got.stderr, "nothing was written")

	interrupted := rollbackFailure(t.Context(), errors.Join(context.Canceled, &project.RollbackError{Paths: []string{"a.md"}, Err: errors.New("x")}))
	assert.Equal(t, "interrupted", asCLIError(t.Context(), "uninstall", interrupted).Code)
	plain := errors.New("apply b.md: denied")
	assert.Equal(t, plain, rollbackFailure(t.Context(), plain))
}

func TestUninstallConflictWritesNothing(t *testing.T) {
	t.Parallel()
	dir := uninstallRepo(t)
	writeFile(t, dir, "docs/solomonar/rules.md", "mine\n")
	before, tip := tree(t, dir), refTip(t, dir)

	got := uninstallIn(t, dir, "--yes")

	assert.Equal(t, 4, got.exit)
	assert.Contains(t, got.stdout, "Plan\n")
	assert.Contains(t, got.stdout, "  conflict   docs/solomonar/rules.md          modified since installation (manifest sha256 differs)\n")
	assert.Equal(t, "error: uninstall found 1 conflict (file_conflict)\n"+
		"  check: no planned file conflicts with the project's files\n"+
		"  found: 1 conflict: docs/solomonar/rules.md\n"+
		"  fix:   rerun with --keep <path> to keep the project's file, or --overwrite <path> to remove it\n"+
		"nothing was written\n", got.stderr)
	assertNothingWritten(t, dir, before, tip)

	asJSON := uninstallIn(t, dir, "--yes", "--json")
	assert.Equal(t, 4, asJSON.exit)
	assert.Contains(t, asJSON.stdout, `"code": "file_conflict"`)
	assert.NotContains(t, asJSON.stdout, `"data"`)
	assert.Empty(t, asJSON.stderr)
	assertNothingWritten(t, dir, before, tip)
}

func TestUninstallResolvesAConflict(t *testing.T) {
	t.Parallel()
	const rules = "docs/solomonar/rules.md"

	tests := []struct {
		name     string
		args     []string
		kept     bool
		warnings string
	}{
		{
			name: "keep", args: []string{"--keep", rules}, kept: true,
			warnings: "warning: " + rules + " is kept and no longer managed; remove it by hand if unneeded\n",
		},
		{name: "overwrite", args: []string{"--overwrite", rules}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := uninstallRepo(t)
			writeFile(t, dir, rules, "mine\n")

			got := uninstallIn(t, dir, append(tt.args, "--yes")...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, tt.warnings, got.stderr)
			_, err := os.Stat(filepath.Join(dir, rules))
			assert.Equal(t, tt.kept, err == nil)
			assert.NoFileExists(t, filepath.Join(dir, project.ManifestPath))
		})
	}
}

func TestUninstallWarnsAboutManualReferences(t *testing.T) {
	t.Parallel()
	dir := uninstallRepo(t)
	readme := readTestFile(t, dir, "docs/README.md")
	writeFile(t, dir, "docs/README.md", strings.Replace(readme, indexLine, "- `docs/solomonar/`: the framework\n", 1))
	writeFile(t, dir, "CLAUDE.md", "@AGENTS.md\nMine.\n")
	writeFile(t, dir, "AGENTS.md", strings.TrimSuffix(readTestFile(t, dir, "AGENTS.md"), "Use tabs.\n"))

	got := uninstallIn(t, dir, "--keep", "CLAUDE.md", "--json", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	var envelope struct {
		Data     uninstallData `json:"data"`
		Warnings []warning     `json:"warnings"`
	}
	require.NoError(t, json.Unmarshal([]byte(got.stdout), &envelope))
	assert.Equal(t, []warning{
		{Code: "manual_reference", Message: "CLAUDE.md is kept and no longer managed; remove it by hand if unneeded"},
		{Code: "manual_reference", Message: "CLAUDE.md still imports @AGENTS.md, which uninstall removes; edit it by hand"},
		{Code: "manual_reference", Message: "docs/README.md still mentions docs/solomonar; review it by hand"},
	}, envelope.Warnings)
	assert.Equal(t, []string{"CLAUDE.md", "docs/README.md", moneySource, "refs/heads/workflow-state"},
		envelope.Data.Preserved)
	assert.NoFileExists(t, filepath.Join(dir, "AGENTS.md"))
}

func TestUninstallChecksTheManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		edit  func(t *testing.T, dir string)
		exit  int
		error string
	}{
		{name: "newer", exit: 3, error: "(cli_outdated)", edit: func(t *testing.T, dir string) {
			t.Helper()
			m := readTestManifest(t, dir)
			m.FrameworkVersion = "99.0.0"
			writeTestManifest(t, dir, m)
		}},
		{name: "malformed", exit: 1, error: "(manifest_invalid)", edit: func(t *testing.T, dir string) {
			t.Helper()
			writeFile(t, dir, project.ManifestPath, "{")
		}},
		{
			name: "symlink", exit: 4, error: "error: uninstall rejected (file_conflict)\n" +
				"  check: the manifest is a regular file\n" +
				"  found: .solomonar/manifest.json exists and is not a regular file\n" +
				"  fix:   restore .solomonar/manifest.json as a regular file, then rerun solomonar uninstall\n",
			edit: func(t *testing.T, dir string) {
				t.Helper()
				manifest := filepath.Join(dir, project.ManifestPath)
				require.NoError(t, os.Rename(manifest, filepath.Join(dir, "manifest.json")))
				require.NoError(t, os.Symlink("../manifest.json", manifest))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := uninstallRepo(t)
			tt.edit(t, dir)
			before := tree(t, dir)

			got := uninstallIn(t, dir, "--yes")

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, tt.error)
			assert.Equal(t, before, tree(t, dir))
		})
	}
}

func TestUninstallRemovesAnOlderLayoutWithoutUpgrade(t *testing.T) {
	t.Parallel()
	dir := legacyRepo(t)

	got := uninstallIn(t, dir, "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.NoDirExists(t, filepath.Join(dir, "docs/solomonar"))
	assert.NoDirExists(t, filepath.Join(dir, ".solomonar"))
	assert.FileExists(t, filepath.Join(dir, "docs/decisions.md"))
}

func TestUninstallRunsOnlyRepositoryDiscovery(t *testing.T) {
	t.Parallel()
	dir := uninstallRepo(t)
	addRemote(t, dir)

	got := uninstallIn(t, dir, "-v", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "+ git rev-parse --path-format=absolute --show-toplevel --git-common-dir\n+ git version\n", got.stderr)
}
