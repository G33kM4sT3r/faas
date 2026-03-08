# Python

## Runtime

| Property | Value |
|----------|-------|
| Base Image | `python:3.14-alpine3.23` |
| Runtime Image | — (single-stage) |
| Build | Single-stage (interpreted) |
| Internal Port | 8080 |
| Health Path | `/health` |

Python functions run on CPython 3.14 in an Alpine container. The function file is embedded inline into a generated HTTP server that uses Python's built-in `http.server` module — no framework dependencies.

## Handler Specification

```python
def handler(request: dict) -> dict
```

- **Function name**: Must be `handler` (lowercase)
- **Input**: `dict` — the parsed JSON request body. Empty requests receive an empty `{}` dict.
- **Output**: `dict` (or any JSON-serializable object) — serialized as the JSON response with `Content-Type: application/json`
- **Errors**: Uncaught exceptions return a `500` response with `{"error": "<message>"}` and log the error to stdout

### Embedding Strategy

Python uses **inline embedding**. Your entire function file is inserted directly into the generated `server.py` via the `{{.UserFunction}}` template variable. This works because Python allows imports and function definitions anywhere at module level.

Your file should:
- Define a `def handler(request)` function
- Import any standard library modules it needs at the top

### Constraints

- The handler runs synchronously — long-running operations block the server
- The server is single-threaded (`http.server.HTTPServer`)
- Imports at the top of your file are fine — they become module-level imports in the generated server

## Example

**hello.py**

```python
def handler(request):
    name = request.get("name", "world")
    return {"message": f"Hello, {name}!"}
```

**config.toml**

```toml
[function]
name = "hello-python"
language = "python"
entrypoint = "hello.py"

[runtime]
port = 0
health_path = "/health"

[dependencies]
packages = []
```

## Deploy

```bash
faas up hello.py
```

## Dependencies

Python dependencies are installed via `pip` from a generated `requirements.txt`. Only generated when packages are declared — no package manager overhead for zero-dependency functions.

```toml
[dependencies]
packages = ["requests@2.31.0", "flask"]
```

The `@` separator translates to `==` in `requirements.txt`. Omit the version for latest (`requests` becomes `requests` without a version pin).

## Invocation

```bash
curl -X POST http://localhost:<port> \
  -H "Content-Type: application/json" \
  -d '{"name": "Pythonista"}'
# {"message": "Hello, Pythonista!"}
```
