package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

// defaultRemote and defaultMainBranch apply only when the worktree has no manifest (I §3.2).
const (
	defaultRemote     = "origin"
	defaultMainBranch = "main"
)

// openStore opens the registry store with the remote and main branch the worktree's manifest names.
func openStore(ctx context.Context, env Env, flags sharedFlags) (*task.Store, string, error) {
	repo, err := openRepo(ctx, env, flags, flags.dir, "run inside a Git repository or pass -C <path>")
	if err != nil {
		return nil, "", err
	}
	settings, err := readProject(env, repo.Dir)
	if err != nil {
		return nil, "", err
	}
	return &task.Store{Git: repo, Remote: settings.Remote}, settings.MainBranch, nil
}

func readProject(env Env, dir string) (project.Project, error) {
	framework, err := frameworkVersion(env.Assets)
	if err != nil {
		return project.Project{}, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return project.Project{}, fmt.Errorf("open worktree: %w", err)
	}
	defer root.Close()
	manifest, err := project.ReadManifest(root, framework)
	if err != nil {
		return project.Project{}, err
	}
	if manifest == nil {
		return project.Project{MainBranch: defaultMainBranch, Remote: defaultRemote}, nil
	}
	return manifest.Project, nil
}

// openRepo discovers the repository containing dir and checks the Git version.
func openRepo(ctx context.Context, env Env, flags sharedFlags, dir, fix string) (*git.Repo, error) {
	trace := traceWriter(env, flags)
	repo, err := git.Discover(ctx, dir, trace)
	if errors.Is(err, git.ErrNotRepository) {
		return nil, &Error{
			Code: "not_a_repository", Exit: 3, Message: absDir(dir) + " is not a Git repository", Fix: fix, Err: err,
		}
	}
	if err != nil {
		return nil, err
	}
	if err := checkVersion(ctx, trace); err != nil {
		return nil, err
	}
	return repo, nil
}

func checkVersion(ctx context.Context, trace io.Writer) error {
	_, err := git.Version(ctx, trace)
	if errors.Is(err, git.ErrTooOld) {
		return &Error{Code: "git_too_old", Exit: 1, Message: err.Error(), Fix: "install git 2.40 or later", Err: err}
	}
	return err
}

func traceWriter(env Env, flags sharedFlags) io.Writer {
	if flags.verbose {
		return env.Stderr
	}
	return nil
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
