package tools

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDecodeQRCodeReturnsUTF8Payload(t *testing.T) {
	options := validQRCodeOptions()
	options.Text = "https://simpletools.local/测试?emoji=✅"
	img, err := RenderQRCode(options)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeQRCode(img)
	if err != nil {
		t.Fatal(err)
	}
	if got != options.Text {
		t.Fatalf("decoded text = %q, want %q", got, options.Text)
	}
}

func TestDecodeQRCodeSupportsInvertedColors(t *testing.T) {
	options := validQRCodeOptions()
	options.Text = "反色二维码"
	options.Foreground = "#F8FAFC"
	options.Background = "#101827"
	img, err := RenderQRCode(options)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeQRCode(img)
	if err != nil {
		t.Fatal(err)
	}
	if got != options.Text {
		t.Fatalf("decoded text = %q, want %q", got, options.Text)
	}
}

func TestDecodeQRCodeRejectsImageWithoutQRCode(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 240, 160))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 245, G: 245, B: 245, A: 255})
		}
	}
	if _, err := DecodeQRCode(img); err == nil || !strings.Contains(err.Error(), "no QR code found") {
		t.Fatalf("expected no-QR error, got %v", err)
	}
}

func TestDecodeQRCodeRejectsDamagedSymbol(t *testing.T) {
	options := validQRCodeOptions()
	img, err := RenderQRCode(options)
	if err != nil {
		t.Fatal(err)
	}
	damaged := image.NewNRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			damaged.Set(x, y, img.At(x, y))
		}
	}
	// Erase the center and one finder pattern so the detector cannot accept a
	// partially valid symbol by accident.
	for y := 0; y < damaged.Bounds().Dy(); y++ {
		for x := 0; x < damaged.Bounds().Dx(); x++ {
			if (x > 150 && x < 360 && y > 150 && y < 360) || (x < 150 && y < 150) {
				damaged.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}
	if _, err := DecodeQRCode(damaged); err == nil {
		t.Fatal("expected damaged QR code to be rejected")
	}
}

func TestDecodeQRCodeRejectsOversizedImageBeforeScanning(t *testing.T) {
	img := boundsOnlyImage{bounds: image.Rect(0, 0, 9000, 9000)}
	if _, err := DecodeQRCode(img); err == nil || !strings.Contains(err.Error(), "safety limit") {
		t.Fatalf("expected pixel safety error, got %v", err)
	}
}

func TestBoundQRCodeTextPreservesUTF8AndReportsOriginalLimit(t *testing.T) {
	input := strings.Repeat("水印✅", 30000)
	bounded, truncated, err := BoundQRCodeText(input, 64*1024)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || len(bounded) > 64*1024 || !utf8.ValidString(bounded) {
		t.Fatalf("unexpected bounded text: truncated=%v bytes=%d valid=%v", truncated, len(bounded), utf8.ValidString(bounded))
	}
	if !strings.HasPrefix(input, bounded) {
		t.Fatal("bounded text is not a prefix of the original payload")
	}
}

func TestBoundQRCodeTextRejectsInvalidUTF8(t *testing.T) {
	if _, _, err := BoundQRCodeText(string([]byte{0xff}), 10); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("expected invalid UTF-8 error, got %v", err)
	}
}
