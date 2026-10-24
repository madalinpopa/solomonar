package git

import (
	"bytes"
	"context"
	"fmt"
)

// Files lists the worktree's tracked and untracked, not ignored, files relative to its root.
func (r *Repo) Files(ctx context.Context) ([]string, error) {
	out, err := r.RunRaw(ctx, nil, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	var files []string
	for file := range bytes.SplitSeq(out, []byte{0}) {
		if len(file) > 0 {
			files = append(files, string(file))
		}
	}
	return files, nil
}
