package cli

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

type initFlags struct {
	requested       project.Requested
	objective       string
	keep, overwrite []string
	models          []string
	yes, dryRun     bool
	gitInit         bool
}

type initData struct {
	Project          string             `json:"project"`
	FrameworkVersion string             `json:"framework_version"`
	Choices          initChoices        `json:"choices"`
	Decisions        []project.Decision `json:"decisions"`
	Files            []initFile         `json:"files"`
	Registry         registryState      `json:"registry"`
	GitInit          bool               `json:"git_init"`
	FirstCommit      bool               `json:"first_commit_required"`
	Next             []string           `json:"next,omitzero"`
	DryRun           bool               `json:"dry_run,omitzero"`
}

type initChoices struct {
	Runtimes         []string            `json:"runtimes"`
	Extensions       map[string][]string `json:"extensions"`
	AdversarialModel map[string]string   `json:"adversarial_model"`
}

type initFile struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Owner  string `json:"owner"`
	Reason string `json:"reason,omitempty"`
}

type registryState struct {
	Ref    string `json:"ref"`
	Commit string `json:"commit,omitempty"`
	Action string `json:"action"`
	Pushed *bool  `json:"pushed,omitzero"`
}

// initRun is what one init found: the repository, the plan, and the registry tip ("" when absent).
type initRun struct {
	root         string
	framework    string
	mainBranch   string
	plan         project.Plan
	tip          string
	gitInit      bool
	firstCommit  bool
	switchOrphan bool
	runtimes     []string
	models       map[string]string
	extensions   []project.CatalogExtension
	decisions    []project.Decision
	warnings     []warning
}

func initCommand() command {
	return command{
		name:    "init",
		summary: "Install the framework core and create the registry",
		usage: "solomonar init [<dir>] [--name <name>] [--objective <text>] [--runtime <claude|codex|none>]...\n" +
			"                      [--ext <name|none>]... [--adversarial-model <runtime>=<model>]...\n" +
			"                      [--main-branch <name>] [--remote <name>] [--keep <path>]... [--overwrite <path>]...\n" +
			"                      [--git-init] [-y|--yes] [--dry-run] [--json]",
		description: "Install the framework core in the worktree containing <dir>, write .solomonar/manifest.json,\n" +
			"and create refs/heads/workflow-state when it is absent. A re-run with the same choices is a no-op;\n" +
			"--keep or --overwrite resolves a conflicting file; --dry-run prints the plan and writes nothing;\n" +
			"--git-init runs git init -b <main-branch> when <dir> is not a repository.",
		bind: func(fs *flag.FlagSet) runFunc {
			var f initFlags
			fs.StringVar(&f.requested.Name, "name", "", "")
			fs.StringVar(&f.objective, "objective", "", "")
			fs.Func("runtime", "", appendList(&f.requested.Runtimes))
			fs.Func("ext", "", appendList(&f.requested.Extensions))
			fs.Func("adversarial-model", "", func(value string) error {
				f.models = append(f.models, value)
				return nil
			})
			fs.StringVar(&f.requested.MainBranch, "main-branch", "", "")
			fs.StringVar(&f.requested.Remote, "remote", "", "")
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
			fs.BoolVar(&f.gitInit, "git-init", false, "")
			return f.run
		},
	}
}

// appendList collects a repeatable, comma-separated flag; a given flag leaves the list non-nil.
func appendList(list *[]string) func(string) error {
	return func(value string) error {
		*list = append(*list, strings.Split(value, ",")...)
		return nil
	}
}

func (f *initFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	if len(args) > 1 {
		return output{}, unexpectedArgument(args[1])
	}
	var err error
	if f.requested.Runtimes, err = knownRuntimes(f.requested.Runtimes); err != nil {
		return output{}, err
	}
	if f.requested.AdversarialModels, err = adversarialModels(f.models); err != nil {
		return output{}, err
	}
	catalog, err := project.LoadCatalog(env.Assets)
	if err != nil {
		return output{}, err
	}
	if f.requested.Extensions, err = requestedExtensions(catalog, f.requested.Extensions); err != nil {
		return output{}, err
	}
	dir := flags.dir
	if len(args) == 1 {
		dir = args[0]
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(flags.dir, dir)
		}
	}
	fix := "run `git init -b " + cmp.Or(f.requested.MainBranch, defaultMainBranch) + "` first, or rerun with --git-init"
	repo, err := openRepo(ctx, env, flags, dir, fix)
	gitInit := false
	if notRepo, ok := errors.AsType[*Error](err); ok && notRepo.Code == "not_a_repository" && f.gitInit {
		gitInit = true
		repo = &git.Repo{Dir: absDir(dir), Trace: traceWriter(env, flags)}
		err = checkVersion(ctx, repo.Trace)
	}
	if err != nil {
		return output{}, err
	}
	framework, err := frameworkVersion(env.Assets)
	if err != nil {
		return output{}, err
	}
	root, err := os.OpenRoot(repo.Dir)
	if err != nil {
		return output{}, fmt.Errorf("open worktree: %w", err)
	}
	defer root.Close()
	var ask *initPrompts
	if env.Interactive && !f.yes && !flags.json {
		ask = &initPrompts{prompter: newPrompter(env)}
	}
	return f.initialize(ctx, env, repo, gitInit, root, framework, catalog, ask)
}

// knownRuntimes returns the given runtimes sorted and unique; a given none requests an empty list.
func knownRuntimes(values []string) ([]string, error) {
	if values == nil {
		return nil, nil
	}
	for _, v := range values {
		if v != "claude" && v != "codex" && v != "none" {
			return nil, &project.Rejection{
				Code:  "unknown_runtime",
				Check: "--runtime names a known runtime",
				Found: fmt.Sprintf("%q is not a runtime; known: claude, codex", v),
				Fix:   "pass --runtime claude, codex, or none",
			}
		}
	}
	if !slices.Contains(values, "none") {
		return slices.Compact(slices.Sorted(slices.Values(values))), nil
	}
	if i := slices.IndexFunc(values, func(v string) bool { return v != "none" }); i >= 0 {
		return nil, &project.Rejection{
			Code:  "invalid_value",
			Check: "--runtime names runtimes or none",
			Found: "none cannot be combined with " + values[i],
			Fix:   "pass --runtime none alone",
		}
	}
	return []string{}, nil
}

// adversarialModels maps each runtime to its model, the last given winning; nil when none is given.
func adversarialModels(values []string) (map[string]string, error) {
	if values == nil {
		return nil, nil
	}
	models := map[string]string{}
	for _, v := range values {
		runtime, model, ok := strings.Cut(v, "=")
		var found string
		switch {
		case !ok:
			found = fmt.Sprintf(`%q has no "="`, v)
		case runtime == "":
			found = fmt.Sprintf("%q has an empty runtime", v)
		case model == "":
			found = fmt.Sprintf("%q has an empty model", v)
		}
		if found != "" {
			return nil, &project.Rejection{
				Code:  "invalid_value",
				Check: "--adversarial-model is <runtime>=<model>",
				Found: found,
				Fix:   "pass --adversarial-model claude=<model>",
			}
		}
		models[runtime] = model
	}
	return models, nil
}

// requestedExtensions resolves the given extensions; a given none requests an empty list.
func requestedExtensions(catalog project.Catalog, values []string) ([]string, error) {
	if values == nil {
		return nil, nil
	}
	if !slices.Contains(values, "none") {
		return catalog.Resolve(values)
	}
	if i := slices.IndexFunc(values, func(v string) bool { return v != "none" }); i >= 0 {
		return nil, &project.Rejection{
			Code:  "invalid_value",
			Check: "--ext names extensions or none",
			Found: "none cannot be combined with " + values[i],
			Fix:   "pass --ext none alone",
		}
	}
	return []string{}, nil
}

// selectExtensions returns the requested extensions, else the manifest's, else go for a go.mod (UC-2).
func selectExtensions(catalog project.Catalog, requested []string, existing *project.Manifest, root *os.Root,
) ([]project.CatalogExtension, error) {
	names := requested
	if names == nil && existing != nil {
		names = slices.Sorted(maps.Keys(existing.Extensions))
	}
	if names == nil {
		info, err := root.Lstat("go.mod")
		if _, ok := catalog["go"]; ok && err == nil && info.Mode().IsRegular() {
			names = []string{"go"}
		}
	}
	selected := make([]project.CatalogExtension, 0, len(names))
	for _, name := range names {
		e, ok := catalog[name]
		if !ok {
			return nil, fmt.Errorf("read extension %s: not in the catalog", name)
		}
		selected = append(selected, e)
	}
	return selected, nil
}

// selectRuntimes returns the requested runtimes, else the manifest's, else those found on PATH (UC-2).
func selectRuntimes(requested []string, existing *project.Manifest) []string {
	switch {
	case requested != nil:
		return requested
	case existing != nil:
		return slices.Sorted(maps.Keys(existing.Runtimes))
	}
	var found []string
	for _, runtime := range []string{"claude", "codex"} {
		if git.OnPath(runtime) {
			found = append(found, runtime)
		}
	}
	return found
}

// selectModels checks that every requested model names a selected runtime and fills in the manifest's.
func selectModels(requested map[string]string, runtimes []string, existing *project.Manifest,
) (map[string]string, error) {
	for _, runtime := range slices.Sorted(maps.Keys(requested)) {
		if !slices.Contains(runtimes, runtime) {
			return nil, &project.Rejection{
				Code:  "invalid_value",
				Check: "--adversarial-model names a selected runtime",
				Found: runtime + " is not selected",
				Fix:   "add --runtime " + runtime + " or drop the model",
			}
		}
	}
	models := map[string]string{}
	for _, runtime := range runtimes {
		if existing != nil && existing.Runtimes[runtime].AdversarialModel != "" {
			models[runtime] = existing.Runtimes[runtime].AdversarialModel
		}
		if model, ok := requested[runtime]; ok {
			models[runtime] = model
		}
	}
	return models, nil
}

// initialize plans against repo; with gitInit, repo is the directory git init turns into one (UC-4).
// A non-nil ask asks for the missing choices of a first init and confirms the plan.
func (f *initFlags) initialize(ctx context.Context, env Env, repo *git.Repo, gitInit bool, root *os.Root,
	framework string, catalog project.Catalog, ask *initPrompts,
) (output, error) {
	existing, err := project.ReadManifest(root, framework)
	if err != nil {
		return output{}, err
	}
	if ask != nil && existing == nil {
		if err := ask.choices(f, root, filepath.Base(repo.Dir), catalog); err != nil {
			return output{}, err
		}
	}
	runtimes := selectRuntimes(f.requested.Runtimes, existing)
	models, err := selectModels(f.requested.AdversarialModels, runtimes, existing)
	if err != nil {
		return output{}, err
	}
	var recorded project.Project
	if existing != nil {
		if err := existing.CheckChoices(f.requested); err != nil {
			return output{}, err
		}
		recorded = existing.Project
	}
	extensions, err := selectExtensions(catalog, f.requested.Extensions, existing, root)
	if err != nil {
		return output{}, err
	}
	p := project.Project{
		Name:       cmp.Or(f.requested.Name, recorded.Name, filepath.Base(repo.Dir)),
		MainBranch: cmp.Or(f.requested.MainBranch, recorded.MainBranch, defaultMainBranch),
		Remote:     cmp.Or(f.requested.Remote, recorded.Remote, defaultRemote),
	}
	store := &task.Store{Git: repo, Remote: p.Remote}
	r := initRun{
		root: repo.Dir, framework: framework, mainBranch: p.MainBranch, gitInit: gitInit, firstCommit: gitInit,
		runtimes: runtimes, models: models, extensions: extensions, warnings: jujutsuWarnings(root),
	}
	if !gitInit {
		if r.tip, err = existingRegistry(ctx, store); err != nil {
			return output{}, err
		}
		if r.firstCommit, r.switchOrphan, err = unbornMain(ctx, repo, p.MainBranch); err != nil {
			return output{}, err
		}
	}
	if r.plan, r.decisions, err = f.plan(env, root, existing, p, r); err != nil {
		return output{}, err
	}
	if ask != nil {
		ask.section()
	}
	if conflicts := r.plan.Conflicts(); len(conflicts) > 0 {
		return output{text: "Plan\n" + planLines(conflicts)}, fileConflict(conflicts)
	}
	switch {
	case f.dryRun:
		return r.dryRun(), nil
	case r.tip != "" && !slices.ContainsFunc(r.plan.Changes, writes):
		return r.noOp(), nil
	case ask != nil:
		return ask.proceed(ctx, r, root, store)
	case !f.yes:
		return output{}, &Error{
			Code: "interactive_required", Exit: exitUsage, Message: "init needs a confirmation",
			Fix: "pass --yes to accept the plan, or --dry-run to print it",
		}
	}
	return r.apply(ctx, root, store)
}

// plan wants the core, extension, and adapter files, then decision documents and the manifest.
func (f *initFlags) plan(env Env, root *os.Root, existing *project.Manifest, p project.Project, r initRun,
) (project.Plan, []project.Decision, error) {
	decisions, err := project.PlanDecisions(root, existing, r.extensions, env.Now().Format(time.DateOnly))
	if err != nil {
		return project.Plan{}, nil, err
	}
	core, err := project.CoreWants(env.Assets, project.Choices{
		Name: p.Name, Objective: f.objective, FrameworkVersion: r.framework,
		Extensions: r.extensions, Documents: decisions.Index,
	})
	if err != nil {
		return project.Plan{}, nil, err
	}
	extensions, entries, err := project.ExtensionWants(env.Assets, r.extensions, decisions.ByGroup)
	if err != nil {
		return project.Plan{}, nil, err
	}
	adapters, err := project.RuntimeWants(env.Assets, r.runtimes, r.models, r.framework, r.extensions)
	if err != nil {
		return project.Plan{}, nil, err
	}
	runtimes := map[string]project.Runtime{}
	for _, runtime := range r.runtimes {
		runtimes[runtime] = project.Runtime{AdversarialModel: r.models[runtime]}
	}
	plan, _, err := project.InitPlan(root, existing, slices.Concat(core, extensions, adapters), f.keep, f.overwrite,
		p, runtimes, entries, decisions, r.framework)
	return plan, decisions.Records, err
}

// unbornMain reports whether the main branch has no commit and HEAD points elsewhere (UC-4).
func unbornMain(ctx context.Context, repo *git.Repo, mainBranch string) (unborn, elsewhere bool, err error) {
	_, err = repo.Run(ctx, "rev-parse", "--verify", "--quiet", "refs/heads/"+mainBranch+"^{commit}")
	if err == nil {
		return false, false, nil
	}
	if gitErr, ok := errors.AsType[*git.Error](err); !ok || gitErr.ExitCode != 1 {
		return false, false, fmt.Errorf("check main branch: %w", err)
	}
	head, err := repo.Run(ctx, "symbolic-ref", "--quiet", "HEAD")
	if gitErr, ok := errors.AsType[*git.Error](err); ok && gitErr.ExitCode == 1 {
		head, err = "", nil
	}
	if err != nil {
		return false, false, fmt.Errorf("read HEAD: %w", err)
	}
	return true, head != "refs/heads/"+mainBranch, nil
}

func jujutsuWarnings(root *os.Root) []warning {
	if _, err := root.Lstat(".jj"); err != nil {
		return nil
	}
	return []warning{{
		Code: "jujutsu_detected", Message: "Jujutsu detected; v0.1 supports Git only. Never run jj commands on workflow-state.",
	}}
}

// existingRegistry returns the registry tip, or "" when the ref is absent (UC-3).
func existingRegistry(ctx context.Context, store *task.Store) (string, error) {
	_, tip, err := store.Read(ctx)
	rejection, ok := errors.AsType[*task.Rejection](err)
	switch {
	case ok && rejection.Code == "not_initialized":
		return "", nil
	case ok && rejection.Code == "registry_invalid":
		return "", &task.Rejection{
			Code:  "registry_exists_invalid",
			Check: "an existing workflow-state holds a valid registry",
			Found: rejection.Found,
			Fix:   "inspect with git show " + task.Ref + ":state.json; the CLI never overwrites a registry",
		}
	}
	return tip, err
}

func writes(c project.Change) bool {
	return c.Action == project.ActionCreate || c.Action == project.ActionUpdate || c.Action == project.ActionAppend ||
		c.Action == project.ActionRemove
}

// apply rolls the files back until the registry compare-and-swap succeeds (AD-6, F-09).
func (r initRun) apply(ctx context.Context, root *os.Root, store *task.Store) (output, error) {
	if r.gitInit {
		if err := r.initRepository(ctx, store); err != nil {
			return output{}, err
		}
	}
	journal, err := project.Apply(ctx, root, r.plan)
	if err != nil {
		return output{}, err
	}
	registry := registryState{Ref: task.Ref, Commit: r.tip, Action: "unchanged", Pushed: new(false)}
	var created task.Commit
	if r.tip == "" {
		created, err = store.Create(ctx)
		if err != nil {
			return output{}, errors.Join(err, journal.Rollback())
		}
		registry = registryState{Ref: task.Ref, Commit: created.SHA, Action: "create", Pushed: new(created.Pushed)}
	}
	data := r.data(registry)
	data.Next = []string{"git add " + strings.Join(r.generatedPaths(), " "), r.commitCommand()}
	if r.switchOrphan {
		data.Next = slices.Insert(data.Next, 0, "git switch --orphan "+r.mainBranch)
	}
	text := fmt.Sprintf("Initialized solomonar %s in %s\n  %s\n", r.framework, r.root, counts(r.plan.Changes)) +
		"  registry  " + task.Ref + " @ " + registry.Commit
	if r.tip != "" {
		text += " unchanged"
	}
	text += "\n"
	if created.Pushed {
		text += "  pushed    " + store.Remote + "/" + strings.TrimPrefix(task.Ref, "refs/heads/") + "\n"
	}
	commit := strings.Join(data.Next[len(data.Next)-2:], " && ")
	step := "  1. " + commit + "\n"
	if r.firstCommit {
		step = "  1. First commit required before task worktrees:\n"
		for _, line := range append(data.Next[:len(data.Next)-2:len(data.Next)-2], commit) {
			step += "       " + line + "\n"
		}
	}
	text += "\nNext:\n" + step + r.sessionStep()
	return output{data: data, text: text, warnings: slices.Concat(r.warnings, outputWarnings(created.Warnings))}, nil
}

// generatedPaths lists what the first commit adds, in the UC-2 order.
func (r initRun) generatedPaths() []string {
	paths := []string{"AGENTS.md"}
	if slices.ContainsFunc(r.plan.Changes, func(c project.Change) bool {
		return c.Path == "CLAUDE.md" && c.Action == project.ActionCreate
	}) {
		paths = append(paths, "CLAUDE.md")
	}
	if slices.Contains(r.runtimes, "claude") {
		paths = append(paths, ".claude")
	}
	if slices.Contains(r.runtimes, "codex") {
		paths = append(paths, ".agents", ".codex")
	}
	return append(paths, ".solomonar", "docs")
}

// sessionStep is the Next: step starting the top-level session in the selected runtimes (UC-1, UC-2).
func (r initRun) sessionStep() string {
	const explore = "Explore the next useful flow.\n"
	claude, codex := slices.Contains(r.runtimes, "claude"), slices.Contains(r.runtimes, "codex")
	switch {
	case claude && codex:
		return "  2. Start a top-level session in " + r.root + " with your selected outcome:\n" +
			"       Claude Code: /solomonar " + explore +
			"       Codex:       $solomonar " + explore +
			"       Portable:    Use the solomonar skill to explore the next useful flow.\n"
	case claude:
		return "  2. Start a top-level Claude Code session in " + r.root + ":\n       /solomonar " + explore
	case codex:
		return "  2. Start a top-level Codex session in " + r.root + ":\n       $solomonar " + explore
	}
	return "  2. Start a top-level session in " + r.root + ":\n" +
		"       Use the solomonar skill to explore the next useful flow.\n"
}

// initRepository runs git init in the planned root and points the store at the new repository (UC-4).
func (r initRun) initRepository(ctx context.Context, store *task.Store) error {
	if _, err := store.Git.Run(ctx, "init", "-b", r.mainBranch, r.root); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	repo, err := git.Discover(ctx, r.root, store.Git.Trace)
	if err != nil {
		return err
	}
	store.Git = repo
	return nil
}

func (r initRun) noOp() output {
	data := r.data(registryState{Ref: task.Ref, Commit: r.tip, Action: "unchanged", Pushed: new(false)})
	data.Next = []string{}
	text := fmt.Sprintf("Already initialized with solomonar %s; nothing to do.\n", r.framework) +
		fmt.Sprintf("  %s · registry %s unchanged\n", counts(r.plan.Changes), task.Ref) +
		"  Change choices with `solomonar runtime|extension add|remove`; move versions with `solomonar upgrade`.\n"
	return output{data: data, text: text, warnings: r.warnings}
}

func (r initRun) dryRun() output {
	action := "unchanged"
	if r.tip == "" {
		action = "create"
	}
	data := r.data(registryState{Ref: task.Ref, Action: action})
	data.DryRun = true
	text := r.planText() +
		fmt.Sprintf("Would initialize solomonar %s in %s\n  %s\n", r.framework, r.root, counts(r.plan.Changes))
	return output{data: data, text: text, warnings: r.warnings}
}

// planText lists the git init, the file changes, and the registry change.
func (r initRun) planText() string {
	registry := "unchanged " + task.Ref
	if r.tip == "" {
		registry = "create " + task.Ref + " with an empty state.json"
	}
	text := "Plan\n"
	if r.gitInit {
		text += fmt.Sprintf("  %-9s  %s\n", "git", "init -b "+r.mainBranch)
	}
	return text + planLines(r.plan.Changes) + fmt.Sprintf("  %-9s  %s\n", "registry", registry)
}

func (r initRun) data(registry registryState) initData {
	files := make([]initFile, 0, len(r.plan.Changes))
	for _, c := range r.plan.Changes {
		files = append(files, initFile{Path: c.Path, Action: string(c.Action), Owner: c.Owner, Reason: c.Reason})
	}
	extensions := map[string][]string{}
	for _, e := range r.extensions {
		extensions[e.Group] = append(extensions[e.Group], e.Name)
	}
	return initData{
		Project: r.root, FrameworkVersion: r.framework, Files: files, Registry: registry, Decisions: r.decisions,
		Choices: initChoices{Runtimes: r.runtimes, Extensions: extensions, AdversarialModel: r.models},
		GitInit: r.gitInit, FirstCommit: r.firstCommit,
	}
}

func (r initRun) commitCommand() string {
	return fmt.Sprintf("git commit -m \"chore: initialize solomonar %s\"", r.framework)
}

// planLines prints one change per line, aligning the reasons after the widest path that has one.
func planLines(changes []project.Change) string {
	width := 0
	for _, c := range changes {
		if c.Reason != "" {
			width = max(width, len(c.Path))
		}
	}
	var b strings.Builder
	for _, c := range changes {
		if c.Reason == "" {
			fmt.Fprintf(&b, "  %-9s  %s\n", c.Action, c.Path)
		} else {
			fmt.Fprintf(&b, "  %-9s  %-*s  %s\n", c.Action, width, c.Path, c.Reason)
		}
	}
	return b.String()
}

// counts reads "<n> file(s) <action>, <n> <action>, …" over the actions that occur.
func counts(changes []project.Change) string {
	words := []struct {
		actions []project.Action
		word    string
	}{
		{[]project.Action{project.ActionCreate}, "created"},
		{[]project.Action{project.ActionUpdate, project.ActionAppend}, "updated"},
		{[]project.Action{project.ActionKeep}, "kept"},
		{[]project.Action{project.ActionUnchanged}, "unchanged"},
		{[]project.Action{project.ActionSkip}, "skipped"},
	}
	var parts []string
	for _, w := range words {
		n := 0
		for _, c := range changes {
			if slices.Contains(w.actions, c.Action) {
				n++
			}
		}
		switch {
		case n == 0:
		case parts == nil:
			parts = append(parts, fmt.Sprintf("%d %s %s", n, plural(n, "file", "files"), w.word))
		default:
			parts = append(parts, fmt.Sprintf("%d %s", n, w.word))
		}
	}
	return strings.Join(parts, ", ")
}

func fileConflict(conflicts []project.Change) *Error {
	paths := make([]string, 0, len(conflicts))
	for _, c := range conflicts {
		paths = append(paths, c.Path)
	}
	n := fmt.Sprintf("%d %s", len(conflicts), plural(len(conflicts), "conflict", "conflicts"))
	return &Error{
		Code: "file_conflict", Exit: 4, Message: "init found " + n,
		Check: "no planned file conflicts with the project's files",
		Found: n + ": " + strings.Join(paths, ", "),
		Fix:   "rerun with --keep <path> to keep the project's file, or --overwrite <path> to replace it",
	}
}
