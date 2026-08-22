// Package client provides the small, stable host-application API for WailsPlugSystem.
package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/devwatch"
	"github.com/illussioon/WailsPlugSystem/integration/httpmiddleware"
	"github.com/illussioon/WailsPlugSystem/loader"
)

// Options configures a host-side Client.
type Options struct {
	// Loader is used when set and takes precedence over Directory and SHA256.
	Loader wailsplugs.PackageLoader
	// Directory loads .plugs files from this directory.
	Directory string
	// Recursive controls recursive Directory scanning.
	Recursive bool
	// SHA256 restricts loading to these archive hashes.
	SHA256 []string

	PackageOptions wailsplugs.PackageOptions
	// DecryptionKey is an optional 32-byte AES-256 key for encrypted packages.
	DecryptionKey []byte
	// DecryptionKeyProvider obtains encrypted package keys at load time.
	DecryptionKeyProvider wailsplugs.DecryptionKeyProvider
	// HostLogger receives messages sent by Wails.print.console.
	HostLogger         func(wailsplugs.ConsoleMessage)
	AllowJavaScript    bool
	AllowRootReplace   bool
	StrictDependencies bool
	MaxPlugins         int
}

// Client is the recommended host-side facade over the WailsPlugSystem runtime.
type Client struct {
	manager   *wailsplugs.Manager
	directory string
	recursive bool
}

// New creates a Client. Exactly one loading strategy is selected in this order:
// custom Loader, SHA256 allowlist, or Directory.
func New(options Options) (*Client, error) {
	packageOptions := options.PackageOptions
	if len(options.DecryptionKey) > 0 {
		packageOptions.DecryptionKey = append([]byte(nil), options.DecryptionKey...)
	}
	if options.DecryptionKeyProvider != nil {
		packageOptions.DecryptionKeyProvider = options.DecryptionKeyProvider
	}
	packageLoader := options.Loader
	if packageLoader == nil && len(options.SHA256) > 0 {
		packageLoader = loader.SHA256Allowlist{
			Dir:            options.Directory,
			SHA256:         options.SHA256,
			Recursive:      options.Recursive,
			PackageOptions: packageOptions,
		}
	}
	if packageLoader == nil && options.Directory != "" {
		packageLoader = loader.Directory{
			Dir:            options.Directory,
			Recursive:      options.Recursive,
			PackageOptions: packageOptions,
		}
	}
	if packageLoader == nil {
		return nil, fmt.Errorf("wailsplugs/client: configure Loader, Directory, or SHA256")
	}
	return &Client{
		manager: wailsplugs.NewManager(wailsplugs.ManagerOptions{
			Loader:             packageLoader,
			HostLogger:         options.HostLogger,
			AllowJavaScript:    options.AllowJavaScript,
			AllowRootReplace:   options.AllowRootReplace,
			StrictDependencies: options.StrictDependencies,
			MaxPlugins:         options.MaxPlugins,
		}),
		directory: options.Directory,
		recursive: options.Recursive,
	}, nil
}

// Reload atomically loads the current plugin snapshot.
func (c *Client) Reload(ctx context.Context) error {
	if c == nil || c.manager == nil {
		return fmt.Errorf("wailsplugs/client: nil client")
	}
	return c.manager.Reload(ctx)
}

// Render applies the active plugin snapshot to an HTML document.
func (c *Client) Render(source string) (wailsplugs.RenderResult, error) {
	if c == nil || c.manager == nil {
		return wailsplugs.RenderResult{}, fmt.Errorf("wailsplugs/client: nil client")
	}
	return c.manager.Render(source)
}

// Handler wraps a Wails-compatible asset handler and transforms HTML responses.
func (c *Client) Handler(next http.Handler) http.Handler {
	if c == nil {
		return next
	}
	return httpmiddleware.New(c.manager, next)
}

// Packages returns the currently active package snapshot.
func (c *Client) Packages() []wailsplugs.Package {
	if c == nil || c.manager == nil {
		return nil
	}
	return c.manager.Packages()
}

// WatchOptions configures development-time plugin hot reload.
type WatchOptions struct {
	Directory  string
	Recursive  bool
	Interval   time.Duration
	RunInitial bool
	OnReload   func(context.Context, devwatch.Change) error
}

// Watch monitors plugin files and reloads the client after each change. It is
// intended for development; production hosts should use an explicit reload flow.
func (c *Client) Watch(ctx context.Context, options WatchOptions) error {
	if c == nil || c.manager == nil {
		return fmt.Errorf("wailsplugs/client: nil client")
	}
	if options.Directory == "" {
		options.Directory = c.directory
		options.Recursive = c.recursive
	}
	if options.Directory == "" {
		return fmt.Errorf("wailsplugs/client: watch directory is required")
	}
	return devwatch.Watch(ctx, devwatch.Options{
		Directory:  options.Directory,
		Recursive:  options.Recursive,
		Interval:   options.Interval,
		Extensions: []string{".plugs"},
		RunInitial: options.RunInitial,
		OnChange: func(ctx context.Context, change devwatch.Change) error {
			if err := c.Reload(ctx); err != nil {
				return err
			}
			if options.OnReload != nil {
				return options.OnReload(ctx, change)
			}
			return nil
		},
	})
}

func directoryFromOptions(options Options) string {
	return options.Directory
}

// Manager exposes the advanced low-level runtime for integrations that need it.
func (c *Client) Manager() *wailsplugs.Manager {
	if c == nil {
		return nil
	}
	return c.manager
}
