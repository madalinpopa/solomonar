package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
)

// registryOf returns a registry holding before as W-001.
func registryOf(before *Record) *Registry {
	registry := NewRegistry()
	registry.Tasks["W-001"] = before
	return registry
}

func moveTask(t *testing.T, registry *Registry, repo *git.Repo, as string, m Move) (MoveResult, error) {
	t.Helper()
	return registry.MoveTask(t.Context(), repo, "W-001", as, m, installedRoles(), clock())
}

func TestMoveTaskStoresMovedRecord(t *testing.T) {
	f := newFactsRepo(t)
	registry := registryOf(moving(StatusRunning, StageSpecification, nil))
	m := Move{Spec: specPath, Stage: "implementation", Agents: []string{"impl-1:implementer"}, Next: "impl-1:Implement AC-1"}

	result, err := moveTask(t, registry, f.repo, "orchestrator-1", m)
	require.NoError(t, err)

	assert.Equal(t, MoveResult{
		From:      Position{Status: StatusRunning, Stage: StageSpecification},
		To:        Position{Status: StatusRunning, Stage: StageImplementation},
		Changed:   []string{"stage", "spec", "active_agents", "next_action"},
		SpecFound: f.worktree,
		Subject:   "state(W-001): running/implementation, stage implementation, spec " + specPath + ", agents impl-1, next impl-1",
	}, result)
	stored := registry.Tasks["W-001"]
	assert.Equal(t, StageImplementation, stored.Stage)
	assert.Equal(t, "2026-10-20T14:58:03Z", stored.UpdatedAt)
}

func TestMoveTaskRejects(t *testing.T) {
	tests := []struct {
		name   string
		before *Record
		as     string
		move   Move
		code   string
	}{
		{name: "done before ownership", before: &Record{Title: "Login", Status: StatusDone, Stage: StageComplete}, as: "orchestrator-9", move: Move{Next: "user:Review"}, code: "done_final"},
		{name: "owner before move rules", before: moving(StatusRunning, StageSpecification, nil), as: "orchestrator-2", move: Move{Stage: "review"}, code: "owner_mismatch"},
		{name: "move rule", before: moving(StatusRunning, StageSpecification, nil), as: "orchestrator-1", move: Move{Stage: "review", Next: "user:Review"}, code: "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := registryOf(tt.before)
			want := *tt.before

			_, err := moveTask(t, registry, nil, tt.as, tt.move)

			requireRejection(t, err, tt.code)
			assert.Equal(t, &want, registry.Tasks["W-001"])
		})
	}
}

func TestMoveTaskUnknownIDIsTaskNotFound(t *testing.T) {
	_, err := NewRegistry().MoveTask(t.Context(), nil, "W-404", "orchestrator-1", Move{Next: "user:Review"}, installedRoles(), clock())

	requireRejection(t, err, "task_not_found")
}

func TestMoveTaskFactRejectionKeepsRecord(t *testing.T) {
	f := newFactsRepo(t)
	before := moving(StatusRunning, StageImplementation, nil)
	registry := registryOf(before)
	want := *before

	_, err := moveTask(t, registry, f.repo, "orchestrator-1", Move{Revision: f.onMain})

	requireRejection(t, err, "revision_not_on_branch")
	assert.Equal(t, &want, registry.Tasks["W-001"])
}

func TestMoveTaskWithoutChangeHasNoSubject(t *testing.T) {
	before := moving(StatusRunning, StageImplementation, nil)
	registry := registryOf(before)
	want := *before

	result, err := moveTask(t, registry, nil, "orchestrator-1", Move{Next: "orchestrator-1:Continue"})
	require.NoError(t, err)

	assert.Empty(t, result.Subject)
	assert.Empty(t, result.Changed)
	assert.Equal(t, result.From, result.To)
	assert.Equal(t, &want, registry.Tasks["W-001"])
}

func TestMoveTaskResumingCancelledWarns(t *testing.T) {
	registry := registryOf(moving(StatusCancelled, StageImplementation, nil))

	result, err := moveTask(t, registry, nil, "orchestrator-1", Move{Status: "running", ResumeCancelled: true})
	require.NoError(t, err)

	assert.Equal(t, []Warning{{
		Code:    "cancelled_resumed",
		Message: "resuming a cancelled task requires the user's explicit instruction (rules §7)",
	}}, result.Warnings)
	assert.Equal(t, "state(W-001): running/implementation, status running", result.Subject)
}

func TestMoveSubject(t *testing.T) {
	full := moving(StatusRunning, StageIntegration, new(2))
	full.ActiveAgents = []Agent{{ID: "sec-1", Role: "security-reviewer"}, {ID: "cor-1", Role: "correctness-reviewer"}}
	full.Revision = "7cd18e45bb55c3be9465c9f893683bc05216f529"
	full.PR = "https://github.com/acme/ledger/pull/42"
	tests := []struct {
		name    string
		record  *Record
		changed []string
		want    string
	}{
		{name: "status and stage", record: full, changed: []string{"status", "stage"}, want: "status running, stage integration"},
		{name: "spec and short revision", record: full, changed: []string{"spec", "revision"}, want: "spec " + specPath + ", revision 7cd18e4"},
		{name: "tier and agents", record: full, changed: []string{"tier", "active_agents"}, want: "tier 2, agents sec-1 cor-1"},
		{name: "agents cleared", record: moving(StatusRunning, StageImplementation, nil), changed: []string{"active_agents"}, want: "agents none"},
		{name: "pr and next", record: full, changed: []string{"pr", "next_action"}, want: "pr https://github.com/acme/ledger/pull/42, next orchestrator-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.record.moveSubject("W-001", tt.changed)

			assert.Equal(t, tt.record.Subject("W-001", tt.want), got)
		})
	}
}
