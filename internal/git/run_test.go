package git_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

func TestRunReturnsTrimmedStdout(t *testing.T) {
	dir := gittest.NewRepo(t)
	repo := &git.Repo{Dir: dir}

	out, err := repo.Run(t.Context(), "rev-parse", "--show-toplevel")

	require.NoError(t, err)
	assert.Equal(t, dir, out)
}

func TestRunFailureReturnsErrorWithExitStatusAndStderr(t *testing.T) {
	repo := &git.Repo{Dir: gittest.NewRepo(t)}

	_, err := repo.Run(t.Context(), "rev-parse", "--verify", "missing")

	gitErr, ok := errors.AsType[*git.Error](err)
	require.True(t, ok, "want *git.Error, got %v", err)
	assert.Equal(t, []string{"rev-parse", "--verify", "missing"}, gitErr.Args)
	assert.Equal(t, 128, gitErr.ExitCode)
	assert.Equal(t, "fatal: Needed a single revision", gitErr.Stderr)
	assert.Equal(t, "git rev-parse --verify missing: exit status 128: fatal: Needed a single revision", err.Error())
}

func TestRunInputFeedsStdin(t *testing.T) {
	repo := &git.Repo{Dir: gittest.NewRepo(t)}

	id, err := repo.RunInput(t.Context(), strings.NewReader("hello\n"), "hash-object", "--stdin")

	require.NoError(t, err)
	assert.Regexp(t, `^[0-9a-f]{40}$`, id)
}

func TestRunWithCancelledContextReturnsCanceled(t *testing.T) {
	repo := &git.Repo{Dir: gittest.NewRepo(t)}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Run(ctx, "status")

	require.ErrorIs(t, err, context.Canceled)
}

func TestRunWritesTraceLineBeforeCommand(t *testing.T) {
	var trace strings.Builder
	repo := &git.Repo{Dir: gittest.NewRepo(t), Trace: &trace}

	_, err := repo.Run(t.Context(), "rev-parse", "--git-dir")
	require.NoError(t, err)
	_, err = repo.Run(t.Context(), "status", "--short")
	require.NoError(t, err)

	assert.Equal(t, "+ git rev-parse --git-dir\n+ git status --short\n", trace.String())
}

func TestRunWithoutTraceSucceeds(t *testing.T) {
	repo := &git.Repo{Dir: gittest.NewRepo(t)}

	_, err := repo.Run(t.Context(), "status")

	require.NoError(t, err)
}

func TestRunGHReturnsStdoutAndTraces(t *testing.T) {
	gittest.FakeGH(t, `{"number": 42}`+"\n", "", 0)
	var trace strings.Builder
	repo := &git.Repo{Dir: t.TempDir(), Trace: &trace}

	out, err := repo.RunGH(t.Context(), "pr", "view", "feature/W-001")

	require.NoError(t, err)
	assert.JSONEq(t, `{"number": 42}`, out)
	assert.Equal(t, "+ gh pr view feature/W-001\n", trace.String())
}

func TestRunGHFailureReturnsErrorWithExitStatusAndStderr(t *testing.T) {
	gittest.FakeGH(t, "", "no pull requests found for branch \"feature/W-001\"\n", 1)
	repo := &git.Repo{Dir: t.TempDir()}

	_, err := repo.RunGH(t.Context(), "pr", "view", "feature/W-001")

	gitErr, ok := errors.AsType[*git.Error](err)
	require.True(t, ok, "want *git.Error, got %v", err)
	assert.Equal(t, 1, gitErr.ExitCode)
	assert.Equal(t, `gh pr view feature/W-001: exit status 1: no pull requests found for branch "feature/W-001"`, err.Error())
	assert.NotErrorIs(t, err, git.ErrNotFound)
}

func TestRunGHWithoutGHOnPathIsNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	repo := &git.Repo{Dir: t.TempDir()}

	_, err := repo.RunGH(t.Context(), "pr", "view", "feature/W-001")

	require.ErrorIs(t, err, git.ErrNotFound)
}
