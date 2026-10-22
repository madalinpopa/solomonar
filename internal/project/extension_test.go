package project

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func extensionAssets() fstest.MapFS {
	return fstest.MapFS{
		"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", ""),
		"extensions/language/go/SKILL.md": {Data: []byte("---\nname: go\ndescription: Go rules.\n---\n\n# Go\n\n" +
			"## With templ\n<!-- solomonar:with templ -->\nT\n")},
		"extensions/language/go/references/style.md": {Data: []byte("## With x\n<!-- solomonar:with x -->\nas is\n")},
		"extensions/language/go/references.md":       {Data: []byte("r\n")},
		"extensions/frontend/templ/extension.json":   extensionJSON("templ", "skill", "frontend", ""),
		"extensions/frontend/templ/SKILL.md":         {Data: []byte("# templ\n")},
		"extensions/review/contrast/extension.json":  extensionJSON("contrast", "agent", "review", ""),
		"extensions/review/contrast/contrast.md": {Data: []byte("---\nname: contrast\ndescription: Checks.\n---\n\n" +
			"# Contrast\n\n## With go\n<!-- solomonar:with go -->\nG\n")},
		"extensions/review/contrast/notes.md": {Data: []byte("not installed\n")},
	}
}

func catalogExtensions(t *testing.T, assets fstest.MapFS, names ...string) []CatalogExtension {
	t.Helper()
	catalog, err := LoadCatalog(assets)
	require.NoError(t, err)
	var out []CatalogExtension
	for _, name := range names {
		require.Contains(t, catalog, name)
		out = append(out, catalog[name])
	}
	return out
}

func TestExtensionWantsInstallsRenderedFilesPerExtension(t *testing.T) {
	t.Parallel()

	assets := extensionAssets()
	installed := catalogExtensions(t, assets, "templ", "contrast", "go")
	decisions := map[string]string{"language": "P/D-1", "frontend": "P/D-2"}

	wants, extensions, err := ExtensionWants(assets, installed, decisions)

	require.NoError(t, err)
	assert.Equal(t, []Want{
		{Path: "docs/solomonar/agents/contrast.md", Owner: "ext:contrast", Content: []byte(
			"---\nname: contrast\ndescription: Checks.\n---\n\n# Contrast\n\n" +
				"Installed by solomonar; rendered with: go.\n\n## With go\nG\n")},
		{Path: "docs/solomonar/skills/go/SKILL.md", Owner: "ext:go", Content: []byte(
			"---\nname: go\ndescription: Go rules.\n---\n\n# Go\n\n" +
				"Installed by P/D-1; rendered with: templ.\n\n## With templ\nT\n")},
		{Path: "docs/solomonar/skills/go/references.md", Owner: "ext:go", Content: []byte("r\n")},
		{Path: "docs/solomonar/skills/go/references/style.md", Owner: "ext:go", Content: []byte(
			"## With x\n<!-- solomonar:with x -->\nas is\n")},
		{Path: "docs/solomonar/skills/templ/SKILL.md", Owner: "ext:templ", Content: []byte(
			"# templ\n\nInstalled by P/D-2.\n")},
	}, wants)
	assert.Equal(t, map[string]Extension{
		"contrast": {State: "installed", Kind: "agent", Group: "review", Source: "catalog", RenderedWith: []string{"go"}},
		"go": {
			State: "installed", Kind: "skill", Group: "language", Source: "catalog", Decision: "P/D-1",
			RenderedWith: []string{"templ"},
		},
		"templ": {State: "installed", Kind: "skill", Group: "frontend", Source: "catalog", Decision: "P/D-2"},
	}, extensions)
}

func TestExtensionWantsDropsSectionsOfMissingExtensions(t *testing.T) {
	t.Parallel()

	assets := extensionAssets()

	wants, extensions, err := ExtensionWants(assets, catalogExtensions(t, assets, "go"),
		map[string]string{"language": "P/D-1"})

	require.NoError(t, err)
	assert.Equal(t, "---\nname: go\ndescription: Go rules.\n---\n\n# Go\n\nInstalled by P/D-1.\n",
		string(wants[0].Content))
	assert.Nil(t, extensions["go"].RenderedWith)
}

func TestExtensionWantsRejectsAStackGroupWithoutDecision(t *testing.T) {
	t.Parallel()

	assets := extensionAssets()

	wants, extensions, err := ExtensionWants(assets, catalogExtensions(t, assets, "go", "templ"),
		map[string]string{"language": "P/D-1"})

	require.EqualError(t, err, "render templ: no decision for group frontend")
	assert.Nil(t, wants)
	assert.Nil(t, extensions)
}

func TestExtensionWantsInstallsTheEmbeddedGoExtension(t *testing.T) {
	t.Parallel()

	assets := os.DirFS("../../assets")
	catalog, err := LoadCatalog(assets)
	require.NoError(t, err)

	wants, extensions, err := ExtensionWants(assets, []CatalogExtension{catalog["go"]},
		map[string]string{"language": "P/D-4"})

	require.NoError(t, err)
	require.Len(t, wants, 1)
	assert.Equal(t, "docs/solomonar/skills/go/SKILL.md", wants[0].Path)
	assert.Equal(t, "ext:go", wants[0].Owner)
	assert.Contains(t, string(wants[0].Content), "\n# Go language rules\n\nInstalled by P/D-4.\n\n")
	assert.NotContains(t, string(wants[0].Content), "## With tdd")
	assert.Equal(t, Extension{
		State: "installed", Kind: "skill", Group: "language", Source: "catalog",
		Decision: "P/D-4",
	}, extensions["go"])
}
