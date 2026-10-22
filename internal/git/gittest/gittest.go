// Package gittest builds isolated Git repositories for tests.
package gittest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madalinpopa/solomonar/internal/git"
)

// Main points Git at an empty global config and a fixed identity and date, then runs the tests.
// Setting them once for the test binary lets tests run in parallel.
func Main(m *testing.M) {
	// os.Setenv fails only on a key or value holding '=' or NUL.
	_ = os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, role := range []string{"AUTHOR", "COMMITTER"} {
		_ = os.Setenv("GIT_"+role+"_NAME", "Test Author")
		_ = os.Setenv("GIT_"+role+"_EMAIL", "test@example.com")
		_ = os.Setenv("GIT_"+role+"_DATE", "2026-01-01T00:00:00Z")
	}
	m.Run()
}

// NewRepo initializes a repository on branch main and returns its directory.
func NewRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := Resolved(t, t.TempDir())
	Git(t, dir, "init", "-b", "main")
	return dir
}

// NewBareRemote initializes a bare repository and returns its directory.
func NewBareRemote(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := Resolved(t, t.TempDir())
	Git(t, dir, "init", "--bare", "-b", "main")
	return dir
}

func requireGit(t *testing.T) {
	t.Helper()
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

// FakeGH sets PATH to a gh script that prints stdout and stderr and exits with code, next to git.
func FakeGH(t *testing.T, stdout, stderr string, code int) {
	t.Helper()
	dir := NoGH(t)
	writeScript(t, filepath.Join(dir, "gh"),
		fmt.Sprintf("printf '%%s' %s\nprintf '%%s' %s >&2\nexit %d", quote(stdout), quote(stderr), code))
}

// NoGH sets PATH to a directory holding only a git wrapper, so gh is not found, and returns it.
func NoGH(t *testing.T) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("find git: %v", err)
	}
	gitPath, err = filepath.Abs(gitPath)
	if err != nil {
		t.Fatalf("find git: %v", err)
	}
	dir := t.TempDir()
	writeScript(t, filepath.Join(dir, "git"), "exec "+quote(gitPath)+` "$@"`)
	t.Setenv("PATH", dir)
	return dir
}

// PathWith sets PATH to a git wrapper plus an executable that does nothing per name, and returns it.
func PathWith(t *testing.T, names ...string) string {
	t.Helper()
	dir := NoGH(t)
	for _, name := range names {
		writeScript(t, filepath.Join(dir, name), "exit 0")
	}
	return dir
}

func writeScript(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil { //nolint:gosec // the script must be executable
		t.Fatalf("write %s: %v", path, err)
	}
}

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
