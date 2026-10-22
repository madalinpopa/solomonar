package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
)

const fixChoices = "; init never changes choices"

// Requested holds the choices given to init; an empty string or a nil list or map was not given.
type Requested struct {
	Name, MainBranch, Remote string
	Runtimes, Extensions     []string
	AdversarialModels        map[string]string
}

// InitPlan plans init's files and builds the manifest, whose change comes last (I section 4.7).
func InitPlan(root *os.Root, existing *Manifest, wants []Want, keep, overwrite []string, p Project,
	runtimes map[string]Runtime, extensions map[string]Extension, decisions Decisions, framework string,
) (Plan, *Manifest, error) {
	plan, files, err := Classify(root, existing, wants, keep, overwrite)
	if err != nil {
		return Plan{}, nil, err
	}
	for i, c := range plan.Changes {
		if c.Path == "docs/README.md" && c.Action == ActionSkip && len(decisions.Index) > 0 {
			if plan.Changes[i], err = indexReadme(root, c, decisions.Index); err != nil {
				return Plan{}, nil, err
			}
		}
	}
	plan.Changes = append(plan.Changes, decisions.Changes...)
	maps.Copy(files, decisions.Files)
	files[ManifestPath] = File{Owner: "manifest"}
	if runtimes == nil {
		runtimes = map[string]Runtime{}
	}
	if extensions == nil {
		extensions = map[string]Extension{}
	}
	manifest := &Manifest{
		SchemaVersion:    ManifestSchemaVersion,
		FrameworkVersion: framework,
		Project:          p,
		Runtimes:         runtimes,
		Extensions:       extensions,
		Files:            files,
	}
	content, err := manifest.Encode()
	if err != nil {
		return Plan{}, nil, err
	}
	change := Change{Path: ManifestPath, Owner: "manifest", Action: ActionUpdate, Content: content}
	disk, err := root.ReadFile(ManifestPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		change.Action = ActionCreate
	case err != nil:
		return Plan{}, nil, fmt.Errorf("read manifest: %w", err)
	case bytes.Equal(disk, content):
		change = Change{Path: ManifestPath, Owner: "manifest", Action: ActionUnchanged}
	}
	plan.Changes = append(plan.Changes, change)
	return plan, manifest, nil
}

// indexReadme adds the index lines to a present README's documents section, or says where they go.
func indexReadme(root *os.Root, skip Change, index []string) (Change, error) {
	content, _, err := readDocument(root, skip.Path)
	if err != nil {
		return Change{}, err
	}
	if added, ok := addDocuments(content, index); ok {
		return Change{Path: skip.Path, Action: ActionAppend, Owner: skip.Owner, Content: added}, nil
	}
	skip.Reason = `exists; add under "` + documentsHeading + `": ` + strings.Join(index, "; ")
	return skip, nil
}

// CheckChoices rejects the first requested choice that differs from the manifest (UC-3).
func (m *Manifest) CheckChoices(r Requested) error {
	values := []struct{ field, flag, recorded, requested string }{
		{"name", "--name", m.Project.Name, r.Name},
		{"main_branch", "--main-branch", m.Project.MainBranch, r.MainBranch},
		{"remote", "--remote", m.Project.Remote, r.Remote},
	}
	for _, s := range values {
		if s.requested != "" && s.requested != s.recorded {
			return alreadyInitialized(fmt.Sprintf("manifest %s %q; requested %q", s.field, s.recorded, s.requested),
				"rerun without "+s.flag)
		}
	}
	lists := []struct {
		field     string
		recorded  []string
		requested []string
	}{
		{"runtimes", slices.Sorted(maps.Keys(m.Runtimes)), r.Runtimes},
		{"extensions", slices.Sorted(maps.Keys(m.Extensions)), r.Extensions},
	}
	for _, l := range lists {
		if l.requested == nil {
			continue
		}
		requested := slices.Compact(slices.Sorted(slices.Values(l.requested)))
		if !slices.Equal(requested, l.recorded) {
			return alreadyInitialized(
				fmt.Sprintf("manifest %s %s; requested %s", l.field, formatList(l.recorded), formatList(requested)),
				fmt.Sprintf("change %s with `solomonar %s add|remove`", l.field, strings.TrimSuffix(l.field, "s")))
		}
	}
	for _, runtime := range slices.Sorted(maps.Keys(r.AdversarialModels)) {
		recorded, requested := m.Runtimes[runtime].AdversarialModel, r.AdversarialModels[runtime]
		if requested == recorded {
			continue
		}
		shown := "none"
		if recorded != "" {
			shown = fmt.Sprintf("%q", recorded)
		}
		return alreadyInitialized(
			fmt.Sprintf("manifest adversarial_model %s %s; requested %q", runtime, shown, requested),
			"change models with `solomonar runtime add "+runtime+" --adversarial-model <model>`")
	}
	return nil
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func alreadyInitialized(found, fix string) *Rejection {
	return &Rejection{Code: "already_initialized", Found: found, Fix: fix + fixChoices}
}
