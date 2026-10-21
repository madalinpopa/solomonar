package task

import (
	"fmt"
	"time"
)

// HandoffTask gives the task id to a new owner on the user's instruction (UC-19).
func (r *Registry) HandoffTask(id, owner, next string, now time.Time) (MoveResult, error) {
	before, err := r.Lookup(id)
	if err != nil {
		return MoveResult{}, err
	}
	if err := r.checkHandoff(id, before, owner); err != nil {
		return MoveResult{}, err
	}
	after := *before
	after.Owner = owner
	if next != "" {
		if _, err := after.Move(Move{Next: next}, nil, now); err != nil {
			return MoveResult{}, err
		}
	} else if before.NextAction != nil && before.NextAction.Actor == before.Owner {
		after.NextAction = &NextAction{Actor: owner, Action: before.NextAction.Action}
	}
	after.UpdatedAt = timestamp(now)
	r.Tasks[id] = &after
	return MoveResult{
		From:    before.position(),
		To:      after.position(),
		Changed: changedFields(before, &after),
		Subject: after.Subject(id, fmt.Sprintf("handoff %s -> %s", before.Owner, owner)),
	}, nil
}

func (r *Registry) checkHandoff(id string, before *Record, owner string) error {
	if before.Status == StatusDone {
		return doneFinal(StatusDone)
	}
	if before.Status == StatusPlanned {
		return &Rejection{
			Code:  "not_registered",
			Check: "the task is registered",
			Found: id + " is planned",
			Fix:   fmt.Sprintf("solomonar task register %s --title <title> --owner %s", id, owner),
		}
	}
	if err := checkOwnerID(owner); err != nil {
		return err
	}
	if owner == before.Owner {
		return &Rejection{
			Code:  "handoff_same_owner",
			Check: "the new owner differs from the current owner",
			Found: fmt.Sprintf("%s is owned by %s", id, owner),
		}
	}
	return r.checkOwnerFree(owner)
}
