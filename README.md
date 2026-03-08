# FaaS - Self-hosted Function as a Service

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![CI](https://github.com/G33kM4sT3r/faas/actions/workflows/ci.yml/badge.svg)](https://github.com/G33kM4sT3r/faas/actions/workflows/ci.yml)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy_Me_A_Coffee-FFDD00?logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/martin.willig)

Write a function, deploy it as an HTTP service — single binary, no infrastructure setup, no YAML manifests. Just your code in a container.

## Install

Requires **Docker** to run functions.

**Download a release binary** (recommended):

Download the archive for your platform from the [latest release](https://github.com/G33kM4sT3r/faas/releases/latest), then extract and install:

```bash
tar xzf faas-*-linux-amd64.tar.gz    # or darwin-arm64, etc.
sudo mv faas-*-linux-amd64 /usr/local/bin/faas
```

Available for Linux and macOS (amd64/arm64).

**Build from source** (requires Go 1.26+):

```bash
git clone git@github.com:G33kM4sT3r/faas.git
cd faas && make build
```

The binary is written to `bin/faas`.

## Quick Start

```bash
# Write a function
cat > hello.py << 'EOF'
def handler(request):
    name = request.get("name", "world")
    return {"message": f"Hello, {name}!"}
EOF

# Deploy it
./bin/faas up hello.py

# Call it
curl -X POST http://localhost:<port> \
  -H "Content-Type: application/json" \
  -d '{"name": "World"}'

# {"message": "Hello, World!"}
```

That's it. No Dockerfile, no config file, no boilerplate. FaaS detects the language, generates the container, and starts serving.

## Supported Languages

| Language | Extension | Base Image | Runtime Image |
|----------|-----------|------------|---------------|
| Go | `.go` | `golang:1.26-alpine3.23` | `alpine:3.23` |
| Python | `.py` | `python:3.14-alpine3.23` | — |
| Rust | `.rs` | `rust:1.94-alpine3.23` | `alpine:3.23` |
| PHP | `.php` | `php:8.5-cli-alpine3.23` | — |
| TypeScript | `.ts` | `oven/bun:1-alpine` | — |
| JavaScript | `.js` | `oven/bun:1-alpine` | — |

Go and Rust use multi-stage builds — the final container runs on a minimal `alpine:3.23` image with no compiler toolchain. Python, PHP, and Bun-based languages run directly on their base image.

Each function implements a `handler` that receives a JSON body and returns a JSON response. See [docs/examples/](docs/examples/) for complete working examples, handler specifications, and dependency usage for every language.

## Commands

```
faas up <file|dir>     Build and deploy a function as an HTTP service
faas down [name]       Stop and remove a running function
faas ls                List deployed functions
faas logs <name>       Stream function logs
faas init <file>       Generate a config.toml for a function
```

```bash
faas up hello.py --name my-func --port 3000 --env API_KEY=secret
faas down my-func                    # or: faas down --all
faas ls --json                       # table, JSON, or --quiet
faas logs my-func --level error      # filter + follow by default
```

See [docs/cli-reference.md](docs/cli-reference.md) for the full CLI reference with all flags, configuration options, and dependency management.

## Architecture

Single binary CLI built on [Cobra](https://github.com/spf13/cobra). Dependencies flow inward — lower layers never import higher layers.

```
cmd/faas/          CLI commands (up, down, ls, logs, init)
  ↓
internal/
├── config/        config.toml parsing + generation
├── template/      Language detection, template rendering
├── builder/       Docker build context preparation
├── runtime/       Container runtime interface (Docker)
├── health/        Health check polling
├── state/         Deployed function state (~/.faas/state.json)
├── logs/          Structured JSON log streaming
├── logging/       CLI logging (zerolog + rotation)
└── ui/            Terminal styles + spinner (lipgloss, bubbletea)
```

The runtime is pluggable — Docker ships built-in, future backends (Podman, Kubernetes) implement the same interface.

## Custom Templates

Override built-in templates or add new languages by placing templates in `~/.faas/templates/<language>/`:

```
~/.faas/templates/ruby/
├── Dockerfile
├── server.rb.tmpl
└── template.toml
```

User-defined templates take precedence over built-in ones.

## Development

```bash
make build            # Build binary
make test             # Run tests with race detection
make check            # Format + vet + lint + compile audit
make test-coverage    # Generate coverage report
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for code conventions and workflow.

## License

[Apache License 2.0](LICENSE)
