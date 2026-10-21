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
	root, dir := openTree(t, nil)

	plan, manifest, err := InitPlan(root, nil, initWants(), nil, nil, ledger(), framework)

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
	root, _ := openTree(t, nil)
	plan, _, err := InitPlan(root, nil, initWants(), nil, nil, ledger(), framework)
	require.NoError(t, err)
	_, err = Apply(t.Context(), root, plan)
	require.NoError(t, err)
	existing, err := ReadManifest(root, framework)
	require.NoError(t, err)

	plan, _, err = InitPlan(root, existing, initWants(), nil, nil, ledger(), framework)

	require.NoError(t, err)
	assert.Equal(t, map[string]Action{rulesPath: ActionUnchanged, "docs/README.md": ActionSkip, ManifestPath: ActionUnchanged},
		actions(plan))
	assert.Equal(t, Change{Path: ManifestPath, Action: ActionUnchanged, Owner: "manifest"}, plan.Changes[2])
}

// TestInitPlanAdoptsIdenticalFilesWithoutManifest is CLI-D17.
func TestInitPlanAdoptsIdenticalFilesWithoutManifest(t *testing.T) {
	root, _ := openTree(t, map[string]entry{rulesPath: {content: contentA, mode: 0o644}})

	plan, manifest, err := InitPlan(root, nil, initWants()[:1], nil, nil, ledger(), framework)

	require.NoError(t, err)
	assert.Equal(t, Change{Path: rulesPath, Action: ActionUnchanged, Owner: "core", Reason: "adopted"}, plan.Changes[0])
	assert.Equal(t, Change{Path: ManifestPath, Action: ActionCreate, Owner: "manifest", Content: []byte(encoded(t, manifest))},
		plan.Changes[1])
}

func TestInitPlanUpdatesADifferingManifest(t *testing.T) {
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
			root, _ := openTree(t, map[string]entry{
				rulesPath:    {content: tt.rules, mode: 0o644},
				ManifestPath: {content: tt.manifest, mode: 0o644},
			})

			plan, manifest, err := InitPlan(root, tt.existing, initWants()[:1], nil, tt.overwrite, ledger(), framework)

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
	root, dir := openTree(t, map[string]entry{rulesPath: {content: contentB, mode: 0o644}})
	before := snapshot(t, dir)

	plan, manifest, err := InitPlan(root, nil, initWants()[:1], nil, nil, ledger(), framework)

	require.NoError(t, err)
	assert.Equal(t, []Change{{Path: rulesPath, Action: ActionConflict, Owner: "core", Reason: notManaged}}, plan.Conflicts())
	require.NotNil(t, manifest)
	assert.Equal(t, File{Owner: "core", SHA256: Hash([]byte(contentA))}, manifest.Files[rulesPath])
	assert.Equal(t, before, snapshot(t, dir))
}

func TestInitPlanPassesClassifyRejectionsThrough(t *testing.T) {
	root, dir := openTree(t, nil)

	plan, manifest, err := InitPlan(root, nil, initWants(), []string{rulesPath}, nil, ledger(), framework)

	assert.Equal(t, invalidResolution("%s is not a conflict", rulesPath), err)
	assert.Empty(t, plan.Changes)
	assert.Nil(t, manifest)
	assert.Empty(t, snapshot(t, dir))
}

func TestInitPlanFailsWhenTheManifestCannotBeRead(t *testing.T) {
	root, _ := openTree(t, map[string]entry{ManifestPath + "/inner": {content: "x", mode: 0o644}})

	_, _, err := InitPlan(root, nil, nil, nil, nil, ledger(), framework)

	require.ErrorContains(t, err, "read manifest: ")
}

func TestCheckChoices(t *testing.T) {
	recorded := &Manifest{
		Project:    ledger(),
		Runtimes:   map[string]Runtime{"codex": {}, "claude": {}},
		Extensions: map[string]Extension{},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	recorded := &Manifest{Project: ledger(), Extensions: map[string]Extension{"go": {}}}

	err := recorded.CheckChoices(Requested{Extensions: []string{}})

	assert.Equal(t, "manifest extensions [go]; requested none", rejectionOf(t, err).Found)
}
