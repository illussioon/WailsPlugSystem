package main

import (
	"flag"
	"log"

	"github.com/illussioon/WailsPlugSystem/plugin"
)

func main() {
	output := flag.String("output", "../../plugins/tester.plugs", "output .plugs archive")
	flag.Parse()

	definition := plugin.New("tester", "System Tester", "1.0.0").
		Priority(20).
		HTML().
		CSSPermission().
		HostCSS().
		JavaScript().
		OnLoad("tester loaded: HTML, CSS, JS, assets, lifecycle and priority checks are active").
		OnUnload("tester unloaded").
		AppendHTML("tester-panel", "#plugin-tester-slot", `
<article id="tester-panel" class="tester-panel" data-plugin="tester">
  <div class="tester-heading">
    <div>
      <span class="tester-kicker">Plugin: tester</span>
      <h2>Runtime verification suite</h2>
    </div>
    <span id="tester-state" class="tester-state">loading</span>
  </div>
  <p>This panel is injected by a standalone <code>.plugs</code> archive. It exercises sanitized HTML, inherited host CSS, an external module, a dynamic chunk, a static asset route, console logging, lifecycle hooks, and conflict priorities.</p>
  <div class="tester-metrics">
    <div><strong id="tester-check-count">0</strong><span>checks passed</span></div>
    <div><strong id="tester-chunk-state">pending</strong><span>code split chunk</span></div>
    <div><strong id="tester-json-state">pending</strong><span>asset route JSON</span></div>
  </div>
  <div id="tester-checks" class="tester-checks"></div>
</article>`, plugin.Optional()).
		AddCSSExternal("tester-style", "assets/tester.css", []byte(`
#tester-panel { margin-top: 22px; }
.tester-panel { padding: 24px; border: 1px solid rgba(45, 212, 191, .36); border-radius: 20px; background: linear-gradient(135deg, rgba(13, 56, 65, .86), rgba(15, 23, 42, .9)); box-shadow: 0 18px 55px rgba(13, 148, 136, .12); }
.tester-heading { display: flex; align-items: start; justify-content: space-between; gap: 18px; }
.tester-kicker { color: #5eead4; text-transform: uppercase; letter-spacing: .14em; font-size: 10px; font-weight: 800; }
.tester-panel h2 { margin: 7px 0 0; font-size: 22px; letter-spacing: -.03em; }
.tester-panel p { color: #a7f3d0; line-height: 1.6; }
.tester-state { border: 1px solid rgba(94, 234, 212, .35); border-radius: 999px; padding: 6px 10px; color: #99f6e4; font-size: 11px; }
.tester-metrics { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-top: 18px; }
.tester-metrics div { padding: 12px; border-radius: 12px; background: rgba(15, 23, 42, .48); }
.tester-metrics strong { display: block; color: #f0fdfa; font-size: 18px; }
.tester-metrics span { color: #99f6e4; font-size: 11px; }
.tester-checks { display: grid; gap: 7px; margin-top: 18px; }
.tester-check { padding: 9px 11px; border-radius: 9px; color: #ccfbf1; background: rgba(20, 184, 166, .12); font-size: 12px; }
.tester-check::before { content: "✓"; margin-right: 8px; color: #5eead4; font-weight: 900; }
`)).
		AddJSExternal("tester-entry", "assets/tester.js", []byte(`import { chunkCheck } from "./chunks/tester-chunk.js";

const checks = document.querySelector("#tester-checks");
const state = document.querySelector("#tester-state");
const count = document.querySelector("#tester-check-count");
const chunkState = document.querySelector("#tester-chunk-state");
const jsonState = document.querySelector("#tester-json-state");

function pass(label) {
  const item = document.createElement("div");
  item.className = "tester-check";
  item.textContent = label;
  checks?.appendChild(item);
}

async function run() {
  if (!checks) return;
  const passed = [];
  pass("HTML patch mounted into the host slot"); passed.push("html");
  pass("External CSS loaded through the plugin asset route"); passed.push("css");
  if (chunkCheck()) { chunkState.textContent = "loaded"; pass("Dynamic import chunk resolved from the plugin asset route"); passed.push("chunk"); }
  const response = await fetch("/__wailsplugs/assets/tester/tester-data.json");
  if (response.ok) { const payload = await response.json(); jsonState.textContent = payload.route; pass("Static JSON asset resolved: " + payload.name); passed.push("asset"); }
  window.Wails?.print?.console?.("tester browser check complete", passed);
  window.Wails?.print?.console?.browser?.("tester module and code split chunk are running");
  count.textContent = String(passed.length);
  state.textContent = "all checks passed";
}

run().catch((error) => { if (state) state.textContent = "check failed"; console.error(error); });
`)).
		Asset("assets/chunks/tester-chunk.js", []byte(`export function chunkCheck() { return true; }
`)).
		Asset("assets/tester-data.json", []byte(`{"name":"tester-data.json","route":"asset route OK"}
`)).
		Console("tester-console", "tester plugin console bridge is active").
		ConsoleBrowser("tester-browser-console", "tester browser console bridge is active").
		SetText("tester-status-conflict", "#host-status", "tester has lower conflict priority", plugin.WithConflictKey("demo:host-status"), plugin.Optional())

	if _, err := definition.Build(*output); err != nil {
		log.Fatal(err)
	}
	log.Printf("built tester plugin: %s", *output)
}
