<div align="center">
  <img src="build/appicon.png" width="112" alt="SimpleTools application icon">
  <h1>SimpleTools</h1>
  <p>A fast, private desktop toolbox for image, QR code, and PDF workflows.</p>
  <p>
    <strong>English</strong> | <a href="README.zh-CN.md">简体中文</a>
  </p>
  <p>
    <a href="https://github.com/65658dsf/SimpleTools/actions/workflows/ci.yml"><img src="https://github.com/65658dsf/SimpleTools/actions/workflows/ci.yml/badge.svg" alt="CI status"></a>
    <a href="https://github.com/65658dsf/SimpleTools/releases/latest"><img src="https://img.shields.io/github/v/release/65658dsf/SimpleTools?display_name=tag&amp;sort=semver" alt="Latest release"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0--or--later-16a34a" alt="AGPL-3.0-or-later license"></a>
    <img src="https://img.shields.io/badge/platform-Windows%2010%2F11%20%7C%20macOS%2012%2B-4b5563" alt="Windows 10 and 11, macOS 12 or later">
  </p>
</div>

SimpleTools is a local-first desktop application for everyday image and document tasks. It has no
accounts, telemetry, cloud storage, or remote processing. Apart from the optional signed update
check, all processing stays on your device.

[Download the latest release](https://github.com/65658dsf/SimpleTools/releases/latest) |
[Report an issue](https://github.com/65658dsf/SimpleTools/issues) |
[Architecture](docs/ai/architecture.md)

## Features

| Tool | Capabilities |
| --- | --- |
| Image conversion | Batch-convert PNG, JPEG, WebP, AVIF, ICO, and self-contained SVG files with embedded raster snapshots. |
| Image compression | Keep the source format and reduce size with quality, lossless, or best-effort target-file-size controls. |
| Text watermark | Apply Simplified Chinese-capable text, presets, nine anchor positions, tiling, rotation, shadow, and a live before/after preview. |
| QR code | Generate customizable PNG QR codes with L/M/Q/H error correction, or decode QR code images locally and copy the result. |
| PDF to PNG | Render all pages or a one-based page range at 72, 150, 300, or 600 DPI. |

The shared workflow also includes:

- File and folder drag-and-drop, recursive folder import, and batch queues.
- Live file-level progress, cancellation, retry, and partial-success handling.
- Atomic output writes and deterministic collision suffixes; existing files are never overwritten.
- Automatic output reveal after a successful or partially successful job.
- English and Simplified Chinese interfaces with light, dark, and system themes.

## Download

Download a package from [GitHub Releases](https://github.com/65658dsf/SimpleTools/releases/latest):

| Platform | Supported targets | Packages |
| --- | --- | --- |
| Windows 10/11 | x64 | NSIS installer and portable ZIP |
| macOS 12+ | Intel and Apple Silicon | Universal DMG and ZIP |

Windows uses WebView2; the installer includes its bootstrapper, while the portable build expects a
compatible WebView2 runtime on the machine.

The current release workflow does not use Windows or Apple platform-signing certificates, so
SmartScreen or Gatekeeper may show a warning on first launch. Published update assets are still
protected by Ed25519 signatures and SHA-256 checksums. Verify the release files before proceeding
past an operating-system warning.

## Basic Workflow

1. Choose a tool from the sidebar.
2. Drop files or a folder into the workspace, or use the native file picker. QR generation accepts
   text directly.
3. Adjust the tool settings and optionally choose an output directory.
4. Start processing. You can cancel a running batch or retry failed items without losing successful
   outputs.

When no output directory is selected, SimpleTools creates `output` beside the running executable.
If that location is read-only, choose another writable directory. On completion, the native file
manager reveals the generated files when the platform supports it.

## Format Notes

- JPEG output flattens transparency onto white. PNG remains lossless; WebP and AVIF support lossy
  and lossless encoding.
- ICO output contains one PNG-backed image and is limited to 256 pixels on its longest side.
- SVG output is a self-contained wrapper around a PNG snapshot. SVG input must contain a
  supported embedded raster image; arbitrary vector markup is not rasterized.
- Target-size compression is best effort. If the requested size cannot be reached, the completed
  item includes a warning rather than silently changing format.
- Metadata is removed by the current encoders. Selecting metadata preservation reports a warning
  instead of claiming that metadata was retained. JPEG EXIF orientation is applied to pixels before
  encoding.
- PDF rendering is page-by-page. Password-protected documents are rejected. Native builds include
  a Simplified Chinese Noto Sans SC fallback subset; other CJK scripts and characters outside that
  subset require embedded PDF fonts.

## Development

### Prerequisites

| Requirement | Version or notes |
| --- | --- |
| Go | 1.26.x; the exact version is pinned in [.go-version](.go-version) |
| Node.js and npm | Node.js 24.x and npm 12.x; Node.js is pinned in [.nvmrc](.nvmrc) |
| Wails CLI | v2.15.0 |
| Native toolchain | CGO-capable C toolchain for MuPDF; Xcode Command Line Tools on macOS |
| Windows packaging | NSIS and the WebView2 bootstrapper |

Install the Wails CLI and frontend dependencies, then start the development application:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
cd frontend
npm ci
cd ..
wails dev
```

The default development build does not compile the native MuPDF renderer. Image, watermark, and QR
code tools remain available. To enable PDF rendering, use a native machine with CGO and the required
C toolchain:

```powershell
# Windows PowerShell
$env:CGO_ENABLED = "1"
wails dev -tags "mupdf,nodynamic"
```

```bash
# macOS
CGO_ENABLED=1 wails dev -tags "mupdf,nodynamic"
```

Release builds use the same `mupdf,nodynamic` tags on their native Windows and macOS runners.

## Verification

Run the checks for every affected layer before submitting a change:

| Area | Working directory | Commands |
| --- | --- | --- |
| Go | Repository root | `gofmt -l .`, `go vet ./...`, `go test ./...` |
| Wails | Repository root | `wails doctor`, `wails generate module` |
| Frontend | `frontend/` | `npm run typecheck`, `npm run lint`, `npm run test`, `npm run build` |

After binding generation, inspect `frontend/wailsjs` and confirm a second generation produces no
additional changes. The complete local, native, and packaging matrix is documented in
[docs/ai/verification.md](docs/ai/verification.md).

## Project Structure

The dependency direction is `frontend -> Wails bindings -> internal/app -> internal/tools` and
`internal/platform`. Full media files never cross the native bridge; the frontend exchanges paths,
options, job IDs, progress events, and bounded previews.

| Path | Responsibility |
| --- | --- |
| `main.go` | Wails entry point and application bootstrap |
| `internal/app` | Request validation, job lifecycle, bindings, and event emission |
| `internal/tools` | Image codecs and operations, QR processing, PDF rendering, and validation |
| `internal/platform` | Filesystem safety, native dialogs, output reveal, and signed updates |
| `frontend/src` | Vue views, Pinia state, Wails adapter, translations, and styles |
| `docs/ai` | Architecture and verification contracts |
| `.github/workflows` | CI and release automation |

See [docs/ai/architecture.md](docs/ai/architecture.md) for the full processing and platform
contracts.

## Releases

The manual [release workflow](.github/workflows/release.yml) builds Windows x64 and native macOS
Intel/Apple Silicon applications, merges the macOS binaries into a universal package, generates
checksums, and records Ed25519 asset signatures in an update manifest. Redistributable artifacts
include the application license and required third-party notices.

Maintainers configure the update repository and Ed25519 keys with the variables described in
[.env.example](.env.example). The private key belongs only in CI secrets and must never be shipped
with the application.

## License

SimpleTools is licensed under the [GNU Affero General Public License, version 3 or later](LICENSE).
MuPDF and other bundled components have their own obligations; see
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for the complete dependency and redistribution
notices.
