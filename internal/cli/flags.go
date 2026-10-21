package cli

import (
	"flag"
	"io"
)

type sharedFlags struct {
	json    bool
	verbose bool
	noColor bool
	dir     string
}

// newFlagSet registers the shared flags, keeping values already parsed as defaults.
func newFlagSet(name string, shared *sharedFlags) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&shared.json, "json", shared.json, "")
	fs.BoolVar(&shared.verbose, "v", shared.verbose, "")
	fs.BoolVar(&shared.verbose, "verbose", shared.verbose, "")
	fs.BoolVar(&shared.noColor, "no-color", shared.noColor, "")
	fs.StringVar(&shared.dir, "C", shared.dir, "")
	return fs
}

// parseInterspersed parses flags placed anywhere among the positional arguments until "--".
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positionals []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		consumed := args[:len(args)-len(rest)]
		if len(consumed) > 0 && consumed[len(consumed)-1] == "--" {
			return append(positionals, rest...), nil
		}
		if len(rest) == 0 {
			return positionals, nil
		}
		positionals = append(positionals, rest[0])
		args = rest[1:]
	}
}
