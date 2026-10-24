package project

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	goLegacySkill = "docs/solomonar/skills/go/SKILL.md"
	claudeEntry   = ".claude/skills/solomonar/SKILL.md"
	untracked     = "untracked; left in place"
)

func removeManifest() Change {
	return Change{Path: ManifestPath, Action: ActionRemove, Owner: "manifest"}
}

func hashed(owner, content string) File {
	return File{Owner: owner, SHA256: Hash([]byte(content))}
}

func installed(files map[string]File) *Manifest {
	return &Manifest{
		Runtimes:   map[string]Runtime{"claude": {}},
		Extensions: map[string]Extension{"go": {State: "installed", Kind: "skill", Group: "language", Source: "catalog"}},
		Files:      files,
	}
}

func TestUninstallPlansManagedFilesFromTheirRecords(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{
		rulesPath:                    {content: contentA, mode: 0o644},
		"docs/solomonar/reviewer.md": {content: contentB, mode: 0o644},
		goLegacySkill:                {content: contentA, mode: 0o644},
		claudeEntry:                  {content: contentA, mode: 0o644},
		".claude/skills/mine/x.md":   {content: contentC, mode: 0o644},
	})
	m := installed(map[string]File{
		rulesPath:                    hashed("core", contentA),
		"docs/solomonar/report.md":   hashed("core", contentA),
		"docs/solomonar/reviewer.md": hashed("core", contentA),
		goLegacySkill:                hashed("ext:go", contentA),
		claudeEntry:                  hashed("runtime:claude", contentA),
		"docs/decisions.md":          {Owner: "decision", Entries: []string{"P/D-1"}},
	})

	u, err := PlanUninstall(root, m, Resolutions{})

	require.NoError(t, err)
	assert.Equal(t, []Change{
		{Path: claudeEntry, Action: ActionRemove, Owner: "runtime:claude"},
		{Path: "docs/solomonar/reviewer.md", Action: ActionConflict, Owner: "core", Reason: modified},
		{Path: rulesPath, Action: ActionRemove, Owner: "core"},
		{Path: goLegacySkill, Action: ActionRemove, Owner: "ext:go"},
		removeManifest(),
	}, u.Plan.Changes)
}

func TestUninstallResolvesConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		keep      []string
		overwrite []string
		change    Change
	}{
		{name: "unresolved", change: Change{Path: rulesPath, Action: ActionConflict, Owner: "core", Reason: modified}},
		{
			name: "keep untracks", keep: []string{rulesPath},
			change: Change{Path: rulesPath, Action: ActionKeep, Owner: "core", Reason: untracked},
		},
		{
			name: "overwrite removes", overwrite: []string{rulesPath},
			change: Change{Path: rulesPath, Action: ActionRemove, Owner: "core"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, map[string]entry{rulesPath: {content: contentB, mode: 0o644}})
			m := installed(map[string]File{rulesPath: hashed("core", contentA)})

			u, err := PlanUninstall(root, m, Resolutions{Keep: tt.keep, Overwrite: tt.overwrite})

			require.NoError(t, err)
			assert.Equal(t, []Change{tt.change, removeManifest()}, u.Plan.Changes)
		})
	}
}

func TestUninstallRejectsInvalidResolutions(t *testing.T) {
	t.Parallel()

	const report = "docs/solomonar/report.md"
	tests := []struct {
		name      string
		keep      []string
		overwrite []string
		err       error
	}{
		{
			name: "in both flags", keep: []string{rulesPath}, overwrite: []string{rulesPath},
			err: Resolutions{}.invalid("%s is in both --keep and --overwrite", rulesPath),
		},
		{name: "plain removal", keep: []string{report}, err: Resolutions{}.invalid("%s is not a conflict", report)},
		{name: "unplanned path", overwrite: []string{"src/main.go"}, err: Resolutions{}.invalid("%s is not a conflict", "src/main.go")},
		{
			name: "overwrite of a directory", overwrite: []string{goLegacySkill},
			err: &Rejection{
				Code:  "invalid_value",
				Check: "--keep and --overwrite resolve conflicts",
				Found: goLegacySkill + " is not a regular file",
				Fix:   "remove it by hand or pass --keep " + goLegacySkill,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, map[string]entry{
				rulesPath:                {content: contentB, mode: 0o644},
				report:                   {content: contentA, mode: 0o644},
				goLegacySkill + "/inner": {content: contentA, mode: 0o644},
			})
			m := installed(map[string]File{
				rulesPath:     hashed("core", contentA),
				report:        hashed("core", contentA),
				goLegacySkill: hashed("ext:go", contentA),
			})

			_, err := PlanUninstall(root, m, Resolutions{Keep: tt.keep, Overwrite: tt.overwrite})

			assert.Equal(t, tt.err, err)
		})
	}
}

func TestUninstallNeverFollowsUnsafeTargets(t *testing.T) {
	t.Parallel()

	const (
		linked   = "docs/solomonar/rules.md"
		fifo     = "docs/solomonar/report.md"
		inside   = ".claude/agents/reviewer.md"
		outside  = ".claude/skills/go/SKILL.md"
		dirEntry = goLegacySkill
	)
	root, dir := openTree(t, map[string]entry{
		"docs/target.md":          {content: contentA, mode: 0o644},
		"docs/agents/reviewer.md": {content: contentA, mode: 0o644},
		dirEntry + "/inner":       {content: contentA, mode: 0o644},
	})
	elsewhere := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(elsewhere, "go"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(elsewhere, "go", "SKILL.md"), []byte(contentA), 0o644))
	require.NoError(t, os.Symlink("../target.md", filepath.Join(dir, linked)))
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, fifo), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))
	require.NoError(t, os.Symlink("../docs/agents", filepath.Join(dir, ".claude/agents")))
	require.NoError(t, os.Symlink(elsewhere, filepath.Join(dir, ".claude/skills")))
	m := installed(map[string]File{
		linked:   hashed("core", contentA),
		fifo:     hashed("core", contentA),
		inside:   hashed("runtime:claude", contentA),
		outside:  hashed("runtime:claude", contentA),
		dirEntry: hashed("ext:go", contentA),
	})

	u, err := PlanUninstall(root, m, Resolutions{Keep: []string{outside}})

	require.NoError(t, err)
	assert.Equal(t, []Change{
		{Path: inside, Action: ActionConflict, Owner: "runtime:claude", Reason: notRegular},
		{Path: outside, Action: ActionKeep, Owner: "runtime:claude", Reason: untracked},
		{Path: fifo, Action: ActionConflict, Owner: "core", Reason: notRegular},
		{Path: linked, Action: ActionConflict, Owner: "core", Reason: notRegular},
		{Path: dirEntry, Action: ActionConflict, Owner: "ext:go", Reason: notRegular},
		removeManifest(),
	}, u.Plan.Changes)
	_, err = PlanUninstall(root, m, Resolutions{Overwrite: []string{inside}})
	assert.Equal(t, inside+" is not a regular file", rejectionOf(t, err).Found)
}

func TestUninstallConflictsOnRecordsItCannotRemove(t *testing.T) {
	t.Parallel()

	const (
		settings = ".claude/settings.json"
		source   = "src/main.go"
		gitMeta  = ".git/config"
	)
	root, _ := openTree(t, map[string]entry{
		settings: {content: "{}\n", mode: 0o644},
		source:   {content: contentA, mode: 0o644},
		gitMeta:  {content: contentA, mode: 0o644},
	})
	m := installed(map[string]File{
		settings: {Owner: "runtime:claude", Entry: "hooks.PreToolUse[solomonar-guard]"},
		source:   hashed("core", contentA),
		gitMeta:  hashed("core", contentA),
	})
	outside := func(path string) Change {
		return Change{Path: path, Action: ActionConflict, Owner: "core", Reason: "outside the paths solomonar uninstall removes"}
	}
	hook := Change{Path: settings, Action: ActionConflict, Owner: "runtime:claude", Reason: "hook entry removal is not supported"}

	u, err := PlanUninstall(root, m, Resolutions{})
	require.NoError(t, err)
	assert.Equal(t, []Change{hook, outside(gitMeta), outside(source), removeManifest()}, u.Plan.Changes)

	u, err = PlanUninstall(root, m, Resolutions{Keep: []string{source}})
	require.NoError(t, err)
	assert.Contains(t, u.Plan.Changes, Change{Path: source, Action: ActionKeep, Owner: "core", Reason: untracked})

	_, err = PlanUninstall(root, m, Resolutions{Overwrite: []string{source}})
	assert.Equal(t, &Rejection{
		Code:  "invalid_value",
		Check: "--keep and --overwrite resolve conflicts",
		Found: source + " is outside the paths solomonar uninstall removes",
		Fix:   "remove it by hand or pass --keep " + source,
	}, err)

	for _, r := range []Resolutions{{Keep: []string{settings}}, {Overwrite: []string{settings}}} {
		_, err = PlanUninstall(root, m, r)
		assert.Equal(t, &Rejection{
			Code:  "invalid_value",
			Check: "--keep and --overwrite resolve conflicts",
			Found: settings + " holds a hook entry, which uninstall cannot remove yet",
			Fix:   "keep the installation until hook entry removal ships with the guard hooks",
		}, err)
	}
}

func TestUninstallKeepsProjectExtensionSources(t *testing.T) {
	t.Parallel()

	const (
		store     = "docs/solomonar/extensions/"
		kept      = "project extension; source kept"
		goFile    = store + "go/SKILL.md"
		mineFile  = store + "mine/SKILL.md"
		oldFile   = store + "legacy/SKILL.md"
		neighbors = store + "mine/extra.md"
	)
	files := map[string]entry{
		goFile:                      {content: contentA, mode: 0o644},
		mineFile:                    {content: contentA, mode: 0o644},
		neighbors:                   {content: contentB, mode: 0o644},
		oldFile:                     {content: contentB, mode: 0o644},
		store + "draft/SKILL.md":    {content: contentC, mode: 0o644},
		store + "unlisted/SKILL.md": {content: contentC, mode: 0o644},
	}
	root, _ := openTree(t, files)
	project := func(name, state string) Extension {
		return Extension{State: state, Kind: "skill", Group: "workflow", Source: "path", Path: store + name}
	}
	m := installed(map[string]File{
		goFile:   hashed("ext:go", contentA),
		mineFile: hashed("ext:mine", contentA),
		oldFile:  hashed("project-ext:legacy", contentA),
	})
	m.Extensions["mine"] = project("mine", "installed")
	m.Extensions["draft"] = project("draft", "draft")
	m.Extensions["legacy"] = project("legacy", "installed")

	u, err := PlanUninstall(root, m, Resolutions{})

	require.NoError(t, err)
	assert.Equal(t, []Change{
		{Path: store + "draft", Action: ActionSkip, Owner: "project-ext:draft", Reason: kept},
		{Path: goFile, Action: ActionRemove, Owner: "ext:go"},
		{Path: store + "legacy", Action: ActionSkip, Owner: "project-ext:legacy", Reason: kept},
		{Path: oldFile, Action: ActionSkip, Owner: "project-ext:legacy", Reason: kept},
		{Path: store + "mine", Action: ActionSkip, Owner: "project-ext:mine", Reason: kept},
		{Path: mineFile, Action: ActionSkip, Owner: "ext:mine", Reason: kept},
		removeManifest(),
	}, u.Plan.Changes)
	_, err = PlanUninstall(root, m, Resolutions{Keep: []string{oldFile}})
	assert.Equal(t, Resolutions{}.invalid("%s is not a conflict", oldFile), err)
}

func TestUninstallRemovesOnlyTheManagedPartOfAgents(t *testing.T) {
	t.Parallel()

	const (
		managedPart = "# Framework\n\nrules\n\n"
		comment     = "<!-- Project-owned instructions; preserved by framework upgrades. Keep this heading unchanged. -->\n"
		oldComment  = "<!-- Project-specific instructions go here, e.g. adopted style guides (such as Go Code Review Comments),\n" +
			"branch rules for direct commits, and the test policy. This section is owned by the project and preserved by\n" +
			"framework upgrades. -->\n"
		template   = notesHeading + "\n\n" + comment
		ownNotes   = template + "\n- Run go test before committing.\n"
		adopted    = notesHeading + "\n\n# Ledger\n\nUse tabs.\n"
		notesKept  = "managed part removed; project notes kept"
		noBoundary = `no "## Project notes" boundary`
	)
	update := func(notes string) Change {
		return Change{Path: "AGENTS.md", Action: ActionUpdate, Owner: "core+notes", Content: []byte(notes), Reason: notesKept}
	}
	remove := Change{Path: "AGENTS.md", Action: ActionRemove, Owner: "core+notes"}
	tests := []struct {
		name      string
		disk      string
		keep      []string
		overwrite []string
		change    Change
	}{
		{name: "notes edited", disk: managedPart + ownNotes, change: update(ownNotes)},
		{name: "adopted instructions", disk: managedPart + adopted, change: update(adopted)},
		{name: "template notes", disk: managedPart + template, change: remove},
		{name: "older template notes", disk: managedPart + notesHeading + "\n\n" + oldComment + "\n\n", change: remove},
		{name: "heading only", disk: managedPart + notesHeading, change: remove},
		{name: "no notes part", disk: managedPart, change: remove},
		{
			name: "changed managed part", disk: "# Mine\n" + ownNotes,
			change: Change{Path: "AGENTS.md", Action: ActionConflict, Owner: "core+notes", Reason: modified},
		},
		{
			name: "changed managed part kept", disk: "# Mine\n" + ownNotes, keep: []string{"AGENTS.md"},
			change: Change{Path: "AGENTS.md", Action: ActionKeep, Owner: "core+notes", Reason: untracked},
		},
		{name: "changed managed part overwritten", disk: "# Mine\n" + ownNotes, overwrite: []string{"AGENTS.md"}, change: update(ownNotes)},
		{name: "changed template overwritten", disk: "# Mine\n" + template, overwrite: []string{"AGENTS.md"}, change: remove},
		{
			name: "missing boundary", disk: "# Mine\n\n- Run go test.\n",
			change: Change{Path: "AGENTS.md", Action: ActionConflict, Owner: "core+notes", Reason: noBoundary},
		},
		{
			name: "missing boundary kept", disk: "# Mine\n\n- Run go test.\n", keep: []string{"AGENTS.md"},
			change: Change{Path: "AGENTS.md", Action: ActionKeep, Owner: "core+notes", Reason: untracked},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, map[string]entry{"AGENTS.md": {content: tt.disk, mode: 0o644}})
			m := installed(map[string]File{"AGENTS.md": hashed("core+notes", managedPart)})

			u, err := PlanUninstall(root, m, Resolutions{Keep: tt.keep, Overwrite: tt.overwrite})

			require.NoError(t, err)
			assert.Equal(t, []Change{tt.change, removeManifest()}, u.Plan.Changes)
		})
	}
}

func TestUninstallRejectsOverwritingAgentsWithoutBoundary(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{"AGENTS.md": {content: "# Mine\n", mode: 0o644}})
	m := installed(map[string]File{"AGENTS.md": hashed("core+notes", "# Framework\n")})

	_, err := PlanUninstall(root, m, Resolutions{Overwrite: []string{"AGENTS.md"}})

	assert.Equal(t, &Rejection{
		Code:  "invalid_value",
		Check: "--keep and --overwrite resolve conflicts",
		Found: `AGENTS.md has no "## Project notes" boundary`,
		Fix:   "restore the boundary by hand or pass --keep AGENTS.md",
	}, err)
}

func TestUninstallRemovesOnlyTheGeneratedIndexLine(t *testing.T) {
	t.Parallel()

	const (
		readme     = "docs/README.md"
		line       = "- `docs/solomonar/`: framework rules, roles, and skills"
		removed    = "framework index line removed"
		customized = "framework index entry customized; review by hand"
	)
	update := func(content string) Change {
		return Change{Path: readme, Action: ActionUpdate, Owner: "seed", Content: []byte(content), Reason: removed}
	}
	skip := func(reason string) Change {
		return Change{Path: readme, Action: ActionSkip, Owner: "seed", Reason: reason}
	}
	tests := []struct {
		name   string
		disk   string
		change *Change
		review bool
	}{
		{
			name:   "exact line",
			disk:   "# P\n\n## Documents\n\n" + line + "\n- `docs/stack.md`: stack\n\n## Commands\n",
			change: new(update("# P\n\n## Documents\n\n- `docs/stack.md`: stack\n\n## Commands\n")),
		},
		{
			name:   "line endings kept",
			disk:   "## Documents\r\n\r\n" + line + "\r\n- `docs/stack.md`: stack\r\n",
			change: new(update("## Documents\r\n\r\n- `docs/stack.md`: stack\r\n")),
		},
		{name: "last line", disk: "## Documents\n\n" + line, change: new(update("## Documents\n\n"))},
		{
			name: "other references remain", disk: "## Documents\n\n" + line + "\n- `docs/solomonar/rules.md`: rules\n",
			change: new(update("## Documents\n\n- `docs/solomonar/rules.md`: rules\n")), review: true,
		},
		{
			name: "customized entry", disk: "## Documents\n\n- `docs/solomonar/`: the Solomonar process\n",
			change: new(skip(customized)), review: true,
		},
		{name: "indented entry", disk: "## Documents\n\n  " + line + "\n", change: new(skip(customized)), review: true},
		{name: "outside Documents", disk: "## Notes\n\n" + line + "\n", change: new(skip(customized)), review: true},
		{name: "no reference", disk: "## Documents\n\n- `docs/stack.md`: stack\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, map[string]entry{readme: {content: tt.disk, mode: 0o644}})
			m := installed(map[string]File{readme: {Owner: "seed"}})

			u, err := PlanUninstall(root, m, Resolutions{})

			require.NoError(t, err)
			want := []Change{removeManifest()}
			if tt.change != nil {
				want = []Change{*tt.change, removeManifest()}
			}
			assert.Equal(t, want, u.Plan.Changes)
			if tt.review {
				assert.Equal(t, []string{readme}, u.Review)
			} else {
				assert.Empty(t, u.Review)
			}
		})
	}
}

func TestUninstallNeverRewritesAnUnsafeIndex(t *testing.T) {
	t.Parallel()

	for _, link := range []string{"docs", "docs/README.md"} {
		t.Run(link, func(t *testing.T) {
			t.Parallel()

			root, dir := openTree(t, map[string]entry{
				"real/README.md": {content: "## Documents\n\n- `docs/solomonar/`: framework rules, roles, and skills\n", mode: 0o644},
			})
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs"), 0o755))
			if link == "docs" {
				require.NoError(t, os.Remove(filepath.Join(dir, "docs")))
				require.NoError(t, os.Symlink("real", filepath.Join(dir, "docs")))
			} else {
				require.NoError(t, os.Symlink("../real/README.md", filepath.Join(dir, link)))
			}

			u, err := PlanUninstall(root, installed(map[string]File{}), Resolutions{})

			require.NoError(t, err)
			assert.Equal(t, []Change{
				{Path: "docs/README.md", Action: ActionSkip, Owner: "seed", Reason: notRegular}, removeManifest(),
			}, u.Plan.Changes)
			assert.Equal(t, []string{"docs/README.md"}, u.Review)
		})
	}
}

func TestUninstallWithoutIndexPlansNoIndexChange(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, nil)

	u, err := PlanUninstall(root, installed(map[string]File{"docs/README.md": {Owner: "seed"}}), Resolutions{})

	require.NoError(t, err)
	assert.Equal(t, []Change{removeManifest()}, u.Plan.Changes)
}

func TestUninstallReportsADanglingAgentsImport(t *testing.T) {
	t.Parallel()

	const managedPart = "# Framework\n"
	tests := []struct {
		name   string
		agents string
		claude File
		keep   []string
		disk   string
		review []string
	}{
		{name: "unmanaged import", agents: managedPart, disk: "# Mine\n@AGENTS.md\n", review: []string{"CLAUDE.md"}},
		{name: "notes kept", agents: managedPart + notesHeading + "\n\nmine\n", disk: "@AGENTS.md\n"},
		{name: "no import", agents: managedPart, disk: "# Mine\n"},
		{name: "managed and removed", agents: managedPart, claude: hashed("runtime:claude", "@AGENTS.md\n"), disk: "@AGENTS.md\n"},
		{
			name: "managed and kept", agents: managedPart, claude: hashed("runtime:claude", "@AGENTS.md\n"),
			keep: []string{"CLAUDE.md"}, disk: "@AGENTS.md\r\nmine\r\n", review: []string{"CLAUDE.md"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _ := openTree(t, map[string]entry{
				"AGENTS.md": {content: tt.agents, mode: 0o644},
				"CLAUDE.md": {content: tt.disk, mode: 0o644},
			})
			files := map[string]File{"AGENTS.md": hashed("core+notes", managedPart)}
			if tt.claude.Owner != "" {
				files["CLAUDE.md"] = tt.claude
			}

			u, err := PlanUninstall(root, installed(files), Resolutions{Keep: tt.keep})

			require.NoError(t, err)
			assert.Equal(t, tt.review, u.Review)
		})
	}
}

func TestUninstallPlanningWritesNothing(t *testing.T) {
	t.Parallel()

	const managedPart = "# Framework\n"
	root, dir := openTree(t, map[string]entry{
		ManifestPath:     {content: "{}\n", mode: 0o644},
		"AGENTS.md":      {content: managedPart + notesHeading + "\n\nmine\n", mode: 0o644},
		"CLAUDE.md":      {content: "@AGENTS.md\n", mode: 0o600},
		"docs/README.md": {content: "## Documents\n\n- `docs/solomonar/`: framework rules, roles, and skills\n", mode: 0o644},
		rulesPath:        {content: contentA, mode: 0o644},
		goLegacySkill:    {content: contentB, mode: 0o644},
		"docs/solomonar/extensions/mine/SKILL.md": {content: contentC, mode: 0o644},
		"real/SKILL.md": {content: contentA, mode: 0o644},
	})
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))
	require.NoError(t, os.Symlink("../real", filepath.Join(dir, ".claude/skills")))
	m := installed(map[string]File{
		"AGENTS.md":      hashed("core+notes", managedPart),
		rulesPath:        hashed("core", contentA),
		goLegacySkill:    hashed("ext:go", contentA),
		claudeEntry:      hashed("runtime:claude", contentA),
		"src/main.go":    hashed("core", contentA),
		"docs/README.md": {Owner: "seed"},
	})
	m.Extensions["mine"] = Extension{State: "draft", Kind: "skill", Group: "workflow", Source: "path", Path: "docs/solomonar/extensions/mine"}
	before := snapshot(t, dir)

	u, err := PlanUninstall(root, m, Resolutions{})
	require.NoError(t, err)
	assert.Len(t, u.Plan.Conflicts(), 3)
	_, err = PlanUninstall(root, m, Resolutions{Overwrite: []string{claudeEntry}})
	require.Error(t, err)
	u, err = PlanUninstall(root, m, Resolutions{Keep: []string{claudeEntry, "src/main.go"}, Overwrite: []string{goLegacySkill}})

	require.NoError(t, err)
	assert.Empty(t, u.Plan.Conflicts())
	assert.Equal(t, before, snapshot(t, dir))
}

func TestReadUninstallManifestNeverFollowsAnUnsafeManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
	}{
		{name: "symlink", setup: func(t *testing.T, dir string) {
			t.Helper()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "real.json"), []byte("{}\n"), 0o644))
			require.NoError(t, os.Mkdir(filepath.Join(dir, ".solomonar"), 0o755))
			require.NoError(t, os.Symlink("../real.json", filepath.Join(dir, ManifestPath)))
		}},
		{name: "symlinked directory", setup: func(t *testing.T, dir string) {
			t.Helper()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "real"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "real", "manifest.json"), []byte("{}\n"), 0o644))
			require.NoError(t, os.Symlink("real", filepath.Join(dir, ".solomonar")))
		}},
		{name: "directory", setup: func(t *testing.T, dir string) {
			t.Helper()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, ManifestPath), 0o755))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, dir := openTree(t, nil)
			tt.setup(t, dir)

			data, err := ReadUninstallManifest(root)

			assert.Nil(t, data)
			assert.Equal(t, &Rejection{
				Code:  "file_conflict",
				Check: "the manifest is a regular file",
				Found: ManifestPath + " exists and is not a regular file",
				Fix:   "restore " + ManifestPath + " as a regular file, then rerun solomonar uninstall",
			}, err)
		})
	}
}

func TestReadUninstallManifestReadsARegularManifest(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{ManifestPath: {content: "{}\n", mode: 0o644}})
	data, err := ReadUninstallManifest(root)
	require.NoError(t, err)
	assert.Equal(t, "{}\n", string(data))

	absent, _ := openTree(t, nil)
	data, err = ReadUninstallManifest(absent)
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestUninstallWithoutManifestOnlyChecksResolutions(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{rulesPath: {content: contentA, mode: 0o644}})

	u, err := PlanUninstall(root, nil, Resolutions{})
	require.NoError(t, err)
	assert.Equal(t, Uninstall{}, u)

	_, err = PlanUninstall(root, nil, Resolutions{Keep: []string{rulesPath}})
	assert.Equal(t, rulesPath+" is not a conflict", rejectionOf(t, err).Found)
}

func TestUninstallPrunesTheFrameworkDirectories(t *testing.T) {
	t.Parallel()

	root, _ := openTree(t, map[string]entry{ManifestPath: {content: "{}\n", mode: 0o644}})

	u, err := PlanUninstall(root, installed(map[string]File{}), Resolutions{})

	require.NoError(t, err)
	assert.Equal(t, []string{"docs/solomonar/extensions", "docs/solomonar", ".solomonar"}, u.Plan.Prune)
}
