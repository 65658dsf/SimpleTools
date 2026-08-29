package tools

import (
	"bytes"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func validQRCodeOptions() QRCodeOptions {
	return QRCodeOptions{
		Text:            "SimpleTools 二维码",
		Size:            256,
		ErrorCorrection: "medium",
		Foreground:      "#123456",
		Background:      "#F0E1D2",
	}
}

func TestQRCodeOptionsValidation(t *testing.T) {
	invalidUTF8 := string([]byte{0xff})
	tests := []struct {
		name    string
		mutate  func(*QRCodeOptions)
		message string
	}{
		{name: "missing text", mutate: func(o *QRCodeOptions) { o.Text = "" }, message: "text is required"},
		{name: "blank text", mutate: func(o *QRCodeOptions) { o.Text = "  \r\n\t" }, message: "text is required"},
		{name: "invalid UTF-8", mutate: func(o *QRCodeOptions) { o.Text = invalidUTF8 }, message: "valid UTF-8"},
		{name: "size below minimum", mutate: func(o *QRCodeOptions) { o.Size = 127 }, message: "between 128 and 2048"},
		{name: "size above maximum", mutate: func(o *QRCodeOptions) { o.Size = 2049 }, message: "between 128 and 2048"},
		{name: "unknown correction", mutate: func(o *QRCodeOptions) { o.ErrorCorrection = "maximum" }, message: "low, medium, quartile, or high"},
		{name: "short foreground", mutate: func(o *QRCodeOptions) { o.Foreground = "#fff" }, message: "#RRGGBB"},
		{name: "bad background", mutate: func(o *QRCodeOptions) { o.Background = "#GGGGGG" }, message: "invalid"},
		{name: "matching colors", mutate: func(o *QRCodeOptions) { o.Background = o.Foreground }, message: "must differ"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := validQRCodeOptions()
			test.mutate(&options)
			err := options.Validate()
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected error containing %q, got %v", test.message, err)
			}
		})
	}
}

func TestEncodeQRCodePNGUsesRequestedSizeAndColors(t *testing.T) {
	options := validQRCodeOptions()
	data, err := EncodeQRCodePNG(options)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.Bounds().Size(); got.X != options.Size || got.Y != options.Size {
		t.Fatalf("decoded size = %v, want %dx%d", got, options.Size, options.Size)
	}

	wantForeground := color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}
	wantBackground := color.RGBA{R: 0xf0, G: 0xe1, B: 0xd2, A: 0xff}
	foundForeground, foundBackground := false, false
	for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y && (!foundForeground || !foundBackground); y++ {
		for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
			pixel := color.RGBAModel.Convert(decoded.At(x, y)).(color.RGBA)
			foundForeground = foundForeground || pixel == wantForeground
			foundBackground = foundBackground || pixel == wantBackground
		}
	}
	if !foundForeground || !foundBackground {
		t.Fatalf("PNG colors foreground=%v background=%v", foundForeground, foundBackground)
	}
}

func TestEncodeQRCodePNGSupportsAllErrorCorrectionLevels(t *testing.T) {
	for _, level := range []string{"low", "medium", "quartile", "high"} {
		t.Run(level, func(t *testing.T) {
			options := validQRCodeOptions()
			options.ErrorCorrection = level
			if _, err := EncodeQRCodePNG(options); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEncodeQRCodePNGRejectsContentBeyondCapacity(t *testing.T) {
	options := validQRCodeOptions()
	options.Size = MaxQRCodeSize
	options.Text = strings.Repeat("x", 5000)
	if _, err := EncodeQRCodePNG(options); err == nil || !strings.Contains(err.Error(), "content too long") {
		t.Fatalf("expected capacity error, got %v", err)
	}
}
