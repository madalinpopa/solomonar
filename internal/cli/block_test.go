package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/task"
)

func implementationTask() task.Record {
	return task.Record{Status: task.StatusRunning, Stage: task.StageImplementation, Spec: moveSpec}
}

// blockedTask is blocked at implementation with B-1, a decision, and B-2, a check.
func blockedTask() task.Record {
	return task.Record{
		Status: task.StatusBlocked, Stage: task.StageImplementation, Spec: moveSpec, LastBlocker: 2,
		Blockers: []task.Blocker{
			{ID: "B-1", Kind: task.BlockerDecision, Summary: "Session storage", Ref: "W-001/D-1"},
			{ID: "B-2", Kind: task.BlockerCheck, Summary: "CI is red"},
		},
	}
}

// ucBlockArgs are the task block arguments of UC-15.
func ucBlockArgs() []string {
	return []string{
		"task", "block", "W-001", "--as", "orchestrator-1", "--kind", "decision", "--summary", "Session storage",
		"--ref", "W-001/D-1", "--next", "user:Decide W-001/D-1: session storage", "--status", "blocked",
	}
}

func TestTaskBlockAndUnblockDecision(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	record := implementationTask()
	record.ActiveAgents = []task.Agent{{ID: "impl-1", Role: "implementer"}}
	putTask(t, dir, record)

	block := runInRepo(t, dir, ucBlockArgs()...)

	require.Equal(t, 0, block.exit, block.stderr)
	assert.Equal(t, "W-001  running/implementation → blocked/implementation\n"+
		"  blocker  B-1 decision: Session storage (W-001/D-1)\n"+
		"  next     user: Decide W-001/D-1: session storage\n"+
		"  state(W-001): blocked/implementation, block B-1 decision\n"+
		pushedLine(t, dir), block.stdout)
	assert.Empty(t, block.stderr)

	unblock := runInRepo(t, dir, "task", "unblock", "W-001", "--as", "orchestrator-1", "B-1", "--status", "running",
		"--next", "impl-1:Apply W-001/D-1, continue AC-2")

	require.Equal(t, 0, unblock.exit, unblock.stderr)
	assert.Equal(t, "W-001  blocked/implementation → running/implementation\n"+
		"  cleared  B-1 decision: Session storage\n"+
		"  next     impl-1: Apply W-001/D-1, continue AC-2\n"+
		"  state(W-001): running/implementation, unblock B-1\n"+
		pushedLine(t, dir), unblock.stdout)
}

func TestTaskBlockWithoutRefKeepsStatus(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, implementationTask())

	got := runInRepo(t, dir, "task", "block", "W-001", "--as", "orchestrator-1", "--kind", "check",
		"--summary", "CI is red")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "W-001  running/implementation\n"+
		"  blocker  B-1 check: CI is red\n"+
		"  state(W-001): running/implementation, block B-1 check\n"+
		pushedLine(t, dir), got.stdout)
}

func TestTaskUnblockLeavesOtherBlockerOpen(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, blockedTask())

	got := runInRepo(t, dir, "task", "unblock", "W-001", "B-2", "--as", "orchestrator-1")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "W-001  blocked/implementation\n"+
		"  cleared  B-2 check: CI is red\n"+
		"  state(W-001): blocked/implementation, unblock B-2\n"+
		pushedLine(t, dir), got.stdout)
}

const blockedRecordJSON = `{"title": "Login", "status": "blocked", "stage": "implementation",
	"owner": "orchestrator-1", "confirmed": true, "branch": "feature/W-001", "spec": "docs/work/W-001-login.md",
	"blockers": [{"id": "B-1", "kind": "decision", "summary": "Session storage", "ref": "W-001/D-1"}],
	"last_blocker": 1, "updated_at": "2026-10-21T12:00:00Z",
	"next_action": {"actor": "user", "action": "Decide W-001/D-1: session storage"}}`

func TestTaskBlockJSON(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, implementationTask())

	got := runInRepo(t, dir, append(ucBlockArgs(), "--json")...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task block",
		"data": {
			"id": "W-001",
			"from": {"status": "running", "stage": "implementation"},
			"to": {"status": "blocked", "stage": "implementation"},
			"changed": ["status", "blockers", "next_action"],
			"record": `+blockedRecordJSON+`,
			"commit": "`+registryTip(t, dir)+`",
			"pushed": true
		},
		"warnings": []
	}`, got.stdout)
}

func TestTaskUnblockJSON(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, blockedTask())

	got := runInRepo(t, dir, "task", "unblock", "W-001", "B-2", "--as", "orchestrator-1", "--json")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task unblock",
		"data": {
			"id": "W-001",
			"from": {"status": "blocked", "stage": "implementation"},
			"to": {"status": "blocked", "stage": "implementation"},
			"changed": ["blockers"],
			"record": {"title": "Login", "status": "blocked", "stage": "implementation",
				"owner": "orchestrator-1", "confirmed": true, "branch": "feature/W-001", "spec": "docs/work/W-001-login.md",
				"blockers": [{"id": "B-1", "kind": "decision", "summary": "Session storage", "ref": "W-001/D-1"}],
				"last_blocker": 2, "updated_at": "2026-10-21T12:00:00Z",
				"next_action": {"actor": "orchestrator-1", "action": "Continue"}},
			"commit": "`+registryTip(t, dir)+`",
			"pushed": true
		},
		"warnings": []
	}`, got.stdout)
}

func TestTaskBlockAndUnblockDryRunWriteNothing(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, implementationTask())
	before := registryTip(t, dir)

	human := runInRepo(t, dir, append(ucBlockArgs(), "--dry-run")...)
	asJSON := runInRepo(t, dir, append(ucBlockArgs(), "--dry-run", "--json")...)

	require.Equal(t, 0, human.exit, human.stderr)
	assert.Equal(t, "Would block W-001  running/implementation → blocked/implementation\n"+
		"  blocker  B-1 decision: Session storage (W-001/D-1)\n"+
		"  next     user: Decide W-001/D-1: session storage\n"+
		"  state(W-001): blocked/implementation, block B-1 decision\n", human.stdout)
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task block",
		"data": {
			"id": "W-001",
			"from": {"status": "running", "stage": "implementation"},
			"to": {"status": "blocked", "stage": "implementation"},
			"changed": ["status", "blockers", "next_action"],
			"record": `+blockedRecordJSON+`,
			"dry_run": true
		},
		"warnings": []
	}`, asJSON.stdout)
	assert.Equal(t, before, registryTip(t, dir))

	putTask(t, dir, blockedTask())
	before = registryTip(t, dir)

	unblock := runInRepo(t, dir, "task", "unblock", "W-001", "B-2", "--as", "orchestrator-1", "--dry-run")

	require.Equal(t, 0, unblock.exit, unblock.stderr)
	assert.Equal(t, "Would unblock W-001  blocked/implementation\n"+
		"  cleared  B-2 check: CI is red\n"+
		"  state(W-001): blocked/implementation, unblock B-2\n", unblock.stdout)
	assert.Equal(t, before, registryTip(t, dir))
}

func TestTaskBlockAndUnblockUsageErrors(t *testing.T) {
	t.Parallel()

	const fix = "  fix:   run solomonar help\nnothing was written\n"
	tests := []struct {
		name   string
		args   []string
		stderr string
	}{
		{
			name: "block without ID", args: []string{"block", "--kind", "check", "--summary", "CI"},
			stderr: "error: task ID is required (invalid_usage)\n" + fix,
		},
		{
			name: "block with extra argument", args: []string{"block", "W-001", "B-1", "--kind", "check"},
			stderr: "error: unexpected argument \"B-1\" (invalid_usage)\n" + fix,
		},
		{
			name: "block without kind", args: []string{"block", "W-001", "--summary", "CI"},
			stderr: "error: --kind is required (invalid_usage)\n" + fix,
		},
		{
			name: "block without summary", args: []string{"block", "W-001", "--kind", "check"},
			stderr: "error: --summary is required (invalid_usage)\n" + fix,
		},
		{
			name: "block without owner", args: []string{"block", "W-001", "--kind", "check", "--summary", "CI"},
			stderr: "error: task block needs an acting owner (owner_required)\n  fix:   pass --as <owner>\nnothing was written\n",
		},
		{
			name: "unblock without ID", args: []string{"unblock"},
			stderr: "error: task ID is required (invalid_usage)\n" + fix,
		},
		{
			name: "unblock without blocker", args: []string{"unblock", "W-001", "--as", "orchestrator-1"},
			stderr: "error: blocker ID is required (invalid_usage)\n" + fix,
		},
		{
			name: "unblock with extra argument", args: []string{"unblock", "W-001", "B-1", "B-2"},
			stderr: "error: unexpected argument \"B-2\" (invalid_usage)\n" + fix,
		},
		{
			name: "unblock without owner", args: []string{"unblock", "W-001", "B-1"},
			stderr: "error: task unblock needs an acting owner (owner_required)\n  fix:   pass --as <owner>\nnothing was written\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := runCLI(t, repoAssets(t), append([]string{"-C", t.TempDir(), "task"}, tt.args...)...)

			assert.Equal(t, 2, got.exit)
			assert.Empty(t, got.stdout)
			assert.Equal(t, tt.stderr, got.stderr)
		})
	}
}

func TestTaskBlockTakesOwnerFromEnvironment(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, implementationTask())
	getenv := func(name string) string {
		if name == "SOLOMONAR_OWNER" {
			return "orchestrator-1"
		}
		return ""
	}

	got := runCLIWithEnv(t, repoAssets(t), getenv, "-C", dir, "task", "block", "W-001", "--kind", "check",
		"--summary", "CI is red")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  blocker  B-1 check: CI is red\n")
}

func TestTaskBlockAndUnblockRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		record task.Record
		args   []string
		exit   int
		stderr string
	}{
		{
			name: "decision without ref", record: implementationTask(), exit: 2,
			args: []string{"block", "W-001", "--as", "orchestrator-1", "--kind", "decision", "--summary", "Session storage"},
			stderr: "error: task block W-001 rejected (invalid_value)\n" +
				"  check: --kind decision needs --ref <task ID>/D-<n>\n" +
				"  fix:   pass --ref W-001/D-1\n" +
				"nothing was written\n",
		},
		{
			name: "unknown blocker", record: blockedTask(), exit: 3,
			args: []string{"unblock", "W-001", "B-9", "--as", "orchestrator-1"},
			stderr: "error: task unblock W-001 rejected (blocker_not_found)\n" +
				"  check: the blocker is open on the task\n" +
				"  found: B-9 is not an open blocker of W-001\n" +
				"  fix:   solomonar task show W-001\n" +
				"nothing was written\n",
		},
		{
			name: "last blocker without status", exit: 3,
			record: task.Record{
				Status: task.StatusBlocked, Stage: task.StageImplementation, Spec: moveSpec, LastBlocker: 1,
				Blockers: []task.Blocker{{ID: "B-1", Kind: task.BlockerCheck, Summary: "CI is red"}},
			},
			args: []string{"unblock", "W-001", "B-1", "--as", "orchestrator-1"},
			stderr: "error: task unblock W-001 rejected (blockers_required)\n" +
				"  check: a blocked task needs an open blocker\n" +
				"  found: no open blocker\n" +
				"  fix:   pass --status running\n" +
				"nothing was written\n",
		},
		{
			name: "another owner", record: implementationTask(), exit: 3,
			args: []string{"block", "W-001", "--as", "orchestrator-2", "--kind", "check", "--summary", "CI is red"},
			stderr: "error: task block W-001 rejected (owner_mismatch)\n" +
				"  check: only the owner writes a task's record\n" +
				"  found: owner is orchestrator-1; acting as orchestrator-2\n" +
				"  fix:   act as orchestrator-1, or ask the user for an explicit handoff: " +
				"solomonar task handoff W-001 --owner orchestrator-2\n" +
				"nothing was written\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir, _, _ := moveRepo(t)
			putTask(t, dir, tt.record)
			before := registryTip(t, dir)

			got := runInRepo(t, dir, append([]string{"task"}, tt.args...)...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Equal(t, tt.stderr, got.stderr)
			assert.Equal(t, before, registryTip(t, dir))
		})
	}
}
