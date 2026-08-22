# WailsPlugSystem SDK — русская документация

[English — основная](README.md) · **Русский** · [Українська](uk.md)

**WailsPlugSystem** — Go SDK для загрузки, композиции и разработки переносимых плагинов `.plugs` для Wails-приложений. SDK разделён на два удобных публичных пакета: `client` для host-приложения и `plugin` для разработчика плагина.

## Установка

```bash
go get github.com/illussioon/WailsPlugSystem@v0.6.0
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

## React и Vue templates

CLI содержит готовые шаблоны:

```bash
plugs init --template react-vite --output ./my-react-plugin
plugs init --template vue-vite --output ./my-vue-plugin
```

Шаблон содержит Vite-проект, Go entrypoint плагина, mount HTML, stylesheet с наследованием host CSS и deterministic entry/chunk layout. Запуск сборки: `npm install`, `npm run build`, затем `go run ./plugin`. Для Vite chunks используются `JSFileExternal` и `AssetsDirAs`.

## Code splitting и static assets

External injection обслуживает файлы плагина через:

```text
/__wailsplugs/assets/{plugin-id}/{asset-path}
```

Пример для Vite entrypoint и chunks:

```go
definition := plugin.New("acme.ui", "Acme UI", "1.0.0").
    HostCSS().
    CSSFileExternal("css", "assets/ui.css", "dist/ui.css").
    JSFileExternal("entry", "assets/ui.js", "dist/ui.js").
    AssetsDirAs("dist/chunks", "assets/chunks")
```

Runtime создаёт `<link>` и `<script type="module" src>` вместо inline-кода. Поэтому относительные imports, lazy chunks, картинки и fonts могут загружаться через тот же plugin route. External mode разрешён только для CSS/JS injection и сохраняет обычные permission checks.

## Наследование CSS приложения

Плагин монтируется в общий host document, а не в iframe. Явно укажите это через `HostCSS()`:

```go
definition := plugin.New("acme.panel", "Acme Panel", "1.0.0").
    HostCSS().
    AppendHTMLFile("mount", "body", "ui/mount.html").
    CSSFileExternal("styles", "assets/panel.css", "ui/panel.css")
```

Плагин наследует `font`, `color`, CSS custom properties, reset rules и typography host-приложения. Не используйте iframe или shadow root, если нужно наследование. `HostCSS` — это metadata и не даёт дополнительных host API permissions.

## Hot reload в development

Host может отслеживать `.plugs` и автоматически вызывать `Reload`:

```go
err := app.Watch(ctx, client.WatchOptions{
    Interval: 300 * time.Millisecond,
    OnReload: func(ctx context.Context, change devwatch.Change) error {
        // Здесь вызывается reload окна конкретной версии Wails.
        return nil
    },
})
```

Для пересборки source-layout plugin при изменениях используйте:

```bash
plugs watch --input . --output ./plugins/acme.ui.plugs
```

Watcher использует polling, поэтому одинаково работает в Linux, Windows и macOS. Это development-инструмент; в production используйте explicit reload и verification.

## File-based authoring

Для больших интерфейсов не обязательно вставлять HTML/CSS/JS строками. SDK прочитает обычные файлы проекта, проверит их, добавит в allowlist assets, рассчитает SHA-256 при упаковке и создаст нужные patches:

```go
definition := plugin.New("acme.react-ui", "Acme React UI", "1.0.0").
    AppendHTMLFile("mount", "body", "ui/mount.html").
    CSSFile("styles", "styles/ui.css", "ui/styles.css").
    JSFile("bundle", "scripts/ui.js", "dist/ui.bundle.js").
    AssetFile("assets/icon.svg", "ui/icon.svg").
    AssetsDir("ui/static")

_, err := definition.Build("dist/acme.react-ui.plugs")
```

`AppendHTMLFile` и `ReplaceHTMLFile` используют содержимое файла как HTML fragment. `CSSFile` и `JSFile` сохраняют существующие permission checks. `AssetFile` добавляет один произвольный файл, а `AssetsDir` рекурсивно добавляет regular files в `assets/`. Symlinks, небезопасные archive paths, non-regular files и файлы больше 8 MiB отклоняются. После сборки `.plugs` содержит сами bytes, а не ссылки на локальную файловую систему разработчика.

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

### Lifecycle logging плагина

Плагин может объявить сообщения жизненного цикла в manifest `.plugs`:

```go
definition := plugin.New("acme.plugin", "Acme Plugin", "1.0.0").
    OnLoad("Acme Plugin загружен").
    OnUnload("Acme Plugin выгружен")
```

При активации плагина runtime генерирует:

```javascript
Wails.plugin.print.load("Acme Plugin загружен");
```

При удалении или замене плагина во время `Reload` генерируется:

```javascript
Wails.plugin.print.unload("Acme Plugin выгружен");
```

Сообщение lifecycle сразу передаётся host logger, а browser-console вызов ставится в очередь до следующего HTML render. Каждая transition отправляется один раз: initial load, пара `unload/load` при замене или финальный unload. При `AllowJavaScript: false` host logging сохраняется, но выполнение в browser console отключается.

## CLI и публикация

```bash
go install github.com/illussioon/WailsPlugSystem/cmd/plugs@v0.6.0
plugs pack --input ./my-plugin --output ./dist/my-plugin.plugs
plugs verify --file ./dist/my-plugin.plugs
plugs hash --file ./dist/my-plugin.plugs
```

После публичного semver tag пакет подключается стандартно:

```bash
go get github.com/illussioon/WailsPlugSystem@v0.6.0
```

После публикации тега Go tooling и `pkg.go.dev` смогут индексировать модуль и его GoDoc.
