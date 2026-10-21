package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
)

// Want is a path the CLI would write, its owner, and the offered content.
type Want struct {
	Path    string
	Owner   string
	Content []byte
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
		change, err := classify(root, m, want)
		if err != nil {
			return Plan{}, nil, err
		}
		if change.Action == ActionConflict {
			conflicts = append(conflicts, want.Path)
			change = resolve(change, want, keep, overwrite)
		}
		plan.Changes = append(plan.Changes, change)
		entry := File{Owner: want.Owner}
		if want.Owner != "seed" {
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

func resolve(change Change, want Want, keep, overwrite []string) Change {
	switch {
	case slices.Contains(keep, want.Path):
		return Change{Path: want.Path, Action: ActionKeep, Owner: want.Owner}
	case slices.Contains(overwrite, want.Path):
		return Change{Path: want.Path, Action: ActionUpdate, Owner: want.Owner, Content: want.Content}
	}
	return change
}

func classify(root *os.Root, m *Manifest, want Want) (Change, error) {
	change := Change{Path: want.Path, Owner: want.Owner}
	info, err := root.Lstat(want.Path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return offered(change, ActionCreate, want), nil
	case err != nil:
		return Change{}, fmt.Errorf("classify %s: %w", want.Path, err)
	case want.Owner == "seed":
		change.Action = ActionSkip
		return change, nil
	case !info.Mode().IsRegular():
		return conflict(change, "exists and is not a regular file"), nil
	}
	disk, err := root.ReadFile(want.Path)
	if err != nil {
		return Change{}, fmt.Errorf("classify %s: %w", want.Path, err)
	}
	base := ""
	if m != nil {
		base = m.Files[want.Path].SHA256
	}
	switch {
	case bytes.Equal(disk, want.Content):
		change.Action = ActionUnchanged
		if base == "" {
			change.Reason = "adopted"
		}
	case base == "":
		change = conflict(change, "exists and is not managed by solomonar")
	case Hash(disk) == base:
		change = offered(change, ActionUpdate, want)
	case Hash(want.Content) == base:
		change.Action = ActionKeep
	default:
		change = conflict(change, "modified since installation (manifest sha256 differs)")
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
