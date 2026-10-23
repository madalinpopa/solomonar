package task

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	specPath  = "docs/work/W-001-login.md"
	otherSHA  = "89abcdef0123456789abcdef0123456789abcdef"
	updatedAt = "2026-10-20T14:00:00Z"
)

// moving returns a valid active record whose required fields match its stage.
func moving(status Status, stage Stage, tier *int) *Record {
	r := record(status, stage, tier)
	r.Title, r.Owner, r.Branch, r.UpdatedAt = "Login", "orchestrator-1", "feature/W-001", updatedAt
	r.NextAction = &NextAction{Actor: "orchestrator-1", Action: "Continue"}
	r.Confirmed = stage != StageSetup
	if stage.atOrAfter(StageImplementation) {
		r.Spec = specPath
	}
	if stage.atOrAfter(StageSpecializedReview) {
		r.Revision = sha
	}
	return r
}

// installedRoles lists orchestrator too, as the project manifest does; --agent must still refuse it.
func installedRoles() []string {
	return []string{"correctness-reviewer", "implementer", "orchestrator", "security-reviewer"}
}

// clock is 2026-10-20T14:58:03Z with sub-second noise in a non-UTC zone.
func clock() time.Time {
	return time.Date(2026, 10, 20, 16, 58, 3, 999, time.FixedZone("CEST", 2*60*60))
}

func withAgent(r *Record) *Record {
	r.ActiveAgents = []Agent{{ID: "impl-1", Role: "implementer"}}
	return r
}

func withBlocker(r *Record) *Record {
	r.Blockers = []Blocker{{ID: "B-1", Kind: BlockerCheck, Summary: "CI is red"}}
	r.LastBlocker = 1
	return r
}

func TestRecordMoveUC13(t *testing.T) {
	t.Parallel()

	r := moving(StatusReserved, StageSetup, nil)
	steps := []struct {
		move    Move
		changed []string
	}{
		{move: Move{Status: "running", Stage: "specification", Next: "orchestrator-1:Write the spec"}, changed: []string{"status", "stage", "next_action"}},
		{move: Move{Spec: specPath, Stage: "implementation", Agents: []string{"impl-1:implementer"}, Next: "impl-1:Implement AC-1"}, changed: []string{"stage", "spec", "active_agents", "next_action"}},
		{move: Move{Revision: sha, Tier: "2", Stage: "specialized_review", Agents: []string{"sec-1:security-reviewer", "cor-1:correctness-reviewer"}, Next: "orchestrator-1:Collect reviews"}, changed: []string{"stage", "revision", "tier", "active_agents", "next_action"}},
	}
	for _, step := range steps {
		changed, err := r.Move(step.move, installedRoles(), clock())

		require.NoError(t, err)
		assert.Equal(t, step.changed, changed)
	}
	assert.Equal(t, StageSpecializedReview, r.Stage)
	assert.Equal(t, new(2), r.Tier)
	assert.Equal(t, []Agent{{ID: "sec-1", Role: "security-reviewer"}, {ID: "cor-1", Role: "correctness-reviewer"}}, r.ActiveAgents)
}

func TestRecordMove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		before  *Record
		move    Move
		code    string
		changed []string
	}{
		{name: "unknown status", before: moving(StatusRunning, StageSpecification, nil), move: Move{Status: "active"}, code: "invalid_value"},
		{name: "unknown stage", before: moving(StatusRunning, StageSpecification, nil), move: Move{Stage: "review"}, code: "invalid_value"},
		{name: "status done", before: moving(StatusRunning, StageIntegration, new(1)), move: Move{Status: "done"}, code: "transition_not_allowed"},
		{name: "stage complete", before: moving(StatusRunning, StageIntegration, new(1)), move: Move{Stage: "complete"}, code: "stage_not_allowed"},
		{name: "invalid tier", before: moving(StatusRunning, StageImplementation, nil), move: Move{Revision: sha, Tier: "3"}, code: "invalid_value"},
		{name: "invalid agent", before: moving(StatusRunning, StageImplementation, nil), move: Move{Agents: []string{"impl-1"}}, code: "invalid_value"},
		{name: "short revision", before: moving(StatusRunning, StageImplementation, nil), move: Move{Revision: "7cd18e4", Tier: "1", Stage: "specialized_review"}, code: "revision_not_full_sha"},
		{name: "tier skips stage", before: moving(StatusRunning, StageSpecializedReview, new(1)), move: Move{Stage: "adversarial_review"}, code: "stage_not_allowed"},
		{name: "revision at implementation", before: moving(StatusRunning, StageImplementation, nil), move: Move{Revision: sha}, changed: []string{"revision"}},
		{name: "revision into integration", before: moving(StatusRunning, StageSpecializedReview, new(1)), move: Move{Revision: otherSHA, Stage: "integration"}, changed: []string{"stage", "revision"}},
		{name: "revision at specialized review", before: moving(StatusRunning, StageSpecializedReview, new(1)), move: Move{Revision: otherSHA}, code: "field_not_allowed"},
		{name: "revision at specification", before: moving(StatusRunning, StageSpecification, nil), move: Move{Revision: sha}, code: "field_not_allowed"},
		{name: "tier with revision", before: moving(StatusRunning, StageImplementation, nil), move: Move{Revision: sha, Tier: "0"}, changed: []string{"revision", "tier"}},
		{name: "tier re-triage", before: moving(StatusRunning, StageSpecializedReview, new(1)), move: Move{Tier: "2"}, changed: []string{"tier"}},
		{name: "tier without recorded tier", before: moving(StatusRunning, StageImplementation, nil), move: Move{Tier: "1"}, code: "field_not_allowed"},
		{name: "spec at specification", before: moving(StatusRunning, StageSpecification, nil), move: Move{Spec: specPath}, changed: []string{"spec"}},
		{name: "spec into specification", before: moving(StatusReserved, StageSetup, nil), move: Move{Status: "running", Stage: "specification", Spec: specPath}, changed: []string{"status", "stage", "spec"}},
		{name: "spec at setup", before: moving(StatusReserved, StageSetup, nil), move: Move{Spec: specPath}, code: "field_not_allowed"},
		{name: "pr at integration", before: moving(StatusRunning, StageIntegration, new(1)), move: Move{PR: "42"}, changed: []string{"pr"}},
		{name: "pr into integration", before: moving(StatusRunning, StageSpecializedReview, new(1)), move: Move{Stage: "integration", PR: "42"}, changed: []string{"stage", "pr"}},
		{name: "pr at implementation", before: moving(StatusRunning, StageImplementation, nil), move: Move{PR: "42"}, code: "field_not_allowed"},
		{name: "agent and next while blocked", before: withBlocker(moving(StatusBlocked, StageImplementation, nil)), move: Move{Agents: []string{"impl-1:implementer"}, Next: "user:Decide"}, changed: []string{"active_agents", "next_action"}},
		{name: "next on done", before: &Record{Status: StatusDone, Stage: StageComplete}, move: Move{Next: "user:Decide"}, code: "done_final"},
		{name: "agent none clears", before: &Record{Status: StatusRunning, Stage: StageSpecification, ActiveAgents: []Agent{{ID: "impl-1", Role: "implementer"}}}, move: Move{Agents: []string{"none"}}, changed: []string{"active_agents"}},
		{name: "equal values", before: moving(StatusRunning, StageImplementation, nil), move: Move{Spec: specPath, Next: "orchestrator-1:Continue"}, changed: []string{}},
		{name: "spec required", before: moving(StatusRunning, StageSpecification, nil), move: Move{Stage: "implementation"}, code: "spec_required"},
		{name: "revision required", before: moving(StatusRunning, StageImplementation, new(0)), move: Move{Stage: "integration"}, code: "revision_required"},
		{name: "tier required before revision required", before: moving(StatusRunning, StageImplementation, nil), move: Move{Stage: "specialized_review"}, code: "tier_required"},
		{name: "pause needs next", before: moving(StatusRunning, StageSpecification, nil), move: Move{Status: "paused"}, code: "next_required"},
		{name: "cancel needs next", before: moving(StatusRunning, StageSpecification, nil), move: Move{Status: "cancelled"}, code: "next_required"},
		{name: "pause with next", before: moving(StatusRunning, StageSpecification, nil), move: Move{Status: "paused", Next: "user:Resume later"}, changed: []string{"status", "next_action"}},
		{name: "block needs blocker", before: moving(StatusRunning, StageSpecification, nil), move: Move{Status: "blocked"}, code: "blockers_required"},
		{name: "uppercase agent id", before: moving(StatusRunning, StageImplementation, nil), move: Move{Agents: []string{"Impl-1:implementer"}}, code: "invalid_value"},
		{name: "unknown agent role", before: moving(StatusRunning, StageImplementation, nil), move: Move{Agents: []string{"impl-1:designer"}}, code: "invalid_value"},
		{name: "orchestrator agent role", before: moving(StatusRunning, StageImplementation, nil), move: Move{Agents: []string{"orc-2:orchestrator"}}, code: "invalid_value"},
		{name: "next for user", before: moving(StatusRunning, StageImplementation, nil), move: Move{Next: "user:Approve"}, changed: []string{"next_action"}},
		{name: "next for owner", before: moving(StatusRunning, StageImplementation, nil), move: Move{Next: "orchestrator-1:Review"}, changed: []string{"next_action"}},
		{name: "next for active agent", before: withAgent(moving(StatusRunning, StageImplementation, nil)), move: Move{Next: "impl-1:Fix AC-2"}, changed: []string{"next_action"}},
		{name: "next for agent added in the same move", before: moving(StatusRunning, StageImplementation, nil), move: Move{Agents: []string{"impl-1:implementer"}, Next: "impl-1:Implement AC-1"}, changed: []string{"active_agents", "next_action"}},
		{name: "next for cleared agent", before: withAgent(moving(StatusRunning, StageImplementation, nil)), move: Move{Agents: []string{"none"}, Next: "impl-1:Fix AC-2"}, code: "invalid_value"},
		{name: "next for stranger", before: moving(StatusRunning, StageImplementation, nil), move: Move{Next: "impl-9:Implement"}, code: "invalid_value"},
		{name: "next action too long", before: moving(StatusRunning, StageImplementation, nil), move: Move{Next: "user:" + strings.Repeat("é", 121)}, code: "invalid_value"},
		{name: "spec nested", before: moving(StatusRunning, StageSpecification, nil), move: Move{Spec: "docs/work/W-001-login-v2.md"}, changed: []string{"spec"}},
		{name: "spec absolute", before: moving(StatusRunning, StageSpecification, nil), move: Move{Spec: "/docs/work/W-001-login.md"}, code: "invalid_value"},
		{name: "spec parent", before: moving(StatusRunning, StageSpecification, nil), move: Move{Spec: "../W-001-login.md"}, code: "invalid_value"},
		{name: "spec escaping through a directory", before: moving(StatusRunning, StageSpecification, nil), move: Move{Spec: "a/../../b"}, code: "invalid_value"},
		{name: "spec with inner parent", before: moving(StatusRunning, StageSpecification, nil), move: Move{Spec: "docs/../W-001-login.md"}, code: "invalid_value"},
		{name: "spec dot", before: moving(StatusRunning, StageSpecification, nil), move: Move{Spec: "."}, code: "invalid_value"},
		{name: "spec unclean", before: moving(StatusRunning, StageSpecification, nil), move: Move{Spec: "./docs//W-001-login.md"}, code: "invalid_value"},
		{name: "block with blocker", before: withBlocker(moving(StatusRunning, StageSpecification, nil)), move: Move{Status: "blocked"}, changed: []string{"status"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			changed, err := tt.before.Move(tt.move, installedRoles(), clock())

			assertCode(t, err, tt.code)
			assert.Equal(t, tt.changed, changed)
		})
	}
}

func TestRecordMoveRejectionTexts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before *Record
		move   Move
		check  string
		fix    string
	}{
		{name: "tier required", before: moving(StatusRunning, StageImplementation, nil), move: Move{Stage: "specialized_review"}, check: "a stage after implementation needs a recorded tier", fix: "triage the candidate and pass --revision <full-sha> --tier <0|1|2> with the stage move"},
		{name: "tier skips stage", before: moving(StatusRunning, StageSpecializedReview, new(1)), move: Move{Stage: "adversarial_review"}, check: "tier 1 skips adversarial_review (rules section 7)", fix: "--stage integration, or record tier 2 if the triage changed: --tier 2"},
		{name: "status done", before: moving(StatusReserved, StageSetup, nil), move: Move{Status: "done"}, check: "status change must follow I section 8.1", fix: "use task done"},
		{name: "next for stranger", before: moving(StatusRunning, StageImplementation, nil), move: Move{Next: "impl-9:Implement"}, check: "--next actor is user, the owner, or an active agent", fix: "allowed actors: user, orchestrator-1"},
		{name: "next for stranger with agents", before: withAgent(moving(StatusRunning, StageImplementation, nil)), move: Move{Next: "impl-9:Implement"}, check: "--next actor is user, the owner, or an active agent", fix: "allowed actors: user, orchestrator-1, impl-1"},
		{name: "next action too long", before: moving(StatusRunning, StageImplementation, nil), move: Move{Next: "user:" + strings.Repeat("é", 121)}, check: "--next action takes 1 to 120 characters", fix: "shorten the action; put details in the spec or a decision"},
		{name: "spec outside the worktree", before: moving(StatusRunning, StageSpecification, nil), move: Move{Spec: "../W-001-login.md"}, check: "--spec takes a clean path relative to the worktree, without ..", fix: "pass --spec docs/work/<task ID>-<slug>.md"},
		{name: "field not allowed", before: moving(StatusRunning, StageImplementation, nil), move: Move{PR: "42"}, check: "--pr is accepted in integration (I section 8.2)", fix: "pass --pr at integration or with --stage integration"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.before.Move(tt.move, installedRoles(), clock())

			rejection := rejectionOf(t, err)
			assert.Equal(t, tt.check, rejection.Check)
			assert.Equal(t, tt.fix, rejection.Fix)
		})
	}
}

func TestRecordMoveKeepsRecordOnRejection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		move Move
	}{
		{name: "late rejection", move: Move{Revision: sha, Tier: "1", Agents: []string{"impl-1:implementer"}, Stage: "integration"}},
		{name: "orchestrator agent", move: Move{Agents: []string{"orc-2:orchestrator"}, Next: "user:Review"}},
		{name: "next for stranger", move: Move{Agents: []string{"impl-1:implementer"}, Next: "impl-9:Implement"}},
		{name: "next action too long", move: Move{Spec: specPath, Next: "user:" + strings.Repeat("é", 121)}},
		{name: "spec absolute", move: Move{Spec: "/etc/passwd", Next: "user:Review"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := moving(StatusRunning, StageImplementation, nil)
			want := *r

			_, err := r.Move(tt.move, installedRoles(), clock())

			require.Error(t, err)
			assert.Equal(t, &want, r)
		})
	}
}

func TestRecordCheckOwner(t *testing.T) {
	t.Parallel()

	r := moving(StatusRunning, StageSpecification, nil)

	require.NoError(t, r.CheckOwner("W-001", "orchestrator-1"))

	rejection := rejectionOf(t, r.CheckOwner("W-001", "orchestrator-2"))
	assert.Equal(t, "owner_mismatch", rejection.Code)
	assert.Equal(t, "only the owner writes a task's record", rejection.Check)
	assert.Equal(t, "owner is orchestrator-1; acting as orchestrator-2", rejection.Found)
	assert.Equal(t, "act as orchestrator-1, or ask the user for an explicit handoff: "+
		"solomonar task handoff W-001 --owner orchestrator-2", rejection.Fix)
}

func TestRecordMoveUpdatedAt(t *testing.T) {
	t.Parallel()

	r := moving(StatusRunning, StageImplementation, nil)
	_, err := r.Move(Move{Revision: sha}, installedRoles(), clock())
	require.NoError(t, err)
	assert.Equal(t, "2026-10-20T14:58:03Z", r.UpdatedAt)

	unchanged := moving(StatusRunning, StageImplementation, nil)
	_, err = unchanged.Move(Move{Spec: specPath}, installedRoles(), clock())
	require.NoError(t, err)
	assert.Equal(t, updatedAt, unchanged.UpdatedAt)
}
