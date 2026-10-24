package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

type uninstallFlags struct {
	keep, overwrite []string
	yes, dryRun     bool
}

type uninstallData struct {
	Project       string     `json:"project"`
	AlreadyAbsent bool       `json:"already_absent,omitzero"`
	Files         []initFile `json:"files"`
	Preserved     []string   `json:"preserved"`
	Next          []string   `json:"next,omitzero"`
	DryRun        bool       `json:"dry_run,omitzero"`
}

// uninstallRun is the removal planned from the manifest bytes it was read from; unresolved is the
// same plan without --keep and --overwrite.
type uninstallRun struct {
	root       string
	manifest   []byte
	existing   *project.Manifest
	unresolved project.Plan
	project.Uninstall
}

func uninstallCommand() command {
	return command{
		name:    "uninstall",
		summary: "Remove the framework installation from the project",
		usage:   "solomonar uninstall [--keep <path>]... [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]",
		description: "Remove the framework core, the installed catalog extensions, the runtime adapters, the managed\n" +
			"part of AGENTS.md, the docs/README.md index line, and the manifest from this checkout. --keep\n" +
			"leaves a conflicting file in place, --overwrite removes it; --dry-run prints the plan and writes\n" +
			"nothing. Project extensions, decisions, project notes, and the registry are kept.",
		bind: func(fs *flag.FlagSet) runFunc {
			var f uninstallFlags
			fs.Func("keep", "", appendList(&f.keep))
			fs.Func("overwrite", "", appendList(&f.overwrite))
			fs.BoolVar(&f.yes, "y", false, "")
			fs.BoolVar(&f.yes, "yes", false, "")
			fs.BoolVar(&f.dryRun, "dry-run", false, "")
			return f.run
		},
	}
}

func (f *uninstallFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	if len(args) > 0 {
		return output{}, unexpectedArgument(args[0])
	}
	repo, err := openRepo(ctx, env, flags, flags.dir, "run inside a Git repository or pass -C <path>")
	if err != nil {
		return output{}, err
	}
	root, err := os.OpenRoot(repo.Dir)
	if err != nil {
		return output{}, fmt.Errorf("open worktree: %w", err)
	}
	defer root.Close()
	framework, err := frameworkVersion(env.Assets)
	if err != nil {
		return output{}, err
	}
	u, err := f.plan(root, repo.Dir, framework)
	switch {
	case err != nil:
		return output{}, err
	case u.existing == nil:
		return u.absent(), nil
	}
	if conflicts := u.Plan.Conflicts(); len(conflicts) > 0 {
		return output{text: u.planText()}, fileConflict("uninstall", "--overwrite <path> to remove it", conflicts)
	}
	if f.dryRun {
		return u.dryRun(), nil
	}
	return confirmed(env, flags, "uninstall", f.yes, u.confirmation(), func() (output, error) {
		if err := f.recheck(root, u); err != nil {
			return output{}, err
		}
		return u.apply(ctx, root)
	})
}

func (f *uninstallFlags) resolutions() project.Resolutions {
	return project.Resolutions{Keep: f.keep, Overwrite: f.overwrite}
}

// plan reads the manifest directly, so an absent one is the no-op, and plans the removal (UC-27).
func (f *uninstallFlags) plan(root *os.Root, dir, framework string) (uninstallRun, error) {
	u := uninstallRun{root: dir}
	var err error
	if u.manifest, err = project.ReadUninstallManifest(root); err != nil {
		return uninstallRun{}, err
	}
	if u.manifest != nil {
		if u.existing, err = project.DecodeManifest(u.manifest, framework); err != nil {
			return uninstallRun{}, err
		}
	}
	if u.Uninstall, err = project.PlanUninstall(root, u.existing, f.resolutions()); err != nil || u.existing == nil {
		return u, err
	}
	unresolved, err := project.PlanUninstall(root, u.existing, project.Resolutions{})
	u.unresolved = unresolved.Plan
	return u, err
}

// recheck plans again after the confirmation and rejects any change to the manifest or a planned
// path, unresolved conflicts included, before the first write (UC-27).
func (f *uninstallFlags) recheck(root *os.Root, u uninstallRun) error {
	manifest, err := project.ReadUninstallManifest(root)
	if _, unsafe := errors.AsType[*project.Rejection](err); unsafe || err == nil && !bytes.Equal(manifest, u.manifest) {
		return changedSincePlan(project.ManifestPath)
	}
	if err != nil {
		return err
	}
	unresolved, err := project.PlanUninstall(root, u.existing, project.Resolutions{})
	if err != nil {
		return err
	}
	if path, changed := u.unresolved.Changed(unresolved.Plan); changed {
		return changedSincePlan(path)
	}
	resolved, err := project.PlanUninstall(root, u.existing, f.resolutions())
	if err != nil {
		return err
	}
	if path, changed := u.Plan.Changed(resolved.Plan); changed {
		return changedSincePlan(path)
	}
	return nil
}

func changedSincePlan(path string) *Error {
	return &Error{
		Code: "file_conflict", Exit: 4, Message: "uninstall found a change since the plan was shown",
		Check: "the planned files are unchanged since the plan was shown",
		Found: path + " changed since the plan was shown", Fix: "rerun solomonar uninstall",
	}
}

// apply removes the installation; project.Apply rolls every change back on failure.
func (u uninstallRun) apply(ctx context.Context, root *os.Root) (output, error) {
	if _, err := project.Apply(ctx, root, u.Plan); err != nil {
		return output{}, rollbackFailure(ctx, err)
	}
	data := u.data()
	data.Next = []string{"Review the removal with git status, then commit it."}
	text := fmt.Sprintf("Uninstalled solomonar from %s\n  %s\n", u.root, counts(u.Plan.Changes))
	if kept := slices.DeleteFunc(slices.Clone(data.Preserved), func(p string) bool { return p == task.Ref }); len(kept) > 0 {
		text += "  kept: " + strings.Join(kept, ", ") + "\n"
	}
	text += "  The registry (" + task.Ref + ") and other worktrees are unchanged.\n\nNext:\n  1. " + data.Next[0] + "\n"
	return output{data: data, text: text, warnings: u.warnings()}, nil
}

// rollbackFailure names the paths a failed rollback left changed, in place of "nothing was written".
func rollbackFailure(ctx context.Context, err error) error {
	rollback, ok := errors.AsType[*project.RollbackError](err)
	if !ok {
		return err
	}
	failure := *asCLIError(ctx, "uninstall", err)
	failure.Found = "not restored: " + strings.Join(rollback.Paths, ", ")
	failure.Fix = "restore these paths by hand, for example with git restore, then rerun solomonar uninstall"
	failure.Unrestored = rollback.Paths
	return &failure
}

// planText is every file line, as --dry-run and a conflict print them.
func (u uninstallRun) planText() string {
	return "Plan\n" + planLines(u.Plan.Changes)
}

// confirmation is the UC-9 grouped plan shown before the prompt.
func (u uninstallRun) confirmation() string {
	r := initRun{framework: u.existing.FrameworkVersion, plan: u.Plan, models: map[string]string{}}
	for name, runtime := range u.existing.Runtimes {
		r.models[name] = runtime.AdversarialModel
	}
	for _, name := range slices.Sorted(maps.Keys(u.existing.Extensions)) {
		r.removed = append(r.removed, project.CatalogExtension{Name: name})
	}
	return r.changedPlan("Plan\n")
}

func (u uninstallRun) dryRun() output {
	data := u.data()
	data.DryRun = true
	text := u.planText() + fmt.Sprintf("Would uninstall solomonar from %s\n  %s\n", u.root, counts(u.Plan.Changes))
	return output{data: data, text: text, warnings: u.warnings()}
}

func (u uninstallRun) data() uninstallData {
	preserved := []string{task.Ref}
	for _, c := range u.Plan.Changes {
		if c.Action == project.ActionSkip || c.Action == project.ActionKeep || c.Action == project.ActionUpdate {
			preserved = append(preserved, c.Path)
		}
	}
	preserved = slices.Compact(slices.Sorted(slices.Values(slices.Concat(preserved, u.Review))))
	return uninstallData{Project: u.root, Files: fileEntries(u.Plan.Changes), Preserved: preserved}
}

// warnings name the kept managed files and the framework references left for manual cleanup.
func (u uninstallRun) warnings() []warning {
	var warnings []warning
	for _, c := range u.Plan.Changes {
		if c.Action == project.ActionKeep {
			warnings = append(warnings, warning{
				Code: "manual_reference", Message: c.Path + " is kept and no longer managed; remove it by hand if unneeded",
			})
		}
	}
	for _, path := range u.Review {
		message := path + " still mentions docs/solomonar; review it by hand"
		if path == "CLAUDE.md" {
			message = "CLAUDE.md still imports @AGENTS.md, which uninstall removes; edit it by hand"
		}
		warnings = append(warnings, warning{Code: "manual_reference", Message: message})
	}
	return warnings
}

func (u uninstallRun) absent() output {
	data := uninstallData{Project: u.root, AlreadyAbsent: true, Files: []initFile{}, Preserved: []string{}, Next: []string{}}
	return output{data: data, text: "No Solomonar manifest; nothing to uninstall automatically.\n"}
}
