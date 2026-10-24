package project

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	documentsHeading = "## Documents"
	decisionsPath    = "docs/decisions.md"
	statusPrefix     = "- Status: "
)

// Decision is one approved decision a plan records (I section 5 UC-1).
type Decision struct {
	ID         string   `json:"id"`
	Group      string   `json:"group"`
	Selected   []string `json:"selected"`
	Supersedes string   `json:"-"`
}

// Selection joins the selected names, or is none for an emptied group.
func (d Decision) Selection() string {
	return cmp.Or(strings.Join(d.Selected, ", "), "none")
}

// Decider decides a changed group: the approved ID, or else the user's command (I section 4.3).
type Decider struct {
	Date, Command, Rationale, ID string
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

// PlanDecisions plans a decision for each stack group whose selection differs from the recorded one
// (I section 4.3).
func PlanDecisions(root *os.Root, existing *Manifest, installed []CatalogExtension, by Decider) (Decisions, error) {
	d := Decisions{Files: decisionFiles(existing), Records: []Decision{}, ByGroup: map[string]string{}}
	selection := stackSelection(installed)
	recorded, err := recordedSelection(root, existing)
	if err != nil {
		return Decisions{}, err
	}
	blocks, ids, superseded := map[string][]string{}, map[string][]string{}, map[string]string{}
	numbers := numbering{root: root, existing: existing}
	for _, group := range groups() {
		selected, old := selection[group], recorded[group]
		if len(selected) == 0 && len(old.names) == 0 {
			continue
		}
		if old.id != "" && slices.Equal(names(selected), old.names) {
			d.ByGroup[group] = old.id
			continue
		}
		record := Decision{ID: by.ID, Group: group, Selected: names(selected)}
		if by.ID == "" {
			id, err := numbers.next()
			if err != nil {
				return Decisions{}, err
			}
			record.ID, record.Supersedes = id, old.id
			blocks[decisionsPath] = append(blocks[decisionsPath], decisionEntry(record, selected, by))
			ids[decisionsPath] = append(ids[decisionsPath], record.ID)
			if old.id != "" {
				superseded[old.id] = record.ID
			}
		}
		d.Records = append(d.Records, record)
		d.ByGroup[group] = record.ID
		doc := decisionDocument(group)
		blocks[doc] = append(blocks[doc], fmt.Sprintf("- %s: %s (%s)\n", group, record.Selection(), record.ID))
		ids[doc] = append(ids[doc], record.ID)
	}
	for _, doc := range documents() {
		if err := d.planDocument(root, doc, blocks[doc.path], ids[doc.path], superseded); err != nil {
			return Decisions{}, err
		}
	}
	return d, nil
}

// numbering hands out P/D-n IDs above every one in docs/decisions.md and the manifest, reading the
// document only for the first one.
type numbering struct {
	root     *os.Root
	existing *Manifest
	last     int
	read     bool
}

func (n *numbering) next() (string, error) {
	if !n.read {
		highest, err := highestDecision(n.root)
		if err != nil {
			return "", err
		}
		n.last, n.read = max(highest, highestRecorded(n.existing)), true
	}
	n.last++
	return "P/D-" + strconv.Itoa(n.last), nil
}

func decisionDocument(group string) string {
	if group == "architecture" {
		return "docs/architecture.md"
	}
	return "docs/stack.md"
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

// decisionEntry returns the rules section 3 entry of a decision the user's command makes.
func decisionEntry(record Decision, selected []CatalogExtension, by Decider) string {
	var refs []string
	for _, e := range selected {
		refs = append(refs, storePath(e, extensionMain(e)))
	}
	references := quoted(refs)
	if record.Supersedes != "" {
		references = strings.TrimSuffix("supersedes "+record.Supersedes+"; "+references, "; ")
	}
	rationale := cmp.Or(by.Rationale, "selected by the user with `solomonar "+by.Command+"`")
	return fmt.Sprintf("## %s: %s\n\n- Status: approved\n- Selected option: %s\n- Rejected options: none\n"+
		"- Scope: project; %s extensions %s\n- Rationale: %s\n- Date: %s\n- References: %s\n",
		record.ID, decisionTitles()[record.Group], record.Selection(), record.Group,
		cmp.Or(quoted(record.Selected), "none"), rationale, by.Date, references)
}

func names(extensions []CatalogExtension) []string {
	out := make([]string, 0, len(extensions))
	for _, e := range extensions {
		out = append(out, e.Name)
	}
	return out
}

// planDocument appends blocks to a decision document, creating it with a title when absent, and marks
// the superseded entries, old ID to new.
func (d *Decisions) planDocument(root *os.Root, doc document, blocks, ids []string,
	superseded map[string]string,
) error {
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
	if doc.path == decisionsPath {
		var edited bool
		if content, edited = supersede(content, superseded); edited {
			change.Action = ActionUpdate
		}
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
	if len(items) == 0 {
		return ""
	}
	return "`" + strings.Join(items, "`, `") + "`"
}

type recorded struct {
	names []string
	id    string
}

// recordedSelection returns each stack group's manifest names, sorted, and the decision of the first
// one by name that has one (I section 4.5); an emptied group has the decision that recorded none.
func recordedSelection(root *os.Root, existing *Manifest) (map[string]recorded, error) {
	groups := map[string]recorded{}
	if existing == nil {
		return groups, nil
	}

	for _, name := range slices.Sorted(maps.Keys(existing.Extensions)) {
		e := existing.Extensions[name]
		if !stackGroup(e.Group) {
			continue
		}
		r := groups[e.Group]
		r.names = append(r.names, name)
		r.id = cmp.Or(r.id, e.Decision)
		groups[e.Group] = r
	}
	emptied, err := emptiedDecisions(root, existing)
	if err != nil {
		return nil, err
	}
	for group, id := range emptied {
		if _, installed := groups[group]; !installed {
			groups[group] = recorded{id: id}
		}
	}
	return groups, nil
}

// emptiedDecisions returns, for each group whose last line in its managed decision document records
// none, that line's decision.
func emptiedDecisions(root *os.Root, existing *Manifest) (map[string]string, error) {
	emptied := map[string]string{}
	for _, doc := range documents() {
		if existing.Files[doc.path].Owner != "decision" {
			continue
		}
		content, _, err := readDocument(root, doc.path)
		if err != nil {
			return nil, err
		}
		for line := range strings.SplitSeq(string(content), "\n") {
			item, listed := strings.CutPrefix(line, "- ")
			group, selection, ok := strings.Cut(item, ": ")
			if !listed || !ok || decisionDocument(group) != doc.path {
				continue
			}
			delete(emptied, group)
			id, none := strings.CutPrefix(selection, "none (")
			if id, closed := strings.CutSuffix(id, ")"); none && closed && ValidDecisionID(id) {
				emptied[group] = id
			}
		}
	}
	return emptied, nil
}

// supersede rewrites the Status line of each superseded entry; edited is false when none is present.
func supersede(content []byte, superseded map[string]string) (out []byte, edited bool) {
	if len(superseded) == 0 {
		return content, false
	}
	lines := strings.Split(string(content), "\n")
	for _, old := range slices.Sorted(maps.Keys(superseded)) {
		if at := statusLine(lines, old); at != -1 && at < len(lines) {
			lines[at] = statusPrefix + "superseded by " + superseded[old]
			edited = true
		}
	}
	return []byte(strings.Join(lines, "\n")), edited
}

// highestDecision returns the largest n of a "## P/D-<n>:" heading in docs/decisions.md, or 0.
func highestDecision(root *os.Root) (int, error) {
	content, _, err := readDocument(root, decisionsPath)
	if err != nil {
		return 0, err
	}
	highest := 0
	for line := range strings.SplitSeq(string(content), "\n") {
		rest, ok := strings.CutPrefix(line, "## ")
		id, _, colon := strings.Cut(rest, ":")
		if ok && colon {
			highest = max(highest, projectNumber(id))
		}
	}
	return highest, nil
}

// highestRecorded returns the largest n of a P/D-<n> recorded in the manifest.
func highestRecorded(existing *Manifest) int {
	highest := 0
	if existing == nil {
		return highest
	}
	for _, e := range existing.Extensions {
		highest = max(highest, projectNumber(e.Decision))
	}
	for _, f := range existing.Files {
		for _, id := range f.Entries {
			highest = max(highest, projectNumber(id))
		}
	}
	return highest
}

// projectNumber returns n of P/D-<n>, or 0 for another ID.
func projectNumber(id string) int {
	digits, ok := strings.CutPrefix(id, "P/D-")
	if !ok || digits == "" || !allDigits(digits) {
		return 0
	}
	n, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}
	return n
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

// DecisionStatus returns the first word of the Status of the "## <id>:" entry in docs/decisions.md.
func DecisionStatus(root *os.Root, id string) (status string, found bool, err error) {
	content, _, err := readDocument(root, decisionsPath)
	if err != nil {
		return "", false, err
	}
	lines := strings.Split(string(content), "\n")
	at := statusLine(lines, id)
	if at == -1 {
		return "", false, nil
	}
	if at < len(lines) {
		status, _, _ = strings.Cut(strings.TrimPrefix(lines[at], statusPrefix), " ")
	}
	return status, true, nil
}

// statusLine returns the index of the Status line of the id's entry, len(lines) when the entry has
// none, or -1 without the entry.
func statusLine(lines []string, id string) int {
	heading := slices.IndexFunc(lines, func(line string) bool { return strings.HasPrefix(line, "## "+id+":") })
	if heading == -1 {
		return -1
	}
	for i := heading + 1; i < len(lines) && !strings.HasPrefix(lines[i], "## "); i++ {
		if strings.HasPrefix(lines[i], statusPrefix) {
			return i
		}
	}
	return len(lines)
}
