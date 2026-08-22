package pack

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
)

type Options struct {
	InputDir string
	Output   string
	// EncryptionKey enables AES-256-GCM encryption of patches and assets.
	// The key must be exactly 32 bytes; manifest.json remains readable.
	EncryptionKey []byte
}

func Build(options Options) (string, error) {
	if options.InputDir == "" || options.Output == "" {
		return "", fmt.Errorf("pack: input and output are required")
	}
	manifestPath := filepath.Join(options.InputDir, "manifest.json")
	rawManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}
	var manifest wailsplugs.Manifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return "", err
	}
	manifest.Files = nil
	var assetPaths []string
	assetsDir := filepath.Join(options.InputDir, "assets")
	if _, err := os.Stat(assetsDir); err == nil {
		err = filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(options.InputDir, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if !strings.HasPrefix(relative, "assets/") || !wailsplugs.ValidAssetForPack(relative) {
				return fmt.Errorf("pack: invalid asset %q", relative)
			}
			assetPaths = append(assetPaths, relative)
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	sort.Strings(assetPaths)
	for _, relative := range assetPaths {
		data, err := os.ReadFile(filepath.Join(options.InputDir, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		manifest.Files = append(manifest.Files, wailsplugs.FileRef{Path: relative, SHA256: hex.EncodeToString(sum[:]), Kind: assetKind(relative)})
	}
	patchPath := filepath.Join(options.InputDir, "patches.json")
	patchBytes, err := os.ReadFile(patchPath)
	if os.IsNotExist(err) {
		patchBytes = []byte("[]\n")
	} else if err != nil {
		return "", err
	}
	if len(options.EncryptionKey) > 0 {
		if len(options.EncryptionKey) != 32 {
			return "", fmt.Errorf("pack: AES-256 encryption key must be exactly 32 bytes")
		}
		manifest.Encryption = wailsplugs.EncryptionAES256GCM
	} else if manifest.Encryption != "" {
		return "", fmt.Errorf("pack: manifest requests encryption but EncryptionKey is missing")
	}
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Dir(options.Output)); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(options.Output), 0755); err != nil {
			return "", err
		}
	}
	temp, err := os.CreateTemp(filepath.Dir(options.Output), ".wailsplugs-*.tmp")
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	cleanup := func() { temp.Close(); os.Remove(tempPath) }
	defer cleanup()
	archive := zip.NewWriter(temp)
	if err := writeZipEntry(archive, "manifest.json", manifestBytes); err != nil {
		return "", err
	}
	if len(options.EncryptionKey) > 0 {
		payload, err := buildPayload(options.InputDir, patchBytes, assetPaths)
		if err != nil {
			return "", err
		}
		envelope, err := wailsplugs.EncryptPayload(payload, options.EncryptionKey)
		if err != nil {
			return "", err
		}
		if err := writeZipEntry(archive, "payload.bin", envelope); err != nil {
			return "", err
		}
	} else {
		if err := writeZipEntry(archive, "patches.json", patchBytes); err != nil {
			return "", err
		}
		for _, relative := range assetPaths {
			data, err := os.ReadFile(filepath.Join(options.InputDir, filepath.FromSlash(relative)))
			if err != nil {
				return "", err
			}
			if err := writeZipEntry(archive, relative, data); err != nil {
				return "", err
			}
		}
	}
	if err := archive.Close(); err != nil {
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, options.Output); err != nil {
		return "", err
	}
	return options.Output, nil
}

func buildPayload(inputDir string, patchBytes []byte, assetPaths []string) ([]byte, error) {
	var payload bytes.Buffer
	archive := zip.NewWriter(&payload)
	if err := writeZipEntry(archive, "patches.json", patchBytes); err != nil {
		return nil, err
	}
	for _, relative := range assetPaths {
		data, err := os.ReadFile(filepath.Join(inputDir, filepath.FromSlash(relative)))
		if err != nil {
			return nil, err
		}
		if err := writeZipEntry(archive, relative, data); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return payload.Bytes(), nil
}

func writeZipEntry(archive *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0644)
	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, bytes.NewReader(data))
	return err
}

func assetKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".css":
		return "css"
	case ".js", ".mjs":
		return "js"
	default:
		return "asset"
	}
}
