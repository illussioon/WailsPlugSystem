package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/illussioon/WailsPlugSystem/plugin"
)

func main() {
	output := flag.String("output", "../../plugins/private-encrypted.plugs", "output .plugs archive")
	keyFile := flag.String("key-file", "../../demo.key", "raw 32-byte or 64-hex-byte AES-256 key file")
	flag.Parse()

	key, err := readKey(*keyFile)
	if err != nil {
		log.Fatal(err)
	}
	definition := plugin.New("private-encrypted", "Private Encrypted Demo", "1.0.0").
		Priority(100).
		HTML().
		CSSPermission().
		HostCSS().
		JavaScript().
		OnLoad("private encrypted plugin loaded after authenticated decryption").
		OnUnload("private encrypted plugin unloaded").
		AppendHTML("private-panel", "#plugin-tester-slot", `
<article id="private-panel" class="private-panel">
  <div class="private-badge">AES-256-GCM payload</div>
  <h2>Encrypted plugin is active</h2>
  <p>This complete panel came from an encrypted <code>.plugs</code> archive. The outer ZIP exposes only metadata and <code>payload.bin</code>.</p>
  <div class="private-result" id="private-result">Waiting for encrypted asset route...</div>
</article>`, plugin.Optional()).
		AddCSSExternal("private-css", "assets/private.css", []byte(`
#private-panel { margin-top: 20px; }
.private-panel { padding: 24px; border-radius: 18px; border: 1px solid rgba(167, 139, 250, .46); background: linear-gradient(135deg, rgba(76, 29, 149, .34), rgba(15, 23, 42, .94)); }
.private-badge { display: inline-block; color: #ddd6fe; font-size: 11px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; }
.private-panel h2 { margin: 8px 0; color: #f5f3ff; }
.private-panel p { color: #ddd6fe; line-height: 1.6; }
.private-result { margin-top: 16px; padding: 12px; border-radius: 10px; color: #ede9fe; background: rgba(124, 58, 237, .22); }
`)).
		AddJSExternal("private-js", "assets/private.js", []byte(`
const result = document.querySelector("#private-result");
async function verifyEncryptedAsset() {
  const response = await fetch("/__wailsplugs/assets/private-encrypted/private-data.json");
  if (!response.ok) throw new Error("encrypted asset route returned " + response.status);
  const data = await response.json();
  result.textContent = data.message;
  window.Wails?.print?.console?.("private encrypted plugin verified", data);
  window.Wails?.print?.console?.browser?.("encrypted plugin asset route is active");
}
verifyEncryptedAsset().catch((error) => { result.textContent = "Asset check failed: " + error.message; });
`)).
		Asset("assets/private-data.json", []byte(`{"message":"Encrypted static asset decrypted and served successfully."}`)).
		Encrypt(key)

	if _, err := definition.Build(*output); err != nil {
		log.Fatal(err)
	}
	log.Printf("built encrypted plugin: %s", *output)
}

func readKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}
	if len(data) == 32 {
		return data, nil
	}
	encoded := strings.TrimSpace(string(data))
	if len(encoded) != 64 {
		return nil, fmt.Errorf("key must be raw 32 bytes or 64 hexadecimal characters")
	}
	key, err := hex.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("key is not valid hexadecimal")
	}
	return key, nil
}
