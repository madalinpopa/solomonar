package project

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
)

// StoreDir is the extension store, relative to the repository root (I section 4.3).
const StoreDir = "docs/solomonar/extensions"

// ProjectExtension is one project extension of the store; Err is the loader's first failure.
type ProjectExtension struct {
	CatalogExtension
	Err error
}

// LoadStore reads every project extension: a store directory that is not an installed catalog
// extension or its leftover, or a manifest entry of source path (I section 4.3).
func LoadStore(root, assets fs.FS, catalog Catalog, manifest *Manifest) (map[string]ProjectExtension, error) {
	entries, err := fs.ReadDir(root, StoreDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", StoreDir, err)
	}
	var installed map[string]Extension
	if manifest != nil {
		installed = manifest.Extensions
	}
	names := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() && installed[entry.Name()].Source != "catalog" && !leftover(root, catalog, entry.Name()) {
			names[entry.Name()] = true
		}
	}
	for name, e := range installed {
		if e.Source == "path" {
			names[name] = true
		}
	}
	store := map[string]ProjectExtension{}
	for name := range names {
		dir := path.Join(StoreDir, name)
		e, err := readExtension(root, dir, true)
		e.Name, e.Dir = name, dir
		if _, ok := catalog[name]; ok {
			err = fmt.Errorf("name %q is also a catalog extension", name)
		}
		if kind := CoreItem(assets, name); kind != "" {
			err = fmt.Errorf("name %q is a core %s", name, kind)
		}
		store[name] = ProjectExtension{CatalogExtension: e, Err: err}
	}
	for _, name := range slices.Sorted(maps.Keys(store)) {
		if e := store[name]; e.Err == nil {
			e.Err = checkProjectReferences(e.CatalogExtension, catalog, store)
			store[name] = e
		}
	}
	return store, nil
}

// leftover reports whether a store directory is what a catalog extension left behind: its name,
// without an extension.json.
func leftover(root fs.FS, catalog Catalog, name string) bool {
	_, inCatalog := catalog[name]
	_, err := fs.Stat(root, path.Join(StoreDir, name, "extension.json"))
	return inCatalog && errors.Is(err, fs.ErrNotExist)
}

// CoreItem returns skill or role when name is a core skill or role, else "" (I section 4.1).
func CoreItem(assets fs.FS, name string) string {
	if _, err := fs.Stat(assets, "framework/skills/"+name+"/SKILL.md"); err == nil {
		return "skill"
	}
	if _, err := fs.Stat(assets, "framework/agents/"+name+".md"); err == nil {
		return "role"
	}
	return ""
}

func checkProjectReferences(e CatalogExtension, catalog Catalog, store map[string]ProjectExtension) error {
	known := func(name string) bool {
		_, inCatalog := catalog[name]
		_, inStore := store[name]
		return inCatalog || inStore
	}
	for _, name := range e.Requires {
		if !known(name) {
			return fmt.Errorf("requires %q is not a known extension", name)
		}
	}
	for _, name := range e.Conflicts {
		if !known(name) {
			return fmt.Errorf("conflicts %q is not a known extension", name)
		}
	}
	return nil
}
