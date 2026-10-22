package task

import (
	"cmp"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

// doneRepo is a repository whose main merged feature/W-001: two spec commits, then the revision.
type doneRepo struct {
	repo                                          *git.Repo
	firstSpec, lastSpec, revision, merge, initial string
}

func (d doneRepo) specRef() string { return d.lastSpec + ":" + specPath }

// record is a task at integration whose revision is the repository's reviewed commit.
func (d doneRepo) record() *Record {
	r := integrating()
	r.Revision = d.revision
	return r
}

func (d doneRepo) flags() Done {
	return Done{Merge: d.merge, SpecRef: d.specRef(), Summary: "Password login"}
}

// newFeatureRepo commits the feature branch and leaves main checked out one commit ahead of the fork.
func newFeatureRepo(t *testing.T) (string, doneRepo) {
	t.Helper()
	dir := gittest.NewRepo(t)
	commitFile(t, dir, "README.md", "initial")
	d := doneRepo{repo: repoAt(dir), initial: head(t, dir)}
	gittest.Git(t, dir, "checkout", "-b", "feature/W-001")
	commitFile(t, dir, specPath, "spec(W-001): draft login")
	d.firstSpec = head(t, dir)
	commitFile(t, dir, specPath, "spec(W-001): settle login")
	d.lastSpec = head(t, dir)
	gittest.Git(t, dir, "commit", "--allow-empty", "-m", "feat: login", "-m", "spec(W-001): named in the body only")
	d.revision = head(t, dir)
	gittest.Git(t, dir, "checkout", "main")
	gittest.Commit(t, dir, "main work")
	return dir, d
}

func newDoneRepo(t *testing.T) doneRepo {
	t.Helper()
	dir, d := newFeatureRepo(t)
	gittest.Git(t, dir, "merge", "--no-ff", "-m", "Merge W-001", "feature/W-001")
	d.merge = head(t, dir)
	return d
}

func commitFile(t *testing.T, dir, name, msg string) {
	t.Helper()
	file := filepath.Join(dir, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
	require.NoError(t, os.WriteFile(file, []byte(msg+"\n"), 0o644))
	gittest.Git(t, dir, "add", ".")
	gittest.Git(t, dir, "commit", "-m", msg)
}

func head(t *testing.T, dir string) string {
	t.Helper()
	return gittest.Git(t, dir, "rev-parse", "HEAD")
}

func integrating() *Record {
	r := withAgent(moving(StatusRunning, StageIntegration, new(1)))
	r.PR, r.RelatedTask, r.SessionRef, r.DependsOn, r.LastBlocker = "42", "W-000", "s-1", []string{"W-000"}, 2
	return r
}

func doneTask(t *testing.T, registry *Registry, repo *git.Repo, id string, d Done) (MoveResult, error) {
	t.Helper()
	return registry.DoneTask(t.Context(), repo, id, "orchestrator-1", "main", d, clock())
}

// requireDoneRejected runs task done on W-001 and checks the rejection and the untouched record.
func requireDoneRejected(t *testing.T, before *Record, repo *git.Repo, d Done, want Rejection) {
	t.Helper()
	registry := registryOf(before)
	unchanged := *before
	_, err := doneTask(t, registry, repo, "W-001", d)
	assert.Equal(t, want, requireRejection(t, err, want.Code))
	assert.Equal(t, &unchanged, registry.Tasks["W-001"])
}

func TestDoneTaskStoresCompactRecord(t *testing.T) {
	t.Parallel()

	d := newDoneRepo(t)
	registry := registryOf(d.record())
	flags := d.flags()
	flags.Summary = "  Password login  "

	result, err := doneTask(t, registry, d.repo, "W-001", flags)
	require.NoError(t, err)

	assert.Equal(t, MoveResult{
		From: Position{Status: StatusRunning, Stage: StageIntegration},
		To:   Position{Status: StatusDone, Stage: StageComplete},
		Changed: []string{
			"status", "stage", "owner", "branch", "spec", "revision", "tier", "active_agents", "next_action",
			"depends_on", "session_ref", "summary", "spec_ref", "merge_commit",
		},
		Subject: "state(W-001): done/complete, merged " + d.merge[:7],
	}, result)
	stored := registry.Tasks["W-001"]
	assert.Equal(t, &Record{
		Title: "Login", Status: StatusDone, Stage: StageComplete, PR: "42", RelatedTask: "W-000",
		Summary: "Password login", SpecRef: d.specRef(), MergeCommit: d.merge, UpdatedAt: "2026-10-20T14:58:03Z",
	}, stored)
	require.NoError(t, validateRecord("W-001", stored))
}

func TestDoneTaskOmitsUnrecordedOptionalFields(t *testing.T) {
	t.Parallel()

	d := newDoneRepo(t)
	before := moving(StatusRunning, StageIntegration, new(1))
	before.Revision = d.revision
	registry := registryOf(before)

	_, err := doneTask(t, registry, d.repo, "W-001", d.flags())
	require.NoError(t, err)

	assert.Empty(t, registry.Tasks["W-001"].PR)
	assert.Empty(t, registry.Tasks["W-001"].RelatedTask)
}

func TestDoneTaskRejectsState(t *testing.T) {
	t.Parallel()

	twoBlockers := moving(StatusRunning, StageIntegration, new(1))
	twoBlockers.Blockers = []Blocker{{ID: "B-1", Kind: BlockerCheck, Summary: "CI"}, {ID: "B-3", Kind: BlockerCheck, Summary: "Lint"}}
	twoBlockers.LastBlocker = 3
	tests := []struct {
		name   string
		before *Record
		want   Rejection
	}{
		{name: "done is final", before: &Record{Title: "Login", Status: StatusDone, Stage: StageComplete}, want: *doneFinal(StatusDone)},
		{name: "owner", before: func() *Record { r := integrating(); r.Owner = "orchestrator-2"; return r }(), want: Rejection{Code: "owner_mismatch"}},
		{name: "blocked at integration", before: withBlocker(moving(StatusBlocked, StageIntegration, new(1))), want: Rejection{
			Code: "transition_not_allowed", Check: "task done needs running/integration",
			Found: "W-001 is blocked/integration", Fix: "solomonar task status W-001",
		}},
		{name: "running before integration", before: moving(StatusRunning, StageAdversarialReview, new(1)), want: Rejection{
			Code: "transition_not_allowed", Check: "task done needs running/integration",
			Found: "W-001 is running/adversarial_review", Fix: "solomonar task status W-001",
		}},
		{name: "open blockers", before: twoBlockers, want: Rejection{
			Code: "blockers_open", Check: "task done needs no open blocker",
			Found: "W-001 has open blockers B-1, B-3", Fix: "solomonar task unblock W-001 <blocker-id>",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := registryOf(tt.before)
			unchanged := *tt.before
			_, err := doneTask(t, registry, nil, "W-001", Done{})
			rejection := requireRejection(t, err, tt.want.Code)
			if tt.want.Check != "" {
				assert.Equal(t, tt.want, rejection)
			}
			assert.Equal(t, &unchanged, registry.Tasks["W-001"])
		})
	}
}

func TestDoneTaskRejectsUnknownTask(t *testing.T) {
	t.Parallel()

	_, err := doneTask(t, NewRegistry(), nil, "W-001", Done{})
	requireRejection(t, err, "task_not_found")
}

func TestDoneTaskRejectsFlagValues(t *testing.T) {
	t.Parallel()

	valid := Done{Merge: sha, SpecRef: sha + ":" + specPath, Summary: "Password login"}
	summaryCheck, summaryFix := "--summary takes 1 to 120 characters", `pass --summary "<one sentence>"`
	specRefCheck, specRefFix := "--spec-ref takes <full SHA>:<path>", `copy it from the PR "Final specification" line`
	tests := []struct {
		name string
		edit func(*Done)
		want Rejection
	}{
		{name: "blank summary", edit: func(d *Done) { d.Summary = "   " }, want: Rejection{Check: summaryCheck, Found: "   ", Fix: summaryFix}},
		{name: "long summary", edit: func(d *Done) { d.Summary = strings.Repeat("é", 121) }, want: Rejection{Check: summaryCheck, Found: strings.Repeat("é", 121), Fix: summaryFix}},
		{name: "short merge", edit: func(d *Done) { d.Merge = "14091a9" }, want: Rejection{Check: "--merge takes a full commit SHA", Found: "14091a9", Fix: "git rev-parse 14091a9"}},
		{name: "spec ref without path", edit: func(d *Done) { d.SpecRef = sha }, want: Rejection{Check: specRefCheck, Found: sha, Fix: specRefFix}},
		{name: "spec ref with short SHA", edit: func(d *Done) { d.SpecRef = "10ec7f3:" + specPath }, want: Rejection{Check: specRefCheck, Found: "10ec7f3:" + specPath, Fix: specRefFix}},
		{name: "spec ref leaving the repository", edit: func(d *Done) { d.SpecRef = sha + ":../spec.md" }, want: Rejection{Check: specRefCheck, Found: sha + ":../spec.md", Fix: specRefFix}},
		{name: "spec ref with unclean path", edit: func(d *Done) { d.SpecRef = sha + ":docs/./spec.md" }, want: Rejection{Check: specRefCheck, Found: sha + ":docs/./spec.md", Fix: specRefFix}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := valid
			tt.edit(&d)
			tt.want.Code = "invalid_value"
			requireDoneRejected(t, integrating(), nil, d, tt.want)
		})
	}
}

func TestDoneTaskRejectsGitFacts(t *testing.T) {
	t.Parallel()

	d := newDoneRepo(t)
	dir := d.repo.Dir
	gittest.Git(t, dir, "checkout", "-b", "side", d.initial)
	gittest.Git(t, dir, "merge", "--no-ff", "-m", "Merge W-001 into side", "feature/W-001")
	sideMerge := head(t, dir)
	unmerged := d.record()
	unmerged.Revision = sideMerge
	moved := d.record()
	moved.Spec = "docs/work/W-001-other.md"
	m8 := d.merge[:8]
	derived := d.lastSpec + ":" + specPath
	notFinal := func(found string) Rejection {
		return Rejection{
			Code: "spec_ref_not_final", Check: "the spec_ref is the last spec(W-001) commit merged with the recorded spec",
			Found: found, Fix: "pass --spec-ref " + derived,
		}
	}
	tests := []struct {
		name   string
		before *Record
		edit   func(*Done)
		want   Rejection
	}{
		{name: "merge on a side branch", edit: func(f *Done) { f.Merge = sideMerge }, want: Rejection{
			Code: "merge_not_on_main", Check: "the merge commit is reachable from main",
			Found: sideMerge[:8] + " is not reachable from main", Fix: "fetch and update main, then rerun",
		}},
		{name: "unknown merge", edit: func(f *Done) { f.Merge = otherSHA }, want: Rejection{
			Code: "merge_not_on_main", Check: "the merge commit is reachable from main",
			Found: otherSHA[:8] + " is not reachable from main", Fix: "fetch and update main, then rerun",
		}},
		{name: "revision outside the merge", before: unmerged, want: Rejection{
			Code: "revision_not_merged", Check: "the reviewed revision is in the merge",
			Found: sideMerge[:8] + " is not contained in " + m8, Fix: "check that the right PR merged",
		}},
		{name: "spec ref path missing", edit: func(f *Done) { f.SpecRef = d.lastSpec + ":docs/missing.md" }, want: Rejection{
			Code: "spec_ref_invalid", Check: "the spec_ref resolves",
			Found: d.lastSpec + ":docs/missing.md does not resolve", Fix: `copy it from the PR "Final specification" line`,
		}},
		{name: "spec ref commit unknown", edit: func(f *Done) { f.SpecRef = otherSHA + ":" + specPath }, want: Rejection{
			Code: "spec_ref_invalid", Check: "the spec_ref resolves",
			Found: otherSHA + ":" + specPath + " does not resolve", Fix: `copy it from the PR "Final specification" line`,
		}},
		{
			name: "earlier spec commit", edit: func(f *Done) { f.SpecRef = d.firstSpec + ":" + specPath },
			want: notFinal(d.firstSpec[:8] + ":" + specPath + " is not the last spec(W-001) commit merged"),
		},
		{
			name: "other path than the recorded spec", edit: func(f *Done) { f.SpecRef = d.lastSpec + ":README.md" },
			want: notFinal(d.lastSpec[:8] + ":README.md is not the last spec(W-001) commit merged"),
		},
		{name: "recorded spec moved", before: moved, want: Rejection{
			Code: "spec_ref_not_final", Check: "the spec_ref is the last spec(W-001) commit merged with the recorded spec",
			Found: d.lastSpec[:8] + ":" + specPath + " is not the last spec(W-001) commit merged",
			Fix:   "pass --spec-ref " + d.lastSpec + ":docs/work/W-001-other.md",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			flags := d.flags()
			if tt.edit != nil {
				tt.edit(&flags)
			}
			requireDoneRejected(t, cmp.Or(tt.before, d.record()), d.repo, flags, tt.want)
		})
	}
}

func TestDoneTaskRejectsMergeWithoutSpecCommit(t *testing.T) {
	t.Parallel()

	d := newDoneRepo(t)
	registry := NewRegistry()
	registry.Tasks["W-002"] = d.record()

	_, err := doneTask(t, registry, d.repo, "W-002", d.flags())

	assert.Equal(t, Rejection{
		Code: "spec_ref_not_final", Check: "the spec_ref is the last spec(W-002) commit merged with the recorded spec",
		Found: "no spec(W-002) commit is reachable from " + d.merge[:8] + "^2",
	}, requireRejection(t, err, "spec_ref_not_final"))
	assert.Equal(t, StatusRunning, registry.Tasks["W-002"].Status)
}

func TestDoneTaskRejectsSingleParentIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		integrate func(t *testing.T, dir string)
	}{
		{name: "squash", integrate: func(t *testing.T, dir string) {
			t.Helper()
			gittest.Git(t, dir, "merge", "--squash", "feature/W-001")
			gittest.Git(t, dir, "commit", "-m", "feat: login (#42)")
		}},
		{name: "rebase", integrate: func(t *testing.T, dir string) {
			t.Helper()
			gittest.Git(t, dir, "rebase", "main", "feature/W-001")
			gittest.Git(t, dir, "checkout", "main")
			gittest.Git(t, dir, "merge", "--ff-only", "feature/W-001")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir, d := newFeatureRepo(t)
			tt.integrate(t, dir)
			d.merge = head(t, dir)

			requireDoneRejected(t, d.record(), d.repo, d.flags(), Rejection{
				Code: "merge_not_merge_commit", Check: "the merge is a merge commit",
				Found: d.merge[:8] + " has one parent", Fix: "spec_ref cannot survive; report to the user (rules §11)",
			})
		})
	}
}
