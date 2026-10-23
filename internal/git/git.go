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

// ErrNotFound is the error of a command whose executable is not on PATH.
var ErrNotFound = exec.ErrNotFound

// OnPath reports whether program's executable is found on PATH.
func OnPath(program string) bool {
	_, err := exec.LookPath(program)
	return err == nil
}

// Error is a git or gh command that failed.
type Error struct {
	Program  string
	Args     []string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *Error) Error() string {
	message := e.Program + " " + strings.Join(e.Args, " ") + ": " + e.Err.Error()
	if e.Stderr == "" {
		return message
	}
	return message + ": " + e.Stderr
}

func (e *Error) Unwrap() error { return e.Err }

// Run runs git in the worktree and returns its stdout without trailing whitespace.
func (r *Repo) Run(ctx context.Context, args ...string) (string, error) {
	return run(ctx, "git", r.Dir, r.Trace, nil, args)
}

// RunGH runs gh in the worktree and returns its stdout without trailing whitespace.
func (r *Repo) RunGH(ctx context.Context, args ...string) (string, error) {
	return run(ctx, "gh", r.Dir, r.Trace, nil, args)
}

// RunInput is Run with stdin fed to git.
func (r *Repo) RunInput(ctx context.Context, stdin io.Reader, args ...string) (string, error) {
	return run(ctx, "git", r.Dir, r.Trace, stdin, args)
}

// RunRaw is RunInput returning stdout unchanged, for output whose trailing bytes matter.
func (r *Repo) RunRaw(ctx context.Context, stdin io.Reader, args ...string) ([]byte, error) {
	return runRaw(ctx, "git", r.Dir, r.Trace, stdin, args)
}

func run(ctx context.Context, program, dir string, trace io.Writer, stdin io.Reader, args []string) (string, error) {
	out, err := runRaw(ctx, program, dir, trace, stdin, args)
	if err != nil {
		return "", err
	}
	return strings.TrimRightFunc(string(out), unicode.IsSpace), nil
}

func runRaw(ctx context.Context, program, dir string, trace io.Writer, stdin io.Reader, args []string) ([]byte, error) {
	if trace != nil {
		if _, err := fmt.Fprintln(trace, "+ "+program+" "+strings.Join(args, " ")); err != nil {
			return nil, fmt.Errorf("write trace: %w", err)
		}
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, commandError(ctx, program, args, &stderr, err)
	}
	return stdout.Bytes(), nil
}

func commandError(ctx context.Context, program string, args []string, stderr *bytes.Buffer, err error) error {
	gitErr := &Error{Program: program, Args: args, ExitCode: -1, Stderr: strings.TrimSpace(stderr.String()), Err: err}
	if ctx.Err() != nil {
		gitErr.Err = ctx.Err()
	} else if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		gitErr.ExitCode = exitErr.ExitCode()
	}
	return gitErr
}

// Branch returns the branch HEAD points to, also an unborn one, or "" when HEAD is detached.
func (r *Repo) Branch(ctx context.Context) (string, error) {
	branch, err := r.Run(ctx, "symbolic-ref", "--short", "--quiet", "HEAD")
	if gitErr, ok := errors.AsType[*Error](err); ok && gitErr.ExitCode == 1 {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read HEAD: %w", err)
	}
	return branch, nil
}
