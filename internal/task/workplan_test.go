package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorkPlan(t *testing.T) {
	t.Parallel()

	content := "# W-001: Login\nStatus: Approved\nType: feature\n\n## Outcome and scope\n\n- not a step\n\n" +
		"## Implementation plan\n<!-- One step per checkpoint commit. -->\n\n" +
		"- S-1: feat(W-001): add the session store\n  Store sessions server-side.\n" +
		"- S-2: feat(W-001): add remember me\n  Outcome: dropped — the user removed AC-4\n" +
		"- S-3: test(W-001): cover expiry\n  Outcome: verified - covered by S-1's tests\n" +
		"- S-10: docs(auth): note the cookie name\n\n## Open questions\n\n- Q-1 · none\n"

	plan, err := ParseWorkPlan(content)

	require.NoError(t, err)
	assert.Equal(t, WorkPlan{Approval: ApprovalApproved, Steps: []Step{
		{ID: "S-1", Subject: "feat(W-001): add the session store"},
		{ID: "S-2", Subject: "feat(W-001): add remember me", Outcome: OutcomeDropped},
		{ID: "S-3", Subject: "test(W-001): cover expiry", Outcome: OutcomeVerified},
		{ID: "S-10", Subject: "docs(auth): note the cookie name"},
	}}, plan)
}

func TestParseWorkPlanWithoutStatusOrPlan(t *testing.T) {
	t.Parallel()

	plan, err := ParseWorkPlan("# W-001: Login\nType: feature\n")

	require.NoError(t, err)
	assert.Equal(t, WorkPlan{}, plan)
}

func TestParseWorkPlanRejects(t *testing.T) {
	t.Parallel()

	tests := []struct{ name, content, want string }{
		{name: "unknown status", content: "# W-001: Login\nStatus: Aprobat\n", want: `status "Aprobat"`},
		{name: "malformed step", content: "# T\n\n## Implementation plan\n- S1: feat: x\n", want: "is not - S-<n>"},
		{name: "empty subject", content: "# T\n\n## Implementation plan\n- S-1: \n", want: "is not - S-<n>"},
		{name: "zero-padded id", content: "# T\n\n## Implementation plan\n- S-01: feat: x\n", want: "is not - S-<n>"},
		{name: "duplicate id", content: "# T\n\n## Implementation plan\n- S-1: a\n- S-1: b\n", want: "S-1 is listed twice"},
		{name: "outcome without reason", content: "# T\n\n## Implementation plan\n- S-1: a\n  Outcome: dropped\n", want: "step S-1"},
		{name: "unknown outcome", content: "# T\n\n## Implementation plan\n- S-1: a\n  Outcome: done — yes\n", want: "step S-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseWorkPlan(tt.content)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}
