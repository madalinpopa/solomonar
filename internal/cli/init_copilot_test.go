package cli

import (
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

const copilotCoreFiles = ".github/skills/prototype/SKILL.md .github/skills/solomonar/SKILL.md " +
	".github/skills/spec/SKILL.md .github/skills/work-spec/SKILL.md " +
	".github/agents/adversarial-reviewer.agent.md .github/agents/correctness-reviewer.agent.md " +
	".github/agents/implementer.agent.md .github/agents/security-reviewer.agent.md .github/agents/spec-reviewer.agent.md"

func TestInitCopilotAndMixedRuntimes(t *testing.T) {
	// Explicit selection must work without any runtime executable.
	gittest.PathWith(t)
	tests := []struct {
		selection, precedingPaths string
		runtimes                  []string
		created                   int
	}{
		{selection: "copilot", runtimes: []string{"copilot"}, created: 49},
		{
			selection: "copilot,claude", precedingPaths: "CLAUDE.md .claude ",
			runtimes: []string{"claude", "copilot"}, created: 59,
		},
		{
			selection: "copilot,codex", precedingPaths: ".agents .codex ",
			runtimes: []string{"codex", "copilot"}, created: 59,
		},
		{
			selection: "copilot,codex,claude", precedingPaths: "CLAUDE.md .claude .agents .codex ",
			runtimes: []string{"claude", "codex", "copilot"}, created: 69,
		},
	}
	for _, tt := range tests {
		t.Run(tt.selection, func(t *testing.T) {
			human, asJSON := committedRepo(t), committedRepo(t)
			writeFile(t, human, ".github/skills/personal/SKILL.md", "personal skill\n")
			writeFile(t, human, ".github/workflows/ci.yml", "project CI\n")
			before, tip := tree(t, human), refTip(t, human)
			args := runtimeArgs(tt.selection, "--runtime", "copilot",
				"--adversarial-model", "copilot=old", "--adversarial-model", "copilot=gpt-5")

			preview := runInRepo(t, human, append(args, "--dry-run", "--json")...)
			require.Equal(t, 0, preview.exit, preview.stderr)
			var plan struct {
				Data initData `json:"data"`
			}
			require.NoError(t, json.Unmarshal([]byte(preview.stdout), &plan))
			assert.Equal(t, tt.runtimes, plan.Data.Choices.Runtimes)
			assert.True(t, plan.Data.DryRun)
			assert.Empty(t, plan.Data.Next)
			var adapters []string
			for _, file := range plan.Data.Files {
				if file.Owner == "runtime:copilot" {
					adapters = append(adapters, file.Path)
					assert.Equal(t, "create", file.Action)
				}
			}
			assert.Equal(t, strings.Fields(copilotCoreFiles), adapters)
			assertNothingWritten(t, human, before, tip)

			text := runInRepo(t, human, args...)
			require.Equal(t, 0, text.exit, text.stderr)
			assert.Contains(t, text.stdout, fmt.Sprintf("  %d files created\n", tt.created))
			paths := "AGENTS.md " + tt.precedingPaths + copilotCoreFiles + " .solomonar docs"
			assert.Contains(t, text.stdout, commitStep(paths))
			if len(tt.runtimes) == 1 {
				assert.Contains(t, text.stdout, "Start a top-level GitHub Copilot CLI session in "+human+":\n"+
					"       /solomonar Explore the next useful flow.\n")
			} else {
				assert.Contains(t, text.stdout, "       GitHub Copilot CLI: /solomonar Explore the next useful flow.\n")
				assert.Equal(t, strings.Contains(tt.selection, "claude"), strings.Contains(text.stdout, "       Claude Code:"))
				assert.Equal(t, strings.Contains(tt.selection, "codex"), strings.Contains(text.stdout, "       Codex:"))
			}
			files := tree(t, human)
			assert.Equal(t, "personal skill\n", files[".github/skills/personal/SKILL.md"])
			assert.Equal(t, "project CI\n", files[".github/workflows/ci.yml"])
			assert.Contains(t, files[".github/agents/adversarial-reviewer.agent.md"], "\nmodel: \"gpt-5\"\n")
			assert.Equal(t, "gpt-5", manifestOf(t, human).Runtimes["copilot"].AdversarialModel)

			data := runInRepo(t, asJSON, append(args, "--json")...)
			require.Equal(t, 0, data.exit, data.stderr)
			var result struct {
				Data initData `json:"data"`
			}
			require.NoError(t, json.Unmarshal([]byte(data.stdout), &result))
			assert.Equal(t, tt.runtimes, result.Data.Choices.Runtimes)
			assert.Equal(t, map[string]string{"copilot": "gpt-5"}, result.Data.Choices.AdversarialModel)
			assert.Equal(t, []string{"git add " + paths, "git commit -m \"chore: initialize solomonar " + shipped + "\""},
				result.Data.Next)

			before, tip = tree(t, human), refTip(t, human)
			rerun := runInRepo(t, human, "init")
			require.Equal(t, 0, rerun.exit, rerun.stderr)
			assert.Contains(t, rerun.stdout, "Already initialized with solomonar "+shipped+"; nothing to do.")
			assertNothingWritten(t, human, before, tip)
		})
	}
}

func TestInitCopilotRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		code string
	}{
		{
			name: "confirmation required", args: []string{"init", "--runtime", "copilot", "--ext", "none"},
			code: "interactive_required",
		},
		{
			name: "json confirmation required", args: []string{"init", "--runtime", "copilot", "--ext", "none", "--json"},
			code: "interactive_required",
		},
		{name: "none is exclusive", args: runtimeArgs("none,copilot"), code: "invalid_value"},
		{
			name: "model names selected runtime", args: runtimeArgs("claude", "--adversarial-model", "copilot=gpt-5"),
			code: "invalid_value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := committedRepo(t)
			before, tip := tree(t, dir), refTip(t, dir)
			got := runInRepo(t, dir, tt.args...)
			assert.Equal(t, 2, got.exit)
			assert.Contains(t, got.stdout+got.stderr, tt.code)
			assertNothingWritten(t, dir, before, tip)
		})
	}
}

func TestInitCopilotConflictsUseExistingResolutions(t *testing.T) {
	t.Parallel()

	const file = ".github/skills/spec/SKILL.md"
	for _, resolution := range []string{"--keep", "--overwrite"} {
		t.Run(resolution, func(t *testing.T) {
			t.Parallel()
			dir := committedRepo(t)
			writeFile(t, dir, file, "project instructions\n")
			before, tip := tree(t, dir), refTip(t, dir)

			conflict := runInRepo(t, dir, runtimeArgs("copilot")...)
			assert.Equal(t, 4, conflict.exit)
			assert.Contains(t, conflict.stdout, file)
			assert.Contains(t, conflict.stderr, "file_conflict")
			assertNothingWritten(t, dir, before, tip)

			got := runInRepo(t, dir, runtimeArgs("copilot", resolution, file)...)
			require.Equal(t, 0, got.exit, got.stderr)
			if resolution == "--keep" {
				assert.Equal(t, "project instructions\n", tree(t, dir)[file])
			} else {
				assert.Contains(t, tree(t, dir)[file], "Load `docs/solomonar/skills/spec/SKILL.md` and follow it.")
			}
			assert.Equal(t, "runtime:copilot", manifestOf(t, dir).Files[file].Owner)
		})
	}
}
