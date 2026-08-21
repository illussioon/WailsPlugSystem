# WailsPlugSystem

**WailsPlugSystem** is a cross-platform Go SDK for building declarative plugins for Wails applications. It provides two simple public packages:

| Package | Purpose |
| --- | --- |
| [`client`](./client) | Host-application API: load, reload, render, and attach plugins to a Wails asset handler. |
| [`plugin`](./plugin) | Plugin-author API: define patches and assets with a fluent builder and produce a `.plugs` archive. |

The low-level root package remains available for advanced integrations and backwards compatibility.

## Documentation

**[English — canonical SDK guide](./docs/README.md)** · [Русский](./docs/ru.md) · [Українська](./docs/uk.md)

## Install

```bash
go get github.com/illussioon/WailsPlugSystem@v0.3.0
```

## Minimal host application

```go
package main

import (
    "context"
    "net/http"

    "github.com/illussioon/WailsPlugSystem/client"
)

func main() {
    plugins, err := client.New(client.Options{
        Directory:          "./plugins",
        StrictDependencies: true,
    })
    if err != nil { panic(err) }
    if err := plugins.Reload(context.Background()); err != nil { panic(err) }

    // In Wails v3, use this as application.AssetOptions.Handler.
    _ = plugins.Handler(http.FileServer(http.Dir("./frontend")))
}
```

## Minimal plugin

```go
package main

import "github.com/illussioon/WailsPlugSystem/plugin"

func main() {
    _, err := plugin.New("acme.welcome", "Acme Welcome", "1.0.0").
        Priority(100).
        HTML().
        SetText("title", "#app-title", "Hello from a plugin").
        AddCSS("theme", "theme.css", []byte("#app-title { color: #2563eb; }"), plugin.WithConflictKey("app-title-style")).
        Build("./dist/acme.welcome.plugs")
    if err != nil { panic(err) }
}
```

The `.plugs` format is a deterministic ZIP container with a manifest, declarative patches, and hashed assets. JavaScript is opt-in at both levels: the plugin requests the `js` permission and the host enables `AllowJavaScript: true`.

## Development

```bash
go test ./...
go test -race ./...
go build ./...
```

See the [English SDK guide](./docs/README.md) for Wails integration, SHA-256 allowlists, conflict priorities, security model, CLI usage, and publishing instructions.

## Wails IP demo

A complete Wails v3 host and independent IP plugin are available in [`examples/wails3`](./examples/wails3). The host contains no IP logic: it loads `./plugins/ip.example.plugs`, while the plugin adds the `Ваш IP: ...` card, refresh button, CSS, and the `api64.ipify.org` request.

## License

MIT License. The project is inspired by the composability ideas documented by [Cordis](https://github.com/cordiverse/cordis) and [the Cordis paper](https://github.com/cordiverse/paper), but the Go SDK is an independent implementation.
