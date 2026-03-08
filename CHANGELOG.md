# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
