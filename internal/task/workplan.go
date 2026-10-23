package task

import (
	"fmt"
	"strings"
)

// Approval values of the work spec's Status line (CLI-D23).
const (
	ApprovalApproved = "Approved"
	ApprovalInReview = "In review"
)

// Step outcomes that close a step without a checkpoint commit.
const (
	OutcomeDropped  = "dropped"
	OutcomeVerified = "verified"
)

const planHeading = "## Implementation plan"

// WorkPlan is the approval status and the implementation plan of a work spec.
type WorkPlan struct {
	Approval string
	Steps    []Step
}

// Step is one planned checkpoint commit: S-<n> and its commit subject.
type Step struct {
	ID      string
	Subject string
	Outcome string
}

// ParseWorkPlan reads the Status line below the title and the steps under ## Implementation plan.
// A missing Status line leaves Approval empty; a malformed line or step is an error.
func ParseWorkPlan(content string) (WorkPlan, error) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var plan WorkPlan
	if len(lines) > 1 && strings.HasPrefix(lines[0], "# ") {
		if value, ok := strings.CutPrefix(lines[1], "Status:"); ok {
			plan.Approval = strings.TrimSpace(value)
			if plan.Approval != ApprovalApproved && plan.Approval != ApprovalInReview {
				return WorkPlan{}, fmt.Errorf("status %q is not %q or %q", plan.Approval, ApprovalInReview, ApprovalApproved)
			}
		}
	}
	steps, err := parseSteps(planSection(lines))
	if err != nil {
		return WorkPlan{}, err
	}
	plan.Steps = steps
	return plan, nil
}

// planSection returns the lines between the plan heading and the next level-two heading.
func planSection(lines []string) []string {
	start := -1
	for i, line := range lines {
		switch {
		case strings.TrimSpace(line) == planHeading:
			start = i + 1
		case start >= 0 && strings.HasPrefix(line, "## "):
			return lines[start:i]
		}
	}
	if start < 0 {
		return nil
	}
	return lines[start:]
}

func parseSteps(lines []string) ([]Step, error) {
	var steps []Step
	seen := map[string]bool{}
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "- "):
			step, err := parseStep(line)
			if err != nil {
				return nil, err
			}
			if seen[step.ID] {
				return nil, fmt.Errorf("step %s is listed twice", step.ID)
			}
			seen[step.ID] = true
			steps = append(steps, step)
		case strings.HasPrefix(line, " ") && len(steps) > 0:
			outcome, err := parseOutcome(strings.TrimSpace(line))
			if err != nil {
				return nil, fmt.Errorf("step %s: %w", steps[len(steps)-1].ID, err)
			}
			if outcome != "" {
				steps[len(steps)-1].Outcome = outcome
			}
		}
	}
	return steps, nil
}

func parseStep(line string) (Step, error) {
	id, subject, ok := strings.Cut(strings.TrimPrefix(line, "- "), ": ")
	subject = strings.TrimSpace(subject)
	if !ok || !validStepID(id) || subject == "" {
		return Step{}, fmt.Errorf("plan line %q is not - S-<n>: <commit subject>", line)
	}
	return Step{ID: id, Subject: subject}, nil
}

// parseOutcome reads an indented "Outcome: dropped — <reason>" or "Outcome: verified — <reason>" line.
func parseOutcome(detail string) (string, error) {
	value, ok := strings.CutPrefix(detail, "Outcome:")
	if !ok {
		return "", nil
	}
	for _, outcome := range []string{OutcomeDropped, OutcomeVerified} {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(value), outcome); ok && strings.TrimSpace(strings.TrimLeft(rest, " —-:")) != "" {
			return outcome, nil
		}
	}
	return "", fmt.Errorf("outcome %q is not dropped or verified with a reason", strings.TrimSpace(value))
}

func validStepID(id string) bool {
	digits, ok := strings.CutPrefix(id, "S-")
	return ok && digits != "" && digits[0] != '0' && allDigits(digits)
}

// step returns the plan step with id, or false.
func (p WorkPlan) step(id string) (Step, bool) {
	for _, s := range p.Steps {
		if s.ID == id {
			return s, true
		}
	}
	return Step{}, false
}
