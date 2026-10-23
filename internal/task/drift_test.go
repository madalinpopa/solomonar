package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const prURL = "https://github.com/acme/ledger/pull/42"

func reviewing() *Record {
	r := moving(StatusRunning, StageSpecializedReview, new(2))
	r.Base, r.Worktree = sha, "../W-001"
	return r
}

// cleanFacts are the facts of a record without drift.
func cleanFacts() StatusFacts {
	return StatusFacts{
		Branch:   BranchFacts{Name: "feature/W-001", Exists: true, Tip: new(sha), ContainsRevision: new(true)},
		Worktree: &WorktreeFacts{Path: "/work/W-001", Relative: "../W-001", Head: sha, AheadOfRevision: new(0), Dirty: []DirtyPath{}},
		Spec:     SpecFacts{InWorktree: true, AtRevision: true},
		Forge:    ForgeFacts{Available: true, Checks: "unknown"},
	}
}

func dirty(n int) *WorktreeFacts {
	w := cleanFacts().Worktree
	for range n {
		w.Dirty = append(w.Dirty, DirtyPath{Path: "a.go", State: "modified"})
	}
	return w
}

func openPR() *PullRequest {
	return &PullRequest{Number: 42, URL: prURL, State: "OPEN"}
}

func TestDriftOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		record   func(r *Record)
		facts    func(f *StatusFacts)
		envOwner string
		want     []Drift
	}{
		{name: "none", want: []Drift{}},
		{
			name:   "branch missing while running past setup",
			record: func(r *Record) { r.Stage = StageImplementation },
			facts: func(f *StatusFacts) {
				f.Branch = BranchFacts{Name: "feature/W-001"}
				f.Worktree = nil
			},
			want: []Drift{{Code: "branch_missing", Message: "feature/W-001 does not exist"}},
		},
		{
			name:   "branch missing while paused",
			record: func(r *Record) { r.Status = StatusPaused },
			facts: func(f *StatusFacts) {
				f.Branch = BranchFacts{Name: "feature/W-001"}
				f.Worktree = nil
			},
			want: []Drift{},
		},
		{
			name: "branch missing while reserved at setup",
			record: func(r *Record) {
				*r = *moving(StatusReserved, StageSetup, nil)
			},
			facts: func(f *StatusFacts) { *f = StatusFacts{Branch: BranchFacts{Name: "feature/W-001"}} },
			want:  []Drift{},
		},
		{
			name:  "worktree missing",
			facts: func(f *StatusFacts) { f.Worktree = nil },
			want:  []Drift{{Code: "worktree_missing", Message: "feature/W-001 has no worktree"}},
		},
		{
			name:  "head one commit ahead",
			facts: func(f *StatusFacts) { f.Worktree.AheadOfRevision = new(1) },
			want:  []Drift{{Code: "head_ahead_of_revision", Message: "HEAD is 1 commit past the recorded revision"}},
		},
		{
			name:  "head three commits ahead",
			facts: func(f *StatusFacts) { f.Worktree.AheadOfRevision = new(3) },
			want:  []Drift{{Code: "head_ahead_of_revision", Message: "HEAD is 3 commits past the recorded revision"}},
		},
		{
			name:  "unknown ahead count",
			facts: func(f *StatusFacts) { f.Worktree.AheadOfRevision = nil },
			want:  []Drift{},
		},
		{
			name:  "revision not on branch",
			facts: func(f *StatusFacts) { f.Branch.ContainsRevision = new(false) },
			want:  []Drift{{Code: "revision_not_on_branch", Message: "01234567 is not on feature/W-001"}},
		},
		{
			name:  "unknown containment",
			facts: func(f *StatusFacts) { f.Branch.ContainsRevision = nil },
			want:  []Drift{},
		},
		{
			name:  "one dirty path",
			facts: func(f *StatusFacts) { f.Worktree = dirty(1) },
			want:  []Drift{{Code: "dirty_worktree", Message: "1 uncommitted path; preserve it (rules section 10)"}},
		},
		{
			name:  "two dirty paths",
			facts: func(f *StatusFacts) { f.Worktree = dirty(2) },
			want:  []Drift{{Code: "dirty_worktree", Message: "2 uncommitted paths; preserve them (rules section 10)"}},
		},
		{
			name:  "spec missing",
			facts: func(f *StatusFacts) { f.Spec = SpecFacts{} },
			want: []Drift{{
				Code: "spec_missing", Message: "docs/work/W-001-login.md is not in the worktree or at the revision",
			}},
		},
		{
			name:  "spec only at revision",
			facts: func(f *StatusFacts) { f.Spec.InWorktree = false },
			want:  []Drift{},
		},
		{
			name:   "no spec recorded",
			record: func(r *Record) { *r = *moving(StatusRunning, StageSpecification, nil); r.Base = sha },
			facts:  func(f *StatusFacts) { f.Spec = SpecFacts{} },
			want:   []Drift{},
		},
		{
			name:   "blocked without blockers",
			record: func(r *Record) { r.Status = StatusBlocked },
			want:   []Drift{{Code: "blocked_without_blockers", Message: "blocked without an open blocker"}},
		},
		{
			name:   "blocked with a blocker",
			record: func(r *Record) { withBlocker(r).Status = StatusBlocked },
			want:   []Drift{},
		},
		{
			name:   "tier missing after implementation",
			record: func(r *Record) { r.Tier = nil },
			want:   []Drift{{Code: "tier_missing", Message: "stage specialized_review needs a tier"}},
		},
		{
			name:   "no tier at implementation",
			record: func(r *Record) { *r = *withRevision(moving(StatusRunning, StageImplementation, nil)); r.Base = sha },
			want:   []Drift{},
		},
		{
			name:  "merged not done",
			facts: func(f *StatusFacts) { f.Merge = MergeFacts{Merged: true, MergeCommit: new(sha)} },
			want: []Drift{{
				Code: "merged_not_done", Message: "feature/W-001 is merged into main but the record is not done",
			}},
		},
		{
			name:  "pr not recorded",
			facts: func(f *StatusFacts) { f.Forge.PR = openPR() },
			want: []Drift{{
				Code: "pr_not_recorded", Message: "gh shows PR #42 for feature/W-001; the record has none",
			}},
		},
		{
			name:   "pr mismatch",
			record: func(r *Record) { r.PR = "https://github.com/acme/ledger/pull/41" },
			facts:  func(f *StatusFacts) { f.Forge.PR = openPR() },
			want: []Drift{{
				Code:    "pr_mismatch",
				Message: "the record has PR https://github.com/acme/ledger/pull/41; gh shows #42 for feature/W-001",
			}},
		},
		{
			name:   "pr recorded as number",
			record: func(r *Record) { r.PR = "42" },
			facts:  func(f *StatusFacts) { f.Forge.PR = openPR() },
			want:   []Drift{},
		},
		{
			name:   "pr recorded as url",
			record: func(r *Record) { r.PR = prURL },
			facts:  func(f *StatusFacts) { f.Forge.PR = openPR() },
			want:   []Drift{},
		},
		{
			name:   "pr recorded without forge",
			record: func(r *Record) { r.PR = prURL },
			want:   []Drift{},
		},
		{
			name:     "owned by other",
			envOwner: "orchestrator-2",
			want: []Drift{{
				Code: "owned_by_other", Message: "SOLOMONAR_OWNER is orchestrator-2; W-001 is owned by orchestrator-1",
			}},
		},
		{name: "owned by self", envOwner: "orchestrator-1", want: []Drift{}},
		{
			name:     "cancelled only owned by other",
			record:   func(r *Record) { r.Status = StatusCancelled },
			facts:    func(f *StatusFacts) { f.Worktree = dirty(1) },
			envOwner: "orchestrator-2",
			want: []Drift{{
				Code: "owned_by_other", Message: "SOLOMONAR_OWNER is orchestrator-2; W-001 is owned by orchestrator-1",
			}},
		},
		{
			name:   "done without drift",
			record: func(r *Record) { r.Status, r.Stage = StatusDone, StageComplete },
			facts:  func(f *StatusFacts) { f.Merge = MergeFacts{Merged: true}; f.Forge.PR = openPR() },
			want:   []Drift{},
		},
		{
			name:     "planned without owner",
			record:   func(r *Record) { *r = Record{Title: "Login", Status: StatusPlanned} },
			facts:    func(f *StatusFacts) { *f = StatusFacts{Branch: BranchFacts{Name: "feature/W-001"}} },
			envOwner: "orchestrator-2",
			want:     []Drift{},
		},
		{
			name:   "recovery fields",
			record: func(r *Record) { r.Base, r.Confirmed = "", false },
			facts:  func(f *StatusFacts) { f.Worktree.Relative = "../moved/W-001" },
			want: []Drift{
				{Code: "worktree_mismatch", Message: "recorded ../W-001; Git lists ../moved/W-001"},
				{Code: "base_missing", Message: "W-001 has no recorded base"},
				{Code: "unconfirmed", Message: "W-001 is not confirmed by orchestrator-1; inspect, then task confirm"},
			},
		},
		{
			name:   "plan",
			record: func(r *Record) { r.Stage = StageImplementation },
			facts: func(f *StatusFacts) {
				f.Plan = &PlanFacts{Approval: PlanInReview, Discrepancies: []Discrepancy{{Code: "step_empty"}}}
			},
			want: []Drift{
				{Code: "plan_not_approved", Message: "the work spec at feature/W-001 has no approved plan (in_review)"},
				{Code: "checkpoint_discrepancy", Message: "1 commit claim to inspect"},
			},
		},
		{
			name:   "migrated record without workspace fields",
			record: func(r *Record) { r.Worktree = "" },
			want:   []Drift{},
		},
		{
			name:     "table order",
			facts:    func(f *StatusFacts) { f.Worktree = dirty(1); f.Worktree.AheadOfRevision = new(1) },
			envOwner: "orchestrator-2",
			want: []Drift{
				{Code: "head_ahead_of_revision", Message: "HEAD is 1 commit past the recorded revision"},
				{Code: "dirty_worktree", Message: "1 uncommitted path; preserve it (rules section 10)"},
				{Code: "owned_by_other", Message: "SOLOMONAR_OWNER is orchestrator-2; W-001 is owned by orchestrator-1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, facts := reviewing(), cleanFacts()
			if tt.record != nil {
				tt.record(r)
			}
			if tt.facts != nil {
				tt.facts(&facts)
			}

			assert.Equal(t, tt.want, DriftOf("W-001", "main", r, facts, tt.envOwner))
		})
	}
}
