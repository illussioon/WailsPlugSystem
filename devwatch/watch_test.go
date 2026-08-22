package devwatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiffIsDeterministic(t *testing.T) {
	previous := map[string]fingerprint{
		"removed.plugs": {Size: 1, ModTime: 1},
		"changed.plugs": {Size: 1, ModTime: 1},
	}
	next := map[string]fingerprint{
		"added.plugs":   {Size: 1, ModTime: 1},
		"changed.plugs": {Size: 2, ModTime: 2},
	}
	change := diff(previous, next)
	if len(change.Added) != 1 || change.Added[0] != "added.plugs" || len(change.Modified) != 1 || change.Modified[0] != "changed.plugs" || len(change.Removed) != 1 || change.Removed[0] != "removed.plugs" {
		t.Fatalf("unexpected diff: %#v", change)
	}
}

func TestWatchReportsFileChanges(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "plugin.plugs")
	if err := os.WriteFile(path, []byte("one"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := make(chan Change, 1)
	errors := make(chan error, 1)
	go func() {
		errors <- Watch(ctx, Options{
			Directory:  directory,
			Interval:   50 * time.Millisecond,
			Extensions: []string{".plugs"},
			OnChange: func(_ context.Context, change Change) error {
				changes <- change
				cancel()
				return nil
			},
		})
	}()
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(path, []byte("two"), 0644); err != nil {
		t.Fatal(err)
	}
	select {
	case change := <-changes:
		if len(change.Modified) != 1 || change.Modified[0] != "plugin.plugs" {
			t.Fatalf("unexpected change: %#v", change)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not report modified file")
	}
	select {
	case err := <-errors:
		if err != context.Canceled {
			t.Fatalf("unexpected watcher error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop")
	}
}
