# Multi-Plugin Wails Demo

This Wails v2 demo has one native host executable and three platform-independent `.plugs` archives. The host loads `./plugins/tester.plugs`, `./plugins/leftmenu.plugs`, and `./plugins/header.plugs`; plugins are not compiled into separate native executables.

The host must be built with Wails' `production` build tag. On an Apple Silicon Mac, run:

```bash
go mod download
go test ./...
go build -tags production -trimpath -ldflags='-s -w' -o wailsplugs-demo .
./wailsplugs-demo
```

The plugin archives can be regenerated with the Go plugin source commands:

```bash
go run ./plugin-src/tester -output ./plugins/tester.plugs
go run ./plugin-src/leftmenu -output ./plugins/leftmenu.plugs
go run ./plugin-src/header -output ./plugins/header.plugs
```

The host report shows all active manifests, SHA-256 values, permissions, priorities, dependencies, and lifecycle messages. The tester plugin also checks the external module, dynamic chunk, and static asset route.

`WAILSPLUGS_WATCH=1 ./wailsplugs-demo` enables development polling hot reload for changed `.plugs` files.
