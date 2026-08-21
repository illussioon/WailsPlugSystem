# WailsPlugSystem

**WailsPlugSystem** — кроссплатформенная система декларативных плагинов для Go/Wails-приложений. Плагин собирается в обычный ZIP-контейнер с расширением `.plugs`, поэтому один и тот же файл можно использовать на Linux, macOS и Windows. Основной runtime не зависит от Wails и не импортирует platform-specific ABI.

Проект наследует практические идеи [Cordis](https://github.com/cordiverse/cordis) и paper [A Programming Paradigm for Spatiotemporal Composability](https://github.com/cordiverse/paper): отслеживаемые изменения, обратимость эффектов, разрешение зависимостей, детерминированную композицию, reconciliation и безопасное отключение. При этом реализация написана на Go и специально ограничивает исполняемый код внутри `.plugs`.

> **Важно:** SHA-256 проверяет целостность и принадлежность файла allowlist, но не доказывает доверенность автора. Произвольный JavaScript считается доверенным кодом webview и включается только явной настройкой приложения.

## Возможности

| Возможность | Реализация |
| --- | --- |
| Кроссплатформенность | Чистый Go и ZIP-пакет; Linux, macOS и Windows используют один `.plugs` |
| HTML-модификации | `set_text`, `set_attr`, `remove`, `replace_html`, `append_html`, `prepend_html`, классы |
| CSS/JS assets | `inject_css` и явно разрешённый `inject_js` |
| Полная перестройка UI | `replace_root` permission и `AllowRootReplace` policy |
| Конфликты | `priority` по убыванию, затем детерминированный `plugin.id`; `conflict_key` для явного владения блоком |
| Откат | Каждый render начинается с исходного HTML; unload/reload не накапливает мутации |
| Зависимости | Strict dependency validation по ID и точной версии при указании версии |
| Загрузка | Директория, рекурсивная директория и SHA-256 allowlist |
| Безопасность | ZIP path traversal protection, symlink rejection, size limits, manifest allowlist, HTML sanitizer |
| Wails integration | Универсальный `http.Handler`, совместимый с Wails v3 AssetOptions и asset server pipeline |

## Установка

```bash
go get github.com/illussioon/WailsPlugSystem

go install github.com/illussioon/WailsPlugSystem/cmd/plugs@latest
```

Для использования текущего исходного дерева:

```bash
go test ./...
go build ./cmd/plugs
```

## Формат исходников плагина

Каталог плагина содержит `manifest.json`, `patches.json` и необязательный каталог `assets/`:

```text
my-plugin/
├── manifest.json
├── patches.json
└── assets/
    ├── app.css
    └── app.js
```

Минимальный манифест:

```json
{
  "format_version": 1,
  "id": "demo.theme",
  "name": "Demo Theme",
  "version": "1.0.0",
  "api_version": "v1",
  "priority": 100,
  "permissions": ["html", "css"],
  "dependencies": []
}
```

`id` содержит только нижний регистр, цифры, `.`, `_` и `-`. При упаковке builder автоматически перечисляет все файлы под `assets/` и записывает их SHA-256 в `manifest.files`. Это предотвращает загрузку незаявленных активов.

## Патчи HTML/CSS/JS

```json
[
  {
    "id": "replace-title",
    "kind": "set_text",
    "selector": "#app-title",
    "value": "My Wails App",
    "conflict_key": "app-title"
  },
  {
    "id": "add-toolbar",
    "kind": "append_html",
    "selector": "body",
    "value": "<div class=\"plugin-toolbar\">Toolbar</div>"
  },
  {
    "id": "load-css",
    "kind": "inject_css",
    "asset": "assets/app.css",
    "conflict_key": "theme-css"
  }
]
```

Поддерживаемый selector — намеренно ограниченное подмножество CSS: `tag`, `#id`, `.class`, `[attr]`, `[attr=value]`, а также последовательность таких компонентов через пробел. Это не полноценный CSS selector engine; ограничение упрощает аудит и делает результат одинаковым на всех ОС.

`replace_html`, `append_html` и `prepend_html` проходят sanitizer. Он удаляет `script`, `iframe`, `object`, `embed`, `link`, event-handler attributes (`onclick` и аналогичные), а также URL schemes `javascript:`, `vbscript:` и `data:`. Для JavaScript используйте отдельный asset и permission `js`.

## Сборка `.plugs`

```bash
plugs pack --input ./my-plugin --output ./dist/my-plugin.plugs
plugs verify --file ./dist/my-plugin.plugs
plugs hash --file ./dist/my-plugin.plugs
```

То же самое из Go:

```go
path, err := pack.Build(pack.Options{
    InputDir: "./my-plugin",
    Output:   "./dist/my-plugin.plugs",
})
```

В контейнер попадают только `manifest.json`, `patches.json` и файлы из `assets/`. Упаковщик создаёт стабильный порядок записей, проверяет пути и пересчитывает manifest allowlist.

## Подключение к приложению

### Загрузка из каталога

```go
package main

import (
    "context"

    wailsplugs "github.com/illussioon/WailsPlugSystem"
    "github.com/illussioon/WailsPlugSystem/loader"
)

func main() {
    pluginLoader := loader.Directory{
        Dir:       "./plugins",
        Recursive: false,
        PackageOptions: wailsplugs.PackageOptions{
            MaxArchiveBytes: 32 << 20,
            MaxFileBytes:    8 << 20,
        },
    }
    manager := wailsplugs.NewManager(wailsplugs.ManagerOptions{
        Loader:             pluginLoader,
        StrictDependencies: true,
        AllowJavaScript:    false,
        AllowRootReplace:   false,
    })
    if err := manager.Reload(context.Background()); err != nil {
        panic(err)
    }
}
```

### Загрузка только по SHA-256

```go
pluginLoader := loader.SHA256Allowlist{
    Dir:       "./trusted-plugins",
    Recursive: true,
    SHA256: []string{
        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    },
}
manager := wailsplugs.NewManager(wailsplugs.ManagerOptions{
    Loader:             pluginLoader,
    StrictDependencies: true,
})
```

Allowlist следует хранить отдельно от пользовательской директории плагинов и обновлять через доверенный канал. Если hash отсутствует в allowlist, пакет игнорируется; если выбранный пакет повреждён, `OpenPackage` возвращает ошибку integrity.

## Интеграция с Wails

Основной адаптер — `integration/httpmiddleware`. Он оборачивает asset handler, перехватывает только HTML-ответы и передаёт их через `Manager.Render`. CSS, изображения, JS-файлы и API-ответы не изменяются.

Для Wails v3:

```go
import (
    "embed"
    "net/http"

    "github.com/wailsapp/wails/v3/pkg/application"
    wailsplugs "github.com/illussioon/WailsPlugSystem"
    "github.com/illussioon/WailsPlugSystem/integration/httpmiddleware"
)

//go:embed frontend/*
var frontend embed.FS

baseAssets := application.AssetFileServerFS(frontend)
pluginAssets := httpmiddleware.New(manager, baseAssets)
app := application.New(application.Options{
    Assets: application.AssetOptions{Handler: pluginAssets},
})
```

Для Wails v2 используется тот же принцип: передайте `httpmiddleware.New(manager, baseHandler)` в asset/HTTP pipeline, который обслуживает frontend. Библиотека намеренно не импортирует `github.com/wailsapp/wails/v2` или `/v3`, поэтому core package остаётся чистым и кросскомпилируется обычными командами Go.

После изменения содержимого каталога можно вызвать `manager.Reload(ctx)`. Следующий `Render` использует новый snapshot. Если приложение применяет собственный кеш HTML, его необходимо инвалидировать после reload.

## Приоритеты и конфликты

Плагин с большим `priority` применяется раньше. Если два патча имеют одинаковый `conflict_key`, применяется только первый. Для одинакового priority используется лексикографический порядок `plugin.id`, поэтому результат не зависит от порядка файлов в каталоге. Неконфликтующие патчи выполняются все.

Например, `theme-dark` с priority `100` и `theme-light` с priority `10` могут оба владеть `theme-css`; всегда победит `theme-dark`. В `RenderResult.Decisions` видны применённые и отклонённые патчи, что позволяет показать пользователю диагностику.

## Модель безопасности

`.plugs` не является исполняемым бинарным модулем. Runtime не загружает `.dll`, `.so`, `.dylib`, ELF и другие native artifacts. Это ключевое условие переносимости и уменьшения attack surface. Если приложению требуются native hooks, их следует регистрировать в самом host-приложении через официальный Go SDK и контролировать отдельно от `.plugs`.

HTML sanitizer не является sandbox для JavaScript. `AllowJavaScript: true` следует включать только для доверенных plugin authors. Для недоверенных плагинов используйте только `html`/`css`, SHA-256 allowlist и запрет `replace_root`.

Архив проверяется на path traversal, абсолютные пути, backslash traversal, symlink entries, неизвестные файлы, размер архива и размер каждого файла. Все операции применения выполняются на DOM-снимке; при ошибке обязательного патча исходный HTML не заменяется.

## Тестирование и CI

```bash
go test ./...
go test -race ./...
GOOS=linux GOARCH=amd64 go build ./...
GOOS=windows GOARCH=amd64 go build ./...
GOOS=darwin GOARCH=amd64 go build ./...
```

GitHub Actions запускает unit/integration tests и cross-platform compile matrix. Реальный Wails frontend smoke test рекомендуется выполнять в приложении-потребителе, поскольку WebView runtime и системные GUI dependencies зависят от версии Wails и платформы.

## Лицензия и источники

Исходный код WailsPlugSystem распространяется под MIT License. Проект вдохновлён концепциями [Cordis](https://github.com/cordiverse/cordis) и [paper](https://github.com/cordiverse/paper), но не включает их TypeScript-исходники и не требует их runtime. Для Wails API см. [официальный репозиторий Wails](https://github.com/wailsapp/wails).

## References

[1]: https://github.com/cordiverse/paper "A Programming Paradigm for Spatiotemporal Composability"
[2]: https://github.com/cordiverse/cordis "Cordis — Meta-Framework of Spatiotemporal Composability"
[3]: https://github.com/wailsapp/wails "Wails — Create beautiful applications using Go"
