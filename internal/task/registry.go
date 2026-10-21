// Package task holds the task registry and its rules.
package task

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// SchemaVersion is the registry schema this CLI reads and writes.
const SchemaVersion = 1

// Registry is the decoded state.json (rules §7).
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
	Branch       string      `json:"branch,omitempty"`
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

// Decode reads state.json, checking the schema version before decoding strictly (CLI-D19, AD-7).
func Decode(data []byte) (*Registry, error) {
	if err := checkSchemaVersion(data); err != nil {
		return nil, err
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

// Encode writes the registry in its canonical form: struct order, sorted task IDs, two-space indent.
func (r *Registry) Encode() ([]byte, error) {
	data, err := json.Marshal(r, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return nil, fmt.Errorf("encode registry: %w", err)
	}
	return append(data, '\n'), nil
}

// Encode writes the record as it appears in state.json, at the top level.
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
	if version > SchemaVersion {
		return &Rejection{
			Code:  "cli_outdated",
			Check: "registry schema version",
			Found: fmt.Sprintf("registry schema %d, this CLI supports %d", version, SchemaVersion),
			Fix:   "install a newer solomonar release",
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
