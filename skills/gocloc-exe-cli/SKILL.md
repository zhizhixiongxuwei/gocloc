---
name: gocloc-exe-cli
description: Run the bundled Windows gocloc binary to count total/code/comment/blank lines without rebuilding from source. Use when a user asks to scan files or folders, list supported languages, or check version through a prebuilt executable.
---

# Gocloc EXE CLI

Use the bundled executable at `assets/gocloc.exe`.

## Run commands

- Run scan: `./assets/gocloc.exe scan <path>`
- Run language list: `./assets/gocloc.exe language`
- Run version: `./assets/gocloc.exe version`

## Output handling

- Return command output directly when the user asks for raw results.
- Summarize totals (`total`, `code`, `comment`, `blank`, `files`) when the user asks for interpretation.
- Keep paths user-provided unless the user asks to expand scope.

## Operational rules

- Execute from the skill directory or use an absolute path to `assets/gocloc.exe`.
- Prefer JSON output flags when the user asks for machine-readable output.
- Report command errors verbatim and include the command that failed.
