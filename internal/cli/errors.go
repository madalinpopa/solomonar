package cli

import (
	"context"
	"errors"
	"fmt"
	"syscall"

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
	if rejection, ok := errors.AsType[*task.Rejection](err); ok {
		return rejectionError(command, rejection)
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

func rejectionError(command string, rejection *task.Rejection) *Error {
	exit := rejectionExit(rejection.Code)
	message := command + " rejected"
	if exit == 1 {
		message = command + " failed"
	}
	return &Error{
		Code: rejection.Code, Exit: exit, Message: message,
		Check: rejection.Check, Found: rejection.Found, Fix: rejection.Fix, Err: rejection,
	}
}

func rejectionExit(code string) int {
	switch code {
	case "invalid_value", "owner_required":
		return exitUsage
	case "registry_invalid":
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
