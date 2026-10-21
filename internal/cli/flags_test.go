package cli

import (
	"flag"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInterspersedAllowsFlagsAnywhere(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		want  []string
		flags sharedFlags
	}{
		{name: "no arguments", args: nil, want: nil},
		{name: "flags first", args: []string{"--json", "a", "b"}, want: []string{"a", "b"}, flags: sharedFlags{json: true}},
		{name: "flags between", args: []string{"a", "-C", "dir", "b"}, want: []string{"a", "b"}, flags: sharedFlags{dir: "dir"}},
		{name: "flags last", args: []string{"a", "-v"}, want: []string{"a"}, flags: sharedFlags{verbose: true}},
		{name: "double dash ends flags", args: []string{"a", "--", "--json", "b"}, want: []string{"a", "--json", "b"}},
		{name: "leading double dash", args: []string{"--", "-v"}, want: []string{"-v"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got sharedFlags
			fs := newFlagSet("test", &got)

			positionals, err := parseInterspersed(fs, tt.args)

			require.NoError(t, err)
			assert.Equal(t, tt.want, positionals)
			assert.Equal(t, tt.flags, got)
		})
	}
}

func TestParseInterspersedRejectsUnknownFlag(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	_, err := parseInterspersed(fs, []string{"a", "--bogus"})

	require.Error(t, err)
}
