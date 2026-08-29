# SimpleTools

## Project Overview

SimpleTools is a local-first Wails desktop application for Windows 10/11 x64 and macOS 12+
(Intel and Apple Silicon). The current release contains image conversion, image compression,
and PDF-to-PNG export.

## Setup

## Technology and prerequisites

- Go 1.26.x with CGO enabled for the MuPDF binding.
- Wails v2.15.0 CLI.
- Node.js 24.x and npm 12.x for `frontend/`.
- Windows builds need a native C toolchain, NSIS, and the WebView2 bootstrapper.
- macOS builds need Xcode Command Line Tools and must run on a native target runner.

## Repository Map

- `main.go`: Wails entry point.
- `internal/app`: application service, task registry, bindings, and event emission.
- `internal/tools`: format codecs, image operations, PDF rendering, validation, and naming.
- `internal/platform`: filesystem, native dialogs, update transport, and platform-specific code.
- `frontend/src`: Vue pages, stores, Wails client, translations, and design system styles.
- `docs/ai`: architecture, verification, and delivery contracts.
- `.github/workflows`: CI and Ed25519-signed release automation.

Dependency direction is `frontend -> Wails bindings -> internal/app -> internal/tools` and
`internal/platform`. Tool code must not import Vue, Wails runtime details, or concrete UI state.
All file-system access stays in Go. Processing methods receive paths and options; large file
contents must never cross the Wails bridge.

## Commands

Run commands from the directory shown. The first command in a fresh checkout is `npm ci` in
`frontend/` after the lockfile has been generated.

```text
frontend/  npm ci
frontend/  npm run typecheck
frontend/  npm run lint
frontend/  npm run test
frontend/  npm run build
root       gofmt -w .
root       go vet ./...
root       go test ./...
root       wails doctor
root       wails dev
root       wails generate module
root       wails build
```

`wails build` and installer commands are native-platform operations. Windows uses the NSIS
target; macOS builds Intel and Apple Silicon separately before creating the universal application
bundle. See `docs/ai/verification.md` for the full matrix.

## Architecture

The frontend calls only typed Wails bindings. The application service validates and schedules
requests, tool packages perform media work, and platform packages own operating-system behavior.
No layer may bypass that direction or move file contents across the Wails bridge.

## Conventions

Use Go formatting and explicit error wrapping in the backend. Keep Vue views focused on composition,
Pinia stores on state transitions, and translations in `frontend/src/i18n.ts`. Do not add network
calls to media processing or read arbitrary local files from browser code.

## Testing

The minimum checks are the Go formatter, vet, Go tests, frontend typecheck, lint, Vitest suite, and
Vite build listed above. Native packaging and MuPDF checks must run on the target operating system.

## Change Checklist

Before handoff, regenerate Wails bindings, run all local checks, inspect the final diff, and record
native platform gaps instead of treating a cross-compiled binary as native verification.

## Non-negotiable invariants

- Processing is offline. The updater is the only network-capable subsystem.
- Jobs are cancellable and report file-level progress. A failed input must not discard successful
  outputs from the same batch.
- Outputs are written to a temporary file and atomically renamed. Existing files are never
  overwritten; use deterministic collision suffixes.
- Input paths are canonicalized and output paths must remain below the user-selected directory.
- JPEG output flattens transparency onto white. PNG remains lossless. Lossy target-size search is
  best effort and must return a warning when the requested size cannot be reached.
- Metadata is removed by default. EXIF orientation is applied to pixels before encoding.
- PDF pages are rendered one at a time. Page ranges are one-based and inclusive. Password-
  protected documents are rejected with a user-facing error.
- MuPDF is distributed under AGPL obligations. Keep `THIRD_PARTY_NOTICES.md` and the source/
  license files in every release artifact.
- The bundled go-fitz static archives do not ship CJK fallback fonts. Native `mupdf` builds install
  the embedded Simplified Chinese Noto Sans SC subset through the MuPDF system-font callback;
  native glyph-pixel verification is still required before release. Documents requiring Japanese,
  Korean, Traditional Chinese, or characters outside the subset need embedded fonts or a broader
  fallback build. The non-native test path does not exercise MuPDF glyph output.

Before changing a public binding or event payload, update the Go DTO, regenerate TypeScript
bindings, update the contract tests, and run both backend and frontend verification.

## Definition of done

A change is complete when the affected Go and frontend checks pass, generated bindings are clean,
the relevant tool fixtures cover success and failure paths, and the handoff records actual command
results. Native packaging changes additionally need a build on each affected OS/architecture.
