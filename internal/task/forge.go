package task

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/madalinpopa/solomonar/internal/git"
)

// prView is the gh pr view output, decoded with gh's own field names.
type prView struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	State       string `json:"state"`
	MergeCommit *struct {
		OID string `json:"oid"`
	} `json:"mergeCommit"` //nolint:tagliatelle // gh's field name
	StatusCheckRollup []check `json:"statusCheckRollup"` //nolint:tagliatelle // gh's field name
}

// check is a CheckRun (status and conclusion) or a StatusContext (state).
type check struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	State      string `json:"state"`
}

// Forge reads the pull request of task id with gh; an unusable gh is a forge_unavailable warning.
func Forge(ctx context.Context, repo *git.Repo, id string) (ForgeFacts, []Warning, error) {
	out, err := repo.RunGH(ctx, "pr", "view", "feature/"+id, "--json", "number,url,state,mergeCommit,statusCheckRollup")
	if ctx.Err() != nil {
		return ForgeFacts{}, nil, fmt.Errorf("read the pull request of feature/%s: %w", id, ctx.Err())
	}
	gitErr, isCommandErr := errors.AsType[*git.Error](err)
	switch {
	case errors.Is(err, git.ErrNotFound):
		return forgeUnavailable("gh not found")
	case isCommandErr && strings.Contains(gitErr.Stderr, "no pull requests found"):
		return ForgeFacts{Available: true, Checks: "unknown"}, nil, nil
	case isCommandErr && gitErr.Stderr != "":
		reason, _, _ := strings.Cut(gitErr.Stderr, "\n")
		return forgeUnavailable(reason)
	case isCommandErr:
		return forgeUnavailable(gitErr.Err.Error())
	case err != nil:
		return ForgeFacts{}, nil, fmt.Errorf("read the pull request of feature/%s: %w", id, err)
	}
	var view prView
	if err := json.Unmarshal([]byte(out), &view); err != nil {
		return forgeUnavailable("unreadable gh pr view output")
	}
	pr := &PullRequest{Number: view.Number, URL: view.URL, State: view.State}
	if view.MergeCommit != nil {
		pr.MergeCommit = &view.MergeCommit.OID
	}
	return ForgeFacts{Available: true, PR: pr, Checks: summarizeChecks(view.StatusCheckRollup)}, nil, nil
}

func forgeUnavailable(reason string) (ForgeFacts, []Warning, error) {
	return NoForge(), []Warning{{Code: "forge_unavailable", Message: "gh unavailable: " + reason}}, nil
}

// summarizeChecks applies the first matching rule of I section 9.1; unrecognized is never
// passing (J6).
func summarizeChecks(rollup []check) string {
	if len(rollup) == 0 {
		return "none"
	}
	var pending, unknown bool
	for _, c := range rollup {
		result, completed := c.Conclusion, c.Status == "COMPLETED"
		if c.State != "" {
			result, completed = c.State, c.State != "PENDING" && c.State != "EXPECTED"
		}
		switch {
		case slices.Contains([]string{"FAILURE", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED", "ERROR", "STARTUP_FAILURE"}, result):
			return "failing"
		case c.Status == "" && c.State == "":
			unknown = true
		case !completed:
			pending = true
		case !slices.Contains([]string{"SUCCESS", "NEUTRAL", "SKIPPED"}, result):
			unknown = true
		}
	}
	switch {
	case pending:
		return "pending"
	case unknown:
		return "unknown"
	default:
		return "passing"
	}
}
