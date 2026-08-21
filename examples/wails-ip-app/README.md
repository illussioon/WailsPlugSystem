# Wails IP Plugin Demo

Это реальное Wails v3 demo-приложение с независимым плагином.

## Архитектура

Host-приложение (`main.go`) содержит только базовый frontend и generic plugin loader:

```text
examples/wails-ip-app/
├── frontend/index.html              # базовая страница без IP-функциональности
├── plugins/                         # runtime directory приложения
│   └── ip.example.plugs             # собранный plugin artifact
├── plugin-src/ip-plugin/main.go     # исходник независимого Go plugin
├── main.go                          # Wails host
└── go.mod
```

В host-коде нет определения IP и нет кода кнопки. При старте он загружает все `.plugs` из `./plugins`, применяет их через `client.Handler`, а затем передаёт HTML в Wails asset server.

IP plugin добавляет HTML card, CSS, кнопку и JavaScript. JavaScript делает запрос к `https://api64.ipify.org?format=json`, показывает результат как `Ваш IP: ...` и позволяет повторить запрос кнопкой `Обновить IP`.

## Запуск

Из этой директории:

```bash
cd examples/wails-ip-app

go mod tidy

go run ./plugin-src/ip-plugin \
  -output ./plugins/ip.example.plugs

# Запуск Wails v3 development application
wails3 dev
```

Либо можно собрать host:

```bash
go build -o wails-ip-demo .
./wails-ip-demo
```

Для обычного Wails workflow установите Wails v3 CLI согласно [официальной документации Wails](https://github.com/wailsapp/wails/tree/master/v3). На Windows и macOS потребуется стандартный набор системных GUI dependencies Wails.

## Важная настройка JavaScript

В приложении установлено `AllowJavaScript: true`, потому что plugin injects JavaScript для выполнения HTTP request. Это не IP-specific permission: host лишь разрешает trusted JavaScript policy, а вся конкретная логика находится в plugin archive.

Если поставить `AllowJavaScript: false`, приложение продолжит запускаться, но IP plugin будет отклонён при render с ошибкой permission policy.

## Перемещение плагина

Runtime directory определяется относительно текущей рабочей директории процесса:

```go
client.New(client.Options{Directory: "./plugins"})
```

Поэтому `.plugs` можно заменить, добавить или удалить в папке `plugins` без изменения host source. Для повторного чтения нового набора вызовите `plugins.Reload(context.Background())`.

## Ограничения demo

Публичный ipify endpoint нужен только для демонстрации. Для production используйте собственный endpoint, задайте trusted SHA-256 allowlist и учитывайте, что внешний HTTP request зависит от сетевого соединения и политики WebView.
