package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

// initPrompts asks for the init choices and the confirmation in a terminal session (I §3.7, UC-1).
type initPrompts struct {
	*prompter
	asked bool
}

func runtimeItems() []choice {
	return []choice{
		{name: "claude", description: "Claude Code: CLAUDE.md import, .claude/skills, .claude/agents, state guard hook"},
		{name: "codex", description: "Codex: AGENTS.md (native), .agents/skills, .codex/agents, state guard hook"},
	}
}

// section separates a group of questions from the previous one with a blank line (UC-1).
func (a *initPrompts) section() {
	if a.asked {
		fmt.Fprintln(a.out)
	}
	a.asked = true
}

// choices asks for every choice whose flag was not given and records the answers as those flags.
func (a *initPrompts) choices(f *initFlags, root *os.Root, defaultName string, catalog project.Catalog) error {
	var err error
	if f.requested.Name == "" || f.objective == "" {
		a.section()
	}
	if f.requested.Name == "" {
		if f.requested.Name, err = a.text("Project name", defaultName); err != nil {
			return err
		}
	}
	if f.objective == "" {
		if f.objective, err = a.text("Current objective, one line (optional)", ""); err != nil {
			return err
		}
	}
	if f.requested.Runtimes == nil {
		a.section()
		fmt.Fprintln(a.out, "Runtimes that will run this project's sessions:")
		if f.requested.Runtimes, err = a.choose(runtimeItems(), selectRuntimes(nil, nil)); err != nil {
			return err
		}
	}
	if err := a.models(f); err != nil {
		return err
	}
	if len(catalog) == 0 || f.requested.Extensions != nil {
		return nil
	}
	f.requested.Extensions, err = a.extensions(catalog, root)
	return err
}

func (a *initPrompts) models(f *initFlags) error {
	titles := map[string]string{"claude": "Claude Code", "codex": "Codex"}
	first := true
	for _, runtime := range f.requested.Runtimes {
		if _, ok := f.requested.AdversarialModels[runtime]; ok {
			continue
		}
		if first {
			a.section()
			first = false
		}
		model, err := a.text("Model for the adversarial reviewer in "+titles[runtime]+
			", different from your main session model\n(blank = runtime default)", "")
		if err != nil {
			return err
		}
		if model != "" {
			if f.requested.AdversarialModels == nil {
				f.requested.AdversarialModels = map[string]string{}
			}
			f.requested.AdversarialModels[runtime] = model
		}
	}
	return nil
}

// extensions asks group by group until the selection resolves, starting from the default selection.
func (a *initPrompts) extensions(catalog project.Catalog, root *os.Root) ([]string, error) {
	defaults, err := selectExtensions(catalog, nil, nil, root)
	if err != nil {
		return nil, err
	}
	var selected []string
	for _, e := range defaults {
		selected = append(selected, e.Name)
	}
	a.section()
	fmt.Fprint(a.out, "Extensions, by decision type. A choice in a stack group is recorded as your approved decision in\n"+
		"docs/decisions.md. Enter with nothing selected skips a group; add more later with\n"+
		"`solomonar extension add`.\n")
	for {
		if selected, err = a.groups(catalog, selected); err != nil {
			return nil, err
		}
		resolved, err := catalog.Resolve(selected)
		if rejection, ok := errors.AsType[*project.Rejection](err); ok {
			fmt.Fprintf(a.out, "  %s; %s\n", rejection.Found, rejection.Fix)
			continue
		}
		if err != nil {
			return nil, err
		}
		return append([]string{}, resolved...), nil
	}
}

func (a *initPrompts) groups(catalog project.Catalog, selected []string) ([]string, error) {
	var chosen []string
	for _, group := range project.Groups() {
		var items []choice
		for _, e := range catalog {
			if e.Group == group {
				items = append(items, extensionItem(e))
			}
		}
		if len(items) == 0 {
			continue
		}
		slices.SortFunc(items, func(x, y choice) int { return strings.Compare(x.name, y.name) })
		a.section()
		fmt.Fprintf(a.out, "%s%s:\n", strings.ToUpper(group[:1]), group[1:])
		names, err := a.choose(items, selected)
		if err != nil {
			return nil, err
		}
		chosen = append(chosen, names...)
	}
	return chosen, nil
}

func extensionItem(e project.CatalogExtension) choice {
	description := e.Title
	if len(e.Requires) > 0 {
		description += " (requires " + strings.Join(e.Requires, ", ") + ")"
	}
	return choice{name: e.Name, description: description}
}

// proceed shows the plan and applies it once the user confirms.
func (a *initPrompts) proceed(ctx context.Context, r initRun, root *os.Root, store *task.Store) (output, error) {
	fmt.Fprint(a.out, r.planText())
	yes, err := a.confirm("Proceed?", true)
	if err != nil {
		return output{}, err
	}
	fmt.Fprintln(a.out)
	if !yes {
		return output{text: "Cancelled; nothing was written.\n", warnings: r.warnings}, nil
	}
	return r.apply(ctx, root, store)
}
