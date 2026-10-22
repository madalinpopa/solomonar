package project

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	documentsHeading = "## Documents"
	decisionsPath    = "docs/decisions.md"
)

// Decision is one approved decision a plan records (I section 5 UC-1).
type Decision struct {
	ID       string   `json:"id"`
	Group    string   `json:"group"`
	Selected []string `json:"selected"`
}

// Decisions is the planned decision entries, their documents, and the README lines for created ones.
type Decisions struct {
	Changes []Change
	Files   map[string]File
	Records []Decision
	ByGroup map[string]string
	Index   []string
}

type document struct{ path, title, index, separator string }

func documents() []document {
	return []document{
		{decisionsPath, "# Decisions", "- `docs/decisions.md`: decision entries and their status", "\n"},
		{
			"docs/stack.md", "# Stack",
			"- `docs/stack.md`: technologies and practices, with the decisions that adopted them", "",
		},
		{
			"docs/architecture.md", "# Architecture",
			"- `docs/architecture.md`: structure and contracts, with the decisions that adopted them", "",
		},
	}
}

func decisionTitles() map[string]string {
	return map[string]string{
		"language": "Programming languages", "architecture": "Architecture style",
		"frontend": "Frontend technologies", "backend": "Backend technologies",
		"persistence": "Persistence technologies", "testing": "Testing practice", "operations": "Build and deployment",
	}
}

// PlanDecisions plans one approved entry per stack group without a recorded decision (I section 4.3).
func PlanDecisions(root *os.Root, existing *Manifest, installed []CatalogExtension, date string) (Decisions, error) {
	d := Decisions{Files: decisionFiles(existing), Records: []Decision{}, ByGroup: map[string]string{}}
	selection := stackSelection(installed)
	blocks, ids := map[string][]string{}, map[string][]string{}
	next := 0
	for _, group := range groups() {
		selected := selection[group]
		if len(selected) == 0 {
			continue
		}
		if id := recordedDecision(existing, selected); id != "" {
			d.ByGroup[group] = id
			continue
		}
		if next == 0 {
			highest, err := highestDecision(root)
			if err != nil {
				return Decisions{}, err
			}
			next = highest + 1
		}
		record, entry, line := newDecision("P/D-"+strconv.Itoa(next), group, selected, date)
		next++
		d.Records = append(d.Records, record)
		d.ByGroup[group] = record.ID
		doc := "docs/stack.md"
		if group == "architecture" {
			doc = "docs/architecture.md"
		}
		blocks[decisionsPath] = append(blocks[decisionsPath], entry)
		blocks[doc] = append(blocks[doc], line)
		ids[decisionsPath] = append(ids[decisionsPath], record.ID)
		ids[doc] = append(ids[doc], record.ID)
	}
	for _, doc := range documents() {
		if err := d.planDocument(root, doc, blocks[doc.path], ids[doc.path]); err != nil {
			return Decisions{}, err
		}
	}
	return d, nil
}

// decisionFiles carries the manifest's decision documents over, unread (re-run, UC-3).
func decisionFiles(existing *Manifest) map[string]File {
	files := map[string]File{}
	if existing != nil {
		for name, f := range existing.Files {
			if f.Owner == "decision" {
				files[name] = f
			}
		}
	}
	return files
}

// stackSelection groups the installed stack-group extensions by group, each sorted by name.
func stackSelection(installed []CatalogExtension) map[string][]CatalogExtension {
	selection := map[string][]CatalogExtension{}
	for _, e := range slices.SortedFunc(slices.Values(installed), func(a, b CatalogExtension) int {
		return cmp.Compare(a.Name, b.Name)
	}) {
		if stackGroup(e.Group) {
			selection[e.Group] = append(selection[e.Group], e)
		}
	}
	return selection
}

// newDecision returns a decision, its rules section 3 entry, and its group document line.
func newDecision(id, group string, selected []CatalogExtension, date string) (record Decision, entry, line string) {
	names := make([]string, 0, len(selected))
	var refs []string
	for _, e := range selected {
		names = append(names, e.Name)
		refs = append(refs, installedPath(e, extensionMain(e)))
	}
	entry = fmt.Sprintf("## %s: %s\n\n- Status: approved\n- Selected option: %s\n- Rejected options: none\n"+
		"- Scope: project; %s extensions %s\n- Rationale: selected by the user with `solomonar init`\n"+
		"- Date: %s\n- References: %s\n",
		id, decisionTitles()[group], strings.Join(names, ", "), group, quoted(names), date, quoted(refs))
	line = fmt.Sprintf("- %s: %s (%s)\n", group, strings.Join(names, ", "), id)
	return Decision{ID: id, Group: group, Selected: names}, entry, line
}

// planDocument appends blocks to a decision document, creating it with a title when absent.
func (d *Decisions) planDocument(root *os.Root, doc document, blocks, ids []string) error {
	carried, recorded := d.Files[doc.path]
	if len(blocks) == 0 {
		if recorded {
			d.Changes = append(d.Changes, Change{Path: doc.path, Action: ActionUnchanged, Owner: "decision"})
		}
		return nil
	}
	content, present, err := readDocument(root, doc.path)
	if err != nil {
		return err
	}
	change := Change{Path: doc.path, Action: ActionAppend, Owner: "decision"}
	if !present {
		change.Action = ActionCreate
		content = []byte(doc.title + "\n\n")
		d.Index = append(d.Index, doc.index)
	}
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		content = append(content, '\n')
	}
	if doc.separator != "" && len(content) > 0 && !bytes.HasSuffix(content, []byte("\n\n")) {
		content = append(content, '\n')
	}
	change.Content = slices.Concat(content, []byte(strings.Join(blocks, doc.separator)))
	d.Changes = append(d.Changes, change)
	d.Files[doc.path] = File{Owner: "decision", Entries: slices.Concat(carried.Entries, ids)}
	return nil
}

func quoted(items []string) string {
	return "`" + strings.Join(items, "`, `") + "`"
}

func recordedDecision(existing *Manifest, selected []CatalogExtension) string {
	if existing == nil {
		return ""
	}
	for _, e := range selected {
		if decision := existing.Extensions[e.Name].Decision; decision != "" {
			return decision
		}
	}
	return ""
}

// highestDecision returns the largest n of a "## P/D-<n>:" heading in docs/decisions.md, or 0.
func highestDecision(root *os.Root) (int, error) {
	content, _, err := readDocument(root, decisionsPath)
	if err != nil {
		return 0, err
	}
	highest := 0
	for line := range strings.SplitSeq(string(content), "\n") {
		rest, ok := strings.CutPrefix(line, "## P/D-")
		digits, _, colon := strings.Cut(rest, ":")
		if !ok || !colon || !allDigits(digits) {
			continue
		}
		if n, err := strconv.Atoi(digits); err == nil {
			highest = max(highest, n)
		}
	}
	return highest, nil
}

// readDocument returns a regular file's content; present is false when the path is absent.
func readDocument(root *os.Root, name string) (content []byte, present bool, err error) {
	info, err := root.Lstat(name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, false, nil
	case err != nil:
		return nil, false, fmt.Errorf("read %s: %w", name, err)
	case !info.Mode().IsRegular():
		return nil, false, fmt.Errorf("read %s: not a regular file", name)
	}
	content, err = root.ReadFile(name)
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", name, err)
	}
	return content, true, nil
}

// addDocuments inserts lines into the "## Documents" section after its last item; ok is false without
// that section.
func addDocuments(content []byte, lines []string) (added []byte, ok bool) {
	all := strings.Split(string(content), "\n")
	heading := slices.Index(all, documentsHeading)
	if heading < 0 {
		return content, false
	}
	at := heading + 1
	for i := heading + 1; i < len(all) && !strings.HasPrefix(all[i], "## "); i++ {
		if strings.HasPrefix(all[i], "- ") {
			at = i + 1
		}
	}
	if at == heading+1 {
		if at < len(all)-1 && all[at] == "" {
			at++
		} else {
			lines = slices.Concat([]string{""}, lines)
		}
		if at < len(all)-1 {
			lines = slices.Concat(lines, []string{""})
		}
	}
	return []byte(strings.Join(slices.Insert(all, at, lines...), "\n")), true
}
