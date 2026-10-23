package cli

import (
	"context"
	"errors"
	"flag"

	"github.com/madalinpopa/solomonar/internal/task"
)

type confirmFlags struct {
	as     string
	dryRun bool
}

func taskConfirmCommand() command {
	return command{
		name:    "task confirm",
		summary: "Confirm a task's continuation after inspecting it",
		usage:   "solomonar task confirm <ID> [--as <owner>] [--json] [--dry-run]",
		description: "Confirm that the owner inspected the task and continues it, after checking its branch,\n" +
			"recorded worktree, and recorded base. Stage changes and task done need a confirmed task.",
		bind: func(fs *flag.FlagSet) runFunc {
			var c confirmFlags
			fs.StringVar(&c.as, "as", "", "")
			fs.BoolVar(&c.dryRun, "dry-run", false, "")
			return c.run
		},
	}
}

func (c *confirmFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	switch {
	case len(args) == 0:
		return output{}, invalidUsage(errors.New("task ID is required"))
	case len(args) > 1:
		return output{}, unexpectedArgument(args[1])
	}
	id := args[0]
	as, err := actingOwner(env, c.as, "task confirm")
	if err != nil {
		return output{}, err
	}
	store, _, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, rejectedAs("task confirm "+id, err)
	}
	out, err := changeTask(ctx, store, id, "confirm", positionHeader(id), c.dryRun, func(reg *task.Registry) (task.MoveResult, []detail, error) {
		result, err := reg.ConfirmTask(ctx, store.Git, id, as, env.Now())
		return result, nil, err
	})
	return out, rejectedAs("task confirm "+id, err)
}
