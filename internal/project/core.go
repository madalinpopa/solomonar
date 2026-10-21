package project

import (
	"bytes"
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"text/template"
)

const (
	notesHeading    = "## Project notes"
	emptyExtensions = "- Skills: none.\n- Agents: none. An installed agent is a specialist reviewer, " +
		"required by triage when its \"applies when\" matches."
)

// Choices are the init answers the core templates render.
type Choices struct {
	Name             string
	Objective        string
	FrameworkVersion string
}

// CoreWants returns the framework files, AGENTS.md, and docs/README.md (I §4.1).
func CoreWants(assets fs.FS, c Choices) ([]Want, error) {
	var wants []Want
	err := fs.WalkDir(assets, "framework", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := fs.ReadFile(assets, name)
		target := path.Join("docs/solomonar", strings.TrimPrefix(name, "framework/"))
		wants = append(wants, Want{Path: target, Owner: "core", Content: content})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read framework files: %w", err)
	}
	slices.SortFunc(wants, func(a, b Want) int { return cmp.Compare(a.Path, b.Path) })
	data := map[string]string{
		"FrameworkVersion": c.FrameworkVersion,
		"Extensions":       emptyExtensions,
		"ProjectName":      c.Name,
		"Objective":        c.Objective,
	}
	agents, err := render(assets, "templates/AGENTS.md", data)
	if err != nil {
		return nil, err
	}
	readme, err := render(assets, "templates/docs/README.md", data)
	if err != nil {
		return nil, err
	}
	return append(wants,
		Want{Path: "AGENTS.md", Owner: "core+notes", Content: agents},
		Want{Path: "docs/README.md", Owner: "seed", Content: readme},
	), nil
}

func render(assets fs.FS, name string, data map[string]string) ([]byte, error) {
	tmpl, err := template.ParseFS(assets, name)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return out.Bytes(), nil
}

// splitNotes splits AGENTS.md before the line "## Project notes"; notes is nil without that line.
func splitNotes(content []byte) (managed, notes []byte) {
	for offset := 0; ; {
		i := bytes.Index(content[offset:], []byte(notesHeading))
		if i < 0 {
			return content, nil
		}
		start, end := offset+i, offset+i+len(notesHeading)
		lineStart := start == 0 || content[start-1] == '\n'
		lineEnd := end == len(content) || content[end] == '\n'
		if lineStart && lineEnd {
			return content[:start], content[start:]
		}
		offset = end
	}
}
