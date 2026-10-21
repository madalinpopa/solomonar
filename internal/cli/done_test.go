package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/task"
)

// doneFixture is a repository whose feature/W-001 holds a spec commit, then the reviewed revision.
type doneFixture struct {
	dir, spec, revision, merge string
}

func (f doneFixture) args(extra ...string) []string {
	return append([]string{
		"task", "done", "W-001", "--as", "orchestrator-1", "--merge", f.merge,
		"--spec-ref", f.spec + ":" + moveSpec, "--summary", "  Password login  ",
	}, extra...)
}

// featureRepo stores W-001 at integration and leaves main one commit ahead of the unmerged fork.
func featureRepo(t *testing.T) doneFixture {
	t.Helper()
	dir, _ := registryRepo(t)
	addRemote(t, dir)
	gittest.Commit(t, dir, "initial")
	gittest.Git(t, dir, "checkout", "-b", "feature/W-001")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "work"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, filepath.FromSlash(moveSpec)), []byte("# W-001\n"), 0o644))
	gittest.Git(t, dir, "add", moveSpec)
	gittest.Git(t, dir, "commit", "-m", "spec(W-001): settle login")
	f := doneFixture{dir: dir, spec: gittest.Git(t, dir, "rev-parse", "HEAD")}
	gittest.Commit(t, dir, "feat: login")
	f.revision = gittest.Git(t, dir, "rev-parse", "HEAD")
	gittest.Git(t, dir, "checkout", "main")
	gittest.Commit(t, dir, "main work")
	putTask(t, dir, task.Record{
		Status: task.StatusRunning, Stage: task.StageIntegration, Spec: moveSpec, Revision: f.revision, Tier: new(1), PR: "42",
	})
	return f
}

// doneRepo merges feature/W-001 into main with a merge commit.
func doneRepo(t *testing.T) doneFixture {
	t.Helper()
	f := featureRepo(t)
	gittest.Git(t, f.dir, "merge", "--no-ff", "-m", "Merge W-001", "feature/W-001")
	f.merge = gittest.Git(t, f.dir, "rev-parse", "HEAD")
	return f
}

func (f doneFixture) verified() string {
	return "Verified\n" +
		"  merge     " + f.merge[:8] + " is a merge commit reachable from main\n" +
		"  revision  " + f.revision[:8] + " is contained in the merge\n" +
		"  spec_ref  resolves; " + f.spec[:8] + " is the last spec(W-001) commit merged\n"
}

func (f doneFixture) compactRecord() string {
	return "Compact record\n" +
		"  {\n" +
		"    \"title\": \"Login\",\n" +
		"    \"status\": \"done\",\n" +
		"    \"stage\": \"complete\",\n" +
		"    \"pr\": \"42\",\n" +
		"    \"summary\": \"Password login\",\n" +
		"    \"spec_ref\": \"" + f.spec + ":" + moveSpec + "\",\n" +
		"    \"merge_commit\": \"" + f.merge + "\",\n" +
		"    \"updated_at\": \"2026-10-21T12:00:00Z\"\n" +
		"  }\n"
}

func (f doneFixture) recordJSON() string {
	return `{"title": "Login", "status": "done", "stage": "complete", "pr": "42", "summary": "Password login",
		"spec_ref": "` + f.spec + ":" + moveSpec + `", "merge_commit": "` + f.merge + `",
		"updated_at": "2026-10-21T12:00:00Z"}`
}

func TestTaskDoneUC20(t *testing.T) {
	f := doneRepo(t)
	sibling := filepath.Join("..", filepath.Base(f.dir)+"-worktrees", "W-001")
	gittest.Git(t, f.dir, "worktree", "add", sibling, "feature/W-001")

	got := runInRepo(t, f.dir, f.args()...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, f.verified()+
		"W-001  running/integration → done/complete\n"+
		"  state(W-001): done/complete, merged "+f.merge[:7]+"\n"+
		pushedLine(t, f.dir)+
		f.compactRecord()+
		"Next: delete the worktree only after confirming it holds no uncommitted or unmerged work:\n"+
		"  git worktree remove "+sibling+"\n", got.stdout)
	assert.Empty(t, got.stderr)
	assert.Equal(t, "state(W-001): done/complete, merged "+f.merge[:7], gittest.Git(t, f.dir, "log", "-1", "--format=%s", task.Ref))
}

func TestTaskDoneNextLineNamesTheWorktree(t *testing.T) {
	tests := []struct {
		name     string
		worktree bool
	}{
		{name: "without a worktree", worktree: false},
		{name: "with a worktree outside the sibling directory", worktree: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := doneRepo(t)
			var next string
			if tt.worktree {
				path := filepath.Join(gittest.Resolved(t, t.TempDir()), "W-001")
				gittest.Git(t, f.dir, "worktree", "add", path, "feature/W-001")
				next = "Next: delete the worktree only after confirming it holds no uncommitted or unmerged work:\n" +
					"  git worktree remove " + path + "\n"
			}

			got := runInRepo(t, f.dir, f.args()...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, f.verified()+
				"W-001  running/integration → done/complete\n"+
				"  state(W-001): done/complete, merged "+f.merge[:7]+"\n"+
				pushedLine(t, f.dir)+f.compactRecord()+next, got.stdout)
		})
	}
}

func TestTaskDoneJSON(t *testing.T) {
	f := doneRepo(t)

	got := runInRepo(t, f.dir, f.args("--json")...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task done",
		"data": {
			"id": "W-001",
			"from": {"status": "running", "stage": "integration"},
			"to": {"status": "done", "stage": "complete"},
			"changed": ["status", "stage", "owner", "branch", "spec", "revision", "tier", "next_action",
				"summary", "spec_ref", "merge_commit"],
			"record": `+f.recordJSON()+`,
			"commit": "`+registryTip(t, f.dir)+`",
			"pushed": true
		},
		"warnings": []
	}`, got.stdout)
}

func TestTaskDoneDryRunWritesNothing(t *testing.T) {
	f := doneRepo(t)
	gittest.Git(t, f.dir, "worktree", "add", filepath.Join("..", filepath.Base(f.dir)+"-worktrees", "W-001"), "feature/W-001")
	before := registryTip(t, f.dir)

	human := runInRepo(t, f.dir, f.args("--dry-run")...)
	asJSON := runInRepo(t, f.dir, f.args("--dry-run", "--json")...)

	require.Equal(t, 0, human.exit, human.stderr)
	assert.Equal(t, f.verified()+
		"Would complete W-001  running/integration → done/complete\n"+
		"  state(W-001): done/complete, merged "+f.merge[:7]+"\n"+
		f.compactRecord(), human.stdout)
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task done",
		"data": {
			"id": "W-001",
			"from": {"status": "running", "stage": "integration"},
			"to": {"status": "done", "stage": "complete"},
			"changed": ["status", "stage", "owner", "branch", "spec", "revision", "tier", "next_action",
				"summary", "spec_ref", "merge_commit"],
			"record": `+f.recordJSON()+`,
			"dry_run": true
		},
		"warnings": []
	}`, asJSON.stdout)
	assert.Equal(t, before, registryTip(t, f.dir))
}

func TestTaskDoneUsageErrors(t *testing.T) {
	const fix = "  fix:   run solomonar help\nnothing was written\n"
	sha := "14091a9f2461267ee7e02525b4f1f2923f1c9849"
	tests := []struct {
		name   string
		args   []string
		stderr string
	}{
		{name: "without ID", args: []string{}, stderr: "error: task ID is required (invalid_usage)\n" + fix},
		{name: "extra argument", args: []string{"W-001", "W-002"}, stderr: "error: unexpected argument \"W-002\" (invalid_usage)\n" + fix},
		{name: "without merge", args: []string{"W-001", "--spec-ref", sha + ":" + moveSpec}, stderr: "error: --merge is required (invalid_usage)\n" + fix},
		{name: "without spec_ref", args: []string{"W-001", "--merge", sha}, stderr: "error: --spec-ref is required (invalid_usage)\n" + fix},
		{
			name: "without summary", args: []string{"W-001", "--merge", sha, "--spec-ref", sha + ":" + moveSpec},
			stderr: "error: --summary is required (invalid_usage)\n" + fix,
		},
		{
			name: "without owner", args: []string{"W-001", "--merge", sha, "--spec-ref", sha + ":" + moveSpec, "--summary", "Login"},
			stderr: "error: task done needs an acting owner (owner_required)\n  fix:   pass --as <owner>\nnothing was written\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runCLI(t, repoAssets(t), append([]string{"-C", t.TempDir(), "task", "done"}, tt.args...)...)

			assert.Equal(t, 2, got.exit)
			assert.Empty(t, got.stdout)
			assert.Equal(t, tt.stderr, got.stderr)
		})
	}
}

func TestTaskDoneRejectsSquashMerge(t *testing.T) {
	f := featureRepo(t)
	gittest.Git(t, f.dir, "merge", "--squash", "feature/W-001")
	gittest.Git(t, f.dir, "commit", "-m", "Login (#42)")
	f.merge = gittest.Git(t, f.dir, "rev-parse", "HEAD")
	before := registryTip(t, f.dir)

	got := runInRepo(t, f.dir, f.args()...)

	assert.Equal(t, 3, got.exit)
	assert.Empty(t, got.stdout)
	assert.Equal(t, "error: task done W-001 rejected (merge_not_merge_commit)\n"+
		"  check: the merge is a merge commit\n"+
		"  found: "+f.merge[:8]+" has one parent\n"+
		"  fix:   spec_ref cannot survive; report to the user (rules §11)\n"+
		"nothing was written\n", got.stderr)
	assert.Equal(t, before, registryTip(t, f.dir))
}

func TestTaskDoneRejections(t *testing.T) {
	tests := []struct {
		name   string
		args   func(f doneFixture) []string
		exit   int
		stderr func(f doneFixture) string
	}{
		{
			name: "spec_ref is not the last spec commit", exit: 3,
			args: func(f doneFixture) []string {
				return []string{
					"task", "done", "W-001", "--as", "orchestrator-1", "--merge", f.merge,
					"--spec-ref", f.revision + ":" + moveSpec, "--summary", "Password login",
				}
			},
			stderr: func(f doneFixture) string {
				return "error: task done W-001 rejected (spec_ref_not_final)\n" +
					"  check: the spec_ref is the last spec(W-001) commit merged with the recorded spec\n" +
					"  found: " + f.revision[:8] + ":" + moveSpec + " is not the last spec(W-001) commit merged\n" +
					"  fix:   pass --spec-ref " + f.spec + ":" + moveSpec + "\n" +
					"nothing was written\n"
			},
		},
		{
			name: "short merge SHA", exit: 2,
			args: func(f doneFixture) []string {
				return []string{
					"task", "done", "W-001", "--as", "orchestrator-1", "--merge", f.merge[:7],
					"--spec-ref", f.spec + ":" + moveSpec, "--summary", "Password login",
				}
			},
			stderr: func(f doneFixture) string {
				return "error: task done W-001 rejected (invalid_value)\n" +
					"  check: --merge takes a full commit SHA\n" +
					"  found: " + f.merge[:7] + "\n" +
					"  fix:   git rev-parse " + f.merge[:7] + "\n" +
					"nothing was written\n"
			},
		},
		{
			name: "another owner", exit: 3,
			args: func(f doneFixture) []string { return append(f.args()[:3], f.args()[5:]...) },
			stderr: func(doneFixture) string {
				return "error: task done W-001 rejected (owner_mismatch)\n"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := doneRepo(t)
			before := registryTip(t, f.dir)
			getenv := func(name string) string {
				if name == "SOLOMONAR_OWNER" {
					return "orchestrator-2"
				}
				return ""
			}

			got := runCLIWithEnv(t, repoAssets(t), getenv, append([]string{"-C", f.dir}, tt.args(f)...)...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, tt.stderr(f))
			assert.Equal(t, before, registryTip(t, f.dir))
		})
	}
}

func TestTaskDoneHelp(t *testing.T) {
	got := runCLI(t, repoAssets(t), "help", "task", "done")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Usage: solomonar task done <ID> --merge <full-sha> --spec-ref <full-sha>:<path> --summary <text>\n"+
		"                    [--as <owner>] [--json] [--dry-run]\n\n"+
		"Complete a task after its merge: verify the merge commit and the final spec commit with Git only,\n"+
		"then write the compact record as one registry commit. --as defaults to SOLOMONAR_OWNER.\n\n"+
		"Global flags: see 'solomonar help'.\n", got.stdout)
}
