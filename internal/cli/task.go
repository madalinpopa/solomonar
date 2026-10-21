package cli

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/madalinpopa/solomonar/internal/task"
)

type planFlags struct {
	title     string
	dependsOn []string
	related   string
	dryRun    bool
}

type writeData struct {
	ID     string       `json:"id"`
	Record *task.Record `json:"record"`
	Commit string       `json:"commit"`
	Pushed bool         `json:"pushed"`
}

type dryRunData struct {
	ID     string       `json:"id"`
	Record *task.Record `json:"record"`
	DryRun bool         `json:"dry_run"`
}

type recordData struct {
	ID     string       `json:"id"`
	Record *task.Record `json:"record"`
	Commit string       `json:"commit"`
}

type registryData struct {
	Registry *task.Registry `json:"registry"`
	Commit   string         `json:"commit"`
}

// statusWidth keeps STATUS as wide as the longest status, as in UC-16.
const statusWidth = len(task.StatusCancelled)

func taskPlanCommand() command {
	return command{
		name:    "task plan",
		summary: "Plan a task under the next free ID",
		usage:   "solomonar task plan --title <title> [--depends-on <ID>]... [--related <ID>] [--json] [--dry-run]",
		description: "Add a planned task under the next free W-NNN.\n" +
			"--depends-on and --related must name existing tasks; --dry-run runs every check and writes nothing.",
		bind: func(fs *flag.FlagSet) runFunc {
			var p planFlags
			fs.StringVar(&p.title, "title", "", "")
			fs.Func("depends-on", "", func(id string) error {
				p.dependsOn = append(p.dependsOn, id)
				return nil
			})
			fs.StringVar(&p.related, "related", "", "")
			fs.BoolVar(&p.dryRun, "dry-run", false, "")
			return p.run
		},
	}
}

func (p *planFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	if len(args) > 0 {
		return output{}, unexpectedArgument(args[0])
	}
	if p.title == "" {
		return output{}, invalidUsage(errors.New("--title is required"))
	}
	store, _, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, err
	}
	var id, subject string
	var record *task.Record
	plan := func(reg *task.Registry) (string, error) {
		var err error
		id, subject, err = reg.Plan(p.title, p.dependsOn, p.related, env.Now())
		record = reg.Tasks[id]
		return subject, err
	}
	if p.dryRun {
		reg, _, err := store.Read(ctx)
		if err != nil {
			return output{}, err
		}
		if _, err := plan(reg); err != nil {
			return output{}, err
		}
		text := fmt.Sprintf("Would plan %s: %s\n  %s\n", id, p.title, subject)
		return output{data: dryRunData{ID: id, Record: record, DryRun: true}, text: text}, nil
	}
	commit, err := store.Update(ctx, plan)
	if err != nil {
		return output{}, err
	}
	return output{
		data:     writeData{ID: id, Record: record, Commit: commit.SHA, Pushed: commit.Pushed},
		text:     fmt.Sprintf("Planned %s: %s\n  %s\n", id, p.title, subject) + registryLine(store, commit) + "\n",
		warnings: outputWarnings(commit.Warnings),
	}, nil
}

// registryLine reads "  registry @ <sha>", naming the remote branch only after a push.
func registryLine(store *task.Store, commit task.Commit) string {
	line := "  registry @ " + commit.SHA
	if commit.Pushed {
		line += " · pushed " + store.Remote + "/" + strings.TrimPrefix(task.Ref, "refs/heads/")
	}
	return line
}

func taskShowCommand() command {
	return command{
		name:    "task show",
		summary: "Print the registry overview or one stored record",
		usage:   "solomonar task show [<ID>] [--json]",
		description: "Without an ID, print every task sorted by ID and the registry commit.\n" +
			"With an ID, print the stored record exactly as JSON.",
		bind: withoutFlags(runTaskShow),
	}
}

func runTaskShow(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	if len(args) > 1 {
		return output{}, unexpectedArgument(args[1])
	}
	label := "task show"
	if len(args) == 1 {
		label += " " + args[0]
	}
	store, _, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, rejectedAs(label, err)
	}
	if len(args) == 1 {
		out, err := showRecord(ctx, store, args[0])
		return out, rejectedAs(label, err)
	}
	return showRegistry(ctx, store)
}

func showRecord(ctx context.Context, store *task.Store, id string) (output, error) {
	reg, tip, err := store.Read(ctx)
	if err != nil {
		return output{}, err
	}
	record, err := reg.Lookup(id)
	if err != nil {
		return output{}, err
	}
	text, err := record.Encode()
	if err != nil {
		return output{}, err
	}
	return output{data: recordData{ID: id, Record: record, Commit: tip}, text: string(text)}, nil
}

func showRegistry(ctx context.Context, store *task.Store) (output, error) {
	reg, tip, err := store.Read(ctx)
	if err != nil {
		return output{}, err
	}
	committed, err := commitTime(ctx, store, tip)
	if err != nil {
		return output{}, err
	}
	var b strings.Builder
	table := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintf(table, "ID\t%-*s\tSTAGE\tOWNER\tNEXT\n", statusWidth, "STATUS")
	for _, id := range reg.IDs() {
		r := reg.Tasks[id]
		fmt.Fprintf(table, "%s\t%-*s\t%s\t%s\t%s\n",
			id, statusWidth, r.Status, cmp.Or(string(r.Stage), "-"), cmp.Or(r.Owner, "-"), nextColumn(r))
	}
	if err := table.Flush(); err != nil {
		return output{}, err
	}
	fmt.Fprintf(&b, "registry %s @ %s · %s\n", task.Ref, tip[:7], committed.Format(time.RFC3339))
	return output{data: registryData{Registry: reg, Commit: tip}, text: b.String()}, nil
}

func commitTime(ctx context.Context, store *task.Store, sha string) (time.Time, error) {
	out, err := store.Git.Run(ctx, "log", "-1", "--format=%ct", sha)
	if err != nil {
		return time.Time{}, fmt.Errorf("read registry commit date: %w", err)
	}
	seconds, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("read registry commit date: %w", err)
	}
	return time.Unix(seconds, 0).UTC(), nil
}

func nextColumn(r *task.Record) string {
	switch {
	case r.Status == task.StatusDone:
		return cmp.Or(r.Summary, "-")
	case r.NextAction != nil:
		return r.NextAction.Actor + ": " + r.NextAction.Action
	default:
		return "-"
	}
}
