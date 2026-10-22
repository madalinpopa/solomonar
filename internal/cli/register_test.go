package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/task"
)

func TestTaskRegisterPrintsConfirmationAndWorktree(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	addRemote(t, dir)

	got := runInRepo(t, dir, "task", "register", "--title", "Login", "--owner", "orchestrator-1")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Registered W-001: Login\n"+
		"  status    reserved / setup\n"+
		"  owner     orchestrator-1\n"+
		"  branch    feature/W-001 (create it now)\n"+
		"  next      orchestrator-1: Create feature/W-001 and its worktree\n"+
		"  state(W-001): reserved/setup, registered by orchestrator-1\n"+
		"  registry @ "+registryTip(t, dir)+" · pushed origin/workflow-state\n"+
		"\n"+
		"Registration confirmed. The next task may start.\n"+
		"Create the worktree:\n"+
		"  git worktree add -b feature/W-001 ../"+filepath.Base(dir)+"-worktrees/W-001 main\n", got.stdout)
	assert.Empty(t, got.stderr)
	assert.Equal(t, "state(W-001): reserved/setup, registered by orchestrator-1",
		gittest.Git(t, dir, "log", "-1", "--format=%s", task.Ref))
}

// claimRepo holds a planned W-001 and a planned W-002 that depends on it.
func claimRepo(t *testing.T) string {
	t.Helper()
	dir, _ := registryRepo(t)
	addRemote(t, dir)
	runInRepo(t, dir, "task", "plan", "--title", "Setup")
	runInRepo(t, dir, "task", "plan", "--title", "Export", "--depends-on", "W-001")
	return dir
}

func TestTaskRegisterClaimWarnsInOrder(t *testing.T) {
	t.Parallel()

	dir := claimRepo(t)

	got := runInRepo(t, dir, "task", "register", "W-002", "--title", "Export as CSV", "--owner", "orchestrator-2")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "Registered W-002: Export as CSV\n")
	assert.Equal(t, "warning: title changed from \"Export\"\n"+
		"warning: W-001 is not done; register, then block with task block --kind dependency\n", got.stderr)
}

func TestTaskRegisterJSON(t *testing.T) {
	t.Parallel()

	dir := claimRepo(t)

	got := runInRepo(t, dir, "task", "register", "W-002", "--json", "--title", "Export as CSV", "--owner", "orchestrator-2")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task register",
		"data": {
			"id": "W-002",
			"record": {"title": "Export as CSV", "status": "reserved", "stage": "setup", "owner": "orchestrator-2",
				"branch": "feature/W-002", "updated_at": "2026-10-21T12:00:00Z", "depends_on": ["W-001"],
				"next_action": {"actor": "orchestrator-2", "action": "Create feature/W-002 and its worktree"}},
			"commit": "`+registryTip(t, dir)+`",
			"pushed": true
		},
		"warnings": [
			{"code": "title_changed", "message": "title changed from \"Export\""},
			{"code": "dependency_not_done",
				"message": "W-001 is not done; register, then block with task block --kind dependency"}
		]
	}`, got.stdout)
}

func TestTaskRegisterWithoutRemoteStillConfirms(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)

	got := runInRepo(t, dir, "task", "register", "--title", "Login", "--owner", "orchestrator-1")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "\n  registry @ "+registryTip(t, dir)+"\n\nRegistration confirmed.")
	assert.Equal(t, "warning: no remote \"origin\"; the registry stays local\n", got.stderr)
}

func TestTaskRegisterDryRunWritesNothing(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	before := registryTip(t, dir)

	human := runInRepo(t, dir, "task", "register", "--dry-run", "--title", "Login", "--owner", "orchestrator-1")
	asJSON := runInRepo(t, dir, "task", "register", "--dry-run", "--json", "--title", "Login", "--owner", "orchestrator-1")

	require.Equal(t, 0, human.exit, human.stderr)
	assert.Equal(t, "Would register W-001: Login\n"+
		"  status    reserved / setup\n"+
		"  owner     orchestrator-1\n"+
		"  branch    feature/W-001 (create it now)\n"+
		"  next      orchestrator-1: Create feature/W-001 and its worktree\n"+
		"  state(W-001): reserved/setup, registered by orchestrator-1\n", human.stdout)
	assert.Empty(t, human.stderr)
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task register",
		"data": {
			"id": "W-001",
			"record": {"title": "Login", "status": "reserved", "stage": "setup", "owner": "orchestrator-1",
				"branch": "feature/W-001", "updated_at": "2026-10-21T12:00:00Z",
				"next_action": {"actor": "orchestrator-1", "action": "Create feature/W-001 and its worktree"}},
			"dry_run": true
		},
		"warnings": []
	}`, asJSON.stdout)
	assert.Equal(t, before, registryTip(t, dir))
}

func TestTaskRegisterRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		exit   int
		stderr string
	}{
		{
			name: "missing title", args: []string{"--owner", "orchestrator-1"}, exit: 2,
			stderr: "error: --title is required (invalid_usage)\n",
		},
		{
			name: "missing owner", args: []string{"--title", "t"}, exit: 2,
			stderr: "error: --owner is required (invalid_usage)\n",
		},
		{
			name: "two positionals", args: []string{"W-001", "W-002", "--title", "t", "--owner", "orchestrator-1"}, exit: 2,
			stderr: "error: unexpected argument \"W-002\" (invalid_usage)\n",
		},
		{
			name: "not planned", args: []string{"W-001", "--title", "t", "--owner", "orchestrator-2"}, exit: 3,
			stderr: "error: task register W-001 rejected (not_planned)\n" +
				"  check: task register claims only a planned task\n" +
				"  found: W-001 is reserved\n" +
				"  fix:   solomonar task show W-001; resume the task instead\n",
		},
		{
			name: "owner in use", args: []string{"--title", "t", "--owner", "orchestrator-1"}, exit: 3,
			stderr: "error: task register rejected (owner_in_use)\n",
		},
		{
			name: "invalid owner", args: []string{"--title", "t", "--owner", "Orchestrator"}, exit: 2,
			stderr: "error: task register rejected (invalid_value)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir, _ := registryRepo(t)
			runInRepo(t, dir, "task", "register", "--title", "Login", "--owner", "orchestrator-1")
			before := registryTip(t, dir)

			got := runInRepo(t, dir, append([]string{"task", "register"}, tt.args...)...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, tt.stderr)
			assert.Equal(t, before, registryTip(t, dir))
		})
	}
}

func TestTaskRegisterFromWorktreeUsesAbsolutePath(t *testing.T) {
	t.Parallel()

	dir, _ := registryRepo(t)
	gittest.Commit(t, dir, "initial")
	worktree := gittest.Worktree(t, dir, "feature/W-000")

	got := runInRepo(t, worktree, "task", "register", "--title", "Login", "--owner", "orchestrator-1")

	require.Equal(t, 0, got.exit, got.stderr)
	want := filepath.Join(filepath.Dir(dir), filepath.Base(dir)+"-worktrees", "W-001")
	assert.Contains(t, got.stdout, "\n  git worktree add -b feature/W-001 "+want+" main\n")
}
