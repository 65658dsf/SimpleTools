package tools

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func testImage() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 20), G: uint8(y * 30), B: 180, A: uint8(100 + x*15)})
		}
	}
	return img
}

func TestSupportedCodecRoundTrip(t *testing.T) {
	for _, format := range []Format{FormatPNG, FormatJPEG, FormatWebP, FormatAVIF} {
		t.Run(string(format), func(t *testing.T) {
			data, err := EncodeBytes(testImage(), format, EncodeOptions{Quality: 80})
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := Decode(bytes.NewReader(data), string(format))
			if err != nil {
				t.Fatal(err)
			}
			if got := decoded.Bounds().Size(); got != (image.Point{X: 8, Y: 6}) {
				t.Fatalf("decoded size %v", got)
			}
		})
	}
}

func TestCompressionTargetSize(t *testing.T) {
	data, err := EncodeBytes(testImage(), FormatJPEG, EncodeOptions{Quality: 95})
	if err != nil {
		t.Fatal(err)
	}
	result, err := CompressImage(testImage(), FormatJPEG, 95, int64(len(data)/2), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Data) == 0 || result.Quality < 1 || result.Quality > 95 {
		t.Fatalf("unexpected result: quality=%d bytes=%d", result.Quality, len(result.Data))
	}
}
