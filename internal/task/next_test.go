package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	move   = "solomonar task move W-001 --as orchestrator-1"
	block  = "solomonar task block W-001 --as orchestrator-1 --kind <kind> --summary <text>"
	pause  = move + " --status paused --next <actor>:<action>"
	cancel = move + " --status cancelled --next <actor>:<action>"
	done   = "solomonar task done W-001 --as orchestrator-1 --merge <full-sha>" +
		` --spec-ref <PR "Final specification" line> --summary <one sentence>`
)

func TestAllowedNext(t *testing.T) {
	mergeCommit := "14091a9f2461267ee7e02525b4f1f2923f1c9849"
	merged := cleanFacts()
	merged.Merge = MergeFacts{Merged: true, MergeCommit: &mergeCommit}
	blockedWith := func(stage Stage, ids ...string) *Record {
		r := moving(StatusBlocked, stage, nil)
		for _, id := range ids {
			r.Blockers = append(r.Blockers, Blocker{ID: id, Kind: BlockerCheck, Summary: "CI is red"})
		}
		return r
	}
	tests := []struct {
		name   string
		record *Record
		facts  StatusFacts
		want   []string
	}{
		{
			name:   "UC-16 running at specialized_review, tier 2",
			record: reviewing(),
			want: []string{
				move + " --stage implementation --next <actor>:<action>",
				move + " --stage adversarial_review",
				block, pause, cancel,
			},
		},
		{
			name:   "UC-17 merged at integration",
			record: withRevision(moving(StatusRunning, StageIntegration, new(2))),
			facts:  merged,
			want: []string{
				"solomonar task done W-001 --as orchestrator-1 --merge 14091a9f2461267ee7e02525b4f1f2923f1c9849" +
					` --spec-ref <PR "Final specification" line> --summary <one sentence>`,
			},
		},
		{
			name:   "merged while blocked",
			record: withBlocker(moving(StatusBlocked, StageIntegration, new(1))),
			facts:  merged,
			want: []string{
				"solomonar task done W-001 --as orchestrator-1 --merge 14091a9f2461267ee7e02525b4f1f2923f1c9849" +
					` --spec-ref <PR "Final specification" line> --summary <one sentence>`,
			},
		},
		{
			name:   "integration before the merge, tier 2",
			record: moving(StatusRunning, StageIntegration, new(2)),
			want: []string{
				move + " --stage implementation --next <actor>:<action>",
				move + " --stage specialized_review --next <actor>:<action>",
				done, block, pause, cancel,
			},
		},
		{
			name:   "integration, tier 0",
			record: moving(StatusRunning, StageIntegration, new(0)),
			want:   []string{move + " --stage implementation --next <actor>:<action>", done, block, pause, cancel},
		},
		{
			name:   "reserved at setup",
			record: moving(StatusReserved, StageSetup, nil),
			want:   []string{move + " --status running --stage specification", block, pause, cancel},
		},
		{
			name:   "specification without spec",
			record: moving(StatusRunning, StageSpecification, nil),
			want:   []string{move + " --stage implementation --spec <path>", block, pause, cancel},
		},
		{
			name:   "specification with spec",
			record: withSpec(moving(StatusRunning, StageSpecification, nil)),
			want:   []string{move + " --stage implementation", block, pause, cancel},
		},
		{
			name:   "implementation without revision or tier",
			record: moving(StatusRunning, StageImplementation, nil),
			want: []string{
				move + " --stage specification --next <actor>:<action>",
				move + " --stage specialized_review --revision <full-sha> --tier <n>",
				move + " --stage integration --revision <full-sha> --tier 0",
				block, pause, cancel,
			},
		},
		{
			name:   "implementation with revision, no tier",
			record: withRevision(moving(StatusRunning, StageImplementation, nil)),
			want: []string{
				move + " --stage specification --next <actor>:<action>",
				move + " --stage specialized_review --tier <n>",
				move + " --stage integration --tier 0",
				block, pause, cancel,
			},
		},
		{
			name:   "implementation, tier 2, no revision",
			record: moving(StatusRunning, StageImplementation, new(2)),
			want: []string{
				move + " --stage specification --next <actor>:<action>",
				move + " --stage specialized_review --revision <full-sha>",
				block, pause, cancel,
			},
		},
		{
			name:   "implementation, tier 0, revision recorded",
			record: withRevision(moving(StatusRunning, StageImplementation, new(0))),
			want: []string{
				move + " --stage specification --next <actor>:<action>",
				move + " --stage integration",
				block, pause, cancel,
			},
		},
		{
			name:   "specialized_review, tier 1",
			record: moving(StatusRunning, StageSpecializedReview, new(1)),
			want: []string{
				move + " --stage implementation --next <actor>:<action>",
				move + " --stage integration",
				block, pause, cancel,
			},
		},
		{
			name:   "specialized_review without tier",
			record: moving(StatusRunning, StageSpecializedReview, nil),
			want:   []string{move + " --stage implementation --next <actor>:<action>", block, pause, cancel},
		},
		{
			name:   "adversarial_review, tier 2",
			record: moving(StatusRunning, StageAdversarialReview, new(2)),
			want: []string{
				move + " --stage implementation --next <actor>:<action>",
				move + " --stage specialized_review --next <actor>:<action>",
				move + " --stage integration",
				block, pause, cancel,
			},
		},
		{
			name:   "blocked with one blocker",
			record: blockedWith(StageImplementation, "B-1"),
			want:   []string{"solomonar task unblock W-001 --as orchestrator-1 B-1 --status running", block, cancel},
		},
		{
			name:   "blocked at setup",
			record: blockedWith(StageSetup, "B-2"),
			want:   []string{"solomonar task unblock W-001 --as orchestrator-1 B-2 --status reserved", block, cancel},
		},
		{
			name:   "blocked with two blockers",
			record: blockedWith(StageSpecification, "B-2", "B-1"),
			want: []string{
				"solomonar task unblock W-001 --as orchestrator-1 B-2",
				"solomonar task unblock W-001 --as orchestrator-1 B-1",
				block, cancel,
			},
		},
		{
			name:   "blocked without blockers",
			record: blockedWith(StageSpecification),
			want:   []string{block, cancel},
		},
		{
			name:   "paused",
			record: moving(StatusPaused, StageSpecification, nil),
			want:   []string{move + " --status running", cancel},
		},
		{
			name:   "paused at setup",
			record: moving(StatusPaused, StageSetup, nil),
			want:   []string{move + " --status reserved", cancel},
		},
		{
			name:   "cancelled",
			record: moving(StatusCancelled, StageImplementation, nil),
			want:   []string{move + " --status running --resume-cancelled"},
		},
		{
			name:   "planned",
			record: &Record{Title: "Login", Status: StatusPlanned},
			want:   []string{"solomonar task register W-001 --title <title> --owner <owner>"},
		},
		{
			name:   "done",
			record: &Record{Title: "Login", Status: StatusDone, Stage: StageComplete, Owner: "orchestrator-1"},
			facts:  merged,
			want:   []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AllowedNext("W-001", tt.record, tt.facts))
		})
	}
}
