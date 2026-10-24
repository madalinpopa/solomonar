package cli

import (
	"context"
	"errors"
	"fmt"
	"syscall"

	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

// Error is a failure with a stable code, the exit status, and guidance for the user.
type Error struct {
	Code    string
	Exit    int
	Message string
	Check   string
	Found   string
	Fix     string
	Err     error
}

func (e *Error) Error() string { return e.Message }

func (e *Error) Unwrap() error { return e.Err }

const exitUsage = 2

func asCLIError(ctx context.Context, command string, err error) *Error {
	if cliErr, ok := errors.AsType[*Error](err); ok {
		return cliErr
	}
	if errors.Is(err, context.Canceled) {
		return &Error{
			Code: "interrupted", Exit: interruptExit(ctx), Message: "interrupted", Fix: "rerun", Err: err,
		}
	}
	if rejection, ok := rejectionError(command, err); ok {
		return rejection
	}
	return &Error{Code: "unexpected_error", Exit: 1, Message: err.Error(), Err: err}
}

// interruptExit is 128 plus the signal number.
func interruptExit(ctx context.Context) int {
	// The signal.NotifyContext cause names the signal only in its text.
	if cause := context.Cause(ctx); cause != nil && cause.Error() == syscall.SIGTERM.String()+" signal received" {
		return 128 + int(syscall.SIGTERM)
	}
	return 128 + int(syscall.SIGINT)
}

// rejectedAs names the rejected command with its arguments, as in "task show W-001 rejected".
func rejectedAs(command string, err error) error {
	if rejection, ok := rejectionError(command, err); ok {
		return rejection
	}
	return err
}

// rejectionError finds a task or project rejection in err; both share one shape
// (architecture section 6).
func rejectionError(command string, err error) (*Error, bool) {
	if rejection, ok := errors.AsType[*task.Rejection](err); ok {
		return newRejectionError(command, *rejection, rejection), true
	}
	if rejection, ok := errors.AsType[*project.Rejection](err); ok {
		return newRejectionError(command, task.Rejection(*rejection), rejection), true
	}
	return nil, false
}

func newRejectionError(command string, rejection task.Rejection, cause error) *Error {
	exit := rejectionExit(rejection.Code)
	message := command + " rejected"
	if exit == 1 {
		message = command + " failed"
	}
	return &Error{
		Code: rejection.Code, Exit: exit, Message: message,
		Check: rejection.Check, Found: rejection.Found, Fix: rejection.Fix, Err: cause,
	}
}

func rejectionExit(code string) int {
	switch code {
	case "invalid_value", "owner_required", "unknown_runtime", "unknown_extension", "unknown_group":
		return exitUsage
	case "registry_invalid", "manifest_invalid":
		return 1
	case "cas_exhausted":
		return 4
	default:
		return 3
	}
}

func unknownCommand(name string) *Error {
	message := fmt.Sprintf("unknown command %q", name)
	if name == "" {
		message = "no command given"
	}
	return &Error{Code: "unknown_command", Exit: exitUsage, Message: message, Fix: "run solomonar help"}
}

func invalidUsage(err error) *Error {
	return &Error{Code: "invalid_usage", Exit: exitUsage, Message: err.Error(), Fix: "run solomonar help", Err: err}
}

func unexpectedArgument(arg string) *Error {
	return invalidUsage(fmt.Errorf("unexpected argument %q", arg))
}
