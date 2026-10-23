package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

const moveSpec = "docs/work/W-001-login.md"

// moveRepo holds a registry, a remote, and a feature/W-001 worktree with a committed spec.
func moveRepo(t *testing.T) (dir, worktree, revision string) {
	t.Helper()
	dir, _ = registryRepo(t)
	addRemote(t, dir)
	gittest.Commit(t, dir, "initial")
	worktree = gittest.Worktree(t, dir, "feature/W-001")
	require.NoError(t, os.MkdirAll(filepath.Join(worktree, "docs", "work"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(worktree, filepath.FromSlash(moveSpec)), []byte(approvedSpec), 0o644))
	gittest.Git(t, worktree, "add", moveSpec)
	gittest.Git(t, worktree, "commit", "-m", "spec")
	return dir, worktree, gittest.Git(t, worktree, "rev-parse", "HEAD")
}

const approvedSpec = "# W-001: Login\nStatus: Approved\n\n## Implementation plan\n\n- S-1: feat(W-001): add login\n"

// checkpoint commits the S-1 step in the worktree and returns its full SHA.
func checkpoint(t *testing.T, worktree string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(worktree, "login.go"), []byte("package login\n"), 0o644))
	gittest.Git(t, worktree, "add", "login.go")
	gittest.Git(t, worktree, "commit", "-m", "feat(W-001): add login", "-m", "Step: W-001/S-1")
	return gittest.Git(t, worktree, "rev-parse", "HEAD")
}

// putTask stores record as W-001 owned by orchestrator-1, confirmed past setup, with a next action.
func putTask(t *testing.T, dir string, record task.Record) {
	t.Helper()
	record.Title, record.Owner, record.Branch, record.UpdatedAt = "Login", "orchestrator-1", "feature/W-001", "2026-10-20T15:00:00Z"
	record.Confirmed = record.Stage != task.StageSetup
	if record.NextAction == nil {
		record.NextAction = &task.NextAction{Actor: "orchestrator-1", Action: "Continue"}
	}
	store := &task.Store{Git: &git.Repo{Dir: dir}}
	_, err := store.Update(t.Context(), func(reg *task.Registry) (string, error) {
		reg.Tasks["W-001"] = &record
		return "state: fixture", nil
	})
	require.NoError(t, err)
}

// setupFields returns the base and relative worktree that leaving setup records for moveRepo.
func setupFields(t *testing.T, dir, worktree string) (base, relative string) {
	t.Helper()
	relative, err := filepath.Rel(dir, worktree)
	require.NoError(t, err)
	return gittest.Git(t, dir, "rev-parse", "main"), filepath.ToSlash(relative)
}

func reservedTask() task.Record {
	return task.Record{Status: task.StatusReserved, Stage: task.StageSetup}
}

func pushedLine(t *testing.T, dir string) string {
	t.Helper()
	return "  registry @ " + registryTip(t, dir) + " · pushed origin/workflow-state\n"
}

func TestTaskMoveAdvancesThroughStages(t *testing.T) {
	t.Parallel()

	dir, worktree, _ := moveRepo(t)
	putTask(t, dir, reservedTask())

	setup := runInRepo(t, dir, "task", "move", "W-001", "--as", "orchestrator-1", "--status", "running",
		"--stage", "specification", "--next", "orchestrator-1:Run work-spec")

	require.Equal(t, 0, setup.exit, setup.stderr)
	base, relative := setupFields(t, dir, worktree)
	assert.Equal(t, "W-001  reserved/setup → running/specification\n"+
		"  confirmed  yes\n"+
		"  base       "+base+" (on feature/W-001)\n"+
		"  worktree   "+relative+" (listed by git worktree list)\n"+
		"  next       orchestrator-1: Run work-spec\n"+
		"  state(W-001): running/specification, status running, stage specification, confirmed, base "+base[:7]+
		", worktree "+relative+", next orchestrator-1\n"+
		pushedLine(t, dir), setup.stdout)
	assert.Empty(t, setup.stderr)

	spec := runInRepo(t, dir, "task", "move", "W-001", "--as", "orchestrator-1", "--spec", moveSpec,
		"--stage", "implementation", "--agent", "impl-1:implementer", "--next", "impl-1:Implement AC-1 to AC-4")

	require.Equal(t, 0, spec.exit, spec.stderr)
	assert.Equal(t, "W-001  running/specification → running/implementation\n"+
		"  spec    "+moveSpec+" (found in "+worktree+")\n"+
		"  agents  impl-1 implementer\n"+
		"  next    impl-1: Implement AC-1 to AC-4\n"+
		"  state(W-001): running/implementation, stage implementation, spec "+moveSpec+", agents impl-1, next impl-1\n"+
		pushedLine(t, dir), spec.stdout)

	revision := checkpoint(t, worktree)
	review := runInRepo(t, dir, "task", "move", "W-001", "--as", "orchestrator-1", "--revision", revision,
		"--tier", "2", "--stage", "specialized_review", "--agent", "sec-1:security-reviewer",
		"--agent", "cor-1:correctness-reviewer", "--agent", "spec-1:spec-reviewer",
		"--next", "orchestrator-1:Collect specialist reports")

	require.Equal(t, 0, review.exit, review.stderr)
	assert.Equal(t, "W-001  running/implementation → running/specialized_review\n"+
		"  revision  "+revision+" (on feature/W-001)\n"+
		"  tier      2: specialists, then adversarial\n"+
		"  agents    sec-1 security-reviewer, cor-1 correctness-reviewer, spec-1 spec-reviewer\n"+
		"  next      orchestrator-1: Collect specialist reports\n"+
		"  state(W-001): running/specialized_review, stage specialized_review, revision "+revision[:7]+
		", tier 2, agents sec-1 cor-1 spec-1, next orchestrator-1\n"+
		pushedLine(t, dir), review.stdout)
}

func TestTaskMoveFindsSpecAtRevision(t *testing.T) {
	t.Parallel()

	dir, worktree, revision := moveRepo(t)
	gittest.Git(t, dir, "worktree", "remove", worktree)
	putTask(t, dir, task.Record{Status: task.StatusRunning, Stage: task.StageSpecification, Revision: revision})

	got := runInRepo(t, dir, "task", "move", "W-001", "--as", "orchestrator-1", "--spec", moveSpec,
		"--stage", "implementation")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "\n  spec  "+moveSpec+" (found at "+revision[:7]+")\n")
}

func TestTaskMoveTakesOwnerFromEnvironment(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, reservedTask())
	getenv := func(name string) string {
		if name == "SOLOMONAR_OWNER" {
			return "orchestrator-1"
		}
		return ""
	}

	got := runCLIWithEnv(t, repoAssets(t), getenv, "-C", dir, "task", "move", "W-001", "--status", "running",
		"--stage", "specification", "--next", "orchestrator-1:Run work-spec")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "W-001  reserved/setup → running/specification\n")
}

func TestTaskMovePauseCancelAndResume(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, task.Record{Status: task.StatusRunning, Stage: task.StageImplementation, Spec: moveSpec})
	tests := []struct {
		name   string
		args   []string
		stdout string
		stderr string
	}{
		{
			name: "pause", args: []string{"--status", "paused", "--next", "orchestrator-1:Resume at implementation"},
			stdout: "W-001  running/implementation → paused/implementation\n" +
				"  next  orchestrator-1: Resume at implementation\n" +
				"  state(W-001): paused/implementation, status paused, next orchestrator-1\n",
		},
		{
			name: "resume", args: []string{"--status", "running"},
			stdout: "W-001  paused/implementation → running/implementation\n" +
				"  state(W-001): running/implementation, status running\n",
		},
		{
			name: "cancel", args: []string{"--status", "cancelled", "--next", "user:Superseded by W-005"},
			stdout: "W-001  running/implementation → cancelled/implementation\n" +
				"  next  user: Superseded by W-005\n" +
				"  note: branch feature/W-001 and its worktree are kept untouched (rules section 10)\n" +
				"  state(W-001): cancelled/implementation, status cancelled, next user\n",
		},
		{
			name: "resume cancelled", args: []string{"--status", "running", "--resume-cancelled"},
			stdout: "W-001  cancelled/implementation → running/implementation\n" +
				"  state(W-001): running/implementation, status running\n",
			stderr: "warning: resuming a cancelled task requires the user's explicit instruction (rules section 7)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runInRepo(t, dir, append([]string{"task", "move", "W-001", "--as", "orchestrator-1"}, tt.args...)...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, tt.stdout+pushedLine(t, dir), got.stdout)
			assert.Equal(t, tt.stderr, got.stderr)
		})
	}
}

func TestTaskMoveCancelKeepsPR(t *testing.T) {
	t.Parallel()

	dir, _, revision := moveRepo(t)
	putTask(t, dir, task.Record{
		Status: task.StatusRunning, Stage: task.StageIntegration, Spec: moveSpec, Revision: revision,
		Tier: new(0), PR: "42",
	})

	got := runInRepo(t, dir, "task", "move", "W-001", "--as", "orchestrator-1", "--status", "cancelled",
		"--next", "user:Superseded")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "\n  note: branch feature/W-001, its worktree, and PR are kept untouched (rules section 10)\n")
}

func TestTaskMoveUnchangedWritesNothing(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, task.Record{Status: task.StatusRunning, Stage: task.StageSpecification})
	before := registryTip(t, dir)

	human := runInRepo(t, dir, "task", "move", "W-001", "--as", "orchestrator-1", "--stage", "specification")
	asJSON := runInRepo(t, dir, "task", "move", "W-001", "--json", "--as", "orchestrator-1", "--stage", "specification")

	require.Equal(t, 0, human.exit, human.stderr)
	assert.Equal(t, "W-001  running/specification (unchanged)\n", human.stdout)
	assert.Empty(t, human.stderr)
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task move",
		"data": {
			"id": "W-001",
			"from": {"status": "running", "stage": "specification"},
			"to": {"status": "running", "stage": "specification"},
			"changed": [],
			"record": {"title": "Login", "status": "running", "stage": "specification", "owner": "orchestrator-1", "confirmed": true,
				"branch": "feature/W-001", "updated_at": "2026-10-20T15:00:00Z",
				"next_action": {"actor": "orchestrator-1", "action": "Continue"}},
			"commit": "`+before+`",
			"pushed": false
		},
		"warnings": []
	}`, asJSON.stdout)
	assert.Equal(t, before, registryTip(t, dir))
}

func TestTaskMoveJSON(t *testing.T) {
	t.Parallel()

	dir, worktree, _ := moveRepo(t)
	putTask(t, dir, reservedTask())
	base, relative := setupFields(t, dir, worktree)

	got := runInRepo(t, dir, "task", "move", "W-001", "--json", "--as", "orchestrator-1", "--status", "running",
		"--stage", "specification", "--next", "orchestrator-1:Run work-spec")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task move",
		"data": {
			"id": "W-001",
			"from": {"status": "reserved", "stage": "setup"},
			"to": {"status": "running", "stage": "specification"},
			"changed": ["status", "stage", "confirmed", "base", "worktree", "next_action"],
			"record": {"title": "Login", "status": "running", "stage": "specification", "owner": "orchestrator-1",
				"confirmed": true, "branch": "feature/W-001", "base": "`+base+`", "worktree": "`+relative+`",
				"updated_at": "2026-10-21T12:00:00Z",
				"next_action": {"actor": "orchestrator-1", "action": "Run work-spec"}},
			"commit": "`+registryTip(t, dir)+`",
			"pushed": true
		},
		"warnings": []
	}`, got.stdout)
}

func TestTaskMoveDryRunWritesNothing(t *testing.T) {
	t.Parallel()

	dir, worktree, _ := moveRepo(t)
	putTask(t, dir, reservedTask())
	base, relative := setupFields(t, dir, worktree)
	before := registryTip(t, dir)
	args := []string{
		"task", "move", "W-001", "--dry-run", "--as", "orchestrator-1", "--status", "running",
		"--stage", "specification", "--next", "orchestrator-1:Run work-spec",
	}

	human := runInRepo(t, dir, args...)
	asJSON := runInRepo(t, dir, append(args, "--json")...)
	unchanged := runInRepo(t, dir, "task", "move", "W-001", "--dry-run", "--as", "orchestrator-1", "--stage", "setup")

	require.Equal(t, 0, human.exit, human.stderr)
	assert.Equal(t, "Would move W-001  reserved/setup → running/specification\n"+
		"  confirmed  yes\n"+
		"  base       "+base+" (on feature/W-001)\n"+
		"  worktree   "+relative+" (listed by git worktree list)\n"+
		"  next       orchestrator-1: Run work-spec\n"+
		"  state(W-001): running/specification, status running, stage specification, confirmed, base "+base[:7]+
		", worktree "+relative+", next orchestrator-1\n",
		human.stdout)
	require.Equal(t, 0, asJSON.exit, asJSON.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task move",
		"data": {
			"id": "W-001",
			"from": {"status": "reserved", "stage": "setup"},
			"to": {"status": "running", "stage": "specification"},
			"changed": ["status", "stage", "confirmed", "base", "worktree", "next_action"],
			"record": {"title": "Login", "status": "running", "stage": "specification", "owner": "orchestrator-1",
				"confirmed": true, "branch": "feature/W-001", "base": "`+base+`", "worktree": "`+relative+`",
				"updated_at": "2026-10-21T12:00:00Z",
				"next_action": {"actor": "orchestrator-1", "action": "Run work-spec"}},
			"dry_run": true
		},
		"warnings": []
	}`, asJSON.stdout)
	require.Equal(t, 0, unchanged.exit, unchanged.stderr)
	assert.Equal(t, "Would move W-001  reserved/setup (unchanged)\n", unchanged.stdout)
	assert.Equal(t, before, registryTip(t, dir))
}

func TestTaskMoveRejections(t *testing.T) {
	t.Parallel()

	implementation := task.Record{Status: task.StatusRunning, Stage: task.StageImplementation, Spec: moveSpec}
	tests := []struct {
		name   string
		record task.Record
		args   []string
		exit   int
		stderr string
	}{
		{
			name: "missing ID", record: implementation, args: []string{"--as", "orchestrator-1", "--stage", "integration"},
			exit: 2, stderr: "error: task ID is required (invalid_usage)\n  fix:   run solomonar help\nnothing was written\n",
		},
		{
			name: "second positional", record: implementation, args: []string{"W-001", "W-002", "--stage", "integration"},
			exit: 2, stderr: "error: unexpected argument \"W-002\" (invalid_usage)\n  fix:   run solomonar help\nnothing was written\n",
		},
		{
			name: "no field flag", record: implementation, args: []string{"W-001", "--as", "orchestrator-1", "--dry-run"},
			exit: 2, stderr: "error: at least one field flag is required (invalid_usage)\n  fix:   run solomonar help\nnothing was written\n",
		},
		{
			name: "no acting owner", record: implementation, args: []string{"W-001", "--stage", "integration"},
			exit: 2, stderr: "error: task move needs an acting owner (owner_required)\n  fix:   pass --as <owner>\nnothing was written\n",
		},
		{
			name: "tier required", record: implementation,
			args: []string{"W-001", "--as", "orchestrator-1", "--stage", "specialized_review"}, exit: 3,
			stderr: "error: task move W-001 rejected (tier_required)\n" +
				"  check: a stage after implementation needs a recorded tier\n" +
				"  found: implementation -> specialized_review\n" +
				"  fix:   triage the candidate and pass --revision <full-sha> --tier <0|1|2> with the stage move\n" +
				"nothing was written\n",
		},
		{
			name: "short revision", record: implementation,
			args: []string{"W-001", "--as", "orchestrator-1", "--revision", "7cd18e4", "--tier", "1", "--stage", "specialized_review"},
			exit: 3, stderr: "error: task move W-001 rejected (revision_not_full_sha)\n" +
				"  check: --revision must be a full 40-character SHA\n" +
				"  found: 7cd18e4\n" +
				"  fix:   git rev-parse 7cd18e4\n" +
				"nothing was written\n",
		},
		{
			name: "tier 1 skips adversarial review",
			record: task.Record{
				Status: task.StatusRunning, Stage: task.StageSpecializedReview, Spec: moveSpec,
				Revision: "7cd18e45bb55c3be9465c9f893683bc05216f529", Tier: new(1),
			},
			args: []string{"W-001", "--as", "orchestrator-1", "--stage", "adversarial_review"}, exit: 3,
			stderr: "error: task move W-001 rejected (stage_not_allowed)\n" +
				"  check: tier 1 skips adversarial_review (rules section 7)\n" +
				"  found: specialized_review -> adversarial_review\n" +
				"  fix:   --stage integration, or record tier 2 if the triage changed: --tier 2\n" +
				"nothing was written\n",
		},
		{
			name: "another owner", record: implementation,
			args: []string{"W-001", "--stage", "integration", "--as", "orchestrator-2"}, exit: 3,
			stderr: "error: task move W-001 rejected (owner_mismatch)\n" +
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

			got := runInRepo(t, dir, append([]string{"task", "move"}, tt.args...)...)

			assert.Equal(t, tt.exit, got.exit)
			assert.Empty(t, got.stdout)
			assert.Equal(t, tt.stderr, got.stderr)
			assert.Equal(t, before, registryTip(t, dir))
		})
	}
}

func TestTaskMoveAcceptsInstalledAgentExtension(t *testing.T) {
	t.Parallel()

	dir, _, _ := moveRepo(t)
	putTask(t, dir, task.Record{Status: task.StatusRunning, Stage: task.StageImplementation, Spec: moveSpec})
	putManifest(t, dir, project.ManifestSchemaVersion, "origin", "main")
	manifest := manifestOf(t, dir)
	manifest.Extensions = map[string]project.Extension{
		"a11y-reviewer": {State: "installed", Kind: "agent", Group: "review", Source: "path", Path: "extensions/a11y-reviewer"},
		"perf-reviewer": {State: "draft", Kind: "agent", Group: "review", Source: "path", Path: "extensions/perf-reviewer"},
	}
	data, err := manifest.Encode()
	require.NoError(t, err)
	writeFile(t, dir, project.ManifestPath, string(data))

	installed := runInRepo(t, dir, "task", "move", "W-001", "--as", "orchestrator-1", "--agent", "access-1:a11y-reviewer", "--dry-run")
	draft := runInRepo(t, dir, "task", "move", "W-001", "--as", "orchestrator-1", "--agent", "perf-1:perf-reviewer", "--dry-run")

	require.Equal(t, 0, installed.exit, installed.stderr)
	assert.Contains(t, installed.stdout, "  agents  access-1 a11y-reviewer\n")
	assert.Equal(t, 2, draft.exit)
	assert.Contains(t, draft.stderr, "(invalid_value)")
	assert.Contains(t, draft.stderr, "a11y-reviewer")
}
