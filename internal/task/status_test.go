package task

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

func gitFacts(t *testing.T, repo *git.Repo, r *Record) StatusFacts {
	t.Helper()
	facts, err := GitFacts(t.Context(), repo, "W-001", "main", r)
	require.NoError(t, err)
	return facts
}

func TestGitFactsJSONOfWorktreeAheadAndDirty(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	r := withSpec(moving(StatusRunning, StageSpecializedReview, new(2)))
	r.Revision = f.onBranch
	gittest.Commit(t, f.worktree, "test: session")
	worktreeHead := head(t, f.worktree)
	require.NoError(t, os.WriteFile(filepath.Join(f.worktree, "notes.txt"), nil, 0o644))

	out, err := json.Marshal(gitFacts(t, f.repo, r))

	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{
		"branch": {"name": "feature/W-001", "exists": true, "tip": %[1]q, "contains_revision": true},
		"worktree": {"path": %[2]q, "relative": %[3]q, "head": %[1]q, "ahead_of_revision": 1,
		             "dirty": [{"path": "notes.txt", "state": "untracked"}]},
		"spec": {"in_worktree": true, "at_revision": true},
		"plan": null,
		"merge": {"merged": false, "merge_commit": null, "derived_spec_ref": null},
		"forge": {"available": false, "pr": null, "checks": "unknown"}
	}`, worktreeHead, f.worktree, relativeOf(t, f)), string(out))
}

func relativeOf(t *testing.T, f factsRepo) string {
	t.Helper()
	_, relative := f.workspace(t)
	return relative
}

func TestGitFactsWithoutBranch(t *testing.T) {
	t.Parallel()

	repo := repoWithoutWorktree(t, false)

	facts := gitFacts(t, repo, withSpec(withRevision(moving(StatusRunning, StageImplementation, nil))))

	assert.Equal(t, StatusFacts{
		Branch: BranchFacts{Name: "feature/W-001"},
		Forge:  NoForge(),
	}, facts)
}

func TestGitFactsBranchWithoutWorktreeOrRevision(t *testing.T) {
	t.Parallel()

	repo := repoWithoutWorktree(t, true)
	tip := gittest.Git(t, repo.Dir, "rev-parse", "feature/W-001")

	facts := gitFacts(t, repo, withSpec(moving(StatusRunning, StageSpecification, nil)))

	assert.Equal(t, BranchFacts{Name: "feature/W-001", Exists: true, Tip: &tip}, facts.Branch)
	assert.Nil(t, facts.Worktree)
	assert.Equal(t, SpecFacts{}, facts.Spec)
}

func TestGitFactsContainsRevision(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	tests := []struct {
		name     string
		revision string
		want     bool
	}{
		{name: "on branch", revision: f.onBranch, want: true},
		{name: "only on main", revision: f.onMain, want: false},
		{name: "unknown to git", revision: sha, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := moving(StatusRunning, StageImplementation, nil)
			r.Revision = tt.revision

			facts := gitFacts(t, f.repo, r)

			require.NotNil(t, facts.Branch.ContainsRevision)
			assert.Equal(t, tt.want, *facts.Branch.ContainsRevision)
		})
	}
}

func TestGitFactsWorktreeWithoutRevision(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)

	facts := gitFacts(t, f.repo, withSpec(moving(StatusRunning, StageSpecification, nil)))

	_, relative := f.workspace(t)
	assert.Equal(t, &WorktreeFacts{Path: f.worktree, Relative: relative, Head: f.onBranch, Dirty: []DirtyPath{}}, facts.Worktree)
	assert.Nil(t, facts.Branch.ContainsRevision)
	assert.Equal(t, SpecFacts{InWorktree: true}, facts.Spec)
}

func TestGitFactsWorktreeDirtyPaths(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(f.worktree, filepath.FromSlash(specPath)), []byte("# Changed\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(f.worktree, "added.txt"), nil, 0o644))
	gittest.Git(t, f.worktree, "add", "added.txt")

	facts := gitFacts(t, f.repo, moving(StatusRunning, StageImplementation, nil))

	require.NotNil(t, facts.Worktree)
	assert.Equal(t, []DirtyPath{
		{Path: "added.txt", State: "added"},
		{Path: specPath, State: "modified"},
	}, facts.Worktree.Dirty)
}

func TestParseDirty(t *testing.T) {
	t.Parallel()

	const modes = "N... 100644 100644 100644 45b983be36b73c0788dc9cbcb76cbb80fc7bb057 7fda762c559b374949611c37591f502d137b0fc1"
	tests := []struct {
		name string
		line string
		want []DirtyPath
	}{
		{name: "modified in worktree", line: "1 .M " + modes + " a.txt", want: []DirtyPath{{Path: "a.txt", State: "modified"}}},
		{name: "modified in index", line: "1 M. " + modes + " a.txt", want: []DirtyPath{{Path: "a.txt", State: "modified"}}},
		{name: "added", line: "1 A. " + modes + " a.txt", want: []DirtyPath{{Path: "a.txt", State: "added"}}},
		{name: "worktree side wins", line: "1 AD " + modes + " a.txt", want: []DirtyPath{{Path: "a.txt", State: "deleted"}}},
		{name: "type changed", line: "1 .T " + modes + " a.txt", want: []DirtyPath{{Path: "a.txt", State: "type_changed"}}},
		{name: "path with spaces", line: "1 .M " + modes + " my notes.txt", want: []DirtyPath{{Path: "my notes.txt", State: "modified"}}},
		{name: "renamed", line: "2 R. " + modes + " R100 new.txt\told.txt", want: []DirtyPath{{Path: "new.txt", State: "renamed"}}},
		{name: "renamed then modified", line: "2 RM " + modes + " R100 new.txt\told.txt", want: []DirtyPath{{Path: "new.txt", State: "renamed"}}},
		{name: "copied", line: "2 C. " + modes + " C100 b.txt\ta.txt", want: []DirtyPath{{Path: "b.txt", State: "copied"}}},
		{
			name: "unmerged",
			line: "u UU N... 100644 100644 100644 100644 45b983be36b73c0788dc9cbcb76cbb80fc7bb057 " +
				"45b983be36b73c0788dc9cbcb76cbb80fc7bb057 7fda762c559b374949611c37591f502d137b0fc1 a.txt",
			want: []DirtyPath{{Path: "a.txt", State: "unmerged"}},
		},
		{name: "untracked", line: "? dir/", want: []DirtyPath{{Path: "dir/", State: "untracked"}}},
		{name: "ignored", line: "! build/", want: []DirtyPath{}},
		{name: "clean", line: "", want: []DirtyPath{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, parseDirty(tt.line))
		})
	}
}

func TestGitFactsSpecOnlyInWorktree(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	r := withSpec(moving(StatusRunning, StageImplementation, nil))
	r.Revision = f.onMain

	facts := gitFacts(t, f.repo, r)

	assert.Equal(t, SpecFacts{InWorktree: true}, facts.Spec)
}

func TestGitFactsWithoutRecordedSpec(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	r := moving(StatusRunning, StageImplementation, nil)
	r.Spec, r.Revision = "", f.onBranch

	facts := gitFacts(t, f.repo, r)

	assert.Equal(t, SpecFacts{}, facts.Spec)
	require.NotNil(t, facts.Worktree)
	assert.Equal(t, new(0), facts.Worktree.AheadOfRevision)
}

func TestGitFactsMerge(t *testing.T) {
	t.Parallel()

	d := newDoneRepo(t)
	derived := d.specRef()
	tests := []struct {
		name string
		spec string
		want MergeFacts
	}{
		{name: "with spec", spec: specPath, want: MergeFacts{Merged: true, MergeCommit: &d.merge, DerivedSpecRef: &derived}},
		{name: "without spec", want: MergeFacts{Merged: true, MergeCommit: &d.merge}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := d.record()
			r.Spec = tt.spec

			assert.Equal(t, tt.want, gitFacts(t, d.repo, r).Merge)
		})
	}
}

func TestGitFactsNotMerged(t *testing.T) {
	t.Parallel()

	_, d := newFeatureRepo(t)

	assert.Equal(t, MergeFacts{}, gitFacts(t, d.repo, withSpec(d.record())).Merge)
}

func TestGitFactsMergeWithoutSpecCommit(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	gittest.Commit(t, dir, "initial")
	gittest.Git(t, dir, "checkout", "-b", "feature/W-001")
	gittest.Commit(t, dir, "feat: login")
	revision := head(t, dir)
	gittest.Git(t, dir, "checkout", "main")
	gittest.Git(t, dir, "merge", "--no-ff", "-m", "Merge W-001", "feature/W-001")
	merge := head(t, dir)
	r := withSpec(integrating())
	r.Revision = revision

	assert.Equal(t, MergeFacts{Merged: true, MergeCommit: &merge}, gitFacts(t, repoAt(dir), r).Merge)
}

func TestGitFactsDoneRecordTakesMergeFromRecord(t *testing.T) {
	t.Parallel()

	repo := repoWithoutWorktree(t, false)
	specRef := sha + ":" + specPath
	r := &Record{Title: "Login", Status: StatusDone, Stage: StageComplete, MergeCommit: sha, SpecRef: specRef}

	facts := gitFacts(t, repo, r)

	assert.Equal(t, MergeFacts{Merged: true, MergeCommit: new(sha), DerivedSpecRef: &specRef}, facts.Merge)
}

func TestGitFactsReturnsGitErrors(t *testing.T) {
	t.Parallel()

	repo := &git.Repo{Dir: t.TempDir()}

	_, err := GitFacts(t.Context(), repo, "W-001", "main", moving(StatusRunning, StageImplementation, nil))

	require.ErrorContains(t, err, "check branch feature/W-001")
}

const prViewArgs = "pr view feature/W-001 --json number,url,state,mergeCommit,statusCheckRollup"

func forge(t *testing.T) (ForgeFacts, []Warning) {
	t.Helper()
	facts, warnings, err := Forge(t.Context(), &git.Repo{Dir: t.TempDir()}, "W-001")
	require.NoError(t, err)
	return facts, warnings
}

func unavailable(reason string) []Warning {
	return []Warning{{Code: "forge_unavailable", Message: "gh unavailable: " + reason}}
}

func TestForgeRunsGHPRViewForBranch(t *testing.T) {
	gittest.FakeGH(t, `{"number": 42, "url": "u", "state": "OPEN", "mergeCommit": null, "statusCheckRollup": []}`, "", 0)
	var trace strings.Builder

	_, _, err := Forge(t.Context(), &git.Repo{Dir: t.TempDir(), Trace: &trace}, "W-001")

	require.NoError(t, err)
	assert.Equal(t, "+ gh "+prViewArgs+"\n", trace.String())
}

func TestForgeOutcomes(t *testing.T) {
	const url = "https://github.com/acme/ledger/pull/42"
	tests := []struct {
		name         string
		stdout       string
		stderr       string
		code         int
		want         ForgeFacts
		wantWarnings []Warning
	}{
		{
			name:   "open PR",
			stdout: `{"number": 42, "url": "` + url + `", "state": "OPEN", "mergeCommit": null, "statusCheckRollup": []}`,
			want:   ForgeFacts{Available: true, PR: &PullRequest{Number: 42, URL: url, State: "OPEN"}, Checks: "none"},
		},
		{
			name: "merged PR",
			stdout: `{"number": 42, "url": "` + url + `", "state": "MERGED", "mergeCommit": {"oid": "` + sha +
				`"}, "statusCheckRollup": []}`,
			want: ForgeFacts{Available: true, PR: &PullRequest{Number: 42, URL: url, State: "MERGED", MergeCommit: new(sha)}, Checks: "none"},
		},
		{
			name:   "no PR",
			stderr: "no pull requests found for branch \"feature/W-001\"\n",
			code:   1,
			want:   ForgeFacts{Available: true, Checks: "unknown"},
		},
		{
			name:         "gh fails",
			stderr:       "error connecting to api.github.com\ncheck your internet connection\n",
			code:         1,
			want:         NoForge(),
			wantWarnings: unavailable("error connecting to api.github.com"),
		},
		{name: "gh fails silently", code: 4, want: NoForge(), wantWarnings: unavailable("exit status 4")},
		{name: "malformed output", stdout: "{", want: NoForge(), wantWarnings: unavailable("unreadable gh pr view output")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gittest.FakeGH(t, tt.stdout, tt.stderr, tt.code)

			facts, warnings := forge(t)

			assert.Equal(t, tt.want, facts)
			assert.Equal(t, tt.wantWarnings, warnings)
		})
	}
}

func TestForgeWithoutGHOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	facts, warnings := forge(t)

	assert.Equal(t, NoForge(), facts)
	assert.Equal(t, unavailable("gh not found"), warnings)
}

func TestForgeWithCancelledContextFails(t *testing.T) {
	gittest.FakeGH(t, "{}", "", 0)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := Forge(ctx, &git.Repo{Dir: t.TempDir()}, "W-001")

	require.ErrorIs(t, err, context.Canceled)
}

func TestForgeChecks(t *testing.T) {
	const (
		success   = `{"status": "COMPLETED", "conclusion": "SUCCESS"}`
		failure   = `{"status": "COMPLETED", "conclusion": "FAILURE"}`
		running   = `{"status": "IN_PROGRESS", "conclusion": ""}`
		statusOK  = `{"state": "SUCCESS"}`
		statusErr = `{"state": "ERROR"}`
	)
	tests := []struct {
		name   string
		rollup string
		want   string
	}{
		{name: "empty", rollup: ``, want: "none"},
		{name: "all succeeded", rollup: success + `,` + statusOK, want: "passing"},
		{name: "neutral and skipped pass", rollup: `{"status": "COMPLETED", "conclusion": "NEUTRAL"},{"status": "COMPLETED", "conclusion": "SKIPPED"}`, want: "passing"},
		{name: "failure", rollup: success + `,` + failure, want: "failing"},
		{name: "cancelled", rollup: `{"status": "COMPLETED", "conclusion": "CANCELLED"}`, want: "failing"},
		{name: "timed out", rollup: `{"status": "COMPLETED", "conclusion": "TIMED_OUT"}`, want: "failing"},
		{name: "action required", rollup: `{"status": "COMPLETED", "conclusion": "ACTION_REQUIRED"}`, want: "failing"},
		{name: "startup failure", rollup: `{"status": "COMPLETED", "conclusion": "STARTUP_FAILURE"}`, want: "failing"},
		{name: "status error", rollup: statusErr, want: "failing"},
		{name: "status failure", rollup: `{"state": "FAILURE"}`, want: "failing"},
		{name: "failing wins over pending", rollup: running + `,` + statusErr, want: "failing"},
		{name: "check run in progress", rollup: success + `,` + running, want: "pending"},
		{name: "status pending", rollup: `{"state": "PENDING"}`, want: "pending"},
		{name: "status expected", rollup: `{"state": "EXPECTED"}`, want: "pending"},
		{name: "unrecognized conclusion", rollup: success + `,{"status": "COMPLETED", "conclusion": "STALE"}`, want: "unknown"},
		{name: "unrecognized state", rollup: `{"state": "WHATEVER"}`, want: "unknown"},
		{name: "unrecognized entry", rollup: `{}`, want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gittest.FakeGH(t, `{"number": 1, "url": "u", "state": "OPEN", "mergeCommit": null, "statusCheckRollup": [`+tt.rollup+`]}`, "", 0)

			facts, warnings := forge(t)

			assert.Equal(t, tt.want, facts.Checks)
			assert.Empty(t, warnings)
		})
	}
}

func TestGitFactsReconcileThePlan(t *testing.T) {
	t.Parallel()

	f := newFactsRepo(t)
	s1 := f.checkpoint(t, "feat(W-001): add the session store", "W-001/S-1")

	facts := gitFacts(t, f.repo, f.implementing(t))

	require.NotNil(t, facts.Plan)
	assert.Equal(t, PlanApproved, facts.Plan.Approval)
	assert.Equal(t, s1, facts.Plan.At)
	assert.Equal(t, []string{"S-2"}, facts.Plan.Remaining())
}
