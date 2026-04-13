# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2026-04-13

Three new commands and a sweep of polish around redeploys, error messages, and dependencies.

### Added

- `faas invoke <name>` — call a deployed function directly from the CLI without looking up its port. Send a JSON body inline with `-d '{...}'` or from a file with `-d @payload.json`, customize the method with `-X`, target a specific path with `--path`, add headers with `-H`. Responses come back pretty-printed; very large responses are truncated so a runaway function can't flood your terminal.
- `faas dev <path>` — work-in-progress mode. The function runs as usual, then redeploys automatically when you save the file. Accepts the same `--name`, `--port`, `--env`, `--force`, `--no-cache` flags as `faas up`.
- `faas completion bash|zsh|fish|powershell` — generate a completion script for your shell.
- Custom fields you log from your function (`user_id`, `trace_id`, anything you want) now show up in `faas logs` instead of being dropped.
- `faas up` warns when your `config.toml` references an environment variable (`${VAR}`) that isn't set in your shell, instead of quietly passing an empty value to the container.

### Changed

- A failed deploy no longer leaves a broken container running on your port. If the health check doesn't pass, the container is removed and nothing is recorded for the function.
- `faas up --force` now stops cleanly with a clear error if it can't tear down the previous version, instead of moving on and producing a confusing "name already in use" error from the container runtime.
- `faas down --all` keeps going past one broken function and lists everything that failed at the end, rather than stopping at the first error.
- `faas down`, `faas up --force`: if removing a container fails, the function stays in `faas ls` so you can retry — your state isn't silently dropped on a partial failure.
- `faas down --all` rejects an extra function name argument instead of silently ignoring it.
- `faas up --env KEY=VALUE` rejects malformed entries (missing `=`, empty key) so typos don't slip through unnoticed.
- `faas up` validates `config.toml` up front and points at the missing or invalid field instead of failing midway through a Docker build.
- Error messages across all commands have a consistent look with hint lines suggesting the next step.
- Files in `~/.faas/` and generated `config.toml` files are written with owner-only permissions.

### Fixed

- `faas up` no longer wrongly reports an explicit `--port` as "already in use" when the probe is interrupted or times out — only an actual listener counts.
- `dependencies.packages` entries with constraint specs containing `^`, `~`, `|`, or spaces (PHP, JS/TS) now produce a valid `composer.json` / `package.json` and build correctly.

## [1.0.0] - 2026-03-08

Initial release — deploy functions as containerized HTTP services with a single command.

### Added

- Deploy functions in Go, Python, Rust, PHP, TypeScript, and JavaScript
- Five CLI commands: `up`, `down`, `ls`, `logs`, `init`
- Auto-detect language from file extension — no config needed to get started
- Auto-generated `config.toml` with optional port, environment variables, and dependencies
- External package support for all languages via `package@version` syntax in config
- Auto-assigned ports with optional explicit port binding
- Health check polling — functions are ready when `up` returns
- Structured JSON log streaming with level filtering
- Custom template system — override built-in templates or add new languages via `~/.faas/templates/`
- JSON and quiet output modes for scripting (`ls --json`, `ls --quiet`, `logs --json`)
- Force redeploy (`up --force`) and cache-busting rebuilds (`up --no-cache`)
- Bulk teardown (`down --all`)
- Cross-platform release binaries for Linux and macOS (amd64/arm64)
