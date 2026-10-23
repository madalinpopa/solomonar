package task

import (
	"cmp"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

// workspace returns the base and the relative worktree path that leaving setup records for f.
func (f factsRepo) workspace(t *testing.T) (base, worktree string) {
	t.Helper()
	relative, err := filepath.Rel(f.repo.Dir, f.worktree)
	require.NoError(t, err)
	return gittest.Git(t, f.repo.Dir, "merge-base", "main", "feature/W-001"), filepath.ToSlash(relative)
}

// unconfirmed returns a record at stage, past setup, as a migration or a handoff leaves it.
func (f factsRepo) unconfirmed(t *testing.T, stage Stage) *Record {
	t.Helper()
	r := moving(StatusRunning, stage, nil)
	r.Confirmed = false
	r.Base, r.Worktree = f.workspace(t)
	return r
}

func TestLeavingSetupRecordsWorkspace(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	registry := registryOf(moving(StatusReserved, StageSetup, nil))

	result, err := moveTask(t, registry, f.repo, "orchestrator-1", Move{Status: "running", Stage: "specification"})

	require.NoError(t, err)
	base, worktree := f.workspace(t)
	stored := registry.Tasks["W-001"]
	assert.Equal(t, base, stored.Base)
	assert.Equal(t, worktree, stored.Worktree)
	assert.True(t, stored.Confirmed)
	assert.Equal(t, []string{"status", "stage", "confirmed", "base", "worktree"}, result.Changed)
	assert.Equal(t, "state(W-001): running/specification, status running, stage specification, confirmed, base "+
		base[:7]+", worktree "+worktree, result.Subject)
}

func TestStageChangeNeedsConfirmation(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	before := f.unconfirmed(t, StageSpecification)
	registry := registryOf(before)

	_, err := moveTask(t, registry, f.repo, "orchestrator-1", Move{Stage: "implementation", Spec: specPath})
	paused, pauseErr := moveTask(t, registry, f.repo, "orchestrator-1", Move{Status: "paused", Next: "user:Resume"})

	rejection := requireRejection(t, err, "confirmation_required")
	assert.Equal(t, "W-001 is not confirmed by orchestrator-1", rejection.Found)
	assert.Equal(t, "inspect with solomonar task status W-001, then solomonar task confirm W-001 --as orchestrator-1",
		rejection.Fix)
	require.NoError(t, pauseErr, "a status-only move needs no confirmation")
	assert.Equal(t, []string{"status", "next_action"}, paused.Changed)
}

func TestConfirmTaskSetsConfirmed(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	registry := registryOf(f.unconfirmed(t, StageImplementation))

	result, err := registry.ConfirmTask(t.Context(), f.repo, "W-001", "orchestrator-1", clock())
	again, againErr := registry.ConfirmTask(t.Context(), f.repo, "W-001", "orchestrator-1", clock())

	require.NoError(t, err)
	assert.Equal(t, []string{"confirmed"}, result.Changed)
	assert.Equal(t, "state(W-001): running/implementation, confirmed by orchestrator-1", result.Subject)
	assert.True(t, registry.Tasks["W-001"].Confirmed)
	assert.Equal(t, "2026-10-20T14:58:03Z", registry.Tasks["W-001"].UpdatedAt)
	require.NoError(t, againErr)
	assert.Empty(t, again.Subject, "an already confirmed task is unchanged")
	assert.Empty(t, again.Changed)
}

func TestConfirmTaskRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		edit  func(f factsRepo, r *Record)
		as    string
		code  string
		found string
	}{
		{name: "owner", as: "orchestrator-2", code: "owner_mismatch"},
		{name: "planned", edit: func(_ factsRepo, r *Record) { *r = Record{Title: "Login", Status: StatusPlanned} }, code: "not_registered"},
		{name: "done", edit: func(_ factsRepo, r *Record) { *r = Record{Title: "Login", Status: StatusDone} }, code: "done_final"},
		{name: "setup", edit: func(_ factsRepo, r *Record) { r.Stage, r.Status = StageSetup, StatusReserved }, code: "setup_not_complete"},
		{name: "no worktree recorded", edit: func(_ factsRepo, r *Record) { r.Worktree = "" }, code: "worktree_unrecorded"},
		{
			name: "worktree moved", edit: func(_ factsRepo, r *Record) { r.Worktree = "../elsewhere/W-001" },
			code: "worktree_mismatch",
		},
		{name: "no base recorded", edit: func(_ factsRepo, r *Record) { r.Base = "" }, code: "base_missing"},
		{
			name: "base off the branch", edit: func(f factsRepo, r *Record) { r.Base = f.onMain },
			code: "base_not_on_branch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFactsRepo(t)
			before := f.unconfirmed(t, StageImplementation)
			if tt.edit != nil {
				tt.edit(f, before)
			}
			registry := registryOf(before)
			want := *before

			_, err := registry.ConfirmTask(t.Context(), f.repo, "W-001", cmp.Or(tt.as, "orchestrator-1"), clock())

			requireRejection(t, err, tt.code)
			assert.Equal(t, &want, registry.Tasks["W-001"])
		})
	}
}

func TestWorktreeRecordsRelocation(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	before := f.unconfirmed(t, StageImplementation)
	moved := filepath.Join(filepath.Dir(f.worktree), "moved")
	gittest.Git(t, f.repo.Dir, "worktree", "move", f.worktree, moved)
	relative, err := filepath.Rel(f.repo.Dir, moved)
	require.NoError(t, err)
	registry := registryOf(before)

	_, staleErr := registry.ConfirmTask(t.Context(), f.repo, "W-001", "orchestrator-1", clock())
	_, wrongErr := moveTask(t, registry, f.repo, "orchestrator-1", Move{Worktree: "../nowhere"})
	result, err := moveTask(t, registry, f.repo, "orchestrator-1", Move{Worktree: filepath.ToSlash(relative)})

	rejection := requireRejection(t, staleErr, "worktree_mismatch")
	assert.Equal(t, "record the relocation: solomonar task move W-001 --worktree "+filepath.ToSlash(relative), rejection.Fix)
	requireRejection(t, wrongErr, "worktree_mismatch")
	require.NoError(t, err)
	assert.Equal(t, []string{"worktree"}, result.Changed)
	assert.Equal(t, filepath.ToSlash(relative), registry.Tasks["W-001"].Worktree)
}

func TestBaseFillsOnlyAMissingBase(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	before := f.unconfirmed(t, StageImplementation)
	base := before.Base
	before.Base = ""
	registry := registryOf(before)

	_, offBranch := moveTask(t, registry, f.repo, "orchestrator-1", Move{Base: f.onMain})
	result, err := moveTask(t, registry, f.repo, "orchestrator-1", Move{Base: base})
	_, changed := moveTask(t, registry, f.repo, "orchestrator-1", Move{Base: f.onBranch})
	_, short := moveTask(t, registry, f.repo, "orchestrator-1", Move{Base: base[:7]})

	requireRejection(t, offBranch, "base_not_on_branch")
	require.NoError(t, err)
	assert.Equal(t, []string{"base"}, result.Changed)
	assert.Equal(t, base, registry.Tasks["W-001"].Base)
	requireRejection(t, changed, "base_recorded")
	requireRejection(t, short, "invalid_value")
}

func TestWorkspaceFlagsNeedSpecification(t *testing.T) {
	t.Parallel()

	registry := registryOf(moving(StatusReserved, StageSetup, nil))

	_, err := moveTask(t, registry, nil, "orchestrator-1", Move{Worktree: "../w/W-001"})

	requireRejection(t, err, "field_not_allowed")
}
