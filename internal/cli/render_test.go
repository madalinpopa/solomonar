package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/task"
)

func renderResult(ctx context.Context, asJSON bool, out output, err error) result {
	var stdout, stderr bytes.Buffer
	exit := render(ctx, Env{Stdout: &stdout, Stderr: &stderr}, asJSON, "task move", out, err)
	return result{exit: exit, stdout: stdout.String(), stderr: stderr.String()}
}

func TestRejectionExitFollowsCode(t *testing.T) {
	tests := []struct {
		code    string
		exit    int
		message string
	}{
		{code: "invalid_value", exit: 2, message: "task move rejected"},
		{code: "owner_required", exit: 2, message: "task move rejected"},
		{code: "registry_invalid", exit: 1, message: "task move failed"},
		{code: "cas_exhausted", exit: 4, message: "task move rejected"},
		{code: "transition_not_allowed", exit: 3, message: "task move rejected"},
		{code: "not_initialized", exit: 3, message: "task move rejected"},
		{code: "cli_outdated", exit: 3, message: "task move rejected"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			rejection := &task.Rejection{Code: tt.code, Check: "the check", Found: "what was found", Fix: "the fix"}

			got := renderResult(t.Context(), true, output{}, fmt.Errorf("wrapped: %w", rejection))

			assert.Equal(t, tt.exit, got.exit)
			assert.JSONEq(t, fmt.Sprintf(`{
				"ok": false,
				"command": "task move",
				"error": {"code": %q, "message": %q, "check": "the check", "found": "what was found", "fix": "the fix"},
				"warnings": []
			}`, tt.code, tt.message), got.stdout)
		})
	}
}

func TestRejectionPrintsHumanError(t *testing.T) {
	rejection := &task.Rejection{Code: "cli_outdated", Check: "schema 1", Found: "schema 2", Fix: "upgrade solomonar"}

	got := renderResult(t.Context(), false, output{}, rejection)

	assert.Equal(t, 3, got.exit)
	assert.Empty(t, got.stdout)
	assert.Equal(t, "error: task move rejected (cli_outdated)\n"+
		"  check: schema 1\n"+
		"  found: schema 2\n"+
		"  fix:   upgrade solomonar\n"+
		"nothing was written\n", got.stderr)
}

func TestInterruptExitsWithSignalCode(t *testing.T) {
	tests := []struct {
		signal syscall.Signal
		exit   int
	}{
		{signal: syscall.SIGINT, exit: 130},
		{signal: syscall.SIGTERM, exit: 143},
	}
	for _, tt := range tests {
		t.Run(tt.signal.String(), func(t *testing.T) {
			ctx, stop := signal.NotifyContext(t.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			require.NoError(t, syscall.Kill(os.Getpid(), tt.signal))
			<-ctx.Done()

			got := renderResult(ctx, true, output{}, fmt.Errorf("read workflow-state: %w", ctx.Err()))

			assert.Equal(t, tt.exit, got.exit)
			assert.JSONEq(t, `{
				"ok": false,
				"command": "task move",
				"error": {"code": "interrupted", "message": "interrupted", "fix": "rerun"},
				"warnings": []
			}`, got.stdout)
		})
	}
}

func TestWarningsFollowHumanResultOnStderr(t *testing.T) {
	out := output{text: "moved W-001\n", warnings: []warning{
		{Code: "push_interrupted", Message: "push to origin interrupted; the registry commit is kept locally"},
		{Code: "no_remote", Message: "no remote configured; the registry stays local"},
	}}

	got := renderResult(t.Context(), false, out, nil)

	assert.Equal(t, 0, got.exit)
	assert.Equal(t, "moved W-001\n", got.stdout)
	assert.Equal(t, "warning: push to origin interrupted; the registry commit is kept locally\n"+
		"warning: no remote configured; the registry stays local\n", got.stderr)
}

func TestWarningsAppearInEnvelope(t *testing.T) {
	out := output{data: map[string]string{"id": "W-001"}, warnings: []warning{{Code: "no_remote", Message: "stays local"}}}

	got := renderResult(t.Context(), true, out, nil)

	assert.Equal(t, 0, got.exit)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task move",
		"data": {"id": "W-001"},
		"warnings": [{"code": "no_remote", "message": "stays local"}]
	}`, got.stdout)
}

func TestWarningsAppearAlongsideError(t *testing.T) {
	out := output{warnings: []warning{{Code: "no_remote", Message: "stays local"}}}

	got := renderResult(t.Context(), true, out, errors.New("boom"))

	assert.Equal(t, 1, got.exit)
	assert.JSONEq(t, `{
		"ok": false,
		"command": "task move",
		"error": {"code": "unexpected_error", "message": "boom"},
		"warnings": [{"code": "no_remote", "message": "stays local"}]
	}`, got.stdout)
}
