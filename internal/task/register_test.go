package task

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAllocatesNextIDAsReserved(t *testing.T) {
	registry := NewRegistry()
	registry.Tasks["W-001"] = plannedTask("first")

	id, subject, warnings, err := registry.Register(Registration{Title: "Login", Owner: "orchestrator-1"}, clock())

	require.NoError(t, err)
	assert.Equal(t, "W-002", id)
	assert.Equal(t, "state(W-002): reserved/setup, registered by orchestrator-1", subject)
	assert.Empty(t, warnings)
	assert.Equal(t, &Record{
		Title: "Login", Status: StatusReserved, Stage: StageSetup, Owner: "orchestrator-1", Branch: "feature/W-002",
		NextAction: &NextAction{Actor: "orchestrator-1", Action: "Create feature/W-002 and its worktree"},
		UpdatedAt:  "2026-10-20T14:58:03Z",
	}, registry.Tasks["W-002"])
	require.NoError(t, registry.validate())
}

func TestRegisterClaimsPlannedTask(t *testing.T) {
	registry := NewRegistry()
	registry.Tasks["W-003"] = plannedTask("Export")

	id, _, _, err := registry.Register(Registration{ID: "W-003", Title: "Export", Owner: "orchestrator-2"}, clock())

	require.NoError(t, err)
	assert.Equal(t, "W-003", id)
	assert.Equal(t, StatusReserved, registry.Tasks["W-003"].Status)
	assert.Equal(t, "feature/W-003", registry.Tasks["W-003"].Branch)
	assert.Len(t, registry.Tasks, 1)
}

func TestRegisterReferences(t *testing.T) {
	tests := []struct {
		name        string
		reg         Registration
		wantDepends []string
		wantRelated string
	}{
		{name: "absent flags keep planned values", wantDepends: []string{"W-001"}, wantRelated: "W-001"},
		{
			name:        "given flags replace planned values",
			reg:         Registration{DependsOn: []string{"W-002"}, Related: "W-002"},
			wantDepends: []string{"W-002"}, wantRelated: "W-002",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := doneTasks("W-001", "W-002")
			planned := plannedTask("Export")
			planned.DependsOn, planned.RelatedTask = []string{"W-001"}, "W-001"
			registry.Tasks["W-003"] = planned
			tt.reg.ID, tt.reg.Title, tt.reg.Owner = "W-003", "Export", "orchestrator-2"

			_, _, _, err := registry.Register(tt.reg, clock())

			require.NoError(t, err)
			assert.Equal(t, tt.wantDepends, registry.Tasks["W-003"].DependsOn)
			assert.Equal(t, tt.wantRelated, registry.Tasks["W-003"].RelatedTask)
		})
	}
}

func TestRegisterNextAction(t *testing.T) {
	registry := NewRegistry()

	_, _, _, err := registry.Register(
		Registration{Title: "Login", Owner: "orchestrator-1", Next: "user:Approve the scope"}, clock())

	require.NoError(t, err)
	assert.Equal(t, &NextAction{Actor: "user", Action: "Approve the scope"}, registry.Tasks["W-001"].NextAction)
}

func TestRegisterWarnsAboutTitleChangeThenOpenDependencies(t *testing.T) {
	registry := doneTasks("W-001")
	registry.Tasks["W-002"] = plannedTask("second")
	registry.Tasks["W-003"] = plannedTask("Export")

	_, _, warnings, err := registry.Register(Registration{
		ID: "W-003", Title: "Export CSV", Owner: "orchestrator-2", DependsOn: []string{"W-001", "W-002"},
	}, clock())

	require.NoError(t, err)
	assert.Equal(t, []Warning{
		{Code: "title_changed", Message: `title changed from "Export"`},
		{Code: "dependency_not_done", Message: "W-002 is not done; register, then block with task block --kind dependency"},
	}, warnings)
	assert.Equal(t, "Export CSV", registry.Tasks["W-003"].Title)
}

func TestRegisterRejects(t *testing.T) {
	valid := Registration{Title: "Login", Owner: "orchestrator-1"}
	with := func(edit func(*Registration)) Registration {
		reg := valid
		edit(&reg)
		return reg
	}
	tests := []struct {
		name string
		reg  Registration
		code string
	}{
		{name: "malformed ID", reg: with(func(r *Registration) { r.ID = "W-1" }), code: "invalid_value"},
		{name: "unknown ID", reg: with(func(r *Registration) { r.ID = "W-009" }), code: "task_not_found"},
		{name: "claimed ID", reg: with(func(r *Registration) { r.ID = "W-001" }), code: "not_planned"},
		{name: "malformed owner", reg: with(func(r *Registration) { r.Owner = "Orchestrator" }), code: "invalid_value"},
		{name: "owner of a paused task", reg: with(func(r *Registration) { r.Owner = "orchestrator-7" }), code: "owner_in_use"},
		{name: "empty title", reg: with(func(r *Registration) { r.Title = "" }), code: "invalid_value"},
		{name: "unknown dependency", reg: with(func(r *Registration) { r.DependsOn = []string{"W-009"} }), code: "task_not_found"},
		{name: "unknown related", reg: with(func(r *Registration) { r.Related = "W-009" }), code: "task_not_found"},
		{name: "malformed next", reg: with(func(r *Registration) { r.Next = "user" }), code: "invalid_value"},
		{name: "next for another actor", reg: with(func(r *Registration) { r.Next = "impl-1:Implement" }), code: "invalid_value"},
		{name: "next action too long", reg: with(func(r *Registration) { r.Next = "user:" + strings.Repeat("é", 121) }), code: "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			registry.Tasks["W-001"] = moving(StatusPaused, StageSetup, nil)
			registry.Tasks["W-001"].Owner = "orchestrator-7"
			before, err := registry.Encode()
			require.NoError(t, err)

			_, _, _, err = registry.Register(tt.reg, clock())

			assert.Equal(t, tt.code, rejectionOf(t, err).Code)
			after, err := registry.Encode()
			require.NoError(t, err)
			assert.Equal(t, string(before), string(after))
		})
	}
}

func TestRegisterReusesOwnerOfCancelledTask(t *testing.T) {
	registry := doneTasks("W-001")
	registry.Tasks["W-002"] = moving(StatusCancelled, StageSetup, nil)

	_, _, _, err := registry.Register(Registration{Title: "Login", Owner: "orchestrator-1"}, clock())

	require.NoError(t, err)
}

// doneTasks returns a registry whose given tasks are done.
func doneTasks(ids ...string) *Registry {
	registry := NewRegistry()
	for _, id := range ids {
		registry.Tasks[id] = &Record{Title: id, Status: StatusDone, UpdatedAt: updatedAt}
	}
	return registry
}
