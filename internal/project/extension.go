package project

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// ExtensionWants returns the installed catalog extensions' store files, copied verbatim, and every
// installed extension's manifest entry (I section 4.3).
func ExtensionWants(installed []CatalogExtension, decisions map[string]string,
) ([]Want, map[string]Extension, error) {
	sorted := slices.SortedFunc(slices.Values(installed), func(a, b CatalogExtension) int {
		return cmp.Compare(a.Name, b.Name)
	})
	var wants []Want
	entries := map[string]Extension{}
	for _, e := range sorted {
		decision := decisions[e.Group]
		if !stackGroup(e.Group) {
			decision = ""
		} else if decision == "" {
			return nil, nil, fmt.Errorf("install %s: no decision for group %s", e.Name, e.Group)
		}
		entry := Extension{State: "installed", Kind: e.Kind, Group: e.Group, Source: e.Source, Decision: decision}
		if e.Source == "path" {
			entry.Path = e.Dir
			entries[e.Name] = entry
			continue
		}
		files, err := extensionFiles(e)
		if err != nil {
			return nil, nil, fmt.Errorf("read extension %s: %w", e.Name, err)
		}
		for _, file := range files {
			content, err := fs.ReadFile(e.FS, path.Join(e.Dir, file))
			if err != nil {
				return nil, nil, fmt.Errorf("read extension %s: %w", e.Name, err)
			}
			wants = append(wants, Want{Path: storePath(e, file), Owner: "ext:" + e.Name, Content: content})
		}
		entries[e.Name] = entry
	}
	return wants, entries, nil
}

func stackGroup(group string) bool {
	return group != "workflow" && group != "review"
}

// extensionMain is the file runtime stubs and decisions point at.
func extensionMain(e CatalogExtension) string {
	if e.Kind == "agent" {
		return e.Name + ".md"
	}
	return "SKILL.md"
}

// storePath is where an extension's catalog file is installed.
func storePath(e CatalogExtension, file string) string {
	return path.Join(StoreDir, e.Name, file)
}

// extensionFiles lists every file of an extension's catalog directory, relative to it, sorted.
func extensionFiles(e CatalogExtension) ([]string, error) {
	var files []string
	err := fs.WalkDir(e.FS, e.Dir, func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		files = append(files, strings.TrimPrefix(name, e.Dir+"/"))
		return nil
	})
	slices.Sort(files)
	return files, err
}

// extensionsBlock renders the AGENTS.md skill, agent, and user-invoked skill lists, by group then name
// (I section 4.6).
func extensionsBlock(installed []CatalogExtension, decisions map[string]string) string {
	sorted := slices.SortedFunc(slices.Values(installed), func(a, b CatalogExtension) int {
		return cmp.Or(cmp.Compare(slices.Index(groups(), a.Group), slices.Index(groups(), b.Group)),
			cmp.Compare(a.Name, b.Name))
	})
	skills := []string{"- Skills"}
	agents := []string{"- Agents: an installed agent is a specialist reviewer, required by triage when its " +
		"\"applies when\" matches."}
	users := []string{"- User-invoked skills: load one only when the user starts it, never because it applies."}
	for _, e := range sorted {
		label := e.Group
		if stackGroup(e.Group) {
			label += ", " + decisions[e.Group]
		}
		line := fmt.Sprintf("  - `%s` (%s): %s", e.Name, label, e.AppliesWhen)
		if len(e.Paths) > 0 {
			line += "; paths: `" + strings.Join(e.Paths, "`, `") + "`"
		}
		switch {
		case e.Kind == "agent":
			agents = append(agents, line)
		case e.Invocation == "user":
			users = append(users, line)
		default:
			skills = append(skills, line)
		}
	}
	if len(skills) == 1 {
		skills = []string{"- Skills: none."}
	}
	if len(agents) == 1 {
		agents = []string{"- Agents: none. An installed agent is a specialist reviewer, required by triage when its " +
			"\"applies when\" matches."}
	}
	if len(users) == 1 {
		users = nil
	}
	return strings.Join(slices.Concat(skills, agents, users), "\n")
}
