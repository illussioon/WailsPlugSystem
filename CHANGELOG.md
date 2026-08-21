# Changelog

## 0.3.0 — 2026-08-21

Added plugin SDK helpers `Console` and `ConsoleBrowser`, the runtime global `Wails.print.console` / `Wails.print.console.browser`, host-side `ConsoleMessage` and `HostLogger`, and the `/__wailsplugs/console` JSON endpoint. Added tests for browser-to-host logging, WebView console output, message escaping, and Wails 2/3 examples.

## 0.2.0 — 2026-08-21

Added the public `client` host facade and `plugin` authoring SDK. Added fluent plugin definitions, typed patch helpers, automatic asset hashing, SDK examples, English canonical documentation, Russian and Ukrainian localized guides with language switching, and pkg.go.dev-ready module instructions. Existing root-package APIs remain available for advanced integrations and backwards compatibility.

## 0.1.0 — 2026-08-21

Первая рабочая версия WailsPlugSystem.

Добавлены кроссплатформенный `.plugs` ZIP format, manifest validation, SHA-256 allowlist, directory loader, deterministic priority arbitration, declarative HTML/CSS/JS patches, sanitizer, transactional render snapshots, dependency validation, HTTP middleware for Wails asset handlers, `plugs` CLI, tests, race-compatible code path, GitHub Actions cross-compile matrix and security documentation.

API считается экспериментальным до выпуска `1.0.0`. Native dynamic libraries inside `.plugs`, arbitrary Go plugins, cryptographic signatures and isolated JavaScript sandbox are intentionally out of scope for this release.
