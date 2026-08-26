# Custom Go code-quality analyzer

Static analysis CLI that complements `gofmt`, `go vet`, `golangci-lint`, `go test`, and `govulncheck`.
It is **not** a deployed service.

## Quick start

From the repository root:

```bash
# Analyze the working tree (all current violations)
make quality

# PR-style comparison against a base branch (only NEW degradations fail the gate)
make quality BASE=origin/main

# Write JSON + SARIF reports under ./dist
make quality-report BASE=origin/main
```

Or run the binary directly:

```bash
cd tools/quality
go run ./cmd/quality --root ../.. --base origin/main
```

## Configuration

Repository root [`.quality.yaml`](../../.quality.yaml) controls thresholds and `fail_on`.
If the file is missing, built-in defaults matching that file are used.

| Rule ID | Short name | What it measures | Minor | Major |
|---------|------------|------------------|-------|-------|
| `quality/methods-per-file` | `methods_per_file` | funcs/methods per file | >25 | >35 |
| `quality/returns-per-function` | `returns_per_function` | `return` stmts per function | >5 | >7 |
| `quality/complexity` | `complexity` | cyclomatic complexity | >5 | >10 |
| `quality/loc-per-file` | `loc_per_file` | source LOC per file | >250 | >400 |
| `quality/function-length` | `function_length` | source LOC per function | >20 | >30 |
| `quality/parameters` | `parameters` | parameters (grouped names count separately) | >5 | >8 |
| `quality/nesting-depth` | `nesting_depth` | max control/block nesting | >3 | >5 |

**LOC rules:** blank lines, comments, `package` lines, and `import` blocks are excluded from file LOC.
Function length excludes blanks and comments inside the function body.

**Cyclomatic complexity:** starts at 1; adds 1 for `if`, `for`, `range`, each `case`/`comm` clause, and each `&&` / `||`.

**Severity gate:** by default `fail_on: major` — only **new** MAJOR violations fail (exit 1). MINOR is reported. Override with `--fail-on minor|major|never`.

## Scope / exclusions

The analyzer scans production Go sources under `services/`, `pkg/`, and `tools/` by default.

**Ignored:**
- `*_test.go` test files (never analyzed, including in PR diffs)
- `testdata/` directories
- `vendor/`, `.git/`, `node_modules/`

## PR vs base comparison

```bash
quality --base origin/main
```

1. Resolves `git merge-base HEAD <base>`.
2. Lists changed `.go` files (`git diff`, including dirty working tree).
3. Analyzes current files and the same paths at the merge-base (`git show`).
4. Marks a violation **new** only when the metric is absent on the base or the actual value **increased**.

Unchanged legacy debt does not fail the PR.

## Outputs

| Flag | Description |
|------|-------------|
| `--format terminal` | Human-readable report (default) |
| `--format json --output report.json` | Machine-readable report |
| `--format sarif --output report.sarif` | SARIF 2.1.0 for GitHub code scanning |

Exit codes: `0` pass, `1` gate failed, `2` tool/usage error.

## Layout

```
tools/quality/
├── cmd/quality/     # CLI entrypoint
├── analyzer/        # AST parse, metrics, git diff comparison
├── rules/           # rule IDs, thresholds, severity
├── config/          # .quality.yaml loading
├── reporter/        # terminal, JSON, SARIF
└── testdata/        # fixtures (skipped during repo scans)
```

## Tests

```bash
make quality-test
# or
cd tools/quality && go test ./...
```
