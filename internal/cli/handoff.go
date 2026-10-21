package cli

import (
	"context"
	"errors"
	"flag"

	"github.com/madalinpopa/solomonar/internal/task"
)

type handoffFlags struct {
	owner, next string
	dryRun      bool
}

func taskHandoffCommand() command {
	return command{
		name:    "task handoff",
		summary: "Hand off a task to a new owner",
		usage:   "solomonar task handoff <ID> --owner <new-owner> [--next <actor>:<action>] [--json] [--dry-run]",
		description: "Hand off a task to a new owner, on the user's instruction, as one registry commit.\n" +
			"It takes no --as and ignores SOLOMONAR_OWNER; the previous owner must stop writing the record.",
		bind: func(fs *flag.FlagSet) runFunc {
			var h handoffFlags
			fs.StringVar(&h.owner, "owner", "", "")
			fs.StringVar(&h.next, "next", "", "")
			fs.BoolVar(&h.dryRun, "dry-run", false, "")
			return h.run
		},
	}
}

func (h *handoffFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	switch {
	case len(args) == 0:
		return output{}, invalidUsage(errors.New("task ID is required"))
	case len(args) > 1:
		return output{}, unexpectedArgument(args[1])
	case h.owner == "":
		return output{}, invalidUsage(errors.New("--owner is required"))
	}
	id := args[0]
	store, _, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, rejectedAs("task handoff "+id, err)
	}
	var previous string
	header := func(result task.MoveResult) string {
		return id + "  owner " + previous + " → " + h.owner + " (" + positionText(result.To) + ")"
	}
	out, err := changeTask(ctx, store, id, "hand off", header, h.dryRun, func(reg *task.Registry) (task.MoveResult, []detail, error) {
		if record := reg.Tasks[id]; record != nil {
			previous = record.Owner
		}
		result, err := reg.HandoffTask(id, h.owner, h.next, env.Now())
		return result, nil, err
	})
	if err == nil && !h.dryRun {
		out.text += "  The previous owner must stop writing this record.\n"
	}
	return out, rejectedAs("task handoff "+id, err)
}
