# WailsPlugSystem: архитектура и контракт формата

## Цель

WailsPlugSystem — переносимый Go-runtime для декларативных плагинов Wails-приложений. Плагин собирается в обычный ZIP-контейнер с расширением `.plugs`, поэтому один и тот же артефакт работает на Linux, macOS и Windows. Внутри не должно быть нативной динамической библиотеки: это сознательное ограничение, обеспечивающее переносимость и предсказуемую безопасность.

## Формат `.plugs`

```text
plugin.plugs
├── manifest.json        # обязательный манифест
├── patches.json         # декларативные HTML/CSS/JS операции
└── assets/              # только файлы, перечисленные в манифесте
    ├── styles/*.css
    └── scripts/*.js
```

Манифест содержит `format_version`, `id`, `name`, `version`, `api_version`, `priority`, `permissions`, `dependencies` и `files`. `files` является allowlist активов и защищает от случайной загрузки лишних файлов. Разрешённые permissions: `html`, `css`, `js`, `replace_root`. В v1 нет произвольного native-code hook: `.so`, `.dylib`, `.dll`, исполняемые файлы и symlink запрещены.

## HTML pipeline

Поддерживаются операции `set_text`, `set_attr`, `remove`, `replace_html`, `append_html`, `prepend_html`, `add_class`, `remove_class`, `inject_css` и `inject_js`. Селекторы v1 намеренно ограничены безопасным подмножеством: `tag`, `#id`, `.class`, `[attr]`, `[attr=value]` и их комбинации через пробел. Для каждого действия вычисляется `conflict_key`. Действия с одинаковым ключом конкурируют; побеждает больший `priority`, затем меньший лексикографически `plugin.id`, затем меньший индекс операции. Неконфликтующие операции применяются все.

`replace_html` и `append/prepend_html` проходят sanitizer, который удаляет `script`, `iframe`, `object`, `embed`, event-handler attributes (`on*`) и URL-схемы `javascript:`, `data:` и `vbscript:`. Скрипт можно выполнить только как отдельный asset через permission `js`; приложение должно явно включить JS policy. Это снижает риск XSS из непроверенного HTML, но не делает доверенный JavaScript sandboxed — это явно документируется.

## Runtime semantics

`Loader` сканирует папку, проверяет расширение, размер, ZIP path traversal и SHA-256 allowlist. `Manager` держит активные плагины и при каждом изменении пересобирает итоговый DOM из исходного HTML. Поэтому unload/reload полностью откатывает revertible effects без накопления мутаций. `Apply` транзакционен: ошибка отклоняет весь новый снимок и оставляет предыдущий результат.

Зависимости проверяются до активации. Порядок применения — priority по убыванию, затем plugin ID. Конфликтующие низкоприоритетные операции не исполняются; в отчёте сохраняется причина отклонения. Включён режим strict dependencies и детерминированный результат.

## Wails integration

Основной пакет не импортирует Wails и остаётся чистым Go. Интеграция через `integration/httpmiddleware` оборачивает `http.Handler`: HTML-ответы проходят через `Manager.Render`, а остальные ответы передаются без изменения. Такой адаптер можно подключить к Wails v3 `AssetOptions.Handler`, к Wails v2 asset server или к собственному frontend dev server без OS-specific кода.

Пример:

```go
manager, _ := wailsplugs.NewManager(wailsplugs.ManagerOptions{Loader: loader})
assets := http.FileServer(http.FS(frontend))
handler := httpmiddleware.New(manager, assets)
app := application.New(application.Options{
    Assets: application.AssetOptions{Handler: handler},
})
```

## Отдельные границы безопасности

SHA-256 подтверждает целостность именно указанного артефакта, но не доказывает его происхождение. Для production рекомендуется распространять allowlist хешей из доверенного канала и ограничивать директорию плагинов. Произвольный JavaScript имеет полномочия webview и должен считаться доверенным кодом; для недоверенных поставщиков используйте только HTML/CSS permissions или будущий WASM sandbox.
