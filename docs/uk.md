# WailsPlugSystem SDK — українська документація

[English — основна](README.md) · [Русский](ru.md) · **Українська**

**WailsPlugSystem** — це Go SDK для завантаження, композиції та розробки переносних плагінів `.plugs` для застосунків Wails. SDK має два зручні публічні пакети: `client` для host-застосунку та `plugin` для автора плагіна.

## Встановлення

```bash
go get github.com/illussioon/WailsPlugSystem@v0.5.0
```

Ядро не імпортує Wails і використовує звичайний `net/http`, тому пакет збирається для Linux, Windows і macOS.

## Client для застосунку

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

`client.New` обирає loader у такому порядку: власний `Loader`, потім SHA-256 allowlist, потім `Directory`. Після `Reload` можна викликати `Render`, `Packages` або підключити middleware через `Handler`.

### Завантаження за SHA-256

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

Хеш перевіряє точні байти архіву, але не підтверджує особу автора. Allowlist потрібно зберігати у довіреному release-каналі.

### Підключення до Wails

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

Middleware змінює лише HTML-відповіді. CSS, JS-файли, зображення, API-відповіді та status codes проходять без змін. Той самий `http.Handler` можна використовувати у Wails v2.

### Render і діагностика

```go
result, err := plugins.Render(htmlSource)
if err != nil { return err }
fmt.Println(result.HTML)
for _, decision := range result.Decisions {
    log.Printf("plugin=%s patch=%s applied=%t reason=%s",
        decision.PluginID, decision.PatchID, decision.Applied, decision.Reason)
}
```

Кожен render починається з початкового HTML, тому видалення або перезавантаження плагіна автоматично скасовує його ефекти.

## Plugin SDK для розробника

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

Основні методи: `Priority`, `HTML`, `CSSPermission`, `JavaScript`, `ReplaceRoot`, `DependsOn`, `SetText`, `SetAttr`, `Remove`, `ReplaceHTML`, `AppendHTML`, `PrependHTML`, `AddClass`, `RemoveClass`, `AddCSS`, `AddJS`, `Asset`, `WriteSource` і `Build`.

`WithConflictKey` задає спільний ресурс, за який конкурують плагіни. `Optional()` робить відсутність selector або asset нефатальною.

## Пріоритети та залежності

Плагіни сортуються за `priority` у порядку спадання, потім за ID. Якщо два патчі мають однаковий `conflict_key`, перемагає перший; рішення записується в `RenderResult.Decisions`. Різні ключі об’єднуються. `DependsOn` задає залежність, а `StrictDependencies: true` відхиляє reload при відсутньому або несумісному plugin version.

## Формат `.plugs`

```text
plugin.plugs
├── manifest.json
├── patches.json
└── assets/
    ├── theme.css
    └── app.js
```

Builder автоматично записує SHA-256 кожного asset. Runtime відхиляє незаявлені файли, symlink, traversal paths, абсолютні шляхи, пошкоджені manifest і надто великі архіви.

`.plugs` — це переносний data package, а не native executable. Усередині не завантажуються `.dll`, `.so`, `.dylib` або довільні Go plugins.

## Безпека

HTML fragments проходять sanitizer: видаляються script-like елементи, event-handler attributes та небезпечні URL schemes. JavaScript потребує одночасно `plugin.JavaScript()` і `client.Options{AllowJavaScript: true}`. JavaScript не sandboxed і виконується зі звичайними правами WebView, тому дозволяйте його лише довіреним постачальникам. Для недовірених плагінів використовуйте SHA-256 allowlist, вимкнений JavaScript і не вмикайте `ReplaceRoot`.

## File-based authoring

Для великих інтерфейсів не обов’язково вставляти HTML/CSS/JS рядками. SDK прочитає звичайні файли проєкту, перевірить їх, додасть до allowlist assets, розрахує SHA-256 під час пакування та створить потрібні patches:

```go
definition := plugin.New("acme.react-ui", "Acme React UI", "1.0.0").
    AppendHTMLFile("mount", "body", "ui/mount.html").
    CSSFile("styles", "styles/ui.css", "ui/styles.css").
    JSFile("bundle", "scripts/ui.js", "dist/ui.bundle.js").
    AssetFile("assets/icon.svg", "ui/icon.svg").
    AssetsDir("ui/static")

_, err := definition.Build("dist/acme.react-ui.plugs")
```

`AppendHTMLFile` і `ReplaceHTMLFile` використовують вміст файлу як HTML fragment. `CSSFile` і `JSFile` зберігають наявні permission checks. `AssetFile` додає один довільний файл, а `AssetsDir` рекурсивно додає regular files до `assets/`. Symlinks, небезпечні archive paths, non-regular files і файли більші за 8 MiB відхиляються. Після збірки `.plugs` містить самі bytes, а не посилання на локальну файлову систему розробника.

## Console logging

Плагін може писати в консоль host/Wails і в консоль WebView/browser:

```go
definition := plugin.New("acme.debug", "Acme Debug", "1.0.0").
    JavaScript().
    Console("startup", "Hello World").
    ConsoleBrowser("browser-startup", "Hello World у консолі WebView2/browser")
```

Ці helpers генерують:

```javascript
Wails.print.console("Hello World");
Wails.print.console.browser("Hello World у консолі WebView2/browser");
```

`Wails.print.console` надсилає JSON-повідомлення host-застосунку. Його можна отримати через `client.Options.HostLogger`; якщо callback не заданий, використовується стандартний Go logger. `Wails.print.console.browser` пише безпосередньо в developer console нативного WebView/browser. Для обох методів host має увімкнути `AllowJavaScript: true`.

### Lifecycle logging плагіна

Плагін може оголосити повідомлення життєвого циклу в manifest `.plugs`:

```go
definition := plugin.New("acme.plugin", "Acme Plugin", "1.0.0").
    OnLoad("Acme Plugin завантажено").
    OnUnload("Acme Plugin вивантажено")
```

Під час активації плагіна runtime генерує:

```javascript
Wails.plugin.print.load("Acme Plugin завантажено");
```

Під час видалення або заміни плагіна через `Reload` генерується:

```javascript
Wails.plugin.print.unload("Acme Plugin вивантажено");
```

Lifecycle-повідомлення одразу передається host logger, а browser-console виклик ставиться в чергу до наступного HTML render. Кожна transition надсилається один раз: initial load, пара `unload/load` під час заміни або фінальний unload. При `AllowJavaScript: false` host logging зберігається, але виконання у browser console вимикається.

## CLI та публікація

```bash
go install github.com/illussioon/WailsPlugSystem/cmd/plugs@v0.5.0
plugs pack --input ./my-plugin --output ./dist/my-plugin.plugs
plugs verify --file ./dist/my-plugin.plugs
plugs hash --file ./dist/my-plugin.plugs
```

Після публічного semver tag пакет підключається стандартно:

```bash
go get github.com/illussioon/WailsPlugSystem@v0.5.0
```

Після публікації тега Go tooling і `pkg.go.dev` зможуть індексувати модуль і його GoDoc.
