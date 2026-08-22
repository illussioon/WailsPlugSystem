package plugin_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/plugin"
)

func TestDefinitionEncryptBuildsEncryptedPackage(t *testing.T) {
	key := bytes.Repeat([]byte{0x5A}, 32)
	path := filepath.Join(t.TempDir(), "encrypted.plugs")
	if _, err := plugin.New("sdk.encrypted", "SDK Encrypted", "1.0.0").
		JavaScript().
		AddJS("entry", "assets/entry.js", []byte("window.__sdkSecret = 'encrypted';")).
		Encrypt(key).
		Build(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("window.__sdkSecret")) {
		t.Fatal("SDK encrypted archive contains plaintext JavaScript")
	}
	item, err := wailsplugs.OpenPackage(path, wailsplugs.PackageOptions{DecryptionKey: key})
	if err != nil {
		t.Fatal(err)
	}
	if item.Manifest.Encryption != wailsplugs.EncryptionAES256GCM || len(item.Patches) != 1 {
		t.Fatalf("unexpected encrypted SDK package: %#v", item)
	}
}
