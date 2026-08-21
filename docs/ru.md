# WailsPlugSystem SDK — русская документация

[English — основная](README.md) · **Русский** · [Українська](uk.md)

**WailsPlugSystem** — Go SDK для загрузки, композиции и разработки переносимых плагинов `.plugs` для Wails-приложений. SDK разделён на два удобных публичных пакета: `client` для host-приложения и `plugin` для разработчика плагина.

## Установка

```bash
go get github.com/illussioon/WailsPlugSystem@v0.3.0
```

Ядро не импортирует Wails и использует обычный `net/http`, поэтому пакет собирается для Linux, Windows и macOS.

## Client для приложения

```go
plugins, err := client.New(client.Options{
    Directory:          "./plugins",
    Recursive:          false,
    StrictDependencies: true,
    AllowJavaScript:    false,
    AllowRootReplace:   false,
})
if err != nil { return err }
if err := plugins.Reload(context.Background()); err != nil { return err }
```

`client.New` выбирает loader в следующем порядке: собственный `Loader`, затем SHA-256 allowlist, затем `Directory`. После `Reload` можно вызвать `Render`, `Packages` или подключить middleware через `Handler`.

### Загрузка по SHA-256

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

Хеш проверяет точные байты архива, но не подтверждает личность автора. Allowlist следует хранить в доверенном release-канале.

### Подключение к Wails

Для Wails v3:

```go
//go:embed frontend/*
var frontend embed.FS

base := application.AssetFileServerFS(frontend)
handler := plugins.Handler(base)
app := application.New(application.Options{
    Assets: application.AssetOptions{Handler: handler},
})
```

Middleware изменяет только HTML-ответы. CSS, JS-файлы, изображения, API-ответы и status codes проходят без изменений. Тот же `http.Handler` можно использовать в Wails v2.

### Render и диагностика

```go
result, err := plugins.Render(htmlSource)
if err != nil { return err }
fmt.Println(result.HTML)
for _, decision := range result.Decisions {
    log.Printf("plugin=%s patch=%s applied=%t reason=%s",
        decision.PluginID, decision.PatchID, decision.Applied, decision.Reason)
}
```

Каждый render начинается с исходного HTML, поэтому удаление или перезагрузка плагина автоматически откатывает его эффекты.

## Plugin SDK для разработчика

```go
package main

import "github.com/illussioon/WailsPlugSystem/plugin"

func main() {
    _, err := plugin.New("acme.dashboard", "Acme Dashboard", "1.0.0").
        Priority(100).
        HTML().
        SetText("title", "#app-title", "Dashboard").
        SetAttr("root", "#app", "data-plugin", "acme-dashboard").
        AddCSS("theme", "theme.css", []byte("#app-title { color: #2563eb; }"),
            plugin.WithConflictKey("app-title-style")).
        Build("./dist/acme.dashboard.plugs")
    if err != nil { panic(err) }
}
```

Основные методы: `Priority`, `HTML`, `CSSPermission`, `JavaScript`, `ReplaceRoot`, `DependsOn`, `SetText`, `SetAttr`, `Remove`, `ReplaceHTML`, `AppendHTML`, `PrependHTML`, `AddClass`, `RemoveClass`, `AddCSS`, `AddJS`, `Asset`, `WriteSource` и `Build`.

`WithConflictKey` задаёт общий ресурс, которым конкурируют плагины. `Optional()` делает отсутствие selector или asset нефатальным.

## Приоритеты и зависимости

Плагины сортируются по `priority` по убыванию, затем по ID. Если два патча имеют одинаковый `conflict_key`, побеждает первый; решение записывается в `RenderResult.Decisions`. Разные ключи объединяются. `DependsOn` задаёт зависимость, а `StrictDependencies: true` отклоняет reload при отсутствующем или несовместимом plugin version.

## Формат `.plugs`

```text
plugin.plugs
├── manifest.json
├── patches.json
└── assets/
    ├── theme.css
    └── app.js
```

Builder автоматически записывает SHA-256 каждого asset. Runtime отклоняет незаявленные файлы, symlink, traversal paths, абсолютные пути, повреждённые manifest и слишком большие архивы.

`.plugs` — это переносимый data package, а не native executable. Внутри не загружаются `.dll`, `.so`, `.dylib` или произвольные Go plugins.

## Безопасность

HTML fragments проходят sanitizer: удаляются script-like элементы, event-handler attributes и опасные URL schemes. JavaScript требует одновременно `plugin.JavaScript()` и `client.Options{AllowJavaScript: true}`. JavaScript не sandboxed и выполняется с обычными правами WebView, поэтому разрешайте его только доверенным поставщикам. Для недоверенных плагинов используйте SHA-256 allowlist, отключённый JavaScript и не включайте `ReplaceRoot`.

## Console logging

Плагин может писать в консоль host/Wails и в консоль WebView/browser:

```go
definition := plugin.New("acme.debug", "Acme Debug", "1.0.0").
    JavaScript().
    Console("startup", "Hello World").
    ConsoleBrowser("browser-startup", "Hello World в консоли WebView2/browser")
```

Эти helpers генерируют:

```javascript
Wails.print.console("Hello World");
Wails.print.console.browser("Hello World в консоли WebView2/browser");
```

`Wails.print.console` отправляет JSON-сообщение host-приложению. Его можно получить через `client.Options.HostLogger`; если callback не задан, используется стандартный Go logger. `Wails.print.console.browser` пишет непосредственно в developer console нативного WebView/browser. Для обоих методов host должен включить `AllowJavaScript: true`.

## CLI и публикация

```bash
go install github.com/illussioon/WailsPlugSystem/cmd/plugs@v0.3.0
plugs pack --input ./my-plugin --output ./dist/my-plugin.plugs
plugs verify --file ./dist/my-plugin.plugs
plugs hash --file ./dist/my-plugin.plugs
```

После публичного semver tag пакет подключается стандартно:

```bash
go get github.com/illussioon/WailsPlugSystem@v0.3.0
```

После публикации тега Go tooling и `pkg.go.dev` смогут индексировать модуль и его GoDoc.
