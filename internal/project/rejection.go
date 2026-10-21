package project

import "fmt"

// Rejection is an expected failure with a stable code and guidance for the user.
type Rejection struct {
	Code  string
	Check string
	Found string
	Fix   string
}

func (r *Rejection) Error() string { return r.Code + ": " + r.Found }

func invalidManifest(format string, args ...any) *Rejection {
	return &Rejection{
		Code:  "manifest_invalid",
		Check: "manifest schema",
		Found: fmt.Sprintf(format, args...),
		Fix:   "inspect .solomonar/manifest.json; never edit it by hand",
	}
}

func outdated(check, found string) *Rejection {
	return &Rejection{Code: "cli_outdated", Check: check, Found: found, Fix: "install a newer solomonar release"}
}
