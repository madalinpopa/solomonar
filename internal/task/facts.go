package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/madalinpopa/solomonar/internal/git"
)

// CheckFacts verifies a move against Git (I §8.3) and returns where it found the checked spec.
func CheckFacts(ctx context.Context, repo *git.Repo, id string, before, after *Record, m Move) (specFound string, err error) {
	branch := "feature/" + id
	if m.Revision != "" {
		if err := checkRevisionOnBranch(ctx, repo, branch, m.Revision); err != nil {
			return "", err
		}
	}
	if before.Stage == StageSetup && after.Stage == StageSpecification {
		if err := checkWorktree(ctx, repo, id, branch); err != nil {
			return "", err
		}
	}
	if m.Spec != "" || (after.Stage == StageImplementation && before.Stage != StageImplementation) {
		return findSpec(ctx, repo, branch, after)
	}
	return "", nil
}

func checkRevisionOnBranch(ctx context.Context, repo *git.Repo, branch, revision string) error {
	if err := checkBranch(ctx, repo, branch); err != nil {
		return err
	}
	_, err := repo.Run(ctx, "merge-base", "--is-ancestor", revision, "refs/heads/"+branch)
	if err == nil {
		return nil
	}
	// Exit 1 means not an ancestor; 128 means Git does not know the SHA, which fails too (I §8.3).
	gitErr, ok := errors.AsType[*git.Error](err)
	if !ok || ctx.Err() != nil || (gitErr.ExitCode != 1 && gitErr.ExitCode != 128) {
		return fmt.Errorf("check revision on %s: %w", branch, err)
	}
	worktree, err := Worktree(ctx, repo, branch)
	if err != nil {
		return err
	}
	headCommand := "git rev-parse " + branch
	if worktree != "" {
		headCommand = "git -C " + worktree + " rev-parse HEAD"
	}
	return &Rejection{
		Code:  "revision_not_on_branch",
		Check: "--revision must be a full SHA reachable from " + branch,
		Found: revision + " is not on " + branch,
		Fix:   fmt.Sprintf("commit the candidate on %s and pass its full SHA: %s", branch, headCommand),
	}
}

func checkBranch(ctx context.Context, repo *git.Repo, branch string) error {
	_, err := repo.Run(ctx, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if err == nil {
		return nil
	}
	if gitErr, ok := errors.AsType[*git.Error](err); !ok || gitErr.ExitCode != 1 || ctx.Err() != nil {
		return fmt.Errorf("check branch %s: %w", branch, err)
	}
	return &Rejection{
		Code:  "branch_missing",
		Check: branch + " exists locally",
		Found: branch + " does not exist",
		Fix:   "create the branch and worktree (rules §4)",
	}
}

func checkWorktree(ctx context.Context, repo *git.Repo, id, branch string) error {
	if err := checkBranch(ctx, repo, branch); err != nil {
		return err
	}
	worktree, err := Worktree(ctx, repo, branch)
	if err != nil || worktree != "" {
		return err
	}
	repoName := filepath.Base(filepath.Dir(repo.CommonDir))
	return &Rejection{
		Code:  "worktree_missing",
		Check: "a worktree has " + branch + " checked out",
		Found: "no worktree on " + branch,
		Fix:   fmt.Sprintf("git worktree add ../%s-worktrees/%s %s", repoName, id, branch),
	}
}

// Worktree returns the path of the worktree that has branch checked out, or "" when none has.
func Worktree(ctx context.Context, repo *git.Repo, branch string) (string, error) {
	out, err := repo.Run(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("list worktrees: %w", err)
	}
	var path string
	for line := range strings.Lines(out) {
		line = strings.TrimSuffix(line, "\n")
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			path = p
		} else if line == "branch refs/heads/"+branch {
			return path, nil
		}
	}
	return "", nil
}

func findSpec(ctx context.Context, repo *git.Repo, branch string, after *Record) (string, error) {
	worktree, err := Worktree(ctx, repo, branch)
	if err != nil {
		return "", err
	}
	var searched []string
	if worktree != "" {
		if info, err := os.Stat(filepath.Join(worktree, filepath.FromSlash(after.Spec))); err == nil && info.Mode().IsRegular() {
			return worktree, nil
		}
		searched = append(searched, "in the "+branch+" worktree")
	}
	if after.Revision != "" {
		found, err := isBlob(ctx, repo, after.Revision+":"+after.Spec)
		if err != nil {
			return "", err
		}
		if found {
			return after.Revision[:7], nil
		}
		searched = append(searched, "at the recorded revision")
	}
	found := fmt.Sprintf("no worktree on %s and no recorded revision to look for %s", branch, after.Spec)
	if len(searched) > 0 {
		found = after.Spec + " is not " + strings.Join(searched, " or ")
	}
	return "", &Rejection{
		Code:  "spec_not_found",
		Check: "the spec is a regular file in the " + branch + " worktree or a blob at the recorded revision",
		Found: found,
		Fix:   "commit the work spec in the worktree",
	}
}

// isBlob uses cat-file -t, not -e, so a directory at the revision does not count as the spec.
func isBlob(ctx context.Context, repo *git.Repo, object string) (bool, error) {
	kind, err := repo.Run(ctx, "cat-file", "-t", object)
	if gitErr, ok := errors.AsType[*git.Error](err); ok && gitErr.ExitCode == 128 && ctx.Err() == nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("look up %s: %w", object, err)
	}
	return kind == "blob", nil
}
