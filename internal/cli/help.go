package cli

import (
	"context"
	"fmt"
	"strings"
)

const globalFlagsHelp = `Global flags:
  -C <path>       Run as if started in <path>
  --json          Machine output on stdout
  --no-color      Disable color
  -v, --verbose   Print each Git and gh command run, on stderr
  -h, --help      Help for the command
`

type helpData struct {
	Help string `json:"help"`
}

func runHelp(_ context.Context, _ Env, _ sharedFlags, args []string) (output, error) {
	name, rest := commandName(args)
	if len(rest) > 0 {
		return output{}, unexpectedArgument(rest[0])
	}
	if name == "" {
		return helpOutput(topLevelHelp()), nil
	}
	cmd, ok := findCommand(name)
	if !ok {
		return output{}, unknownCommand(name)
	}
	return helpOutput(commandHelp(cmd)), nil
}

func helpOutput(text string) output {
	return output{data: helpData{Help: text}, text: text}
}

func topLevelHelp() string {
	all := commands()
	width := 0
	for _, cmd := range all {
		width = max(width, len(cmd.name))
	}
	var b strings.Builder
	b.WriteString("Usage: solomonar [global flags] <command> [flags] [args]\n\nCommands:\n")
	for _, cmd := range all {
		fmt.Fprintf(&b, "  %-*s   %s\n", width, cmd.name, cmd.summary)
	}
	b.WriteString("\n" + globalFlagsHelp + "\nRun 'solomonar help <command>' for details.\n")
	return b.String()
}

func commandHelp(cmd command) string {
	return fmt.Sprintf("Usage: %s\n\n%s\n\nGlobal flags: see 'solomonar help'.\n", cmd.usage, cmd.description)
}
