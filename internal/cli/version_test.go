package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

// putRegistry points the registry ref at a commit whose state.json is content.
func putRegistry(t *testing.T, dir, content string) {
	t.Helper()
	repo := &git.Repo{Dir: dir}
	blob, err := repo.RunInput(t.Context(), strings.NewReader(content), "hash-object", "-w", "--stdin")
	require.NoError(t, err)
	tree, err := repo.RunInput(t.Context(), strings.NewReader("100644 blob "+blob+"\tstate.json\n"), "mktree")
	require.NoError(t, err)
	commit, err := repo.Run(t.Context(), "commit-tree", tree, "-m", "state: test")
	require.NoError(t, err)
	gittest.Git(t, dir, "update-ref", task.Ref, commit)
}

func putManifestVersions(t *testing.T, dir, content string) {
	t.Helper()
	writeFile(t, dir, project.ManifestPath, content)
}

func TestVersionReportsTheProject(t *testing.T) {
	t.Parallel()

	current := `{"schema_version": 1, "framework_version": "` + shipped + `"}`
	tests := []struct {
		name     string
		manifest string
		registry string
		want     string
	}{
		{
			name: "manifest and registry", manifest: current, registry: `{"schema_version": 1, "tasks": {}}`,
			want: "framework " + shipped + " · registry schema 1 (older than this CLI; run solomonar task migrate)",
		},
		{
			name: "current registry", manifest: current, registry: `{"schema_version": 2}`,
			want: "framework " + shipped + " · registry schema 2",
		},
		{
			name: "registry missing", manifest: current,
			want: "framework " + shipped + " · registry missing",
		},
		{
			name: "newer framework", manifest: `{"schema_version": 1, "framework_version": "0.10.0"}`, registry: `{"schema_version": 1}`,
			want: "framework 0.10.0 · registry schema 1 (newer than this CLI; install solomonar 0.10.0 or later)",
		},
		{
			name: "newer manifest schema", manifest: `{"schema_version": 2, "framework_version": "` + shipped + `", "added": true}`, registry: `{"schema_version": 1}`,
			want: "framework " + shipped + " · registry schema 1 (newer than this CLI; install a newer solomonar release)",
		},
		{
			name: "newer registry schema", manifest: current, registry: `{"schema_version": 3, "added": true}`,
			want: "framework " + shipped + " · registry schema 3 (newer than this CLI; install a newer solomonar release)",
		},
		{
			name: "unreadable manifest versions", manifest: `{"schema_version": "one"}`, registry: `{"schema_version": 1}`,
			want: "manifest invalid (inspect .solomonar/manifest.json)",
		},
		{
			name: "unreadable registry", manifest: current, registry: `{"tasks": {}}`,
			want: "framework " + shipped + " · registry invalid",
		},
		{
			name: "unreadable registry with newer framework", manifest: `{"schema_version": 1, "framework_version": "0.2.0"}`, registry: `[`,
			want: "framework 0.2.0 · registry invalid (newer than this CLI; install solomonar 0.2.0 or later)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := gittest.NewRepo(t)
			putManifestVersions(t, dir, tt.manifest)
			if tt.registry != "" {
				putRegistry(t, dir, tt.registry)
			}

			got := runInRepo(t, dir, "version")

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, "solomonar "+shipped+"\nproject   "+tt.want+"\n", got.stdout)
			assert.Empty(t, got.stderr)
		})
	}
}

func TestVersionWithoutProjectIsNotInitialized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		repository bool
	}{
		{name: "outside a repository"},
		{name: "repository without manifest", repository: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if tt.repository {
				dir, _ = registryRepo(t)
			}

			got := runInRepo(t, dir, "version")

			require.Equal(t, 0, got.exit, got.stderr)
			assert.Equal(t, "solomonar "+shipped+"\nproject   not initialized\n", got.stdout)
			assert.Empty(t, got.stderr)
		})
	}
}

func TestVersionJSONReportsTheProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest string
		registry string
		want     string
	}{
		{
			name: "manifest and registry", manifest: `{"schema_version": 1, "framework_version": "` + shipped + `"}`,
			registry: `{"schema_version": 1, "tasks": {}}`, want: `"project_framework": "` + shipped + `", "registry_schema": 1`,
		},
		{
			name: "registry missing", manifest: `{"schema_version": 1, "framework_version": "` + shipped + `"}`,
			want: `"project_framework": "` + shipped + `", "registry_schema": null`,
		},
		{
			name: "unreadable manifest versions", manifest: `{}`, registry: `{"schema_version": 1}`,
			want: `"project_framework": null, "registry_schema": null`,
		},
		{
			name: "unreadable registry", manifest: `{"schema_version": 1, "framework_version": "` + shipped + `"}`, registry: `{}`,
			want: `"project_framework": "` + shipped + `", "registry_schema": null`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := gittest.NewRepo(t)
			putManifestVersions(t, dir, tt.manifest)
			if tt.registry != "" {
				putRegistry(t, dir, tt.registry)
			}

			got := runInRepo(t, dir, "version", "--json")

			require.Equal(t, 0, got.exit, got.stderr)
			assert.JSONEq(t, `{
				"ok": true,
				"command": "version",
				"data": {"cli": "`+shipped+`", "framework": "`+shipped+`", `+tt.want+`},
				"warnings": []
			}`, got.stdout)
		})
	}
}
