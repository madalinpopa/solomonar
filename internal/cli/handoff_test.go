package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/task"
)

const handoffClosing = "  The previous owner must stop writing this record.\n" +
	"  Inspect with solomonar task status W-001, then solomonar task confirm W-001 --as orchestrator-3.\n"

func TestTaskHandoffRetargetsNextAction(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, blockedTask())

	got := runInRepo(t, dir, "task", "handoff", "W-001", "--owner", "orchestrator-3")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "W-001  owner orchestrator-1 → orchestrator-3 (blocked/implementation)\n"+
		"  confirmed  no\n"+
		"  next       orchestrator-3: Continue\n"+
		"  state(W-001): blocked/implementation, handoff orchestrator-1 -> orchestrator-3\n"+
		pushedLine(t, dir)+handoffClosing, got.stdout)
	assert.Empty(t, got.stderr)
	assert.Equal(t, "state(W-001): blocked/implementation, handoff orchestrator-1 -> orchestrator-3",
		gittest.Git(t, dir, "log", "-1", "--format=%s", task.Ref))
}

func TestTaskHandoffKeepsNextActionOfUser(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	record := blockedTask()
	record.NextAction = &task.NextAction{Actor: "user", Action: "Decide W-001/D-1"}
	putTask(t, dir, record)

	got := runInRepo(t, dir, "task", "handoff", "W-001", "--owner", "orchestrator-3")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "W-001  owner orchestrator-1 → orchestrator-3 (blocked/implementation)\n"+
		"  confirmed  no\n"+
		"  state(W-001): blocked/implementation, handoff orchestrator-1 -> orchestrator-3\n"+
		pushedLine(t, dir)+handoffClosing, got.stdout)
}

func TestTaskHandoffIgnoresOwnerEnvironment(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, implementationTask())
	getenv := func(name string) string {
		if name == "SOLOMONAR_OWNER" {
			return "orchestrator-1"
		}
		return ""
	}

	got := runCLIWithEnv(t, repoAssets(t), getenv, "-C", dir, "task", "handoff", "W-001", "--owner", "orchestrator-3",
		"--next", "orchestrator-3:Resume at implementation")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "  next       orchestrator-3: Resume at implementation\n")
}

const handoffRecordJSON = `{"title": "Login", "status": "running", "stage": "implementation",
	"owner": "orchestrator-3", "branch": "feature/W-001", "spec": "docs/work/W-001-login.md",
	"updated_at": "2026-10-21T12:00:00Z", "next_action": {"actor": "user", "action": "Review"}}`

func TestTaskHandoffJSON(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, implementationTask())

	got := runInRepo(t, dir, "task", "handoff", "W-001", "--owner", "orchestrator-3", "--next", "user:Review", "--json")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task handoff",
		"data": {
			"id": "W-001",
			"from": {"status": "running", "stage": "implementation"},
			"to": {"status": "running", "stage": "implementation"},
			"changed": ["owner", "confirmed", "next_action"],
			"record": `+handoffRecordJSON+`,
			"commit": "`+registryTip(t, dir)+`",
			"pushed": true
		},
		"warnings": []
	}`, got.stdout)
}

func TestTaskHandoffDryRunWritesNothing(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, implementationTask())
	before := registryTip(t, dir)
	args := []string{"task", "handoff", "W-001", "--owner", "orchestrator-3", "--next", "user:Review", "--dry-run"}

	human := runInRepo(t, dir, args...)
	asJSON := runInRepo(t, dir, append(args, "--json")...)

	require.Equal(t, 0, human.exit, human.stderr)
	assert.Equal(t, "Would hand off W-001  owner orchestrator-1 → orchestrator-3 (running/implementation)\n"+
		"  confirmed  no\n"+
		"  next       user: Review\n"+
		"  state(W-001): running/implementation, handoff orchestrator-1 -> orchestrator-3\n", human.stdout)
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task handoff",
		"data": {
			"id": "W-001",
			"from": {"status": "running", "stage": "implementation"},
			"to": {"status": "running", "stage": "implementation"},
			"changed": ["owner", "confirmed", "next_action"],
			"record": `+handoffRecordJSON+`,
			"dry_run": true
		},
		"warnings": []
	}`, asJSON.stdout)
	assert.Equal(t, before, registryTip(t, dir))
}

func TestTaskHandoffUsageErrors(t *testing.T) {
	t.Parallel()

	const fix = "  fix:   run solomonar help\nnothing was written\n"
	tests := []struct {
		name   string
		args   []string
		stderr string
	}{
		{name: "without ID", args: []string{"--owner", "orchestrator-3"}, stderr: "error: task ID is required (invalid_usage)\n" + fix},
		{name: "extra argument", args: []string{"W-001", "W-002", "--owner", "orchestrator-3"}, stderr: "error: unexpected argument \"W-002\" (invalid_usage)\n" + fix},
		{name: "without owner", args: []string{"W-001"}, stderr: "error: --owner is required (invalid_usage)\n" + fix},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := runCLI(t, repoAssets(t), append([]string{"-C", t.TempDir(), "task", "handoff"}, tt.args...)...)

			assert.Equal(t, 2, got.exit)
			assert.Empty(t, got.stdout)
			assert.Equal(t, tt.stderr, got.stderr)
		})
	}
}

func TestTaskHandoffTakesNoAsFlag(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "-C", t.TempDir(), "task", "handoff", "W-001", "--owner", "orchestrator-3",
		"--as", "orchestrator-1")

	assert.Equal(t, 2, got.exit)
	assert.Contains(t, got.stderr, "(invalid_usage)")
}

func TestTaskHandoffRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		exit   int
		stderr string
	}{
		{
			name: "unknown task", exit: 3, args: []string{"W-009", "--owner", "orchestrator-3"},
			stderr: "error: task handoff W-009 rejected (task_not_found)\n",
		},
		{
			name: "done task", exit: 3, args: []string{"W-003", "--owner", "orchestrator-3"},
			stderr: "error: task handoff W-003 rejected (done_final)\n",
		},
		{
			name: "planned task", exit: 3, args: []string{"W-004", "--owner", "orchestrator-3"},
			stderr: "error: task handoff W-004 rejected (not_registered)\n" +
				"  check: the task is registered\n" +
				"  found: W-004 is planned\n" +
				"  fix:   solomonar task register W-004 --title <title> --owner orchestrator-3\n" +
				"nothing was written\n",
		},
		{
			name: "invalid owner", exit: 2, args: []string{"W-001", "--owner", "Orchestrator"},
			stderr: "error: task handoff W-001 rejected (invalid_value)\n" +
				"  check: --owner takes an agent ID like [a-z]+-[0-9]+\n" +
				"  found: Orchestrator\n" +
				"  fix:   pass --owner orchestrator-1\n" +
				"nothing was written\n",
		},
		{
			name: "same owner", exit: 3, args: []string{"W-001", "--owner", "orchestrator-1"},
			stderr: "error: task handoff W-001 rejected (handoff_same_owner)\n" +
				"  check: the new owner differs from the current owner\n" +
				"  found: W-001 is owned by orchestrator-1\n" +
				"nothing was written\n",
		},
		{
			name: "owner in use", exit: 3, args: []string{"W-001", "--owner", "orchestrator-2"},
			stderr: "error: task handoff W-001 rejected (owner_in_use)\n" +
				"  check: an orchestrator owns exactly one active task\n" +
				"  found: orchestrator-2 owns W-002 (running)\n" +
				"  fix:   choose a new owner ID\n" +
				"nothing was written\n",
		},
		{
			name: "next names the previous owner", exit: 2,
			args: []string{"W-001", "--owner", "orchestrator-3", "--next", "orchestrator-1:Continue"},
			stderr: "error: task handoff W-001 rejected (invalid_value)\n" +
				"  check: --next actor is user, the owner, or an active agent\n" +
				"  found: orchestrator-1:Continue\n" +
				"  fix:   allowed actors: user, orchestrator-3\n" +
				"nothing was written\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir, _, _ := moveRepo(t)
			putTask(t, dir, implementationTask())
			putHandoffFixtures(t, dir)
			before := registryTip(t, dir)

			got := runInRepo(t, dir, append([]string{"task", "handoff"}, tt.args...)...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Contains(t, got.stderr, tt.stderr)
			assert.Equal(t, before, registryTip(t, dir))
		})
	}
}

// putHandoffFixtures stores W-002 running for orchestrator-2, W-003 done, and W-004 planned.
func putHandoffFixtures(t *testing.T, dir string) {
	t.Helper()
	store := &task.Store{Git: &git.Repo{Dir: dir}}
	_, err := store.Update(t.Context(), func(reg *task.Registry) (string, error) {
		reg.Tasks["W-002"] = &task.Record{
			Title: "Export", Status: task.StatusRunning, Stage: task.StageSetup, Owner: "orchestrator-2",
			Branch: "feature/W-002", UpdatedAt: "2026-10-20T15:00:00Z",
			NextAction: &task.NextAction{Actor: "orchestrator-2", Action: "Continue"},
		}
		reg.Tasks["W-003"] = &task.Record{
			Title: "Setup", Status: task.StatusDone, Stage: task.StageComplete, Summary: "Merged the setup",
			SpecRef:     "7cd18e4f2b1c7e5a9d3f6b8a0c2e4d6f8a1b3c5d:docs/spec.md",
			MergeCommit: "8cd18e4f2b1c7e5a9d3f6b8a0c2e4d6f8a1b3c5d", UpdatedAt: "2026-10-20T15:00:00Z",
		}
		reg.Tasks["W-004"] = &task.Record{Title: "Later", Status: task.StatusPlanned, UpdatedAt: "2026-10-20T15:00:00Z"}
		return "state: fixture", nil
	})
	require.NoError(t, err)
}

func TestTaskHandoffHelp(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "help", "task", "handoff")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Usage: solomonar task handoff <ID> --owner <new-owner> [--next <actor>:<action>] [--json] [--dry-run]\n\n"+
		"Hand off a task to a new owner, on the user's instruction, as one registry commit.\n"+
		"It takes no --as and ignores SOLOMONAR_OWNER; the previous owner must stop writing the record.\n\n"+
		"Global flags: see 'solomonar help'.\n", got.stdout)
}
