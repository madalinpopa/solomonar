// Package task holds the task registry and its rules.
package task

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// SchemaVersion is the registry schema this CLI reads and writes.
const SchemaVersion = 2

// legacySchema is the single-file state.json layout that task migrate reads (CLI-D26).
const legacySchema = 1

const (
	headerFile = "state.json"
	tasksDir   = "tasks/"
)

// Registry is the decoded registry: state.json and one tasks/<ID>.json per task (CLI-D20).
type Registry struct {
	SchemaVersion int                `json:"schema_version"`
	Tasks         map[string]*Record `json:"tasks"`
}

// Record is one task in the registry; empty optional fields are omitted.
type Record struct {
	Title        string      `json:"title,omitempty"`
	Status       Status      `json:"status,omitempty"`
	Stage        Stage       `json:"stage,omitempty"`
	Owner        string      `json:"owner,omitempty"`
	Confirmed    bool        `json:"confirmed,omitzero"`
	Branch       string      `json:"branch,omitempty"`
	Base         string      `json:"base,omitempty"`
	Worktree     string      `json:"worktree,omitempty"`
	Spec         string      `json:"spec,omitempty"`
	Revision     string      `json:"revision,omitempty"`
	Tier         *int        `json:"tier,omitzero"`
	ActiveAgents []Agent     `json:"active_agents,omitempty"`
	Blockers     []Blocker   `json:"blockers,omitempty"`
	LastBlocker  int         `json:"last_blocker,omitzero"`
	NextAction   *NextAction `json:"next_action,omitzero"`
	PR           string      `json:"pr,omitempty"`
	DependsOn    []string    `json:"depends_on,omitempty"`
	RelatedTask  string      `json:"related_task,omitempty"`
	SessionRef   string      `json:"session_ref,omitempty"`
	Summary      string      `json:"summary,omitempty"`
	SpecRef      string      `json:"spec_ref,omitempty"`
	MergeCommit  string      `json:"merge_commit,omitempty"`
	UpdatedAt    string      `json:"updated_at,omitempty"`
}

// Agent is a subagent working on a task.
type Agent struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// Blocker is an open reason a task cannot proceed.
type Blocker struct {
	ID      string      `json:"id"`
	Kind    BlockerKind `json:"kind"`
	Summary string      `json:"summary"`
	Ref     string      `json:"ref,omitempty"`
}

// NextAction names who acts next on a task and what they do.
type NextAction struct {
	Actor  string `json:"actor"`
	Action string `json:"action"`
}

// Status is the lifecycle status of a task.
type Status string

const (
	StatusPlanned   Status = "planned"
	StatusReserved  Status = "reserved"
	StatusRunning   Status = "running"
	StatusBlocked   Status = "blocked"
	StatusPaused    Status = "paused"
	StatusCancelled Status = "cancelled"
	StatusDone      Status = "done"
)

// Stage is the workflow stage of a task.
type Stage string

const (
	StageSetup             Stage = "setup"
	StageSpecification     Stage = "specification"
	StageImplementation    Stage = "implementation"
	StageSpecializedReview Stage = "specialized_review"
	StageAdversarialReview Stage = "adversarial_review"
	StageIntegration       Stage = "integration"
	StageComplete          Stage = "complete"
)

// BlockerKind is what a blocker waits on.
type BlockerKind string

const (
	BlockerDecision    BlockerKind = "decision"
	BlockerDependency  BlockerKind = "dependency"
	BlockerCheck       BlockerKind = "check"
	BlockerEnvironment BlockerKind = "environment"
)

// NewRegistry returns an empty registry at the current schema version.
func NewRegistry() *Registry {
	return &Registry{SchemaVersion: SchemaVersion, Tasks: map[string]*Record{}}
}

// DecodeLegacy reads a schema-1 state.json, checking its version before decoding strictly (AD-7).
func DecodeLegacy(data []byte) (*Registry, error) {
	version, err := SchemaOf(data)
	if err != nil {
		return nil, err
	}
	if version != legacySchema {
		return nil, invalidRegistry("schema_version %d is not the single-file schema %d", version, legacySchema)
	}
	var registry Registry
	if err := json.Unmarshal(data, &registry, json.RejectUnknownMembers(true)); err != nil {
		return nil, invalidRegistry("decode state.json: %v", err)
	}
	if registry.Tasks == nil {
		return nil, invalidRegistry("tasks is missing")
	}
	if err := registry.validate(); err != nil {
		return nil, err
	}
	return &registry, nil
}

// DecodeFiles reads the registry from its files keyed by tree path, strictly (CLI-D20).
func DecodeFiles(files map[string][]byte) (*Registry, error) {
	header, ok := files[headerFile]
	if !ok {
		return nil, invalidRegistry("state.json is missing")
	}
	if err := checkSchemaVersion(header); err != nil {
		return nil, err
	}
	registry := &Registry{}
	if err := json.Unmarshal(header, registry, json.RejectUnknownMembers(true)); err != nil {
		return nil, invalidRegistry("decode state.json: %v", err)
	}
	if registry.Tasks != nil {
		return nil, invalidRegistry("state.json holds tasks; records live in tasks/<ID>.json")
	}
	registry.Tasks = map[string]*Record{}
	for _, path := range slices.Sorted(maps.Keys(files)) {
		if path == headerFile {
			continue
		}
		id, ok := recordID(path)
		if !ok {
			return nil, invalidRegistry("unexpected file %s; only state.json and tasks/<ID>.json belong", path)
		}
		var record Record
		if err := json.Unmarshal(files[path], &record, json.RejectUnknownMembers(true)); err != nil {
			return nil, invalidRegistry("decode %s: %v", path, err)
		}
		registry.Tasks[id] = &record
	}
	if err := registry.validate(); err != nil {
		return nil, err
	}
	return registry, nil
}

// Files encodes the registry as its tree files keyed by path, each in canonical form.
func (r *Registry) Files() (map[string][]byte, error) {
	header, err := json.Marshal(struct {
		SchemaVersion int `json:"schema_version"`
	}{r.SchemaVersion}, jsontext.WithIndent("  "))
	if err != nil {
		return nil, fmt.Errorf("encode state.json: %w", err)
	}
	files := map[string][]byte{headerFile: append(header, '\n')}
	for id, record := range r.Tasks {
		data, err := record.Encode()
		if err != nil {
			return nil, err
		}
		files[recordPath(id)] = data
	}
	return files, nil
}

func recordPath(id string) string {
	return tasksDir + id + ".json"
}

func recordID(path string) (string, bool) {
	name, ok := strings.CutPrefix(path, tasksDir)
	if !ok {
		return "", false
	}
	id, ok := strings.CutSuffix(name, ".json")
	return id, ok && validTaskID(id)
}

// Encode writes the registry as one JSON document: the task show --json view.
func (r *Registry) Encode() ([]byte, error) {
	data, err := json.Marshal(r, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return nil, fmt.Errorf("encode registry: %w", err)
	}
	return append(data, '\n'), nil
}

// Encode writes the record as it appears in its tasks/<ID>.json file.
func (r *Record) Encode() ([]byte, error) {
	data, err := json.Marshal(r, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return nil, fmt.Errorf("encode record: %w", err)
	}
	return append(data, '\n'), nil
}

func checkSchemaVersion(data []byte) error {
	version, err := SchemaOf(data)
	if err != nil {
		return err
	}
	switch {
	case version > SchemaVersion:
		return &Rejection{
			Code:  "cli_outdated",
			Check: "registry schema version",
			Found: fmt.Sprintf("registry schema %d, this CLI supports %d", version, SchemaVersion),
			Fix:   "install a newer solomonar release",
		}
	case version < SchemaVersion:
		return &Rejection{
			Code:  "registry_outdated",
			Check: "registry schema version",
			Found: fmt.Sprintf("registry schema %d, this CLI uses %d", version, SchemaVersion),
			Fix:   "solomonar task migrate",
		}
	}
	return nil
}

// SchemaOf reads the registry's schema version without the newer-than check (UC-24).
func SchemaOf(data []byte) (int, error) {
	var header struct {
		SchemaVersion jsontext.Value `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &header, jsontext.AllowDuplicateNames(true)); err != nil {
		return 0, invalidRegistry("decode state.json: %v", err)
	}
	if header.SchemaVersion == nil {
		return 0, invalidRegistry("schema_version is missing")
	}
	var version int
	if err := json.Unmarshal(header.SchemaVersion, &version); err != nil || version < 1 {
		return 0, invalidRegistry("schema_version %s is not a positive integer", header.SchemaVersion)
	}
	return version, nil
}
