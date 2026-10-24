package git_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

func TestFilesListsTrackedAndUntrackedButNotIgnoredFiles(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	write := func(name, content string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
	write(".gitignore", "vendor/\n")
	write("go.mod", "module x\n")
	gittest.Git(t, dir, "add", ".gitignore", "go.mod")
	gittest.Commit(t, dir, "initial")
	write("tools/lint/go.mod", "module lint\n")
	write("vendor/dep/go.mod", "module dep\n")
	write("dåta file.txt", "x\n")

	files, err := (&git.Repo{Dir: dir}).Files(t.Context())

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{".gitignore", "go.mod", "tools/lint/go.mod", "dåta file.txt"}, files)
}
