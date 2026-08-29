# SimpleTools

SimpleTools is a fast, local-first desktop toolbox for image and document workflows.

The source repository is intended to live at <https://github.com/65658dsf/SimpleTools>.

Current tools:

- Convert PNG, JPEG, WebP, AVIF, ICO, and SVG images. SVG output is a self-contained raster
  wrapper and ICO output is a PNG-backed icon (up to 256 pixels).
- Compress supported images with quality or target-size controls.
- Add configurable text watermarks with reusable presets and a live before/after comparison.
- Render PDF pages to PNG at selectable DPI values.

The application is built with Go, Wails, Vue 3, Vite, and shadcn-vue. File processing stays on
the local machine. The only network operation is the optional signed update check.
When no output directory is selected, results are written to `output` beside the running
application. After a successful or partially successful job, the native file manager opens and
reveals the generated files when the platform supports it.

## Development

Install Go 1.26.x, Node.js 24.x/npm 12.x, the Wails v2.15.0 CLI, and the native dependencies
listed in `AGENTS.md`. Then run:

```text
cd frontend
npm ci
cd ..
wails dev
```

The default development build keeps the frontend and image codecs usable without a C compiler.
To enable the native MuPDF PDF renderer locally, run the following command from a machine with the
native toolchain installed:

```text
# PowerShell
$env:CGO_ENABLED = "1"; wails dev -tags "mupdf,nodynamic"

# macOS/Linux
CGO_ENABLED=1 wails dev -tags "mupdf,nodynamic"
```

The release workflow uses the same tags on Windows and macOS runners, so release packages include
PDF rendering.

Backend and frontend verification commands are documented in `docs/ai/verification.md`.

## Releases

The manual GitHub Actions workflow in `.github/workflows/release.yml` builds a Windows x64 NSIS
installer and portable archive, native macOS Intel/Apple Silicon apps, and a merged Universal DMG.
It copies the application and third-party notices into redistributable archives, records SHA-256
checksums, and signs update assets with Ed25519.

Official publication requires only the Ed25519 update key pair. Platform signing certificates are
not used by the release workflow. The resulting Windows and macOS packages are functional, but
Windows SmartScreen and macOS Gatekeeper may show warnings on first launch. Keep the matching
base64 public key in `SIMPLETEOOLS_UPDATE_PUBLIC_KEY` for installed clients; the private key must
remain a CI secret. See `.env.example` for the two required variables.

## License

SimpleTools is licensed under the GNU Affero General Public License, version 3 or later. See
`LICENSE` and `THIRD_PARTY_NOTICES.md` for the complete application and dependency notices.
