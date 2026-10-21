package project

import (
	"cmp"
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// Groups lists the extension groups in the I §4.3 order.
func Groups() []string { return groups() }

func groups() []string {
	return []string{
		"language", "architecture", "frontend", "backend", "persistence", "testing", "operations", "workflow", "review",
	}
}

func (m *Manifest) validate() error {
	if err := m.validateProject(); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(m.Runtimes)) {
		if name != "claude" && name != "codex" {
			return invalidManifest("runtimes.%s is not claude or codex", name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(m.Extensions)) {
		if err := m.validateExtension(name, m.Extensions[name]); err != nil {
			return err
		}
	}
	for _, key := range slices.Sorted(maps.Keys(m.Files)) {
		if err := m.validateFile(key, m.Files[key]); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manifest) validateProject() error {
	switch {
	case m.Project.Name == "":
		return invalidManifest("project.name is missing")
	case m.Project.MainBranch == "":
		return invalidManifest("project.main_branch is missing")
	case m.Project.Remote == "":
		return invalidManifest("project.remote is missing")
	case m.Runtimes == nil:
		return invalidManifest("runtimes is missing")
	case m.Extensions == nil:
		return invalidManifest("extensions is missing")
	case m.Files == nil:
		return invalidManifest("files is missing")
	}
	return nil
}

func (m *Manifest) validateExtension(name string, e Extension) error {
	if !extensionName(name) {
		return invalidManifest("extensions.%q is not a name of [a-z][a-z0-9-]*", name)
	}
	field := "extensions." + name + "."
	if err := cmp.Or(
		oneOf(field+"state", e.State, []string{"installed", "draft"}, "installed or draft"),
		oneOf(field+"kind", e.Kind, []string{"skill", "agent"}, "skill or agent"),
		oneOf(field+"group", e.Group, groups(), "one of "+strings.Join(groups(), ", ")),
		oneOf(field+"source", e.Source, []string{"catalog", "path"}, "catalog or path"),
		validateSourcePath(field+"path", e),
		validateDecision(field+"decision", e),
	); err != nil {
		return err
	}
	if e.State == "draft" && len(e.RenderedWith) > 0 {
		return invalidManifest("%srendered_with is not allowed for a draft", field)
	}
	for _, other := range e.RenderedWith {
		if _, ok := m.Extensions[other]; !ok {
			return invalidManifest("%srendered_with %q is not a known extension", field, other)
		}
	}
	return nil
}

func oneOf(field, value string, allowed []string, phrase string) error {
	if value == "" {
		return invalidManifest("%s is missing", field)
	}
	if !slices.Contains(allowed, value) {
		return invalidManifest("%s %q is not %s", field, value, phrase)
	}
	return nil
}

func validateSourcePath(field string, e Extension) error {
	switch {
	case e.Source == "catalog" && e.Path != "":
		return invalidManifest("%s is only for source path", field)
	case e.Source == "path" && e.Path == "":
		return invalidManifest("%s is missing", field)
	case e.Source == "path" && !cleanRelativePath(e.Path):
		return invalidManifest("%s %q is not a clean repo-relative path", field, e.Path)
	}
	return nil
}

func validateDecision(field string, e Extension) error {
	switch {
	case e.State == "draft" && e.Decision != "":
		return invalidManifest("%s is not allowed for a draft", field)
	case e.State == "draft":
		return nil
	case (e.Group == "workflow" || e.Group == "review") && e.Decision != "":
		return invalidManifest("%s is not allowed for group %s", field, e.Group)
	case e.Group == "workflow" || e.Group == "review":
		return nil
	case e.Decision == "":
		return invalidManifest("%s is missing", field)
	case !projectDecision(e.Decision):
		return invalidManifest("%s %q is not P/D-<n>", field, e.Decision)
	}
	return nil
}

func (m *Manifest) validateFile(key string, f File) error {
	if !cleanRelativePath(key) {
		return invalidManifest("files.%q is not a clean repo-relative path", key)
	}
	field := "files." + key + "."
	if err := m.validateOwner(field+"owner", f.Owner); err != nil {
		return err
	}
	want, reason := fileRecord(f)
	present := map[string]bool{"sha256": f.SHA256 != "", "entry": f.Entry != "", "entries": len(f.Entries) > 0}
	for _, name := range []string{"sha256", "entry", "entries"} {
		if present[name] && name != want {
			return invalidManifest("%s%s is not allowed for %s", field, name, reason)
		}
	}
	if want != "" && !present[want] {
		return invalidManifest("%s%s is missing", field, want)
	}
	if want == "sha256" && !lowerHex64(f.SHA256) {
		return invalidManifest("%ssha256 %q is not 64 lowercase hex", field, f.SHA256)
	}
	for _, entry := range f.Entries {
		if !projectDecision(entry) {
			return invalidManifest("%sentries %q is not P/D-<n>", field, entry)
		}
	}
	return nil
}

func (m *Manifest) validateOwner(field, owner string) error {
	if owner == "" {
		return invalidManifest("%s is missing", field)
	}
	class, name, named := strings.Cut(owner, ":")
	switch {
	case !named && slices.Contains([]string{"core", "core+notes", "decision", "seed", "manifest"}, class):
		return nil
	case named && (class == "ext" || class == "project-ext"):
		if _, ok := m.Extensions[name]; !ok {
			return invalidManifest("%s %q names no known extension", field, owner)
		}
		return nil
	case named && class == "runtime":
		if _, ok := m.Runtimes[name]; !ok {
			return invalidManifest("%s %q names no known runtime", field, owner)
		}
		return nil
	}
	return invalidManifest("%s %q is not an ownership class", field, owner)
}

// fileRecord names the one field a file of this owner carries, and why the others are not allowed.
func fileRecord(f File) (field, reason string) {
	class, _, _ := strings.Cut(f.Owner, ":")
	reason = fmt.Sprintf("owner %q", f.Owner)
	switch {
	case class == "decision":
		return "entries", reason
	case class == "seed" || class == "manifest":
		return "", reason
	case class == "runtime" && f.Entry != "":
		return "entry", "an entry"
	default:
		return "sha256", reason
	}
}

func extensionName(name string) bool {
	return name != "" && name[0] >= 'a' && name[0] <= 'z' &&
		!strings.ContainsFunc(name, func(r rune) bool { return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' })
}

func projectDecision(ref string) bool {
	digits, ok := strings.CutPrefix(ref, "P/D-")
	return ok && allDigits(digits)
}

func lowerHex64(s string) bool {
	return len(s) == 64 &&
		!strings.ContainsFunc(s, func(r rune) bool { return (r < '0' || r > '9') && (r < 'a' || r > 'f') })
}

func cleanRelativePath(value string) bool {
	return value != "." && path.Clean(value) == value && filepath.IsLocal(filepath.FromSlash(value))
}
