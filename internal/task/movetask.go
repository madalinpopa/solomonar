package task

import (
	"cmp"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/madalinpopa/solomonar/internal/git"
)

// MoveResult is the outcome of a task move for rendering.
type MoveResult struct {
	From, To  Position
	Changed   []string
	SpecFound string
	Subject   string
	Warnings  []Warning
}

// Position is a task's status and stage.
type Position struct {
	Status Status `json:"status"`
	Stage  Stage  `json:"stage"`
}

// MoveTask applies m to the task id for the owner as, storing the record only when every check passes.
func (r *Registry) MoveTask(ctx context.Context, repo *git.Repo, id, as string, m Move, roles []string, now time.Time) (MoveResult, error) {
	before, err := r.Lookup(id)
	if err != nil {
		return MoveResult{}, err
	}
	// A compact done record has no owner, so done_final comes before the owner check.
	if before.Status == StatusDone {
		return MoveResult{}, doneFinal(cmp.Or(Status(m.Status), StatusDone))
	}
	if err := before.CheckOwner(id, as); err != nil {
		return MoveResult{}, err
	}
	after := *before
	changed, err := after.Move(m, roles, now)
	if err != nil {
		return MoveResult{}, err
	}
	specFound, err := CheckFacts(ctx, repo, id, before, &after, m)
	if err != nil {
		return MoveResult{}, err
	}
	result := MoveResult{From: before.position(), To: after.position(), Changed: changed, SpecFound: specFound}
	if len(changed) == 0 {
		return result, nil
	}
	if before.Status == StatusCancelled && after.Status != StatusCancelled {
		result.Warnings = []Warning{{
			Code:    "cancelled_resumed",
			Message: "resuming a cancelled task requires the user's explicit instruction (rules section 7)",
		}}
	}
	result.Subject = after.moveSubject(id, changed)
	r.Tasks[id] = &after
	return result, nil
}

func (r *Record) position() Position {
	return Position{Status: r.Status, Stage: r.Stage}
}

// moveSubject names each changed field with its new value (I section 8.5).
func (r *Record) moveSubject(id string, changed []string) string {
	parts := make([]string, len(changed))
	for i, field := range changed {
		parts[i] = r.subjectPart(field)
	}
	return r.Subject(id, strings.Join(parts, ", "))
}

func (r *Record) subjectPart(field string) string {
	switch field {
	case "status":
		return "status " + string(r.Status)
	case "stage":
		return "stage " + string(r.Stage)
	case "spec":
		return "spec " + r.Spec
	case "revision":
		return "revision " + r.Revision[:min(7, len(r.Revision))]
	case "tier":
		return fmt.Sprintf("tier %d", *r.Tier)
	case "active_agents":
		if len(r.ActiveAgents) == 0 {
			return "agents none"
		}
		ids := make([]string, len(r.ActiveAgents))
		for i, agent := range r.ActiveAgents {
			ids[i] = agent.ID
		}
		return "agents " + strings.Join(ids, " ")
	case "pr":
		return "pr " + r.PR
	case "next_action":
		return "next " + r.NextAction.Actor
	default:
		return field
	}
}
