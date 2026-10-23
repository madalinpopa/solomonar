package task

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

// checkpoint commits a change in the worktree with subject and, unless step is empty, a Step trailer.
func (f factsRepo) checkpoint(t *testing.T, subject, step string) string {
	t.Helper()
	name := filepath.Join(f.worktree, "change.txt")
	previous, _ := os.ReadFile(name)
	require.NoError(t, os.WriteFile(name, append(previous, subject+"\n"...), 0o644))
	gittest.Git(t, f.worktree, "add", ".")
	args := []string{"commit", "-m", subject}
	if step != "" {
		args = append(args, "-m", "Progress: done.\n\nStep: "+step)
	}
	gittest.Git(t, f.worktree, args...)
	return head(t, f.worktree)
}

// implementing returns a confirmed record at implementation whose candidate is the branch tip.
func (f factsRepo) implementing(t *testing.T) *Record {
	t.Helper()
	r := withSpec(moving(StatusRunning, StageImplementation, nil))
	r.Base, r.Worktree = f.workspace(t)
	r.Revision = head(t, f.worktree)
	return r
}

func TestInspectPlanReconcilesCheckpoints(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	s1 := f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")
	untagged := f.checkpoint(t, "feat(W-001): add the login form", "")
	foreign := f.checkpoint(t, "feat(W-001): add the login form", "W-002/S-2")
	unknown := f.checkpoint(t, "feat(W-001): add the login form", "W-001/S-9")
	mismatch := f.checkpoint(t, "feat(W-001): add a login form", "W-001/S-2")
	gittest.Git(t, f.worktree, "commit", "--allow-empty", "-m", "feat(W-001): add the login form", "-m", "Step: W-001/S-2")
	empty := head(t, f.worktree)
	duplicate := f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")
	gittest.Git(t, f.worktree, "merge", "--no-ff", "-m", "Merge main", "main")
	f.checkpoint(t, "chore: tidy", "")

	facts, err := InspectPlan(t.Context(), f.repo, "W-001", f.implementing(t), head(t, f.worktree))

	require.NoError(t, err)
	assert.Equal(t, PlanApproved, facts.Approval)
	assert.Equal(t, []StepFacts{
		{ID: "S-1", Subject: "feat(W-001): add the session store", State: StepCommitted, Commit: &s1},
		{ID: "S-2", Subject: "feat(W-001): add the login form", State: StepRemaining},
	}, facts.Steps)
	assert.Equal(t, 2, facts.OtherCommits, "the spec commit and chore: tidy; the merge is skipped")
	assert.Equal(t, []Discrepancy{
		{Code: "step_untagged", Commit: untagged, Message: untagged[:8] + " matches the subject of S-2 but has no Step trailer"},
		{Code: "step_foreign", Commit: foreign, Message: foreign[:8] + " claims W-002/S-2, a step of another task"},
		{Code: "step_unknown", Commit: unknown, Message: unknown[:8] + " claims W-001/S-9, which the plan does not list"},
		{
			Code: "step_subject_mismatch", Commit: mismatch,
			Message: mismatch[:8] + " claims S-2 but its subject differs from the planned one",
		},
		{Code: "step_empty", Commit: empty, Message: empty[:8] + " claims S-2 but changes nothing"},
		{
			Code: "step_duplicate", Commit: duplicate,
			Message: duplicate[:8] + " claims S-1, already checkpointed by " + s1[:8],
		},
	}, facts.Discrepancies)
	assert.Equal(t, []string{"S-2"}, facts.Remaining())
}

func TestInspectPlanReportsApproval(t *testing.T) {
	t.Parallel()

	tests := []struct{ name, spec, approval, problem string }{
		{name: "in review", spec: "# W-001: Login\nStatus: In review\n", approval: PlanInReview},
		{name: "no status", spec: "# W-001: Login\n\n## Implementation plan\n- S-1: feat: x\n", approval: PlanMissing},
		{name: "approved without steps", spec: "# W-001: Login\nStatus: Approved\n", approval: PlanMissing},
		{name: "invalid", spec: "# W-001: Login\nStatus: Aprobat\n", approval: PlanInvalid, problem: `status "Aprobat" is not "In review" or "Approved"`},
		{name: "deleted", approval: PlanMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFactsRepo(t)
			if tt.spec == "" {
				gittest.Git(t, f.worktree, "rm", "-q", specPath)
			} else {
				require.NoError(t, os.WriteFile(filepath.Join(f.worktree, filepath.FromSlash(specPath)), []byte(tt.spec), 0o644))
				gittest.Git(t, f.worktree, "add", ".")
			}
			gittest.Git(t, f.worktree, "commit", "-m", "spec(W-001): revise")

			facts, err := InspectPlan(t.Context(), f.repo, "W-001", f.implementing(t), head(t, f.worktree))

			require.NoError(t, err)
			assert.Equal(t, tt.approval, facts.Approval)
			assert.Equal(t, tt.problem, facts.Problem)
		})
	}
}

func TestInspectPlanNeedsSpecAndBase(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	r := f.implementing(t)
	r.Base = ""

	facts, err := InspectPlan(t.Context(), f.repo, "W-001", r, head(t, f.worktree))

	require.NoError(t, err)
	assert.Nil(t, facts)
}

func TestImplementationEntryNeedsApprovedPlan(t *testing.T) {
	t.Parallel()

	tests := []struct{ name, spec, code string }{
		{name: "in review", spec: "# W-001: Login\nStatus: In review\n\n## Implementation plan\n- S-1: feat: x\n", code: "plan_not_approved"},
		{name: "no steps", spec: "# W-001: Login\nStatus: Approved\n", code: "plan_invalid"},
		{name: "invalid", spec: "# W-001: Login\nStatus: approved\n", code: "plan_invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFactsRepo(t)
			require.NoError(t, os.WriteFile(filepath.Join(f.worktree, filepath.FromSlash(specPath)), []byte(tt.spec), 0o644))
			gittest.Git(t, f.worktree, "commit", "-qam", "spec(W-001): revise")
			before := withSpec(moving(StatusRunning, StageSpecification, nil))

			_, err := checkFacts(t, f.repo, before, Move{Stage: "implementation"})

			requireRejection(t, err, tt.code)
		})
	}
}

func TestImplementationEntryReadsTheCommittedPlan(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(f.worktree, filepath.FromSlash(specPath)),
		[]byte("# W-001: Login\nStatus: In review\n"), 0o644))
	before := withSpec(moving(StatusRunning, StageSpecification, nil))

	_, err := checkFacts(t, f.repo, before, Move{Stage: "implementation"})

	require.NoError(t, err, "an uncommitted edit does not change the committed approval")
}

func TestReviewEntryNeedsCheckpointedCandidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, f factsRepo, r *Record)
		code  string
		found string
	}{
		{
			name: "complete",
			setup: func(t *testing.T, f factsRepo, r *Record) {
				t.Helper()
				f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")
				r.Revision = f.checkpoint(t, "feat(W-001): add the login form", "W-001/S-2")
			},
		},
		{
			name: "outcome closes a step",
			setup: func(t *testing.T, f factsRepo, r *Record) {
				t.Helper()
				spec := approvedSpec + "  Outcome: verified — covered by S-1\n"
				require.NoError(t, os.WriteFile(filepath.Join(f.worktree, filepath.FromSlash(specPath)), []byte(spec), 0o644))
				gittest.Git(t, f.worktree, "commit", "-qam", "spec(W-001): record the S-2 outcome")
				r.Revision = f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")
			},
		},
		{
			name: "remaining steps",
			setup: func(t *testing.T, f factsRepo, r *Record) {
				t.Helper()
				r.Revision = f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")
			},
			code: "plan_incomplete", found: "S-2 has no checkpoint commit",
		},
		{
			name: "discrepancy",
			setup: func(t *testing.T, f factsRepo, r *Record) {
				t.Helper()
				f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")
				r.Revision = f.checkpoint(t, "feat(W-001): add the login form", "")
			},
			code: "checkpoint_discrepancy",
		},
		{
			name: "not the tip",
			setup: func(t *testing.T, f factsRepo, r *Record) {
				t.Helper()
				f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")
				r.Revision = f.checkpoint(t, "feat(W-001): add the login form", "W-001/S-2")
				f.checkpoint(t, "chore: later", "")
			},
			code: "candidate_not_tip",
		},
		{
			name: "dirty",
			setup: func(t *testing.T, f factsRepo, r *Record) {
				t.Helper()
				f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")
				r.Revision = f.checkpoint(t, "feat(W-001): add the login form", "W-001/S-2")
				require.NoError(t, os.WriteFile(filepath.Join(f.worktree, "draft.txt"), nil, 0o644))
			},
			code: "candidate_dirty", found: "1 uncommitted path",
		},
		{
			name:  "no base",
			setup: func(_ *testing.T, _ factsRepo, r *Record) { r.Base = "" },
			code:  "base_missing",
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i)+" "+tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFactsRepo(t)
			before := f.implementing(t)
			tt.setup(t, f, before)

			_, err := checkFacts(t, f.repo, before, Move{Revision: before.Revision, Tier: "1", Stage: "specialized_review"})

			if tt.code == "" {
				require.NoError(t, err)
				return
			}
			rejection := requireRejection(t, err, tt.code)
			assert.Contains(t, rejection.Found, tt.found)
		})
	}
}

func TestConfirmChecksThePlanBeforeIntegration(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(f.worktree, filepath.FromSlash(specPath)),
		[]byte("# W-001: Login\nStatus: In review\n"), 0o644))
	gittest.Git(t, f.worktree, "commit", "-qam", "spec(W-001): reopen the plan")
	review := registryOf(f.unconfirmed(t, StageImplementation))
	integration := registryOf(f.unconfirmed(t, StageIntegration))

	_, reviewErr := review.ConfirmTask(t.Context(), f.repo, "W-001", "orchestrator-1", clock())
	_, integrationErr := integration.ConfirmTask(t.Context(), f.repo, "W-001", "orchestrator-1", clock())

	requireRejection(t, reviewErr, "plan_not_approved")
	require.NoError(t, integrationErr, "closure deletes the work spec at integration")
}

func TestInspectPlanSkipsCommitsMergedFromMain(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	r := f.implementing(t)
	s1 := f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")
	require.NoError(t, os.WriteFile(filepath.Join(f.repo.Dir, "export.go"), []byte("package export\n"), 0o644))
	gittest.Git(t, f.repo.Dir, "add", "export.go")
	gittest.Git(t, f.repo.Dir, "commit", "-m", "feat(export): add csv", "-m", "Step: W-002/S-1")
	gittest.Git(t, f.worktree, "merge", "--no-ff", "-m", "Merge main", "main")

	facts, err := InspectPlan(t.Context(), f.repo, "W-001", r, head(t, f.worktree))

	require.NoError(t, err)
	assert.Empty(t, facts.Discrepancies, "W-002's checkpoint came from main, not from the task branch")
	assert.Equal(t, 1, facts.OtherCommits, "only the spec commit; main's commits are not the task's")
	assert.Equal(t, &s1, facts.Steps[0].Commit)
}
