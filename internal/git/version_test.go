package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		out     string
		want    string
		wantErr bool
	}{
		{name: "plain", out: "git version 2.56.0", want: "2.56.0"},
		{name: "apple suffix", out: "git version 2.39.5 (Apple Git-154)", want: "2.39.5"},
		{name: "windows suffix", out: "git version 2.40.0.windows.1", want: "2.40.0.windows.1"},
		{name: "unexpected output", out: "hello", wantErr: true},
		{name: "empty", out: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseVersion(tt.out)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckMinimumVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		tooOld  bool
	}{
		{name: "minimum", version: "2.40.0", tooOld: false},
		{name: "newer minor", version: "2.56.0", tooOld: false},
		{name: "newer major", version: "3.0", tooOld: false},
		{name: "windows suffix", version: "2.40.0.windows.1", tooOld: false},
		{name: "older minor", version: "2.39.5", tooOld: true},
		{name: "older major", version: "1.99.0", tooOld: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := checkMinimumVersion(tt.version)
			if tt.tooOld {
				require.ErrorIs(t, err, ErrTooOld)
				assert.Contains(t, err.Error(), "git "+tt.version+" is older than 2.40")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCheckMinimumVersionRejectsMalformedVersion(t *testing.T) {
	t.Parallel()

	err := checkMinimumVersion("two.forty")

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrTooOld)
}

func TestVersionOfInstalledGitIsSupported(t *testing.T) {
	t.Parallel()

	version, err := Version(t.Context(), nil)

	require.NoError(t, err)
	assert.NotEmpty(t, version)
}
