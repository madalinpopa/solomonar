package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextTaskID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ids  []string
		want string
	}{
		{name: "empty registry", want: "W-001"},
		{name: "after one", ids: []string{"W-001"}, want: "W-002"},
		{name: "fills no gaps", ids: []string{"W-001", "W-007"}, want: "W-008"},
		{name: "numeric not string order", ids: []string{"W-999", "W-1000", "W-099"}, want: "W-1001"},
		{name: "past three digits", ids: []string{"W-999"}, want: "W-1000"},
		{name: "padding", ids: []string{"W-041"}, want: "W-042"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := NewRegistry()
			for _, id := range tt.ids {
				registry.Tasks[id] = &Record{}
			}

			assert.Equal(t, tt.want, nextTaskID(registry))
		})
	}
}

func TestIssueBlockerID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		record *Record
		want   string
	}{
		{name: "first blocker", record: &Record{}, want: "B-1"},
		{name: "after an open blocker", record: &Record{LastBlocker: 1, Blockers: []Blocker{{ID: "B-1"}}}, want: "B-2"},
		{name: "never reuses a cleared blocker", record: &Record{LastBlocker: 2}, want: "B-3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			last := tt.record.LastBlocker

			assert.Equal(t, tt.want, tt.record.issueBlockerID())
			assert.Equal(t, last+1, tt.record.LastBlocker)
		})
	}
}
