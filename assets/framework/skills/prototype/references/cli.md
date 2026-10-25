# CLI prototypes

Framework version: 0.3.5

- Use the target language's standard argument parser when sufficient; for example Python `argparse` or
  Node.js `util.parseArgs`. Preserve required argument/flag semantics when choosing a parser.
- Demonstrate the agreed commands, positional arguments, flags, and help, with runnable success and failure
  invocations and expected output. Do not add commands solely for completeness.
- Send command results to stdout and diagnostics to stderr. Return zero on success and nonzero on failure;
  preserve any agreed specific exit codes. Make errors identify the problem and a corrective action.
- Keep commands predictable and scriptable: accept required inputs through arguments, flags, or stdin.
  If interaction is part of the agreed flow, provide a non-interactive path; never require a prompt in scripts.
- Use a prototype-local JSON file or equally simple mechanism when state must survive invocations.
  Inspect help, output streams, exit codes, invalid input, and state across the documented command sequence.
