package tools

import (
	"bytes"
	_ "embed"
	"testing"
)

// Keep a non-native test for the asset so a release cannot silently omit or
// replace the fallback font while the CGO/MuPDF test remains platform-only.
//
//go:embed assets/NotoSansSC-Regular.ttf
var testCJKFontData []byte

func TestEmbeddedCJKFallbackFont(t *testing.T) {
	if len(testCJKFontData) < 100_000 {
		t.Fatalf("embedded CJK fallback font is unexpectedly small: %d bytes", len(testCJKFontData))
	}
	if !bytes.Equal(testCJKFontData[:4], []byte{0, 1, 0, 0}) && !bytes.Equal(testCJKFontData[:4], []byte("OTTO")) {
		t.Fatalf("embedded CJK fallback font is not a TrueType/OpenType font (magic %q)", testCJKFontData[:4])
	}
}
