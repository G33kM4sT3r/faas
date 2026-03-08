# Contributing to faas

Thanks for your interest in contributing to faas. This document covers the workflow and conventions you need to follow.

## Getting Started

```bash
git clone https://github.com/G33kM4sT3r/faas.git
cd faas && make build
```

Requires **Go 1.26+** and **Docker** (for building and running functions). The binary is written to `bin/faas`.

## Development Workflow

### Before Every Commit

All steps must pass. No exceptions.

```bash
gofmt -w .                    # Format
go vet ./...                  # Static analysis
golangci-lint run ./...       # Lint (fix ALL issues, including pre-existing)
make test                     # Tests with race detection
```

Or use the shorthand:

```bash
make check                    # fmt-check + vet + lint + compile audit
make test                     # All tests with race detection
```

### Running Tests

```bash
make test                     # All tests with race detection
make test-coverage            # Generate coverage report
make check                    # Format check + vet + lint + compile audit
```

E2E tests require Docker and exercise the full lifecycle (up → ls → invoke → down) for all supported languages.

## Architecture

Dependencies flow **inward only**. Lower layers never import higher layers.

```
cmd/faas/main.go → cmd/faas/ (root, up, down, ls, logs, init)
  ↓
internal/config/      internal/template/    internal/builder/
internal/runtime/     internal/health/      internal/logs/
  ↓
internal/ui/
```

| Package | Purpose |
|---------|---------|
| `cmd/faas/` | Cobra commands, CLI entry point |
| `internal/config/` | config.toml parsing + auto-generation |
| `internal/template/` | Language detection, template discovery + rendering |
| `internal/builder/` | Docker image building from rendered templates |
| `internal/runtime/` | Runtime interface + Docker implementation |
| `internal/health/` | Health check polling |
| `internal/logs/` | Structured JSON log streaming from containers |
| `internal/ui/` | Lipgloss styles + bubbletea spinner |
| `templates/` | External language template directories |

## Code Conventions

### File Organization

Every Go file follows this order:

1. Package doc comment (on the file matching the package name)
2. `package` declaration
3. Imports (stdlib, then external, then internal — separated by blank lines)
4. Public constants, then private constants
5. Public vars, then private vars
6. Public types, then private types
7. **Public functions/methods, then private functions/methods**

Public before private applies to everything.

### Naming

| Type | Convention | Example |
|------|------------|---------|
| Packages | lowercase, short | `config`, `runtime` |
| Exported | PascalCase | `BuildImage` |
| Unexported | camelCase | `loadConfig` |
| Acronyms | ALL CAPS | `URL`, `HTTP`, `ID` |

### Style Rules

- Use newest Go features, no deprecated APIs
- Every exported symbol must have a GoDoc comment
- Return early with guard clauses — avoid deep nesting
- No named or naked returns
- No package-level variables (except `var Err*` sentinels) — state on structs
- No `init()` functions
- No name stuttering (`config.Load` not `config.LoadConfig`)
- Use `errors.Is()` / `errors.As()` for error comparison
- Use `errors.Join(errs...)` for aggregating multiple errors
- Always check error returns — `_ =` only for intentionally discarded cleanup errors
- `defer x.Close()` → `defer func() { _ = x.Close() }()` (to satisfy errcheck)
- Pre-allocate slices when the size is known: `make([]T, 0, n)`
- Use `strings.Builder` for concatenation in loops

### Error Handling

```go
var ErrNotFound = errors.New("function not found")

if err != nil {
    return fmt.Errorf("building image %s: %w", name, err)
}
```

### Testing

- Always run with race detection (`make test`)
- Prefer real filesystem over mocks
- Test error paths, not just happy paths
- Use table-driven tests for similar cases
- Never work around failing tests — investigate root causes and fix the production code

## Implementation Discipline

- **Max performance.** Pre-allocate slices and maps, cache computed values, avoid heap escapes and unnecessary allocations. Value semantics over pointer indirection where it prevents escapes.
- **No TODOs or deferred work.** Never leave `// TODO`, `// FIXME`, or similar comments in committed code. Resolve in the current session or open an issue.
- **No anti-patterns.** Avoid god functions, deep nesting, flag arguments, shotgun surgery, feature envy, data clumps, primitive obsession. Refactor immediately when detected.
- **Circular dependency resolution:** Use pointer indirection closures or structural interfaces. Never create import cycles.
- **Signature change protocol:** When changing a function signature, grep all callers, update them, then run `go build ./...` before proceeding.
- **Stale comment sweep:** When renaming or refactoring, search for comments referencing the old name and update them.

## Template System

Each language template is a self-contained directory:

```
templates/<language>/
├── Dockerfile            # Container build instructions
├── server.<ext>.tmpl     # HTTP wrapper, embeds user function
└── template.toml         # Metadata: name, extensions, port, health path, base image
```

Supported languages: Go, Python, Rust, PHP, TypeScript, JavaScript (via Bun).

Custom templates at `~/.faas/templates/<language>/` override built-in ones. When adding a new language, create a template directory with all three files and add corresponding E2E test fixtures in `test/e2e/testdata/`.

## Lint Patterns

- Test functions: uppercase after `Test` — `TestBuildImage`, not `TestbuildImage`
- `//nolint` on the flagged line, not the enclosing function
- Lint full packages, not individual files; run repeatedly until zero issues remain
- `filepath.Join(dir, "sub/path")` → `filepath.Join(dir, "sub", "path")`
- Structs >80 bytes in parameters trigger `hugeParam` — pass by pointer

## Submitting Changes

1. Open an issue to discuss significant changes before writing code
2. Fork the repository and create a feature branch
3. Follow all conventions above
4. Ensure `make check` and `make test` pass with zero issues
5. Submit a pull request with a clear description of what and why

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
