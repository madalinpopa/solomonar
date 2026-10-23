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
	t.Parallel()

	_, _, err := newStore(t).Read(t.Context())

	rejection := rejectionOf(t, err)
	assert.Equal(t, "not_initialized", rejection.Code)
	assert.Equal(t, "solomonar init", rejection.Fix)
}

func TestCreateWritesEmptyRegistryAsRootCommit(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	store := createdStore(t)
	before := tip(t, store)

	_, err := store.Create(t.Context())

	require.Error(t, err)
	assert.Equal(t, before, tip(t, store))
}

func TestUpdateWritesOneCommitOnTopOfTip(t *testing.T) {
	t.Parallel()

	store := createdStore(t)
	before := tip(t, store)

	commit, err := store.Update(t.Context(), addTask("Login"))
	require.NoError(t, err)

	dir := store.Git.Dir
	assert.Equal(t, commit.SHA, tip(t, store))
	assert.Equal(t, before, gittest.Git(t, dir, "rev-parse", Ref+"^"))
	assert.Equal(t, "state: plan W-001", gittest.Git(t, dir, "log", "-1", "--format=%s", Ref))
	assert.Equal(t, "state.json\ntasks/W-001.json", gittest.Git(t, dir, "ls-tree", "-r", "--name-only", Ref))
	assert.Equal(t, "{\n  \"schema_version\": 2\n}", gittest.Git(t, dir, "cat-file", "blob", Ref+":state.json"))
	encoded, err := plannedTask("Login").Encode()
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(string(encoded)), gittest.Git(t, dir, "cat-file", "blob", Ref+":tasks/W-001.json"))
}

func TestUpdateKeepsOtherTaskFiles(t *testing.T) {
	t.Parallel()

	store := createdStore(t)
	for _, title := range []string{"Login", "Export", "Import"} {
		_, err := store.Update(t.Context(), addTask(title))
		require.NoError(t, err)
	}
	dir := store.Git.Dir
	before := gittest.Git(t, dir, "rev-parse", Ref+":tasks/W-001.json", Ref+":tasks/W-003.json")

	_, err := store.Update(t.Context(), func(reg *Registry) (string, error) {
		reg.Tasks["W-002"].Title = "Export as CSV"
		return "state: retitle W-002", nil
	})
	require.NoError(t, err)

	assert.Equal(t, before, gittest.Git(t, dir, "rev-parse", Ref+":tasks/W-001.json", Ref+":tasks/W-003.json"))
	assert.Equal(t, "tasks/W-002.json", gittest.Git(t, dir, "diff", "--name-only", Ref+"^", Ref))
	reg, _, err := store.Read(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"W-001", "W-002", "W-003"}, reg.IDs())
	assert.Equal(t, "Export as CSV", reg.Tasks["W-002"].Title)
}

func TestReadRejectsUnexpectedTreeEntry(t *testing.T) {
	t.Parallel()

	store := createdStore(t)
	dir := store.Git.Dir
	header := gittest.Git(t, dir, "rev-parse", Ref+":state.json")
	tree := gittestInput(t, dir, "100644 blob "+header+"\tstate.json\n100644 blob "+header+"\tnotes.json\n", "mktree")
	commit := gittest.Git(t, dir, "commit-tree", tree, "-p", Ref, "-m", "state: hand edit")
	gittest.Git(t, dir, "update-ref", Ref, commit)

	_, _, err := store.Read(t.Context())

	rejection := rejectionOf(t, err)
	assert.Equal(t, "registry_invalid", rejection.Code)
	assert.Contains(t, rejection.Found, "unexpected file notes.json")
}

const legacyRegistry = `{"schema_version": 1, "tasks": {"W-001": ` + activeRecord + `, "W-002": ` + doneRecord + `}}`

// legacyStore holds a schema-1 registry, a single state.json written by an older CLI.
func legacyStore(t *testing.T) *Store {
	t.Helper()
	store := newStore(t)
	dir := store.Git.Dir
	blob := gittestInput(t, dir, legacyRegistry, "hash-object", "-w", "--stdin")
	tree := gittestInput(t, dir, "100644 blob "+blob+"\tstate.json\n", "mktree")
	gittest.Git(t, dir, "update-ref", Ref, gittest.Git(t, dir, "commit-tree", tree, "-m", "state: initialize registry"))
	return store
}

func gittestInput(t *testing.T, dir, input string, args ...string) string {
	t.Helper()
	out, err := (&git.Repo{Dir: dir}).RunInput(t.Context(), strings.NewReader(input), args...)
	require.NoError(t, err)
	return out
}

func TestReadRejectsLegacySchema(t *testing.T) {
	t.Parallel()

	_, _, err := legacyStore(t).Read(t.Context())

	rejection := rejectionOf(t, err)
	assert.Equal(t, "registry_outdated", rejection.Code)
	assert.Equal(t, "solomonar task migrate", rejection.Fix)
}

func TestMigrateRewritesLegacyRegistryOnTopOfHistory(t *testing.T) {
	t.Parallel()

	store := legacyStore(t)
	before := tip(t, store)
	legacy, err := DecodeLegacy([]byte(legacyRegistry))
	require.NoError(t, err)

	reg, commit, migrated, err := store.Migrate(t.Context())

	require.NoError(t, err)
	assert.True(t, migrated)
	dir := store.Git.Dir
	assert.Equal(t, commit.SHA, tip(t, store))
	assert.Equal(t, before, gittest.Git(t, dir, "rev-parse", Ref+"^"))
	assert.Equal(t, "state: migrate schema 1 -> 2", gittest.Git(t, dir, "log", "-1", "--format=%s", Ref))
	assert.Equal(t, legacy.Tasks, reg.Tasks)
	read, _, err := store.Read(t.Context())
	require.NoError(t, err)
	assert.Equal(t, reg, read)
	assert.Empty(t, read.Tasks["W-001"].Base)
	assert.False(t, read.Tasks["W-001"].Confirmed)
}

func TestMigrateCurrentSchemaWritesNothing(t *testing.T) {
	t.Parallel()

	store := createdStore(t)
	before := tip(t, store)

	reg, commit, migrated, err := store.Migrate(t.Context())

	require.NoError(t, err)
	assert.False(t, migrated)
	assert.Equal(t, before, commit.SHA)
	assert.Equal(t, NewRegistry(), reg)
	assert.Equal(t, before, tip(t, store))
}

func TestMigrateRejectsNewerSchemaAndInvalidLegacy(t *testing.T) {
	t.Parallel()

	tests := []struct{ name, data, code string }{
		{name: "newer", data: `{"schema_version": 3}`, code: "cli_outdated"},
		{name: "invalid legacy", data: `{"schema_version": 1, "tasks": {"W-1": {}}}`, code: "registry_invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newStore(t)
			dir := store.Git.Dir
			blob := gittestInput(t, dir, tt.data, "hash-object", "-w", "--stdin")
			tree := gittestInput(t, dir, "100644 blob "+blob+"\tstate.json\n", "mktree")
			gittest.Git(t, dir, "update-ref", Ref, gittest.Git(t, dir, "commit-tree", tree, "-m", "state"))
			before := tip(t, store)

			_, _, _, err := store.Migrate(t.Context())

			assert.Equal(t, tt.code, rejectionOf(t, err).Code)
			assert.Equal(t, before, tip(t, store))
		})
	}
}

func TestUpdateRejectionWritesNothing(t *testing.T) {
	t.Parallel()

	store := createdStore(t)
	before := tip(t, store)
	rejected := &Rejection{Code: "task_not_found"}

	_, err := store.Update(t.Context(), func(*Registry) (string, error) { return "", rejected })

	require.ErrorIs(t, err, rejected)
	assert.Equal(t, before, tip(t, store))
}

func TestUpdateWithEmptySubjectWritesNothing(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	store := createdStore(t)
	store.Remote = "origin"

	commit, err := store.Update(t.Context(), addTask("Login"))
	require.NoError(t, err)

	assert.False(t, commit.Pushed)
	assert.Equal(t, []Warning{{Code: "no_remote", Message: `no remote "origin"; the registry stays local`}},
		commit.Warnings)
}

func TestUpdateWithUnreachableRemoteWarnsAndKeepsCommit(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	store := newStore(t)
	ctx, cancel := context.WithCancel(t.Context())
	store.Git.Trace = cancelBefore{subcommand: "commit-tree", cancel: cancel}

	_, err := store.Create(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, gittest.Git(t, store.Git.Dir, "for-each-ref", Ref))
}

func TestCancelAfterSwapKeepsCommitAndWarnsPushInterrupted(t *testing.T) {
	t.Parallel()

	for _, subcommand := range []string{"update-ref", "push"} {
		t.Run(subcommand, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

	store, _ := storeWithRemote(t)
	ctx, cancel := context.WithDeadline(t.Context(), time.Unix(0, 0))
	defer cancel()

	warning := store.pushRef(ctx)

	require.NotNil(t, warning)
	assert.Equal(t, "push_failed", warning.Code)
	assert.Contains(t, warning.Message, "context deadline exceeded")
}

func TestReadSchema(t *testing.T) {
	t.Parallel()

	_, present, err := newStore(t).ReadSchema(t.Context())
	require.NoError(t, err)
	assert.False(t, present)

	schema, present, err := createdStore(t).ReadSchema(t.Context())
	require.NoError(t, err)
	assert.True(t, present)
	assert.Equal(t, SchemaVersion, schema)
}
