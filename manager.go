package wailsplugs

import (
	"context"
	"fmt"
	"html"
	"sort"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func NewManager(options ManagerOptions) *Manager {
	maxPlugins := options.MaxPlugins
	if maxPlugins <= 0 {
		maxPlugins = 256
	}
	return &Manager{
		loader:           options.Loader,
		allowJavaScript:  options.AllowJavaScript,
		allowRootReplace: options.AllowRootReplace,
		strictDeps:       options.StrictDependencies,
		maxPlugins:       maxPlugins,
		packages:         map[string]Package{},
	}
}

func (m *Manager) Reload(ctx context.Context) error {
	if m.loader == nil {
		return fmt.Errorf("wailsplugs: loader is nil")
	}
	packages, err := m.loader.Load(ctx)
	if err != nil {
		return err
	}
	if len(packages) > m.maxPlugins {
		return fmt.Errorf("wailsplugs: %d plugins exceeds limit %d", len(packages), m.maxPlugins)
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Manifest.ID < packages[j].Manifest.ID })
	next := make(map[string]Package, len(packages))
	for _, item := range packages {
		if _, exists := next[item.Manifest.ID]; exists {
			return fmt.Errorf("wailsplugs: duplicate plugin id %q", item.Manifest.ID)
		}
		next[item.Manifest.ID] = item
	}
	if err := validateDependencies(next, m.strictDeps); err != nil {
		return err
	}
	m.mu.Lock()
	m.packages = next
	m.mu.Unlock()
	return nil
}

func validateDependencies(packages map[string]Package, strict bool) error {
	for _, item := range packages {
		for _, dep := range item.Manifest.Dependencies {
			target, ok := packages[dep.ID]
			if !ok {
				if strict {
					return fmt.Errorf("%w: plugin %s needs %s", ErrDependency, item.Manifest.ID, dep.ID)
				}
				continue
			}
			if dep.Version != "" && dep.Version != target.Manifest.Version {
				return fmt.Errorf("%w: plugin %s needs %s@%s, got %s", ErrDependency, item.Manifest.ID, dep.ID, dep.Version, target.Manifest.Version)
			}
		}
	}
	return nil
}

func (m *Manager) Packages() []Package {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Package, 0, len(m.packages))
	for _, item := range m.packages {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Manifest.ID < result[j].Manifest.ID })
	return result
}

func (m *Manager) Render(source string) (RenderResult, error) {
	m.mu.RLock()
	packages := make([]Package, 0, len(m.packages))
	for _, item := range m.packages {
		packages = append(packages, item)
	}
	m.mu.RUnlock()
	sort.Slice(packages, func(i, j int) bool {
		if packages[i].Manifest.Priority != packages[j].Manifest.Priority {
			return packages[i].Manifest.Priority > packages[j].Manifest.Priority
		}
		return packages[i].Manifest.ID < packages[j].Manifest.ID
	})
	document, err := xhtml.Parse(strings.NewReader(source))
	if err != nil {
		return RenderResult{}, fmt.Errorf("wailsplugs: parse html: %w", err)
	}
	decisions := []Decision{}
	claimed := map[string]bool{}
	plugins := make([]string, 0, len(packages))
	for _, item := range packages {
		plugins = append(plugins, item.Manifest.ID)
		for index, patch := range item.Patches {
			patchID := patch.ID
			if patchID == "" {
				patchID = fmt.Sprintf("patch-%d", index)
			}
			key := patch.ConflictKey
			if key == "" {
				key = string(patch.Kind) + ":" + patch.Selector + ":" + patch.Attribute + ":" + patch.Asset
			}
			decision := Decision{PluginID: item.Manifest.ID, PatchID: patchID, ConflictKey: key}
			if claimed[key] {
				decision.Reason = "lower priority than an already selected patch"
				decisions = append(decisions, decision)
				continue
			}
			if err := m.applyPatch(document, item, patch); err != nil {
				if patch.Optional {
					decision.Reason = err.Error()
					decisions = append(decisions, decision)
					continue
				}
				return RenderResult{}, fmt.Errorf("plugin %s patch %s: %w", item.Manifest.ID, patchID, err)
			}
			claimed[key] = true
			decision.Applied = true
			decisions = append(decisions, decision)
		}
	}
	var output strings.Builder
	if err := xhtml.Render(&output, document); err != nil {
		return RenderResult{}, err
	}
	return RenderResult{HTML: output.String(), Decisions: decisions, Plugins: plugins}, nil
}

func (m *Manager) applyPatch(document *xhtml.Node, item Package, patch Patch) error {
	if patch.Kind == PatchInjectCSS || patch.Kind == PatchInjectJS {
		return m.injectAsset(document, item, patch)
	}
	if patch.Kind == PatchReplaceHTML && patch.Selector == "html" && !m.allowRootReplace {
		return fmt.Errorf("%w: root replacement disabled", ErrPermission)
	}
	if !item.Manifest.HasPermission(PermissionHTML) {
		return fmt.Errorf("%w: plugin lacks html permission", ErrPermission)
	}
	if (patch.Kind == PatchReplaceHTML || patch.Kind == PatchAppendHTML || patch.Kind == PatchPrependHTML) && !item.Manifest.HasPermission(PermissionHTML) {
		return fmt.Errorf("%w: plugin lacks html permission", ErrPermission)
	}
	if patch.Kind == PatchReplaceHTML && patch.Selector == "html" && !item.Manifest.HasPermission(PermissionReplaceRoot) {
		return fmt.Errorf("%w: plugin lacks replace_root permission", ErrPermission)
	}
	nodes := selectNodes(document, patch.Selector)
	if len(nodes) == 0 {
		return fmt.Errorf("selector %q matched no nodes", patch.Selector)
	}
	for _, node := range nodes {
		switch patch.Kind {
		case PatchSetText:
			clearChildren(node)
			node.AppendChild(&xhtml.Node{Type: xhtml.TextNode, Data: patch.Value})
		case PatchSetAttr:
			setAttr(node, patch.Attribute, patch.Value)
		case PatchRemove:
			if node.Parent != nil {
				node.Parent.RemoveChild(node)
			}
		case PatchReplaceHTML:
			children, err := sanitizedFragment(patch.Value)
			if err != nil {
				return err
			}
			replaceNodeChildren(node, children)
		case PatchAppendHTML:
			children, err := sanitizedFragment(patch.Value)
			if err != nil {
				return err
			}
			for _, child := range children {
				node.AppendChild(child)
			}
		case PatchPrependHTML:
			children, err := sanitizedFragment(patch.Value)
			if err != nil {
				return err
			}
			for i := len(children) - 1; i >= 0; i-- {
				node.InsertBefore(children[i], node.FirstChild)
			}
		case PatchAddClass:
			addClass(node, patch.Value)
		case PatchRemoveClass:
			removeClass(node, patch.Value)
		default:
			return fmt.Errorf("unsupported patch %q", patch.Kind)
		}
	}
	return nil
}

func (m *Manager) injectAsset(document *xhtml.Node, item Package, patch Patch) error {
	data, ok := item.Assets[patch.Asset]
	if !ok {
		return fmt.Errorf("asset %q not found", patch.Asset)
	}
	head := firstNode(document, func(node *xhtml.Node) bool { return node.Type == xhtml.ElementNode && node.Data == "head" })
	if head == nil {
		return fmt.Errorf("document has no head")
	}
	switch patch.Kind {
	case PatchInjectCSS:
		if !item.Manifest.HasPermission(PermissionCSS) {
			return fmt.Errorf("%w: plugin lacks css permission", ErrPermission)
		}
		style := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Style, Data: "style"}
		style.AppendChild(&xhtml.Node{Type: xhtml.TextNode, Data: string(data)})
		head.AppendChild(style)
	case PatchInjectJS:
		if !m.allowJavaScript || !item.Manifest.HasPermission(PermissionJS) {
			return fmt.Errorf("%w: JavaScript disabled or permission missing", ErrPermission)
		}
		script := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Script, Data: "script", Attr: []xhtml.Attribute{{Key: "type", Val: "module"}}}
		script.AppendChild(&xhtml.Node{Type: xhtml.TextNode, Data: string(data)})
		head.AppendChild(script)
	default:
		return fmt.Errorf("unsupported asset patch %q", patch.Kind)
	}
	return nil
}

func firstNode(root *xhtml.Node, predicate func(*xhtml.Node) bool) *xhtml.Node {
	if predicate(root) {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if result := firstNode(child, predicate); result != nil {
			return result
		}
	}
	return nil
}

func selectNodes(root *xhtml.Node, selector string) []*xhtml.Node {
	parts := strings.Fields(selector)
	if len(parts) == 0 {
		return nil
	}
	current := []*xhtml.Node{root}
	for _, part := range parts {
		next := []*xhtml.Node{}
		for _, parent := range current {
			walkElements(parent, func(node *xhtml.Node) {
				if matchesSimple(node, part) {
					next = append(next, node)
				}
			})
		}
		current = next
	}
	return current
}

func walkElements(root *xhtml.Node, visit func(*xhtml.Node)) {
	if root.Type == xhtml.ElementNode {
		visit(root)
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		walkElements(child, visit)
	}
}

func matchesSimple(node *xhtml.Node, selector string) bool {
	if node.Type != xhtml.ElementNode || selector == ":root" && node.Data != "html" {
		return false
	}
	if selector == ":root" || selector == "*" {
		return true
	}
	rest := selector
	if i := strings.IndexAny(rest, "#.["); i >= 0 {
		if i > 0 && node.Data != rest[:i] {
			return false
		}
		rest = rest[i:]
	} else {
		return node.Data == rest
	}
	for len(rest) > 0 {
		switch rest[0] {
		case '#':
			token, tail := readToken(rest[1:])
			if attr(node, "id") != token {
				return false
			}
			rest = tail
		case '.':
			token, tail := readToken(rest[1:])
			if !hasClass(node, token) {
				return false
			}
			rest = tail
		case '[':
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return false
			}
			expression := rest[1:end]
			pieces := strings.SplitN(expression, "=", 2)
			value, exists := attrValue(node, pieces[0])
			if !exists {
				return false
			}
			if len(pieces) == 2 && strings.Trim(pieces[1], "\"'") != value {
				return false
			}
			rest = rest[end+1:]
		default:
			return false
		}
	}
	return true
}

func readToken(value string) (string, string) {
	for i, r := range value {
		if r == '#' || r == '.' || r == '[' {
			return value[:i], value[i:]
		}
	}
	return value, ""
}

func attr(node *xhtml.Node, key string) string { value, _ := attrValue(node, key); return value }
func attrValue(node *xhtml.Node, key string) (string, bool) {
	for _, item := range node.Attr {
		if item.Key == key {
			return item.Val, true
		}
	}
	return "", false
}
func setAttr(node *xhtml.Node, key, value string) {
	for i := range node.Attr {
		if node.Attr[i].Key == key {
			node.Attr[i].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, xhtml.Attribute{Key: key, Val: value})
}
func clearChildren(node *xhtml.Node) {
	for node.FirstChild != nil {
		node.RemoveChild(node.FirstChild)
	}
}
func replaceNodeChildren(node *xhtml.Node, children []*xhtml.Node) {
	clearChildren(node)
	for _, child := range children {
		node.AppendChild(child)
	}
}
func hasClass(node *xhtml.Node, class string) bool {
	return strings.Contains(" "+attr(node, "class")+" ", " "+class+" ")
}
func addClass(node *xhtml.Node, class string) {
	if !hasClass(node, class) {
		current := strings.TrimSpace(attr(node, "class"))
		setAttr(node, "class", strings.TrimSpace(current+" "+class))
	}
}
func removeClass(node *xhtml.Node, class string) {
	items := strings.Fields(attr(node, "class"))
	keep := items[:0]
	for _, item := range items {
		if item != class {
			keep = append(keep, item)
		}
	}
	setAttr(node, "class", strings.Join(keep, " "))
}

func sanitizedFragment(source string) ([]*xhtml.Node, error) {
	nodes, err := xhtml.ParseFragment(strings.NewReader(source), &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"})
	if err != nil {
		return nil, err
	}
	kept := nodes[:0]
	for _, node := range nodes {
		if isForbiddenElement(node) {
			continue
		}
		sanitizeNode(node)
		kept = append(kept, node)
	}
	return kept, nil
}

func isForbiddenElement(node *xhtml.Node) bool {
	if node == nil || node.Type != xhtml.ElementNode {
		return false
	}
	return node.Data == "script" || node.Data == "iframe" || node.Data == "object" || node.Data == "embed" || node.Data == "link"
}

func sanitizeNode(node *xhtml.Node) {
	if node.Type == xhtml.ElementNode {
		forbidden := isForbiddenElement(node)
		if forbidden && node.Parent != nil {
			node.Parent.RemoveChild(node)
			return
		}
		attrs := node.Attr[:0]
		for _, item := range node.Attr {
			key := strings.ToLower(item.Key)
			value := strings.TrimSpace(strings.ToLower(html.UnescapeString(item.Val)))
			if strings.HasPrefix(key, "on") || strings.HasPrefix(value, "javascript:") || strings.HasPrefix(value, "vbscript:") || strings.HasPrefix(value, "data:") {
				continue
			}
			attrs = append(attrs, item)
		}
		node.Attr = attrs
	}
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		sanitizeNode(child)
		child = next
	}
}
