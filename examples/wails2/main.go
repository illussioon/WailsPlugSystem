package main

import (
	"context"
	"embed"
	"log"
	"net/http"

	"github.com/illussioon/WailsPlugSystem/client"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend/*
var assets embed.FS

func main() {
	plugins, err := client.New(client.Options{
		// Wails v2 host loads every .plugs file from its root ./plugins folder.
		Directory:          "./plugins",
		StrictDependencies: true,
		AllowJavaScript:    true,
		AllowRootReplace:   false,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := plugins.Reload(context.Background()); err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:  "WailsPlugSystem Wails 2 Demo",
		Width:  900,
		Height: 650,
		AssetServer: &assetserver.Options{
			Assets: assets,
			Middleware: assetserver.Middleware(func(next http.Handler) http.Handler {
				return plugins.Handler(next)
			}),
		},
		BackgroundColour: &options.RGBA{R: 17, G: 24, B: 39, A: 1},
	})
	if err != nil {
		log.Fatal(err)
	}
}
