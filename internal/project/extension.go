package project

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// ExtensionWants returns the installed extensions' rendered files and manifest entries (I §4.3).
func ExtensionWants(assets fs.FS, installed []CatalogExtension, decisions map[string]string,
) ([]Want, map[string]Extension, error) {
	sorted := slices.SortedFunc(slices.Values(installed), func(a, b CatalogExtension) int {
		return cmp.Compare(a.Name, b.Name)
	})
	var names []string
	for _, e := range sorted {
		names = append(names, e.Name)
	}
	var wants []Want
	entries := map[string]Extension{}
	for _, e := range sorted {
		decision := decisions[e.Group]
		if !stackGroup(e.Group) {
			decision = ""
		} else if decision == "" {
			return nil, nil, fmt.Errorf("render %s: no decision for group %s", e.Name, e.Group)
		}
		files, err := extensionFiles(assets, e)
		if err != nil {
			return nil, nil, fmt.Errorf("read extension %s: %w", e.Name, err)
		}
		var with []string
		for _, file := range files {
			content, err := fs.ReadFile(assets, path.Join(e.Dir, file))
			if err != nil {
				return nil, nil, fmt.Errorf("read extension %s: %w", e.Name, err)
			}
			if file == extensionMain(e) {
				content, with = Render(content, e.Name, names, decision)
			}
			target := path.Join("docs/solomonar/skills", e.Name, file)
			if e.Kind == "agent" {
				target = path.Join("docs/solomonar/agents", file)
			}
			wants = append(wants, Want{Path: target, Owner: "ext:" + e.Name, Content: content})
		}
		entries[e.Name] = Extension{
			State: "installed", Kind: e.Kind, Group: e.Group, Source: "catalog", Decision: decision, RenderedWith: with,
		}
	}
	return wants, entries, nil
}

func stackGroup(group string) bool {
	return group != "workflow" && group != "review"
}

// extensionMain is the file Render processes and runtime stubs point at.
func extensionMain(e CatalogExtension) string {
	if e.Kind == "agent" {
		return e.Name + ".md"
	}
	return "SKILL.md"
}

// extensionFiles lists the files an extension installs, relative to its catalog directory, sorted.
func extensionFiles(assets fs.FS, e CatalogExtension) ([]string, error) {
	if e.Kind == "agent" {
		return []string{extensionMain(e)}, nil
	}
	var files []string
	err := fs.WalkDir(assets, e.Dir, func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if rel := strings.TrimPrefix(name, e.Dir+"/"); rel != "extension.json" {
			files = append(files, rel)
		}
		return nil
	})
	slices.Sort(files)
	return files, err
}

// extensionsBlock renders the AGENTS.md skill and agent lists, by group then name (I §4.6).
func extensionsBlock(installed []CatalogExtension) string {
	sorted := slices.SortedFunc(slices.Values(installed), func(a, b CatalogExtension) int {
		return cmp.Or(cmp.Compare(slices.Index(groups(), a.Group), slices.Index(groups(), b.Group)),
			cmp.Compare(a.Name, b.Name))
	})
	skills := []string{"- Skills"}
	agents := []string{"- Agents: an installed agent is a specialist reviewer, required by triage when its " +
		"\"applies when\" matches."}
	for _, e := range sorted {
		line := fmt.Sprintf("  - `%s` (%s): %s", e.Name, e.Group, e.AppliesWhen)
		if len(e.Paths) > 0 {
			line += "; paths: `" + strings.Join(e.Paths, "`, `") + "`"
		}
		if e.Kind == "agent" {
			agents = append(agents, line)
		} else {
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
	return strings.Join(slices.Concat(skills, agents), "\n")
}
