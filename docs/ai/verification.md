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
collision-safe names, atomic cleanup, cancellation, partial failure, page range parsing, all four
image formats, alpha handling, EXIF orientation, target-size warnings, and PDF DPI/page selection.

Frontend tests must cover navigation, empty/queued/processing/success/error/cancelled states,
form validation, progress events, retry, locale switching, theme persistence, and compression
preview updates.

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
- macOS Intel app, macOS Apple Silicon app, and a merged universal signed/notarized DMG.

Each packaged smoke test launches the app, opens a native dialog, processes one image and one PDF,
verifies output files, cancels a batch, and checks the update prompt when a signed fixture manifest
is available. Windows also checks missing-WebView2 bootstrap behavior.

## Release evidence

Release notes record the source revision, lockfile state, native build commands, test summaries,
artifact names, SHA-256 values, signing/notarization status, and any platform-specific gaps.
