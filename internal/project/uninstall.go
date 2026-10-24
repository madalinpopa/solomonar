package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
)

const (
	hookEntry  = "hook entry removal is not supported"
	outside    = "outside the paths solomonar uninstall removes"
	sourceKept = "project extension; source kept"
	notesKept  = "managed part removed; project notes kept"
	noBoundary = `no "## Project notes" boundary`
	readmePath = "docs/README.md"
	indexLine  = "- `docs/solomonar/`: framework rules, roles, and skills"
	customized = "framework index entry customized; review by hand"
	// notesComment and oldNotesComment are the comments AGENTS.md templates put under "## Project notes".
	notesComment    = "<!-- Project-owned instructions; preserved by framework upgrades. Keep this heading unchanged. -->"
	oldNotesComment = "<!-- Project-specific instructions go here, e.g. adopted style guides (such as Go Code Review Comments),\n" +
		"branch rules for direct commits, and the test policy. This section is owned by the project and preserved by\n" +
		"framework upgrades. -->"
)

// Uninstall is the removal plan of solomonar uninstall (I UC-27).
type Uninstall struct {
	Plan Plan
	// Review lists retained files whose framework references need manual cleanup (manual_reference).
	Review []string
}

// ReadUninstallManifest returns the manifest's bytes, or nil when it is absent; a manifest that is not
// a regular file, or is reached through a symlinked directory, is never read (I UC-27).
func ReadUninstallManifest(root *os.Root) ([]byte, error) {
	present, regular, err := inspect(root, ManifestPath)
	switch {
	case err != nil || !present:
		return nil, err
	case !regular:
		return nil, &Rejection{
			Code:  "file_conflict",
			Check: "the manifest is a regular file",
			Found: ManifestPath + " " + notRegular,
			Fix:   "restore " + ManifestPath + " as a regular file, then rerun solomonar uninstall",
		}
	}
	data, err := root.ReadFile(ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	return data, nil
}

// PlanUninstall plans removing the installation m records from the files under root, writing nothing
// (I UC-27, AD-16); without a manifest it plans nothing and only checks r.
func PlanUninstall(root *os.Root, m *Manifest, r Resolutions) (Uninstall, error) {
	if err := r.distinct(); err != nil {
		return Uninstall{}, err
	}
	if m == nil {
		return Uninstall{}, r.resolved(nil)
	}
	changes, err := r.uninstallRecords(root, m)
	if err != nil {
		return Uninstall{}, err
	}
	var u Uninstall
	if file := m.Files[readmePath]; file.SHA256 == "" && file.Entry == "" {
		change, present, review, err := uninstallIndex(root)
		if err != nil {
			return Uninstall{}, err
		}
		if present {
			changes = append(changes, change)
		}
		if review {
			u.Review = append(u.Review, readmePath)
		}
	}
	slices.SortStableFunc(changes, func(a, b Change) int { return strings.Compare(a.Path, b.Path) })
	dangling, err := danglingImport(root, changes)
	if err != nil {
		return Uninstall{}, err
	}
	if dangling {
		u.Review = append(u.Review, "CLAUDE.md")
	}
	slices.Sort(u.Review)
	u.Plan.Changes = slices.Concat(changes, []Change{{Path: ManifestPath, Action: ActionRemove, Owner: "manifest"}})
	u.Plan.Prune = []string{"docs/solomonar/extensions", "docs/solomonar", ".solomonar"}
	return u, nil
}

// uninstallRecords plans the project extension directories and every recorded path, with their
// resolutions.
func (r Resolutions) uninstallRecords(root *os.Root, m *Manifest) ([]Change, error) {
	var changes []Change
	var conflicts []string
	sources := projectSources(m)
	for _, name := range slices.Sorted(maps.Keys(sources)) {
		changes = append(changes, Change{Path: sources[name], Action: ActionSkip, Owner: "project-ext:" + name, Reason: sourceKept})
	}
	for _, name := range slices.Sorted(maps.Keys(m.Files)) {
		file := m.Files[name]
		if projectSource(name, file, sources) {
			changes = append(changes, Change{Path: name, Action: ActionSkip, Owner: file.Owner, Reason: sourceKept})
			continue
		}
		change, overwrite, present, err := uninstallRecord(root, name, file)
		if err != nil {
			return nil, err
		}
		if !present {
			continue
		}
		if change.Action == ActionConflict {
			conflicts = append(conflicts, name)
			if change, err = r.uninstallResolve(change, overwrite); err != nil {
				return nil, err
			}
		}
		changes = append(changes, change)
	}
	return changes, r.resolved(conflicts)
}

// uninstallIndex plans removing the generated framework line from docs/README.md (CLI-D35); review
// is true when framework references remain for manual cleanup.
func uninstallIndex(root *os.Root) (change Change, present, review bool, err error) {
	change = Change{Path: readmePath, Owner: "seed", Action: ActionSkip}
	present, regular, err := inspect(root, readmePath)
	switch {
	case err != nil || !present:
		return Change{}, false, false, err
	case !regular:
		change.Reason = notRegular
		return change, true, true, nil
	}
	disk, err := root.ReadFile(readmePath)
	if err != nil {
		return Change{}, false, false, fmt.Errorf("classify %s: %w", readmePath, err)
	}
	content, found := withoutIndexLine(disk)
	mentions := bytes.Contains(content, []byte("docs/solomonar"))
	switch {
	case found:
		change.Action, change.Content, change.Reason = ActionUpdate, content, "framework index line removed"
		return change, true, mentions, nil
	case mentions:
		change.Reason = customized
		return change, true, true, nil
	}
	return Change{}, false, false, nil
}

// withoutIndexLine removes the first exact framework index line of the "## Documents" section,
// with its line ending, keeping every other byte.
func withoutIndexLine(content []byte) ([]byte, bool) {
	documents := false
	for start := 0; start < len(content); {
		next := len(content)
		if i := bytes.IndexByte(content[start:], '\n'); i >= 0 {
			next = start + i + 1
		}
		line := strings.TrimSuffix(strings.TrimSuffix(string(content[start:next]), "\n"), "\r")
		switch {
		case line == "## Documents":
			documents = true
		case strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## "):
			documents = false
		case documents && line == indexLine:
			return slices.Concat(content[:start], content[next:]), true
		}
		start = next
	}
	return content, false
}

// danglingImport reports whether a retained CLAUDE.md imports an AGENTS.md the plan removes.
func danglingImport(root *os.Root, changes []Change) (bool, error) {
	removes := func(name string) bool {
		return slices.ContainsFunc(changes, func(c Change) bool { return c.Path == name && c.Action == ActionRemove })
	}
	if !removes("AGENTS.md") || removes("CLAUDE.md") {
		return false, nil
	}
	_, regular, err := inspect(root, "CLAUDE.md")
	if err != nil || !regular {
		return false, err
	}
	disk, err := root.ReadFile("CLAUDE.md")
	if err != nil {
		return false, fmt.Errorf("classify CLAUDE.md: %w", err)
	}
	for line := range strings.SplitSeq(string(disk), "\n") {
		if strings.TrimSuffix(line, "\r") == "@AGENTS.md" {
			return true, nil
		}
	}
	return false, nil
}

// projectSources maps each path extension to its source directory.
func projectSources(m *Manifest) map[string]string {
	sources := map[string]string{}
	for name, e := range m.Extensions {
		if e.Source == "path" {
			sources[name] = e.Path
		}
	}
	return sources
}

// projectSource reports whether a record is a legacy project-ext record or lies in a project
// extension's source directory; the source wins over its hash record.
func projectSource(name string, file File, sources map[string]string) bool {
	if strings.HasPrefix(file.Owner, "project-ext:") {
		return true
	}
	for _, dir := range sources {
		if name == dir || strings.HasPrefix(name, dir+"/") {
			return true
		}
	}
	return false
}

// uninstallRecord plans one recorded path by the first matching I UC-27 row and returns what
// --overwrite makes of a conflict when that is not a plain removal; present is false when the path
// gets no plan line.
func uninstallRecord(root *os.Root, name string, file File) (change, overwrite Change, present bool, err error) {
	change = Change{Path: name, Owner: file.Owner}
	switch {
	case file.Entry != "":
		return conflict(change, hookEntry), Change{}, true, nil
	case file.SHA256 == "" || name == ManifestPath:
		return Change{}, Change{}, false, nil
	case !removable(name):
		return conflict(change, outside), Change{}, true, nil
	}
	present, regular, err := inspect(root, name)
	switch {
	case err != nil || !present:
		return Change{}, Change{}, false, err
	case !regular:
		return conflict(change, notRegular), Change{}, true, nil
	case file.Owner == "core+notes":
		return uninstallNotes(root, change, file.SHA256)
	}
	change, present, err = classifyRemoval(root, name, file)
	return change, Change{}, present, err
}

// adapterDirs are the runtime stub directories uninstall may remove from (I section 4.2).
func adapterDirs() []string {
	return []string{".claude/skills/", ".claude/agents/", ".agents/skills/", ".codex/agents/", ".github/skills/", ".github/agents/"}
}

// removable reports whether name is in the paths uninstall may remove (I UC-27).
func removable(name string) bool {
	return name == "AGENTS.md" || name == "CLAUDE.md" || strings.HasPrefix(name, "docs/solomonar/") ||
		slices.ContainsFunc(adapterDirs(), func(dir string) bool { return strings.HasPrefix(name, dir) })
}

// inspect reports whether name exists and is a regular file; a path reached through a symlinked
// parent directory exists and is not regular.
func inspect(root *os.Root, name string) (present, regular bool, err error) {
	linked, err := symlinkedParent(root, name)
	if err != nil || linked {
		return linked, false, err
	}
	info, err := root.Lstat(name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return false, false, nil
	case err != nil:
		return false, false, fmt.Errorf("classify %s: %w", name, err)
	}
	return true, info.Mode().IsRegular(), nil
}

// symlinkedParent reports whether a parent directory of name is a symlink; an absent parent is not.
func symlinkedParent(root *os.Root, name string) (bool, error) {
	dirs := strings.Split(path.Dir(name), "/")
	for i := range dirs {
		dir := strings.Join(dirs[:i+1], "/")
		if dir == "." {
			return false, nil
		}
		info, err := root.Lstat(dir)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return false, nil
		case err != nil:
			return false, fmt.Errorf("classify %s: %w", name, err)
		case info.Mode()&fs.ModeSymlink != 0:
			return true, nil
		}
	}
	return false, nil
}

// uninstallNotes plans AGENTS.md: the verified managed part goes and the notes part stays (CLI-D34);
// --overwrite drops an edited managed part the same way.
func uninstallNotes(root *os.Root, change Change, base string) (Change, Change, bool, error) {
	disk, err := root.ReadFile(change.Path)
	if err != nil {
		return Change{}, Change{}, false, fmt.Errorf("classify %s: %w", change.Path, err)
	}
	managed, notes := splitNotes(disk)
	kept := change
	if templateNotes(notes) {
		kept.Action = ActionRemove
	} else {
		kept.Action, kept.Content, kept.Reason = ActionUpdate, notes, notesKept
	}
	switch {
	case Hash(managed) == base:
		return kept, Change{}, true, nil
	case notes == nil:
		return conflict(change, noBoundary), Change{}, true, nil
	}
	return conflict(change, modifiedReason), kept, true, nil
}

// templateNotes reports whether a notes part holds only whitespace and a template heading and comment.
func templateNotes(notes []byte) bool {
	if notes == nil {
		return true
	}
	rest := string(bytes.TrimSpace(notes[len(notesHeading):]))
	return rest == "" || rest == notesComment || rest == oldNotesComment
}

// uninstallResolve applies --keep or --overwrite to a conflict; overwrite, when set, is what
// --overwrite plans instead of a plain removal.
func (r Resolutions) uninstallResolve(change, overwrite Change) (Change, error) {
	keep, overwritten := slices.Contains(r.Keep, change.Path), slices.Contains(r.Overwrite, change.Path)
	switch {
	case change.Reason == hookEntry && (keep || overwritten):
		return Change{}, r.uninstallInvalid(change.Path+" holds a hook entry, which uninstall cannot remove yet",
			"keep the installation until hook entry removal ships with the guard hooks")
	case !overwritten:
	case change.Reason == outside:
		return Change{}, r.uninstallInvalid(change.Path+" is "+outside, "remove it by hand or pass --keep "+change.Path)
	case change.Reason == noBoundary:
		return Change{}, r.uninstallInvalid(change.Path+` has no "## Project notes" boundary`,
			"restore the boundary by hand or pass --keep "+change.Path)
	case overwrite.Path != "":
		return overwrite, nil
	}
	return r.removal(change)
}

func (r Resolutions) uninstallInvalid(found, fix string) *Rejection {
	return &Rejection{Code: "invalid_value", Check: r.check(), Found: found, Fix: fix}
}
