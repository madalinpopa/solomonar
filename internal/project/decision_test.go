package project

import (
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
		"`docs/solomonar/skills/go/SKILL.md`") + "\n" +
		entryText(second, "Architecture style", "architecture", "hexagonal", "`hexagonal`",
			"`docs/solomonar/skills/hexagonal/SKILL.md`") + "\n" +
		entryText(third, "Frontend technologies", "frontend", "htmx, templ", "`htmx`, `templ`",
			"`docs/solomonar/skills/htmx/SKILL.md`, `docs/solomonar/skills/templ/SKILL.md`") + "\n" +
		entryText(fourth, "Build and deployment", "operations", "perf", "`perf`",
			"`docs/solomonar/agents/perf.md`")
}

func TestPlanDecisionsRecordsOneEntryPerStackGroupInGroupOrder(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, nil)

	got, err := PlanDecisions(root, nil, selection(), decisionDate)

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

			got, err := PlanDecisions(root, nil, selection(), decisionDate)

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
	}, decisionDate)

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

	_, err := PlanDecisions(root, nil, selection(), decisionDate)

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
		decisionDate)

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
