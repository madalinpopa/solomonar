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
	Dir         string   `json:"-"`
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
		e, err := readExtension(assets, dir)
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

func readExtension(assets fs.FS, dir string) (CatalogExtension, error) {
	var e CatalogExtension
	data, err := fs.ReadFile(assets, path.Join(dir, "extension.json"))
	if err != nil {
		return e, err
	}
	if err := json.Unmarshal(data, &e, json.RejectUnknownMembers(true)); err != nil {
		return e, fmt.Errorf("decode extension.json: %w", err)
	}
	e.Dir = dir
	group, name := path.Base(path.Dir(dir)), path.Base(dir)
	switch {
	case e.Name != name:
		return e, fmt.Errorf("name %q does not match the directory", e.Name)
	case !extensionName(e.Name):
		return e, fmt.Errorf("name %q is not a name of [a-z][a-z0-9-]*", e.Name)
	case e.Group != group:
		return e, fmt.Errorf("group %q does not match the directory", e.Group)
	case !slices.Contains(groups(), e.Group):
		return e, fmt.Errorf("group %q is not one of %s", e.Group, strings.Join(groups(), ", "))
	case e.Kind != "skill" && e.Kind != "agent":
		return e, fmt.Errorf("kind %q is not skill or agent", e.Kind)
	}
	file := map[string]string{"skill": "SKILL.md", "agent": e.Name + ".md"}[e.Kind]
	if info, err := fs.Stat(assets, path.Join(dir, file)); err != nil || info.IsDir() {
		return e, fmt.Errorf("kind %s has no %s", e.Kind, file)
	}
	switch {
	case strings.TrimSpace(e.Title) == "":
		return e, errors.New("title is missing")
	case strings.TrimSpace(e.AppliesWhen) == "":
		return e, errors.New("applies_when is missing")
	}
	return e, nil
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
	for _, name := range selected {
		var missing []string
		for _, required := range slices.Sorted(slices.Values(c[name].Requires)) {
			if !slices.Contains(selected, required) {
				missing = append(missing, required)
			}
		}
		if len(missing) > 0 {
			return nil, &Rejection{
				Code:  "extension_requires",
				Check: "every required extension is selected",
				Found: name + " requires " + strings.Join(missing, ", "),
				Fix:   "add --ext " + strings.Join(missing, ","),
			}
		}
	}
	for _, name := range selected {
		for _, other := range slices.Sorted(slices.Values(c[name].Conflicts)) {
			if slices.Contains(selected, other) {
				return nil, &Rejection{
					Code:  "extension_conflict",
					Check: "no selected extensions conflict",
					Found: name + " conflicts with " + other,
					Fix:   "drop --ext " + name + " or --ext " + other,
				}
			}
		}
	}
	return selected, nil
}
