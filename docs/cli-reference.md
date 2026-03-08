# CLI Reference

Complete reference for the `faas` command-line interface.

## Global Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--verbose` | `-v` | bool | `false` | Enable debug logging |
| `--version` | — | bool | — | Print version and exit |

Global flags can be used with any command:

```bash
faas -v up hello.py        # Deploy with debug logging
faas --version             # Print version (format: <version> (<commit>))
```

---

## faas up

Deploy a function as a containerized HTTP service.

```
faas up <file|directory> [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `file` | yes | Path to the function file (`.py`, `.go`, `.rs`, `.php`, `.ts`, `.js`) |
| `directory` | yes | Path to a directory containing `config.toml` |

Exactly one argument is required. Pass either a function file (language is auto-detected from the extension) or a directory that contains a `config.toml`.

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--port` | `-p` | int | `0` | Host port to expose. `0` auto-assigns from the OS ephemeral range. |
| `--name` | `-n` | string | — | Override the function name. Defaults to the filename without extension. |
| `--env` | `-e` | string[] | — | Set environment variable as `KEY=VALUE`. Repeatable. |
| `--force` | — | bool | `false` | Redeploy if a function with the same name is already running. |
| `--no-cache` | — | bool | `false` | Force a full Docker rebuild (no layer cache). |

### Behavior

- Auto-generates `config.toml` if none exists alongside the function file
- Detects the language from the file extension and selects the matching template
- Builds a Docker image, starts a container, and waits for the health check to pass
- Assigns the port and stores the deployment in `~/.faas/state.json`
- `--env` values support `${VAR_NAME}` substitution from the shell environment
- `--env` flags override values defined in the `[env]` section of `config.toml`

### Examples

```bash
# Basic deploy — auto-detect everything
faas up hello.py

# Explicit name and port
faas up hello.py --name my-func --port 3000

# Multiple environment variables
faas up hello.py -e DATABASE_URL=postgres://localhost -e API_KEY=secret

# Force rebuild and redeploy
faas up hello.py --force --no-cache

# Deploy from a directory with config.toml
faas up ./my-function/
```

### Error Scenarios

| Situation | Message | Resolution |
|-----------|---------|------------|
| Function already deployed | Suggests `faas down <name>` then redeploy, or use `--force` | Add `--force` to redeploy |
| Port conflict | Port already in use | Choose a different port or use `0` for auto-assign |
| Docker not running | "Cannot connect to Docker daemon — is Docker running?" | Start Docker |
| Health check fails | Deployment reports unhealthy | Run `faas logs <name>` to diagnose |
| Unknown file extension | Language not detected | Create `config.toml` with explicit `language` field |

---

## faas down

Stop and remove a running function.

```
faas down [name] [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | no | Name of the function to stop. Required unless `--all` is set. |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | `false` | Stop and remove all deployed functions. |
| `--keep-image` | bool | `false` | Keep the Docker image after removing the container. |

### Behavior

- Stops the container, removes the container, and removes the Docker image (unless `--keep-image`)
- Updates `~/.faas/state.json` to remove the function entry
- Requires either a function name argument or `--all`
- If the function name is not found, lists all running functions as suggestions

### Examples

```bash
# Stop one function
faas down my-func

# Stop but keep the image for faster redeploy
faas down my-func --keep-image

# Stop and remove everything
faas down --all
```

---

## faas ls

List deployed functions.

```
faas ls [flags]
```

**Alias:** `faas list`

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--json` | — | bool | `false` | Output as indented JSON. |
| `--quiet` | `-q` | bool | `false` | Print function names only, one per line. |

### Output Formats

**Default (table):**

```
NAME        LANGUAGE    PORT     STATUS     CREATED
hello       python      52341    healthy    2m ago
api         go          3000     healthy    1h ago
```

Status values are color-coded: green for `healthy`, red for `error`/`unhealthy`, dim for `stopped`.

**JSON (`--json`):**

```json
[
  {
    "name": "hello",
    "path": "/home/user/hello.py",
    "language": "python",
    "container_id": "abc123def456",
    "image_id": "sha256:...",
    "port": 52341,
    "status": "healthy",
    "created_at": "2026-03-08T12:34:56Z"
  }
]
```

**Quiet (`--quiet`):**

```
hello
api
```

### Behavior

- Checks Docker for actual container status on every invocation
- Updates `~/.faas/state.json` if a container was stopped externally
- Shows "No functions deployed" when the list is empty

### Examples

```bash
# Table view
faas ls

# JSON for scripting
faas ls --json

# Names only (useful for piping)
faas ls -q | xargs -I {} faas down {}
```

---

## faas logs

Stream function logs.

```
faas logs <name> [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | yes | Name of the deployed function. |

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--follow` | `-f` | bool | `true` | Follow log output in real time. |
| `--no-follow` | — | bool | `false` | Print historical logs and exit. Overrides `--follow`. |
| `--lines` | `-l` | int | `50` | Number of historical log lines to show. |
| `--json` | — | bool | `false` | Raw JSON output without formatting. |
| `--level` | — | string | — | Filter logs by level (`info`, `error`, `warn`, `debug`). |

### Behavior

- Streams structured JSON logs from the function's Docker container
- Follow mode (`--follow`) is enabled by default — logs stream until interrupted with Ctrl+C
- Use `--no-follow` to print the last N lines and exit
- Without `--json`, log lines are formatted for readability using the built-in formatter
- `--level` filters log entries by the `level` field in the structured JSON

### Examples

```bash
# Stream logs (default: follow + last 50 lines)
faas logs my-func

# Print last 100 lines and exit
faas logs my-func --no-follow --lines 100

# Stream only errors
faas logs my-func --level error

# Raw JSON for piping to jq
faas logs my-func --json | jq '.msg'

# Short form
faas logs my-func -f -l 20
```

---

## faas init

Generate a `config.toml` for a function file.

```
faas init <file>
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `file` | yes | Path to the function file. |

### Flags

None.

### Behavior

- Detects the language from the file extension
- Generates `config.toml` in the same directory as the function file
- Derives the function name from the filename (without extension)
- Fails if `config.toml` already exists in that directory
- Does not interact with Docker or the state store

### Generated Output

```toml
[function]
name = "hello"
language = "python"
entrypoint = "hello.py"

[runtime]
port = 0
health_path = "/health"

[build]
base_image = ""
runtime_image = ""

[env]

[dependencies]
packages = []
```

### Examples

```bash
# Generate config for a Python function
faas init hello.py
# Created config.toml in /home/user/hello/

# Then customize and deploy
vim config.toml
faas up hello.py
```

---

## Configuration Reference

The `config.toml` file controls all aspects of a function's build and deployment. It is auto-generated by `faas init` or on first `faas up` if missing.

### Sections

#### [function]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | filename | Function and container name. |
| `language` | string | auto-detected | Language identifier (`go`, `python`, `rust`, `php`, `typescript`, `javascript`). |
| `entrypoint` | string | — | Path to the function file relative to `config.toml`. |

#### [runtime]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | int | `0` | Host port. `0` = auto-assign from OS ephemeral range. |
| `health_path` | string | `"/health"` | HTTP path for health check polling. |

#### [build]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `base_image` | string | — | Override the default base Docker image from the template. |
| `runtime_image` | string | — | Override the runtime image (Go and Rust multi-stage builds only). |

#### [env]

Key-value pairs passed as environment variables to the container. Values support `${VAR_NAME}` substitution from the host environment.

```toml
[env]
DATABASE_URL = "postgres://localhost:5432/mydb"
API_KEY = "${SECRET_API_KEY}"
```

#### [dependencies]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `packages` | string[] | `[]` | External packages in `name@version` format. Version is optional. |

```toml
[dependencies]
packages = ["requests@2.31.0", "flask"]
```

The `@` separator is translated to each language's native version format during the Docker build:

| Language | Native Format | Example |
|----------|---------------|---------|
| Python | `requirements.txt` with `==` | `requests==2.31.0` |
| Go | `go.mod` + `go get` | `github.com/fatih/color@v1.18.0` |
| Rust | `Cargo.toml` with quoted version | `serde = "1.0"` |
| PHP | `composer.json` | `"guzzlehttp/guzzle": "^7.0"` |
| JavaScript | `package.json` | `"lodash": "4.17.21"` |
| TypeScript | `package.json` | `"@types/node": "22.0.0"` |

---

## State

FaaS stores deployment state in `~/.faas/state.json`. This file tracks all deployed functions and is updated automatically by `up`, `down`, and `ls` commands.

### Status Values

| Status | Meaning |
|--------|---------|
| `building` | Docker image is being built. |
| `starting` | Container started, waiting for health check. |
| `healthy` | Health check passed, function is serving. |
| `unhealthy` | Health check failed. |
| `stopped` | Container stopped (manually or externally). |
| `error` | Runtime error during build or deploy. |

### Paths

| Path | Purpose |
|------|---------|
| `~/.faas/state.json` | Deployed function state |
| `~/.faas/templates/` | User-defined custom templates |
| `~/.faas/logs/` | CLI debug logs (zerolog + rotation) |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DOCKER_HOST` | Docker daemon socket (inherited by Docker client). |

FaaS does not define its own environment variables. All configuration is via `config.toml` and CLI flags.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (build failure, Docker unavailable, invalid arguments, etc.) |
