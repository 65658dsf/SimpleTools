package tools

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	defaultWatermarkFontSize = 32
	defaultWatermarkOpacity  = 0.68
	maxWatermarkRunes        = 2000
	maxWatermarkTiles        = 100000
	maxWatermarkStampPixels  = 4 * 1024 * 1024
	maxWatermarkStampSide    = 8192
)

// WatermarkOptions describes a deterministic text watermark. FontSize, Margin,
// and Spacing use source-image pixels; Opacity is in the inclusive range 0..1.
type WatermarkOptions struct {
	Text         string  `json:"text"`
	FontFamily   string  `json:"fontFamily,omitempty"`
	FontSize     int     `json:"fontSize,omitempty"`
	Color        string  `json:"color,omitempty"`
	Opacity      float64 `json:"opacity,omitempty"`
	Position     string  `json:"position,omitempty"`
	Margin       int     `json:"margin,omitempty"`
	Rotation     float64 `json:"rotation,omitempty"`
	Tile         bool    `json:"tile,omitempty"`
	Spacing      int     `json:"spacing,omitempty"`
	Shadow       bool    `json:"shadow,omitempty"`
	previewScale float64
}

//go:embed assets/NotoSansSC-Regular.ttf
var watermarkFontData []byte

var (
	watermarkFontOnce sync.Once
	watermarkFont     *opentype.Font
	watermarkFontErr  error
)

func (o WatermarkOptions) normalized() (WatermarkOptions, error) {
	if strings.TrimSpace(o.Text) == "" {
		return WatermarkOptions{}, fmt.Errorf("watermark text is required")
	}
	if !utf8.ValidString(o.Text) {
		return WatermarkOptions{}, fmt.Errorf("watermark text must be valid UTF-8")
	}
	if utf8.RuneCountInString(o.Text) > maxWatermarkRunes {
		return WatermarkOptions{}, fmt.Errorf("watermark text cannot exceed %d characters", maxWatermarkRunes)
	}

	fontFamily := strings.ToLower(strings.TrimSpace(o.FontFamily))
	fontFamily = strings.NewReplacer("_", "-", " ", "-").Replace(fontFamily)
	switch fontFamily {
	case "", "noto-sans-sc", "notosanssc", "sans", "sans-serif", "system":
		o.FontFamily = "noto-sans-sc"
	default:
		return WatermarkOptions{}, fmt.Errorf("unsupported watermark font family %q", o.FontFamily)
	}

	if o.FontSize == 0 {
		o.FontSize = defaultWatermarkFontSize
	}
	if o.FontSize < 6 || o.FontSize > 512 {
		return WatermarkOptions{}, fmt.Errorf("watermark font size must be between 6 and 512 pixels")
	}
	if strings.TrimSpace(o.Color) == "" {
		o.Color = "#FFFFFF"
	}
	if _, err := parseWatermarkColor(o.Color, 1); err != nil {
		return WatermarkOptions{}, err
	}
	if math.IsNaN(o.Opacity) || math.IsInf(o.Opacity, 0) || o.Opacity < 0 || o.Opacity > 1 {
		return WatermarkOptions{}, fmt.Errorf("watermark opacity must be between 0 and 1")
	}
	if o.Opacity == 0 {
		o.Opacity = defaultWatermarkOpacity
	}
	if o.Position == "" {
		o.Position = "bottom-right"
	}
	o.Position = strings.ToLower(strings.TrimSpace(o.Position))
	switch o.Position {
	case "top-left", "top-center", "top-right", "center-left", "center", "center-right", "bottom-left", "bottom-center", "bottom-right":
	default:
		return WatermarkOptions{}, fmt.Errorf("unsupported watermark position %q", o.Position)
	}
	if o.Margin < 0 || o.Margin > 10000 {
		return WatermarkOptions{}, fmt.Errorf("watermark margin must be between 0 and 10000 pixels")
	}
	if math.IsNaN(o.Rotation) || math.IsInf(o.Rotation, 0) || o.Rotation < -360 || o.Rotation > 360 {
		return WatermarkOptions{}, fmt.Errorf("watermark rotation must be between -360 and 360 degrees")
	}
	o.Rotation = math.Mod(o.Rotation, 360)
	if o.Spacing < 0 || o.Spacing > 10000 {
		return WatermarkOptions{}, fmt.Errorf("watermark spacing must be between 0 and 10000 pixels")
	}
	return o, nil
}

// ScaleWatermarkOptions converts source-pixel style values for a bounded
// preview. Text is rasterized at the source font size before the finished
// stamp is reduced, avoiding a minimum-font-size distortion in small previews.
func ScaleWatermarkOptions(options WatermarkOptions, scale float64) (WatermarkOptions, error) {
	opts, err := options.normalized()
	if err != nil {
		return WatermarkOptions{}, err
	}
	if math.IsNaN(scale) || math.IsInf(scale, 0) || scale <= 0 {
		return WatermarkOptions{}, fmt.Errorf("watermark preview scale must be positive")
	}
	if scale > 1 {
		scale = 1
	}
	opts.Margin = maxInt(0, int(math.Round(float64(opts.Margin)*scale)))
	opts.Spacing = maxInt(0, int(math.Round(float64(opts.Spacing)*scale)))
	opts.previewScale = scale
	return opts, nil
}

// ApplyTextWatermark renders a text watermark onto a copy of src. It never
// mutates the source image and preserves its dimensions and alpha channel.
func ApplyTextWatermark(src image.Image, options WatermarkOptions) (image.Image, error) {
	if src == nil {
		return nil, fmt.Errorf("watermark source image is required")
	}
	opts, err := options.normalized()
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 {
		return nil, fmt.Errorf("watermark source dimensions must be positive")
	}

	stamp, err := prepareWatermarkStamp(opts)
	if err != nil {
		return nil, err
	}
	tileCount := 0
	if opts.Tile {
		tileCount, err = watermarkTileCount(bounds.Dx(), bounds.Dy(), stamp.Bounds().Dx(), stamp.Bounds().Dy(), opts.Spacing)
		if err != nil {
			return nil, err
		}
	}
	dst := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Src)
	if opts.Tile {
		if drawn := drawTiledWatermark(dst, stamp, opts.Spacing); drawn != tileCount {
			return nil, fmt.Errorf("watermark tiling count changed during rendering: counted %d, drew %d", tileCount, drawn)
		}
		return dst, nil
	}
	x, y := watermarkPosition(dst.Bounds().Dx(), dst.Bounds().Dy(), stamp.Bounds().Dx(), stamp.Bounds().Dy(), opts.Position, opts.Margin)
	draw.Draw(dst, image.Rect(x, y, x+stamp.Bounds().Dx(), y+stamp.Bounds().Dy()), stamp, stamp.Bounds().Min, draw.Over)
	return dst, nil
}

func prepareWatermarkStamp(opts WatermarkOptions) (*image.NRGBA, error) {
	stamp, err := renderWatermarkStamp(opts)
	if err != nil {
		return nil, err
	}
	stamp, err = rotateWatermarkStamp(stamp, opts.Rotation)
	if err != nil {
		return nil, err
	}
	return scaleWatermarkStamp(stamp, opts.previewScale)
}

func embeddedWatermarkFont() (*opentype.Font, error) {
	watermarkFontOnce.Do(func() {
		watermarkFont, watermarkFontErr = opentype.Parse(watermarkFontData)
		if watermarkFontErr != nil {
			watermarkFontErr = fmt.Errorf("parse embedded watermark font: %w", watermarkFontErr)
		}
	})
	return watermarkFont, watermarkFontErr
}

func renderWatermarkStamp(opts WatermarkOptions) (*image.NRGBA, error) {
	parsedFont, err := embeddedWatermarkFont()
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{Size: float64(opts.FontSize), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, fmt.Errorf("create watermark font face: %w", err)
	}
	defer face.Close()

	lines := strings.Split(strings.ReplaceAll(opts.Text, "\r\n", "\n"), "\n")
	metrics := face.Metrics()
	lineHeight := metrics.Height.Ceil()
	if lineHeight < 1 {
		lineHeight = opts.FontSize
	}
	width := 1
	drawer := &font.Drawer{Face: face}
	for _, line := range lines {
		if measured := drawer.MeasureString(line).Ceil(); measured > width {
			width = measured
		}
	}
	shadowOffset := 0
	if opts.Shadow {
		shadowOffset = maxInt(1, opts.FontSize/14)
	}
	padding := maxInt(2, opts.FontSize/12) + shadowOffset
	stampWidth := width + padding*2
	stampHeight := lineHeight*len(lines) + padding*2
	if err := validateWatermarkStampSize(stampWidth, stampHeight); err != nil {
		return nil, err
	}
	stamp := image.NewNRGBA(image.Rect(0, 0, stampWidth, stampHeight))
	textColor, err := parseWatermarkColor(opts.Color, opts.Opacity)
	if err != nil {
		return nil, err
	}
	baseline := padding + metrics.Ascent.Ceil()
	if opts.Shadow {
		shadowAlpha := uint8(math.Round(float64(textColor.A) * 0.58))
		shadow := image.NewUniform(color.NRGBA{A: shadowAlpha})
		for index, line := range lines {
			dot := fixed.P(padding+shadowOffset, baseline+index*lineHeight+shadowOffset)
			shadowDrawer := font.Drawer{Dst: stamp, Src: shadow, Face: face, Dot: dot}
			shadowDrawer.DrawString(line)
		}
	}
	text := image.NewUniform(textColor)
	for index, line := range lines {
		dot := fixed.P(padding, baseline+index*lineHeight)
		textDrawer := font.Drawer{Dst: stamp, Src: text, Face: face, Dot: dot}
		textDrawer.DrawString(line)
	}
	return stamp, nil
}

func parseWatermarkColor(value string, opacity float64) (color.NRGBA, error) {
	hex := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(hex) == 3 {
		hex = strings.Repeat(hex[0:1], 2) + strings.Repeat(hex[1:2], 2) + strings.Repeat(hex[2:3], 2)
	}
	if len(hex) != 6 {
		return color.NRGBA{}, fmt.Errorf("watermark color must use #RGB or #RRGGBB")
	}
	channels := [3]uint8{}
	for index := range channels {
		parsed, err := strconv.ParseUint(hex[index*2:index*2+2], 16, 8)
		if err != nil {
			return color.NRGBA{}, fmt.Errorf("invalid watermark color %q", value)
		}
		channels[index] = uint8(parsed)
	}
	return color.NRGBA{R: channels[0], G: channels[1], B: channels[2], A: uint8(math.Round(opacity * 255))}, nil
}

func rotateWatermarkStamp(src *image.NRGBA, degrees float64) (*image.NRGBA, error) {
	if math.Abs(degrees) < 0.0001 {
		return src, nil
	}
	radians := degrees * math.Pi / 180
	cosine, sine := math.Cos(radians), math.Sin(radians)
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dw := maxInt(1, int(math.Ceil(math.Abs(float64(sw)*cosine)+math.Abs(float64(sh)*sine))))
	dh := maxInt(1, int(math.Ceil(math.Abs(float64(sw)*sine)+math.Abs(float64(sh)*cosine))))
	if err := validateWatermarkStampSize(dw, dh); err != nil {
		return nil, fmt.Errorf("rotated %w", err)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	srcCX, srcCY := float64(sw-1)/2, float64(sh-1)/2
	dstCX, dstCY := float64(dw-1)/2, float64(dh-1)/2
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			dx, dy := float64(x)-dstCX, float64(y)-dstCY
			sx := cosine*dx + sine*dy + srcCX
			sy := -sine*dx + cosine*dy + srcCY
			dst.SetNRGBA(x, y, bilinearNRGBA(src, sx, sy))
		}
	}
	return dst, nil
}

func scaleWatermarkStamp(src *image.NRGBA, scale float64) (*image.NRGBA, error) {
	if scale == 0 || math.Abs(scale-1) < 0.0001 {
		return src, nil
	}
	if math.IsNaN(scale) || math.IsInf(scale, 0) || scale <= 0 || scale > 1 {
		return nil, fmt.Errorf("watermark preview scale must be between 0 and 1")
	}
	width := maxInt(1, int(math.Round(float64(src.Bounds().Dx())*scale)))
	height := maxInt(1, int(math.Round(float64(src.Bounds().Dy())*scale)))
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	return dst, nil
}

func validateWatermarkStampSize(width, height int) error {
	if width < 1 || height < 1 {
		return fmt.Errorf("watermark stamp dimensions must be positive")
	}
	if width > maxWatermarkStampSide || height > maxWatermarkStampSide || int64(width)*int64(height) > maxWatermarkStampPixels {
		return fmt.Errorf("watermark text produces an image that is too large")
	}
	return nil
}

func bilinearNRGBA(src *image.NRGBA, x, y float64) color.NRGBA {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	weights := [4]float64{(1 - fx) * (1 - fy), fx * (1 - fy), (1 - fx) * fy, fx * fy}
	points := [4]image.Point{{X: x0, Y: y0}, {X: x0 + 1, Y: y0}, {X: x0, Y: y0 + 1}, {X: x0 + 1, Y: y0 + 1}}
	var alpha, red, green, blue float64
	for index, point := range points {
		if !point.In(src.Bounds()) {
			continue
		}
		pixel := src.NRGBAAt(point.X, point.Y)
		weight := weights[index]
		a := float64(pixel.A) / 255
		alpha += float64(pixel.A) * weight
		red += float64(pixel.R) * a * weight
		green += float64(pixel.G) * a * weight
		blue += float64(pixel.B) * a * weight
	}
	if alpha <= 0 {
		return color.NRGBA{}
	}
	alphaFraction := alpha / 255
	return color.NRGBA{
		R: watermarkByte(red / alphaFraction),
		G: watermarkByte(green / alphaFraction),
		B: watermarkByte(blue / alphaFraction),
		A: watermarkByte(alpha),
	}
}

func watermarkByte(value float64) uint8 {
	return uint8(math.Round(math.Max(0, math.Min(255, value))))
}

func watermarkTileCount(canvasWidth, canvasHeight, stampWidth, stampHeight, spacing int) (int, error) {
	if canvasWidth < 1 || canvasHeight < 1 || stampWidth < 1 || stampHeight < 1 {
		return 0, fmt.Errorf("watermark tiling dimensions must be positive")
	}
	stepX, stepY := int64(stampWidth)+int64(spacing), int64(stampHeight)+int64(spacing)
	if stepX < 1 || stepY < 1 {
		return 0, fmt.Errorf("watermark tiling step must be positive")
	}
	startX, startY := -stepX/2, -stepY/2
	rows := watermarkPlacementCount(canvasHeight, startY, stepY)
	evenRows, oddRows := (rows+1)/2, rows/2
	evenColumns := watermarkPlacementCount(canvasWidth, startX, stepX)
	oddColumns := watermarkPlacementCount(canvasWidth, startX+stepX/2, stepX)

	total := uint64(0)
	for _, group := range [][2]uint64{{evenRows, evenColumns}, {oddRows, oddColumns}} {
		rowsInGroup, columnsInGroup := group[0], group[1]
		if rowsInGroup != 0 && columnsInGroup > ^uint64(0)/rowsInGroup {
			return 0, fmt.Errorf("watermark tiling requires more than the safe limit of %d placements", maxWatermarkTiles)
		}
		placements := rowsInGroup * columnsInGroup
		if placements > ^uint64(0)-total {
			return 0, fmt.Errorf("watermark tiling requires more than the safe limit of %d placements", maxWatermarkTiles)
		}
		total += placements
	}
	if total > uint64(maxWatermarkTiles) {
		return 0, fmt.Errorf("watermark tiling requires %d placements; safe limit is %d", total, maxWatermarkTiles)
	}
	return int(total), nil
}

func watermarkPlacementCount(limit int, start, step int64) uint64 {
	if int64(limit) <= start {
		return 0
	}
	distance := uint64(limit) + uint64(-start)
	return (distance + uint64(step) - 1) / uint64(step)
}

func drawTiledWatermark(dst *image.NRGBA, stamp image.Image, spacing int) int {
	stampWidth, stampHeight := stamp.Bounds().Dx(), stamp.Bounds().Dy()
	stepX, stepY := maxInt(1, stampWidth+spacing), maxInt(1, stampHeight+spacing)
	startX, startY := -stepX/2, -stepY/2
	drawn := 0
	row := 0
	for y := startY; y < dst.Bounds().Dy(); y, row = y+stepY, row+1 {
		offset := 0
		if row%2 != 0 {
			offset = stepX / 2
		}
		for x := startX + offset; x < dst.Bounds().Dx(); x += stepX {
			draw.Draw(dst, image.Rect(x, y, x+stampWidth, y+stampHeight), stamp, stamp.Bounds().Min, draw.Over)
			drawn++
		}
	}
	return drawn
}

func watermarkPosition(canvasWidth, canvasHeight, stampWidth, stampHeight int, position string, margin int) (int, int) {
	horizontal, vertical := "center", "center"
	parts := strings.Split(position, "-")
	if len(parts) == 1 {
		horizontal, vertical = "center", parts[0]
	} else {
		vertical, horizontal = parts[0], parts[1]
	}
	x := positionedAxis(canvasWidth, stampWidth, horizontal, margin)
	y := positionedAxis(canvasHeight, stampHeight, vertical, margin)
	return x, y
}

func positionedAxis(canvas, item int, alignment string, margin int) int {
	if item >= canvas {
		return (canvas - item) / 2
	}
	switch alignment {
	case "left", "top":
		return minInt(margin, canvas-item)
	case "right", "bottom":
		return maxInt(0, canvas-item-margin)
	default:
		return (canvas - item) / 2
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
