package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var ErrTooOld = errors.New("unsupported git version")

const (
	minMajor = 2
	minMinor = 40
)

// Version returns the version of git on PATH and fails with ErrTooOld below 2.40.
func Version(ctx context.Context, trace io.Writer) (string, error) {
	out, err := run(ctx, "git", "", trace, nil, []string{"version"})
	if err != nil {
		return "", fmt.Errorf("check git version: %w", err)
	}
	version, err := parseVersion(out)
	if err != nil {
		return "", err
	}
	return version, checkMinimumVersion(version)
}

func parseVersion(out string) (string, error) {
	rest, ok := strings.CutPrefix(out, "git version ")
	fields := strings.Fields(rest)
	if !ok || len(fields) == 0 {
		return "", fmt.Errorf("parse git version: unexpected output %q", out)
	}
	return fields[0], nil
}

func checkMinimumVersion(version string) error {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return fmt.Errorf("check git version: malformed version %q", version)
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil {
		return fmt.Errorf("check git version: malformed version %q", version)
	}
	if major < minMajor || major == minMajor && minor < minMinor {
		return fmt.Errorf("%w: git %s is older than %d.%d", ErrTooOld, version, minMajor, minMinor)
	}
	return nil
}
