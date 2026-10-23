package task

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/madalinpopa/solomonar/internal/git"
)

// ConfirmTask records that the owner inspected the task and continues it (CLI-D22).
func (r *Registry) ConfirmTask(ctx context.Context, repo *git.Repo, id, as string, now time.Time) (MoveResult, error) {
	before, err := r.Lookup(id)
	if err != nil {
		return MoveResult{}, err
	}
	if before.Status == StatusDone {
		return MoveResult{}, doneFinal(StatusDone)
	}
	if before.Status == StatusPlanned {
		return MoveResult{}, &Rejection{
			Code:  "not_registered",
			Check: "the task is registered",
			Found: id + " is planned",
			Fix:   fmt.Sprintf("solomonar task register %s --title <title> --owner %s", id, as),
		}
	}
	if err := before.CheckOwner(id, as); err != nil {
		return MoveResult{}, err
	}
	if before.Stage == StageSetup {
		return MoveResult{}, &Rejection{
			Code:  "setup_not_complete",
			Check: "task confirm needs a stage past setup; leaving setup confirms the task",
			Found: id + " is at setup",
			Fix:   fmt.Sprintf("solomonar task move %s --as %s --status running --stage specification", id, as),
		}
	}
	result := MoveResult{From: before.position(), To: before.position(), Changed: []string{}}
	if before.Confirmed {
		return result, nil
	}
	if err := checkWorkspace(ctx, repo, id, before); err != nil {
		return MoveResult{}, err
	}
	after := *before
	after.Confirmed = true
	after.UpdatedAt = timestamp(now)
	r.Tasks[id] = &after
	result.Changed = changedFields(before, &after)
	result.Subject = after.Subject(id, "confirmed by "+as)
	return result, nil
}

// checkWorkspace is the confirmation preflight: the branch, recorded worktree, and recorded base.
func checkWorkspace(ctx context.Context, repo *git.Repo, id string, r *Record) error {
	branch := "feature/" + id
	if err := checkBranch(ctx, repo, branch); err != nil {
		return err
	}
	if r.Worktree == "" {
		return &Rejection{
			Code:  "worktree_unrecorded",
			Check: "the record names the task worktree",
			Found: id + " has no recorded worktree",
			Fix:   fmt.Sprintf("solomonar task move %s --as %s --worktree <path relative to the main checkout>", id, r.Owner),
		}
	}
	if err := checkRecordedWorktree(ctx, repo, id, r.Worktree); err != nil {
		return err
	}
	if r.Base == "" {
		return &Rejection{
			Code:  "base_missing",
			Check: "the record names the commit the task branched from",
			Found: id + " has no recorded base",
			Fix:   fmt.Sprintf("solomonar task move %s --as %s --base <full sha>, after inspecting the branch history", id, r.Owner),
		}
	}
	return checkBase(ctx, repo, branch, r.Base)
}

// checkConfirmed rejects a stage change of an unconfirmed task; leaving setup confirms it.
func checkConfirmed(id string, before, after *Record) error {
	if before.Stage == after.Stage || before.Stage == StageSetup || before.Confirmed {
		return nil
	}
	return confirmationRequired(id, before)
}

func confirmationRequired(id string, r *Record) *Rejection {
	return &Rejection{
		Code:  "confirmation_required",
		Check: "a stage change needs a task its owner confirmed (CLI-D22)",
		Found: id + " is not confirmed by " + r.Owner,
		Fix:   fmt.Sprintf("inspect with solomonar task status %s, then solomonar task confirm %s --as %s", id, id, r.Owner),
	}
}

// recordWorkspace fills base, worktree, and confirmed as the task leaves setup (CLI-D21).
func recordWorkspace(ctx context.Context, repo *git.Repo, id, main string, after *Record) error {
	branch := "refs/heads/feature/" + id
	base, err := repo.Run(ctx, "merge-base", main, branch)
	if gitErr, ok := errors.AsType[*git.Error](err); ok && ctx.Err() == nil && (gitErr.ExitCode == 1 || gitErr.ExitCode == 128) {
		return &Rejection{
			Code:  "base_not_found",
			Check: "feature/" + id + " branched from " + main,
			Found: fmt.Sprintf("no common commit of %s and feature/%s", main, id),
			Fix:   fmt.Sprintf("create feature/%s from the current %s (rules section 4)", id, main),
		}
	}
	if err != nil {
		return fmt.Errorf("find the base of feature/%s: %w", id, err)
	}
	if after.Worktree == "" {
		listed, err := Worktree(ctx, repo, "feature/"+id)
		if err != nil {
			return err
		}
		if after.Worktree, err = relativeWorktree(ctx, repo, listed); err != nil {
			return err
		}
	}
	after.Base, after.Confirmed = base, true
	return nil
}

func checkBase(ctx context.Context, repo *git.Repo, branch, base string) error {
	onBranch, err := isAncestor(ctx, repo, base, "refs/heads/"+branch)
	if err != nil || onBranch {
		return err
	}
	return &Rejection{
		Code:  "base_not_on_branch",
		Check: "the base is a commit reachable from " + branch,
		Found: fmt.Sprintf("%.8s is not on %s", base, branch),
		Fix:   "inspect the branch history; the base is the commit the task branched from",
	}
}

// checkRecordedWorktree checks that Git lists the worktree of the task branch at the recorded path.
func checkRecordedWorktree(ctx context.Context, repo *git.Repo, id, recorded string) error {
	branch := "feature/" + id
	listed, err := Worktree(ctx, repo, branch)
	if err != nil {
		return err
	}
	if listed == "" {
		return checkWorktree(ctx, repo, id, branch)
	}
	main, err := mainWorktree(ctx, repo)
	if err != nil {
		return err
	}
	if samePath(filepath.Join(main, filepath.FromSlash(recorded)), listed) {
		return nil
	}
	actual, err := relativeWorktree(ctx, repo, listed)
	if err != nil {
		return err
	}
	return &Rejection{
		Code:  "worktree_mismatch",
		Check: "Git lists the " + branch + " worktree at the recorded path",
		Found: fmt.Sprintf("recorded %s; Git lists %s", recorded, actual),
		Fix:   fmt.Sprintf("record the relocation: solomonar task move %s --worktree %s", id, actual),
	}
}

// relativeWorktree expresses a worktree path relative to the main worktree, slash-separated.
func relativeWorktree(ctx context.Context, repo *git.Repo, path string) (string, error) {
	main, err := mainWorktree(ctx, repo)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolved(main), resolved(path))
	if err != nil {
		return "", fmt.Errorf("relate worktree %s to %s: %w", path, main, err)
	}
	return filepath.ToSlash(rel), nil
}

// mainWorktree returns the main worktree, the first one git worktree list names.
func mainWorktree(ctx context.Context, repo *git.Repo) (string, error) {
	out, err := repo.Run(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("list worktrees: %w", err)
	}
	first, _, _ := strings.Cut(out, "\n")
	path, ok := strings.CutPrefix(first, "worktree ")
	if !ok {
		return "", fmt.Errorf("list worktrees: unexpected output %q", first)
	}
	return path, nil
}

func samePath(a, b string) bool {
	return resolved(a) == resolved(b)
}

// resolved follows symbolic links where the path exists, so /var and /private/var compare equal.
func resolved(path string) string {
	if target, err := filepath.EvalSymlinks(path); err == nil {
		return target
	}
	return filepath.Clean(path)
}
