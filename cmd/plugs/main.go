package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/devwatch"
	"github.com/illussioon/WailsPlugSystem/pack"
)

// templates contains the starter React/Vue projects shipped with the CLI.
//
//go:embed templates/react-vite templates/vue-vite
var templates embed.FS

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "pack":
		packCommand(os.Args[2:])
	case "verify":
		verifyCommand(os.Args[2:])
	case "hash":
		hashCommand(os.Args[2:])
	case "init":
		initCommand(os.Args[2:])
	case "watch":
		watchCommand(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: plugs <pack|verify|hash|init|watch> [flags]")
}

func packCommand(args []string) {
	set := flag.NewFlagSet("pack", flag.ExitOnError)
	input := set.String("input", ".", "plugin source directory")
	output := set.String("output", "plugin.plugs", "output .plugs path")
	keyFile := set.String("encrypt-key-file", "", "raw 32-byte or 64-hex-byte AES-256 key file")
	_ = set.Parse(args)
	var key []byte
	var err error
	if *keyFile != "" {
		key, err = readEncryptionKey(*keyFile)
		if err != nil {
			fatal(err)
		}
	}
	path, err := pack.Build(pack.Options{InputDir: *input, Output: *output, EncryptionKey: key})
	if err != nil {
		fatal(err)
	}
	fmt.Println(path)
}

func verifyCommand(args []string) {
	set := flag.NewFlagSet("verify", flag.ExitOnError)
	path := set.String("file", "", ".plugs file")
	keyFile := set.String("key-file", "", "raw 32-byte or 64-hex-byte AES-256 key file")
	_ = set.Parse(args)
	if *path == "" {
		fatal(fmt.Errorf("-file is required"))
	}
	var key []byte
	var err error
	if *keyFile != "" {
		key, err = readEncryptionKey(*keyFile)
		if err != nil {
			fatal(err)
		}
	}
	item, err := wailsplugs.OpenPackage(*path, wailsplugs.PackageOptions{DecryptionKey: key})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("ok: %s %s sha256=%s\n", item.Manifest.ID, item.Manifest.Version, item.SHA256)
}

func hashCommand(args []string) {
	set := flag.NewFlagSet("hash", flag.ExitOnError)
	path := set.String("file", "", fileToHashHelp())
	_ = set.Parse(args)
	if *path == "" {
		fatal(fmt.Errorf("-file is required"))
	}
	file, err := os.Open(*path)
	if err != nil {
		fatal(err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		fatal(err)
	}
	fmt.Println(hex.EncodeToString(hash.Sum(nil)))
}

func initCommand(args []string) {
	set := flag.NewFlagSet("init", flag.ExitOnError)
	templateName := set.String("template", "react-vite", "template: react-vite or vue-vite")
	output := set.String("output", "plugin", "destination directory")
	_ = set.Parse(args)
	if *templateName != "react-vite" && *templateName != "vue-vite" {
		fatal(fmt.Errorf("unknown template %q", *templateName))
	}
	if _, err := os.Stat(*output); err == nil {
		fatal(fmt.Errorf("destination already exists: %s", *output))
	} else if !os.IsNotExist(err) {
		fatal(err)
	}
	root := filepath.ToSlash(filepath.Join("templates", *templateName))
	if err := fs.WalkDir(templates, root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative := strings.TrimPrefix(path, root)
		relative = strings.TrimPrefix(relative, "/")
		if strings.HasSuffix(relative, ".template") {
			relative = strings.TrimSuffix(relative, ".template")
		}
		if relative == "" {
			return nil
		}
		target := filepath.Join(*output, filepath.FromSlash(relative))
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := fs.ReadFile(templates, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	}); err != nil {
		fatal(err)
	}
	fmt.Printf("initialized %s template in %s\n", *templateName, *output)
}

func watchCommand(args []string) {
	set := flag.NewFlagSet("watch", flag.ExitOnError)
	input := set.String("input", ".", "plugin source directory")
	output := set.String("output", "plugin.plugs", "output .plugs path")
	interval := set.Duration("interval", 500*time.Millisecond, "polling interval")
	_ = set.Parse(args)
	ctx := context.Background()
	err := devwatch.Watch(ctx, devwatch.Options{
		Directory:  *input,
		Recursive:  true,
		Interval:   *interval,
		Extensions: []string{".json", ".html", ".css", ".js", ".mjs", ".ts", ".tsx", ".vue"},
		RunInitial: true,
		OnChange: func(context.Context, devwatch.Change) error {
			path, err := pack.Build(pack.Options{InputDir: *input, Output: *output})
			if err != nil {
				return err
			}
			fmt.Println("rebuilt", path)
			return nil
		},
	})
	if err != nil && err != context.Canceled {
		fatal(err)
	}
}

func readEncryptionKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read encryption key: %w", err)
	}
	if len(data) == 32 {
		return data, nil
	}
	encoded := strings.TrimSpace(string(data))
	if len(encoded) != 64 {
		return nil, fmt.Errorf("encryption key must be raw 32 bytes or 64 hexadecimal characters")
	}
	key, err := hex.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("encryption key is not valid hexadecimal")
	}
	return key, nil
}

func fileToHashHelp() string { return "file to hash" }

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
