# WailsPlugSystem — Wails 2 example

Это отдельное Wails 2 host-приложение, которое загружает `.plugs` из корневой папки `./plugins`.

```text
examples/wails2/
├── frontend/index.html
├── plugins/ip.example.plugs
├── plugin-src/ip-plugin/main.go
├── main.go
├── plugin_render_test.go
├── wails.json
└── go.mod
```

Host содержит только базовую страницу и generic middleware. IP-функциональность, HTML card, кнопка, CSS и JavaScript request находятся в независимом plugin source и его `.plugs` archive.

## Run

```bash
cd examples/wails2
go mod tidy
go run ./plugin-src/ip-plugin -output ./plugins/ip.example.plugs
wails dev
```

The repository includes the required `wails.json`. Wails CLI v2 reads this file from the current directory; without it, `wails dev` reports `open .../wails.json: The system cannot find the file specified`. The example is aligned with Wails CLI v2.13.0 and module `github.com/wailsapp/wails/v2 v2.13.0`.

On Ubuntu 24.04, the config enables Wails' `webkit2_41` build tag for WebKitGTK 4.1. Windows and macOS users do not need to change the configuration. Linux users who need the native build should install GTK3 and WebKitGTK 4.1 development packages.

Для production build:

```bash
wails build
```

В Wails 2 middleware подключается через `pkg/options/assetserver.Middleware`. Пакет `client` сам оборачивает `http.Handler`, поэтому plugin runtime не зависит от внутренней реализации Wails.

Плагин выполняет запрос к `https://api64.ipify.org?format=json`, показывает `Ваш IP: ...` и добавляет кнопку `Обновить IP`. Host не содержит IP-кода.

## Проверка

```bash
go test ./...
```

`plugin_render_test.go` загружает реальный `.plugs` из `./plugins` и проверяет injected card, button и JS asset.
