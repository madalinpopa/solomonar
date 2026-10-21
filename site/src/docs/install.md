---
title: Install the CLI
navTitle: Install the CLI
description: Build the Solomonar CLI from source with Go, put it on your PATH, and check the binary, its embedded framework version, and its JSON output.
kind: Tutorial
group: Start here
order: 2
sources:
  - go.mod
  - Taskfile.yml
  - internal/cli/run.go
---

There are no prebuilt releases yet. You build the binary from source; it takes a few seconds.

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
solomonar devel (embedded framework 0.1.0)
project   not initialized
```

A build from a Git checkout may show a source-derived version instead of `devel`, with `+dirty` when the checkout has local changes. Outside an initialized project, the project line reads `not initialized`.

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

## Next

Your binary works. Now [set up a project]({{ '/docs/set-up-a-project/' | url }}) with `solomonar init`.
