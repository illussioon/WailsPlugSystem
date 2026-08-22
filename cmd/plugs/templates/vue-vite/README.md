# Vue plugin template

This starter uses Vite and Vue. The plugin entrypoint is `plugin/main.go`; it packages `ui/mount.html`, the shared host stylesheet, the Vite entry file, and the generated `dist/chunks/` directory.

```bash
npm install
npm run build
go run ./plugin -output ./dist/vue.example.plugs
```

Copy `dist/vue.example.plugs` to the host application's `plugins/` directory. The generated plugin uses external asset URLs, so dynamic imports and hashed Vite chunks remain available through the WailsPlugSystem middleware.

The `HostCSS()` declaration intentionally keeps the Vue mount in the host document. It can inherit host typography, CSS variables, reset rules, and other inherited properties. Avoid relying on an iframe or shadow-root boundary in this template.
