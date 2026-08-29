package tools

import (
	"errors"
	"fmt"
	"image"
	"unicode/utf8"

	gozxing "github.com/makiuchi-d/gozxing"
	zxingqrcode "github.com/makiuchi-d/gozxing/qrcode"
)

const (
	// MaxQRCodeDecodeBytes bounds the compressed input retained while
	// inspecting container formats such as ICO and SVG.
	MaxQRCodeDecodeBytes = 64 << 20

	// MaxQRCodeDecodeTextBytes is the maximum amount of decoded UTF-8 text
	// that the application may expose over the Wails bridge.
	MaxQRCodeDecodeTextBytes = 64 << 10

	// MaxQRCodeDecodePixels bounds the work and temporary memory required by
	// the pure-Go detector before it scans an image supplied by a user.
	MaxQRCodeDecodePixels = 64 << 20
)

// ErrQRCodeNotFound distinguishes an image without a readable QR symbol from
// malformed image data and QR payload errors.
var ErrQRCodeNotFound = errors.New("no QR code found")

// DecodeQRCode detects one QR code in img and returns its UTF-8 payload. It
// performs no filesystem or network access. The detector is deliberately
// bounded because image dimensions are controlled by the input file.
func DecodeQRCode(img image.Image) (string, error) {
	if img == nil {
		return "", fmt.Errorf("QR code image is required")
	}
	bounds := img.Bounds()
	if err := ValidateQRCodeDimensions(bounds.Dx(), bounds.Dy()); err != nil {
		return "", err
	}

	source := gozxing.NewLuminanceSourceFromImage(img)
	var nonNotFoundErr error
	// TRY_HARDER improves detection for photographs and downscaled images;
	// PURE_BARCODE keeps clean generated PNGs reliable. Run both luminance
	// polarities because custom QR colors may use a light foreground on a dark
	// background.
	for _, inverted := range []bool{false, true} {
		luminance := source
		if inverted {
			luminance = source.Invert()
		}
		bitmap, bitmapErr := gozxing.NewBinaryBitmap(gozxing.NewHybridBinarizer(luminance))
		if bitmapErr != nil {
			if nonNotFoundErr == nil {
				nonNotFoundErr = bitmapErr
			}
			continue
		}
		for _, hints := range []map[gozxing.DecodeHintType]interface{}{
			{gozxing.DecodeHintType_TRY_HARDER: true},
			{gozxing.DecodeHintType_PURE_BARCODE: true},
		} {
			result, decodeErr := zxingqrcode.NewQRCodeReader().Decode(bitmap, hints)
			if decodeErr == nil && result != nil {
				return validateDecodedQRCodeText(result.GetText())
			}
			if decodeErr == nil {
				decodeErr = ErrQRCodeNotFound
			}
			if !isQRCodeNotFound(decodeErr) && nonNotFoundErr == nil {
				nonNotFoundErr = decodeErr
			}
		}
	}
	if nonNotFoundErr != nil {
		return "", fmt.Errorf("decode QR code: %w", nonNotFoundErr)
	}
	return "", ErrQRCodeNotFound
}

// ValidateQRCodeDimensions checks the image dimensions before any pixel scan
// or decoder allocation is attempted.
func ValidateQRCodeDimensions(width, height int) error {
	if width < 1 || height < 1 {
		return fmt.Errorf("QR code image dimensions must be positive")
	}
	if exceedsQRCodePixelLimit(width, height) {
		return fmt.Errorf("QR code image exceeds the %d-pixel safety limit", MaxQRCodeDecodePixels)
	}
	return nil
}

// BoundQRCodeText truncates decoded text at a UTF-8 rune boundary. The
// original byte count can be retained by callers before applying the bound.
func BoundQRCodeText(text string, limit int) (string, bool, error) {
	if !utf8.ValidString(text) {
		return "", false, fmt.Errorf("QR code text is not valid UTF-8")
	}
	if limit <= 0 {
		return "", false, fmt.Errorf("QR code text limit must be positive")
	}
	if len(text) <= limit {
		return text, false, nil
	}
	cut := text[:limit]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut, true, nil
}

func validateDecodedQRCodeText(text string) (string, error) {
	if !utf8.ValidString(text) {
		return "", fmt.Errorf("QR code text is not valid UTF-8")
	}
	if text == "" {
		return "", fmt.Errorf("QR code contains no text")
	}
	return text, nil
}

func isQRCodeNotFound(err error) bool {
	if errors.Is(err, ErrQRCodeNotFound) {
		return true
	}
	var notFound gozxing.NotFoundException
	return errors.As(err, &notFound)
}

func exceedsQRCodePixelLimit(width, height int) bool {
	if width < 1 || height < 1 {
		return false
	}
	return width > MaxQRCodeDecodePixels/height
}
