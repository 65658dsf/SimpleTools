package tools

import (
	"path/filepath"
	"strings"
)

func OutputBase(path string) string {
	b := filepath.Base(path)
	return strings.TrimSuffix(b, filepath.Ext(b))
}
