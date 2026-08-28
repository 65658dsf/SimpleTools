//go:build !cgo || !mupdf

package tools

// DefaultPDFRenderer keeps ordinary development and test builds independent of
// MuPDF. Native release builds select MuPDFRenderer from pdf_mupdf.go instead.
func DefaultPDFRenderer() PDFRenderer { return UnavailablePDFRenderer{} }
