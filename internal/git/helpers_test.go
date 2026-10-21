package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func newRepo(t *testing.T) string {
	t.Helper()
	isolateGit(t)
	dir := resolved(t, t.TempDir())
	gitIn(t, dir, "init", "-b", "main")
	return dir
}

func isolateGit(t *testing.T) {
	t.Helper()
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	require.NoError(t, os.WriteFile(globalConfig, nil, 0o644))
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, role := range []string{"AUTHOR", "COMMITTER"} {
		t.Setenv("GIT_"+role+"_NAME", "Test Author")
		t.Setenv("GIT_"+role+"_EMAIL", "test@example.com")
		t.Setenv("GIT_"+role+"_DATE", "2026-01-01T00:00:00Z")
	}
	_, err := Version(t.Context(), nil)
	require.NoError(t, err, "tests need git 2.40 or newer on PATH")
}

func commit(t *testing.T, dir, msg string) {
	t.Helper()
	gitIn(t, dir, "commit", "--allow-empty", "-m", msg)
}

func worktree(t *testing.T, repoDir, branch string) string {
	t.Helper()
	dir := filepath.Join(resolved(t, t.TempDir()), "worktree")
	gitIn(t, repoDir, "worktree", "add", "-b", branch, dir)
	return dir
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := (&Repo{Dir: dir}).Run(t.Context(), args...)
	require.NoError(t, err)
	return out
}

func resolved(t *testing.T, path string) string {
	t.Helper()
	resolvedPath, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)
	return resolvedPath
}
