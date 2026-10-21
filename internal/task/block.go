package task

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"time"
)

// Block holds the raw task block flags; an empty string means the flag was not given.
type Block struct{ Kind, Summary, Ref, Status, Next string }

// BlockTask adds a blocker to the task id for the owner as, with --status and --next in one write.
func (r *Registry) BlockTask(id, as string, b Block, now time.Time) (MoveResult, error) {
	before, err := r.writable(id, as, b.Status)
	if err != nil {
		return MoveResult{}, err
	}
	blocker, err := newBlocker(b.Kind, b.Summary, b.Ref)
	if err != nil {
		return MoveResult{}, err
	}
	if b.Status != "" && Status(b.Status) != StatusBlocked {
		return MoveResult{}, invalidValue("--status on task block takes blocked", b.Status,
			"pass --status blocked, or use task move")
	}
	after := *before
	blocker.ID = after.issueBlockerID()
	after.Blockers = append(slices.Clone(before.Blockers), blocker)
	return r.storeBlockers(id, before, &after, Move{Status: b.Status, Next: b.Next}, now,
		fmt.Sprintf("block %s %s", blocker.ID, blocker.Kind))
}

// UnblockTask clears the open blocker blockerID of the task id for the owner as.
func (r *Registry) UnblockTask(id, as, blockerID, status, next string, now time.Time) (MoveResult, error) {
	before, err := r.writable(id, as, status)
	if err != nil {
		return MoveResult{}, err
	}
	index := slices.IndexFunc(before.Blockers, func(b Blocker) bool { return b.ID == blockerID })
	if index < 0 {
		return MoveResult{}, &Rejection{
			Code:  "blocker_not_found",
			Check: "the blocker is open on the task",
			Found: fmt.Sprintf("%s is not an open blocker of %s", blockerID, id),
			Fix:   "solomonar task show " + id,
		}
	}
	after := *before
	after.Blockers = slices.Delete(slices.Clone(before.Blockers), index, index+1)
	result, err := r.storeBlockers(id, before, &after, Move{Status: status, Next: next}, now, "unblock "+blockerID)
	// Unblocking never picks the status for the caller (author decision, stage 6.1).
	if rejection, ok := errors.AsType[*Rejection](err); ok && rejection.Code == "blockers_required" {
		rejection.Fix = "pass --status " + string(priorStatus(before.Stage))
	}
	return result, err
}

// writable returns the record id when it is not done and as owns it.
func (r *Registry) writable(id, as, status string) (*Record, error) {
	record, err := r.Lookup(id)
	if err != nil {
		return nil, err
	}
	if record.Status == StatusDone {
		return nil, doneFinal(cmp.Or(Status(status), StatusDone))
	}
	if err := record.CheckOwner(id, as); err != nil {
		return nil, err
	}
	return record, nil
}

// storeBlockers applies m to after, whose blockers already changed, and stores it.
func (r *Registry) storeBlockers(id string, before, after *Record, m Move, now time.Time, summary string) (MoveResult, error) {
	if _, err := after.Move(m, nil, now); err != nil {
		return MoveResult{}, err
	}
	after.UpdatedAt = timestamp(now)
	r.Tasks[id] = after
	return MoveResult{
		From:    before.position(),
		To:      after.position(),
		Changed: changedFields(before, after),
		Subject: after.Subject(id, summary),
	}, nil
}
