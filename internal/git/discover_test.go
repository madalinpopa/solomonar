package git_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

func TestDiscoverFromMainCheckout(t *testing.T) {
	dir := gittest.NewRepo(t)

	repo, err := git.Discover(t.Context(), dir, nil)

	require.NoError(t, err)
	assert.Equal(t, dir, repo.Dir)
	assert.Equal(t, filepath.Join(dir, ".git"), repo.CommonDir)
}

func TestDiscoverFromSubdirectoryFindsWorktreeRoot(t *testing.T) {
	dir := gittest.NewRepo(t)
	sub := filepath.Join(dir, "a", "b")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	repo, err := git.Discover(t.Context(), sub, nil)

	require.NoError(t, err)
	assert.Equal(t, dir, repo.Dir)
}

func TestDiscoverFromWorktreeSharesCommonDir(t *testing.T) {
	mainDir := gittest.NewRepo(t)
	gittest.Commit(t, mainDir, "initial")
	featureDir := gittest.Worktree(t, mainDir, "feature/T-1")

	mainRepo, err := git.Discover(t.Context(), mainDir, nil)
	require.NoError(t, err)
	featureRepo, err := git.Discover(t.Context(), featureDir, nil)
	require.NoError(t, err)

	assert.Equal(t, mainRepo.CommonDir, featureRepo.CommonDir)
	assert.Equal(t, mainDir, mainRepo.Dir)
	assert.Equal(t, featureDir, featureRepo.Dir)
}

func TestDiscoverTracesAndKeepsTrace(t *testing.T) {
	dir := gittest.NewRepo(t)
	var trace strings.Builder

	repo, err := git.Discover(t.Context(), dir, &trace)

	require.NoError(t, err)
	assert.Same(t, &trace, repo.Trace)
	assert.Equal(t, "+ git rev-parse --path-format=absolute --show-toplevel --git-common-dir\n", trace.String())
}

func TestDiscoverOutsideRepositoryReturnsErrNotRepository(t *testing.T) {
	gittest.Isolate(t)
	dir := gittest.Resolved(t, t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))

	_, err := git.Discover(t.Context(), dir, nil)

	require.ErrorIs(t, err, git.ErrNotRepository)
	assert.Contains(t, err.Error(), dir)
}

func TestDiscoverInMissingDirectoryIsNotErrNotRepository(t *testing.T) {
	gittest.Isolate(t)
	missing := filepath.Join(t.TempDir(), "missing")

	_, err := git.Discover(t.Context(), missing, nil)

	require.Error(t, err)
	assert.NotErrorIs(t, err, git.ErrNotRepository)
}
