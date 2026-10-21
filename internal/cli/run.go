package cli

import (
	"context"
	"errors"
	"flag"
	"io"
	"io/fs"
	"slices"
	"strings"
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

type runFunc func(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error)

type command struct {
	name        string
	summary     string
	usage       string
	description string
	bind        func(fs *flag.FlagSet) runFunc // registers the command's own flags
}

func commands() []command {
	return []command{
		{
			name:        "help",
			summary:     "Print help for a command",
			usage:       "solomonar help [<command>]",
			description: "Print help for a command; '-h' on any command is equivalent.",
			bind:        withoutFlags(runHelp),
		},
		{
			name:    "version",
			summary: "Print CLI, embedded framework, and project framework versions",
			usage:   "solomonar version",
			description: "Print CLI, embedded framework, and project framework versions.\n" +
				`Outside a project the project line reads "not initialized".`,
			bind: withoutFlags(runVersion),
		},
		taskBlockCommand(),
		taskHandoffCommand(),
		taskMoveCommand(),
		taskPlanCommand(),
		taskRegisterCommand(),
		taskShowCommand(),
		taskUnblockCommand(),
	}
}

func withoutFlags(run runFunc) func(*flag.FlagSet) runFunc {
	return func(*flag.FlagSet) runFunc { return run }
}

// commandName joins a group word such as "task" with the subcommand that follows it.
func commandName(args []string) (name string, rest []string) {
	if len(args) == 0 {
		return "", nil
	}
	name, rest = args[0], args[1:]
	isGroup := slices.ContainsFunc(commands(), func(c command) bool { return strings.HasPrefix(c.name, name+" ") })
	if isGroup && len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		return name + " " + rest[0], rest[1:]
	}
	return name, rest
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
	return render(ctx, env, flags.json, name, out, err)
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
	name, rest := commandName(global.Args())
	cmd, ok := findCommand(name)
	if !ok {
		return name, output{}, unknownCommand(name)
	}
	fs := newFlagSet(name, flags)
	run := cmd.bind(fs)
	args, err := parseInterspersed(fs, rest)
	if errors.Is(err, flag.ErrHelp) {
		return name, helpOutput(commandHelp(cmd)), nil
	}
	if err != nil {
		return name, output{}, invalidUsage(err)
	}
	out, err := run(ctx, env, *flags, args)
	return name, out, err
}
