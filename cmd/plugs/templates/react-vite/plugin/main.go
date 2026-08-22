package main

import (
	"flag"
	"log"

	"github.com/illussioon/WailsPlugSystem/plugin"
)

func main() {
	output := flag.String("output", "./dist/react.example.plugs", "output .plugs path")
	flag.Parse()

	definition := plugin.New("example.react", "React Plugin", "1.0.0").
		OnLoad("React plugin loaded").
		OnUnload("React plugin unloaded").
		HostCSS().
		AppendHTMLFile("react-mount", "body", "ui/mount.html").
		CSSFileExternal("react-styles", "assets/ui.css", "src/style.css").
		JSFileExternal("react-entry", "assets/ui.js", "dist/ui.js").
		AssetsDirAs("dist/chunks", "assets/chunks")

	if _, err := definition.Build(*output); err != nil {
		log.Fatal(err)
	}
	log.Printf("built React plugin: %s", *output)
}
