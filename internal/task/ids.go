package task

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

func nextTaskID(reg *Registry) string {
	highest := 0
	for id := range reg.Tasks {
		highest = max(highest, taskNumber(id))
	}
	return fmt.Sprintf("W-%03d", highest+1)
}

// IDs returns the task IDs in numeric order.
func (r *Registry) IDs() []string {
	return slices.SortedFunc(maps.Keys(r.Tasks), func(a, b string) int {
		return cmp.Or(cmp.Compare(taskNumber(a), taskNumber(b)), cmp.Compare(a, b))
	})
}

func taskNumber(id string) int {
	number, _ := strconv.Atoi(strings.TrimPrefix(id, "W-"))
	return number
}

// issueBlockerID never reuses a cleared blocker's number (I UC-15).
func (r *Record) issueBlockerID() string {
	r.LastBlocker++
	return "B-" + strconv.Itoa(r.LastBlocker)
}
