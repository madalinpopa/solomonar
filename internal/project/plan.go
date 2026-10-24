package project

import (
	"bytes"
	"slices"
)

// Action is what a plan does to one path (I section 3.3).
type Action string

// Plan actions; Apply acts only on create, update, append (written like update), and remove.
const (
	ActionCreate    Action = "create"
	ActionUpdate    Action = "update"
	ActionAppend    Action = "append"
	ActionUnchanged Action = "unchanged"
	ActionKeep      Action = "keep"
	ActionRemove    Action = "remove"
	ActionConflict  Action = "conflict"
	ActionSkip      Action = "skip"
)

// Change is one planned action on a repository-relative path.
type Change struct {
	Path    string
	Action  Action
	Owner   string
	Content []byte
	Reason  string
}

// Plan is the ordered list of changes a command applies.
type Plan struct {
	Changes []Change
	// Prune lists directories Apply removes after the changes when they are empty, in order.
	Prune []string
}

// Conflicts returns the conflict changes in plan order.
func (p Plan) Conflicts() []Change {
	var conflicts []Change
	for _, c := range p.Changes {
		if c.Action == ActionConflict {
			conflicts = append(conflicts, c)
		}
	}
	return conflicts
}

// Changed returns the first path, in plan order, whose change differs in other or is in only one plan.
func (p Plan) Changed(other Plan) (string, bool) {
	for _, c := range p.Changes {
		i := slices.IndexFunc(other.Changes, func(o Change) bool { return o.Path == c.Path })
		if i < 0 {
			return c.Path, true
		}
		o := other.Changes[i]
		if o.Action != c.Action || o.Reason != c.Reason || !bytes.Equal(o.Content, c.Content) {
			return c.Path, true
		}
	}
	for _, o := range other.Changes {
		if !slices.ContainsFunc(p.Changes, func(c Change) bool { return c.Path == o.Path }) {
			return o.Path, true
		}
	}
	return "", false
}
