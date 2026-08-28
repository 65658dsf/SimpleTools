# Third-party notices

SimpleTools is distributed under `AGPL-3.0-or-later`. This file records the components that are
compiled into the desktop application or shipped in its release archives. Versions below are the
direct dependency versions pinned by `go.mod` and `frontend/package-lock.json`; transitive
dependencies remain available in the corresponding source checkout and lockfiles.

## Go and native components

| Component | Version | License | Source and license text | Use |
| --- | --- | --- | --- | --- |
| Wails | v2.15.0 | MIT | <https://github.com/wailsapp/wails/tree/v2.15.0>; `LICENSE` in the Wails source | Desktop shell and Go/JS bindings |
| go-fitz | v1.28.2 | AGPL-3.0-or-later | <https://github.com/gen2brain/go-fitz/tree/v1.28.2>; `COPYING` in the source | Go binding used for PDF rendering |
| MuPDF | bundled by go-fitz v1.28.2 | AGPL-3.0-or-later or commercial | <https://mupdf.com>; license and source offer in the go-fitz/MuPDF source distribution | PDF rasterization in `mupdf` builds |
| Noto Sans SC | 2.004-H2 subset (from `@fontsource/noto-sans-sc` v5.1.1) | OFL-1.1 | `internal/tools/assets/OFL-1.1.txt`; `internal/tools/assets/README.md`; <https://github.com/notofonts/noto-cjk> | Embedded Simplified Chinese fallback font for MuPDF CJK font requests |
| gen2brain/webp | v0.6.4 | MIT | <https://github.com/gen2brain/webp/tree/v0.6.4>; `LICENSE` in the source | WebP Go binding |
| libwebp | bundled by gen2brain/webp v0.6.4 | BSD-3-Clause | <https://chromium.googlesource.com/webm/libwebp/>; `lib/LICENSE.libwebp` in the source | WebP codec |
| gen2brain/avif | v0.6.0 | MIT | <https://github.com/gen2brain/avif/tree/v0.6.0>; `LICENSE` in the source | AVIF Go binding |
| libavif | bundled by gen2brain/avif v0.6.0 | BSD-2-Clause | <https://github.com/AOMediaCodec/libavif>; `lib/LICENSE.libavif` in the source | AVIF container codec |
| libaom | bundled by gen2brain/avif v0.6.0 | BSD-2-Clause | <https://aomediacodec.github.io/av1/>; `lib/LICENSE.aom` in the source | AV1 encoder used by AVIF |
| dav1d and bundled codec support | bundled by gen2brain/avif v0.6.0 | BSD-2-Clause and component notices | <https://code.videolan.org/videolan/dav1d>; notices in the avif source `lib/LICENSE*` files | AV1 decoder support |

The Go module graph also includes indirect MIT, BSD, Apache-2.0 and ISC components. The exact
versions and integrity hashes are in `go.mod` and `go.sum`; release source archives must retain
those files so the complete transitive inventory can be reproduced. Wails' WebView2 bootstrapper
is downloaded by the Windows installer and remains subject to Microsoft's terms.

## Frontend components

The frontend runtime and build dependencies are pinned in `frontend/package-lock.json` and include
Vue 3, Vite, TypeScript, Pinia, Vue Router, vue-i18n, lucide-vue-next, Vitest and shadcn-vue.
Their published packages are MIT-licensed unless their package metadata states otherwise. The
lockfile is the authoritative version and integrity inventory for source and release audits.

## AGPL source and notices

Native MuPDF builds are covered by the GNU AGPL obligations. Every source distribution and release
archive must include:

1. `LICENSE` (the complete AGPL-3.0 text);
2. this notice file and the corresponding dependency license files listed above; and
3. the complete corresponding source for the SimpleTools build, including the exact Go module and
   frontend lockfiles and the MuPDF/go-fitz source or a written offer valid for the applicable
   distribution period.

The release workflow copies `LICENSE` and this file into portable archives and macOS app resources,
and publishes them beside signed release assets. Do not publish a signed artifact until the
generated notice inventory and the MuPDF source offer have been reviewed for that release.
