package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	notRegular  = "exists and is not a regular file"
	modified    = "modified since installation (manifest sha256 differs)"
	notManaged  = "exists and is not managed by solomonar"
	rulesPath   = "docs/solomonar/rules.md"
	contentA    = "A\n"
	contentB    = "B\n"
	contentC    = "C\n"
	contentD    = "D\n"
	contentSeed = "seed\n"
)

func managed(path, base string) *Manifest {
	return &Manifest{Files: map[string]File{path: {Owner: "core", SHA256: Hash([]byte(base))}}}
}

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		disk      map[string]entry
		manifest  *Manifest
		want      Want
		keep      []string
		overwrite []string
		action    Action
		reason    string
		content   string
		file      File
	}{
		{
			name: "seed absent", want: Want{Path: "docs/README.md", Owner: "seed", Content: []byte(contentSeed)},
			action: ActionCreate, content: contentSeed, file: File{Owner: "seed"},
		},
		{
			name: "seed present", disk: map[string]entry{"docs/README.md": {content: "mine\n", mode: 0o644}},
			want:   Want{Path: "docs/README.md", Owner: "seed", Content: []byte(contentSeed)},
			action: ActionSkip, file: File{Owner: "seed"},
		},
		{
			name: "absent", want: Want{Path: rulesPath, Owner: "core", Content: []byte(contentA)},
			action: ActionCreate, content: contentA, file: File{Owner: "core", SHA256: Hash([]byte(contentA))},
		},
		{
			name: "not a regular file", disk: map[string]entry{rulesPath + "/inner": {content: "x", mode: 0o644}},
			want:   Want{Path: rulesPath, Owner: "core", Content: []byte(contentA)},
			action: ActionConflict, reason: notRegular, file: File{Owner: "core", SHA256: Hash([]byte(contentA))},
		},
		{
			name: "managed unchanged", disk: map[string]entry{rulesPath: {content: contentC, mode: 0o644}},
			manifest: managed(rulesPath, contentA), want: Want{Path: rulesPath, Owner: "core", Content: []byte(contentC)},
			action: ActionUnchanged, file: File{Owner: "core", SHA256: Hash([]byte(contentC))},
		},
		{
			name: "managed update", disk: map[string]entry{rulesPath: {content: contentA, mode: 0o644}},
			manifest: managed(rulesPath, contentA), want: Want{Path: rulesPath, Owner: "core", Content: []byte(contentC)},
			action: ActionUpdate, content: contentC, file: File{Owner: "core", SHA256: Hash([]byte(contentC))},
		},
		{
			name: "managed keep", disk: map[string]entry{rulesPath: {content: contentB, mode: 0o644}},
			manifest: managed(rulesPath, contentA), want: Want{Path: rulesPath, Owner: "core", Content: []byte(contentA)},
			action: ActionKeep, file: File{Owner: "core", SHA256: Hash([]byte(contentA))},
		},
		{
			name: "managed conflict", disk: map[string]entry{rulesPath: {content: contentB, mode: 0o644}},
			manifest: managed(rulesPath, contentA), want: Want{Path: rulesPath, Owner: "core", Content: []byte(contentC)},
			action: ActionConflict, reason: modified, file: File{Owner: "core", SHA256: Hash([]byte(contentC))},
		},
		{
			name: "unmanaged adopted", disk: map[string]entry{rulesPath: {content: contentA, mode: 0o644}},
			want:   Want{Path: rulesPath, Owner: "core", Content: []byte(contentA)},
			action: ActionUnchanged, reason: "adopted", file: File{Owner: "core", SHA256: Hash([]byte(contentA))},
		},
		{
			name: "unmanaged conflict", disk: map[string]entry{rulesPath: {content: contentB, mode: 0o644}},
			want:   Want{Path: rulesPath, Owner: "core", Content: []byte(contentA)},
			action: ActionConflict, reason: notManaged, file: File{Owner: "core", SHA256: Hash([]byte(contentA))},
		},
		{
			name: "conflict resolved by keep", disk: map[string]entry{rulesPath: {content: contentB, mode: 0o644}},
			want: Want{Path: rulesPath, Owner: "core", Content: []byte(contentA)}, keep: []string{rulesPath},
			action: ActionKeep, file: File{Owner: "core", SHA256: Hash([]byte(contentA))},
		},
		{
			name: "conflict resolved by overwrite", disk: map[string]entry{rulesPath: {content: contentB, mode: 0o644}},
			manifest: managed(rulesPath, contentA), want: Want{Path: rulesPath, Owner: "core", Content: []byte(contentC)},
			overwrite: []string{rulesPath},
			action:    ActionUpdate, content: contentC, file: File{Owner: "core", SHA256: Hash([]byte(contentC))},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, tt.disk)

			plan, files, err := Classify(root, tt.manifest, []Want{tt.want}, tt.keep, tt.overwrite)

			require.NoError(t, err)
			require.Len(t, plan.Changes, 1)
			change := plan.Changes[0]
			assert.Equal(t, tt.want.Path, change.Path)
			assert.Equal(t, tt.want.Owner, change.Owner)
			assert.Equal(t, tt.action, change.Action)
			assert.Equal(t, tt.reason, change.Reason)
			assert.Equal(t, tt.content, string(change.Content))
			assert.Equal(t, map[string]File{tt.want.Path: tt.file}, files)
		})
	}
}

func TestClassifySymlinkIsNotRegular(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, map[string]entry{"target.md": {content: contentA, mode: 0o644}})
	require.NoError(t, os.Symlink("target.md", filepath.Join(dir, "link.md")))

	plan, _, err := Classify(root, nil, []Want{{Path: "link.md", Owner: "core", Content: []byte(contentA)}}, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, ActionConflict, plan.Changes[0].Action)
	assert.Equal(t, notRegular, plan.Changes[0].Reason)
}

func TestClassifyKeepsWantedOrder(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{"b.md": {content: contentB, mode: 0o644}})
	wants := []Want{
		{Path: "c.md", Owner: "core", Content: []byte(contentC)},
		{Path: "b.md", Owner: "core", Content: []byte(contentB)},
		{Path: "a.md", Owner: "core", Content: []byte(contentA)},
	}

	plan, _, err := Classify(root, nil, wants, nil, nil)

	require.NoError(t, err)
	var paths []string
	for _, c := range plan.Changes {
		paths = append(paths, c.Path)
	}
	assert.Equal(t, []string{"c.md", "b.md", "a.md"}, paths)
}

func TestClassifyRejectsInvalidResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		keep      []string
		overwrite []string
		found     string
	}{
		{name: "keep names a non-conflict", keep: []string{"same.md"}, found: "same.md is not a conflict"},
		{name: "overwrite names an unknown path", overwrite: []string{"other.md"}, found: "other.md is not a conflict"},
		{
			name: "same path in both", keep: []string{"differs.md"}, overwrite: []string{"differs.md"},
			found: "differs.md is in both --keep and --overwrite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, map[string]entry{
				"same.md":    {content: contentA, mode: 0o644},
				"differs.md": {content: contentB, mode: 0o644},
			})
			wants := []Want{
				{Path: "same.md", Owner: "core", Content: []byte(contentA)},
				{Path: "differs.md", Owner: "core", Content: []byte(contentA)},
			}

			_, _, err := Classify(root, nil, wants, tt.keep, tt.overwrite)

			assert.Equal(t, &Rejection{
				Code:  "invalid_value",
				Check: "--keep and --overwrite resolve conflicts",
				Found: tt.found,
				Fix:   "pass --keep or --overwrite once per conflicting path",
			}, err)
		})
	}
}

// TestClassifyKeepFollowsUpstreamBase is F-02: the base follows the offers; kept bytes stay.
func TestClassifyKeepFollowsUpstreamBase(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{rulesPath: {content: contentB, mode: 0o644}})
	var manifest *Manifest
	step := func(offer string, keep []string) Change {
		t.Helper()
		wants := []Want{{Path: rulesPath, Owner: "core", Content: []byte(offer)}}
		plan, files, err := Classify(root, manifest, wants, keep, nil)
		require.NoError(t, err)
		if len(plan.Conflicts()) == 0 {
			manifest = &Manifest{Files: files}
		}
		return plan.Changes[0]
	}

	assert.Equal(t, Change{Path: rulesPath, Action: ActionConflict, Owner: "core", Reason: notManaged}, step(contentA, nil))
	assert.Equal(t, ActionKeep, step(contentA, []string{rulesPath}).Action)
	assert.Equal(t, Hash([]byte(contentA)), manifest.Files[rulesPath].SHA256)

	assert.Equal(t, Change{Path: rulesPath, Action: ActionConflict, Owner: "core", Reason: modified}, step(contentC, nil))
	assert.Equal(t, ActionKeep, step(contentC, []string{rulesPath}).Action)
	assert.Equal(t, Hash([]byte(contentC)), manifest.Files[rulesPath].SHA256)

	assert.Equal(t, ActionKeep, step(contentC, nil).Action)
	assert.Equal(t, Change{Path: rulesPath, Action: ActionConflict, Owner: "core", Reason: modified}, step(contentD, nil))

	disk, err := root.ReadFile(rulesPath)
	require.NoError(t, err)
	assert.Equal(t, contentB, string(disk))
}

// TestClassifyAdoptsIdenticalFile is CLI-D17: an identical unmanaged file is adopted.
func TestClassifyAdoptsIdenticalFile(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{rulesPath: {content: contentA, mode: 0o644}})

	plan, files, err := Classify(root, nil, []Want{{Path: rulesPath, Owner: "core", Content: []byte(contentA)}}, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, Change{Path: rulesPath, Action: ActionUnchanged, Owner: "core", Reason: "adopted"}, plan.Changes[0])
	assert.Equal(t, File{Owner: "core", SHA256: Hash([]byte(contentA))}, files[rulesPath])
	assert.Empty(t, plan.Conflicts())
}
