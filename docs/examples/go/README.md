# Go

## Runtime

| Property | Value |
|----------|-------|
| Base Image | `golang:1.26-alpine3.23` |
| Runtime Image | `alpine:3.23` |
| Build | Multi-stage (compiled binary, no Go toolchain in final image) |
| Internal Port | 8080 |
| Health Path | `/health` |

Go functions are compiled during the Docker build stage and the resulting static binary is copied to a minimal Alpine image. The final container has no Go runtime — just your binary.

## Handler Specification

```go
func Handler(req map[string]any) map[string]any
```

- **Function name**: Must be `Handler` (exported, PascalCase)
- **Package**: Must be `package main`
- **Input**: `map[string]any` — the parsed JSON request body. Empty requests receive an initialized empty map.
- **Output**: `map[string]any` — serialized as the JSON response with `Content-Type: application/json`

### Embedding Strategy

Go uses the **separate handler file** pattern. Your function file is written as `handler.go` alongside a generated `main.go` that contains the HTTP server. This is required because Go enforces `package` declarations and `import` blocks at the file level — inline embedding would create duplicate `package main` declarations.

Your file must:
- Declare `package main`
- Define an exported `func Handler(req map[string]any) map[string]any`
- Import any packages it needs (standard library imports are fine)

### Constraints

- No `func main()` — the generated `main.go` provides it
- The handler runs synchronously on each request
- Panics in the handler crash the container — recover in your code if needed

## Example

**handler.go**

```go
package main

func Handler(req map[string]any) map[string]any {
	name, _ := req["name"].(string)
	if name == "" {
		name = "world"
	}
	return map[string]any{"message": "Hello, " + name + "!"}
}
```

**config.toml**

```toml
[function]
name = "hello-go"
language = "go"
entrypoint = "handler.go"

[runtime]
port = 0
health_path = "/health"

[dependencies]
packages = []
```

## Deploy

```bash
faas up handler.go
```

## Dependencies

Go modules are always generated (required for compilation). Add external packages in `config.toml`:

```toml
[dependencies]
packages = ["github.com/fatih/color@v1.18.0"]
```

The builder generates a `go.mod` with module name `faas-func` and runs `go get` for each listed package. Use full module paths with Go-style version prefixes (`v1.2.3`).

## Invocation

```bash
curl -X POST http://localhost:<port> \
  -H "Content-Type: application/json" \
  -d '{"name": "Gopher"}'
# {"message":"Hello, Gopher!"}
```
