package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type promptRig struct {
	prompter *prompter
	out      *bytes.Buffer
	raw      int
	restored int
}

func newPromptRig(t *testing.T, input string) *promptRig {
	t.Helper()
	rig := &promptRig{out: &bytes.Buffer{}}
	rig.prompter = newPrompter(Env{
		Stdin:  strings.NewReader(input),
		Stdout: rig.out,
		RawMode: func() (func(), error) {
			rig.raw++
			return func() { rig.restored++ }, nil
		},
	})
	return rig
}

func TestTextPromptShowsDefaultAndTrimsAnswer(t *testing.T) {
	tests := []struct {
		name   string
		label  string
		def    string
		input  string
		want   string
		prompt string
	}{
		{
			name: "empty answer gives default", label: "Project name", def: "ledger", input: "\n",
			want: "ledger", prompt: "Project name [ledger]: ",
		},
		{
			name: "answer replaces default", label: "Project name", def: "ledger", input: "  books \n",
			want: "books", prompt: "Project name [ledger]: ",
		},
		{
			name: "label without default", label: "Objective", input: "Track expenses\r\n",
			want: "Track expenses", prompt: "Objective: ",
		},
		{name: "empty answer without default", label: "Objective", input: "\n", want: "", prompt: "Objective: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rig := newPromptRig(t, tt.input)

			got, err := rig.prompter.text(tt.label, tt.def)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.prompt, rig.out.String())
		})
	}
}

func TestConfirmPromptAcceptsYesNoAndDefault(t *testing.T) {
	tests := []struct {
		name   string
		def    bool
		input  string
		want   bool
		prompt string
	}{
		{name: "empty takes yes default", def: true, input: "\n", want: true, prompt: "Commit? [Y/n] "},
		{name: "empty takes no default", def: false, input: "\n", want: false, prompt: "Commit? [y/N] "},
		{name: "y", def: false, input: "y\n", want: true, prompt: "Commit? [y/N] "},
		{name: "YES in capitals", def: false, input: "YES\n", want: true, prompt: "Commit? [y/N] "},
		{name: "N in capitals", def: true, input: "N\n", want: false, prompt: "Commit? [Y/n] "},
		{name: "No mixed case", def: true, input: " No \n", want: false, prompt: "Commit? [Y/n] "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rig := newPromptRig(t, tt.input)

			got, err := rig.prompter.confirm("Commit?", tt.def)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.prompt, rig.out.String())
		})
	}
}

func TestConfirmPromptAsksAgainOnOtherAnswers(t *testing.T) {
	rig := newPromptRig(t, "maybe\nyep\nn\n")

	got, err := rig.prompter.confirm("Adopt AGENTS.md?", true)

	require.NoError(t, err)
	assert.False(t, got)
	ask := "Adopt AGENTS.md? [Y/n] "
	assert.Equal(t, ask+"  answer y or n\n"+ask+"  answer y or n\n"+ask, rig.out.String())
}

func TestLinePromptsFailWhenInputCloses(t *testing.T) {
	tests := []struct {
		name string
		ask  func(p *prompter) error
	}{
		{name: "text", ask: func(p *prompter) error { _, err := p.text("Name", "x"); return err }},
		{name: "confirm", ask: func(p *prompter) error { _, err := p.confirm("Commit?", true); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rig := newPromptRig(t, "")

			err := tt.ask(rig.prompter)

			require.EqualError(t, err, "read answer: input closed")
		})
	}
}

func runtimeChoices() []choice {
	return []choice{
		{name: "claude", description: "Claude Code"},
		{name: "codex", description: "Codex CLI"},
		{name: "go", description: "Go code"},
	}
}

const (
	chooseHelp = "  ↑/↓ move · Space toggle · Enter accept\r\n"
	chooseUp   = "\x1b[4A\r\x1b[J"
)

func TestChooseRendersListWithCursorAndHelp(t *testing.T) {
	rig := newPromptRig(t, "\r")

	got, err := rig.prompter.choose(runtimeChoices(), []string{"claude"})

	require.NoError(t, err)
	assert.Equal(t, []string{"claude"}, got)
	assert.Equal(t,
		"  > [x] claude  Claude Code\r\n"+
			"    [ ] codex   Codex CLI\r\n"+
			"    [ ] go      Go code\r\n"+
			chooseHelp+
			chooseUp+
			"    [x] claude  Claude Code\r\n"+
			"    [ ] codex   Codex CLI\r\n"+
			"    [ ] go      Go code\r\n",
		rig.out.String())
	assert.Equal(t, 1, rig.raw)
	assert.Equal(t, 1, rig.restored)
}

func TestChooseMovesAndToggles(t *testing.T) {
	tests := []struct {
		name     string
		keys     string
		selected []string
		want     []string
	}{
		{name: "enter accepts defaults", keys: "\r", selected: []string{"codex", "go"}, want: []string{"codex", "go"}},
		{name: "newline accepts", keys: "\n", selected: []string{"go"}, want: []string{"go"}},
		{name: "space toggles cursor row", keys: " \r", selected: []string{"claude"}, want: []string{}},
		{name: "arrow down then toggle", keys: "\x1b[B \r", want: []string{"codex"}},
		{name: "j down then toggle", keys: "jj \r", want: []string{"go"}},
		{name: "arrow up after down", keys: "jj\x1b[A \r", want: []string{"codex"}},
		{name: "k up after down", keys: "jk \r", want: []string{"claude"}},
		{name: "down clamps at last", keys: "jjjjj \r", want: []string{"go"}},
		{name: "up clamps at first", keys: "kk\x1b[A \r", want: []string{"claude"}},
		{name: "a selects all", keys: "a\r", want: []string{"claude", "codex", "go"}},
		{name: "n selects none", keys: "n\r", selected: []string{"claude", "go"}, want: []string{}},
		{name: "other keys ignored", keys: "xq\x1b[C\x1bO\t \r", want: []string{"claude"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rig := newPromptRig(t, tt.keys)

			got, err := rig.prompter.choose(runtimeChoices(), tt.selected)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, 1, rig.restored)
		})
	}
}

func TestChooseRedrawsAfterEachKey(t *testing.T) {
	rig := newPromptRig(t, "j \r")

	_, err := rig.prompter.choose(runtimeChoices()[:2], nil)

	require.NoError(t, err)
	up := "\x1b[3A\r\x1b[J"
	assert.Equal(t,
		"  > [ ] claude  Claude Code\r\n    [ ] codex   Codex CLI\r\n"+chooseHelp+
			up+"    [ ] claude  Claude Code\r\n  > [ ] codex   Codex CLI\r\n"+chooseHelp+
			up+"    [ ] claude  Claude Code\r\n  > [x] codex   Codex CLI\r\n"+chooseHelp+
			up+"    [ ] claude  Claude Code\r\n    [x] codex   Codex CLI\r\n",
		rig.out.String())
}

func TestChooseIgnoredKeyRedrawsNothing(t *testing.T) {
	rig := newPromptRig(t, "x\r")

	_, err := rig.prompter.choose(runtimeChoices(), nil)

	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(rig.out.String(), chooseUp))
}

func TestChooseFailsAndRestoresTerminal(t *testing.T) {
	tests := []struct {
		name    string
		keys    string
		wantErr string
		isErr   error
	}{
		{name: "input closes", keys: "j", wantErr: "read answer: input closed"},
		{name: "input closes inside escape", keys: "\x1b[", wantErr: "read answer: input closed"},
		{name: "ctrl-c interrupts", keys: "j\x03", isErr: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rig := newPromptRig(t, tt.keys)

			got, err := rig.prompter.choose(runtimeChoices(), nil)

			assert.Nil(t, got)
			if tt.isErr != nil {
				require.ErrorIs(t, err, tt.isErr)
			} else {
				require.EqualError(t, err, tt.wantErr)
			}
			assert.Equal(t, 1, rig.raw)
			assert.Equal(t, 1, rig.restored)
		})
	}
}

func TestChooseReturnsRawModeFailure(t *testing.T) {
	rawErr := errors.New("raw terminal: not a terminal")
	p := newPrompter(Env{
		Stdin:   strings.NewReader("\r"),
		Stdout:  &bytes.Buffer{},
		RawMode: func() (func(), error) { return nil, rawErr },
	})

	got, err := p.choose(runtimeChoices(), nil)

	assert.Nil(t, got)
	require.ErrorIs(t, err, rawErr)
}

func TestPromptsShareOneInputStream(t *testing.T) {
	rig := newPromptRig(t, "ledger\nj \ry\n")

	name, err := rig.prompter.text("Project name", "x")
	require.NoError(t, err)
	chosen, err := rig.prompter.choose(runtimeChoices(), nil)
	require.NoError(t, err)
	ok, err := rig.prompter.confirm("Commit?", false)
	require.NoError(t, err)

	assert.Equal(t, "ledger", name)
	assert.Equal(t, []string{"codex"}, chosen)
	assert.True(t, ok)
	assert.True(t, strings.HasSuffix(rig.out.String(), "    [ ] go      Go code\r\nCommit? [y/N] "))
}
