# Rust

## Runtime

| Property | Value |
|----------|-------|
| Base Image | `rust:1.94-alpine3.23` |
| Runtime Image | `alpine:3.23` |
| Build | Multi-stage (compiled binary, no Rust toolchain in final image) |
| Internal Port | 8080 |
| Health Path | `/health` |

Rust functions are compiled during the Docker build stage with `cargo build --release` and the resulting binary is copied to a minimal Alpine image. The final container has no Rust toolchain — just your statically linked binary.

## Handler Specification

```rust
fn handler(input: Value) -> Value
```

- **Function name**: Must be `handler` (lowercase)
- **Input**: `serde_json::Value` — the parsed JSON request body. Empty requests receive `Value::Object(Default::default())`.
- **Output**: `serde_json::Value` — serialized as the JSON response with `Content-Type: application/json`

### Embedding Strategy

Rust uses **inline embedding**. Your function is inserted directly into the generated `main.rs` via the `{{.UserFunction}}` template variable. The generated code imports `serde_json::{json, Value}` and provides the TCP server and `main()` function.

Your file should:
- Define a `fn handler(input: Value) -> Value` function
- Use `serde_json::json!` macro for constructing return values
- Add `use` statements for any additional crates

### Constraints

- No `fn main()` — the generated `main.rs` provides it
- `serde_json` is always available (auto-included in `Cargo.toml`)
- The handler runs synchronously on a raw TCP listener (no async runtime by default)
- Panics in the handler crash the container

## Example

**handler.rs**

```rust
use serde_json::{json, Value};

fn handler(input: Value) -> Value {
    let name = input["name"].as_str().unwrap_or("world");
    json!({"message": format!("Hello, {}!", name)})
}
```

**config.toml**

```toml
[function]
name = "hello-rust"
language = "rust"
entrypoint = "handler.rs"

[runtime]
port = 0
health_path = "/health"

[dependencies]
packages = []
```

## Deploy

```bash
faas up handler.rs
```

## Dependencies

A `Cargo.toml` is always generated with `serde_json = "1"` as a required dependency. Add additional crates in `config.toml`:

```toml
[dependencies]
packages = ["serde@1.0", "tokio@1.0"]
```

The `@` separator translates to a quoted version string in `Cargo.toml` (e.g., `tokio = "1.0"`). Omit the version for `"*"` (latest).

## Invocation

```bash
curl -X POST http://localhost:<port> \
  -H "Content-Type: application/json" \
  -d '{"name": "Rustacean"}'
# {"message":"Hello, Rustacean!"}
```
