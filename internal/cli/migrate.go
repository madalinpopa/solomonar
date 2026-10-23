package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/madalinpopa/solomonar/internal/task"
)

type migrateData struct {
	Migrated    bool     `json:"migrated"`
	Schema      int      `json:"schema"`
	Tasks       int      `json:"tasks"`
	Unconfirmed []string `json:"unconfirmed"`
	Commit      string   `json:"commit"`
	Pushed      bool     `json:"pushed"`
}

func taskMigrateCommand() command {
	return command{
		name:    "task migrate",
		summary: "Rewrite a schema-1 registry as one file per task",
		usage:   "solomonar task migrate [--json]",
		description: "Rewrite a schema-1 registry as schema 2, one commit on top of its history.\n" +
			"It infers no base, worktree, or confirmation; confirm each active task after inspecting it.",
		bind: withoutFlags(runTaskMigrate),
	}
}

func runTaskMigrate(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	if len(args) > 0 {
		return output{}, unexpectedArgument(args[0])
	}
	store, _, err := openStore(ctx, env, flags)
	if err != nil {
		return output{}, err
	}
	reg, commit, migrated, err := store.Migrate(ctx)
	if err != nil {
		return output{}, err
	}
	data := migrateData{
		Migrated: migrated, Schema: task.SchemaVersion, Tasks: len(reg.Tasks), Unconfirmed: unconfirmed(reg),
		Commit: commit.SHA, Pushed: commit.Pushed,
	}
	if !migrated {
		text := fmt.Sprintf("Registry already at schema %d; nothing to migrate.\n", task.SchemaVersion)
		return output{data: data, text: text}, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Migrated registry to schema %d: %d tasks, one tasks/<ID>.json each\n", task.SchemaVersion, len(reg.Tasks))
	if len(data.Unconfirmed) > 0 {
		fmt.Fprintf(&b, "  unconfirmed  %s\n", strings.Join(data.Unconfirmed, ", "))
		b.WriteString("  next         inspect each with solomonar task status <ID>, then solomonar task confirm <ID>\n")
	}
	b.WriteString(registryLine(store, commit) + "\n")
	return output{data: data, text: b.String(), warnings: outputWarnings(commit.Warnings)}, nil
}

// unconfirmed lists the tasks past setup that need task confirm before their next stage change.
func unconfirmed(reg *task.Registry) []string {
	ids := []string{}
	for _, id := range reg.IDs() {
		r := reg.Tasks[id]
		if r.Owner != "" && r.Status != task.StatusDone && r.Stage != task.StageSetup && !r.Confirmed {
			ids = append(ids, id)
		}
	}
	return ids
}
