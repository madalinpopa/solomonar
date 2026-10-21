package git_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

func TestBranchNamesTheCurrentBranch(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, dir string)
		branch string
	}{
		{name: "unborn", setup: func(*testing.T, string) {}, branch: "main"},
		{name: "committed", setup: func(t *testing.T, dir string) {
			t.Helper()
			gittest.Commit(t, dir, "initial")
			gittest.Git(t, dir, "switch", "-c", "feature/T-1")
		}, branch: "feature/T-1"},
		{name: "detached", setup: func(t *testing.T, dir string) {
			t.Helper()
			gittest.Commit(t, dir, "initial")
			gittest.Git(t, dir, "switch", "--detach")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := gittest.NewRepo(t)
			tt.setup(t, dir)

			branch, err := (&git.Repo{Dir: dir}).Branch(t.Context())

			require.NoError(t, err)
			assert.Equal(t, tt.branch, branch)
		})
	}
}
