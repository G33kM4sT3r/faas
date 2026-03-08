# PHP

## Runtime

| Property | Value |
|----------|-------|
| Base Image | `php:8.5-cli-alpine3.23` |
| Runtime Image | — (single-stage) |
| Build | Single-stage (interpreted), with optional Composer dependencies |
| Internal Port | 8080 |
| Health Path | `/health` |

PHP functions run on PHP 8.5 CLI in an Alpine container. The generated server uses `stream_socket_server` for a lightweight TCP-based HTTP server — no Apache, Nginx, or PHP-FPM.

### Pre-installed Extensions

The base image includes common extensions out of the box. The Dockerfile additionally installs: `intl`, `zip`, `bcmath`, `sockets`, `pcntl`, `pdo_mysql`, `pdo_pgsql`, and `gd` (with freetype and jpeg support).

Already built into `php:8.5-cli-alpine`: `mbstring`, `curl`, `dom`, `PDO`, `pdo_sqlite`, `opcache`, `xml`, `fileinfo`.

## Handler Specification

```php
function handler(array $input): array
```

- **Function name**: Must be `handler` (lowercase)
- **Input**: `array` — the parsed JSON request body as an associative array. Empty requests receive `[]`.
- **Output**: `array` — serialized as the JSON response via `json_encode` with `Content-Type: application/json`

### Embedding Strategy

PHP uses the **separate handler file** pattern. Your function file is written as `handler.php` and included via `require_once __DIR__ . '/handler.php'` in the generated `server.php`. This is required because PHP enforces `<?php` opening tags and `use` statements at the file level — inline embedding would create duplicate declarations.

Your file must:
- Start with `<?php`
- Define a `function handler(array $input): array`
- Use `use` or `require` statements for any dependencies

### Constraints

- The server is single-process, single-threaded (sequential request handling via `stream_socket_accept`)
- No framework is loaded unless you add one via dependencies
- When Composer dependencies are declared, `vendor/autoload.php` is auto-loaded before your handler

## Example

**handler.php**

```php
<?php

function handler(array $input): array
{
    $name = $input['name'] ?? 'world';
    return ['message' => "Hello, {$name}!"];
}
```

**config.toml**

```toml
[function]
name = "hello-php"
language = "php"
entrypoint = "handler.php"

[runtime]
port = 0
health_path = "/health"

[dependencies]
packages = []
```

## Deploy

```bash
faas up handler.php
```

## Dependencies

PHP dependencies are installed via Composer in a multi-stage build (Composer runs in a `composer:2` stage, `vendor/` is copied to the final image). Only generated when packages are declared.

```toml
[dependencies]
packages = ["guzzlehttp/guzzle@^7.0", "cocur/slugify@4.6"]
```

The `@` separator translates to Composer's version constraint format in `composer.json`. Use Composer-style constraints (`^7.0`, `~4.6`, `>=1.0`).

## Invocation

```bash
curl -X POST http://localhost:<port> \
  -H "Content-Type: application/json" \
  -d '{"name": "PHP Dev"}'
# {"message":"Hello, PHP Dev!"}
```
