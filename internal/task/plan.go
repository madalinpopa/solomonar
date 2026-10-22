package task

import (
	"cmp"
	"fmt"
	"time"
)

// Plan adds a planned task under the next free ID and returns the ID and the commit subject.
func (r *Registry) Plan(title string, dependsOn []string, related string, now time.Time) (id, subject string, err error) {
	if err := cmp.Or(checkTitle(title), r.checkReferences(dependsOn, related)); err != nil {
		return "", "", err
	}
	id = nextTaskID(r)
	record := &Record{
		Title: title, Status: StatusPlanned, UpdatedAt: timestamp(now), DependsOn: dependsOn, RelatedTask: related,
	}
	r.Tasks[id] = record
	return id, record.Subject(id, fmt.Sprintf("plan %q", title)), nil
}

// Lookup returns the record of id, rejecting a malformed or unknown ID.
func (r *Registry) Lookup(id string) (*Record, error) {
	if err := checkTaskID(id); err != nil {
		return nil, err
	}
	record, ok := r.Tasks[id]
	if !ok {
		return nil, &Rejection{
			Code:  "task_not_found",
			Check: "the task exists in the registry",
			Found: id + " is not in the registry",
			Fix:   "solomonar task show",
		}
	}
	return record, nil
}

// Subject is the registry commit subject for a change to the task id (I section 8.5).
func (r *Record) Subject(id, summary string) string {
	state := string(r.Status)
	if r.Stage != "" {
		state += "/" + string(r.Stage)
	}
	return fmt.Sprintf("state(%s): %s, %s", id, state, summary)
}

func timestamp(now time.Time) string {
	return now.UTC().Truncate(time.Second).Format(time.RFC3339)
}
