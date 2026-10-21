package cli

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
)

type output struct {
	data any
	text string
}

type envelope struct {
	OK       bool       `json:"ok"`
	Command  string     `json:"command"`
	Data     any        `json:"data,omitzero"`
	Error    *errorBody `json:"error,omitzero"`
	Warnings []warning  `json:"warnings"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Check   string `json:"check,omitempty"`
	Found   string `json:"found,omitempty"`
	Fix     string `json:"fix,omitempty"`
}

type warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func render(env Env, asJSON bool, command string, out output, err error) int {
	var cliErr *Error
	if err != nil {
		cliErr = asCLIError(err)
	}
	if asJSON {
		writeJSON(env, command, out, cliErr)
	} else {
		writeHuman(env, out, cliErr)
	}
	if cliErr != nil {
		return cliErr.Exit
	}
	return 0
}

func writeJSON(env Env, command string, out output, cliErr *Error) {
	result := envelope{OK: cliErr == nil, Command: command, Data: out.data, Warnings: []warning{}}
	if cliErr != nil {
		result.Data = nil
		result.Error = &errorBody{
			Code: cliErr.Code, Message: cliErr.Message, Check: cliErr.Check, Found: cliErr.Found, Fix: cliErr.Fix,
		}
	}
	if err := json.MarshalWrite(env.Stdout, result, jsontext.WithIndent("  ")); err != nil {
		fmt.Fprintf(env.Stderr, "error: write json output: %v\n", err)
		return
	}
	_, _ = io.WriteString(env.Stdout, "\n")
}

func writeHuman(env Env, out output, cliErr *Error) {
	if cliErr == nil {
		_, _ = io.WriteString(env.Stdout, out.text)
		return
	}
	fmt.Fprintf(env.Stderr, "error: %s (%s)\n", cliErr.Message, cliErr.Code)
	writeDetail(env.Stderr, "check: ", cliErr.Check)
	writeDetail(env.Stderr, "found: ", cliErr.Found)
	writeDetail(env.Stderr, "fix:   ", cliErr.Fix)
	_, _ = io.WriteString(env.Stderr, "nothing was written\n")
}

func writeDetail(w io.Writer, label, value string) {
	if value != "" {
		fmt.Fprintf(w, "  %s%s\n", label, value)
	}
}
