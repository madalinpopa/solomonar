// Package gittest builds isolated Git repositories for tests.
package gittest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/madalinpopa/solomonar/internal/git"
)

// NewRepo initializes an isolated repository on branch main and returns its directory.
func NewRepo(t *testing.T) string {
	t.Helper()
	Isolate(t)
	dir := Resolved(t, t.TempDir())
	Git(t, dir, "init", "-b", "main")
	return dir
}

// NewBareRemote initializes an isolated bare repository and returns its directory.
func NewBareRemote(t *testing.T) string {
	t.Helper()
	Isolate(t)
	dir := Resolved(t, t.TempDir())
	Git(t, dir, "init", "--bare", "-b", "main")
	return dir
}

// Isolate points Git at an empty global config and a fixed identity and date.
func Isolate(t *testing.T) {
	t.Helper()
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(globalConfig, nil, 0o644); err != nil {
		t.Fatalf("write git config: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, role := range []string{"AUTHOR", "COMMITTER"} {
		t.Setenv("GIT_"+role+"_NAME", "Test Author")
		t.Setenv("GIT_"+role+"_EMAIL", "test@example.com")
		t.Setenv("GIT_"+role+"_DATE", "2026-01-01T00:00:00Z")
	}
	if _, err := git.Version(t.Context(), nil); err != nil {
		t.Fatalf("tests need git 2.40 or newer on PATH: %v", err)
	}
}

// Commit records an empty commit in dir.
func Commit(t *testing.T, dir, msg string) {
	t.Helper()
	Git(t, dir, "commit", "--allow-empty", "-m", msg)
}

// Worktree adds a worktree of repoDir on a new branch and returns its directory.
func Worktree(t *testing.T, repoDir, branch string) string {
	t.Helper()
	dir := filepath.Join(Resolved(t, t.TempDir()), "worktree")
	Git(t, repoDir, "worktree", "add", "-b", branch, dir)
	return dir
}

// Git runs git in dir and returns its trimmed stdout, failing the test on error.
func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := (&git.Repo{Dir: dir}).Run(t.Context(), args...)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return out
}

// Resolved returns path with symlinks resolved.
func Resolved(t *testing.T, path string) string {
	t.Helper()
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("resolve %s: %v", path, err)
	}
	return resolvedPath
}
