package cli

import (
	"bytes"
	"io/fs"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
)

// runInteractive runs init in dir as a terminal session answering with input.
func runInteractive(t *testing.T, assets fs.FS, dir, input string, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	exit := Run(t.Context(), Env{
		Args:        append([]string{"-C", dir}, args...),
		Stdin:       strings.NewReader(input),
		Stdout:      &stdout,
		Stderr:      &stderr,
		Getenv:      func(string) string { return "" },
		Now:         func() time.Time { return time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC) },
		Interactive: true,
		RawMode:     func() (func(), error) { return func() {}, nil },
		Assets:      assets,
	})
	return result{exit: exit, stdout: stdout.String(), stderr: stderr.String()}
}

var redraw = regexp.MustCompile(`\x1b\[(\d+)A\r\x1b\[J`)

// screen replays the multi-select redraws in out and returns what stays on the terminal.
func screen(t *testing.T, out string) string {
	t.Helper()
	for {
		loc := redraw.FindStringSubmatchIndex(out)
		if loc == nil {
			return strings.ReplaceAll(out, "\r\n", "\n")
		}
		n, err := strconv.Atoi(out[loc[2]:loc[3]])
		require.NoError(t, err)
		kept := out[:loc[0]]
		for range n {
			kept, _, _ = strings.CutLast(strings.TrimSuffix(kept, "\n"), "\n")
			kept += "\n"
		}
		out = kept + out[loc[1]:]
	}
}

const extensionsIntro = "\nExtensions, by decision type. A choice in a stack group is recorded as your approved decision in\n" +
	"docs/decisions.md. Enter with nothing selected skips a group; add more later with\n" +
	"`solomonar extension add`.\n"

func TestInteractiveInitUC1(t *testing.T) {
	dir := committedRepo(t)
	writeFile(t, dir, "go.mod", "module example.com/ledger\n")
	gittest.PathWith(t, "claude")
	input := "\n" + "Track shared household expenses\n" + "j \r" + "opus\n" + "\n" + "\r" + " jj \r" + "y\n"

	got := runInteractive(t, catalogAssets(t), dir, input, "init")

	require.Equal(t, 0, got.exit, got.stderr)
	transcript := screen(t, got.stdout)
	prompts, rest, ok := strings.Cut(transcript, "Plan\n")
	require.True(t, ok, transcript)
	assert.Equal(t, "Project name ["+filepath.Base(dir)+"]: Current objective, one line (optional): \n"+
		"Runtimes that will run this project's sessions:\n"+
		"    [x] claude  Claude Code: CLAUDE.md import, .claude/skills, .claude/agents, state guard hook\n"+
		"    [x] codex   Codex: AGENTS.md (native), .agents/skills, .codex/agents, state guard hook\n"+
		"\nModel for the adversarial reviewer in Claude Code, different from your main session model\n"+
		"(blank = runtime default): "+
		"Model for the adversarial reviewer in Codex, different from your main session model\n"+
		"(blank = runtime default): "+
		extensionsIntro+
		"\nLanguage:\n    [x] go  Go language rules\n"+
		"\nFrontend:\n    [x] htmx   T\n    [ ] react  T\n    [x] templ  T (requires htmx)\n\n", prompts)
	assert.Contains(t, rest, "  create     .solomonar/manifest.json\n"+
		"  registry   create refs/heads/workflow-state with an empty state.json\n"+
		"Proceed? [Y/n] \nInitialized solomonar 0.1.0 in "+dir+"\n")
	manifest := manifestOf(t, dir)
	assert.Equal(t, filepath.Base(dir), manifest.Project.Name)
	assert.Equal(t, map[string]project.Runtime{"claude": {AdversarialModel: "opus"}, "codex": {}}, manifest.Runtimes)
	assert.ElementsMatch(t, []string{"go", "htmx", "templ"}, slices.Collect(maps.Keys(manifest.Extensions)))
	assert.Contains(t, tree(t, dir)["docs/README.md"], "Track shared household expenses")
}

func TestInteractiveInitAsksOnlyForMissingFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		input   string
		prompts string
	}{
		{
			name: "every choice given", input: "y\n",
			args: []string{"init", "--name", "ledger", "--objective", "Track", "--runtime", "none", "--ext", "none"},
		},
		{
			name: "name and objective asked", input: "\n\ny\n",
			args:    []string{"init", "--runtime", "claude", "--adversarial-model", "claude=opus", "--ext", "none"},
			prompts: "Project name [<base>]: Current objective, one line (optional): \n",
		},
		{
			name: "model asked", input: "\ny\n",
			args: []string{"init", "--name", "ledger", "--objective", "Track", "--runtime", "claude", "--ext", "none"},
			prompts: "Model for the adversarial reviewer in Claude Code, different from your main session model\n" +
				"(blank = runtime default): \n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := committedRepo(t)
			gittest.NoGH(t)

			got := runInteractive(t, repoAssets(t), dir, tt.input, tt.args...)

			require.Equal(t, 0, got.exit, got.stderr)
			prompts := strings.ReplaceAll(tt.prompts, "<base>", filepath.Base(dir))
			assert.True(t, strings.HasPrefix(screen(t, got.stdout), prompts+"Plan\n"), got.stdout)
			assert.Contains(t, got.stdout, "Proceed? [Y/n] \nInitialized solomonar 0.1.0 in ")
		})
	}
}

func TestInteractiveInitRerunAsksNoChoices(t *testing.T) {
	dir := initializedRepo(t)

	got := runInteractive(t, repoAssets(t), dir, "", "init")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, "Already initialized with solomonar 0.1.0; nothing to do.\n"),
		got.stdout)
}

func TestInteractiveInitNeverPromptsWithYesOrJSON(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		exit   int
		stdout string
	}{
		{name: "yes", args: []string{"init", "--yes"}, stdout: "Initialized solomonar 0.1.0 in "},
		{name: "json", args: []string{"init", "--json"}, exit: 2, stdout: `"code": "interactive_required"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := committedRepo(t)
			gittest.NoGH(t)

			got := runInteractive(t, repoAssets(t), dir, "", tt.args...)

			assert.Equal(t, tt.exit, got.exit, got.stderr)
			assert.Contains(t, got.stdout, tt.stdout)
			assert.NotContains(t, got.stdout, "Project name")
		})
	}
}

func TestInteractiveInitDeclinedWritesNothing(t *testing.T) {
	dir := committedRepo(t)
	gittest.NoGH(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runInteractive(t, repoAssets(t), dir, "n\n",
		"init", "--name", "ledger", "--objective", "Track", "--runtime", "none", "--ext", "none")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, "Plan\n"), got.stdout)
	assert.True(t, strings.HasSuffix(got.stdout,
		"  registry   create refs/heads/workflow-state with an empty state.json\n"+
			"Proceed? [Y/n] \nCancelled; nothing was written.\n"), got.stdout)
	assertNothingWritten(t, dir, before, tip)
}

func TestInteractiveInitDryRunAsksChoicesWithoutConfirmation(t *testing.T) {
	dir := committedRepo(t)
	gittest.NoGH(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runInteractive(t, repoAssets(t), dir, "\n\n\r\r", "init", "--dry-run")

	require.Equal(t, 0, got.exit, got.stderr)
	transcript := screen(t, got.stdout)
	assert.True(t, strings.HasPrefix(transcript,
		"Project name ["+filepath.Base(dir)+"]: Current objective, one line (optional): \n"+
			"Runtimes that will run this project's sessions:\n"+
			"    [ ] claude  Claude Code: CLAUDE.md import, .claude/skills, .claude/agents, state guard hook\n"+
			"    [ ] codex   Codex: AGENTS.md (native), .agents/skills, .codex/agents, state guard hook\n"+
			extensionsIntro+
			"\nLanguage:\n    [ ] go  Go language rules\n\nPlan\n"), transcript)
	assert.Contains(t, transcript, "\nWould initialize solomonar 0.1.0 in "+dir+"\n")
	assert.NotContains(t, transcript, "Proceed?")
	assertNothingWritten(t, dir, before, tip)
}

func TestInteractiveInitAsksExtensionsAgainAfterRejection(t *testing.T) {
	dir := committedRepo(t)
	gittest.NoGH(t)
	input := "\r" + "jj \r" + "\r" + " \r" + "y\n"

	got := runInteractive(t, catalogAssets(t), dir, input,
		"init", "--name", "ledger", "--objective", "Track", "--runtime", "none")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, screen(t, got.stdout),
		"\nFrontend:\n    [ ] htmx   T\n    [ ] react  T\n    [x] templ  T (requires htmx)\n"+
			"  templ requires htmx; add --ext htmx\n"+
			"\nLanguage:\n    [ ] go  Go language rules\n"+
			"\nFrontend:\n    [x] htmx   T\n    [ ] react  T\n    [x] templ  T (requires htmx)\n\nPlan\n")
	assert.ElementsMatch(t, []string{"htmx", "templ"}, slices.Collect(maps.Keys(manifestOf(t, dir).Extensions)))
}

func TestInteractiveInitFailsOnClosedInput(t *testing.T) {
	dir := committedRepo(t)
	gittest.NoGH(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runInteractive(t, repoAssets(t), dir, "\n", "init")

	assert.Equal(t, 1, got.exit)
	assert.Contains(t, got.stderr, "read answer: input closed")
	assertNothingWritten(t, dir, before, tip)
}
