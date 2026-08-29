package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Tool string

const (
	ToolConvert   Tool = "convert"
	ToolCompress  Tool = "compress"
	ToolPDF       Tool = "pdf"
	ToolWatermark Tool = "watermark"
)

type JobRequest struct {
	Tool   Tool     `json:"tool"`
	Inputs []string `json:"inputs"`
	// InputRelativeDirs carries the directory relative to the selected folder
	// for each expanded file. It lets the UI remove individual files while
	// retaining the folder layout in the output tree.
	InputRelativeDirs map[string]string `json:"inputRelativeDirs,omitempty"`
	OutputDirectory   string            `json:"outputDirectory"`
	Format            string            `json:"format,omitempty"`
	Quality           int               `json:"quality,omitempty"`
	TargetBytes       int64             `json:"targetBytes,omitempty"`
	Lossless          bool              `json:"lossless,omitempty"`
	PreserveMetadata  bool              `json:"preserveMetadata,omitempty"`
	Recursive         bool              `json:"recursive,omitempty"`
	DPI               int               `json:"dpi,omitempty"`
	PageRange         string            `json:"pageRange,omitempty"`
	MaxPixels         int64             `json:"maxPixels,omitempty"`
	Watermark         *WatermarkOptions `json:"watermark,omitempty"`
}

func (r JobRequest) normalizedTool() (Tool, error) {
	tool := r.Tool
	if tool == "" {
		tool = ToolConvert
	}
	switch tool {
	case ToolConvert, ToolCompress, ToolPDF, ToolWatermark:
		return tool, nil
	default:
		return "", fmt.Errorf("unsupported tool %q", tool)
	}
}

func (r JobRequest) Validate() (Format, error) {
	tool, err := r.normalizedTool()
	if err != nil {
		return "", err
	}
	if len(r.Inputs) == 0 {
		return "", fmt.Errorf("at least one input is required")
	}
	if strings.TrimSpace(r.OutputDirectory) == "" {
		return "", fmt.Errorf("output directory is required")
	}
	output, err := filepath.Abs(filepath.Clean(r.OutputDirectory))
	if err != nil {
		return "", fmt.Errorf("output directory: %w", err)
	}
	st, err := os.Stat(output)
	if err != nil {
		return "", fmt.Errorf("output directory: %w", err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("output directory is not a directory")
	}
	for _, p := range r.Inputs {
		if strings.TrimSpace(p) == "" {
			return "", fmt.Errorf("input path cannot be empty")
		}
		if _, err := os.Stat(filepath.Clean(p)); err != nil {
			return "", fmt.Errorf("input %q: %w", p, err)
		}
	}
	if tool == ToolPDF {
		if r.DPI != 0 && r.DPI != 72 && r.DPI != 150 && r.DPI != 300 && r.DPI != 600 {
			return "", fmt.Errorf("DPI must be one of 72, 150, 300, or 600")
		}
		if r.TargetBytes != 0 {
			return "", fmt.Errorf("target size is not supported for PDF output")
		}
		return FormatPNG, nil
	}
	if tool == ToolConvert {
		if strings.TrimSpace(r.Format) == "" {
			return "", fmt.Errorf("output format is required")
		}
		format, err := ParseFormat(r.Format)
		if err != nil {
			return "", err
		}
		if r.Quality != 0 && (r.Quality < 1 || r.Quality > 100) {
			return "", fmt.Errorf("quality must be between 1 and 100")
		}
		return format, nil
	}
	if tool == ToolWatermark {
		if r.Watermark == nil {
			return "", fmt.Errorf("watermark options are required")
		}
		if _, err := r.Watermark.normalized(); err != nil {
			return "", err
		}
		if r.TargetBytes != 0 {
			return "", fmt.Errorf("target size is not supported for watermark output")
		}
		return "", nil
	}
	if r.Quality != 0 && (r.Quality < 1 || r.Quality > 100) {
		return "", fmt.Errorf("quality must be between 1 and 100")
	}
	if r.TargetBytes < 0 {
		return "", fmt.Errorf("target size cannot be negative")
	}
	return "", nil
}

func (r JobRequest) ToolKind() Tool {
	tool, _ := r.normalizedTool()
	return tool
}
