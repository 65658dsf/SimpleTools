//go:build cgo && mupdf

package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	fitz "github.com/gen2brain/go-fitz"
)

// MuPDFRenderer is enabled only for native release builds. go-fitz bundles the
// platform static MuPDF archives; CI must provide a C toolchain and use the
// `mupdf,nodynamic` tags so the AVIF codec never loads a shared library.
type MuPDFRenderer struct{}

func DefaultPDFRenderer() PDFRenderer { return MuPDFRenderer{} }

func (MuPDFRenderer) Render(ctx context.Context, path string, options PDFOptions, onPage func(PDFPage) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if options.DPI == 0 {
		options.DPI = 150
	}
	if options.DPI != 72 && options.DPI != 150 && options.DPI != 300 && options.DPI != 600 {
		return fmt.Errorf("DPI must be one of 72, 150, 300, or 600")
	}
	if options.MaxPixels <= 0 {
		options.MaxPixels = 120_000_000
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	doc, err := fitz.NewFromReader(f)
	if doc != nil {
		// go-fitz may return a non-nil document together with ErrNeedsPassword;
		// release that native context on every error path.
		defer doc.Close()
	}
	if err != nil {
		if errors.Is(err, fitz.ErrNeedsPassword) {
			return ErrPDFPasswordRequired
		}
		return fmt.Errorf("open PDF: %w", err)
	}
	if err := installCJKFallback(doc); err != nil {
		return fmt.Errorf("install CJK fallback font: %w", err)
	}
	pages, err := ParsePageRange(options.PageRange, doc.NumPage())
	if err != nil {
		return err
	}
	for index, pageNumber := range pages {
		if err := ctx.Err(); err != nil {
			return err
		}
		bounds, err := doc.Bound(pageNumber - 1)
		if err != nil {
			return fmt.Errorf("page %d: %w", pageNumber, err)
		}
		width := int64((bounds.Dx()*options.DPI + 71) / 72)
		height := int64((bounds.Dy()*options.DPI + 71) / 72)
		if width <= 0 || height <= 0 || width*height > options.MaxPixels {
			return fmt.Errorf("page %d: %w", pageNumber, ErrPDFPageTooLarge)
		}
		img, err := doc.ImageDPI(pageNumber-1, float64(options.DPI))
		if err != nil {
			return fmt.Errorf("render page %d: %w", pageNumber, err)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, flattenPDFPage(img)); err != nil {
			return fmt.Errorf("encode page %d: %w", pageNumber, err)
		}
		if err := onPage(PDFPage{Number: pageNumber, Index: index, Total: len(pages), Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), PNG: buf.Bytes()}); err != nil {
			return err
		}
	}
	return nil
}

func flattenPDFPage(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}
