# Changelog

## 0.7.0 — 2026-08-22

Added opt-in AES-256-GCM encrypted `.plugs` payloads. Patches and assets are stored in authenticated `payload.bin`, while the manifest remains available for routing and dependency metadata. Added SDK `Encrypt`, host `DecryptionKey` and `DecryptionKeyProvider`, CLI `--encrypt-key-file`/`--key-file`, encrypted package validation, key-provider loading, and anti-dump threat-model documentation. Plaintext packages remain backward compatible.

## 0.6.0 — 2026-08-22

Added React/Vue Vite plugin templates, CLI `plugs init`, external CSS/JavaScript injection, plugin static asset routes for code splitting and dynamic imports, `AssetsDirAs`, explicit `HostCSS` permission, and cross-platform polling hot reload through `client.Watch` and `plugs watch`. Added asset route, code-splitting, template scaffolding, watcher, and host CSS metadata coverage.

## 0.5.0 — 2026-08-21

Added file-based plugin authoring helpers `AppendHTMLFile`, `ReplaceHTMLFile`, `CSSFile`, `JSFile`, `AssetFile`, and recursive `AssetsDir`. The SDK now reads source files at build time, validates regular files and archive paths, rejects symlinks and files larger than 8 MiB, and packages copied bytes with the existing SHA-256 allowlist.

## 0.4.0 — 2026-08-21

Added plugin lifecycle manifest fields and fluent SDK methods `OnLoad` and `OnUnload`. The runtime now emits `Wails.plugin.print.load(text)` and `Wails.plugin.print.unload(text)` exactly once per activation, replacement, or removal transition, forwarding messages to the host logger and the WebView/browser console when JavaScript is enabled. Added lifecycle ordering, replacement, unload, one-shot injection, manifest limits, and Wails 2/3 example coverage.

## 0.3.0 — 2026-08-21

Added plugin SDK helpers `Console` and `ConsoleBrowser`, the runtime global `Wails.print.console` / `Wails.print.console.browser`, host-side `ConsoleMessage` and `HostLogger`, and the `/__wailsplugs/console` JSON endpoint. Added tests for browser-to-host logging, WebView console output, message escaping, and Wails 2/3 examples.

## 0.2.0 — 2026-08-21

Added the public `client` host facade and `plugin` authoring SDK. Added fluent plugin definitions, typed patch helpers, automatic asset hashing, SDK examples, English canonical documentation, Russian and Ukrainian localized guides with language switching, and pkg.go.dev-ready module instructions. Existing root-package APIs remain available for advanced integrations and backwards compatibility.

## 0.1.0 — 2026-08-21

Первая рабочая версия WailsPlugSystem.

Добавлены кроссплатформенный `.plugs` ZIP format, manifest validation, SHA-256 allowlist, directory loader, deterministic priority arbitration, declarative HTML/CSS/JS patches, sanitizer, transactional render snapshots, dependency validation, HTTP middleware for Wails asset handlers, `plugs` CLI, tests, race-compatible code path, GitHub Actions cross-compile matrix and security documentation.

API считается экспериментальным до выпуска `1.0.0`. Native dynamic libraries inside `.plugs`, arbitrary Go plugins, cryptographic signatures and isolated JavaScript sandbox are intentionally out of scope for this release.
