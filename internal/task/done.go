package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/madalinpopa/solomonar/internal/git"
)

// Done holds the raw task done flags.
type Done struct{ Merge, SpecRef, Summary string }

// DoneTask replaces the task id with its compact record once its merge into main checks out (UC-20).
func (r *Registry) DoneTask(ctx context.Context, repo *git.Repo, id, as, main string, d Done, now time.Time) (MoveResult, error) {
	before, err := r.writable(id, as, "")
	if err != nil {
		return MoveResult{}, err
	}
	if err := checkDoneState(id, before); err != nil {
		return MoveResult{}, err
	}
	summary, err := parseDone(d)
	if err != nil {
		return MoveResult{}, err
	}
	if err := checkMerge(ctx, repo, main, d.Merge, before.Revision); err != nil {
		return MoveResult{}, err
	}
	if err := checkSpecRef(ctx, repo, id, d.Merge, d.SpecRef, before.Spec); err != nil {
		return MoveResult{}, err
	}
	after := &Record{
		Title:       before.Title,
		Status:      StatusDone,
		Stage:       StageComplete,
		Summary:     summary,
		SpecRef:     d.SpecRef,
		MergeCommit: d.Merge,
		PR:          before.PR,
		RelatedTask: before.RelatedTask,
		UpdatedAt:   timestamp(now),
	}
	r.Tasks[id] = after
	return MoveResult{
		From:    before.position(),
		To:      after.position(),
		Subject: after.Subject(id, "merged "+d.Merge[:7]),
	}, nil
}

func checkDoneState(id string, r *Record) error {
	if r.Status != StatusRunning || r.Stage != StageIntegration {
		return &Rejection{
			Code:  "transition_not_allowed",
			Check: "task done needs running/integration",
			Found: fmt.Sprintf("%s is %s/%s", id, r.Status, r.Stage),
			Fix:   "solomonar task status " + id,
		}
	}
	if len(r.Blockers) == 0 {
		return nil
	}
	ids := make([]string, len(r.Blockers))
	for i, blocker := range r.Blockers {
		ids[i] = blocker.ID
	}
	return &Rejection{
		Code:  "blockers_open",
		Check: "task done needs no open blocker",
		Found: fmt.Sprintf("%s has open blockers %s", id, strings.Join(ids, ", ")),
		Fix:   fmt.Sprintf("solomonar task unblock %s <blocker-id>", id),
	}
}

// parseDone checks the flag values and returns the trimmed summary.
func parseDone(d Done) (string, error) {
	summary := strings.TrimSpace(d.Summary)
	if summary == "" || utf8.RuneCountInString(summary) > maxActionRunes {
		return "", invalidValue("--summary takes 1 to 120 characters", d.Summary, `pass --summary "<one sentence>"`)
	}
	if !fullSHA(d.Merge) {
		return "", invalidValue("--merge takes a full commit SHA", d.Merge, "git rev-parse "+d.Merge)
	}
	commit, path, _ := strings.Cut(d.SpecRef, ":")
	if !fullSHA(commit) || !cleanRelativePath(path) {
		return "", invalidValue("--spec-ref takes <full SHA>:<path>", d.SpecRef,
			`copy it from the PR "Final specification" line`)
	}
	return summary, nil
}

func checkMerge(ctx context.Context, repo *git.Repo, main, merge, revision string) error {
	onMain, err := isAncestor(ctx, repo, merge, main)
	if err != nil {
		return err
	}
	if !onMain {
		return &Rejection{
			Code:  "merge_not_on_main",
			Check: "the merge commit is reachable from " + main,
			Found: merge[:8] + " is not reachable from " + main,
			Fix:   fmt.Sprintf("fetch and update %s, then rerun", main),
		}
	}
	out, err := repo.Run(ctx, "rev-list", "--parents", "-n", "1", merge)
	if err != nil {
		return fmt.Errorf("read parents of %s: %w", merge, err)
	}
	if parents := len(strings.Fields(out)) - 1; parents < 2 {
		found := merge[:8] + " has one parent"
		if parents == 0 {
			found = merge[:8] + " has no parent"
		}
		return &Rejection{
			Code:  "merge_not_merge_commit",
			Check: "the merge is a merge commit",
			Found: found,
			Fix:   "spec_ref cannot survive; report to the user (rules §11)",
		}
	}
	merged, err := isAncestor(ctx, repo, revision, merge)
	if err != nil || merged {
		return err
	}
	return &Rejection{
		Code:  "revision_not_merged",
		Check: "the reviewed revision is in the merge",
		Found: fmt.Sprintf("%s is not contained in %s", revision[:8], merge[:8]),
		Fix:   "check that the right PR merged",
	}
}

func checkSpecRef(ctx context.Context, repo *git.Repo, id, merge, specRef, spec string) error {
	_, err := repo.Run(ctx, "cat-file", "-e", specRef)
	if gitErr, ok := errors.AsType[*git.Error](err); ok && (gitErr.ExitCode == 1 || gitErr.ExitCode == 128) && ctx.Err() == nil {
		return &Rejection{
			Code:  "spec_ref_invalid",
			Check: "the spec_ref resolves",
			Found: specRef + " does not resolve",
			Fix:   `copy it from the PR "Final specification" line`,
		}
	}
	if err != nil {
		return fmt.Errorf("resolve spec_ref %s: %w", specRef, err)
	}
	last, err := lastSpecCommit(ctx, repo, id, merge+"^2")
	if err != nil {
		return err
	}
	rejection := &Rejection{
		Code:  "spec_ref_not_final",
		Check: fmt.Sprintf("the spec_ref is the last spec(%s) commit merged with the recorded spec", id),
	}
	if last == "" {
		rejection.Found = fmt.Sprintf("no spec(%s) commit is reachable from %s^2", id, merge[:8])
		return rejection
	}
	commit, path, _ := strings.Cut(specRef, ":")
	if commit == last && path == spec {
		return nil
	}
	rejection.Found = fmt.Sprintf("%s:%s is not the last spec(%s) commit merged", commit[:8], path, id)
	rejection.Fix = fmt.Sprintf("pass --spec-ref %s:%s", last, spec)
	return rejection
}

// lastSpecCommit matches subjects itself because log --grep would also match body lines.
func lastSpecCommit(ctx context.Context, repo *git.Repo, id, from string) (string, error) {
	out, err := repo.Run(ctx, "log", "--topo-order", "--format=%H %s", from)
	if err != nil {
		return "", fmt.Errorf("list commits of %s: %w", from, err)
	}
	for line := range strings.Lines(out) {
		commit, subject, _ := strings.Cut(strings.TrimSuffix(line, "\n"), " ")
		if strings.HasPrefix(subject, "spec("+id+"):") {
			return commit, nil
		}
	}
	return "", nil
}

// isAncestor treats exit 128, an unknown SHA, as not an ancestor (I §8.3).
func isAncestor(ctx context.Context, repo *git.Repo, commit, of string) (bool, error) {
	_, err := repo.Run(ctx, "merge-base", "--is-ancestor", commit, of)
	if err == nil {
		return true, nil
	}
	if gitErr, ok := errors.AsType[*git.Error](err); ok && (gitErr.ExitCode == 1 || gitErr.ExitCode == 128) && ctx.Err() == nil {
		return false, nil
	}
	return false, fmt.Errorf("check %s is an ancestor of %s: %w", commit, of, err)
}
