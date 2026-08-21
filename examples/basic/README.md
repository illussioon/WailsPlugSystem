# Basic example

Соберите пример из корня репозитория:

```bash
go run ./cmd/plugs pack \
  --input ./examples/basic/plugin \
  --output ./examples/basic/example.welcome.plugs

go run ./cmd/plugs verify \
  --file ./examples/basic/example.welcome.plugs
```

Для host-приложения скопируйте `example.welcome.plugs` в `./plugins`, создайте `loader.Directory{Dir: "./plugins"}` и подключите `httpmiddleware.New(manager, application.AssetFileServerFS(frontend))` к Wails v3 `application.AssetOptions.Handler`. Пример меняет текст `#app-title` и добавляет CSS без выполнения JavaScript.
