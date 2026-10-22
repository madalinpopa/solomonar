package task

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Registration holds the raw task register flags; an empty string means the flag was not given.
type Registration struct {
	ID, Title, Owner, Related, Next string
	DependsOn                       []string
}

// Register reserves a task for an orchestrator and returns its ID, the commit subject, and warnings.
func (r *Registry) Register(reg Registration, now time.Time) (id, subject string, warnings []Warning, err error) {
	id, planned, err := r.claim(reg.ID)
	if err != nil {
		return "", "", nil, err
	}
	if err := cmp.Or(r.checkOwner(reg.Owner), checkTitle(reg.Title)); err != nil {
		return "", "", nil, err
	}
	dependsOn := planned.DependsOn
	if len(reg.DependsOn) > 0 {
		dependsOn = reg.DependsOn
	}
	related := cmp.Or(reg.Related, planned.RelatedTask)
	if err := r.checkReferences(dependsOn, related); err != nil {
		return "", "", nil, err
	}
	next, err := registrationNext(reg.Next, reg.Owner, id)
	if err != nil {
		return "", "", nil, err
	}
	record := &Record{
		Title: reg.Title, Status: StatusReserved, Stage: StageSetup, Owner: reg.Owner, Branch: "feature/" + id,
		NextAction: next, DependsOn: dependsOn, RelatedTask: related, UpdatedAt: timestamp(now),
	}
	warnings = r.registrationWarnings(planned, record)
	r.Tasks[id] = record
	return id, record.Subject(id, "registered by "+reg.Owner), warnings, nil
}

// claim returns the ID to register and its planned record, empty when allocating a new ID.
func (r *Registry) claim(id string) (string, *Record, error) {
	if id == "" {
		return nextTaskID(r), &Record{}, nil
	}
	planned, err := r.Lookup(id)
	if err != nil {
		return "", nil, err
	}
	if planned.Status != StatusPlanned {
		return "", nil, &Rejection{
			Code:  "not_planned",
			Check: "task register claims only a planned task",
			Found: fmt.Sprintf("%s is %s", id, planned.Status),
			Fix:   fmt.Sprintf("solomonar task show %s; resume the task instead", id),
		}
	}
	return id, planned, nil
}

func (r *Registry) checkOwner(owner string) error {
	if err := checkOwnerID(owner); err != nil {
		return err
	}
	return r.checkOwnerFree(owner)
}

func checkOwnerID(owner string) error {
	if !validAgentID(owner) {
		return invalidValue("--owner takes an agent ID like [a-z]+-[0-9]+", owner, "pass --owner orchestrator-1")
	}
	return nil
}

// checkOwnerFree rejects an owner that already owns an active task.
func (r *Registry) checkOwnerFree(owner string) error {
	for _, id := range r.IDs() {
		record := r.Tasks[id]
		if record.Owner == owner && ownsActively(record.Status) {
			return &Rejection{
				Code:  "owner_in_use",
				Check: "an orchestrator owns exactly one active task",
				Found: fmt.Sprintf("%s owns %s (%s)", owner, id, record.Status),
				Fix:   "choose a new owner ID",
			}
		}
	}
	return nil
}

func ownsActively(status Status) bool {
	return slices.Contains([]Status{StatusReserved, StatusRunning, StatusBlocked, StatusPaused}, status)
}

func checkTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return invalidValue("--title takes a non-empty title", title, `pass --title "Login"`)
	}
	return nil
}

func (r *Registry) checkReferences(dependsOn []string, related string) error {
	references := slices.Clone(dependsOn)
	if related != "" {
		references = append(references, related)
	}
	for _, reference := range references {
		if _, err := r.Lookup(reference); err != nil {
			return err
		}
	}
	return nil
}

// registrationNext limits the actor to user or the owner, since no agents exist yet (I section 8.3).
func registrationNext(value, owner, id string) (*NextAction, error) {
	if value == "" {
		return &NextAction{Actor: owner, Action: fmt.Sprintf("Create feature/%s and its worktree", id)}, nil
	}
	next, err := parseNextAction(value)
	if err != nil {
		return nil, err
	}
	if next.Actor != "user" && next.Actor != owner {
		return nil, invalidValue("--next actor is user or the owner at registration", value,
			fmt.Sprintf(`pass --next "%s:<action>"`, owner))
	}
	return next, nil
}

func (r *Registry) registrationWarnings(planned, record *Record) []Warning {
	var warnings []Warning
	if planned.Title != "" && planned.Title != record.Title {
		warnings = append(warnings, Warning{Code: "title_changed", Message: fmt.Sprintf("title changed from %q", planned.Title)})
	}
	for _, dependency := range record.DependsOn {
		if r.Tasks[dependency].Status != StatusDone {
			warnings = append(warnings, Warning{
				Code:    "dependency_not_done",
				Message: dependency + " is not done; register, then block with task block --kind dependency",
			})
		}
	}
	return warnings
}

func validAgentID(id string) bool {
	name, number, ok := strings.Cut(id, "-")
	return ok && name != "" && number != "" && allDigits(number) &&
		!strings.ContainsFunc(name, func(r rune) bool { return r < 'a' || r > 'z' })
}
