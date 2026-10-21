package project

// Action is what a plan does to one path (I §3.3).
type Action string

// Plan actions; Apply acts only on create, update, and remove.
const (
	ActionCreate    Action = "create"
	ActionUpdate    Action = "update"
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
type Plan struct{ Changes []Change }

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
