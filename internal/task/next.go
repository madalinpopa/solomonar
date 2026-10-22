package task

import (
	"slices"
	"strconv"
	"strings"
)

const (
	nextPlaceholder = "--next <actor>:<action>"
	cancelFlags     = "--status cancelled " + nextPlaceholder
)

// AllowedNext lists the commands valid from the record under I sections 8.1 and 8.2
// (I section 9.3).
func AllowedNext(id string, record *Record, facts StatusFacts) []string {
	c := commands{id: id, owner: record.Owner}
	if record.Status != StatusDone && facts.Merge.Merged {
		return []string{c.done(facts.Merge.MergeCommit)}
	}
	prior := "--status " + string(priorStatus(record.Stage))
	switch record.Status {
	case StatusPlanned:
		return []string{"solomonar task register " + id + " --title <title> --owner <owner>"}
	case StatusReserved, StatusRunning:
		next := c.stageMoves(record)
		if record.Stage == StageIntegration {
			next = append(next, c.done(nil))
		}
		return append(next, c.block(), c.task("move", "--status paused", nextPlaceholder), c.task("move", cancelFlags))
	case StatusBlocked:
		next := []string{}
		for _, blocker := range record.Blockers {
			flags := []string{blocker.ID}
			if len(record.Blockers) == 1 {
				flags = append(flags, prior)
			}
			next = append(next, c.task("unblock", flags...))
		}
		return append(next, c.block(), c.task("move", cancelFlags))
	case StatusPaused:
		return []string{c.task("move", prior), c.task("move", cancelFlags)}
	case StatusCancelled:
		return []string{c.task("move", prior, "--resume-cancelled")}
	case StatusDone:
	}
	return []string{}
}

// commands builds task commands acting as the record's owner.
type commands struct{ id, owner string }

func (c commands) task(verb string, flags ...string) string {
	return strings.Join(append([]string{"solomonar task", verb, c.id, "--as", c.owner}, flags...), " ")
}

func (c commands) block() string {
	return c.task("block", "--kind <kind> --summary <text>")
}

func (c commands) done(merge *string) string {
	commit := "<full-sha>"
	if merge != nil {
		commit = *merge
	}
	return c.task("done", "--merge "+commit, `--spec-ref <PR "Final specification" line>`, "--summary <one sentence>")
}

// stageMoves lists r's stage moves in stage order; implementation without a tier offers both.
func (c commands) stageMoves(r *Record) []string {
	moves := slices.DeleteFunc(stageMoves(), func(m stageMove) bool {
		return m.from != r.Stage || !m.allows(r.Tier) && (r.Tier != nil || m.from != StageImplementation)
	})
	slices.SortFunc(moves, func(a, b stageMove) int { return compareStages(a.to, b.to) })
	next := make([]string, 0, len(moves))
	for _, m := range moves {
		flags := stageFlags(r, m)
		if r.Status == StatusReserved {
			flags = slices.Insert(flags, 0, "--status running")
		}
		next = append(next, c.task("move", flags...))
	}
	return next
}

// stageFlags adds the flags a stage move needs and the record lacks, as placeholders (I section 8.2).
func stageFlags(r *Record, m stageMove) []string {
	flags := []string{"--stage " + string(m.to)}
	switch {
	case !m.to.atOrAfter(m.from):
		flags = append(flags, nextPlaceholder)
	case m.to == StageImplementation && r.Spec == "":
		flags = append(flags, "--spec <path>")
	case m.from == StageImplementation:
		if r.Revision == "" {
			flags = append(flags, "--revision <full-sha>")
		}
		if r.Tier == nil {
			flags = append(flags, "--tier "+tierPlaceholder(m))
		}
	}
	return flags
}

func tierPlaceholder(m stageMove) string {
	if len(m.tiers) == 1 {
		return strconv.Itoa(m.tiers[0])
	}
	return "<n>"
}
