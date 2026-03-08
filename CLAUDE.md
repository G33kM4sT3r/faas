# CLAUDE.md

**faas** — self-hosted Function as a Service CLI in Go. Deploy stateless functions as containerized HTTP services. Single binary, plugin-based container runtime (Docker first), external language templates (Go, Python, Rust, PHP, TypeScript, JavaScript via Bun), config.toml per function, auto-assigned ports, structured JSON logging.

## Commands

```bash
make build                    # Production binary
make test                     # Tests with race detection
make test-coverage            # Coverage report
make check                    # fmt-check + vet + lint + compile audit
make help                     # Show all targets with descriptions
```

```bash
./bin/faas up [func]          # Build and deploy function
./bin/faas down [func]        # Stop and remove function
./bin/faas ls                 # List deployed functions
./bin/faas logs [func]        # Stream function logs
./bin/faas init [func]        # Generate config.toml
```

## Architecture

Dependencies flow **inward only** — lower layers never import higher layers:

```
cmd/faas/main.go → cmd/faas/ (root, up, down, ls, logs, init)
  ↓
internal/config/      internal/template/    internal/builder/
internal/runtime/     internal/health/      internal/logs/
  ↓
internal/ui/
```

### Key Components

| Package | Purpose |
|---------|---------|
| `cmd/faas/` | Cobra commands, CLI entry point |
| `internal/config/` | config.toml parsing (go-toml v2) + auto-generation |
| `internal/template/` | Language detection, template discovery + rendering |
| `internal/builder/` | Docker image building from rendered templates |
| `internal/runtime/` | Runtime interface + Docker implementation |
| `internal/health/` | Health check polling |
| `internal/logs/` | Structured JSON log streaming from containers |
| `internal/ui/` | Lipgloss styles + bubbletea spinner |
| `templates/` | External language template directories |

### State

- `~/.faas/state.json` — maps function names to paths, container IDs, ports
- `~/.faas/templates/` — user-defined custom language templates
- `~/.faas/logs/` — CLI logs (zerolog + lumberjack rotation)

## Dependencies

| Library | Version | Module Path |
|---------|---------|-------------|
| Cobra | v1.10.2 | `github.com/spf13/cobra` |
| Bubbletea | v2.0.1 | `charm.land/bubbletea/v2` |
| Bubbles | v2.0.0 | `charm.land/bubbles/v2` |
| Lipgloss | v2.0.0 | `charm.land/lipgloss/v2` |
| Zerolog | v1.34.0 | `github.com/rs/zerolog` |
| Lumberjack | v2.2.1 | `gopkg.in/natefinch/lumberjack.v2` |
| go-toml | v2.2.4 | `github.com/pelletier/go-toml/v2` |

## Implementation Discipline

- **Max performance.** Pre-allocate slices/maps, cache computed values, avoid heap escapes, eliminate hot-path `fmt.Sprintf`. Value semantics over pointer indirection where it prevents escapes.
- **Zero-allocation mindset.** Stack arrays only if backed slice never escapes. Prefer inlined type-switches over shared helpers when helper adds overhead or forces escapes.
- **Performance over DRY on hot paths.** Duplication acceptable when it eliminates function calls, extra comparisons, or heap escapes.
- **No TODOs in code.** Resolve in current session.
- **No anti-patterns.** No god functions, deep nesting, flag arguments, shotgun surgery, feature envy, data clumps, primitive obsession.
- **Circular deps:** Pointer indirection closures or structural interfaces. Never import cycles.
- **Signature changes:** Grep all callers, update them, `go build ./...` before proceeding.
- **Stale comments:** When editing/renaming, search for comments referencing old names.
- **Don't remove functional code.** Wire/implement unused exports rather than deleting.
- **Verify before fixing.** Read actual code before applying prescribed fixes — plans may be wrong.

## Go Style (Mandatory)

### File Organization

Package doc → package → imports (stdlib → external → internal) → public constants → private constants → public vars → private vars → public types → private types → **public functions/methods → private functions/methods**. Public before private applies to EVERYTHING.

### Naming

| Type | Convention | Example |
|------|------------|---------|
| Packages | lowercase, short | `config`, `runtime` |
| Exported | PascalCase | `BuildImage` |
| Unexported | camelCase | `loadConfig` |
| Acronyms | ALL CAPS | `URL`, `HTTP`, `ID` |

### Rules

- Newest Go features, no deprecated APIs; every export has GoDoc
- Return early, guard clauses — no deep nesting; no named/naked returns
- No name stuttering (`config.Load` not `config.LoadConfig`)
- No package-level vars (except `var Err*` sentinels); no `init()` functions
- `errors.Is()`/`errors.As()` for sentinels; `errors.Join(errs...)` for aggregation
- Always check error returns — `_ =` only for intentionally discarded cleanup errors
- Pre-allocate slices: `make([]T, 0, expectedLen)`; `strings.Builder` for loop concatenation

### Testing

- Race detection: `make test`; real filesystem over mocking; test error paths; table-driven for similar cases
- **Never work around failing tests.** Fix production code, not the test.

### CLI & Output

- `internal/ui/` uses lipgloss styles + bubbletea spinner
- Lipgloss `Style` vars are immutable — annotate with `//nolint:gochecknoglobals`

## Runtime Interface

```go
// internal/runtime/runtime.go
type Runtime interface {
    Start(ctx context.Context, opts StartOpts) (Container, error)
    Stop(ctx context.Context, id string) error
    Remove(ctx context.Context, id string) error
    Status(ctx context.Context, id string) (Status, error)
    Logs(ctx context.Context, id string, opts LogOpts) (io.ReadCloser, error)
    Build(ctx context.Context, opts BuildOpts) (Image, error)
}
```

Docker implements this interface. Future runtimes (K8s, Podman) add new files in `internal/runtime/`.

## Template System

Each language template is a self-contained directory:

```
templates/<language>/
├── Dockerfile            # Container build instructions
├── server.<ext>.tmpl     # HTTP wrapper, embeds user function
└── template.toml         # Metadata: name, extensions, port, health path, base image
```

Two embedding strategies based on language constraints:
- **Inline embedding** (Python, Rust, JS, TS): `{{.UserFunction}}` in template — works because these allow imports anywhere at module level
- **Separate handler file** (Go, PHP): Builder writes `handler.go`/`handler.php`, template uses native include (`Handler()` call / `require_once`). Required because Go's `package`+`import` and PHP's `<?php`+`use` break with inline embedding.

Custom templates at `~/.faas/templates/<language>/` override built-in ones.

### Dependency Support

- Universal `@` version separator in config.toml: `"requests@2.32"`, `"cocur/slugify@4.6"`
- `parsePackageVersion()` in builder translates to native format per language (`==` for pip, `"ver"` for npm/Cargo/Composer)
- `writeLanguageFiles()` dispatches to per-language manifest generators (requirements.txt, go.mod, Cargo.toml, composer.json, package.json)
- Go/Rust: manifests always generated (required for builds). Python/JS/TS/PHP: only when packages non-empty.
- PHP Dockerfile installs extensions (intl, zip, bcmath, sockets, pcntl, pdo_mysql, pdo_pgsql, gd) — many others already built into `php:8.5-cli-alpine`

## Template Gotchas

- **PHP extensions**: `php:8.5-cli-alpine` has mbstring, curl, dom, PDO, pdo_sqlite, opcache, xml, fileinfo built-in. Never re-install them via `docker-php-ext-install` (causes `cp: can't stat 'modules/*'`). `gd` requires `docker-php-ext-configure gd --with-freetype --with-jpeg` before install.
- **Adding new languages**: If the language has strict file-level declarations (like Go's `package`/`import` or PHP's `<?php`/`use`), use the separate handler file pattern — never inline embed.

## Port Management

- Default: auto-assign via OS ephemeral range (`:0`)
- Explicit port: validated against state.json + TCP dial check
- Conflict: fail fast with actionable error message

## Pre-Commit Workflow (Mandatory)

`make check` runs all four steps: fmt-check + vet + lint + compile audit. **Always run `make check` before committing.** Also run `make test` for race-detected tests.

## Lint Patterns

- Test functions: uppercase after `Test` — `TestBuildImage`, not `TestbuildImage`
- `//nolint` on the flagged line, not enclosing function
- Signature changes cascade: update callers, remove unused vars, re-check unparam
- Lint full packages, not individual files; run repeatedly until zero remain
- `defer x.Close()` → `defer func() { _ = x.Close() }()` (errcheck)
- `filepath.Join(dir, "sub/path")` → `filepath.Join(dir, "sub", "path")` (gocritic filepathJoin)
- Structs >80 bytes in params trigger `hugeParam` — pass by pointer
- Before removing imports, search ENTIRE file for usages

## Dead Code Removal

After refactoring: search for old function names, `make check` catches unused imports, verify no references in tests.

## Git & GitHub

- Always use SSH remotes (`git@github.com:...`), not HTTPS
- Use `gh` CLI for GitHub operations (releases, PRs, issues)
- CHANGELOG.md follows [Keep a Changelog](https://keepachangelog.com/) format

## GitHub Workflows

- `.github/workflows/ci.yml` — push/PR to main: fmt-check, vet, lint, test with coverage threshold, govulncheck
- `.github/workflows/release.yml` — on `v*` tag: test → cross-compile (linux/darwin × amd64/arm64) → GitHub Release with checksums
- Release archives use full tag in filename: `faas-v1.0.0-linux-amd64.tar.gz`
- Action versions: checkout@v6, setup-go@v6, upload-artifact@v7, download-artifact@v8, golangci-lint-action@v9, action-gh-release@v2
