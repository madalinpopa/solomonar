package cli

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/madalinpopa/solomonar/internal/task"
)

type moveFlags struct {
	move   task.Move
	as     string
	dryRun bool
}

type moveData struct {
	ID      string        `json:"id"`
	From    task.Position `json:"from"`
	To      task.Position `json:"to"`
	Changed []string      `json:"changed"`
	Record  *task.Record  `json:"record"`
}

type moveWriteData struct {
	moveData `json:",inline"`

	Commit string `json:"commit"`
	Pushed bool   `json:"pushed"`
}

type moveDryRunData struct {
	moveData `json:",inline"`

	DryRun bool `json:"dry_run"`
}

type detail struct {
	label, value string
}

func taskMoveCommand() command {
	return command{
		name:    "task move",
		summary: "Apply a validated status, stage, or field change",
		usage: "solomonar task move <ID> [--as <owner>]\n" +
			"  [--status <reserved|running|blocked|paused|cancelled>] [--resume-cancelled]\n" +
			"  [--stage <setup|specification|implementation|specialized_review|adversarial_review|integration>]\n" +
			"  [--revision <full-sha>] [--spec <worktree-relative path>] [--tier <0|1|2>]\n" +
			"  [--agent <id>:<role>]... | [--agent none]\n" +
			"  [--pr <url|number>] [--next <actor>:<action>]\n" +
			"  [--json] [--dry-run]",
		description: "Apply every given field flag as one validated update and one registry commit.\n" +
			"--as defaults to SOLOMONAR_OWNER; done and complete are written only by task done.",
		bind: func(fs *flag.FlagSet) runFunc {
			var m moveFlags
			fs.StringVar(&m.as, "as", "", "")
			fs.StringVar(&m.move.Status, "status", "", "")
			fs.BoolVar(&m.move.ResumeCancelled, "resume-cancelled", false, "")
			fs.StringVar(&m.move.Stage, "stage", "", "")
			fs.StringVar(&m.move.Revision, "revision", "", "")
			fs.StringVar(&m.move.Spec, "spec", "", "")
			fs.StringVar(&m.move.Tier, "tier", "", "")
			fs.Func("agent", "", func(agent string) error {
				m.move.Agents = append(m.move.Agents, agent)
				return nil
			})
			fs.StringVar(&m.move.PR, "pr", "", "")
			fs.StringVar(&m.move.Next, "next", "", "")
			fs.BoolVar(&m.dryRun, "dry-run", false, "")
			return m.run
		},
	}
}

func (m *moveFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	switch {
	case len(args) == 0:
		return output{}, invalidUsage(errors.New("task ID is required"))
	case len(args) > 1:
		return output{}, unexpectedArgument(args[1])
	case !m.hasField():
		return output{}, invalidUsage(errors.New("at least one field flag is required"))
	}
	as := cmp.Or(m.as, env.Getenv("SOLOMONAR_OWNER"))
	if as == "" {
		return output{}, &Error{
			Code: "owner_required", Exit: exitUsage, Message: "task move needs an acting owner", Fix: "pass --as <owner>",
		}
	}
	store, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, err
	}
	out, err := m.moveTask(ctx, env, store, args[0], as)
	return out, rejectedAs("task move "+args[0], err)
}

func (m *moveFlags) hasField() bool {
	f := m.move
	return cmp.Or(f.Status, f.Stage, f.Revision, f.Spec, f.Tier, f.PR, f.Next) != "" || len(f.Agents) > 0
}

func (m *moveFlags) moveTask(ctx context.Context, env Env, store *task.Store, id, as string) (output, error) {
	roles, err := installedRoles(env.Assets)
	if err != nil {
		return output{}, err
	}
	var result task.MoveResult
	var record *task.Record
	move := func(reg *task.Registry) (string, error) {
		var err error
		result, err = reg.MoveTask(ctx, store.Git, id, as, m.move, roles, env.Now())
		record = reg.Tasks[id]
		return result.Subject, err
	}
	if m.dryRun {
		reg, _, err := store.Read(ctx)
		if err != nil {
			return output{}, err
		}
		if _, err := move(reg); err != nil {
			return output{}, err
		}
		return output{
			data:     moveDryRunData{moveData: newMoveData(id, result, record), DryRun: true},
			text:     "Would move " + moveSummary(id, result, record),
			warnings: outputWarnings(result.Warnings),
		}, nil
	}
	commit, err := store.Update(ctx, move)
	if err != nil {
		return output{}, err
	}
	text := moveSummary(id, result, record)
	if result.Subject != "" {
		text += registryLine(store, commit) + "\n"
	}
	return output{
		data:     moveWriteData{moveData: newMoveData(id, result, record), Commit: commit.SHA, Pushed: commit.Pushed},
		text:     text,
		warnings: outputWarnings(append(result.Warnings, commit.Warnings...)),
	}, nil
}

func newMoveData(id string, result task.MoveResult, record *task.Record) moveData {
	return moveData{ID: id, From: result.From, To: result.To, Changed: result.Changed, Record: record}
}

// installedRoles lists the embedded core agents until the manifest names the installed ones (step 8).
func installedRoles(assets fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(assets, "framework/agents")
	if err != nil {
		return nil, fmt.Errorf("read installed roles: %w", err)
	}
	var roles []string
	for _, entry := range entries {
		role, ok := strings.CutSuffix(entry.Name(), ".md")
		if ok && role != "orchestrator" {
			roles = append(roles, role)
		}
	}
	slices.Sort(roles)
	return roles, nil
}

func moveSummary(id string, result task.MoveResult, record *task.Record) string {
	header := id + "  " + positionText(result.From)
	if result.Subject == "" {
		return header + " (unchanged)\n"
	}
	if result.From != result.To {
		header += " → " + positionText(result.To)
	}
	var b strings.Builder
	b.WriteString(header + "\n")
	details := moveDetails(id, result, record)
	width := 0
	for _, d := range details {
		width = max(width, len(d.label))
	}
	for _, d := range details {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, d.label, d.value)
	}
	if result.To.Status == task.StatusCancelled && result.From.Status != task.StatusCancelled {
		kept := "branch feature/" + id + " and its worktree are"
		if record.PR != "" {
			kept = "branch feature/" + id + ", its worktree, and PR are"
		}
		fmt.Fprintf(&b, "  note: %s kept untouched (rules §10)\n", kept)
	}
	fmt.Fprintf(&b, "  %s\n", result.Subject)
	return b.String()
}

func positionText(p task.Position) string {
	return string(p.Status) + "/" + string(p.Stage)
}

func moveDetails(id string, result task.MoveResult, record *task.Record) []detail {
	var details []detail
	for _, field := range result.Changed {
		switch field {
		case "spec":
			details = append(details, detail{"spec", specText(record.Spec, result.SpecFound)})
		case "revision":
			details = append(details, detail{"revision", record.Revision + " (on feature/" + id + ")"})
		case "tier":
			details = append(details, detail{"tier", tierText(*record.Tier)})
		case "active_agents":
			details = append(details, detail{"agents", agentsText(record.ActiveAgents)})
		case "pr":
			details = append(details, detail{"pr", record.PR})
		case "next_action":
			details = append(details, detail{"next", record.NextAction.Actor + ": " + record.NextAction.Action})
		}
	}
	return details
}

// specText names where the spec was found: an absolute worktree path, or the short revision.
func specText(spec, found string) string {
	switch {
	case found == "":
		return spec
	case filepath.IsAbs(found):
		return spec + " (found in " + found + ")"
	default:
		return spec + " (found at " + found + ")"
	}
}

// tierText names the review stages a tier runs (rules §7).
func tierText(tier int) string {
	switch tier {
	case 0:
		return "0: no review stages"
	case 1:
		return "1: specialists only"
	default:
		return fmt.Sprintf("%d: specialists, then adversarial", tier)
	}
}

func agentsText(agents []task.Agent) string {
	if len(agents) == 0 {
		return "none"
	}
	parts := make([]string, len(agents))
	for i, agent := range agents {
		parts[i] = agent.ID + " " + agent.Role
	}
	return strings.Join(parts, ", ")
}
