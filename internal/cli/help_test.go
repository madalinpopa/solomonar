package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wantTopLevelHelp = `Usage: solomonar [global flags] <command> [flags] [args]

Commands:
  help               Print help for a command
  init               Install the framework core and create the registry
  upgrade            Move the project to this CLI's framework version
  version            Print the CLI and project framework versions
  extension list     List catalog and project extensions
  extension add      Install catalog or project extensions
  extension remove   Remove catalog or deactivate project extensions
  task block         Add a blocker to a task
  task confirm       Confirm a task's continuation after inspecting it
  task done          Complete a task after its merge
  task handoff       Hand off a task to a new owner
  task migrate       Rewrite a schema-1 registry as one file per task
  task move          Apply a validated status, stage, or field change
  task plan          Plan a task under the next free ID
  task register      Reserve a task for an orchestrator
  task show          Print the registry overview or one stored record
  task status        Show a task's record, Git and forge facts, drift, and allowed next commands
  task unblock       Clear a blocker of a task

Global flags:
  -C <path>       Run as if started in <path>
  --json          Machine output on stdout
  --no-color      Disable color
  -v, --verbose   Print each Git and gh command run, on stderr
  -h, --help      Help for the command

Run 'solomonar help <command>' for details.
`

func TestHelpPrintsTopLevelHelp(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := runCLI(t, repoAssets(t), tt.args...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, wantTopLevelHelp, got.stdout)
		})
	}
}

func TestCommandHelpIsTheSameEitherWay(t *testing.T) {
	t.Parallel()

	viaHelp := runCLI(t, repoAssets(t), "help", "version")
	viaFlag := runCLI(t, repoAssets(t), "version", "-h")

	require.Equal(t, 0, viaHelp.exit, viaHelp.stderr)
	require.Equal(t, 0, viaFlag.exit, viaFlag.stderr)
	assert.Contains(t, viaHelp.stdout, "Usage: solomonar version\n")
	assert.Equal(t, viaHelp.stdout, viaFlag.stdout)
}

func TestHelpFlagWinsOverExtraArguments(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "version", "extra", "-h")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "Usage: solomonar version\n")
}

func TestHelpForUpgrade(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "upgrade", "-h")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Usage: solomonar upgrade [--keep <path>]... [--take-new <path>]... [-y|--yes] [--dry-run] [--json]\n\n"+
		"Update the managed files to the framework embedded in this CLI, with the choices the manifest\n"+
		"records, and remove the managed files no longer offered. --keep keeps the project's version of a\n"+
		"conflicting file, --take-new takes the new one; --dry-run prints the plan and writes nothing.\n"+
		"Decision documents, project extensions, and the registry are never touched.\n\n"+
		"Global flags: see 'solomonar help'.\n", got.stdout)
}

func TestHelpForHelpCommand(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "help", "-h")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, "Usage: solomonar help [<command>]\n")
}

func TestHelpForUnknownCommandExitsTwo(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "help", "nope")

	assert.Equal(t, 2, got.exit)
	assert.Contains(t, got.stderr, "error: unknown command \"nope\" (unknown_command)")
}

func TestHelpWithTwoArgumentsExitsTwo(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "help", "a", "b")

	assert.Equal(t, 2, got.exit)
	assert.Contains(t, got.stderr, "error: unexpected argument \"b\" (invalid_usage)")
}

func TestHelpJSONWrapsTheText(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "--json", "help", "version")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "help",
		"data": {"help": "Usage: solomonar version\n\nPrint the CLI and project framework versions.\nOutside a project the project line reads \"not initialized\".\n\nGlobal flags: see 'solomonar help'.\n"},
		"warnings": []
	}`, got.stdout)
}

func TestCommandHelpFlagJSONNamesTheCommand(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "version", "--json", "-h")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, got.stdout, `"command": "version"`)
}

func TestHelpForExtensionListThroughTheAlias(t *testing.T) {
	t.Parallel()

	viaHelp := runCLI(t, repoAssets(t), "help", "ext", "list")
	viaFlag := runCLI(t, repoAssets(t), "extension", "list", "-h")
	viaJSON := runCLI(t, repoAssets(t), "--json", "ext", "list", "-h")

	require.Equal(t, 0, viaHelp.exit, viaHelp.stderr)
	assert.Equal(t, "Usage: solomonar extension list [--group <group>] [--kind <skill|agent>] [--installed] [--json]\n\n"+
		"List the catalog extensions and the project extensions in docs/solomonar/extensions/, with\n"+
		"their status, decision, and detection reason; 'ext' is an alias for 'extension'. Read-only.\n\n"+
		"Global flags: see 'solomonar help'.\n", viaHelp.stdout)
	assert.Equal(t, viaHelp.stdout, viaFlag.stdout)
	assert.Contains(t, viaJSON.stdout, `"command": "extension list"`)
}

func TestHelpForExtensionAdd(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "help", "ext", "add")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "Usage: solomonar extension add <name>... [--decision <ID>] [--rationale <text>] [--keep <path>]...\n"+
		"                               [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]\n\n"+
		"Install catalog extensions into docs/solomonar/extensions/, or activate the project extensions\n"+
		"there, with their runtime stubs, AGENTS.md line, and manifest entry. A changed stack group records\n"+
		"a decision: the approved --decision entry, or a new P/D-n entry with --rationale. --keep or\n"+
		"--overwrite resolves a conflicting file; --dry-run prints the plan and writes nothing.\n\n"+
		"Global flags: see 'solomonar help'.\n", got.stdout)
}

func TestExtensionGroupAloneIsUnknown(t *testing.T) {
	t.Parallel()

	got := runCLI(t, repoAssets(t), "ext")

	assert.Equal(t, 2, got.exit)
	assert.Contains(t, got.stderr, "error: unknown command \"extension\" (unknown_command)")
}
