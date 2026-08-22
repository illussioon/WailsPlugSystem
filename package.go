package wailsplugs

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type PackageOptions struct {
	MaxArchiveBytes int64
	MaxFileBytes    int64
	// DecryptionKey is a 32-byte AES-256 key for encrypted packages.
	DecryptionKey []byte
	// DecryptionKeyProvider can obtain a package key from a license service or
	// OS secure storage after the readable manifest has been parsed.
	DecryptionKeyProvider DecryptionKeyProvider
}

func OpenPackage(path string, options PackageOptions) (Package, error) {
	if options.MaxArchiveBytes <= 0 {
		options.MaxArchiveBytes = 32 << 20
	}
	if options.MaxFileBytes <= 0 {
		options.MaxFileBytes = 8 << 20
	}
	info, err := os.Stat(path)
	if err != nil {
		return Package{}, err
	}
	if info.Size() > options.MaxArchiveBytes {
		return Package{}, fmt.Errorf("%w: %s", ErrPackageTooLarge, path)
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return Package{}, fmt.Errorf("%w: open zip: %v", ErrUnsafeArchive, err)
	}
	defer archive.Close()

	var manifest Manifest
	var manifestFound, patchesFound, payloadFound bool
	var patchBytes, payloadBytes []byte
	assets := map[string][]byte{}
	for _, file := range archive.File {
		name, err := safeArchivePath(file.Name)
		if err != nil || name != file.Name || file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return Package{}, fmt.Errorf("%w: %q", ErrUnsafeArchive, file.Name)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		limit := options.MaxFileBytes
		if name == "payload.bin" {
			limit = options.MaxArchiveBytes
		}
		data, err := readArchiveFile(file, limit, name)
		if err != nil {
			return Package{}, err
		}
		switch name {
		case "manifest.json":
			if manifestFound {
				return Package{}, fmt.Errorf("%w: duplicate manifest", ErrUnsafeArchive)
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return Package{}, fmt.Errorf("%w: manifest: %v", ErrInvalidManifest, err)
			}
			manifestFound = true
		case "patches.json":
			if patchesFound {
				return Package{}, fmt.Errorf("%w: duplicate patches", ErrUnsafeArchive)
			}
			patchBytes, patchesFound = data, true
		case "payload.bin":
			if payloadFound {
				return Package{}, fmt.Errorf("%w: duplicate encrypted payload", ErrUnsafeArchive)
			}
			payloadBytes, payloadFound = data, true
		default:
			if !validAssetPath(name) {
				return Package{}, fmt.Errorf("%w: unexpected file %q", ErrUnsafeArchive, name)
			}
			assets[name] = data
		}
	}
	if !manifestFound {
		return Package{}, fmt.Errorf("%w: manifest.json is required", ErrInvalidManifest)
	}
	if err := manifest.Validate(); err != nil {
		return Package{}, err
	}

	if manifest.Encryption == EncryptionAES256GCM {
		if !payloadFound || patchesFound || len(assets) > 0 {
			return Package{}, fmt.Errorf("%w: encrypted package must contain manifest.json and payload.bin only", ErrUnsafeArchive)
		}
		key, err := packageDecryptionKey(manifest, options)
		if err != nil {
			return Package{}, err
		}
		plainPayload, err := DecryptPayload(payloadBytes, key)
		if err != nil {
			return Package{}, fmt.Errorf("%w: %s", err, manifest.ID)
		}
		patchBytes, assets, err = parsePayload(plainPayload, options)
		if err != nil {
			return Package{}, err
		}
		patchesFound = true
	} else {
		if payloadFound {
			return Package{}, fmt.Errorf("%w: payload.bin requires encryption=%q", ErrInvalidManifest, EncryptionAES256GCM)
		}
		if !patchesFound {
			patchBytes = []byte("[]")
		}
	}

	var patches []Patch
	if err := json.Unmarshal(patchBytes, &patches); err != nil {
		return Package{}, fmt.Errorf("%w: patches.json: %v", ErrInvalidManifest, err)
	}
	if err := validatePatches(patches); err != nil {
		return Package{}, err
	}
	if err := verifyAssets(manifest, assets); err != nil {
		return Package{}, err
	}
	digest, err := fileSHA256(path)
	if err != nil {
		return Package{}, err
	}
	return Package{Manifest: manifest, Patches: patches, Assets: assets, Path: path, SHA256: digest}, nil
}

func packageDecryptionKey(manifest Manifest, options PackageOptions) ([]byte, error) {
	key := options.DecryptionKey
	if len(key) == 0 && options.DecryptionKeyProvider != nil {
		var err error
		key, err = options.DecryptionKeyProvider(manifest)
		if err != nil {
			return nil, fmt.Errorf("%w: key provider: %v", ErrDecryption, err)
		}
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: a 32-byte key is required for %s", ErrDecryption, manifest.ID)
	}
	return key, nil
}

func parsePayload(payload []byte, options PackageOptions) ([]byte, map[string][]byte, error) {
	archive, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: encrypted payload is not a valid container", ErrDecryption)
	}
	var patches []byte
	var patchesFound bool
	assets := map[string][]byte{}
	for _, file := range archive.File {
		name, err := safeArchivePath(file.Name)
		if err != nil || name != file.Name || file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, nil, fmt.Errorf("%w: unsafe encrypted payload path %q", ErrUnsafeArchive, file.Name)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		data, err := readArchiveFile(file, options.MaxFileBytes, name)
		if err != nil {
			return nil, nil, err
		}
		switch name {
		case "patches.json":
			if patchesFound {
				return nil, nil, fmt.Errorf("%w: duplicate encrypted patches", ErrUnsafeArchive)
			}
			patches, patchesFound = data, true
		default:
			if !validAssetPath(name) {
				return nil, nil, fmt.Errorf("%w: unexpected encrypted payload file %q", ErrUnsafeArchive, name)
			}
			assets[name] = data
		}
	}
	if !patchesFound {
		patches = []byte("[]")
	}
	return patches, assets, nil
}

func readArchiveFile(file *zip.File, limit int64, name string) ([]byte, error) {
	if file.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("%w: %s", ErrPackageTooLarge, name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(reader, limit+1))
	closeErr := reader.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: %s", ErrPackageTooLarge, name)
	}
	return data, nil
}

func validatePatches(patches []Patch) error {
	seen := map[string]bool{}
	for index, patch := range patches {
		if patch.ID == "" {
			patch.ID = fmt.Sprintf("patch-%d", index)
		}
		if seen[patch.ID] {
			return fmt.Errorf("%w: duplicate patch id %q", ErrInvalidManifest, patch.ID)
		}
		seen[patch.ID] = true
		switch patch.Kind {
		case PatchSetText, PatchSetAttr, PatchRemove, PatchReplaceHTML, PatchAppendHTML, PatchPrependHTML, PatchAddClass, PatchRemoveClass, PatchInjectCSS, PatchInjectJS:
		default:
			return fmt.Errorf("%w: unknown patch kind %q", ErrInvalidManifest, patch.Kind)
		}
		if patch.External && patch.Kind != PatchInjectCSS && patch.Kind != PatchInjectJS {
			return fmt.Errorf("%w: patch %q external mode requires css/js injection", ErrInvalidManifest, patch.ID)
		}
		if patch.Kind != PatchInjectCSS && patch.Kind != PatchInjectJS && patch.Selector == "" {
			return fmt.Errorf("%w: patch %q needs selector", ErrInvalidManifest, patch.ID)
		}
		if patch.Kind == PatchSetAttr && patch.Attribute == "" {
			return fmt.Errorf("%w: patch %q needs attribute", ErrInvalidManifest, patch.ID)
		}
		if patch.Kind == PatchInjectCSS || patch.Kind == PatchInjectJS {
			if patch.Asset == "" || !validAssetPath(patch.Asset) {
				return fmt.Errorf("%w: patch %q has invalid asset", ErrInvalidManifest, patch.ID)
			}
		}
		if patch.ConflictKey == "" {
			object := patch.Selector
			if object == "" {
				object = patch.Asset
			}
			patch.ConflictKey = string(patch.Kind) + ":" + object + ":" + patch.Attribute
		}
	}
	return nil
}

func verifyAssets(manifest Manifest, assets map[string][]byte) error {
	if len(manifest.Files) != len(assets) {
		return fmt.Errorf("%w: manifest file allowlist does not match archive", ErrIntegrity)
	}
	for _, ref := range manifest.Files {
		data, ok := assets[ref.Path]
		if !ok {
			return fmt.Errorf("%w: missing %s", ErrIntegrity, ref.Path)
		}
		sum := sha256.Sum256(data)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), ref.SHA256) {
			return fmt.Errorf("%w: %s", ErrIntegrity, ref.Path)
		}
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func loadPackages(ctx context.Context, paths []string, options PackageOptions) ([]Package, error) {
	sort.Strings(paths)
	packages := make([]Package, 0, len(paths))
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		item, err := OpenPackage(path, options)
		if err != nil {
			return nil, err
		}
		packages = append(packages, item)
	}
	return packages, nil
}
