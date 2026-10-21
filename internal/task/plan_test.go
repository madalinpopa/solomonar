package task

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanAddsPlannedRecordUnderNextID(t *testing.T) {
	registry := NewRegistry()
	registry.Tasks["W-001"] = plannedTask("first")
	now := time.Date(2026, 10, 20, 14, 58, 3, 500, time.FixedZone("CEST", 2*3600))

	id, subject, err := registry.Plan("Export expenses as CSV", []string{"W-001"}, "W-001", now)

	require.NoError(t, err)
	assert.Equal(t, "W-002", id)
	assert.Equal(t, `state(W-002): planned, plan "Export expenses as CSV"`, subject)
	assert.Equal(t, &Record{
		Title: "Export expenses as CSV", Status: StatusPlanned, UpdatedAt: "2026-10-20T12:58:03Z",
		DependsOn: []string{"W-001"}, RelatedTask: "W-001",
	}, registry.Tasks["W-002"])
}

func TestPlanRejectsBadReferences(t *testing.T) {
	tests := []struct {
		name      string
		dependsOn []string
		related   string
		code      string
	}{
		{name: "unknown dependency", dependsOn: []string{"W-009"}, code: "task_not_found"},
		{name: "unknown related", related: "W-009", code: "task_not_found"},
		{name: "malformed dependency", dependsOn: []string{"W-1"}, code: "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()

			_, _, err := registry.Plan("title", tt.dependsOn, tt.related, time.Time{})

			rejection, ok := errors.AsType[*Rejection](err)
			require.True(t, ok, err)
			assert.Equal(t, tt.code, rejection.Code)
			assert.Empty(t, registry.Tasks)
		})
	}
}

func TestPlanRejectsBlankTitle(t *testing.T) {
	registry := NewRegistry()

	_, _, err := registry.Plan("  ", nil, "", time.Time{})

	rejection, ok := errors.AsType[*Rejection](err)
	require.True(t, ok, err)
	assert.Equal(t, "invalid_value", rejection.Code)
	assert.Empty(t, registry.Tasks)
}

func TestSubjectIncludesStageWhenSet(t *testing.T) {
	record := &Record{Status: StatusRunning, Stage: StageImplementation}

	assert.Equal(t, "state(W-001): running/implementation, status running", record.Subject("W-001", "status running"))
}

func TestIDsSortsNumerically(t *testing.T) {
	registry := NewRegistry()
	for _, id := range []string{"W-1000", "W-002", "W-999"} {
		registry.Tasks[id] = &Record{}
	}

	assert.Equal(t, []string{"W-002", "W-999", "W-1000"}, registry.IDs())
}

func TestRecordEncodeMatchesRegistryIndent(t *testing.T) {
	data, err := plannedTask("one").Encode()

	require.NoError(t, err)
	//nolint:testifylint // the exact bytes are the contract, not only the JSON value
	assert.Equal(t, "{\n  \"title\": \"one\",\n  \"status\": \"planned\",\n  \"updated_at\": \"2026-01-01T00:00:00Z\"\n}\n",
		string(data))
}
