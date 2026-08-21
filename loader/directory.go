package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
)

type Directory struct {
	Dir            string
	PackageOptions wailsplugs.PackageOptions
	Recursive      bool
}

func (d Directory) Load(ctx context.Context) ([]wailsplugs.Package, error) {
	var paths []string
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path != d.Dir && !d.Recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(info.Name()), ".plugs") {
			paths = append(paths, path)
		}
		return nil
	}
	if d.Recursive {
		if err := filepath.Walk(d.Dir, walk); err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(d.Dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".plugs") {
				continue
			}
			paths = append(paths, filepath.Join(d.Dir, entry.Name()))
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return []wailsplugs.Package{}, nil
	}
	packages := make([]wailsplugs.Package, 0, len(paths))
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		item, err := wailsplugs.OpenPackage(path, d.PackageOptions)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", path, err)
		}
		packages = append(packages, item)
	}
	return packages, nil
}

type SHA256Allowlist struct {
	Dir            string
	SHA256         []string
	Recursive      bool
	PackageOptions wailsplugs.PackageOptions
}

func (a SHA256Allowlist) Load(ctx context.Context) ([]wailsplugs.Package, error) {
	wanted := map[string]bool{}
	for _, item := range a.SHA256 {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if len(normalized) != sha256.Size*2 {
			return nil, fmt.Errorf("invalid sha256 %q", item)
		}
		if _, err := hex.DecodeString(normalized); err != nil {
			return nil, fmt.Errorf("invalid sha256 %q: %w", item, err)
		}
		wanted[normalized] = true
	}
	packages, err := (Directory{Dir: a.Dir, PackageOptions: a.PackageOptions, Recursive: a.Recursive}).Load(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]wailsplugs.Package, 0, len(packages))
	for _, item := range packages {
		if wanted[item.SHA256] {
			result = append(result, item)
		}
	}
	return result, nil
}
