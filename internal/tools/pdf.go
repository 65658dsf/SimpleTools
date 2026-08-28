package tools

import (
	"context"
	"errors"
)

var (
	ErrPDFRendererUnavailable = errors.New("PDF rendering is unavailable on this build; enable the mupdf build tag")
	ErrPDFPasswordRequired    = errors.New("PDF is password protected")
	ErrPDFPageTooLarge        = errors.New("PDF page exceeds the rendering safety limit")
)

type PDFOptions struct {
	DPI       int
	PageRange string
	MaxPixels int64
}

type PDFPage struct {
	Number int
	// Index and Total describe the selected page sequence, allowing callers to
	// report meaningful page-level progress for sparse page ranges.
	Index  int
	Total  int
	Width  int
	Height int
	PNG    []byte
}

// PDFRenderer is deliberately small so a future sidecar or platform renderer
// can be substituted without changing the app/job contract.
type PDFRenderer interface {
	Render(ctx context.Context, path string, options PDFOptions, onPage func(PDFPage) error) error
}

// UnavailablePDFRenderer is used by default development/test builds. Release
// targets enable the `mupdf` build tag, which supplies MuPDFRenderer.
type UnavailablePDFRenderer struct{}

func (UnavailablePDFRenderer) Render(context.Context, string, PDFOptions, func(PDFPage) error) error {
	return ErrPDFRendererUnavailable
}
