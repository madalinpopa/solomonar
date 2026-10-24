package project

import (
	"cmp"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	decisionDate   = "2026-10-21"
	decisionsIndex = "- `docs/decisions.md`: decision entries and their status"
	stackIndex     = "- `docs/stack.md`: technologies and practices, with the decisions that adopted them"
	archIndex      = "- `docs/architecture.md`: structure and contracts, with the decisions that adopted them"
)

func byInit() Decider { return Decider{Date: decisionDate, Command: "init"} }

func selection() []CatalogExtension {
	return []CatalogExtension{
		{Name: "templ", Kind: "skill", Group: "frontend"},
		{Name: "tdd", Kind: "skill", Group: "workflow"},
		{Name: "htmx", Kind: "skill", Group: "frontend"},
		{Name: "contrast", Kind: "agent", Group: "review"},
		{Name: "hexagonal", Kind: "skill", Group: "architecture"},
		{Name: "go", Kind: "skill", Group: "language"},
		{Name: "perf", Kind: "agent", Group: "operations"},
	}
}

func entryText(id, title, group, names, scope, refs string) string {
	return "## " + id + ": " + title + "\n\n- Status: approved\n- Selected option: " + names +
		"\n- Rejected options: none\n- Scope: project; " + group + " extensions " + scope +
		"\n- Rationale: selected by the user with `solomonar init`\n- Date: " + decisionDate +
		"\n- References: " + refs + "\n"
}

func freshEntries(first, second, third, fourth string) string {
	return entryText(first, "Programming languages", "language", "go", "`go`",
		"`docs/solomonar/extensions/go/SKILL.md`") + "\n" +
		entryText(second, "Architecture style", "architecture", "hexagonal", "`hexagonal`",
			"`docs/solomonar/extensions/hexagonal/SKILL.md`") + "\n" +
		entryText(third, "Frontend technologies", "frontend", "htmx, templ", "`htmx`, `templ`",
			"`docs/solomonar/extensions/htmx/SKILL.md`, `docs/solomonar/extensions/templ/SKILL.md`") + "\n" +
		entryText(fourth, "Build and deployment", "operations", "perf", "`perf`",
			"`docs/solomonar/extensions/perf/perf.md`")
}

func TestPlanDecisionsRecordsOneEntryPerStackGroupInGroupOrder(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, nil)

	got, err := PlanDecisions(root, nil, selection(), byInit())

	require.NoError(t, err)
	assert.Equal(t, []Change{
		{
			Path: "docs/decisions.md", Action: ActionCreate, Owner: "decision",
			Content: []byte("# Decisions\n\n" + freshEntries("P/D-1", "P/D-2", "P/D-3", "P/D-4")),
		},
		{
			Path: "docs/stack.md", Action: ActionCreate, Owner: "decision",
			Content: []byte("# Stack\n\n- language: go (P/D-1)\n- frontend: htmx, templ (P/D-3)\n" +
				"- operations: perf (P/D-4)\n"),
		},
		{
			Path: "docs/architecture.md", Action: ActionCreate, Owner: "decision",
			Content: []byte("# Architecture\n\n- architecture: hexagonal (P/D-2)\n"),
		},
	}, got.Changes)
	assert.Equal(t, map[string]File{
		"docs/decisions.md":    {Owner: "decision", Entries: []string{"P/D-1", "P/D-2", "P/D-3", "P/D-4"}},
		"docs/stack.md":        {Owner: "decision", Entries: []string{"P/D-1", "P/D-3", "P/D-4"}},
		"docs/architecture.md": {Owner: "decision", Entries: []string{"P/D-2"}},
	}, got.Files)
	assert.Equal(t, []Decision{
		{ID: "P/D-1", Group: "language", Selected: []string{"go"}},
		{ID: "P/D-2", Group: "architecture", Selected: []string{"hexagonal"}},
		{ID: "P/D-3", Group: "frontend", Selected: []string{"htmx", "templ"}},
		{ID: "P/D-4", Group: "operations", Selected: []string{"perf"}},
	}, got.Records)
	assert.Equal(t, map[string]string{
		"language": "P/D-1", "architecture": "P/D-2", "frontend": "P/D-3", "operations": "P/D-4",
	}, got.ByGroup)
	assert.Equal(t, []string{decisionsIndex, stackIndex, archIndex}, got.Index)
}

func TestPlanDecisionsAppendsToPresentDocuments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, decisions, stack, separator, lineSeparator string
	}{
		{
			name: "with trailing newline", decisions: "# Log\n\n## P/D-7: Old\n\n## P/D-12: Older\n",
			stack: "# Stack\n\n- a\n", separator: "\n",
		},
		{
			name: "without trailing newline", decisions: "# Log\n\n## P/D-7: Old\n\n## P/D-12: Older",
			stack: "# Stack\n\n- a", separator: "\n\n", lineSeparator: "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, map[string]entry{
				"docs/decisions.md": {content: tt.decisions, mode: 0o644},
				"docs/stack.md":     {content: tt.stack, mode: 0o644},
			})

			got, err := PlanDecisions(root, nil, selection(), byInit())

			require.NoError(t, err)
			require.Len(t, got.Changes, 3)
			assert.Equal(t, Change{
				Path: "docs/decisions.md", Action: ActionAppend, Owner: "decision",
				Content: []byte(tt.decisions + tt.separator + freshEntries("P/D-13", "P/D-14", "P/D-15", "P/D-16")),
			}, got.Changes[0])
			assert.Equal(t, Change{
				Path: "docs/stack.md", Action: ActionAppend, Owner: "decision",
				Content: []byte(tt.stack + tt.lineSeparator + "- language: go (P/D-13)\n" +
					"- frontend: htmx, templ (P/D-15)\n- operations: perf (P/D-16)\n"),
			}, got.Changes[1])
			assert.Equal(t, ActionCreate, got.Changes[2].Action)
			assert.Equal(t, []string{archIndex}, got.Index)
		})
	}
}

func TestPlanDecisionsIgnoresWorkflowAndReviewGroups(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, nil)

	got, err := PlanDecisions(root, nil, []CatalogExtension{
		{Name: "tdd", Kind: "skill", Group: "workflow"},
		{Name: "contrast", Kind: "agent", Group: "review"},
	}, byInit())

	require.NoError(t, err)
	assert.Empty(t, got.Changes)
	assert.Empty(t, got.Files)
	assert.Equal(t, []Decision{}, got.Records)
	assert.Empty(t, got.ByGroup)
	assert.Empty(t, got.Index)
}

func TestPlanDecisionsRejectsANonRegularDocument(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{"docs/stack.md/inner": {content: "x", mode: 0o644}})

	_, err := PlanDecisions(root, nil, selection(), byInit())

	require.EqualError(t, err, "read docs/stack.md: not a regular file")
}

func TestPlanDecisionsRerunCarriesTheRecordedDecisionsOver(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, nil)
	existing := &Manifest{
		Extensions: map[string]Extension{"go": {Group: "language", Decision: "P/D-3"}},
		Files: map[string]File{
			"docs/decisions.md": {Owner: "decision", Entries: []string{"P/D-3"}},
			"docs/stack.md":     {Owner: "decision", Entries: []string{"P/D-3"}},
			"docs/README.md":    {Owner: "seed"},
		},
	}

	got, err := PlanDecisions(root, existing, []CatalogExtension{{Name: "go", Kind: "skill", Group: "language"}},
		byInit())

	require.NoError(t, err)
	assert.Equal(t, []Change{
		{Path: "docs/decisions.md", Action: ActionUnchanged, Owner: "decision"},
		{Path: "docs/stack.md", Action: ActionUnchanged, Owner: "decision"},
	}, got.Changes)
	assert.Equal(t, map[string]File{
		"docs/decisions.md": {Owner: "decision", Entries: []string{"P/D-3"}},
		"docs/stack.md":     {Owner: "decision", Entries: []string{"P/D-3"}},
	}, got.Files)
	assert.Equal(t, []Decision{}, got.Records)
	assert.Equal(t, map[string]string{"language": "P/D-3"}, got.ByGroup)
	assert.Empty(t, got.Index)
}

func TestAddDocuments(t *testing.T) {
	t.Parallel()

	lines := []string{"- `a`", "- `b`"}
	tests := []struct {
		name, content, want string
		ok                  bool
	}{
		{
			name: "after the last item", content: "# R\n\n## Documents\n\n<!-- c -->\n\n- x\n- y\n\n## Next\n",
			want: "# R\n\n## Documents\n\n<!-- c -->\n\n- x\n- y\n- `a`\n- `b`\n\n## Next\n", ok: true,
		},
		{
			name: "no items", content: "## Documents\n\ntext\n## Next\n",
			want: "## Documents\n\n- `a`\n- `b`\n\ntext\n## Next\n", ok: true,
		},
		{name: "no items at the end", content: "## Documents\n", want: "## Documents\n\n- `a`\n- `b`\n", ok: true},
		{
			name: "items of a later section", content: "## Documents\n\n## Next\n\n- z\n",
			want: "## Documents\n\n- `a`\n- `b`\n\n## Next\n\n- z\n", ok: true,
		},
		{name: "no section", content: "# R\n\n### Documents\n", want: "# R\n\n### Documents\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := addDocuments([]byte(tt.content), lines)

			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestDecisionStatus(t *testing.T) {
	t.Parallel()

	const log = "# Decisions\n\n## P/D-1: Programming languages\n\n- Status: approved\n\n" +
		"## W-002/D-1: Session storage\n\n- Status: approved by the user\n\n" +
		"## W-002/D-2: Frontend\n\n- Selected option: templ\n- Status: pending\n\n" +
		"## W-002/D-3: Empty\n\n- Rationale: none\n\n## W-002/D-4: Next\n\n- Status: approved\n"
	tests := []struct {
		name, content, id, status string
		found                     bool
	}{
		{name: "approved", content: log, id: "P/D-1", status: "approved", found: true},
		{name: "first word", content: log, id: "W-002/D-1", status: "approved", found: true},
		{name: "pending", content: log, id: "W-002/D-2", status: "pending", found: true},
		{name: "no status line in the entry", content: log, id: "W-002/D-3", found: true},
		{name: "absent", content: log, id: "W-002/D-9"},
		{name: "prefix of another ID", content: log, id: "P/D-"},
		{name: "no document", id: "P/D-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := map[string]entry{}
			if tt.content != "" {
				files["docs/decisions.md"] = entry{content: tt.content, mode: 0o644}
			}
			root, _ := openTree(t, files)

			status, found, err := DecisionStatus(root, tt.id)

			require.NoError(t, err)
			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.status, status)
		})
	}
}

// frontendProject records go under P/D-1 and htmx under P/D-2 in a manifest and its documents.
func frontendProject(t *testing.T, decisions string) (*os.Root, *Manifest) {
	t.Helper()
	files := map[string]entry{"docs/stack.md": {content: "# Stack\n\n- language: go (P/D-1)\n- frontend: htmx (P/D-2)\n", mode: 0o644}}
	if decisions != "" {
		files["docs/decisions.md"] = entry{content: decisions, mode: 0o644}
	}
	root, _ := openTree(t, files)
	return root, &Manifest{
		Extensions: map[string]Extension{
			"go":   {Group: "language", Decision: "P/D-1"},
			"htmx": {Group: "frontend", Decision: "P/D-2"},
		},
		Files: map[string]File{
			"docs/decisions.md": {Owner: "decision", Entries: []string{"P/D-1", "P/D-2"}},
			"docs/stack.md":     {Owner: "decision", Entries: []string{"P/D-1", "P/D-2"}},
		},
	}
}

func frontend() []CatalogExtension {
	return []CatalogExtension{
		{Name: "templ", Kind: "skill", Group: "frontend"},
		{Name: "go", Kind: "skill", Group: "language"},
		{Name: "htmx", Kind: "skill", Group: "frontend"},
	}
}

const (
	goEntry   = "## P/D-1: Programming languages\n\n- Status: approved\n- Selected option: go\n"
	htmxEntry = "## P/D-2: Frontend technologies\n\n- Status: approved\n- Selected option: htmx\n"
	templRefs = "`docs/solomonar/extensions/htmx/SKILL.md`, `docs/solomonar/extensions/templ/SKILL.md`"
)

func TestPlanDecisionsSupersedesAChangedGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, decisions, old, want, rationale string
		by                                    Decider
		action                                Action
	}{
		{
			name: "with the rationale", decisions: "# Decisions\n\n" + goEntry + "\n" + htmxEntry,
			by:     Decider{Date: decisionDate, Command: "extension add", Rationale: "Type-checked components"},
			action: ActionUpdate, rationale: "Type-checked components",
			want: "# Decisions\n\n" + goEntry + "\n" + strings.Replace(htmxEntry, "- Status: approved",
				"- Status: superseded by P/D-3", 1) + "\n",
		},
		{
			name: "without the old entry", decisions: "# Decisions\n\n" + goEntry + "\n" + htmxEntry, old: "W-002/D-1",
			by:     Decider{Date: decisionDate, Command: "extension add"},
			action: ActionAppend, rationale: "selected by the user with `solomonar extension add`",
			want: "# Decisions\n\n" + goEntry + "\n" + htmxEntry + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, existing := frontendProject(t, tt.decisions)
			old := cmp.Or(tt.old, "P/D-2")
			existing.Extensions["htmx"] = Extension{Group: "frontend", Decision: old}

			got, err := PlanDecisions(root, existing, frontend(), tt.by)

			require.NoError(t, err)
			entry := "## P/D-3: Frontend technologies\n\n- Status: approved\n- Selected option: htmx, templ\n" +
				"- Rejected options: none\n- Scope: project; frontend extensions `htmx`, `templ`\n" +
				"- Rationale: " + tt.rationale + "\n- Date: " + decisionDate + "\n" +
				"- References: supersedes " + old + "; " + templRefs + "\n"
			assert.Equal(t, []Change{
				{Path: "docs/decisions.md", Action: tt.action, Owner: "decision", Content: []byte(tt.want + entry)},
				{
					Path: "docs/stack.md", Action: ActionAppend, Owner: "decision",
					Content: []byte("# Stack\n\n- language: go (P/D-1)\n- frontend: htmx (P/D-2)\n" +
						"- frontend: htmx, templ (P/D-3)\n"),
				},
			}, got.Changes)
			assert.Equal(t, []Decision{
				{ID: "P/D-3", Group: "frontend", Selected: []string{"htmx", "templ"}, Supersedes: old},
			}, got.Records)
			assert.Equal(t, map[string]string{"language": "P/D-1", "frontend": "P/D-3"}, got.ByGroup)
			assert.Equal(t, map[string]File{
				"docs/decisions.md": {Owner: "decision", Entries: []string{"P/D-1", "P/D-2", "P/D-3"}},
				"docs/stack.md":     {Owner: "decision", Entries: []string{"P/D-1", "P/D-2", "P/D-3"}},
			}, got.Files)
		})
	}
}

func TestPlanDecisionsRecordsAnApprovedDecisionWithoutAnEntry(t *testing.T) {
	t.Parallel()
	root, existing := frontendProject(t, "# Decisions\n\n"+goEntry+"\n"+htmxEntry)

	got, err := PlanDecisions(root, existing, frontend(), Decider{Date: decisionDate, ID: "W-002/D-1"})

	require.NoError(t, err)
	assert.Equal(t, []Change{
		{Path: "docs/decisions.md", Action: ActionUnchanged, Owner: "decision"},
		{
			Path: "docs/stack.md", Action: ActionAppend, Owner: "decision",
			Content: []byte("# Stack\n\n- language: go (P/D-1)\n- frontend: htmx (P/D-2)\n" +
				"- frontend: htmx, templ (W-002/D-1)\n"),
		},
	}, got.Changes)
	assert.Equal(t, []Decision{{ID: "W-002/D-1", Group: "frontend", Selected: []string{"htmx", "templ"}}}, got.Records)
	assert.Equal(t, map[string]string{"language": "P/D-1", "frontend": "W-002/D-1"}, got.ByGroup)
	assert.Equal(t, []string{"P/D-1", "P/D-2", "W-002/D-1"}, got.Files["docs/stack.md"].Entries)
}

func TestPlanDecisionsRecordsANewGroupWithoutSupersession(t *testing.T) {
	t.Parallel()
	root, existing := frontendProject(t, "# Decisions\n\n"+goEntry+"\n"+htmxEntry)
	installed := append(frontend(), CatalogExtension{Name: "sql", Kind: "skill", Group: "persistence"})
	existing.Extensions["templ"] = Extension{Group: "frontend", Decision: "P/D-2"}

	got, err := PlanDecisions(root, existing, installed, Decider{Date: decisionDate, Command: "extension add"})

	require.NoError(t, err)
	assert.Equal(t, []Decision{{ID: "P/D-3", Group: "persistence", Selected: []string{"sql"}}}, got.Records)
	assert.Equal(t, map[string]string{"language": "P/D-1", "frontend": "P/D-2", "persistence": "P/D-3"}, got.ByGroup)
	require.Len(t, got.Changes, 2)
	assert.Equal(t, ActionAppend, got.Changes[0].Action)
	assert.Contains(t, string(got.Changes[0].Content), htmxEntry+"\n## P/D-3: Persistence technologies\n")
	assert.Contains(t, string(got.Changes[0].Content), "- References: `docs/solomonar/extensions/sql/SKILL.md`\n")
}

func TestPlanDecisionsNeverReusesARecordedID(t *testing.T) {
	t.Parallel()
	root, existing := frontendProject(t, "# Decisions\n\n"+goEntry)

	got, err := PlanDecisions(root, existing, frontend(), Decider{Date: decisionDate, Command: "extension add"})

	require.NoError(t, err)
	assert.Equal(t, []Decision{
		{ID: "P/D-3", Group: "frontend", Selected: []string{"htmx", "templ"}, Supersedes: "P/D-2"},
	}, got.Records)
	require.NotEmpty(t, got.Changes)
	assert.Equal(t, ActionAppend, got.Changes[0].Action)
}

func TestPlanDecisionsRecordsAnEmptiedGroupAsNone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, id, entries string
		by                Decider
	}{
		{
			name: "by the command", id: "P/D-3", by: Decider{Date: decisionDate, Command: "extension remove"},
			entries: "# Decisions\n\n" + goEntry + "\n" + strings.Replace(htmxEntry, "- Status: approved",
				"- Status: superseded by P/D-3", 1) + "\n## P/D-3: Frontend technologies\n\n- Status: approved\n" +
				"- Selected option: none\n- Rejected options: none\n- Scope: project; frontend extensions none\n" +
				"- Rationale: selected by the user with `solomonar extension remove`\n- Date: " + decisionDate + "\n" +
				"- References: supersedes P/D-2\n",
		},
		{
			name: "by an approved decision", id: "W-002/D-3", by: Decider{Date: decisionDate, ID: "W-002/D-3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, existing := frontendProject(t, "# Decisions\n\n"+goEntry+"\n"+htmxEntry)
			installed := []CatalogExtension{{Name: "go", Kind: "skill", Group: "language"}}

			got, err := PlanDecisions(root, existing, installed, tt.by)

			require.NoError(t, err)
			record := Decision{ID: tt.id, Group: "frontend", Selected: []string{}}
			if tt.entries != "" {
				record.Supersedes = "P/D-2"
				assert.Equal(t, tt.entries, string(got.Changes[0].Content))
			}
			assert.Equal(t, []Decision{record}, got.Records)
			assert.Equal(t, "# Stack\n\n- language: go (P/D-1)\n- frontend: htmx (P/D-2)\n- frontend: none ("+
				tt.id+")\n", string(got.Changes[1].Content))
			assert.Equal(t, map[string]string{"language": "P/D-1", "frontend": tt.id}, got.ByGroup)
		})
	}
}

func TestPlanDecisionsSupersedesTheNoneOfAnEmptiedGroup(t *testing.T) {
	t.Parallel()
	root, _ := openTree(t, map[string]entry{
		"docs/stack.md": {
			content: "# Stack\n\n- language: go (P/D-1)\n- frontend: htmx (P/D-2)\n- frontend: none (P/D-3)\n", mode: 0o644,
		},
	})
	existing := &Manifest{
		Extensions: map[string]Extension{"go": {Group: "language", Decision: "P/D-1"}},
		Files: map[string]File{
			"docs/decisions.md": {Owner: "decision", Entries: []string{"P/D-1", "P/D-2", "P/D-3"}},
			"docs/stack.md":     {Owner: "decision", Entries: []string{"P/D-1", "P/D-2", "P/D-3"}},
		},
	}
	installed := []CatalogExtension{{Name: "go", Group: "language"}, {Name: "templ", Group: "frontend"}}

	got, err := PlanDecisions(root, existing, installed, Decider{Date: decisionDate, Command: "extension add"})

	require.NoError(t, err)
	assert.Equal(t, []Decision{{ID: "P/D-4", Group: "frontend", Selected: []string{"templ"}, Supersedes: "P/D-3"}},
		got.Records)
}
