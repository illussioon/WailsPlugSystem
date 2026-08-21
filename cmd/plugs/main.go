package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/pack"
)

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
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: plugs <pack|verify|hash> [flags]")
}

func packCommand(args []string) {
	set := flag.NewFlagSet("pack", flag.ExitOnError)
	input := set.String("input", ".", "plugin source directory")
	output := set.String("output", "plugin.plugs", "output .plugs path")
	_ = set.Parse(args)
	path, err := pack.Build(pack.Options{InputDir: *input, Output: *output})
	if err != nil {
		fatal(err)
	}
	fmt.Println(path)
}

func verifyCommand(args []string) {
	set := flag.NewFlagSet("verify", flag.ExitOnError)
	path := set.String("file", "", ".plugs file")
	_ = set.Parse(args)
	if *path == "" {
		fatal(fmt.Errorf("-file is required"))
	}
	item, err := wailsplugs.OpenPackage(*path, wailsplugs.PackageOptions{})
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

func fileToHashHelp() string { return "file to hash" }

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
