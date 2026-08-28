# Go Backend

`internal/app` is the Wails-facing application boundary. It validates requests, owns job
lifecycle, and translates worker results to the `job:*` event contract. It may depend on
`internal/tools` and `internal/platform`; those packages must not depend on Wails runtime APIs.

`internal/tools` owns deterministic, testable media behavior. Keep codecs behind the format/PDF
interfaces and preserve the default no-native-dependency test path. The MuPDF implementation is
compiled only with `CGO_ENABLED=1 -tags "mupdf,nodynamic"` on a target-native runner.

`internal/platform` owns path discovery, atomic writes, native update/install behavior, and OS
differences. Never concatenate untrusted input into an output path without canonicalizing it and
checking the selected output directory boundary.

Run `gofmt -w .`, `go vet ./...`, and `go test ./...` after backend changes. Add a regression test
for every new failure mode, especially cancellation, corrupted input, partial batch failure, and
platform-specific packaging behavior.
