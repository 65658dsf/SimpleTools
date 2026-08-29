package tools

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
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
	FormatICO  Format = "ico"
	FormatSVG  Format = "svg"
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
	case "ico":
		return FormatICO, nil
	case "svg":
		return FormatSVG, nil
	default:
		return "", fmt.Errorf("unsupported image format %q", v)
	}
}

func FormatFromPath(p string) (Format, error) { return ParseFormat(filepath.Ext(p)) }

// DecodeConfig reads an image header and returns its dimensions without
// allocating the decoded pixel buffer. The custom ICO and SVG readers inspect
// their embedded raster payloads with the same config-only path.
func DecodeConfig(r io.Reader, ext string) (image.Config, error) {
	if r == nil {
		return image.Config{}, fmt.Errorf("image reader is required")
	}
	format, err := ParseFormat(ext)
	if err != nil {
		return image.Config{}, err
	}
	switch format {
	case FormatPNG, FormatJPEG:
		config, _, err := image.DecodeConfig(io.LimitReader(r, maxImageBytes+1))
		if err != nil {
			return image.Config{}, fmt.Errorf("decode %s config: %w", format, err)
		}
		return config, nil
	case FormatWebP:
		config, err := webp.DecodeConfig(io.LimitReader(r, maxImageBytes+1))
		if err != nil {
			return image.Config{}, fmt.Errorf("decode %s config: %w", format, err)
		}
		return config, nil
	case FormatAVIF:
		config, err := avif.DecodeConfig(io.LimitReader(r, maxImageBytes+1))
		if err != nil {
			return image.Config{}, fmt.Errorf("decode %s config: %w", format, err)
		}
		return config, nil
	case FormatICO, FormatSVG:
		data, err := io.ReadAll(io.LimitReader(r, maxImageBytes+1))
		if err != nil {
			return image.Config{}, err
		}
		if len(data) > maxImageBytes {
			return image.Config{}, fmt.Errorf("image exceeds the %d MiB safety limit", maxImageBytes/(1024*1024))
		}
		if format == FormatICO {
			return decodeICOConfig(data)
		}
		return decodeSVGConfig(data)
	default:
		return image.Config{}, fmt.Errorf("unsupported image format %q", format)
	}
}

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
	case FormatICO:
		img, err = decodeICO(data)
	case FormatSVG:
		img, err = decodeSVG(data)
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
	case FormatICO:
		return encodeICO(w, img)
	case FormatSVG:
		return encodeSVG(w, img)
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

// encodeICO writes a modern PNG-backed ICO containing a single 32-bit image.
// Windows treats a zero directory dimension as 256, which is the largest
// image size supported by this encoder; larger images are scaled down first.
func encodeICO(w io.Writer, src image.Image) error {
	img := fitICOImage(src, 256)
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		return err
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width < 1 || height < 1 || width > 256 || height > 256 {
		return fmt.Errorf("ICO dimensions must be between 1 and 256 pixels")
	}
	// ICONDIR followed by one ICONDIRENTRY. The image payload is PNG, which is
	// supported by current Windows, macOS, and Linux icon readers.
	var header [6 + 16]byte
	binary.LittleEndian.PutUint16(header[0:2], 0)
	binary.LittleEndian.PutUint16(header[2:4], 1)
	binary.LittleEndian.PutUint16(header[4:6], 1)
	if width == 256 {
		header[6] = 0
	} else {
		header[6] = byte(width)
	}
	if height == 256 {
		header[7] = 0
	} else {
		header[7] = byte(height)
	}
	// Color count and reserved remain zero. Planes and bits-per-pixel describe
	// the source image for readers that inspect the directory metadata.
	binary.LittleEndian.PutUint16(header[10:12], 1)
	binary.LittleEndian.PutUint16(header[12:14], 32)
	binary.LittleEndian.PutUint32(header[14:18], uint32(pngData.Len()))
	binary.LittleEndian.PutUint32(header[18:22], uint32(len(header)))
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(pngData.Bytes())
	return err
}

func fitICOImage(src image.Image, maxDimension int) image.Image {
	b := src.Bounds()
	width, height := b.Dx(), b.Dy()
	if width <= maxDimension && height <= maxDimension {
		return src
	}
	scale := float64(maxDimension) / float64(max(width, height))
	dw := max(1, int(float64(width)*scale))
	dh := max(1, int(float64(height)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			sx := b.Min.X + x*width/dw
			sy := b.Min.Y + y*height/dh
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func decodeICO(data []byte) (image.Image, error) {
	if len(data) < 22 {
		return nil, fmt.Errorf("ICO header is truncated")
	}
	if binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return nil, fmt.Errorf("invalid ICO header")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count < 1 || len(data) < 6+count*16 {
		return nil, fmt.Errorf("ICO directory is truncated")
	}
	var firstErr error
	for index := 0; index < count; index++ {
		entry := data[6+index*16 : 6+(index+1)*16]
		size := uint64(binary.LittleEndian.Uint32(entry[8:12]))
		offset := uint64(binary.LittleEndian.Uint32(entry[12:16]))
		if size == 0 || offset > uint64(len(data)) || size > uint64(len(data))-offset {
			if firstErr == nil {
				firstErr = fmt.Errorf("ICO image entry %d is outside the file", index+1)
			}
			continue
		}
		payload := data[offset : offset+size]
		if len(payload) < 8 || !bytes.Equal(payload[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
			if firstErr == nil {
				firstErr = fmt.Errorf("ICO image entry %d uses an unsupported bitmap encoding", index+1)
			}
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(payload))
		if err == nil {
			return img, nil
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("decode ICO image entry %d: %w", index+1, err)
		}
	}
	if firstErr == nil {
		firstErr = fmt.Errorf("ICO contains no decodable image")
	}
	return nil, firstErr
}

func decodeICOConfig(data []byte) (image.Config, error) {
	if len(data) < 22 {
		return image.Config{}, fmt.Errorf("ICO header is truncated")
	}
	if binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return image.Config{}, fmt.Errorf("invalid ICO header")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count < 1 || len(data) < 6+count*16 {
		return image.Config{}, fmt.Errorf("ICO directory is truncated")
	}
	var firstErr error
	for index := 0; index < count; index++ {
		entry := data[6+index*16 : 6+(index+1)*16]
		size := uint64(binary.LittleEndian.Uint32(entry[8:12]))
		offset := uint64(binary.LittleEndian.Uint32(entry[12:16]))
		if size == 0 || offset > uint64(len(data)) || size > uint64(len(data))-offset {
			if firstErr == nil {
				firstErr = fmt.Errorf("ICO image entry %d is outside the file", index+1)
			}
			continue
		}
		payload := data[offset : offset+size]
		if len(payload) < 8 || !bytes.Equal(payload[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
			if firstErr == nil {
				firstErr = fmt.Errorf("ICO image entry %d uses an unsupported bitmap encoding", index+1)
			}
			continue
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(payload))
		if err == nil {
			return config, nil
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("decode ICO image entry %d: %w", index+1, err)
		}
	}
	if firstErr == nil {
		firstErr = fmt.Errorf("ICO contains no decodable image")
	}
	return image.Config{}, firstErr
}

type svgImageElement struct {
	Href      string `xml:"href,attr"`
	XLinkHref string `xml:"xlink:href,attr"`
}

type svgDocument struct {
	Images []svgImageElement `xml:"image"`
}

// encodeSVG keeps SVG conversion deterministic and offline by embedding a PNG
// snapshot. This preserves alpha and makes the generated SVG self-contained.
func encodeSVG(w io.Writer, src image.Image) error {
	b := src.Bounds()
	width, height := b.Dx(), b.Dy()
	if width < 1 || height < 1 {
		return fmt.Errorf("SVG dimensions must be positive")
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, src); err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(pngData.Bytes())
	_, err := fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<svg xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\"><image width=\"%d\" height=\"%d\" preserveAspectRatio=\"none\" href=\"data:image/png;base64,%s\"/></svg>\n", width, height, width, height, width, height, encoded)
	return err
}

func decodeSVG(data []byte) (image.Image, error) {
	var doc svgDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse SVG: %w", err)
	}
	for _, element := range doc.Images {
		href := strings.TrimSpace(element.Href)
		if href == "" {
			href = strings.TrimSpace(element.XLinkHref)
		}
		img, err := decodeSVGDataURI(href)
		if err == nil {
			return img, nil
		}
	}
	return nil, fmt.Errorf("SVG does not contain a supported embedded raster image")
}

func decodeSVGConfig(data []byte) (image.Config, error) {
	var doc svgDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return image.Config{}, fmt.Errorf("parse SVG: %w", err)
	}
	var firstErr error
	for _, element := range doc.Images {
		href := strings.TrimSpace(element.Href)
		if href == "" {
			href = strings.TrimSpace(element.XLinkHref)
		}
		config, err := decodeSVGDataURIConfig(href)
		if err == nil {
			return config, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr == nil {
		firstErr = fmt.Errorf("SVG does not contain a supported embedded raster image")
	}
	return image.Config{}, firstErr
}

func decodeSVGDataURI(href string) (image.Image, error) {
	mime, payload, err := decodeSVGDataURIBytes(href)
	if err != nil {
		return nil, err
	}
	switch mime {
	case "image/png", "image/jpeg", "image/jpg":
		img, _, err := image.Decode(bytes.NewReader(payload))
		return img, err
	case "image/webp":
		return webp.Decode(bytes.NewReader(payload), webp.Options{AutoRotate: true})
	case "image/avif":
		return avif.Decode(bytes.NewReader(payload), avif.Options{AutoRotate: true})
	default:
		return nil, fmt.Errorf("unsupported embedded SVG image type %q", mime)
	}
}

func decodeSVGDataURIConfig(href string) (image.Config, error) {
	mime, payload, err := decodeSVGDataURIBytes(href)
	if err != nil {
		return image.Config{}, err
	}
	switch mime {
	case "image/png", "image/jpeg", "image/jpg":
		config, _, err := image.DecodeConfig(bytes.NewReader(payload))
		return config, err
	case "image/webp":
		return webp.DecodeConfig(bytes.NewReader(payload))
	case "image/avif":
		return avif.DecodeConfig(bytes.NewReader(payload))
	default:
		return image.Config{}, fmt.Errorf("unsupported embedded SVG image type %q", mime)
	}
}

func decodeSVGDataURIBytes(href string) (string, []byte, error) {
	if !strings.HasPrefix(strings.ToLower(href), "data:") {
		return "", nil, fmt.Errorf("SVG image source must be a data URI")
	}
	header, encoded, ok := strings.Cut(href[5:], ",")
	if !ok || !strings.Contains(strings.ToLower(header), ";base64") {
		return "", nil, fmt.Errorf("SVG image source must use base64 encoding")
	}
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, fmt.Errorf("decode SVG image data: %w", err)
	}
	mime := strings.ToLower(strings.TrimSpace(strings.SplitN(header, ";", 2)[0]))
	return mime, payload, nil
}
