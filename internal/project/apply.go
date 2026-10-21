package project

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
)

const (
	fileMode = 0o644
	dirMode  = 0o755
)

// Journal records what Apply changed so Rollback can restore it (AD-6, F-09).
type Journal struct {
	root    *os.Root
	entries []journalEntry
}

type journalEntry struct {
	path       string
	existed    bool
	content    []byte
	mode       fs.FileMode
	createdDir bool
}

// Apply writes the plan's acting changes in order, rolling back on failure.
func Apply(ctx context.Context, root *os.Root, p Plan) (*Journal, error) {
	j := &Journal{root: root}
	for _, c := range p.Changes {
		err := ctx.Err()
		if err == nil {
			err = j.apply(c)
		}
		if err != nil {
			return nil, errors.Join(fmt.Errorf("apply %s: %w", c.Path, err), j.Rollback())
		}
	}
	return j, nil
}

func (j *Journal) apply(c Change) error {
	switch c.Action {
	case ActionCreate:
		return j.create(c.Path, c.Content)
	case ActionUpdate, ActionAppend:
		return j.update(c.Path, c.Content)
	case ActionRemove:
		return j.remove(c.Path)
	case ActionUnchanged, ActionKeep, ActionConflict, ActionSkip:
		return nil
	}
	return fmt.Errorf("unknown action %q", c.Action)
}

func (j *Journal) create(name string, content []byte) error {
	_, err := j.root.Lstat(name)
	if err == nil {
		return fs.ErrExist
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := j.makeParents(path.Dir(name)); err != nil {
		return err
	}
	if err := writeFile(j.root, name, content, fileMode); err != nil {
		return err
	}
	j.entries = append(j.entries, journalEntry{path: name})
	return nil
}

func (j *Journal) update(name string, content []byte) error {
	previous, err := j.readRegular(name)
	if err != nil {
		return err
	}
	if err := writeFile(j.root, name, content, previous.mode); err != nil {
		return err
	}
	j.entries = append(j.entries, previous)
	return nil
}

func (j *Journal) remove(name string) error {
	previous, err := j.readRegular(name)
	if err != nil {
		return err
	}
	if err := j.root.Remove(name); err != nil {
		return err
	}
	j.entries = append(j.entries, previous)
	return nil
}

func (j *Journal) readRegular(name string) (journalEntry, error) {
	info, err := j.root.Lstat(name)
	if err != nil {
		return journalEntry{}, err
	}
	if !info.Mode().IsRegular() {
		return journalEntry{}, fmt.Errorf("%s is not a regular file", name)
	}
	content, err := j.root.ReadFile(name)
	if err != nil {
		return journalEntry{}, err
	}
	return journalEntry{path: name, existed: true, content: content, mode: info.Mode().Perm()}, nil
}

func (j *Journal) makeParents(dir string) error {
	if dir == "." {
		return nil
	}
	if err := j.makeParents(path.Dir(dir)); err != nil {
		return err
	}
	_, err := j.root.Lstat(dir)
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := j.root.Mkdir(dir, dirMode); err != nil {
		return err
	}
	j.entries = append(j.entries, journalEntry{path: dir, createdDir: true})
	return nil
}

// Rollback undoes the journal in reverse order; entries it fails to restore stay for a retry.
func (j *Journal) Rollback() error {
	var failed []journalEntry
	var errs []error
	for _, e := range slices.Backward(j.entries) {
		if err := j.undo(e); err != nil {
			failed = append(failed, e)
			errs = append(errs, fmt.Errorf("restore %s: %w", e.path, err))
		}
	}
	slices.Reverse(failed)
	j.entries = failed
	return errors.Join(errs...)
}

func (j *Journal) undo(e journalEntry) error {
	switch {
	case e.createdDir:
		return j.removeIfEmpty(e.path)
	case e.existed:
		return writeFile(j.root, e.path, e.content, e.mode)
	}
	err := j.root.Remove(e.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (j *Journal) removeIfEmpty(dir string) error {
	d, err := j.root.Open(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	names, err := d.Readdirnames(1)
	_ = d.Close()
	if len(names) > 0 {
		return nil
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return j.root.Remove(dir)
}

// writeFile replaces name through a temp file in its directory so a reader never sees a partial file.
func writeFile(root *os.Root, name string, content []byte, mode fs.FileMode) error {
	temp := path.Join(path.Dir(name), "."+path.Base(name)+".solomonar-tmp")
	err := root.WriteFile(temp, content, mode)
	if err == nil {
		err = root.Chmod(temp, mode) // WriteFile's mode passes through the umask
	}
	if err == nil {
		err = root.Rename(temp, name)
	}
	if err != nil {
		_ = root.Remove(temp)
	}
	return err
}
