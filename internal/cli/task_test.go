package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

func runInRepo(t *testing.T, dir string, args ...string) result {
	t.Helper()
	return runCLI(t, repoAssets(t), append([]string{"-C", dir}, args...)...)
}

// registryRepo creates a repository with an empty registry and no remote.
func registryRepo(t *testing.T) (string, *task.Store) {
	t.Helper()
	dir := gittest.NewRepo(t)
	store := &task.Store{Git: &git.Repo{Dir: dir}}
	_, err := store.Create(t.Context())
	require.NoError(t, err)
	return dir, store
}

func addRemote(t *testing.T, dir string) {
	t.Helper()
	gittest.Git(t, dir, "remote", "add", "origin", gittest.NewBareRemote(t))
}

func registryTip(t *testing.T, dir string) string {
	t.Helper()
	return gittest.Git(t, dir, "rev-parse", task.Ref)
}

func TestTaskPlanPrintsCommitAndPush(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	addRemote(t, dir)

	got := runInRepo(t, dir, "task", "plan", "--title", "Export expenses as CSV")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Planned W-001: Export expenses as CSV\n"+
		"  state(W-001): planned, plan \"Export expenses as CSV\"\n"+
		"  registry @ "+registryTip(t, dir)+" · pushed origin/workflow-state\n", got.stdout)
	assert.Empty(t, got.stderr)
	assert.Equal(t, `state(W-001): planned, plan "Export expenses as CSV"`,
		gittest.Git(t, dir, "log", "-1", "--format=%s", task.Ref))
}

func TestTaskPlanJSON(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	addRemote(t, dir)
	runInRepo(t, dir, "task", "plan", "--title", "first")

	got := runInRepo(t, dir, "task", "plan", "--json", "--title", "second", "--depends-on", "W-001", "--related", "W-001")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task plan",
		"data": {
			"id": "W-002",
			"record": {"title": "second", "status": "planned", "updated_at": "2026-10-21T12:00:00Z",
				"depends_on": ["W-001"], "related_task": "W-001"},
			"commit": "`+registryTip(t, dir)+`",
			"pushed": true
		},
		"warnings": []
	}`, got.stdout)
}

func TestTaskPlanWithoutRemoteWarns(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)

	got := runInRepo(t, dir, "task", "plan", "--title", "local")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "\n  registry @ "+registryTip(t, dir)+"\n")
	assert.Equal(t, "warning: no remote \"origin\"; the registry stays local\n", got.stderr)
}

func TestTaskPlanDryRunWritesNothing(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	before := registryTip(t, dir)

	human := runInRepo(t, dir, "task", "plan", "--dry-run", "--title", "maybe")
	asJSON := runInRepo(t, dir, "task", "plan", "--dry-run", "--json", "--title", "maybe")

	require.Equal(t, 0, human.exit, human.stderr)
	assert.Equal(t, "Would plan W-001: maybe\n  state(W-001): planned, plan \"maybe\"\n", human.stdout)
	assert.Empty(t, human.stderr)
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task plan",
		"data": {
			"id": "W-001",
			"record": {"title": "maybe", "status": "planned", "updated_at": "2026-10-21T12:00:00Z"},
			"dry_run": true
		},
		"warnings": []
	}`, asJSON.stdout)
	assert.Equal(t, before, registryTip(t, dir))
}

func TestTaskPlanRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		exit   int
		stderr string
	}{
		{
			name: "missing title", args: []string{"--depends-on", "W-001"}, exit: 2,
			stderr: "error: --title is required (invalid_usage)\n",
		},
		{
			name: "unknown dependency", args: []string{"--title", "t", "--depends-on", "W-007"}, exit: 3,
			stderr: "error: task plan rejected (task_not_found)\n" +
				"  check: the task exists in the registry\n" +
				"  found: W-007 is not in the registry\n" +
				"  fix:   solomonar task show\n",
		},
		{
			name: "blank title", args: []string{"--title", "  "}, exit: 2,
			stderr: "error: task plan rejected (invalid_value)\n",
		},
		{
			name: "malformed related", args: []string{"--title", "t", "--related", "7"}, exit: 2,
			stderr: "error: task plan rejected (invalid_value)\n",
		},
		{
			name: "dry run checks too", args: []string{"--dry-run", "--title", "t", "--depends-on", "W-007"}, exit: 3,
			stderr: "error: task plan rejected (task_not_found)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir, _ := registryRepo(t)
			before := registryTip(t, dir)

			got := runInRepo(t, dir, append([]string{"task", "plan"}, tt.args...)...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, tt.stderr)
			assert.Equal(t, before, registryTip(t, dir))
		})
	}
}

func TestTaskCommandsOutsideRegistry(t *testing.T) {
	outside := gittest.Resolved(t, t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(outside))
	uninitialized := gittest.NewRepo(t)
	tests := []struct {
		name string
		dir  string
		args []string
		want string
	}{
		{
			name: "plan outside a repository", dir: outside, args: []string{"task", "plan", "--title", "t"},
			want: "error: " + outside + " is not a Git repository (not_a_repository)\n" +
				"  fix:   run inside a Git repository or pass -C <path>\n",
		},
		{
			name: "show outside a repository", dir: outside, args: []string{"task", "show"},
			want: "(not_a_repository)",
		},
		{
			name: "plan without registry", dir: uninitialized, args: []string{"task", "plan", "--title", "t"},
			want: "error: task plan rejected (not_initialized)\n",
		},
		{
			name: "show with ID without registry", dir: uninitialized, args: []string{"task", "show", "W-001"},
			want: "error: task show W-001 rejected (not_initialized)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runInRepo(t, tt.dir, tt.args...)

			assert.Equal(t, 3, got.exit)
			assert.Contains(t, got.stderr, tt.want)
		})
	}
}

// showRepo holds a running W-001, a done W-002, and a planned W-010.
func showRepo(t *testing.T) string {
	t.Helper()
	dir, store := registryRepo(t)
	_, err := store.Update(t.Context(), func(reg *task.Registry) (string, error) {
		reg.Tasks["W-010"] = &task.Record{Title: "Later", Status: task.StatusPlanned, UpdatedAt: "2026-10-20T15:00:00Z"}
		reg.Tasks["W-001"] = &task.Record{
			Title: "Import", Status: task.StatusRunning, Stage: task.StageSpecializedReview, Owner: "orchestrator-1",
			Branch: "feature/W-001", Spec: "docs/spec.md", Revision: "7cd18e4f2b1c7e5a9d3f6b8a0c2e4d6f8a1b3c5d",
			Tier: new(2), UpdatedAt: "2026-10-20T15:00:00Z",
			NextAction: &task.NextAction{Actor: "orchestrator-1", Action: "Collect specialist reports"},
		}
		reg.Tasks["W-002"] = &task.Record{
			Title: "Setup", Status: task.StatusDone, Stage: task.StageComplete, Summary: "Merged the setup",
			SpecRef:     "7cd18e4f2b1c7e5a9d3f6b8a0c2e4d6f8a1b3c5d:docs/spec.md",
			MergeCommit: "8cd18e4f2b1c7e5a9d3f6b8a0c2e4d6f8a1b3c5d", UpdatedAt: "2026-10-20T15:00:00Z",
		}
		return "state: fixture", nil
	})
	require.NoError(t, err)
	return dir
}

func TestTaskShowOverview(t *testing.T) {
	t.Parallel()

	dir := showRepo(t)

	got := runInRepo(t, dir, "task", "show")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, ""+
		"ID     STATUS     STAGE               OWNER           NEXT\n"+
		"W-001  running    specialized_review  orchestrator-1  orchestrator-1: Collect specialist reports\n"+
		"W-002  done       complete            -               Merged the setup\n"+
		"W-010  planned    -                   -               -\n"+
		"registry refs/heads/workflow-state @ "+registryTip(t, dir)[:7]+" · 2026-01-01T00:00:00Z\n", got.stdout)
}

func TestTaskShowEmptyRegistry(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)

	got := runInRepo(t, dir, "task", "show")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "ID  STATUS     STAGE  OWNER  NEXT\n"+
		"registry refs/heads/workflow-state @ "+registryTip(t, dir)[:7]+" · 2026-01-01T00:00:00Z\n", got.stdout)
}

func TestTaskShowRecordPrintsStoredJSON(t *testing.T) {
	t.Parallel()

	dir := showRepo(t)

	got := runInRepo(t, dir, "task", "show", "W-010")

	require.Equal(t, 0, got.exit, got.stderr)
	//nolint:testifylint // the exact bytes are the contract, not only the JSON value
	assert.Equal(t, "{\n  \"title\": \"Later\",\n  \"status\": \"planned\",\n  \"updated_at\": \"2026-10-20T15:00:00Z\"\n}\n",
		got.stdout)
}

func TestTaskShowJSON(t *testing.T) {
	t.Parallel()

	dir := showRepo(t)
	tip := registryTip(t, dir)

	overview := runInRepo(t, dir, "task", "show", "--json")
	record := runInRepo(t, dir, "task", "show", "W-010", "--json")

	require.Equal(t, 0, overview.exit, overview.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task show",
		"data": {"registry": `+gittest.Git(t, dir, "show", tip+":state.json")+`, "commit": "`+tip+`"},
		"warnings": []
	}`, overview.stdout)
	require.Equal(t, 0, record.exit, record.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task show",
		"data": {
			"id": "W-010",
			"record": {"title": "Later", "status": "planned", "updated_at": "2026-10-20T15:00:00Z"},
			"commit": "`+tip+`"
		},
		"warnings": []
	}`, record.stdout)
}

func TestTaskShowUnknownID(t *testing.T) {
	t.Parallel()

	dir := showRepo(t)

	got := runInRepo(t, dir, "task", "show", "W-404")

	assert.Equal(t, 3, got.exit)
	assert.Contains(t, got.stderr, "error: task show W-404 rejected (task_not_found)\n")
}

func TestTaskGroupNeedsSubcommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "task alone", args: []string{"task"}, want: `unknown command "task"`},
		{name: "unknown subcommand", args: []string{"task", "nope"}, want: `unknown command "task nope"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := runCLI(t, repoAssets(t), tt.args...)

			assert.Equal(t, 2, got.exit)
			assert.Contains(t, got.stderr, "error: "+tt.want+" (unknown_command)")
		})
	}
}

func TestTaskHelpIsTheSameEitherWay(t *testing.T) {
	t.Parallel()

	viaHelp := runCLI(t, repoAssets(t), "help", "task", "plan")
	viaFlag := runCLI(t, repoAssets(t), "task", "plan", "-h")

	require.Equal(t, 0, viaHelp.exit, viaHelp.stderr)
	assert.Contains(t, viaHelp.stdout, "Usage: solomonar task plan --title <title>")
	assert.Equal(t, viaHelp.stdout, viaFlag.stdout)
}

func TestVerboseTracesGitOnStderr(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)

	got := runInRepo(t, dir, "-v", "task", "show")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stderr, "+ git rev-parse --path-format=absolute --show-toplevel --git-common-dir\n+ git version\n")
	assert.Contains(t, got.stderr, "+ git cat-file blob ")
}

// putManifest writes a manifest of the given schema naming remote and mainBranch.
func putManifest(t *testing.T, dir string, schema int, remote, mainBranch string) {
	t.Helper()
	manifest := project.Manifest{
		SchemaVersion: schema, FrameworkVersion: "0.1.0",
		Project:  project.Project{Name: "ledger", MainBranch: mainBranch, Remote: remote},
		Runtimes: map[string]project.Runtime{}, Extensions: map[string]project.Extension{}, Files: map[string]project.File{},
	}
	data, err := manifest.Encode()
	require.NoError(t, err)
	writeFile(t, dir, project.ManifestPath, string(data))
}

// trunkRepo is doneRepo with main renamed to trunk and a manifest naming trunk.
func trunkRepo(t *testing.T) doneFixture {
	t.Helper()
	f := doneRepo(t)
	gittest.Git(t, f.dir, "branch", "-m", "main", "trunk")
	putManifest(t, f.dir, project.ManifestSchemaVersion, "origin", "trunk")
	return f
}

func TestTaskPlanPushesToManifestRemote(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	gittest.Git(t, dir, "remote", "add", "upstream", gittest.NewBareRemote(t))
	putManifest(t, dir, project.ManifestSchemaVersion, "upstream", "trunk")

	got := runInRepo(t, dir, "task", "plan", "--title", "Export")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  registry @ "+registryTip(t, dir)+" · pushed upstream/workflow-state\n")
	assert.Empty(t, got.stderr)
}

func TestTaskRegisterBranchesFromManifestMainBranch(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	putManifest(t, dir, project.ManifestSchemaVersion, "origin", "trunk")

	got := runInRepo(t, dir, "task", "register", "--title", "Login", "--owner", "orchestrator-1")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "\n  git worktree add -b feature/W-001 ../"+filepath.Base(dir)+"-worktrees/W-001 trunk\n")
}

func TestTaskDoneChecksManifestMainBranch(t *testing.T) {
	t.Parallel()

	f := trunkRepo(t)

	got := runInRepo(t, f.dir, f.args()...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  merge     "+f.merge[:8]+" is a merge commit reachable from trunk\n")
}

func TestTaskStatusMergedOnManifestMainBranch(t *testing.T) {
	t.Parallel()

	f := trunkRepo(t)

	got := runInRepo(t, f.dir, "task", "status", "W-001", "--no-forge")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  merged    yes, by "+f.merge+" (merge commit on trunk)\n")
	assert.Contains(t, got.stdout, "  merged_not_done   feature/W-001 is merged into trunk but the record is not done\n")
}

func TestTaskCommandsRejectUnreadableManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema int
		remote string
		args   []string
		exit   int
		stderr string
	}{
		{
			name: "newer manifest on a write", schema: 2, remote: "origin",
			args: []string{"task", "move", "W-001", "--as", "orchestrator-1", "--status", "running", "--stage", "specification"}, exit: 3,
			stderr: "error: task move W-001 rejected (cli_outdated)\n" +
				"  check: manifest schema version\n" +
				"  found: manifest schema 2, this CLI supports 1\n" +
				"  fix:   install a newer solomonar release\n",
		},
		{
			name: "invalid manifest on a write", schema: 1, remote: "",
			args: []string{"task", "move", "W-001", "--as", "orchestrator-1", "--status", "running", "--stage", "specification"}, exit: 1,
			stderr: "error: task move W-001 failed (manifest_invalid)\n" +
				"  check: manifest schema\n" +
				"  found: project.remote is missing\n" +
				"  fix:   inspect .solomonar/manifest.json; never edit it by hand\n",
		},
		{
			name: "newer manifest on show", schema: 2, remote: "origin", args: []string{"task", "show", "W-001"}, exit: 3,
			stderr: "error: task show W-001 rejected (cli_outdated)\n",
		},
		{
			name: "invalid manifest on show", schema: 1, remote: "", args: []string{"task", "show", "W-001"}, exit: 1,
			stderr: "error: task show W-001 failed (manifest_invalid)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir, _ := registryRepo(t)
			putTask(t, dir, reservedTask())
			putManifest(t, dir, tt.schema, tt.remote, "main")
			before := registryTip(t, dir)

			got := runInRepo(t, dir, tt.args...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, tt.stderr)
			assert.Equal(t, before, registryTip(t, dir))
		})
	}
}
