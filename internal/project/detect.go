package project

import (
	"path"
	"slices"
	"strings"
)

// Suggested maps each default or detected extension to its reason: the first matching file, else
// "default" (I section 4.3).
func (c Catalog) Suggested(files []string) map[string]string {
	sorted := slices.Sorted(slices.Values(files))
	suggested := map[string]string{}
	for name, e := range c {
		for _, file := range sorted {
			if slices.ContainsFunc(e.Detect, func(glob string) bool { return matchGlob(glob, file) }) {
				suggested[name] = file
				break
			}
		}
		if _, ok := suggested[name]; !ok && e.Default {
			suggested[name] = "default"
		}
	}
	return suggested
}

// matchGlob matches name segment by segment; "**" spans zero or more whole segments.
func matchGlob(glob, name string) bool {
	return matchSegments(strings.Split(glob, "/"), strings.Split(name, "/"))
}

func matchSegments(glob, name []string) bool {
	if len(glob) == 0 {
		return len(name) == 0
	}
	if glob[0] == "**" {
		for i := range len(name) + 1 {
			if matchSegments(glob[1:], name[i:]) {
				return true
			}
		}
		return false
	}
	if len(name) == 0 {
		return false
	}
	ok, err := path.Match(glob[0], name[0])
	return err == nil && ok && matchSegments(glob[1:], name[1:])
}

func validGlob(glob string) bool {
	if glob == "" || strings.HasPrefix(glob, "/") {
		return false
	}
	for segment := range strings.SplitSeq(glob, "/") {
		switch {
		case segment == "**":
		case segment == "" || segment == "." || segment == "..", strings.Contains(segment, "**"):
			return false
		default:
			if _, err := path.Match(segment, ""); err != nil {
				return false
			}
		}
	}
	return true
}
