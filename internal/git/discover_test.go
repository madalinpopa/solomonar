package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverFromMainCheckout(t *testing.T) {
	dir := newRepo(t)

	repo, err := Discover(t.Context(), dir, nil)

	require.NoError(t, err)
	assert.Equal(t, dir, repo.Dir)
	assert.Equal(t, filepath.Join(dir, ".git"), repo.CommonDir)
}

func TestDiscoverFromSubdirectoryFindsWorktreeRoot(t *testing.T) {
	dir := newRepo(t)
	sub := filepath.Join(dir, "a", "b")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	repo, err := Discover(t.Context(), sub, nil)

	require.NoError(t, err)
	assert.Equal(t, dir, repo.Dir)
}

func TestDiscoverFromWorktreeSharesCommonDir(t *testing.T) {
	mainDir := newRepo(t)
	commit(t, mainDir, "initial")
	featureDir := worktree(t, mainDir, "feature/T-1")

	mainRepo, err := Discover(t.Context(), mainDir, nil)
	require.NoError(t, err)
	featureRepo, err := Discover(t.Context(), featureDir, nil)
	require.NoError(t, err)

	assert.Equal(t, mainRepo.CommonDir, featureRepo.CommonDir)
	assert.Equal(t, mainDir, mainRepo.Dir)
	assert.Equal(t, featureDir, featureRepo.Dir)
}

func TestDiscoverTracesAndKeepsTrace(t *testing.T) {
	dir := newRepo(t)
	var trace strings.Builder

	repo, err := Discover(t.Context(), dir, &trace)

	require.NoError(t, err)
	assert.Same(t, &trace, repo.Trace)
	assert.Equal(t, "+ git rev-parse --path-format=absolute --show-toplevel --git-common-dir\n", trace.String())
}

func TestDiscoverOutsideRepositoryReturnsErrNotRepository(t *testing.T) {
	isolateGit(t)
	dir := resolved(t, t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))

	_, err := Discover(t.Context(), dir, nil)

	require.ErrorIs(t, err, ErrNotRepository)
	assert.Contains(t, err.Error(), dir)
}

func TestDiscoverInMissingDirectoryIsNotErrNotRepository(t *testing.T) {
	isolateGit(t)
	missing := filepath.Join(t.TempDir(), "missing")

	_, err := Discover(t.Context(), missing, nil)

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotRepository)
}
