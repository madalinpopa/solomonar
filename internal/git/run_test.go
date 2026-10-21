package git

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunReturnsTrimmedStdout(t *testing.T) {
	dir := newRepo(t)
	repo := &Repo{Dir: dir}

	out, err := repo.Run(t.Context(), "rev-parse", "--show-toplevel")

	require.NoError(t, err)
	assert.Equal(t, dir, out)
}

func TestRunFailureReturnsErrorWithExitStatusAndStderr(t *testing.T) {
	repo := &Repo{Dir: newRepo(t)}

	_, err := repo.Run(t.Context(), "rev-parse", "--verify", "missing")

	gitErr, ok := errors.AsType[*Error](err)
	require.True(t, ok, "want *git.Error, got %v", err)
	assert.Equal(t, []string{"rev-parse", "--verify", "missing"}, gitErr.Args)
	assert.Equal(t, 128, gitErr.ExitCode)
	assert.Equal(t, "fatal: Needed a single revision", gitErr.Stderr)
	assert.Equal(t, "git rev-parse --verify missing: exit status 128: fatal: Needed a single revision", err.Error())
}

func TestRunInputFeedsStdin(t *testing.T) {
	repo := &Repo{Dir: newRepo(t)}

	id, err := repo.RunInput(t.Context(), strings.NewReader("hello\n"), "hash-object", "--stdin")

	require.NoError(t, err)
	assert.Regexp(t, `^[0-9a-f]{40}$`, id)
}

func TestRunWithCancelledContextReturnsCanceled(t *testing.T) {
	repo := &Repo{Dir: newRepo(t)}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Run(ctx, "status")

	require.ErrorIs(t, err, context.Canceled)
}

func TestRunWritesTraceLineBeforeCommand(t *testing.T) {
	var trace strings.Builder
	repo := &Repo{Dir: newRepo(t), Trace: &trace}

	_, err := repo.Run(t.Context(), "rev-parse", "--git-dir")
	require.NoError(t, err)
	_, err = repo.Run(t.Context(), "status", "--short")
	require.NoError(t, err)

	assert.Equal(t, "+ git rev-parse --git-dir\n+ git status --short\n", trace.String())
}

func TestRunWithoutTraceSucceeds(t *testing.T) {
	repo := &Repo{Dir: newRepo(t)}

	_, err := repo.Run(t.Context(), "status")

	require.NoError(t, err)
}
