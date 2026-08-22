# Security Policy

## Scope

WailsPlugSystem is a declarative plugin runtime. The `.plugs` archive is not a sandbox for arbitrary code. HTML and CSS patches are sanitized, while JavaScript assets execute in the host WebView when the application explicitly enables `AllowJavaScript`.

## Production recommendations

Use a separate trusted SHA-256 allowlist, keep `AllowJavaScript` disabled by default, set `AllowRootReplace` to false unless required, enable `StrictDependencies`, and configure archive/file size limits. Do not place untrusted plugin archives in a directory that is automatically loaded.

SHA-256 verifies the exact artifact but does not authenticate its author. For publisher identity, distribute the allowlist through a trusted release process and consider signing the allowlist or the package manifest at the application layer.

## Reporting a vulnerability

Do not open a public issue for a security-sensitive report. Contact the repository owner privately through GitHub with a minimal reproduction, affected version, impact, and proposed mitigation. Reports involving path traversal, archive extraction, sanitizer bypass, permission bypass, or unexpected JavaScript execution are especially important.

## External assets and host CSS

External CSS and JavaScript assets are served only from the active, already verified `.plugs` snapshot through the `/__wailsplugs/assets/` route. The route does not make JavaScript trusted: Vite chunks, dynamic imports, fonts, and images remain package content and must come from an approved plugin artifact.

`HostCSS` means that a plugin intentionally renders in the host document and can inherit its CSS cascade. It does not grant access to Go APIs, filesystem APIs, or host methods. Do not use `HostCSS` for untrusted plugins that can inject JavaScript, because CSS inheritance and JavaScript execution still occur in the host WebView context.

The development watcher and `plugs watch` are build-time conveniences. Do not use an unrestricted development directory as a production plugin source; production should load a verified directory or SHA-256 allowlist and keep JavaScript policy explicit.
