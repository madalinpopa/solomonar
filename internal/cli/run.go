package cli

import (
	"context"
	"errors"
	"flag"
	"io"
	"io/fs"
	"slices"
	"time"
)

// Env holds every process input a command may use.
type Env struct {
	Args        []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Getenv      func(string) string
	Now         func() time.Time
	Interactive bool
	Assets      fs.FS
}

type command struct {
	name        string
	summary     string
	usage       string
	description string
	run         func(ctx context.Context, env Env, args []string) (output, error)
}

func commands() []command {
	return []command{
		{
			name:        "help",
			summary:     "Print help for a command",
			usage:       "solomonar help [<command>]",
			description: "Print help for a command; '-h' on any command is equivalent.",
			run:         runHelp,
		},
		{
			name:    "version",
			summary: "Print CLI, embedded framework, and project framework versions",
			usage:   "solomonar version",
			description: "Print CLI, embedded framework, and project framework versions.\n" +
				`Outside a project the project line reads "not initialized".`,
			run: runVersion,
		},
	}
}

func findCommand(name string) (command, bool) {
	all := commands()
	i := slices.IndexFunc(all, func(c command) bool { return c.name == name })
	if i < 0 {
		return command{}, false
	}
	return all[i], true
}

// Run executes the command named in env.Args and returns the process exit code.
func Run(ctx context.Context, env Env) int {
	var flags sharedFlags
	name, out, err := dispatch(ctx, env, &flags)
	return render(env, flags.json, name, out, err)
}

func dispatch(ctx context.Context, env Env, flags *sharedFlags) (string, output, error) {
	global := newFlagSet("solomonar", flags)
	err := global.Parse(env.Args)
	if errors.Is(err, flag.ErrHelp) {
		return "help", helpOutput(topLevelHelp()), nil
	}
	if err != nil {
		return "", output{}, invalidUsage(err)
	}
	name := global.Arg(0)
	cmd, ok := findCommand(name)
	if !ok {
		return name, output{}, unknownCommand(name)
	}
	args, err := parseInterspersed(newFlagSet(name, flags), global.Args()[1:])
	if errors.Is(err, flag.ErrHelp) {
		return name, helpOutput(commandHelp(cmd)), nil
	}
	if err != nil {
		return name, output{}, invalidUsage(err)
	}
	out, err := cmd.run(ctx, env, args)
	return name, out, err
}
