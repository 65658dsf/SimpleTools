package tools

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

func TestApplyTextWatermarkSupportsChinese(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 420, 240))
	background := color.NRGBA{R: 18, G: 31, B: 46, A: 255}
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			source.SetNRGBA(x, y, background)
		}
	}

	result, err := ApplyTextWatermark(source, WatermarkOptions{
		Text:       "测试水印",
		FontFamily: "noto-sans-sc",
		FontSize:   52,
		Color:      "#f4f7fb",
		Opacity:    0.85,
		Position:   "center",
		Rotation:   -18,
		Shadow:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Bounds() != source.Bounds() {
		t.Fatalf("bounds = %v, want %v", result.Bounds(), source.Bounds())
	}
	changed := 0
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			if result.At(x, y) != source.At(x, y) {
				changed++
			}
			if source.NRGBAAt(x, y) != background {
				t.Fatal("source image was mutated")
			}
		}
	}
	if changed < 100 {
		t.Fatalf("Chinese watermark changed only %d pixels", changed)
	}
}

func TestWatermarkNormalizationPreservesZeroMarginAndSpacing(t *testing.T) {
	opts, err := (WatermarkOptions{
		Text:     "edge",
		FontSize: 20,
		Color:    "#fff",
		Opacity:  1,
		Position: "top-left",
		Margin:   0,
		Tile:     true,
		Spacing:  0,
	}).normalized()
	if err != nil {
		t.Fatal(err)
	}
	if opts.Margin != 0 || opts.Spacing != 0 {
		t.Fatalf("zero values were replaced: margin=%d spacing=%d", opts.Margin, opts.Spacing)
	}
}

func TestScaleWatermarkOptionsKeepsRelativeStyle(t *testing.T) {
	opts, err := ScaleWatermarkOptions(WatermarkOptions{
		Text: "preview", FontSize: 80, Color: "#fff", Opacity: 0.5,
		Position: "bottom-right", Margin: 40, Spacing: 100,
	}, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	if opts.FontSize != 80 || opts.Margin != 10 || opts.Spacing != 25 || opts.previewScale != 0.25 {
		t.Fatalf("unexpected scaled options: %#v", opts)
	}
}

func TestScaleWatermarkOptionsPreservesSmallFontProportion(t *testing.T) {
	const previewScale = 960.0 / 4096.0
	sourceOptions, err := (WatermarkOptions{
		Text: "small", FontSize: 12, Color: "#fff", Opacity: 0.7,
		Position: "center", Rotation: -24,
	}).normalized()
	if err != nil {
		t.Fatal(err)
	}
	sourceStamp, err := prepareWatermarkStamp(sourceOptions)
	if err != nil {
		t.Fatal(err)
	}
	previewOptions, err := ScaleWatermarkOptions(sourceOptions, previewScale)
	if err != nil {
		t.Fatal(err)
	}
	previewStamp, err := prepareWatermarkStamp(previewOptions)
	if err != nil {
		t.Fatal(err)
	}
	wantWidth := maxInt(1, int(math.Round(float64(sourceStamp.Bounds().Dx())*previewScale)))
	wantHeight := maxInt(1, int(math.Round(float64(sourceStamp.Bounds().Dy())*previewScale)))
	if previewOptions.FontSize != 12 {
		t.Fatalf("source font size was changed to %d", previewOptions.FontSize)
	}
	if previewStamp.Bounds().Dx() != wantWidth || previewStamp.Bounds().Dy() != wantHeight {
		t.Fatalf("preview stamp = %v, want %dx%d from source stamp %v", previewStamp.Bounds(), wantWidth, wantHeight, sourceStamp.Bounds())
	}
}

func TestApplyTextWatermarkUsesVisibleDefaults(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 160, 80))
	result, err := ApplyTextWatermark(source, WatermarkOptions{Text: "default"})
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	for y := 0; y < result.Bounds().Dy() && !changed; y++ {
		for x := 0; x < result.Bounds().Dx(); x++ {
			_, _, _, alpha := result.At(x, y).RGBA()
			if alpha != 0 {
				changed = true
				break
			}
		}
	}
	if !changed {
		t.Fatal("default watermark is fully transparent")
	}
}

func TestApplyTextWatermarkHonorsAnchorAndPreservesUntouchedAlpha(t *testing.T) {
	background := color.NRGBA{R: 14, G: 25, B: 36, A: 90}
	source := image.NewNRGBA(image.Rect(0, 0, 360, 220))
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			source.SetNRGBA(x, y, background)
		}
	}
	result, err := ApplyTextWatermark(source, WatermarkOptions{
		Text: "corner", FontSize: 32, Color: "#ffffff", Opacity: 0.9,
		Position: "top-left", Margin: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	watermarked := result.(*image.NRGBA)
	if got := watermarked.NRGBAAt(350, 210); got != background {
		t.Fatalf("untouched alpha pixel = %#v, want %#v", got, background)
	}
	changedTopLeft := 0
	changedBottomRight := 0
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			if watermarked.NRGBAAt(x, y) == background {
				continue
			}
			if x < 180 && y < 110 {
				changedTopLeft++
			} else if x >= 180 && y >= 110 {
				changedBottomRight++
			}
		}
	}
	if changedTopLeft < 50 || changedBottomRight != 0 {
		t.Fatalf("unexpected anchored pixels: top-left=%d bottom-right=%d", changedTopLeft, changedBottomRight)
	}
}

func TestApplyTextWatermarkTilesAcrossImage(t *testing.T) {
	background := color.NRGBA{R: 32, G: 48, B: 64, A: 255}
	source := image.NewNRGBA(image.Rect(0, 0, 640, 420))
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			source.SetNRGBA(x, y, background)
		}
	}
	result, err := ApplyTextWatermark(source, WatermarkOptions{
		Text: "tile", FontSize: 30, Color: "#ffffff", Opacity: 0.6,
		Position: "center", Rotation: -20, Tile: true, Spacing: 45,
	})
	if err != nil {
		t.Fatal(err)
	}
	quadrants := [4]int{}
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			if result.At(x, y) == source.At(x, y) {
				continue
			}
			index := 0
			if x >= source.Bounds().Dx()/2 {
				index++
			}
			if y >= source.Bounds().Dy()/2 {
				index += 2
			}
			quadrants[index]++
		}
	}
	for index, changed := range quadrants {
		if changed < 20 {
			t.Fatalf("quadrant %d changed only %d pixels", index, changed)
		}
	}
}

func TestWatermarkTileCountAllowsDense8KLayout(t *testing.T) {
	opts, err := (WatermarkOptions{
		Text: "i", FontSize: 12, Color: "#fff", Opacity: 0.6,
		Position: "center", Tile: true, Spacing: 20,
	}).normalized()
	if err != nil {
		t.Fatal(err)
	}
	stamp, err := prepareWatermarkStamp(opts)
	if err != nil {
		t.Fatal(err)
	}
	count, err := watermarkTileCount(8192, 8192, stamp.Bounds().Dx(), stamp.Bounds().Dy(), opts.Spacing)
	if err != nil {
		t.Fatal(err)
	}
	if count <= 10000 || count > maxWatermarkTiles {
		t.Fatalf("unexpected 8K tile count %d", count)
	}
}

func TestWatermarkTileCountMatchesStaggeredLayout(t *testing.T) {
	count, err := watermarkTileCount(100, 80, 20, 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	if count != 27 {
		t.Fatalf("tile count = %d, want 27", count)
	}
}

func TestApplyTextWatermarkRejectsUnsafeTileCount(t *testing.T) {
	source := boundsOnlyImage{bounds: image.Rect(0, 0, 8192, 8192)}
	_, err := ApplyTextWatermark(source, WatermarkOptions{
		Text: "i", FontSize: 6, Color: "#fff", Opacity: 0.6,
		Position: "center", Tile: true, Spacing: 0,
	})
	if err == nil || !strings.Contains(err.Error(), "placements; safe limit is") {
		t.Fatalf("expected tile safety error, got %v", err)
	}
}

func TestApplyTextWatermarkRejectsOversizedStamp(t *testing.T) {
	_, err := ApplyTextWatermark(image.NewNRGBA(image.Rect(0, 0, 32, 32)), WatermarkOptions{
		Text:     strings.Repeat("W", maxWatermarkRunes),
		FontSize: 512,
		Color:    "#fff",
		Opacity:  1,
		Position: "center",
	})
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected oversized watermark error, got %v", err)
	}
}

func TestApplyTextWatermarkRejectsOversizedRotatedStamp(t *testing.T) {
	// This unrotated layer remains below the pixel cap, while a 45-degree
	// rotation expands its bounding box beyond the cap.
	_, err := ApplyTextWatermark(image.NewNRGBA(image.Rect(0, 0, 32, 32)), WatermarkOptions{
		Text:     strings.Repeat("M", 8),
		FontSize: 512,
		Color:    "#fff",
		Opacity:  1,
		Position: "center",
		Rotation: 45,
	})
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected oversized rotated watermark error, got %v", err)
	}
}

func TestApplyTextWatermarkValidatesStyle(t *testing.T) {
	tests := []struct {
		name    string
		options WatermarkOptions
		message string
	}{
		{name: "empty text", options: WatermarkOptions{}, message: "text is required"},
		{name: "bad color", options: WatermarkOptions{Text: "x", Color: "red"}, message: "color"},
		{name: "bad font", options: WatermarkOptions{Text: "x", FontFamily: "Comic Sans"}, message: "font family"},
		{name: "bad opacity", options: WatermarkOptions{Text: "x", Opacity: 1.1}, message: "opacity"},
		{name: "bad position", options: WatermarkOptions{Text: "x", Position: "outside"}, message: "position"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ApplyTextWatermark(image.NewNRGBA(image.Rect(0, 0, 16, 16)), test.options)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected error containing %q, got %v", test.message, err)
			}
		})
	}
}

type boundsOnlyImage struct {
	bounds image.Rectangle
}

func (i boundsOnlyImage) ColorModel() color.Model { return color.NRGBAModel }
func (i boundsOnlyImage) Bounds() image.Rectangle { return i.bounds }
func (i boundsOnlyImage) At(int, int) color.Color { return color.NRGBA{} }
