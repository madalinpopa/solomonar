package task

import (
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxSummaryRunes = 80
	maxActionRunes  = 120
)

func invalidValue(check, found, fix string) *Rejection {
	return &Rejection{Code: "invalid_value", Check: check, Found: found, Fix: fix}
}

// parseAgents reads repeated --agent values for the installed roles; clears reports a lone none.
func parseAgents(values, roles []string) (agents []Agent, clears bool, err error) {
	if slices.Contains(values, "none") {
		if len(values) > 1 {
			return nil, false, agentRejection(strings.Join(values, " "))
		}
		return nil, true, nil
	}
	delegated := slices.DeleteFunc(slices.Clone(roles), func(role string) bool { return role == "orchestrator" })
	for _, value := range values {
		id, role, _ := strings.Cut(value, ":")
		if !validAgentID(id) || role == "" {
			return nil, false, agentRejection(value)
		}
		if !slices.Contains(delegated, role) {
			return nil, false, invalidValue("--agent role is an installed delegated role, never orchestrator", value,
				"allowed roles: "+join(delegated))
		}
		agents = append(agents, Agent{ID: id, Role: role})
	}
	return agents, false, nil
}

func agentRejection(found string) *Rejection {
	return invalidValue("--agent takes <id>:<role>, or none alone", found,
		"pass --agent impl-1:implementer, or --agent none to clear the list")
}

func parseNextAction(value string) (*NextAction, error) {
	actor, action, _ := strings.Cut(value, ":")
	action = strings.TrimSpace(action)
	if actor == "" || action == "" {
		return nil, invalidValue("--next takes <actor>:<action>", value,
			`pass --next "impl-1:Implement AC-1"`)
	}
	if utf8.RuneCountInString(action) > maxActionRunes {
		return nil, invalidValue("--next action takes 1 to 120 characters", value,
			"shorten the action; put details in the spec or a decision")
	}
	return &NextAction{Actor: actor, Action: action}, nil
}

// checkSpec accepts a clean worktree-relative path; existence is checked against Git separately.
func checkSpec(value string) error {
	if value != "." && path.Clean(value) == value && filepath.IsLocal(filepath.FromSlash(value)) {
		return nil
	}
	return invalidValue("--spec takes a clean path relative to the worktree, without ..", value,
		"pass --spec docs/work/<task ID>-<slug>.md")
}

func parseTier(value string) (*int, error) {
	switch value {
	case "0", "1", "2":
		tier, _ := strconv.Atoi(value)
		return &tier, nil
	default:
		return nil, invalidValue("--tier takes 0, 1, or 2", value, "pass --tier 0, --tier 1, or --tier 2")
	}
}

func checkRevision(value string) error {
	if fullSHA(value) {
		return nil
	}
	return &Rejection{
		Code:  "revision_not_full_sha",
		Check: "--revision must be a full 40-character SHA",
		Found: value,
		Fix:   "git rev-parse " + value,
	}
}

func checkPR(value string) error {
	if number, err := strconv.Atoi(value); err == nil && allDigits(value) && number > 0 {
		return nil
	}
	if u, err := url.Parse(value); err == nil && u.Scheme == "https" && u.Host != "" && strings.Trim(u.Path, "/") != "" {
		return nil
	}
	return invalidValue("--pr takes an https URL or a positive PR number", value,
		"pass --pr 42 or --pr https://github.com/<owner>/<repo>/pull/42")
}

// newBlocker builds a blocker from the block flags; the caller issues its ID.
func newBlocker(kind, summary, ref string) (Blocker, error) {
	blockerKind := BlockerKind(kind)
	if !slices.Contains(blockerKinds(), blockerKind) {
		return Blocker{}, invalidValue("--kind takes decision, dependency, check, or environment", kind,
			"pass one of the listed kinds")
	}
	if strings.TrimSpace(summary) == "" || utf8.RuneCountInString(summary) > maxSummaryRunes {
		return Blocker{}, invalidValue("--summary takes 1 to 80 characters", summary,
			"shorten the summary; put details in the referenced decision")
	}
	if blockerKind == BlockerDecision && !decisionRef(ref) {
		return Blocker{}, invalidValue("--kind decision needs --ref <task ID>/D-<n>", ref, "pass --ref W-001/D-1")
	}
	return Blocker{Kind: blockerKind, Summary: summary, Ref: ref}, nil
}

func decisionRef(ref string) bool {
	id, decision, _ := strings.Cut(ref, "/")
	digits, ok := strings.CutPrefix(decision, "D-")
	return validTaskID(id) && ok && digits != "" && allDigits(digits)
}

func checkTaskID(value string) error {
	if validTaskID(value) {
		return nil
	}
	return invalidValue("task IDs look like W-001", value, "pass the task ID as shown by solomonar task show")
}
