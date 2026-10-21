package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
)

const adoptedNotes = `adopted; existing content moved under "## Project notes"`

// Want is a path the CLI would write, its owner, and the offered content.
type Want struct {
	Path    string
	Owner   string
	Content []byte
	// CreateOnly leaves a present unmanaged file alone and unrecorded (CLI-D3).
	CreateOnly bool
}

// Classify plans the wanted paths (I §4.5) and returns their manifest files entries.
func Classify(root *os.Root, m *Manifest, wants []Want, keep, overwrite []string) (Plan, map[string]File, error) {
	for _, path := range keep {
		if slices.Contains(overwrite, path) {
			return Plan{}, nil, invalidResolution("%s is in both --keep and --overwrite", path)
		}
	}
	var plan Plan
	var conflicts []string
	files := make(map[string]File, len(wants))
	for _, want := range wants {
		change, written, err := classify(root, m, want)
		if err != nil {
			return Plan{}, nil, err
		}
		if change.Reason == adoptedNotes && slices.Contains(keep, want.Path) {
			return Plan{}, nil, &Rejection{
				Code:  "invalid_value",
				Check: "--keep and --overwrite resolve conflicts",
				Found: want.Path + " cannot be kept",
				Fix:   "adopt it (default) or pass --overwrite " + want.Path,
			}
		}
		if change.Action == ActionConflict || change.Reason == adoptedNotes {
			conflicts = append(conflicts, want.Path)
			change = resolve(change, written, keep, overwrite)
		}
		plan.Changes = append(plan.Changes, change)
		if want.CreateOnly && change.Action == ActionSkip {
			continue
		}
		entry := File{Owner: want.Owner}
		switch want.Owner {
		case "seed":
		case "core+notes":
			managed, _ := splitNotes(want.Content)
			entry.SHA256 = Hash(managed)
		default:
			entry.SHA256 = Hash(want.Content)
		}
		files[want.Path] = entry
	}
	for _, path := range slices.Concat(keep, overwrite) {
		if !slices.Contains(conflicts, path) {
			return Plan{}, nil, invalidResolution("%s is not a conflict", path)
		}
	}
	return plan, files, nil
}

// resolve applies --keep or --overwrite to a conflict; an overwrite writes written.
func resolve(change Change, written []byte, keep, overwrite []string) Change {
	switch {
	case slices.Contains(keep, change.Path):
		return Change{Path: change.Path, Action: ActionKeep, Owner: change.Owner}
	case slices.Contains(overwrite, change.Path):
		return Change{Path: change.Path, Action: ActionUpdate, Owner: change.Owner, Content: written}
	}
	return change
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
		return conflict(change, "exists and is not a regular file"), want.Content, nil
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

// compare applies the I §4.5 rules to disk, offer, and base; an update writes written.
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

func invalidResolution(format string, args ...any) *Rejection {
	return &Rejection{
		Code:  "invalid_value",
		Check: "--keep and --overwrite resolve conflicts",
		Found: fmt.Sprintf(format, args...),
		Fix:   "pass --keep or --overwrite once per conflicting path",
	}
}
