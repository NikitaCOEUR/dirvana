---
title: "CLI Reference"
weight: 35
---

# CLI Reference

All commands accept the global `--log-level` flag (`debug`, `info`, `warn`, `error`;
also settable via `DIRVANA_LOG_LEVEL`). Print the version with `dirvana --version`.

## Setup & hooks

### `dirvana setup`

Automatically installs the shell hook for the detected shell (or `--shell bash|zsh|fish`).
Use `--uninstall` / `-u` to remove it.

```bash
dirvana setup
dirvana setup --shell zsh
dirvana setup --uninstall
```

### `dirvana hook`

Prints the hook code for manual installation, if you prefer editing your RC file yourself.

```bash
dirvana hook --shell bash >> ~/.bashrc
```

## Authorization

### `dirvana allow [path]`

Authorizes a directory (default: current directory) so its config loads automatically.

If the effective configuration (inherited configs included) contains
`env: {sh: ...}` commands — which run automatically on every `cd` — they are
shown and consented to in the same interaction. You will only be prompted
again when those commands change. `--auto-approve-shell` skips the prompt
(useful for CI).

### `dirvana revoke [path]`

Removes authorization for a directory and invalidates its cache (including subdirectories).

### `dirvana list`

Lists all authorized directories.

## Configuration

### `dirvana init`

Creates a commented sample `.dirvana.yml` in the current directory,
or the global config with `--global` / `-g`.

### `dirvana edit`

Opens the local (or `--global`) config in `$VISUAL`/`$EDITOR`, creating it if needed.

### `dirvana validate [config-file]`

Validates a configuration file against the JSON Schema plus dirvana-specific rules.

### `dirvana schema [output-file]`

Prints (or writes with `--output`) the JSON Schema used for validation and editor integration.

## Environment

### `dirvana export`

Prints the shell code for the current directory (aliases, functions, env vars,
completion registration) plus cleanup code for the directory being left.
This is what the shell hook evaluates on every directory change — you rarely
call it directly, except to reload manually:

```bash
eval "$(dirvana export)"
```

### `dirvana status`

Shows a full report: detected shell, hook installation, configuration hierarchy,
authorization state, merged aliases/functions/env and cache contents.

On a terminal it opens a foldable view — `↑↓` to move, `→`/`←` to unfold and
fold a section, `a` for all of them, `q` to leave. When something stands
between dirvana and a working shell, a Setup section says so and `⏎` applies
the fix without leaving the view.

Redirect it and the same report is printed at once, every section unfolded;
`--plain` forces that on a terminal too.

### `dirvana clean`

Removes the cache entries of the current directory hierarchy, or everything with `--all`.

## Internal commands

Two hidden commands exist for the generated shell code; they are not meant
to be called manually:

- `dirvana exec <alias> [args...]` — every generated alias wraps this; it
  resolves the alias (evaluating `when:`/`else:` conditions) and replaces the
  process with the resolved command.
- `dirvana completion [words...]` — called by the generated completion
  functions to produce suggestions.
