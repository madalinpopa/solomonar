package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	framework = "0.1.0"
	emptyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// example is the I §4.7 manifest with real hashes.
const example = `{
  "schema_version": 1,
  "framework_version": "0.1.0",
  "project": {"name": "ledger", "main_branch": "main", "remote": "origin"},
  "runtimes": {
    "claude": {"adversarial_model": "opus"},
    "codex": {}
  },
  "extensions": {
    "go": {"state": "installed", "kind": "skill", "group": "language", "source": "catalog", "decision": "P/D-1", "rendered_with": ["tdd"]},
    "htmx": {"state": "installed", "kind": "skill", "group": "frontend", "source": "catalog", "decision": "P/D-2", "rendered_with": ["templ"]},
    "templ": {"state": "installed", "kind": "skill", "group": "frontend", "source": "catalog", "decision": "P/D-2", "rendered_with": ["htmx"]},
    "tdd": {"state": "installed", "kind": "skill", "group": "testing", "source": "catalog", "decision": "P/D-3", "rendered_with": ["go"]}
  },
  "files": {
    "AGENTS.md": {"owner": "core+notes", "sha256": "` + emptyHash + `"},
    "docs/solomonar/rules.md": {"owner": "core", "sha256": "` + emptyHash + `"},
    "docs/solomonar/skills/templ/SKILL.md": {"owner": "ext:templ", "sha256": "` + emptyHash + `"},
    "docs/decisions.md": {"owner": "decision", "entries": ["P/D-1", "P/D-2", "P/D-3"]},
    ".claude/skills/templ/SKILL.md": {"owner": "runtime:claude", "sha256": "` + emptyHash + `"},
    ".claude/settings.json": {"owner": "runtime:claude", "entry": "hooks.PreToolUse/solomonar-guard"},
    "docs/README.md": {"owner": "seed"}
  }
}`

const canonicalExample = `{
  "schema_version": 1,
  "framework_version": "0.1.0",
  "project": {
    "name": "ledger",
    "main_branch": "main",
    "remote": "origin"
  },
  "runtimes": {
    "claude": {
      "adversarial_model": "opus"
    },
    "codex": {}
  },
  "extensions": {
    "go": {
      "state": "installed",
      "kind": "skill",
      "group": "language",
      "source": "catalog",
      "decision": "P/D-1",
      "rendered_with": [
        "tdd"
      ]
    },
    "htmx": {
      "state": "installed",
      "kind": "skill",
      "group": "frontend",
      "source": "catalog",
      "decision": "P/D-2",
      "rendered_with": [
        "templ"
      ]
    },
    "tdd": {
      "state": "installed",
      "kind": "skill",
      "group": "testing",
      "source": "catalog",
      "decision": "P/D-3",
      "rendered_with": [
        "go"
      ]
    },
    "templ": {
      "state": "installed",
      "kind": "skill",
      "group": "frontend",
      "source": "catalog",
      "decision": "P/D-2",
      "rendered_with": [
        "htmx"
      ]
    }
  },
  "files": {
    ".claude/settings.json": {
      "owner": "runtime:claude",
      "entry": "hooks.PreToolUse/solomonar-guard"
    },
    ".claude/skills/templ/SKILL.md": {
      "owner": "runtime:claude",
      "sha256": "` + emptyHash + `"
    },
    "AGENTS.md": {
      "owner": "core+notes",
      "sha256": "` + emptyHash + `"
    },
    "docs/README.md": {
      "owner": "seed"
    },
    "docs/decisions.md": {
      "owner": "decision",
      "entries": [
        "P/D-1",
        "P/D-2",
        "P/D-3"
      ]
    },
    "docs/solomonar/rules.md": {
      "owner": "core",
      "sha256": "` + emptyHash + `"
    },
    "docs/solomonar/skills/templ/SKILL.md": {
      "owner": "ext:templ",
      "sha256": "` + emptyHash + `"
    }
  }
}
`

const draft = `"money": {
      "state": "draft", "kind": "skill", "group": "backend", "source": "path",
      "path": "docs/solomonar/skills/money", "title": "Money handling",
      "requires": [], "conflicts": [], "paths": []
    },
    "go": {`

func withDraft() string {
	return strings.Replace(example, `"go": {`, draft, 1)
}

func rejectionOf(t *testing.T, err error) *Rejection {
	t.Helper()
	rejection, ok := errors.AsType[*Rejection](err)
	require.True(t, ok, "want a *Rejection, got %v", err)
	return rejection
}

func TestDecodeManifestRoundTripsTheInterfaceExample(t *testing.T) {
	want := canonicalExample
	manifest, err := DecodeManifest([]byte(example), framework)
	require.NoError(t, err)

	data, err := manifest.Encode()

	require.NoError(t, err)
	assert.Equal(t, want, string(data))
}

func TestDecodeManifestKeepsDraftEmptyLists(t *testing.T) {
	manifest, err := DecodeManifest([]byte(withDraft()), framework)
	require.NoError(t, err)

	data, err := manifest.Encode()

	require.NoError(t, err)
	assert.Equal(t, Extension{
		State: "draft", Kind: "skill", Group: "backend", Source: "path", Path: "docs/solomonar/skills/money",
		Title: "Money handling", Requires: []string{}, Conflicts: []string{}, Paths: []string{},
	}, manifest.Extensions["money"])
	assert.Contains(t, string(data), `"title": "Money handling",
      "requires": [],
      "conflicts": [],
      "paths": []
    },`)
}

func TestDecodeManifestRejectsInvalidManifest(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		found string
	}{
		{name: "missing schema version", data: `{"framework_version": "0.1.0"}`, found: "schema_version is missing"},
		{name: "schema version zero", data: `{"schema_version": 0}`, found: "schema_version 0 is not a positive integer"},
		{name: "missing framework version", data: `{"schema_version": 1}`, found: "framework_version is missing"},
		{name: "malformed framework version", data: `{"schema_version": 1, "framework_version": "0.1"}`, found: `framework_version "0.1" is not major.minor.patch`},
		{name: "missing runtimes", data: `{"schema_version": 1, "framework_version": "0.1.0", "project": {"name": "a", "main_branch": "main", "remote": "origin"}, "extensions": {}, "files": {}}`, found: "runtimes is missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeManifest([]byte(tt.data), framework)

			rejection := rejectionOf(t, err)
			assert.Equal(t, "manifest_invalid", rejection.Code)
			assert.Equal(t, "manifest schema", rejection.Check)
			assert.Equal(t, tt.found, rejection.Found)
			assert.Equal(t, "inspect .solomonar/manifest.json; never edit it by hand", rejection.Fix)
		})
	}
}

func TestDecodeManifestRejectsUnknownMembersAndDuplicateKeys(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		found string
	}{
		{name: "unknown member", data: strings.Replace(example, `"owner": "seed"`, `"owner": "seed", "mode": 1`, 1), found: "mode"},
		{name: "duplicate key", data: strings.Replace(example, `"owner": "seed"`, `"owner": "seed", "owner": "seed"`, 1), found: "owner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeManifest([]byte(tt.data), framework)

			rejection := rejectionOf(t, err)
			assert.Equal(t, "manifest_invalid", rejection.Code)
			assert.True(t, strings.HasPrefix(rejection.Found, "decode manifest.json: "), rejection.Found)
			assert.Contains(t, rejection.Found, tt.found)
		})
	}
}

func TestDecodeManifestRejectsEveryValidationRule(t *testing.T) {
	tests := []struct {
		name, old, new, found string
	}{
		{"missing project name", `"name": "ledger"`, `"name": ""`, "project.name is missing"},
		{"missing main branch", `"main_branch": "main"`, `"main_branch": ""`, "project.main_branch is missing"},
		{"missing remote", `"remote": "origin"`, `"remote": ""`, "project.remote is missing"},
		{"unknown runtime", `"codex": {}`, `"cursor": {}`, `runtimes.cursor is not claude or codex`},
		{"bad extension name", `"go": {`, `"Go": {`, `extensions."Go" is not a name of [a-z][a-z0-9-]*`},
		{"unknown state", `"state": "installed", "kind": "skill", "group": "language"`, `"state": "active", "kind": "skill", "group": "language"`, `extensions.go.state "active" is not installed or draft`},
		{"unknown kind", `"kind": "skill", "group": "language"`, `"kind": "rule", "group": "language"`, `extensions.go.kind "rule" is not skill or agent`},
		{"unknown group", `"group": "language"`, `"group": "lang"`, `extensions.go.group "lang" is not one of language, architecture, frontend, backend, persistence, testing, operations, workflow, review`},
		{"unknown source", `"group": "language", "source": "catalog"`, `"group": "language", "source": "git"`, `extensions.go.source "git" is not catalog or path`},
		{"catalog with path", `"group": "language", "source": "catalog"`, `"group": "language", "source": "catalog", "path": "x"`, "extensions.go.path is only for source path"},
		{"path source without path", `"path": "docs/solomonar/skills/money"`, `"path": ""`, "extensions.money.path is missing"},
		{"unclean path", `"path": "docs/solomonar/skills/money"`, `"path": "../money"`, `extensions.money.path "../money" is not a clean repo-relative path`},
		{"stack group without decision", `"decision": "P/D-1", `, ``, "extensions.go.decision is missing"},
		{"malformed decision", `"decision": "P/D-1"`, `"decision": "D-1"`, `extensions.go.decision "D-1" is not P/D-<n>`},
		{"workflow with decision", `"group": "language"`, `"group": "workflow"`, `extensions.go.decision is not allowed for group workflow`},
		{"draft with decision", `"title": "Money handling"`, `"decision": "P/D-4", "title": "Money handling"`, "extensions.money.decision is not allowed for a draft"},
		{"draft with rendered_with", `"title": "Money handling"`, `"rendered_with": ["go"], "title": "Money handling"`, "extensions.money.rendered_with is not allowed for a draft"},
		{"rendered_with unknown", `"rendered_with": ["tdd"]`, `"rendered_with": ["rust"]`, `extensions.go.rendered_with "rust" is not a known extension`},
		{"unclean file path", `"docs/README.md": {`, `"docs/../README.md": {`, `files."docs/../README.md" is not a clean repo-relative path`},
		{"missing owner", `"owner": "seed"`, `"owner": ""`, "files.docs/README.md.owner is missing"},
		{"unknown owner", `"owner": "seed"`, `"owner": "foo"`, `files.docs/README.md.owner "foo" is not an ownership class`},
		{"owner with unknown extension", `"owner": "ext:templ"`, `"owner": "ext:rust"`, `files.docs/solomonar/skills/templ/SKILL.md.owner "ext:rust" names no known extension`},
		{"owner with unknown runtime", `"owner": "runtime:claude", "entry"`, `"owner": "runtime:cursor", "entry"`, `files..claude/settings.json.owner "runtime:cursor" names no known runtime`},
		{"missing hash", `"owner": "core", "sha256": "` + emptyHash + `"`, `"owner": "core"`, "files.docs/solomonar/rules.md.sha256 is missing"},
		{"malformed hash", `"owner": "core", "sha256": "` + emptyHash + `"`, `"owner": "core", "sha256": "ABC"`, `files.docs/solomonar/rules.md.sha256 "ABC" is not 64 lowercase hex`},
		{"entry file with hash", `"entry": "hooks.PreToolUse/solomonar-guard"`, `"sha256": "` + emptyHash + `", "entry": "hooks.PreToolUse/solomonar-guard"`, `files..claude/settings.json.sha256 is not allowed for an entry`},
		{"core with entry", `"owner": "core", "sha256": "` + emptyHash + `"`, `"owner": "core", "sha256": "` + emptyHash + `", "entry": "x"`, `files.docs/solomonar/rules.md.entry is not allowed for owner "core"`},
		{"decision without entries", `"owner": "decision", "entries": ["P/D-1", "P/D-2", "P/D-3"]`, `"owner": "decision"`, "files.docs/decisions.md.entries is missing"},
		{"decision with hash", `"owner": "decision",`, `"owner": "decision", "sha256": "` + emptyHash + `",`, `files.docs/decisions.md.sha256 is not allowed for owner "decision"`},
		{"malformed entry", `"P/D-3"]`, `"W-001/D-3"]`, `files.docs/decisions.md.entries "W-001/D-3" is not P/D-<n>`},
		{"seed with hash", `"owner": "seed"`, `"owner": "seed", "sha256": "` + emptyHash + `"`, `files.docs/README.md.sha256 is not allowed for owner "seed"`},
		{"manifest with entries", `"owner": "seed"`, `"owner": "manifest", "entries": ["P/D-1"]`, `files.docs/README.md.entries is not allowed for owner "manifest"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := strings.Replace(withDraft(), tt.old, tt.new, 1)
			require.NotEqual(t, withDraft(), data, "the case changes nothing")

			_, err := DecodeManifest([]byte(data), framework)

			rejection := rejectionOf(t, err)
			assert.Equal(t, "manifest_invalid", rejection.Code)
			assert.Equal(t, tt.found, rejection.Found)
		})
	}
}

func TestDecodeManifestRejectsNewerVersionsBeforeStrictDecoding(t *testing.T) {
	tests := []struct {
		name, data, check, found string
	}{
		{
			name:  "schema",
			data:  `{"schema_version": 2, "framework_version": "bad", "added_in_v2": true}`,
			check: "manifest schema version",
			found: "manifest schema 2, this CLI supports 1",
		},
		{
			name:  "framework",
			data:  `{"schema_version": 1, "framework_version": "0.10.0", "project": {}}`,
			check: "manifest framework version",
			found: "manifest framework 0.10.0, this CLI ships 0.2.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeManifest([]byte(tt.data), "0.2.0")

			rejection := rejectionOf(t, err)
			assert.Equal(t, "cli_outdated", rejection.Code)
			assert.Equal(t, tt.check, rejection.Check)
			assert.Equal(t, tt.found, rejection.Found)
			assert.Equal(t, "install a newer solomonar release", rejection.Fix)
		})
	}
}

func TestDecodeManifestAcceptsOlderFramework(t *testing.T) {
	manifest, err := DecodeManifest([]byte(example), "0.10.0")

	require.NoError(t, err)
	assert.Equal(t, "0.1.0", manifest.FrameworkVersion)
}

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	manifest, err := ReadManifest(root, framework)
	require.NoError(t, err)
	assert.Nil(t, manifest)

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".solomonar"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ManifestPath), []byte(example), 0o644))
	manifest, err = ReadManifest(root, framework)
	require.NoError(t, err)
	assert.Equal(t, "ledger", manifest.Project.Name)
}

func TestReadManifestRejectsSymlinkOutsideRoot(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, os.WriteFile(outside, []byte(example), 0o644))
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".solomonar"), 0o755))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, ManifestPath)))
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	manifest, err := ReadManifest(root, framework)

	require.ErrorContains(t, err, "read manifest")
	assert.Nil(t, manifest)
}

func TestHashIsLowercaseHexSHA256(t *testing.T) {
	assert.Equal(t, emptyHash, Hash(nil))
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", Hash([]byte("hello")))
}
