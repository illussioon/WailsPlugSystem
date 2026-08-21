package main

import (
	"flag"
	"log"

	"github.com/illussioon/WailsPlugSystem/plugin"
)

func main() {
	output := flag.String("output", "./dist/sdk.example.plugs", "output .plugs path")
	flag.Parse()

	definition := plugin.New("sdk.example", "SDK Example", "1.0.0").
		Priority(100).
		HTML().
		SetText("title", "#app-title", "Hello from the Go plugin SDK").
		AppendHTML("badge", "body", `<div class="plugin-badge">Loaded from .plugs</div>`).
		AddCSS("theme", "theme.css", []byte(".plugin-badge { color: #2563eb; font-weight: 600; }"), plugin.WithConflictKey("sdk-badge-style"))

	if _, err := definition.Build(*output); err != nil {
		log.Fatal(err)
	}
	log.Printf("built %s", *output)
}
