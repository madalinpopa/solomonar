package project

import (
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	noExtensions = "- Skills: none.\n- Agents: none. An installed agent is a specialist reviewer, " +
		"required by triage when its \"applies when\" matches."
	adoptedReason = `adopted; existing content moved under "## Project notes"`
	agentsPath    = "AGENTS.md"
	managedA      = "# Project\n\nA\n\n"
	managedB      = "# Project\n\nB\n\n"
	managedC      = "# Project\n\nC\n\n"
	offerNotes    = "## Project notes\n\n<!-- notes -->\n"
	myNotes       = "## Project notes\n\nmine\n"
)

func TestCoreWantsSmallTree(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"framework/rules.md":            {Data: []byte("rules\n")},
		"framework/agents/z.md":         {Data: []byte("z\n")},
		"framework/agents-x/a.md":       {Data: []byte("x\n")},
		"templates/AGENTS.md":           {Data: []byte("v{{ .FrameworkVersion }}\n{{ .Extensions }}\n")},
		"templates/docs/README.md":      {Data: []byte("# {{ .ProjectName }}\n{{ .Objective }}\n")},
		"templates/state.json":          {Data: []byte("{}\n")},
		"extensions/language/go/x.json": {Data: []byte("{}\n")},
	}

	wants, err := CoreWants(assets, Choices{Name: "ledger", Objective: "ship", FrameworkVersion: "1.2.3"})

	require.NoError(t, err)
	assert.Equal(t, []Want{
		{Path: "docs/solomonar/agents-x/a.md", Owner: "core", Content: []byte("x\n")},
		{Path: "docs/solomonar/agents/z.md", Owner: "core", Content: []byte("z\n")},
		{Path: "docs/solomonar/rules.md", Owner: "core", Content: []byte("rules\n")},
		{Path: agentsPath, Owner: "core+notes", Content: []byte("v1.2.3\n" + noExtensions + "\n")},
		{Path: "docs/README.md", Owner: "seed", Content: []byte("# ledger\nship\n")},
	}, wants)
}

func TestCoreWantsMissingTemplate(t *testing.T) {
	t.Parallel()

	_, err := CoreWants(fstest.MapFS{"framework/rules.md": {Data: []byte("x")}}, Choices{})

	require.ErrorContains(t, err, "render templates/AGENTS.md")
}

func TestCoreWantsEmbeddedAssets(t *testing.T) {
	t.Parallel()

	assets := os.DirFS("../../assets")
	var framework []string
	require.NoError(t, fs.WalkDir(assets, "framework", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			framework = append(framework, "docs/solomonar/"+strings.TrimPrefix(path, "framework/"))
		}
		return err
	}))
	slices.Sort(framework)

	wants, err := CoreWants(assets, Choices{Name: "ledger", FrameworkVersion: "9.8.7"})

	require.NoError(t, err)
	var paths, owners []string
	for _, w := range wants {
		paths = append(paths, w.Path)
		owners = append(owners, w.Owner)
	}
	assert.Equal(t, append(framework, agentsPath, "docs/README.md"), paths)
	assert.Equal(t, "core+notes", owners[len(owners)-2])
	assert.Equal(t, "seed", owners[len(owners)-1])
	assert.Equal(t, []string{"core"}, compactOwners(owners[:len(owners)-2]))
	agents := string(wants[len(wants)-2].Content)
	assert.Contains(t, agents, "Framework version: 9.8.7.")
	assert.Contains(t, agents, "\n"+noExtensions+"\n")
	assert.Contains(t, agents, "\n## Project notes\n")
	readme := string(wants[len(wants)-1].Content)
	assert.True(t, strings.HasPrefix(readme, "# ledger\n"))
	assert.Contains(t, readme, "## Current objective\n\nNot established.\n")

	wants, err = CoreWants(assets, Choices{Name: "ledger", Objective: "Track spending.", FrameworkVersion: "9.8.7"})

	require.NoError(t, err)
	assert.Contains(t, string(wants[len(wants)-1].Content), "## Current objective\n\nTrack spending.\n")
}

func compactOwners(owners []string) []string {
	var out []string
	for _, o := range owners {
		if len(out) == 0 || out[len(out)-1] != o {
			out = append(out, o)
		}
	}
	return out
}

func agentsManifest(managedPart string) *Manifest {
	return &Manifest{Files: map[string]File{agentsPath: {Owner: "core+notes", SHA256: Hash([]byte(managedPart))}}}
}

func TestClassifyAgents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		disk      string
		manifest  *Manifest
		overwrite []string
		action    Action
		reason    string
		content   string
	}{
		{name: "absent", action: ActionCreate, content: managedC + offerNotes},
		{
			name: "managed with edited notes", disk: managedC + myNotes, manifest: agentsManifest(managedC),
			action: ActionUnchanged,
		},
		{
			name: "managed upstream change keeps notes", disk: managedA + myNotes, manifest: agentsManifest(managedA),
			action: ActionUpdate, content: managedC + myNotes,
		},
		{
			name: "managed without the heading", disk: managedA, manifest: agentsManifest(managedA),
			action: ActionUpdate, content: managedC + offerNotes,
		},
		{
			name: "managed local edit kept", disk: managedB + myNotes, manifest: agentsManifest(managedC),
			action: ActionKeep,
		},
		{
			name: "managed local edit and upstream change", disk: managedB + myNotes, manifest: agentsManifest(managedA),
			action: ActionConflict, reason: modified,
		},
		{
			name: "managed conflict overwritten", disk: managedB + myNotes, manifest: agentsManifest(managedA),
			overwrite: []string{agentsPath}, action: ActionUpdate, content: managedC + myNotes,
		},
		{
			name: "managed conflict without the heading overwritten", disk: managedB, manifest: agentsManifest(managedA),
			overwrite: []string{agentsPath}, action: ActionUpdate, content: managedC + offerNotes,
		},
		{name: "unmanaged identical", disk: managedC + offerNotes, action: ActionUnchanged, reason: "adopted"},
		{
			name: "unmanaged different", disk: "# Old\n\nrules\n", action: ActionUpdate, reason: adoptedReason,
			content: managedC + "## Project notes\n\n# Old\n\nrules\n",
		},
		{
			name: "unmanaged overwritten", disk: "# Old\n", overwrite: []string{agentsPath},
			action: ActionUpdate, content: managedC + offerNotes,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			disk := map[string]entry{}
			if tt.disk != "" {
				disk[agentsPath] = entry{content: tt.disk, mode: 0o644}
			}
			root, _ := openTree(t, disk)
			want := Want{Path: agentsPath, Owner: "core+notes", Content: []byte(managedC + offerNotes)}

			plan, files, err := Classify(root, tt.manifest, []Want{want}, Resolutions{Overwrite: tt.overwrite})

			require.NoError(t, err)
			assert.Equal(t, Change{
				Path: agentsPath, Action: tt.action, Owner: "core+notes", Reason: tt.reason,
				Content: bytesOrNil(tt.content),
			}, plan.Changes[0])
			assert.Equal(t, map[string]File{agentsPath: {Owner: "core+notes", SHA256: Hash([]byte(managedC))}}, files)
		})
	}
}

func bytesOrNil(s string) []byte {
	if s == "" {
		return nil
	}
	return []byte(s)
}

func TestClassifyAgentsCannotBeKept(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{agentsPath: {content: "# Old\n", mode: 0o644}})
	want := Want{Path: agentsPath, Owner: "core+notes", Content: []byte(managedC + offerNotes)}

	_, _, err := Classify(root, nil, []Want{want}, Resolutions{Keep: []string{agentsPath}})

	assert.Equal(t, &Rejection{
		Code:  "invalid_value",
		Check: "--keep and --overwrite resolve conflicts",
		Found: "AGENTS.md cannot be kept",
		Fix:   "adopt it (default) or pass --overwrite AGENTS.md",
	}, err)
}

func TestCoreWantsRendersTheExtensionsBlock(t *testing.T) {
	t.Parallel()

	const agentsNote = "- Agents: an installed agent is a specialist reviewer, required by triage when its " +
		"\"applies when\" matches.\n"
	gin := CatalogExtension{Name: "gin", Kind: "skill", Group: "backend", AppliesWhen: "A change adds routes"}
	golang := CatalogExtension{
		Name: "go", Kind: "skill", Group: "language", AppliesWhen: "A change touches Go code",
		Paths: []string{"**/*.go", "**/go.mod"},
	}
	chi := CatalogExtension{Name: "chi", Kind: "skill", Group: "backend", AppliesWhen: "Routes", Paths: []string{}}
	contrast := CatalogExtension{Name: "contrast", Kind: "agent", Group: "review", AppliesWhen: "UI changes"}
	perf := CatalogExtension{
		Name: "perf", Kind: "agent", Group: "operations", AppliesWhen: "Hot paths", Paths: []string{"cmd/**"},
	}
	author := CatalogExtension{
		Name: "author", Kind: "skill", Group: "workflow", AppliesWhen: "A request authors an extension",
		Invocation: "user",
	}
	tests := []struct {
		name       string
		extensions []CatalogExtension
		block      string
	}{
		{name: "none", block: noExtensions},
		{
			name: "skills only, by group then name", extensions: []CatalogExtension{gin, golang, chi},
			block: "- Skills\n" +
				"  - `go` (language, P/D-1): A change touches Go code; paths: `**/*.go`, `**/go.mod`\n" +
				"  - `chi` (backend, P/D-2): Routes\n" +
				"  - `gin` (backend, P/D-2): A change adds routes\n" +
				"- Agents: none. An installed agent is a specialist reviewer, required by triage when its " +
				"\"applies when\" matches.",
		},
		{
			name: "agents only", extensions: []CatalogExtension{contrast, perf},
			block: "- Skills: none.\n" + agentsNote +
				"  - `perf` (operations, P/D-3): Hot paths; paths: `cmd/**`\n" +
				"  - `contrast` (review): UI changes",
		},
		{
			name: "user-invoked skills after the agents", extensions: []CatalogExtension{author, golang, contrast},
			block: "- Skills\n" +
				"  - `go` (language, P/D-1): A change touches Go code; paths: `**/*.go`, `**/go.mod`\n" + agentsNote +
				"  - `contrast` (review): UI changes\n" +
				"- User-invoked skills: load one only when the user starts it, never because it applies.\n" +
				"  - `author` (workflow): A request authors an extension",
		},
		{
			name: "skills and agents", extensions: []CatalogExtension{contrast, golang},
			block: "- Skills\n" +
				"  - `go` (language, P/D-1): A change touches Go code; paths: `**/*.go`, `**/go.mod`\n" + agentsNote +
				"  - `contrast` (review): UI changes",
		},
	}
	assets := fstest.MapFS{
		"framework/rules.md":       {Data: []byte("rules\n")},
		"templates/AGENTS.md":      {Data: []byte("## Extensions\n\n{{ .Extensions }}\n\n## Next\n")},
		"templates/docs/README.md": {Data: []byte("# {{ .ProjectName }}\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wants, err := CoreWants(assets, Choices{
				Name: "ledger", Extensions: tt.extensions,
				Decisions: map[string]string{"language": "P/D-1", "backend": "P/D-2", "operations": "P/D-3"},
			})

			require.NoError(t, err)
			got := wantedContent(t, wants, agentsPath)
			assert.Equal(t, "## Extensions\n\n"+tt.block+"\n\n## Next\n", got)
		})
	}
}

func TestCoreWantsListsTheCreatedDocumentsInTheReadme(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"framework/rules.md":       {Data: []byte("rules\n")},
		"templates/AGENTS.md":      {Data: []byte("a\n")},
		"templates/docs/README.md": {Data: []byte("# {{ .ProjectName }}\n\n## Documents\n\n- `x`\n\n## Next\n")},
	}

	wants, err := CoreWants(assets, Choices{Name: "ledger", Documents: []string{decisionsIndex, stackIndex}})

	require.NoError(t, err)
	assert.Equal(t, "# ledger\n\n## Documents\n\n- `x`\n"+decisionsIndex+"\n"+stackIndex+"\n\n## Next\n",
		wantedContent(t, wants, "docs/README.md"))
}
