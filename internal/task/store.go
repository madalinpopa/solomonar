package task

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/madalinpopa/solomonar/internal/git"
)

// Ref is the branch that holds state.json.
const Ref = "refs/heads/workflow-state"

const (
	maxSwapAttempts = 5
	pushTimeout     = 30 * time.Second
	initSubject     = "state: initialize registry"
)

// Store reads and writes the registry on Ref with compare-and-swap (AD-5).
type Store struct {
	Git    *git.Repo
	Remote string
}

// Commit is a registry commit and the outcome of pushing it.
type Commit struct {
	SHA      string
	Pushed   bool
	Warnings []Warning
}

// Warning is a problem that did not fail the write.
type Warning struct {
	Code    string
	Message string
}

// Read returns the registry at the tip of Ref and the tip itself.
func (s *Store) Read(ctx context.Context) (*Registry, string, error) {
	snap, err := s.read(ctx)
	if err != nil {
		return nil, "", err
	}
	return snap.registry, snap.tip, nil
}

// snapshot is the registry at tip with the blob of each file, so a write reuses the unchanged ones.
type snapshot struct {
	registry *Registry
	tip      string
	blobs    map[string]blob
}

type blob struct {
	sha  string
	data []byte
}

func (s *Store) read(ctx context.Context) (snapshot, error) {
	tip, err := s.tip(ctx)
	if err != nil {
		return snapshot{}, err
	}
	if tip == "" {
		return snapshot{}, notInitialized()
	}
	blobs, err := s.treeBlobs(ctx, tip)
	if err != nil {
		return snapshot{}, err
	}
	files := make(map[string][]byte, len(blobs))
	for path, b := range blobs {
		files[path] = b.data
	}
	reg, err := DecodeFiles(files)
	if err != nil {
		return snapshot{}, err
	}
	return snapshot{registry: reg, tip: tip, blobs: blobs}, nil
}

func notInitialized() *Rejection {
	return &Rejection{
		Code:  "not_initialized",
		Check: "the registry ref " + Ref + " exists",
		Found: Ref + " is missing",
		Fix:   "solomonar init",
	}
}

// treeBlobs reads every file of the commit's tree with one ls-tree and one cat-file --batch.
func (s *Store) treeBlobs(ctx context.Context, commit string) (map[string]blob, error) {
	listing, err := s.Git.RunRaw(ctx, nil, "ls-tree", "-r", "-z", "--full-tree", commit)
	if err != nil {
		return nil, fmt.Errorf("read workflow-state: %w", err)
	}
	var paths, shas []string
	for entry := range strings.SplitSeq(strings.TrimSuffix(string(listing), "\x00"), "\x00") {
		if entry == "" {
			continue
		}
		info, path, _ := strings.Cut(entry, "\t")
		fields := strings.Fields(info)
		if len(fields) != 3 || fields[1] != "blob" {
			return nil, invalidRegistry("unexpected %s entry %s", fields[1], path)
		}
		paths = append(paths, path)
		shas = append(shas, fields[2])
	}
	contents, err := s.catBlobs(ctx, shas)
	if err != nil {
		return nil, err
	}
	blobs := make(map[string]blob, len(paths))
	for i, path := range paths {
		blobs[path] = blob{sha: shas[i], data: contents[i]}
	}
	return blobs, nil
}

func (s *Store) catBlobs(ctx context.Context, shas []string) ([][]byte, error) {
	if len(shas) == 0 {
		return nil, nil
	}
	out, err := s.Git.RunRaw(ctx, strings.NewReader(strings.Join(shas, "\n")+"\n"), "cat-file", "--batch")
	if err != nil {
		return nil, fmt.Errorf("read workflow-state: %w", err)
	}
	contents := make([][]byte, len(shas))
	for i := range shas {
		header, rest, ok := bytes.Cut(out, []byte("\n"))
		fields := strings.Fields(string(header))
		if !ok || len(fields) != 3 {
			return nil, fmt.Errorf("read workflow-state: unexpected cat-file output %q", header)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size+1 > len(rest) {
			return nil, fmt.Errorf("read workflow-state: unexpected cat-file output %q", header)
		}
		contents[i] = rest[:size]
		out = rest[size+1:]
	}
	return contents, nil
}

// ReadSchema returns the schema version at the tip of Ref without decoding the registry strictly.
func (s *Store) ReadSchema(ctx context.Context) (schema int, present bool, err error) {
	tip, err := s.tip(ctx)
	if err != nil || tip == "" {
		return 0, false, err
	}
	data, err := s.header(ctx, tip)
	if err != nil {
		return 0, true, err
	}
	schema, err = SchemaOf(data)
	return schema, true, err
}

func (s *Store) header(ctx context.Context, tip string) ([]byte, error) {
	data, err := s.Git.Run(ctx, "cat-file", "blob", tip+":"+headerFile)
	if err != nil {
		return nil, fmt.Errorf("read workflow-state: %w", err)
	}
	return []byte(data), nil
}

// Create writes an empty registry as the root commit of Ref and fails if Ref exists.
func (s *Store) Create(ctx context.Context) (Commit, error) {
	tip, err := s.tip(ctx)
	if err != nil {
		return Commit{}, err
	}
	if tip != "" {
		return Commit{}, fmt.Errorf("create workflow-state: %s already exists", Ref)
	}
	sha, err := s.writeCommit(ctx, NewRegistry(), nil, initSubject, "")
	if err != nil {
		return Commit{}, err
	}
	// An empty old value makes update-ref fail when the ref exists, so a concurrent init is never overwritten.
	if err := s.swap(ctx, sha, ""); err != nil {
		return Commit{}, fmt.Errorf("create workflow-state: %w", err)
	}
	return s.push(ctx, sha), nil
}

// Update runs fn on the tip registry and swaps in the result, re-running fn on a lost swap.
// An empty subject from fn means no change: Update writes nothing and returns the current tip.
func (s *Store) Update(ctx context.Context, fn func(*Registry) (subject string, err error)) (Commit, error) {
	for range maxSwapAttempts {
		snap, err := s.read(ctx)
		if err != nil {
			return Commit{}, err
		}
		old := snap.tip
		subject, err := fn(snap.registry)
		if err != nil {
			return Commit{}, err
		}
		if subject == "" {
			return Commit{SHA: old}, nil
		}
		sha, err := s.writeCommit(ctx, snap.registry, snap.blobs, subject, old)
		if err != nil {
			return Commit{}, err
		}
		swapErr := s.swap(ctx, sha, old)
		if swapErr == nil {
			return s.push(ctx, sha), nil
		}
		if ctx.Err() != nil {
			return Commit{}, fmt.Errorf("update workflow-state: %w", swapErr)
		}
		current, err := s.tip(ctx)
		if err != nil {
			return Commit{}, err
		}
		if current == old {
			return Commit{}, fmt.Errorf("update workflow-state: %w", swapErr)
		}
	}
	return Commit{}, casExhausted()
}

func casExhausted() *Rejection {
	return &Rejection{
		Code:  "cas_exhausted",
		Check: fmt.Sprintf("the registry compare-and-swap succeeds within %d attempts", maxSwapAttempts),
		Found: fmt.Sprintf("%s moved during each of %d attempts", Ref, maxSwapAttempts),
		Fix:   "retry the command; no state was written",
	}
}

// Migrate rewrites a schema-1 registry as the current schema on top of its history (CLI-D26).
// It returns migrated false, writing nothing, when the registry already has the current schema.
func (s *Store) Migrate(ctx context.Context) (reg *Registry, commit Commit, migrated bool, err error) {
	for range maxSwapAttempts {
		old, err := s.tip(ctx)
		if err != nil {
			return nil, Commit{}, false, err
		}
		reg, legacy, err := s.readForMigration(ctx, old)
		if err != nil || !legacy {
			return reg, Commit{SHA: old}, false, err
		}
		subject := fmt.Sprintf("state: migrate schema %d -> %d", legacySchema, SchemaVersion)
		sha, err := s.writeCommit(ctx, reg, nil, subject, old)
		if err != nil {
			return nil, Commit{}, false, err
		}
		swapErr := s.swap(ctx, sha, old)
		if swapErr == nil {
			return reg, s.push(ctx, sha), true, nil
		}
		if ctx.Err() != nil {
			return nil, Commit{}, false, fmt.Errorf("migrate workflow-state: %w", swapErr)
		}
		if current, err := s.tip(ctx); err != nil || current == old {
			return nil, Commit{}, false, cmp.Or(err, fmt.Errorf("migrate workflow-state: %w", swapErr))
		}
	}
	return nil, Commit{}, false, casExhausted()
}

// readForMigration decodes the registry at tip; legacy reports a schema-1 registry, now current.
func (s *Store) readForMigration(ctx context.Context, tip string) (reg *Registry, legacy bool, err error) {
	if tip == "" {
		return nil, false, notInitialized()
	}
	data, err := s.header(ctx, tip)
	if err != nil {
		return nil, false, err
	}
	version, err := SchemaOf(data)
	if err != nil {
		return nil, false, err
	}
	if version != legacySchema {
		reg, _, err := s.Read(ctx)
		return reg, false, err
	}
	reg, err = DecodeLegacy(data)
	if err != nil {
		return nil, false, err
	}
	reg.SchemaVersion = SchemaVersion
	return reg, true, nil
}

// Tip returns the commit Ref points to, or "" when Ref does not exist.
func (s *Store) Tip(ctx context.Context) (string, error) {
	return s.tip(ctx)
}

func (s *Store) tip(ctx context.Context) (string, error) {
	out, err := s.Git.Run(ctx, "rev-parse", "--verify", "--quiet", Ref)
	// With --quiet, a missing ref exits 1 without output.
	if gitErr, ok := errors.AsType[*git.Error](err); ok && gitErr.ExitCode == 1 && ctx.Err() == nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read workflow-state: %w", err)
	}
	return out, nil
}

// writeCommit writes the registry's tree, hashing only files whose content differs from previous.
func (s *Store) writeCommit(ctx context.Context, reg *Registry, previous map[string]blob, subject, parent string) (string, error) {
	files, err := reg.Files()
	if err != nil {
		return "", err
	}
	shas := make(map[string]string, len(files))
	for path, data := range files {
		if old, ok := previous[path]; ok && bytes.Equal(old.data, data) {
			shas[path] = old.sha
			continue
		}
		sha, err := s.Git.RunInput(ctx, bytes.NewReader(data), "hash-object", "-w", "--stdin")
		if err != nil {
			return "", fmt.Errorf("write %s: %w", path, err)
		}
		shas[path] = sha
	}
	tree, err := s.writeTree(ctx, shas)
	if err != nil {
		return "", err
	}
	args := []string{"commit-tree", tree, "-m", subject}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	sha, err := s.Git.Run(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("write registry commit: %w", err)
	}
	return sha, nil
}

// writeTree builds the tasks subtree, when there are tasks, and the root tree around it.
func (s *Store) writeTree(ctx context.Context, shas map[string]string) (string, error) {
	var tasks strings.Builder
	for _, path := range slices.Sorted(maps.Keys(shas)) {
		if name, ok := strings.CutPrefix(path, tasksDir); ok {
			fmt.Fprintf(&tasks, "100644 blob %s\t%s\n", shas[path], name)
		}
	}
	root := fmt.Sprintf("100644 blob %s\t%s\n", shas[headerFile], headerFile)
	if tasks.Len() > 0 {
		subtree, err := s.Git.RunInput(ctx, strings.NewReader(tasks.String()), "mktree")
		if err != nil {
			return "", fmt.Errorf("write tasks tree: %w", err)
		}
		root += fmt.Sprintf("040000 tree %s\t%s\n", subtree, strings.TrimSuffix(tasksDir, "/"))
	}
	tree, err := s.Git.RunInput(ctx, strings.NewReader(root), "mktree")
	if err != nil {
		return "", fmt.Errorf("write registry tree: %w", err)
	}
	return tree, nil
}

// swap moves Ref from old to sha unless ctx is already done.
func (s *Store) swap(ctx context.Context, sha, old string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// The swap is the commit point (CLI-D18): once started it runs to completion, so the outcome is known.
	_, err := s.Git.Run(context.WithoutCancel(ctx), "update-ref", Ref, sha, old)
	return err
}

func (s *Store) push(ctx context.Context, sha string) Commit {
	warning := s.pushWarning(ctx)
	if warning == nil {
		return Commit{SHA: sha, Pushed: true}
	}
	return Commit{SHA: sha, Warnings: []Warning{*warning}}
}

func (s *Store) pushWarning(ctx context.Context) *Warning {
	if s.Remote == "" {
		return new(noRemote(s.Remote))
	}
	if _, err := s.Git.Run(ctx, "remote", "get-url", s.Remote); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return new(s.pushInterrupted())
		}
		return new(noRemote(s.Remote))
	}
	return s.pushRef(ctx)
}

func (s *Store) pushRef(ctx context.Context) *Warning {
	pushCtx, cancel := context.WithTimeout(ctx, pushTimeout)
	defer cancel()
	_, err := s.Git.Run(pushCtx, "push", s.Remote, Ref+":"+Ref)
	switch {
	case err == nil:
		return nil
	case errors.Is(ctx.Err(), context.Canceled):
		return new(s.pushInterrupted())
	default:
		return &Warning{
			Code:    "push_failed",
			Message: fmt.Sprintf("git push %s workflow-state failed: %s", s.Remote, pushFailure(err)),
		}
	}
}

func (s *Store) pushInterrupted() Warning {
	return Warning{
		Code:    "push_interrupted",
		Message: fmt.Sprintf("push to %s interrupted; the registry commit is kept locally", s.Remote),
	}
}

func noRemote(remote string) Warning {
	if remote == "" {
		return Warning{Code: "no_remote", Message: "no remote configured; the registry stays local"}
	}
	return Warning{Code: "no_remote", Message: fmt.Sprintf("no remote %q; the registry stays local", remote)}
}

func pushFailure(err error) string {
	if gitErr, ok := errors.AsType[*git.Error](err); ok && gitErr.Stderr != "" {
		return gitErr.Stderr
	}
	return err.Error()
}
