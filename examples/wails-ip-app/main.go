package main

import (
	"context"
	"embed"
	"log"

	"github.com/illussioon/WailsPlugSystem/client"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/*
var frontend embed.FS

func main() {
	plugins, err := client.New(client.Options{
		// The application loads every .plugs file from its root ./plugins folder.
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

	// The host application only knows how to serve its frontend and apply the
	// generic plugin middleware. IP detection and the extra button live in the
	// independently built plugin package.
	assets := application.BundledAssetFileServer(frontend)
	app := application.New(application.Options{
		Name:        "WailsPlugSystem IP Demo",
		Description: "A Wails app whose UI is extended by a .plugs plugin",
		Assets: application.AssetOptions{
			Handler: plugins.Handler(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "WailsPlugSystem IP Demo",
		Width:  900,
		Height: 650,
		URL:    "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
