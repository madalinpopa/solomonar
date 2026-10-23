package task

import (
	"fmt"
	"slices"
	"strconv"
)

// Drift is a difference between the record and the facts (I section 9.2).
type Drift struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DriftOf lists the drift of task id in the I section 9.2 table order; main names the main branch.
func DriftOf(id, main string, record *Record, facts StatusFacts, envOwner string) []Drift {
	drift := []Drift{}
	add := func(applies bool, code, message string) {
		if applies {
			drift = append(drift, Drift{Code: code, Message: message})
		}
	}
	branch := "feature/" + id
	if slices.Contains([]Status{StatusReserved, StatusRunning, StatusBlocked, StatusPaused}, record.Status) {
		ahead, dirty := worktreeCounts(facts.Worktree)
		pr := facts.Forge.PR
		add(record.Status == StatusRunning && record.Stage != StageSetup && !facts.Branch.Exists,
			"branch_missing", branch+" does not exist")
		add(facts.Branch.Exists && facts.Worktree == nil, "worktree_missing", branch+" has no worktree")
		add(record.Worktree != "" && facts.Worktree != nil && record.Worktree != facts.Worktree.Relative,
			"worktree_mismatch", fmt.Sprintf("recorded %s; Git lists %s", record.Worktree, worktreeRelative(facts.Worktree)))
		add(record.Stage != StageSetup && record.Base == "", "base_missing", id+" has no recorded base")
		add(needsConfirmation(record), "unconfirmed", fmt.Sprintf("%s is not confirmed by %s; inspect, then task confirm",
			id, record.Owner))
		add(ahead > 0, "head_ahead_of_revision",
			fmt.Sprintf("HEAD is %d %s past the recorded revision", ahead, plural(ahead, "commit", "commits")))
		add(facts.Branch.ContainsRevision != nil && !*facts.Branch.ContainsRevision, "revision_not_on_branch",
			fmt.Sprintf("%.8s is not on %s", record.Revision, branch))
		add(dirty > 0, "dirty_worktree", fmt.Sprintf("%d uncommitted %s; preserve %s (rules section 10)",
			dirty, plural(dirty, "path", "paths"), plural(dirty, "it", "them")))
		add(record.Spec != "" && !facts.Spec.InWorktree && !facts.Spec.AtRevision, "spec_missing",
			record.Spec+" is not in the worktree or at the revision")
		add(record.Status == StatusBlocked && len(record.Blockers) == 0,
			"blocked_without_blockers", "blocked without an open blocker")
		add(record.Tier == nil && record.Stage.atOrAfter(StageSpecializedReview),
			"tier_missing", fmt.Sprintf("stage %s needs a tier", record.Stage))
		add(facts.Merge.Merged, "merged_not_done",
			fmt.Sprintf("%s is merged into %s but the record is not done", branch, main))
		add(pr != nil && record.PR == "", "pr_not_recorded",
			fmt.Sprintf("gh shows PR #%d for %s; the record has none", prNumber(pr), branch))
		add(pr != nil && record.PR != "" && record.PR != strconv.Itoa(prNumber(pr)) && record.PR != pr.URL,
			"pr_mismatch", fmt.Sprintf("the record has PR %s; gh shows #%d for %s", record.PR, prNumber(pr), branch))
	}
	add(envOwner != "" && record.Owner != "" && envOwner != record.Owner, "owned_by_other",
		fmt.Sprintf("SOLOMONAR_OWNER is %s; %s is owned by %s", envOwner, id, record.Owner))
	return drift
}

func worktreeRelative(w *WorktreeFacts) string {
	if w == nil {
		return ""
	}
	return w.Relative
}

// needsConfirmation reports an active task past setup whose owner has not confirmed it (CLI-D22).
func needsConfirmation(r *Record) bool {
	return ownsActively(r.Status) && r.Stage != StageSetup && !r.Confirmed
}

func worktreeCounts(w *WorktreeFacts) (ahead, dirty int) {
	if w == nil {
		return 0, 0
	}
	if w.AheadOfRevision != nil {
		ahead = *w.AheadOfRevision
	}
	return ahead, len(w.Dirty)
}

// prNumber reads the number of a pull request that may be absent; messages using it are then dropped.
func prNumber(pr *PullRequest) int {
	if pr == nil {
		return 0
	}
	return pr.Number
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
