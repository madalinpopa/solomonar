package project

import (
	"encoding/json/v2"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ledger() Project { return Project{Name: "ledger", MainBranch: "main", Remote: "origin"} }

func initWants() []Want {
	return []Want{
		{Path: rulesPath, Owner: "core", Content: []byte(contentA)},
		{Path: "docs/README.md", Owner: "seed", Content: []byte(contentSeed)},
	}
}

func withManifestEntry(files map[string]File) map[string]File {
	out := maps.Clone(files)
	out[ManifestPath] = File{Owner: "manifest"}
	return out
}

func actions(plan Plan) map[string]Action {
	got := map[string]Action{}
	for _, c := range plan.Changes {
		got[c.Path] = c.Action
	}
	return got
}

func encoded(t *testing.T, m *Manifest) string {
	t.Helper()
	data, err := m.Encode()
	require.NoError(t, err)
	return string(data)
}

func TestInitPlanFirstInitCreatesEverythingAndTheManifestLast(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, nil)

	plan, manifest, err := InitPlan(root, nil, initWants(), Resolutions{}, ledger(), nil, nil, Decisions{}, framework)

	require.NoError(t, err)
	require.Len(t, plan.Changes, 3)
	assert.Equal(t, map[string]Action{rulesPath: ActionCreate, "docs/README.md": ActionCreate, ManifestPath: ActionCreate},
		actions(plan))
	last := plan.Changes[2]
	assert.Equal(t, ManifestPath, last.Path)
	assert.Equal(t, "manifest", last.Owner)
	decoded, err := DecodeManifest(last.Content, framework)
	require.NoError(t, err)
	assert.Equal(t, manifest, decoded)
	assert.Equal(t, &Manifest{
		SchemaVersion: ManifestSchemaVersion, FrameworkVersion: framework, Project: ledger(),
		Runtimes: map[string]Runtime{}, Extensions: map[string]Extension{},
		Files: map[string]File{
			rulesPath:        {Owner: "core", SHA256: Hash([]byte(contentA))},
			"docs/README.md": {Owner: "seed"},
			ManifestPath:     {Owner: "manifest"},
		},
	}, manifest)
	assert.Empty(t, snapshot(t, dir))
}

func TestInitPlanRerunAfterApplyIsUnchanged(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, nil)
	plan, _, err := InitPlan(root, nil, initWants(), Resolutions{}, ledger(), nil, nil, Decisions{}, framework)
	require.NoError(t, err)
	_, err = Apply(t.Context(), root, plan)
	require.NoError(t, err)
	existing, err := ReadManifest(root, framework)
	require.NoError(t, err)

	plan, _, err = InitPlan(root, existing, initWants(), Resolutions{}, ledger(), nil, nil, Decisions{}, framework)

	require.NoError(t, err)
	assert.Equal(t, map[string]Action{rulesPath: ActionUnchanged, "docs/README.md": ActionSkip, ManifestPath: ActionUnchanged},
		actions(plan))
	assert.Equal(t, Change{Path: ManifestPath, Action: ActionUnchanged, Owner: "manifest"}, plan.Changes[2])
}

// TestInitPlanAdoptsIdenticalFilesWithoutManifest is CLI-D17.
func TestInitPlanAdoptsIdenticalFilesWithoutManifest(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{rulesPath: {content: contentA, mode: 0o644}})

	plan, manifest, err := InitPlan(root, nil, initWants()[:1], Resolutions{}, ledger(), nil, nil, Decisions{}, framework)

	require.NoError(t, err)
	assert.Equal(t, Change{Path: rulesPath, Action: ActionUnchanged, Owner: "core", Reason: "adopted"}, plan.Changes[0])
	assert.Equal(t, Change{Path: ManifestPath, Action: ActionCreate, Owner: "manifest", Content: []byte(encoded(t, manifest))},
		plan.Changes[1])
}

func TestInitPlanUpdatesADifferingManifest(t *testing.T) {
	t.Parallel()

	stale := &Manifest{
		SchemaVersion: ManifestSchemaVersion, FrameworkVersion: framework, Project: ledger(),
		Runtimes: map[string]Runtime{}, Extensions: map[string]Extension{},
		Files: withManifestEntry(map[string]File{rulesPath: {Owner: "core", SHA256: Hash([]byte(contentB))}}),
	}
	current := *stale
	current.Files = withManifestEntry(map[string]File{rulesPath: {Owner: "core", SHA256: Hash([]byte(contentA))}})
	compact, err := json.Marshal(&current, json.Deterministic(true))
	require.NoError(t, err)
	tests := []struct {
		name      string
		rules     string
		manifest  string
		existing  *Manifest
		overwrite []string
	}{
		{
			name: "base advanced by overwrite", rules: contentC, manifest: encoded(t, stale), existing: stale,
			overwrite: []string{rulesPath},
		},
		{name: "same content encoded differently", rules: contentA, manifest: string(compact), existing: &current},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, map[string]entry{
				rulesPath:    {content: tt.rules, mode: 0o644},
				ManifestPath: {content: tt.manifest, mode: 0o644},
			})

			plan, manifest, err := InitPlan(root, tt.existing, initWants()[:1], Resolutions{Overwrite: tt.overwrite}, ledger(), nil, nil, Decisions{}, framework)

			require.NoError(t, err)
			assert.Equal(t, Change{
				Path: ManifestPath, Action: ActionUpdate, Owner: "manifest",
				Content: []byte(encoded(t, manifest)),
			}, plan.Changes[1])
			assert.Equal(t, &current, manifest)
		})
	}
}

func TestInitPlanReturnsTheManifestWithConflicts(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, map[string]entry{rulesPath: {content: contentB, mode: 0o644}})
	before := snapshot(t, dir)

	plan, manifest, err := InitPlan(root, nil, initWants()[:1], Resolutions{}, ledger(), nil, nil, Decisions{}, framework)

	require.NoError(t, err)
	assert.Equal(t, []Change{{Path: rulesPath, Action: ActionConflict, Owner: "core", Reason: notManaged}}, plan.Conflicts())
	require.NotNil(t, manifest)
	assert.Equal(t, File{Owner: "core", SHA256: Hash([]byte(contentA))}, manifest.Files[rulesPath])
	assert.Equal(t, before, snapshot(t, dir))
}

func TestInitPlanPassesClassifyRejectionsThrough(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, nil)

	plan, manifest, err := InitPlan(root, nil, initWants(), Resolutions{Keep: []string{rulesPath}}, ledger(), nil, nil, Decisions{}, framework)

	assert.Equal(t, Resolutions{}.invalid("%s is not a conflict", rulesPath), err)
	assert.Empty(t, plan.Changes)
	assert.Nil(t, manifest)
	assert.Empty(t, snapshot(t, dir))
}

func TestInitPlanFailsWhenTheManifestCannotBeRead(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{ManifestPath + "/inner": {content: "x", mode: 0o644}})

	_, _, err := InitPlan(root, nil, nil, Resolutions{}, ledger(), nil, nil, Decisions{}, framework)

	require.ErrorContains(t, err, "read manifest: ")
}

func TestCheckChoices(t *testing.T) {
	t.Parallel()

	recorded := &Manifest{
		Project:    ledger(),
		Runtimes:   map[string]Runtime{"codex": {}, "claude": {AdversarialModel: "opus"}},
		Extensions: map[string]Extension{},
	}
	modelFix := func(runtime string) string {
		return "change models with `solomonar runtime add " + runtime +
			" --adversarial-model <model>`; init never changes choices"
	}
	runtimeFix := "change runtimes with `solomonar runtime add|remove`; init never changes choices"
	tests := []struct {
		name      string
		requested Requested
		found     string
		fix       string
	}{
		{name: "nothing given"},
		{name: "same choices", requested: Requested{
			Name: "ledger", MainBranch: "main", Remote: "origin", Runtimes: []string{"codex", "claude"}, Extensions: []string{},
		}},
		{
			name: "name differs", requested: Requested{Name: "x", MainBranch: "trunk"},
			found: `manifest name "ledger"; requested "x"`, fix: "rerun without --name; init never changes choices",
		},
		{
			name: "main branch differs", requested: Requested{MainBranch: "trunk", Remote: "upstream"},
			found: `manifest main_branch "main"; requested "trunk"`, fix: "rerun without --main-branch; init never changes choices",
		},
		{
			name: "remote differs", requested: Requested{Remote: "upstream", Runtimes: []string{"claude"}},
			found: `manifest remote "origin"; requested "upstream"`, fix: "rerun without --remote; init never changes choices",
		},
		{
			name: "runtimes differ", requested: Requested{Runtimes: []string{"claude"}, Extensions: []string{"go"}},
			found: "manifest runtimes [claude, codex]; requested [claude]", fix: runtimeFix,
		},
		{
			name: "runtimes requested empty", requested: Requested{Runtimes: []string{}},
			found: "manifest runtimes [claude, codex]; requested none", fix: runtimeFix,
		},
		{
			name: "extensions differ", requested: Requested{Extensions: []string{"tdd", "go"}},
			found: "manifest extensions none; requested [go, tdd]",
			fix:   "change extensions with `solomonar extension add|remove`; init never changes choices",
		},
		{name: "same model", requested: Requested{AdversarialModels: map[string]string{"claude": "opus"}}},
		{
			name: "model differs", requested: Requested{AdversarialModels: map[string]string{"claude": "sonnet"}},
			found: `manifest adversarial_model claude "opus"; requested "sonnet"`, fix: modelFix("claude"),
		},
		{
			name: "model absent from the manifest", requested: Requested{AdversarialModels: map[string]string{"codex": "o3"}},
			found: `manifest adversarial_model codex none; requested "o3"`, fix: modelFix("codex"),
		},
		{
			name:      "extensions checked before models",
			requested: Requested{Extensions: []string{"go"}, AdversarialModels: map[string]string{"claude": "x"}},
			found:     "manifest extensions none; requested [go]",
			fix:       "change extensions with `solomonar extension add|remove`; init never changes choices",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := recorded.CheckChoices(tt.requested)

			if tt.found == "" {
				assert.NoError(t, err)
				return
			}
			assert.Equal(t, &Rejection{Code: "already_initialized", Found: tt.found, Fix: tt.fix}, err)
		})
	}
}

func TestCheckChoicesPrintsAnEmptyManifestListAsNone(t *testing.T) {
	t.Parallel()

	recorded := &Manifest{Project: ledger(), Extensions: map[string]Extension{"go": {}}}

	err := recorded.CheckChoices(Requested{Extensions: []string{}})

	assert.Equal(t, "manifest extensions [go]; requested none", rejectionOf(t, err).Found)
}

func TestInitPlanRecordsTheRuntimes(t *testing.T) {
	t.Parallel()

	runtimes := map[string]Runtime{"claude": {AdversarialModel: "opus"}, "codex": {}}
	root, _ := openTree(t, nil)

	_, manifest, err := InitPlan(root, nil, initWants(), Resolutions{}, ledger(), runtimes, nil, Decisions{}, framework)

	require.NoError(t, err)
	assert.Equal(t, runtimes, manifest.Runtimes)
}

func TestInitPlanRecordsTheExtensions(t *testing.T) {
	t.Parallel()

	extensions := map[string]Extension{
		"go":       {State: "installed", Kind: "skill", Group: "language", Source: "catalog", Decision: "P/D-1"},
		"contrast": {State: "installed", Kind: "agent", Group: "review", Source: "catalog"},
	}
	root, _ := openTree(t, nil)

	plan, manifest, err := InitPlan(root, nil, initWants(), Resolutions{}, ledger(), nil, extensions, Decisions{}, framework)

	require.NoError(t, err)
	assert.Equal(t, extensions, manifest.Extensions)
	decoded, err := DecodeManifest(plan.Changes[len(plan.Changes)-1].Content, framework)
	require.NoError(t, err)
	assert.Equal(t, extensions, decoded.Extensions)
}

func TestInitPlanPutsTheDecisionChangesBeforeTheManifest(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, nil)
	decisions, err := PlanDecisions(root, nil, []CatalogExtension{{Name: "go", Kind: "skill", Group: "language"}},
		byInit())
	require.NoError(t, err)
	extensions := map[string]Extension{
		"go": {State: "installed", Kind: "skill", Group: "language", Source: "catalog", Decision: "P/D-1"},
	}

	plan, manifest, err := InitPlan(root, nil, initWants(), Resolutions{}, ledger(), nil, extensions, decisions, framework)

	require.NoError(t, err)
	var paths []string
	for _, c := range plan.Changes {
		paths = append(paths, c.Path)
	}
	assert.Equal(t, []string{rulesPath, "docs/README.md", "docs/decisions.md", "docs/stack.md", ManifestPath}, paths)
	decoded, err := DecodeManifest(plan.Changes[4].Content, framework)
	require.NoError(t, err)
	assert.Equal(t, manifest, decoded)
	assert.Equal(t, File{Owner: "decision", Entries: []string{"P/D-1"}}, manifest.Files["docs/stack.md"])
}

func TestInitPlanAddsTheIndexToAPresentReadme(t *testing.T) {
	t.Parallel()

	index := []string{decisionsIndex, stackIndex}
	tests := []struct {
		name, readme string
		change       Change
	}{
		{
			name: "with items", readme: "# R\n\n## Documents\n\n- `x`\n\n## Next\n",
			change: Change{
				Path: "docs/README.md", Action: ActionAppend, Owner: "seed",
				Content: []byte("# R\n\n## Documents\n\n- `x`\n" + decisionsIndex + "\n" + stackIndex + "\n\n## Next\n"),
			},
		},
		{
			name: "without items", readme: "## Documents\n\n## Next\n",
			change: Change{
				Path: "docs/README.md", Action: ActionAppend, Owner: "seed",
				Content: []byte("## Documents\n\n" + decisionsIndex + "\n" + stackIndex + "\n\n## Next\n"),
			},
		},
		{
			name: "without the section", readme: "# R\n",
			change: Change{
				Path: "docs/README.md", Action: ActionSkip, Owner: "seed",
				Reason: `exists; add under "## Documents": ` + decisionsIndex + "; " + stackIndex,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, map[string]entry{"docs/README.md": {content: tt.readme, mode: 0o644}})

			plan, manifest, err := InitPlan(root, nil, initWants(), Resolutions{}, ledger(), nil, nil,
				Decisions{Index: index}, framework)

			require.NoError(t, err)
			assert.Equal(t, tt.change, plan.Changes[1])
			assert.Equal(t, File{Owner: "seed"}, manifest.Files["docs/README.md"])
		})
	}
}

func TestInitPlanSkipsAPresentReadmeWithoutIndexLines(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{"docs/README.md": {content: "## Documents\n", mode: 0o644}})

	plan, _, err := InitPlan(root, nil, initWants(), Resolutions{}, ledger(), nil, nil, Decisions{}, framework)

	require.NoError(t, err)
	assert.Equal(t, Change{Path: "docs/README.md", Action: ActionSkip, Owner: "seed"}, plan.Changes[1])
}
