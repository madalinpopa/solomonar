package release_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/release"
)

func TestVersionReportsTheRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		build string
		want  string
	}{
		{name: "release tag", build: "v0.3.8", want: "0.3.8"},
		{name: "dirty release tag", build: "v0.3.8+dirty", want: "0.3.8"},
		{name: "after a release", build: "v0.3.8-0.20261025132652-46231677d757", want: "0.3.7"},
		{name: "dirty after a release", build: "v0.3.8-0.20261025132652-46231677d757+dirty", want: "0.3.7"},
		{name: "after a release at patch zero", build: "v0.4.0-0.20261025132652-46231677d757", want: release.Unknown},
		{name: "prerelease tag", build: "v0.4.0-rc.1", want: "0.4.0"},
		{name: "after a prerelease", build: "v0.4.0-rc.1.0.20261025132652-46231677d757", want: "0.4.0"},
		{name: "no tag", build: "v0.0.0-20261025132652-46231677d757", want: release.Unknown},
		{name: "no build information", build: "(devel)", want: release.Unknown},
		{name: "empty", build: "", want: release.Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, release.Version(tt.build))
		})
	}
}

func TestAssetsRendersTheVersion(t *testing.T) {
	t.Parallel()

	assets := release.Assets(fstest.MapFS{
		"framework/rules.md":              {Data: []byte("Framework version: {{ .Version }}\n")},
		"extensions/language/go/SKILL.md": {Data: []byte("Framework version: {{ .Version }}\n")},
		"extensions/language/go/ext.json": {Data: []byte(`{"v": "{{ .Version }}"}`)},
		"templates/AGENTS.md":             {Data: []byte("Framework version: {{ .FrameworkVersion }}.\n")},
	}, "1.2.3")

	for name, want := range map[string]string{
		"framework/rules.md":              "Framework version: 1.2.3\n",
		"extensions/language/go/SKILL.md": "Framework version: 1.2.3\n",
		"extensions/language/go/ext.json": `{"v": "{{ .Version }}"}`,
		"templates/AGENTS.md":             "Framework version: {{ .FrameworkVersion }}.\n",
	} {
		got, err := fs.ReadFile(assets, name)
		require.NoError(t, err, name)
		assert.Equal(t, want, string(got), name)

		info, err := fs.Stat(assets, name)
		require.NoError(t, err, name)
		assert.Equal(t, int64(len(want)), info.Size(), name)
	}
	var walked []string
	require.NoError(t, fs.WalkDir(assets, "framework", func(name string, _ fs.DirEntry, err error) error {
		walked = append(walked, name)
		return err
	}))
	assert.Equal(t, []string{"framework", "framework/rules.md"}, walked)
}

func TestAssetsRejectsAnInvalidTemplate(t *testing.T) {
	t.Parallel()

	for name, content := range map[string]string{
		"syntax":  "Framework version: {{ .Version\n",
		"unknown": "Framework version: {{ .Release }}\n",
	} {
		assets := release.Assets(fstest.MapFS{"framework/rules.md": {Data: []byte(content)}}, "1.2.3")

		_, err := fs.ReadFile(assets, "framework/rules.md")
		require.ErrorContains(t, err, "open framework/rules.md: render version:", name)
	}
}
