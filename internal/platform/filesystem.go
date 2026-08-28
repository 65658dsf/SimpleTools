package platform

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// Rename semantics differ across operating systems: Unix may replace an
// existing destination while Windows rejects it. Serialize the final check
// and rename so concurrent jobs in this process cannot overwrite each other.
var atomicRenameMu sync.Mutex

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".avif": true,
}

func IsPDFPath(path string) bool   { return strings.EqualFold(filepath.Ext(path), ".pdf") }
func IsImagePath(path string) bool { return imageExts[strings.ToLower(filepath.Ext(path))] }

// CollectPaths expands a file or directory into supported regular files. kind
// is "image" or "pdf"; an empty kind accepts both. Results are sorted so job
// order and generated names remain deterministic across operating systems.
func CollectPaths(path, kind string, recursive bool) ([]string, error) {
	path = filepath.Clean(path)
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	accept := func(candidate string) bool {
		switch strings.ToLower(kind) {
		case "image":
			return IsImagePath(candidate)
		case "pdf":
			return IsPDFPath(candidate)
		default:
			return IsImagePath(candidate) || IsPDFPath(candidate)
		}
	}
	if !st.IsDir() {
		if !accept(path) {
			return nil, fmt.Errorf("unsupported input type: %s", filepath.Ext(path))
		}
		return []string{path}, nil
	}
	var out []string
	if recursive {
		err = filepath.Walk(path, func(p string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.Mode().IsRegular() && accept(p) {
				out = append(out, filepath.Clean(p))
			}
			return nil
		})
	} else {
		var entries []os.DirEntry
		entries, err = os.ReadDir(path)
		if err == nil {
			for _, entry := range entries {
				candidate := filepath.Join(path, entry.Name())
				if !entry.IsDir() && accept(candidate) {
					out = append(out, filepath.Clean(candidate))
				}
			}
		}
	}
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func CollectImages(path string, recursive bool) ([]string, error) {
	return CollectPaths(path, "image", recursive)
}

func UniqueName(dir, base, ext string) string {
	base = strings.TrimSuffix(base, filepath.Ext(base))
	ext = strings.ToLower(ext)
	name := filepath.Join(dir, base+ext)
	if _, err := os.Lstat(name); os.IsNotExist(err) {
		return name
	}
	for i := 1; ; i++ {
		name = filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
		if _, err := os.Lstat(name); os.IsNotExist(err) {
			return name
		}
	}
}

func AtomicWrite(dir, name string, write func(io.Writer) error) error {
	dir, err := filepath.Abs(filepath.Clean(dir))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// A caller can create a nested output directory from user-controlled
	// relative input paths. Refuse symlinked directories so the lexical output
	// root cannot be redirected outside itself between allocation and rename.
	linked, err := pathContainsLink(dir)
	if err != nil {
		return err
	}
	if linked {
		return fmt.Errorf("output directory %q contains a symbolic link", dir)
	}
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	resolvedDir, err = filepath.Abs(filepath.Clean(resolvedDir))
	if err != nil {
		return err
	}
	if !samePath(dir, resolvedDir) {
		return fmt.Errorf("output directory %q contains a symbolic link", dir)
	}
	dir = resolvedDir
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		return err
	}
	target := name
	if !filepath.IsAbs(target) {
		target = filepath.Join(dir, target)
	}
	target, err = filepath.Abs(filepath.Clean(target))
	if err != nil {
		return err
	}
	within, err := IsWithin(dir, target)
	if err != nil {
		return err
	}
	if !within {
		return fmt.Errorf("target path %q is outside output directory", target)
	}
	tmp := filepath.Join(filepath.Dir(target), fmt.Sprintf(".%s.%x.tmp", filepath.Base(target), token))
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err = write(f); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	atomicRenameMu.Lock()
	defer atomicRenameMu.Unlock()
	if _, statErr := os.Lstat(target); statErr == nil {
		return fmt.Errorf("target path %q already exists", target)
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	if err = os.Rename(tmp, target); err != nil {
		return err
	}
	ok = true
	return nil
}

func IsWithin(parent, child string) (bool, error) {
	parent, err := filepath.Abs(filepath.Clean(parent))
	if err != nil {
		return false, err
	}
	child, err = filepath.Abs(filepath.Clean(child))
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false, err
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel), nil
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	originalA, originalB := a, b
	if normalized, err := normalizePathAliases(a); err == nil {
		a = filepath.Clean(normalized)
	}
	if normalized, err := normalizePathAliases(b); err == nil {
		b = filepath.Clean(normalized)
	}
	if runtime.GOOS == "windows" {
		if strings.EqualFold(a, b) {
			return true
		}
	} else if a == b {
		return true
	}
	// EvalSymlinks canonicalizes trusted system aliases (for example,
	// macOS /var -> /private/var). Compare file identity for those aliases.
	// A symbolic link used as the output directory itself describes a different
	// file under Lstat, so it still fails this check.
	left, leftErr := os.Lstat(originalA)
	right, rightErr := os.Lstat(originalB)
	return leftErr == nil && rightErr == nil && os.SameFile(left, right)
}

// pathContainsLink reports whether any existing component of path is a
// symbolic link or another path reparse point. AtomicWrite calls this after
// creating the directory tree so a user-controlled relative directory cannot
// redirect output outside the selected root.
func pathContainsLink(path string) (bool, error) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false, err
	}
	volume := filepath.VolumeName(abs)
	rest := strings.TrimPrefix(abs, volume)
	current := volume
	if strings.HasPrefix(rest, string(filepath.Separator)) {
		current += string(filepath.Separator)
		rest = strings.TrimLeft(rest, string(filepath.Separator))
	}
	for rest != "" {
		part := rest
		if index := strings.IndexByte(part, filepath.Separator); index >= 0 {
			part, rest = part[:index], strings.TrimLeft(part[index+1:], string(filepath.Separator))
		} else {
			rest = ""
		}
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			return false, nil
		}
		if statErr != nil {
			return false, statErr
		}
		if isLinkComponent(current, info) && !isTrustedLinkComponent(current) {
			return true, nil
		}
	}
	return false, nil
}
