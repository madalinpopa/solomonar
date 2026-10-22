---
title: solomonar help
navTitle: solomonar help
description: Print the Solomonar command table or the exact usage of one command with solomonar help, or with -h and --help on any command.
kind: Reference
group: Commands
order: 16
sources:
  - internal/cli/help.go
  - internal/cli/run.go
---

Prints the command table and global flags, or the usage and description of one command. `-h` or `--help` on any command prints the same text.

{% cliUsage "Before you run a command you have not used yet, or to check the exact flags your build accepts.", "You, and agents that need to confirm a command's arguments." %}

## Synopsis

```text
solomonar help [<command>]
```

| Argument | Meaning |
| --- | --- |
| `<command>` | Optional. A command such as `init` or `task block`. An unknown command exits with a usage error. |

## Examples

```sh
solomonar help task plan
```

```text
Usage: solomonar task plan --title <title> [--depends-on <ID>]... [--related <ID>] [--json] [--dry-run]

Add a planned task under the next free W-NNN.
--depends-on and --related must name existing tasks; --dry-run runs every check and writes nothing.

Global flags: see 'solomonar help'.
```

`solomonar task plan --help` prints the same text. A missing command, a bare command group such as `task`, or an unknown command exits 2.

The JSON form wraps the text in the [output envelope]({{ '/docs/cli/#output-formats' | url }}):

```sh
solomonar help --json
```
