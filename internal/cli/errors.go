package cli

import (
	"errors"
	"fmt"
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

func asCLIError(err error) *Error {
	if cliErr, ok := errors.AsType[*Error](err); ok {
		return cliErr
	}
	return &Error{Code: "unexpected_error", Exit: 1, Message: err.Error(), Err: err}
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
