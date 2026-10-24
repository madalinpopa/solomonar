package project

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skillFile(name string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("---\nname: " + name + "\ndescription: Use for " + name + ".\n---\n# " + name + "\n")}
}

func storeRoot(files fstest.MapFS) fstest.MapFS {
	root := fstest.MapFS{}
	for name, file := range files {
		root["docs/solomonar/extensions/"+name] = file
	}
	return root
}

func storeReasons(t *testing.T, root fstest.MapFS, manifest *Manifest) map[string]string {
	t.Helper()
	store, err := LoadStore(root, fstest.MapFS{}, testCatalog(t), manifest)
	require.NoError(t, err)
	reasons := map[string]string{}
	for name, e := range store {
		reasons[name] = ""
		if e.Err != nil {
			reasons[name] = e.Err.Error()
		}
	}
	return reasons
}

func TestLoadStoreReadsAValidProjectExtension(t *testing.T) {
	t.Parallel()

	root := storeRoot(fstest.MapFS{
		"money/extension.json": extensionJSON("money", "skill", "backend", `, "requires": ["go"], "detect": ["**/money.go"]`),
		"money/SKILL.md":       skillFile("money"),
	})

	store, err := LoadStore(root, fstest.MapFS{}, testCatalog(t), nil)

	require.NoError(t, err)
	require.NoError(t, store["money"].Err)
	assert.Equal(t, "backend", store["money"].Group)
	assert.Equal(t, "docs/solomonar/extensions/money", store["money"].Dir)
	assert.Equal(t, []string{"**/money.go"}, store["money"].Detect)
}

func TestLoadStoreWithoutStoreIsEmpty(t *testing.T) {
	t.Parallel()

	store, err := LoadStore(fstest.MapFS{"go.mod": {Data: []byte("module x\n")}}, fstest.MapFS{}, testCatalog(t), nil)

	require.NoError(t, err)
	assert.Empty(t, store)
}

func TestLoadStoreSkipsInstalledCatalogExtensions(t *testing.T) {
	t.Parallel()

	root := storeRoot(fstest.MapFS{
		"go/extension.json": extensionJSON("go", "skill", "language", ""),
		"go/SKILL.md":       {Data: []byte("# Go\n")},
	})
	manifest := &Manifest{Extensions: map[string]Extension{"go": {State: "installed", Source: "catalog"}}}

	assert.Empty(t, storeReasons(t, root, manifest))
}

func TestLoadStoreReportsTheFirstFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files fstest.MapFS
		want  string
	}{
		{
			name: "missing applies_when",
			files: fstest.MapFS{
				"x/extension.json": {Data: []byte(`{"name": "x", "kind": "skill", "group": "review", "title": "T"}`)},
				"x/SKILL.md":       skillFile("x"),
			},
			want: "applies_when is missing",
		},
		{
			name: "unknown group",
			files: fstest.MapFS{
				"x/extension.json": extensionJSON("x", "skill", "money", ""),
				"x/SKILL.md":       skillFile("x"),
			},
			want: `group "money" is not one of language, architecture, frontend, backend, persistence, testing, ` +
				`operations, workflow, review`,
		},
		{
			name: "frontmatter name",
			files: fstest.MapFS{
				"x/extension.json": extensionJSON("x", "skill", "review", ""),
				"x/SKILL.md":       skillFile("y"),
			},
			want: `frontmatter name "y" does not match the directory`,
		},
		{
			name: "frontmatter description",
			files: fstest.MapFS{
				"x/extension.json": extensionJSON("x", "agent", "review", ""),
				"x/x.md":           {Data: []byte("---\nname: x\n---\n# x\n")},
			},
			want: "frontmatter description is missing",
		},
		{
			name: "unknown requires",
			files: fstest.MapFS{
				"x/extension.json": extensionJSON("x", "skill", "review", `, "requires": ["nope"]`),
				"x/SKILL.md":       skillFile("x"),
			},
			want: `requires "nope" is not a known extension`,
		},
		{
			name: "unknown conflicts",
			files: fstest.MapFS{
				"x/extension.json": extensionJSON("x", "skill", "review", `, "conflicts": ["nope"]`),
				"x/SKILL.md":       skillFile("x"),
			},
			want: `conflicts "nope" is not a known extension`,
		},
		{
			name: "catalog name",
			files: fstest.MapFS{
				"x/extension.json":  extensionJSON("x", "skill", "review", ""),
				"x/SKILL.md":        skillFile("x"),
				"go/extension.json": extensionJSON("go", "skill", "language", ""),
				"go/SKILL.md":       skillFile("go"),
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, storeReasons(t, storeRoot(tt.files), nil)["x"])
		})
	}
}

func TestLoadStoreRequiresMayNameProjectExtensions(t *testing.T) {
	t.Parallel()

	root := storeRoot(fstest.MapFS{
		"a/extension.json": extensionJSON("a", "skill", "review", `, "requires": ["b"]`),
		"a/SKILL.md":       skillFile("a"),
		"b/extension.json": extensionJSON("b", "skill", "review", ""),
		"b/SKILL.md":       skillFile("b"),
	})

	assert.Equal(t, map[string]string{"a": "", "b": ""}, storeReasons(t, root, nil))
}

func TestLoadStoreMarksACatalogNameAsInvalid(t *testing.T) {
	t.Parallel()

	root := storeRoot(fstest.MapFS{
		"go/extension.json": extensionJSON("go", "skill", "language", ""),
		"go/SKILL.md":       skillFile("go"),
	})
	path := &Manifest{Extensions: map[string]Extension{"go": {State: "installed", Source: "path"}}}

	for _, manifest := range []*Manifest{nil, path} {
		store, err := LoadStore(root, fstest.MapFS{}, testCatalog(t), manifest)

		require.NoError(t, err)
		require.EqualError(t, store["go"].Err, `name "go" is also a catalog extension`)
		assert.Equal(t, "language", store["go"].Group)
	}
}

func TestLoadStoreListsAnInstalledProjectExtensionWithoutDirectory(t *testing.T) {
	t.Parallel()

	manifest := &Manifest{Extensions: map[string]Extension{"money": {State: "installed", Source: "path"}}}

	reasons := storeReasons(t, fstest.MapFS{}, manifest)

	assert.Equal(t, "extension.json is missing", reasons["money"])
}

func TestLoadStoreReportsADirectoryWithoutExtensionJSON(t *testing.T) {
	t.Parallel()

	root := storeRoot(fstest.MapFS{"x/SKILL.md": skillFile("x")})

	assert.Equal(t, "extension.json is missing", storeReasons(t, root, nil)["x"])
}

func TestLoadStoreMarksACoreNameAsInvalid(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"framework/skills/work-spec/SKILL.md": {Data: []byte("# work-spec\n")},
		"framework/agents/implementer.md":     {Data: []byte("# implementer\n")},
	}
	root := storeRoot(fstest.MapFS{
		"work-spec/extension.json":   extensionJSON("work-spec", "skill", "review", ""),
		"work-spec/SKILL.md":         skillFile("work-spec"),
		"implementer/extension.json": extensionJSON("implementer", "skill", "review", ""),
		"implementer/SKILL.md":       skillFile("implementer"),
	})

	store, err := LoadStore(root, assets, testCatalog(t), nil)

	require.NoError(t, err)
	require.EqualError(t, store["work-spec"].Err, `name "work-spec" is a core skill`)
	require.EqualError(t, store["implementer"].Err, `name "implementer" is a core role`)
}

func TestLoadStoreIgnoresTheLeftoverOfACatalogExtension(t *testing.T) {
	t.Parallel()

	root := storeRoot(fstest.MapFS{"go/SKILL.md": {Data: []byte("# our go rules\n")}})

	assert.Empty(t, storeReasons(t, root, nil))
}
