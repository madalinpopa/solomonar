package project

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(name)
			tree[filepath.ToSlash(rel)] = entry{content: "-> " + target, mode: info.Mode()}
			return err
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

// uninstallTree is an installation whose removal empties the framework directories.
func uninstallTree() map[string]entry {
	return map[string]entry{
		"AGENTS.md":                             {content: "# Framework\n## Project notes\nmine\n", mode: 0o644},
		"docs/README.md":                        {content: "## Documents\n\n- line\n", mode: 0o600},
		"docs/decisions.md":                     {content: "decisions\n", mode: 0o644},
		"docs/solomonar/rules.md":               {content: "rules\n", mode: 0o644},
		"docs/solomonar/extensions/go/SKILL.md": {content: "go\n", mode: 0o644},
		".claude/skills/go/SKILL.md":            {content: "stub\n", mode: 0o644},
		ManifestPath:                            {content: "{}\n", mode: 0o644},
	}
}

func uninstallPlan() Plan {
	return Plan{
		Changes: []Change{
			{Path: ".claude/skills/go/SKILL.md", Action: ActionRemove},
			{Path: "AGENTS.md", Action: ActionUpdate, Content: []byte("## Project notes\nmine\n")},
			{Path: "docs/README.md", Action: ActionUpdate, Content: []byte("## Documents\n\n")},
			{Path: "docs/solomonar/extensions/go/SKILL.md", Action: ActionRemove},
			{Path: "docs/solomonar/rules.md", Action: ActionRemove},
			{Path: ManifestPath, Action: ActionRemove},
		},
		Prune: []string{"docs/solomonar/extensions", "docs/solomonar", ".solomonar"},
	}
}

func TestApplyPrunesEmptyDirectoriesAndRollsBack(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, uninstallTree())
	require.NoError(t, os.Chmod(filepath.Join(dir, "docs/solomonar"), 0o700))
	before := snapshot(t, dir)

	journal, err := Apply(t.Context(), root, uninstallPlan())

	require.NoError(t, err)
	assert.Equal(t, map[string]entry{
		".claude":           {mode: 0o755, dir: true},
		"AGENTS.md":         {content: "## Project notes\nmine\n", mode: 0o644},
		"docs":              {mode: 0o755, dir: true},
		"docs/README.md":    {content: "## Documents\n\n", mode: 0o600},
		"docs/decisions.md": {content: "decisions\n", mode: 0o644},
	}, snapshot(t, dir))
	require.NoError(t, journal.Rollback())
	assert.Equal(t, before, snapshot(t, dir))
}

func TestApplyPruneKeepsDirectoriesThatAreNotEmpty(t *testing.T) {
	t.Parallel()

	files := uninstallTree()
	files["docs/solomonar/extensions/mine/SKILL.md"] = entry{content: "mine\n", mode: 0o644}
	files[".solomonar/notes.txt"] = entry{content: "x\n", mode: 0o644}
	root, dir := openTree(t, files)
	require.NoError(t, os.Symlink("docs", filepath.Join(dir, "linked")))

	plan := uninstallPlan()
	plan.Prune = append(plan.Prune, "linked")
	_, err := Apply(t.Context(), root, plan)

	require.NoError(t, err)
	got := snapshot(t, dir)
	assert.Equal(t, entry{content: "mine\n", mode: 0o644}, got["docs/solomonar/extensions/mine/SKILL.md"])
	assert.Equal(t, entry{content: "x\n", mode: 0o644}, got[".solomonar/notes.txt"])
	assert.Contains(t, got, "linked")
}

func TestApplyRollsBackEverythingWhenAPruneFails(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}

	root, dir := openTree(t, uninstallTree())
	solomonar := filepath.Join(dir, "docs/solomonar")
	require.NoError(t, os.Chmod(solomonar, 0o500)) // removing extensions/ fails
	t.Cleanup(func() { _ = os.Chmod(solomonar, 0o755) })
	before := snapshot(t, dir)
	plan := uninstallPlan()
	plan.Changes = slices.DeleteFunc(plan.Changes, func(c Change) bool { return c.Path == "docs/solomonar/rules.md" })

	journal, err := Apply(t.Context(), root, plan)

	require.ErrorContains(t, err, "apply docs/solomonar/extensions: ")
	assert.Nil(t, journal)
	assert.Equal(t, before, snapshot(t, dir))
}

func TestApplyStopsBeforeAPruneWhenCancelled(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, map[string]entry{".solomonar/x": {content: "x\n", mode: 0o644}})
	require.NoError(t, os.Remove(filepath.Join(dir, ".solomonar/x")))
	before := snapshot(t, dir)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := Apply(ctx, root, Plan{Prune: []string{".solomonar"}})

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, before, snapshot(t, dir))
}

func TestRollbackNamesThePathsItCouldNotRestore(t *testing.T) {
	t.Parallel()

	root, dir := openTree(t, original())
	journal, err := Apply(t.Context(), root, Plan{Changes: []Change{
		{Path: "docs/old.md", Action: ActionRemove},
		{Path: "bin/run.sh", Action: ActionUpdate, Content: []byte("changed\n")},
		{Path: "README.md", Action: ActionRemove},
	}})
	require.NoError(t, err)
	for _, name := range []string{"docs/old.md", "README.md"} {
		require.NoError(t, os.Mkdir(filepath.Join(dir, name), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name, "x"), nil, 0o644))
	}

	err = journal.Rollback()

	rollback, ok := errors.AsType[*RollbackError](err)
	require.True(t, ok, "want a *RollbackError, got %v", err)
	assert.Equal(t, []string{"docs/old.md", "README.md"}, rollback.Paths)
	require.ErrorContains(t, err, "restore README.md: ")
	assert.Equal(t, original()["bin/run.sh"], snapshot(t, dir)["bin/run.sh"])
	assert.NoError(t, (&Journal{root: root}).Rollback())
}

func TestPlanChangedNamesTheFirstDifferentPath(t *testing.T) {
	t.Parallel()

	plan := Plan{Changes: []Change{
		{Path: "a.md", Action: ActionRemove},
		{Path: "b.md", Action: ActionUpdate, Content: []byte("b\n")},
		{Path: "d.md", Action: ActionSkip, Reason: "kept"},
	}}
	with := func(edit func(changes []Change) []Change) Plan {
		return Plan{Changes: edit(slices.Clone(plan.Changes))}
	}
	tests := []struct {
		name  string
		other Plan
		want  string
	}{
		{name: "same", other: with(func(c []Change) []Change { return c })},
		{name: "action", other: with(func(c []Change) []Change {
			c[0].Action = ActionConflict
			return c
		}), want: "a.md"},
		{name: "content", other: with(func(c []Change) []Change {
			c[1].Content = []byte("B\n")
			return c
		}), want: "b.md"},
		{name: "reason", other: with(func(c []Change) []Change {
			c[2].Reason = "other"
			return c
		}), want: "d.md"},
		{name: "added", other: with(func(c []Change) []Change {
			return slices.Insert(c, 2, Change{Path: "c.md", Action: ActionRemove})
		}), want: "c.md"},
		{name: "dropped", other: with(func(c []Change) []Change { return c[1:] }), want: "a.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			changed, ok := plan.Changed(tt.other)

			assert.Equal(t, tt.want, changed)
			assert.Equal(t, tt.want != "", ok)
		})
	}
}
