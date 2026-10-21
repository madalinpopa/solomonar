package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/task"
)

type registerFlags struct {
	registration task.Registration
	dryRun       bool
}

func taskRegisterCommand() command {
	return command{
		name:    "task register",
		summary: "Reserve a task for an orchestrator",
		usage: "solomonar task register [<ID>] --title <title> --owner <owner> [--depends-on <ID>]... [--related <ID>]\n" +
			"                        [--next <actor>:<action>] [--json] [--dry-run]",
		description: "Reserve the next free W-NNN, or claim the planned <ID>, for an orchestrator.\n" +
			"The next task may start only after the registration is confirmed; --dry-run writes nothing.",
		bind: func(fs *flag.FlagSet) runFunc {
			var r registerFlags
			fs.StringVar(&r.registration.Title, "title", "", "")
			fs.StringVar(&r.registration.Owner, "owner", "", "")
			fs.Func("depends-on", "", func(id string) error {
				r.registration.DependsOn = append(r.registration.DependsOn, id)
				return nil
			})
			fs.StringVar(&r.registration.Related, "related", "", "")
			fs.StringVar(&r.registration.Next, "next", "", "")
			fs.BoolVar(&r.dryRun, "dry-run", false, "")
			return r.run
		},
	}
}

func (r *registerFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	if len(args) > 1 {
		return output{}, unexpectedArgument(args[1])
	}
	label := "task register"
	if len(args) == 1 {
		r.registration.ID = args[0]
		label += " " + args[0]
	}
	switch {
	case r.registration.Title == "":
		return output{}, invalidUsage(errors.New("--title is required"))
	case r.registration.Owner == "":
		return output{}, invalidUsage(errors.New("--owner is required"))
	}
	store, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, err
	}
	out, err := r.register(ctx, env, store)
	return out, rejectedAs(label, err)
}

func (r *registerFlags) register(ctx context.Context, env Env, store *task.Store) (output, error) {
	var id, subject string
	var record *task.Record
	var warnings []task.Warning
	register := func(reg *task.Registry) (string, error) {
		var err error
		id, subject, warnings, err = reg.Register(r.registration, env.Now())
		record = reg.Tasks[id]
		return subject, err
	}
	if r.dryRun {
		reg, _, err := store.Read(ctx)
		if err != nil {
			return output{}, err
		}
		if _, err := register(reg); err != nil {
			return output{}, err
		}
		return output{
			data:     dryRunData{ID: id, Record: record, DryRun: true},
			text:     registrationSummary("Would register", id, record, subject),
			warnings: outputWarnings(warnings),
		}, nil
	}
	commit, err := store.Update(ctx, register)
	if err != nil {
		return output{}, err
	}
	text := registrationSummary("Registered", id, record, subject) + registryLine(store, commit) + "\n\n" +
		"Registration confirmed. The next task may start.\nCreate the worktree:\n" +
		fmt.Sprintf("  git worktree add -b %s %s %s\n", record.Branch, worktreePath(store.Git, id), defaultMainBranch)
	return output{
		data:     writeData{ID: id, Record: record, Commit: commit.SHA, Pushed: commit.Pushed},
		text:     text,
		warnings: outputWarnings(append(warnings, commit.Warnings...)),
	}, nil
}

func registrationSummary(verb, id string, record *task.Record, subject string) string {
	return fmt.Sprintf("%s %s: %s\n", verb, id, record.Title) +
		fmt.Sprintf("  status    %s / %s\n", record.Status, record.Stage) +
		fmt.Sprintf("  owner     %s\n", record.Owner) +
		fmt.Sprintf("  branch    %s (create it now)\n", record.Branch) +
		fmt.Sprintf("  next      %s: %s\n", record.NextAction.Actor, record.NextAction.Action) +
		fmt.Sprintf("  %s\n", subject)
}

// worktreePath is the sibling of the main checkout, relative only when run from the main checkout.
func worktreePath(repo *git.Repo, id string) string {
	checkout := filepath.Dir(repo.CommonDir)
	sibling := filepath.Join(filepath.Base(checkout)+"-worktrees", id)
	if repo.Dir == checkout {
		return filepath.Join("..", sibling)
	}
	return filepath.Join(filepath.Dir(checkout), sibling)
}
