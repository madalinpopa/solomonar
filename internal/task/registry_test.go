package task

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sha = "0123456789abcdef0123456789abcdef01234567"

const plannedRecord = `{"title": "Export expenses as CSV", "status": "planned", "depends_on": ["W-001"],
	"related_task": "W-002", "updated_at": "2026-10-20T14:58:03Z"}`

const activeRecord = `{"title": "Login", "status": "blocked", "stage": "implementation",
	"owner": "orchestrator-1", "branch": "feature/W-001", "spec": "docs/work/W-001-login.md",
	"revision": "` + sha + `", "tier": 2, "active_agents": [{"id": "impl-1", "role": "implementer"}],
	"blockers": [{"id": "B-1", "kind": "decision", "summary": "Session storage", "ref": "W-001/D-1"}],
	"last_blocker": 1, "next_action": {"actor": "user", "action": "Decide W-001/D-1: session storage"},
	"updated_at": "2026-01-15T10:00:00Z"}`

const doneRecord = `{"title": "Login", "status": "done", "stage": "complete",
	"summary": "Password login with sessions", "spec_ref": "` + sha + `:docs/work/W-001-login.md",
	"merge_commit": "` + sha + `", "pr": "https://example.com/pr/1", "updated_at": "2026-01-20T09:00:00Z"}`

func registryWith(id, record string) []byte {
	return []byte(`{"schema_version": 1, "tasks": {"` + id + `": ` + record + `}}`)
}

func rejectionOf(t *testing.T, err error) *Rejection {
	t.Helper()
	rejection, ok := errors.AsType[*Rejection](err)
	require.True(t, ok, "want a *Rejection, got %v", err)
	return rejection
}

func TestDecodeAcceptsValidRecords(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		record string
	}{
		{name: "planned", id: "W-003", record: plannedRecord},
		{name: "active", id: "W-001", record: activeRecord},
		{name: "done", id: "W-001", record: doneRecord},
		{name: "four digit id", id: "W-1000", record: plannedRecord},
		{name: "cleared blockers", id: "W-001", record: replace(activeRecord, `"last_blocker": 1`, `"last_blocker": 3`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, err := Decode(registryWith(tt.id, tt.record))

			require.NoError(t, err)
			assert.Equal(t, 1, registry.SchemaVersion)
			assert.Contains(t, registry.Tasks, tt.id)
		})
	}
}

func TestDecodeKeepsTierZero(t *testing.T) {
	registry, err := Decode(registryWith("W-001", replace(activeRecord, `"tier": 2`, `"tier": 0`)))

	require.NoError(t, err)
	require.NotNil(t, registry.Tasks["W-001"].Tier)
	assert.Equal(t, 0, *registry.Tasks["W-001"].Tier)
}

func TestDecodeRejectsInvalidRegistry(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		found string
	}{
		{name: "not json", data: `{`, found: "decode"},
		{name: "missing schema version", data: `{"tasks": {}}`, found: "schema_version"},
		{name: "schema version not an integer", data: `{"schema_version": "1", "tasks": {}}`, found: "schema_version"},
		{name: "schema version zero", data: `{"schema_version": 0, "tasks": {}}`, found: "schema_version"},
		{name: "missing tasks", data: `{"schema_version": 1}`, found: "tasks"},
		{name: "unknown top-level member", data: `{"schema_version": 1, "tasks": {}, "extra": 1}`, found: "extra"},
		{
			name:  "duplicate key",
			data:  `{"schema_version": 1, "tasks": {}, "tasks": {}}`,
			found: "tasks",
		},
		{name: "unknown record member", data: string(registryWith("W-001", replace(doneRecord, `"pr"`, `"prr"`))), found: "prr"},
		{name: "bad id", data: string(registryWith("W-01", plannedRecord)), found: "W-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode([]byte(tt.data))

			rejection := rejectionOf(t, err)
			assert.Equal(t, "registry_invalid", rejection.Code)
			assert.Contains(t, rejection.Found, tt.found)
			assert.Equal(t, "inspect with git show refs/heads/workflow-state:state.json; never edit it by hand", rejection.Fix)
		})
	}
}

func TestDecodeRejectsInvalidRecord(t *testing.T) {
	tests := []struct {
		name   string
		record string
		found  string
	}{
		{name: "unknown status", record: replace(plannedRecord, `"planned"`, `"waiting"`), found: "status"},
		{name: "unknown stage", record: replace(activeRecord, `"implementation"`, `"coding"`), found: "stage"},
		{name: "unknown blocker kind", record: replace(activeRecord, `"decision"`, `"mood"`), found: "kind"},
		{name: "duplicate blocker id", record: replace(activeRecord, `"blockers": [`, `"blockers": [{"id": "B-1", "kind": "check", "summary": "CI"}, `), found: "B-1"},
		{name: "bad blocker id", record: replace(activeRecord, `"B-1"`, `"B1"`), found: "B1"},
		{name: "blocker above last_blocker", record: replace(activeRecord, `"B-1"`, `"B-2"`), found: "last_blocker"},
		{name: "blockers without last_blocker", record: replace(activeRecord, `"last_blocker": 1,`, ""), found: "last_blocker"},
		{name: "negative last_blocker", record: replace(activeRecord, `"last_blocker": 1`, `"last_blocker": -1`), found: "last_blocker"},
		{name: "planned with last_blocker", record: replace(plannedRecord, `"status": "planned",`, `"status": "planned", "last_blocker": 1,`), found: "last_blocker"},
		{name: "done with last_blocker", record: replace(doneRecord, `"status": "done",`, `"status": "done", "last_blocker": 1,`), found: "last_blocker"},
		{name: "tier out of range", record: replace(activeRecord, `"tier": 2`, `"tier": 3`), found: "tier"},
		{name: "short revision", record: replace(activeRecord, sha, "0123456"), found: "revision"},
		{name: "short merge commit", record: replace(doneRecord, `"merge_commit": "`+sha, `"merge_commit": "0123456`), found: "merge_commit"},
		{name: "spec ref without path", record: replace(doneRecord, ":docs/work/W-001-login.md", ":"), found: "spec_ref"},
		{name: "non-UTC timestamp", record: replace(plannedRecord, "14:58:03Z", "14:58:03+02:00"), found: "updated_at"},
		{name: "bad dependency", record: replace(plannedRecord, `["W-001"]`, `["login"]`), found: "depends_on"},
		{name: "bad related task", record: replace(plannedRecord, `"W-002"`, `"W2"`), found: "related_task"},
		{name: "agent without role", record: replace(activeRecord, `"implementer"`, `""`), found: "active_agents"},
		{name: "next action without actor", record: replace(activeRecord, `"actor": "user"`, `"actor": ""`), found: "next_action"},
		{name: "missing title", record: replace(plannedRecord, `"title": "Export expenses as CSV",`, ""), found: "title"},
		{name: "missing updated_at", record: `{"title": "Login", "status": "planned"}`, found: "updated_at"},
		{name: "planned with owner", record: replace(plannedRecord, `"status": "planned",`, `"status": "planned", "owner": "o-1",`), found: "owner"},
		{name: "active without owner", record: replace(activeRecord, `"owner": "orchestrator-1",`, ""), found: "owner"},
		{name: "active with foreign branch", record: replace(activeRecord, "feature/W-001", "feature/W-002"), found: "branch"},
		{name: "active without stage", record: replace(activeRecord, `"stage": "implementation",`, ""), found: "stage"},
		{name: "active at complete", record: replace(activeRecord, `"implementation"`, `"complete"`), found: "stage"},
		{name: "active without next action", record: replace(activeRecord, `"next_action": {"actor": "user", "action": "Decide W-001/D-1: session storage"},`, ""), found: "next_action"},
		{name: "implementation without spec", record: replace(activeRecord, `"spec": "docs/work/W-001-login.md",`, ""), found: "spec"},
		{name: "review without tier", record: replace(replace(activeRecord, `"implementation"`, `"specialized_review"`), `"tier": 2,`, ""), found: "tier"},
		{name: "review without revision", record: replace(replace(activeRecord, `"implementation"`, `"integration"`), `"revision": "`+sha+`",`, ""), found: "revision"},
		{name: "done without summary", record: replace(doneRecord, `"summary": "Password login with sessions",`, ""), found: "summary"},
		{name: "done not at complete", record: replace(doneRecord, `"complete"`, `"integration"`), found: "stage"},
		{name: "done with owner", record: replace(doneRecord, `"status": "done",`, `"status": "done", "owner": "o-1",`), found: "owner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(registryWith("W-001", tt.record))

			rejection := rejectionOf(t, err)
			assert.Equal(t, "registry_invalid", rejection.Code)
			assert.Contains(t, rejection.Found, "W-001")
			assert.Contains(t, rejection.Found, tt.found)
		})
	}
}

func TestDecodeRejectsNewerSchemaBeforeStrictDecoding(t *testing.T) {
	_, err := Decode([]byte(`{"schema_version": 2, "tasks": {}, "added_in_v2": true}`))

	rejection := rejectionOf(t, err)
	assert.Equal(t, "cli_outdated", rejection.Code)
	assert.Equal(t, "registry schema 2, this CLI supports 1", rejection.Found)
	assert.Equal(t, "install a newer solomonar release", rejection.Fix)
}

func TestEncodeEmptyRegistry(t *testing.T) {
	want := "{\n  \"schema_version\": 1,\n  \"tasks\": {}\n}\n"

	data, err := NewRegistry().Encode()

	require.NoError(t, err)
	assert.Equal(t, want, string(data))
}

func TestEncodeIsCanonical(t *testing.T) {
	input := `{"tasks": {"W-002": {"updated_at": "2026-10-20T14:58:03Z", "status": "planned", "title": "B"},
		"W-001": {"title": "A", "status": "planned", "updated_at": "2026-10-20T14:58:03Z"}}, "schema_version": 1}`
	want := `{
  "schema_version": 1,
  "tasks": {
    "W-001": {
      "title": "A",
      "status": "planned",
      "updated_at": "2026-10-20T14:58:03Z"
    },
    "W-002": {
      "title": "B",
      "status": "planned",
      "updated_at": "2026-10-20T14:58:03Z"
    }
  }
}
`
	registry, err := Decode([]byte(input))
	require.NoError(t, err)

	data, err := registry.Encode()

	require.NoError(t, err)
	assert.Equal(t, want, string(data))
}

func TestEncodeRoundTrips(t *testing.T) {
	data := `{"schema_version": 1, "tasks": {"W-001": ` + activeRecord + `, "W-002": ` + doneRecord +
		`, "W-003": ` + plannedRecord + `}}`
	registry, err := Decode([]byte(data))
	require.NoError(t, err)

	encoded, err := registry.Encode()
	require.NoError(t, err)
	decoded, err := Decode(encoded)
	require.NoError(t, err)
	reencoded, err := decoded.Encode()

	require.NoError(t, err)
	assert.Equal(t, registry, decoded)
	assert.Equal(t, string(encoded), string(reencoded))
}

func replace(s, old, replacement string) string {
	return strings.Replace(s, old, replacement, 1)
}

func TestSchemaOfReadsNewerSchema(t *testing.T) {
	schema, err := SchemaOf([]byte(`{"schema_version": 2, "added_in_v2": true}`))

	require.NoError(t, err)
	assert.Equal(t, 2, schema)
}

func TestSchemaOfRejectsUnreadableSchema(t *testing.T) {
	tests := []struct {
		name, data, found string
	}{
		{name: "not json", data: `[`, found: "decode state.json"},
		{name: "missing", data: `{"tasks": {}}`, found: "schema_version is missing"},
		{name: "zero", data: `{"schema_version": 0}`, found: "schema_version 0 is not a positive integer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SchemaOf([]byte(tt.data))

			rejection := rejectionOf(t, err)
			assert.Equal(t, "registry_invalid", rejection.Code)
			assert.Contains(t, rejection.Found, tt.found)
		})
	}
}
