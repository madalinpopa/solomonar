package task

import (
	"cmp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandoffTaskRetargetsOwnersNextAction(t *testing.T) {
	t.Parallel()

	before := withAgent(withBlocker(moving(StatusBlocked, StageSpecification, nil)))
	registry := registryOf(before)

	result, err := registry.HandoffTask("W-001", "orchestrator-3", "", clock())
	require.NoError(t, err)

	position := Position{Status: StatusBlocked, Stage: StageSpecification}
	assert.Equal(t, MoveResult{
		From:    position,
		To:      position,
		Changed: []string{"owner", "next_action"},
		Subject: "state(W-001): blocked/specification, handoff orchestrator-1 -> orchestrator-3",
	}, result)
	stored := registry.Tasks["W-001"]
	assert.Equal(t, "orchestrator-3", stored.Owner)
	assert.Equal(t, &NextAction{Actor: "orchestrator-3", Action: "Continue"}, stored.NextAction)
	assert.Equal(t, before.Blockers, stored.Blockers)
	assert.Equal(t, before.ActiveAgents, stored.ActiveAgents)
	assert.Equal(t, "2026-10-20T14:58:03Z", stored.UpdatedAt)
	assert.Equal(t, "orchestrator-1", before.NextAction.Actor)
}

func TestHandoffTaskKeepsNextActionOfAnotherActor(t *testing.T) {
	t.Parallel()

	registry := registryOf(blocked(StageImplementation))

	result, err := registry.HandoffTask("W-001", "orchestrator-3", "", clock())
	require.NoError(t, err)

	assert.Equal(t, []string{"owner"}, result.Changed)
	assert.Equal(t, &NextAction{Actor: "user", Action: "Fix CI"}, registry.Tasks["W-001"].NextAction)
}

func TestHandoffTaskSetsNextForNewOwner(t *testing.T) {
	t.Parallel()

	registry := registryOf(moving(StatusRunning, StageSpecification, nil))

	result, err := registry.HandoffTask("W-001", "orchestrator-3", "orchestrator-3:Resume at specification", clock())
	require.NoError(t, err)

	assert.Equal(t, []string{"owner", "next_action"}, result.Changed)
	assert.Equal(t, &NextAction{Actor: "orchestrator-3", Action: "Resume at specification"},
		registry.Tasks["W-001"].NextAction)
}

func TestHandoffTaskAcceptsCancelledTask(t *testing.T) {
	t.Parallel()

	registry := registryOf(moving(StatusCancelled, StageImplementation, nil))

	result, err := registry.HandoffTask("W-001", "orchestrator-3", "", clock())
	require.NoError(t, err)

	assert.Equal(t, "state(W-001): cancelled/implementation, handoff orchestrator-1 -> orchestrator-3", result.Subject)
}

func TestHandoffTaskRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before *Record
		id     string
		owner  string
		next   string
		want   Rejection
	}{
		{
			name: "unknown task", before: moving(StatusRunning, StageSetup, nil), id: "W-009", owner: "orchestrator-3",
			want: Rejection{Code: "task_not_found"},
		},
		{
			name: "done task", before: &Record{Title: "Login", Status: StatusDone, Stage: StageComplete}, owner: "orchestrator-3",
			want: *doneFinal(StatusDone),
		},
		{
			name: "planned task", before: &Record{Title: "Login", Status: StatusPlanned}, owner: "orchestrator-3",
			want: Rejection{
				Code: "not_registered", Check: "the task is registered", Found: "W-001 is planned",
				Fix: "solomonar task register W-001 --title <title> --owner orchestrator-3",
			},
		},
		{
			name: "invalid owner", before: moving(StatusRunning, StageSetup, nil), owner: "Orchestrator",
			want: Rejection{
				Code: "invalid_value", Check: "--owner takes an agent ID like [a-z]+-[0-9]+", Found: "Orchestrator",
				Fix: "pass --owner orchestrator-1",
			},
		},
		{
			name: "same owner", before: moving(StatusRunning, StageSetup, nil), owner: "orchestrator-1",
			want: Rejection{
				Code: "handoff_same_owner", Check: "the new owner differs from the current owner",
				Found: "W-001 is owned by orchestrator-1",
			},
		},
		{
			name: "owner in use", before: moving(StatusRunning, StageSetup, nil), owner: "orchestrator-2",
			want: Rejection{
				Code: "owner_in_use", Check: "an orchestrator owns exactly one active task",
				Found: "orchestrator-2 owns W-002 (running)", Fix: "choose a new owner ID",
			},
		},
		{
			name: "next names the previous owner", before: moving(StatusRunning, StageSetup, nil), owner: "orchestrator-3",
			next: "orchestrator-1:Continue",
			want: Rejection{
				Code: "invalid_value", Check: "--next actor is user, the owner, or an active agent",
				Found: "orchestrator-1:Continue", Fix: "allowed actors: user, orchestrator-3",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := registryOf(tt.before)
			other := moving(StatusRunning, StageImplementation, nil)
			other.Owner = "orchestrator-2"
			registry.Tasks["W-002"] = other
			want := *tt.before

			_, err := registry.HandoffTask(cmp.Or(tt.id, "W-001"), tt.owner, tt.next, clock())

			rejection := requireRejection(t, err, tt.want.Code)
			if tt.want.Check != "" {
				assert.Equal(t, tt.want, rejection)
			}
			assert.Equal(t, &want, registry.Tasks["W-001"])
		})
	}
}
