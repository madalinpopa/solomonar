---
title: Install the CLI
navTitle: Install the CLI
description: Install the Solomonar CLI with Homebrew or build it from source with Go, put it on your PATH, and check the binary, its version, and its JSON output.
kind: Tutorial
group: Start here
order: 2
sources:
  - go.mod
  - .goreleaser.yaml
  - Taskfile.yml
  - internal/cli/run.go
---

{% if release.version == "dev" %}There are no prebuilt releases yet. You build the binary from source; it takes a few seconds. From the first release on, the CLI also installs with Homebrew.
{% else %}Install a release with Homebrew, or build the binary from source.

## Install with Homebrew

On macOS and Linux:

```sh
brew install --cask madalinpopa/tap/solomonar
```

The cask puts `solomonar` on your `PATH`, so you can go straight to [checking the binary](#check-the-binary). On macOS it removes the quarantine attribute, because the binaries are not notarized. Upgrade to a new release with:

```sh
brew upgrade --cask solomonar
```
{% endif %}
## Build from source

Clone the repository and build:

```sh
git clone https://github.com/madalinpopa/solomonar.git
cd solomonar
go build -trimpath -o bin/solomonar .
```

The binary is `bin/solomonar`. It embeds the framework files it will install into your projects, so it needs nothing else at runtime besides Git.

## Put it on your PATH

The rest of the docs call `solomonar` by name. Copy the binary to a directory on your `PATH`, for example:

```sh
install -m 0755 bin/solomonar ~/.local/bin/solomonar
```

You can also skip this step and call `./bin/solomonar` from the checkout, or use its full path.

## Check the binary

```sh
solomonar version
```

```text
solomonar 0.3.7
project   not initialized
```

A build from a Git checkout between releases shows the last release version. Outside an initialized project, the project line reads `not initialized`.

List the commands this build supports:

```sh
solomonar help
solomonar help init
```

Every command also accepts `-h` and `--help`.

## Try the JSON output

Agents and scripts read the CLI through `--json`. Each command prints exactly one JSON object on stdout:

```sh
solomonar version --json
```

The object has `ok`, `command`, `data`, and `warnings` fields. The [output contract]({{ '/docs/cli/#output-formats' | url }}) describes it.

## Remove the CLI

Removing the binary does not touch your projects. To stop using Solomonar in a project, first run [`solomonar uninstall`]({{ '/docs/cli/uninstall/' | url }}) in each checkout to clean up, then commit the result there. It keeps the registry, your decisions, and your project extensions.

Then, only if you no longer need the CLI, remove the executable. A Homebrew install is removed with:

```sh
brew uninstall --cask solomonar
```

A binary you copied by hand is deleted by hand, for example `rm ~/.local/bin/solomonar`. Neither step changes any repository or your runtime applications' configuration.

## Next

Your binary works. Now [set up a project]({{ '/docs/set-up-a-project/' | url }}) with `solomonar init`.
