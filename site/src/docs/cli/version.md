---
title: solomonar version
navTitle: solomonar version
description: Print the Solomonar CLI version and the framework version and registry schema recorded in the current project with solomonar version.
kind: Reference
group: Commands
order: 15
sources:
  - internal/cli/version.go
  - docs/cli/interface.md
---

Prints the CLI version, which is also the version of the framework it installs, and the framework version and registry schema recorded in the project. Outside a project the project line reads `not initialized`.

{% cliUsage "After installing or updating the CLI, and when a project was set up with another release.", "You. It takes no flags besides the global ones." %}

## Synopsis

```text
solomonar version
```

## Examples

Inside an initialized project:

```sh
solomonar version
```

```text
solomonar 0.1.7
project   framework 0.1.7 · registry schema 2
```

Outside a project:

```text
solomonar 0.1.7
project   not initialized
```
