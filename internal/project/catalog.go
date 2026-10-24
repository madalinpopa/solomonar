package project

import (
	"cmp"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
)

// Catalog maps each embedded extension name to its metadata (I section 4.3).
type Catalog map[string]CatalogExtension

// CatalogExtension is one extension.json and the catalog directory that holds it.
type CatalogExtension struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Group       string   `json:"group"`
	Title       string   `json:"title"`
	Requires    []string `json:"requires"`
	Conflicts   []string `json:"conflicts"`
	AppliesWhen string   `json:"applies_when"`
	Paths       []string `json:"paths"`
	Detect      []string `json:"detect"`
	Default     bool     `json:"default"`
	Invocation  string   `json:"invocation"`
	Dir         string   `json:"-"`
	// Source is the manifest source, catalog or path; FS holds Dir.
	Source string `json:"-"`
	FS     fs.FS  `json:"-"`
}

// LoadCatalog reads and checks every extensions/<group>/<name>/extension.json in assets.
func LoadCatalog(assets fs.FS) (Catalog, error) {
	files, err := fs.Glob(assets, "extensions/*/*/extension.json")
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	catalog := Catalog{}
	for _, file := range files {
		dir := path.Dir(file)
		e, err := readExtension(assets, dir, false)
		if err == nil {
			if other, ok := catalog[e.Name]; ok {
				err = fmt.Errorf("name %q is also in %s", e.Name, other.Dir)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("read catalog %s: %w", dir, err)
		}
		catalog[e.Name] = e
	}
	for _, name := range slices.Sorted(maps.Keys(catalog)) {
		if err := catalog.checkReferences(catalog[name]); err != nil {
			return nil, fmt.Errorf("read catalog %s: %w", catalog[name].Dir, err)
		}
	}
	return catalog, nil
}

// readExtension is the one strict loader; only a project extension has its frontmatter checked.
func readExtension(assets fs.FS, dir string, project bool) (CatalogExtension, error) {
	var e CatalogExtension
	data, err := fs.ReadFile(assets, path.Join(dir, "extension.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return e, errors.New("extension.json is missing")
	}
	if err != nil {
		return e, err
	}
	if err := json.Unmarshal(data, &e, json.RejectUnknownMembers(true)); err != nil {
		return e, fmt.Errorf("decode extension.json: %w", err)
	}
	e.Dir, e.FS, e.Source = dir, assets, "catalog"
	if project {
		e.Source = "path"
	}
	group, name := path.Base(path.Dir(dir)), path.Base(dir)
	switch {
	case e.Name != name:
		return e, fmt.Errorf("name %q does not match the directory", e.Name)
	case !extensionName(e.Name):
		return e, fmt.Errorf("name %q is not a name of [a-z][a-z0-9-]*", e.Name)
	case !project && e.Group != group:
		return e, fmt.Errorf("group %q does not match the directory", e.Group)
	case !slices.Contains(groups(), e.Group):
		return e, fmt.Errorf("group %q is not one of %s", e.Group, strings.Join(groups(), ", "))
	case e.Kind != "skill" && e.Kind != "agent":
		return e, fmt.Errorf("kind %q is not skill or agent", e.Kind)
	}
	file := extensionMain(e)
	if info, err := fs.Stat(assets, path.Join(dir, file)); err != nil || info.IsDir() {
		return e, fmt.Errorf("kind %s has no %s", e.Kind, file)
	}
	switch {
	case strings.TrimSpace(e.Title) == "":
		return e, errors.New("title is missing")
	case strings.TrimSpace(e.AppliesWhen) == "":
		return e, errors.New("applies_when is missing")
	}
	for _, glob := range e.Detect {
		if !validGlob(glob) {
			return e, fmt.Errorf("detect %q is not a valid glob", glob)
		}
	}
	e.Invocation = cmp.Or(e.Invocation, "model")
	switch {
	case e.Invocation != "model" && e.Invocation != "user":
		return e, fmt.Errorf("invocation %q is not model or user", e.Invocation)
	case e.Invocation == "user" && e.Kind != "skill":
		return e, errors.New("invocation user is only for a skill")
	}
	if project {
		return e, checkFrontmatter(assets, path.Join(dir, file), name)
	}
	return e, nil
}

func checkFrontmatter(assets fs.FS, file, name string) error {
	data, err := fs.ReadFile(assets, file)
	if err != nil {
		return err
	}
	p, _ := parseFrontmatter(data)
	switch {
	case p.name != name:
		return fmt.Errorf("frontmatter name %q does not match the directory", p.name)
	case p.description == "":
		return errors.New("frontmatter description is missing")
	}
	return nil
}

func (c Catalog) checkReferences(e CatalogExtension) error {
	for _, name := range e.Requires {
		if _, ok := c[name]; !ok {
			return fmt.Errorf("requires %q is not a catalog extension", name)
		}
	}
	for _, name := range e.Conflicts {
		if _, ok := c[name]; !ok {
			return fmt.Errorf("conflicts %q is not a catalog extension", name)
		}
	}
	return nil
}

// Resolve returns the sorted unique names when all are known, complete, and free of conflicts.
func (c Catalog) Resolve(names []string) ([]string, error) {
	selected := slices.Compact(slices.Sorted(slices.Values(names)))
	for _, name := range selected {
		if _, ok := c[name]; !ok {
			known := cmp.Or(strings.Join(slices.Sorted(maps.Keys(c)), ", "), "none")
			return nil, &Rejection{
				Code:  "unknown_extension",
				Check: "--ext names a catalog extension",
				Found: fmt.Sprintf("%q is not in the catalog; known: %s", name, known),
				Fix:   "pass --ext with a known name, or none",
			}
		}
	}
	if name, missing := c.Unmet(selected); name != "" {
		return nil, &Rejection{
			Code:  "extension_requires",
			Check: "every required extension is selected",
			Found: name + " requires " + strings.Join(missing, ", "),
			Fix:   "add --ext " + strings.Join(missing, ","),
		}
	}
	if name, other := c.Conflict(selected); name != "" {
		return nil, &Rejection{
			Code:  "extension_conflict",
			Check: "no selected extensions conflict",
			Found: name + " conflicts with " + other,
			Fix:   "drop --ext " + name + " or --ext " + other,
		}
	}
	return selected, nil
}

// Unmet returns the first name whose requires are not all selected, and the sorted missing ones.
func (c Catalog) Unmet(selected []string) (name string, missing []string) {
	for _, name := range selected {
		for _, required := range slices.Sorted(slices.Values(c[name].Requires)) {
			if !slices.Contains(selected, required) {
				missing = append(missing, required)
			}
		}
		if len(missing) > 0 {
			return name, missing
		}
	}
	return "", nil
}

// Conflict returns the first selected pair whose first name declares a conflict with the second.
func (c Catalog) Conflict(selected []string) (name, other string) {
	for _, name := range selected {
		for _, other := range slices.Sorted(slices.Values(c[name].Conflicts)) {
			if slices.Contains(selected, other) {
				return name, other
			}
		}
	}
	return "", ""
}
