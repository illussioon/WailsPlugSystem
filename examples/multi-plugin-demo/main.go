package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/client"
	"github.com/illussioon/WailsPlugSystem/devwatch"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend/*
var frontend embed.FS

type PluginInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Priority    int      `json:"priority"`
	SHA256      string   `json:"sha256"`
	Files       int      `json:"files"`
	Permissions []string `json:"permissions"`
	OnLoad      string   `json:"on_load"`
	OnUnload    string   `json:"on_unload"`
}

type App struct {
	plugins *client.Client
	mu      sync.RWMutex
	logs    []wailsplugs.ConsoleMessage
}

func (a *App) record(message wailsplugs.ConsoleMessage) {
	log.Printf("[plugin:%s] %s", message.PluginID, message.Message)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logs = append(a.logs, message)
	if len(a.logs) > 100 {
		a.logs = a.logs[len(a.logs)-100:]
	}
}

// GetPluginReport is called by the demo UI to show the active package snapshot.
func (a *App) GetPluginReport() []PluginInfo {
	if a == nil || a.plugins == nil {
		return nil
	}
	packages := a.plugins.Packages()
	result := make([]PluginInfo, 0, len(packages))
	for _, item := range packages {
		permissions := make([]string, 0, len(item.Manifest.Permissions))
		for _, permission := range item.Manifest.Permissions {
			permissions = append(permissions, string(permission))
		}
		result = append(result, PluginInfo{
			ID:          item.Manifest.ID,
			Name:        item.Manifest.Name,
			Version:     item.Manifest.Version,
			Priority:    item.Manifest.Priority,
			SHA256:      item.SHA256,
			Files:       len(item.Manifest.Files),
			Permissions: permissions,
			OnLoad:      item.Manifest.Lifecycle.Load,
			OnUnload:    item.Manifest.Lifecycle.Unload,
		})
	}
	return result
}

// ReloadPlugins reloads all plugin archives from the demo's plugins directory.
func (a *App) ReloadPlugins() error {
	if a == nil || a.plugins == nil {
		return nil
	}
	return a.plugins.Reload(context.Background())
}

func main() {
	app := &App{}
	plugins, err := client.New(client.Options{
		Directory:          "./plugins",
		StrictDependencies: true,
		AllowJavaScript:    true,
		AllowRootReplace:   false,
		HostLogger:         app.record,
	})
	if err != nil {
		log.Fatal(err)
	}
	app.plugins = plugins
	if err := plugins.Reload(context.Background()); err != nil {
		log.Fatal(err)
	}

	if os.Getenv("WAILSPLUGS_WATCH") == "1" {
		go func() {
			if err := plugins.Watch(context.Background(), client.WatchOptions{
				Directory:  "./plugins",
				Interval:   300 * time.Millisecond,
				RunInitial: false,
				OnReload: func(_ context.Context, change devwatch.Change) error {
					log.Printf("plugin reload: added=%v modified=%v removed=%v", change.Added, change.Modified, change.Removed)
					return nil
				},
			}); err != nil {
				log.Printf("plugin watcher stopped: %v", err)
			}
		}()
	}

	err = wails.Run(&options.App{
		Title:  "WailsPlugSystem Multi-Plugin Lab",
		Width:  1180,
		Height: 760,
		Bind:   []interface{}{app},
		AssetServer: &assetserver.Options{
			Assets: frontend,
			Middleware: assetserver.Middleware(func(next http.Handler) http.Handler {
				return plugins.Handler(next)
			}),
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 1},
	})
	if err != nil {
		log.Fatal(err)
	}
}
