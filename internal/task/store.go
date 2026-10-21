package task

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/madalinpopa/solomonar/internal/git"
)

// Ref is the branch that holds state.json.
const Ref = "refs/heads/workflow-state"

const (
	maxSwapAttempts = 5
	pushTimeout     = 30 * time.Second
	initSubject     = "state: initialize registry"
)

// Store reads and writes the registry on Ref with compare-and-swap (AD-5).
type Store struct {
	Git    *git.Repo
	Remote string
}

// Commit is a registry commit and the outcome of pushing it.
type Commit struct {
	SHA      string
	Pushed   bool
	Warnings []Warning
}

// Warning is a problem that did not fail the write.
type Warning struct {
	Code    string
	Message string
}

// Read returns the registry at the tip of Ref and the tip itself.
func (s *Store) Read(ctx context.Context) (*Registry, string, error) {
	tip, err := s.tip(ctx)
	if err != nil {
		return nil, "", err
	}
	if tip == "" {
		return nil, "", &Rejection{
			Code:  "not_initialized",
			Check: "the registry ref " + Ref + " exists",
			Found: Ref + " is missing",
			Fix:   "solomonar init",
		}
	}
	data, err := s.blob(ctx, tip)
	if err != nil {
		return nil, "", err
	}
	reg, err := Decode(data)
	if err != nil {
		return nil, "", err
	}
	return reg, tip, nil
}

// ReadSchema returns the schema version at the tip of Ref without decoding the registry strictly.
func (s *Store) ReadSchema(ctx context.Context) (schema int, present bool, err error) {
	tip, err := s.tip(ctx)
	if err != nil || tip == "" {
		return 0, false, err
	}
	data, err := s.blob(ctx, tip)
	if err != nil {
		return 0, true, err
	}
	schema, err = SchemaOf(data)
	return schema, true, err
}

func (s *Store) blob(ctx context.Context, tip string) ([]byte, error) {
	data, err := s.Git.Run(ctx, "cat-file", "blob", tip+":state.json")
	if err != nil {
		return nil, fmt.Errorf("read workflow-state: %w", err)
	}
	return []byte(data), nil
}

// Create writes an empty registry as the root commit of Ref and fails if Ref exists.
func (s *Store) Create(ctx context.Context) (Commit, error) {
	tip, err := s.tip(ctx)
	if err != nil {
		return Commit{}, err
	}
	if tip != "" {
		return Commit{}, fmt.Errorf("create workflow-state: %s already exists", Ref)
	}
	sha, err := s.writeCommit(ctx, NewRegistry(), initSubject, "")
	if err != nil {
		return Commit{}, err
	}
	// An empty old value makes update-ref fail when the ref exists, so a concurrent init is never overwritten.
	if err := s.swap(ctx, sha, ""); err != nil {
		return Commit{}, fmt.Errorf("create workflow-state: %w", err)
	}
	return s.push(ctx, sha), nil
}

// Update runs fn on the tip registry and swaps in the result, re-running fn on a lost swap.
// An empty subject from fn means no change: Update writes nothing and returns the current tip.
func (s *Store) Update(ctx context.Context, fn func(*Registry) (subject string, err error)) (Commit, error) {
	for range maxSwapAttempts {
		reg, old, err := s.Read(ctx)
		if err != nil {
			return Commit{}, err
		}
		subject, err := fn(reg)
		if err != nil {
			return Commit{}, err
		}
		if subject == "" {
			return Commit{SHA: old}, nil
		}
		sha, err := s.writeCommit(ctx, reg, subject, old)
		if err != nil {
			return Commit{}, err
		}
		swapErr := s.swap(ctx, sha, old)
		if swapErr == nil {
			return s.push(ctx, sha), nil
		}
		if ctx.Err() != nil {
			return Commit{}, fmt.Errorf("update workflow-state: %w", swapErr)
		}
		current, err := s.tip(ctx)
		if err != nil {
			return Commit{}, err
		}
		if current == old {
			return Commit{}, fmt.Errorf("update workflow-state: %w", swapErr)
		}
	}
	return Commit{}, &Rejection{
		Code:  "cas_exhausted",
		Check: fmt.Sprintf("the registry compare-and-swap succeeds within %d attempts", maxSwapAttempts),
		Found: fmt.Sprintf("%s moved during each of %d attempts", Ref, maxSwapAttempts),
		Fix:   "retry the command; no state was written",
	}
}

// tip returns the commit Ref points to, or "" when Ref does not exist.
func (s *Store) tip(ctx context.Context) (string, error) {
	out, err := s.Git.Run(ctx, "rev-parse", "--verify", "--quiet", Ref)
	// With --quiet, a missing ref exits 1 without output.
	if gitErr, ok := errors.AsType[*git.Error](err); ok && gitErr.ExitCode == 1 && ctx.Err() == nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read workflow-state: %w", err)
	}
	return out, nil
}

func (s *Store) writeCommit(ctx context.Context, reg *Registry, subject, parent string) (string, error) {
	data, err := reg.Encode()
	if err != nil {
		return "", err
	}
	blob, err := s.Git.RunInput(ctx, bytes.NewReader(data), "hash-object", "-w", "--stdin")
	if err != nil {
		return "", fmt.Errorf("write state.json: %w", err)
	}
	tree, err := s.Git.RunInput(ctx, strings.NewReader("100644 blob "+blob+"\tstate.json\n"), "mktree")
	if err != nil {
		return "", fmt.Errorf("write registry tree: %w", err)
	}
	args := []string{"commit-tree", tree, "-m", subject}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	sha, err := s.Git.Run(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("write registry commit: %w", err)
	}
	return sha, nil
}

// swap moves Ref from old to sha unless ctx is already done.
func (s *Store) swap(ctx context.Context, sha, old string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// The swap is the commit point (CLI-D18): once started it runs to completion, so the outcome is known.
	_, err := s.Git.Run(context.WithoutCancel(ctx), "update-ref", Ref, sha, old)
	return err
}

func (s *Store) push(ctx context.Context, sha string) Commit {
	warning := s.pushWarning(ctx)
	if warning == nil {
		return Commit{SHA: sha, Pushed: true}
	}
	return Commit{SHA: sha, Warnings: []Warning{*warning}}
}

func (s *Store) pushWarning(ctx context.Context) *Warning {
	if s.Remote == "" {
		return new(noRemote(s.Remote))
	}
	if _, err := s.Git.Run(ctx, "remote", "get-url", s.Remote); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return new(s.pushInterrupted())
		}
		return new(noRemote(s.Remote))
	}
	return s.pushRef(ctx)
}

func (s *Store) pushRef(ctx context.Context) *Warning {
	pushCtx, cancel := context.WithTimeout(ctx, pushTimeout)
	defer cancel()
	_, err := s.Git.Run(pushCtx, "push", s.Remote, Ref+":"+Ref)
	switch {
	case err == nil:
		return nil
	case errors.Is(ctx.Err(), context.Canceled):
		return new(s.pushInterrupted())
	default:
		return &Warning{
			Code:    "push_failed",
			Message: fmt.Sprintf("git push %s workflow-state failed: %s", s.Remote, pushFailure(err)),
		}
	}
}

func (s *Store) pushInterrupted() Warning {
	return Warning{
		Code:    "push_interrupted",
		Message: fmt.Sprintf("push to %s interrupted; the registry commit is kept locally", s.Remote),
	}
}

func noRemote(remote string) Warning {
	if remote == "" {
		return Warning{Code: "no_remote", Message: "no remote configured; the registry stays local"}
	}
	return Warning{Code: "no_remote", Message: fmt.Sprintf("no remote %q; the registry stays local", remote)}
}

func pushFailure(err error) string {
	if gitErr, ok := errors.AsType[*git.Error](err); ok && gitErr.Stderr != "" {
		return gitErr.Stderr
	}
	return err.Error()
}
