package project

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func extensionJSON(name, kind, group, extra string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(`{"name": "` + name + `", "kind": "` + kind + `", "group": "` + group +
		`", "title": "T", "applies_when": "A change"` + extra + `}`)}
}

func catalogAssets() fstest.MapFS {
	return fstest.MapFS{
		"extensions/language/go/extension.json":     extensionJSON("go", "skill", "language", ""),
		"extensions/language/go/SKILL.md":           {Data: []byte("# Go\n")},
		"extensions/frontend/templ/extension.json":  extensionJSON("templ", "skill", "frontend", `, "requires": ["go"]`),
		"extensions/frontend/templ/SKILL.md":        {Data: []byte("# templ\n")},
		"extensions/frontend/react/extension.json":  extensionJSON("react", "skill", "frontend", `, "conflicts": ["templ"]`),
		"extensions/frontend/react/SKILL.md":        {Data: []byte("# React\n")},
		"extensions/review/a11y/extension.json":     extensionJSON("a11y", "agent", "review", ""),
		"extensions/review/a11y/a11y.md":            {Data: []byte("# a11y\n")},
		"extensions/testing/tdd/extension.json":     extensionJSON("tdd", "skill", "testing", ""),
		"extensions/testing/tdd/SKILL.md":           {Data: []byte("# tdd\n")},
		"extensions/frontend/templ/template/x.html": {Data: []byte("x\n")},
	}
}

func testCatalog(t *testing.T) Catalog {
	t.Helper()
	catalog, err := LoadCatalog(catalogAssets())
	require.NoError(t, err)
	return catalog
}

func TestLoadCatalogReadsTheEmbeddedGoExtension(t *testing.T) {
	t.Parallel()

	assets := os.DirFS("../../assets")
	catalog, err := LoadCatalog(assets)

	require.NoError(t, err)
	assert.Equal(t, CatalogExtension{
		Name:        "go",
		Kind:        "skill",
		Group:       "language",
		Title:       "Go language rules",
		Requires:    []string{},
		Conflicts:   []string{},
		AppliesWhen: "A change touches Go code (*.go, go.mod, go.sum)",
		Paths:       []string{"**/*.go", "**/go.mod", "**/go.sum"},
		Detect:      []string{"**/go.mod"},
		Invocation:  "model",
		Dir:         "extensions/language/go",
		Source:      "catalog",
		FS:          assets,
	}, catalog["go"])
	assert.True(t, catalog["extension-authoring"].Default)
	assert.Equal(t, "user", catalog["extension-authoring"].Invocation)
	assert.Empty(t, catalog["extension-authoring"].Detect)
}

func TestLoadCatalogReadsSkillsAndAgents(t *testing.T) {
	t.Parallel()

	catalog := testCatalog(t)

	assert.Len(t, catalog, 5)
	assert.Equal(t, []string{"go"}, catalog["templ"].Requires)
	assert.Equal(t, "agent", catalog["a11y"].Kind)
	assert.Equal(t, "extensions/review/a11y", catalog["a11y"].Dir)
}

func TestLoadCatalogWithoutExtensionsIsEmpty(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog(fstest.MapFS{"framework/rules.md": {Data: []byte("r\n")}})

	require.NoError(t, err)
	assert.Empty(t, catalog)
}

func TestLoadCatalogRejectsADefectiveExtension(t *testing.T) {
	t.Parallel()

	skill := &fstest.MapFile{Data: []byte("# S\n")}
	// decodeFailure is a stable prefix: the json/v2 wording after it differs between toolchains.
	decodeFailure := "read catalog extensions/language/go: decode extension.json: "
	tests := []struct {
		name  string
		files fstest.MapFS
		err   string
		part  string
	}{
		{
			name:  "name differs from its directory",
			files: fstest.MapFS{"extensions/language/go/extension.json": extensionJSON("golang", "skill", "language", "")},
			err:   `read catalog extensions/language/go: name "golang" does not match the directory`,
		},
		{
			name:  "name is not a valid name",
			files: fstest.MapFS{"extensions/language/Go/extension.json": extensionJSON("Go", "skill", "language", "")},
			err:   `read catalog extensions/language/Go: name "Go" is not a name of [a-z][a-z0-9-]*`,
		},
		{
			name:  "group differs from its directory",
			files: fstest.MapFS{"extensions/language/go/extension.json": extensionJSON("go", "skill", "backend", "")},
			err:   `read catalog extensions/language/go: group "backend" does not match the directory`,
		},
		{
			name:  "group is unknown",
			files: fstest.MapFS{"extensions/tools/go/extension.json": extensionJSON("go", "skill", "tools", "")},
			err: `read catalog extensions/tools/go: group "tools" is not one of language, architecture, frontend, ` +
				`backend, persistence, testing, operations, workflow, review`,
		},
		{
			name:  "kind is unknown",
			files: fstest.MapFS{"extensions/language/go/extension.json": extensionJSON("go", "rule", "language", "")},
			err:   `read catalog extensions/language/go: kind "rule" is not skill or agent`,
		},
		{
			name:  "skill has no SKILL.md",
			files: fstest.MapFS{"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", "")},
			err:   "read catalog extensions/language/go: kind skill has no SKILL.md",
		},
		{
			name: "agent has no file named after it",
			files: fstest.MapFS{
				"extensions/review/a11y/extension.json": extensionJSON("a11y", "agent", "review", ""),
				"extensions/review/a11y/SKILL.md":       skill,
			},
			err: "read catalog extensions/review/a11y: kind agent has no a11y.md",
		},
		{
			name: "member is unknown",
			files: fstest.MapFS{
				"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", `, "version": "1"`),
			},
			err:  decodeFailure,
			part: `unknown object member name "version"`,
		},
		{
			name: "requires names no catalog extension",
			files: fstest.MapFS{
				"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", `, "requires": ["c"]`),
				"extensions/language/go/SKILL.md":       skill,
			},
			err: `read catalog extensions/language/go: requires "c" is not a catalog extension`,
		},
		{
			name: "conflicts names no catalog extension",
			files: fstest.MapFS{
				"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", `, "conflicts": ["c"]`),
				"extensions/language/go/SKILL.md":       skill,
			},
			err: `read catalog extensions/language/go: conflicts "c" is not a catalog extension`,
		},
		{
			name: "name is in two groups",
			files: fstest.MapFS{
				"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", ""),
				"extensions/language/go/SKILL.md":       skill,
				"extensions/backend/go/extension.json":  extensionJSON("go", "skill", "backend", ""),
				"extensions/backend/go/SKILL.md":        skill,
			},
			err: `read catalog extensions/language/go: name "go" is also in extensions/backend/go`,
		},
		{
			name: "detect glob is invalid",
			files: fstest.MapFS{
				"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", `, "detect": ["a**"]`),
				"extensions/language/go/SKILL.md":       skill,
			},
			err: `read catalog extensions/language/go: detect "a**" is not a valid glob`,
		},
		{
			name: "invocation is unknown",
			files: fstest.MapFS{
				"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", `, "invocation": "auto"`),
				"extensions/language/go/SKILL.md":       skill,
			},
			err: `read catalog extensions/language/go: invocation "auto" is not model or user`,
		},
		{
			name: "agent is user-invoked",
			files: fstest.MapFS{
				"extensions/review/a11y/extension.json": extensionJSON("a11y", "agent", "review", `, "invocation": "user"`),
				"extensions/review/a11y/a11y.md":        skill,
			},
			err: "read catalog extensions/review/a11y: invocation user is only for a skill",
		},
		{
			name: "default is not a bool",
			files: fstest.MapFS{
				"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", `, "default": "yes"`),
			},
			err:  decodeFailure,
			part: decodeFailure,
		},
		{
			name: "member is duplicated",
			files: fstest.MapFS{
				"extensions/language/go/extension.json": extensionJSON("go", "skill", "language", `, "title": "U"`),
			},
			err:  decodeFailure,
			part: "duplicate",
		},
		{
			name:  "JSON is malformed",
			files: fstest.MapFS{"extensions/language/go/extension.json": {Data: []byte(`{"name": "go"`)}},
			err:   decodeFailure,
			part:  decodeFailure,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := LoadCatalog(tt.files)

			if tt.part == "" {
				require.EqualError(t, err, tt.err)
				return
			}
			require.Error(t, err)
			assert.True(t, strings.HasPrefix(err.Error(), tt.err), "%q lacks prefix %q", err, tt.err)
			assert.ErrorContains(t, err, tt.part)
		})
	}
}

func TestLoadCatalogRejectsEmptyRequiredText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		json string
		err  string
	}{
		{
			name: "empty title",
			json: `{"name": "go", "kind": "skill", "group": "language", "title": "", "applies_when": "A change"}`,
			err:  "read catalog extensions/language/go: title is missing",
		},
		{
			name: "empty applies_when",
			json: `{"name": "go", "kind": "skill", "group": "language", "title": "T", "applies_when": " "}`,
			err:  "read catalog extensions/language/go: applies_when is missing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := LoadCatalog(fstest.MapFS{
				"extensions/language/go/extension.json": {Data: []byte(tt.json)},
				"extensions/language/go/SKILL.md":       {Data: []byte("# Go\n")},
			})

			require.EqualError(t, err, tt.err)
		})
	}
}

func TestResolveSelectsCatalogExtensions(t *testing.T) {
	t.Parallel()

	catalog := testCatalog(t)
	known := "a11y, go, react, tdd, templ"
	tests := []struct {
		name  string
		names []string
		want  []string
		err   *Rejection
	}{
		{name: "no names", names: nil, want: nil},
		{name: "sorted and unique", names: []string{"templ", "go", "templ"}, want: []string{"go", "templ"}},
		{
			name:  "first sorted unknown name",
			names: []string{"zig", "go", "rust"},
			err: &Rejection{
				Code:  "unknown_extension",
				Check: "--ext names a catalog extension",
				Found: `"rust" is not in the catalog; known: ` + known,
				Fix:   "pass --ext with a known name, or none",
			},
		},
		{
			name:  "missing requirement",
			names: []string{"templ"},
			err: &Rejection{
				Code:  "extension_requires",
				Check: "every required extension is selected",
				Found: "templ requires go",
				Fix:   "add --ext go",
			},
		},
		{
			name:  "conflict",
			names: []string{"templ", "react", "go"},
			err: &Rejection{
				Code:  "extension_conflict",
				Check: "no selected extensions conflict",
				Found: "react conflicts with templ",
				Fix:   "drop --ext react or --ext templ",
			},
		},
		{
			name:  "unknown before requirement",
			names: []string{"templ", "zig"},
			err: &Rejection{
				Code:  "unknown_extension",
				Check: "--ext names a catalog extension",
				Found: `"zig" is not in the catalog; known: ` + known,
				Fix:   "pass --ext with a known name, or none",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := catalog.Resolve(tt.names)

			if tt.err != nil {
				assert.Equal(t, tt.err, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveNamesEveryMissingRequirement(t *testing.T) {
	t.Parallel()

	catalog := Catalog{
		"app": {Name: "app", Requires: []string{"go", "sql"}},
		"go":  {Name: "go"},
		"sql": {Name: "sql"},
	}

	_, err := catalog.Resolve([]string{"app"})

	assert.Equal(t, &Rejection{
		Code:  "extension_requires",
		Check: "every required extension is selected",
		Found: "app requires go, sql",
		Fix:   "add --ext go,sql",
	}, err)
}
