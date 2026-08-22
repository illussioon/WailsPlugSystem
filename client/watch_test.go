package client_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/client"
	"github.com/illussioon/WailsPlugSystem/devwatch"
)

type watchLoader struct {
	packages []wailsplugs.Package
}

func (l *watchLoader) Load(context.Context) ([]wailsplugs.Package, error) {
	return append([]wailsplugs.Package(nil), l.packages...), nil
}

func TestClientWatchReloadsOnPluginFileChange(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "example.plugs")
	if err := os.WriteFile(path, []byte("one"), 0644); err != nil {
		t.Fatal(err)
	}
	loader := &watchLoader{packages: []wailsplugs.Package{{Manifest: wailsplugs.Manifest{
		FormatVersion: wailsplugs.FormatVersion,
		ID:            "watch.example",
		Name:          "Watch Example",
		Version:       "1.0.0",
		APIVersion:    wailsplugs.APIVersion,
	}}}}
	app, err := client.New(client.Options{Loader: loader, Directory: directory})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := make(chan devwatch.Change, 1)
	errors := make(chan error, 1)
	go func() {
		errors <- app.Watch(ctx, client.WatchOptions{
			Interval: 50 * time.Millisecond,
			OnReload: func(_ context.Context, change devwatch.Change) error {
				changes <- change
				cancel()
				return nil
			},
		})
	}()
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(path, []byte("two-plus-a-different-size"), 0644); err != nil {
		t.Fatal(err)
	}
	select {
	case change := <-changes:
		if len(change.Modified) != 1 || change.Modified[0] != "example.plugs" {
			t.Fatalf("unexpected watch change: %#v", change)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("client watch did not reload after plugin change")
	}
	select {
	case err := <-errors:
		if err != context.Canceled {
			t.Fatalf("unexpected watch error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("client watcher did not stop")
	}
}
