package project

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderKeepsSectionsWhoseExtensionsAreInstalled(t *testing.T) {
	t.Parallel()

	content := "---\nname: templ\n---\n\n# templ\n\nIntro.\n\n## Rules\n\n- templ/C-1\n\n" +
		"## With htmx\n<!-- solomonar:with htmx -->\n\n- templ/C-4\n\n" +
		"## With go and htmx\n<!-- solomonar:with go,htmx -->\n\n- templ/C-5\n\n" +
		"## With tdd\n<!-- solomonar:with tdd -->\n\n- templ/C-6\n"
	tests := []struct {
		name      string
		installed []string
		decision  string
		want      string
		with      []string
	}{
		{
			name:      "every section kept",
			installed: []string{"templ", "tdd", "htmx", "go"},
			decision:  "P/D-2",
			want: "---\nname: templ\n---\n\n# templ\n\nInstalled by P/D-2; rendered with: go, htmx, tdd.\n\nIntro.\n\n" +
				"## Rules\n\n- templ/C-1\n\n## With htmx\n\n- templ/C-4\n\n## With go and htmx\n\n- templ/C-5\n\n" +
				"## With tdd\n\n- templ/C-6\n",
			with: []string{"go", "htmx", "tdd"},
		},
		{
			name:      "multi-name section needs every name",
			installed: []string{"templ", "htmx"},
			decision:  "P/D-2",
			want: "---\nname: templ\n---\n\n# templ\n\nInstalled by P/D-2; rendered with: htmx.\n\nIntro.\n\n" +
				"## Rules\n\n- templ/C-1\n\n## With htmx\n\n- templ/C-4\n",
			with: []string{"htmx"},
		},
		{
			name:      "every section dropped",
			installed: []string{"templ"},
			decision:  "P/D-2",
			want:      "---\nname: templ\n---\n\n# templ\n\nInstalled by P/D-2.\n\nIntro.\n\n## Rules\n\n- templ/C-1\n",
		},
		{
			name:      "empty decision",
			installed: []string{"templ", "tdd"},
			want: "---\nname: templ\n---\n\n# templ\n\nInstalled by solomonar; rendered with: tdd.\n\nIntro.\n\n" +
				"## Rules\n\n- templ/C-1\n\n## With tdd\n\n- templ/C-6\n",
			with: []string{"tdd"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rendered, with := Render([]byte(content), "templ", tt.installed, tt.decision)

			assert.Equal(t, tt.want, string(rendered))
			assert.Equal(t, tt.with, with)
		})
	}
}

func TestRenderExcludesItselfFromRenderedWith(t *testing.T) {
	t.Parallel()

	content := "# a\n\n## With a\n<!-- solomonar:with a, b -->\n\n- a/C-1\n"

	rendered, with := Render([]byte(content), "a", []string{"a", "b"}, "P/D-1")

	assert.Equal(t, "# a\n\nInstalled by P/D-1; rendered with: b.\n\n## With a\n\n- a/C-1\n", string(rendered))
	assert.Equal(t, []string{"b"}, with)
}

func TestRenderKeepsAWithHeadingWithoutMarker(t *testing.T) {
	t.Parallel()

	content := "# a\n\n## With care\n\nText.\n"

	rendered, with := Render([]byte(content), "a", []string{"a"}, "P/D-1")

	assert.Equal(t, "# a\n\nInstalled by P/D-1.\n\n## With care\n\nText.\n", string(rendered))
	assert.Nil(t, with)
}

func TestRenderWithoutTitleAddsNoProvenance(t *testing.T) {
	t.Parallel()

	content := "Intro.\n\n## With b\n<!-- solomonar:with b -->\n\n- a/C-1\n"

	rendered, with := Render([]byte(content), "a", []string{"a"}, "P/D-1")

	assert.Equal(t, "Intro.\n", string(rendered))
	assert.Nil(t, with)
}

func TestRenderTheGoSkill(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../assets/extensions/language/go/SKILL.md")
	require.NoError(t, err)
	tests := []struct {
		name       string
		installed  []string
		provenance string
		with       []string
		keepsC20   bool
	}{
		{
			name:       "with tdd",
			installed:  []string{"go", "tdd"},
			provenance: "# Go language rules\n\nInstalled by P/D-1; rendered with: tdd.\n\nFramework version:",
			with:       []string{"tdd"},
			keepsC20:   true,
		},
		{
			name:       "without tdd",
			installed:  []string{"go"},
			provenance: "# Go language rules\n\nInstalled by P/D-1.\n\nFramework version:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rendered, with := Render(content, "go", tt.installed, "P/D-1")

			assert.Contains(t, string(rendered), tt.provenance)
			if tt.keepsC20 {
				assert.Contains(t, string(rendered), "\n\n- go/C-20 (must):")
			} else {
				assert.NotContains(t, string(rendered), "go/C-20")
			}
			assert.NotContains(t, string(rendered), "solomonar:with")
			assert.Contains(t, string(rendered), "\n## Suggested checks\n")
			assert.Equal(t, tt.with, with)
		})
	}
}
