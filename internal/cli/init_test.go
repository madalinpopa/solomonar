package cli

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

func runtimeArgs(runtimes string, extra ...string) []string {
	return append([]string{"init", "--runtime", runtimes, "--ext", "none", "--yes"}, extra...)
}

func initArgs(extra ...string) []string {
	return append([]string{"init", "--runtime", "none", "--ext", "none", "--yes"}, extra...)
}

// initPaths lists the paths a core-only init plans, in plan order.
func initPaths() []string {
	framework := []string{
		"agents/adversarial-reviewer.md", "agents/correctness-reviewer.md", "agents/implementer.md",
		"agents/orchestrator.md", "agents/security-reviewer.md", "agents/spec-reviewer.md",
		"report.md", "reviewer.md", "rules.md",
		"skills/prototype/SKILL.md", "skills/prototype/template.md", "skills/solomonar/SKILL.md",
		"skills/spec/SKILL.md", "skills/spec/template.md", "skills/work-spec/SKILL.md", "skills/work-spec/template.md",
	}
	paths := make([]string, 0, len(framework)+3)
	for _, p := range framework {
		paths = append(paths, "docs/solomonar/"+p)
	}
	return append(paths, "AGENTS.md", "docs/README.md", project.ManifestPath)
}

func initOwner(path string) string {
	switch path {
	case "AGENTS.md":
		return "core+notes"
	case "docs/README.md":
		return "seed"
	case project.ManifestPath:
		return "manifest"
	}
	return "core"
}

// createdFilesJSON is the JSON files list of a first core-only init.
func createdFilesJSON() string {
	entries := make([]string, 0, len(initPaths()))
	for _, p := range initPaths() {
		entries = append(entries, fmt.Sprintf(`{"path": %q, "action": "create", "owner": %q}`, p, initOwner(p)))
	}
	return "[" + strings.Join(entries, ",") + "]"
}

// tree maps every file and directory under dir, outside .git, to its content ("/" marks a directory).
func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == dir {
			return err
		}
		if d.Name() == ".git" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil || d.IsDir() {
			files[filepath.ToSlash(rel)] = "/"
			return err
		}
		content, err := os.ReadFile(path)
		files[filepath.ToSlash(rel)] = string(content)
		return err
	})
	require.NoError(t, err)
	return files
}

// refTip returns the registry tip, or "" when the ref is absent.
func refTip(t *testing.T, dir string) string {
	t.Helper()
	out, err := (&git.Repo{Dir: dir}).Run(t.Context(), "rev-parse", "--verify", "--quiet", task.Ref)
	if err != nil {
		return ""
	}
	return out
}

// assertNothingWritten checks the worktree and the registry ref still match the snapshot.
func assertNothingWritten(t *testing.T, dir string, before map[string]string, tip string) {
	t.Helper()
	assert.Equal(t, before, tree(t, dir))
	assert.Equal(t, tip, refTip(t, dir))
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func initializedRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.NewRepo(t)
	got := runInRepo(t, dir, initArgs()...)
	require.Equal(t, 0, got.exit, got.stderr)
	return dir
}

// nextBlock is the Next: block of a repository whose main branch has no commit and HEAD is on main.
func nextBlock(dir string) string {
	return "\nNext:\n" +
		"  1. First commit required before task worktrees:\n" +
		"       git add AGENTS.md .solomonar docs && git commit -m \"chore: initialize solomonar " + shipped + "\"\n" +
		sessionStep(dir)
}

func sessionStep(dir string) string {
	return "  2. Start a top-level session in " + dir + ":\n" +
		"       Use the solomonar skill to explore the next useful flow.\n"
}

func TestInitUC2CoreOnly(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	addRemote(t, dir)

	got := runCLI(t, repoAssets(t), "init", dir, "--runtime", "none", "--ext", "none",
		"--objective", "Track shared household expenses", "--yes")

	require.Equal(t, 0, got.exit, got.stderr)
	tip := refTip(t, dir)
	assert.Equal(t, "Initialized solomonar "+shipped+" in "+dir+"\n"+
		"  19 files created\n"+
		"  registry  refs/heads/workflow-state @ "+tip+"\n"+
		"  pushed    origin/workflow-state\n"+nextBlock(dir), got.stdout)
	assert.Empty(t, got.stderr)
	files := tree(t, dir)
	for _, p := range initPaths() {
		assert.Contains(t, files, p)
	}
	assert.Contains(t, files["docs/README.md"], "# "+filepath.Base(dir)+"\n")
	assert.Contains(t, files["docs/README.md"], "Track shared household expenses")
	manifest, err := project.DecodeManifest([]byte(files[project.ManifestPath]), shipped)
	require.NoError(t, err)
	assert.Equal(t, project.Project{Name: filepath.Base(dir), MainBranch: "main", Remote: "origin"}, manifest.Project)
	assert.Len(t, manifest.Files, len(initPaths()))
	assert.Equal(t, "state: initialize registry", gittest.Git(t, dir, "log", "-1", "--format=%s%n%P", task.Ref))
	assert.JSONEq(t, `{"schema_version": 2}`, gittest.Git(t, dir, "show", task.Ref+":state.json"))
	assert.Equal(t, "state.json", gittest.Git(t, dir, "ls-tree", "-r", "--name-only", task.Ref))
	assert.Equal(t, tip, gittest.Git(t, dir, "ls-remote", "origin", task.Ref)[:len(tip)])
}

func TestInitWithoutRemoteWarns(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)

	got := runInRepo(t, dir, initArgs()...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Initialized solomonar "+shipped+" in "+dir+"\n"+
		"  19 files created\n"+
		"  registry  refs/heads/workflow-state @ "+refTip(t, dir)+"\n"+nextBlock(dir), got.stdout)
	assert.Equal(t, "warning: no remote \"origin\"; the registry stays local\n", got.stderr)
}

func TestInitJSON(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)

	got := runInRepo(t, dir, initArgs("--json")...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "init",
		"data": {
			"project": "`+dir+`",
			"framework_version": "`+shipped+`",
			"choices": {"runtimes": [], "extensions": {}, "adversarial_model": {}},
			"decisions": [],
			"files": `+createdFilesJSON()+`,
			"registry": {"ref": "refs/heads/workflow-state", "commit": "`+refTip(t, dir)+`", "action": "create", "pushed": false},
			"git_init": false,
			"first_commit_required": true,
			"next": ["git add AGENTS.md .solomonar docs", "git commit -m \"chore: initialize solomonar `+shipped+`\""]
		},
		"warnings": [{"code": "no_remote", "message": "no remote \"origin\"; the registry stays local"}]
	}`, got.stdout)
}

func TestInitRerunIsNoOp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "same choices", args: initArgs()},
		{name: "choices from the manifest", args: []string{"init", "--yes"}},
		{name: "without --yes", args: []string{"init"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := initializedRepo(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runInRepo(t, dir, tt.args...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, "Already initialized with solomonar "+shipped+"; nothing to do.\n"+
				"  18 files unchanged, 1 skipped · registry refs/heads/workflow-state unchanged\n"+
				"  Change choices with `solomonar runtime|extension add|remove`; move versions with `solomonar upgrade`.\n",
				got.stdout)
			assert.Empty(t, got.stderr)
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestInitRejectsDifferentChoice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  []string
		found func(dir string) string
	}{
		{
			name: "name", args: initArgs("--name", "billing"),
			found: func(dir string) string {
				return fmt.Sprintf(`manifest name %q; requested "billing"`, filepath.Base(dir))
			},
		},
		{
			name: "remote", args: initArgs("--remote", "upstream"),
			found: func(string) string { return `manifest remote "origin"; requested "upstream"` },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := initializedRepo(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runInRepo(t, dir, tt.args...)

			assert.Equal(t, 3, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, "error: init rejected (already_initialized)\n  found: "+tt.found(dir)+"\n")
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestInitCompletesInterruptedInit(t *testing.T) {
	t.Parallel()

	dir := initializedRepo(t)
	require.NoError(t, os.Remove(filepath.Join(dir, project.ManifestPath)))
	gittest.Git(t, dir, "update-ref", "-d", task.Ref)

	plan := runInRepo(t, dir, initArgs("--dry-run")...)
	got := runInRepo(t, dir, initArgs()...)

	require.Equal(t, 0, plan.exit, plan.stderr)
	assert.Contains(t, plan.stdout, fmt.Sprintf("  unchanged  %-45s  adopted\n", "docs/solomonar/rules.md"))
	assert.Contains(t, plan.stdout, fmt.Sprintf("  unchanged  %-45s  adopted\n", "AGENTS.md"))
	assert.Contains(t, plan.stdout, "  skip       docs/README.md\n")
	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  1 file created, 17 unchanged, 1 skipped\n")
	assert.NotEmpty(t, refTip(t, dir))
	assert.FileExists(t, filepath.Join(dir, project.ManifestPath))
}

func TestInitKeepsExistingRegistry(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	tip := refTip(t, dir)

	got := runInRepo(t, dir, initArgs()...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Initialized solomonar "+shipped+" in "+dir+"\n"+
		"  19 files created\n"+
		"  registry  refs/heads/workflow-state @ "+tip+" unchanged\n"+nextBlock(dir), got.stdout)
	assert.Empty(t, got.stderr)
	assert.Equal(t, tip, refTip(t, dir))
}

func TestInitKeepsLocallyEditedFile(t *testing.T) {
	t.Parallel()

	dir := initializedRepo(t)
	writeFile(t, dir, "docs/solomonar/rules.md", "edited\n")
	before, tip := tree(t, dir), refTip(t, dir)

	plan := runInRepo(t, dir, initArgs("--dry-run")...)
	got := runInRepo(t, dir, initArgs()...)

	require.Equal(t, 0, plan.exit, plan.stderr)
	assert.Contains(t, plan.stdout, "  keep       docs/solomonar/rules.md\n")
	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  1 file kept, 17 unchanged, 1 skipped · registry")
	assertNothingWritten(t, dir, before, tip)
}

func TestInitConflictWritesNothing(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	writeFile(t, dir, "docs/solomonar/rules.md", "project rules\n")
	writeFile(t, dir, "docs/solomonar/report.md", "project report\n")
	before := tree(t, dir)

	got := runInRepo(t, dir, initArgs()...)

	assert.Equal(t, 4, got.exit)
	assert.Equal(t, "Plan\n"+
		"  conflict   docs/solomonar/report.md  exists and is not managed by solomonar\n"+
		"  conflict   docs/solomonar/rules.md   exists and is not managed by solomonar\n", got.stdout)
	assert.Equal(t, "error: init found 2 conflicts (file_conflict)\n"+
		"  check: no planned file conflicts with the project's files\n"+
		"  found: 2 conflicts: docs/solomonar/report.md, docs/solomonar/rules.md\n"+
		"  fix:   rerun with --keep <path> to keep the project's file, or --overwrite <path> to replace it\n"+
		"nothing was written\n", got.stderr)
	assertNothingWritten(t, dir, before, "")
}

func TestInitConflictJSONHasNoData(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	writeFile(t, dir, "docs/solomonar/rules.md", "project rules\n")

	got := runInRepo(t, dir, initArgs("--json")...)

	assert.Equal(t, 4, got.exit)
	assert.JSONEq(t, `{
		"ok": false,
		"command": "init",
		"error": {
			"code": "file_conflict",
			"message": "init found 1 conflict",
			"check": "no planned file conflicts with the project's files",
			"found": "1 conflict: docs/solomonar/rules.md",
			"fix": "rerun with --keep <path> to keep the project's file, or --overwrite <path> to replace it"
		},
		"warnings": []
	}`, got.stdout)
}

func TestInitResolvesConflict(t *testing.T) {
	t.Parallel()

	assets := repoAssets(t)
	rules, err := fs.ReadFile(assets, "framework/rules.md")
	require.NoError(t, err)
	tests := []struct {
		name string
		flag string
		want string
	}{
		{name: "keep", flag: "--keep", want: "project rules\n"},
		{name: "overwrite", flag: "--overwrite", want: string(rules)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := gittest.NewRepo(t)
			writeFile(t, dir, "docs/solomonar/rules.md", "project rules\n")

			got := runInRepo(t, dir, initArgs(tt.flag, "docs/solomonar/rules.md")...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, tt.want, tree(t, dir)["docs/solomonar/rules.md"])
			assert.NotEmpty(t, refTip(t, dir))
		})
	}
}

func TestInitDryRunWritesNothing(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	before := tree(t, dir)

	human := runInRepo(t, dir, initArgs("--dry-run")...)
	asJSON := runInRepo(t, dir, initArgs("--dry-run", "--json")...)

	var want strings.Builder
	want.WriteString("Plan\n")
	for _, p := range initPaths() {
		want.WriteString("  create     " + p + "\n")
	}
	want.WriteString("  registry   create refs/heads/workflow-state with an empty registry\n" +
		"Would initialize solomonar " + shipped + " in " + dir + "\n  19 files created\n")
	require.Equal(t, 0, human.exit, human.stderr)
	assert.Equal(t, want.String(), human.stdout)
	assert.Empty(t, human.stderr)
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "init",
		"data": {
			"project": "`+dir+`",
			"framework_version": "`+shipped+`",
			"choices": {"runtimes": [], "extensions": {}, "adversarial_model": {}},
			"decisions": [],
			"files": `+createdFilesJSON()+`,
			"registry": {"ref": "refs/heads/workflow-state", "action": "create"},
			"git_init": false,
			"first_commit_required": true,
			"dry_run": true
		},
		"warnings": []
	}`, asJSON.stdout)
	assertNothingWritten(t, dir, before, "")
}

// putState points the registry ref at a root commit whose state.json holds content.
func putState(t *testing.T, dir, content string) {
	t.Helper()
	repo := &git.Repo{Dir: dir}
	blob, err := repo.RunInput(t.Context(), strings.NewReader(content), "hash-object", "-w", "--stdin")
	require.NoError(t, err)
	treeSHA, err := repo.RunInput(t.Context(), strings.NewReader("100644 blob "+blob+"\tstate.json\n"), "mktree")
	require.NoError(t, err)
	commit := gittest.Git(t, dir, "commit-tree", treeSHA, "-m", "state: broken")
	gittest.Git(t, dir, "update-ref", task.Ref, commit)
}

func TestInitRejections(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		state  string
		exit   int
		stderr string
	}{
		{
			name: "unknown runtime", args: []string{"init", "--runtime", "claude,gemini", "--yes"}, exit: 2,
			stderr: "error: init rejected (unknown_runtime)\n" +
				"  check: --runtime names a known runtime\n" +
				"  found: \"gemini\" is not a runtime; known: claude, codex, copilot\n" +
				"  fix:   pass --runtime claude, codex, copilot, or none\n",
		},
		{
			name: "none beside a runtime", args: []string{"init", "--runtime", "none", "--runtime", "codex", "--yes"},
			exit: 2,
			stderr: "error: init rejected (invalid_value)\n" +
				"  check: --runtime names runtimes or none\n" +
				"  found: none cannot be combined with codex\n" +
				"  fix:   pass --runtime none alone\n",
		},
		{
			name: "model without =", args: runtimeArgs("claude", "--adversarial-model", "opus"), exit: 2,
			stderr: "error: init rejected (invalid_value)\n" +
				"  check: --adversarial-model is <runtime>=<model>\n" +
				"  found: \"opus\" has no \"=\"\n" +
				"  fix:   pass --adversarial-model claude=<model>\n",
		},
		{
			name: "model without a model", args: runtimeArgs("claude", "--adversarial-model", "claude="), exit: 2,
			stderr: "  found: \"claude=\" has an empty model\n",
		},
		{
			name: "model without a runtime", args: runtimeArgs("claude", "--adversarial-model", "=opus"), exit: 2,
			stderr: "  found: \"=opus\" has an empty runtime\n",
		},
		{
			name: "model for an unselected runtime", args: runtimeArgs("claude", "--adversarial-model", "codex=o3"),
			exit: 2,
			stderr: "error: init rejected (invalid_value)\n" +
				"  check: --adversarial-model names a selected runtime\n" +
				"  found: codex is not selected\n" +
				"  fix:   add --runtime codex or drop the model\n",
		},
		{
			name: "model for a runtime absent from PATH",
			args: []string{"init", "--ext", "none", "--adversarial-model", "claude=opus", "--yes"}, exit: 2,
			stderr: "  found: claude is not selected\n",
		},
		{
			name: "extension none beside a name", args: []string{"init", "--ext", "go,none", "--ext", "rust", "--yes"},
			exit: 2,
			stderr: "error: init rejected (invalid_value)\n" +
				"  check: --ext names extensions or none\n" +
				"  found: none cannot be combined with go\n" +
				"  fix:   pass --ext none alone\n",
		},
		{
			name: "unknown extension", args: []string{"init", "--runtime", "none", "--ext", "go,rust", "--yes"}, exit: 2,
			stderr: "error: init rejected (unknown_extension)\n" +
				"  check: --ext names a catalog extension\n" +
				"  found: \"rust\" is not in the catalog; known: go\n" +
				"  fix:   pass --ext with a known name, or none\n",
		},
		{
			name: "extension path", args: []string{"init", "--runtime", "none", "--ext", "./extensions/go", "--yes"},
			exit: 2, stderr: "  found: \"./extensions/go\" is not in the catalog; known: go\n",
		},
		{
			name: "extension checked before the model's runtime",
			args: []string{"init", "--ext", "rust", "--adversarial-model", "claude=opus", "--yes"}, exit: 2,
			stderr: "error: init rejected (unknown_extension)\n",
		},
		{
			name: "keep without conflict", args: initArgs("--keep", "docs/solomonar/rules.md"), exit: 2,
			stderr: "error: init rejected (invalid_value)\n" +
				"  check: --keep and --overwrite resolve conflicts\n" +
				"  found: docs/solomonar/rules.md is not a conflict\n",
		},
		{
			name: "without --yes", args: []string{"init"}, exit: 2,
			stderr: "error: init needs a confirmation (interactive_required)\n" +
				"  fix:   pass --yes to accept the plan, or --dry-run to print it\nnothing was written\n",
		},
		{
			name: "invalid registry", args: initArgs(), state: "not json", exit: 3,
			stderr: "error: init rejected (registry_exists_invalid)\n" +
				"  check: an existing workflow-state holds a valid registry\n",
		},
		{
			name: "two directories", args: []string{"init", "a", "b"}, exit: 2,
			stderr: "error: unexpected argument \"b\" (invalid_usage)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gittest.NoGH(t)
			dir := gittest.NewRepo(t)
			if tt.state != "" {
				putState(t, dir, tt.state)
			}
			before, tip := tree(t, dir), refTip(t, dir)

			got := runInRepo(t, dir, tt.args...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, tt.stderr)
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestInitInvalidRegistryNamesTheFix(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	putState(t, dir, "not json")

	got := runInRepo(t, dir, initArgs()...)

	assert.Contains(t, got.stderr,
		"  fix:   inspect with git show refs/heads/workflow-state:state.json; the CLI never overwrites a registry\n")
}

// plainDir returns an empty directory "ledger" that no Git repository contains.
func plainDir(t *testing.T) string {
	t.Helper()
	gittest.NoGH(t)
	outside := gittest.Resolved(t, t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", outside)
	dir := filepath.Join(outside, "ledger")
	require.NoError(t, os.Mkdir(dir, 0o755))
	return dir
}

func TestInitOutsideRepository(t *testing.T) {
	dir := plainDir(t)

	got := runCLI(t, repoAssets(t), "-C", filepath.Dir(dir), "init", "ledger", "--main-branch", "trunk", "--yes")

	assert.Equal(t, 3, got.exit)
	assert.Equal(t, "error: "+dir+" is not a Git repository (not_a_repository)\n"+
		"  fix:   run `git init -b trunk` first, or rerun with --git-init\nnothing was written\n", got.stderr)
	assert.NoDirExists(t, filepath.Join(dir, ".git"))
}

func TestInitGitInitCreatesRepository(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		branch string
	}{
		{name: "default main branch", branch: "main"},
		{name: "named main branch", args: []string{"--main-branch", "trunk"}, branch: "trunk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := plainDir(t)

			got := runCLI(t, repoAssets(t), append([]string{"-C", dir}, initArgs(append(tt.args, "--git-init")...)...)...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, "Initialized solomonar "+shipped+" in "+dir+"\n"+
				"  19 files created\n"+
				"  registry  refs/heads/workflow-state @ "+refTip(t, dir)+"\n"+nextBlock(dir), got.stdout)
			assert.Equal(t, "warning: no remote \"origin\"; the registry stays local\n", got.stderr)
			assert.Equal(t, "refs/heads/"+tt.branch, gittest.Git(t, dir, "symbolic-ref", "HEAD"))
			assert.Equal(t, dir, gittest.Git(t, dir, "rev-parse", "--show-toplevel"))
			files := tree(t, dir)
			for _, p := range initPaths() {
				assert.Contains(t, files, p)
			}
			manifest, err := project.DecodeManifest([]byte(files[project.ManifestPath]), shipped)
			require.NoError(t, err)
			assert.Equal(t, tt.branch, manifest.Project.MainBranch)
		})
	}
}

func TestInitGitInitDryRunRunsNoGit(t *testing.T) {
	dir := plainDir(t)

	human := runCLI(t, repoAssets(t), "-C", dir, "init", "--git-init", "--dry-run")
	asJSON := runCLI(t, repoAssets(t), "-C", dir, "init", "--git-init", "--dry-run", "--json")

	require.Equal(t, 0, human.exit, human.stderr)
	assert.True(t, strings.HasPrefix(human.stdout, "Plan\n  git        init -b main\n  create     docs/solomonar/"),
		human.stdout)
	assert.Contains(t, human.stdout, "  registry   create refs/heads/workflow-state with an empty registry\n")
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "init",
		"data": {
			"project": "`+dir+`",
			"framework_version": "`+shipped+`",
			"choices": {"runtimes": [], "extensions": {}, "adversarial_model": {}},
			"decisions": [],
			"files": `+createdFilesJSON()+`,
			"registry": {"ref": "refs/heads/workflow-state", "action": "create"},
			"git_init": true,
			"first_commit_required": true,
			"dry_run": true
		},
		"warnings": []
	}`, asJSON.stdout)
	assert.NoDirExists(t, filepath.Join(dir, ".git"))
	assert.Empty(t, tree(t, dir))
}

func TestInitGitInitNeedsConfirmation(t *testing.T) {
	dir := plainDir(t)

	got := runCLI(t, repoAssets(t), "-C", dir, "init", "--git-init")

	assert.Equal(t, 2, got.exit)
	assert.Contains(t, got.stderr, "error: init needs a confirmation (interactive_required)\n")
	assert.NoDirExists(t, filepath.Join(dir, ".git"))
	assert.Empty(t, tree(t, dir))
}

func TestInitGitInitInsideRepositoryIsIgnored(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)

	with := runInRepo(t, dir, initArgs("--git-init", "--dry-run", "--json")...)
	without := runInRepo(t, dir, initArgs("--dry-run", "--json")...)

	require.Equal(t, 0, with.exit, with.stderr)
	assert.Equal(t, without.stdout, with.stdout)
	assert.Contains(t, with.stdout, `"git_init": false`)
}

func TestInitGitInitFailingRegistryCreateKeepsGit(t *testing.T) {
	dir := plainDir(t)
	template := t.TempDir()
	writeFile(t, template, "refs/heads/workflow-state.lock", "")
	t.Setenv("GIT_TEMPLATE_DIR", template)

	got := runCLI(t, repoAssets(t), append([]string{"-C", dir}, initArgs("--git-init")...)...)

	assert.Equal(t, 1, got.exit)
	assert.Contains(t, got.stderr, "error: create workflow-state: ")
	assert.Empty(t, tree(t, dir))
	assert.DirExists(t, filepath.Join(dir, ".git"))
}

func committedRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.NewRepo(t)
	gittest.Commit(t, dir, "initial")
	return dir
}

func unbornFeatureRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.Resolved(t, t.TempDir())
	gittest.Git(t, dir, "init", "-b", "feature")
	return dir
}

func TestInitNextNamesTheFirstCommit(t *testing.T) {
	t.Parallel()

	commit := "git commit -m \"chore: initialize solomonar " + shipped + "\""
	tests := []struct {
		name     string
		repo     func(t *testing.T) string
		step     string
		next     []string
		required bool
	}{
		{
			name: "main has a commit",
			repo: committedRepo,
			step: "  1. git add AGENTS.md .solomonar docs && " + commit + "\n",
			next: []string{"git add AGENTS.md .solomonar docs", commit},
		},
		{
			name:     "main has no commit",
			repo:     gittest.NewRepo,
			step:     "  1. First commit required before task worktrees:\n       git add AGENTS.md .solomonar docs && " + commit + "\n",
			next:     []string{"git add AGENTS.md .solomonar docs", commit},
			required: true,
		},
		{
			name: "HEAD on another unborn branch",
			repo: unbornFeatureRepo,
			step: "  1. First commit required before task worktrees:\n" +
				"       git switch --orphan main\n" +
				"       git add AGENTS.md .solomonar docs && " + commit + "\n",
			next:     []string{"git switch --orphan main", "git add AGENTS.md .solomonar docs", commit},
			required: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			human, asJSON := tt.repo(t), tt.repo(t)

			text := runInRepo(t, human, initArgs()...)
			data := runInRepo(t, asJSON, initArgs("--json")...)

			require.Equal(t, 0, text.exit, text.stderr)
			assert.True(t, strings.HasSuffix(text.stdout, "\nNext:\n"+tt.step+sessionStep(human)), text.stdout)
			require.Equal(t, 0, data.exit, data.stderr)
			var envelope struct {
				Data struct {
					Next                []string `json:"next"`
					FirstCommitRequired bool     `json:"first_commit_required"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal([]byte(data.stdout), &envelope))
			assert.Equal(t, tt.next, envelope.Data.Next)
			assert.Equal(t, tt.required, envelope.Data.FirstCommitRequired)
		})
	}
}

func TestInitWarnsAboutJujutsu(t *testing.T) {
	t.Parallel()

	const message = "Jujutsu detected; v0.1 supports Git only. Never run jj commands on workflow-state."
	tests := []struct {
		name string
		jj   string
	}{
		{name: "directory", jj: ".jj/repo/store"},
		{name: "file", jj: ".jj"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := gittest.NewRepo(t)
			writeFile(t, dir, tt.jj, "")

			plan := runInRepo(t, dir, initArgs("--dry-run", "--json")...)
			applied := runInRepo(t, dir, initArgs()...)
			noOp := runInRepo(t, dir, initArgs()...)

			require.Equal(t, 0, plan.exit, plan.stderr)
			assert.Contains(t, plan.stdout, `"warnings": [
    {
      "code": "jujutsu_detected",
      "message": "`+message+`"
    }
  ]`)
			require.Equal(t, 0, applied.exit, applied.stderr)
			assert.Equal(t, "warning: "+message+"\nwarning: no remote \"origin\"; the registry stays local\n",
				applied.stderr)
			require.Equal(t, 0, noOp.exit, noOp.stderr)
			assert.Equal(t, "warning: "+message+"\n", noOp.stderr)
		})
	}
}

func TestInitFailingManifestWriteRollsBack(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	solomonar := filepath.Join(dir, ".solomonar")
	require.NoError(t, os.Mkdir(solomonar, 0o555))
	t.Cleanup(func() { _ = os.Chmod(solomonar, 0o755) })
	before := tree(t, dir)

	got := runInRepo(t, dir, initArgs()...)

	assert.Equal(t, 1, got.exit)
	assert.Contains(t, got.stderr, "error: apply .solomonar/manifest.json: ")
	assertNothingWritten(t, dir, before, "")
}

func TestInitFailingRegistryCreateRollsBack(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	writeFile(t, dir, "AGENTS.md", "# Our agents\n")
	before := tree(t, dir)
	lock := filepath.Join(dir, ".git", "refs", "heads", "workflow-state.lock")
	require.NoError(t, os.WriteFile(lock, nil, 0o644))

	got := runInRepo(t, dir, initArgs()...)

	assert.Equal(t, 1, got.exit)
	assert.Contains(t, got.stderr, "error: create workflow-state: ")
	assertNothingWritten(t, dir, before, "")
}

func TestInitFailingPushWarns(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	gittest.Git(t, dir, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))

	got := runInRepo(t, dir, initArgs()...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.NotContains(t, got.stdout, "pushed")
	assert.True(t, strings.HasPrefix(got.stderr, "warning: git push origin workflow-state failed: "), got.stderr)
	assert.NotEmpty(t, refTip(t, dir))
}

func TestInitHelp(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "help", "init")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "Usage: solomonar init [<dir>] [--name <name>]")
	assert.Contains(t, got.stdout, "[--runtime <claude|codex|copilot|none>]...")
	assert.Contains(t, got.stdout, "[--adversarial-model <runtime>=<model>]...")
}

func commitStep(paths string) string {
	return "  1. git add " + paths + " && git commit -m \"chore: initialize solomonar " + shipped + "\"\n"
}

func TestInitUC2WithClaude(t *testing.T) {
	t.Parallel()

	dir := committedRepo(t)

	got := runInRepo(t, dir, runtimeArgs("claude")...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Initialized solomonar "+shipped+" in "+dir+"\n"+
		"  29 files created\n"+
		"  registry  refs/heads/workflow-state @ "+refTip(t, dir)+"\n"+
		"\nNext:\n"+commitStep("AGENTS.md CLAUDE.md .claude .solomonar docs")+
		"  2. Start a top-level Claude Code session in "+dir+":\n"+
		"       /solomonar Explore the next useful flow.\n", got.stdout)
	files := tree(t, dir)
	assert.Equal(t, "@AGENTS.md\n", files["CLAUDE.md"])
	skill := files[".claude/skills/spec/SKILL.md"]
	assert.True(t, strings.HasPrefix(skill, "---\nname: spec\ndescription: Maintains"), skill)
	assert.True(t, strings.HasSuffix(skill, "---\n\nGenerated by solomonar "+shipped+" from "+
		"`docs/solomonar/skills/spec/SKILL.md`; edit that file.\n"+
		"Body edits apply directly; metadata changes need adapter regeneration. Check `solomonar help` for `sync` availability.\n"+
		"Load `docs/solomonar/skills/spec/SKILL.md` and follow it.\n"), skill)
	role := files[".claude/agents/spec-reviewer.md"]
	assert.Contains(t, role, "\ntools: Read, Grep, Glob, Bash\n---\n\n")
	assert.True(t, strings.HasSuffix(role, "Load `AGENTS.md`, then `docs/solomonar/agents/spec-reviewer.md` "+
		"and every file its first lines require.\n"), role)
	assert.NotContains(t, files, ".agents")
	assert.Equal(t, map[string]project.Runtime{"claude": {}}, manifestOf(t, dir).Runtimes)
}

func manifestOf(t *testing.T, dir string) *project.Manifest {
	t.Helper()
	manifest, err := project.DecodeManifest([]byte(tree(t, dir)[project.ManifestPath]), shipped)
	require.NoError(t, err)
	return manifest
}

func TestInitWithBothRuntimes(t *testing.T) {
	t.Parallel()

	human, asJSON := committedRepo(t), committedRepo(t)
	args := runtimeArgs("codex,claude", "--adversarial-model", "codex=o3")
	paths := "AGENTS.md CLAUDE.md .claude .agents .codex .solomonar docs"

	text := runInRepo(t, human, args...)
	data := runInRepo(t, asJSON, append(args, "--json")...)

	require.Equal(t, 0, text.exit, text.stderr)
	assert.Contains(t, text.stdout, "  39 files created\n")
	assert.True(t, strings.HasSuffix(text.stdout, "\nNext:\n"+commitStep(paths)+
		"  2. Start a top-level session in "+human+" with your selected outcome:\n"+
		"       Claude Code: /solomonar Explore the next useful flow.\n"+
		"       Codex:       $solomonar Explore the next useful flow.\n"+
		"       Portable:    Use the solomonar skill to explore the next useful flow.\n"), text.stdout)
	files := tree(t, human)
	assert.Contains(t, files, ".agents/skills/solomonar/agents/openai.yaml")
	assert.Contains(t, files[".codex/agents/adversarial-reviewer.toml"], "\nmodel = \"o3\"\n")
	require.Equal(t, 0, data.exit, data.stderr)
	var envelope struct {
		Data struct {
			Choices jsontext.Value `json:"choices"`
			Next    []string       `json:"next"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(data.stdout), &envelope))
	assert.JSONEq(t, `{"runtimes": ["claude", "codex"], "extensions": {}, "adversarial_model": {"codex": "o3"}}`,
		string(envelope.Data.Choices))
	assert.Equal(t, []string{"git add " + paths, "git commit -m \"chore: initialize solomonar " + shipped + "\""},
		envelope.Data.Next)
}

func TestInitWithCodexNamesTheCodexSession(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)

	got := runInRepo(t, dir, runtimeArgs("codex")...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasSuffix(got.stdout, "\nNext:\n"+
		"  1. First commit required before task worktrees:\n"+
		"       git add AGENTS.md .agents .codex .solomonar docs && git commit -m \"chore: initialize solomonar "+shipped+"\"\n"+
		"  2. Start a top-level Codex session in "+dir+":\n"+
		"       $solomonar Explore the next useful flow.\n"), got.stdout)
}

func TestInitDetectsRuntimesOnPath(t *testing.T) {
	tests := []struct {
		name  string
		found []string
		want  map[string]project.Runtime
	}{
		{name: "claude", found: []string{"claude"}, want: map[string]project.Runtime{"claude": {}}},
		{name: "both", found: []string{"codex", "claude"}, want: map[string]project.Runtime{"claude": {}, "codex": {}}},
		{name: "copilot", found: []string{"copilot"}, want: map[string]project.Runtime{"copilot": {}}},
		{
			name: "all", found: []string{"copilot", "codex", "claude"},
			want: map[string]project.Runtime{"claude": {}, "codex": {}, "copilot": {}},
		},
		{name: "none found", want: map[string]project.Runtime{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gittest.PathWith(t, tt.found...)
			dir := gittest.NewRepo(t)

			got := runInRepo(t, dir, "init", "--ext", "none", "--yes")

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, tt.want, manifestOf(t, dir).Runtimes)
		})
	}
}

func TestInitRecordsTheLastAdversarialModel(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)

	got := runInRepo(t, dir, runtimeArgs("claude",
		"--adversarial-model", "claude=sonnet", "--adversarial-model", "claude=opus")...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, tree(t, dir)[".claude/agents/adversarial-reviewer.md"], "\nmodel: opus\n---\n")
	assert.Equal(t, map[string]project.Runtime{"claude": {AdversarialModel: "opus"}}, manifestOf(t, dir).Runtimes)
}

// claudeRepo is a repository initialized with claude and the adversarial model opus.
func claudeRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.NewRepo(t)
	got := runInRepo(t, dir, runtimeArgs("claude", "--adversarial-model", "claude=opus")...)
	require.Equal(t, 0, got.exit, got.stderr)
	return dir
}

func TestInitRerunWithRuntimesIsNoOp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "same choices", args: runtimeArgs("claude", "--adversarial-model", "claude=opus")},
		{name: "runtimes from the manifest, not PATH", args: []string{"init", "--yes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := claudeRepo(t)
			gittest.PathWith(t, "codex", "copilot")
			before, tip := tree(t, dir), refTip(t, dir)

			got := runInRepo(t, dir, tt.args...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Contains(t, got.stdout, "Already initialized with solomonar "+shipped+"; nothing to do.\n"+
				"  28 files unchanged, 1 skipped · registry")
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestInitRejectsDifferentRuntimeChoice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  []string
		found string
	}{
		{name: "runtime", args: runtimeArgs("codex"), found: "manifest runtimes [claude]; requested [codex]"},
		{
			name: "add copilot", args: runtimeArgs("claude,copilot"),
			found: "manifest runtimes [claude]; requested [claude, copilot]",
		},
		{
			name: "model", args: []string{"init", "--adversarial-model", "claude=sonnet", "--yes"},
			found: `manifest adversarial_model claude "opus"; requested "sonnet"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := claudeRepo(t)
			before, tip := tree(t, dir), refTip(t, dir)

			got := runInRepo(t, dir, tt.args...)

			assert.Equal(t, 3, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, "error: init rejected (already_initialized)\n  found: "+tt.found+"\n")
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestInitSkipsUnmanagedClaudeMD(t *testing.T) {
	t.Parallel()

	dir := committedRepo(t)
	writeFile(t, dir, "CLAUDE.md", "# Our notes\n")

	plan := runInRepo(t, dir, runtimeArgs("claude", "--dry-run")...)
	got := runInRepo(t, dir, runtimeArgs("claude")...)

	require.Equal(t, 0, plan.exit, plan.stderr)
	assert.Contains(t, plan.stdout,
		"  skip       CLAUDE.md  exists; add this line so Claude Code loads the framework:  @AGENTS.md\n")
	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "\nNext:\n"+commitStep("AGENTS.md .claude .solomonar docs"))
	assert.Equal(t, "# Our notes\n", tree(t, dir)["CLAUDE.md"])
}
