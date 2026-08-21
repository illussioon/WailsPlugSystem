package wailsplugs

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var identifierRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
var versionRE = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._+~-]{0,63}$`)

func (m Manifest) Validate() error {
	if m.FormatVersion != FormatVersion {
		return fmt.Errorf("%w: unsupported format_version %d", ErrInvalidManifest, m.FormatVersion)
	}
	if !identifierRE.MatchString(m.ID) {
		return fmt.Errorf("%w: id %q must match %s", ErrInvalidManifest, m.ID, identifierRE.String())
	}
	if m.Name == "" || len(m.Name) > 200 {
		return fmt.Errorf("%w: name is empty or too long", ErrInvalidManifest)
	}
	if !versionRE.MatchString(m.Version) {
		return fmt.Errorf("%w: invalid version %q", ErrInvalidManifest, m.Version)
	}
	if m.APIVersion != APIVersion {
		return fmt.Errorf("%w: unsupported api_version %q", ErrInvalidManifest, m.APIVersion)
	}
	if len(m.Lifecycle.Load) > 4096 || len(m.Lifecycle.Unload) > 4096 {
		return fmt.Errorf("%w: lifecycle message is too long", ErrInvalidManifest)
	}
	seenPerms := map[Permission]bool{}
	for _, permission := range m.Permissions {
		switch permission {
		case PermissionHTML, PermissionCSS, PermissionJS, PermissionReplaceRoot:
		default:
			return fmt.Errorf("%w: unknown permission %q", ErrInvalidManifest, permission)
		}
		if seenPerms[permission] {
			return fmt.Errorf("%w: duplicate permission %q", ErrInvalidManifest, permission)
		}
		seenPerms[permission] = true
	}
	seenDeps := map[string]bool{}
	for _, dependency := range m.Dependencies {
		if !identifierRE.MatchString(dependency.ID) || dependency.ID == m.ID {
			return fmt.Errorf("%w: invalid dependency %q", ErrInvalidManifest, dependency.ID)
		}
		if seenDeps[dependency.ID] {
			return fmt.Errorf("%w: duplicate dependency %q", ErrInvalidManifest, dependency.ID)
		}
		seenDeps[dependency.ID] = true
		if dependency.Version != "" && !versionRE.MatchString(dependency.Version) {
			return fmt.Errorf("%w: invalid dependency version %q", ErrInvalidManifest, dependency.Version)
		}
	}
	seenFiles := map[string]bool{}
	for _, file := range m.Files {
		normalized, err := safeArchivePath(file.Path)
		if err != nil || normalized != file.Path || !strings.HasPrefix(file.Path, "assets/") {
			return fmt.Errorf("%w: asset path %q is not allowed", ErrInvalidManifest, file.Path)
		}
		if seenFiles[file.Path] {
			return fmt.Errorf("%w: duplicate asset %q", ErrInvalidManifest, file.Path)
		}
		seenFiles[file.Path] = true
		if len(file.SHA256) != 64 || strings.Trim(file.SHA256, "0123456789abcdefABCDEF") != "" {
			return fmt.Errorf("%w: invalid sha256 for %q", ErrInvalidManifest, file.Path)
		}
	}
	return nil
}

func (m Manifest) HasPermission(permission Permission) bool {
	for _, item := range m.Permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func safeArchivePath(name string) (string, error) {
	if name == "" || strings.ContainsRune(name, 0) || strings.HasPrefix(name, "/") || strings.Contains(name, `\`) {
		return "", ErrUnsafeArchive
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", ErrUnsafeArchive
	}
	return cleaned, nil
}

func validAssetPath(name string) bool {
	normalized, err := safeArchivePath(name)
	return err == nil && normalized == name && strings.HasPrefix(name, "assets/")
}

// ValidAssetForPack exposes the same path policy to the pack builder.
func ValidAssetForPack(name string) bool {
	return validAssetPath(name)
}
