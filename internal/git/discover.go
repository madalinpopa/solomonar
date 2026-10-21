package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

var ErrNotRepository = errors.New("not a git repository")

const exitFatal = 128

// Discover finds the repository containing dir, or the working directory when dir is empty.
func Discover(ctx context.Context, dir string, trace io.Writer) (*Repo, error) {
	out, err := run(ctx, dir, trace, nil, []string{
		"rev-parse", "--path-format=absolute", "--show-toplevel", "--git-common-dir",
	})
	if gitErr, ok := errors.AsType[*Error](err); ok && gitErr.ExitCode == exitFatal {
		return nil, fmt.Errorf("discover repository: %s: %w", displayDir(dir), ErrNotRepository)
	}
	if err != nil {
		return nil, fmt.Errorf("discover repository: %w", err)
	}
	top, commonDir, ok := strings.Cut(out, "\n")
	if !ok {
		return nil, fmt.Errorf("discover repository: unexpected rev-parse output %q", out)
	}
	return &Repo{Dir: top, CommonDir: commonDir, Trace: trace}, nil
}

func displayDir(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}
