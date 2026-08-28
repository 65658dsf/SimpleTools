//go:build cgo && mupdf && nocgo

package tools

import fitz "github.com/gen2brain/go-fitz"

// The purego go-fitz implementation cannot share the C callback hook. Keep
// the mupdf,nocgo build usable while native release builds use pdf_font_mupdf.go.
func installCJKFallback(*fitz.Document) error { return nil }
