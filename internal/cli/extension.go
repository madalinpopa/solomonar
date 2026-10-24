package cli

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/madalinpopa/solomonar/internal/project"
)

type extensionListFlags struct {
	group, kind string
	installed   bool
}

type extensionListData struct {
	Extensions []extensionRow `json:"extensions"`
}

type extensionRow struct {
	Name        string   `json:"name"`
	Group       string   `json:"group"`
	Kind        string   `json:"kind"`
	Source      string   `json:"source"`
	Status      string   `json:"status"`
	Reason      string   `json:"reason,omitempty"`
	Decision    string   `json:"decision,omitempty"`
	Suggested   string   `json:"suggested,omitempty"`
	Title       string   `json:"title"`
	AppliesWhen string   `json:"applies_when"`
	Paths       []string `json:"paths"`
	Requires    []string `json:"requires"`
	Conflicts   []string `json:"conflicts"`
	Invocation  string   `json:"invocation"`
	Path        string   `json:"path,omitempty"`
}

func extensionListCommand() command {
	return command{
		name:    "extension list",
		summary: "List catalog and project extensions",
		usage:   "solomonar extension list [--group <group>] [--kind <skill|agent>] [--installed] [--json]",
		description: "List the catalog extensions and the project extensions in docs/solomonar/extensions/, with\n" +
			"their status, decision, and detection reason; 'ext' is an alias for 'extension'. Read-only.",
		bind: func(fs *flag.FlagSet) runFunc {
			var f extensionListFlags
			fs.StringVar(&f.group, "group", "", "")
			fs.StringVar(&f.kind, "kind", "", "")
			fs.BoolVar(&f.installed, "installed", false, "")
			return f.run
		},
	}
}

func (f *extensionListFlags) run(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	if len(args) > 0 {
		return output{}, unexpectedArgument(args[0])
	}
	if err := f.validate(); err != nil {
		return output{}, err
	}
	repo, err := openRepo(ctx, env, flags, flags.dir, "run inside a Git repository or pass -C <path>")
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
	manifest, err := project.ReadManifest(root, framework)
	if err != nil {
		return output{}, err
	}
	catalog, err := project.LoadCatalog(env.Assets)
	if err != nil {
		return output{}, err
	}
	store, err := project.LoadStore(root.FS(), env.Assets, catalog, manifest)
	if err != nil {
		return output{}, err
	}
	files, err := repo.Files(ctx)
	if err != nil {
		return output{}, err
	}
	rows := slices.DeleteFunc(extensionRows(catalog, store, manifest, files), func(r extensionRow) bool {
		return f.group != "" && r.Group != f.group || f.kind != "" && r.Kind != f.kind ||
			f.installed && r.Status != "installed"
	})
	return output{data: extensionListData{Extensions: rows}, text: extensionTable(rows)}, nil
}

func (f *extensionListFlags) validate() error {
	if f.group != "" && !slices.Contains(project.Groups(), f.group) {
		return &project.Rejection{
			Code:  "unknown_group",
			Check: "--group names a group",
			Found: fmt.Sprintf("%q is not a group; known: %s", f.group, strings.Join(project.Groups(), ", ")),
			Fix:   "pass --group with a known group",
		}
	}
	if f.kind != "" && f.kind != "skill" && f.kind != "agent" {
		return &project.Rejection{
			Code:  "invalid_value",
			Check: "--kind is skill or agent",
			Found: fmt.Sprintf("%q is not skill or agent", f.kind),
			Fix:   "pass --kind skill or --kind agent",
		}
	}
	return nil
}

// extensionRows merges the catalog, the store, and the manifest, by group, name, then source (UC-6).
func extensionRows(catalog project.Catalog, store map[string]project.ProjectExtension, manifest *project.Manifest,
	files []string,
) []extensionRow {
	var installed map[string]project.Extension
	if manifest != nil {
		installed = manifest.Extensions
	}
	valid := project.Catalog{}
	for name, e := range store {
		if e.Err == nil {
			valid[name] = e.CatalogExtension
		}
	}
	suggested := catalog.Suggested(files)
	maps.Copy(suggested, valid.Suggested(files))
	var rows []extensionRow
	for name, e := range catalog {
		row := newExtensionRow(e, "catalog")
		if entry, ok := installed[name]; ok && entry.Source == "catalog" {
			row.Status, row.Decision, row.Path = "installed", entry.Decision, project.StoreDir+"/"+name
		} else {
			row.Suggested = suggested[name]
		}
		rows = append(rows, row)
	}
	for name, e := range store {
		row := newExtensionRow(e.CatalogExtension, "project")
		row.Path = e.Dir
		entry, ok := installed[name]
		switch {
		case e.Err != nil:
			row.Status, row.Reason = "invalid", e.Err.Error()
		case ok:
			row.Status, row.Decision = "installed", entry.Decision
		default:
			row.Suggested = suggested[name]
		}
		rows = append(rows, row)
	}
	groupIndex := func(group string) int {
		if i := slices.Index(project.Groups(), group); i >= 0 {
			return i
		}
		return len(project.Groups())
	}
	slices.SortFunc(rows, func(a, b extensionRow) int {
		return cmp.Or(cmp.Compare(groupIndex(a.Group), groupIndex(b.Group)), cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Source, b.Source))
	})
	return rows
}

func newExtensionRow(e project.CatalogExtension, source string) extensionRow {
	return extensionRow{
		Name: e.Name, Group: e.Group, Kind: e.Kind, Source: source, Status: "available",
		Title: e.Title, AppliesWhen: e.AppliesWhen, Paths: e.Paths, Requires: e.Requires, Conflicts: e.Conflicts,
		Invocation: e.Invocation,
	}
}

func extensionTable(rows []extensionRow) string {
	var b strings.Builder
	table := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "GROUP\tNAME\tKIND\tSOURCE\tSTATUS\tDECISION\tSUGGESTED\tAPPLIES WHEN")
	for _, r := range rows {
		status := r.Status
		if r.Reason != "" {
			status += ": " + r.Reason
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", cmp.Or(r.Group, "-"), r.Name, cmp.Or(r.Kind, "-"),
			r.Source, status, cmp.Or(r.Decision, "-"), cmp.Or(r.Suggested, "-"), cmp.Or(r.AppliesWhen, "-"))
	}
	_ = table.Flush()
	return b.String()
}

// extensionFlags are the flags of extension add and remove (UC-6).
type extensionFlags struct {
	decisionFlags
	keep, overwrite []string
	yes, dryRun     bool
}

// decisionFlags are the --decision and --rationale of extension add and remove (UC-6).
type decisionFlags struct {
	decision, rationale string
}

type extensionData struct {
	Project   string             `json:"project"`
	Added     []string           `json:"added,omitzero"`
	Removed   []string           `json:"removed,omitzero"`
	Decisions []project.Decision `json:"decisions"`
	Files     []initFile         `json:"files"`
	Next      []string           `json:"next,omitzero"`
	DryRun    bool               `json:"dry_run,omitzero"`
}

// extensionRun is an init run over the manifest's choices with the named extensions added or removed.
type extensionRun struct {
	initRun
	existing *project.Manifest
	verb     string
	named    []project.CatalogExtension
}

func extensionAddCommand() command {
	return command{
		name:    "extension add",
		summary: "Install catalog or project extensions",
		usage: "solomonar extension add <name>... [--decision <ID>] [--rationale <text>] [--keep <path>]...\n" +
			"                               [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]",
		description: "Install catalog extensions into docs/solomonar/extensions/, or activate the project extensions\n" +
			"there, with their runtime stubs, AGENTS.md line, and manifest entry. A changed stack group records\n" +
			"a decision: the approved --decision entry, or a new P/D-n entry with --rationale. --keep or\n" +
			"--overwrite resolves a conflicting file; --dry-run prints the plan and writes nothing.",
		bind: func(fs *flag.FlagSet) runFunc {
			var f extensionFlags
			f.bind(fs)
			return f.runner("add")
		},
	}
}

func extensionRemoveCommand() command {
	return command{
		name:    "extension remove",
		summary: "Remove catalog or deactivate project extensions",
		usage: "solomonar extension remove <name>... [--decision <ID>] [--rationale <text>] [--keep <path>]...\n" +
			"                                  [--overwrite <path>]... [-y|--yes] [--dry-run] [--json]",
		description: "Delete installed catalog extensions from docs/solomonar/extensions/, or deactivate project\n" +
			"extensions there, removing their runtime stubs, AGENTS.md line, and manifest entry. An emptied\n" +
			"stack group records the selection none. --keep untracks an edited file, --overwrite deletes it;\n" +
			"--dry-run prints the plan and writes nothing.",
		bind: func(fs *flag.FlagSet) runFunc {
			var f extensionFlags
			f.bind(fs)
			return f.runner("remove")
		},
	}
}

func (f *extensionFlags) bind(fs *flag.FlagSet) {
	fs.StringVar(&f.decision, "decision", "", "")
	fs.StringVar(&f.rationale, "rationale", "", "")
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

func (f *extensionFlags) runner(verb string) runFunc {
	command := "extension " + verb
	return func(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
		if len(args) == 0 {
			return output{}, invalidUsage(errors.New("extension name is required"))
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
		a, err := f.plan(env, root, repo.Dir, verb, args)
		if err != nil {
			return output{}, err
		}
		if conflicts := a.plan.Conflicts(); len(conflicts) > 0 {
			return output{text: "Plan\n" + planLines(a.plan.Changes)},
				fileConflict(command, "--overwrite <path> to replace it", conflicts)
		}
		switch {
		case f.dryRun:
			return a.dryRun(), nil
		case !slices.ContainsFunc(a.plan.Changes, writes):
			return a.noOp(), nil
		}
		return confirmed(env, flags, command, f.yes, a.changedPlan("Plan\n"), func() (output, error) {
			return a.apply(ctx, root)
		})
	}
}

// plan checks the project and the names in the UC-6 order, then plans the resulting installed set.
func (f *extensionFlags) plan(env Env, root *os.Root, dir, verb string, args []string) (extensionRun, error) {
	framework, err := frameworkVersion(env.Assets)
	if err != nil {
		return extensionRun{}, err
	}
	existing, err := initializedManifest(root, framework, dir)
	if err != nil {
		return extensionRun{}, err
	}
	if err := currentFramework(existing, framework); err != nil {
		return extensionRun{}, err
	}
	catalog, err := project.LoadCatalog(env.Assets)
	if err != nil {
		return extensionRun{}, err
	}
	store, err := project.LoadStore(root.FS(), env.Assets, catalog, existing)
	if err != nil {
		return extensionRun{}, err
	}
	var named, extensions []project.CatalogExtension
	if verb == "remove" {
		named, extensions, err = f.removal(env.Assets, catalog, store, existing, args)
	} else {
		named, extensions, err = f.addition(catalog, store, existing, args)
	}
	if err != nil {
		return extensionRun{}, err
	}
	if err := f.approved(root); err != nil {
		return extensionRun{}, err
	}
	runtimes := selectRuntimes(nil, existing)
	models, err := selectModels(nil, runtimes, existing)
	if err != nil {
		return extensionRun{}, err
	}
	a := extensionRun{
		root: dir, framework: framework, runtimes: runtimes, models: models, extensions: extensions,
		decider: f.decider("extension " + verb), existing: existing, verb: verb, named: named,
	}
	if verb == "remove" {
		a.removed = named
	}
	a.plan, a.decisions, err = a.planFiles(env, root, existing, existing.Project, "",
		project.Resolutions{Keep: f.keep, Overwrite: f.overwrite})
	return a, err
}

// addition resolves the added names and checks the installed set they result in (UC-6).
func (f *extensionFlags) addition(catalog project.Catalog, store map[string]project.ProjectExtension,
	existing *project.Manifest, args []string,
) (added, extensions []project.CatalogExtension, err error) {
	if added, err = resolveNames(catalog, store, existing, args); err != nil {
		return nil, nil, err
	}
	if err := f.applies(added); err != nil {
		return nil, nil, err
	}
	if extensions, err = installedExtensions(catalog, store, existing, added); err != nil {
		return nil, nil, err
	}
	return added, extensions, checkSelection(extensions, added)
}

// removal resolves the installed names and checks the installed set left without them (UC-6).
func (f *extensionFlags) removal(assets fs.FS, catalog project.Catalog, store map[string]project.ProjectExtension,
	existing *project.Manifest, args []string,
) (removed, extensions []project.CatalogExtension, err error) {
	left := *existing
	left.Extensions = maps.Clone(existing.Extensions)
	for _, name := range slices.Compact(slices.Sorted(slices.Values(args))) {
		entry, installed := existing.Extensions[name]
		switch {
		case project.CoreItem(assets, name) != "":
			return nil, nil, &project.Rejection{
				Code:  "core_item",
				Check: "extension remove names an extension",
				Found: name + " is part of the core",
				Fix:   "the core cannot be removed (4.1)",
			}
		case !installed:
			return nil, nil, &project.Rejection{
				Code:  "unknown_extension",
				Check: "extension remove names an installed extension",
				Found: fmt.Sprintf("%q is not installed; installed: %s", name,
					cmp.Or(strings.Join(slices.Sorted(maps.Keys(existing.Extensions)), ", "), "none")),
				Fix: "pass a name from solomonar extension list --installed",
			}
		}
		removed = append(removed, project.CatalogExtension{
			Name: name, Kind: entry.Kind, Group: entry.Group, Source: entry.Source,
		})
		delete(left.Extensions, name)
	}
	if err := f.applies(removed); err != nil {
		return nil, nil, err
	}
	if extensions, err = installedExtensions(catalog, store, &left, nil); err != nil {
		return nil, nil, err
	}
	if other, missing := byName(extensions).Unmet(names(extensions)); other != "" {
		return nil, nil, &project.Rejection{
			Code:  "extension_requires",
			Check: "every required extension is installed",
			Found: other + " requires " + strings.Join(missing, ", "),
			Fix:   "remove both: solomonar extension remove " + strings.Join(append(names(removed), other), " "),
		}
	}
	return removed, extensions, nil
}

// currentFramework rejects a project whose framework is older than this CLI's (I section 3.6).
func currentFramework(existing *project.Manifest, framework string) error {
	older, err := project.NewerFramework(framework, existing.FrameworkVersion)
	if err != nil || !older {
		return err
	}
	return &project.Rejection{
		Code:  "framework_outdated",
		Check: "the project's framework matches this CLI",
		Found: "manifest framework " + existing.FrameworkVersion + ", this CLI ships " + framework,
		Fix:   "run solomonar upgrade",
	}
}

// resolveNames finds each sorted unique name in the catalog, then in the store (UC-6).
func resolveNames(catalog project.Catalog, store map[string]project.ProjectExtension, existing *project.Manifest,
	names []string,
) ([]project.CatalogExtension, error) {
	var resolved []project.CatalogExtension
	for _, name := range slices.Compact(slices.Sorted(slices.Values(names))) {
		e, inCatalog := catalog[name]
		p, inStore := store[name]
		switch {
		case inCatalog && inStore:
			return nil, &project.Rejection{
				Code:  "name_collision",
				Check: "the name is not both a catalog and a project extension",
				Found: name + " is in the catalog and in " + project.StoreDir + "/" + name,
				Fix:   "rename or remove " + project.StoreDir + "/" + name + "/",
			}
		case inCatalog:
			resolved = append(resolved, e)
		case inStore && p.Err != nil:
			return nil, invalidExtension(name, p.Err)
		case inStore:
			resolved = append(resolved, p.CatalogExtension)
		default:
			return nil, unknownExtension(catalog, store, existing, name)
		}
	}
	return resolved, nil
}

func unknownExtension(catalog project.Catalog, store map[string]project.ProjectExtension, existing *project.Manifest,
	name string,
) *project.Rejection {
	var available []string
	for n := range catalog {
		if _, installed := existing.Extensions[n]; !installed {
			available = append(available, n)
		}
	}
	for n, p := range store {
		if _, installed := existing.Extensions[n]; !installed && p.Err == nil {
			available = append(available, n)
		}
	}
	slices.Sort(available)
	return &project.Rejection{
		Code:  "unknown_extension",
		Check: "extension add names a catalog or project extension",
		Found: fmt.Sprintf("%q is not in the catalog or %s/; known: %s", name, project.StoreDir,
			cmp.Or(strings.Join(available, ", "), "none")),
		Fix: "pass a name from solomonar extension list",
	}
}

// applies rejects --decision and --rationale when no named extension is in a stack group.
func (f *decisionFlags) applies(named []project.CatalogExtension) error {
	if f.decision == "" && f.rationale == "" || slices.ContainsFunc(named, func(e project.CatalogExtension) bool {
		return e.Group != "workflow" && e.Group != "review"
	}) {
		return nil
	}
	return invalidUsage(errors.New("--decision and --rationale apply to stack groups only"))
}

// approved checks that --decision names an approved entry in docs/decisions.md (I section 4.3).
func (f *decisionFlags) approved(root *os.Root) error {
	if f.decision == "" {
		return nil
	}
	if !project.ValidDecisionID(f.decision) {
		return &project.Rejection{
			Code:  "invalid_value",
			Check: "--decision is P/D-<n> or <task-ID>/D-<n>",
			Found: fmt.Sprintf("%q is not P/D-<n> or <task-ID>/D-<n>", f.decision),
			Fix:   "pass the ID of an approved entry in docs/decisions.md",
		}
	}
	status, found, err := project.DecisionStatus(root, f.decision)
	switch {
	case err != nil:
		return err
	case !found:
		return &project.Rejection{
			Code:  "decision_not_found",
			Check: "--decision must name an approved entry in docs/decisions.md",
			Found: f.decision + " is not in docs/decisions.md",
			Fix:   "record the decision first (rules section 3), then rerun",
		}
	case status != "approved":
		return &project.Rejection{
			Code:  "decision_not_approved",
			Check: "--decision must name an approved entry in docs/decisions.md",
			Found: f.decision + " has status " + cmp.Or(status, "none"),
			Fix:   "wait for the user's approval (rules section 3), then rerun",
		}
	}
	return nil
}

func (f *decisionFlags) decider(command string) project.Decider {
	return project.Decider{Command: command, Rationale: f.rationale, ID: f.decision}
}

// checkSelection runs the I section 4.3 selection checks over the resulting installed set.
func checkSelection(extensions, added []project.CatalogExtension) error {
	known := byName(extensions)
	selected := slices.Sorted(maps.Keys(known))
	if name, missing := known.Unmet(selected); name != "" {
		return &project.Rejection{
			Code:  "extension_requires",
			Check: "every required extension is installed",
			Found: name + " requires " + strings.Join(missing, ", "),
			Fix:   "add them in the same call: solomonar extension add " + strings.Join(slices.Concat(names(added), missing), " "),
		}
	}
	if name, other := known.Conflict(selected); name != "" {
		return &project.Rejection{
			Code:  "extension_conflict",
			Check: "no installed extensions conflict",
			Found: name + " conflicts with " + other,
			Fix:   "remove " + other + " first, or drop " + name,
		}
	}
	return nil
}

func byName(extensions []project.CatalogExtension) project.Catalog {
	catalog := project.Catalog{}
	for _, e := range extensions {
		catalog[e.Name] = e
	}
	return catalog
}

func names(extensions []project.CatalogExtension) []string {
	out := make([]string, 0, len(extensions))
	for _, e := range extensions {
		out = append(out, e.Name)
	}
	return out
}

func (a extensionRun) apply(ctx context.Context, root *os.Root) (output, error) {
	if _, err := project.Apply(ctx, root, a.plan); err != nil {
		return output{}, err
	}
	data := a.data()
	data.Next = []string{"git add " + strings.Join(a.generatedPaths(), " ") +
		fmt.Sprintf(" && git commit -m \"chore: %s solomonar extension %s\"", a.verb, strings.Join(names(a.named), " "))}
	past := "Added"
	var warnings []warning
	if a.verb == "remove" {
		past = "Removed"
		for _, e := range a.named {
			warnings = append(warnings, warning{
				Code: "rules_removed", Message: "code governed by " + e.Name + " loses its rules",
			})
		}
	}
	text := fmt.Sprintf("%s %s in %s\n  %s\n\nNext:\n  1. %s\n", past, a.subject(), a.root, counts(a.plan.Changes),
		data.Next[0])
	return output{data: data, text: text, warnings: warnings}, nil
}

// subject names one extension with its group, kind, source, and decision, or several by name.
func (a extensionRun) subject() string {
	if len(a.named) > 1 {
		return "extensions " + strings.Join(names(a.named), ", ")
	}
	e := a.named[0]
	source := "catalog"
	if e.Source == "path" {
		source = "project"
	}
	details := []string{e.Group + " " + e.Kind, source}
	decision := a.existing.Extensions[e.Name].Decision
	for _, d := range a.decisions {
		if d.Group == e.Group {
			decision = d.ID
		}
	}
	if decision != "" {
		details = append(details, "under "+decision)
	}
	return "extension " + e.Name + " (" + strings.Join(details, ", ") + ")"
}

func (a extensionRun) noOp() output {
	data := a.data()
	data.Next = []string{}
	text := fmt.Sprintf("Already installed: %s; nothing to do.\n  %s\n", strings.Join(names(a.named), ", "),
		counts(a.plan.Changes))
	return output{data: data, text: text}
}

func (a extensionRun) dryRun() output {
	data := a.data()
	data.DryRun = true
	text := "Plan\n" + planLines(a.plan.Changes) + fmt.Sprintf("Would %s extension %s in %s\n  %s\n",
		a.verb, strings.Join(names(a.named), ", "), a.root, counts(a.plan.Changes))
	return output{data: data, text: text}
}

func (a extensionRun) data() extensionData {
	data := extensionData{Project: a.root, Decisions: a.decisions, Files: fileEntries(a.plan.Changes)}
	if a.verb == "remove" {
		data.Removed = names(a.named)
	} else {
		data.Added = names(a.named)
	}
	return data
}
