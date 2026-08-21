// Package plugin provides the authoring SDK for WailsPlugSystem plugins.
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/pack"
)

// Definition is a fluent plugin definition. A zero Definition is not valid;
// create one with New.
type Definition struct {
	manifest wailsplugs.Manifest
	patches  []wailsplugs.Patch
	assets   map[string][]byte
}

// New creates a plugin definition with the required manifest fields.
func New(id, name, version string) *Definition {
	return &Definition{
		manifest: wailsplugs.Manifest{
			FormatVersion: wailsplugs.FormatVersion,
			ID:            id,
			Name:          name,
			Version:       version,
			APIVersion:    wailsplugs.APIVersion,
		},
		assets: map[string][]byte{},
	}
}

// Priority sets the conflict priority. Higher values win.
func (d *Definition) Priority(value int) *Definition { d.manifest.Priority = value; return d }

// OnLoad configures the message emitted when this plugin becomes active.
// The runtime sends it to both host and browser consoles through the lifecycle bridge.
func (d *Definition) OnLoad(message string) *Definition {
	d.manifest.Lifecycle.Load = message
	return d
}

// OnUnload configures the message emitted when this plugin is removed or replaced.
func (d *Definition) OnUnload(message string) *Definition {
	d.manifest.Lifecycle.Unload = message
	return d
}

// Permission adds a manifest permission.
func (d *Definition) Permission(value wailsplugs.Permission) *Definition {
	for _, current := range d.manifest.Permissions {
		if current == value {
			return d
		}
	}
	d.manifest.Permissions = append(d.manifest.Permissions, value)
	return d
}

// HTML enables HTML patch operations.
func (d *Definition) HTML() *Definition { return d.Permission(wailsplugs.PermissionHTML) }

// CSS enables CSS asset injection.
func (d *Definition) CSSPermission() *Definition { return d.Permission(wailsplugs.PermissionCSS) }

// JavaScript enables JavaScript asset injection. The host must also set AllowJavaScript.
func (d *Definition) JavaScript() *Definition { return d.Permission(wailsplugs.PermissionJS) }

// ReplaceRoot enables replace_html operations targeting the html root.
func (d *Definition) ReplaceRoot() *Definition {
	return d.Permission(wailsplugs.PermissionReplaceRoot).HTML()
}

// DependsOn declares an exact dependency version when version is non-empty.
func (d *Definition) DependsOn(id, version string) *Definition {
	d.manifest.Dependencies = append(d.manifest.Dependencies, wailsplugs.Dependency{ID: id, Version: version})
	return d
}

// PatchOption customizes a patch created by a helper.
type PatchOption func(*wailsplugs.Patch)

// WithConflictKey makes the patch compete for an explicit shared resource.
func WithConflictKey(value string) PatchOption {
	return func(p *wailsplugs.Patch) { p.ConflictKey = value }
}

// Optional marks a patch as non-fatal when its selector or asset is unavailable.
func Optional() PatchOption { return func(p *wailsplugs.Patch) { p.Optional = true } }

func (d *Definition) add(id string, kind wailsplugs.PatchKind, selector, value string, options []PatchOption) *Definition {
	patch := wailsplugs.Patch{ID: id, Kind: kind, Selector: selector, Value: value}
	for _, option := range options {
		option(&patch)
	}
	d.patches = append(d.patches, patch)
	return d
}

// SetText replaces the text content of every matched element.
func (d *Definition) SetText(id, selector, value string, options ...PatchOption) *Definition {
	return d.add(id, wailsplugs.PatchSetText, selector, value, options)
}

// SetAttr sets an HTML attribute.
func (d *Definition) SetAttr(id, selector, attribute, value string, options ...PatchOption) *Definition {
	patch := wailsplugs.Patch{ID: id, Kind: wailsplugs.PatchSetAttr, Selector: selector, Attribute: attribute, Value: value}
	for _, option := range options {
		option(&patch)
	}
	d.patches = append(d.patches, patch)
	return d
}

// Remove removes matched elements.
func (d *Definition) Remove(id, selector string, options ...PatchOption) *Definition {
	return d.add(id, wailsplugs.PatchRemove, selector, "", options)
}

// ReplaceHTML replaces child markup after sanitizer processing.
func (d *Definition) ReplaceHTML(id, selector, value string, options ...PatchOption) *Definition {
	return d.add(id, wailsplugs.PatchReplaceHTML, selector, value, options)
}

// AppendHTML appends sanitized child markup.
func (d *Definition) AppendHTML(id, selector, value string, options ...PatchOption) *Definition {
	return d.add(id, wailsplugs.PatchAppendHTML, selector, value, options)
}

// PrependHTML prepends sanitized child markup.
func (d *Definition) PrependHTML(id, selector, value string, options ...PatchOption) *Definition {
	return d.add(id, wailsplugs.PatchPrependHTML, selector, value, options)
}

// AddClass adds a CSS class.
func (d *Definition) AddClass(id, selector, value string, options ...PatchOption) *Definition {
	return d.add(id, wailsplugs.PatchAddClass, selector, value, options)
}

// RemoveClass removes a CSS class.
func (d *Definition) RemoveClass(id, selector, value string, options ...PatchOption) *Definition {
	return d.add(id, wailsplugs.PatchRemoveClass, selector, value, options)
}

// Asset adds an arbitrary file under assets/.
func (d *Definition) Asset(path string, data []byte) *Definition {
	if d.assets == nil {
		d.assets = map[string][]byte{}
	}
	d.assets[normalizeAssetPath(path)] = append([]byte(nil), data...)
	return d
}

// CSS adds a CSS asset and an inject_css patch.
func (d *Definition) AddCSS(id, assetPath string, data []byte, options ...PatchOption) *Definition {
	d.CSSPermission().Asset(assetPath, data)
	patch := wailsplugs.Patch{ID: id, Kind: wailsplugs.PatchInjectCSS, Asset: normalizeAssetPath(assetPath)}
	for _, option := range options {
		option(&patch)
	}
	d.patches = append(d.patches, patch)
	return d
}

// JS adds a JavaScript asset and an inject_js patch. Host policy still applies.
func (d *Definition) AddJS(id, assetPath string, data []byte, options ...PatchOption) *Definition {
	d.JavaScript().Asset(assetPath, data)
	patch := wailsplugs.Patch{ID: id, Kind: wailsplugs.PatchInjectJS, Asset: normalizeAssetPath(assetPath)}
	for _, option := range options {
		option(&patch)
	}
	d.patches = append(d.patches, patch)
	return d
}

// Console adds a runtime call that sends a message to the host/Wails logger.
// The host receives it through ConsoleEndpoint and the configured HostLogger.
func (d *Definition) Console(id, message string, options ...PatchOption) *Definition {
	return d.AddJS(id, consoleAssetPath(id), []byte(fmt.Sprintf("Wails.print.console(%s);", jsonString(message))), options...)
}

// ConsoleBrowser adds a runtime call that writes a message to the native
// WebView/browser developer console. It also requires the host JavaScript policy.
func (d *Definition) ConsoleBrowser(id, message string, options ...PatchOption) *Definition {
	return d.AddJS(id, consoleAssetPath(id+"-browser"), []byte(fmt.Sprintf("Wails.print.console.browser(%s);", jsonString(message))), options...)
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func consoleAssetPath(id string) string {
	var builder strings.Builder
	for _, char := range id {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('_')
		}
	}
	name := builder.String()
	if name == "" {
		name = "message"
	}
	return "console/" + name + ".js"
}

// Manifest returns a copy of the generated manifest before build-time asset hashes.
func (d *Definition) Manifest() wailsplugs.Manifest { return d.manifest }

// Patches returns a copy of the current patch list.
func (d *Definition) Patches() []wailsplugs.Patch {
	return append([]wailsplugs.Patch(nil), d.patches...)
}

// WriteSource writes manifest.json, patches.json and assets/ to a directory.
func (d *Definition) WriteSource(dir string) error {
	if d == nil {
		return fmt.Errorf("wailsplugs/plugin: nil definition")
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0755); err != nil {
		return err
	}
	if err := d.manifest.Validate(); err != nil {
		// Files are intentionally empty before the pack builder computes hashes.
		if !strings.Contains(err.Error(), "file allowlist") {
			return err
		}
	}
	manifest, err := json.MarshalIndent(d.manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), append(manifest, '\n'), 0644); err != nil {
		return err
	}
	patches, err := json.MarshalIndent(d.patches, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "patches.json"), append(patches, '\n'), 0644); err != nil {
		return err
	}
	keys := make([]string, 0, len(d.assets))
	for key := range d.assets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !wailsplugs.ValidAssetForPack(key) {
			return fmt.Errorf("wailsplugs/plugin: invalid asset path %q", key)
		}
		path := filepath.Join(dir, filepath.FromSlash(key))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, d.assets[key], 0644); err != nil {
			return err
		}
	}
	return nil
}

// Build writes a temporary source tree and packages it as a .plugs archive.
func (d *Definition) Build(output string) (string, error) {
	if d == nil {
		return "", fmt.Errorf("wailsplugs/plugin: nil definition")
	}
	dir, err := os.MkdirTemp("", "wailsplugs-plugin-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	if err := d.WriteSource(dir); err != nil {
		return "", err
	}
	return pack.Build(pack.Options{InputDir: dir, Output: output})
}

func normalizeAssetPath(value string) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	if !strings.HasPrefix(value, "assets/") {
		value = "assets/" + value
	}
	return value
}
