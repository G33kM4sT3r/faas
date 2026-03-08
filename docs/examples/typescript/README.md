# TypeScript

## Runtime

| Property | Value |
|----------|-------|
| Base Image | `oven/bun:1-alpine` |
| Runtime Image | — (single-stage) |
| Build | Single-stage (Bun runtime, native TypeScript support) |
| Internal Port | 8080 |
| Health Path | `/health` |

TypeScript functions run on Bun — a fast JavaScript/TypeScript runtime with native TypeScript support (no transpilation step). The generated server uses Bun's built-in `Bun.serve()` HTTP server.

## Handler Specification

```typescript
function handler(body: Record<string, any>): Record<string, any>
```

- **Function name**: Must be `handler` (lowercase)
- **Input**: `Record<string, any>` — the parsed JSON request body. Empty requests receive `{}`.
- **Output**: `Record<string, any>` (or any JSON-serializable object) — serialized via `Response.json()` with `Content-Type: application/json`
- **HTTP Methods**: POST to `/` invokes your handler. GET to `/health` returns `200 ok`. All other methods return `405 Method Not Allowed`.

### Embedding Strategy

TypeScript uses **inline embedding**. Your function is inserted directly into the generated `server.ts` via the `{{.UserFunction}}` template variable. This works because TypeScript/JavaScript allow function declarations anywhere at module level.

Your file should:
- Define a `function handler(body: Record<string, any>)` function
- Import any modules it needs (Bun built-ins, npm packages)

### Constraints

- Bun executes TypeScript natively — no `tsconfig.json` or build step needed
- The handler runs synchronously (return a value, not a Promise)
- Request processing includes timing — duration is logged in structured JSON to stdout

## Example

**handler.ts**

```typescript
function handler(body: Record<string, any>): Record<string, any> {
    const name = body.name || "world";
    return { message: `Hello, ${name}!` };
}
```

**config.toml**

```toml
[function]
name = "hello-typescript"
language = "typescript"
entrypoint = "handler.ts"

[runtime]
port = 0
health_path = "/health"

[dependencies]
packages = []
```

## Deploy

```bash
faas up handler.ts
```

## Dependencies

Dependencies are installed via `bun install` from a generated `package.json`. Only generated when packages are declared.

```toml
[dependencies]
packages = ["zod@3.23.0", "@types/node@22.0.0"]
```

The `@` separator works naturally with npm-style package names. Scoped packages (`@scope/name@version`) are handled correctly — the parser identifies the version separator after the scope prefix.

## Invocation

```bash
curl -X POST http://localhost:<port> \
  -H "Content-Type: application/json" \
  -d '{"name": "TypeScript Dev"}'
# {"message":"Hello, TypeScript Dev!"}
```
