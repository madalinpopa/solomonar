package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wantTopLevelHelp = `Usage: solomonar [global flags] <command> [flags] [args]

Commands:
  help            Print help for a command
  version         Print CLI, embedded framework, and project framework versions
  task move       Apply a validated status, stage, or field change
  task plan       Plan a task under the next free ID
  task register   Reserve a task for an orchestrator
  task show       Print the registry overview or one stored record

Global flags:
  -C <path>       Run as if started in <path>
  --json          Machine output on stdout
  --no-color      Disable color
  -v, --verbose   Print each Git and gh command run, on stderr
  -h, --help      Help for the command

Run 'solomonar help <command>' for details.
`

func TestHelpPrintsTopLevelHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "help command", args: []string{"help"}},
		{name: "short flag", args: []string{"-h"}},
		{name: "long flag", args: []string{"--help"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runCLI(t, repoAssets(t), tt.args...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, wantTopLevelHelp, got.stdout)
		})
	}
}

func TestCommandHelpIsTheSameEitherWay(t *testing.T) {
	viaHelp := runCLI(t, repoAssets(t), "help", "version")
	viaFlag := runCLI(t, repoAssets(t), "version", "-h")

	require.Equal(t, 0, viaHelp.exit, viaHelp.stderr)
	require.Equal(t, 0, viaFlag.exit, viaFlag.stderr)
	assert.Contains(t, viaHelp.stdout, "Usage: solomonar version\n")
	assert.Equal(t, viaHelp.stdout, viaFlag.stdout)
}

func TestHelpFlagWinsOverExtraArguments(t *testing.T) {
	got := runCLI(t, repoAssets(t), "version", "extra", "-h")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "Usage: solomonar version\n")
}

func TestHelpForHelpCommand(t *testing.T) {
	got := runCLI(t, repoAssets(t), "help", "-h")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "Usage: solomonar help [<command>]\n")
}

func TestHelpForUnknownCommandExitsTwo(t *testing.T) {
	got := runCLI(t, repoAssets(t), "help", "nope")

	assert.Equal(t, 2, got.exit)
	assert.Contains(t, got.stderr, "error: unknown command \"nope\" (unknown_command)")
}

func TestHelpWithTwoArgumentsExitsTwo(t *testing.T) {
	got := runCLI(t, repoAssets(t), "help", "a", "b")

	assert.Equal(t, 2, got.exit)
	assert.Contains(t, got.stderr, "error: unexpected argument \"b\" (invalid_usage)")
}

func TestHelpJSONWrapsTheText(t *testing.T) {
	got := runCLI(t, repoAssets(t), "--json", "help", "version")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "help",
		"data": {"help": "Usage: solomonar version\n\nPrint CLI, embedded framework, and project framework versions.\nOutside a project the project line reads \"not initialized\".\n\nGlobal flags: see 'solomonar help'.\n"},
		"warnings": []
	}`, got.stdout)
}

func TestCommandHelpFlagJSONNamesTheCommand(t *testing.T) {
	got := runCLI(t, repoAssets(t), "version", "--json", "-h")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, `"command": "version"`)
}
