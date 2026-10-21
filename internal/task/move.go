package task

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

// Move holds the raw task move flags; an empty string means the flag was not given.
type Move struct {
	Status, Stage, Revision, Spec, Tier, PR, Next string
	Agents                                        []string // repeated --agent; ["none"] clears
	ResumeCancelled                               bool
}

// CheckOwner rejects a write by anyone but the record's owner.
func (r *Record) CheckOwner(id, as string) error {
	if as == r.Owner {
		return nil
	}
	return &Rejection{
		Code:  "owner_mismatch",
		Check: "only the owner writes a task's record",
		Found: fmt.Sprintf("owner is %s; acting as %s", r.Owner, as),
		Fix: fmt.Sprintf("act as %s, or ask the user for an explicit handoff: solomonar task handoff %s --owner %s",
			r.Owner, id, as),
	}
}

// Move applies m to r when every check passes and returns the JSON names of the changed fields.
func (r *Record) Move(m Move, roles []string, now time.Time) (changed []string, err error) {
	after, err := m.apply(r, roles)
	if err != nil {
		return nil, err
	}
	err = cmp.Or(
		checkTransition(r, after, m.ResumeCancelled),
		checkFieldRules(r, after, m),
		checkRequiredFields(r, after, m),
	)
	if err != nil {
		return nil, err
	}
	changed = changedFields(r, after)
	if len(changed) > 0 {
		*r = *after
		r.UpdatedAt = timestamp(now)
	}
	return changed, nil
}

func (m Move) apply(r *Record, roles []string) (*Record, error) {
	after := *r
	if err := cmp.Or(m.applyStatus(&after), m.applyStage(&after), m.applyCommit(&after), m.applyWork(&after, roles)); err != nil {
		return nil, err
	}
	return &after, nil
}

func (m Move) applyStatus(r *Record) error {
	if m.Status == "" {
		return nil
	}
	status := Status(m.Status)
	switch {
	case status == StatusDone:
		return &Rejection{
			Code:  "transition_not_allowed",
			Check: "status change must follow I §8.1",
			Found: fmt.Sprintf("%s -> %s", r.Status, status),
			Fix:   "use task done",
		}
	case !slices.Contains(statuses(), status):
		return invalidValue("--status takes a status of I §8.1", m.Status, "allowed: "+join(allowedStatuses(r)))
	}
	r.Status = status
	return nil
}

func (m Move) applyStage(r *Record) error {
	if m.Stage == "" {
		return nil
	}
	stage := Stage(m.Stage)
	switch {
	case stage == StageComplete:
		return &Rejection{
			Code:  "stage_not_allowed",
			Check: "stage complete is written by task done",
			Found: fmt.Sprintf("%s -> %s", r.Stage, stage),
			Fix:   "use task done",
		}
	case !slices.Contains(stages(), stage):
		return invalidValue("--stage takes a stage of I §8.2", m.Stage, "allowed: "+join(stageTargets(r.Stage, r.Tier)))
	}
	r.Stage = stage
	return nil
}

func (m Move) applyCommit(r *Record) error {
	if m.Revision != "" {
		if err := checkRevision(m.Revision); err != nil {
			return err
		}
		r.Revision = m.Revision
	}
	if m.Tier != "" {
		tier, err := parseTier(m.Tier)
		if err != nil {
			return err
		}
		r.Tier = tier
	}
	if m.PR != "" {
		if err := checkPR(m.PR); err != nil {
			return err
		}
		r.PR = m.PR
	}
	return nil
}

func (m Move) applyWork(r *Record, roles []string) error {
	if m.Spec != "" {
		if err := checkSpec(m.Spec); err != nil {
			return err
		}
		r.Spec = m.Spec
	}
	if len(m.Agents) > 0 {
		agents, _, err := parseAgents(m.Agents, roles)
		if err != nil {
			return err
		}
		r.ActiveAgents = agents
	}
	if m.Next != "" {
		next, err := parseNextAction(m.Next)
		if err != nil {
			return err
		}
		if err := checkActor(r, next.Actor, m.Next); err != nil {
			return err
		}
		r.NextAction = next
	}
	return nil
}

// checkActor accepts user, the owner, or an agent active after the move (I §8.3).
func checkActor(r *Record, actor, value string) error {
	actors := []string{"user", r.Owner}
	for _, agent := range r.ActiveAgents {
		actors = append(actors, agent.ID)
	}
	if slices.Contains(actors, actor) {
		return nil
	}
	return invalidValue("--next actor is user, the owner, or an active agent", value, "allowed actors: "+join(actors))
}

// checkFieldRules applies the field rules of I §8.2; --agent and --next need no rule, done is final.
func checkFieldRules(before, after *Record, m Move) error {
	found := func(flag string) string { return flag + " at " + stageChange(before.Stage, after.Stage) }
	switch {
	case m.Revision != "" && !acceptsRevision(before.Stage) && !acceptsRevision(after.Stage):
		return fieldNotAllowed("--revision is accepted when the current or resulting stage is implementation or integration",
			found("--revision"), "pass --revision at implementation or integration, or with a move to either")
	case m.Tier != "" && m.Revision == "" && !tierReaches(before.Tier, before.Stage):
		return fieldNotAllowed("--tier is accepted with --revision or when the recorded tier allows the current stage",
			found("--tier"), "pass --tier with --revision <full-sha>")
	case m.Spec != "" && !after.Stage.atOrAfter(StageSpecification):
		return fieldNotAllowed("--spec is accepted from specification onward", found("--spec"),
			"pass --spec at specification or later, or with --stage specification")
	case m.PR != "" && after.Stage != StageIntegration:
		return fieldNotAllowed("--pr is accepted in integration", found("--pr"),
			"pass --pr at integration or with --stage integration")
	}
	return nil
}

func fieldNotAllowed(rule, found, fix string) *Rejection {
	return &Rejection{Code: "field_not_allowed", Check: rule + " (I §8.2)", Found: found, Fix: fix}
}

func acceptsRevision(stage Stage) bool {
	return stage == StageImplementation || stage == StageIntegration
}

// tierReaches reports whether a recorded tier has a stage move into stage.
func tierReaches(tier *int, stage Stage) bool {
	return tier != nil && slices.ContainsFunc(stageMoves(), func(m stageMove) bool {
		return m.to == stage && m.allows(tier)
	})
}

func checkRequiredFields(before, after *Record, m Move) error {
	switch {
	case after.Stage.atOrAfter(StageImplementation) && after.Spec == "":
		return required("spec_required", "a stage from implementation onward needs a recorded spec",
			fmt.Sprintf("%s without spec", after.Stage), "pass --spec <worktree-relative path>")
	case after.Stage.atOrAfter(StageSpecializedReview) && after.Revision == "":
		return required("revision_required", "a review or integration stage needs a recorded revision",
			fmt.Sprintf("%s without revision", after.Stage), "pass --revision <full-sha>")
	case before.Status != after.Status && (after.Status == StatusPaused || after.Status == StatusCancelled) && m.Next == "":
		return required("next_required", "pausing or canceling a task needs --next",
			fmt.Sprintf("%s -> %s", before.Status, after.Status), "pass --next <actor>:<action>")
	case after.Status == StatusBlocked && len(after.Blockers) == 0:
		return required("blockers_required", "a blocked task needs an open blocker", "no open blocker",
			"task block first, or pass --status blocked to it")
	}
	return nil
}

func required(code, check, found, fix string) *Rejection {
	return &Rejection{Code: code, Check: check, Found: found, Fix: fix}
}

// changedFields lists the fields that differ in record order, except last_blocker and updated_at.
func changedFields(before, after *Record) []string {
	changed := []string{}
	add := func(name string, differs bool) {
		if differs {
			changed = append(changed, name)
		}
	}
	add("title", before.Title != after.Title)
	add("status", before.Status != after.Status)
	add("stage", before.Stage != after.Stage)
	add("owner", before.Owner != after.Owner)
	add("branch", before.Branch != after.Branch)
	add("spec", before.Spec != after.Spec)
	add("revision", before.Revision != after.Revision)
	add("tier", !samePointee(before.Tier, after.Tier))
	add("active_agents", !slices.Equal(before.ActiveAgents, after.ActiveAgents))
	add("blockers", !slices.Equal(before.Blockers, after.Blockers))
	add("next_action", !samePointee(before.NextAction, after.NextAction))
	add("pr", before.PR != after.PR)
	add("depends_on", !slices.Equal(before.DependsOn, after.DependsOn))
	add("related_task", before.RelatedTask != after.RelatedTask)
	add("session_ref", before.SessionRef != after.SessionRef)
	add("summary", before.Summary != after.Summary)
	add("spec_ref", before.SpecRef != after.SpecRef)
	add("merge_commit", before.MergeCommit != after.MergeCommit)
	return changed
}

func samePointee[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func stageChange(from, to Stage) string {
	if from == to {
		return string(from)
	}
	return fmt.Sprintf("%s -> %s", from, to)
}
