package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/madalinpopa/solomonar/internal/project"
)

type upgradeFlags struct {
	keep, takeNew []string
	yes, dryRun   bool
}

type upgradeData struct {
	Project string     `json:"project"`
	From    string     `json:"from"`
	To      string     `json:"to"`
	Files   []initFile `json:"files"`
	Next    []string   `json:"next,omitzero"`
	DryRun  bool       `json:"dry_run,omitzero"`
}

// upgradeRun is an init run over the manifest's choices that moves the framework from one version.
type upgradeRun struct {
	initRun
	from string
}

func upgradeCommand() command {
	return command{
		name:    "upgrade",
		summary: "Move the project to this CLI's framework version",
		usage:   "solomonar upgrade [--keep <path>]... [--take-new <path>]... [-y|--yes] [--dry-run] [--json]",
		description: "Update the managed files to the framework embedded in this CLI, with the choices the manifest\n" +
			"records, and remove the managed files no longer offered. --keep keeps the project's version of a\n" +
			"conflicting file, --take-new takes the new one; --dry-run prints the plan and writes nothing.\n" +
			"Decision documents, project extensions, and the registry are never touched.",
		bind: func(fs *flag.FlagSet) runFunc {
			var f upgradeFlags
			fs.Func("keep", "", func(path string) error {
				f.keep = append(f.keep, path)
				return nil
			})
			fs.Func("take-new", "", func(path string) error {
				f.takeNew = append(f.takeNew, path)
				return nil
			})
			fs.BoolVar(&f.yes, "y", false, "")
			fs.BoolVar(&f.yes, "yes", false, "")
			fs.BoolVar(&f.dryRun, "dry-run", false, "")
			return f.run
		},
	}
}

func (f *upgradeFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
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
	u, err := f.plan(env, root, repo.Dir)
	if err != nil {
		return output{}, err
	}
	if conflicts := u.plan.Conflicts(); len(conflicts) > 0 {
		return output{text: u.planText()},
			fileConflict("upgrade", "--take-new <path> to take the new version", conflicts)
	}
	switch {
	case f.dryRun:
		return u.dryRun(), nil
	case u.from == u.framework && !slices.ContainsFunc(u.plan.Changes, writes):
		return u.noOp(), nil
	}
	return confirmed(env, flags, "upgrade", f.yes, u.changedPlan(u.header()), func() (output, error) {
		return u.apply(ctx, root)
	})
}

// confirmed applies a plan with --yes, or once the user confirms it in a terminal (UC-6, UC-9).
func confirmed(env Env, flags sharedFlags, command string, yes bool, plan string, apply func() (output, error),
) (output, error) {
	switch {
	case env.Interactive && !yes && !flags.json:
		ok, err := confirmPlan(env, plan)
		if err != nil {
			return output{}, err
		}
		if !ok {
			return output{text: cancelled}, nil
		}
	case !yes:
		return output{}, &Error{
			Code: "interactive_required", Exit: exitUsage, Message: command + " needs a confirmation",
			Fix: "pass --yes to accept the plan, or --dry-run to print it",
		}
	}
	return apply()
}

// plan reuses the manifest's runtimes, models, extensions, and decisions (UC-9).
func (f *upgradeFlags) plan(env Env, root *os.Root, dir string) (upgradeRun, error) {
	framework, err := frameworkVersion(env.Assets)
	if err != nil {
		return upgradeRun{}, err
	}
	existing, err := initializedManifest(root, framework, dir)
	if err != nil {
		return upgradeRun{}, err
	}
	catalog, err := project.LoadCatalog(env.Assets)
	if err != nil {
		return upgradeRun{}, err
	}
	store, err := project.LoadStore(root.FS(), env.Assets, catalog, existing)
	if err != nil {
		return upgradeRun{}, err
	}
	extensions, err := selectExtensions(catalog, store, nil, existing, nil)
	if err != nil {
		return upgradeRun{}, err
	}
	runtimes := selectRuntimes(nil, existing)
	models, err := selectModels(nil, runtimes, existing)
	if err != nil {
		return upgradeRun{}, err
	}
	u := upgradeRun{
		root: dir, framework: framework, runtimes: runtimes, models: models, extensions: extensions,
		from: existing.FrameworkVersion,
	}
	u.plan, u.decisions, err = u.planFiles(env, root, existing, existing.Project, "",
		project.Resolutions{Keep: f.keep, Overwrite: f.takeNew, OverwriteFlag: "--take-new"})
	return u, err
}

// initializedManifest reads the manifest that upgrade and extension add|remove need (I section 3.6).
func initializedManifest(root *os.Root, framework, dir string) (*project.Manifest, error) {
	existing, err := project.ReadManifest(root, framework)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, &project.Rejection{
			Code:  "not_initialized",
			Check: "the project is initialized",
			Found: "no " + project.ManifestPath + " in " + dir,
			Fix:   "run solomonar init",
		}
	}
	return existing, nil
}

func (u upgradeRun) header() string {
	return fmt.Sprintf("Upgrade framework %s → %s\n", u.from, u.framework)
}

// planText is the header and every file line, as --dry-run and a conflict print them.
func (u upgradeRun) planText() string {
	return u.header() + planLines(u.plan.Changes)
}

// apply writes the plan; project.Apply rolls every written file back on failure.
func (u upgradeRun) apply(ctx context.Context, root *os.Root) (output, error) {
	if _, err := project.Apply(ctx, root, u.plan); err != nil {
		return output{}, err
	}
	data := u.data()
	commit := "git add " + strings.Join(u.generatedPaths(), " ") +
		fmt.Sprintf(" && git commit -m \"chore: upgrade solomonar to %s\"", u.framework)
	data.Next = []string{commit, "Running tasks pick up the new framework after they update from main."}
	text := fmt.Sprintf("Upgraded solomonar %s → %s in %s\n  %s\n\nNext:\n  1. %s\n  2. %s\n", u.from, u.framework,
		u.root, counts(u.plan.Changes), data.Next[0], data.Next[1])
	return output{data: data, text: text}, nil
}

func (u upgradeRun) noOp() output {
	data := u.data()
	data.Next = []string{}
	text := fmt.Sprintf("Already at solomonar %s; nothing to do.\n  %s\n", u.framework, counts(u.plan.Changes))
	return output{data: data, text: text}
}

func (u upgradeRun) dryRun() output {
	data := u.data()
	data.DryRun = true
	text := u.planText() + fmt.Sprintf("Would upgrade solomonar %s → %s in %s\n  %s\n", u.from, u.framework, u.root,
		counts(u.plan.Changes))
	return output{data: data, text: text}
}

func (u upgradeRun) data() upgradeData {
	return upgradeData{Project: u.root, From: u.from, To: u.framework, Files: fileEntries(u.plan.Changes)}
}
