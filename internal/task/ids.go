package task

import (
	"fmt"
	"strconv"
	"strings"
)

func nextTaskID(reg *Registry) string {
	highest := 0
	for id := range reg.Tasks {
		if number, err := strconv.Atoi(strings.TrimPrefix(id, "W-")); err == nil {
			highest = max(highest, number)
		}
	}
	return fmt.Sprintf("W-%03d", highest+1)
}

// issueBlockerID never reuses a cleared blocker's number (I UC-15).
func (r *Record) issueBlockerID() string {
	r.LastBlocker++
	return "B-" + strconv.Itoa(r.LastBlocker)
}
