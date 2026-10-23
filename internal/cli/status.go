package cli

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/madalinpopa/solomonar/internal/task"
)

type statusFlags struct {
	noForge bool
}

type statusData struct {
	ID             string           `json:"id"`
	Record         *task.Record     `json:"record"`
	RegistryCommit string           `json:"registry_commit"`
	Facts          task.StatusFacts `json:"facts"`
	Drift          []task.Drift     `json:"drift"`
	AllowedNext    []string         `json:"allowed_next"`
}

func taskStatusCommand() command {
	return command{
		name:    "task status",
		summary: "Show a task's record, Git and forge facts, drift, and allowed next commands",
		usage:   "solomonar task status <ID> [--json] [--no-forge]",
		description: "Compare the stored record with Git and gh facts, then print the drift and the commands valid next.\n" +
			"Read-only; exit 0 whatever the drift. --no-forge skips gh; SOLOMONAR_OWNER feeds owned_by_other.",
		bind: func(fs *flag.FlagSet) runFunc {
			var s statusFlags
			fs.BoolVar(&s.noForge, "no-forge", false, "")
			return s.run
		},
	}
}

func (s *statusFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	switch {
	case len(args) == 0:
		return output{}, invalidUsage(errors.New("task ID is required"))
	case len(args) > 1:
		return output{}, unexpectedArgument(args[1])
	}
	store, mainBranch, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, rejectedAs("task status "+args[0], err)
	}
	out, err := s.status(ctx, env, store, mainBranch, args[0])
	return out, rejectedAs("task status "+args[0], err)
}

func (s *statusFlags) status(ctx context.Context, env Env, store *task.Store, mainBranch, id string) (output, error) {
	reg, tip, err := store.Read(ctx)
	if err != nil {
		return output{}, err
	}
	record, err := reg.Lookup(id)
	if err != nil {
		return output{}, err
	}
	facts, err := task.GitFacts(ctx, store.Git, id, mainBranch, record)
	if err != nil {
		return output{}, err
	}
	var warnings []task.Warning
	if !s.noForge {
		if facts.Forge, warnings, err = task.Forge(ctx, store.Git, id); err != nil {
			return output{}, err
		}
	}
	data := statusData{
		ID: id, Record: record, RegistryCommit: tip, Facts: facts,
		Drift:       task.DriftOf(id, mainBranch, record, facts, env.Getenv("SOLOMONAR_OWNER")),
		AllowedNext: task.AllowedNext(id, record, facts),
	}
	return output{data: data, text: s.statusText(data, mainBranch), warnings: outputWarnings(warnings)}, nil
}

func (s *statusFlags) statusText(d statusData, mainBranch string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s\n", d.ID, d.Record.Title)
	writeSection(&b, "record", recordDetails(d.Record, d.RegistryCommit))
	writeSection(&b, "git", gitDetails(d.ID, d.Record, d.Facts, mainBranch))
	writeSection(&b, "forge (gh)", s.forgeDetails(d.Facts))
	drift := make([]detail, len(d.Drift))
	for i, entry := range d.Drift {
		drift[i] = detail{entry.Code, entry.Message}
	}
	writeSection(&b, "drift", drift)
	b.WriteString("allowed next\n")
	if len(d.AllowedNext) == 0 {
		b.WriteString("  none\n")
	}
	for _, next := range d.AllowedNext {
		b.WriteString("  " + next + "\n")
	}
	return b.String()
}

func writeSection(b *strings.Builder, title string, details []detail) {
	b.WriteString(title + "\n")
	if len(details) == 0 {
		b.WriteString("  none\n")
	}
	writeDetails(b, details)
}

func recordDetails(r *task.Record, tip string) []detail {
	tier := "none"
	if r.Tier != nil {
		tier = strconv.Itoa(*r.Tier)
	}
	status := detail{"status", fmt.Sprintf("%s / %s   owner %s   tier %s",
		r.Status, cmp.Or(string(r.Stage), "-"), cmp.Or(r.Owner, "-"), tier)}
	pr := detail{"pr", cmp.Or(r.PR, "none")}
	updated := detail{"updated", r.UpdatedAt + " (registry @ " + tip[:7] + ")"}
	if r.Status == task.StatusDone {
		return []detail{status, {"summary", r.Summary}, {"spec_ref", r.SpecRef}, {"merge", r.MergeCommit}, pr, updated}
	}
	agents := "none"
	if len(r.ActiveAgents) > 0 {
		agents = agentsText(r.ActiveAgents) + " (last reported, not liveness)"
	}
	next := "none"
	if r.NextAction != nil {
		next = r.NextAction.Actor + ": " + r.NextAction.Action
	}
	if r.Owner != "" && r.Stage != task.StageSetup {
		status.value = strings.Replace(status.value, "owner "+r.Owner, "owner "+r.Owner+confirmation(r), 1)
	}
	return []detail{
		status,
		{"base", cmp.Or(r.Base, "none")},
		{"worktree", cmp.Or(r.Worktree, "none")},
		{"spec", cmp.Or(r.Spec, "none")},
		{"revision", cmp.Or(r.Revision, "none")},
		{"agents", agents},
		{"blockers", blockersText(r.Blockers)},
		{"next", next},
		pr, updated,
	}
}

func confirmation(r *task.Record) string {
	if r.Confirmed {
		return " (confirmed)"
	}
	return " (unconfirmed)"
}

func blockersText(blockers []task.Blocker) string {
	if len(blockers) == 0 {
		return "none"
	}
	parts := make([]string, len(blockers))
	for i, blocker := range blockers {
		parts[i] = fmt.Sprintf("%s %s: %s", blocker.ID, blocker.Kind, blocker.Summary)
		if blocker.Ref != "" {
			parts[i] += " (" + blocker.Ref + ")"
		}
	}
	return strings.Join(parts, "; ")
}

func gitDetails(id string, r *task.Record, f task.StatusFacts, mainBranch string) []detail {
	branch := f.Branch.Name + " missing"
	if f.Branch.Exists {
		branch = f.Branch.Name + " @ " + *f.Branch.Tip
	}
	worktree, head, dirty := "missing", "unknown (no worktree)", "unknown (no worktree)"
	if w := f.Worktree; w != nil {
		worktree, head, dirty = w.Path, w.Head[:8], dirtyText(w.Dirty)
		if n := w.AheadOfRevision; n != nil {
			head += fmt.Sprintf(" · %d %s ahead of revision", *n, plural(*n, "commit", "commits"))
		}
	}
	merged := "no"
	if f.Merge.MergeCommit != nil {
		merged = "yes, by " + *f.Merge.MergeCommit + " (merge commit on " + mainBranch + ")"
	}
	details := []detail{
		{"branch", branch},
		{"worktree", worktree},
		{"HEAD", head},
		{"dirty", dirty},
		{"spec", specPresence(r.Spec, f.Spec)},
		{"merged", merged},
	}
	// A done record's spec_ref is recorded, not derived; the record section shows it.
	if ref := f.Merge.DerivedSpecRef; ref != nil && r.Status != task.StatusDone {
		details = append(details, detail{"spec_ref", "derived " + *ref + " (last spec(" + id + ") commit merged)"})
	}
	return details
}

func dirtyText(dirty []task.DirtyPath) string {
	if len(dirty) == 0 {
		return "clean"
	}
	parts := make([]string, len(dirty))
	for i, path := range dirty {
		parts[i] = path.Path + " (" + path.State + ")"
	}
	return fmt.Sprintf("%d %s: %s", len(dirty), plural(len(dirty), "path", "paths"), strings.Join(parts, ", "))
}

func specPresence(spec string, f task.SpecFacts) string {
	switch {
	case spec == "":
		return "none recorded"
	case f.InWorktree && f.AtRevision:
		return "present in worktree and at revision"
	case f.InWorktree:
		return "present in worktree only"
	case f.AtRevision:
		return "present at revision only"
	default:
		return "missing"
	}
}

func (s *statusFlags) forgeDetails(f task.StatusFacts) []detail {
	var pr string
	checks := f.Forge.Checks
	switch {
	case s.noForge:
		pr = "unknown (--no-forge)"
	case !f.Forge.Available:
		pr = "unknown (gh unavailable)"
	case f.Forge.PR == nil:
		pr, checks = "none for "+f.Branch.Name, "unknown (no PR)"
	default:
		pr = fmt.Sprintf("#%d %s", f.Forge.PR.Number, strings.ToLower(f.Forge.PR.State))
	}
	return []detail{{"pr", pr}, {"checks", checks}}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
