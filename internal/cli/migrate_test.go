package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/madalinpopa/solomonar/internal/git/gittest"
	"github.com/madalinpopa/solomonar/internal/task"
)

const legacyState = `{"schema_version": 1, "tasks": {
	"W-001": {"title": "Login", "status": "running", "stage": "implementation", "owner": "orchestrator-1",
		"branch": "feature/W-001", "spec": "docs/work/W-001-login.md",
		"next_action": {"actor": "orchestrator-1", "action": "Delegate"}, "updated_at": "2026-10-20T15:00:00Z"},
	"W-002": {"title": "Export", "status": "planned", "updated_at": "2026-10-20T15:00:00Z"}}}`

func TestTaskMigrateRewritesLegacyRegistry(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepo(t)
	putState(t, dir, legacyState)
	legacyTip := registryTip(t, dir)

	rejected := runInRepo(t, dir, "task", "show")
	got := runInRepo(t, dir, "task", "migrate")
	again := runInRepo(t, dir, "task", "migrate", "--json")

	assert.Equal(t, 3, rejected.exit)
	assert.Contains(t, rejected.stderr, "registry schema 1, this CLI uses 2")
	assert.Contains(t, rejected.stderr, "fix:   solomonar task migrate")
	require.Equal(t, 0, got.exit, got.stderr)
	tip := registryTip(t, dir)
	assert.Equal(t, "Migrated registry to schema 2: 2 tasks, one tasks/<ID>.json each\n"+
		"  unconfirmed  W-001\n"+
		"  next         inspect each with solomonar task status <ID>, then solomonar task confirm <ID>\n"+
		"  registry @ "+tip+"\n", got.stdout)
	assert.Equal(t, legacyTip, gittest.Git(t, dir, "rev-parse", task.Ref+"^"))
	assert.Equal(t, "state.json\ntasks/W-001.json\ntasks/W-002.json", gittest.Git(t, dir, "ls-tree", "-r", "--name-only", task.Ref))
	require.Equal(t, 0, again.exit, again.stderr)
	assert.JSONEq(t, `{
		"ok": true,
		"command": "task migrate",
		"data": {"migrated": false, "schema": 2, "tasks": 2, "unconfirmed": ["W-001"], "commit": "`+tip+`", "pushed": false},
		"warnings": []
	}`, again.stdout)
	assert.Equal(t, tip, registryTip(t, dir))
}
