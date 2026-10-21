package task

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

func (r *Registry) validate() error {
	for _, id := range slices.Sorted(maps.Keys(r.Tasks)) {
		if err := validateRecord(id, r.Tasks[id]); err != nil {
			return invalidRegistry("%s: %v", id, err)
		}
	}
	return nil
}

func validateRecord(id string, r *Record) error {
	if !validTaskID(id) {
		return errors.New("task ID is not W- followed by three or more digits")
	}
	if r == nil {
		return errors.New("record is null")
	}
	if err := validateValues(r); err != nil {
		return err
	}
	return validateFieldsForStatus(id, r)
}

func validateValues(r *Record) error {
	return cmp.Or(
		validateSHAs(r),
		validateReferences(r),
		validateAgents(r.ActiveAgents),
		validateBlockers(r.Blockers),
		validateNextAction(r.NextAction),
	)
}

func validateSHAs(r *Record) error {
	if r.Revision != "" && !fullSHA(r.Revision) {
		return fmt.Errorf("revision %q is not a full SHA", r.Revision)
	}
	if r.MergeCommit != "" && !fullSHA(r.MergeCommit) {
		return fmt.Errorf("merge_commit %q is not a full SHA", r.MergeCommit)
	}
	if r.SpecRef != "" {
		commit, path, _ := strings.Cut(r.SpecRef, ":")
		if !fullSHA(commit) || path == "" {
			return fmt.Errorf("spec_ref %q is not <full sha>:<path>", r.SpecRef)
		}
	}
	return nil
}

func validateReferences(r *Record) error {
	if r.Tier != nil && (*r.Tier < 0 || *r.Tier > 2) {
		return fmt.Errorf("tier %d is not 0, 1, or 2", *r.Tier)
	}
	if r.UpdatedAt != "" && !utcTimestamp(r.UpdatedAt) {
		return fmt.Errorf("updated_at %q is not an RFC 3339 UTC timestamp", r.UpdatedAt)
	}
	for _, dependency := range r.DependsOn {
		if !validTaskID(dependency) {
			return fmt.Errorf("depends_on %q is not a task ID", dependency)
		}
	}
	if r.RelatedTask != "" && !validTaskID(r.RelatedTask) {
		return fmt.Errorf("related_task %q is not a task ID", r.RelatedTask)
	}
	return nil
}

func validateAgents(agents []Agent) error {
	for _, agent := range agents {
		if agent.ID == "" || agent.Role == "" {
			return errors.New("active_agents entry needs an id and a role")
		}
	}
	return nil
}

func validateBlockers(blockers []Blocker) error {
	seen := map[string]bool{}
	for _, blocker := range blockers {
		if !validBlockerID(blocker.ID) {
			return fmt.Errorf("blocker ID %q is not B-<n>", blocker.ID)
		}
		if seen[blocker.ID] {
			return fmt.Errorf("blocker ID %s is not unique", blocker.ID)
		}
		seen[blocker.ID] = true
		if !slices.Contains(blockerKinds(), blocker.Kind) {
			return fmt.Errorf("blocker %s has unknown kind %q", blocker.ID, blocker.Kind)
		}
	}
	return nil
}

func validateNextAction(next *NextAction) error {
	if next != nil && (next.Actor == "" || next.Action == "") {
		return errors.New("next_action needs an actor and an action")
	}
	return nil
}

func validateFieldsForStatus(id string, r *Record) error {
	if err := requireFields(r, "title", "status", "updated_at"); err != nil {
		return err
	}
	if r.Stage != "" && !slices.Contains(stages(), r.Stage) {
		return fmt.Errorf("unknown stage %q", r.Stage)
	}
	switch r.Status {
	case StatusPlanned:
		return allowOnly(r, "title", "status", "updated_at", "depends_on", "related_task")
	case StatusReserved, StatusRunning, StatusBlocked, StatusPaused, StatusCancelled:
		return validateActive(id, r)
	case StatusDone:
		return validateDone(r)
	default:
		return fmt.Errorf("unknown status %q", r.Status)
	}
}

func validateActive(id string, r *Record) error {
	if err := requireFields(r, "owner", "branch", "stage", "next_action"); err != nil {
		return err
	}
	if r.Branch != "feature/"+id {
		return fmt.Errorf("branch %q is not feature/%s", r.Branch, id)
	}
	if r.Stage == StageComplete {
		return fmt.Errorf("stage complete is only for done tasks, not %s", r.Status)
	}
	if r.Stage.atOrAfter(StageImplementation) {
		if err := requireFields(r, "spec"); err != nil {
			return err
		}
	}
	if r.Stage.atOrAfter(StageSpecializedReview) {
		return requireFields(r, "revision", "tier")
	}
	return nil
}

func validateDone(r *Record) error {
	if r.Stage != StageComplete {
		return fmt.Errorf("stage %q is not complete", r.Stage)
	}
	if err := requireFields(r, "summary", "spec_ref", "merge_commit"); err != nil {
		return err
	}
	return allowOnly(r, "title", "status", "stage", "summary", "spec_ref", "merge_commit", "pr",
		"updated_at", "related_task")
}

func requireFields(r *Record, names ...string) error {
	present := presentFields(r)
	for _, name := range names {
		if !slices.Contains(present, name) {
			return fmt.Errorf("%s is required for status %q", name, r.Status)
		}
	}
	return nil
}

func allowOnly(r *Record, names ...string) error {
	for _, name := range presentFields(r) {
		if !slices.Contains(names, name) {
			return fmt.Errorf("%s is not allowed for status %s", name, r.Status)
		}
	}
	return nil
}

func presentFields(r *Record) []string {
	var present []string
	add := func(name string, set bool) {
		if set {
			present = append(present, name)
		}
	}
	add("title", r.Title != "")
	add("status", r.Status != "")
	add("stage", r.Stage != "")
	add("owner", r.Owner != "")
	add("branch", r.Branch != "")
	add("spec", r.Spec != "")
	add("revision", r.Revision != "")
	add("tier", r.Tier != nil)
	add("active_agents", len(r.ActiveAgents) > 0)
	add("blockers", len(r.Blockers) > 0)
	add("next_action", r.NextAction != nil)
	add("pr", r.PR != "")
	add("depends_on", len(r.DependsOn) > 0)
	add("related_task", r.RelatedTask != "")
	add("session_ref", r.SessionRef != "")
	add("summary", r.Summary != "")
	add("spec_ref", r.SpecRef != "")
	add("merge_commit", r.MergeCommit != "")
	add("updated_at", r.UpdatedAt != "")
	return present
}

func stages() []Stage {
	return []Stage{
		StageSetup, StageSpecification, StageImplementation, StageSpecializedReview,
		StageAdversarialReview, StageIntegration, StageComplete,
	}
}

func blockerKinds() []BlockerKind {
	return []BlockerKind{BlockerDecision, BlockerDependency, BlockerCheck, BlockerEnvironment}
}

func (s Stage) atOrAfter(other Stage) bool {
	return slices.Index(stages(), s) >= slices.Index(stages(), other)
}

func validTaskID(id string) bool {
	digits, ok := strings.CutPrefix(id, "W-")
	return ok && len(digits) >= 3 && allDigits(digits)
}

func validBlockerID(id string) bool {
	digits, ok := strings.CutPrefix(id, "B-")
	return ok && digits != "" && allDigits(digits)
}

func allDigits(s string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
}

// fullSHA accepts SHA-1 and SHA-256 object names.
func fullSHA(s string) bool {
	return (len(s) == 40 || len(s) == 64) &&
		!strings.ContainsFunc(s, func(r rune) bool { return (r < '0' || r > '9') && (r < 'a' || r > 'f') })
}

func utcTimestamp(s string) bool {
	_, err := time.Parse(time.RFC3339, s)
	return err == nil && strings.HasSuffix(s, "Z")
}
