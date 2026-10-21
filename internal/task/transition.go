package task

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// statusMove is one row of I §8.1; command is the only command allowed to make it, if any.
type statusMove struct {
	from, to    Status
	command     string
	needsResume bool
}

// stageMove is one row of I §8.2; tiers lists the tiers that allow it, nil for any tier.
type stageMove struct {
	from, to Stage
	tiers    []int
}

func statusMoves(prior Status) []statusMove {
	return []statusMove{
		{from: "", to: StatusPlanned, command: "task plan"},
		{from: "", to: StatusReserved, command: "task register"},
		{from: StatusPlanned, to: StatusReserved, command: "task register"},
		{from: StatusReserved, to: StatusRunning},
		{from: StatusReserved, to: StatusBlocked},
		{from: StatusReserved, to: StatusPaused},
		{from: StatusReserved, to: StatusCancelled},
		{from: StatusRunning, to: StatusBlocked},
		{from: StatusRunning, to: StatusPaused},
		{from: StatusRunning, to: StatusCancelled},
		{from: StatusRunning, to: StatusDone, command: "task done"},
		{from: StatusBlocked, to: prior},
		{from: StatusBlocked, to: StatusCancelled},
		{from: StatusPaused, to: prior},
		{from: StatusPaused, to: StatusCancelled},
		{from: StatusCancelled, to: prior, needsResume: true},
	}
}

func stageMoves() []stageMove {
	reviewed := []int{1, 2}
	return []stageMove{
		{from: StageSetup, to: StageSpecification},
		{from: StageSpecification, to: StageImplementation},
		{from: StageImplementation, to: StageSpecification},
		{from: StageImplementation, to: StageSpecializedReview, tiers: reviewed},
		{from: StageImplementation, to: StageIntegration, tiers: []int{0}},
		{from: StageSpecializedReview, to: StageImplementation},
		{from: StageSpecializedReview, to: StageAdversarialReview, tiers: []int{2}},
		{from: StageSpecializedReview, to: StageIntegration, tiers: []int{1}},
		{from: StageAdversarialReview, to: StageImplementation},
		{from: StageAdversarialReview, to: StageSpecializedReview},
		{from: StageAdversarialReview, to: StageIntegration, tiers: []int{2}},
		{from: StageIntegration, to: StageImplementation},
		{from: StageIntegration, to: StageSpecializedReview, tiers: reviewed},
	}
}

// priorStatus is the status a blocked, paused, or cancelled task resumes to (CLI-D16).
func priorStatus(stage Stage) Status {
	if stage == StageSetup {
		return StatusReserved
	}
	return StatusRunning
}

// checkTransition checks a task move from before to after against I §8.1, CLI-D16, and I §8.2.
func checkTransition(before, after *Record, resumeCancelled bool) error {
	if before.Status == StatusDone {
		return &Rejection{
			Code:  "done_final",
			Check: "status done is final (I §8.1)",
			Found: fmt.Sprintf("%s -> %s", before.Status, after.Status),
			Fix:   "register a related task (--related)",
		}
	}
	if before.Status != after.Status {
		if err := checkStatusMove(before, after.Status, resumeCancelled); err != nil {
			return err
		}
	}
	if err := checkReservedAtSetup(after); err != nil {
		return err
	}
	if before.Stage != after.Stage {
		return checkStageMove(before.Stage, after)
	}
	return nil
}

func checkStatusMove(before *Record, to Status, resumeCancelled bool) error {
	rejection := &Rejection{
		Code:  "transition_not_allowed",
		Check: "status change must follow I §8.1",
		Found: fmt.Sprintf("%s -> %s", before.Status, to),
	}
	moves := statusMoves(priorStatus(before.Stage))
	index := slices.IndexFunc(moves, func(m statusMove) bool { return m.from == before.Status && m.to == to })
	switch {
	case index < 0:
		rejection.Fix = "allowed: " + join(allowedStatuses(before))
	case moves[index].command != "":
		rejection.Fix = "use " + moves[index].command
	case moves[index].needsResume && !resumeCancelled:
		rejection.Code = "cancelled_requires_flag"
		rejection.Fix = "pass --resume-cancelled"
	default:
		return nil
	}
	return rejection
}

func checkReservedAtSetup(r *Record) error {
	running := r.Status == StatusRunning && r.Stage == StageSetup
	reserved := r.Status == StatusReserved && r.Stage != StageSetup
	if !running && !reserved {
		return nil
	}
	return &Rejection{
		Code:  "transition_not_allowed",
		Check: "a task is reserved exactly while at stage setup (CLI-D16)",
		Found: fmt.Sprintf("%s at %s", r.Status, r.Stage),
		Fix:   "--status running --stage specification",
	}
}

func checkStageMove(from Stage, after *Record) error {
	rejection := &Rejection{
		Code:  "stage_not_allowed",
		Check: "stage change must follow I §8.2",
		Found: fmt.Sprintf("%s -> %s", from, after.Stage),
	}
	if after.Status != StatusRunning {
		rejection.Fix = fmt.Sprintf("a %s task keeps its stage; change the stage with status running", after.Status)
		return rejection
	}
	moves := slices.DeleteFunc(stageMoves(), func(m stageMove) bool {
		return m.from != from || m.to != after.Stage
	})
	switch {
	case len(moves) == 0:
		rejection.Fix = "allowed: " + join(stageTargets(from, after.Tier))
	case after.Tier != nil && !moves[0].allows(after.Tier):
		rejection.Check, rejection.Fix = tierMismatch(moves[0], *after.Tier)
	case after.Tier == nil && after.Stage.atOrAfter(StageSpecializedReview):
		rejection.Code = "tier_required"
		rejection.Check = "a stage after implementation needs a recorded tier"
		rejection.Fix = "triage the candidate and pass --revision <full-sha> --tier <0|1|2> with the stage move"
	default:
		return nil
	}
	return rejection
}

// tierMismatch words a stage move the tier does not allow, naming the lowest tier that does (I UC-14).
func tierMismatch(move stageMove, tier int) (check, fix string) {
	check = fmt.Sprintf("tier %d does not go from %s to %s (rules §7)", tier, move.from, move.to)
	if !tierReaches(&tier, move.to) {
		check = fmt.Sprintf("tier %d skips %s (rules §7)", tier, move.to)
	}
	targets := stageTargets(move.from, &tier)
	if forward := slices.DeleteFunc(slices.Clone(targets), func(s Stage) bool { return !s.atOrAfter(move.from) }); len(forward) > 0 {
		targets = forward
	}
	lowest := slices.Min(move.tiers)
	return check, fmt.Sprintf("--stage %s, or record tier %d if the triage changed: --tier %d", join(targets), lowest, lowest)
}

// allowedStatuses lists the task move targets of r's status in declaration order.
func allowedStatuses(r *Record) []Status {
	var allowed []Status
	for _, move := range statusMoves(priorStatus(r.Stage)) {
		if move.from == r.Status && move.command == "" {
			allowed = append(allowed, move.to)
		}
	}
	return allowed
}

// allowedStages lists r's next stages in stage order; tier-dependent ones need a recorded tier.
func allowedStages(r *Record) []Stage {
	if r.Status != StatusReserved && r.Status != StatusRunning {
		return nil
	}
	return stageTargets(r.Stage, r.Tier)
}

func stageTargets(from Stage, tier *int) []Stage {
	var targets []Stage
	for _, move := range stageMoves() {
		if move.from == from && move.allows(tier) {
			targets = append(targets, move.to)
		}
	}
	slices.SortFunc(targets, func(a, b Stage) int {
		return cmp.Compare(slices.Index(stages(), a), slices.Index(stages(), b))
	})
	return targets
}

func (m stageMove) allows(tier *int) bool {
	return m.tiers == nil || tier != nil && slices.Contains(m.tiers, *tier)
}

func join[T ~string](values []T) string {
	names := make([]string, len(values))
	for i, value := range values {
		names[i] = string(value)
	}
	return strings.Join(names, ", ")
}
