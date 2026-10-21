package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/madalinpopa/solomonar/internal/cli"
)

//go:embed assets
var embedded embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// Restoring the default handlers on the first signal lets a second one end the process (CLI-D18).
	context.AfterFunc(ctx, stop)
	code := run(ctx)
	stop()
	os.Exit(code)
}

func run(ctx context.Context) int {
	assets, err := fs.Sub(embedded, "assets")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open embedded assets: %v\n", err)
		return 1
	}
	return cli.Run(ctx, cli.Env{
		Args:        os.Args[1:],
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Getenv:      os.Getenv,
		Now:         time.Now,
		Interactive: isTerminal(os.Stdin) && isTerminal(os.Stdout),
		Assets:      assets,
	})
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
