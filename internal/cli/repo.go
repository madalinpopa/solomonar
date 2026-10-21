package cli

import (
	"context"
	"errors"
	"io"
	"path/filepath"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/task"
)

// defaultRemote stands in until the manifest names the remote and main branch (step 8).
const (
	defaultRemote     = "origin"
	defaultMainBranch = "main"
)

func openStore(ctx context.Context, env Env, flags sharedFlags) (*task.Store, error) {
	var trace io.Writer
	if flags.verbose {
		trace = env.Stderr
	}
	repo, err := git.Discover(ctx, flags.dir, trace)
	if errors.Is(err, git.ErrNotRepository) {
		return nil, &Error{
			Code: "not_a_repository", Exit: 3, Message: absDir(flags.dir) + " is not a Git repository",
			Fix: "run inside a Git repository or pass -C <path>", Err: err,
		}
	}
	if err != nil {
		return nil, err
	}
	if _, err := git.Version(ctx, trace); errors.Is(err, git.ErrTooOld) {
		return nil, &Error{Code: "git_too_old", Exit: 1, Message: err.Error(), Fix: "install git 2.40 or later", Err: err}
	} else if err != nil {
		return nil, err
	}
	return &task.Store{Git: repo, Remote: defaultRemote}, nil
}

func absDir(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}

func outputWarnings(warnings []task.Warning) []warning {
	converted := make([]warning, 0, len(warnings))
	for _, w := range warnings {
		converted = append(converted, warning{Code: w.Code, Message: w.Message})
	}
	return converted
}
