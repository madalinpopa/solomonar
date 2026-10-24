package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/project"
)

type runtimeListData struct {
	Runtimes []runtimeRow `json:"runtimes"`
}

type runtimeRow struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	OnPath   bool     `json:"on_path"`
	Adapters []string `json:"adapters"`
}

// runtimeAdapters lists the supported runtimes in plan order with their adapters (I section 4.2).
func runtimeAdapters() []runtimeRow {
	return []runtimeRow{
		{Name: "claude", Adapters: []string{"CLAUDE.md", ".claude/skills", ".claude/agents"}},
		{Name: "codex", Adapters: []string{"AGENTS.md", ".agents/skills", ".codex/agents"}},
		{Name: "copilot", Adapters: []string{"AGENTS.md", ".github/skills", ".github/agents"}},
	}
}

func runtimeListCommand() command {
	return command{
		name:    "runtime list",
		summary: "List the supported runtimes and their adapters",
		usage:   "solomonar runtime list [--json]",
		description: "List claude, codex, and copilot with whether the project installs each, whether its executable\n" +
			"is on PATH, and its adapter locations. Works without a manifest. Read-only.",
		bind: withoutFlags(runRuntimeList),
	}
}

func runRuntimeList(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	if len(args) > 0 {
		return output{}, unexpectedArgument(args[0])
	}
	repo, err := openRepo(ctx, env, flags, flags.dir, "run inside a Git repository or pass -C <path>")
	if err != nil {
		return output{}, err
	}
	manifest, err := readManifest(env, repo.Dir)
	if err != nil {
		return output{}, err
	}
	rows := runtimeAdapters()
	for i := range rows {
		rows[i].Status = "available"
		if manifest != nil {
			if _, ok := manifest.Runtimes[rows[i].Name]; ok {
				rows[i].Status = "installed"
			}
		}
		rows[i].OnPath = git.OnPath(rows[i].Name)
	}
	return output{data: runtimeListData{Runtimes: rows}, text: runtimeTable(rows)}, nil
}

func runtimeTable(rows []runtimeRow) string {
	var b strings.Builder
	table := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "RUNTIME\tSTATUS\tON PATH\tADAPTER")
	for _, r := range rows {
		onPath := "no"
		if r.OnPath {
			onPath = "yes"
		}
		adapters := strings.Join(r.Adapters, ", ")
		adapters = strings.Replace(adapters, "AGENTS.md", "AGENTS.md (native)", 1)
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", r.Name, r.Status, onPath, adapters)
	}
	_ = table.Flush()
	return b.String()
}

// runtimeFlags are the flags of runtime add and remove (UC-5).
type runtimeFlags struct {
	keep, overwrite []string
	yes, dryRun     bool
}

// runtimeAddFlags adds --adversarial-model, recording whether it was given.
type runtimeAddFlags struct {
	runtimeFlags
	model    string
	modelSet bool
}

type runtimeData struct {
	Project          string     `json:"project"`
	Runtime          string     `json:"runtime"`
	AdversarialModel string     `json:"adversarial_model,omitzero"`
	Files            []initFile `json:"files"`
	Next             []string   `json:"next,omitzero"`
	DryRun           bool       `json:"dry_run,omitzero"`
}

// runtimeRun is an init run over the manifest's choices with one runtime added or removed.
type runtimeRun struct {
	initRun
	name, verb   string
	notInstalled bool
}

func runtimeAddCommand() command {
	return command{
		name:    "runtime add",
		summary: "Install a runtime's adapters",
		usage: "solomonar runtime add <claude|codex|copilot> [--adversarial-model <model>] [--keep <path>]...\n" +
			"                             [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]",
		description: "Generate the runtime's skill and role stubs for the core and the installed extensions, and\n" +
			"record it in the manifest; --adversarial-model sets its adversarial reviewer's model. Other choices\n" +
			"stay as recorded. --keep or --overwrite resolves a conflicting file; --dry-run prints the plan and\n" +
			"writes nothing. No hook or runtime settings file is written.",
		bind: func(fs *flag.FlagSet) runFunc {
			var f runtimeAddFlags
			f.bind(fs)
			fs.Func("adversarial-model", "", func(model string) error {
				f.model, f.modelSet = model, true
				return nil
			})
			return f.run
		},
	}
}

func runtimeRemoveCommand() command {
	return command{
		name:    "runtime remove",
		summary: "Remove a runtime's adapters",
		usage: "solomonar runtime remove <claude|codex|copilot> [--keep <path>]... [--overwrite <path>]...\n" +
			"                                [-y|--yes] [--dry-run] [--json]",
		description: "Delete the runtime's managed adapter files and drop it and its model from the manifest. Other\n" +
			"choices, unmanaged files, and the framework stay. --keep untracks an edited file, --overwrite\n" +
			"deletes it; --dry-run prints the plan and writes nothing.",
		bind: func(fs *flag.FlagSet) runFunc {
			var f runtimeFlags
			f.bind(fs)
			return f.remove
		},
	}
}

func (f *runtimeFlags) bind(fs *flag.FlagSet) {
	fs.Func("keep", "", func(path string) error {
		f.keep = append(f.keep, path)
		return nil
	})
	fs.Func("overwrite", "", func(path string) error {
		f.overwrite = append(f.overwrite, path)
		return nil
	})
	fs.BoolVar(&f.yes, "y", false, "")
	fs.BoolVar(&f.yes, "yes", false, "")
	fs.BoolVar(&f.dryRun, "dry-run", false, "")
}

func (f *runtimeAddFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	name, err := runtimeName(args)
	if err != nil {
		return output{}, err
	}
	if f.modelSet && f.model == "" {
		return output{}, &project.Rejection{
			Code:  "invalid_value",
			Check: "--adversarial-model names a model",
			Found: "empty model",
			Fix:   "pass a model or omit --adversarial-model",
		}
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
	r, err := f.plan(env, root, repo.Dir, "add", name, func(existing *project.Manifest) ([]string, map[string]string, error) {
		runtimes := slices.Compact(slices.Sorted(slices.Values(append(selectRuntimes(nil, existing), name))))
		var requested map[string]string
		if f.modelSet {
			requested = map[string]string{name: f.model}
		}
		models, err := selectModels(requested, runtimes, existing)
		return runtimes, models, err
	})
	if err != nil {
		return output{}, err
	}
	return f.finish(ctx, env, flags, root, r)
}

func (f *runtimeFlags) remove(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	name, err := runtimeName(args)
	if err != nil {
		return output{}, err
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
	r, err := f.plan(env, root, repo.Dir, "remove", name, func(existing *project.Manifest) ([]string, map[string]string, error) {
		runtimes := slices.DeleteFunc(selectRuntimes(nil, existing), func(runtime string) bool { return runtime == name })
		models, err := selectModels(nil, runtimes, existing)
		return runtimes, models, err
	})
	if err != nil {
		return output{}, err
	}
	return f.finish(ctx, env, flags, root, r)
}

// runtimeName checks for exactly one argument naming a supported runtime (UC-5).
func runtimeName(args []string) (string, error) {
	switch {
	case len(args) == 0:
		return "", invalidUsage(errors.New("runtime name is required"))
	case len(args) > 1:
		return "", unexpectedArgument(args[1])
	}
	name := args[0]
	if !slices.ContainsFunc(runtimeAdapters(), func(r runtimeRow) bool { return r.Name == name }) {
		return "", &project.Rejection{
			Code:  "unknown_runtime",
			Check: "<runtime> names a known runtime",
			Found: fmt.Sprintf("%q is not a runtime; known: claude, codex, copilot", name),
			Fix:   "pass claude, codex, or copilot",
		}
	}
	return name, nil
}

// plan checks the project in the UC-5 order, then plans the manifest's choices as choose changes them.
func (f *runtimeFlags) plan(env Env, root *os.Root, dir, verb, name string,
	choose func(existing *project.Manifest) ([]string, map[string]string, error),
) (runtimeRun, error) {
	framework, err := frameworkVersion(env.Assets)
	if err != nil {
		return runtimeRun{}, err
	}
	existing, err := initializedManifest(root, framework, dir)
	if err != nil {
		return runtimeRun{}, err
	}
	if err := currentFramework(existing, framework); err != nil {
		return runtimeRun{}, err
	}
	catalog, err := project.LoadCatalog(env.Assets)
	if err != nil {
		return runtimeRun{}, err
	}
	store, err := project.LoadStore(root.FS(), env.Assets, catalog, existing)
	if err != nil {
		return runtimeRun{}, err
	}
	extensions, err := installedExtensions(catalog, store, existing, nil)
	if err != nil {
		return runtimeRun{}, err
	}
	runtimes, models, err := choose(existing)
	if err != nil {
		return runtimeRun{}, err
	}
	r := runtimeRun{
		root: dir, framework: framework, runtimes: runtimes, models: models, extensions: extensions,
		decider: project.Decider{Command: "runtime " + verb}, name: name, verb: verb,
	}
	resolutions := project.Resolutions{Keep: f.keep, Overwrite: f.overwrite}
	if _, installed := existing.Runtimes[name]; verb == "remove" && !installed {
		r.notInstalled = true
		_, _, err = project.Classify(root, nil, nil, resolutions)
		return r, err
	}
	r.plan, r.decisions, err = r.planFiles(env, root, existing, existing.Project, "", resolutions)
	return r, err
}

// finish rejects conflicts, then prints a dry run or a no-op, or applies the confirmed plan.
func (f *runtimeFlags) finish(ctx context.Context, env Env, flags sharedFlags, root *os.Root, r runtimeRun,
) (output, error) {
	command := "runtime " + r.verb
	if conflicts := r.plan.Conflicts(); len(conflicts) > 0 {
		return output{text: "Plan\n" + planLines(r.plan.Changes)},
			fileConflict(command, "--overwrite <path> to replace it", conflicts)
	}
	switch {
	case r.notInstalled:
		return r.noOp(), nil
	case f.dryRun:
		return r.dryRun(), nil
	case !slices.ContainsFunc(r.plan.Changes, writes):
		return r.noOp(), nil
	}
	return confirmed(env, flags, command, f.yes, r.changedPlan("Plan\n"), func() (output, error) {
		return r.apply(ctx, root)
	})
}

func (r runtimeRun) apply(ctx context.Context, root *os.Root) (output, error) {
	if _, err := project.Apply(ctx, root, r.plan); err != nil {
		return output{}, err
	}
	data := r.data()
	data.Next = []string{"git add " + strings.Join(stagedPaths(r.plan.Changes), " ") +
		fmt.Sprintf(" && git commit -m \"chore: %s solomonar runtime %s\"", r.verb, r.name)}
	text := fmt.Sprintf("%s runtime %s in %s\n  %s\n\nNext:\n  1. %s\n", r.past(), r.name, r.root,
		counts(r.plan.Changes), data.Next[0])
	if r.verb == "add" {
		text += initRun{root: r.root, runtimes: []string{r.name}}.sessionStep()
	}
	return output{data: data, text: text}, nil
}

func (r runtimeRun) past() string {
	if r.verb == "remove" {
		return "Removed"
	}
	return "Added"
}

// stagedPaths lists the written and removed paths by top-level directory, except files under .github.
func stagedPaths(changes []project.Change) []string {
	var paths []string
	for _, c := range changes {
		if !writes(c) {
			continue
		}
		path := c.Path
		if dir, _, nested := strings.Cut(path, "/"); nested && dir != ".github" {
			path = dir
		}
		if !slices.Contains(paths, path) {
			paths = append(paths, path)
		}
	}
	return paths
}

func (r runtimeRun) noOp() output {
	data := r.data()
	data.Next = []string{}
	if r.notInstalled {
		return output{data: data, text: fmt.Sprintf("Not installed: %s; nothing to do.\n", r.name)}
	}
	text := fmt.Sprintf("Already installed: %s; nothing to do.\n  %s\n", r.name, counts(r.plan.Changes))
	return output{data: data, text: text}
}

func (r runtimeRun) dryRun() output {
	data := r.data()
	data.DryRun = true
	text := "Plan\n" + planLines(r.plan.Changes) +
		fmt.Sprintf("Would %s runtime %s in %s\n  %s\n", r.verb, r.name, r.root, counts(r.plan.Changes))
	return output{data: data, text: text}
}

func (r runtimeRun) data() runtimeData {
	return runtimeData{
		Project: r.root, Runtime: r.name, AdversarialModel: r.models[r.name], Files: fileEntries(r.plan.Changes),
	}
}
