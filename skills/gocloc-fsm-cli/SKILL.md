---
name: gocloc-fsm-cli
description: Build, extend, and maintain the gocloc CLI that counts total/code/comment/blank lines using per-language FSM parsers (not regex). Use when implementing scan/language/version commands, adding a new language parser, fixing parsing edge cases (inline code+comment, strings with comment tokens, nested block comments, begin/end comments), or updating scan output contracts (including total.files), tests, and benchmarks.
---

# gocloc FSM CLI

Use this skill to implement or evolve `gocloc` safely and consistently.

## Follow these non-negotiable constraints

1. Use FSM parsing, not regex-only parsing, for line classification.
2. Keep one language per file in `internal/languages/`; do not build a generic shared parser type.
3. Keep stream-based file reading and concurrent scanning behavior.
4. Preserve line semantics:
   - Allow one line to count as both `code` and `comment`.
   - Avoid counting comment tokens inside string literals as real comments.
   - Handle nested block comments where language supports them.
   - Handle begin/end style block comments where language supports them.
5. Keep method-level comments in each FSM engine clear, especially inside `analyze` and `processLine`.

## Apply this workflow

1. Confirm scope:
   - Command-layer change (`cmd/`)?
   - Scanner aggregation/output change (`internal/scanner`, `internal/report`, `internal/model`)?
   - Language-parser change (`internal/languages`)?
2. Implement smallest safe change first.
3. If adding a language:
   - Create `internal/languages/<language>_fsm.go`.
   - Implement `Analyzer` (`Name`, `Extensions`, `Analyze`).
   - Keep language-specific engine/state local to that file.
   - Register in `internal/languages/registry.go`.
4. Add or update tests:
   - Parser behavior tests in `internal/languages/`.
   - Scanner behavior tests in `internal/scanner/`.
   - Include single-file scan coverage.
5. Keep benchmarks current:
   - Update `internal/scanner/scanner_benchmark_test.go` when scan behavior/performance paths change.
6. Run formatting and validation:
   - `gofmt -w cmd internal main.go`
   - `go test ./...`
   - `go test -bench BenchmarkScan -run ^$ ./internal/scanner` (when touching scanner/perf-sensitive code)

## Preserve output contracts

For `scan --format json`, keep:

- File-level metrics under `files[]`.
- Language-level summaries under `languages[]` with `files` count.
- Project summary under `total` with:
  - `files`
  - `total`
  - `code`
  - `comment`
  - `blank`

For table output, keep total row aligned with files + line metrics.

## Quick implementation checklist

- FSM state transitions are explicit and comment-documented.
- EOF and final line without trailing newline are handled.
- Unsupported single-file extensions return a clear error.
- New behavior has tests and existing tests remain green.
- No unrelated files are modified.
