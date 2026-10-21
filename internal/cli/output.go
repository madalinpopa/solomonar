package cli

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
)

type output struct {
	data     any
	text     string
	warnings []warning
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

func render(ctx context.Context, env Env, asJSON bool, command string, out output, err error) int {
	var cliErr *Error
	if err != nil {
		cliErr = asCLIError(ctx, command, err)
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
	result := envelope{OK: cliErr == nil, Command: command, Data: out.data, Warnings: out.warnings}
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
	_, _ = io.WriteString(env.Stdout, out.text)
	if cliErr != nil {
		writeHumanError(env.Stderr, cliErr)
	}
	for _, w := range out.warnings {
		fmt.Fprintf(env.Stderr, "warning: %s\n", w.Message)
	}
}

func writeHumanError(w io.Writer, cliErr *Error) {
	fmt.Fprintf(w, "error: %s (%s)\n", cliErr.Message, cliErr.Code)
	writeDetail(w, "check: ", cliErr.Check)
	writeDetail(w, "found: ", cliErr.Found)
	writeDetail(w, "fix:   ", cliErr.Fix)
	_, _ = io.WriteString(w, "nothing was written\n")
}

func writeDetail(w io.Writer, label, value string) {
	if value != "" {
		fmt.Fprintf(w, "  %s%s\n", label, value)
	}
}
