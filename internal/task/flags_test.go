package task

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAgents(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   []Agent
		clears bool
		code   string
	}{
		{name: "one agent", values: []string{"impl-1:implementer"}, want: []Agent{{ID: "impl-1", Role: "implementer"}}},
		{name: "two agents", values: []string{"sec-1:security-reviewer", "cor-1:correctness-reviewer"}, want: []Agent{{ID: "sec-1", Role: "security-reviewer"}, {ID: "cor-1", Role: "correctness-reviewer"}}},
		{name: "split at first colon", values: []string{"impl-1:role:x"}, want: []Agent{{ID: "impl-1", Role: "role:x"}}},
		{name: "none clears", values: []string{"none"}, clears: true},
		{name: "none mixed", values: []string{"none", "impl-1:implementer"}, code: "invalid_value"},
		{name: "no colon", values: []string{"impl-1"}, code: "invalid_value"},
		{name: "empty id", values: []string{":implementer"}, code: "invalid_value"},
		{name: "empty role", values: []string{"impl-1:"}, code: "invalid_value"},
		{name: "space", values: []string{"impl 1:implementer"}, code: "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents, clears, err := parseAgents(tt.values)

			assertCode(t, err, tt.code)
			assert.Equal(t, tt.want, agents)
			assert.Equal(t, tt.clears, clears)
		})
	}
}

func TestParseAgentsGuidance(t *testing.T) {
	_, _, err := parseAgents([]string{"impl-1"})

	rejection := rejectionOf(t, err)
	assert.Equal(t, "--agent takes <id>:<role>, or none alone", rejection.Check)
	assert.Equal(t, "impl-1", rejection.Found)
	assert.Equal(t, "pass --agent impl-1:implementer, or --agent none to clear the list", rejection.Fix)
}

func TestParseNextAction(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  *NextAction
		code  string
	}{
		{name: "simple", value: "impl-1:Implement AC-1", want: &NextAction{Actor: "impl-1", Action: "Implement AC-1"}},
		{name: "colons in action", value: "user:Decide W-001/D-1: session storage", want: &NextAction{Actor: "user", Action: "Decide W-001/D-1: session storage"}},
		{name: "trims action", value: "user:  Review  ", want: &NextAction{Actor: "user", Action: "Review"}},
		{name: "no colon", value: "user", code: "invalid_value"},
		{name: "empty actor", value: ":Review", code: "invalid_value"},
		{name: "blank action", value: "user:  ", code: "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := parseNextAction(tt.value)

			assertCode(t, err, tt.code)
			assert.Equal(t, tt.want, next)
		})
	}
}

func TestParseTier(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  *int
		code  string
	}{
		{name: "zero", value: "0", want: new(0)},
		{name: "two", value: "2", want: new(2)},
		{name: "three", value: "3", code: "invalid_value"},
		{name: "signed", value: "+1", code: "invalid_value"},
		{name: "word", value: "high", code: "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier, err := parseTier(tt.value)

			assertCode(t, err, tt.code)
			assert.Equal(t, tt.want, tier)
		})
	}
}

func TestCheckRevision(t *testing.T) {
	tests := []struct {
		name  string
		value string
		code  string
	}{
		{name: "sha-1", value: sha},
		{name: "sha-256", value: sha + sha[:24]},
		{name: "short", value: "0123456", code: "revision_not_full_sha"},
		{name: "uppercase", value: strings.ToUpper(sha), code: "revision_not_full_sha"},
		{name: "ref name", value: "HEAD", code: "revision_not_full_sha"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCode(t, checkRevision(tt.value), tt.code)
		})
	}
}

func TestCheckRevisionGuidance(t *testing.T) {
	rejection := rejectionOf(t, checkRevision("0123456"))

	assert.Equal(t, "--revision must be a full 40-character SHA", rejection.Check)
	assert.Equal(t, "0123456", rejection.Found)
	assert.Equal(t, "git rev-parse 0123456", rejection.Fix)
}

func TestCheckPR(t *testing.T) {
	tests := []struct {
		name  string
		value string
		code  string
	}{
		{name: "number", value: "42"},
		{name: "https url", value: "https://github.com/o/r/pull/42"},
		{name: "zero", value: "0", code: "invalid_value"},
		{name: "negative", value: "-1", code: "invalid_value"},
		{name: "http url", value: "http://github.com/o/r/pull/42", code: "invalid_value"},
		{name: "bare host", value: "https://github.com", code: "invalid_value"},
		{name: "root path", value: "https://github.com/", code: "invalid_value"},
		{name: "no host", value: "https:///o/r/pull/42", code: "invalid_value"},
		{name: "empty", value: "", code: "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCode(t, checkPR(tt.value), tt.code)
		})
	}
}

func TestNewBlocker(t *testing.T) {
	tests := []struct {
		name               string
		kind, summary, ref string
		want               Blocker
		code               string
	}{
		{name: "decision", kind: "decision", summary: "Session storage", ref: "W-001/D-1", want: Blocker{Kind: BlockerDecision, Summary: "Session storage", Ref: "W-001/D-1"}},
		{name: "check without ref", kind: "check", summary: "CI red", want: Blocker{Kind: BlockerCheck, Summary: "CI red"}},
		{name: "dependency with ref", kind: "dependency", summary: "Needs W-002", ref: "W-002", want: Blocker{Kind: BlockerDependency, Summary: "Needs W-002", Ref: "W-002"}},
		{name: "80 runes", kind: "environment", summary: strings.Repeat("é", 80), want: Blocker{Kind: BlockerEnvironment, Summary: strings.Repeat("é", 80)}},
		{name: "81 runes", kind: "environment", summary: strings.Repeat("é", 81), code: "invalid_value"},
		{name: "blank summary", kind: "check", summary: "  ", code: "invalid_value"},
		{name: "unknown kind", kind: "mood", summary: "Tired", code: "invalid_value"},
		{name: "decision without ref", kind: "decision", summary: "Session storage", code: "invalid_value"},
		{name: "decision ref without decision", kind: "decision", summary: "Session storage", ref: "W-001", code: "invalid_value"},
		{name: "decision ref with bad task", kind: "decision", summary: "Session storage", ref: "W-1/D-1", code: "invalid_value"},
		{name: "decision ref with bad number", kind: "decision", summary: "Session storage", ref: "W-001/D-x", code: "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocker, err := newBlocker(tt.kind, tt.summary, tt.ref)

			assertCode(t, err, tt.code)
			assert.Equal(t, tt.want, blocker)
		})
	}
}

func TestNewBlockerGuidance(t *testing.T) {
	_, err := newBlocker("decision", "Session storage", "")

	rejection := rejectionOf(t, err)

	assert.Equal(t, "--kind decision needs --ref <task ID>/D-<n>", rejection.Check)
	assert.Equal(t, "pass --ref W-001/D-1", rejection.Fix)
}

func TestCheckTaskID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		code  string
	}{
		{name: "three digits", value: "W-001"},
		{name: "four digits", value: "W-1000"},
		{name: "two digits", value: "W-01", code: "invalid_value"},
		{name: "lowercase", value: "w-001", code: "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCode(t, checkTaskID(tt.value), tt.code)
		})
	}
}

func TestCheckTaskIDGuidance(t *testing.T) {
	rejection := rejectionOf(t, checkTaskID("W-01"))

	assert.Equal(t, "task IDs look like W-001", rejection.Check)
	assert.Equal(t, "W-01", rejection.Found)
}
