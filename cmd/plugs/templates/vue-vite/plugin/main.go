package main

import (
	"flag"
	"log"

	"github.com/illussioon/WailsPlugSystem/plugin"
)

func main() {
	output := flag.String("output", "./dist/vue.example.plugs", "output .plugs path")
	flag.Parse()

	definition := plugin.New("example.vue", "Vue Plugin", "1.0.0").
		OnLoad("Vue plugin loaded").
		OnUnload("Vue plugin unloaded").
		HostCSS().
		AppendHTMLFile("vue-mount", "body", "ui/mount.html").
		CSSFileExternal("vue-styles", "assets/ui.css", "src/style.css").
		JSFileExternal("vue-entry", "assets/ui.js", "dist/ui.js").
		AssetsDirAs("dist/chunks", "assets/chunks")

	if _, err := definition.Build(*output); err != nil {
		log.Fatal(err)
	}
	log.Printf("built Vue plugin: %s", *output)
}
