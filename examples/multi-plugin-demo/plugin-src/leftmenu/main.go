package main

import (
	"flag"
	"log"

	"github.com/illussioon/WailsPlugSystem/plugin"
)

func main() {
	output := flag.String("output", "../../plugins/leftmenu.plugs", "output .plugs archive")
	flag.Parse()

	definition := plugin.New("leftmenu", "Left Menu", "1.0.0").
		Priority(40).
		HTML().
		CSSPermission().
		HostCSS().
		JavaScript().
		OnLoad("leftmenu loaded").
		OnUnload("leftmenu unloaded").
		AppendHTML("leftmenu", "#plugin-leftmenu-slot", `
<div id="leftmenu-plugin" class="leftmenu-plugin" data-plugin="leftmenu">
  <div class="leftmenu-title">Plugin menu</div>
  <button type="button" data-leftmenu-target="overview">Overview</button>
  <button type="button" data-leftmenu-target="diagnostics">Diagnostics</button>
  <button type="button" data-leftmenu-target="assets">Assets &amp; chunks</button>
  <div id="leftmenu-feedback" class="leftmenu-feedback">Choose a plugin view</div>
</div>`, plugin.Optional()).
		AddCSSExternal("leftmenu-style", "assets/leftmenu.css", []byte(`
.leftmenu-plugin { display: grid; gap: 7px; }
.leftmenu-title { margin: 0 10px 6px; color: #f8fafc; font-size: 13px; font-weight: 800; }
.leftmenu-plugin button { width: 100%; border: 1px solid transparent; border-radius: 10px; padding: 10px 11px; text-align: left; color: #cbd5e1; background: transparent; cursor: pointer; }
.leftmenu-plugin button:hover, .leftmenu-plugin button.is-active { border-color: rgba(56, 189, 248, .3); color: #e0f2fe; background: rgba(14, 116, 144, .25); }
.leftmenu-feedback { margin: 8px 10px 0; color: #64748b; font-size: 11px; line-height: 1.4; }
`)).
		AddJSExternal("leftmenu-entry", "assets/leftmenu.js", []byte(`
const feedback = document.querySelector("#leftmenu-feedback");
document.querySelectorAll("[data-leftmenu-target]").forEach((button) => {
  button.addEventListener("click", () => {
    document.querySelectorAll("[data-leftmenu-target]").forEach((item) => item.classList.remove("is-active"));
    button.classList.add("is-active");
    if (feedback) feedback.textContent = "Selected: " + button.dataset.leftmenuTarget;
    window.Wails?.print?.console?.("leftmenu selected", button.dataset.leftmenuTarget);
  });
});
`)).
		Console("leftmenu-console", "leftmenu navigation is ready").
		SetText("leftmenu-status", "#host-status", "leftmenu loaded the host", plugin.WithConflictKey("demo:host-status"), plugin.Optional())

	if _, err := definition.Build(*output); err != nil {
		log.Fatal(err)
	}
	log.Printf("built leftmenu plugin: %s", *output)
}
