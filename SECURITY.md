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

## Encrypted payloads and anti-dump limits

The optional `aes-256-gcm` package mode keeps `manifest.json` readable and encrypts `patches.json` plus all assets inside authenticated `payload.bin`. A normal ZIP extractor therefore cannot recover the plugin source. The decryption key must be delivered by the host through `PackageOptions.DecryptionKey`, `client.Options.DecryptionKey`, or a `DecryptionKeyProvider`; it is never written into the archive.

Do not hardcode a reusable production key in a public host binary. Prefer a license/backend service, OS secure storage such as Keychain, or a device/account-bound key derivation scheme. Keep signing and license credentials outside frontend assets.

Encryption raises the cost of casual extraction but is not DRM. A user who controls the running host can potentially capture the key, inspect decrypted memory, hook the asset/DOM boundary, or debug JavaScript. Sensitive algorithms and secrets should remain server-side; minification and obfuscation can supplement encryption but do not replace access control or code signing.
