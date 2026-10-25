package cli

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

// planLine is one line of the grouped plan; path may summarize a collapsed set of files.
type planLine struct{ action, path, note string }

// groupedPlan is the interactive plan: changes grouped by owner, uniform file sets collapsed (UC-1).
func (r initRun) groupedPlan() string {
	byGroup := map[string][]project.Change{}
	for _, c := range r.plan.Changes {
		byGroup[planGroup(c.Owner)] = append(byGroup[planGroup(c.Owner)], c)
	}
	registry := planLine{action: "unchanged", path: task.Ref}
	if r.tip == "" {
		registry = planLine{action: "create", path: task.Ref + " with an empty registry"}
	}
	var b strings.Builder
	if r.gitInit {
		writeGroup(&b, "git", []planLine{{action: "init", path: "-b " + r.mainBranch}})
	}
	writeGroup(&b, "framework "+r.framework, collapse(byGroup["framework"], frameworkSet, frameworkLabel))
	writeGroup(&b, "extensions", collapse(byGroup["extensions"], extensionSet, extensionLabel))
	writeGroup(&b, "decisions (approved, selected by you)", r.decisionLines(byGroup["decision"]))
	for _, runtime := range []string{"claude", "codex", "copilot"} {
		writeGroup(&b, "runtime "+runtime, collapse(byGroup["runtime "+runtime], stubSet, r.stubLabel))
	}
	writeGroup(&b, "registry", []planLine{registry})
	writeGroup(&b, "manifest", collapse(byGroup["manifest"], noSet, nil))
	return b.String()
}

const cancelled = "Cancelled; nothing was written.\n"

// changedPlan is the UC-1 grouped plan without unchanged lines and skips that give no reason.
func (r initRun) changedPlan(header string) string {
	byGroup := map[string][]project.Change{}
	for _, c := range r.plan.Changes {
		if c.Action == project.ActionUnchanged || c.Action == project.ActionSkip && c.Reason == "" {
			continue
		}
		byGroup[planGroup(c.Owner)] = append(byGroup[planGroup(c.Owner)], c)
	}
	var b strings.Builder
	b.WriteString(header)
	writeGroup(&b, "framework "+r.framework, collapse(byGroup["framework"], frameworkSet, frameworkLabel))
	writeGroup(&b, "extensions", collapse(byGroup["extensions"], extensionSet, extensionLabel))
	writeGroup(&b, "decisions (approved, selected by you)", r.decisionLines(byGroup["decision"]))
	for _, runtime := range []string{"claude", "codex", "copilot"} {
		writeGroup(&b, "runtime "+runtime, collapse(byGroup["runtime "+runtime], stubSet, r.stubLabel))
	}
	writeGroup(&b, "manifest", collapse(byGroup["manifest"], noSet, nil))
	return b.String()
}

// confirmPlan shows plan on stderr, since a prompt follows (AD-2), and asks to proceed.
func confirmPlan(env Env, plan string) (bool, error) {
	ask := newPrompter(env)
	ask.out = env.Stderr
	fmt.Fprint(ask.out, plan)
	yes, err := ask.confirm("Proceed?", true)
	if err != nil {
		return false, err
	}
	fmt.Fprintln(ask.out)
	return yes, nil
}

func planGroup(owner string) string {
	class, name, _ := strings.Cut(owner, ":")
	switch class {
	case "core", "core+notes", "seed":
		return "framework"
	case "ext", "project-ext":
		return "extensions"
	case "runtime":
		return "runtime " + name
	}
	return class
}

// writeGroup prints a non-empty group, aligning its notes after the widest path that has one.
func writeGroup(b *strings.Builder, header string, lines []planLine) {
	if len(lines) == 0 {
		return
	}
	width := 0
	for _, l := range lines {
		if l.note != "" {
			width = max(width, len(l.path))
		}
	}
	fmt.Fprintf(b, "  %s\n", header)
	for _, l := range lines {
		if l.note == "" {
			fmt.Fprintf(b, "    %-9s  %s\n", l.action, l.path)
		} else {
			fmt.Fprintf(b, "    %-9s  %-*s  %s\n", l.action, width, l.path, l.note)
		}
	}
}

// collapse labels each set sharing one action and no reason in one line; others get a line per change.
func collapse(changes []project.Change, set func(string) string, label func(string, []project.Change) string,
) []planLine {
	sets := map[string][]project.Change{}
	for _, c := range changes {
		if set(c.Path) != "" {
			sets[set(c.Path)] = append(sets[set(c.Path)], c)
		}
	}
	var lines []planLine
	for _, c := range changes {
		members := sets[set(c.Path)]
		switch {
		case !uniform(members):
			lines = append(lines, planLine{action: string(c.Action), path: c.Path, note: c.Reason})
		case members[0].Path == c.Path:
			lines = append(lines, planLine{action: string(c.Action), path: label(set(c.Path), members)})
		}
	}
	return lines
}

func noSet(string) string { return "" }

func uniform(changes []project.Change) bool {
	return len(changes) > 0 && !slices.ContainsFunc(changes, func(c project.Change) bool {
		return c.Action != changes[0].Action || c.Reason != ""
	})
}

// frameworkSet names the core roles directory or a core skill's directory.
func frameworkSet(p string) string {
	if rest, ok := strings.CutPrefix(p, "docs/solomonar/agents/"); ok && !strings.Contains(rest, "/") {
		return "docs/solomonar/agents/"
	}
	if rest, ok := strings.CutPrefix(p, "docs/solomonar/skills/"); ok {
		skill, _, _ := strings.Cut(rest, "/")
		return "docs/solomonar/skills/" + skill + "/"
	}
	return ""
}

func frameworkLabel(dir string, set []project.Change) string {
	if dir == "docs/solomonar/agents/" {
		return fmt.Sprintf("%s (%d %s)", dir, len(set), plural(len(set), "role", "roles"))
	}
	files := make([]string, 0, len(set))
	for _, c := range set {
		files = append(files, strings.TrimPrefix(c.Path, dir))
	}
	slices.Sort(files)
	return dir + " (" + strings.Join(files, ", ") + ")"
}

// stubSet names the runtime skills or agents directory a stub is in.
func stubSet(p string) string {
	for _, dir := range []string{
		".claude/skills/", ".claude/agents/", ".agents/skills/", ".codex/agents/", ".github/skills/", ".github/agents/",
	} {
		if strings.HasPrefix(p, dir) {
			return dir
		}
	}
	return ""
}

// stubLabel counts a runtime directory's stubs, core versus installed extensions (UC-1).
func (r initRun) stubLabel(dir string, set []project.Change) string {
	kind := "skill"
	if strings.HasSuffix(dir, "agents/") {
		kind = "role"
	}
	var n, extensions int
	var adversarial bool
	var extra []string
	for _, c := range set {
		rel := strings.TrimPrefix(c.Path, dir)
		id, file, _ := strings.Cut(rel, "/")
		if kind == "role" {
			id = strings.TrimSuffix(id, path.Ext(id))
			if dir == ".github/agents/" {
				id = strings.TrimSuffix(id, ".agent")
			}
		} else if file != "SKILL.md" {
			extra = append(extra, rel)
			continue
		}
		n++
		adversarial = adversarial || kind == "role" && id == "adversarial-reviewer"
		if slices.ContainsFunc(slices.Concat(r.extensions, r.removed), func(e project.CatalogExtension) bool { return e.Name == id }) {
			extensions++
		}
	}
	summary := fmt.Sprintf("%d %s %s", n, kind, plural(n, "stub", "stubs"))
	if kind == "skill" || extensions > 0 {
		summary += fmt.Sprintf(": %d core", n-extensions)
	}
	if extensions > 0 {
		summary += fmt.Sprintf(", %d %s", extensions, plural(extensions, "extension", "extensions"))
	}
	runtime := "codex"
	if strings.HasPrefix(dir, ".claude/") {
		runtime = "claude"
	} else if strings.HasPrefix(dir, ".github/") {
		runtime = "copilot"
	}
	if model := r.models[runtime]; adversarial && model != "" {
		extra = append(extra, "adversarial-reviewer model "+model)
	}
	return dir + " (" + strings.Join(append([]string{summary}, extra...), "; ") + ")"
}

// extensionSet names the store directory of an extension file, or its v0.1.7 install directory.
func extensionSet(p string) string {
	for _, dir := range []string{"docs/solomonar/extensions/", "docs/solomonar/skills/"} {
		if rest, ok := strings.CutPrefix(p, dir); ok {
			name, _, _ := strings.Cut(rest, "/")
			return dir + name + "/"
		}
	}
	return ""
}

func extensionLabel(dir string, set []project.Change) string {
	return fmt.Sprintf("%s (%d %s)", dir, len(set), plural(len(set), "file", "files"))
}

// decisionLines notes the decision documents with the entries this run records.
func (r initRun) decisionLines(changes []project.Change) []planLine {
	lines := collapse(changes, noSet, nil)
	for i, c := range changes {
		var notes []string
		for _, d := range r.decisions {
			names := d.Selection()
			switch {
			case c.Action == project.ActionUnchanged:
			case c.Path == "docs/decisions.md" && d.Supersedes != "":
				notes = append(notes, d.ID+" "+d.Group+": "+names, d.Supersedes+" superseded")
			case c.Path == "docs/decisions.md":
				notes = append(notes, d.ID+" "+d.Group+": "+names)
			case c.Path == decisionDocument(d.Group):
				notes = append(notes, d.Group+": "+names+" ("+d.ID+")")
			}
		}
		lines[i].note = joinNotes(lines[i].note, strings.Join(notes, " · "))
	}
	return lines
}

func decisionDocument(group string) string {
	if group == "architecture" {
		return "docs/architecture.md"
	}
	return "docs/stack.md"
}

func joinNotes(notes ...string) string {
	return strings.Join(slices.DeleteFunc(notes, func(n string) bool { return n == "" }), "; ")
}
