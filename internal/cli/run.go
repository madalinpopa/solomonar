package cli

import (
	"context"
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
	name string
	run  func(ctx context.Context, env Env, args []string) (output, error)
}

func commands() []command {
	return []command{
		{name: "version", run: runVersion},
	}
}

// Run executes the command named in env.Args and returns the process exit code.
func Run(ctx context.Context, env Env) int {
	var flags sharedFlags
	name, out, err := dispatch(ctx, env, &flags)
	return render(env, flags.json, name, out, err)
}

func dispatch(ctx context.Context, env Env, flags *sharedFlags) (string, output, error) {
	global := newFlagSet("solomonar", flags)
	if err := global.Parse(env.Args); err != nil {
		return "", output{}, invalidUsage(err)
	}
	name := global.Arg(0)
	all := commands()
	i := slices.IndexFunc(all, func(c command) bool { return c.name == name })
	if i < 0 {
		return name, output{}, unknownCommand(name)
	}
	args, err := parseInterspersed(newFlagSet(name, flags), global.Args()[1:])
	if err != nil {
		return name, output{}, invalidUsage(err)
	}
	out, err := all[i].run(ctx, env, args)
	return name, out, err
}
