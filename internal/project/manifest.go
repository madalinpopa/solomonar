// Package project holds the project's managed files and their manifest.
package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
)

// ManifestSchemaVersion is the manifest schema this CLI reads and writes.
const ManifestSchemaVersion = 1

// ManifestPath is where the manifest lives, relative to the repository root.
const ManifestPath = ".solomonar/manifest.json"

// Manifest is the decoded .solomonar/manifest.json (I §4.7).
type Manifest struct {
	SchemaVersion    int                  `json:"schema_version"`
	FrameworkVersion string               `json:"framework_version"`
	Project          Project              `json:"project"`
	Runtimes         map[string]Runtime   `json:"runtimes"`
	Extensions       map[string]Extension `json:"extensions"`
	Files            map[string]File      `json:"files"`
}

// Project names the repository's main branch and remote.
type Project struct {
	Name       string `json:"name"`
	MainBranch string `json:"main_branch"`
	Remote     string `json:"remote"`
}

// Runtime is one installed runtime adapter.
type Runtime struct {
	AdversarialModel string `json:"adversarial_model,omitempty"`
}

// Extension is one installed or draft extension.
type Extension struct {
	State        string   `json:"state"`
	Kind         string   `json:"kind"`
	Group        string   `json:"group"`
	Source       string   `json:"source"`
	Path         string   `json:"path,omitempty"`
	Decision     string   `json:"decision,omitempty"`
	RenderedWith []string `json:"rendered_with,omitempty"`
	Title        string   `json:"title,omitempty"`
	// omitzero keeps a project extension's empty lists, as in the I §4.7 draft.
	Requires    []string `json:"requires,omitzero"`
	Conflicts   []string `json:"conflicts,omitzero"`
	AppliesWhen string   `json:"applies_when,omitempty"`
	Paths       []string `json:"paths,omitzero"`
}

// File is one managed file and its ownership class (I §4.5).
type File struct {
	Owner   string   `json:"owner"`
	SHA256  string   `json:"sha256,omitempty"`
	Entry   string   `json:"entry,omitempty"`
	Entries []string `json:"entries,omitempty"`
}

// DecodeManifest reads the manifest, checking its versions before decoding strictly (CLI-D19, AD-7).
func DecodeManifest(data []byte, framework string) (*Manifest, error) {
	if err := checkVersions(data, framework); err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest, json.RejectUnknownMembers(true)); err != nil {
		return nil, invalidManifest("decode manifest.json: %v", err)
	}
	if err := manifest.validate(); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// ReadManifest decodes the manifest under root; it returns nil without error when the file is absent.
func ReadManifest(root *os.Root, framework string) (*Manifest, error) {
	data, err := root.ReadFile(ManifestPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	return DecodeManifest(data, framework)
}

// Encode writes the manifest in its canonical form: struct order, sorted map keys, two-space indent.
func (m *Manifest) Encode() ([]byte, error) {
	data, err := json.Marshal(m, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return nil, fmt.Errorf("encode manifest: %w", err)
	}
	return append(data, '\n'), nil
}

// Hash is the lowercase hex SHA-256 of content.
func Hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func checkVersions(data []byte, framework string) error {
	var header struct {
		SchemaVersion    jsontext.Value `json:"schema_version"`
		FrameworkVersion jsontext.Value `json:"framework_version"`
	}
	if err := json.Unmarshal(data, &header, jsontext.AllowDuplicateNames(true)); err != nil {
		return invalidManifest("decode manifest.json: %v", err)
	}
	if header.SchemaVersion == nil {
		return invalidManifest("schema_version is missing")
	}
	var schema int
	if err := json.Unmarshal(header.SchemaVersion, &schema); err != nil || schema < 1 {
		return invalidManifest("schema_version %s is not a positive integer", header.SchemaVersion)
	}
	if schema > ManifestSchemaVersion {
		return outdated("manifest schema version",
			fmt.Sprintf("manifest schema %d, this CLI supports %d", schema, ManifestSchemaVersion))
	}
	if header.FrameworkVersion == nil {
		return invalidManifest("framework_version is missing")
	}
	var version string
	err := json.Unmarshal(header.FrameworkVersion, &version)
	project, ok := parseVersion(version)
	if err != nil || !ok {
		return invalidManifest("framework_version %s is not major.minor.patch", header.FrameworkVersion)
	}
	shipped, ok := parseVersion(framework)
	if !ok {
		return fmt.Errorf("compare framework versions: embedded version %q is not major.minor.patch", framework)
	}
	if slices.Compare(project, shipped) > 0 {
		return outdated("manifest framework version",
			fmt.Sprintf("manifest framework %s, this CLI ships %s", version, framework))
	}
	return nil
}

func parseVersion(version string) ([]int, bool) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return nil, false
	}
	numbers := make([]int, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || !allDigits(part) {
			return nil, false
		}
		numbers = append(numbers, number)
	}
	return numbers, true
}

func allDigits(s string) bool {
	return s != "" && !strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
}
