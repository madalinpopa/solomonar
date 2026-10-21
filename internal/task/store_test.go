package task

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/git/gittest"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	return &Store{Git: &git.Repo{Dir: gittest.NewRepo(t)}}
}

func createdStore(t *testing.T) *Store {
	t.Helper()
	store := newStore(t)
	_, err := store.Create(t.Context())
	require.NoError(t, err)
	return store
}

func tip(t *testing.T, store *Store) string {
	t.Helper()
	return gittest.Git(t, store.Git.Dir, "rev-parse", Ref)
}

func addTask(title string) func(*Registry) (string, error) {
	return func(reg *Registry) (string, error) {
		id := nextTaskID(reg)
		reg.Tasks[id] = plannedTask(title)
		return "state: plan " + id, nil
	}
}

func plannedTask(title string) *Record {
	return &Record{Title: title, Status: StatusPlanned, UpdatedAt: "2026-01-01T00:00:00Z"}
}

// moveRefBehind commits a competing registry change so the caller's swap is lost.
func moveRefBehind(t *testing.T, store *Store) {
	t.Helper()
	competitor := &Store{Git: store.Git}
	_, err := competitor.Update(t.Context(), addTask("competing"))
	require.NoError(t, err)
}

func storeWithRemote(t *testing.T) (*Store, string) {
	t.Helper()
	store := createdStore(t)
	remote := gittest.NewBareRemote(t)
	gittest.Git(t, store.Git.Dir, "remote", "add", "origin", remote)
	store.Remote = "origin"
	return store, remote
}

// cancelBefore cancels when the store is about to run the named git subcommand.
type cancelBefore struct {
	subcommand string
	cancel     context.CancelFunc
}

func (c cancelBefore) Write(p []byte) (int, error) {
	if strings.HasPrefix(string(p), "+ git "+c.subcommand+" ") {
		c.cancel()
	}
	return len(p), nil
}

func TestReadWithoutRefIsNotInitialized(t *testing.T) {
	_, _, err := newStore(t).Read(t.Context())

	rejection := rejectionOf(t, err)
	assert.Equal(t, "not_initialized", rejection.Code)
	assert.Equal(t, "solomonar init", rejection.Fix)
}

func TestCreateWritesEmptyRegistryAsRootCommit(t *testing.T) {
	store := newStore(t)

	commit, err := store.Create(t.Context())
	require.NoError(t, err)

	reg, readTip, err := store.Read(t.Context())
	require.NoError(t, err)
	assert.Equal(t, NewRegistry(), reg)
	assert.Equal(t, commit.SHA, readTip)
	dir := store.Git.Dir
	assert.Equal(t, commit.SHA, gittest.Git(t, dir, "rev-list", "--max-parents=0", Ref))
	assert.Equal(t, "state: initialize registry", gittest.Git(t, dir, "log", "-1", "--format=%s", Ref))
	assert.Equal(t, "state.json", gittest.Git(t, dir, "ls-tree", "--name-only", Ref))
}

func TestCreateTwiceFailsAndKeepsRef(t *testing.T) {
	store := createdStore(t)
	before := tip(t, store)

	_, err := store.Create(t.Context())

	require.Error(t, err)
	assert.Equal(t, before, tip(t, store))
}

func TestUpdateWritesOneCommitOnTopOfTip(t *testing.T) {
	store := createdStore(t)
	before := tip(t, store)

	commit, err := store.Update(t.Context(), addTask("Login"))
	require.NoError(t, err)

	dir := store.Git.Dir
	assert.Equal(t, commit.SHA, tip(t, store))
	assert.Equal(t, before, gittest.Git(t, dir, "rev-parse", Ref+"^"))
	assert.Equal(t, "state: plan W-001", gittest.Git(t, dir, "log", "-1", "--format=%s", Ref))
	want := NewRegistry()
	want.Tasks["W-001"] = plannedTask("Login")
	encoded, err := want.Encode()
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(string(encoded)), gittest.Git(t, dir, "cat-file", "blob", Ref+":state.json"))
}

func TestUpdateRejectionWritesNothing(t *testing.T) {
	store := createdStore(t)
	before := tip(t, store)
	rejected := &Rejection{Code: "task_not_found"}

	_, err := store.Update(t.Context(), func(*Registry) (string, error) { return "", rejected })

	require.ErrorIs(t, err, rejected)
	assert.Equal(t, before, tip(t, store))
}

func TestUpdateWithEmptySubjectWritesNothing(t *testing.T) {
	store, remote := storeWithRemote(t)
	before := tip(t, store)

	commit, err := store.Update(t.Context(), func(*Registry) (string, error) { return "", nil })
	require.NoError(t, err)

	assert.Equal(t, Commit{SHA: before}, commit)
	assert.Equal(t, before, tip(t, store))
	assert.Equal(t, "1", gittest.Git(t, store.Git.Dir, "rev-list", "--count", Ref))
	assert.Empty(t, gittest.Git(t, remote, "for-each-ref", Ref))
}

func TestConcurrentUpdatesGetDistinctIDs(t *testing.T) {
	store := createdStore(t)
	var wg sync.WaitGroup
	for _, title := range []string{"first", "second"} {
		wg.Go(func() {
			_, err := (&Store{Git: store.Git}).Update(t.Context(), addTask(title))
			assert.NoError(t, err)
		})
	}
	wg.Wait()

	reg, _, err := store.Read(t.Context())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"W-001", "W-002"}, slices.Collect(maps.Keys(reg.Tasks)))
}

func TestLostSwapRerunsValidationOnNewTip(t *testing.T) {
	store := createdStore(t)
	calls := 0
	claimW001 := func(reg *Registry) (string, error) {
		calls++
		if calls == 1 {
			moveRefBehind(t, store)
		}
		if _, taken := reg.Tasks["W-001"]; taken {
			return "", &Rejection{Code: "task_exists"}
		}
		reg.Tasks["W-001"] = plannedTask("mine")
		return "state: plan W-001", nil
	}

	_, err := store.Update(t.Context(), claimW001)

	assert.Equal(t, "task_exists", rejectionOf(t, err).Code)
	assert.Equal(t, 2, calls)
}

func TestUpdateGivesUpAfterFiveLostSwaps(t *testing.T) {
	store := createdStore(t)
	calls := 0

	_, err := store.Update(t.Context(), func(reg *Registry) (string, error) {
		calls++
		moveRefBehind(t, store)
		return addTask("mine")(reg)
	})

	assert.Equal(t, "cas_exhausted", rejectionOf(t, err).Code)
	assert.Equal(t, 5, calls)
}

func TestUpdatePushesToRemote(t *testing.T) {
	store := createdStore(t)
	remote := gittest.NewBareRemote(t)
	gittest.Git(t, store.Git.Dir, "remote", "add", "origin", remote)
	store.Remote = "origin"

	commit, err := store.Update(t.Context(), addTask("Login"))
	require.NoError(t, err)

	assert.True(t, commit.Pushed)
	assert.Empty(t, commit.Warnings)
	assert.Equal(t, commit.SHA, gittest.Git(t, remote, "rev-parse", Ref))
}

func TestUpdateWithUnconfiguredRemoteWarnsNoRemote(t *testing.T) {
	store := createdStore(t)
	store.Remote = "origin"

	commit, err := store.Update(t.Context(), addTask("Login"))
	require.NoError(t, err)

	assert.False(t, commit.Pushed)
	assert.Equal(t, []Warning{{Code: "no_remote", Message: `no remote "origin"; the registry stays local`}},
		commit.Warnings)
}

func TestUpdateWithUnreachableRemoteWarnsAndKeepsCommit(t *testing.T) {
	store := createdStore(t)
	gittest.Git(t, store.Git.Dir, "remote", "add", "origin", store.Git.Dir+"/missing")
	store.Remote = "origin"

	commit, err := store.Update(t.Context(), addTask("Login"))
	require.NoError(t, err)

	assert.False(t, commit.Pushed)
	require.Len(t, commit.Warnings, 1)
	assert.Equal(t, "push_failed", commit.Warnings[0].Code)
	assert.True(t, strings.HasPrefix(commit.Warnings[0].Message, "git push origin workflow-state failed: "))
	assert.Equal(t, commit.SHA, tip(t, store))
}

func TestUpdateCancelledBeforeSwapWritesNothing(t *testing.T) {
	store := createdStore(t)
	before := tip(t, store)
	ctx, cancel := context.WithCancel(t.Context())

	_, err := store.Update(ctx, func(reg *Registry) (string, error) {
		cancel()
		return addTask("Login")(reg)
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, before, tip(t, store))
}

func TestCreateCancelledBeforeSwapWritesNothing(t *testing.T) {
	store := newStore(t)
	ctx, cancel := context.WithCancel(t.Context())
	store.Git.Trace = cancelBefore{subcommand: "commit-tree", cancel: cancel}

	_, err := store.Create(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, gittest.Git(t, store.Git.Dir, "for-each-ref", Ref))
}

func TestCancelAfterSwapKeepsCommitAndWarnsPushInterrupted(t *testing.T) {
	for _, subcommand := range []string{"update-ref", "push"} {
		t.Run(subcommand, func(t *testing.T) {
			store, remote := storeWithRemote(t)
			ctx, cancel := context.WithCancel(t.Context())
			store.Git.Trace = cancelBefore{subcommand: subcommand, cancel: cancel}

			commit, err := store.Update(ctx, addTask("Login"))
			require.NoError(t, err)

			assert.Equal(t, commit.SHA, tip(t, store))
			assert.False(t, commit.Pushed)
			assert.Equal(t, []Warning{{
				Code:    "push_interrupted",
				Message: "push to origin interrupted; the registry commit is kept locally",
			}}, commit.Warnings)
			assert.Empty(t, gittest.Git(t, remote, "for-each-ref", Ref))
		})
	}
}

func TestPushTimeoutWarnsPushFailed(t *testing.T) {
	store, _ := storeWithRemote(t)
	ctx, cancel := context.WithDeadline(t.Context(), time.Unix(0, 0))
	defer cancel()

	warning := store.pushRef(ctx)

	require.NotNil(t, warning)
	assert.Equal(t, "push_failed", warning.Code)
	assert.Contains(t, warning.Message, "context deadline exceeded")
}

func TestReadSchema(t *testing.T) {
	_, present, err := newStore(t).ReadSchema(t.Context())
	require.NoError(t, err)
	assert.False(t, present)

	schema, present, err := createdStore(t).ReadSchema(t.Context())
	require.NoError(t, err)
	assert.True(t, present)
	assert.Equal(t, SchemaVersion, schema)
}
