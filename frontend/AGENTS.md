# Vue Frontend

The frontend is a Vue 3 + TypeScript + Vite app. `src/views` owns page composition, `src/stores`
owns queue and settings state, and `src/services` owns the Wails bridge/fallback adapter. Do not
read arbitrary local files from browser code; pass paths and options to Go through the typed
service contract.

Keep all five tools usable at `/convert`, `/compress`, `/watermark`, `/qrcode`, and `/pdf`. Every
queue state must have a visible loading, success, failure, retry, and cancellation representation.
The QR code tool uses the dedicated `previewQRCode`, `saveQRCode`, and `decodeQRCode` service
methods instead of a file job. Browser mode may render or decode locally but must not pretend to
save a file. New backend DTO fields must be reflected in `src/types.ts` and the generated binding
adapter.

Run `npm run typecheck`, `npm run lint`, `npm run test`, and `npm run build` from this directory.
