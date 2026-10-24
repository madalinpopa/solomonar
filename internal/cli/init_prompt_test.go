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

// repoHeader is the header of an interactive init in dir on branch main.
func repoHeader(dir string) string {
	return "solomonar " + shipped + " · " + dir + " (git, branch main)\n\n"
}

const extensionsIntro = "\nExtensions, by decision type. A choice in a stack group is recorded as your approved decision in\n" +
	"docs/decisions.md. Enter with nothing selected skips a group; add more later with\n" +
	"`solomonar extension add`.\n"

func TestInteractiveInitUC1(t *testing.T) {
	dir := committedRepo(t)
	writeFile(t, dir, "go.mod", "module example.com/ledger\n")
	gittest.PathWith(t, "claude")
	input := "\n" + "Track shared household expenses\n" + "j \r" + "opus\n" + "\n" + "\r" + " jj \r" + "\r" + "y\n"

	got := runInteractive(t, catalogAssets(t), dir, input, "init")

	require.Equal(t, 0, got.exit, got.stderr)
	transcript := screen(t, got.stdout)
	prompts, rest, ok := strings.Cut(transcript, "Plan\n")
	require.True(t, ok, transcript)
	assert.Equal(t, repoHeader(dir)+"Project name ["+filepath.Base(dir)+"]: Current objective, one line (optional): \n"+
		"Runtimes that will run this project's sessions:\n"+
		"    [x] claude   Claude Code: CLAUDE.md import, .claude/skills, .claude/agents, state guard hook\n"+
		"    [x] codex    Codex: AGENTS.md (native), .agents/skills, .codex/agents, state guard hook\n"+
		"    [ ] copilot  GitHub Copilot CLI: AGENTS.md (native), .github/skills, .github/agents\n"+
		"\nModel for the adversarial reviewer in Claude Code, different from your main session model\n"+
		"(blank = runtime default): "+
		"Model for the adversarial reviewer in Codex, different from your main session model\n"+
		"(blank = runtime default): "+
		extensionsIntro+
		"\nLanguage:\n    [x] go  Go language rules\n"+
		"\nFrontend:\n    [x] htmx   T\n    [ ] react  T\n    [x] templ  T (requires htmx)\n"+
		"\nWorkflow:\n    [ ] extension-authoring  Evidence-based extension authoring\n\n", prompts)
	assert.Contains(t, rest, "  registry\n    create     refs/heads/workflow-state with an empty registry\n"+
		"  manifest\n    create     .solomonar/manifest.json\n"+
		"Proceed? [Y/n] \nInitialized solomonar "+shipped+" in "+dir+"\n")
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
		{
			name: "copilot model asked", input: "gpt-5\ny\n",
			args: []string{"init", "--name", "ledger", "--objective", "Track", "--runtime", "copilot", "--ext", "none"},
			prompts: "Model for the adversarial reviewer in GitHub Copilot CLI, different from your main session model\n" +
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
			assert.True(t, strings.HasPrefix(screen(t, got.stdout), repoHeader(dir)+prompts+"Plan\n"), got.stdout)
			assert.Contains(t, got.stdout, "Proceed? [Y/n] \nInitialized solomonar "+shipped+" in ")
		})
	}
}

func TestInteractiveInitCopilotSelectionAndPlan(t *testing.T) {
	gittest.PathWith(t, "copilot")
	dir := committedRepo(t)

	got := runInteractive(t, repoAssets(t), dir, "\rgpt-5\ny\n",
		"init", "--name", "ledger", "--objective", "Track", "--ext", "none")

	require.Equal(t, 0, got.exit, got.stderr)
	transcript := screen(t, got.stdout)
	assert.Contains(t, transcript, "[x] copilot  GitHub Copilot CLI: AGENTS.md (native), .github/skills, .github/agents")
	assert.Contains(t, transcript, "Model for the adversarial reviewer in GitHub Copilot CLI, different from your main session model")
	assert.Contains(t, transcript, "  runtime copilot\n"+
		"    create     .github/skills/ (4 skill stubs: 4 core)\n"+
		"    create     .github/agents/ (5 role stubs; adversarial-reviewer model gpt-5)\n")
	assert.Equal(t, map[string]project.Runtime{"copilot": {AdversarialModel: "gpt-5"}}, manifestOf(t, dir).Runtimes)
}

func TestInteractiveInitRerunAsksNoChoices(t *testing.T) {
	t.Parallel()

	dir := initializedRepo(t)

	got := runInteractive(t, repoAssets(t), dir, "", "init")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, repoHeader(dir)+"Already initialized with solomonar "+shipped+"; nothing to do.\n"),
		got.stdout)
}

func TestInteractiveInitNeverPromptsWithYesOrJSON(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		exit   int
		stdout string
	}{
		{name: "yes", args: []string{"init", "--yes"}, stdout: "Initialized solomonar " + shipped + " in "},
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
	assert.True(t, strings.HasPrefix(got.stdout, repoHeader(dir)+"Plan\n"), got.stdout)
	assert.True(t, strings.HasSuffix(got.stdout,
		"  manifest\n    create     .solomonar/manifest.json\n"+
			"Proceed? [Y/n] \nCancelled; nothing was written.\n"), got.stdout)
	assertNothingWritten(t, dir, before, tip)
}

func TestInteractiveInitDryRunAsksChoicesWithoutConfirmation(t *testing.T) {
	dir := committedRepo(t)
	gittest.NoGH(t)
	before, tip := tree(t, dir), refTip(t, dir)

	got := runInteractive(t, repoAssets(t), dir, "\n\n\r\r\r", "init", "--dry-run")

	require.Equal(t, 0, got.exit, got.stderr)
	transcript := screen(t, got.stdout)
	assert.True(t, strings.HasPrefix(transcript, repoHeader(dir)+
		"Project name ["+filepath.Base(dir)+"]: Current objective, one line (optional): \n"+
		"Runtimes that will run this project's sessions:\n"+
		"    [ ] claude   Claude Code: CLAUDE.md import, .claude/skills, .claude/agents, state guard hook\n"+
		"    [ ] codex    Codex: AGENTS.md (native), .agents/skills, .codex/agents, state guard hook\n"+
		"    [ ] copilot  GitHub Copilot CLI: AGENTS.md (native), .github/skills, .github/agents\n"+
		extensionsIntro+
		"\nLanguage:\n    [ ] go  Go language rules\n"+
		"\nWorkflow:\n    [ ] extension-authoring  Evidence-based extension authoring\n\nPlan\n"), transcript)
	assert.Contains(t, transcript, "\nWould initialize solomonar "+shipped+" in "+dir+"\n")
	assert.NotContains(t, transcript, "Proceed?")
	assertNothingWritten(t, dir, before, tip)
}

func TestInteractiveInitAsksExtensionsAgainAfterRejection(t *testing.T) {
	dir := committedRepo(t)
	gittest.NoGH(t)
	input := "\r" + "jj \r" + "\r" + "\r" + " \r" + "\r" + "y\n"

	got := runInteractive(t, catalogAssets(t), dir, input,
		"init", "--name", "ledger", "--objective", "Track", "--runtime", "none")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Contains(t, screen(t, got.stdout),
		"\nFrontend:\n    [ ] htmx   T\n    [ ] react  T\n    [x] templ  T (requires htmx)\n"+
			"\nWorkflow:\n    [ ] extension-authoring  Evidence-based extension authoring\n"+
			"  templ requires htmx; add --ext htmx\n"+
			"\nLanguage:\n    [ ] go  Go language rules\n"+
			"\nFrontend:\n    [x] htmx   T\n    [ ] react  T\n    [x] templ  T (requires htmx)\n"+
			"\nWorkflow:\n    [ ] extension-authoring  Evidence-based extension authoring\n\nPlan\n")
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

// givenChoices answers every init choice by flag, so only the 10.3 questions and Proceed? remain.
func givenChoices(extra ...string) []string {
	return slices.Concat([]string{"init", "--name", "ledger", "--objective", "Track", "--runtime", "none", "--ext", "none"},
		extra)
}

func TestInteractiveInitHeaderNamesTheRepositoryState(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) string
		args  []string
		where string
	}{
		{name: "branch", setup: committedRepo, where: "(git, branch main)"},
		{name: "unborn branch", setup: unbornFeatureRepo, where: "(git, branch feature)"},
		{name: "detached HEAD", setup: func(t *testing.T) string {
			t.Helper()
			dir := committedRepo(t)
			gittest.Git(t, dir, "switch", "--detach")
			return dir
		}, where: "(git, detached HEAD)"},
		{name: "not a repository", setup: plainDir, args: []string{"--git-init"}, where: "(not a Git repository)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			gittest.NoGH(t)

			got := runInteractive(t, repoAssets(t), dir, "", givenChoices(append(tt.args, "--dry-run")...)...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.True(t, strings.HasPrefix(got.stdout,
				"solomonar "+shipped+" · "+dir+" "+tt.where+"\n\nPlan\n"), got.stdout)
		})
	}
}

func TestInteractiveInitOutsideRepositoryAsksToInitializeGit(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		input  string
		prompt string
		branch string
	}{
		{
			name: "default main branch", input: "y\ny\n", branch: "main",
			prompt: "Initialize a Git repository here with main branch \"main\"? [y/N] \n",
		},
		{
			name: "named main branch", args: []string{"--main-branch", "trunk"}, input: "y\ny\n", branch: "trunk",
			prompt: "Initialize a Git repository here with main branch \"trunk\"? [y/N] \n",
		},
		{name: "--git-init given", args: []string{"--git-init"}, input: "y\n", branch: "main"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := plainDir(t)

			got := runInteractive(t, repoAssets(t), dir, tt.input, givenChoices(tt.args...)...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.True(t, strings.HasPrefix(got.stdout,
				"solomonar "+shipped+" · "+dir+" (not a Git repository)\n\n"+tt.prompt+
					"Plan\n  git\n    init       -b "+tt.branch+"\n"), got.stdout)
			assert.Contains(t, got.stdout, "Proceed? [Y/n] \nInitialized solomonar "+shipped+" in "+dir+"\n")
			assert.Contains(t, got.stdout, "  1. First commit required before task worktrees:\n")
			assert.Equal(t, "refs/heads/"+tt.branch, gittest.Git(t, dir, "symbolic-ref", "HEAD"))
			assert.Equal(t, tt.branch, manifestOf(t, dir).Project.MainBranch)
		})
	}
}

func TestInteractiveInitDeclinedGitInitWritesNothing(t *testing.T) {
	dir := plainDir(t)

	got := runInteractive(t, repoAssets(t), dir, "n\n", givenChoices()...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "solomonar "+shipped+" · "+dir+" (not a Git repository)\n\n"+
		"Initialize a Git repository here with main branch \"main\"? [y/N] \n"+
		"Cancelled; nothing was written.\n", got.stdout)
	assert.NoDirExists(t, filepath.Join(dir, ".git"))
	assert.Empty(t, tree(t, dir))
}

const (
	adoptQuestion = "AGENTS.md exists and is not managed by solomonar.\n" +
		"  Adopt it: write the framework part above \"## Project notes\" and move the current content into " +
		"that section? [Y/n] "
	replaceQuestion = "  Replace it with the framework template, discarding its content? [y/N] "
)

func TestInteractiveInitAsksToAdoptAgentsMD(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		input   string
		prompts string
		plan    string
		adopted bool
	}{
		{
			name: "adopt", input: "\ny\n", prompts: adoptQuestion + "\n", adopted: true,
			plan: "  update     AGENTS.md  adopted; existing content moved under \"## Project notes\"\n",
		},
		{
			name: "replace", input: "n\ny\ny\n", prompts: adoptQuestion + replaceQuestion + "\n",
			plan: "  update     AGENTS.md\n",
		},
		{
			name: "--overwrite AGENTS.md", args: []string{"--overwrite", "AGENTS.md"}, input: "y\n",
			plan: "  update     AGENTS.md\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := committedRepo(t)
			gittest.NoGH(t)
			writeFile(t, dir, "AGENTS.md", "# Our agents\n")

			got := runInteractive(t, repoAssets(t), dir, tt.input, givenChoices(tt.args...)...)

			require.Equal(t, 0, got.exit, got.stderr)
			assert.True(t, strings.HasPrefix(got.stdout, repoHeader(dir)+tt.prompts+"Plan\n"), got.stdout)
			assert.Contains(t, got.stdout, tt.plan)
			agents := tree(t, dir)["AGENTS.md"]
			assert.Equal(t, tt.adopted, strings.Contains(agents, "## Project notes\n\n# Our agents\n"), agents)
			assert.Equal(t, tt.adopted, strings.Contains(agents, "# Our agents"), agents)
		})
	}
}

func TestInteractiveInitDryRunShowsAdoptionWithoutAsking(t *testing.T) {
	dir := committedRepo(t)
	gittest.NoGH(t)
	writeFile(t, dir, "AGENTS.md", "# Our agents\n")
	before, tip := tree(t, dir), refTip(t, dir)

	got := runInteractive(t, repoAssets(t), dir, "", givenChoices("--dry-run")...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.True(t, strings.HasPrefix(got.stdout, repoHeader(dir)+"Plan\n"), got.stdout)
	assert.Contains(t, got.stdout,
		"  update     AGENTS.md  adopted; existing content moved under \"## Project notes\"\n")
	assertNothingWritten(t, dir, before, tip)
}

func TestInteractiveInitDeclinedAdoptionAndReplaceWritesNothing(t *testing.T) {
	dir := committedRepo(t)
	gittest.NoGH(t)
	writeFile(t, dir, "AGENTS.md", "# Our agents\n")
	before, tip := tree(t, dir), refTip(t, dir)

	got := runInteractive(t, repoAssets(t), dir, "n\nn\n", givenChoices()...)

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, repoHeader(dir)+adoptQuestion+replaceQuestion+"\nCancelled; nothing was written.\n", got.stdout)
	assertNothingWritten(t, dir, before, tip)
}
