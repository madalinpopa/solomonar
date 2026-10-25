package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/madalinpopa/solomonar/internal/cli"
	"github.com/madalinpopa/solomonar/internal/release"
)

//go:embed assets
var embedded embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// Restoring the default handlers on the first signal lets a second one end the process (CLI-D18).
	context.AfterFunc(ctx, stop)
	code := run(ctx, release.Version(buildVersion()))
	stop()
	os.Exit(code)
}

// buildVersion is the module version Go stamped from the VCS tag, or "" without build information.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}

func run(ctx context.Context, version string) int {
	sub, err := fs.Sub(embedded, "assets")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open embedded assets: %v\n", err)
		return 1
	}
	assets := release.Assets(sub, version)
	return cli.Run(ctx, cli.Env{
		Args:        os.Args[1:],
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Getenv:      os.Getenv,
		Now:         time.Now,
		Interactive: isTerminal(os.Stdin) && isTerminal(os.Stdout),
		RawMode:     rawMode,
		Assets:      assets,
	})
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func rawMode() (restore func(), err error) {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, fmt.Errorf("raw terminal: %w", err)
	}
	return func() { _ = term.Restore(fd, state) }, nil
}
