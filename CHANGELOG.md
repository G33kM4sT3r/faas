# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- CLI commands: `up`, `down`, `ls`, `logs`, `init`
- Language support: Go, Python, Rust, PHP, TypeScript, JavaScript (Bun)
- Docker container runtime with pluggable interface
- Auto-generated `config.toml` per function
- Auto-assigned ports via OS ephemeral range
- Explicit port assignment with conflict detection
- Custom template override via `~/.faas/templates/`
- Structured JSON logging with level filtering
- Health check polling before reporting readiness
- TUI spinner and styled output (bubbletea + lipgloss)
- E2E tests for all supported languages
- CI workflow with lint, vet, race-detected tests, coverage threshold, and security checks
- Release workflow with cross-compiled binaries for Linux and macOS (amd64/arm64)
