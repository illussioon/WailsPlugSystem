package wailsplugs

import (
	"context"
	"errors"
	"sync"
)

const (
	FormatVersion = 1
	APIVersion    = "v1"
)

type Permission string

const (
	PermissionHTML        Permission = "html"
	PermissionCSS         Permission = "css"
	PermissionJS          Permission = "js"
	PermissionReplaceRoot Permission = "replace_root"
)

var (
	ErrInvalidManifest = errors.New("wailsplugs: invalid manifest")
	ErrIntegrity       = errors.New("wailsplugs: integrity check failed")
	ErrPermission      = errors.New("wailsplugs: permission denied")
	ErrConflict        = errors.New("wailsplugs: patch conflict")
	ErrDependency      = errors.New("wailsplugs: dependency not satisfied")
	ErrUnsafeArchive   = errors.New("wailsplugs: unsafe archive")
	ErrPackageTooLarge = errors.New("wailsplugs: package exceeds configured limit")
)

type Dependency struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type FileRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Kind   string `json:"kind,omitempty"`
}

type Manifest struct {
	FormatVersion int          `json:"format_version"`
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Version       string       `json:"version"`
	APIVersion    string       `json:"api_version"`
	Priority      int          `json:"priority"`
	Permissions   []Permission `json:"permissions,omitempty"`
	Dependencies  []Dependency `json:"dependencies,omitempty"`
	Files         []FileRef    `json:"files,omitempty"`
}

type PatchKind string

const (
	PatchSetText     PatchKind = "set_text"
	PatchSetAttr     PatchKind = "set_attr"
	PatchRemove      PatchKind = "remove"
	PatchReplaceHTML PatchKind = "replace_html"
	PatchAppendHTML  PatchKind = "append_html"
	PatchPrependHTML PatchKind = "prepend_html"
	PatchAddClass    PatchKind = "add_class"
	PatchRemoveClass PatchKind = "remove_class"
	PatchInjectCSS   PatchKind = "inject_css"
	PatchInjectJS    PatchKind = "inject_js"
)

type Patch struct {
	ID          string    `json:"id"`
	Kind        PatchKind `json:"kind"`
	Selector    string    `json:"selector,omitempty"`
	Attribute   string    `json:"attribute,omitempty"`
	Value       string    `json:"value,omitempty"`
	Asset       string    `json:"asset,omitempty"`
	ConflictKey string    `json:"conflict_key,omitempty"`
	Optional    bool      `json:"optional,omitempty"`
}

type Package struct {
	Manifest Manifest
	Patches  []Patch
	Assets   map[string][]byte
	Path     string
	SHA256   string
}

type PackageLoader interface {
	Load(context.Context) ([]Package, error)
}

type Decision struct {
	PluginID    string `json:"plugin_id"`
	PatchID     string `json:"patch_id"`
	ConflictKey string `json:"conflict_key,omitempty"`
	Applied     bool   `json:"applied"`
	Reason      string `json:"reason,omitempty"`
}

type RenderResult struct {
	HTML      string     `json:"html"`
	Decisions []Decision `json:"decisions"`
	Plugins   []string   `json:"plugins"`
}

type ManagerOptions struct {
	Loader             PackageLoader
	AllowJavaScript    bool
	AllowRootReplace   bool
	StrictDependencies bool
	MaxPlugins         int
}

type Manager struct {
	mu               sync.RWMutex
	loader           PackageLoader
	allowJavaScript  bool
	allowRootReplace bool
	strictDeps       bool
	maxPlugins       int
	packages         map[string]Package
}
