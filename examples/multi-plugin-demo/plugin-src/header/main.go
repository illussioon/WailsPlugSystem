package main

import (
	"flag"
	"log"

	"github.com/illussioon/WailsPlugSystem/plugin"
)

func main() {
	output := flag.String("output", "../../plugins/header.plugs", "output .plugs archive")
	flag.Parse()

	definition := plugin.New("header", "Plugin Header", "1.0.0").
		Priority(60).
		DependsOn("leftmenu", "1.0.0").
		HTML().
		CSSPermission().
		HostCSS().
		JavaScript().
		OnLoad("header loaded after leftmenu dependency").
		OnUnload("header unloaded").
		AppendHTML("header-bar", "#plugin-header-slot", `
<div id="header-plugin" class="header-plugin" data-plugin="header">
  <span class="header-badge">3 plugins connected</span>
  <button type="button" id="header-ping">Run plugin ping</button>
  <button type="button" id="header-theme">Host theme</button>
</div>`, plugin.Optional()).
		AddCSSExternal("header-style", "assets/header.css", []byte(`
.header-plugin { display: flex; align-items: center; gap: 9px; }
.header-badge { border: 1px solid rgba(45, 212, 191, .34); border-radius: 999px; padding: 7px 10px; color: #99f6e4; background: rgba(13, 148, 136, .15); font-size: 11px; font-weight: 800; }
.header-plugin button { border: 1px solid rgba(148, 163, 184, .25); border-radius: 9px; padding: 8px 10px; color: #cbd5e1; background: rgba(30, 41, 59, .8); cursor: pointer; font-size: 11px; }
.header-plugin button:hover { border-color: rgba(125, 211, 252, .65); color: #e0f2fe; }
`)).
		AddJSExternal("header-entry", "assets/header.js", []byte(`
const ping = document.querySelector("#header-ping");
const theme = document.querySelector("#header-theme");
ping?.addEventListener("click", () => {
  window.Wails?.print?.console?.browser?.("header ping: tester, leftmenu and header are active");
  ping.textContent = "Ping sent";
});
theme?.addEventListener("click", () => {
  document.body.classList.toggle("header-theme-toggled");
  theme.textContent = document.body.classList.contains("header-theme-toggled") ? "Theme toggled" : "Host theme";
});
`)).
		SetText("header-status", "#host-status", "header is the priority winner", plugin.WithConflictKey("demo:host-status"), plugin.Optional()).
		Console("header-console", "header controls are ready")

	if _, err := definition.Build(*output); err != nil {
		log.Fatal(err)
	}
	log.Printf("built header plugin: %s", *output)
}
