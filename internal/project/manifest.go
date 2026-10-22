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

// Manifest is the decoded .solomonar/manifest.json (I section 4.7).
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
	// omitzero keeps a project extension's empty lists, as in the I section 4.7 draft.
	Requires    []string `json:"requires,omitzero"`
	Conflicts   []string `json:"conflicts,omitzero"`
	AppliesWhen string   `json:"applies_when,omitempty"`
	Paths       []string `json:"paths,omitzero"`
}

// File is one managed file and its ownership class (I section 4.5).
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

// ManifestVersions reads the manifest's schema and framework versions without the newer-than checks.
func ManifestVersions(data []byte) (schema int, framework string, err error) {
	header, err := decodeHeader(data)
	if err != nil {
		return 0, "", err
	}
	if schema, err = header.schema(); err != nil {
		return 0, "", err
	}
	if framework, err = header.framework(); err != nil {
		return 0, "", err
	}
	return schema, framework, nil
}

// NewerFramework reports whether the project framework version is newer than the shipped one.
func NewerFramework(project, shipped string) (bool, error) {
	projectParts, ok := parseVersion(project)
	if !ok {
		return false, fmt.Errorf("compare framework versions: project version %q is not major.minor.patch", project)
	}
	shippedParts, ok := parseVersion(shipped)
	if !ok {
		return false, fmt.Errorf("compare framework versions: embedded version %q is not major.minor.patch", shipped)
	}
	return slices.Compare(projectParts, shippedParts) > 0, nil
}

type versionHeader struct {
	SchemaVersion    jsontext.Value `json:"schema_version"`
	FrameworkVersion jsontext.Value `json:"framework_version"`
}

func decodeHeader(data []byte) (versionHeader, error) {
	var header versionHeader
	if err := json.Unmarshal(data, &header, jsontext.AllowDuplicateNames(true)); err != nil {
		return versionHeader{}, invalidManifest("decode manifest.json: %v", err)
	}
	return header, nil
}

func (h versionHeader) schema() (int, error) {
	if h.SchemaVersion == nil {
		return 0, invalidManifest("schema_version is missing")
	}
	var schema int
	if err := json.Unmarshal(h.SchemaVersion, &schema); err != nil || schema < 1 {
		return 0, invalidManifest("schema_version %s is not a positive integer", h.SchemaVersion)
	}
	return schema, nil
}

func (h versionHeader) framework() (string, error) {
	if h.FrameworkVersion == nil {
		return "", invalidManifest("framework_version is missing")
	}
	var version string
	err := json.Unmarshal(h.FrameworkVersion, &version)
	if _, ok := parseVersion(version); err != nil || !ok {
		return "", invalidManifest("framework_version %s is not major.minor.patch", h.FrameworkVersion)
	}
	return version, nil
}

func checkVersions(data []byte, framework string) error {
	header, err := decodeHeader(data)
	if err != nil {
		return err
	}
	schema, err := header.schema()
	if err != nil {
		return err
	}
	if schema > ManifestSchemaVersion {
		return outdated("manifest schema version",
			fmt.Sprintf("manifest schema %d, this CLI supports %d", schema, ManifestSchemaVersion))
	}
	version, err := header.framework()
	if err != nil {
		return err
	}
	newer, err := NewerFramework(version, framework)
	if err != nil {
		return err
	}
	if newer {
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
