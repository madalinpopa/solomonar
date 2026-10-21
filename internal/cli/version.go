package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"runtime/debug"
	"strings"
)

type versionData struct {
	CLI              string  `json:"cli"`
	Framework        string  `json:"framework"`
	ProjectFramework *string `json:"project_framework"`
	RegistrySchema   *int    `json:"registry_schema"`
}

func runVersion(_ context.Context, env Env, _ sharedFlags, args []string) (output, error) {
	if len(args) > 0 {
		return output{}, unexpectedArgument(args[0])
	}
	framework, err := frameworkVersion(env.Assets)
	if err != nil {
		return output{}, err
	}
	data := versionData{CLI: cliVersion(buildVersion()), Framework: framework}
	text := fmt.Sprintf("solomonar %s (embedded framework %s)\nproject   not initialized\n", data.CLI, data.Framework)
	return output{data: data, text: text}, nil
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}

func cliVersion(raw string) string {
	if raw == "" || raw == "(devel)" {
		return "devel"
	}
	return strings.TrimPrefix(raw, "v")
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
