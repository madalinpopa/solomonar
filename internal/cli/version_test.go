package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCLIVersionFromBuildInfo(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "tagged module", raw: "v0.1.0", want: "0.1.0"},
		{name: "no leading v", raw: "0.2.0", want: "0.2.0"},
		{name: "pseudo version", raw: "v0.1.1-0.20261021120000-abcdef123456", want: "0.1.1-0.20261021120000-abcdef123456"},
		{name: "development build", raw: "(devel)", want: "devel"},
		{name: "empty", raw: "", want: "devel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cliVersion(tt.raw))
		})
	}
}
