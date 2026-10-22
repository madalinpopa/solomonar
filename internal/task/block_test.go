package task

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sessionStorage() Block {
	return Block{Kind: "decision", Summary: "Session storage", Ref: "W-001/D-1"}
}

func blocked(stage Stage) *Record {
	r := withBlocker(moving(StatusBlocked, stage, nil))
	r.NextAction = &NextAction{Actor: "user", Action: "Fix CI"}
	return r
}

func TestBlockTaskAddsBlocker(t *testing.T) {
	t.Parallel()

	registry := registryOf(moving(StatusRunning, StageImplementation, nil))

	result, err := registry.BlockTask("W-001", "orchestrator-1", sessionStorage(), clock())
	require.NoError(t, err)

	assert.Equal(t, MoveResult{
		From:    Position{Status: StatusRunning, Stage: StageImplementation},
		To:      Position{Status: StatusRunning, Stage: StageImplementation},
		Changed: []string{"blockers"},
		Subject: "state(W-001): running/implementation, block B-1 decision",
	}, result)
	stored := registry.Tasks["W-001"]
	assert.Equal(t, []Blocker{{ID: "B-1", Kind: BlockerDecision, Summary: "Session storage", Ref: "W-001/D-1"}}, stored.Blockers)
	assert.Equal(t, 1, stored.LastBlocker)
	assert.Equal(t, "2026-10-20T14:58:03Z", stored.UpdatedAt)
}

func TestBlockTaskWithStatusAndNextIsOneWrite(t *testing.T) {
	t.Parallel()

	registry := registryOf(moving(StatusRunning, StageImplementation, nil))
	b := sessionStorage()
	b.Status, b.Next = "blocked", "user:Decide W-001/D-1: session storage"

	result, err := registry.BlockTask("W-001", "orchestrator-1", b, clock())
	require.NoError(t, err)

	assert.Equal(t, Position{Status: StatusBlocked, Stage: StageImplementation}, result.To)
	assert.Equal(t, []string{"status", "blockers", "next_action"}, result.Changed)
	assert.Equal(t, "state(W-001): blocked/implementation, block B-1 decision", result.Subject)
	assert.Equal(t, &NextAction{Actor: "user", Action: "Decide W-001/D-1: session storage"}, registry.Tasks["W-001"].NextAction)
}

func TestBlockTaskAtSetupBlocksReservedTask(t *testing.T) {
	t.Parallel()

	registry := registryOf(moving(StatusReserved, StageSetup, nil))

	result, err := registry.BlockTask("W-001", "orchestrator-1",
		Block{Kind: "environment", Summary: "No database", Status: "blocked"}, clock())
	require.NoError(t, err)

	assert.Equal(t, Position{Status: StatusBlocked, Stage: StageSetup}, result.To)
	assert.Equal(t, "state(W-001): blocked/setup, block B-1 environment", result.Subject)
}

func TestBlockerIDsAreNeverReused(t *testing.T) {
	t.Parallel()

	registry := registryOf(moving(StatusRunning, StageImplementation, nil))
	b := Block{Kind: "check", Summary: "CI is red"}

	_, err := registry.BlockTask("W-001", "orchestrator-1", b, clock())
	require.NoError(t, err)
	_, err = registry.UnblockTask("W-001", "orchestrator-1", "B-1", "", "", clock())
	require.NoError(t, err)
	result, err := registry.BlockTask("W-001", "orchestrator-1", b, clock())
	require.NoError(t, err)

	assert.Equal(t, "state(W-001): running/implementation, block B-2 check", result.Subject)
	assert.Equal(t, []Blocker{{ID: "B-2", Kind: BlockerCheck, Summary: "CI is red"}}, registry.Tasks["W-001"].Blockers)
}

func TestBlockTaskRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before *Record
		as     string
		block  Block
		code   string
	}{
		{name: "decision without ref", before: moving(StatusRunning, StageImplementation, nil), as: "orchestrator-1", block: Block{Kind: "decision", Summary: "Session storage"}, code: "invalid_value"},
		{name: "summary of 81 characters", before: moving(StatusRunning, StageImplementation, nil), as: "orchestrator-1", block: Block{Kind: "check", Summary: strings.Repeat("é", 81)}, code: "invalid_value"},
		{name: "status other than blocked", before: moving(StatusRunning, StageImplementation, nil), as: "orchestrator-1", block: Block{Kind: "check", Summary: "CI is red", Status: "running"}, code: "invalid_value"},
		{name: "next actor not active", before: moving(StatusRunning, StageImplementation, nil), as: "orchestrator-1", block: Block{Kind: "check", Summary: "CI is red", Next: "impl-9:Fix"}, code: "invalid_value"},
		{name: "done task", before: &Record{Title: "Login", Status: StatusDone, Stage: StageComplete}, as: "orchestrator-9", block: sessionStorage(), code: "done_final"},
		{name: "not the owner", before: moving(StatusRunning, StageImplementation, nil), as: "orchestrator-2", block: sessionStorage(), code: "owner_mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := registryOf(tt.before)
			want := *tt.before

			_, err := registry.BlockTask("W-001", tt.as, tt.block, clock())

			requireRejection(t, err, tt.code)
			assert.Equal(t, &want, registry.Tasks["W-001"])
		})
	}
}

func TestBlockTaskStatusFix(t *testing.T) {
	t.Parallel()

	registry := registryOf(moving(StatusRunning, StageImplementation, nil))

	_, err := registry.BlockTask("W-001", "orchestrator-1", Block{Kind: "check", Summary: "CI is red", Status: "paused"}, clock())

	rejection := requireRejection(t, err, "invalid_value")
	assert.Equal(t, "pass --status blocked, or use task move", rejection.Fix)
}

func TestUnblockTaskResumesPriorStatus(t *testing.T) {
	t.Parallel()

	before := moving(StatusRunning, StageImplementation, nil)
	before.ActiveAgents = []Agent{{ID: "impl-1", Role: "implementer"}}
	registry := registryOf(before)
	b := sessionStorage()
	b.Status, b.Next = "blocked", "user:Decide W-001/D-1: session storage"
	_, err := registry.BlockTask("W-001", "orchestrator-1", b, clock())
	require.NoError(t, err)

	result, err := registry.UnblockTask("W-001", "orchestrator-1", "B-1", "running", "impl-1:Apply W-001/D-1, continue AC-2", clock())
	require.NoError(t, err)

	assert.Equal(t, MoveResult{
		From:    Position{Status: StatusBlocked, Stage: StageImplementation},
		To:      Position{Status: StatusRunning, Stage: StageImplementation},
		Changed: []string{"status", "blockers", "next_action"},
		Subject: "state(W-001): running/implementation, unblock B-1",
	}, result)
	stored := registry.Tasks["W-001"]
	assert.Empty(t, stored.Blockers)
	assert.Equal(t, 1, stored.LastBlocker)
}

func TestUnblockTaskKeepsBlockedWhileBlockersRemain(t *testing.T) {
	t.Parallel()

	before := blocked(StageImplementation)
	before.Blockers = append(before.Blockers, Blocker{ID: "B-2", Kind: BlockerDependency, Summary: "W-002 lands first"})
	before.LastBlocker = 2
	registry := registryOf(before)

	result, err := registry.UnblockTask("W-001", "orchestrator-1", "B-1", "", "", clock())
	require.NoError(t, err)

	assert.Equal(t, Position{Status: StatusBlocked, Stage: StageImplementation}, result.To)
	assert.Equal(t, []string{"blockers"}, result.Changed)
	assert.Equal(t, "state(W-001): blocked/implementation, unblock B-1", result.Subject)
	assert.Equal(t, []Blocker{{ID: "B-2", Kind: BlockerDependency, Summary: "W-002 lands first"}}, registry.Tasks["W-001"].Blockers)
	assert.Len(t, before.Blockers, 2)
}

func TestUnblockTaskLastBlockerNeedsPriorStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before *Record
		status string
		fix    string
	}{
		{name: "after setup", before: blocked(StageImplementation), fix: "pass --status running"},
		{name: "at setup", before: blocked(StageSetup), fix: "pass --status reserved"},
		{name: "explicitly blocked", before: blocked(StageImplementation), status: "blocked", fix: "pass --status running"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := registryOf(tt.before)
			want := *tt.before

			_, err := registry.UnblockTask("W-001", "orchestrator-1", "B-1", tt.status, "", clock())

			rejection := requireRejection(t, err, "blockers_required")
			assert.Equal(t, tt.fix, rejection.Fix)
			assert.Equal(t, &want, registry.Tasks["W-001"])
		})
	}
}

func TestUnblockTaskRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		before    *Record
		as        string
		blockerID string
		status    string
		code      string
	}{
		{name: "status other than prior", before: blocked(StageImplementation), as: "orchestrator-1", blockerID: "B-1", status: "paused", code: "transition_not_allowed"},
		{name: "unknown blocker", before: blocked(StageImplementation), as: "orchestrator-1", blockerID: "B-9", status: "running", code: "blocker_not_found"},
		{name: "malformed blocker", before: blocked(StageImplementation), as: "orchestrator-1", blockerID: "X", status: "running", code: "blocker_not_found"},
		{name: "done task", before: &Record{Title: "Login", Status: StatusDone, Stage: StageComplete}, as: "orchestrator-9", blockerID: "B-1", code: "done_final"},
		{name: "not the owner", before: blocked(StageImplementation), as: "orchestrator-2", blockerID: "B-1", status: "running", code: "owner_mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := registryOf(tt.before)
			want := *tt.before

			_, err := registry.UnblockTask("W-001", tt.as, tt.blockerID, tt.status, "", clock())

			requireRejection(t, err, tt.code)
			assert.Equal(t, &want, registry.Tasks["W-001"])
		})
	}
}

func TestUnblockTaskBlockerNotFoundWording(t *testing.T) {
	t.Parallel()

	registry := registryOf(blocked(StageImplementation))

	_, err := registry.UnblockTask("W-001", "orchestrator-1", "B-9", "running", "", clock())

	assert.Equal(t, Rejection{
		Code:  "blocker_not_found",
		Check: "the blocker is open on the task",
		Found: "B-9 is not an open blocker of W-001",
		Fix:   "solomonar task show W-001",
	}, requireRejection(t, err, "blocker_not_found"))
}
