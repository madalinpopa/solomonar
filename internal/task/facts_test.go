package task

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

// factsRepo is a repository with feature/W-001 checked out in a worktree that holds a committed spec.
type factsRepo struct {
	repo     *git.Repo
	worktree string
	onBranch string
	onMain   string
}

func newFactsRepo(t *testing.T) factsRepo {
	t.Helper()
	dir := gittest.NewRepo(t)
	gittest.Commit(t, dir, "initial")
	worktree := gittest.Worktree(t, dir, "feature/W-001")
	gittest.Commit(t, dir, "main only")
	spec := filepath.Join(worktree, filepath.FromSlash(specPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(spec), 0o755))
	require.NoError(t, os.WriteFile(spec, []byte("# Login\n"), 0o644))
	gittest.Git(t, worktree, "add", ".")
	gittest.Git(t, worktree, "commit", "-m", "spec")
	return factsRepo{
		repo:     repoAt(dir),
		worktree: worktree,
		onBranch: gittest.Git(t, worktree, "rev-parse", "HEAD"),
		onMain:   gittest.Git(t, dir, "rev-parse", "HEAD"),
	}
}

// repoWithoutWorktree returns a repository whose feature/W-001 branch, if any, is not checked out.
func repoWithoutWorktree(t *testing.T, withBranch bool) *git.Repo {
	t.Helper()
	dir := gittest.NewRepo(t)
	gittest.Commit(t, dir, "initial")
	if withBranch {
		gittest.Git(t, dir, "branch", "feature/W-001")
	}
	return repoAt(dir)
}

func repoAt(dir string) *git.Repo {
	return &git.Repo{Dir: dir, CommonDir: filepath.Join(dir, ".git")}
}

func moved(t *testing.T, before *Record, m Move) *Record {
	t.Helper()
	after := *before
	_, err := after.Move(m, installedRoles(), clock())
	require.NoError(t, err)
	return &after
}

func checkFacts(t *testing.T, repo *git.Repo, before *Record, m Move) (string, error) {
	t.Helper()
	return CheckFacts(t.Context(), repo, "W-001", before, moved(t, before, m), m)
}

func requireRejection(t *testing.T, err error, code string) Rejection {
	t.Helper()
	rejection, ok := errors.AsType[*Rejection](err)
	require.True(t, ok, "want rejection %s, got %v", code, err)
	assert.Equal(t, code, rejection.Code)
	return *rejection
}

func TestCheckFactsRevision(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	tests := []struct {
		name     string
		revision string
		code     string
	}{
		{name: "on branch", revision: f.onBranch},
		{name: "on main only", revision: f.onMain, code: "revision_not_on_branch"},
		{name: "unknown to git", revision: otherSHA, code: "revision_not_on_branch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := checkFacts(t, f.repo, moving(StatusRunning, StageImplementation, nil), Move{Revision: tt.revision})

			if tt.code == "" {
				require.NoError(t, err)
				return
			}
			requireRejection(t, err, tt.code)
		})
	}
}

func TestCheckFactsRevisionNotOnBranchTexts(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)

	_, err := checkFacts(t, f.repo, moving(StatusRunning, StageImplementation, nil), Move{Revision: f.onMain})

	assert.Equal(t, &Rejection{
		Code:  "revision_not_on_branch",
		Check: "--revision must be a full SHA reachable from feature/W-001",
		Found: f.onMain + " is not on feature/W-001",
		Fix:   "commit the candidate on feature/W-001 and pass its full SHA: git -C " + f.worktree + " rev-parse HEAD",
	}, err)
}

func TestCheckFactsRevisionWithoutBranchIsBranchMissing(t *testing.T) {
	t.Parallel()

	repo := repoWithoutWorktree(t, false)

	_, err := checkFacts(t, repo, moving(StatusRunning, StageImplementation, nil), Move{Revision: sha})

	assert.Equal(t, &Rejection{
		Code:  "branch_missing",
		Check: "feature/W-001 exists locally",
		Found: "feature/W-001 does not exist",
		Fix:   "create the branch and worktree (rules §4)",
	}, err)
}

func TestCheckFactsSetupToSpecification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		repo func(t *testing.T) *git.Repo
		code string
	}{
		{name: "worktree on branch", repo: func(t *testing.T) *git.Repo { t.Helper(); return newFactsRepo(t).repo }},
		{name: "branch without worktree", repo: func(t *testing.T) *git.Repo { t.Helper(); return repoWithoutWorktree(t, true) }, code: "worktree_missing"},
		{name: "no branch", repo: func(t *testing.T) *git.Repo { t.Helper(); return repoWithoutWorktree(t, false) }, code: "branch_missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := checkFacts(t, tt.repo(t), moving(StatusRunning, StageSetup, nil), Move{Stage: "specification"})

			if tt.code == "" {
				require.NoError(t, err)
				return
			}
			requireRejection(t, err, tt.code)
		})
	}
}

func TestCheckFactsWorktreeMissingTexts(t *testing.T) {
	t.Parallel()

	repo := repoWithoutWorktree(t, true)

	_, err := checkFacts(t, repo, moving(StatusRunning, StageSetup, nil), Move{Stage: "specification"})

	assert.Equal(t, &Rejection{
		Code:  "worktree_missing",
		Check: "a worktree has feature/W-001 checked out",
		Found: "no worktree on feature/W-001",
		Fix:   "git worktree add ../" + filepath.Base(repo.Dir) + "-worktrees/W-001 feature/W-001",
	}, err)
}

func TestCheckFactsSpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		before    *Record
		move      func(f factsRepo) Move
		deleted   bool
		found     func(f factsRepo) string
		code      string
		foundText string
	}{
		{
			name:   "in worktree",
			before: moving(StatusRunning, StageSpecification, nil),
			move:   func(factsRepo) Move { return Move{Spec: specPath} },
			found:  func(f factsRepo) string { return f.worktree },
		},
		{
			name:   "recorded spec entering implementation",
			before: withSpec(moving(StatusRunning, StageSpecification, nil)),
			move:   func(factsRepo) Move { return Move{Stage: "implementation"} },
			found:  func(f factsRepo) string { return f.worktree },
		},
		{
			name:    "only at recorded revision",
			before:  moving(StatusRunning, StageImplementation, nil),
			move:    func(f factsRepo) Move { return Move{Revision: f.onBranch, Spec: specPath} },
			deleted: true,
			found:   func(f factsRepo) string { return f.onBranch[:7] },
		},
		{
			name:      "missing",
			before:    moving(StatusRunning, StageImplementation, nil),
			move:      func(f factsRepo) Move { return Move{Revision: f.onBranch, Spec: "docs/work/missing.md"} },
			code:      "spec_not_found",
			foundText: "docs/work/missing.md is not in the feature/W-001 worktree or at the recorded revision",
		},
		{
			name:      "directory",
			before:    moving(StatusRunning, StageImplementation, nil),
			move:      func(f factsRepo) Move { return Move{Revision: f.onBranch, Spec: "docs/work"} },
			code:      "spec_not_found",
			foundText: "docs/work is not in the feature/W-001 worktree or at the recorded revision",
		},
		{
			name:      "missing without revision",
			before:    moving(StatusRunning, StageSpecification, nil),
			move:      func(factsRepo) Move { return Move{Spec: "docs/work/missing.md"} },
			code:      "spec_not_found",
			foundText: "docs/work/missing.md is not in the feature/W-001 worktree",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFactsRepo(t)
			if tt.deleted {
				require.NoError(t, os.Remove(filepath.Join(f.worktree, filepath.FromSlash(specPath))))
			}

			found, err := checkFacts(t, f.repo, tt.before, tt.move(f))

			if tt.code == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.found(f), found)
				return
			}
			rejection := requireRejection(t, err, tt.code)
			assert.Equal(t, tt.foundText, rejection.Found)
			assert.Equal(t, "commit the work spec in the worktree", rejection.Fix)
		})
	}
}

func TestCheckFactsWithoutFactFieldsRunsNoGit(t *testing.T) {
	t.Parallel()

	repo := &git.Repo{Dir: filepath.Join(t.TempDir(), "missing")}
	moves := []struct {
		name   string
		before *Record
		move   Move
	}{
		{name: "next action", before: moving(StatusRunning, StageImplementation, nil), move: Move{Next: "user:Review"}},
		{name: "stage past implementation", before: withRevision(moving(StatusRunning, StageImplementation, new(2))), move: Move{Stage: "specialized_review"}},
	}
	for _, tt := range moves {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			found, err := checkFacts(t, repo, tt.before, tt.move)

			require.NoError(t, err)
			assert.Empty(t, found)
		})
	}
}

func withSpec(r *Record) *Record {
	r.Spec = specPath
	return r
}

func withRevision(r *Record) *Record {
	r.Revision = sha
	return r
}
