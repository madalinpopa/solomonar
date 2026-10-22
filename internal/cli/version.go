package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/madalinpopa/solomonar/internal/git"
	"github.com/madalinpopa/solomonar/internal/project"
	"github.com/madalinpopa/solomonar/internal/task"
)

type versionData struct {
	CLI              string  `json:"cli"`
	Framework        string  `json:"framework"`
	ProjectFramework *string `json:"project_framework"`
	RegistrySchema   *int    `json:"registry_schema"`
}

func runVersion(ctx context.Context, env Env, flags sharedFlags, args []string) (output, error) {
	if len(args) > 0 {
		return output{}, unexpectedArgument(args[0])
	}
	framework, err := frameworkVersion(env.Assets)
	if err != nil {
		return output{}, err
	}
	data := versionData{CLI: framework, Framework: framework}
	line, err := projectVersions(ctx, env, flags, &data)
	if err != nil {
		return output{}, err
	}
	text := fmt.Sprintf("solomonar %s\nproject   %s\n", data.CLI, line)
	return output{data: data, text: text}, nil
}

// projectVersions reads only version fields, so a newer project still reports them (CLI-D19).
func projectVersions(ctx context.Context, env Env, flags sharedFlags, data *versionData) (string, error) {
	repo, err := openRepo(ctx, env, flags, flags.dir, "")
	if errors.Is(err, git.ErrNotRepository) {
		return "not initialized", nil
	}
	if err != nil {
		return "", err
	}
	manifest, err := readManifestFile(repo.Dir)
	if err != nil || manifest == nil {
		return "not initialized", err
	}
	schema, projectFramework, err := project.ManifestVersions(manifest)
	if isRejection(err, "manifest_invalid") {
		return "manifest invalid (inspect " + project.ManifestPath + ")", nil
	}
	if err != nil {
		return "", err
	}
	data.ProjectFramework = &projectFramework
	newerFramework, err := project.NewerFramework(projectFramework, data.Framework)
	if err != nil {
		return "", err
	}
	line := "framework " + projectFramework + " · "
	registrySchema, present, err := (&task.Store{Git: repo}).ReadSchema(ctx)
	switch {
	case isRejection(err, "registry_invalid"):
		line += "registry invalid"
	case err != nil:
		return "", err
	case !present:
		line += "registry missing"
	default:
		data.RegistrySchema = &registrySchema
		line += fmt.Sprintf("registry schema %d", registrySchema)
	}
	switch {
	case newerFramework:
		line += " (newer than this CLI; install solomonar " + projectFramework + " or later)"
	case schema > project.ManifestSchemaVersion || registrySchema > task.SchemaVersion:
		line += " (newer than this CLI; install a newer solomonar release)"
	}
	return line, nil
}

func readManifestFile(dir string) ([]byte, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open worktree: %w", err)
	}
	defer root.Close()
	data, err := root.ReadFile(project.ManifestPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	return data, nil
}

func isRejection(err error, code string) bool {
	rejection, ok := rejectionError("version", err)
	return ok && rejection.Code == code
}

func frameworkVersion(assets fs.FS) (string, error) {
	rules, err := fs.ReadFile(assets, "framework/rules.md")
	if err != nil {
		return "", fmt.Errorf("read framework version: %w", err)
	}
	for line := range strings.Lines(string(rules)) {
		if version, ok := strings.CutPrefix(strings.TrimSpace(line), "Framework version: "); ok {
			return version, nil
		}
	}
	return "", errors.New("read framework version: no version line in framework/rules.md")
}
