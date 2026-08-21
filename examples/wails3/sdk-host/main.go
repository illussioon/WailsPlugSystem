package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/illussioon/WailsPlugSystem/client"
)

func main() {
	directory := flag.String("plugins", "./plugins", "plugin directory")
	flag.Parse()

	plugins, err := client.New(client.Options{
		Directory:          *directory,
		StrictDependencies: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := plugins.Reload(context.Background()); err != nil {
		log.Fatal(err)
	}

	source := `<html><head><title>SDK Host</title></head><body><h1 id="app-title">Original title</h1></body></html>`
	result, err := plugins.Render(source)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.HTML)
	if err := os.WriteFile("./rendered.html", []byte(result.HTML), 0644); err != nil {
		log.Fatal(err)
	}
}
