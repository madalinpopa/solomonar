package project

import (
	"io/fs"
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

func TestExtensionWantsCopiesEveryFileVerbatimIntoTheStore(t *testing.T) {
	t.Parallel()

	assets := extensionAssets()
	installed := catalogExtensions(t, assets, "templ", "contrast", "go")
	decisions := map[string]string{"language": "P/D-1", "frontend": "P/D-2"}

	wants, extensions, err := ExtensionWants(installed, decisions)

	require.NoError(t, err)
	stored := func(owner, store, source string) Want {
		return Want{
			Path: "docs/solomonar/extensions/" + store, Owner: "ext:" + owner, Content: assets["extensions/"+source].Data,
		}
	}
	assert.Equal(t, []Want{
		stored("contrast", "contrast/contrast.md", "review/contrast/contrast.md"),
		stored("contrast", "contrast/extension.json", "review/contrast/extension.json"),
		stored("contrast", "contrast/notes.md", "review/contrast/notes.md"),
		stored("go", "go/SKILL.md", "language/go/SKILL.md"),
		stored("go", "go/extension.json", "language/go/extension.json"),
		stored("go", "go/references.md", "language/go/references.md"),
		stored("go", "go/references/style.md", "language/go/references/style.md"),
		stored("templ", "templ/SKILL.md", "frontend/templ/SKILL.md"),
		stored("templ", "templ/extension.json", "frontend/templ/extension.json"),
	}, wants)
	assert.Equal(t, map[string]Extension{
		"contrast": {State: "installed", Kind: "agent", Group: "review", Source: "catalog"},
		"go":       {State: "installed", Kind: "skill", Group: "language", Source: "catalog", Decision: "P/D-1"},
		"templ":    {State: "installed", Kind: "skill", Group: "frontend", Source: "catalog", Decision: "P/D-2"},
	}, extensions)
}

func TestExtensionWantsRejectsAStackGroupWithoutDecision(t *testing.T) {
	t.Parallel()

	assets := extensionAssets()

	wants, extensions, err := ExtensionWants(catalogExtensions(t, assets, "go", "templ"),
		map[string]string{"language": "P/D-1"})

	require.EqualError(t, err, "install templ: no decision for group frontend")
	assert.Nil(t, wants)
	assert.Nil(t, extensions)
}

func TestExtensionWantsInstallsTheEmbeddedGoExtension(t *testing.T) {
	t.Parallel()

	assets := os.DirFS("../../assets")
	catalog, err := LoadCatalog(assets)
	require.NoError(t, err)

	wants, extensions, err := ExtensionWants([]CatalogExtension{catalog["go"]},
		map[string]string{"language": "P/D-4"})

	require.NoError(t, err)
	require.Len(t, wants, 6)
	var paths []string
	for _, want := range wants {
		paths = append(paths, want.Path)
		assert.Equal(t, "ext:go", want.Owner)
	}
	assert.Equal(t, []string{
		"docs/solomonar/extensions/go/SKILL.md",
		"docs/solomonar/extensions/go/extension.json",
		"docs/solomonar/extensions/go/rules/go/concurrency.md",
		"docs/solomonar/extensions/go/rules/go/design.md",
		"docs/solomonar/extensions/go/rules/go/io.md",
		"docs/solomonar/extensions/go/rules/go/testing.md",
	}, paths)
	source, err := fs.ReadFile(assets, "extensions/language/go/SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, source, wants[0].Content)
	assert.Equal(t, Extension{
		State: "installed", Kind: "skill", Group: "language", Source: "catalog",
		Decision: "P/D-4",
	}, extensions["go"])
}

func TestExtensionWantsActivatesAProjectExtensionWithoutFiles(t *testing.T) {
	t.Parallel()

	root := storeRoot(fstest.MapFS{
		"money/extension.json": extensionJSON("money", "agent", "review", ""),
		"money/money.md":       skillFile("money"),
	})
	store, err := LoadStore(root, fstest.MapFS{}, testCatalog(t), nil)
	require.NoError(t, err)
	require.NoError(t, store["money"].Err)

	wants, extensions, err := ExtensionWants([]CatalogExtension{store["money"].CatalogExtension}, nil)

	require.NoError(t, err)
	assert.Empty(t, wants)
	assert.Equal(t, map[string]Extension{
		"money": {
			State: "installed", Kind: "agent", Group: "review", Source: "path", Path: "docs/solomonar/extensions/money",
		},
	}, extensions)
}
