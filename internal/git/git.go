// Package git runs the git executable.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"unicode"
)

// Repo is a Git worktree and the common Git directory shared by all its worktrees.
type Repo struct {
	Dir       string
	CommonDir string
	Trace     io.Writer
}

// Error is a git command that failed.
type Error struct {
	Args     []string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *Error) Error() string {
	message := "git " + strings.Join(e.Args, " ") + ": " + e.Err.Error()
	if e.Stderr == "" {
		return message
	}
	return message + ": " + e.Stderr
}

func (e *Error) Unwrap() error { return e.Err }

// Run runs git in the worktree and returns its stdout without trailing whitespace.
func (r *Repo) Run(ctx context.Context, args ...string) (string, error) {
	return run(ctx, r.Dir, r.Trace, nil, args)
}

// RunInput is Run with stdin fed to git.
func (r *Repo) RunInput(ctx context.Context, stdin io.Reader, args ...string) (string, error) {
	return run(ctx, r.Dir, r.Trace, stdin, args)
}

func run(ctx context.Context, dir string, trace io.Writer, stdin io.Reader, args []string) (string, error) {
	if trace != nil {
		if _, err := fmt.Fprintln(trace, "+ git "+strings.Join(args, " ")); err != nil {
			return "", fmt.Errorf("write trace: %w", err)
		}
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", commandError(ctx, args, &stderr, err)
	}
	return strings.TrimRightFunc(stdout.String(), unicode.IsSpace), nil
}

func commandError(ctx context.Context, args []string, stderr *bytes.Buffer, err error) error {
	gitErr := &Error{Args: args, ExitCode: -1, Stderr: strings.TrimSpace(stderr.String()), Err: err}
	if ctx.Err() != nil {
		gitErr.Err = ctx.Err()
	} else if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		gitErr.ExitCode = exitErr.ExitCode()
	}
	return gitErr
}
