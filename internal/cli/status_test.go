package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/task"
)

const noPR = `no pull requests found for branch "feature/W-001"`

// reviewRepo stores W-001 at specialized_review, its worktree one commit ahead and one path modified.
func reviewRepo(t *testing.T) (dir, worktree, revision string) {
	t.Helper()
	dir, worktree, revision = moveRepo(t)
	gittest.Commit(t, worktree, "fix: session expiry")
	require.NoError(t, os.WriteFile(filepath.Join(worktree, filepath.FromSlash(moveSpec)), []byte("# W-001 draft\n"), 0o644))
	putTask(t, dir, task.Record{
		Status: task.StatusRunning, Stage: task.StageSpecializedReview, Spec: moveSpec, Revision: revision, Tier: new(2),
		ActiveAgents: []task.Agent{{ID: "sec-1", Role: "security-reviewer"}, {ID: "cor-1", Role: "correctness-reviewer"}},
		NextAction:   &task.NextAction{Actor: "orchestrator-1", Action: "Collect specialist reports"},
	})
	return dir, worktree, revision
}

// putRecord stores record as W-001 unchanged.
func putRecord(t *testing.T, dir string, record task.Record) {
	t.Helper()
	store := &task.Store{Git: &git.Repo{Dir: dir}}
	_, err := store.Update(t.Context(), func(reg *task.Registry) (string, error) {
		reg.Tasks["W-001"] = &record
		return "state: fixture", nil
	})
	require.NoError(t, err)
}

func updatedLine(t *testing.T, dir, updatedAt string) string {
	t.Helper()
	return "  updated   " + updatedAt + " (registry @ " + registryTip(t, dir)[:7] + ")\n"
}

func TestTaskStatusUC16(t *testing.T) {
	dir, worktree, revision := reviewRepo(t)
	head := gittest.Git(t, worktree, "rev-parse", "HEAD")
	gittest.FakeGH(t, "", noPR+"\n", 1)

	got := runInRepo(t, dir, "task", "status", "W-001")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "W-001  Login\n"+
		"record\n"+
		"  status    running / specialized_review   owner orchestrator-1   tier 2\n"+
		"  spec      "+moveSpec+"\n"+
		"  revision  "+revision+"\n"+
		"  agents    sec-1 security-reviewer, cor-1 correctness-reviewer (last reported, not liveness)\n"+
		"  blockers  none\n"+
		"  next      orchestrator-1: Collect specialist reports\n"+
		"  pr        none\n"+
		updatedLine(t, dir, "2026-10-20T15:00:00Z")+
		"git\n"+
		"  branch    feature/W-001 @ "+head+"\n"+
		"  worktree  "+worktree+"\n"+
		"  HEAD      "+head[:8]+" · 1 commit ahead of revision\n"+
		"  dirty     1 path: "+moveSpec+" (modified)\n"+
		"  spec      present in worktree and at revision\n"+
		"  merged    no\n"+
		"forge (gh)\n"+
		"  pr      none for feature/W-001\n"+
		"  checks  unknown (no PR)\n"+
		"drift\n"+
		"  head_ahead_of_revision  HEAD is 1 commit past the recorded revision\n"+
		"  dirty_worktree          1 uncommitted path; preserve it (rules §10)\n"+
		"allowed next\n"+
		"  solomonar task move W-001 --as orchestrator-1 --stage implementation --next <actor>:<action>\n"+
		"  solomonar task move W-001 --as orchestrator-1 --stage adversarial_review\n"+
		"  solomonar task block W-001 --as orchestrator-1 --kind <kind> --summary <text>\n"+
		"  solomonar task move W-001 --as orchestrator-1 --status paused --next <actor>:<action>\n"+
		"  solomonar task move W-001 --as orchestrator-1 --status cancelled --next <actor>:<action>\n", got.stdout)
	assert.Empty(t, got.stderr)
}

func TestTaskStatusJSON(t *testing.T) {
	dir, worktree, revision := reviewRepo(t)
	head := gittest.Git(t, worktree, "rev-parse", "HEAD")
	gittest.FakeGH(t, "", noPR+"\n", 1)

	got := runInRepo(t, dir, "task", "status", "W-001", "--json")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task status",
		"data": {
			"id": "W-001",
			"record": {"title": "Login", "status": "running", "stage": "specialized_review", "owner": "orchestrator-1",
				"branch": "feature/W-001", "spec": "`+moveSpec+`", "revision": "`+revision+`", "tier": 2,
				"active_agents": [{"id": "sec-1", "role": "security-reviewer"}, {"id": "cor-1", "role": "correctness-reviewer"}],
				"next_action": {"actor": "orchestrator-1", "action": "Collect specialist reports"},
				"updated_at": "2026-10-20T15:00:00Z"},
			"registry_commit": "`+registryTip(t, dir)+`",
			"facts": {
				"branch": {"name": "feature/W-001", "exists": true, "tip": "`+head+`", "contains_revision": true},
				"worktree": {"path": "`+worktree+`", "head": "`+head+`", "ahead_of_revision": 1,
					"dirty": [{"path": "`+moveSpec+`", "state": "modified"}]},
				"spec": {"in_worktree": true, "at_revision": true},
				"merge": {"merged": false, "merge_commit": null, "derived_spec_ref": null},
				"forge": {"available": true, "pr": null, "checks": "unknown"}
			},
			"drift": [
				{"code": "head_ahead_of_revision", "message": "HEAD is 1 commit past the recorded revision"},
				{"code": "dirty_worktree", "message": "1 uncommitted path; preserve it (rules §10)"}
			],
			"allowed_next": [
				"solomonar task move W-001 --as orchestrator-1 --stage implementation --next <actor>:<action>",
				"solomonar task move W-001 --as orchestrator-1 --stage adversarial_review",
				"solomonar task block W-001 --as orchestrator-1 --kind <kind> --summary <text>",
				"solomonar task move W-001 --as orchestrator-1 --status paused --next <actor>:<action>",
				"solomonar task move W-001 --as orchestrator-1 --status cancelled --next <actor>:<action>"
			]
		},
		"warnings": []
	}`, got.stdout)
}

func TestTaskStatusUC17MergedNotDone(t *testing.T) {
	f := doneRepo(t)
	worktree := filepath.Join(gittest.Resolved(t, t.TempDir()), "W-001")
	gittest.Git(t, f.dir, "worktree", "add", worktree, "feature/W-001")
	gittest.FakeGH(t, `{"number": 42, "url": "https://github.com/acme/ledger/pull/42", "state": "MERGED",
		"mergeCommit": {"oid": "`+f.merge+`"}, "statusCheckRollup": []}`, "", 0)

	got := runInRepo(t, f.dir, "task", "status", "W-001")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "W-001  Login\n"+
		"record\n"+
		"  status    running / integration   owner orchestrator-1   tier 1\n"+
		"  spec      "+moveSpec+"\n"+
		"  revision  "+f.revision+"\n"+
		"  agents    none\n"+
		"  blockers  none\n"+
		"  next      orchestrator-1: Continue\n"+
		"  pr        42\n"+
		updatedLine(t, f.dir, "2026-10-20T15:00:00Z")+
		"git\n"+
		"  branch    feature/W-001 @ "+f.revision+"\n"+
		"  worktree  "+worktree+"\n"+
		"  HEAD      "+f.revision[:8]+" · 0 commits ahead of revision\n"+
		"  dirty     clean\n"+
		"  spec      present in worktree and at revision\n"+
		"  merged    yes, by "+f.merge+" (merge commit on main)\n"+
		"  spec_ref  derived "+f.spec+":"+moveSpec+" (last spec(W-001) commit merged)\n"+
		"forge (gh)\n"+
		"  pr      #42 merged\n"+
		"  checks  none\n"+
		"drift\n"+
		"  merged_not_done  feature/W-001 is merged into main but the record is not done\n"+
		"allowed next\n"+
		"  solomonar task done W-001 --as orchestrator-1 --merge "+f.merge+
		" --spec-ref <PR \"Final specification\" line> --summary <one sentence>\n", got.stdout)
	assert.Empty(t, got.stderr)
}

func TestTaskStatusDoneRecord(t *testing.T) {
	t.Parallel()

	f := doneRepo(t)
	putRecord(t, f.dir, task.Record{
		Title: "Login", Status: task.StatusDone, Stage: task.StageComplete, PR: "42", Summary: "Password login",
		SpecRef: f.spec + ":" + moveSpec, MergeCommit: f.merge, UpdatedAt: "2026-10-21T12:00:00Z",
	})

	got := runInRepo(t, f.dir, "task", "status", "W-001", "--no-forge")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "W-001  Login\n"+
		"record\n"+
		"  status    done / complete   owner -   tier none\n"+
		"  summary   Password login\n"+
		"  spec_ref  "+f.spec+":"+moveSpec+"\n"+
		"  merge     "+f.merge+"\n"+
		"  pr        42\n"+
		updatedLine(t, f.dir, "2026-10-21T12:00:00Z")+
		"git\n"+
		"  branch    feature/W-001 @ "+f.revision+"\n"+
		"  worktree  missing\n"+
		"  HEAD      unknown (no worktree)\n"+
		"  dirty     unknown (no worktree)\n"+
		"  spec      none recorded\n"+
		"  merged    yes, by "+f.merge+" (merge commit on main)\n"+
		"forge (gh)\n"+
		"  pr      unknown (--no-forge)\n"+
		"  checks  unknown\n"+
		"drift\n"+
		"  none\n"+
		"allowed next\n"+
		"  none\n", got.stdout)
}

func TestTaskStatusPlannedRecord(t *testing.T) {
	dir, _ := registryRepo(t)
	putRecord(t, dir, task.Record{Title: "Export", Status: task.StatusPlanned, UpdatedAt: "2026-10-20T09:00:00Z"})
	gittest.FakeGH(t, "", "gh must not run with --no-forge", 1)

	got := runInRepo(t, dir, "task", "status", "W-001", "--no-forge")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "W-001  Export\n"+
		"record\n"+
		"  status    planned / -   owner -   tier none\n"+
		"  spec      none\n"+
		"  revision  none\n"+
		"  agents    none\n"+
		"  blockers  none\n"+
		"  next      none\n"+
		"  pr        none\n"+
		updatedLine(t, dir, "2026-10-20T09:00:00Z")+
		"git\n"+
		"  branch    feature/W-001 missing\n"+
		"  worktree  missing\n"+
		"  HEAD      unknown (no worktree)\n"+
		"  dirty     unknown (no worktree)\n"+
		"  spec      none recorded\n"+
		"  merged    no\n"+
		"forge (gh)\n"+
		"  pr      unknown (--no-forge)\n"+
		"  checks  unknown\n"+
		"drift\n"+
		"  none\n"+
		"allowed next\n"+
		"  solomonar task register W-001 --title <title> --owner <owner>\n", got.stdout)
	assert.Empty(t, got.stderr)
}

func TestTaskStatusWithoutGHWarns(t *testing.T) {
	dir, _, _ := reviewRepo(t)
	gittest.NoGH(t)

	human := runInRepo(t, dir, "task", "status", "W-001")
	asJSON := runInRepo(t, dir, "task", "status", "W-001", "--json")

	require.Equal(t, 0, human.exit, human.stderr)
	assert.Contains(t, human.stdout, "forge (gh)\n  pr      unknown (gh unavailable)\n  checks  unknown\ndrift\n")
	assert.Equal(t, "warning: gh unavailable: gh not found\n", human.stderr)
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.Contains(t, asJSON.stdout, `"warnings": [
    {
      "code": "forge_unavailable",
      "message": "gh unavailable: gh not found"
    }
  ]`)
}

func TestTaskStatusBlockedOwnedByOther(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	putTask(t, dir, blockedTask())
	getenv := func(name string) string {
		if name == "SOLOMONAR_OWNER" {
			return "orchestrator-2"
		}
		return ""
	}

	got := runCLIWithEnv(t, repoAssets(t), getenv, "-C", dir, "task", "status", "W-001", "--no-forge")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  status    blocked / implementation   owner orchestrator-1   tier none\n")
	assert.Contains(t, got.stdout, "  blockers  B-1 decision: Session storage (W-001/D-1); B-2 check: CI is red\n")
	assert.Contains(t, got.stdout, "drift\n"+
		"  spec_missing    "+moveSpec+" is not in the worktree or at the revision\n"+
		"  owned_by_other  SOLOMONAR_OWNER is orchestrator-2; W-001 is owned by orchestrator-1\n"+
		"allowed next\n")
}

func TestTaskStatusRejections(t *testing.T) {
	t.Parallel()

	const fix = "  fix:   run solomonar help\nnothing was written\n"
	tests := []struct {
		name   string
		args   []string
		exit   int
		stderr string
	}{
		{name: "without ID", args: []string{}, exit: 2, stderr: "error: task ID is required (invalid_usage)\n" + fix},
		{name: "extra argument", args: []string{"W-001", "W-002"}, exit: 2, stderr: "error: unexpected argument \"W-002\" (invalid_usage)\n" + fix},
		{
			name: "unknown task", args: []string{"W-007"}, exit: 3,
			stderr: "error: task status W-007 rejected (task_not_found)\n" +
				"  check: the task exists in the registry\n" +
				"  found: W-007 is not in the registry\n" +
				"  fix:   solomonar task show\n" +
				"nothing was written\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir, _ := registryRepo(t)

			got := runInRepo(t, dir, append([]string{"task", "status"}, tt.args...)...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Equal(t, tt.stderr, got.stderr)
		})
	}
}

func TestTaskStatusHelp(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "help", "task", "status")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Usage: solomonar task status <ID> [--json] [--no-forge]\n\n"+
		"Compare the stored record with Git and gh facts, then print the drift and the commands valid next.\n"+
		"Read-only; exit 0 whatever the drift. --no-forge skips gh; SOLOMONAR_OWNER feeds owned_by_other.\n\n"+
		"Global flags: see 'solomonar help'.\n", got.stdout)
}
