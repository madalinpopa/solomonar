package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		glob, name string
		want       bool
	}{
		{glob: "**/*.ts", name: "app.ts", want: true},
		{glob: "**/*.ts", name: "web/src/app.ts", want: true},
		{glob: "**/*.ts", name: "web/src/app.tsx", want: false},
		{glob: "go.mod", name: "go.mod", want: true},
		{glob: "go.mod", name: "tools/go.mod", want: false},
		{glob: "**/go.mod", name: "go.mod", want: true},
		{glob: "**/go.mod", name: "tools/lint/go.mod", want: true},
		{glob: "**/go.mod", name: "go.mod.bak", want: false},
		{glob: "*.go", name: "cmd/main.go", want: false},
		{glob: "internal/**", name: "internal", want: true},
		{glob: "internal/**", name: "internal/money/a.go", want: true},
		{glob: "a/**/b", name: "a/b", want: true},
		{glob: "a/**/b", name: "a/x/y/b", want: true},
		{glob: "a/**/b", name: "a/x/y/c", want: false},
		{glob: "?.md", name: "a.md", want: true},
		{glob: "[ab].md", name: "c.md", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.glob+" "+tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, matchGlob(tt.glob, tt.name))
		})
	}
}

func TestValidGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		glob string
		want bool
	}{
		{glob: "**/go.mod", want: true},
		{glob: "go.mod", want: true},
		{glob: "src/[a-z]*.ts", want: true},
		{glob: "", want: false},
		{glob: "/go.mod", want: false},
		{glob: "a//b", want: false},
		{glob: "a/", want: false},
		{glob: "./go.mod", want: false},
		{glob: "../go.mod", want: false},
		{glob: "a**/b", want: false},
		{glob: "[a.go", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.glob, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, validGlob(tt.glob))
		})
	}
}

func TestSuggestedNamesDefaultAndDetectedExtensions(t *testing.T) {
	t.Parallel()

	catalog := Catalog{
		"go":   {Name: "go", Detect: []string{"**/go.mod"}},
		"ts":   {Name: "ts", Detect: []string{"**/*.ts"}},
		"auth": {Name: "auth", Default: true},
		"both": {Name: "both", Default: true, Detect: []string{"**/*.go"}},
		"none": {Name: "none"},
	}
	// The list is git ls-files output, so an ignored node_modules/x.ts is never in it.
	files := []string{"tools/go.mod", "go.mod", "cmd/main.go"}

	assert.Equal(t, map[string]string{"go": "go.mod", "auth": "default", "both": "cmd/main.go"},
		catalog.Suggested(files))
}
