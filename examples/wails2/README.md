# WailsPlugSystem — Wails 2 example

Это отдельное Wails 2 host-приложение, которое загружает `.plugs` из корневой папки `./plugins`.

```text
examples/wails2/
├── frontend/index.html
├── plugins/ip.example.plugs
├── plugin-src/ip-plugin/main.go
├── main.go
├── plugin_render_test.go
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
