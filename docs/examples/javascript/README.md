# JavaScript

## Runtime

| Property | Value |
|----------|-------|
| Base Image | `oven/bun:1-alpine` |
| Runtime Image | — (single-stage) |
| Build | Single-stage (Bun runtime) |
| Internal Port | 8080 |
| Health Path | `/health` |

JavaScript functions run on Bun — a fast JavaScript runtime. The generated server uses Bun's built-in `Bun.serve()` HTTP server. Note: this is **not** Node.js — Bun provides its own standard library and runtime APIs.

## Handler Specification

```javascript
function handler(body) { return { ... } }
```

- **Function name**: Must be `handler` (lowercase)
- **Input**: `object` — the parsed JSON request body. Empty requests receive `{}`.
- **Output**: `object` (any JSON-serializable value) — serialized via `Response.json()` with `Content-Type: application/json`
- **HTTP Methods**: POST to `/` invokes your handler. GET to `/health` returns `200 ok`. All other methods return `405 Method Not Allowed`.

### Embedding Strategy

JavaScript uses **inline embedding**. Your function is inserted directly into the generated `server.js` via the `{{.UserFunction}}` template variable. This works because JavaScript allows function declarations anywhere at module level.

Your file should:
- Define a `function handler(body)` function
- Import any modules it needs (Bun built-ins, npm packages)

### Constraints

- Runs on Bun, not Node.js — most Node.js APIs are compatible but not all
- The handler runs synchronously (return a value, not a Promise)
- Request processing includes timing — duration is logged in structured JSON to stdout

## Example

**handler.js**

```javascript
function handler(body) {
    const name = body.name || "world";
    return { message: `Hello, ${name}!` };
}
```

**config.toml**

```toml
[function]
name = "hello-javascript"
language = "javascript"
entrypoint = "handler.js"

[runtime]
port = 0
health_path = "/health"

[dependencies]
packages = []
```

## Deploy

```bash
faas up handler.js
```

## Dependencies

Dependencies are installed via `bun install` from a generated `package.json`. Only generated when packages are declared.

```toml
[dependencies]
packages = ["lodash@4.17.21", "dayjs@1.11.0"]
```

The `@` separator works naturally with npm-style package names. Scoped packages (`@scope/name@version`) are handled correctly.

## Invocation

```bash
curl -X POST http://localhost:<port> \
  -H "Content-Type: application/json" \
  -d '{"name": "JS Dev"}'
# {"message":"Hello, JS Dev!"}
```
