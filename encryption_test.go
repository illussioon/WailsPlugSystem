package wailsplugs_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/pack"
)

func TestEncryptDecryptPayload(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	ciphertext, err := wailsplugs.EncryptPayload([]byte("private plugin payload"), key)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := wailsplugs.DecryptPayload(ciphertext, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(plaintext) != "private plugin payload" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
	wrongKey := bytes.Repeat([]byte{0x24}, 32)
	if _, err := wailsplugs.DecryptPayload(ciphertext, wrongKey); !errors.Is(err, wailsplugs.ErrDecryption) {
		t.Fatalf("wrong key error = %v, want ErrDecryption", err)
	}
	if _, err := wailsplugs.EncryptPayload([]byte("x"), []byte("short")); !errors.Is(err, wailsplugs.ErrDecryption) {
		t.Fatalf("short encryption key error = %v, want ErrDecryption", err)
	}
}

func TestEncryptedPackageHidesPayloadAndRequiresKey(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	if err := os.MkdirAll(assets, 0755); err != nil {
		t.Fatal(err)
	}
	secretJS := []byte("window.__privatePluginLogic = 'do not expose this source';")
	if err := os.WriteFile(filepath.Join(assets, "main.js"), secretJS, 0644); err != nil {
		t.Fatal(err)
	}
	manifest := wailsplugs.Manifest{FormatVersion: wailsplugs.FormatVersion, ID: "encrypted", Name: "Encrypted", Version: "1.0.0", APIVersion: wailsplugs.APIVersion, Permissions: []wailsplugs.Permission{wailsplugs.PermissionJS}}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifestBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "patches.json"), []byte(`[{"id":"main","kind":"inject_js","asset":"assets/main.js"}]`), 0644); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{0xA7}, 32)
	path := filepath.Join(root, "encrypted.plugs")
	if _, err := pack.Build(pack.Options{InputDir: root, Output: path, EncryptionKey: key}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, secretJS) {
		t.Fatal("encrypted archive contains plaintext asset source")
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) != 2 {
		t.Fatalf("encrypted archive has %d entries, want manifest.json and payload.bin", len(archive.File))
	}
	for _, file := range archive.File {
		if file.Name != "manifest.json" && file.Name != "payload.bin" {
			t.Fatalf("encrypted archive exposes unexpected entry %q", file.Name)
		}
	}
	if _, err := wailsplugs.OpenPackage(path, wailsplugs.PackageOptions{}); !errors.Is(err, wailsplugs.ErrDecryption) {
		t.Fatalf("missing key error = %v, want ErrDecryption", err)
	}
	wrongKey := bytes.Repeat([]byte{0xB8}, 32)
	if _, err := wailsplugs.OpenPackage(path, wailsplugs.PackageOptions{DecryptionKey: wrongKey}); !errors.Is(err, wailsplugs.ErrDecryption) {
		t.Fatalf("wrong key error = %v, want ErrDecryption", err)
	}
	item, err := wailsplugs.OpenPackage(path, wailsplugs.PackageOptions{DecryptionKey: key})
	if err != nil {
		t.Fatal(err)
	}
	if item.Manifest.Encryption != wailsplugs.EncryptionAES256GCM || string(item.Assets["assets/main.js"]) != string(secretJS) {
		t.Fatalf("encrypted package did not round-trip: %#v", item)
	}

	providerCalled := false
	provided, err := wailsplugs.OpenPackage(path, wailsplugs.PackageOptions{DecryptionKeyProvider: func(manifest wailsplugs.Manifest) ([]byte, error) {
		providerCalled = manifest.ID == "encrypted" && manifest.Encryption == wailsplugs.EncryptionAES256GCM
		return key, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !providerCalled || len(provided.Patches) != 1 {
		t.Fatalf("key provider package result invalid: called=%v package=%#v", providerCalled, provided)
	}
}
