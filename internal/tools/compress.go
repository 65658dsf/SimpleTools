package tools

import (
	"fmt"
	"image"
)

type CompressionResult struct {
	Data       []byte
	Quality    int
	Warning    string
	Original   int64
	Compressed int64
}

// CompressImage encodes an image once, or performs a bounded binary search for
// the requested target size. PNG, ICO, and SVG are intentionally kept
// lossless because their encoders do not expose a useful quality control.
//
// The image value does not retain the size of its source encoding, so callers
// that have that information should use CompressImageWithOriginal. This
// compatibility wrapper keeps the original API for callers that only need the
// encoded bytes.
func CompressImage(img image.Image, format Format, quality int, targetBytes int64, lossless bool) (CompressionResult, error) {
	return CompressImageWithOriginal(img, format, quality, targetBytes, lossless, 0)
}

// CompressImageWithOriginal is CompressImage with the source file size carried
// through to the result for size comparisons in the application layer.
func CompressImageWithOriginal(img image.Image, format Format, quality int, targetBytes int64, lossless bool, originalBytes int64) (CompressionResult, error) {
	if originalBytes < 0 {
		originalBytes = 0
	}
	if quality < 1 {
		quality = 85
	}
	if quality > 100 {
		quality = 100
	}
	if targetBytes <= 0 || format == FormatPNG || format == FormatICO || format == FormatSVG || lossless {
		data, err := EncodeBytes(img, format, EncodeOptions{Quality: quality, Lossless: lossless})
		if err != nil {
			return CompressionResult{}, err
		}
		result := newCompressionResult(data, quality, originalBytes)
		if targetBytes > 0 && float64(len(data)) > float64(targetBytes)*1.05 {
			result.Warning = fmt.Sprintf("target size %d bytes could not be reached; result is %d bytes", targetBytes, len(data))
		}
		return result, nil
	}
	// Encode at the user's quality first. If it already fits, preserve the
	// requested quality rather than degrading it unnecessarily.
	best, err := EncodeBytes(img, format, EncodeOptions{Quality: quality})
	if err != nil {
		return CompressionResult{}, err
	}
	if int64(len(best)) <= targetBytes {
		return newCompressionResult(best, quality, originalBytes), nil
	}
	low, high := 1, quality
	bestQuality := 0
	var bestData []byte
	for attempt := 0; attempt < 8 && low <= high; attempt++ {
		candidateQuality := (low + high) / 2
		candidate, encodeErr := EncodeBytes(img, format, EncodeOptions{Quality: candidateQuality})
		if encodeErr != nil {
			return CompressionResult{}, encodeErr
		}
		if int64(len(candidate)) > targetBytes {
			high = candidateQuality - 1
			continue
		}
		bestQuality, bestData = candidateQuality, candidate
		low = candidateQuality + 1
	}
	if bestData == nil {
		bestQuality = 1
		bestData, err = EncodeBytes(img, format, EncodeOptions{Quality: bestQuality})
		if err != nil {
			return CompressionResult{}, err
		}
	}
	result := newCompressionResult(bestData, bestQuality, originalBytes)
	if float64(len(bestData)) > float64(targetBytes)*1.05 {
		result.Warning = fmt.Sprintf("target size %d bytes could not be reached; smallest result is %d bytes", targetBytes, len(bestData))
	}
	return result, nil
}

func newCompressionResult(data []byte, quality int, originalBytes int64) CompressionResult {
	return CompressionResult{
		Data:       data,
		Quality:    quality,
		Original:   originalBytes,
		Compressed: int64(len(data)),
	}
}
