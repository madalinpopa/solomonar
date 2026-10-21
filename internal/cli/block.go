package cli

import (
	"context"
	"errors"
	"flag"
	"slices"

	"github.com/madalinpopa/solomonar/internal/task"
)

type blockFlags struct {
	block  task.Block
	as     string
	dryRun bool
}

type unblockFlags struct {
	status, next, as string
	dryRun           bool
}

func taskBlockCommand() command {
	return command{
		name:    "task block",
		summary: "Add a blocker to a task",
		usage: "solomonar task block <ID> --kind <decision|dependency|check|environment> --summary <text> [--ref <ref>]\n" +
			"                     [--next <actor>:<action>] [--status blocked] [--as <owner>] [--json] [--dry-run]",
		description: "Add a blocker, with the --status and --next shorthands of task move, as one registry commit.\n" +
			"--kind decision needs --ref <ID>/D-n; --as defaults to SOLOMONAR_OWNER.",
		bind: func(fs *flag.FlagSet) runFunc {
			var b blockFlags
			fs.StringVar(&b.block.Kind, "kind", "", "")
			fs.StringVar(&b.block.Summary, "summary", "", "")
			fs.StringVar(&b.block.Ref, "ref", "", "")
			fs.StringVar(&b.block.Next, "next", "", "")
			fs.StringVar(&b.block.Status, "status", "", "")
			fs.StringVar(&b.as, "as", "", "")
			fs.BoolVar(&b.dryRun, "dry-run", false, "")
			return b.run
		},
	}
}

func (b *blockFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	switch {
	case len(args) == 0:
		return output{}, invalidUsage(errors.New("task ID is required"))
	case len(args) > 1:
		return output{}, unexpectedArgument(args[1])
	case b.block.Kind == "":
		return output{}, invalidUsage(errors.New("--kind is required"))
	case b.block.Summary == "":
		return output{}, invalidUsage(errors.New("--summary is required"))
	}
	id := args[0]
	as, err := actingOwner(env, b.as, "task block")
	if err != nil {
		return output{}, err
	}
	store, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, err
	}
	out, err := changeTask(ctx, store, id, "block", b.dryRun, func(reg *task.Registry) (task.MoveResult, []detail, error) {
		result, err := reg.BlockTask(id, as, b.block, env.Now())
		if err != nil {
			return result, nil, err
		}
		added := reg.Tasks[id].Blockers[len(reg.Tasks[id].Blockers)-1]
		value := blockerText(added)
		if added.Ref != "" {
			value += " (" + added.Ref + ")"
		}
		return result, []detail{{"blocker", value}}, nil
	})
	return out, rejectedAs("task block "+id, err)
}

func taskUnblockCommand() command {
	return command{
		name:    "task unblock",
		summary: "Clear a blocker of a task",
		usage: "solomonar task unblock <ID> <blocker-id> [--status <prior status>] [--next <actor>:<action>]\n" +
			"                       [--as <owner>] [--json] [--dry-run]",
		description: "Clear an open blocker, with the --status and --next shorthands of task move, as one registry commit.\n" +
			"Clearing the last blocker of a blocked task needs --status <prior status>; --as defaults to SOLOMONAR_OWNER.",
		bind: func(fs *flag.FlagSet) runFunc {
			var u unblockFlags
			fs.StringVar(&u.status, "status", "", "")
			fs.StringVar(&u.next, "next", "", "")
			fs.StringVar(&u.as, "as", "", "")
			fs.BoolVar(&u.dryRun, "dry-run", false, "")
			return u.run
		},
	}
}

func (u *unblockFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	switch {
	case len(args) == 0:
		return output{}, invalidUsage(errors.New("task ID is required"))
	case len(args) == 1:
		return output{}, invalidUsage(errors.New("blocker ID is required"))
	case len(args) > 2:
		return output{}, unexpectedArgument(args[2])
	}
	id, blockerID := args[0], args[1]
	as, err := actingOwner(env, u.as, "task unblock")
	if err != nil {
		return output{}, err
	}
	store, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, err
	}
	out, err := changeTask(ctx, store, id, "unblock", u.dryRun, func(reg *task.Registry) (task.MoveResult, []detail, error) {
		var cleared task.Blocker
		if record := reg.Tasks[id]; record != nil {
			if i := slices.IndexFunc(record.Blockers, func(b task.Blocker) bool { return b.ID == blockerID }); i >= 0 {
				cleared = record.Blockers[i]
			}
		}
		result, err := reg.UnblockTask(id, as, blockerID, u.status, u.next, env.Now())
		return result, []detail{{"cleared", blockerText(cleared)}}, err
	})
	return out, rejectedAs("task unblock "+id, err)
}

func blockerText(b task.Blocker) string {
	return b.ID + " " + string(b.Kind) + ": " + b.Summary
}
