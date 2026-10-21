package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/madalinpopa/solomonar/internal/git"
)

// StatusFacts are the Git and forge facts of task status (I §9.1).
type StatusFacts struct {
	Branch   BranchFacts    `json:"branch"`
	Worktree *WorktreeFacts `json:"worktree"`
	Spec     SpecFacts      `json:"spec"`
	Merge    MergeFacts     `json:"merge"`
	Forge    ForgeFacts     `json:"forge"`
}

// BranchFacts describe the task branch.
type BranchFacts struct {
	Name             string  `json:"name"`
	Exists           bool    `json:"exists"`
	Tip              *string `json:"tip"`
	ContainsRevision *bool   `json:"contains_revision"`
}

// WorktreeFacts describe the worktree that has the task branch checked out.
type WorktreeFacts struct {
	Path            string      `json:"path"`
	Head            string      `json:"head"`
	AheadOfRevision *int        `json:"ahead_of_revision"`
	Dirty           []DirtyPath `json:"dirty"`
}

// DirtyPath is an uncommitted change in the worktree.
type DirtyPath struct {
	Path  string `json:"path"`
	State string `json:"state"`
}

// SpecFacts tell where the recorded spec is present.
type SpecFacts struct {
	InWorktree bool `json:"in_worktree"`
	AtRevision bool `json:"at_revision"`
}

// MergeFacts describe the merge of the revision into main.
type MergeFacts struct {
	Merged         bool    `json:"merged"`
	MergeCommit    *string `json:"merge_commit"`
	DerivedSpecRef *string `json:"derived_spec_ref"`
}

// ForgeFacts describe the branch's pull request as gh reports it.
type ForgeFacts struct {
	Available bool         `json:"available"`
	PR        *PullRequest `json:"pr"`
	Checks    string       `json:"checks"`
}

// PullRequest is the pull request of the task branch.
type PullRequest struct {
	Number      int     `json:"number"`
	URL         string  `json:"url"`
	State       string  `json:"state"`
	MergeCommit *string `json:"merge_commit"`
}

// NoForge returns the forge facts without gh, also used for --no-forge.
func NoForge() ForgeFacts { return ForgeFacts{Checks: "unknown"} }

// GitFacts collects the Git facts of task id, leaving the forge facts unavailable.
func GitFacts(ctx context.Context, repo *git.Repo, id, main string, record *Record) (StatusFacts, error) {
	facts := StatusFacts{Branch: BranchFacts{Name: "feature/" + id}, Forge: NoForge()}
	if err := branchFacts(ctx, repo, &facts.Branch, record.Revision); err != nil {
		return StatusFacts{}, err
	}
	worktree, err := Worktree(ctx, repo, facts.Branch.Name)
	if err != nil {
		return StatusFacts{}, err
	}
	if worktree != "" {
		if facts.Worktree, err = worktreeFacts(ctx, &git.Repo{Dir: worktree, Trace: repo.Trace}, record.Revision); err != nil {
			return StatusFacts{}, err
		}
	}
	if facts.Spec, err = specFacts(ctx, repo, worktree, record); err != nil {
		return StatusFacts{}, err
	}
	if facts.Merge, err = mergeFacts(ctx, repo, id, main, record); err != nil {
		return StatusFacts{}, err
	}
	return facts, nil
}

func branchFacts(ctx context.Context, repo *git.Repo, branch *BranchFacts, revision string) error {
	ref := "refs/heads/" + branch.Name
	tip, err := repo.Run(ctx, "rev-parse", "--verify", "--quiet", ref)
	if gitErr, ok := errors.AsType[*git.Error](err); ok && gitErr.ExitCode == 1 && ctx.Err() == nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check branch %s: %w", branch.Name, err)
	}
	branch.Exists, branch.Tip = true, &tip
	if revision == "" {
		return nil
	}
	contains, err := isAncestor(ctx, repo, revision, ref)
	branch.ContainsRevision = &contains
	return err
}

func worktreeFacts(ctx context.Context, worktree *git.Repo, revision string) (*WorktreeFacts, error) {
	head, err := worktree.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("read HEAD of %s: %w", worktree.Dir, err)
	}
	facts := &WorktreeFacts{Path: worktree.Dir, Head: head}
	if revision != "" {
		out, err := worktree.Run(ctx, "rev-list", "--count", revision+"..HEAD")
		if err != nil && !unknownObject(ctx, err) {
			return nil, fmt.Errorf("count commits after %s: %w", revision, err)
		}
		if err == nil {
			count, err := strconv.Atoi(out)
			if err != nil {
				return nil, fmt.Errorf("count commits after %s: %w", revision, err)
			}
			facts.AheadOfRevision = &count
		}
	}
	status, err := worktree.Run(ctx, "status", "--porcelain=v2")
	if err != nil {
		return nil, fmt.Errorf("read status of %s: %w", worktree.Dir, err)
	}
	facts.Dirty = parseDirty(status)
	return facts, nil
}

// unknownObject reports exit 128, which Git gives when the recorded revision is not in the repository.
func unknownObject(ctx context.Context, err error) bool {
	gitErr, ok := errors.AsType[*git.Error](err)
	return ok && gitErr.ExitCode == 128 && ctx.Err() == nil
}

func parseDirty(status string) []DirtyPath {
	dirty := []DirtyPath{}
	for line := range strings.Lines(status) {
		kind, rest, _ := strings.Cut(strings.TrimSuffix(line, "\n"), " ")
		var entry DirtyPath
		switch kind {
		case "1":
			entry = DirtyPath{Path: afterFields(rest, 7), State: changeState(rest)}
		case "2":
			entry.Path, _, _ = strings.Cut(afterFields(rest, 8), "\t")
			entry.State = "renamed"
			if strings.Contains(rest[:2], "C") {
				entry.State = "copied"
			}
		case "u":
			entry = DirtyPath{Path: afterFields(rest, 9), State: "unmerged"}
		case "?":
			entry = DirtyPath{Path: rest, State: "untracked"}
		default:
			continue
		}
		dirty = append(dirty, entry)
	}
	return dirty
}

// afterFields returns what follows the first n space-separated fields of s.
func afterFields(s string, n int) string {
	fields := strings.SplitN(s, " ", n+1)
	return fields[len(fields)-1]
}

// changeState reads the worktree side of the XY code that starts fields, else the index side.
func changeState(fields string) string {
	change := fields[1]
	if change == '.' {
		change = fields[0]
	}
	switch change {
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'T':
		return "type_changed"
	default:
		return "modified"
	}
}

func specFacts(ctx context.Context, repo *git.Repo, worktree string, record *Record) (SpecFacts, error) {
	var facts SpecFacts
	if record.Spec == "" {
		return facts, nil
	}
	if worktree != "" {
		info, err := os.Stat(filepath.Join(worktree, filepath.FromSlash(record.Spec)))
		facts.InWorktree = err == nil && info.Mode().IsRegular()
	}
	if record.Revision == "" {
		return facts, nil
	}
	var err error
	facts.AtRevision, err = isBlob(ctx, repo, record.Revision+":"+record.Spec)
	return facts, err
}

func mergeFacts(ctx context.Context, repo *git.Repo, id, main string, record *Record) (MergeFacts, error) {
	if record.Status == StatusDone {
		return MergeFacts{Merged: true, MergeCommit: optional(record.MergeCommit), DerivedSpecRef: optional(record.SpecRef)}, nil
	}
	if record.Revision == "" {
		return MergeFacts{}, nil
	}
	out, err := repo.Run(ctx, "rev-list", "--ancestry-path", "--merges", "--reverse", record.Revision+".."+main)
	if unknownObject(ctx, err) {
		return MergeFacts{}, nil
	}
	if err != nil {
		return MergeFacts{}, fmt.Errorf("find the merge of %s into %s: %w", record.Revision, main, err)
	}
	merge, _, _ := strings.Cut(out, "\n")
	if merge == "" {
		return MergeFacts{}, nil
	}
	facts := MergeFacts{Merged: true, MergeCommit: &merge}
	if record.Spec == "" {
		return facts, nil
	}
	last, err := lastSpecCommit(ctx, repo, id, merge+"^2")
	if last != "" {
		facts.DerivedSpecRef = new(last + ":" + record.Spec)
	}
	return facts, err
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
