# Security Policy

## Scope

WailsPlugSystem is a declarative plugin runtime. The `.plugs` archive is not a sandbox for arbitrary code. HTML and CSS patches are sanitized, while JavaScript assets execute in the host WebView when the application explicitly enables `AllowJavaScript`.

## Production recommendations

Use a separate trusted SHA-256 allowlist, keep `AllowJavaScript` disabled by default, set `AllowRootReplace` to false unless required, enable `StrictDependencies`, and configure archive/file size limits. Do not place untrusted plugin archives in a directory that is automatically loaded.

SHA-256 verifies the exact artifact but does not authenticate its author. For publisher identity, distribute the allowlist through a trusted release process and consider signing the allowlist or the package manifest at the application layer.

## Reporting a vulnerability

Do not open a public issue for a security-sensitive report. Contact the repository owner privately through GitHub with a minimal reproduction, affected version, impact, and proposed mitigation. Reports involving path traversal, archive extraction, sanitizer bypass, permission bypass, or unexpected JavaScript execution are especially important.
