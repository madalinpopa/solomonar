package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/madalinpopa/solomonar/internal/git"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){"solomonar": main})
}

func TestScript(t *testing.T) {
	if run, _ := strconv.ParseBool(os.Getenv("INTEGRATION")); !run {
		t.Skip("set INTEGRATION=true to run the transcript tests")
	}
	if _, err := git.Version(t.Context(), nil); err != nil {
		t.Fatalf("tests need git 2.40 or newer on PATH: %v", err)
	}
	assets, err := filepath.Abs("assets")
	if err != nil {
		t.Fatalf("locate assets: %v", err)
	}
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			env.Setenv("ASSETS", assets)
			return isolateGit(env)
		},
		Cmds:                map[string]func(*testscript.TestScript, bool, []string){"exitcode": exitCode},
		RequireExplicitExec: true,
		RequireUniqueNames:  true,
	})
}

// isolateGit points Git at an empty global config and a fixed identity and date.
func isolateGit(env *testscript.Env) error {
	env.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	env.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, role := range []string{"AUTHOR", "COMMITTER"} {
		env.Setenv("GIT_"+role+"_NAME", "Test Author")
		env.Setenv("GIT_"+role+"_EMAIL", "test@example.com")
		env.Setenv("GIT_"+role+"_DATE", "2026-01-01T00:00:00Z")
	}
	return nil
}

// exitCode runs "exitcode <code> <command> [args...]" and fails unless the command exits with code.
func exitCode(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) < 2 {
		ts.Fatalf("usage: exitcode <code> <command> [args...]")
	}
	want, err := strconv.Atoi(args[0])
	ts.Check(err)
	got := 0
	if err := ts.Exec(args[1], args[2:]...); err != nil {
		exitErr, ok := errors.AsType[*exec.ExitError](err)
		if !ok {
			ts.Fatalf("%v", err)
		}
		got = exitErr.ExitCode()
	}
	if got != want {
		ts.Fatalf("exit code %d, want %d", got, want)
	}
}
