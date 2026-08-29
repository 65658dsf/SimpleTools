package tools

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"
	"unicode/utf8"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	MinQRCodeSize = 128
	MaxQRCodeSize = 2048
)

// QRCodeOptions describes a deterministic text QR code. Size is the width and
// height of the PNG in pixels; colors use opaque #RRGGBB values.
type QRCodeOptions struct {
	Text            string `json:"text"`
	Size            int    `json:"size"`
	ErrorCorrection string `json:"errorCorrection"`
	Foreground      string `json:"foreground"`
	Background      string `json:"background"`
}

type normalizedQRCodeOptions struct {
	QRCodeOptions
	level      qrcode.RecoveryLevel
	foreground color.RGBA
	background color.RGBA
}

// Validate verifies that the options can be passed to the QR renderer. The
// encoder performs the final content-capacity check when rendering.
func (o QRCodeOptions) Validate() error {
	_, err := o.normalized()
	return err
}

func (o QRCodeOptions) normalized() (normalizedQRCodeOptions, error) {
	if strings.TrimSpace(o.Text) == "" {
		return normalizedQRCodeOptions{}, fmt.Errorf("QR code text is required")
	}
	if !utf8.ValidString(o.Text) {
		return normalizedQRCodeOptions{}, fmt.Errorf("QR code text must be valid UTF-8")
	}
	if o.Size < MinQRCodeSize || o.Size > MaxQRCodeSize {
		return normalizedQRCodeOptions{}, fmt.Errorf("QR code size must be between %d and %d pixels", MinQRCodeSize, MaxQRCodeSize)
	}

	var level qrcode.RecoveryLevel
	o.ErrorCorrection = strings.ToLower(strings.TrimSpace(o.ErrorCorrection))
	switch o.ErrorCorrection {
	case "low":
		level = qrcode.Low
	case "medium":
		level = qrcode.Medium
	case "quartile":
		level = qrcode.High
	case "high":
		level = qrcode.Highest
	default:
		return normalizedQRCodeOptions{}, fmt.Errorf("QR code error correction must be one of low, medium, quartile, or high")
	}

	foreground, err := parseQRCodeColor("foreground", o.Foreground)
	if err != nil {
		return normalizedQRCodeOptions{}, err
	}
	background, err := parseQRCodeColor("background", o.Background)
	if err != nil {
		return normalizedQRCodeOptions{}, err
	}
	if foreground == background {
		return normalizedQRCodeOptions{}, fmt.Errorf("QR code foreground and background colors must differ")
	}

	return normalizedQRCodeOptions{
		QRCodeOptions: o,
		level:         level,
		foreground:    foreground,
		background:    background,
	}, nil
}

// RenderQRCode creates an opaque, square QR code image without filesystem or
// network access. It rejects content that cannot fit in the selected symbol or
// whose minimum module grid is larger than the requested pixel size.
func RenderQRCode(options QRCodeOptions) (image.Image, error) {
	opts, err := options.normalized()
	if err != nil {
		return nil, err
	}
	code, err := qrcode.New(opts.Text, opts.level)
	if err != nil {
		return nil, fmt.Errorf("encode QR code text: %w", err)
	}
	code.ForegroundColor = opts.foreground
	code.BackgroundColor = opts.background
	img := code.Image(opts.Size)
	if width, height := img.Bounds().Dx(), img.Bounds().Dy(); width != opts.Size || height != opts.Size {
		return nil, fmt.Errorf("QR code content requires at least %d pixels at the selected error correction level; increase the size", max(width, height))
	}
	return img, nil
}

// EncodeQRCodePNG renders options and returns a metadata-free PNG.
func EncodeQRCodePNG(options QRCodeOptions) ([]byte, error) {
	img, err := RenderQRCode(options)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode QR code PNG: %w", err)
	}
	return buf.Bytes(), nil
}

func parseQRCodeColor(name, value string) (color.RGBA, error) {
	value = strings.TrimSpace(value)
	if len(value) != 7 || value[0] != '#' {
		return color.RGBA{}, fmt.Errorf("QR code %s color must use #RRGGBB", name)
	}
	parsed, err := strconv.ParseUint(value[1:], 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid QR code %s color %q", name, value)
	}
	return color.RGBA{
		R: uint8(parsed >> 16),
		G: uint8(parsed >> 8),
		B: uint8(parsed),
		A: 0xff,
	}, nil
}
