# FaaS Examples

Complete working examples for every supported language. Each directory contains a ready-to-deploy function, its `config.toml`, and a language-specific README with handler specifications, dependency usage, and deployment instructions.

## Languages

| Language | Directory | Runtime | Handler Signature |
|----------|-----------|---------|-------------------|
| [Go](go/) | `docs/examples/go/` | Go 1.26 (Alpine 3.23) | `func Handler(req map[string]any) map[string]any` |
| [Python](python/) | `docs/examples/python/` | Python 3.14 (Alpine 3.23) | `def handler(request): -> dict` |
| [Rust](rust/) | `docs/examples/rust/` | Rust 1.94 (Alpine 3.23) | `fn handler(input: Value) -> Value` |
| [PHP](php/) | `docs/examples/php/` | PHP 8.5 (Alpine 3.23) | `function handler(array $input): array` |
| [TypeScript](typescript/) | `docs/examples/typescript/` | Bun 1 (Alpine) | `function handler(body: Record<string, any>): Record<string, any>` |
| [JavaScript](javascript/) | `docs/examples/javascript/` | Bun 1 (Alpine) | `function handler(body) { return {...} }` |

## Quick Start

Pick any example, copy the function file, and deploy:

```bash
# Deploy the Python example
cp docs/examples/python/hello.py .
faas up hello.py

# Deploy with custom config
cp docs/examples/python/hello.py .
cp docs/examples/python/config.toml .
faas up hello.py
```

## How It Works

Every function follows the same pattern regardless of language:

1. **Receive** — FaaS sends a JSON body to your `handler` function as a parsed object/map/dict
2. **Process** — your function does whatever it needs (compute, transform, call APIs)
3. **Return** — return a JSON-serializable object/map/dict as the response

FaaS wraps your function in a lightweight HTTP server, builds a container image, and starts it. The container exposes a single POST endpoint at `/` and a health check at `/health`.

## Configuration

The `config.toml` in each example shows the full configuration surface. All fields are optional — FaaS auto-generates sensible defaults if no config exists.

```toml
[function]
name = "hello"           # Container name (derived from filename if omitted)
language = "python"      # Auto-detected from file extension
entrypoint = "hello.py"  # The function file

[runtime]
port = 0                 # 0 = auto-assign from OS ephemeral range
health_path = "/health"  # Health check endpoint

[build]
base_image = ""          # Override default base image
runtime_image = ""       # Override runtime image (Go/Rust multi-stage only)

[env]
# API_KEY = "your-key"  # Environment variables passed to the container

[dependencies]
packages = []            # External packages in "name@version" format
```

### Dependency Format

All languages use a universal `package@version` separator. The version is optional — omit it for the latest.

| Language | Example | Native Translation |
|----------|---------|-------------------|
| Python | `"requests@2.31.0"` | `requirements.txt`: `requests==2.31.0` |
| Go | `"github.com/fatih/color@v1.18.0"` | `go.mod` + `go get` |
| Rust | `"serde@1.0"` | `Cargo.toml`: `serde = "1.0"` |
| PHP | `"guzzlehttp/guzzle@^7.0"` | `composer.json`: `"guzzlehttp/guzzle": "^7.0"` |
| JavaScript | `"lodash@4.17.21"` | `package.json`: `"lodash": "4.17.21"` |
| TypeScript | `"@types/node@22.0.0"` | `package.json`: `"@types/node": "22.0.0"` |
