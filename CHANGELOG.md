# Changelog

## 0.1.0 — 2026-08-21

Первая рабочая версия WailsPlugSystem.

Добавлены кроссплатформенный `.plugs` ZIP format, manifest validation, SHA-256 allowlist, directory loader, deterministic priority arbitration, declarative HTML/CSS/JS patches, sanitizer, transactional render snapshots, dependency validation, HTTP middleware for Wails asset handlers, `plugs` CLI, tests, race-compatible code path, GitHub Actions cross-compile matrix and security documentation.

API считается экспериментальным до выпуска `1.0.0`. Native dynamic libraries inside `.plugs`, arbitrary Go plugins, cryptographic signatures and isolated JavaScript sandbox are intentionally out of scope for this release.
