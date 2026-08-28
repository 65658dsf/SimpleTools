package tools

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/gen2brain/avif"
	"github.com/gen2brain/webp"
)

type Format string

const (
	FormatPNG  Format = "png"
	FormatJPEG Format = "jpeg"
	FormatWebP Format = "webp"
	FormatAVIF Format = "avif"
)

func ParseFormat(v string) (Format, error) {
	v = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(v)), ".")
	switch v {
	case "png":
		return FormatPNG, nil
	case "jpg", "jpeg":
		return FormatJPEG, nil
	case "webp":
		return FormatWebP, nil
	case "avif":
		return FormatAVIF, nil
	default:
		return "", fmt.Errorf("unsupported image format %q", v)
	}
}

func FormatFromPath(p string) (Format, error) { return ParseFormat(filepath.Ext(p)) }

// Decode reads a supported image and applies the orientation stored in JPEG EXIF.
// The codec packages use their own format sniffers, so the extension is only used
// to select a helpful error when the bytes are not a supported image.
func Decode(r io.Reader, ext string) (image.Image, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxImageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxImageBytes {
		return nil, fmt.Errorf("image exceeds the %d MiB safety limit", maxImageBytes/(1024*1024))
	}

	format, formatErr := ParseFormat(ext)
	if formatErr != nil {
		return nil, formatErr
	}
	var img image.Image
	switch format {
	case FormatPNG, FormatJPEG:
		img, _, err = image.Decode(bytes.NewReader(data))
	case FormatWebP:
		img, err = webp.Decode(bytes.NewReader(data), webp.Options{AutoRotate: true})
	case FormatAVIF:
		img, err = avif.Decode(bytes.NewReader(data), avif.Options{AutoRotate: true})
	}
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", format, err)
	}
	if format == FormatJPEG {
		img = ApplyOrientation(img, JPEGOrientation(data))
	}
	return img, nil
}

type EncodeOptions struct {
	Quality  int
	Lossless bool
}

func (o EncodeOptions) normalized() EncodeOptions {
	if o.Quality < 1 {
		o.Quality = 85
	}
	if o.Quality > 100 {
		o.Quality = 100
	}
	return o
}

func Encode(w io.Writer, img image.Image, f Format, o EncodeOptions) error {
	o = o.normalized()
	switch f {
	case FormatPNG:
		return png.Encode(w, img)
	case FormatJPEG:
		return jpeg.Encode(w, flattenOnWhite(img), &jpeg.Options{Quality: o.Quality})
	case FormatWebP:
		return webp.Encode(w, img, webp.Options{Quality: o.Quality, Lossless: o.Lossless})
	case FormatAVIF:
		return avif.Encode(w, img, avif.Options{Quality: o.Quality, QualityAlpha: o.Quality, Lossless: o.Lossless})
	default:
		return fmt.Errorf("unsupported output format %q", f)
	}
}

func EncodeBytes(img image.Image, f Format, o EncodeOptions) ([]byte, error) {
	var buf bytes.Buffer
	if err := Encode(&buf, img, f, o); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const maxImageBytes = 512 << 20

func flattenOnWhite(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			if a == 0xffff {
				dst.SetRGBA(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xff})
				continue
			}
			inv := uint64(0xffff - a)
			dst.SetRGBA(x, y, color.RGBA{
				uint8((uint64(r) + inv) >> 8),
				uint8((uint64(g) + inv) >> 8),
				uint8((uint64(b) + inv) >> 8),
				0xff,
			})
		}
	}
	return dst
}
