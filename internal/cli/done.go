package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/madalinpopa/solomonar/internal/task"
)

type doneFlags struct {
	done   task.Done
	as     string
	dryRun bool
}

func taskDoneCommand() command {
	return command{
		name:    "task done",
		summary: "Complete a task after its merge",
		usage: "solomonar task done <ID> --merge <full-sha> --spec-ref <full-sha>:<path> --summary <text>\n" +
			"                    [--as <owner>] [--json] [--dry-run]",
		description: "Complete a task after its merge: verify the merge commit and the final spec commit with Git only,\n" +
			"then write the compact record as one registry commit. --as defaults to SOLOMONAR_OWNER.",
		bind: func(fs *flag.FlagSet) runFunc {
			var d doneFlags
			fs.StringVar(&d.done.Merge, "merge", "", "")
			fs.StringVar(&d.done.SpecRef, "spec-ref", "", "")
			fs.StringVar(&d.done.Summary, "summary", "", "")
			fs.StringVar(&d.as, "as", "", "")
			fs.BoolVar(&d.dryRun, "dry-run", false, "")
			return d.run
		},
	}
}

func (d *doneFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	switch {
	case len(args) == 0:
		return output{}, invalidUsage(errors.New("task ID is required"))
	case len(args) > 1:
		return output{}, unexpectedArgument(args[1])
	case d.done.Merge == "":
		return output{}, invalidUsage(errors.New("--merge is required"))
	case d.done.SpecRef == "":
		return output{}, invalidUsage(errors.New("--spec-ref is required"))
	case d.done.Summary == "":
		return output{}, invalidUsage(errors.New("--summary is required"))
	}
	id := args[0]
	as, err := actingOwner(env, d.as, "task done")
	if err != nil {
		return output{}, err
	}
	store, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, err
	}
	out, err := d.doneTask(ctx, env, store, id, as)
	return out, rejectedAs("task done "+id, err)
}

func (d *doneFlags) doneTask(ctx context.Context, env Env, store *task.Store, id, as string) (output, error) {
	var next string
	if !d.dryRun {
		worktree, err := task.Worktree(ctx, store.Git, "feature/"+id)
		if err != nil {
			return output{}, err
		}
		next = nextLines(store, id, worktree)
	}
	var revision string
	var record *task.Record
	out, err := changeTask(ctx, store, id, "complete", positionHeader(id), d.dryRun, func(reg *task.Registry) (task.MoveResult, []detail, error) {
		if before := reg.Tasks[id]; before != nil {
			revision = before.Revision
		}
		result, err := reg.DoneTask(ctx, store.Git, id, as, defaultMainBranch, d.done, env.Now())
		record = reg.Tasks[id]
		return result, nil, err
	})
	if err != nil {
		return output{}, err
	}
	compact, err := json.MarshalIndent(record, "  ", "  ")
	if err != nil {
		return output{}, fmt.Errorf("render compact record: %w", err)
	}
	specCommit, _, _ := strings.Cut(d.done.SpecRef, ":")
	out.text = "Verified\n" +
		fmt.Sprintf("  merge     %s is a merge commit reachable from %s\n", d.done.Merge[:8], defaultMainBranch) +
		fmt.Sprintf("  revision  %s is contained in the merge\n", revision[:8]) +
		fmt.Sprintf("  spec_ref  resolves; %s is the last spec(%s) commit merged\n", specCommit[:8], id) +
		out.text + "Compact record\n  " + string(compact) + "\n" + next
	return out, nil
}

// nextLines names the worktree to remove, in the task register form for the conventional sibling.
func nextLines(store *task.Store, id, worktree string) string {
	if worktree == "" {
		return ""
	}
	path := worktreePath(store.Git, id)
	conventional := path
	if !filepath.IsAbs(path) {
		conventional = filepath.Join(store.Git.Dir, path)
	}
	if conventional != worktree {
		path = worktree
	}
	return "Next: delete the worktree only after confirming it holds no uncommitted or unmerged work:\n" +
		"  git worktree remove " + path + "\n"
}
