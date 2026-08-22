package devwatch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Change describes files changed since the previous snapshot.
type Change struct {
	Added    []string
	Modified []string
	Removed  []string
}

func (c Change) Empty() bool { return len(c.Added) == 0 && len(c.Modified) == 0 && len(c.Removed) == 0 }

// Options configures a polling watcher. Polling is intentional: it works
// consistently on Linux, Windows, and macOS without native watcher bindings.
type Options struct {
	Directory string
	Recursive bool
	Interval  time.Duration
	// Extensions optionally restricts files by lower-case extension, e.g. []string{".plugs"}.
	Extensions []string
	RunInitial bool
	OnChange   func(context.Context, Change) error
}

// Watch monitors a directory until ctx is cancelled or OnChange returns an error.
func Watch(ctx context.Context, options Options) error {
	if options.Directory == "" {
		return fmt.Errorf("devwatch: directory is required")
	}
	if options.OnChange == nil {
		return fmt.Errorf("devwatch: OnChange is required")
	}
	if options.Interval <= 0 {
		options.Interval = 500 * time.Millisecond
	}
	if options.Interval < 50*time.Millisecond {
		options.Interval = 50 * time.Millisecond
	}
	for index := range options.Extensions {
		options.Extensions[index] = strings.ToLower(options.Extensions[index])
		if !strings.HasPrefix(options.Extensions[index], ".") {
			options.Extensions[index] = "." + options.Extensions[index]
		}
	}
	previous, err := snapshot(options)
	if err != nil {
		return err
	}
	if options.RunInitial {
		if err := options.OnChange(ctx, Change{}); err != nil {
			return err
		}
	}
	ticker := time.NewTicker(options.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			next, err := snapshot(options)
			if err != nil {
				return err
			}
			change := diff(previous, next)
			if change.Empty() {
				continue
			}
			if err := options.OnChange(ctx, change); err != nil {
				return err
			}
			previous = next
		}
	}
}

type fingerprint struct {
	Size    int64
	ModTime int64
}

func snapshot(options Options) (map[string]fingerprint, error) {
	root, err := os.Stat(options.Directory)
	if err != nil {
		return nil, fmt.Errorf("devwatch: stat directory: %w", err)
	}
	if !root.IsDir() {
		return nil, fmt.Errorf("devwatch: path is not a directory: %s", options.Directory)
	}
	files := map[string]fingerprint{}
	walkErr := filepath.Walk(options.Directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if !options.Recursive && path != options.Directory {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() || !matchesExtension(path, options.Extensions) {
			return nil
		}
		relative, err := filepath.Rel(options.Directory, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = fingerprint{Size: info.Size(), ModTime: info.ModTime().UnixNano()}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("devwatch: scan directory: %w", walkErr)
	}
	return files, nil
}

func matchesExtension(path string, extensions []string) bool {
	if len(extensions) == 0 {
		return true
	}
	extension := strings.ToLower(filepath.Ext(path))
	for _, allowed := range extensions {
		if extension == allowed {
			return true
		}
	}
	return false
}

func diff(previous, next map[string]fingerprint) Change {
	change := Change{}
	for path, current := range next {
		old, ok := previous[path]
		if !ok {
			change.Added = append(change.Added, path)
		} else if old != current {
			change.Modified = append(change.Modified, path)
		}
	}
	for path := range previous {
		if _, ok := next[path]; !ok {
			change.Removed = append(change.Removed, path)
		}
	}
	sort.Strings(change.Added)
	sort.Strings(change.Modified)
	sort.Strings(change.Removed)
	return change
}
