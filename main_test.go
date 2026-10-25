package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/release"
)

// shipped is the release the transcripts run as; a test binary has no VCS stamp.
const shipped = "0.9.0"

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){"solomonar": func() { os.Exit(run(context.Background(), shipped)) }})
}

func TestScript(t *testing.T) {
	if run, _ := strconv.ParseBool(os.Getenv("INTEGRATION")); !run {
		t.Skip("set INTEGRATION=true to run the transcript tests")
	}
	if _, err := git.Version(t.Context(), nil); err != nil {
		t.Fatalf("tests need git 2.40 or newer on PATH: %v", err)
	}
	assets := filepath.Join(t.TempDir(), "assets")
	if err := os.CopyFS(assets, release.Assets(os.DirFS("assets"), shipped)); err != nil {
		t.Fatalf("render assets: %v", err)
	}
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			env.Setenv("ASSETS", assets)
			return isolateGit(env)
		},
		Cmds: map[string]func(*testscript.TestScript, bool, []string){
			"exitcode": exitCode,
			"capture":  capture,
		},
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

// capture runs "capture <name>" and sets $name to the previous command's trimmed stdout.
func capture(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) != 1 {
		ts.Fatalf("usage: capture <name>")
	}
	ts.Setenv(args[0], strings.TrimSpace(ts.ReadFile("stdout")))
}
