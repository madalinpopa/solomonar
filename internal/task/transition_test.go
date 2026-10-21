package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func record(status Status, stage Stage, tier *int) *Record {
	return &Record{Status: status, Stage: stage, Tier: tier}
}

func TestPriorStatus(t *testing.T) {
	assert.Equal(t, StatusReserved, priorStatus(StageSetup))
	assert.Equal(t, StatusRunning, priorStatus(StageSpecification))
	assert.Equal(t, StatusRunning, priorStatus(StageIntegration))
}

func TestCheckStatusTransition(t *testing.T) {
	tests := []struct {
		name   string
		before *Record
		after  *Record
		resume bool
		code   string
	}{
		{name: "reserved to running", before: record(StatusReserved, StageSetup, nil), after: record(StatusRunning, StageSpecification, nil)},
		{name: "reserved to blocked", before: record(StatusReserved, StageSetup, nil), after: record(StatusBlocked, StageSetup, nil)},
		{name: "reserved to paused", before: record(StatusReserved, StageSetup, nil), after: record(StatusPaused, StageSetup, nil)},
		{name: "reserved to cancelled", before: record(StatusReserved, StageSetup, nil), after: record(StatusCancelled, StageSetup, nil)},
		{name: "reserved to planned", before: record(StatusReserved, StageSetup, nil), after: record(StatusPlanned, StageSetup, nil), code: "transition_not_allowed"},
		{name: "reserved to done", before: record(StatusReserved, StageSetup, nil), after: record(StatusDone, StageSetup, nil), code: "transition_not_allowed"},
		{name: "running to blocked", before: record(StatusRunning, StageImplementation, nil), after: record(StatusBlocked, StageImplementation, nil)},
		{name: "running to paused", before: record(StatusRunning, StageImplementation, nil), after: record(StatusPaused, StageImplementation, nil)},
		{name: "running to cancelled", before: record(StatusRunning, StageImplementation, nil), after: record(StatusCancelled, StageImplementation, nil)},
		{name: "running to done needs task done", before: record(StatusRunning, StageIntegration, new(1)), after: record(StatusDone, StageComplete, new(1)), code: "transition_not_allowed"},
		{name: "running to planned", before: record(StatusRunning, StageImplementation, nil), after: record(StatusPlanned, StageImplementation, nil), code: "transition_not_allowed"},
		{name: "running to reserved", before: record(StatusRunning, StageImplementation, nil), after: record(StatusReserved, StageImplementation, nil), code: "transition_not_allowed"},
		{name: "blocked at setup resumes reserved", before: record(StatusBlocked, StageSetup, nil), after: record(StatusReserved, StageSetup, nil)},
		{name: "blocked later resumes running", before: record(StatusBlocked, StageImplementation, nil), after: record(StatusRunning, StageImplementation, nil)},
		{name: "blocked at setup to running", before: record(StatusBlocked, StageSetup, nil), after: record(StatusRunning, StageSetup, nil), code: "transition_not_allowed"},
		{name: "blocked later to reserved", before: record(StatusBlocked, StageImplementation, nil), after: record(StatusReserved, StageImplementation, nil), code: "transition_not_allowed"},
		{name: "blocked to cancelled", before: record(StatusBlocked, StageImplementation, nil), after: record(StatusCancelled, StageImplementation, nil)},
		{name: "blocked to paused", before: record(StatusBlocked, StageImplementation, nil), after: record(StatusPaused, StageImplementation, nil), code: "transition_not_allowed"},
		{name: "paused at setup resumes reserved", before: record(StatusPaused, StageSetup, nil), after: record(StatusReserved, StageSetup, nil)},
		{name: "paused later resumes running", before: record(StatusPaused, StageSpecification, nil), after: record(StatusRunning, StageSpecification, nil)},
		{name: "paused at setup to running", before: record(StatusPaused, StageSetup, nil), after: record(StatusRunning, StageSetup, nil), code: "transition_not_allowed"},
		{name: "paused later to reserved", before: record(StatusPaused, StageSpecification, nil), after: record(StatusReserved, StageSpecification, nil), code: "transition_not_allowed"},
		{name: "paused to cancelled", before: record(StatusPaused, StageSpecification, nil), after: record(StatusCancelled, StageSpecification, nil)},
		{name: "paused to blocked", before: record(StatusPaused, StageSpecification, nil), after: record(StatusBlocked, StageSpecification, nil), code: "transition_not_allowed"},
		{name: "cancelled at setup resumes reserved with flag", before: record(StatusCancelled, StageSetup, nil), after: record(StatusReserved, StageSetup, nil), resume: true},
		{name: "cancelled later resumes running with flag", before: record(StatusCancelled, StageImplementation, nil), after: record(StatusRunning, StageImplementation, nil), resume: true},
		{name: "cancelled resumes without flag", before: record(StatusCancelled, StageImplementation, nil), after: record(StatusRunning, StageImplementation, nil), code: "cancelled_requires_flag"},
		{name: "cancelled later to reserved", before: record(StatusCancelled, StageImplementation, nil), after: record(StatusReserved, StageImplementation, nil), resume: true, code: "transition_not_allowed"},
		{name: "cancelled to paused", before: record(StatusCancelled, StageImplementation, nil), after: record(StatusPaused, StageImplementation, nil), resume: true, code: "transition_not_allowed"},
		{name: "planned to reserved needs task register", before: record(StatusPlanned, "", nil), after: record(StatusReserved, StageSetup, nil), code: "transition_not_allowed"},
		{name: "planned to running", before: record(StatusPlanned, "", nil), after: record(StatusRunning, StageSpecification, nil), code: "transition_not_allowed"},
		{name: "done to running", before: record(StatusDone, StageComplete, new(1)), after: record(StatusRunning, StageIntegration, new(1)), code: "done_final"},
		{name: "done unchanged", before: record(StatusDone, StageComplete, new(1)), after: record(StatusDone, StageComplete, new(1)), code: "done_final"},
		{name: "running stays running", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageImplementation, nil)},
		{name: "invariant: running at setup", before: record(StatusReserved, StageSetup, nil), after: record(StatusRunning, StageSetup, nil), code: "transition_not_allowed"},
		{name: "invariant: reserved past setup", before: record(StatusReserved, StageSetup, nil), after: record(StatusReserved, StageSpecification, nil), code: "transition_not_allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCode(t, checkTransition(tt.before, tt.after, tt.resume), tt.code)
		})
	}
}

func TestCheckStageTransition(t *testing.T) {
	tests := []struct {
		name   string
		before *Record
		after  *Record
		code   string
	}{
		{name: "setup to specification", before: record(StatusReserved, StageSetup, nil), after: record(StatusRunning, StageSpecification, nil)},
		{name: "setup to implementation", before: record(StatusReserved, StageSetup, nil), after: record(StatusRunning, StageImplementation, nil), code: "stage_not_allowed"},
		{name: "specification to implementation", before: record(StatusRunning, StageSpecification, nil), after: record(StatusRunning, StageImplementation, nil)},
		{name: "specification to setup", before: record(StatusRunning, StageSpecification, nil), after: record(StatusRunning, StageSetup, nil), code: "transition_not_allowed"},
		{name: "specification to specialized review", before: record(StatusRunning, StageSpecification, new(1)), after: record(StatusRunning, StageSpecializedReview, new(1)), code: "stage_not_allowed"},
		{name: "implementation to specialized review tier 1", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageSpecializedReview, new(1))},
		{name: "implementation to specialized review tier 2", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageSpecializedReview, new(2))},
		{name: "implementation to specialized review tier 0", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageSpecializedReview, new(0)), code: "stage_not_allowed"},
		{name: "implementation to integration tier 0", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageIntegration, new(0))},
		{name: "implementation to integration tier 1", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageIntegration, new(1)), code: "stage_not_allowed"},
		{name: "implementation to integration tier 2", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageIntegration, new(2)), code: "stage_not_allowed"},
		{name: "implementation to adversarial review tier 2", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageAdversarialReview, new(2)), code: "stage_not_allowed"},
		{name: "implementation to specification", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageSpecification, nil)},
		{name: "implementation to specialized review without tier", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageSpecializedReview, nil), code: "tier_required"},
		{name: "implementation to integration without tier", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageIntegration, nil), code: "tier_required"},
		{name: "specialized review to adversarial review tier 2", before: record(StatusRunning, StageSpecializedReview, new(2)), after: record(StatusRunning, StageAdversarialReview, new(2))},
		{name: "specialized review to adversarial review tier 1", before: record(StatusRunning, StageSpecializedReview, new(1)), after: record(StatusRunning, StageAdversarialReview, new(1)), code: "stage_not_allowed"},
		{name: "specialized review to integration tier 1", before: record(StatusRunning, StageSpecializedReview, new(1)), after: record(StatusRunning, StageIntegration, new(1))},
		{name: "specialized review to integration tier 2", before: record(StatusRunning, StageSpecializedReview, new(2)), after: record(StatusRunning, StageIntegration, new(2)), code: "stage_not_allowed"},
		{name: "specialized review to implementation", before: record(StatusRunning, StageSpecializedReview, new(1)), after: record(StatusRunning, StageImplementation, new(1))},
		{name: "specialized review to specification", before: record(StatusRunning, StageSpecializedReview, new(1)), after: record(StatusRunning, StageSpecification, new(1)), code: "stage_not_allowed"},
		{name: "adversarial review to integration tier 2", before: record(StatusRunning, StageAdversarialReview, new(2)), after: record(StatusRunning, StageIntegration, new(2))},
		{name: "adversarial review to integration tier 1", before: record(StatusRunning, StageAdversarialReview, new(1)), after: record(StatusRunning, StageIntegration, new(1)), code: "stage_not_allowed"},
		{name: "adversarial review to implementation", before: record(StatusRunning, StageAdversarialReview, new(2)), after: record(StatusRunning, StageImplementation, new(2))},
		{name: "adversarial review to specialized review", before: record(StatusRunning, StageAdversarialReview, new(2)), after: record(StatusRunning, StageSpecializedReview, new(2))},
		{name: "adversarial review to specification", before: record(StatusRunning, StageAdversarialReview, new(2)), after: record(StatusRunning, StageSpecification, new(2)), code: "stage_not_allowed"},
		{name: "integration to complete", before: record(StatusRunning, StageIntegration, new(1)), after: record(StatusRunning, StageComplete, new(1)), code: "stage_not_allowed"},
		{name: "integration to implementation", before: record(StatusRunning, StageIntegration, new(0)), after: record(StatusRunning, StageImplementation, new(0))},
		{name: "integration to specialized review tier 1", before: record(StatusRunning, StageIntegration, new(1)), after: record(StatusRunning, StageSpecializedReview, new(1))},
		{name: "integration to specialized review tier 2", before: record(StatusRunning, StageIntegration, new(2)), after: record(StatusRunning, StageSpecializedReview, new(2))},
		{name: "integration to specialized review tier 0", before: record(StatusRunning, StageIntegration, new(0)), after: record(StatusRunning, StageSpecializedReview, new(0)), code: "stage_not_allowed"},
		{name: "integration to adversarial review tier 2", before: record(StatusRunning, StageIntegration, new(2)), after: record(StatusRunning, StageAdversarialReview, new(2)), code: "stage_not_allowed"},
		{name: "stage change while blocked", before: record(StatusBlocked, StageImplementation, nil), after: record(StatusBlocked, StageSpecification, nil), code: "stage_not_allowed"},
		{name: "stage change while paused", before: record(StatusPaused, StageImplementation, nil), after: record(StatusPaused, StageSpecification, nil), code: "stage_not_allowed"},
		{name: "stage change while cancelled", before: record(StatusCancelled, StageImplementation, nil), after: record(StatusCancelled, StageSpecification, nil), code: "stage_not_allowed"},
		{name: "stage change into blocked", before: record(StatusRunning, StageSpecification, nil), after: record(StatusBlocked, StageImplementation, nil), code: "stage_not_allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCode(t, checkTransition(tt.before, tt.after, false), tt.code)
		})
	}
}

func TestCheckTransitionFixNamesAllowedTargets(t *testing.T) {
	tests := []struct {
		name   string
		before *Record
		after  *Record
		found  string
		fix    string
	}{
		{name: "status", before: record(StatusRunning, StageImplementation, nil), after: record(StatusPlanned, StageImplementation, nil), found: "running -> planned", fix: "allowed: blocked, paused, cancelled"},
		{name: "stage", before: record(StatusRunning, StageImplementation, nil), after: record(StatusRunning, StageIntegration, new(1)), found: "implementation -> integration", fix: "allowed: specification, specialized_review"},
		{name: "done", before: record(StatusRunning, StageIntegration, new(1)), after: record(StatusDone, StageComplete, new(1)), found: "running -> done", fix: "use task done"},
		{name: "invariant", before: record(StatusReserved, StageSetup, nil), after: record(StatusRunning, StageSetup, nil), found: "running at setup", fix: "--status running --stage specification"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rejection := rejectionOf(t, checkTransition(tt.before, tt.after, false))

			assert.Equal(t, tt.found, rejection.Found)
			assert.Equal(t, tt.fix, rejection.Fix)
		})
	}
}

func TestAllowedStatuses(t *testing.T) {
	tests := []struct {
		name   string
		record *Record
		want   []Status
	}{
		{name: "planned", record: record(StatusPlanned, "", nil), want: nil},
		{name: "reserved", record: record(StatusReserved, StageSetup, nil), want: []Status{StatusRunning, StatusBlocked, StatusPaused, StatusCancelled}},
		{name: "running", record: record(StatusRunning, StageImplementation, nil), want: []Status{StatusBlocked, StatusPaused, StatusCancelled}},
		{name: "blocked at setup", record: record(StatusBlocked, StageSetup, nil), want: []Status{StatusReserved, StatusCancelled}},
		{name: "paused later", record: record(StatusPaused, StageIntegration, new(0)), want: []Status{StatusRunning, StatusCancelled}},
		{name: "cancelled later", record: record(StatusCancelled, StageSpecification, nil), want: []Status{StatusRunning}},
		{name: "done", record: record(StatusDone, StageComplete, nil), want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, allowedStatuses(tt.record))
		})
	}
}

func TestAllowedStages(t *testing.T) {
	tests := []struct {
		name   string
		record *Record
		want   []Stage
	}{
		{name: "reserved at setup", record: record(StatusReserved, StageSetup, nil), want: []Stage{StageSpecification}},
		{name: "implementation without tier", record: record(StatusRunning, StageImplementation, nil), want: []Stage{StageSpecification}},
		{name: "implementation tier 0", record: record(StatusRunning, StageImplementation, new(0)), want: []Stage{StageSpecification, StageIntegration}},
		{name: "implementation tier 2", record: record(StatusRunning, StageImplementation, new(2)), want: []Stage{StageSpecification, StageSpecializedReview}},
		{name: "adversarial review", record: record(StatusRunning, StageAdversarialReview, new(2)), want: []Stage{StageImplementation, StageSpecializedReview, StageIntegration}},
		{name: "integration tier 1", record: record(StatusRunning, StageIntegration, new(1)), want: []Stage{StageImplementation, StageSpecializedReview}},
		{name: "blocked", record: record(StatusBlocked, StageImplementation, new(1)), want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, allowedStages(tt.record))
		})
	}
}

func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	if code == "" {
		require.NoError(t, err)
		return
	}
	assert.Equal(t, code, rejectionOf(t, err).Code)
}
