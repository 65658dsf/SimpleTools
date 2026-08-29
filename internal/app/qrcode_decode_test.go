package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/65658dsf/SimpleTools/internal/tools"
)

func writeQRCodePNG(t *testing.T, path string, options tools.QRCodeOptions) int64 {
	t.Helper()
	img, err := tools.RenderQRCode(options)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

func appQRCodeOptions() tools.QRCodeOptions {
	return tools.QRCodeOptions{
		Text:            "SimpleTools 扫码测试 ✅",
		Size:            384,
		ErrorCorrection: "quartile",
		Foreground:      "#18212F",
		Background:      "#FAFBFD",
	}
}

func TestDecodeQRCodeReturnsTextAndImageMetadata(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "scan.png")
	options := appQRCodeOptions()
	size := writeQRCodePNG(t, path, options)

	result, err := New().DecodeQRCode(path)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != canonical {
		t.Fatalf("path = %q, want canonical %q", result.Path, canonical)
	}
	if result.Width != options.Size || result.Height != options.Size {
		t.Fatalf("dimensions = %dx%d, want %dx%d", result.Width, result.Height, options.Size, options.Size)
	}
	if result.Format != "png" || result.Size != size {
		t.Fatalf("metadata format=%q size=%d, want png/%d", result.Format, result.Size, size)
	}
	if result.Text != options.Text || result.TextBytes != len(options.Text) || result.Truncated {
		t.Fatalf("decoded result = %#v", result)
	}
}

func TestDecodeQRCodeUsesImageConcurrencySlot(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "scan.png")
	writeQRCodePNG(t, path, appQRCodeOptions())

	app := New()
	app.imageSlots = make(chan struct{}, 1)
	app.imageSlots <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app.ctx = ctx

	if _, err := app.DecodeQRCode(path); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected occupied image slot to honor cancellation, got %v", err)
	}
}

func TestDecodeQRCodeCanonicalizesSymlinkInput(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "target.png")
	writeQRCodePNG(t, target, appQRCodeOptions())
	alias := filepath.Join(d, "alias.png")
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	result, err := New().DecodeQRCode(alias)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != canonical {
		t.Fatalf("path = %q, want canonical target %q", result.Path, canonical)
	}
}

func TestDecodeQRCodeRejectsImageWithoutQRCode(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "plain.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	plain := image.NewNRGBA(image.Rect(0, 0, 240, 160))
	for y := 0; y < plain.Bounds().Dy(); y++ {
		for x := 0; x < plain.Bounds().Dx(); x++ {
			plain.SetNRGBA(x, y, color.NRGBA{R: 250, G: 250, B: 250, A: 255})
		}
	}
	if err := png.Encode(f, plain); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := New().DecodeQRCode(path); err == nil || !strings.Contains(err.Error(), "no QR code found") {
		t.Fatalf("expected no-QR error, got %v", err)
	}
}

func TestDecodeQRCodeRejectsCorruptAndUnsupportedInputs(t *testing.T) {
	d := t.TempDir()
	corrupt := filepath.Join(d, "broken.png")
	if err := os.WriteFile(corrupt, []byte("not a PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New().DecodeQRCode(corrupt); err == nil || !strings.Contains(err.Error(), "decode QR code image") {
		t.Fatalf("expected corrupt-image error, got %v", err)
	}

	unsupported := filepath.Join(d, "payload.txt")
	if err := os.WriteFile(unsupported, []byte("QR text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New().DecodeQRCode(unsupported); err == nil || !strings.Contains(err.Error(), "unsupported QR code image type") {
		t.Fatalf("expected unsupported-type error, got %v", err)
	}

	if _, err := New().DecodeQRCode(filepath.Join(d, "missing.png")); err == nil {
		t.Fatal("expected missing-file error")
	}
}

func TestDecodeQRCodeRejectsOversizedPNGBeforePixelAllocation(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "oversized.png")
	if err := os.WriteFile(path, oversizedPNGHeader(9000, 9000), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New().DecodeQRCode(path); err == nil || !strings.Contains(err.Error(), "safety limit") {
		t.Fatalf("expected pre-decode safety error, got %v", err)
	}
}

func TestDecodeQRCodeRejectsOversizedFileBeforeReading(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "oversized-file.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(tools.MaxQRCodeDecodeBytes + 1); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := New().DecodeQRCode(path); err == nil || !strings.Contains(err.Error(), "file safety limit") {
		t.Fatalf("expected file safety error, got %v", err)
	}
}

func TestPreviewImageHonorsPixelLimitBeforeDecode(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "oversized-preview.png")
	if err := os.WriteFile(path, oversizedPNGHeader(9000, 9000), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New().PreviewImage(path, PreviewOptions{MaxPixels: tools.MaxQRCodeDecodePixels}); err == nil || !strings.Contains(err.Error(), "safety limit") {
		t.Fatalf("expected preview safety error, got %v", err)
	}
}

func oversizedPNGHeader(width, height uint32) []byte {
	var out bytes.Buffer
	out.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	var ihdr [13]byte
	binary.BigEndian.PutUint32(ihdr[0:4], width)
	binary.BigEndian.PutUint32(ihdr[4:8], height)
	ihdr[8] = 8
	ihdr[9] = 2
	writePNGChunk(&out, "IHDR", ihdr[:])
	writePNGChunk(&out, "IEND", nil)
	return out.Bytes()
}

func writePNGChunk(out *bytes.Buffer, kind string, payload []byte) {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(payload)))
	out.Write(length[:])
	out.WriteString(kind)
	out.Write(payload)
	crc := crc32.NewIEEE()
	_, _ = crc.Write([]byte(kind))
	_, _ = crc.Write(payload)
	var checksum [4]byte
	binary.BigEndian.PutUint32(checksum[:], crc.Sum32())
	out.Write(checksum[:])
}

func TestDecodeQRCodeRejectsNonRegularPath(t *testing.T) {
	d := filepath.Join(t.TempDir(), "folder.png")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := New().DecodeQRCode(d); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("expected non-regular path error, got %v", err)
	}
}

func TestBoundQRCodeDecodeResultTextTracksTruncation(t *testing.T) {
	input := strings.Repeat("解析✅", 30000)
	bounded, textBytes, truncated, err := boundQRCodeDecodeResultText(input)
	if err != nil {
		t.Fatal(err)
	}
	if textBytes != len(input) || !truncated || len(bounded) > tools.MaxQRCodeDecodeTextBytes {
		t.Fatalf("bounded bytes=%d original=%d truncated=%v", len(bounded), textBytes, truncated)
	}
	if !strings.HasPrefix(input, bounded) {
		t.Fatal("bounded QR text is not a prefix of the original text")
	}
}
