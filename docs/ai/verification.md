# Verification Contract

## Local checks

Run the smallest affected checks first, then the full set before a release:

| Area | Command | Success criteria |
| --- | --- | --- |
| Go formatting | `gofmt -l .` | no file names are printed |
| Go static checks | `go vet ./...` | exit code 0 |
| Go tests | `go test ./...` | all packages pass |
| Frontend types | `npm run typecheck` in `frontend/` | vue-tsc exits 0 |
| Frontend lint | `npm run lint` in `frontend/` | no errors |
| Frontend tests | `npm run test` in `frontend/` | all Vitest tests pass |
| Frontend build | `npm run build` in `frontend/` | Vite produces `frontend/dist` |
| Wails environment | `wails doctor` | required native tools are detected |
| Binding generation | `wails generate module` | generated bindings leave a clean diff |

## Behavior coverage

Backend tests must cover option validation, path containment, recursive folder expansion,
collision-safe names, atomic cleanup, cancellation, partial failure, page range parsing, all six
image formats, alpha handling, EXIF orientation, target-size warnings, and PDF DPI/page selection.
QR code coverage must include UTF-8 and option validation, all four error-correction levels,
capacity failures, exact PNG size and colors, bounded preview payloads, safe/default output names,
collision suffixes, decoding atomically saved output, image metadata, no-code images, corrupt or
unsupported inputs, and pixel/text safety limits.

Frontend tests must cover navigation, empty/queued/processing/success/error/cancelled states,
form validation, progress events, retry, locale switching, theme persistence, and compression
preview updates. Watermark coverage must include option validation, CJK text rendering, anchor and
tiled placement, preset selection, stale-preview suppression, and keyboard operation of the
before/after comparison slider. QR code coverage must include navigation, accessible secondary
tabs, persisted settings, text/byte validation, size and error-correction controls, custom colors,
live preview updates, save state, decode success/failure/clear states, and browser/native adapter
behavior.

Fixtures should include a transparent image, an EXIF-rotated JPEG, one fixture for each supported
codec, a corrupt file, a three-page PDF containing CJK text that relies on a non-embedded font,
and a password-protected PDF. Native `mupdf` release runners must verify non-white glyph pixels for
the CJK/generic fallback fixture before publishing. The current Windows checkout cannot run that native build
because no C compiler is installed; this is a platform verification gap, not a default-build test
failure.

## Native matrix

CI runs the backend/frontend checks on Windows x64 and macOS Intel. Release validation additionally
runs a native macOS Apple Silicon job. Packaging output must include:

- Windows x64 NSIS installer and portable archive.
- macOS Intel app, macOS Apple Silicon app, and a merged universal DMG.

Platform signing certificates are intentionally not part of the release workflow. Official
publication requires the Ed25519 update key pair; unsigned platform packages may trigger Windows
SmartScreen or macOS Gatekeeper warnings. The update manifest and downloadable assets remain
protected by Ed25519 signatures and SHA-256 checksums.

Each packaged smoke test launches the app, opens a native dialog, processes one image and one PDF,
generates and saves one QR code, verifies output files, cancels a batch, and checks the update
prompt when a signed fixture manifest is available. Windows also checks missing-WebView2 bootstrap
behavior.

## Release evidence

Release notes record the source revision, lockfile state, native build commands, test summaries,
artifact names, SHA-256 values, Ed25519 signature status, and any platform-specific gaps such as
unsigned package warnings.
