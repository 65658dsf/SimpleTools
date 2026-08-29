---
name: simpletools-feature-development
description: Develop or extend a user-facing SimpleTools feature that crosses the Vue, Wails, and Go boundary, such as a new media tool, format, option, job workflow, or platform behavior. Use for feature implementation; do not use for isolated typo fixes, dependency-only maintenance, or release-only operations.
---

# SimpleTools Feature Development

Use this workflow when a change adds or changes observable product behavior. Keep the feature
local-first, preserve the existing layer boundaries, and leave a reproducible verification trail.

## Mandatory Preflight

Read these files before editing:

- [`AGENTS.md`](../../../AGENTS.md)
- [`docs/ai/architecture.md`](../../../docs/ai/architecture.md)
- [`docs/ai/verification.md`](../../../docs/ai/verification.md)

Also read the nearest scoped guide for every area being changed:

- [`frontend/AGENTS.md`](../../../frontend/AGENTS.md) for Vue, Pinia, routing, styles, or translations.
- [`internal/AGENTS.md`](../../../internal/AGENTS.md) for Go application, platform, or tool code.

Inspect the current implementation and tests before deciding on file names. Treat code, Go DTOs,
generated bindings, and CI commands as facts; do not infer a new abstraction from the feature name.

## Scope

This skill covers:

- adding a workspace tool or route;
- adding an image/PDF format, processing option, preview, or job state;
- changing a Wails method, DTO, event payload, or native platform behavior;
- coordinated frontend/backend changes that must remain cross-platform.

It does not cover a standalone copy edit, an isolated CSS adjustment with no behavior change,
dependency housekeeping, or release signing/publication work. For those, use the smallest local
workflow and the root verification rules.

## Change Map

Start by writing a short change map in the task notes or plan:

| Concern | Actual source locations | Required question |
| --- | --- | --- |
| Tool identity and navigation | `frontend/src/types.ts`, `frontend/src/router.ts`, `frontend/src/App.vue` | Does the tool have one stable `ToolId` and route? |
| UI and state | `frontend/src/views/WorkspaceView.vue`, `frontend/src/stores/workspace.ts`, `frontend/src/i18n.ts` | Are empty, queued, processing, success, failure, retry, and cancelled states represented in both locales? |
| Wails contract | `internal/app/app.go`, `frontend/src/services/wails.ts`, `frontend/src/types.ts` | Are paths/options/IDs the only bridge payloads, and are Go types the source of truth? |
| Processing | `internal/tools/*.go`, `internal/platform/*.go` | Which layer owns validation, codecs, filesystem access, and OS behavior? |
| Generated output | `frontend/wailsjs/**` | Can it be regenerated instead of hand-edited? |
| Regression coverage | `internal/**/*_test.go`, `frontend/src/*.test.ts` | Which observable success and failure paths prove the change? |

If a feature does not touch one of these areas, explain why in the handoff rather than adding a
placeholder file.

## Ordered Workflow

1. **Define the observable contract.** Record inputs, defaults, valid ranges, output naming, errors,
   cancellation behavior, progress semantics, and whether the change affects a public binding or
   event. Preserve one-based PDF page ranges and existing format semantics where applicable.
2. **Trace the existing path.** Follow `frontend -> typed Wails service -> internal/app ->
   internal/tools` and/or `internal/platform`. Reuse the current job registry, worker pools,
   output allocator, atomic writer, preview limits, and event names instead of creating parallel
   paths.
3. **Change the source of truth first.** For a public method or payload, update the Go DTO and
   application implementation, then run `wails generate module`, then update the TypeScript
   adapter/types and UI. Never hand-edit `frontend/wailsjs` generated bindings.
4. **Implement backend behavior.** Validate canonical input and output paths in Go. Keep media
   processing offline and keep complete file contents out of IPC. Make jobs cancellable, bound
   concurrency, preserve successful items after partial failure, write to a temporary file in the
   destination directory, atomically rename it, and never overwrite an input or existing output.
   Use deterministic `-1`, `-2`, ... collision suffixes.
5. **Implement frontend behavior.** Add both `zh` and `en` translation keys, route/navigation
   state, form validation, loading/error/cancel states, and theme-safe styling. Keep filesystem
   access in the Wails service; browser mode may only use the existing mock adapter.
6. **Add regression tests before broad cleanup.** Cover the normal path plus invalid input,
   cancellation, partial failure, duplicate/collision behavior, and platform-specific branches
   relevant to the feature. For compression or previews, test byte estimates/limits without
   moving full image data through Wails.
7. **Regenerate and inspect.** Run binding generation, review any expected generated changes, make
   sure no unrelated files changed, and remove temporary fixtures or build output. For a contract
   change, generated binding changes are expected and must be committed; the final clean-tree
   generation check must pass after those files are included.
8. **Verify in increasing scope.** Run the targeted checks below, then the full local set. If the
   feature touches MuPDF, packaging, or native dialogs, record native-runner checks separately;
   cross-compilation is not native verification.

## Non-Negotiable Boundaries

- The updater is the only network-capable subsystem. Do not add upload, telemetry, account, or
  cloud-processing calls as part of a media feature.
- The frontend sends paths, options, and opaque IDs only. Do not read arbitrary local files from
  browser code or return full image/PDF contents over IPC; previews stay bounded thumbnails.
- `internal/tools` must not import Vue or Wails runtime details. `internal/platform` owns OS
  commands and filesystem helpers; `internal/app` owns orchestration and event translation.
- Preserve the default metadata policy (remove metadata, apply EXIF orientation) unless the
  contract explicitly changes it and tests cover the new policy.
- JPEG transparency is flattened onto white; PNG is lossless; lossy target-size search remains
  best effort and must report an unattainable target as a warning.
- PDF rendering remains page-by-page, RGB with a white background, bounded by the pixel safety
  limit, and rejects password-protected documents with a user-facing error.
- Keep AGPL/MuPDF and third-party notices complete when adding or changing native dependencies.

## Commands and Acceptance

Run commands from the stated directory. A command passes only when it exits with code 0 and its
stated artifact or output is present.

| Stage | Working directory | Command | Acceptance |
| --- | --- | --- | --- |
| Backend format | repository root | `gofmt -l .` | Prints no Go files |
| Backend unit check | repository root | `go test ./...` | All packages pass |
| Backend static check | repository root | `go vet ./...` | Exit code 0 |
| Binding generation | repository root | `wails generate module` | Generation succeeds |
| Binding drift check | repository root | `git diff -- frontend/wailsjs` | Only expected generated changes are present; rerunning generation after the source change produces no additional diff |
| Frontend types | `frontend/` | `npm run typecheck` | `vue-tsc` exits 0 |
| Frontend lint | `frontend/` | `npm run lint` | No errors |
| Frontend behavior | `frontend/` | `npm run test` | All Vitest tests pass |
| Frontend build | `frontend/` | `npm run build` | Vite creates `frontend/dist` |

For a narrow change, run its relevant rows first. Before handoff, run every applicable row and
mention commands that could not run. Native MuPDF/package checks require the target operating
system, CGO, and the documented build tags (`mupdf,nodynamic`).

## Failure Handling

- Stop before editing when the required scoped guide, native prerequisite, or contract decision is
  missing. Record the blocker and continue only with safe read-only discovery.
- Preserve the first failing command and classify it as code, fixture, generated-file drift, or
  environment failure. Do not hide it behind a passing fallback/mock test.
- If generation changes unexpected files, inspect the diff and fix the source or generation
  inputs; do not manually patch generated bindings or discard unrelated user changes.
- If a native check is unavailable, report the exact OS/tool gap and leave the portable checks
  green. Do not claim cross-compiled output proves native behavior.

## Handoff

Report:

- the user-visible behavior and affected files/layers;
- public binding/event or compatibility impact;
- tests and commands actually run, with concise results;
- generated files and documentation updated;
- native checks not run, remaining risks, and rollback/retry notes.
