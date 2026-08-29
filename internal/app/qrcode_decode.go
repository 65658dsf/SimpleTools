package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/65658dsf/SimpleTools/internal/platform"
	"github.com/65658dsf/SimpleTools/internal/tools"
)

// QRCodeDecodeResult contains bounded decoded text and metadata for the
// source image. Image bytes never cross the Wails bridge.
type QRCodeDecodeResult struct {
	Path      string `json:"path"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Format    string `json:"format"`
	Size      int64  `json:"size"`
	Text      string `json:"text"`
	TextBytes int    `json:"textBytes"`
	Truncated bool   `json:"truncated,omitempty"`
}

// DecodeQRCode reads one supported image from a canonical regular path and
// returns the first QR payload found in it. The decoded text is bounded before
// it is returned through Wails.
func (a *App) DecodeQRCode(path string) (*QRCodeDecodeResult, error) {
	rawPath := strings.TrimSpace(path)
	if rawPath == "" {
		return nil, fmt.Errorf("QR code image path is required")
	}

	canonical, err := canonicalInputPath(rawPath)
	if err != nil {
		return nil, fmt.Errorf("canonicalize QR code image path: %w", err)
	}
	if !platform.IsImagePath(canonical) {
		return nil, fmt.Errorf("unsupported QR code image type %q", filepath.Ext(canonical))
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return nil, fmt.Errorf("stat QR code image: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("QR code image path is not a regular file")
	}
	if info.Size() > tools.MaxQRCodeDecodeBytes {
		return nil, fmt.Errorf("QR code image exceeds the %d MiB file safety limit", tools.MaxQRCodeDecodeBytes/(1024*1024))
	}
	ctx := context.Background()
	a.mu.RLock()
	if a.ctx != nil {
		ctx = a.ctx
	}
	a.mu.RUnlock()
	release, acquired := a.acquireToolSlot(ctx, tools.ToolConvert)
	if !acquired {
		return nil, fmt.Errorf("QR code decoding was cancelled")
	}
	defer release()

	f, err := os.Open(canonical)
	if err != nil {
		return nil, fmt.Errorf("open QR code image: %w", err)
	}
	defer f.Close()
	ext := filepath.Ext(canonical)
	config, err := tools.DecodeConfig(f, ext)
	if err != nil {
		return nil, fmt.Errorf("decode QR code image configuration: %w", err)
	}
	if err := tools.ValidateQRCodeDimensions(config.Width, config.Height); err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind QR code image: %w", err)
	}
	img, err := tools.Decode(f, ext)
	if err != nil {
		return nil, fmt.Errorf("decode QR code image: %w", err)
	}
	text, err := tools.DecodeQRCode(img)
	if err != nil {
		return nil, fmt.Errorf("read QR code: %w", err)
	}
	bounded, textBytes, truncated, err := boundQRCodeDecodeResultText(text)
	if err != nil {
		return nil, fmt.Errorf("bound QR code text: %w", err)
	}
	bounds := img.Bounds()
	return &QRCodeDecodeResult{
		Path:      canonical,
		Width:     bounds.Dx(),
		Height:    bounds.Dy(),
		Format:    strings.TrimPrefix(strings.ToLower(filepath.Ext(canonical)), "."),
		Size:      info.Size(),
		Text:      bounded,
		TextBytes: textBytes,
		Truncated: truncated,
	}, nil
}

func boundQRCodeDecodeResultText(text string) (bounded string, textBytes int, truncated bool, err error) {
	textBytes = len(text)
	bounded, truncated, err = tools.BoundQRCodeText(text, tools.MaxQRCodeDecodeTextBytes)
	return bounded, textBytes, truncated, err
}
