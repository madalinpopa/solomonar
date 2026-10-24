package project

import (
	"context"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type entry struct {
	content string
	mode    fs.FileMode
	dir     bool
}

func openTree(t *testing.T, files map[string]entry) (*os.Root, string) {
	t.Helper()
	dir := t.TempDir()
	for name, e := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
		require.NoError(t, os.WriteFile(target, []byte(e.content), e.mode))
		require.NoError(t, os.Chmod(target, e.mode))
	}
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })
	return root, dir
}

func snapshot(t *testing.T, dir string) map[string]entry {
	t.Helper()
	tree := map[string]entry{}
	err := filepath.WalkDir(dir, func(name string, d fs.DirEntry, err error) error {
		if err != nil || name == dir {
			return err
		}
		rel, err := filepath.Rel(dir, name)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			tree[filepath.ToSlash(rel)] = entry{mode: info.Mode().Perm(), dir: true}
			return nil
		}
		content, err := os.ReadFile(name)
		tree[filepath.ToSlash(rel)] = entry{content: string(content), mode: info.Mode().Perm()}
		return err
	})
	require.NoError(t, err)
	return tree
}

func original() map[string]entry {
	return map[string]entry{
		"README.md":       {content: "readme\n", mode: 0o644},
		"bin/run.sh":      {content: "#!/bin/sh\n", mode: 0o755},
		"docs/old.md":     {content: "old\n", mode: 0o600},
		"docs/notes.txt":  {content: "notes\n", mode: 0o644},
		"blocked/file.md": {content: "x\n", mode: 0o644},
	}
}

func TestApplyWritesActingChangesInPlanOrder(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, original())
	plan := Plan{Changes: []Change{
		{Path: "docs/new/deep/a.md", Action: ActionCreate, Content: []byte("first\n")},
		{Path: "docs/new/deep/a.md", Action: ActionUpdate, Content: []byte("second\n")},
		{Path: "bin/run.sh", Action: ActionUpdate, Content: []byte("#!/bin/sh\nexit 0\n")},
		{Path: "docs/old.md", Action: ActionRemove},
		{Path: "README.md", Action: ActionUnchanged, Content: []byte("ignored")},
		{Path: "docs/notes.txt", Action: ActionKeep, Content: []byte("ignored")},
		{Path: "docs/conflict.md", Action: ActionConflict, Content: []byte("ignored")},
		{Path: "docs/skip.md", Action: ActionSkip, Content: []byte("ignored")},
	}}

	_, err := Apply(t.Context(), root, plan)
	require.NoError(t, err)

	want := original()
	delete(want, "docs/old.md")
	want["bin/run.sh"] = entry{content: "#!/bin/sh\nexit 0\n", mode: 0o755}
	want["docs/new"] = entry{mode: 0o755, dir: true}
	want["docs/new/deep"] = entry{mode: 0o755, dir: true}
	want["docs/new/deep/a.md"] = entry{content: "second\n", mode: 0o644}
	want["bin"] = entry{mode: 0o755, dir: true}
	want["docs"] = entry{mode: 0o755, dir: true}
	want["blocked"] = entry{mode: 0o755, dir: true}
	assert.Equal(t, want, snapshot(t, dir))
}

func TestApplyRejectsChangesThatDisagreeWithTheDisk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change Change
	}{
		{name: "create over an existing file", change: Change{Path: "README.md", Action: ActionCreate}},
		{name: "update a missing file", change: Change{Path: "missing.md", Action: ActionUpdate}},
		{name: "remove a missing file", change: Change{Path: "missing.md", Action: ActionRemove}},
		{name: "update a directory", change: Change{Path: "docs", Action: ActionUpdate}},
		{name: "path escaping the root", change: Change{Path: "../outside.md", Action: ActionCreate}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, dir := openTree(t, original())
			before := snapshot(t, dir)

			_, err := Apply(t.Context(), root, Plan{Changes: []Change{tt.change}})

			require.ErrorContains(t, err, "apply "+tt.change.Path+": ")
			assert.Equal(t, before, snapshot(t, dir))
		})
	}
	assert.NoFileExists(t, filepath.Join(filepath.Dir(t.TempDir()), "outside.md"))
}

func TestApplyRollsBackEarlierChangesWhenAWriteFails(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, original())
	before := snapshot(t, dir)
	plan := Plan{Changes: []Change{
		{Path: "docs/new/a.md", Action: ActionCreate, Content: []byte("a\n")},
		{Path: "bin/run.sh", Action: ActionUpdate, Content: []byte("changed\n")},
		{Path: "docs/old.md", Action: ActionRemove},
		{Path: "blocked/file.md/inner.md", Action: ActionCreate, Content: []byte("fails\n")},
	}}

	journal, err := Apply(t.Context(), root, plan)

	require.ErrorContains(t, err, "apply blocked/file.md/inner.md: ")
	assert.Nil(t, journal)
	assert.Equal(t, before, snapshot(t, dir))
}

func TestApplyStopsWhenCancelled(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, original())
	before := snapshot(t, dir)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := Apply(ctx, root, Plan{Changes: []Change{
		{Path: "docs/new.md", Action: ActionCreate, Content: []byte("new\n")},
	}})

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "apply docs/new.md: ")
	assert.Equal(t, before, snapshot(t, dir))
}

func TestRollbackRestoresTheTreeAfterASuccessfulApply(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, original())
	before := snapshot(t, dir)
	plan := Plan{Changes: []Change{
		{Path: "docs/new/deep/a.md", Action: ActionCreate, Content: []byte("a\n")},
		{Path: "bin/run.sh", Action: ActionUpdate, Content: []byte("changed\n")},
		{Path: "docs/old.md", Action: ActionRemove},
		{Path: "README.md", Action: ActionUpdate, Content: []byte("one\n")},
		{Path: "README.md", Action: ActionUpdate, Content: []byte("two\n")},
	}}
	journal, err := Apply(t.Context(), root, plan)
	require.NoError(t, err)

	require.NoError(t, journal.Rollback())
	assert.Equal(t, before, snapshot(t, dir))

	require.NoError(t, journal.Rollback())
	assert.Equal(t, before, snapshot(t, dir))
}

func TestRollbackKeepsACreatedDirectoryThatIsNotEmpty(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, original())
	journal, err := Apply(t.Context(), root, Plan{Changes: []Change{
		{Path: "docs/new/a.md", Action: ActionCreate, Content: []byte("a\n")},
	}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "new", "user.md"), nil, 0o644))

	require.NoError(t, journal.Rollback())

	assert.NoFileExists(t, filepath.Join(dir, "docs", "new", "a.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "new", "user.md"))
}

func TestRollbackReportsAFailedRestoreAndRestoresTheRest(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, original())
	journal, err := Apply(t.Context(), root, Plan{Changes: []Change{
		{Path: "docs/old.md", Action: ActionRemove},
		{Path: "bin/run.sh", Action: ActionUpdate, Content: []byte("changed\n")},
	}})
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(filepath.Join(dir, "docs", "old.md"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "old.md", "x"), nil, 0o644))

	err = journal.Rollback()

	require.ErrorContains(t, err, "restore docs/old.md: ")
	assert.Equal(t, original()["bin/run.sh"], snapshot(t, dir)["bin/run.sh"])
}

func TestPlanConflictsReturnsConflictsInPlanOrder(t *testing.T) {
	t.Parallel()

	plan := Plan{Changes: []Change{
		{Path: "b.md", Action: ActionConflict, Reason: "modified"},
		{Path: "c.md", Action: ActionCreate},
		{Path: "a.md", Action: ActionConflict, Reason: "untracked"},
	}}

	assert.Equal(t, []Change{plan.Changes[0], plan.Changes[2]}, plan.Conflicts())
	assert.Empty(t, Plan{}.Conflicts())
}

func TestApplyAppendWritesTheNewContentAndRollsBack(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, original())
	before := snapshot(t, dir)

	journal, err := Apply(t.Context(), root, Plan{Changes: []Change{
		{Path: "docs/old.md", Action: ActionAppend, Content: []byte("old\nmore\n")},
	}})

	require.NoError(t, err)
	want := maps.Clone(before)
	want["docs/old.md"] = entry{content: "old\nmore\n", mode: 0o600}
	assert.Equal(t, want, snapshot(t, dir))
	require.NoError(t, journal.Rollback())
	assert.Equal(t, before, snapshot(t, dir))
}

func TestApplyRemoveDeletesEmptiedDirectoriesAndRollsBack(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, map[string]entry{
		"docs/solomonar/skills/go/SKILL.md":        {content: "go\n", mode: 0o644},
		"docs/solomonar/extensions/money/SKILL.md": {content: "money\n", mode: 0o644},
		".claude/skills/go/SKILL.md":               {content: "stub\n", mode: 0o644},
		"docs/solomonar/rules.md":                  {content: "rules\n", mode: 0o644},
	})
	before := snapshot(t, dir)
	plan := Plan{Changes: []Change{
		{Path: "docs/solomonar/skills/go/SKILL.md", Action: ActionRemove},
		{Path: "docs/solomonar/extensions/money/SKILL.md", Action: ActionRemove},
		{Path: ".claude/skills/go/SKILL.md", Action: ActionRemove},
	}}

	journal, err := Apply(t.Context(), root, plan)

	require.NoError(t, err)
	assert.Equal(t, map[string]entry{
		".claude":                   {mode: 0o755, dir: true},
		"docs":                      {mode: 0o755, dir: true},
		"docs/solomonar":            {mode: 0o755, dir: true},
		"docs/solomonar/extensions": {mode: 0o755, dir: true},
		"docs/solomonar/rules.md":   {content: "rules\n", mode: 0o644},
	}, snapshot(t, dir))
	require.NoError(t, journal.Rollback())
	assert.Equal(t, before, snapshot(t, dir))
}
