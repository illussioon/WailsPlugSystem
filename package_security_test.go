package wailsplugs

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestRejectsUnsafeArchivePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unsafe.plugs")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entries := map[string]string{
		"manifest.json":  `{"format_version":1,"id":"unsafe","name":"unsafe","version":"1.0.0","api_version":"v1"}`,
		"../outside.txt": "must not escape",
	}
	for name, value := range entries {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenPackage(path, PackageOptions{}); err == nil {
		t.Fatal("unsafe archive path was accepted")
	}
}
