package project

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
)

const (
	adoptedNotes = `adopted; existing content moved under "## Project notes"`
	notRegular   = "exists and is not a regular file"
)

// Adopted reports whether the change moves an unmanaged file's content under "## Project notes".
func (c Change) Adopted() bool { return c.Reason == adoptedNotes }

// Want is a path the CLI would write, its owner, and the offered content.
type Want struct {
	Path    string
	Owner   string
	Content []byte
	// CreateOnly leaves a present unmanaged file alone and unrecorded (CLI-D3).
	CreateOnly bool
}

// Resolutions are the --keep and --overwrite paths; OverwriteFlag names --overwrite in rejections,
// --take-new for upgrade.
type Resolutions struct {
	Keep, Overwrite []string
	OverwriteFlag   string
}

func (r Resolutions) overwriteFlag() string {
	return cmp.Or(r.OverwriteFlag, "--overwrite")
}

// Classify plans the wanted paths, then the managed paths no longer wanted (I section 4.5),
// and returns the new manifest files entries.
func Classify(root *os.Root, m *Manifest, wants []Want, r Resolutions) (Plan, map[string]File, error) {
	if err := r.distinct(); err != nil {
		return Plan{}, nil, err
	}
	var plan Plan
	var conflicts []string
	files := make(map[string]File, len(wants))
	for _, want := range wants {
		if slices.ContainsFunc(plan.Changes, func(c Change) bool { return c.Path == want.Path }) {
			return Plan{}, nil, fmt.Errorf("plan %s: wanted twice", want.Path)
		}
		change, written, err := classify(root, m, want)
		if err != nil {
			return Plan{}, nil, err
		}
		if change.Reason == adoptedNotes && slices.Contains(r.Keep, want.Path) {
			return Plan{}, nil, &Rejection{
				Code:  "invalid_value",
				Check: r.check(),
				Found: want.Path + " cannot be kept",
				Fix:   "adopt it (default) or pass " + r.overwriteFlag() + " " + want.Path,
			}
		}
		if change.Action == ActionConflict || change.Reason == adoptedNotes {
			conflicts = append(conflicts, want.Path)
			change = r.resolve(change, written)
		}
		plan.Changes = append(plan.Changes, change)
		if !want.CreateOnly || change.Action != ActionSkip {
			files[want.Path] = fileEntry(want)
		}
	}
	removals, removed, err := r.planRemovals(root, m, wants)
	if err != nil {
		return Plan{}, nil, err
	}
	for _, change := range removals {
		if change.Action == ActionConflict {
			files[change.Path] = m.Files[change.Path]
		}
	}
	conflicts = append(conflicts, removed...)
	plan.Changes = append(plan.Changes, removals...)
	if err := r.resolved(conflicts); err != nil {
		return Plan{}, nil, err
	}
	return plan, files, nil
}

// fileEntry is the manifest entry of a wanted path: its owner and the hash of its managed content.
func fileEntry(want Want) File {
	entry := File{Owner: want.Owner}
	switch want.Owner {
	case "seed":
	case "core+notes":
		managed, _ := splitNotes(want.Content)
		entry.SHA256 = Hash(managed)
	default:
		entry.SHA256 = Hash(want.Content)
	}
	return entry
}

// planRemovals plans the managed paths no longer wanted with their resolutions, and returns the
// conflicting paths.
func (r Resolutions) planRemovals(root *os.Root, m *Manifest, wants []Want) ([]Change, []string, error) {
	removals, err := classifyRemovals(root, m, wants)
	if err != nil {
		return nil, nil, err
	}
	var conflicts []string
	for i, change := range removals {
		if change.Action != ActionConflict {
			continue
		}
		conflicts = append(conflicts, change.Path)
		if removals[i], err = r.removal(change); err != nil {
			return nil, nil, err
		}
	}
	return removals, conflicts, nil
}

// distinct rejects the first path in both --keep and --overwrite.
func (r Resolutions) distinct() error {
	for _, path := range r.Keep {
		if slices.Contains(r.Overwrite, path) {
			return r.invalid("%s is in both --keep and %s", path, r.overwriteFlag())
		}
	}
	return nil
}

// resolved rejects the first --keep or --overwrite path that is not a conflict of the plan.
func (r Resolutions) resolved(conflicts []string) error {
	for _, path := range slices.Concat(r.Keep, r.Overwrite) {
		if !slices.Contains(conflicts, path) {
			return r.invalid("%s is not a conflict", path)
		}
	}
	return nil
}

// resolve applies --keep or --overwrite to a conflict; an overwrite writes written.
func (r Resolutions) resolve(change Change, written []byte) Change {
	switch {
	case slices.Contains(r.Keep, change.Path):
		return Change{Path: change.Path, Action: ActionKeep, Owner: change.Owner}
	case slices.Contains(r.Overwrite, change.Path):
		return Change{Path: change.Path, Action: ActionUpdate, Owner: change.Owner, Content: written}
	}
	return change
}

// removal applies --keep (untrack) or --overwrite (delete) to a removal conflict; a path that
// is not a regular file cannot be deleted.
func (r Resolutions) removal(change Change) (Change, error) {
	switch {
	case slices.Contains(r.Keep, change.Path):
		return Change{Path: change.Path, Action: ActionKeep, Owner: change.Owner, Reason: "untracked; left in place"}, nil
	case !slices.Contains(r.Overwrite, change.Path):
		return change, nil
	case change.Reason == notRegular:
		return Change{}, &Rejection{
			Code:  "invalid_value",
			Check: r.check(),
			Found: change.Path + " is not a regular file",
			Fix:   "remove it by hand or pass --keep " + change.Path,
		}
	}
	return Change{Path: change.Path, Action: ActionRemove, Owner: change.Owner}, nil
}

// classifyRemovals plans the managed paths that no want names, in path order; absent ones are dropped.
func classifyRemovals(root *os.Root, m *Manifest, wants []Want) ([]Change, error) {
	if m == nil {
		return nil, nil
	}
	var changes []Change
	for _, path := range slices.Sorted(maps.Keys(m.Files)) {
		file := m.Files[path]
		wanted := slices.ContainsFunc(wants, func(w Want) bool { return w.Path == path })
		if file.SHA256 == "" || wanted {
			continue
		}
		change := Change{Path: path, Owner: file.Owner}
		info, err := root.Lstat(path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return nil, fmt.Errorf("classify %s: %w", path, err)
		case !info.Mode().IsRegular():
			changes = append(changes, conflict(change, notRegular))
			continue
		}
		disk, err := root.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("classify %s: %w", path, err)
		}
		if Hash(disk) == file.SHA256 {
			change.Action = ActionRemove
		} else {
			change = conflict(change, "modified since installation (manifest sha256 differs)")
		}
		changes = append(changes, change)
	}
	return changes, nil
}

// classify plans one wanted path and returns the content an update or --overwrite writes.
func classify(root *os.Root, m *Manifest, want Want) (Change, []byte, error) {
	change := Change{Path: want.Path, Owner: want.Owner}
	base := ""
	if m != nil {
		base = m.Files[want.Path].SHA256
	}
	info, err := root.Lstat(want.Path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return offered(change, ActionCreate, want), want.Content, nil
	case err != nil:
		return Change{}, nil, fmt.Errorf("classify %s: %w", want.Path, err)
	case want.Owner == "seed":
		change.Action = ActionSkip
		return change, want.Content, nil
	case want.CreateOnly && base == "":
		change, err := skipUnmanaged(root, change, info.Mode().IsRegular())
		return change, want.Content, err
	case !info.Mode().IsRegular():
		return conflict(change, notRegular), want.Content, nil
	}
	disk, err := root.ReadFile(want.Path)
	if err != nil {
		return Change{}, nil, fmt.Errorf("classify %s: %w", want.Path, err)
	}
	if want.Owner == "core+notes" {
		change, written := classifyNotes(change, want.Content, disk, base)
		return change, written, nil
	}
	return compare(change, disk, want.Content, want.Content, base), want.Content, nil
}

// classifyNotes classifies the managed part of AGENTS.md and keeps or adopts the project's notes;
// an overwrite of an adopted file writes the full offer, of a managed one keeps its notes.
func classifyNotes(change Change, content, disk []byte, base string) (Change, []byte) {
	offer, offerNotes := splitNotes(content)
	managed, notes := splitNotes(disk)
	if base == "" && !bytes.Equal(managed, offer) {
		change.Action = ActionUpdate
		change.Reason = adoptedNotes
		change.Content = slices.Concat(offer, []byte(notesHeading+"\n\n"), disk)
		return change, content
	}
	if notes == nil {
		notes = offerNotes
	}
	written := slices.Concat(offer, notes)
	return compare(change, managed, offer, written, base), written
}

// compare applies the I section 4.5 rules to disk, offer, and base; an update writes written.
func compare(change Change, disk, offer, written []byte, base string) Change {
	switch {
	case bytes.Equal(disk, offer):
		change.Action = ActionUnchanged
		if base == "" {
			change.Reason = "adopted"
		}
	case base == "":
		change = conflict(change, "exists and is not managed by solomonar")
	case Hash(disk) == base:
		change.Action = ActionUpdate
		change.Content = written
	case Hash(offer) == base:
		change.Action = ActionKeep
	default:
		change = conflict(change, "modified since installation (manifest sha256 differs)")
	}
	return change
}

// skipUnmanaged skips an unmanaged create-only file and says whether it imports AGENTS.md.
func skipUnmanaged(root *os.Root, change Change, regular bool) (Change, error) {
	change.Action = ActionSkip
	change.Reason = "exists; add this line so Claude Code loads the framework:  @AGENTS.md"
	if !regular {
		return change, nil
	}
	disk, err := root.ReadFile(change.Path)
	if err != nil {
		return Change{}, fmt.Errorf("classify %s: %w", change.Path, err)
	}
	for line := range strings.SplitSeq(string(disk), "\n") {
		if line == "@AGENTS.md" {
			change.Reason = "exists and imports AGENTS.md"
		}
	}
	return change, nil
}

func offered(change Change, action Action, want Want) Change {
	change.Action = action
	change.Content = want.Content
	return change
}

func conflict(change Change, reason string) Change {
	change.Action = ActionConflict
	change.Reason = reason
	return change
}

func (r Resolutions) check() string {
	return "--keep and " + r.overwriteFlag() + " resolve conflicts"
}

func (r Resolutions) invalid(format string, args ...any) *Rejection {
	return &Rejection{
		Code:  "invalid_value",
		Check: r.check(),
		Found: fmt.Sprintf(format, args...),
		Fix:   "pass --keep or " + r.overwriteFlag() + " once per conflicting path",
	}
}
