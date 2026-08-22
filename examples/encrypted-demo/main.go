package main

import (
	"context"
	"embed"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/client"
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
	Encryption  string   `json:"encryption"`
	Priority    int      `json:"priority"`
	SHA256      string   `json:"sha256"`
	Files       int      `json:"files"`
	Permissions []string `json:"permissions"`
}

type App struct {
	plugins    *client.Client
	keyPath    string
	mu         sync.RWMutex
	logs       []wailsplugs.ConsoleMessage
	loadStatus string
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

func (a *App) keyProvider(_ wailsplugs.Manifest) ([]byte, error) {
	return readKey(a.keyPath)
}

func (a *App) setLoadStatus(status string) {
	a.mu.Lock()
	a.loadStatus = status
	a.mu.Unlock()
}

func (a *App) GetSecurityStatus() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.loadStatus
}

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
			ID: item.Manifest.ID, Name: item.Manifest.Name, Version: item.Manifest.Version,
			Encryption: item.Manifest.Encryption, Priority: item.Manifest.Priority,
			SHA256: item.SHA256, Files: len(item.Manifest.Files), Permissions: permissions,
		})
	}
	return result
}

func (a *App) ReloadPlugins() error {
	if a == nil || a.plugins == nil {
		return nil
	}
	if err := a.plugins.Reload(context.Background()); err != nil {
		a.setLoadStatus("FAILED: " + err.Error())
		return err
	}
	a.setLoadStatus(fmt.Sprintf("OK: %d encrypted plugin(s) loaded with authenticated decryption", len(a.plugins.Packages())))
	return nil
}

func main() {
	app := &App{keyPath: "./demo.key", loadStatus: "Waiting for ./demo.key"}
	plugins, err := client.New(client.Options{
		Directory:             "./plugins",
		StrictDependencies:    true,
		AllowJavaScript:       true,
		AllowRootReplace:      false,
		HostLogger:            app.record,
		DecryptionKeyProvider: app.keyProvider,
	})
	if err != nil {
		log.Fatal(err)
	}
	app.plugins = plugins
	if err := app.ReloadPlugins(); err != nil {
		// The UI intentionally remains available so a user can test the
		// no-key/wrong-key failure and then retry after installing demo.key.
		log.Printf("encrypted plugin load skipped: %v", err)
	}

	err = wails.Run(&options.App{
		Title:  "WailsPlugSystem Encrypted Plugin Lab",
		Width:  1080,
		Height: 720,
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

func readKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 32 {
		return data, nil
	}
	encoded := strings.TrimSpace(string(data))
	if len(encoded) != 64 {
		return nil, fmt.Errorf("%s must contain 32 raw bytes or 64 hexadecimal characters", path)
	}
	key, err := hex.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("%s is not valid hexadecimal AES-256 key", path)
	}
	return key, nil
}
