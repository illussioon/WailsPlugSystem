# WailsPlugSystem SDK

[English](README.md) · [Русский](ru.md) · [Українська](uk.md)

WailsPlugSystem is a Go SDK for loading, composing, and authoring portable `.plugs` packages for Wails applications. The SDK is intentionally split into a small **host client** and a dedicated **plugin authoring API**.

> The recommended public surface is `github.com/illussioon/WailsPlugSystem/client` for application authors and `github.com/illussioon/WailsPlugSystem/plugin` for plugin authors.

## 1. Install the SDK

```bash
go get github.com/illussioon/WailsPlugSystem@v0.4.0
```

The module contains no Wails dependency in its core. It is ordinary Go code and can be cross-compiled for Linux, Windows, and macOS. Wails is connected through the standard `net/http` asset handler interface.

## 2. Host application API

Create a client with one loading strategy:

```go
plugins, err := client.New(client.Options{
    Directory:          "./plugins",
    Recursive:          false,
    StrictDependencies: true,
    AllowJavaScript:    false,
    AllowRootReplace:   false,
})
if err != nil {
    return err
}
if err := plugins.Reload(context.Background()); err != nil {
    return err
}
```

`client.New` selects a custom loader first, then a SHA-256 allowlist when `SHA256` is non-empty, and finally a directory loader. The host can reload the current snapshot without restarting the Wails process.

### SHA-256 allowlist

```go
plugins, err := client.New(client.Options{
    Directory: "./trusted-plugins",
    Recursive: true,
    SHA256: []string{
        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    },
    StrictDependencies: true,
})
```

The hash verifies the exact archive bytes. It does not authenticate the publisher, so keep the allowlist in a trusted release channel.

### Rendering and diagnostics

```go
result, err := plugins.Render(htmlSource)
if err != nil {
    return err
}
fmt.Println(result.HTML)
for _, decision := range result.Decisions {
    log.Printf("plugin=%s patch=%s applied=%t reason=%s",
        decision.PluginID, decision.PatchID, decision.Applied, decision.Reason)
}
```

The `RenderResult` contains transformed HTML, active plugin IDs, and conflict decisions. Rendering starts from the host-provided source every time; removing or replacing a plugin therefore reverts its previous effects instead of accumulating mutations.

### Wails asset handler

Wails v3 accepts an `http.Handler` in `application.AssetOptions.Handler`:

```go
//go:embed frontend/*
var frontend embed.FS

base := application.AssetFileServerFS(frontend)
handler := plugins.Handler(base)

app := application.New(application.Options{
    Assets: application.AssetOptions{Handler: handler},
})
```

The middleware transforms HTML responses and passes CSS, JavaScript files, images, API responses, status codes, and non-HTML content through unchanged. The same handler pattern works with Wails v2 asset servers and custom frontend servers.

For a custom server:

```go
mux := http.NewServeMux()
mux.Handle("/", plugins.Handler(frontendHandler))
```

### Advanced API

The facade exposes the underlying runtime when an application needs custom reconciliation or inspection:

```go
manager := plugins.Manager()
active := plugins.Packages()
_ = manager
_ = active
```

Most applications should stay on the `client` facade. The root package remains available for lower-level loaders and custom orchestration.

## 3. Plugin author API

The `plugin` package replaces manual JSON authoring for most plugins:

```go
package main

import "github.com/illussioon/WailsPlugSystem/plugin"

func main() {
    definition := plugin.New("acme.dashboard", "Acme Dashboard", "1.0.0").
        Priority(100).
        HTML().
        SetText("dashboard-title", "#app-title", "Dashboard").
        SetAttr("dashboard-root", "#app", "data-plugin", "acme-dashboard").
        AppendHTML("toolbar", "body", `<div class="plugin-toolbar">Tools</div>`)

    if _, err := definition.Build("./dist/acme.dashboard.plugs"); err != nil {
        panic(err)
    }
}
```

### Fluent helpers

| Helper | Effect |
| --- | --- |
| `Priority(value)` | Sets conflict priority; larger values win. |
| `HTML()` | Grants HTML patch permission. |
| `CSSPermission()` | Grants CSS asset permission. |
| `JavaScript()` | Grants JS asset permission; host policy is still required. |
| `ReplaceRoot()` | Grants the `replace_root` permission and HTML permission. |
| `DependsOn(id, version)` | Declares a required exact dependency version. |
| `SetText` | Replaces text content. |
| `SetAttr` | Sets an attribute. |
| `Remove` | Removes matched nodes. |
| `ReplaceHTML`, `AppendHTML`, `PrependHTML` | Applies sanitized HTML fragments. |
| `AddClass`, `RemoveClass` | Changes CSS classes. |
| `AddCSS`, `AddJS` | Adds an asset and an injection patch. |
| `Asset` | Adds an arbitrary allowlisted asset. |
| `Build` | Writes a temporary source tree and creates a `.plugs` archive. |
| `WriteSource` | Writes `manifest.json`, `patches.json`, and `assets/` for inspection or custom packaging. |

### Console logging

A plugin can write to the host/Wails console without manually creating a fetch call:

```go
definition := plugin.New("acme.debug", "Acme Debug", "1.0.0").
    JavaScript().
    Console("startup", "Hello World").
    ConsoleBrowser("browser-startup", "Hello World in WebView2/browser console")
```

The helpers generate these runtime calls:

```javascript
Wails.print.console("Hello World");
Wails.print.console.browser("Hello World in WebView2/browser console");
```

`Wails.print.console` sends a small JSON message to the host through the SDK endpoint and the host forwards it to the configured `client.Options.HostLogger` callback. If no callback is supplied, the standard Go logger is used. `Wails.print.console.browser` writes directly to the native WebView/browser developer console. Both methods require the host's `AllowJavaScript: true` policy because they execute in the WebView.

### Plugin lifecycle logging

A plugin can declare lifecycle messages in its `.plugs` manifest:

```go
definition := plugin.New("acme.plugin", "Acme Plugin", "1.0.0").
    OnLoad("Acme Plugin loaded").
    OnUnload("Acme Plugin unloaded")
```

When the plugin becomes active, the runtime generates:

```javascript
Wails.plugin.print.load("Acme Plugin loaded");
```

When the plugin is removed or replaced during `Reload`, it generates:

```javascript
Wails.plugin.print.unload("Acme Plugin unloaded");
```

The runtime sends lifecycle messages to the host logger immediately and queues the corresponding browser-console call for the next HTML render. Each transition is emitted once: the initial load, a replacement unload/load pair, or a final unload. With `AllowJavaScript: false`, host logging remains available but browser-side execution is intentionally disabled.

Patch helpers accept options:

```go
definition.SetText(
    "primary-title",
    "#app-title",
    "Preferred title",
    plugin.WithConflictKey("app-title"),
)

definition.AddCSS(
    "theme",
    "theme.css",
    cssBytes,
    plugin.WithConflictKey("theme-css"),
    plugin.Optional(),
)
```

A shared `conflict_key` makes patches compete for the same resource. `Optional()` makes an unmatched selector or missing asset non-fatal.

## 4. `.plugs` package format

A built package contains:

```text
plugin.plugs
├── manifest.json
├── patches.json
└── assets/
    ├── theme.css
    └── app.js
```

The builder computes SHA-256 hashes for every file under `assets/` and writes them to `manifest.files`. The runtime rejects undeclared files, symlinks, absolute paths, traversal paths, oversized archives, and invalid manifests.

The package is a portable data artifact, not a native executable. It does not load `.dll`, `.so`, `.dylib`, ELF, or arbitrary Go plugins. This is the reason one artifact can be used across desktop operating systems.

## 5. Priorities and composition

Plugins are ordered by descending `manifest.priority` and then by ascending plugin ID. For a shared `conflict_key`, the first patch wins and later patches are recorded as rejected in `RenderResult.Decisions`. Different conflict keys compose together.

This produces a deterministic result regardless of filesystem ordering. A plugin can declare dependencies with `DependsOn`; with `StrictDependencies: true`, a missing or version-mismatched dependency rejects the reload before the active snapshot changes.

## 6. HTML, CSS, and JavaScript security

HTML fragments are sanitized before insertion. The sanitizer removes script-like elements, event-handler attributes, and dangerous URL schemes. A JavaScript asset requires both plugin `JavaScript()` permission and host `AllowJavaScript: true`.

JavaScript is not a sandbox. When enabled, it runs with the host WebView's normal JavaScript privileges. Enable it only for trusted plugin publishers. For untrusted packages, use a SHA-256 allowlist, keep JavaScript disabled, and avoid `ReplaceRoot()`.

## 7. CLI

Manual source directories can still be packaged with the CLI:

```bash
go install github.com/illussioon/WailsPlugSystem/cmd/plugs@v0.4.0
plugs pack --input ./my-plugin --output ./dist/my-plugin.plugs
plugs verify --file ./dist/my-plugin.plugs
plugs hash --file ./dist/my-plugin.plugs
```

The fluent `plugin.Definition` API is preferred for Go-authored plugins because it validates the same manifest and asset rules while keeping the authoring code type-safe.

## 8. Complete Wails IP demo

The repository includes two version-specific examples:

| Example | Wails API | Run command |
| --- | --- | --- |
| [`examples/wails2`](../examples/wails2) | Wails 2 `assetserver.Options.Middleware` | `cd examples/wails2 && wails dev` |
| [`examples/wails3`](../examples/wails3) | Wails 3 `application.AssetOptions.Handler` | `cd examples/wails3 && wails3 dev` |

Both hosts load `.plugs` from their own `./plugins` directory. Neither host contains IP implementation. The shared demo plugin adds the IP card, refresh button, CSS, and JavaScript request to `https://api64.ipify.org?format=json`.

Run Wails 3:

```bash
cd examples/wails3
go mod tidy
go run ./plugin-src/ip-plugin -output ./plugins/ip.example.plugs
wails3 dev
```

Run Wails 2:

```bash
cd examples/wails2
go mod tidy
go run ./plugin-src/ip-plugin -output ./plugins/ip.example.plugs
wails dev
```

## 9. Versioning and publishing

Use semantic version tags:

```bash
git tag v0.2.0
git push origin v0.2.0
go get github.com/illussioon/WailsPlugSystem@v0.4.0
```

After a public tag is pushed, Go tooling can resolve the module and `pkg.go.dev` can index its documentation. The module path is:

```text
github.com/illussioon/WailsPlugSystem
```

## References

[1]: https://github.com/wailsapp/wails "Wails official repository"
[2]: https://github.com/cordiverse/cordis "Cordis plugin/composability framework"
[3]: https://pkg.go.dev/ "Go package documentation service"
