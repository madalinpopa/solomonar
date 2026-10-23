package task

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/madalinpopa/solomonar/internal/git"
)

// Plan approval states reported by task status (I section 9.1).
const (
	PlanApproved = "approved"
	PlanInReview = "in_review"
	PlanMissing  = "missing"
	PlanInvalid  = "invalid"
)

// Step states after reconciliation.
const (
	StepCommitted = "committed"
	StepRemaining = "remaining"
)

// PlanFacts reconcile the approved plan with the checkpoint commits since the base (CLI-D24).
type PlanFacts struct {
	Approval      string        `json:"approval"`
	Problem       string        `json:"problem,omitempty"`
	At            string        `json:"at"`
	Steps         []StepFacts   `json:"steps"`
	OtherCommits  int           `json:"other_commits"`
	Discrepancies []Discrepancy `json:"discrepancies"`
}

// StepFacts is a plan step and the commit that checkpoints it, if any.
type StepFacts struct {
	ID      string  `json:"id"`
	Subject string  `json:"subject"`
	State   string  `json:"state"`
	Commit  *string `json:"commit"`
}

// Discrepancy is a commit whose checkpoint claim needs inspection.
type Discrepancy struct {
	Code    string `json:"code"`
	Commit  string `json:"commit"`
	Message string `json:"message"`
}

// commit is one non-merge commit on the task branch since the base.
type commit struct {
	sha, subject string
	steps        []string
	empty        bool
}

// InspectPlan reads the work spec committed at the commit at and reconciles its steps with the
// commits in base..at. It returns nil when the record has no spec or base to reconcile.
func InspectPlan(ctx context.Context, repo *git.Repo, id string, r *Record, at string) (*PlanFacts, error) {
	if r.Spec == "" || r.Base == "" || at == "" {
		return nil, nil
	}
	facts := &PlanFacts{At: at, Steps: []StepFacts{}, Discrepancies: []Discrepancy{}}
	content, found, err := readBlob(ctx, repo, at+":"+r.Spec)
	if err != nil {
		return nil, err
	}
	if !found {
		facts.Approval = PlanMissing
		return facts, nil
	}
	plan, problem := parsePlan(content)
	if problem != "" {
		facts.Approval, facts.Problem = PlanInvalid, problem
		return facts, nil
	}
	facts.Approval = approvalState(plan)
	commits, err := listCommits(ctx, repo, r.Base, at)
	if err != nil {
		return nil, err
	}
	reconcile(facts, id, plan, commits)
	return facts, nil
}

// parsePlan reports a malformed work spec as a problem to show, not an error to return.
func parsePlan(content string) (WorkPlan, string) {
	plan, err := ParseWorkPlan(content)
	if err != nil {
		return WorkPlan{}, err.Error()
	}
	return plan, ""
}

func approvalState(plan WorkPlan) string {
	switch {
	case plan.Approval == ApprovalApproved && len(plan.Steps) > 0:
		return PlanApproved
	case plan.Approval == ApprovalInReview:
		return PlanInReview
	default:
		return PlanMissing
	}
}

func reconcile(facts *PlanFacts, id string, plan WorkPlan, commits []commit) {
	committed := map[string]string{}
	bySubject := map[string]string{}
	for _, s := range plan.Steps {
		bySubject[s.Subject] = s.ID
	}
	add := func(code string, c commit, format string, args ...any) {
		facts.Discrepancies = append(facts.Discrepancies,
			Discrepancy{Code: code, Commit: c.sha, Message: fmt.Sprintf("%.8s ", c.sha) + fmt.Sprintf(format, args...)})
	}
	for _, c := range commits {
		if len(c.steps) == 0 {
			if step, ok := bySubject[c.subject]; ok {
				add("step_untagged", c, "matches the subject of %s but has no Step trailer", step)
			} else {
				facts.OtherCommits++
			}
			continue
		}
		for _, claim := range c.steps {
			task, stepID, _ := strings.Cut(claim, "/")
			step, inPlan := plan.step(stepID)
			switch {
			case task != id:
				add("step_foreign", c, "claims %s, a step of another task", claim)
			case !inPlan:
				add("step_unknown", c, "claims %s, which the plan does not list", claim)
			case c.subject != step.Subject:
				add("step_subject_mismatch", c, "claims %s but its subject differs from the planned one", stepID)
			case c.empty:
				add("step_empty", c, "claims %s but changes nothing", stepID)
			case committed[stepID] != "":
				add("step_duplicate", c, "claims %s, already checkpointed by %.8s", stepID, committed[stepID])
			default:
				committed[stepID] = c.sha
			}
		}
	}
	for _, s := range plan.Steps {
		step := StepFacts{ID: s.ID, Subject: s.Subject, State: StepRemaining}
		switch {
		case committed[s.ID] != "":
			step.State, step.Commit = StepCommitted, new(committed[s.ID])
		case s.Outcome != "":
			step.State = s.Outcome
		}
		facts.Steps = append(facts.Steps, step)
	}
}

// Remaining lists the steps with neither a checkpoint commit nor an outcome.
func (p *PlanFacts) Remaining() []string {
	var ids []string
	for _, s := range p.Steps {
		if s.State == StepRemaining {
			ids = append(ids, s.ID)
		}
	}
	return ids
}

const (
	fieldSep  = "\x1f"
	commitSep = "\x1e"
)

// listCommits reads the non-merge commits of base..tip, oldest first, with their Step trailers.
func listCommits(ctx context.Context, repo *git.Repo, base, tip string) ([]commit, error) {
	format := commitSep + "%H" + fieldSep + "%s" + fieldSep + "%(trailers:key=Step,valueonly,separator=%x1f)" + fieldSep
	out, err := repo.Run(ctx, "log", "--no-merges", "--reverse", "--shortstat", "--format="+format, base+".."+tip)
	if err != nil {
		return nil, fmt.Errorf("list commits of %.8s..%.8s: %w", base, tip, err)
	}
	var commits []commit
	for record := range strings.SplitSeq(out, commitSep) {
		fields := strings.Split(record, fieldSep)
		if len(fields) < 3 {
			continue
		}
		stat := fields[len(fields)-1]
		c := commit{sha: fields[0], subject: fields[1], empty: !strings.Contains(stat, "changed")}
		for _, step := range fields[2 : len(fields)-1] {
			if step = strings.TrimSpace(step); step != "" {
				c.steps = append(c.steps, step)
			}
		}
		commits = append(commits, c)
	}
	return commits, nil
}

// readBlob reads object as a blob; found is false when it does not exist or is not a blob.
func readBlob(ctx context.Context, repo *git.Repo, object string) (content string, found bool, err error) {
	isFile, err := isBlob(ctx, repo, object)
	if err != nil || !isFile {
		return "", false, err
	}
	content, err = repo.Run(ctx, "cat-file", "blob", object)
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", object, err)
	}
	return content, true, nil
}

// branchTip returns the commit refs/heads/<branch> points to, or "" when the branch does not exist.
func branchTip(ctx context.Context, repo *git.Repo, branch string) (string, error) {
	tip, err := repo.Run(ctx, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if gitErr, ok := errors.AsType[*git.Error](err); ok && gitErr.ExitCode == 1 && ctx.Err() == nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("check branch %s: %w", branch, err)
	}
	return tip, nil
}

// checkApprovedPlan is the implementation entry gate: the spec committed at the commit at
// has Status: Approved and at least one step (CLI-D23).
func checkApprovedPlan(ctx context.Context, repo *git.Repo, id, spec, at string) error {
	content, found, err := readBlob(ctx, repo, at+":"+spec)
	if err != nil {
		return err
	}
	fix := "record the user's approval as Status: Approved below the title and commit the work spec"
	if !found {
		return &Rejection{
			Code:  "plan_not_approved",
			Check: "the work spec committed on feature/" + id + " has an approved implementation plan",
			Found: fmt.Sprintf("%s is not committed at %.8s", spec, at),
			Fix:   "commit the work spec in the worktree",
		}
	}
	plan, err := ParseWorkPlan(content)
	if err != nil {
		return &Rejection{
			Code:  "plan_invalid",
			Check: "the work spec's Status line and implementation plan follow the work-spec template",
			Found: err.Error(),
			Fix:   "correct the work spec and commit it",
		}
	}
	switch {
	case plan.Approval != ApprovalApproved:
		return &Rejection{
			Code:  "plan_not_approved",
			Check: "the work spec committed on feature/" + id + " has an approved implementation plan",
			Found: fmt.Sprintf("status is %q at %.8s", plan.Approval, at),
			Fix:   fix,
		}
	case len(plan.Steps) == 0:
		return &Rejection{
			Code:  "plan_invalid",
			Check: "the work spec's Status line and implementation plan follow the work-spec template",
			Found: "## Implementation plan lists no step",
			Fix:   "list each step as - S-<n>: <commit subject>, then ask the user to approve the plan",
		}
	}
	return nil
}

// checkCandidate is the review entry gate: a clean branch tip with every plan step checkpointed.
func checkCandidate(ctx context.Context, repo *git.Repo, id string, r *Record) error {
	branch := "feature/" + id
	if r.Base == "" {
		return baseMissing(id, r.Owner)
	}
	tip, err := branchTip(ctx, repo, branch)
	if err != nil {
		return err
	}
	if tip != r.Revision {
		return &Rejection{
			Code:  "candidate_not_tip",
			Check: "the candidate is the tip of " + branch,
			Found: fmt.Sprintf("revision %.8s; %s is at %.8s", r.Revision, branch, tip),
			Fix:   "pass --revision with the full SHA of the branch tip",
		}
	}
	if err := checkCleanWorktree(ctx, repo, branch); err != nil {
		return err
	}
	if err := checkApprovedPlan(ctx, repo, id, r.Spec, r.Revision); err != nil {
		return err
	}
	facts, err := InspectPlan(ctx, repo, id, r, r.Revision)
	if err != nil {
		return err
	}
	if n := len(facts.Discrepancies); n > 0 {
		return &Rejection{
			Code:  "checkpoint_discrepancy",
			Check: "every commit that claims a step checkpoints it (CLI-D24)",
			Found: fmt.Sprintf("%d %s; first: %s", n, plural(n, "discrepancy", "discrepancies"), facts.Discrepancies[0].Message),
			Fix:   "inspect with solomonar task status " + id + "; correct the commits or the plan",
		}
	}
	if remaining := facts.Remaining(); len(remaining) > 0 {
		return &Rejection{
			Code:  "plan_incomplete",
			Check: "every plan step has a checkpoint commit or an outcome",
			Found: strings.Join(remaining, ", ") + " " + plural(len(remaining), "has", "have") + " no checkpoint commit",
			Fix:   "delegate the remaining steps, or record an outcome in the plan",
		}
	}
	return nil
}

func checkCleanWorktree(ctx context.Context, repo *git.Repo, branch string) error {
	worktree, err := Worktree(ctx, repo, branch)
	if err != nil || worktree == "" {
		return err
	}
	status, err := (&git.Repo{Dir: worktree, Trace: repo.Trace}).Run(ctx, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("read status of %s: %w", worktree, err)
	}
	if status == "" {
		return nil
	}
	n := len(strings.Split(status, "\n"))
	return &Rejection{
		Code:  "candidate_dirty",
		Check: "the " + branch + " worktree has no uncommitted change",
		Found: fmt.Sprintf("%d uncommitted %s", n, plural(n, "path", "paths")),
		Fix:   "commit the work as a checkpoint, or preserve it outside the candidate (rules section 10)",
	}
}
