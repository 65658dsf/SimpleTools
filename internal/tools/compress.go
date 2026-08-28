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
// the requested target size. PNG is intentionally kept lossless.
func CompressImage(img image.Image, format Format, quality int, targetBytes int64, lossless bool) (CompressionResult, error) {
	if quality < 1 {
		quality = 85
	}
	if quality > 100 {
		quality = 100
	}
	if targetBytes <= 0 || format == FormatPNG || lossless {
		data, err := EncodeBytes(img, format, EncodeOptions{Quality: quality, Lossless: lossless})
		result := CompressionResult{Data: data, Quality: quality, Compressed: int64(len(data))}
		if err == nil && targetBytes > 0 && float64(len(data)) > float64(targetBytes)*1.05 {
			result.Warning = fmt.Sprintf("target size %d bytes could not be reached; result is %d bytes", targetBytes, len(data))
		}
		return result, err
	}
	// Encode at the user's quality first. If it already fits, preserve the
	// requested quality rather than degrading it unnecessarily.
	best, err := EncodeBytes(img, format, EncodeOptions{Quality: quality})
	if err != nil {
		return CompressionResult{}, err
	}
	if int64(len(best)) <= targetBytes {
		return CompressionResult{Data: best, Quality: quality, Compressed: int64(len(best))}, nil
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
	result := CompressionResult{Data: bestData, Quality: bestQuality, Compressed: int64(len(bestData))}
	if float64(len(bestData)) > float64(targetBytes)*1.05 {
		result.Warning = fmt.Sprintf("target size %d bytes could not be reached; smallest result is %d bytes", targetBytes, len(bestData))
	}
	return result, nil
}
