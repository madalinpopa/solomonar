package task

import "fmt"

// Rejection is an expected failure with a stable code and guidance for the user.
type Rejection struct {
	Code  string
	Check string
	Found string
	Fix   string
}

func (r *Rejection) Error() string { return r.Code + ": " + r.Found }

func invalidRegistry(format string, args ...any) *Rejection {
	return &Rejection{
		Code:  "registry_invalid",
		Check: "registry schema",
		Found: fmt.Sprintf(format, args...),
		Fix:   "inspect with git show refs/heads/workflow-state:state.json; never edit it by hand",
	}
}
