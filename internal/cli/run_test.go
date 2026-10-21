package cli

import (
	"bytes"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type result struct {
	exit   int
	stdout string
	stderr string
}

func runCLI(t *testing.T, assets fs.FS, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	exit := Run(t.Context(), Env{
		Args:   args,
		Stdin:  &bytes.Buffer{},
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: func(string) string { return "" },
		Now:    func() time.Time { return time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC) },
		Assets: assets,
	})
	return result{exit: exit, stdout: stdout.String(), stderr: stderr.String()}
}

func repoAssets(t *testing.T) fs.FS {
	t.Helper()
	return os.DirFS("../../assets")
}

func TestVersionPrintsEmbeddedFramework(t *testing.T) {
	got := runCLI(t, repoAssets(t), "version")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.Equal(t, "solomonar devel (embedded framework 0.1.0)\nproject   not initialized\n", got.stdout)
	assert.Empty(t, got.stderr)
}

func TestVersionJSONPrintsOneEnvelope(t *testing.T) {
	got := runCLI(t, repoAssets(t), "version", "--json")

	require.Equal(t, 0, got.exit, got.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "version",
		"data": {"cli": "devel", "framework": "0.1.0", "project_framework": null, "registry_schema": null},
		"warnings": []
	}`, got.stdout)
	assert.Empty(t, got.stderr)
}

func TestSharedFlagsWorkBeforeAndAfterCommand(t *testing.T) {
	before := runCLI(t, repoAssets(t), "--json", "-v", "--no-color", "-C", "/tmp", "version")
	after := runCLI(t, repoAssets(t), "version", "--verbose", "-C", "/tmp", "--json", "--no-color")

	require.Equal(t, 0, before.exit, before.stderr)
	require.Equal(t, 0, after.exit, after.stderr)
	assert.Equal(t, before.stdout, after.stdout)
	assert.Contains(t, before.stdout, `"ok":`)
}

func TestUnknownCommandExitsTwo(t *testing.T) {
	got := runCLI(t, repoAssets(t), "nope")

	assert.Equal(t, 2, got.exit)
	assert.Empty(t, got.stdout)
	assert.Equal(t, "error: unknown command \"nope\" (unknown_command)\n"+
		"  fix:   run solomonar help\n"+
		"nothing was written\n", got.stderr)
}

func TestNoCommandExitsTwo(t *testing.T) {
	got := runCLI(t, repoAssets(t))

	assert.Equal(t, 2, got.exit)
	assert.Contains(t, got.stderr, "(unknown_command)")
}

func TestUnknownCommandJSONNamesTypedWord(t *testing.T) {
	got := runCLI(t, repoAssets(t), "--json", "nope")

	assert.Equal(t, 2, got.exit)
	assert.JSONEq(t, `{
		"ok": false,
		"command": "nope",
		"error": {"code": "unknown_command", "message": "unknown command \"nope\"", "fix": "run solomonar help"},
		"warnings": []
	}`, got.stdout)
	assert.Empty(t, got.stderr)
}

func TestInvalidUsageExitsTwo(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown flag", args: []string{"version", "--bogus"}},
		{name: "unknown global flag", args: []string{"--bogus", "version"}},
		{name: "missing flag value", args: []string{"version", "-C"}},
		{name: "extra argument", args: []string{"version", "extra"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runCLI(t, repoAssets(t), tt.args...)

			assert.Equal(t, 2, got.exit)
			assert.Contains(t, got.stderr, "(invalid_usage)")
		})
	}
}

func TestMissingFrameworkVersionExitsOne(t *testing.T) {
	assets := fstest.MapFS{"framework/rules.md": {Data: []byte("# Framework rules\n")}}

	got := runCLI(t, assets, "--json", "version")

	assert.Equal(t, 1, got.exit)
	assert.JSONEq(t, `{
		"ok": false,
		"command": "version",
		"error": {"code": "unexpected_error", "message": "read framework version: no version line in framework/rules.md"},
		"warnings": []
	}`, got.stdout)
}

func TestMissingRulesFileExitsOne(t *testing.T) {
	got := runCLI(t, fstest.MapFS{}, "version")

	assert.Equal(t, 1, got.exit)
	assert.Contains(t, got.stderr, "error: read framework version: ")
	assert.Contains(t, got.stderr, "(unexpected_error)\nnothing was written\n")
}
